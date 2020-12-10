package portfolio

import (
	"errors"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (p *Portfolio) Reset() {
	p.ExchangeAssetPairSettings = nil
}

// OnSignal receives the event from the strategy on whether it has signalled to buy, do nothing or sell
// on buy/sell, the portfolio manager will size the order and assess the risk of the order
// if successful, it will pass on an order.Order to be used by the exchange event handler to place an order based on
// the portfolio manager's recommendations
func (p *Portfolio) OnSignal(signal signal.SignalEvent, data data.Handler, cs *exchange.CurrencySettings) (*order.Order, error) {
	o := &order.Order{
		Event: event.Event{
			Exchange:     signal.GetExchange(),
			Time:         signal.GetTime(),
			CurrencyPair: signal.Pair(),
			AssetType:    signal.GetAssetType(),
			Interval:     signal.GetInterval(),
			Why:          signal.GetWhy(),
		},
		Direction: signal.GetDirection(),
	}
	if signal.GetDirection() == "" {
		return o, errors.New("invalid Direction")
	}

	lookup := p.ExchangeAssetPairSettings[signal.GetExchange()][signal.GetAssetType()][signal.Pair()]
	prevHolding := lookup.HoldingsSnapshots.GetLatestSnapshot()
	if prevHolding.InitialFunds == 0 {
		prevHolding.InitialFunds = lookup.InitialFunds
		prevHolding.RemainingFunds = lookup.InitialFunds
		prevHolding.Exchange = signal.GetExchange()
		prevHolding.Pair = signal.Pair()
		prevHolding.Asset = signal.GetAssetType()
		prevHolding.Timestamp = signal.GetTime()
	}

	if signal.GetDirection() == common.DoNothing {
		return o, nil
	}

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && prevHolding.PositionsSize == 0 {
		o.AppendWhy("no holdings to sell")
		o.SetDirection(common.CouldNotSell)
		signal.SetDirection(o.Direction)
		return o, nil
	}

	if (signal.GetDirection() == gctorder.Buy || signal.GetDirection() == gctorder.Bid) && prevHolding.RemainingFunds <= 0 {
		o.AppendWhy("not enough funds to buy")
		o.SetDirection(common.CouldNotBuy)
		signal.SetDirection(o.Direction)
		return o, nil
	}

	o.Price = signal.GetPrice()
	o.Amount = signal.GetAmount()
	o.OrderType = gctorder.Market
	latest := data.Latest()
	sizingFunds := prevHolding.RemainingFunds
	if signal.GetDirection() == gctorder.Sell {
		sizingFunds = prevHolding.PositionsSize
	}
	sizedOrder, err := p.SizeManager.SizeOrder(
		o,
		latest,
		sizingFunds,
		cs,
	)
	if err != nil {
		o.AppendWhy(err.Error())
		if o.Direction == gctorder.Buy {
			o.Direction = common.CouldNotBuy
		} else if o.Direction == gctorder.Sell {
			o.Direction = common.CouldNotSell
		} else {
			o.Direction = common.DoNothing
		}
		signal.SetDirection(o.Direction)
		return o, nil
	}

	var eo *order.Order
	eo, err = p.RiskManager.EvaluateOrder(sizedOrder, latest, p.GetLatestHoldingsForAllCurrencies())
	if err != nil {
		o.AppendWhy(err.Error())
		if signal.GetDirection() == gctorder.Buy {
			o.Direction = common.CouldNotBuy
		} else if signal.GetDirection() == gctorder.Sell {
			o.Direction = common.CouldNotSell
		} else {
			o.Direction = common.DoNothing
		}
		signal.SetDirection(o.Direction)
		return o, nil
	}

	return eo, nil
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions
func (p *Portfolio) OnFill(fillEvent fill.FillEvent, _ data.Handler) (*fill.Fill, error) {
	lookup := p.ExchangeAssetPairSettings[fillEvent.GetExchange()][fillEvent.GetAssetType()][fillEvent.Pair()]
	var err error
	// Get the holding from the previous iteration, create it if it doesn't yet have a timestamp
	h := p.ViewHoldingAtTimePeriod(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime().Add(-fillEvent.GetInterval().Duration()))
	if !h.Timestamp.IsZero() {
		h.Update(fillEvent)
	} else {
		h, err = holdings.Create(fillEvent, lookup.InitialFunds, p.RiskFreeRate)
		if err != nil {
			return nil, err
		}
	}
	err = p.SetHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime(), h, true)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	err = p.addComplianceSnapshot(fillEvent)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	if fillEvent.GetDirection() == common.DoNothing || fillEvent.GetDirection() == common.CouldNotBuy || fillEvent.GetDirection() == common.CouldNotSell {
		fe := fillEvent.(*fill.Fill)
		fe.ExchangeFee = 0
		return fe, nil
	}

	return fillEvent.(*fill.Fill), nil
}

// addComplianceSnapshot gets the previous snapshot of compliance events, updates with the latest fillevent
// then saves the snapshot to the c
func (p *Portfolio) addComplianceSnapshot(fillEvent fill.FillEvent) error {
	complianceManager, err := p.GetComplianceManager(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair())
	if err != nil {
		return err
	}
	if complianceManager.Interval == 0 {
		complianceManager.SetInterval(fillEvent.GetInterval())
	}
	prevSnap := complianceManager.GetPreviousSnapshot(fillEvent.GetTime())
	fo := fillEvent.GetOrder()
	if fo != nil {
		snapOrder := compliance.SnapshotOrder{
			ClosePrice:          fillEvent.GetClosePrice(),
			VolumeAdjustedPrice: fillEvent.GetVolumeAdjustedPrice(),
			SlippageRate:        fillEvent.GetSlippageRate(),
			Detail:              fo,
			CostBasis:           fo.Price + fo.Fee,
		}
		prevSnap.Orders = append(prevSnap.Orders, snapOrder)
	}
	err = complianceManager.AddSnapshot(prevSnap.Orders, fillEvent.GetTime(), true)
	if err != nil {
		return err
	}
	return nil
}

func (p *Portfolio) GetComplianceManager(exchangeName string, a asset.Item, cp currency.Pair) (*compliance.Manager, error) {
	lookup := p.ExchangeAssetPairSettings[exchangeName][a][cp]
	if lookup == nil {
		return nil, errors.New("not found")
	}
	return &lookup.ComplianceManager, nil
}

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.SizeManager = size
}

func (p *Portfolio) SetFee(exch string, a asset.Item, cp currency.Pair, fee float64) {
	lookup := p.ExchangeAssetPairSettings[exch][a][cp]
	lookup.Fee = fee
}

// GetFee can panic for bad requests, but why are you getting things that don't exist?
func (p *Portfolio) GetFee(exchangeName string, a asset.Item, cp currency.Pair) float64 {
	return p.ExchangeAssetPairSettings[exchangeName][a][cp].Fee
}

func (p *Portfolio) IsInvested(exchangeName string, a asset.Item, cp currency.Pair) (pos holdings.Holding, ok bool) {
	h := p.ExchangeAssetPairSettings[exchangeName][a][cp].GetLatestHoldings()
	if ok && (h.PositionsSize > 0) {
		return h, true
	}
	return h, false
}

func (p *Portfolio) Update(d interfaces.DataEventHandler) {
	if h, ok := p.IsInvested(d.GetExchange(), d.GetAssetType(), d.Pair()); ok {
		h.UpdateValue(d)
		err := p.SetHoldings(d.GetExchange(), d.GetAssetType(), d.Pair(), d.GetTime(), h, true)
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}
}

// ViewHoldingAtTimePeriod retrieves a snapshot of holdings at a specific time period,
// returning empty when not found
func (p *Portfolio) ViewHoldingAtTimePeriod(exch string, a asset.Item, cp currency.Pair, t time.Time) holdings.Holding {
	exchangeAssetPairSettings := p.ExchangeAssetPairSettings[exch][a][cp]
	for i := range exchangeAssetPairSettings.HoldingsSnapshots.Holdings {
		if t.Equal(exchangeAssetPairSettings.HoldingsSnapshots.Holdings[i].Timestamp) {
			return exchangeAssetPairSettings.HoldingsSnapshots.Holdings[i]
		}
	}

	return holdings.Holding{}
}

func (p *Portfolio) SetInitialFunds(exch string, a asset.Item, cp currency.Pair, funds float64) {
	p.ExchangeAssetPairSettings[exch][a][cp].InitialFunds = funds
}

func (p *Portfolio) GetInitialFunds(exch string, a asset.Item, cp currency.Pair) float64 {
	return p.ExchangeAssetPairSettings[exch][a][cp].InitialFunds
}

// GetLatestHoldingsForAllCurrencies will return the current holdings for all loaded currencies
// this is useful to assess the position of your entire portfolio in order to help with risk decisions
func (p *Portfolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	var resp []holdings.Holding
	for _, x := range p.ExchangeAssetPairSettings {
		for _, y := range x {
			for _, z := range y {
				resp = append(resp, z.HoldingsSnapshots.GetLatestSnapshot())
			}
		}
	}
	return resp
}

func (p *Portfolio) SetHoldings(exch string, a asset.Item, cp currency.Pair, t time.Time, pos holdings.Holding, force bool) error {
	lookup := p.ExchangeAssetPairSettings[exch][a][cp]
	found := false
	for i := range lookup.HoldingsSnapshots.Holdings {
		if lookup.HoldingsSnapshots.Holdings[i].Timestamp.Equal(t) {
			found = true
		}
	}
	if !found {
		lookup.HoldingsSnapshots.Holdings = append(lookup.HoldingsSnapshots.Holdings, pos)
		p.ExchangeAssetPairSettings[exch][a][cp] = lookup
	}
	return nil
}

func (p *Portfolio) SetupExchangeAssetPairMap(exch string, a asset.Item, cp currency.Pair) *ExchangeAssetPairSettings {
	if p.ExchangeAssetPairSettings == nil {
		p.ExchangeAssetPairSettings = make(map[string]map[asset.Item]map[currency.Pair]*ExchangeAssetPairSettings)
	}
	if p.ExchangeAssetPairSettings[exch] == nil {
		p.ExchangeAssetPairSettings[exch] = make(map[asset.Item]map[currency.Pair]*ExchangeAssetPairSettings)
	}
	if p.ExchangeAssetPairSettings[exch][a] == nil {
		p.ExchangeAssetPairSettings[exch][a] = make(map[currency.Pair]*ExchangeAssetPairSettings)
	}
	if _, ok := p.ExchangeAssetPairSettings[exch][a][cp]; !ok {
		p.ExchangeAssetPairSettings[exch][a][cp] = &ExchangeAssetPairSettings{}
	}

	return p.ExchangeAssetPairSettings[exch][a][cp]
}

func (e *ExchangeAssetPairSettings) GetLatestHoldings() holdings.Holding {
	if e.HoldingsSnapshots.Holdings == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	sort.SliceStable(e.HoldingsSnapshots.Holdings, func(i, j int) bool {
		return e.HoldingsSnapshots.Holdings[i].Timestamp.Before(e.HoldingsSnapshots.Holdings[j].Timestamp)
	})

	return e.HoldingsSnapshots.Holdings[len(e.HoldingsSnapshots.Holdings)-1]
}

func (e *ExchangeAssetPairSettings) SetInitialFunds(initial float64) {
	e.InitialFunds = initial
}

func (e *ExchangeAssetPairSettings) GetInitialFunds() float64 {
	return e.InitialFunds
}

func (e *ExchangeAssetPairSettings) Value() float64 {
	latest := e.GetLatestHoldings()
	return latest.TotalValue
}

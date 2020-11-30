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
	if signal.GetDirection() == "" {
		return &order.Order{}, errors.New("invalid Direction")
	}

	exchangeAssetPairHoldings := p.ViewHoldingAtTimePeriod(
		signal.GetExchange(),
		signal.GetAssetType(),
		signal.Pair(),
		signal.GetTime().Add(-signal.GetInterval().Duration()))
	lookup := p.ExchangeAssetPairSettings[signal.GetExchange()][signal.GetAssetType()][signal.Pair()]
	currFunds := lookup.GetFunds()

	o := &order.Order{
		Event: event.Event{
			Exchange:     signal.GetExchange(),
			Time:         signal.GetTime(),
			CurrencyPair: signal.Pair(),
			AssetType:    signal.GetAssetType(),
			Interval:     signal.GetInterval(),
		},
		Direction: signal.GetDirection(),
		Why:       signal.GetWhy(),
	}
	if signal.GetDirection() == common.DoNothing {
		return o, nil
	}

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && exchangeAssetPairHoldings.RemainingFunds <= signal.GetAmount() {
		o.SetWhy("no holdings to sell. " + signal.GetWhy())
		o.Direction = common.DoNothing
		return o, nil
	}

	if (signal.GetDirection() == gctorder.Buy || signal.GetDirection() == gctorder.Bid) && currFunds <= 0 {
		o.SetWhy("not enough funds to buy. " + signal.GetWhy())
		o.Direction = common.DoNothing
		return o, nil
	}

	o.Price = signal.GetPrice()
	o.Amount = signal.GetAmount()
	o.OrderType = gctorder.Market
	latest := data.Latest()
	sizedOrder, err := p.SizeManager.SizeOrder(
		o,
		latest,
		currFunds,
		cs,
	)
	if err != nil {
		o.SetWhy(err.Error() + ". " + signal.GetWhy())
		o.Direction = common.DoNothing
		return o, nil
	}

	var eo *order.Order
	eo, err = p.RiskManager.EvaluateOrder(sizedOrder, latest, exchangeAssetPairHoldings)
	if err != nil {
		o.SetWhy(err.Error() + ". " + signal.GetWhy())
		o.Direction = common.DoNothing
		return o, nil
	}

	return eo, nil
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions
func (p *Portfolio) OnFill(fillEvent fill.FillEvent, _ data.Handler) (*fill.Fill, error) {
	lookup := p.ExchangeAssetPairSettings[fillEvent.GetExchange()][fillEvent.GetAssetType()][fillEvent.Pair()]
	// Get the holding from the previous iteration, create it if it doesn't yet have a timestamp
	h := p.ViewHoldingAtTimePeriod(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime().Add(-fillEvent.GetInterval().Duration()))
	if !h.Timestamp.IsZero() {
		h.Update(fillEvent)
	} else {
		h = holdings.Create(fillEvent, lookup.InitialFunds)
	}
	err := p.SetHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime(), h, true)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	err = p.addComplianceSnapshot(fillEvent)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	switch fillEvent.GetDirection() {
	case common.DoNothing:
		fe := fillEvent.(*fill.Fill)
		fe.ExchangeFee = 0
		return fe, nil
	case gctorder.Buy, gctorder.Bid:
		lookup.Funds -= fillEvent.NetValue()
	case gctorder.Sell, gctorder.Ask:
		lookup.Funds += fillEvent.NetValue()
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
	err = complianceManager.AddSnapshot(prevSnap.Orders, fillEvent.GetTime(), false)
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
	holdings := p.ExchangeAssetPairSettings[exchangeName][a][cp].GetLatestHoldings()
	if ok && (holdings.PositionsSize > 0) {
		return holdings, true
	}
	return holdings, false
}

func (p *Portfolio) Update(d interfaces.DataEventHandler) {
	if holdings, ok := p.IsInvested(d.GetExchange(), d.GetAssetType(), d.Pair()); ok {
		holdings.UpdateValue(d)
		err := p.SetHoldings(d.GetExchange(), d.GetAssetType(), d.Pair(), d.GetTime(), holdings, true)
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}
}

// ViewHoldingAtTimePeriod retrieves a snapshot of holdings at a specific time period,
// returning empty when not found
func (p *Portfolio) ViewHoldingAtTimePeriod(exch string, a asset.Item, cp currency.Pair, t time.Time) holdings.Holding {
	exchangeAssetPairSettings := p.ExchangeAssetPairSettings[exch][a][cp]
	for i := range exchangeAssetPairSettings.HoldingsSnapshots.Hodlings {
		if t.Equal(exchangeAssetPairSettings.HoldingsSnapshots.Hodlings[i].Timestamp) {
			return exchangeAssetPairSettings.HoldingsSnapshots.Hodlings[i]
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

func (p *Portfolio) SetFunds(exch string, a asset.Item, cp currency.Pair, funds float64) {
	p.ExchangeAssetPairSettings[exch][a][cp].Funds = funds
}

func (p *Portfolio) GetFunds(exch string, a asset.Item, cp currency.Pair) float64 {
	return p.ExchangeAssetPairSettings[exch][a][cp].Funds
}

func (p *Portfolio) SetHoldings(exch string, a asset.Item, cp currency.Pair, t time.Time, pos holdings.Holding, force bool) error {
	lookup := p.ExchangeAssetPairSettings[exch][a][cp]
	found := false
	for i := range lookup.HoldingsSnapshots.Hodlings {
		if lookup.HoldingsSnapshots.Hodlings[i].Timestamp.Equal(t) {
			found = true
		}
	}
	if !found {
		lookup.HoldingsSnapshots.Hodlings = append(lookup.HoldingsSnapshots.Hodlings, pos)
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
	if e.HoldingsSnapshots.Hodlings == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	sort.SliceStable(e.HoldingsSnapshots.Hodlings, func(i, j int) bool {
		return e.HoldingsSnapshots.Hodlings[i].Timestamp.Before(e.HoldingsSnapshots.Hodlings[j].Timestamp)
	})

	return e.HoldingsSnapshots.Hodlings[len(e.HoldingsSnapshots.Hodlings)-1]
}

func (e *ExchangeAssetPairSettings) SetInitialFunds(initial float64) {
	e.InitialFunds = initial
}

func (e *ExchangeAssetPairSettings) GetInitialFunds() float64 {
	return e.InitialFunds
}

func (e *ExchangeAssetPairSettings) SetFunds(funds float64) {
	e.Funds = funds
}

func (e *ExchangeAssetPairSettings) GetFunds() float64 {
	return e.Funds
}

func (e *ExchangeAssetPairSettings) Value() float64 {
	latest := e.GetLatestHoldings()
	return latest.TotalValue
}

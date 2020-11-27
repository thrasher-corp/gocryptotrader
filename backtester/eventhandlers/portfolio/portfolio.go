package portfolio

import (
	"errors"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics/position"
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
func (p *Portfolio) OnSignal(signal signal.SignalEvent, data interfaces.DataHandler, cs *exchange.CurrencySettings) (*order.Order, error) {
	if signal.GetDirection() == "" {
		return &order.Order{}, errors.New("invalid Direction")
	}

	exchangeAssetPairHoldings, err := p.ViewHoldings(
		signal.GetExchange(),
		signal.GetAssetType(),
		signal.Pair(),
		signal.GetTime().Add(-signal.GetInterval().Duration()))
	if err != nil {
		log.Error(log.BackTester, err)
	}
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

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && exchangeAssetPairHoldings.Amount <= signal.GetAmount() {
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

	eo, err := p.RiskManager.EvaluateOrder(sizedOrder, latest, exchangeAssetPairHoldings)
	if err != nil {
		o.SetWhy(err.Error() + ". " + signal.GetWhy())
		o.Direction = common.DoNothing
		return o, nil
	}

	return eo, nil
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions
func (p *Portfolio) OnFill(fillEvent fill.FillEvent, _ interfaces.DataHandler) (*fill.Fill, error) {
	lookup := p.ExchangeAssetPairSettings[fillEvent.GetExchange()][fillEvent.GetAssetType()][fillEvent.Pair()]
	holdings, err := p.ViewHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime().Add(-fillEvent.GetInterval().Duration()))
	if err != nil {
		return nil, err
	}
	if !holdings.Timestamp.IsZero() {
		holdings.Update(fillEvent)
	} else {
		holdings = position.Position{}
		holdings.Amount = lookup.InitialFunds
		holdings.Create(fillEvent)
	}
	err = p.SetHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), fillEvent.GetTime(), holdings, true)
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

func (p *Portfolio) IsInvested(exchangeName string, a asset.Item, cp currency.Pair) (pos position.Position, ok bool) {
	holdings := p.ExchangeAssetPairSettings[exchangeName][a][cp].GetLatestHoldings()
	if ok && (holdings.Amount != 0) {
		return holdings, true
	}
	return holdings, false
}

func (p *Portfolio) Update(d interfaces.DataEventHandler) {
	if pos, ok := p.IsInvested(d.GetExchange(), d.GetAssetType(), d.Pair()); ok {
		pos.UpdateValue(d)
		err := p.SetHoldings(d.GetExchange(), d.GetAssetType(), d.Pair(), d.GetTime(), pos, false)
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}
}

func (p *Portfolio) ViewHoldings(exch string, a asset.Item, cp currency.Pair, t time.Time) (position.Position, error) {
	exchangeAssetPairSettings := p.ExchangeAssetPairSettings[exch][a][cp]
	for i := range exchangeAssetPairSettings.PositionSnapshots.Positions {
		if t.Equal(exchangeAssetPairSettings.PositionSnapshots.Positions[i].Timestamp) {
			return exchangeAssetPairSettings.PositionSnapshots.Positions[i], nil
		}
	}

	return position.Position{}, nil
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

func (p *Portfolio) SetHoldings(exch string, a asset.Item, cp currency.Pair, t time.Time, pos position.Position, force bool) error {
	lookup := p.ExchangeAssetPairSettings[exch][a][cp]
	found := false
	for i := range lookup.PositionSnapshots.Positions {
		if lookup.PositionSnapshots.Positions[i].Timestamp.Equal(t) {
			found = true
		}
	}
	if !found {
		lookup.PositionSnapshots.Positions = append(lookup.PositionSnapshots.Positions, pos)
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

func (e *ExchangeAssetPairSettings) GetLatestHoldings() position.Position {
	if e.PositionSnapshots.Positions == nil {
		// no holdings yet
		return position.Position{}
	}
	sort.SliceStable(e.PositionSnapshots.Positions, func(i, j int) bool {
		return e.PositionSnapshots.Positions[i].Timestamp.Before(e.PositionSnapshots.Positions[j].Timestamp)
	})

	return e.PositionSnapshots.Positions[len(e.PositionSnapshots.Positions)-1]
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
	return latest.Value
}

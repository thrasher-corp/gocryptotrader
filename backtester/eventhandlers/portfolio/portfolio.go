package portfolio

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/settings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup creates a portfolio manager instance and sets private fields
func Setup(sh SizeHandler, r risk.Handler, riskFreeRate float64) (*Portfolio, error) {
	if sh == nil {
		return nil, errSizeManagerUnset
	}
	if riskFreeRate < 0 {
		return nil, errNegativeRiskFreeRate
	}
	if r == nil {
		return nil, errRiskManagerUnset
	}
	p := &Portfolio{}
	p.sizeManager = sh
	p.riskManager = r
	p.riskFreeRate = riskFreeRate

	return p, nil
}

// Reset returns the portfolio manager to its default state
func (p *Portfolio) Reset() {
	p.exchangeAssetPairSettings = nil
}

// OnSignal receives the event from the strategy on whether it has signalled to buy, do nothing or sell
// on buy/sell, the portfolio manager will size the order and assess the risk of the order
// if successful, it will pass on an order.Order to be used by the exchange event handler to place an order based on
// the portfolio manager's recommendations
func (p *Portfolio) OnSignal(signal signal.Event, cs *exchange.Settings) (*order.Order, error) {
	if signal == nil || cs == nil {
		return nil, common.ErrNilArguments
	}
	if p.sizeManager == nil {
		return nil, errSizeManagerUnset
	}
	if p.riskManager == nil {
		return nil, errRiskManagerUnset
	}

	o := &order.Order{
		Base: event.Base{
			Offset:       signal.GetOffset(),
			Exchange:     signal.GetExchange(),
			Time:         signal.GetTime(),
			CurrencyPair: signal.Pair(),
			AssetType:    signal.GetAssetType(),
			Interval:     signal.GetInterval(),
			Reason:       signal.GetReason(),
		},
		Direction: signal.GetDirection(),
	}
	if signal.GetDirection() == "" {
		return o, errInvalidDirection
	}

	lookup := p.exchangeAssetPairSettings[signal.GetExchange()][signal.GetAssetType()][signal.Pair()]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v",
			errNoPortfolioSettings,
			signal.GetExchange(),
			signal.GetAssetType(),
			signal.Pair())
	}
	prevHolding := lookup.GetLatestHoldings()
	if p.iteration == 0 {
		prevHolding.InitialFunds = lookup.InitialFunds
		prevHolding.RemainingFunds = lookup.InitialFunds
		prevHolding.Exchange = signal.GetExchange()
		prevHolding.Pair = signal.Pair()
		prevHolding.Asset = signal.GetAssetType()
		prevHolding.Timestamp = signal.GetTime()
	}
	p.iteration++

	if signal.GetDirection() == common.DoNothing || signal.GetDirection() == common.MissingData || signal.GetDirection() == "" {
		return o, nil
	}

	if signal.GetDirection() == gctorder.Sell && prevHolding.PositionsSize == 0 {
		o.AppendReason("no holdings to sell")
		o.SetDirection(common.CouldNotSell)
		signal.SetDirection(o.Direction)
		return o, nil
	}

	// for simplicity, the backtester will round to 8 decimal places
	remainingFundsRounded := math.Floor(prevHolding.RemainingFunds*100000000) / 100000000
	if signal.GetDirection() == gctorder.Buy && remainingFundsRounded <= 0 {
		o.AppendReason("not enough funds to buy")
		o.SetDirection(common.CouldNotBuy)
		signal.SetDirection(o.Direction)
		return o, nil
	}

	o.Price = signal.GetPrice()
	o.OrderType = gctorder.Market
	o.BuyLimit = signal.GetBuyLimit()
	o.SellLimit = signal.GetSellLimit()
	sizingFunds := prevHolding.RemainingFunds
	if signal.GetDirection() == gctorder.Sell {
		sizingFunds = prevHolding.PositionsSize
	}

	sizedOrder := p.sizeOrder(signal, cs, o, sizingFunds)
	o.Funds = sizingFunds
	sizedAmountRounded := math.Floor(sizedOrder.Amount*100000000) / 100000000
	if sizedAmountRounded <= 0 {
		o.AppendReason("sized amount is zero")
		if o.Direction == gctorder.Buy {
			o.SetDirection(common.CouldNotBuy)
		} else if o.Direction == gctorder.Sell {
			o.SetDirection(common.CouldNotSell)
		}
		return o, nil
	}

	return p.evaluateOrder(signal, o, sizedOrder)
}

func (p *Portfolio) evaluateOrder(d common.Directioner, originalOrderSignal, sizedOrder *order.Order) (*order.Order, error) {
	var evaluatedOrder *order.Order
	cm, err := p.GetComplianceManager(originalOrderSignal.GetExchange(), originalOrderSignal.GetAssetType(), originalOrderSignal.Pair())
	if err != nil {
		return nil, err
	}

	evaluatedOrder, err = p.riskManager.EvaluateOrder(sizedOrder, p.GetLatestHoldingsForAllCurrencies(), cm.GetLatestSnapshot())
	if err != nil {
		originalOrderSignal.AppendReason(err.Error())
		switch d.GetDirection() {
		case gctorder.Buy:
			originalOrderSignal.Direction = common.CouldNotBuy
		case gctorder.Sell:
			originalOrderSignal.Direction = common.CouldNotSell
		case common.CouldNotBuy, common.CouldNotSell:
		default:
			originalOrderSignal.Direction = common.DoNothing
		}
		d.SetDirection(originalOrderSignal.Direction)
		return originalOrderSignal, nil
	}

	return evaluatedOrder, nil
}

func (p *Portfolio) sizeOrder(d common.Directioner, cs *exchange.Settings, originalOrderSignal *order.Order, sizingFunds float64) *order.Order {
	sizedOrder, err := p.sizeManager.SizeOrder(originalOrderSignal, sizingFunds, cs)
	if err != nil {
		originalOrderSignal.AppendReason(err.Error())
		switch originalOrderSignal.Direction {
		case gctorder.Buy:
			originalOrderSignal.Direction = common.CouldNotBuy
		case gctorder.Sell:
			originalOrderSignal.Direction = common.CouldNotSell
		default:
			originalOrderSignal.Direction = common.DoNothing
		}
		d.SetDirection(originalOrderSignal.Direction)
		return originalOrderSignal
	}

	if sizedOrder.Amount == 0 {
		switch originalOrderSignal.Direction {
		case gctorder.Buy:
			originalOrderSignal.Direction = common.CouldNotBuy
		case gctorder.Sell:
			originalOrderSignal.Direction = common.CouldNotSell
		default:
			originalOrderSignal.Direction = common.DoNothing
		}
		d.SetDirection(originalOrderSignal.Direction)
		originalOrderSignal.AppendReason("sized order to 0")
	}

	return sizedOrder
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions.
func (p *Portfolio) OnFill(fillEvent fill.Event) (*fill.Fill, error) {
	if fillEvent == nil {
		return nil, common.ErrNilEvent
	}
	lookup := p.exchangeAssetPairSettings[fillEvent.GetExchange()][fillEvent.GetAssetType()][fillEvent.Pair()]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair())
	}
	var err error
	// Get the holding from the previous iteration, create it if it doesn't yet have a timestamp
	h := lookup.GetHoldingsForTime(fillEvent.GetTime().Add(-fillEvent.GetInterval().Duration()))
	if !h.Timestamp.IsZero() {
		h.Update(fillEvent)
	} else {
		h = lookup.GetLatestHoldings()
		if !h.Timestamp.IsZero() {
			h.Update(fillEvent)
		} else {
			h, err = holdings.Create(fillEvent, lookup.InitialFunds, p.riskFreeRate)
			if err != nil {
				return nil, err
			}
		}
	}
	err = p.setHoldingsForOffset(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), &h, true)
	if errors.Is(err, errNoHoldings) {
		err = p.setHoldingsForOffset(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), &h, false)
	}
	if err != nil {
		log.Error(log.BackTester, err)
	}

	err = p.addComplianceSnapshot(fillEvent)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	direction := fillEvent.GetDirection()
	if direction == common.DoNothing ||
		direction == common.CouldNotBuy ||
		direction == common.CouldNotSell ||
		direction == common.MissingData ||
		direction == "" {
		fe := fillEvent.(*fill.Fill)
		fe.ExchangeFee = 0
		return fe, nil
	}

	return fillEvent.(*fill.Fill), nil
}

// addComplianceSnapshot gets the previous snapshot of compliance events, updates with the latest fillevent
// then saves the snapshot to the c
func (p *Portfolio) addComplianceSnapshot(fillEvent fill.Event) error {
	if fillEvent == nil {
		return common.ErrNilEvent
	}
	complianceManager, err := p.GetComplianceManager(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair())
	if err != nil {
		return err
	}
	prevSnap := complianceManager.GetLatestSnapshot()
	fo := fillEvent.GetOrder()
	if fo != nil {
		snapOrder := compliance.SnapshotOrder{
			ClosePrice:          fillEvent.GetClosePrice(),
			VolumeAdjustedPrice: fillEvent.GetVolumeAdjustedPrice(),
			SlippageRate:        fillEvent.GetSlippageRate(),
			Detail:              fo,
			CostBasis:           (fo.Price * fo.Amount) + fo.Fee,
		}
		prevSnap.Orders = append(prevSnap.Orders, snapOrder)
	}
	return complianceManager.AddSnapshot(prevSnap.Orders, fillEvent.GetTime(), fillEvent.GetOffset(), false)
}

// GetComplianceManager returns the order snapshots for a given exchange, asset, pair
func (p *Portfolio) GetComplianceManager(exchangeName string, a asset.Item, cp currency.Pair) (*compliance.Manager, error) {
	lookup := p.exchangeAssetPairSettings[exchangeName][a][cp]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v could not retrieve compliance manager", errNoPortfolioSettings, exchangeName, a, cp)
	}
	return &lookup.ComplianceManager, nil
}

// SetFee sets the fee rate
func (p *Portfolio) SetFee(exch string, a asset.Item, cp currency.Pair, fee float64) {
	lookup := p.exchangeAssetPairSettings[exch][a][cp]
	lookup.Fee = fee
}

// GetFee can panic for bad requests, but why are you getting things that don't exist?
func (p *Portfolio) GetFee(exchangeName string, a asset.Item, cp currency.Pair) float64 {
	if p.exchangeAssetPairSettings == nil {
		return 0
	}
	lookup := p.exchangeAssetPairSettings[exchangeName][a][cp]
	if lookup == nil {
		return 0
	}
	return lookup.Fee
}

// IsInvested determines if there are any holdings for a given exchange, asset, pair
func (p *Portfolio) IsInvested(exchangeName string, a asset.Item, cp currency.Pair) (holdings.Holding, bool) {
	s := p.exchangeAssetPairSettings[exchangeName][a][cp]
	if s == nil {
		return holdings.Holding{}, false
	}
	h := s.GetLatestHoldings()
	if h.PositionsSize > 0 {
		return h, true
	}
	return h, false
}

// Update updates the portfolio holdings for the data event
func (p *Portfolio) Update(d common.DataEventHandler) error {
	if d == nil {
		return common.ErrNilEvent
	}
	h, ok := p.IsInvested(d.GetExchange(), d.GetAssetType(), d.Pair())
	if !ok {
		return nil
	}
	h.UpdateValue(d)
	err := p.setHoldingsForOffset(d.GetExchange(), d.GetAssetType(), d.Pair(), &h, true)
	if errors.Is(err, errNoHoldings) {
		err = p.setHoldingsForOffset(d.GetExchange(), d.GetAssetType(), d.Pair(), &h, false)
	}
	return err
}

// SetInitialFunds sets the initial funds
func (p *Portfolio) SetInitialFunds(exch string, a asset.Item, cp currency.Pair, funds float64) error {
	lookup, ok := p.exchangeAssetPairSettings[exch][a][cp]
	if !ok {
		var err error
		lookup, err = p.SetupCurrencySettingsMap(exch, a, cp)
		if err != nil {
			return err
		}
	}
	lookup.InitialFunds = funds

	return nil
}

// GetInitialFunds returns the initial funds
func (p *Portfolio) GetInitialFunds(exch string, a asset.Item, cp currency.Pair) float64 {
	lookup, ok := p.exchangeAssetPairSettings[exch][a][cp]
	if !ok {
		return 0
	}
	return lookup.InitialFunds
}

// GetLatestHoldingsForAllCurrencies will return the current holdings for all loaded currencies
// this is useful to assess the position of your entire portfolio in order to help with risk decisions
func (p *Portfolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	var resp []holdings.Holding
	for _, x := range p.exchangeAssetPairSettings {
		for _, y := range x {
			for _, z := range y {
				holds := z.GetLatestHoldings()
				if holds.Offset != 0 {
					resp = append(resp, holds)
				}
			}
		}
	}
	return resp
}

func (p *Portfolio) setHoldingsForOffset(exch string, a asset.Item, cp currency.Pair, h *holdings.Holding, overwriteExisting bool) error {
	if h.Timestamp.IsZero() {
		return errHoldingsNoTimestamp
	}
	lookup := p.exchangeAssetPairSettings[exch][a][cp]
	if lookup == nil {
		var err error
		lookup, err = p.SetupCurrencySettingsMap(exch, a, cp)
		if err != nil {
			return err
		}
	}
	if overwriteExisting && len(lookup.HoldingsSnapshots) == 0 {
		return errNoHoldings
	}
	for i := len(lookup.HoldingsSnapshots) - 1; i >= 0; i-- {
		if lookup.HoldingsSnapshots[i].Offset == h.Offset {
			if overwriteExisting {
				lookup.HoldingsSnapshots[i] = *h
				return nil
			}
			return errHoldingsAlreadySet
		}
	}
	if overwriteExisting {
		return fmt.Errorf("%w at %v", errNoHoldings, h.Timestamp)
	}

	lookup.HoldingsSnapshots = append(lookup.HoldingsSnapshots, *h)
	return nil
}

// ViewHoldingAtTimePeriod retrieves a snapshot of holdings at a specific time period,
// returning empty when not found
func (p *Portfolio) ViewHoldingAtTimePeriod(exch string, a asset.Item, cp currency.Pair, t time.Time) (holdings.Holding, error) {
	exchangeAssetPairSettings := p.exchangeAssetPairSettings[exch][a][cp]
	if exchangeAssetPairSettings == nil {
		return holdings.Holding{}, fmt.Errorf("%w for %v %v %v", errNoHoldings, exch, a, cp)
	}

	for i := len(exchangeAssetPairSettings.HoldingsSnapshots) - 1; i >= 0; i-- {
		if t.Equal(exchangeAssetPairSettings.HoldingsSnapshots[i].Timestamp) {
			return exchangeAssetPairSettings.HoldingsSnapshots[i], nil
		}
	}

	return holdings.Holding{}, fmt.Errorf("%w for %v %v %v at %v", errNoHoldings, exch, a, cp, t)
}

// SetupCurrencySettingsMap ensures a map is created and no panics happen
func (p *Portfolio) SetupCurrencySettingsMap(exch string, a asset.Item, cp currency.Pair) (*settings.Settings, error) {
	if exch == "" {
		return nil, errExchangeUnset
	}
	if a == "" {
		return nil, errAssetUnset
	}
	if cp.IsEmpty() {
		return nil, errCurrencyPairUnset
	}
	if p.exchangeAssetPairSettings == nil {
		p.exchangeAssetPairSettings = make(map[string]map[asset.Item]map[currency.Pair]*settings.Settings)
	}
	if p.exchangeAssetPairSettings[exch] == nil {
		p.exchangeAssetPairSettings[exch] = make(map[asset.Item]map[currency.Pair]*settings.Settings)
	}
	if p.exchangeAssetPairSettings[exch][a] == nil {
		p.exchangeAssetPairSettings[exch][a] = make(map[currency.Pair]*settings.Settings)
	}
	if _, ok := p.exchangeAssetPairSettings[exch][a][cp]; !ok {
		p.exchangeAssetPairSettings[exch][a][cp] = &settings.Settings{}
	}

	return p.exchangeAssetPairSettings[exch][a][cp], nil
}

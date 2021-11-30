package portfolio

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup creates a portfolio manager instance and sets private fields
func Setup(sh SizeHandler, r risk.Handler, riskFreeRate decimal.Decimal) (*Portfolio, error) {
	if sh == nil {
		return nil, errSizeManagerUnset
	}
	if riskFreeRate.IsNegative() {
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
func (p *Portfolio) OnSignal(ev signal.Event, cs *exchange.Settings, funds funding.IFundReserver) (*order.Order, error) {
	if ev == nil || cs == nil {
		return nil, common.ErrNilArguments
	}
	if p.sizeManager == nil {
		return nil, errSizeManagerUnset
	}
	if p.riskManager == nil {
		return nil, errRiskManagerUnset
	}
	if funds == nil {
		return nil, funding.ErrFundsNotFound
	}

	o := &order.Order{
		Base: event.Base{
			Offset:       ev.GetOffset(),
			Exchange:     ev.GetExchange(),
			Time:         ev.GetTime(),
			CurrencyPair: ev.Pair(),
			AssetType:    ev.GetAssetType(),
			Interval:     ev.GetInterval(),
			Reason:       ev.GetReason(),
		},
		Direction:     ev.GetDirection(),
		LinkedOrderID: ev.GetLinkedOrderID(),
	}
	if ev.GetDirection() == "" {
		return o, errInvalidDirection
	}

	lookup := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v",
			errNoPortfolioSettings,
			ev.GetExchange(),
			ev.GetAssetType(),
			ev.Pair())
	}

	if ev.GetDirection() == common.DoNothing ||
		ev.GetDirection() == common.MissingData ||
		ev.GetDirection() == common.TransferredFunds ||
		ev.GetDirection() == "" {
		return o, nil
	}

	dir := ev.GetDirection()
	if !funds.CanPlaceOrder(dir) {
		o.AppendReason(notEnoughFundsTo + " " + dir.Lower())
		switch ev.GetDirection() {
		case gctorder.Sell:
			o.SetDirection(common.CouldNotSell)
		case gctorder.Buy:
			o.SetDirection(common.CouldNotBuy)
		case gctorder.Short:
			o.SetDirection(common.CouldNotShort)
		case gctorder.Long:
			o.SetDirection(common.CouldNotLong)
		}
		ev.SetDirection(o.Direction)
		return o, nil
	}

	o.Price = ev.GetPrice()
	o.OrderType = gctorder.Market
	o.BuyLimit = ev.GetBuyLimit()
	o.SellLimit = ev.GetSellLimit()
	var sizingFunds decimal.Decimal
	if ev.GetAssetType() == asset.Spot {
		pReader, err := funds.GetPairReader()
		if err != nil {
			return nil, err
		}
		if ev.GetDirection() == gctorder.Sell {
			sizingFunds = pReader.BaseAvailable()
		} else {
			sizingFunds = pReader.QuoteAvailable()
		}
	} else if ev.GetAssetType() == asset.Futures {
		cReader, err := funds.GetCollateralReader()
		if err != nil {
			return nil, err
		}
		sizingFunds = cReader.AvailableFunds()
	}
	sizedOrder := p.sizeOrder(ev, cs, o, sizingFunds, funds)

	return p.evaluateOrder(ev, o, sizedOrder)
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

func (p *Portfolio) sizeOrder(d common.Directioner, cs *exchange.Settings, originalOrderSignal *order.Order, sizingFunds decimal.Decimal, funds funding.IFundReserver) *order.Order {
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

	if sizedOrder.Amount.IsZero() {
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
	if d.GetDirection() == gctorder.Sell {
		err = funds.Reserve(sizedOrder.Amount, gctorder.Sell)
		sizedOrder.AllocatedFunds = sizedOrder.Amount
	} else {
		err = funds.Reserve(sizedOrder.Amount.Mul(sizedOrder.Price), gctorder.Buy)
		sizedOrder.AllocatedFunds = sizedOrder.Amount.Mul(sizedOrder.Price)
	}
	if err != nil {
		sizedOrder.Direction = common.DoNothing
		sizedOrder.AppendReason(err.Error())
	}
	return sizedOrder
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions.
func (p *Portfolio) OnFill(ev fill.Event, funding funding.IFundReleaser) (*fill.Fill, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	lookup := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	}
	var err error

	if ev.GetLinkedOrderID() != "" {
		// this means we're closing an order
		snap := lookup.ComplianceManager.GetLatestSnapshot()
		for i := range snap.Orders {
			if ev.GetLinkedOrderID() == snap.Orders[i].FuturesOrder.OpeningPosition.ID {
				snap.Orders[i].FuturesOrder.ClosingPosition = ev.GetOrder()
				snap.Orders[i].FuturesOrder.RealisedPNL = snap.Orders[i].FuturesOrder.UnrealisedPNL

			}
		}
	}

	if ev.GetAssetType() == asset.Spot {
		fp, err := funding.GetPairReleaser()
		if err != nil {
			return nil, err
		}
		// Get the holding from the previous iteration, create it if it doesn't yet have a timestamp
		h := lookup.GetHoldingsForTime(ev.GetTime().Add(-ev.GetInterval().Duration()))
		if !h.Timestamp.IsZero() {
			h.Update(ev, fp)
		} else {
			h = lookup.GetLatestHoldings()
			if h.Timestamp.IsZero() {
				h, err = holdings.Create(ev, funding)
				if err != nil {
					return nil, err
				}
			} else {
				h.Update(ev, fp)
			}
		}
		err = p.setHoldingsForOffset(&h, true)
		if errors.Is(err, errNoHoldings) {
			err = p.setHoldingsForOffset(&h, false)
		}
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}

	err = p.addComplianceSnapshot(ev)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	fe, ok := ev.(*fill.Fill)
	if !ok {
		return nil, fmt.Errorf("%w expected fill event", common.ErrInvalidDataType)
	}

	direction := ev.GetDirection()
	if direction == common.DoNothing ||
		direction == common.CouldNotBuy ||
		direction == common.CouldNotSell ||
		direction == common.MissingData ||
		direction == common.CouldNotCloseLong ||
		direction == common.CouldNotCloseShort ||
		direction == common.CouldNotLong ||
		direction == common.CouldNotShort ||
		direction == "" {
		fe.ExchangeFee = decimal.Zero
		return fe, nil
	}

	return fe, nil
}

// GetLatestOrderSnapshotForEvent gets orders related to the event
func (p *Portfolio) GetLatestOrderSnapshotForEvent(e common.EventHandler) (compliance.Snapshot, error) {
	eapSettings, ok := p.exchangeAssetPairSettings[e.GetExchange()][e.GetAssetType()][e.Pair()]
	if !ok {
		return compliance.Snapshot{}, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	return eapSettings.ComplianceManager.GetLatestSnapshot(), nil
}

// GetLatestOrderSnapshots returns the latest snapshots from all stored pair data
func (p *Portfolio) GetLatestOrderSnapshots() ([]compliance.Snapshot, error) {
	var resp []compliance.Snapshot
	for _, exchangeMap := range p.exchangeAssetPairSettings {
		for _, assetMap := range exchangeMap {
			for _, pairMap := range assetMap {
				resp = append(resp, pairMap.ComplianceManager.GetLatestSnapshot())
			}
		}
	}
	if len(resp) == 0 {
		return nil, errNoPortfolioSettings
	}
	return resp, nil
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
	if fo := fillEvent.GetOrder(); fo != nil {
		price := decimal.NewFromFloat(fo.Price)
		amount := decimal.NewFromFloat(fo.Amount)
		fee := decimal.NewFromFloat(fo.Fee)
		snapOrder := compliance.SnapshotOrder{
			ClosePrice:          fillEvent.GetClosePrice(),
			VolumeAdjustedPrice: fillEvent.GetVolumeAdjustedPrice(),
			SlippageRate:        fillEvent.GetSlippageRate(),
			CostBasis:           price.Mul(amount).Add(fee),
		}
		if fo.AssetType == asset.Spot {
			snapOrder.SpotOrder = fo
			prevSnap.Orders = append(prevSnap.Orders, snapOrder)
		} else if fo.AssetType == asset.Futures {
			var linked bool
			for i := range prevSnap.Orders {
				if prevSnap.Orders[i].FuturesOrder != nil &&
					prevSnap.Orders[i].FuturesOrder.OpeningPosition != nil &&
					prevSnap.Orders[i].FuturesOrder.OpeningPosition.ID == fillEvent.GetLinkedOrderID() {
					prevSnap.Orders[i].FuturesOrder.ClosingPosition = fo
					linked = true
				}
			}
			if !linked {
				snapOrder.FuturesOrder = &gctorder.Futures{
					Side:            fillEvent.GetDirection(),
					OpeningPosition: fo,
				}
				prevSnap.Orders = append(prevSnap.Orders, snapOrder)
			}
		}
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
func (p *Portfolio) SetFee(exch string, a asset.Item, cp currency.Pair, fee decimal.Decimal) {
	lookup := p.exchangeAssetPairSettings[exch][a][cp]
	lookup.Fee = fee
}

// GetFee can panic for bad requests, but why are you getting things that don't exist?
func (p *Portfolio) GetFee(exchangeName string, a asset.Item, cp currency.Pair) decimal.Decimal {
	if p.exchangeAssetPairSettings == nil {
		return decimal.Zero
	}
	lookup := p.exchangeAssetPairSettings[exchangeName][a][cp]
	if lookup == nil {
		return decimal.Zero
	}
	return lookup.Fee
}

// UpdateHoldings updates the portfolio holdings for the data event
func (p *Portfolio) UpdateHoldings(ev common.DataEventHandler, funds funding.IFundReleaser) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if funds == nil {
		return funding.ErrFundsNotFound
	}
	lookup, ok := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if !ok {
		return fmt.Errorf("%w for %v %v %v",
			errNoPortfolioSettings,
			ev.GetExchange(),
			ev.GetAssetType(),
			ev.Pair())
	}
	h := lookup.GetLatestHoldings()
	var err error
	if h.Timestamp.IsZero() {
		h, err = holdings.Create(ev, funds)
		if err != nil {
			return err
		}
	}
	h.UpdateValue(ev)
	err = p.setHoldingsForOffset(&h, true)
	if errors.Is(err, errNoHoldings) {
		err = p.setHoldingsForOffset(&h, false)
	}
	return err
}

// GetLatestHoldingsForAllCurrencies will return the current holdings for all loaded currencies
// this is useful to assess the position of your entire portfolio in order to help with risk decisions
func (p *Portfolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	var resp []holdings.Holding
	for _, x := range p.exchangeAssetPairSettings {
		for _, y := range x {
			for _, z := range y {
				holds := z.GetLatestHoldings()
				if !holds.Timestamp.IsZero() {
					resp = append(resp, holds)
				}
			}
		}
	}
	return resp
}

func (p *Portfolio) setHoldingsForOffset(h *holdings.Holding, overwriteExisting bool) error {
	if h.Timestamp.IsZero() {
		return errHoldingsNoTimestamp
	}
	lookup, ok := p.exchangeAssetPairSettings[h.Exchange][h.Asset][h.Pair]
	if !ok {
		return fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, h.Exchange, h.Asset, h.Pair)
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
func (p *Portfolio) ViewHoldingAtTimePeriod(ev common.EventHandler) (*holdings.Holding, error) {
	exchangeAssetPairSettings := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if exchangeAssetPairSettings == nil {
		return nil, fmt.Errorf("%w for %v %v %v", errNoHoldings, ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	}

	for i := len(exchangeAssetPairSettings.HoldingsSnapshots) - 1; i >= 0; i-- {
		if ev.GetTime().Equal(exchangeAssetPairSettings.HoldingsSnapshots[i].Timestamp) {
			return &exchangeAssetPairSettings.HoldingsSnapshots[i], nil
		}
	}

	return nil, fmt.Errorf("%w for %v %v %v at %v", errNoHoldings, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetTime())
}

// SetupCurrencySettingsMap ensures a map is created and no panics happen
func (p *Portfolio) SetupCurrencySettingsMap(settings *exchange.Settings, exch gctexchange.IBotExchange) error {
	if settings == nil {
		return errNoPortfolioSettings
	}
	if settings.Exchange == "" {
		return errExchangeUnset
	}
	if settings.Asset == "" {
		return errAssetUnset
	}
	if settings.Pair.IsEmpty() {
		return errCurrencyPairUnset
	}
	if p.exchangeAssetPairSettings == nil {
		p.exchangeAssetPairSettings = make(map[string]map[asset.Item]map[currency.Pair]*Settings)
	}
	if p.exchangeAssetPairSettings[settings.Exchange] == nil {
		p.exchangeAssetPairSettings[settings.Exchange] = make(map[asset.Item]map[currency.Pair]*Settings)
	}
	if p.exchangeAssetPairSettings[settings.Exchange][settings.Asset] == nil {
		p.exchangeAssetPairSettings[settings.Exchange][settings.Asset] = make(map[currency.Pair]*Settings)
	}
	if _, ok := p.exchangeAssetPairSettings[settings.Exchange][settings.Asset][settings.Pair]; ok {
		return nil
	}
	p.exchangeAssetPairSettings[settings.Exchange][settings.Asset][settings.Pair] = &Settings{
		Fee:            settings.ExchangeFee,
		BuySideSizing:  settings.BuySide,
		SellSideSizing: settings.SellSide,
		Leverage:       settings.Leverage,
		ComplianceManager: compliance.Manager{
			Snapshots: []compliance.Snapshot{},
		},
		Exchange: exch,
	}
	return nil
}

// GetLatestHoldings returns the latest holdings after being sorted by time
func (e *Settings) GetLatestHoldings() holdings.Holding {
	if len(e.HoldingsSnapshots) == 0 {
		return holdings.Holding{}
	}

	return e.HoldingsSnapshots[len(e.HoldingsSnapshots)-1]
}

// GetHoldingsForTime returns the holdings for a time period, or an empty holding if not found
func (e *Settings) GetHoldingsForTime(t time.Time) holdings.Holding {
	if e.HoldingsSnapshots == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	for i := len(e.HoldingsSnapshots) - 1; i >= 0; i-- {
		if e.HoldingsSnapshots[i].Timestamp.Equal(t) {
			return e.HoldingsSnapshots[i]
		}
	}
	return holdings.Holding{}
}

// CalculatePNL will analyse any futures orders that have been placed over the backtesting run
// that are not closed and calculate their PNL
func (p *Portfolio) CalculatePNL(e common.DataEventHandler, funds funding.ICollateralReleaser) error {
	settings, ok := p.exchangeAssetPairSettings[e.GetExchange()][e.GetAssetType()][e.Pair()]
	if !ok {
		return errNoPortfolioSettings
	}

	snapshot, err := p.GetLatestOrderSnapshotForEvent(e)
	if err != nil {
		return err
	}
	for i := range snapshot.Orders {
		if snapshot.Orders[i].FuturesOrder == nil {
			continue
		}
		if snapshot.Orders[i].FuturesOrder.ClosingPosition != nil {
			continue
		}
		if snapshot.Orders[i].FuturesOrder.OpeningPosition.Leverage == 0 {
			snapshot.Orders[i].FuturesOrder.OpeningPosition.Leverage = 1
		}

		var result *gctexchange.PNLResult
		result, err = settings.Exchange.CalculatePNL(&gctexchange.PNLCalculator{
			Asset:              e.GetAssetType(),
			Leverage:           snapshot.Orders[i].FuturesOrder.OpeningPosition.Leverage,
			EntryPrice:         snapshot.Orders[i].FuturesOrder.OpeningPosition.Price,
			OpeningAmount:      snapshot.Orders[i].FuturesOrder.OpeningPosition.Amount,
			CurrentPrice:       e.GetClosePrice().InexactFloat64(),
			CollateralAmount:   funds.AvailableFunds(),
			CalculateOffline:   true,
			CollateralCurrency: funds.CollateralCurrency(),
			Amount:             snapshot.Orders[i].FuturesOrder.OpeningPosition.Amount,
			MarkPrice:          e.GetClosePrice().InexactFloat64(),
			PrevMarkPrice:      e.GetOpenPrice().InexactFloat64(),
		})
		if err != nil {
			return err
		}

		if result.IsLiquidated {
			funds.Liquidate()
			snapshot.Orders[i].FuturesOrder.UnrealisedPNL = decimal.Zero
			snapshot.Orders[i].FuturesOrder.RealisedPNL = decimal.Zero
			or := snapshot.Orders[i].FuturesOrder.OpeningPosition.Copy()
			or.Side = common.Liquidated
			snapshot.Orders[i].FuturesOrder.ClosingPosition = &or
			return nil
		}
		snapshot.Orders[i].FuturesOrder.RealisedPNL.Add(snapshot.Orders[i].FuturesOrder.UnrealisedPNL)
		snapshot.Orders[i].FuturesOrder.UnrealisedPNL = result.UnrealisedPNL
		snapshot.Orders[i].FuturesOrder.OpeningPosition.UnrealisedPNL = result.UnrealisedPNL
		snapshot.Orders[i].FuturesOrder.UpsertPNLEntry(gctorder.PNLHistory{
			Time:          e.GetTime(),
			UnrealisedPNL: result.UnrealisedPNL,
			RealisedPNL:   snapshot.Orders[i].FuturesOrder.RealisedPNL,
		})
	}
	return nil
}

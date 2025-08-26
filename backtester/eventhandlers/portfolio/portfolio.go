package portfolio

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// OnSignal receives the event from the strategy on whether it has signalled to buy, do nothing or sell
// on buy/sell, the portfolio manager will size the order and assess the risk of the order
// if successful, it will pass on an order.Order to be used by the exchange event handler to place an order based on
// the portfolio manager's recommendations
func (p *Portfolio) OnSignal(ev signal.Event, exchangeSettings *exchange.Settings, funds funding.IFundReserver) (*order.Order, error) {
	if ev == nil {
		return nil, fmt.Errorf("%w signal event", gctcommon.ErrNilPointer)
	}
	if exchangeSettings == nil {
		return nil, fmt.Errorf("%w exchange settings", gctcommon.ErrNilPointer)
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
		Base:               ev.GetBase(),
		Direction:          ev.GetDirection(),
		FillDependentEvent: ev.GetFillDependentEvent(),
		Amount:             ev.GetAmount(),
		ClosePrice:         ev.GetClosePrice(),
	}
	if ev.GetDirection() == gctorder.UnknownSide {
		return o, errInvalidDirection
	}

	lookup := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(ev.GetExchange(), ev.GetAssetType(), ev.Pair())]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v",
			errNoPortfolioSettings,
			ev.GetExchange(),
			ev.GetAssetType(),
			ev.Pair())
	}

	if ev.GetDirection() == gctorder.DoNothing ||
		ev.GetDirection() == gctorder.MissingData ||
		ev.GetDirection() == gctorder.TransferredFunds {
		return o, nil
	}
	if !funds.CanPlaceOrder(ev.GetDirection()) {
		return cannotPurchase(ev, o)
	}

	o.OrderType = gctorder.Market
	o.BuyLimit = ev.GetBuyLimit()
	o.SellLimit = ev.GetSellLimit()
	var sizingFunds decimal.Decimal
	side := ev.GetDirection()
	if ev.GetAssetType() == asset.Spot {
		if side == gctorder.ClosePosition {
			side = gctorder.Sell
		}
		pReader, err := funds.GetPairReader()
		if err != nil {
			return nil, err
		}
		switch side {
		case gctorder.Buy, gctorder.Bid:
			sizingFunds = pReader.QuoteAvailable()
		case gctorder.Sell, gctorder.Ask:
			sizingFunds = pReader.BaseAvailable()
		}
	} else if ev.GetAssetType().IsFutures() {
		if ev.GetDirection() == gctorder.ClosePosition {
			// lookup position
			positions := lookup.FuturesTracker.GetPositions()
			if len(positions) == 0 {
				// cannot close a non existent position
				return nil, errNoHoldings
			}
			sizingFunds = positions[len(positions)-1].LatestSize
			d := positions[len(positions)-1].LatestDirection
			switch d {
			case gctorder.Short, gctorder.Sell, gctorder.Ask:
				side = gctorder.Long
			case gctorder.Long, gctorder.Buy, gctorder.Bid:
				side = gctorder.Short
			}
		} else {
			collateralFunds, err := funds.GetCollateralReader()
			if err != nil {
				return nil, err
			}
			sizingFunds = collateralFunds.AvailableFunds()
		}
	}
	if sizingFunds.LessThanOrEqual(decimal.Zero) {
		return cannotPurchase(ev, o)
	}
	sizedOrder, err := p.sizeOrder(ev, exchangeSettings, o, sizingFunds, funds)
	if err != nil {
		return sizedOrder, err
	}
	if common.CanTransact(sizedOrder.Direction) {
		sizedOrder.SetDirection(side)
	}
	if ev.GetDirection() == gctorder.ClosePosition {
		sizedOrder.ClosingPosition = true
	}
	return p.evaluateOrder(ev, o, sizedOrder)
}

func cannotPurchase(ev signal.Event, o *order.Order) (*order.Order, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	if o == nil {
		return nil, fmt.Errorf("%w received nil order for %v %v %v", gctcommon.ErrNilPointer, ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	}
	o.AppendReason(notEnoughFundsTo + " " + ev.GetDirection().Lower())
	switch ev.GetDirection() {
	case gctorder.Buy, gctorder.Bid:
		o.SetDirection(gctorder.CouldNotBuy)
	case gctorder.Sell, gctorder.Ask:
		o.SetDirection(gctorder.CouldNotSell)
	case gctorder.Short:
		o.SetDirection(gctorder.CouldNotShort)
	case gctorder.Long:
		o.SetDirection(gctorder.CouldNotLong)
	default:
		// ensure that unknown scenarios don't affect anything
		o.SetDirection(gctorder.DoNothing)
	}
	ev.SetDirection(o.Direction)
	return o, nil
}

func (p *Portfolio) evaluateOrder(d common.Directioner, originalOrderSignal, ev *order.Order) (*order.Order, error) {
	var evaluatedOrder *order.Order
	cm, err := p.getComplianceManager(originalOrderSignal.GetExchange(), originalOrderSignal.GetAssetType(), originalOrderSignal.Pair())
	if err != nil {
		return nil, err
	}

	evaluatedOrder, err = p.riskManager.EvaluateOrder(ev, p.GetLatestHoldingsForAllCurrencies(), cm.GetLatestSnapshot())
	if err != nil {
		originalOrderSignal.AppendReason(err.Error())
		switch d.GetDirection() {
		case gctorder.Buy, gctorder.CouldNotBuy:
			originalOrderSignal.Direction = gctorder.CouldNotBuy
		case gctorder.Sell, gctorder.CouldNotSell:
			originalOrderSignal.Direction = gctorder.CouldNotSell
		case gctorder.Short:
			originalOrderSignal.Direction = gctorder.CouldNotShort
		case gctorder.Long:
			originalOrderSignal.Direction = gctorder.CouldNotLong
		default:
			originalOrderSignal.Direction = gctorder.DoNothing
		}
		d.SetDirection(originalOrderSignal.Direction)
		return originalOrderSignal, nil
	}

	return evaluatedOrder, nil
}

func (p *Portfolio) sizeOrder(d common.Directioner, cs *exchange.Settings, originalOrderSignal *order.Order, sizingFunds decimal.Decimal, funds funding.IFundReserver) (*order.Order, error) {
	sizedOrder, estFee, err := p.sizeManager.SizeOrder(originalOrderSignal, sizingFunds, cs)
	if err != nil || sizedOrder.Amount.IsZero() {
		switch originalOrderSignal.Direction {
		case gctorder.Buy, gctorder.Bid:
			originalOrderSignal.Direction = gctorder.CouldNotBuy
		case gctorder.Sell, gctorder.Ask:
			originalOrderSignal.Direction = gctorder.CouldNotSell
		case gctorder.Long:
			originalOrderSignal.Direction = gctorder.CouldNotLong
		case gctorder.Short:
			originalOrderSignal.Direction = gctorder.CouldNotShort
		default:
			originalOrderSignal.Direction = gctorder.DoNothing
		}
		d.SetDirection(originalOrderSignal.Direction)
		if err != nil {
			originalOrderSignal.AppendReason(err.Error())
			return originalOrderSignal, nil
		}
		originalOrderSignal.AppendReason("sized order to 0")
	}
	switch d.GetDirection() {
	case gctorder.Buy,
		gctorder.Bid,
		gctorder.Short,
		gctorder.Long:
		sizedOrder.AllocatedFunds = sizedOrder.Amount.Mul(sizedOrder.ClosePrice).Add(estFee)
	case gctorder.Sell,
		gctorder.Ask,
		gctorder.ClosePosition:
		sizedOrder.AllocatedFunds = sizedOrder.Amount
	default:
		return nil, errInvalidDirection
	}
	err = funds.Reserve(sizedOrder.AllocatedFunds, d.GetDirection())
	if err != nil {
		sizedOrder.Direction = gctorder.DoNothing
		return sizedOrder, err
	}
	return sizedOrder, nil
}

// OnFill processes the event after an order has been placed by the exchange. Its purpose is to track holdings for future portfolio decisions.
func (p *Portfolio) OnFill(ev fill.Event, funds funding.IFundReleaser) (fill.Event, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	lookup := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(ev.GetExchange(), ev.GetAssetType(), ev.Pair())]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	}

	h, err := lookup.GetHoldingsForTime(ev.GetTime().Add(-ev.GetInterval().Duration()))
	if err != nil {
		if !errors.Is(err, errNoHoldings) {
			return nil, err
		}
		h, err = holdings.Create(ev, funds)
		if err != nil {
			return nil, err
		}
	}

	err = h.Update(ev, funds)
	if err != nil {
		return nil, err
	}
	err = p.SetHoldingsForTimestamp(h)
	if err != nil {
		return nil, err
	}
	err = p.addComplianceSnapshot(ev)
	if err != nil {
		return nil, err
	}
	return ev, nil
}

// addComplianceSnapshot gets the previous snapshot of compliance events, updates with the latest fill event
// then saves the snapshot to the c
func (p *Portfolio) addComplianceSnapshot(fillEvent fill.Event) error {
	if fillEvent == nil {
		return common.ErrNilEvent
	}
	complianceManager, err := p.getComplianceManager(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair())
	if err != nil {
		return err
	}
	prevSnap := complianceManager.GetLatestSnapshot()
	if filledOrder := fillEvent.GetOrder(); filledOrder != nil {
		price := decimal.NewFromFloat(filledOrder.Price)
		amount := decimal.NewFromFloat(filledOrder.Amount)
		fee := decimal.NewFromFloat(filledOrder.Fee)
		snapOrder := compliance.SnapshotOrder{
			ClosePrice:          fillEvent.GetClosePrice(),
			VolumeAdjustedPrice: fillEvent.GetVolumeAdjustedPrice(),
			SlippageRate:        fillEvent.GetSlippageRate(),
			CostBasis:           price.Mul(amount).Add(fee),
		}
		snapOrder.Order = filledOrder
		prevSnap.Orders = append(prevSnap.Orders, snapOrder)
	}
	snap := &compliance.Snapshot{
		Offset:    fillEvent.GetOffset(),
		Timestamp: fillEvent.GetTime(),
		Orders:    prevSnap.Orders,
	}
	return complianceManager.AddSnapshot(snap, false)
}

// GetLatestOrderSnapshotForEvent gets orders related to the event
func (p *Portfolio) GetLatestOrderSnapshotForEvent(ev common.Event) (compliance.Snapshot, error) {
	eapSettings, ok := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(ev.GetExchange(), ev.GetAssetType(), ev.Pair())]
	if !ok {
		return compliance.Snapshot{}, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	}
	return eapSettings.ComplianceManager.GetLatestSnapshot(), nil
}

// GetLatestOrderSnapshots returns the latest snapshots from all stored pair data
func (p *Portfolio) GetLatestOrderSnapshots() ([]compliance.Snapshot, error) {
	resp := make([]compliance.Snapshot, 0, len(p.exchangeAssetPairPortfolioSettings))
	for _, d := range p.exchangeAssetPairPortfolioSettings {
		resp = append(resp, d.ComplianceManager.GetLatestSnapshot())
	}
	if len(resp) == 0 {
		return nil, errNoPortfolioSettings
	}
	return resp, nil
}

// GetLatestComplianceSnapshot returns the latest compliance snapshot for a given exchange, asset, pair
func (p *Portfolio) GetLatestComplianceSnapshot(exchangeName string, a asset.Item, cp currency.Pair) (*compliance.Snapshot, error) {
	cm, err := p.getComplianceManager(exchangeName, a, cp)
	if err != nil {
		return nil, err
	}
	snap := cm.GetLatestSnapshot()

	return &snap, nil
}

// getComplianceManager returns the order snapshots for a given exchange, asset, pair
func (p *Portfolio) getComplianceManager(exchangeName string, a asset.Item, cp currency.Pair) (*compliance.Manager, error) {
	lookup := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(exchangeName, a, cp)]
	if lookup == nil {
		return nil, fmt.Errorf("%w for %v %v %v could not retrieve compliance manager", errNoPortfolioSettings, exchangeName, a, cp)
	}
	return &lookup.ComplianceManager, nil
}

// GetPositions returns all futures positions for an event's exchange, asset, pair
func (p *Portfolio) GetPositions(e common.Event) ([]futures.Position, error) {
	settings, err := p.getFuturesSettingsFromEvent(e)
	if err != nil {
		return nil, err
	}
	return settings.FuturesTracker.GetPositions(), nil
}

// GetLatestPosition returns all futures positions for an event's exchange, asset, pair
func (p *Portfolio) GetLatestPosition(e common.Event) (*futures.Position, error) {
	settings, err := p.getFuturesSettingsFromEvent(e)
	if err != nil {
		return nil, err
	}
	positions := settings.FuturesTracker.GetPositions()
	if len(positions) == 0 {
		return nil, fmt.Errorf("%w %v %v %v", futures.ErrPositionNotFound, e.GetExchange(), e.GetAssetType(), e.Pair())
	}
	return &positions[len(positions)-1], nil
}

// UpdatePNL will analyse any futures orders that have been placed over the backtesting run
// that are not closed and calculate their PNL
func (p *Portfolio) UpdatePNL(e common.Event, closePrice decimal.Decimal) error {
	settings, err := p.getFuturesSettingsFromEvent(e)
	if err != nil {
		return err
	}
	_, err = settings.FuturesTracker.UpdateOpenPositionUnrealisedPNL(closePrice.InexactFloat64(), e.GetTime())
	if err != nil && !errors.Is(err, futures.ErrPositionClosed) {
		return err
	}

	return nil
}

// TrackFuturesOrder updates the futures tracker with a new order
// from a fill event
func (p *Portfolio) TrackFuturesOrder(ev fill.Event, fund funding.IFundReleaser) (*PNLSummary, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	if fund == nil {
		return nil, fmt.Errorf("%w missing funding", gctcommon.ErrNilPointer)
	}
	detail := ev.GetOrder()
	if detail == nil {
		return nil, gctorder.ErrSubmissionIsNil
	}
	if !detail.AssetType.IsFutures() {
		return nil, fmt.Errorf("order '%v' %w", detail.OrderID, futures.ErrNotFuturesAsset)
	}

	collateralReleaser, err := fund.CollateralReleaser()
	if err != nil {
		return nil, fmt.Errorf("%v %v %v %w", detail.Exchange, detail.AssetType, detail.Pair, err)
	}
	settings, err := p.getSettings(detail.Exchange, detail.AssetType, detail.Pair)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	err = settings.FuturesTracker.TrackNewOrder(detail)
	if err != nil {
		return nil, err
	}

	pos := settings.FuturesTracker.GetPositions()
	if len(pos) == 0 {
		return nil, fmt.Errorf("%w should not happen", errNoHoldings)
	}
	amount := decimal.NewFromFloat(detail.Amount)
	switch {
	case ev.IsLiquidated():
		collateralReleaser.Liquidate()
		err = settings.FuturesTracker.Liquidate(ev.GetClosePrice(), ev.GetTime())
		if err != nil {
			return nil, err
		}
	case pos[len(pos)-1].OpeningDirection != detail.Side:
		err = collateralReleaser.TakeProfit(amount, pos[len(pos)-1].RealisedPNL)
		if err != nil {
			return nil, err
		}
		err = p.UpdatePNL(ev, ev.GetClosePrice())
		if err != nil {
			return nil, fmt.Errorf("%v %v %v %w", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
	default:
		err = collateralReleaser.UpdateContracts(detail.Side, amount)
		if err != nil {
			return nil, err
		}
	}

	return p.GetLatestPNLForEvent(ev)
}

// GetLatestPNLForEvent takes in an event and returns the latest PNL data
// if it exists
func (p *Portfolio) GetLatestPNLForEvent(e common.Event) (*PNLSummary, error) {
	if e == nil {
		return nil, common.ErrNilEvent
	}
	response := &PNLSummary{
		Exchange: e.GetExchange(),
		Asset:    e.GetAssetType(),
		Pair:     e.Pair(),
		Offset:   e.GetOffset(),
	}
	position, err := p.GetLatestPosition(e)
	if err != nil {
		return nil, err
	}
	pnlHistory := position.PNLHistory
	if len(pnlHistory) == 0 {
		return response, nil
	}
	response.Result = pnlHistory[len(pnlHistory)-1]
	response.CollateralCurrency = position.CollateralCurrency
	return response, nil
}

// CheckLiquidationStatus checks funding against position
// and liquidates and removes funding if position unable to continue
func (p *Portfolio) CheckLiquidationStatus(ev data.Event, collateralReader funding.ICollateralReader, pnl *PNLSummary) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if collateralReader == nil {
		return fmt.Errorf("%w collateral reader missing", gctcommon.ErrNilPointer)
	}
	if pnl == nil {
		return fmt.Errorf("%w pnl summary missing", gctcommon.ErrNilPointer)
	}
	availableFunds := collateralReader.AvailableFunds()
	position, err := p.GetLatestPosition(ev)
	if err != nil {
		return err
	}
	if !position.Status.IsInactive() &&
		pnl.Result.UnrealisedPNL.IsNegative() &&
		pnl.Result.UnrealisedPNL.Abs().GreaterThan(availableFunds) {
		return futures.ErrPositionLiquidated
	}

	return nil
}

// CreateLiquidationOrdersForExchange creates liquidation orders, for any that exist on the same exchange where a liquidation is occurring
func (p *Portfolio) CreateLiquidationOrdersForExchange(ev data.Event, funds funding.IFundingManager) ([]order.Event, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	if funds == nil {
		return nil, fmt.Errorf("%w, requires funding manager", gctcommon.ErrNilPointer)
	}
	var closingOrders []order.Event
	for mapKey, settings := range p.exchangeAssetPairPortfolioSettings {
		if !mapKey.MatchesExchange(ev.GetExchange()) {
			continue
		}
		switch {
		case mapKey.Asset.IsFutures():
			positions := settings.FuturesTracker.GetPositions()
			if len(positions) == 0 {
				continue
			}
			pos := positions[len(positions)-1]
			if !pos.LatestSize.IsPositive() {
				continue
			}
			direction := gctorder.Short
			if pos.LatestDirection == gctorder.Short {
				direction = gctorder.Long
			}
			closingOrders = append(closingOrders, &order.Order{
				Base: &event.Base{
					Offset:         ev.GetOffset(),
					Exchange:       pos.Exchange,
					Time:           ev.GetTime(),
					Interval:       ev.GetInterval(),
					CurrencyPair:   pos.Pair,
					UnderlyingPair: ev.GetUnderlyingPair(),
					AssetType:      pos.Asset,
					Reasons:        []string{"LIQUIDATED"},
				},
				Direction:           direction,
				Status:              gctorder.Liquidated,
				ClosePrice:          ev.GetClosePrice(),
				Amount:              pos.LatestSize,
				AllocatedFunds:      pos.LatestSize,
				OrderType:           gctorder.Market,
				LiquidatingPosition: true,
			})
		case mapKey.Asset == asset.Spot:
			allFunds, err := funds.GetAllFunding()
			if err != nil {
				return nil, err
			}
			for i := range allFunds {
				if allFunds[i].Asset.IsFutures() {
					continue
				}
				if allFunds[i].Currency.IsFiatCurrency() || allFunds[i].Currency.IsStableCurrency() {
					// close orders for assets
					// funding manager will zero for fiat/stable
					continue
				}
				cp := currency.NewPair(mapKey.Base.Currency(), mapKey.Quote.Currency())
				closingOrders = append(closingOrders, &order.Order{
					Base: &event.Base{
						Offset:       ev.GetOffset(),
						Exchange:     ev.GetExchange(),
						Time:         ev.GetTime(),
						Interval:     ev.GetInterval(),
						CurrencyPair: cp,
						AssetType:    mapKey.Asset,
						Reasons:      []string{"LIQUIDATED"},
					},
					Direction:           gctorder.Sell,
					Status:              gctorder.Liquidated,
					Amount:              allFunds[i].Available,
					OrderType:           gctorder.Market,
					AllocatedFunds:      allFunds[i].Available,
					LiquidatingPosition: true,
				})
			}
		}
	}

	return closingOrders, nil
}

func (p *Portfolio) getFuturesSettingsFromEvent(e common.Event) (*Settings, error) {
	if e == nil {
		return nil, common.ErrNilEvent
	}
	if !e.GetAssetType().IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	settings, err := p.getSettings(e.GetExchange(), e.GetAssetType(), e.Pair())
	if err != nil {
		return nil, fmt.Errorf("%v %v %v %w", e.GetExchange(), e.GetAssetType(), e.Pair(), err)
	}

	if settings.FuturesTracker == nil {
		return nil, fmt.Errorf("%w for %v %v %v", errUnsetFuturesTracker, e.GetExchange(), e.GetAssetType(), e.Pair())
	}

	return settings, nil
}

func (p *Portfolio) getSettings(exch string, item asset.Item, pair currency.Pair) (*Settings, error) {
	exch = strings.ToLower(exch)
	settings, ok := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(exch, item, pair)]
	if !ok {
		return nil, fmt.Errorf("%w for %v %v %v", errNoPortfolioSettings, exch, item, pair)
	}
	return settings, nil
}

// SetHoldingsForTimestamp stores a holding snapshot for the holding's timestamp
func (p *Portfolio) SetHoldingsForTimestamp(h *holdings.Holding) error {
	if h.Timestamp.IsZero() {
		return errHoldingsNoTimestamp
	}
	lookup, err := p.getSettings(h.Exchange, h.Asset, h.Pair)
	if err != nil {
		return err
	}
	lookup.HoldingsSnapshots[h.Timestamp.UnixNano()] = h
	return nil
}

// UpdateHoldings updates the portfolio holdings for the data event
func (p *Portfolio) UpdateHoldings(e data.Event, funds funding.IFundReleaser) error {
	if e == nil {
		return common.ErrNilEvent
	}
	if funds == nil {
		return funding.ErrFundsNotFound
	}
	settings, err := p.getSettings(e.GetExchange(), e.GetAssetType(), e.Pair())
	if err != nil {
		return fmt.Errorf("%v %v %v %w", e.GetExchange(), e.GetAssetType(), e.Pair(), err)
	}
	h, err := settings.GetLatestHoldings()
	if err != nil {
		if !errors.Is(err, errNoHoldings) {
			return err
		}
		h, err = holdings.Create(e, funds)
		if err != nil {
			return err
		}
	}
	err = h.UpdateValue(e)
	if err != nil {
		return err
	}
	return p.SetHoldingsForTimestamp(h)
}

// GetLatestHoldingsForAllCurrencies will return the current holdings for all loaded currencies
// this is useful to assess the position of your entire portfolio in order to help with risk decisions
func (p *Portfolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	resp := make([]holdings.Holding, 0, len(p.exchangeAssetPairPortfolioSettings))
	for _, d := range p.exchangeAssetPairPortfolioSettings {
		holds, err := d.GetLatestHoldings()
		if err != nil {
			continue
		}
		resp = append(resp, *holds)
	}

	return resp
}

// ViewHoldingAtTimePeriod retrieves a snapshot of holdings at a specific time period,
// returning an error if not found
func (p *Portfolio) ViewHoldingAtTimePeriod(ev common.Event) (*holdings.Holding, error) {
	settings, err := p.getSettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		return nil, err
	}

	h, ok := settings.HoldingsSnapshots[ev.GetTime().UnixNano()]
	if !ok {
		return nil, fmt.Errorf("%w for %v %v %v at %v", errNoHoldings, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetTime())
	}
	return h, nil
}

// GetLatestHoldings returns the latest holdings after being sorted by time
func (s *Settings) GetLatestHoldings() (*holdings.Holding, error) {
	if len(s.HoldingsSnapshots) == 0 {
		return nil, errNoHoldings
	}
	var latestTime int64
	for k := range s.HoldingsSnapshots {
		if k > latestTime {
			latestTime = k
		}
	}
	return s.HoldingsSnapshots[latestTime], nil
}

// GetHoldingsForTime returns the holdings for a time period, or an error holding if not found
func (s *Settings) GetHoldingsForTime(t time.Time) (*holdings.Holding, error) {
	h, ok := s.HoldingsSnapshots[t.UnixNano()]
	if !ok {
		return nil, fmt.Errorf("%w for %v %v %v at %v", errNoHoldings, s.exchangeName, s.assetType, s.pair, t)
	}
	return h, nil
}

// SetHoldingsForEvent re-sets offset details at the events time,
// based on current funding levels
func (p *Portfolio) SetHoldingsForEvent(fm funding.IFundReader, e common.Event) error {
	if fm == nil {
		return fmt.Errorf("%w funding manager", gctcommon.ErrNilPointer)
	}
	if e == nil {
		return common.ErrNilEvent
	}
	settings, err := p.getSettings(e.GetExchange(), e.GetAssetType(), e.Pair())
	if err != nil {
		return err
	}
	h, err := settings.GetHoldingsForTime(e.GetTime())
	if err != nil {
		if !errors.Is(err, errNoHoldings) {
			return err
		}
		h, err = holdings.Create(e, fm)
		if err != nil {
			return err
		}
	}
	if e.GetAssetType().IsFutures() {
		var c funding.ICollateralReader
		c, err = fm.GetCollateralReader()
		if err != nil {
			return err
		}
		h.BaseSize = c.CurrentHoldings()
		h.QuoteSize = c.AvailableFunds()
	} else {
		var pr funding.IPairReader
		pr, err = fm.GetPairReader()
		if err != nil {
			return err
		}
		h.BaseSize = pr.BaseAvailable()
		h.QuoteSize = pr.QuoteAvailable()
	}
	err = h.UpdateValue(e)
	if err != nil {
		return err
	}
	return p.SetHoldingsForTimestamp(h)
}

// GetUnrealisedPNL returns a basic struct containing unrealised PNL
func (p *PNLSummary) GetUnrealisedPNL() BasicPNLResult {
	return BasicPNLResult{
		Time:     p.Result.Time,
		PNL:      p.Result.UnrealisedPNL,
		Currency: p.CollateralCurrency,
	}
}

// GetRealisedPNL returns a basic struct containing realised PNL
func (p *PNLSummary) GetRealisedPNL() BasicPNLResult {
	return BasicPNLResult{
		Time:     p.Result.Time,
		PNL:      p.Result.RealisedPNL,
		Currency: p.CollateralCurrency,
	}
}

// GetExposure returns the position exposure
func (p *PNLSummary) GetExposure() decimal.Decimal {
	return p.Result.Exposure
}

// GetCollateralCurrency returns the collateral currency
func (p *PNLSummary) GetCollateralCurrency() currency.Code {
	return p.CollateralCurrency
}

// GetDirection returns the direction
func (p *PNLSummary) GetDirection() gctorder.Side {
	return p.Result.Direction
}

// GetPositionStatus returns the position status
func (p *PNLSummary) GetPositionStatus() gctorder.Status {
	return p.Result.Status
}

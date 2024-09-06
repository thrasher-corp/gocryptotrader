package futures

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SetupPositionController creates a position controller
// to track futures orders
func SetupPositionController() PositionController {
	return PositionController{
		multiPositionTrackers: make(map[key.ExchangePairAsset]*MultiPositionTracker),
	}
}

// TrackNewOrder sets up the maps to then create a
// multi position tracker which funnels down into the
// position tracker, to then track an order's pnl
func (c *PositionController) TrackNewOrder(d *order.Detail) error {
	if c == nil {
		return fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	if d == nil {
		return errNilOrder
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.AssetType, d.Pair)
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	exchMap, ok := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: d.Exchange,
		Base:     d.Pair.Base.Item,
		Quote:    d.Pair.Quote.Item,
		Asset:    d.AssetType,
	}]
	if !ok {
		exchMap, err = SetupMultiPositionTracker(&MultiPositionTrackerSetup{
			Exchange:   d.Exchange,
			Asset:      d.AssetType,
			Pair:       d.Pair,
			Underlying: d.Pair.Base,
		})
		if err != nil {
			return err
		}
		c.multiPositionTrackers[key.ExchangePairAsset{
			Exchange: d.Exchange,
			Base:     d.Pair.Base.Item,
			Quote:    d.Pair.Quote.Item,
			Asset:    d.AssetType,
		}] = exchMap
	}
	err = exchMap.TrackNewOrder(d)
	if err != nil {
		return err
	}
	c.updated = time.Now()
	return nil
}

// SetCollateralCurrency allows the setting of a collateral currency to all child trackers
// when using position controller for futures orders tracking
func (c *PositionController) SetCollateralCurrency(exch string, item asset.Item, pair currency.Pair, collateralCurrency currency.Code) error {
	if c == nil {
		return fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	var err error
	exch, err = checkTrackerPrerequisitesLowerExchange(exch, item, pair)
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()

	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}]
	if tracker == nil {
		return fmt.Errorf("%w no open position for %v %v %v", ErrPositionNotFound, exch, item, pair)
	}
	tracker.m.Lock()
	defer tracker.m.Unlock()
	tracker.collateralCurrency = collateralCurrency
	for i := range tracker.positions {
		tracker.positions[i].m.Lock()
		tracker.positions[i].collateralCurrency = collateralCurrency
		tracker.positions[i].m.Unlock()
	}
	return nil
}

// GetPositionsForExchange returns all positions for an
// exchange, asset pair that is stored in the position controller
func (c *PositionController) GetPositionsForExchange(exch string, item asset.Item, pair currency.Pair) ([]Position, error) {
	if c == nil {
		return nil, fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	var err error
	exch, err = checkTrackerPrerequisitesLowerExchange(exch, item, pair)
	if err != nil {
		return nil, err
	}
	c.m.Lock()
	defer c.m.Unlock()
	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}]
	if tracker == nil {
		return nil, fmt.Errorf("%w no open position for %v %v %v", ErrPositionNotFound, exch, item, pair)
	}

	return tracker.GetPositions(), nil
}

// TrackFundingDetails applies funding rate details to a tracked position
func (c *PositionController) TrackFundingDetails(d *fundingrate.HistoricalRates) error {
	if c == nil {
		return fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	if d == nil {
		return fmt.Errorf("%w funding rate details", common.ErrNilPointer)
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.Asset, d.Pair)
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: d.Exchange,
		Base:     d.Pair.Base.Item,
		Quote:    d.Pair.Quote.Item,
		Asset:    d.Asset,
	}]
	if tracker == nil {
		return fmt.Errorf("%w no open position for %v %v %v", ErrPositionNotFound, d.Exchange, d.Asset, d.Pair)
	}
	err = tracker.TrackFundingDetails(d)
	if err != nil {
		return err
	}
	c.updated = time.Now()
	return nil
}

// LastUpdated is used for the order manager as a way of knowing
// what span of time to check for orders
func (c *PositionController) LastUpdated() (time.Time, error) {
	if c == nil {
		return time.Time{}, fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	c.m.Lock()
	defer c.m.Unlock()
	return c.updated, nil
}

// GetOpenPosition returns an open positions that matches the exchange, asset, pair
func (c *PositionController) GetOpenPosition(exch string, item asset.Item, pair currency.Pair) (*Position, error) {
	if c == nil {
		return nil, fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	var err error
	exch, err = checkTrackerPrerequisitesLowerExchange(exch, item, pair)
	if err != nil {
		return nil, err
	}
	c.m.Lock()
	defer c.m.Unlock()
	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}]
	if tracker == nil {
		return nil, fmt.Errorf("%w no open position for %v %v %v", ErrPositionNotFound, exch, item, pair)
	}
	positions := tracker.GetPositions()
	for i := range positions {
		if positions[i].Status.IsInactive() {
			continue
		}
		return &positions[i], nil
	}

	return nil, fmt.Errorf("%w no open position for %v %v %v", ErrPositionNotFound, exch, item, pair)
}

// GetAllOpenPositions returns all open positions with optional filters
func (c *PositionController) GetAllOpenPositions() ([]Position, error) {
	if c == nil {
		return nil, fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	c.m.Lock()
	defer c.m.Unlock()
	var openPositions []Position
	for _, multiPositionTracker := range c.multiPositionTrackers {
		positions := multiPositionTracker.GetPositions()
		for i := range positions {
			if positions[i].Status.IsInactive() {
				continue
			}
			openPositions = append(openPositions, positions[i])
		}
	}
	if len(openPositions) == 0 {
		return nil, ErrNoPositionsFound
	}
	return openPositions, nil
}

// UpdateOpenPositionUnrealisedPNL finds an open position from
// an exchange asset pair, then calculates the unrealisedPNL
// using the latest ticker data
func (c *PositionController) UpdateOpenPositionUnrealisedPNL(exch string, item asset.Item, pair currency.Pair, last float64, updated time.Time) (decimal.Decimal, error) {
	if c == nil {
		return decimal.Zero, fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	var err error
	exch, err = checkTrackerPrerequisitesLowerExchange(exch, item, pair)
	if err != nil {
		return decimal.Zero, err
	}
	c.m.Lock()
	defer c.m.Unlock()
	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}]
	if tracker == nil {
		return decimal.Zero, fmt.Errorf("%v %v %v %w", exch, item, pair, ErrPositionNotFound)
	}

	tracker.m.Lock()
	defer tracker.m.Unlock()
	pos := tracker.positions
	if len(pos) == 0 {
		return decimal.Zero, fmt.Errorf("%v %v %v %w", exch, item, pair, ErrPositionNotFound)
	}
	latestPos := pos[len(pos)-1]
	if latestPos.status != order.Open {
		return decimal.Zero, fmt.Errorf("%v %v %v %w", exch, item, pair, ErrPositionClosed)
	}
	err = latestPos.TrackPNLByTime(updated, last)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w for position %v %v %v", err, exch, item, pair)
	}
	latestPos.m.Lock()
	defer latestPos.m.Unlock()
	return latestPos.unrealisedPNL, nil
}

// SetupMultiPositionTracker creates a futures order tracker for a specific exchange
func SetupMultiPositionTracker(setup *MultiPositionTrackerSetup) (*MultiPositionTracker, error) {
	if setup == nil {
		return nil, errNilSetup
	}
	if setup.Exchange == "" {
		return nil, errExchangeNameEmpty
	}
	var err error
	setup.Exchange, err = checkTrackerPrerequisitesLowerExchange(setup.Exchange, setup.Asset, setup.Pair)
	if err != nil {
		return nil, err
	}
	if setup.Underlying.IsEmpty() {
		return nil, errEmptyUnderlying
	}
	if setup.ExchangePNLCalculation == nil && setup.UseExchangePNLCalculation {
		return nil, errMissingPNLCalculationFunctions
	}
	return &MultiPositionTracker{
		exchange:                   setup.Exchange,
		asset:                      setup.Asset,
		pair:                       setup.Pair,
		underlying:                 setup.Underlying,
		offlinePNLCalculation:      setup.OfflineCalculation,
		orderPositions:             make(map[string]*PositionTracker),
		useExchangePNLCalculations: setup.UseExchangePNLCalculation,
		exchangePNLCalculation:     setup.ExchangePNLCalculation,
		collateralCurrency:         setup.CollateralCurrency,
	}, nil
}

// UpdateOpenPositionUnrealisedPNL updates the pnl for the latest open position
// based on the last price and the time
func (m *MultiPositionTracker) UpdateOpenPositionUnrealisedPNL(last float64, updated time.Time) (decimal.Decimal, error) {
	m.m.Lock()
	defer m.m.Unlock()
	pos := m.positions
	if len(pos) == 0 {
		return decimal.Zero, fmt.Errorf("%v %v %v %w", m.exchange, m.asset, m.pair, ErrPositionNotFound)
	}
	latestPos := pos[len(pos)-1]
	if latestPos.status.IsInactive() {
		return decimal.Zero, fmt.Errorf("%v %v %v %w", m.exchange, m.asset, m.pair, ErrPositionClosed)
	}
	err := latestPos.TrackPNLByTime(updated, last)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w for position %v %v %v", err, m.exchange, m.asset, m.pair)
	}
	latestPos.m.Lock()
	defer latestPos.m.Unlock()
	return latestPos.unrealisedPNL, nil
}

// ClearPositionsForExchange resets positions for an
// exchange, asset, pair that has been stored
func (c *PositionController) ClearPositionsForExchange(exch string, item asset.Item, pair currency.Pair) error {
	if c == nil {
		return fmt.Errorf("position controller %w", common.ErrNilPointer)
	}
	var err error
	exch, err = checkTrackerPrerequisitesLowerExchange(exch, item, pair)
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()

	tracker := c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}]
	if tracker == nil {
		return fmt.Errorf("%v %v %v %w", exch, item, pair, ErrPositionNotFound)
	}

	newMPT, err := SetupMultiPositionTracker(&MultiPositionTrackerSetup{
		Exchange:                  exch,
		Asset:                     item,
		Pair:                      pair,
		Underlying:                tracker.underlying,
		OfflineCalculation:        tracker.offlinePNLCalculation,
		UseExchangePNLCalculation: tracker.useExchangePNLCalculations,
		ExchangePNLCalculation:    tracker.exchangePNLCalculation,
		CollateralCurrency:        tracker.collateralCurrency,
	})
	if err != nil {
		return err
	}
	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: exch,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}] = newMPT
	return nil
}

// GetPositions returns all positions
func (m *MultiPositionTracker) GetPositions() []Position {
	if m == nil {
		return nil
	}
	m.m.Lock()
	defer m.m.Unlock()
	resp := make([]Position, len(m.positions))
	for i := range m.positions {
		resp[i] = *m.positions[i].GetStats()
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].OpeningDate.Before(resp[j].OpeningDate)
	})
	return resp
}

// TrackNewOrder upserts an order to the tracker and updates position
// status and exposure. PNL is calculated separately as it requires mark prices
func (m *MultiPositionTracker) TrackNewOrder(d *order.Detail) error {
	if m == nil {
		return fmt.Errorf("multi-position tracker %w", common.ErrNilPointer)
	}
	if d == nil {
		return fmt.Errorf("order detail %w", common.ErrNilPointer)
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.AssetType, d.Pair)
	if err != nil {
		return err
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.exchange != d.Exchange {
		return fmt.Errorf("%w received %v expected %v", errExchangeNameMismatch, d.Exchange, m.exchange)
	}
	if d.AssetType != m.asset {
		return errAssetMismatch
	}
	if tracker, ok := m.orderPositions[d.OrderID]; ok {
		// this has already been associated
		// update the tracker
		return tracker.TrackNewOrder(d, false)
	}
	if len(m.positions) > 0 {
		for i := range m.positions {
			if m.positions[i].status == order.Open && i != len(m.positions)-1 {
				return fmt.Errorf("%w %v at position %v/%v", errPositionDiscrepancy, m.positions[i], i, len(m.positions)-1)
			}
		}
		if m.positions[len(m.positions)-1].status == order.Open {
			err = m.positions[len(m.positions)-1].TrackNewOrder(d, false)
			if err != nil {
				return err
			}
			m.orderPositions[d.OrderID] = m.positions[len(m.positions)-1]
			return nil
		}
	}
	setup := &PositionTrackerSetup{
		Pair:                      d.Pair,
		EntryPrice:                decimal.NewFromFloat(d.Price),
		Underlying:                d.Pair.Base,
		CollateralCurrency:        m.collateralCurrency,
		Asset:                     d.AssetType,
		Side:                      d.Side,
		UseExchangePNLCalculation: m.useExchangePNLCalculations,
		OfflineCalculation:        m.offlinePNLCalculation,
		PNLCalculator:             m.exchangePNLCalculation,
		Exchange:                  m.exchange,
	}
	tracker, err := SetupPositionTracker(setup)
	if err != nil {
		return err
	}
	m.positions = append(m.positions, tracker)
	err = tracker.TrackNewOrder(d, true)
	if err != nil {
		return err
	}
	m.orderPositions[d.OrderID] = tracker
	return nil
}

// TrackFundingDetails applies funding rate details to a tracked position
func (m *MultiPositionTracker) TrackFundingDetails(d *fundingrate.HistoricalRates) error {
	if m == nil {
		return fmt.Errorf("multi-position tracker %w", common.ErrNilPointer)
	}
	if d == nil {
		return fmt.Errorf("%w FundingRates", common.ErrNilPointer)
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.Asset, d.Pair)
	if err != nil {
		return err
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.exchange != d.Exchange {
		return fmt.Errorf("%w received '%v' expected '%v'", errExchangeNameMismatch, d.Exchange, m.exchange)
	}
	if d.Asset != m.asset {
		return fmt.Errorf("%w tracker: %v supplied: %v", errAssetMismatch, m.asset, d.Asset)
	}
	if len(m.positions) == 0 {
		return fmt.Errorf("%w %v %v %v", ErrPositionNotFound, d.Exchange, d.Asset, d.Pair)
	}
	for i := range m.positions {
		err = m.positions[i].TrackFundingDetails(d)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetupPositionTracker creates a new position tracker to track n futures orders
// until the position(s) are closed
func SetupPositionTracker(setup *PositionTrackerSetup) (*PositionTracker, error) {
	if setup == nil {
		return nil, errNilSetup
	}
	var err error
	setup.Exchange, err = checkTrackerPrerequisitesLowerExchange(setup.Exchange, setup.Asset, setup.Pair)
	if err != nil {
		return nil, err
	}
	resp := &PositionTracker{
		exchange:                  setup.Exchange,
		asset:                     setup.Asset,
		contractPair:              setup.Pair,
		underlying:                setup.Underlying,
		status:                    order.Open,
		openingPrice:              setup.EntryPrice,
		latestDirection:           setup.Side,
		openingDirection:          setup.Side,
		useExchangePNLCalculation: setup.UseExchangePNLCalculation,
		offlinePNLCalculation:     setup.OfflineCalculation,
		lastUpdated:               time.Now(),
	}
	if !setup.UseExchangePNLCalculation {
		// use position tracker's pnl calculation by default
		resp.PNLCalculation = &PNLCalculator{}
	} else {
		if setup.PNLCalculator == nil {
			return nil, ErrNilPNLCalculator
		}
		resp.PNLCalculation = setup.PNLCalculator
	}
	return resp, nil
}

// Liquidate will update the latest open position's
// to reflect its liquidated status
func (m *MultiPositionTracker) Liquidate(price decimal.Decimal, t time.Time) error {
	if m == nil {
		return fmt.Errorf("multi-position tracker %w", common.ErrNilPointer)
	}
	m.m.Lock()
	defer m.m.Unlock()
	if len(m.positions) == 0 {
		return fmt.Errorf("%v %v %v %w", m.exchange, m.asset, m.pair, ErrPositionNotFound)
	}
	return m.positions[len(m.positions)-1].Liquidate(price, t)
}

// GetStats returns a summary of a future position
func (p *PositionTracker) GetStats() *Position {
	if p == nil {
		return nil
	}
	p.m.Lock()
	defer p.m.Unlock()
	var orders []order.Detail
	orders = append(orders, p.longPositions...)
	orders = append(orders, p.shortPositions...)
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Date.Before(orders[j].Date)
	})

	pos := &Position{
		Exchange:         p.exchange,
		Asset:            p.asset,
		Pair:             p.contractPair,
		Underlying:       p.underlying,
		RealisedPNL:      p.realisedPNL,
		UnrealisedPNL:    p.unrealisedPNL,
		Status:           p.status,
		OpeningDate:      p.openingDate,
		OpeningPrice:     p.openingPrice,
		OpeningSize:      p.openingSize,
		OpeningDirection: p.openingDirection,
		LatestPrice:      p.latestPrice,
		LatestSize:       p.exposure,
		LatestDirection:  p.latestDirection,
		CloseDate:        p.closingDate,
		Orders:           orders,
		PNLHistory:       p.pnlHistory,
		LastUpdated:      p.lastUpdated,
	}

	if p.fundingRateDetails != nil {
		frs := make([]fundingrate.Rate, len(p.fundingRateDetails.FundingRates))
		copy(frs, p.fundingRateDetails.FundingRates)
		pos.FundingRates = fundingrate.HistoricalRates{
			Exchange:              p.fundingRateDetails.Exchange,
			Asset:                 p.fundingRateDetails.Asset,
			Pair:                  p.fundingRateDetails.Pair,
			StartDate:             p.fundingRateDetails.StartDate,
			EndDate:               p.fundingRateDetails.EndDate,
			LatestRate:            p.fundingRateDetails.LatestRate,
			PredictedUpcomingRate: p.fundingRateDetails.PredictedUpcomingRate,
			FundingRates:          frs,
			PaymentSum:            p.fundingRateDetails.PaymentSum,
		}
	}

	return pos
}

// TrackPNLByTime calculates the PNL based on a position tracker's exposure
// and current pricing. Adds the entry to PNL history to track over time
func (p *PositionTracker) TrackPNLByTime(t time.Time, currentPrice float64) error {
	if p == nil {
		return fmt.Errorf("position tracker %w", common.ErrNilPointer)
	}
	p.m.Lock()
	defer func() {
		p.latestPrice = decimal.NewFromFloat(currentPrice)
		p.m.Unlock()
	}()
	price := decimal.NewFromFloat(currentPrice)
	result := &PNLResult{
		Time:   t,
		Price:  price,
		Status: p.status,
	}
	if p.latestDirection.IsLong() {
		diff := price.Sub(p.openingPrice)
		result.UnrealisedPNL = p.exposure.Mul(diff)
	} else if p.latestDirection.IsShort() {
		diff := p.openingPrice.Sub(price)
		result.UnrealisedPNL = p.exposure.Mul(diff)
	}
	if len(p.pnlHistory) > 0 {
		latest := p.pnlHistory[len(p.pnlHistory)-1]
		result.RealisedPNLBeforeFees = latest.RealisedPNLBeforeFees
		result.Exposure = latest.Exposure
		result.Direction = latest.Direction
		result.RealisedPNL = latest.RealisedPNL
		result.IsLiquidated = latest.IsLiquidated
	}
	var err error
	p.pnlHistory, err = upsertPNLEntry(p.pnlHistory, result)
	p.unrealisedPNL = result.UnrealisedPNL
	p.lastUpdated = time.Now()

	return err
}

// GetRealisedPNL returns the realised pnl if the order
// is closed
func (p *PositionTracker) GetRealisedPNL() decimal.Decimal {
	if p == nil {
		return decimal.Zero
	}
	p.m.Lock()
	defer p.m.Unlock()
	return calculateRealisedPNL(p.pnlHistory)
}

// Liquidate will update the positions stats to reflect its liquidation
func (p *PositionTracker) Liquidate(price decimal.Decimal, t time.Time) error {
	if p == nil {
		return fmt.Errorf("position tracker %w", common.ErrNilPointer)
	}
	p.m.Lock()
	defer p.m.Unlock()
	latest, err := p.GetLatestPNLSnapshot()
	if err != nil {
		return err
	}
	if !latest.Time.Equal(t) {
		return fmt.Errorf("%w cannot liquidate from a different time. PNL snapshot %v. Liquidation request on %v Status: %v", order.ErrCannotLiquidate, latest.Time, t, p.status)
	}
	p.status = order.Liquidated
	p.latestDirection = order.ClosePosition
	p.exposure = decimal.Zero
	p.realisedPNL = decimal.Zero
	p.unrealisedPNL = decimal.Zero
	_, err = upsertPNLEntry(p.pnlHistory, &PNLResult{
		Time:         t,
		Price:        price,
		Direction:    order.ClosePosition,
		IsLiquidated: true,
		IsOrder:      true,
		Status:       p.status,
	})

	return err
}

// GetLatestPNLSnapshot takes the latest pnl history value
// and returns it
func (p *PositionTracker) GetLatestPNLSnapshot() (PNLResult, error) {
	if len(p.pnlHistory) == 0 {
		return PNLResult{}, fmt.Errorf("%v %v %v %w", p.exchange, p.asset, p.contractPair, errNoPNLHistory)
	}
	return p.pnlHistory[len(p.pnlHistory)-1], nil
}

// TrackFundingDetails sets funding rates to a position
func (p *PositionTracker) TrackFundingDetails(d *fundingrate.HistoricalRates) error {
	if p == nil {
		return fmt.Errorf("position tracker %w", common.ErrNilPointer)
	}
	if d == nil {
		return fmt.Errorf("funding rate details %w", common.ErrNilPointer)
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.Asset, d.Pair)
	if err != nil {
		return err
	}
	p.m.Lock()
	defer p.m.Unlock()
	if p.exchange != d.Exchange ||
		p.asset != d.Asset ||
		!p.contractPair.Equal(d.Pair) {
		return fmt.Errorf("provided details %v %v %v %w %v %v %v tracker",
			d.Exchange, d.Asset, d.Pair, errDoesntMatch, p.exchange, p.asset, p.contractPair)
	}
	if err := common.StartEndTimeCheck(d.StartDate, d.EndDate); err != nil && !errors.Is(err, common.ErrStartEqualsEnd) {
		// start end being equal is valid if only one funding rate is retrieved
		return err
	}
	if len(p.pnlHistory) == 0 {
		return fmt.Errorf("%w for timeframe %v %v %v %v-%v", ErrNoPositionsFound, p.exchange, p.asset, p.contractPair, d.StartDate, d.EndDate)
	}
	if p.fundingRateDetails == nil {
		p.fundingRateDetails = &fundingrate.HistoricalRates{
			Exchange:              d.Exchange,
			Asset:                 d.Asset,
			Pair:                  d.Pair,
			StartDate:             d.StartDate,
			EndDate:               d.EndDate,
			LatestRate:            d.LatestRate,
			PredictedUpcomingRate: d.PredictedUpcomingRate,
			PaymentSum:            d.PaymentSum,
		}
	}
	rates := make([]fundingrate.Rate, 0, len(d.FundingRates))
fundingRates:
	for i := range d.FundingRates {
		if d.FundingRates[i].Time.Before(p.openingDate) ||
			(!p.closingDate.IsZero() && d.FundingRates[i].Time.After(p.closingDate)) {
			continue
		}
		for j := range p.fundingRateDetails.FundingRates {
			if !p.fundingRateDetails.FundingRates[j].Time.Equal(d.FundingRates[i].Time) {
				continue
			}
			p.fundingRateDetails.FundingRates[j] = d.FundingRates[i]
			continue fundingRates
		}
		rates = append(rates, d.FundingRates[i])
	}

	p.fundingRateDetails.FundingRates = append(p.fundingRateDetails.FundingRates, rates...)
	p.lastUpdated = time.Now()
	return nil
}

// TrackNewOrder knows how things are going for a given
// futures contract
func (p *PositionTracker) TrackNewOrder(d *order.Detail, isInitialOrder bool) error {
	if p == nil {
		return fmt.Errorf("position tracker %w", common.ErrNilPointer)
	}
	if d == nil {
		return fmt.Errorf("order %w", common.ErrNilPointer)
	}
	var err error
	d.Exchange, err = checkTrackerPrerequisitesLowerExchange(d.Exchange, d.AssetType, d.Pair)
	if err != nil {
		return err
	}
	p.m.Lock()
	defer p.m.Unlock()
	if isInitialOrder && len(p.pnlHistory) > 0 {
		return fmt.Errorf("%w received isInitialOrder = true with existing position", errCannotTrackInvalidParams)
	}
	if p.status.IsInactive() {
		for i := range p.longPositions {
			if p.longPositions[i].OrderID == d.OrderID {
				return nil
			}
		}
		for i := range p.shortPositions {
			if p.shortPositions[i].OrderID == d.OrderID {
				return nil
			}
		}
		// adding a new position to something that is already closed
		return fmt.Errorf("%w cannot process new order %v", ErrPositionClosed, d.OrderID)
	}
	if !p.contractPair.Equal(d.Pair) {
		return fmt.Errorf("%w pair '%v' received: '%v'",
			errOrderNotEqualToTracker, d.Pair, p.contractPair)
	}
	if p.exchange != d.Exchange {
		return fmt.Errorf("%w exchange '%v' received: '%v'",
			errOrderNotEqualToTracker, d.Exchange, p.exchange)
	}
	if p.asset != d.AssetType {
		return fmt.Errorf("%w asset '%v' received: '%v'",
			errOrderNotEqualToTracker, d.AssetType, p.asset)
	}

	if d.Side == order.UnknownSide {
		return order.ErrSideIsInvalid
	}
	if d.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if d.Date.IsZero() {
		return fmt.Errorf("%w for %v %v %v order ID: %v unset",
			errTimeUnset, d.Exchange, d.AssetType, d.Pair, d.OrderID)
	}
	if len(p.shortPositions) == 0 && len(p.longPositions) == 0 {
		p.openingPrice = decimal.NewFromFloat(d.Price)
		p.openingSize = decimal.NewFromFloat(d.Amount)
		p.openingDate = d.Date
	}

	var updated bool
	for i := range p.shortPositions {
		if p.shortPositions[i].OrderID != d.OrderID {
			continue
		}
		ord := p.shortPositions[i].Copy()
		err = ord.UpdateOrderFromDetail(d)
		if err != nil {
			return err
		}
		p.shortPositions[i] = ord
		updated = true
		p.lastUpdated = time.Now()
		break
	}
	for i := range p.longPositions {
		if p.longPositions[i].OrderID != d.OrderID {
			continue
		}
		ord := p.longPositions[i].Copy()
		err = ord.UpdateOrderFromDetail(d)
		if err != nil {
			return err
		}
		p.longPositions[i] = ord
		updated = true
		p.lastUpdated = time.Now()
		break
	}

	if !updated {
		if d.Side.IsShort() {
			p.shortPositions = append(p.shortPositions, d.Copy())
		} else {
			p.longPositions = append(p.longPositions, d.Copy())
		}
	}
	var shortSideAmount, longSideAmount decimal.Decimal
	for i := range p.shortPositions {
		shortSideAmount = shortSideAmount.Add(decimal.NewFromFloat(p.shortPositions[i].Amount))
	}
	for i := range p.longPositions {
		longSideAmount = longSideAmount.Add(decimal.NewFromFloat(p.longPositions[i].Amount))
	}

	if isInitialOrder {
		p.openingDirection = d.Side
		p.latestDirection = d.Side
	}

	var result *PNLResult
	var price, amount, leverage decimal.Decimal
	price = decimal.NewFromFloat(d.Price)
	amount = decimal.NewFromFloat(d.Amount)
	leverage = decimal.NewFromFloat(d.Leverage)
	cal := &PNLCalculatorRequest{
		Underlying:       p.underlying,
		Asset:            p.asset,
		OrderDirection:   d.Side,
		Leverage:         leverage,
		EntryPrice:       p.openingPrice,
		Amount:           amount,
		CurrentPrice:     price,
		Pair:             p.contractPair,
		Time:             d.Date,
		OpeningDirection: p.openingDirection,
		CurrentDirection: p.latestDirection,
		PNLHistory:       p.pnlHistory,
		Exposure:         p.exposure,
		Fee:              decimal.NewFromFloat(d.Fee),
		CalculateOffline: p.offlinePNLCalculation,
	}
	if len(p.pnlHistory) != 0 {
		cal.PreviousPrice = p.pnlHistory[len(p.pnlHistory)-1].Price
	}
	switch {
	case isInitialOrder:
		result = &PNLResult{
			IsOrder:       true,
			Time:          cal.Time,
			Price:         cal.CurrentPrice,
			Exposure:      cal.Amount,
			Fee:           cal.Fee,
			Direction:     cal.OpeningDirection,
			UnrealisedPNL: cal.Fee.Neg(),
		}
	case (cal.OrderDirection.IsShort() && cal.CurrentDirection.IsLong() || cal.OrderDirection.IsLong() && cal.CurrentDirection.IsShort()) && cal.Exposure.LessThan(amount):
		// latest order swaps directions!
		// split the order to calculate PNL from each direction
		first := cal.Exposure
		second := amount.Sub(cal.Exposure)
		baseFee := cal.Fee.Div(amount)
		cal.Fee = baseFee.Mul(first)
		cal.Amount = first
		result, err = p.PNLCalculation.CalculatePNL(context.TODO(), cal)
		if err != nil {
			return err
		}
		result.Status = p.status
		p.pnlHistory, err = upsertPNLEntry(cal.PNLHistory, result)
		if err != nil {
			return err
		}
		if cal.OrderDirection.IsLong() {
			cal.OrderDirection = order.Short
		} else if cal.OrderDirection.IsShort() {
			cal.OrderDirection = order.Long
		}
		if p.openingDirection.IsLong() {
			p.openingDirection = order.Short
		} else if p.openingDirection.IsShort() {
			p.openingDirection = order.Long
		}

		cal.Fee = baseFee.Mul(second)
		cal.Amount = second
		cal.EntryPrice = price
		cal.Time = cal.Time.Add(1)
		cal.PNLHistory = p.pnlHistory
		result, err = p.PNLCalculation.CalculatePNL(context.TODO(), cal)
	default:
		result, err = p.PNLCalculation.CalculatePNL(context.TODO(), cal)
	}
	if err != nil {
		if !errors.Is(err, ErrPositionLiquidated) {
			return err
		}
		result.UnrealisedPNL = decimal.Zero
		result.RealisedPNLBeforeFees = decimal.Zero
		p.closingPrice = result.Price
		p.closingDate = result.Time
		p.status = order.Closed
	}
	result.Status = p.status
	p.pnlHistory, err = upsertPNLEntry(p.pnlHistory, result)
	if err != nil {
		return err
	}
	p.unrealisedPNL = result.UnrealisedPNL

	switch {
	case longSideAmount.GreaterThan(shortSideAmount):
		p.latestDirection = order.Long
	case shortSideAmount.GreaterThan(longSideAmount):
		p.latestDirection = order.Short
	default:
		p.latestDirection = order.ClosePosition
	}

	if p.latestDirection.IsLong() {
		p.exposure = longSideAmount.Sub(shortSideAmount)
	} else {
		p.exposure = shortSideAmount.Sub(longSideAmount)
	}

	if p.exposure.Equal(decimal.Zero) {
		p.status = order.Closed
		p.closingPrice = decimal.NewFromFloat(d.Price)
		p.realisedPNL = calculateRealisedPNL(p.pnlHistory)
		p.unrealisedPNL = decimal.Zero
		p.pnlHistory[len(p.pnlHistory)-1].RealisedPNL = p.realisedPNL
		p.pnlHistory[len(p.pnlHistory)-1].UnrealisedPNL = p.unrealisedPNL
		p.pnlHistory[len(p.pnlHistory)-1].Direction = p.latestDirection
		p.closingDate = d.Date
	} else if p.exposure.IsNegative() {
		if p.latestDirection.IsLong() {
			p.latestDirection = order.Short
		} else {
			p.latestDirection = order.Long
		}
		p.exposure = p.exposure.Abs()
	}
	return nil
}

// GetCurrencyForRealisedPNL is a generic handling of determining the asset
// to assign realised PNL into, which is just itself
func (p *PNLCalculator) GetCurrencyForRealisedPNL(realisedAsset asset.Item, realisedPair currency.Pair) (currency.Code, asset.Item, error) {
	return realisedPair.Base, realisedAsset, nil
}

// CalculatePNL this is a localised generic way of calculating open
// positions' worth, it is an implementation of the PNLCalculation interface
func (p *PNLCalculator) CalculatePNL(_ context.Context, calc *PNLCalculatorRequest) (*PNLResult, error) {
	if calc == nil {
		return nil, ErrNilPNLCalculator
	}
	var previousPNL *PNLResult
	if len(calc.PNLHistory) > 0 {
		for i := len(calc.PNLHistory) - 1; i >= 0; i-- {
			if calc.PNLHistory[i].Time.Equal(calc.Time) || !calc.PNLHistory[i].IsOrder {
				continue
			}
			previousPNL = &calc.PNLHistory[i]
			break
		}
	}
	var prevExposure decimal.Decimal
	if previousPNL != nil {
		prevExposure = previousPNL.Exposure
	}
	var currentExposure, realisedPNL, unrealisedPNL, first, second decimal.Decimal
	if calc.OpeningDirection.IsLong() {
		first = calc.CurrentPrice
		if previousPNL != nil {
			second = previousPNL.Price
		}
	} else if calc.OpeningDirection.IsShort() {
		if previousPNL != nil {
			first = previousPNL.Price
		}
		second = calc.CurrentPrice
	}
	switch {
	case calc.OpeningDirection.IsShort() && calc.OrderDirection.IsShort(),
		calc.OpeningDirection.IsLong() && calc.OrderDirection.IsLong():
		// appending to your position
		currentExposure = prevExposure.Add(calc.Amount)
		unrealisedPNL = currentExposure.Mul(first.Sub(second))
	case calc.OpeningDirection.IsShort() && calc.OrderDirection.IsLong(),
		calc.OpeningDirection.IsLong() && calc.OrderDirection.IsShort():
		// selling/closing your position by "amount"
		currentExposure = prevExposure.Sub(calc.Amount)
		unrealisedPNL = currentExposure.Mul(first.Sub(second))
		realisedPNL = calc.Amount.Mul(first.Sub(second))
	default:
		return nil, fmt.Errorf("%w openinig direction: '%v' order direction: '%v' exposure: '%v'", errCannotCalculateUnrealisedPNL, calc.OpeningDirection, calc.OrderDirection, currentExposure)
	}
	totalFees := calc.Fee
	for i := range calc.PNLHistory {
		totalFees = totalFees.Add(calc.PNLHistory[i].Fee)
	}
	if !unrealisedPNL.IsZero() {
		unrealisedPNL = unrealisedPNL.Sub(totalFees)
	}

	response := &PNLResult{
		IsOrder:               true,
		Time:                  calc.Time,
		UnrealisedPNL:         unrealisedPNL,
		RealisedPNLBeforeFees: realisedPNL,
		Price:                 calc.CurrentPrice,
		Exposure:              currentExposure,
		Fee:                   calc.Fee,
		Direction:             calc.CurrentDirection,
	}

	return response, nil
}

// calculateRealisedPNL calculates the total realised PNL
// based on PNL history, minus fees
func calculateRealisedPNL(pnlHistory []PNLResult) decimal.Decimal {
	var realisedPNL, totalFees decimal.Decimal
	for i := range pnlHistory {
		if !pnlHistory[i].IsOrder {
			continue
		}
		realisedPNL = realisedPNL.Add(pnlHistory[i].RealisedPNLBeforeFees)
		totalFees = totalFees.Add(pnlHistory[i].Fee)
	}
	return realisedPNL.Sub(totalFees)
}

// upsertPNLEntry upserts an entry to PNLHistory field
// with some basic checks
func upsertPNLEntry(pnlHistory []PNLResult, entry *PNLResult) ([]PNLResult, error) {
	if entry.Time.IsZero() {
		return nil, errTimeUnset
	}
	for i := range pnlHistory {
		if !entry.Time.Equal(pnlHistory[i].Time) {
			continue
		}
		pnlHistory[i].UnrealisedPNL = entry.UnrealisedPNL
		pnlHistory[i].RealisedPNL = entry.RealisedPNL
		pnlHistory[i].RealisedPNLBeforeFees = entry.RealisedPNLBeforeFees
		pnlHistory[i].Exposure = entry.Exposure
		pnlHistory[i].Direction = entry.Direction
		pnlHistory[i].Price = entry.Price
		pnlHistory[i].Status = entry.Status
		pnlHistory[i].Fee = entry.Fee
		if entry.IsOrder {
			pnlHistory[i].IsOrder = true
		}
		if entry.IsLiquidated {
			pnlHistory[i].IsLiquidated = true
		}
		return pnlHistory, nil
	}
	pnlHistory = append(pnlHistory, *entry)
	sort.Slice(pnlHistory, func(i, j int) bool {
		return pnlHistory[i].Time.Before(pnlHistory[j].Time)
	})
	return pnlHistory, nil
}

// CheckFundingRatePrerequisites is a simple check to see if the requested data meets the prerequisite
func CheckFundingRatePrerequisites(getFundingData, includePredicted, includePayments bool) error {
	if !getFundingData && includePredicted {
		return fmt.Errorf("%w please include in request to get predicted funding rates", ErrGetFundingDataRequired)
	}
	if !getFundingData && includePayments {
		return fmt.Errorf("%w please include in request to get predicted funding rates", ErrGetFundingDataRequired)
	}
	return nil
}

// checkTrackerPrerequisitesLowerExchange is a common set of checks for futures position tracking
func checkTrackerPrerequisitesLowerExchange(exch string, item asset.Item, cp currency.Pair) (string, error) {
	if exch == "" {
		return "", errExchangeNameEmpty
	}
	exch = strings.ToLower(exch)
	if !item.IsFutures() {
		return exch, fmt.Errorf("%w %v %v %v", ErrNotFuturesAsset, exch, item, cp)
	}
	if cp.IsEmpty() {
		return exch, fmt.Errorf("%w %v %v", order.ErrPairIsEmpty, exch, item)
	}
	return exch, nil
}

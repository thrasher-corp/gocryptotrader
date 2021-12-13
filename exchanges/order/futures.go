package order

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupPositionController creates a futures order tracker for a specific exchange
func SetupPositionController(setup *PositionControllerSetup) (*PositionController, error) {
	if setup == nil {
		return nil, errNilSetup
	}
	if setup.Exchange == "" {
		return nil, errExchangeNameEmpty
	}
	if !setup.Asset.IsValid() || !setup.Asset.IsFutures() {
		return nil, errNotFutureAsset
	}
	if setup.Pair.IsEmpty() {
		return nil, ErrPairIsEmpty
	}
	if setup.Underlying.IsEmpty() {
		return nil, errEmptyUnderlying
	}
	if setup.PNLCalculator == nil {
		return nil, errMissingPNLCalculationFunctions
	}
	return &PositionController{
		exchange:              strings.ToLower(setup.Exchange),
		asset:                 setup.Asset,
		pair:                  setup.Pair,
		underlying:            setup.Underlying,
		pnlCalculation:        setup.PNLCalculator,
		offlinePNLCalculation: setup.OfflineCalculation,
		orderPositions:        make(map[string]*PositionTracker),
	}, nil
}

func (e *PositionController) GetPositions() []*PositionTracker {
	return e.positions
}

// TrackNewOrder upserts an order to the tracker and updates position
// status and exposure. PNL is calculated separately as it requires mark prices
func (e *PositionController) TrackNewOrder(d *Detail) error {
	if d == nil {
		return ErrSubmissionIsNil
	}
	if d.AssetType != e.asset {
		return errAssetMismatch
	}
	if tracker, ok := e.orderPositions[d.ID]; ok {
		// this has already been associated
		// update the tracker
		return tracker.TrackNewOrder(d)
	}
	if len(e.positions) > 0 {
		for i := range e.positions {
			if e.positions[i].status == Open && i != len(e.positions)-1 {
				return fmt.Errorf("%w %v at position %v/%v", errPositionDiscrepancy, e.positions[i], i, len(e.positions)-1)
			}
		}
		if e.positions[len(e.positions)-1].status == Open {
			err := e.positions[len(e.positions)-1].TrackNewOrder(d)
			if err != nil && !errors.Is(err, errPositionClosed) {
				return err
			}
			e.orderPositions[d.ID] = e.positions[len(e.positions)-1]
			return nil
		}
	}
	tracker, err := e.SetupPositionTracker(d.AssetType, d.Pair, d.Pair.Base)
	if err != nil {
		return err
	}
	e.positions = append(e.positions, tracker)

	err = tracker.TrackNewOrder(d)
	if err != nil {
		return err
	}
	e.orderPositions[d.ID] = tracker
	return nil
}

// SetupPositionTracker creates a new position tracker to track n futures orders
// until the position(s) are closed
func (e *PositionController) SetupPositionTracker(item asset.Item, pair currency.Pair, underlying currency.Code) (*PositionTracker, error) {
	if e.exchange == "" {
		return nil, errExchangeNameEmpty
	}
	if !item.IsValid() || !item.IsFutures() {
		return nil, errNotFutureAsset
	}
	if pair.IsEmpty() {
		return nil, ErrPairIsEmpty
	}

	resp := &PositionTracker{
		exchange:        strings.ToLower(e.exchange),
		asset:           item,
		contractPair:    pair,
		underlyingAsset: underlying,
		status:          Open,
		pnlCalculation:  e.pnlCalculation,
	}
	if e.pnlCalculation == nil {
		log.Warnf(log.OrderMgr, "no pnl calculation functions supplied for %v %v %v, using default calculations", e.exchange, e.asset, e.pair)
		e.pnlCalculation = resp
	}
	return resp, nil
}

// CalculatePNL this is a localised generic way of calculating open
// positions' worth
func (p *PositionTracker) CalculatePNL(calc *PNLCalculator) (*PNLResult, error) {
	result := &PNLResult{}
	var amount, price decimal.Decimal
	if calc.OrderBasedCalculation != nil {
		result.Time = calc.OrderBasedCalculation.Date
		amount = decimal.NewFromFloat(calc.OrderBasedCalculation.Amount)
		price = decimal.NewFromFloat(calc.OrderBasedCalculation.Price)
		if (p.currentDirection == Long && calc.OrderBasedCalculation.Side.IsShort()) ||
			(p.currentDirection == Short && calc.OrderBasedCalculation.Side.IsLong()) {
			currPos := p.exposure.Mul(price)
			result.UnrealisedPNL = currPos.Sub(price.Mul(amount))

		} else if (p.currentDirection == Long && calc.OrderBasedCalculation.Side.IsLong()) ||
			(p.currentDirection == Short && calc.OrderBasedCalculation.Side.IsShort()) {
			currPos := p.exposure.Mul(price)
			result.UnrealisedPNL = currPos.Add(price.Mul(amount))
		}
		return result, nil
	} else if calc.TimeBasedCalculation != nil {
		price = decimal.NewFromFloat(calc.TimeBasedCalculation.CurrentPrice)
		result.UnrealisedPNL = p.exposure.Mul(price)
		return result, nil
	}

	return nil, errMissingPNLCalculationFunctions
}

// TrackPNLByTime calculates the PNL based on a position tracker's exposure
// and current pricing. Adds the entry to PNL history to track over time
func (p *PositionTracker) TrackPNLByTime(t time.Time, currentPrice float64) error {
	defer func() {
		p.latestPrice = decimal.NewFromFloat(currentPrice)
	}()
	pnl, err := p.pnlCalculation.CalculatePNL(&PNLCalculator{
		TimeBasedCalculation: &TimeBasedCalculation{
			currentPrice,
		},
	})
	if err != nil {
		return err
	}
	return p.UpsertPNLEntry(PNLResult{
		Time:          t,
		UnrealisedPNL: pnl.UnrealisedPNL,
		RealisedPNL:   pnl.RealisedPNL,
	})
}

func (p *PositionTracker) GetRealisedPNL() (decimal.Decimal, error) {
	if p.status != Closed {
		return decimal.Zero, errors.New("position not closed")
	}
	return p.realisedPNL, nil
}

// TrackNewOrder knows how things are going for a given
// futures contract
func (p *PositionTracker) TrackNewOrder(d *Detail) error {
	if p.status == Closed {
		return errPositionClosed
	}
	if d == nil {
		return ErrSubmissionIsNil
	}
	if !p.contractPair.Equal(d.Pair) {
		return fmt.Errorf("%w pair '%v' received: '%v'", errOrderNotEqualToTracker, d.Pair, p.contractPair)
	}
	if p.exchange != strings.ToLower(d.Exchange) {
		return fmt.Errorf("%w exchange '%v' received: '%v'", errOrderNotEqualToTracker, d.Exchange, p.exchange)
	}
	if p.asset != d.AssetType {
		return fmt.Errorf("%w asset '%v' received: '%v'", errOrderNotEqualToTracker, d.AssetType, p.asset)
	}
	if d.Side == "" {
		return ErrSideIsInvalid
	}
	if d.ID == "" {
		return ErrOrderIDNotSet
	}
	if len(p.shortPositions) == 0 && len(p.longPositions) == 0 {
		p.entryPrice = decimal.NewFromFloat(d.Price)
	}

	for i := range p.shortPositions {
		if p.shortPositions[i].ID == d.ID {
			// update, not overwrite
			ord := p.shortPositions[i].Copy()
			ord.UpdateOrderFromDetail(d)
			p.shortPositions[i] = ord
			break
		}
	}
	for i := range p.longPositions {
		if p.longPositions[i].ID == d.ID {
			ord := p.longPositions[i].Copy()
			ord.UpdateOrderFromDetail(d)
			p.longPositions[i] = ord
			break
		}
	}

	if d.Side.IsShort() {
		p.shortPositions = append(p.shortPositions, d.Copy())
	} else {
		p.longPositions = append(p.longPositions, d.Copy())
	}
	var shortSide, longSide, averageLeverage decimal.Decimal

	for i := range p.shortPositions {
		shortSide = shortSide.Add(decimal.NewFromFloat(p.shortPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(p.shortPositions[i].Leverage)
	}
	for i := range p.longPositions {
		longSide = longSide.Add(decimal.NewFromFloat(p.longPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(p.longPositions[i].Leverage)
	}

	averageLeverage.Div(decimal.NewFromInt(int64(len(p.shortPositions))).Add(decimal.NewFromInt(int64(len(p.longPositions)))))

	switch {
	case longSide.GreaterThan(shortSide):
		p.currentDirection = Long
	case shortSide.GreaterThan(longSide):
		p.currentDirection = Short
	default:
		p.currentDirection = UnknownSide
	}

	result, err := p.CalculatePNL(&PNLCalculator{OrderBasedCalculation: d})
	if err != nil {
		return err
	}
	err = p.UpsertPNLEntry(*result)
	if err != nil {
		return err
	}

	if p.currentDirection.IsLong() {
		p.exposure = longSide.Sub(shortSide)
	} else {
		p.exposure = shortSide.Sub(longSide)
	}
	if p.exposure.Equal(decimal.Zero) {
		p.status = Closed
		p.closingPrice = decimal.NewFromFloat(d.Price)
		p.realisedPNL = p.unrealisedPNL
		p.unrealisedPNL = decimal.Zero
	} else if p.exposure.IsNegative() {
		if p.currentDirection.IsLong() {
			p.currentDirection = Short
		} else {
			p.currentDirection = Long
		}
		p.exposure = p.exposure.Abs()
	}
	return nil
}

func (p *PositionTracker) TrackPrice(price decimal.Decimal) decimal.Decimal {
	return p.exposure.Mul(price)
}

// UpsertPNLEntry upserts an entry to PNLHistory field
// with some basic checks
func (p *PositionTracker) UpsertPNLEntry(entry PNLResult) error {
	if entry.Time.IsZero() {
		return errTimeUnset
	}
	for i := range p.pnlHistory {
		if entry.Time.Equal(p.pnlHistory[i].Time) {
			p.pnlHistory[i] = entry
			return nil
		}
	}
	p.pnlHistory = append(p.pnlHistory, entry)
	return nil
}

// IsShort returns if the side is short
func (s Side) IsShort() bool {
	return s == Short || s == Sell
}

// IsLong returns if the side is long
func (s Side) IsLong() bool {
	return s == Long || s == Buy
}

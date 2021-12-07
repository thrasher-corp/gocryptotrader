package order

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// SetupExchangeAssetPositionTracker creates a futures order tracker for a specific exchange
func SetupExchangeAssetPositionTracker(exch string, item asset.Item, offlineCalculation bool, calculation PNLManagement) (*ExchangeAssetPositionTracker, error) {
	if exch == "" {
		return nil, errExchangeNameEmpty
	}
	if !item.IsValid() || !item.IsFutures() {
		return nil, errNotFutureAsset
	}
	if calculation == nil {
		return nil, errMissingPNLCalculationFunctions
	}
	return &ExchangeAssetPositionTracker{
		Exchange:              strings.ToLower(exch),
		Asset:                 item,
		PNLCalculation:        calculation,
		OfflinePNLCalculation: offlineCalculation,
	}, nil
}

// TrackNewOrder upserts an order to the tracker and updates position
// status and exposure. PNL is calculated separately as it requires mark prices
func (e *ExchangeAssetPositionTracker) TrackNewOrder(d *Detail) error {
	if d == nil {
		return ErrSubmissionIsNil
	}
	if d.AssetType != e.Asset {
		return errAssetMismatch
	}
	if len(e.Positions) > 0 {
		for i := range e.Positions {
			if e.Positions[i].Status == Open && i != len(e.Positions)-1 {
				return fmt.Errorf("%w %v at position %v/%v", errPositionDiscrepancy, e.Positions[i], i, len(e.Positions)-1)
			}
		}
		err := e.Positions[len(e.Positions)-1].TrackNewOrder(d)
		if !errors.Is(err, errPositionClosed) {
			return err
		}
	}
	tracker, err := e.SetupPositionTracker(d.AssetType, d.Pair, d.Pair.Base)
	if err != nil {
		return err
	}
	e.Positions = append(e.Positions, tracker)

	return tracker.TrackNewOrder(d)
}

// SetupPositionTracker creates a new position tracker to track n futures orders
// until the position(s) are closed
func (e *ExchangeAssetPositionTracker) SetupPositionTracker(item asset.Item, pair currency.Pair, underlying currency.Code) (*PositionTracker, error) {
	if e.Exchange == "" {
		return nil, errExchangeNameEmpty
	}
	if e.PNLCalculation == nil {
		return nil, errMissingPNLCalculationFunctions
	}
	if !item.IsValid() || !item.IsFutures() {
		return nil, errNotFutureAsset
	}
	if pair.IsEmpty() {
		return nil, ErrPairIsEmpty
	}

	return &PositionTracker{
		Exchange:              strings.ToLower(e.Exchange),
		Asset:                 item,
		ContractPair:          pair,
		UnderlyingAsset:       underlying,
		Status:                Open,
		PNLCalculation:        e.PNLCalculation,
		OfflinePNLCalculation: e.OfflinePNLCalculation,
	}, nil
}

// TrackPNL calculates the PNL based on a position tracker's exposure
// and current pricing. Adds the entry to PNL history to track over time
func (p *PositionTracker) TrackPNL(t time.Time, markPrice, prevMarkPrice decimal.Decimal) error {
	pnl, err := p.PNLCalculation.CalculatePNL(&PNLCalculator{
		CalculateOffline: p.OfflinePNLCalculation,
		Amount:           p.Exposure.InexactFloat64(),
		MarkPrice:        markPrice.InexactFloat64(),
		PrevMarkPrice:    prevMarkPrice.InexactFloat64(),
	})
	if err != nil {
		return err
	}
	return p.UpsertPNLEntry(PNLHistory{
		Time:          t,
		UnrealisedPNL: pnl.UnrealisedPNL,
	})
}

// TrackNewOrder knows how things are going for a given
// futures contract
func (p *PositionTracker) TrackNewOrder(d *Detail) error {
	if p.Status == Closed {
		return errPositionClosed
	}
	if d == nil {
		return ErrSubmissionIsNil
	}
	if !p.ContractPair.Equal(d.Pair) {
		return fmt.Errorf("%w pair '%v' received: '%v'", errOrderNotEqualToTracker, d.Pair, p.ContractPair)
	}
	if p.Exchange != strings.ToLower(d.Exchange) {
		return fmt.Errorf("%w exchange '%v' received: '%v'", errOrderNotEqualToTracker, d.Exchange, p.Exchange)
	}
	if p.Asset != d.AssetType {
		return fmt.Errorf("%w asset '%v' received: '%v'", errOrderNotEqualToTracker, d.AssetType, p.Asset)
	}
	if d.Side == "" {
		return ErrSideIsInvalid
	}
	if d.ID == "" {
		return ErrOrderIDNotSet
	}
	if len(p.ShortPositions) == 0 && len(p.LongPositions) == 0 {
		p.EntryPrice = decimal.NewFromFloat(d.Price)
	}

	for i := range p.ShortPositions {
		if p.ShortPositions[i].ID == d.ID {
			// update, not overwrite
			ord := p.ShortPositions[i].Copy()
			ord.UpdateOrderFromDetail(d)
			p.ShortPositions[i] = ord
			break
		}
	}

	if d.Side.IsShort() {
		p.ShortPositions = append(p.ShortPositions, d.Copy())
	} else {
		p.LongPositions = append(p.LongPositions, d.Copy())
	}
	var shortSide, longSide, averageLeverage decimal.Decimal

	for i := range p.ShortPositions {
		shortSide = shortSide.Add(decimal.NewFromFloat(p.ShortPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(p.ShortPositions[i].Leverage)
	}
	for i := range p.LongPositions {
		longSide = longSide.Add(decimal.NewFromFloat(p.LongPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(p.LongPositions[i].Leverage)
	}

	averageLeverage.Div(decimal.NewFromInt(int64(len(p.ShortPositions))).Add(decimal.NewFromInt(int64(len(p.LongPositions)))))

	switch {
	case longSide.GreaterThan(shortSide):
		p.CurrentDirection = Long
	case shortSide.GreaterThan(longSide):
		p.CurrentDirection = Short
	default:
		p.CurrentDirection = UnknownSide
	}
	if p.CurrentDirection.IsLong() {
		p.Exposure = longSide.Sub(shortSide)
	} else {
		p.Exposure = shortSide.Sub(longSide)
	}
	if p.Exposure.Equal(decimal.Zero) {
		// the order is closed
		p.Status = Closed
		p.ClosingPrice = decimal.NewFromFloat(d.Price)
		p.RealisedPNL = p.UnrealisedPNL
		p.UnrealisedPNL = decimal.Zero
	}
	if p.Exposure.IsNegative() {
		// tracking here has changed!
		if p.CurrentDirection.IsLong() {
			p.CurrentDirection = Short
		} else {
			p.CurrentDirection = Long
		}
		p.Exposure = p.Exposure.Abs()
	}
	return nil
}

// UpsertPNLEntry upserts an entry to PNLHistory field
// with some basic checks
func (p *PositionTracker) UpsertPNLEntry(entry PNLHistory) error {
	if entry.Time.IsZero() {
		return errTimeUnset
	}
	for i := range p.PNLHistory {
		if entry.Time.Equal(p.PNLHistory[i].Time) {
			p.PNLHistory[i] = entry
			return nil
		}
	}
	p.PNLHistory = append(p.PNLHistory, entry)
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

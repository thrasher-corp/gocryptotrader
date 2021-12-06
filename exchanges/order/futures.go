package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// IsShort returns if the side is short
func (s Side) IsShort() bool {
	return s == Short || s == Sell
}

// IsLong returns if the side is long
func (s Side) IsLong() bool {
	return s == Long || s == Buy
}

func SetupFuturesTracker(exch string, item asset.Item, pair currency.Pair, underlying currency.Code, calculation exchange.PNLManagement) (*FuturesTracker, error) {
	if exch == "" {
		return nil, errExchangeNameEmpty
	}
	if !item.IsFutures() {
		return nil, errNotFutureAsset
	}
	if pair.IsEmpty() {
		return nil, ErrPairIsEmpty
	}

	return &FuturesTracker{
		Exchange:        exch,
		Asset:           item,
		ContractPair:    pair,
		UnderlyingAsset: underlying,
		Status:          Open,
		PNLCalculation:  calculation,
	}, nil
}

var errOrderNotEqualToTracker = errors.New("order does not match tracker data")
var errPositionClosed = errors.New("the position is closed, time for a new one")

func (f *FuturesTracker) TrackPNL(t time.Time, markPrice, prevMarkPrice decimal.Decimal) error {
	pnl, err := f.PNLCalculation.CalculatePNL(&exchange.PNLCalculator{
		CalculateOffline: f.OfflinePNLCalculation,
		Amount:           f.Exposure.InexactFloat64(),
		MarkPrice:        markPrice.InexactFloat64(),
		PrevMarkPrice:    prevMarkPrice.InexactFloat64(),
	})
	if err != nil {
		return err
	}
	f.UpsertPNLEntry(PNLHistory{
		Time:          t,
		UnrealisedPNL: pnl.UnrealisedPNL,
	})
	return nil
}

// TrackNewOrder knows how things are going for a given
// futures contract
func (f *FuturesTracker) TrackNewOrder(d *Detail) error {
	if f.Status == Closed {
		return errPositionClosed
	}
	if d == nil {
		return ErrSubmissionIsNil
	}
	if !f.ContractPair.Equal(d.Pair) {
		return fmt.Errorf("%w pair '%v' received: '%v'", errOrderNotEqualToTracker, d.Pair, f.ContractPair)
	}
	if f.Exchange != d.Exchange {
		return fmt.Errorf("%w exchange '%v' received: '%v'", errOrderNotEqualToTracker, d.Exchange, f.Exchange)
	}
	if f.Asset != d.AssetType {
		return fmt.Errorf("%w asset '%v' received: '%v'", errOrderNotEqualToTracker, d.AssetType, f.Asset)
	}
	if d.Side == "" {
		return ErrSideIsInvalid
	}
	if len(f.ShortPositions) == 0 && len(f.LongPositions) == 0 {
		f.EntryPrice = decimal.NewFromFloat(d.Price)
	}

	for i := range f.ShortPositions {
		if f.ShortPositions[i].ID == d.ID {
			f.ShortPositions[i] = d.Copy()
			break
		}
	}

	if d.Side.IsShort() {
		f.ShortPositions = append(f.ShortPositions, d.Copy())
	} else {
		f.LongPositions = append(f.LongPositions, d.Copy())
	}
	var shortSide, longSide, averageLeverage decimal.Decimal

	for i := range f.ShortPositions {
		shortSide = shortSide.Add(decimal.NewFromFloat(f.ShortPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(f.ShortPositions[i].Leverage)
	}
	for i := range f.LongPositions {
		longSide = longSide.Add(decimal.NewFromFloat(f.LongPositions[i].Amount))
		averageLeverage = decimal.NewFromFloat(f.LongPositions[i].Leverage)
	}

	averageLeverage.Div(decimal.NewFromInt(int64(len(f.ShortPositions))).Add(decimal.NewFromInt(int64(len(f.LongPositions)))))

	switch {
	case longSide.GreaterThan(shortSide):
		f.CurrentDirection = Long
	case shortSide.GreaterThan(longSide):
		f.CurrentDirection = Short
	default:
		f.CurrentDirection = UnknownSide
	}
	if f.CurrentDirection.IsLong() {
		f.UnrealisedPNL = longSide.Sub(shortSide)
	} else {
		f.UnrealisedPNL = shortSide.Sub(longSide)
	}
	if f.Exposure.Equal(decimal.Zero) {
		// the order is closed
		f.Status = Closed
		f.ClosingPrice = decimal.NewFromFloat(d.Price)
		f.RealisedPNL = f.UnrealisedPNL
		f.UnrealisedPNL = decimal.Zero
	}
	if f.Exposure.IsNegative() {
		// tracking here has changed!
		if f.CurrentDirection.IsLong() {
			f.CurrentDirection = Short
		} else {
			f.CurrentDirection = Long
		}
	}
	f.PNLCalculation.CalculatePNL(&exchange.PNLCalculator{
		CalculateOffline: f.OfflinePNLCalculation,
		Amount:           f.Exposure.InexactFloat64(),
		MarkPrice:        0,
		PrevMarkPrice:    0,
	})
	f.UpsertPNLEntry(PNLHistory{
		Time:          time.Time{},
		UnrealisedPNL: decimal.Decimal{},
		RealisedPNL:   decimal.Decimal{},
	})

	return nil
}

// UpsertPNLEntry upserts an entry to PNLHistory field
// with some basic checks
func (f *FuturesTracker) UpsertPNLEntry(entry PNLHistory) {
	if entry.Time.IsZero() {
		return
	}
	for i := range f.PNLHistory {
		if entry.Time.Equal(f.PNLHistory[i].Time) {
			f.PNLHistory[i] = entry
			return
		}
	}
	f.PNLHistory = append(f.PNLHistory, entry)
}

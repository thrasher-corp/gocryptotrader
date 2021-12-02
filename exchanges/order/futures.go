package order

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

// TrackNewOrder knows how things are going for a given
// futures contract
func (f *FuturesTracker) TrackNewOrder(d *Detail) error {
	if !f.ContractPair.Equal(d.Pair) {
		return errors.New("not the same")
	}
	if f.Exchange != d.Exchange {
		return errors.New("not the same")
	}
	if f.Asset != d.AssetType {
		return errors.New("not the same")
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
	var shortSide, longSide decimal.Decimal
	for i := range f.ShortPositions {
		shortSide = shortSide.Add(decimal.NewFromFloat(f.ShortPositions[i].Amount))
	}
	for i := range f.LongPositions {
		longSide = shortSide.Add(decimal.NewFromFloat(f.ShortPositions[i].Amount))
	}
	if f.CurrentDirection.IsLong() {
		f.Exposure = longSide.Sub(shortSide)
	} else {
		f.Exposure = shortSide.Sub(longSide)
	}
	if f.Exposure.Equal(decimal.Zero) {
		// the order is closed
		f.Status = Closed
	} else {
		f.Status = Open
	}
	if f.Exposure.IsNegative() {
		// tracking here has changed!
		if f.CurrentDirection.IsLong() {
			f.CurrentDirection = Short
		} else {
			f.CurrentDirection = Long
		}
	}

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

// FuturesTracker order is a concept which holds both the opening and closing orders
// for a futures contract. This allows for PNL calculations
type FuturesTracker struct {
	Exchange         string
	Asset            asset.Item
	ContractPair     currency.Pair
	UnderlyingAsset  currency.Code
	Exposure         decimal.Decimal
	CurrentDirection Side
	Status           Status
	UnrealisedPNL    decimal.Decimal
	RealisedPNL      decimal.Decimal
	ShortPositions   []Detail
	LongPositions    []Detail
	PNLHistory       []PNLHistory
}

// PNLHistory tracks how a futures contract
// pnl is going over the history of exposure
type PNLHistory struct {
	Time          time.Time
	UnrealisedPNL decimal.Decimal
	RealisedPNL   decimal.Decimal
}

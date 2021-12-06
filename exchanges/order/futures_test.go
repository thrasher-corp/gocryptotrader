package order

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestTrackNewOrder(t *testing.T) {
	exch := "test"
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	f, err := SetupFuturesTracker(exch, item, pair, pair.Base)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = f.TrackNewOrder(nil)
	if !errors.Is(err, ErrSubmissionIsNil) {
		t.Error(err)
	}
	err = f.TrackNewOrder(&Detail{})
	if !errors.Is(err, errOrderNotEqualToTracker) {
		t.Error(err)
	}

	od := &Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		ID:        "1",
		Price:     1337,
	}
	err = f.TrackNewOrder(od)
	if !errors.Is(err, ErrSideIsInvalid) {
		t.Error(err)
	}

	od.Side = Long
	od.Amount = 1
	od.ID = "2"
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if !f.EntryPrice.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected 1337, received %v", f.EntryPrice)
	}
	if len(f.LongPositions) != 1 {
		t.Error("expected a long")
	}
	if f.CurrentDirection != Long {
		t.Error("expected recognition that its long")
	}
	if f.Exposure.InexactFloat64() != od.Amount {
		t.Error("expected 1")
	}

	od.Amount = 0.4
	od.Side = Short
	od.ID = "3"
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.ShortPositions) != 1 {
		t.Error("expected a short")
	}
	if f.CurrentDirection != Long {
		t.Error("expected recognition that its long")
	}
	if f.Exposure.InexactFloat64() != 0.6 {
		t.Error("expected 0.6")
	}
	od.Amount = 0.8
	od.Side = Short
	od.ID = "4"
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	if f.CurrentDirection != Short {
		t.Error("expected recognition that its short")
	}
	if !f.Exposure.Equal(decimal.NewFromFloat(0.2)) {
		t.Errorf("expected %v received %v", 0.2, f.Exposure)
	}

	od.ID = "5"
	od.Side = Long
	od.Amount = 0.2
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if f.CurrentDirection != UnknownSide {
		t.Error("expected recognition that its unknown")
	}
	if f.Status != Closed {
		t.Error("expected closed position")
	}

	err = f.TrackNewOrder(od)
	if !errors.Is(err, errPositionClosed) {
		t.Error(err)
	}
	if f.CurrentDirection != UnknownSide {
		t.Error("expected recognition that its unknown")
	}
	if f.Status != Closed {
		t.Error("expected closed position")
	}
}

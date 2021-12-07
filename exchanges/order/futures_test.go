package order

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type FakePNL struct {
	err    error
	result *PNLResult
}

func (f *FakePNL) CalculatePNL(*PNLCalculator) (*PNLResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestTrackPNL(t *testing.T) {
	t.Parallel()
	exch := "test"
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	fPNL := &FakePNL{
		result: &PNLResult{},
	}
	e := ExchangeAssetPositionTracker{
		Exchange:       exch,
		PNLCalculation: fPNL,
	}
	f, err := e.SetupPositionTracker(item, pair, pair.Base)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	err = f.TrackPNL(time.Now(), decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	fPNL.err = errMissingPNLCalculationFunctions
	err = f.TrackPNL(time.Now(), decimal.Zero, decimal.Zero)
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}
}

func TestUpsertPNLEntry(t *testing.T) {
	t.Parallel()
	f := &PositionTracker{}
	err := f.UpsertPNLEntry(PNLHistory{})
	if !errors.Is(err, errTimeUnset) {
		t.Error(err)
	}
	tt := time.Now()
	err = f.UpsertPNLEntry(PNLHistory{Time: tt})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.PNLHistory) != 1 {
		t.Errorf("expected 1 received %v", len(f.PNLHistory))
	}

	err = f.UpsertPNLEntry(PNLHistory{Time: tt})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.PNLHistory) != 1 {
		t.Errorf("expected 1 received %v", len(f.PNLHistory))
	}
}

func TestTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := "test"
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	e := ExchangeAssetPositionTracker{
		Exchange:       "test",
		PNLCalculation: &FakePNL{},
	}
	f, err := e.SetupPositionTracker(item, pair, pair.Base)
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

func TestSetupFuturesTracker(t *testing.T) {
	t.Parallel()
	_, err := SetupExchangeAssetPositionTracker("", "", false, nil)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	_, err = SetupExchangeAssetPositionTracker("test", "", false, nil)
	if !errors.Is(err, errNotFutureAsset) {
		t.Error(err)
	}

	_, err = SetupExchangeAssetPositionTracker("test", asset.Futures, false, nil)
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}

	resp, err := SetupExchangeAssetPositionTracker("test", asset.Futures, false, &FakePNL{})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if resp.Exchange != "test" {
		t.Errorf("expected 'test' received %v", resp.Exchange)
	}
}

func TestExchangeTrackNewOrder(t *testing.T) {
	t.Parallel()
	resp, err := SetupExchangeAssetPositionTracker("test", asset.Futures, false, &FakePNL{})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	err = resp.TrackNewOrder(&Detail{
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      Short,
		ID:        "1",
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.Positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.Positions))
	}

	err = resp.TrackNewOrder(&Detail{
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      Short,
		ID:        "1",
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.Positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.Positions))
	}

	err = resp.TrackNewOrder(&Detail{
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.Positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.Positions))
	}
	if resp.Positions[0].Status != Closed {
		t.Errorf("expected 'closed' received %v", resp.Positions[0].Status)
	}
	resp.Positions[0].Status = Open
	resp.Positions = append(resp.Positions, resp.Positions...)
	err = resp.TrackNewOrder(&Detail{
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, errPositionDiscrepancy) {
		t.Error(err)
	}
	if len(resp.Positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.Positions))
	}

	resp.Positions[0].Status = Closed
	err = resp.TrackNewOrder(&Detail{
		Exchange:  "test",
		AssetType: asset.USDTMarginedFutures,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, errAssetMismatch) {
		t.Error(err)
	}

}

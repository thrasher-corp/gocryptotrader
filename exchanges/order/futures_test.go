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
	e := PositionController{
		exchange:       exch,
		pnlCalculation: fPNL,
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
	if len(f.pnlHistory) != 1 {
		t.Errorf("expected 1 received %v", len(f.pnlHistory))
	}

	err = f.UpsertPNLEntry(PNLHistory{Time: tt})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.pnlHistory) != 1 {
		t.Errorf("expected 1 received %v", len(f.pnlHistory))
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
	e := PositionController{
		exchange:       "test",
		pnlCalculation: &FakePNL{},
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
	if !f.entryPrice.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected 1337, received %v", f.entryPrice)
	}
	if len(f.longPositions) != 1 {
		t.Error("expected a long")
	}
	if f.currentDirection != Long {
		t.Error("expected recognition that its long")
	}
	if f.exposure.InexactFloat64() != od.Amount {
		t.Error("expected 1")
	}

	od.Amount = 0.4
	od.Side = Short
	od.ID = "3"
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.shortPositions) != 1 {
		t.Error("expected a short")
	}
	if f.currentDirection != Long {
		t.Error("expected recognition that its long")
	}
	if f.exposure.InexactFloat64() != 0.6 {
		t.Error("expected 0.6")
	}
	od.Amount = 0.8
	od.Side = Short
	od.ID = "4"
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if f.currentDirection != Short {
		t.Error("expected recognition that its short")
	}
	if !f.exposure.Equal(decimal.NewFromFloat(0.2)) {
		t.Errorf("expected %v received %v", 0.2, f.exposure)
	}

	od.ID = "5"
	od.Side = Long
	od.Amount = 0.2
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if f.currentDirection != UnknownSide {
		t.Error("expected recognition that its unknown")
	}
	if f.status != Closed {
		t.Error("expected closed position")
	}

	err = f.TrackNewOrder(od)
	if !errors.Is(err, errPositionClosed) {
		t.Error(err)
	}
	if f.currentDirection != UnknownSide {
		t.Error("expected recognition that its unknown")
	}
	if f.status != Closed {
		t.Error("expected closed position")
	}
}

func TestSetupFuturesTracker(t *testing.T) {
	t.Parallel()

	_, err := SetupPositionController(nil)
	if !errors.Is(err, errNilSetup) {
		t.Error(err)
	}

	setup := &PositionControllerSetup{}
	_, err = SetupPositionController(setup)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}
	setup.Exchange = "test"
	_, err = SetupPositionController(setup)
	if !errors.Is(err, errNotFutureAsset) {
		t.Error(err)
	}
	setup.Asset = asset.Futures
	_, err = SetupPositionController(setup)
	if !errors.Is(err, ErrPairIsEmpty) {
		t.Error(err)
	}

	setup.Pair = currency.NewPair(currency.BTC, currency.USDT)
	_, err = SetupPositionController(setup)
	if !errors.Is(err, errEmptyUnderlying) {
		t.Error(err)
	}

	setup.Underlying = currency.BTC
	_, err = SetupPositionController(setup)
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}

	setup.PNLCalculator = &FakePNL{}
	resp, err := SetupPositionController(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if resp.exchange != "test" {
		t.Errorf("expected 'test' received %v", resp.exchange)
	}
}

func TestExchangeTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := "test"
	item := asset.Futures
	pair := currency.NewPair(currency.BTC, currency.USDT)
	setup := &PositionControllerSetup{
		Exchange:      exch,
		Asset:         item,
		Pair:          pair,
		Underlying:    pair.Base,
		PNLCalculator: &FakePNL{},
	}
	resp, err := SetupPositionController(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = resp.TrackNewOrder(&Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Short,
		ID:        "1",
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Short,
		ID:        "1",
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}
	if resp.positions[0].status != Closed {
		t.Errorf("expected 'closed' received %v", resp.positions[0].status)
	}
	resp.positions[0].status = Open
	resp.positions = append(resp.positions, resp.positions...)
	err = resp.TrackNewOrder(&Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, errPositionDiscrepancy) {
		t.Error(err)
	}
	if len(resp.positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.positions))
	}

	resp.positions[0].status = Closed
	err = resp.TrackNewOrder(&Detail{
		Exchange:  exch,
		Pair:      pair,
		AssetType: asset.USDTMarginedFutures,
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, errAssetMismatch) {
		t.Error(err)
	}

}

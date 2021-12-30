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

func (f *FakePNL) CalculatePNL(*PNLCalculatorRequest) (*PNLResult, error) {
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
		result: &PNLResult{
			Time: time.Now(),
		},
	}
	e := MultiPositionTracker{
		exchange:                   exch,
		useExchangePNLCalculations: true,
		exchangePNLCalculation:     fPNL,
	}
	setup := &PositionTrackerSetup{
		Pair:  pair,
		Asset: item,
	}

	f, err := e.SetupPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	err = f.TrackPNLByTime(time.Now(), 0)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	fPNL.err = errMissingPNLCalculationFunctions
	err = f.TrackPNLByTime(time.Now(), 0)
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}
}

func TestUpsertPNLEntry(t *testing.T) {
	t.Parallel()
	f := &PositionTracker{}
	err := f.upsertPNLEntry(PNLResult{})
	if !errors.Is(err, errTimeUnset) {
		t.Error(err)
	}
	tt := time.Now()
	err = f.upsertPNLEntry(PNLResult{Time: tt})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(f.pnlHistory) != 1 {
		t.Errorf("expected 1 received %v", len(f.pnlHistory))
	}

	err = f.upsertPNLEntry(PNLResult{Time: tt})
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
	e := MultiPositionTracker{
		exchange:               "test",
		exchangePNLCalculation: &FakePNL{},
	}
	setup := &PositionTrackerSetup{
		Pair:  pair,
		Asset: item,
	}
	f, err := e.SetupPositionTracker(setup)
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
	if !errors.Is(err, errTimeUnset) {
		t.Error(err)
	}

	od.Date = time.Now()
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
		t.Errorf("expected recognition that its unknown, received '%v'", f.currentDirection)
	}
	if f.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", f.status)
	}

	err = f.TrackNewOrder(od)
	if !errors.Is(err, ErrPositionClosed) {
		t.Error(err)
	}
	if f.currentDirection != UnknownSide {
		t.Errorf("expected recognition that its unknown, received '%v'", f.currentDirection)
	}
	if f.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", f.status)
	}
}

func TestSetupFuturesTracker(t *testing.T) {
	t.Parallel()

	_, err := SetupMultiPositionTracker(nil)
	if !errors.Is(err, errNilSetup) {
		t.Error(err)
	}

	setup := &PositionControllerSetup{}
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}
	setup.Exchange = "test"
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, errNotFutureAsset) {
		t.Error(err)
	}
	setup.Asset = asset.Futures
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, ErrPairIsEmpty) {
		t.Error(err)
	}

	setup.Pair = currency.NewPair(currency.BTC, currency.USDT)
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, errEmptyUnderlying) {
		t.Error(err)
	}

	setup.Underlying = currency.BTC
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	setup.UseExchangePNLCalculation = true
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}

	setup.ExchangePNLCalculation = &FakePNL{}
	resp, err := SetupMultiPositionTracker(setup)
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
		Exchange:               exch,
		Asset:                  item,
		Pair:                   pair,
		Underlying:             pair.Base,
		ExchangePNLCalculation: &FakePNL{},
	}
	resp, err := SetupMultiPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	tt := time.Now()

	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
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
		Date:      tt,
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
		Date:      tt,
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
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Long,
		ID:        "2",
		Amount:    2,
	})
	if !errors.Is(err, errCannotCalculateUnrealisedPNL) {
		t.Error(err)
	}
	if len(resp.positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.positions))
	}

	resp.positions[0].status = Closed
	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
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

func TestSetupPositionControllerReal(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	if pc.positionTrackerControllers == nil {
		t.Error("unexpected nil")
	}
}

func TestPositionControllerTestTrackNewOrder(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	err := pc.TrackNewOrder(nil)
	if !errors.Is(err, errNilOrder) {
		t.Error(err)
	}

	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		AssetType: asset.Spot,
		Side:      Long,
		ID:        "lol",
	})
	if !errors.Is(err, errNotFutureAsset) {
		t.Error(err)
	}

	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		AssetType: asset.Futures,
		Side:      Long,
		ID:        "lol",
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
}

func TestGetRealisedPNL(t *testing.T) {
	t.Parallel()
	pt := PositionTracker{}
	_, err := pt.GetRealisedPNL()
	if !errors.Is(err, errPositionNotClosed) {
		t.Error(err)
	}
	pt.status = Closed
	pt.realisedPNL = decimal.NewFromInt(1337)
	result, err := pt.GetRealisedPNL()
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if !pt.realisedPNL.Equal(result) {
		t.Errorf("expected 1337, received '%v'", result)
	}
}

func TestGetLatestPNLSnapshot(t *testing.T) {
	t.Parallel()
	pt := PositionTracker{}
	_, err := pt.GetLatestPNLSnapshot()
	if !errors.Is(err, errNoPNLHistory) {
		t.Error(err)
	}

	pnl := PNLResult{
		Time:                  time.Now(),
		UnrealisedPNL:         decimal.NewFromInt(1337),
		RealisedPNLBeforeFees: decimal.NewFromInt(1337),
	}
	pt.pnlHistory = append(pt.pnlHistory, pnl)

	result, err := pt.GetLatestPNLSnapshot()
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if result != pt.pnlHistory[0] {
		t.Error("unexpected result")
	}
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	pt := PositionTracker{}
	_, err := pt.CalculatePNL(nil)
	if !errors.Is(err, ErrNilPNLCalculator) {
		t.Error(err)
	}

	_, err = pt.CalculatePNL(&PNLCalculatorRequest{})
	if !errors.Is(err, errMissingPNLCalculationFunctions) {
		t.Error(err)
	}
	tt := time.Now()
	result, err := pt.CalculatePNL(&PNLCalculatorRequest{
		TimeBasedCalculation: &TimeBasedCalculation{
			Time:         tt,
			CurrentPrice: 1337,
		},
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if !result.Time.Equal(tt) {
		t.Error("unexpected result")
	}

	pt.status = Open
	pt.currentDirection = Long
	result, err = pt.CalculatePNL(&PNLCalculatorRequest{
		OrderBasedCalculation: &Detail{
			Date:      tt,
			Price:     1337,
			Exchange:  "test",
			AssetType: asset.Spot,
			Side:      Long,
			Status:    Active,
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			Amount:    5,
		},
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	pt.exposure = decimal.NewFromInt(5)
	result, err = pt.CalculatePNL(&PNLCalculatorRequest{
		OrderBasedCalculation: &Detail{
			Date:      tt,
			Price:     1337,
			Exchange:  "test",
			AssetType: asset.Spot,
			Side:      Short,
			Status:    Active,
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			Amount:    10,
		},
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	// todo do a proper test of values to ensure correct split calculation!
}

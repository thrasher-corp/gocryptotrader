package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const testExchange = "test"

// FakePNL implements PNL interface
type FakePNL struct {
	err    error
	result *PNLResult
}

// CalculatePNL overrides default pnl calculations
func (f *FakePNL) CalculatePNL(context.Context, *PNLCalculatorRequest) (*PNLResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// GetCurrencyForRealisedPNL  overrides default pnl calculations
func (f *FakePNL) GetCurrencyForRealisedPNL(realisedAsset asset.Item, realisedPair currency.Pair) (currency.Code, asset.Item, error) {
	if f.err != nil {
		return realisedPair.Base, "", f.err
	}
	return realisedPair.Base, realisedAsset, nil
}

func TestUpsertPNLEntry(t *testing.T) {
	t.Parallel()
	var results []PNLResult
	result := &PNLResult{
		IsOrder: true,
	}
	_, err := upsertPNLEntry(results, result)
	if !errors.Is(err, errTimeUnset) {
		t.Error(err)
	}
	tt := time.Now()
	result.Time = tt
	results, err = upsertPNLEntry(results, result)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 received %v", len(results))
	}
	result.Fee = decimal.NewFromInt(1337)
	results, err = upsertPNLEntry(results, result)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 received %v", len(results))
	}
	if !results[0].Fee.Equal(result.Fee) {
		t.Errorf("expected %v received %v", result.Fee, results[0].Fee)
	}
}

func TestTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := testExchange
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	e := MultiPositionTracker{
		exchange:               testExchange,
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
	f.openingDirection = Long
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

	od.Date = od.Date.Add(1)
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

	od.Date = od.Date.Add(1)
	od.Amount = 0.8
	od.Side = Short
	od.ID = "4"
	od.Fee = 0.1
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

	od.Date = od.Date.Add(1)
	od.ID = "5"
	od.Side = Long
	od.Amount = 0.2
	err = f.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if f.currentDirection != SideNA {
		t.Errorf("expected recognition that its unknown, received '%v'", f.currentDirection)
	}
	if f.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", f.status)
	}

	err = f.TrackNewOrder(od)
	if !errors.Is(err, ErrPositionClosed) {
		t.Error(err)
	}
	if f.currentDirection != SideNA {
		t.Errorf("expected recognition that its unknown, received '%v'", f.currentDirection)
	}
	if f.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", f.status)
	}
}

func TestSetupMultiPositionTracker(t *testing.T) {
	t.Parallel()

	_, err := SetupMultiPositionTracker(nil)
	if !errors.Is(err, errNilSetup) {
		t.Error(err)
	}

	setup := &MultiPositionTrackerSetup{}
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}
	setup.Exchange = testExchange
	_, err = SetupMultiPositionTracker(setup)
	if !errors.Is(err, ErrNotFuturesAsset) {
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
	if resp.exchange != testExchange {
		t.Errorf("expected 'test' received %v", resp.exchange)
	}
}

func TestExchangeTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := testExchange
	item := asset.Futures
	pair := currency.NewPair(currency.BTC, currency.USDT)
	setup := &MultiPositionTrackerSetup{
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
		ID:        "2",
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
		ID:        "3",
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
		ID:        "4",
		Amount:    2,
	})
	if !errors.Is(err, errPositionDiscrepancy) {
		t.Errorf("received '%v' expected '%v", err, errPositionDiscrepancy)
	}

	resp.positions = []*PositionTracker{resp.positions[0]}
	resp.positions[0].status = Closed
	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Long,
		ID:        "4",
		Amount:    2,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
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
		ID:        "5",
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
	if !errors.Is(err, ErrNotFuturesAsset) {
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

func TestGetRealisedPNL(t *testing.T) {
	t.Parallel()
	p := PositionTracker{}
	result := p.GetRealisedPNL()
	if !result.IsZero() {
		t.Error("expected zero")
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()

	p := &PositionTracker{}
	stats := p.GetStats()
	if len(stats.Orders) != 0 {
		t.Error("expected 0")
	}

	p.exchange = testExchange
	stats = p.GetStats()
	if stats.Exchange != p.exchange {
		t.Errorf("expected '%v' received '%v'", p.exchange, stats.Exchange)
	}

	p = nil
	stats = p.GetStats()
	if len(stats.Orders) != 0 {
		t.Error("expected 0")
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	p := &MultiPositionTracker{}
	positions := p.GetPositions()
	if len(positions) > 0 {
		t.Error("expected 0")
	}

	p.positions = append(p.positions, &PositionTracker{
		exchange: testExchange,
	})
	positions = p.GetPositions()
	if len(positions) != 1 {
		t.Fatal("expected 1")
	}
	if positions[0].Exchange != testExchange {
		t.Error("expected 'test'")
	}

	p = nil
	positions = p.GetPositions()
	if len(positions) > 0 {
		t.Error("expected 0")
	}
}

func TestGetPositionsForExchange(t *testing.T) {
	t.Parallel()
	c := &PositionController{}
	p := currency.NewPair(currency.BTC, currency.USDT)
	pos, err := c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}
	if len(pos) != 0 {
		t.Error("expected zero")
	}
	c.positionTrackerControllers = make(map[string]map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}
	c.positionTrackerControllers[testExchange] = make(map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange][asset.Futures] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForPair) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForPair)
	}
	_, err = c.GetPositionsForExchange(testExchange, asset.Spot, p)
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}

	c.positionTrackerControllers[testExchange][asset.Futures] = make(map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange][asset.Futures][p] = &MultiPositionTracker{
		exchange: testExchange,
	}

	pos, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if len(pos) != 0 {
		t.Fatal("expected zero")
	}
	c.positionTrackerControllers[testExchange][asset.Futures][p] = &MultiPositionTracker{
		exchange: testExchange,
		positions: []*PositionTracker{
			{
				exchange: testExchange,
			},
		},
	}
	pos, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if len(pos) != 1 {
		t.Fatal("expected 1")
	}
	if pos[0].Exchange != testExchange {
		t.Error("expected test")
	}
	c = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestClearPositionsForExchange(t *testing.T) {
	t.Parallel()
	c := &PositionController{}
	p := currency.NewPair(currency.BTC, currency.USDT)
	err := c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}
	c.positionTrackerControllers = make(map[string]map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange] = nil
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}
	c.positionTrackerControllers[testExchange] = make(map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange][asset.Futures] = nil
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionsNotLoadedForPair) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForPair)
	}
	err = c.ClearPositionsForExchange(testExchange, asset.Spot, p)
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}

	c.positionTrackerControllers[testExchange][asset.Futures] = make(map[currency.Pair]*MultiPositionTracker)
	c.positionTrackerControllers[testExchange][asset.Futures][p] = &MultiPositionTracker{
		exchange: testExchange,
	}
	c.positionTrackerControllers[testExchange][asset.Futures][p] = &MultiPositionTracker{
		exchange:   testExchange,
		underlying: currency.DOGE,
		positions: []*PositionTracker{
			{
				exchange: testExchange,
			},
		},
	}
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if len(c.positionTrackerControllers[testExchange][asset.Futures][p].positions) != 0 {
		t.Fatal("expected 0")
	}
	c = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestCalculateRealisedPNL(t *testing.T) {
	t.Parallel()
	result := calculateRealisedPNL(nil)
	if !result.IsZero() {
		t.Errorf("received '%v' expected '0'", result)
	}
	result = calculateRealisedPNL([]PNLResult{
		{
			IsOrder:               true,
			RealisedPNLBeforeFees: decimal.NewFromInt(1337),
		},
	})
	if !result.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '1337'", result)
	}

	result = calculateRealisedPNL([]PNLResult{
		{
			IsOrder:               true,
			RealisedPNLBeforeFees: decimal.NewFromInt(1339),
			Fee:                   decimal.NewFromInt(2),
		},
		{
			IsOrder:               true,
			RealisedPNLBeforeFees: decimal.NewFromInt(2),
			Fee:                   decimal.NewFromInt(2),
		},
	})
	if !result.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '1337'", result)
	}
}

func TestSetupPositionTracker(t *testing.T) {
	t.Parallel()
	m := &MultiPositionTracker{}
	p, err := m.SetupPositionTracker(nil)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}
	if p != nil {
		t.Error("expected nil")
	}
	m.exchange = testExchange
	p, err = m.SetupPositionTracker(nil)
	if !errors.Is(err, errNilSetup) {
		t.Errorf("received '%v' expected '%v", err, errNilSetup)
	}
	if p != nil {
		t.Error("expected nil")
	}

	p, err = m.SetupPositionTracker(&PositionTrackerSetup{
		Asset: asset.Spot,
	})
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}
	if p != nil {
		t.Error("expected nil")
	}

	p, err = m.SetupPositionTracker(&PositionTrackerSetup{
		Asset: asset.Futures,
	})
	if !errors.Is(err, ErrPairIsEmpty) {
		t.Errorf("received '%v' expected '%v", err, ErrPairIsEmpty)
	}
	if p != nil {
		t.Error("expected nil")
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	p, err = m.SetupPositionTracker(&PositionTrackerSetup{
		Asset: asset.Futures,
		Pair:  cp,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v", err, nil)
	}
	if p == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected not nil")
	}
	if p.exchange != testExchange { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Error("expected test")
	}

	_, err = m.SetupPositionTracker(&PositionTrackerSetup{
		Asset:                     asset.Futures,
		Pair:                      cp,
		UseExchangePNLCalculation: true,
	})
	if !errors.Is(err, ErrNilPNLCalculator) {
		t.Errorf("received '%v' expected '%v", err, ErrNilPNLCalculator)
	}
	m.exchangePNLCalculation = &PNLCalculator{}
	p, err = m.SetupPositionTracker(&PositionTrackerSetup{
		Asset:                     asset.Futures,
		Pair:                      cp,
		UseExchangePNLCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !p.useExchangePNLCalculation {
		t.Error("expected true")
	}
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	p := &PNLCalculator{}
	_, err := p.CalculatePNL(context.Background(), nil)
	if !errors.Is(err, ErrNilPNLCalculator) {
		t.Errorf("received '%v' expected '%v", err, ErrNilPNLCalculator)
	}
	_, err = p.CalculatePNL(context.Background(), &PNLCalculatorRequest{})
	if !errors.Is(err, errCannotCalculateUnrealisedPNL) {
		t.Errorf("received '%v' expected '%v", err, errCannotCalculateUnrealisedPNL)
	}

	_, err = p.CalculatePNL(context.Background(),
		&PNLCalculatorRequest{
			OrderDirection:   Short,
			CurrentDirection: Long,
		})
	if !errors.Is(err, errCannotCalculateUnrealisedPNL) {
		t.Errorf("received '%v' expected '%v", err, errCannotCalculateUnrealisedPNL)
	}
}

func TestTrackPNLByTime(t *testing.T) {
	t.Parallel()
	p := &PositionTracker{}
	err := p.TrackPNLByTime(time.Now(), 1)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = p.TrackPNLByTime(time.Now(), 2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !p.latestPrice.Equal(decimal.NewFromInt(2)) {
		t.Error("expected 2")
	}
	p = nil
	err = p.TrackPNLByTime(time.Now(), 2)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()

	_, err := pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}

	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		AssetType: asset.Futures,
		Side:      Long,
		ID:        "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi2", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForExchange)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionsNotLoadedForAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForAsset)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.DOGE), 2, time.Now())
	if !errors.Is(err, ErrPositionsNotLoadedForPair) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionsNotLoadedForPair)
	}

	pnl, err := pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !pnl.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v' expected '%v", pnl, 1)
	}

	pc = nil
	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestSetCollateralCurrency(t *testing.T) {
	t.Parallel()
	var expectedError = ErrNotFuturesAsset
	pc := SetupPositionController()
	err := pc.SetCollateralCurrency("hi", asset.Spot, currency.Pair{}, currency.Code{})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	pc.positionTrackerControllers = make(map[string]map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, cp, currency.DOGE)
	expectedError = ErrPositionsNotLoadedForExchange
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
	pc.positionTrackerControllers["hi"] = make(map[asset.Item]map[currency.Pair]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, cp, currency.DOGE)
	expectedError = ErrPositionsNotLoadedForAsset
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	pc.positionTrackerControllers["hi"][asset.Futures] = make(map[currency.Pair]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, cp, currency.DOGE)
	expectedError = ErrPositionsNotLoadedForPair
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	pc.positionTrackerControllers["hi"][asset.Futures][cp] = nil
	err = pc.SetCollateralCurrency("hi", asset.Futures, cp, currency.DOGE)
	expectedError = common.ErrNilPointer
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	pc.positionTrackerControllers["hi"][asset.Futures][cp] = &MultiPositionTracker{
		exchange:       "hi",
		asset:          asset.Futures,
		pair:           cp,
		orderPositions: make(map[string]*PositionTracker),
	}
	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      Long,
		ID:        "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v", err, nil)
	}

	err = pc.SetCollateralCurrency("hi", asset.Futures, cp, currency.DOGE)
	expectedError = nil
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	if !pc.positionTrackerControllers["hi"][asset.Futures][cp].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.positionTrackerControllers["hi"][asset.Futures][cp].collateralCurrency, currency.DOGE)
	}

	if !pc.positionTrackerControllers["hi"][asset.Futures][cp].positions[0].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.positionTrackerControllers["hi"][asset.Futures][cp].positions[0].collateralCurrency, currency.DOGE)
	}

	pc = nil
	err = pc.SetCollateralCurrency("hi", asset.Spot, currency.Pair{}, currency.Code{})
	expectedError = common.ErrNilPointer
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
}

func TestMPTUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	var err, expectedError error
	expectedError = nil
	cp := currency.NewPair(currency.BTC, currency.USDT)
	pc := SetupPositionController()
	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      Long,
		ID:        "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
	result, err := pc.positionTrackerControllers["hi"][asset.Futures][cp].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
	if result.Equal(decimal.NewFromInt(1337)) {
		t.Error("")
	}

	expectedError = ErrPositionClosed
	pc.positionTrackerControllers["hi"][asset.Futures][cp].positions[0].status = Closed
	_, err = pc.positionTrackerControllers["hi"][asset.Futures][cp].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = ErrPositionsNotLoadedForPair
	pc.positionTrackerControllers["hi"][asset.Futures][cp].positions = nil
	_, err = pc.positionTrackerControllers["hi"][asset.Futures][cp].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
}

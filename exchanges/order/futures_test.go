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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
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
		return realisedPair.Base, asset.Empty, f.err
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
	setup := &PositionTrackerSetup{
		Exchange: exch,
		Asset:    item,
		Pair:     pair,
	}
	c, err := SetupPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = c.TrackNewOrder(nil, false)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}
	err = c.TrackNewOrder(&Detail{}, false)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	od := &Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		OrderID:   "1",
		Price:     1337,
	}
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, ErrSideIsInvalid) {
		t.Error(err)
	}

	od.Side = Long
	od.Amount = 1
	od.OrderID = "2"
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, errTimeUnset) {
		t.Error(err)
	}
	c.openingDirection = Long
	od.Date = time.Now()
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if !c.openingPrice.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected 1337, received %v", c.openingPrice)
	}
	if len(c.longPositions) != 1 {
		t.Error("expected a long")
	}
	if c.latestDirection != Long {
		t.Error("expected recognition that its long")
	}
	if c.exposure.InexactFloat64() != od.Amount {
		t.Error("expected 1")
	}

	od.Date = od.Date.Add(1)
	od.Amount = 0.4
	od.Side = Short
	od.OrderID = "3"
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(c.shortPositions) != 1 {
		t.Error("expected a short")
	}
	if c.latestDirection != Long {
		t.Error("expected recognition that its long")
	}
	if c.exposure.InexactFloat64() != 0.6 {
		t.Error("expected 0.6")
	}

	od.Date = od.Date.Add(1)
	od.Amount = 0.8
	od.Side = Short
	od.OrderID = "4"
	od.Fee = 0.1
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if c.latestDirection != Short {
		t.Error("expected recognition that its short")
	}
	if !c.exposure.Equal(decimal.NewFromFloat(0.2)) {
		t.Errorf("expected %v received %v", 0.2, c.exposure)
	}

	od.Date = od.Date.Add(1)
	od.OrderID = "5"
	od.Side = Long
	od.Amount = 0.2
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if c.latestDirection != ClosePosition {
		t.Errorf("expected recognition that its closed, received '%v'", c.latestDirection)
	}
	if c.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", c.status)
	}

	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	od.OrderID = "hellomoto"
	err = c.TrackNewOrder(od, false)
	if !errors.Is(err, ErrPositionClosed) {
		t.Errorf("received %v expected %v", err, ErrPositionClosed)
	}
	if c.latestDirection != ClosePosition {
		t.Errorf("expected recognition that its closed, received '%v'", c.latestDirection)
	}
	if c.status != Closed {
		t.Errorf("expected recognition that its closed, received '%v'", c.status)
	}

	err = c.TrackNewOrder(od, true)
	if !errors.Is(err, errCannotTrackInvalidParams) {
		t.Error(err)
	}

	c, err = SetupPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	err = c.TrackNewOrder(od, true)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	var ptp *PositionTracker
	err = ptp.TrackNewOrder(nil, false)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
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

func TestMultiPositionTrackerTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := testExchange
	item := asset.Futures
	pair := currency.NewPair(currency.BTC, currency.USDT)
	setup := &MultiPositionTrackerSetup{
		Asset:                  item,
		Pair:                   pair,
		Underlying:             pair.Base,
		ExchangePNLCalculation: &FakePNL{},
	}
	_, err := SetupMultiPositionTracker(setup)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	setup.Exchange = testExchange
	resp, err := SetupMultiPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	tt := time.Now()
	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
		AssetType: item,
		Pair:      pair,
		Side:      Short,
		OrderID:   "1",
		Amount:    1,
	})
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Short,
		OrderID:   "1",
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
		OrderID:   "2",
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
		OrderID:   "3",
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
		OrderID:   "4",
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
		OrderID:   "4",
		Amount:    2,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if len(resp.positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      Long,
		OrderID:   "4",
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
		OrderID:   "5",
		Amount:    2,
	})
	if !errors.Is(err, errAssetMismatch) {
		t.Error(err)
	}

	err = resp.TrackNewOrder(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}

	resp = nil
	err = resp.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  exch,
		Pair:      pair,
		AssetType: asset.USDTMarginedFutures,
		Side:      Long,
		OrderID:   "5",
		Amount:    2,
	})
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}
}

func TestSetupPositionControllerReal(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	if pc.multiPositionTrackers == nil {
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
		OrderID:   "lol",
	})
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Error(err)
	}

	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
	})
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	err = pc.TrackNewOrder(&Detail{
		Exchange:  testExchange,
		Date:      time.Now(),
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	var pcp *PositionController
	err = pcp.TrackNewOrder(nil)
	if !errors.Is(err, common.ErrNilPointer) {
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
	p.fundingRateDetails = &fundingrate.Rates{
		FundingRates: []fundingrate.Rate{
			{},
		},
	}

	stats = p.GetStats()
	if stats.Exchange != p.exchange {
		t.Errorf("expected '%v' received '%v'", p.exchange, stats.Exchange)
	}

	p = nil
	stats = p.GetStats()
	if stats != nil {
		t.Errorf("expected '%v' received '%v'", nil, stats)
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

	_, err := c.GetPositionsForExchange("", asset.Futures, p)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	pos, err := c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	if len(pos) != 0 {
		t.Error("expected zero")
	}
	c.multiPositionTrackers = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	c.multiPositionTrackers[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	_, err = c.GetPositionsForExchange(testExchange, asset.Spot, p)
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}

	c.multiPositionTrackers[testExchange][asset.Futures] = make(map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item] = make(map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item] = &MultiPositionTracker{
		exchange: testExchange,
	}

	pos, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if len(pos) != 0 {
		t.Fatal("expected zero")
	}
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item] = &MultiPositionTracker{
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
	err := c.ClearPositionsForExchange("", asset.Futures, p)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	c.multiPositionTrackers = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange] = nil
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	c.multiPositionTrackers[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures] = nil
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}
	err = c.ClearPositionsForExchange(testExchange, asset.Spot, p)
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}

	c.multiPositionTrackers[testExchange][asset.Futures] = make(map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item] = make(map[*currency.Item]*MultiPositionTracker)
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item] = &MultiPositionTracker{
		exchange: testExchange,
	}
	c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item] = &MultiPositionTracker{
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
	if len(c.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item].positions) != 0 {
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
	p, err := SetupPositionTracker(nil)
	if !errors.Is(err, errNilSetup) {
		t.Errorf("received '%v' expected '%v", err, errNilSetup)
	}
	if p != nil {
		t.Error("expected nil")
	}
	m.exchange = testExchange
	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Asset: asset.Spot,
	})
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}
	if p != nil {
		t.Error("expected nil")
	}

	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Spot,
	})
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}
	if p != nil {
		t.Error("expected nil")
	}

	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Futures,
	})
	if !errors.Is(err, ErrPairIsEmpty) {
		t.Errorf("received '%v' expected '%v", err, ErrPairIsEmpty)
	}
	if p != nil {
		t.Error("expected nil")
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
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

	_, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange:                  testExchange,
		Asset:                     asset.Futures,
		Pair:                      cp,
		UseExchangePNLCalculation: true,
	})
	if !errors.Is(err, ErrNilPNLCalculator) {
		t.Errorf("received '%v' expected '%v", err, ErrNilPNLCalculator)
	}
	m.exchangePNLCalculation = &PNLCalculator{}
	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange:                  testExchange,
		Asset:                     asset.Futures,
		Pair:                      cp,
		UseExchangePNLCalculation: true,
		PNLCalculator:             &PNLCalculator{},
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

	_, err := pc.UpdateOpenPositionUnrealisedPNL("", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
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
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi2", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.DOGE), 2, time.Now())
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	pnl, err := pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !pnl.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v' expected '%v", pnl, 1)
	}

	var nilPC *PositionController
	_, err = nilPC.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.USDT), 2, time.Now())
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestSetCollateralCurrency(t *testing.T) {
	t.Parallel()
	var expectedError = errExchangeNameEmpty
	pc := SetupPositionController()
	err := pc.SetCollateralCurrency("", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = ErrNotFuturesAsset
	err = pc.SetCollateralCurrency("hi", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	p := currency.NewPair(currency.BTC, currency.USDT)
	pc.multiPositionTrackers = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	expectedError = ErrPositionNotFound
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
	pc.multiPositionTrackers["hi"] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	pc.multiPositionTrackers["hi"][asset.Futures] = make(map[*currency.Item]map[*currency.Item]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = ErrPositionNotFound
	pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item] = make(map[*currency.Item]*MultiPositionTracker)
	pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item] = nil
	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item] = &MultiPositionTracker{
		exchange:       "hi",
		asset:          asset.Futures,
		pair:           p,
		orderPositions: make(map[string]*PositionTracker),
	}
	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      p,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v", err, nil)
	}

	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	expectedError = nil
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	if !pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].collateralCurrency, currency.DOGE)
	}

	if !pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].positions[0].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].positions[0].collateralCurrency, currency.DOGE)
	}

	var nilPC *PositionController
	err = nilPC.SetCollateralCurrency("hi", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	expectedError = common.ErrNilPointer
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
}

func TestMPTUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	var err, expectedError error
	expectedError = nil
	p := currency.NewPair(currency.BTC, currency.USDT)
	pc := SetupPositionController()
	err = pc.TrackNewOrder(&Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      p,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	result, err := pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
	if result.Equal(decimal.NewFromInt(1337)) {
		t.Error("")
	}

	expectedError = ErrPositionClosed
	pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].positions[0].status = Closed
	_, err = pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = ErrPositionNotFound
	pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].positions = nil
	_, err = pc.multiPositionTrackers["hi"][asset.Futures][p.Base.Item][p.Quote.Item].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v", err, expectedError)
	}
}

func TestMPTLiquidate(t *testing.T) {
	t.Parallel()
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	e := &MultiPositionTracker{
		exchange:               testExchange,
		exchangePNLCalculation: &FakePNL{},
		asset:                  item,
		orderPositions:         make(map[string]*PositionTracker),
	}

	err = e.Liquidate(decimal.Zero, time.Time{})
	if !errors.Is(err, ErrPositionNotFound) {
		t.Error(err)
	}

	setup := &PositionTrackerSetup{
		Pair:  pair,
		Asset: item,
	}
	_, err = SetupPositionTracker(setup)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Error(err)
	}

	setup.Exchange = "exch"
	_, err = SetupPositionTracker(setup)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	tt := time.Now()
	err = e.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  testExchange,
		Pair:      pair,
		AssetType: item,
		Side:      Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = e.Liquidate(decimal.Zero, time.Time{})
	if !errors.Is(err, errCannotLiquidate) {
		t.Error(err)
	}

	err = e.Liquidate(decimal.Zero, tt)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	if e.positions[0].status != Liquidated {
		t.Errorf("received '%v' expected '%v'", e.positions[0].status, Liquidated)
	}
	if !e.positions[0].exposure.IsZero() {
		t.Errorf("received '%v' expected '%v'", e.positions[0].exposure, 0)
	}

	e = nil
	err = e.Liquidate(decimal.Zero, tt)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}
}

func TestPositionLiquidate(t *testing.T) {
	t.Parallel()
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	p := &PositionTracker{
		contractPair:     pair,
		asset:            item,
		exchange:         testExchange,
		PNLCalculation:   &PNLCalculator{},
		status:           Open,
		openingDirection: Long,
	}

	tt := time.Now()
	err = p.TrackNewOrder(&Detail{
		Date:      tt,
		Exchange:  testExchange,
		Pair:      pair,
		AssetType: item,
		Side:      Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	}, false)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	err = p.Liquidate(decimal.Zero, time.Time{})
	if !errors.Is(err, errCannotLiquidate) {
		t.Error(err)
	}

	err = p.Liquidate(decimal.Zero, tt)
	if !errors.Is(err, nil) {
		t.Error(err)
	}

	if p.status != Liquidated {
		t.Errorf("received '%v' expected '%v'", p.status, Liquidated)
	}
	if !p.exposure.IsZero() {
		t.Errorf("received '%v' expected '%v'", p.exposure, 0)
	}

	p = nil
	err = p.Liquidate(decimal.Zero, tt)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Error(err)
	}
}

func TestGetOpenPosition(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	cp := currency.NewPair(currency.BTC, currency.PERP)
	tn := time.Now()

	_, err := pc.GetOpenPosition("", asset.Futures, cp)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	_, err = pc.GetOpenPosition(testExchange, asset.Futures, cp)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	err = pc.TrackNewOrder(&Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	_, err = pc.GetOpenPosition(testExchange, asset.Futures, cp)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
}

func TestGetAllOpenPositions(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()

	_, err := pc.GetAllOpenPositions()
	if !errors.Is(err, ErrNoPositionsFound) {
		t.Errorf("received '%v' expected '%v", err, ErrNoPositionsFound)
	}

	cp := currency.NewPair(currency.BTC, currency.PERP)
	tn := time.Now()
	err = pc.TrackNewOrder(&Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	_, err = pc.GetAllOpenPositions()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
}

func TestPCTrackFundingDetails(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	err := pc.TrackFundingDetails(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}

	p := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.Rates{
		Asset: asset.Futures,
		Pair:  p,
	}
	err = pc.TrackFundingDetails(rates)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	rates.Exchange = testExchange
	err = pc.TrackFundingDetails(rates)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	tn := time.Now()
	err = pc.TrackNewOrder(&Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      p,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	rates.StartDate = tn.Add(-time.Hour)
	rates.EndDate = tn
	rates.FundingRates = []fundingrate.Rate{
		{
			Time:    tn,
			Rate:    decimal.NewFromInt(1337),
			Payment: decimal.NewFromInt(1337),
		},
	}
	pc.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item].orderPositions["lol"].openingDate = tn.Add(-time.Hour)
	pc.multiPositionTrackers[testExchange][asset.Futures][p.Base.Item][p.Quote.Item].orderPositions["lol"].lastUpdated = tn
	err = pc.TrackFundingDetails(rates)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
}

func TestMPTTrackFundingDetails(t *testing.T) {
	t.Parallel()
	mpt := &MultiPositionTracker{
		orderPositions: make(map[string]*PositionTracker),
	}

	err := mpt.TrackFundingDetails(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}

	cp := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.Rates{
		Asset: asset.Futures,
		Pair:  cp,
	}
	err = mpt.TrackFundingDetails(rates)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	mpt.exchange = testExchange
	rates = &fundingrate.Rates{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
	}
	err = mpt.TrackFundingDetails(rates)
	if !errors.Is(err, errAssetMismatch) {
		t.Errorf("received '%v' expected '%v", err, errAssetMismatch)
	}

	mpt.asset = rates.Asset
	mpt.pair = cp
	err = mpt.TrackFundingDetails(rates)
	if !errors.Is(err, ErrPositionNotFound) {
		t.Errorf("received '%v' expected '%v", err, ErrPositionNotFound)
	}

	tn := time.Now()
	err = mpt.TrackNewOrder(&Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	rates.StartDate = tn.Add(-time.Hour)
	rates.EndDate = tn
	rates.FundingRates = []fundingrate.Rate{
		{
			Time:    tn,
			Rate:    decimal.NewFromInt(1337),
			Payment: decimal.NewFromInt(1337),
		},
	}
	mpt.orderPositions["lol"].openingDate = tn.Add(-time.Hour)
	mpt.orderPositions["lol"].lastUpdated = tn
	rates.Exchange = "lol"
	err = mpt.TrackFundingDetails(rates)
	if !errors.Is(err, errExchangeNameMismatch) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameMismatch)
	}
}

func TestPTTrackFundingDetails(t *testing.T) {
	t.Parallel()
	p := &PositionTracker{}
	err := p.TrackFundingDetails(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}

	cp := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.Rates{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
	}
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, errDoesntMatch) {
		t.Errorf("received '%v' expected '%v", err, errDoesntMatch)
	}

	p.exchange = testExchange
	p.asset = asset.Futures
	p.contractPair = cp
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received '%v' expected '%v", err, common.ErrDateUnset)
	}

	rates.StartDate = time.Now().Add(-time.Hour)
	rates.EndDate = time.Now()
	p.openingDate = rates.StartDate
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, ErrNoPositionsFound) {
		t.Errorf("received '%v' expected '%v", err, ErrNoPositionsFound)
	}

	p.pnlHistory = append(p.pnlHistory, PNLResult{
		Time:                  rates.EndDate,
		UnrealisedPNL:         decimal.NewFromInt(1337),
		RealisedPNLBeforeFees: decimal.NewFromInt(1337),
		Price:                 decimal.NewFromInt(1337),
		Exposure:              decimal.NewFromInt(1337),
		Fee:                   decimal.NewFromInt(1337),
	})
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	rates.FundingRates = []fundingrate.Rate{
		{
			Time:    rates.StartDate,
			Rate:    decimal.NewFromInt(1337),
			Payment: decimal.NewFromInt(1337),
		},
	}
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	rates.StartDate = rates.StartDate.Add(-time.Hour)
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	rates.Exchange = ""
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}

	p = nil
	err = p.TrackFundingDetails(rates)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestAreFundingRatePrerequisitesMet(t *testing.T) {
	t.Parallel()
	err := CheckFundingRatePrerequisites(false, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = CheckFundingRatePrerequisites(true, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = CheckFundingRatePrerequisites(true, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = CheckFundingRatePrerequisites(true, true, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = CheckFundingRatePrerequisites(true, false, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}

	err = CheckFundingRatePrerequisites(false, false, true)
	if !errors.Is(err, ErrGetFundingDataRequired) {
		t.Errorf("received '%v' expected '%v", err, ErrGetFundingDataRequired)
	}

	err = CheckFundingRatePrerequisites(false, true, true)
	if !errors.Is(err, ErrGetFundingDataRequired) {
		t.Errorf("received '%v' expected '%v", err, ErrGetFundingDataRequired)
	}

	err = CheckFundingRatePrerequisites(false, true, false)
	if !errors.Is(err, ErrGetFundingDataRequired) {
		t.Errorf("received '%v' expected '%v", err, ErrGetFundingDataRequired)
	}
}

func TestLastUpdated(t *testing.T) {
	t.Parallel()
	p := &PositionController{}
	tm, err := p.LastUpdated()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !tm.IsZero() {
		t.Errorf("received '%v' expected '%v", tm, time.Time{})
	}
	p.updated = time.Now()
	tm, err = p.LastUpdated()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if !tm.Equal(p.updated) {
		t.Errorf("received '%v' expected '%v", tm, p.updated)
	}
	p = nil
	_, err = p.LastUpdated()
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilPointer)
	}
}

func TestGetCurrencyForRealisedPNL(t *testing.T) {
	p := PNLCalculator{}
	code, a, err := p.GetCurrencyForRealisedPNL(asset.Spot, currency.NewPair(currency.DOGE, currency.XRP))
	if err != nil {
		t.Error(err)
	}
	if !code.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v", code, currency.DOGE)
	}
	if a != asset.Spot {
		t.Errorf("received '%v' expected '%v", a, asset.Spot)
	}
}

func TestCheckTrackerPrerequisitesLowerExchange(t *testing.T) {
	t.Parallel()
	_, err := checkTrackerPrerequisitesLowerExchange("", asset.Spot, currency.EMPTYPAIR)
	if !errors.Is(err, errExchangeNameEmpty) {
		t.Errorf("received '%v' expected '%v", err, errExchangeNameEmpty)
	}
	upperExch := "IM UPPERCASE"
	_, err = checkTrackerPrerequisitesLowerExchange(upperExch, asset.Spot, currency.EMPTYPAIR)
	if !errors.Is(err, ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, ErrNotFuturesAsset)
	}
	_, err = checkTrackerPrerequisitesLowerExchange(upperExch, asset.Futures, currency.EMPTYPAIR)
	if !errors.Is(err, ErrPairIsEmpty) {
		t.Errorf("received '%v' expected '%v", err, ErrPairIsEmpty)
	}
	lowerExch, err := checkTrackerPrerequisitesLowerExchange(upperExch, asset.Futures, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if lowerExch != "im uppercase" {
		t.Error("expected lowercase")
	}
}

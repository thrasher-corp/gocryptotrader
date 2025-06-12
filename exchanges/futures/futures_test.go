package futures

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	assert.ErrorIs(t, err, errTimeUnset)

	tt := time.Now()
	result.Time = tt
	results, err = upsertPNLEntry(results, result)
	assert.NoError(t, err)

	if len(results) != 1 {
		t.Errorf("expected 1 received %v", len(results))
	}
	result.Fee = decimal.NewFromInt(1337)
	results, err = upsertPNLEntry(results, result)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	setup := &PositionTrackerSetup{
		Exchange: exch,
		Asset:    item,
		Pair:     pair,
	}
	c, err := SetupPositionTracker(setup)
	assert.NoError(t, err)

	err = c.TrackNewOrder(nil, false)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	err = c.TrackNewOrder(&order.Detail{}, false)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	od := &order.Detail{
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		OrderID:   "1",
		Price:     1337,
	}
	err = c.TrackNewOrder(od, false)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	od.Side = order.Long
	od.Amount = 1
	od.OrderID = "2"
	err = c.TrackNewOrder(od, false)
	assert.ErrorIs(t, err, errTimeUnset)

	c.openingDirection = order.Long
	od.Date = time.Now()
	err = c.TrackNewOrder(od, false)
	assert.NoError(t, err)

	if !c.openingPrice.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected 1337, received %v", c.openingPrice)
	}
	if len(c.longPositions) != 1 {
		t.Error("expected a long")
	}
	if c.latestDirection != order.Long {
		t.Error("expected recognition that its long")
	}
	if c.exposure.InexactFloat64() != od.Amount {
		t.Error("expected 1")
	}

	od.Date = od.Date.Add(1)
	od.Amount = 0.4
	od.Side = order.Short
	od.OrderID = "3"
	err = c.TrackNewOrder(od, false)
	assert.NoError(t, err)

	if len(c.shortPositions) != 1 {
		t.Error("expected a short")
	}
	if c.latestDirection != order.Long {
		t.Error("expected recognition that its long")
	}
	if c.exposure.InexactFloat64() != 0.6 {
		t.Error("expected 0.6")
	}

	od.Date = od.Date.Add(1)
	od.Amount = 0.8
	od.Side = order.Short
	od.OrderID = "4"
	od.Fee = 0.1
	err = c.TrackNewOrder(od, false)
	assert.NoError(t, err)

	if c.latestDirection != order.Short {
		t.Error("expected recognition that its short")
	}
	if !c.exposure.Equal(decimal.NewFromFloat(0.2)) {
		t.Errorf("expected %v received %v", 0.2, c.exposure)
	}

	od.Date = od.Date.Add(1)
	od.OrderID = "5"
	od.Side = order.Long
	od.Amount = 0.2
	err = c.TrackNewOrder(od, false)
	assert.NoError(t, err)

	if c.latestDirection != order.ClosePosition {
		t.Errorf("expected recognition that its closed, received '%v'", c.latestDirection)
	}
	if c.status != order.Closed {
		t.Errorf("expected recognition that its closed, received '%v'", c.status)
	}

	err = c.TrackNewOrder(od, false)
	assert.NoError(t, err)

	od.OrderID = "hellomoto"
	err = c.TrackNewOrder(od, false)
	assert.ErrorIs(t, err, ErrPositionClosed)

	if c.latestDirection != order.ClosePosition {
		t.Errorf("expected recognition that its closed, received '%v'", c.latestDirection)
	}
	if c.status != order.Closed {
		t.Errorf("expected recognition that its closed, received '%v'", c.status)
	}

	err = c.TrackNewOrder(od, true)
	assert.ErrorIs(t, err, errCannotTrackInvalidParams)

	c, err = SetupPositionTracker(setup)
	assert.NoError(t, err)

	err = c.TrackNewOrder(od, true)
	assert.NoError(t, err)

	var ptp *PositionTracker
	err = ptp.TrackNewOrder(nil, false)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestSetupMultiPositionTracker(t *testing.T) {
	t.Parallel()

	_, err := SetupMultiPositionTracker(nil)
	assert.ErrorIs(t, err, errNilSetup)

	setup := &MultiPositionTrackerSetup{}
	_, err = SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	setup.Exchange = testExchange
	_, err = SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	setup.Asset = asset.Futures
	_, err = SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)

	setup.Pair = currency.NewBTCUSDT()
	_, err = SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, errEmptyUnderlying)

	setup.Underlying = currency.BTC
	_, err = SetupMultiPositionTracker(setup)
	assert.NoError(t, err)

	setup.UseExchangePNLCalculation = true
	_, err = SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, errMissingPNLCalculationFunctions)

	setup.ExchangePNLCalculation = &FakePNL{}
	resp, err := SetupMultiPositionTracker(setup)
	assert.NoError(t, err)

	if resp.exchange != testExchange {
		t.Errorf("expected 'test' received %v", resp.exchange)
	}
}

func TestMultiPositionTrackerTrackNewOrder(t *testing.T) {
	t.Parallel()
	exch := testExchange
	item := asset.Futures
	pair := currency.NewBTCUSDT()
	setup := &MultiPositionTrackerSetup{
		Asset:                  item,
		Pair:                   pair,
		Underlying:             pair.Base,
		ExchangePNLCalculation: &FakePNL{},
	}
	_, err := SetupMultiPositionTracker(setup)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	setup.Exchange = testExchange
	resp, err := SetupMultiPositionTracker(setup)
	assert.NoError(t, err)

	tt := time.Now()
	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		AssetType: item,
		Pair:      pair,
		Side:      order.Short,
		OrderID:   "1",
		Amount:    1,
	})
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Short,
		OrderID:   "1",
		Amount:    1,
	})
	assert.NoError(t, err)

	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Short,
		OrderID:   "2",
		Amount:    1,
	})
	assert.NoError(t, err)

	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Long,
		OrderID:   "3",
		Amount:    2,
	})
	assert.NoError(t, err)

	if len(resp.positions) != 1 {
		t.Errorf("expected '1' received %v", len(resp.positions))
	}
	if resp.positions[0].status != order.Closed {
		t.Errorf("expected 'closed' received %v", resp.positions[0].status)
	}
	resp.positions[0].status = order.Open
	resp.positions = append(resp.positions, resp.positions...)
	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Long,
		OrderID:   "4",
		Amount:    2,
	})
	assert.ErrorIs(t, err, errPositionDiscrepancy)

	resp.positions = []*PositionTracker{resp.positions[0]}
	resp.positions[0].status = order.Closed
	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Long,
		OrderID:   "4",
		Amount:    2,
	})
	assert.NoError(t, err)

	if len(resp.positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.positions))
	}

	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		AssetType: item,
		Pair:      pair,
		Side:      order.Long,
		OrderID:   "4",
		Amount:    2,
	})
	assert.NoError(t, err)

	if len(resp.positions) != 2 {
		t.Errorf("expected '2' received %v", len(resp.positions))
	}

	resp.positions[0].status = order.Closed
	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		Pair:      pair,
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Long,
		OrderID:   "5",
		Amount:    2,
	})
	assert.ErrorIs(t, err, errAssetMismatch)

	err = resp.TrackNewOrder(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	resp = nil
	err = resp.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  exch,
		Pair:      pair,
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Long,
		OrderID:   "5",
		Amount:    2,
	})
	assert.ErrorIs(t, err, common.ErrNilPointer)
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
	assert.ErrorIs(t, err, errNilOrder)

	err = pc.TrackNewOrder(&order.Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Spot,
		Side:      order.Long,
		OrderID:   "lol",
	})
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	err = pc.TrackNewOrder(&order.Detail{
		Date:      time.Now(),
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
	})
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	err = pc.TrackNewOrder(&order.Detail{
		Exchange:  testExchange,
		Date:      time.Now(),
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
	})
	assert.NoError(t, err)

	var pcp *PositionController
	err = pcp.TrackNewOrder(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestGetLatestPNLSnapshot(t *testing.T) {
	t.Parallel()
	pt := PositionTracker{}
	_, err := pt.GetLatestPNLSnapshot()
	assert.ErrorIs(t, err, errNoPNLHistory)

	pnl := PNLResult{
		Time:                  time.Now(),
		UnrealisedPNL:         decimal.NewFromInt(1337),
		RealisedPNLBeforeFees: decimal.NewFromInt(1337),
	}
	pt.pnlHistory = append(pt.pnlHistory, pnl)

	result, err := pt.GetLatestPNLSnapshot()
	assert.NoError(t, err)

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
	p.fundingRateDetails = &fundingrate.HistoricalRates{
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
	p := currency.NewBTCUSDT()

	_, err := c.GetPositionsForExchange("", asset.Futures, p)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	pos, err := c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	if len(pos) != 0 {
		t.Error("expected zero")
	}
	c.multiPositionTrackers = make(map[key.ExchangePairAsset]*MultiPositionTracker)
	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}] = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	_, err = c.GetPositionsForExchange(testExchange, asset.Spot, p)
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}] = &MultiPositionTracker{
		exchange: testExchange,
	}

	pos, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.NoError(t, err)

	if len(pos) != 0 {
		t.Fatal("expected zero")
	}
	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}] = &MultiPositionTracker{
		exchange: testExchange,
		positions: []*PositionTracker{
			{
				exchange: testExchange,
			},
		},
	}
	pos, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.NoError(t, err)

	if len(pos) != 1 {
		t.Fatal("expected 1")
	}
	if pos[0].Exchange != testExchange {
		t.Error("expected test")
	}
	c = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestClearPositionsForExchange(t *testing.T) {
	t.Parallel()
	c := &PositionController{}
	p := currency.NewBTCUSDT()
	err := c.ClearPositionsForExchange("", asset.Futures, p)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	c.multiPositionTrackers = make(map[key.ExchangePairAsset]*MultiPositionTracker)
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	err = c.ClearPositionsForExchange(testExchange, asset.Spot, p)
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}] = &MultiPositionTracker{
		exchange:   testExchange,
		underlying: currency.DOGE,
		positions: []*PositionTracker{
			{
				exchange: testExchange,
			},
		},
	}
	err = c.ClearPositionsForExchange(testExchange, asset.Futures, p)
	assert.NoError(t, err)

	if len(c.multiPositionTrackers[key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}].positions) != 0 {
		t.Fatal("expected 0")
	}
	c = nil
	_, err = c.GetPositionsForExchange(testExchange, asset.Futures, p)
	assert.ErrorIs(t, err, common.ErrNilPointer)
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
	p, err := SetupPositionTracker(nil)
	assert.ErrorIs(t, err, errNilSetup)

	if p != nil {
		t.Error("expected nil")
	}
	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	if p != nil {
		t.Error("expected nil")
	}

	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Spot,
	})
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	if p != nil {
		t.Error("expected nil")
	}

	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Futures,
	})
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)

	if p != nil {
		t.Error("expected nil")
	}

	cp := currency.NewBTCUSDT()
	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
	})
	require.NoError(t, err)

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
	assert.ErrorIs(t, err, ErrNilPNLCalculator)

	p, err = SetupPositionTracker(&PositionTrackerSetup{
		Exchange:                  testExchange,
		Asset:                     asset.Futures,
		Pair:                      cp,
		UseExchangePNLCalculation: true,
		PNLCalculator:             &PNLCalculator{},
	})
	assert.NoError(t, err)

	if !p.useExchangePNLCalculation {
		t.Error("expected true")
	}
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	p := &PNLCalculator{}
	_, err := p.CalculatePNL(t.Context(), nil)
	assert.ErrorIs(t, err, ErrNilPNLCalculator)

	_, err = p.CalculatePNL(t.Context(), &PNLCalculatorRequest{})
	assert.ErrorIs(t, err, errCannotCalculateUnrealisedPNL)

	_, err = p.CalculatePNL(t.Context(),
		&PNLCalculatorRequest{
			OrderDirection:   order.Short,
			CurrentDirection: order.Long,
		})
	assert.ErrorIs(t, err, errCannotCalculateUnrealisedPNL)
}

func TestTrackPNLByTime(t *testing.T) {
	t.Parallel()
	p := &PositionTracker{}
	err := p.TrackPNLByTime(time.Now(), 1)
	assert.NoError(t, err)

	err = p.TrackPNLByTime(time.Now(), 2)
	assert.NoError(t, err)

	if !p.latestPrice.Equal(decimal.NewFromInt(2)) {
		t.Error("expected 2")
	}
	p = nil
	err = p.TrackPNLByTime(time.Now(), 2)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()

	_, err := pc.UpdateOpenPositionUnrealisedPNL("", asset.Futures, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, ErrPositionNotFound)

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Spot, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	err = pc.TrackNewOrder(&order.Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	assert.NoError(t, err)

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi2", asset.Futures, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, ErrPositionNotFound)

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.PerpetualSwap, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, ErrPositionNotFound)

	_, err = pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewPair(currency.BTC, currency.DOGE), 2, time.Now())
	assert.ErrorIs(t, err, ErrPositionNotFound)

	pnl, err := pc.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewBTCUSDT(), 2, time.Now())
	assert.NoError(t, err)

	if !pnl.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v' expected '%v", pnl, 1)
	}

	var nilPC *PositionController
	_, err = nilPC.UpdateOpenPositionUnrealisedPNL("hi", asset.Futures, currency.NewBTCUSDT(), 2, time.Now())
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestSetCollateralCurrency(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	err := pc.SetCollateralCurrency("", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	err = pc.SetCollateralCurrency("hi", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	p := currency.NewBTCUSDT()
	pc.multiPositionTrackers = make(map[key.ExchangePairAsset]*MultiPositionTracker)
	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	require.ErrorIs(t, err, ErrPositionNotFound)

	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	require.ErrorIs(t, err, ErrPositionNotFound)

	mapKey := key.ExchangePairAsset{
		Exchange: "hi",
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}

	pc.multiPositionTrackers[mapKey] = &MultiPositionTracker{
		exchange:       "hi",
		asset:          asset.Futures,
		pair:           p,
		orderPositions: make(map[string]*PositionTracker),
	}
	err = pc.TrackNewOrder(&order.Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      p,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	require.NoError(t, err)

	err = pc.SetCollateralCurrency("hi", asset.Futures, p, currency.DOGE)
	require.NoError(t, err)

	if !pc.multiPositionTrackers[mapKey].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.multiPositionTrackers[mapKey].collateralCurrency, currency.DOGE)
	}

	if !pc.multiPositionTrackers[mapKey].positions[0].collateralCurrency.Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", pc.multiPositionTrackers[mapKey].positions[0].collateralCurrency, currency.DOGE)
	}

	var nilPC *PositionController
	err = nilPC.SetCollateralCurrency("hi", asset.Spot, currency.EMPTYPAIR, currency.Code{})
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestMPTUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	pc := SetupPositionController()
	err := pc.TrackNewOrder(&order.Detail{
		Date:      time.Now(),
		Exchange:  "hi",
		Pair:      p,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	require.NoError(t, err)

	mapKey := key.ExchangePairAsset{
		Exchange: "hi",
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}

	result, err := pc.multiPositionTrackers[mapKey].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	require.NoError(t, err)

	if result.Equal(decimal.NewFromInt(1337)) {
		t.Error("")
	}

	pc.multiPositionTrackers[mapKey].positions[0].status = order.Closed
	_, err = pc.multiPositionTrackers[mapKey].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	require.ErrorIs(t, err, ErrPositionClosed)

	pc.multiPositionTrackers[mapKey].positions = nil
	_, err = pc.multiPositionTrackers[mapKey].UpdateOpenPositionUnrealisedPNL(1337, time.Now())
	require.ErrorIs(t, err, ErrPositionNotFound)
}

func TestMPTLiquidate(t *testing.T) {
	t.Parallel()
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	assert.NoError(t, err)

	e := &MultiPositionTracker{
		exchange:               testExchange,
		exchangePNLCalculation: &FakePNL{},
		asset:                  item,
		orderPositions:         make(map[string]*PositionTracker),
	}

	err = e.Liquidate(decimal.Zero, time.Time{})
	assert.ErrorIs(t, err, ErrPositionNotFound)

	setup := &PositionTrackerSetup{
		Pair:  pair,
		Asset: item,
	}
	_, err = SetupPositionTracker(setup)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	setup.Exchange = "exch"
	_, err = SetupPositionTracker(setup)
	assert.NoError(t, err)

	tt := time.Now()
	err = e.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  testExchange,
		Pair:      pair,
		AssetType: item,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	})
	assert.NoError(t, err)

	err = e.Liquidate(decimal.Zero, time.Time{})
	assert.ErrorIs(t, err, order.ErrCannotLiquidate)

	err = e.Liquidate(decimal.Zero, tt)
	assert.NoError(t, err)

	if e.positions[0].status != order.Liquidated {
		t.Errorf("received '%v' expected '%v'", e.positions[0].status, order.Liquidated)
	}
	if !e.positions[0].exposure.IsZero() {
		t.Errorf("received '%v' expected '%v'", e.positions[0].exposure, 0)
	}

	e = nil
	err = e.Liquidate(decimal.Zero, tt)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestPositionLiquidate(t *testing.T) {
	t.Parallel()
	item := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	assert.NoError(t, err)

	p := &PositionTracker{
		contractPair:     pair,
		asset:            item,
		exchange:         testExchange,
		PNLCalculation:   &PNLCalculator{},
		status:           order.Open,
		openingDirection: order.Long,
	}

	tt := time.Now()
	err = p.TrackNewOrder(&order.Detail{
		Date:      tt,
		Exchange:  testExchange,
		Pair:      pair,
		AssetType: item,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1,
		Amount:    1,
	}, false)
	assert.NoError(t, err)

	err = p.Liquidate(decimal.Zero, time.Time{})
	assert.ErrorIs(t, err, order.ErrCannotLiquidate)

	err = p.Liquidate(decimal.Zero, tt)
	assert.NoError(t, err)

	if p.status != order.Liquidated {
		t.Errorf("received '%v' expected '%v'", p.status, order.Liquidated)
	}
	if !p.exposure.IsZero() {
		t.Errorf("received '%v' expected '%v'", p.exposure, 0)
	}

	p = nil
	err = p.Liquidate(decimal.Zero, tt)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestGetOpenPosition(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	cp := currency.NewPair(currency.BTC, currency.PERP)
	tn := time.Now()

	_, err := pc.GetOpenPosition("", asset.Futures, cp)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	_, err = pc.GetOpenPosition(testExchange, asset.Futures, cp)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	err = pc.TrackNewOrder(&order.Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	assert.NoError(t, err)

	_, err = pc.GetOpenPosition(testExchange, asset.Futures, cp)
	assert.NoError(t, err)
}

func TestGetAllOpenPositions(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()

	_, err := pc.GetAllOpenPositions()
	assert.ErrorIs(t, err, ErrNoPositionsFound)

	cp := currency.NewPair(currency.BTC, currency.PERP)
	tn := time.Now()
	err = pc.TrackNewOrder(&order.Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	assert.NoError(t, err)

	_, err = pc.GetAllOpenPositions()
	assert.NoError(t, err)
}

func TestPCTrackFundingDetails(t *testing.T) {
	t.Parallel()
	pc := SetupPositionController()
	err := pc.TrackFundingDetails(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	p := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.HistoricalRates{
		Asset: asset.Futures,
		Pair:  p,
	}
	err = pc.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	rates.Exchange = testExchange
	err = pc.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	tn := time.Now()
	err = pc.TrackNewOrder(&order.Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      p,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	assert.NoError(t, err)

	rates.StartDate = tn.Add(-time.Hour)
	rates.EndDate = tn
	rates.FundingRates = []fundingrate.Rate{
		{
			Time:    tn,
			Rate:    decimal.NewFromInt(1337),
			Payment: decimal.NewFromInt(1337),
		},
	}

	mapKey := key.ExchangePairAsset{
		Exchange: testExchange,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    asset.Futures,
	}

	pc.multiPositionTrackers[mapKey].orderPositions["lol"].openingDate = tn.Add(-time.Hour)
	pc.multiPositionTrackers[mapKey].orderPositions["lol"].lastUpdated = tn
	err = pc.TrackFundingDetails(rates)
	assert.NoError(t, err)
}

func TestMPTTrackFundingDetails(t *testing.T) {
	t.Parallel()
	mpt := &MultiPositionTracker{
		orderPositions: make(map[string]*PositionTracker),
	}

	err := mpt.TrackFundingDetails(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	cp := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.HistoricalRates{
		Asset: asset.Futures,
		Pair:  cp,
	}
	err = mpt.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	mpt.exchange = testExchange
	rates = &fundingrate.HistoricalRates{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
	}
	err = mpt.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, errAssetMismatch)

	mpt.asset = rates.Asset
	mpt.pair = cp
	err = mpt.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, ErrPositionNotFound)

	tn := time.Now()
	err = mpt.TrackNewOrder(&order.Detail{
		Date:      tn,
		Exchange:  testExchange,
		Pair:      cp,
		AssetType: asset.Futures,
		Side:      order.Long,
		OrderID:   "lol",
		Price:     1337,
		Amount:    1337,
	})
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, errExchangeNameMismatch)
}

func TestPTTrackFundingDetails(t *testing.T) {
	t.Parallel()
	p := &PositionTracker{}
	err := p.TrackFundingDetails(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	cp := currency.NewPair(currency.BTC, currency.PERP)
	rates := &fundingrate.HistoricalRates{
		Exchange: testExchange,
		Asset:    asset.Futures,
		Pair:     cp,
	}
	err = p.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, errDoesntMatch)

	p.exchange = testExchange
	p.asset = asset.Futures
	p.contractPair = cp
	err = p.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	rates.StartDate = time.Now().Add(-time.Hour)
	rates.EndDate = time.Now()
	p.openingDate = rates.StartDate
	err = p.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, ErrNoPositionsFound)

	p.pnlHistory = append(p.pnlHistory, PNLResult{
		Time:                  rates.EndDate,
		UnrealisedPNL:         decimal.NewFromInt(1337),
		RealisedPNLBeforeFees: decimal.NewFromInt(1337),
		Price:                 decimal.NewFromInt(1337),
		Exposure:              decimal.NewFromInt(1337),
		Fee:                   decimal.NewFromInt(1337),
	})
	err = p.TrackFundingDetails(rates)
	assert.NoError(t, err)

	rates.FundingRates = []fundingrate.Rate{
		{
			Time:    rates.StartDate,
			Rate:    decimal.NewFromInt(1337),
			Payment: decimal.NewFromInt(1337),
		},
	}
	err = p.TrackFundingDetails(rates)
	assert.NoError(t, err)

	err = p.TrackFundingDetails(rates)
	assert.NoError(t, err)

	rates.StartDate = rates.StartDate.Add(-time.Hour)
	err = p.TrackFundingDetails(rates)
	assert.NoError(t, err)

	rates.Exchange = ""
	err = p.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	p = nil
	err = p.TrackFundingDetails(rates)
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestAreFundingRatePrerequisitesMet(t *testing.T) {
	t.Parallel()
	err := CheckFundingRatePrerequisites(false, false, false)
	assert.NoError(t, err)

	err = CheckFundingRatePrerequisites(true, false, false)
	assert.NoError(t, err)

	err = CheckFundingRatePrerequisites(true, true, false)
	assert.NoError(t, err)

	err = CheckFundingRatePrerequisites(true, true, true)
	assert.NoError(t, err)

	err = CheckFundingRatePrerequisites(true, false, true)
	assert.NoError(t, err)

	err = CheckFundingRatePrerequisites(false, false, true)
	assert.ErrorIs(t, err, ErrGetFundingDataRequired)

	err = CheckFundingRatePrerequisites(false, true, true)
	assert.ErrorIs(t, err, ErrGetFundingDataRequired)

	err = CheckFundingRatePrerequisites(false, true, false)
	assert.ErrorIs(t, err, ErrGetFundingDataRequired)
}

func TestLastUpdated(t *testing.T) {
	t.Parallel()
	p := &PositionController{}
	tm, err := p.LastUpdated()
	assert.NoError(t, err)

	if !tm.IsZero() {
		t.Errorf("received '%v' expected '%v", tm, time.Time{})
	}
	p.updated = time.Now()
	tm, err = p.LastUpdated()
	assert.NoError(t, err)

	if !tm.Equal(p.updated) {
		t.Errorf("received '%v' expected '%v", tm, p.updated)
	}
	p = nil
	_, err = p.LastUpdated()
	assert.ErrorIs(t, err, common.ErrNilPointer)
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
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	upperExch := "IM UPPERCASE"
	_, err = checkTrackerPrerequisitesLowerExchange(upperExch, asset.Spot, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, ErrNotFuturesAsset)

	_, err = checkTrackerPrerequisitesLowerExchange(upperExch, asset.Futures, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)

	lowerExch, err := checkTrackerPrerequisitesLowerExchange(upperExch, asset.Futures, currency.NewBTCUSDT())
	assert.NoError(t, err)

	if lowerExch != "im uppercase" {
		t.Error("expected lowercase")
	}
}

package orderbook

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMain(m *testing.M) {
	err := dispatch.Start(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit*10)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestSubscribeToExchangeOrderbooks(t *testing.T) {
	t.Parallel()
	_, err := SubscribeToExchangeOrderbooks("")
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	p := currency.NewBTCUSD()

	b := Book{
		Pair:     p,
		Asset:    asset.Spot,
		Exchange: "SubscribeToExchangeOrderbooks",
		Bids:     []Level{{Price: 100, Amount: 1}, {Price: 99, Amount: 1}},
	}

	require.NoError(t, b.Process(), "process must not error")

	_, err = SubscribeToExchangeOrderbooks("SubscribeToExchangeOrderbooks")
	assert.NoError(t, err, "SubscribeToExchangeOrderbooks should not error")
}

func TestValidate(t *testing.T) {
	t.Parallel()
	b := Book{
		Exchange:          "TestExchange",
		Asset:             asset.Spot,
		Pair:              currency.NewBTCUSD(),
		ValidateOrderbook: true,
	}

	require.NoError(t, b.Validate())

	b.Asks = []Level{{ID: 1337, Price: 99, Amount: 1}, {ID: 1337, Price: 100, Amount: 1}}
	err := b.Validate()
	require.ErrorIs(t, err, errIDDuplication)

	b.Asks = []Level{{Price: 100, Amount: 1}, {Price: 100, Amount: 1}}
	err = b.Validate()
	require.ErrorIs(t, err, errDuplication)

	b.Asks = []Level{{Price: 100, Amount: 1}, {Price: 99, Amount: 1}}
	b.IsFundingRate = true
	err = b.Validate()
	require.ErrorIs(t, err, errPeriodUnset)

	b.IsFundingRate = false

	err = b.Validate()
	require.ErrorIs(t, err, errPriceOutOfOrder)

	b.Asks = []Level{{Price: 100, Amount: 1}, {Price: 100, Amount: 0}}
	err = b.Validate()
	require.ErrorIs(t, err, errAmountInvalid)

	b.Asks = []Level{{Price: 100, Amount: 1}, {Price: 0, Amount: 100}}
	err = b.Validate()
	require.ErrorIs(t, err, ErrPriceZero)

	b.Bids = []Level{{ID: 1337, Price: 100, Amount: 1}, {ID: 1337, Price: 99, Amount: 1}}
	err = b.Validate()
	require.ErrorIs(t, err, errIDDuplication)

	b.Bids = []Level{{Price: 100, Amount: 1}, {Price: 100, Amount: 1}}
	err = b.Validate()
	require.ErrorIs(t, err, errDuplication)

	b.Bids = []Level{{Price: 99, Amount: 1}, {Price: 100, Amount: 1}}
	b.IsFundingRate = true
	err = b.Validate()
	require.ErrorIs(t, err, errPeriodUnset)

	b.IsFundingRate = false

	err = b.Validate()
	require.ErrorIs(t, err, errPriceOutOfOrder)

	b.Bids = []Level{{Price: 100, Amount: 1}, {Price: 100, Amount: 0}}
	err = b.Validate()
	require.ErrorIs(t, err, errAmountInvalid)

	b.Bids = []Level{{Price: 100, Amount: 1}, {Price: 0, Amount: 100}}
	err = b.Validate()
	require.ErrorIs(t, err, ErrPriceZero)
}

func TestTotalBidsAmount(t *testing.T) {
	t.Parallel()
	b := Book{Pair: currency.NewBTCUSD(), Bids: []Level{{Price: 100, Amount: 10}}, LastUpdated: time.Now()}
	ac, total := b.TotalBidsAmount()
	assert.Equal(t, 10.0, ac, "should return amount")
	assert.Equal(t, 1000.0, total, "should return total")
}

func TestTotalAsksAmount(t *testing.T) {
	t.Parallel()
	b := Book{Pair: currency.NewBTCUSD(), Asks: []Level{{Price: 100, Amount: 10}}}
	ac, total := b.TotalAsksAmount()
	assert.Equal(t, 10.0, ac, "should return correct amount")
	assert.Equal(t, 1000.0, total, "should return correct total")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSD()
	b := &Book{
		Pair:     pair,
		Asks:     []Level{{Price: 100, Amount: 10}},
		Bids:     []Level{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	require.NoError(t, b.Process(), "Process must not error")

	result, err := Get("Exchange", pair, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, result.Pair.Equal(pair))

	_, err = Get("nonexistent", pair, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	pair.Base = currency.NewCode("blah")
	_, err = Get("Exchange", pair, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	newCurrency := currency.NewPair(currency.BTC, currency.AUD)
	_, err = Get("Exchange", newCurrency, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	b.Pair = newCurrency
	require.NoError(t, b.Process(), "Process must not error")

	got, err := Get("Exchange", newCurrency, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, got.Pair.Equal(newCurrency))
}

func TestGetDepth(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSD()
	b := &Book{
		Pair:     pair,
		Asks:     []Level{{Price: 100, Amount: 10}},
		Bids:     []Level{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	require.NoError(t, b.Process(), "Process must not error")

	result, err := GetDepth("Exchange", pair, asset.Spot)
	require.NoError(t, err, "GetDepth must not error")
	assert.True(t, result.pair.Equal(pair))

	_, err = GetDepth("nonexistent", pair, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	pair.Base = currency.NewCode("blah")
	_, err = GetDepth("Exchange", pair, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	newCurrency := currency.NewPair(currency.BTC, currency.DOGE)
	_, err = GetDepth("Exchange", newCurrency, asset.Futures)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	b.Pair = newCurrency
	require.NoError(t, b.Process(), "Process must not error")

	_, err = GetDepth("Exchange", newCurrency, asset.Empty)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)
}

func TestBookGetDepth(t *testing.T) {
	t.Parallel()

	pair := currency.NewPair(currency.BTC, currency.UST)
	b := &Book{
		Pair:     pair,
		Asks:     []Level{{Price: 100, Amount: 10}},
		Bids:     []Level{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	_, err := b.GetDepth()
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	require.NoError(t, b.Process(), "Process must not error")

	result, err := b.GetDepth()
	require.NoError(t, err, "GetDepth must not error")
	assert.True(t, result.pair.Equal(pair))
}

func TestDeployDepth(t *testing.T) {
	pair := currency.NewBTCUSD()
	_, err := DeployDepth("", pair, asset.Spot)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)
	_, err = DeployDepth("test", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, errPairNotSet)
	_, err = DeployDepth("test", pair, asset.Empty)
	require.ErrorIs(t, err, errAssetTypeNotSet)
	d, err := DeployDepth("test", pair, asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, d)
	_, err = DeployDepth("test", pair, asset.Spot)
	require.NoError(t, err)
}

func TestProcessOrderbook(t *testing.T) {
	b := Book{
		Asks:     []Level{{Price: 100, Amount: 10}},
		Bids:     []Level{{Price: 200, Amount: 10}},
		Exchange: "ProcessOrderbook",
	}

	// test for empty pair
	err := b.Process()
	assert.ErrorIs(t, err, errPairNotSet)

	// test for empty asset type
	pair := currency.NewBTCUSD()
	b.Pair = pair
	err = b.Process()
	require.ErrorIs(t, err, errAssetTypeNotSet)

	// now process a valid orderbook
	b.Asset = asset.Spot
	require.NoError(t, b.Process(), "Process must not error")

	result, err := Get("ProcessOrderbook", currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, result.Pair.Equal(pair))

	// now test for processing a pair with a different quote currency
	pair, err = currency.NewPairFromStrings("BTC", "GBP")
	require.NoError(t, err)

	b.Pair = pair
	require.NoError(t, b.Process(), "Process must not error")

	result, err = Get("ProcessOrderbook", pair, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, result.Pair.Equal(pair))

	// now test for processing a pair which has a different base currency
	pair, err = currency.NewPairFromStrings("LTC", "GBP")
	require.NoError(t, err, "NewPairFromStrings must not error")

	b.Pair = pair
	require.NoError(t, b.Process(), "Process must not error")

	result, err = Get("ProcessOrderbook", pair, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, result.Pair.Equal(pair))

	b.Asks = []Level{{Price: 200, Amount: 200}}
	b.Asset = asset.Spot
	require.NoError(t, b.Process(), "Process must not error")

	result, err = Get("ProcessOrderbook", pair, asset.Spot)
	require.NoError(t, err, "Get must not error")

	ac, total := result.TotalAsksAmount()
	assert.Equal(t, 200.0, ac, "TotalAsksAmount should return 200")
	assert.Equal(t, 40000.0, total, "TotalAsksAmount should return 40000")

	b.Bids = []Level{{Price: 420, Amount: 200}}
	b.Exchange = "Blah"
	b.Asset = asset.CoinMarginedFutures

	require.NoError(t, b.Process(), "Process must not error")

	result, err = Get("Blah", pair, asset.CoinMarginedFutures)
	require.NoError(t, err, "Get must not error")

	ac, total = result.TotalBidsAmount()
	assert.Equal(t, 200.0, ac, "TotalBidsAmount should return 200")
	assert.Equal(t, 84000.0, total, "TotalBidsAmount should return 84000")

	type quick struct {
		Name string
		P    currency.Pair
		Bids []Level
		Asks []Level
	}

	var testArray []quick

	var wg sync.WaitGroup
	var m sync.Mutex

	var catastrophicFailure bool

	for range 500 {
		m.Lock()
		if catastrophicFailure {
			m.Unlock()
			break
		}
		m.Unlock()
		wg.Go(func() {
			newName := "Exchange" + strconv.FormatInt(rand.Int64(), 10) //nolint:gosec // no need to import crypto/rand for testing
			newPairs := currency.NewPair(currency.NewCode("BTC"+strconv.FormatInt(rand.Int64(), 10)),
				currency.NewCode("USD"+strconv.FormatInt(rand.Int64(), 10))) //nolint:gosec // no need to import crypto/rand for testing

			asks := []Level{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypto/rand for testing
			bids := []Level{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypto/rand for testing
			b := &Book{
				Pair:     newPairs,
				Asks:     asks,
				Bids:     bids,
				Exchange: newName,
				Asset:    asset.Spot,
			}

			m.Lock()
			err = b.Process()
			if err != nil {
				t.Error(err)
				catastrophicFailure = true
				m.Unlock()
				return
			}
			testArray = append(testArray, quick{Name: newName, P: newPairs, Bids: bids, Asks: asks})
			m.Unlock()
		})
	}

	wg.Wait()
	if catastrophicFailure {
		t.Fatal("Process() error", err)
	}

	var collector common.ErrorCollector
	for _, test := range testArray {
		collector.Go(func() error {
			result, err := Get(test.Name, test.P, asset.Spot)
			if err != nil {
				return fmt.Errorf("TestProcessOrderbook failed to retrieve new orderbook: %w", err)
			}

			if result.Asks[0] != test.Asks[0] {
				return errors.New("TestProcessOrderbook failed bad ask values")
			}

			if result.Bids[0] != test.Bids[0] {
				return errors.New("TestProcessOrderbook failed bad bid values")
			}
			return nil
		})
	}
	require.NoError(t, collector.Collect())
}

func levelsFixtureRandom() Levels {
	lvls := make([]Level, 1000)
	for x := range 1000 {
		lvls[x] = Level{Amount: 1, Price: rand.Float64(), ID: rand.Int64()} //nolint:gosec // Not needed in tests
	}
	return lvls
}

func TestSorting(t *testing.T) {
	var b Book
	b.ValidateOrderbook = true

	b.Asks = levelsFixtureRandom()
	err := b.Validate()
	require.ErrorIs(t, err, errPriceOutOfOrder)

	b.Asks.SortAsks()
	err = b.Validate()
	require.NoError(t, err)

	b.Bids = levelsFixtureRandom()
	err = b.Validate()
	require.ErrorIs(t, err, errPriceOutOfOrder)

	b.Bids.SortBids()
	err = b.Validate()
	require.NoError(t, err)
}

func TestSortAsks(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		levels Levels
	}{
		{name: "nil"},
		{name: "empty", levels: Levels{}},
		{name: "singleton", levels: Levels{{Price: 1, Amount: 2, ID: 1}}},
		{name: "ordered duplicates", levels: Levels{{Price: 1, ID: 1}, {Price: 2, ID: 2}, {Price: 2, ID: 3}, {Price: 3, ID: 4}}},
		{name: "long ordered duplicates", levels: Levels{{Price: 1, ID: 1}, {Price: 1, ID: 2}, {Price: 2, ID: 3}, {Price: 2, ID: 4}, {Price: 3, ID: 5}, {Price: 3, ID: 6}, {Price: 4, ID: 7}, {Price: 4, ID: 8}, {Price: 5, ID: 9}, {Price: 5, ID: 10}, {Price: 6, ID: 11}, {Price: 6, ID: 12}, {Price: 7, ID: 13}, {Price: 7, ID: 14}, {Price: 8, ID: 15}, {Price: 8, ID: 16}, {Price: 9, ID: 17}, {Price: 9, ID: 18}, {Price: 10, ID: 19}, {Price: 10, ID: 20}}},
		{
			name: "long scattered duplicates",
			levels: func() Levels {
				rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Deterministic test fixture.
				levels := make(Levels, 100)
				for i := range levels {
					levels[i] = Level{Price: float64(rng.IntN(16) + 1), ID: int64(i + 1)}
				}
				return levels
			}(),
		},
		{name: "final inversion", levels: Levels{{Price: 1, ID: 1}, {Price: 2, ID: 2}, {Price: 2, ID: 3}, {Price: 0.5, ID: 4}}},
		{name: "reverse ordered", levels: Levels{{Price: 3, ID: 1}, {Price: 2, ID: 2}, {Price: 1, ID: 3}}},
		{name: "infinities", levels: Levels{{Price: math.Inf(1), ID: 1}, {Price: 0, ID: 2}, {Price: math.Inf(-1), ID: 3}}},
		{name: "signed zero ordered", levels: Levels{{Price: -1, ID: 1}, {Price: math.Copysign(0, -1), ID: 2}, {Price: 0, ID: 3}, {Price: 1, ID: 4}}},
		{name: "signed zero after inversion", levels: Levels{{Price: 1, ID: 1}, {Price: math.Copysign(0, -1), ID: 2}, {Price: 0, ID: 3}, {Price: -1, ID: 4}}},
		{name: "NaN first", levels: Levels{{Price: math.NaN(), ID: 1}, {Price: 1, ID: 2}, {Price: 2, ID: 3}}},
		{name: "NaN final", levels: Levels{{Price: 1, ID: 1}, {Price: 2, ID: 2}, {Price: math.NaN(), ID: 3}}},
		{name: "NaN after inversion", levels: Levels{{Price: 2, ID: 1}, {Price: 1, ID: 2}, {Price: math.NaN(), ID: 3}}},
		{name: "multiple NaNs", levels: Levels{{Price: 3, ID: 1}, {Price: math.NaN(), ID: 2}, {Price: 1, ID: 3}, {Price: math.NaN(), ID: 4}, {Price: 2, ID: 5}}},
		{
			name: "NaN masks non-adjacent inversion",
			levels: Levels{
				{Price: 1, ID: 1},
				{Price: 2, ID: 2},
				{Price: 3, ID: 3},
				{Price: 4, ID: 4},
				{Price: 10, ID: 5},
				{Price: math.NaN(), ID: 6},
				{Price: math.NaN(), ID: 7},
				{Price: math.NaN(), ID: 8},
				{Price: 5, ID: 9},
				{Price: 6, ID: 10},
				{Price: 7, ID: 11},
				{Price: math.NaN(), ID: 12},
				{Price: 1, ID: 13},
				{Price: 2, ID: 14},
				{Price: 3, ID: 15},
				{Price: 4, ID: 16},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expected := slices.Clone(tc.levels)
			sort.Slice(expected, func(i, j int) bool { return expected[i].Price < expected[j].Price })
			actual := slices.Clone(tc.levels)
			actual.SortAsks()
			if !slices.ContainsFunc(tc.levels, func(level Level) bool { return math.IsNaN(level.Price) }) {
				assert.Equal(t, expected, actual, "SortAsks should preserve legacy ordering")
				return
			}

			expectedIDs := make([]int64, len(expected))
			actualIDs := make([]int64, len(actual))
			for i := range expected {
				expectedIDs[i] = expected[i].ID
				actualIDs[i] = actual[i].ID
			}
			assert.Equal(t, expectedIDs, actualIDs, "SortAsks should preserve legacy NaN ordering")
		})
	}
}

func TestSortBids(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		levels Levels
	}{
		{name: "nil"},
		{name: "empty", levels: Levels{}},
		{name: "singleton", levels: Levels{{Price: 1, Amount: 2, ID: 1}}},
		{name: "ordered duplicates", levels: Levels{{Price: 3, ID: 1}, {Price: 2, ID: 2}, {Price: 2, ID: 3}, {Price: 1, ID: 4}}},
		{name: "long ordered duplicates", levels: Levels{{Price: 10, ID: 1}, {Price: 10, ID: 2}, {Price: 9, ID: 3}, {Price: 9, ID: 4}, {Price: 8, ID: 5}, {Price: 8, ID: 6}, {Price: 7, ID: 7}, {Price: 7, ID: 8}, {Price: 6, ID: 9}, {Price: 6, ID: 10}, {Price: 5, ID: 11}, {Price: 5, ID: 12}, {Price: 4, ID: 13}, {Price: 4, ID: 14}, {Price: 3, ID: 15}, {Price: 3, ID: 16}, {Price: 2, ID: 17}, {Price: 2, ID: 18}, {Price: 1, ID: 19}, {Price: 1, ID: 20}}},
		{
			name: "long scattered duplicates",
			levels: func() Levels {
				rng := rand.New(rand.NewPCG(2, 2)) //nolint:gosec // Deterministic test fixture.
				levels := make(Levels, 100)
				for i := range levels {
					levels[i] = Level{Price: float64(rng.IntN(16) + 1), ID: int64(i + 1)}
				}
				return levels
			}(),
		},
		{name: "final inversion", levels: Levels{{Price: 3, ID: 1}, {Price: 2, ID: 2}, {Price: 2, ID: 3}, {Price: 4, ID: 4}}},
		{name: "reverse ordered", levels: Levels{{Price: 1, ID: 1}, {Price: 2, ID: 2}, {Price: 3, ID: 3}}},
		{name: "infinities", levels: Levels{{Price: math.Inf(-1), ID: 1}, {Price: 0, ID: 2}, {Price: math.Inf(1), ID: 3}}},
		{name: "signed zero ordered", levels: Levels{{Price: 1, ID: 1}, {Price: 0, ID: 2}, {Price: math.Copysign(0, -1), ID: 3}, {Price: -1, ID: 4}}},
		{name: "signed zero after inversion", levels: Levels{{Price: -1, ID: 1}, {Price: 0, ID: 2}, {Price: math.Copysign(0, -1), ID: 3}, {Price: 1, ID: 4}}},
		{name: "NaN first", levels: Levels{{Price: math.NaN(), ID: 1}, {Price: 2, ID: 2}, {Price: 1, ID: 3}}},
		{name: "NaN final", levels: Levels{{Price: 2, ID: 1}, {Price: 1, ID: 2}, {Price: math.NaN(), ID: 3}}},
		{name: "NaN after inversion", levels: Levels{{Price: 1, ID: 1}, {Price: 2, ID: 2}, {Price: math.NaN(), ID: 3}}},
		{name: "multiple NaNs", levels: Levels{{Price: 1, ID: 1}, {Price: math.NaN(), ID: 2}, {Price: 3, ID: 3}, {Price: math.NaN(), ID: 4}, {Price: 2, ID: 5}}},
		{
			name: "NaN masks non-adjacent inversion",
			levels: Levels{
				{Price: 10, ID: 1},
				{Price: 9, ID: 2},
				{Price: 8, ID: 3},
				{Price: 7, ID: 4},
				{Price: 1, ID: 5},
				{Price: math.NaN(), ID: 6},
				{Price: math.NaN(), ID: 7},
				{Price: math.NaN(), ID: 8},
				{Price: 6, ID: 9},
				{Price: 5, ID: 10},
				{Price: 4, ID: 11},
				{Price: math.NaN(), ID: 12},
				{Price: 10, ID: 13},
				{Price: 9, ID: 14},
				{Price: 8, ID: 15},
				{Price: 7, ID: 16},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expected := slices.Clone(tc.levels)
			sort.Slice(expected, func(i, j int) bool { return expected[i].Price > expected[j].Price })
			actual := slices.Clone(tc.levels)
			actual.SortBids()
			if !slices.ContainsFunc(tc.levels, func(level Level) bool { return math.IsNaN(level.Price) }) {
				assert.Equal(t, expected, actual, "SortBids should preserve legacy ordering")
				return
			}

			expectedIDs := make([]int64, len(expected))
			actualIDs := make([]int64, len(actual))
			for i := range expected {
				expectedIDs[i] = expected[i].ID
				actualIDs[i] = actual[i].ID
			}
			assert.Equal(t, expectedIDs, actualIDs, "SortBids should preserve legacy NaN ordering")
		})
	}
}

func levelsFixture() Levels {
	lvls := make(Levels, 1000)
	for i := range 1000 {
		lvls[i] = Level{Amount: 1, Price: float64(i + 1), ID: rand.Int64()} //nolint:gosec // Not needed in tests
	}
	return lvls
}

func TestReverse(t *testing.T) {
	b := Book{ValidateOrderbook: true, Bids: levelsFixture()}
	assert.ErrorIs(t, b.Validate(), errPriceOutOfOrder)

	b.Bids.Reverse()
	assert.NoError(t, b.Validate())

	b.Asks = slices.Clone(b.Bids)
	assert.ErrorIs(t, b.Validate(), errPriceOutOfOrder)

	b.Asks.Reverse()
	assert.NoError(t, b.Validate())
}

func TestCheckAlignment(t *testing.T) {
	t.Parallel()
	itemWithFunding := Levels{{Amount: 1337, Price: 0, Period: 1337}}
	err := checkAlignment(itemWithFunding, true, true, false, false, isDsc, "Bitfinex")
	if err != nil {
		t.Error(err)
	}
	err = checkAlignment(itemWithFunding, false, true, false, false, isDsc, "Bitfinex")
	require.ErrorIs(t, err, ErrPriceZero)

	err = checkAlignment(itemWithFunding, true, true, false, false, isDsc, "Binance")
	require.ErrorIs(t, err, ErrPriceZero)

	itemWithFunding[0].Price = 1337
	err = checkAlignment(itemWithFunding, true, true, false, true, isDsc, "Binance")
	require.ErrorIs(t, err, errChecksumStringNotSet)

	itemWithFunding[0].StrAmount = "1337.0000000"
	itemWithFunding[0].StrPrice = "1337.0000000"
	err = checkAlignment(itemWithFunding, true, true, false, true, isDsc, "Binance")
	require.NoError(t, err)
}

func TestLevelsArrayPriceAmountUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var asks LevelsArrayPriceAmount
	err := asks.UnmarshalJSON([]byte(`[[1,2],["3","4"]]`))
	require.NoError(t, err)
	assert.Len(t, asks, 2)
	assert.Equal(t, 1.0, asks[0].Price)
	assert.Equal(t, 2.0, asks[0].Amount)
	assert.Equal(t, 3.0, asks[1].Price)
	assert.Equal(t, 4.0, asks[1].Amount)
	assert.Equal(t, 2, len(asks.Levels()))

	err = asks.UnmarshalJSON([]byte(`invalid`))
	assert.Error(t, err)
}

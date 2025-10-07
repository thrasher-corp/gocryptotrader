package orderbook

import (
	"log"
	"math/rand"
	"os"
	"slices"
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

	_ = rand.NewSource(time.Now().Unix())

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
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10) //nolint:gosec // no need to import crypo/rand for testing
			newPairs := currency.NewPair(currency.NewCode("BTC"+strconv.FormatInt(rand.Int63(), 10)),
				currency.NewCode("USD"+strconv.FormatInt(rand.Int63(), 10))) //nolint:gosec // no need to import crypo/rand for testing

			asks := []Level{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypo/rand for testing
			bids := []Level{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypo/rand for testing
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

	for _, test := range testArray {
		wg.Add(1)
		fatalErr := false
		go func(q quick) {
			result, err := Get(q.Name, q.P, asset.Spot)
			if err != nil {
				fatalErr = true
				return
			}

			if result.Asks[0] != q.Asks[0] {
				t.Error("TestProcessOrderbook failed bad values")
			}

			if result.Bids[0] != q.Bids[0] {
				t.Error("TestProcessOrderbook failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
		}
	}
	wg.Wait()
}

func levelsFixtureRandom() Levels {
	lvls := make([]Level, 1000)
	for x := range 1000 {
		lvls[x] = Level{Amount: 1, Price: rand.Float64(), ID: rand.Int63()} //nolint:gosec // Not needed in tests
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

func levelsFixture() Levels {
	lvls := make(Levels, 1000)
	for i := range 1000 {
		lvls[i] = Level{Amount: 1, Price: float64(i + 1), ID: rand.Int63()} //nolint:gosec // Not needed in tests
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

// 705985	      1856 ns/op	       0 B/op	       0 allocs/op
func BenchmarkReverse(b *testing.B) {
	lvls := levelsFixture()
	if len(lvls) != 1000 {
		b.Fatal("incorrect length")
	}

	for b.Loop() {
		lvls.Reverse()
	}
}

// 361266	      3556 ns/op	      24 B/op	       1 allocs/op (old)
// 385783	      3000 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksDecending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 266998	      4292 ns/op	      40 B/op	       2 allocs/op (old)
// 372396	      3001 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsAscending(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// 22119	     46532 ns/op	      35 B/op	       1 allocs/op (old)
// 16233	     76951 ns/op	     167 B/op	       3 allocs/op (new)
func BenchmarkSortAsksStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 19504	     62518 ns/op	      53 B/op	       2 allocs/op (old)
// 15698	     72859 ns/op	     168 B/op	       3 allocs/op (new)
func BenchmarkSortBidsStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// 376708	      3559 ns/op	      24 B/op 		   1 allocs/op (old)
// 377113	      3020 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksAscending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 262874	      4364 ns/op	      40 B/op	       2 allocs/op (old)
// 401788	      3348 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsDescending(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
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

// 5572401	       210.9 ns/op	       0 B/op	       0 allocs/op (current)
// 3748009	       312.7 ns/op	      32 B/op	       1 allocs/op (previous)
func BenchmarkProcess(b *testing.B) {
	book := &Book{
		Pair:     currency.NewBTCUSD(),
		Asks:     make(Levels, 100),
		Bids:     make(Levels, 100),
		Exchange: "BenchmarkProcessOrderbook",
		Asset:    asset.Spot,
	}

	for b.Loop() {
		if err := book.Process(); err != nil {
			b.Fatal(err)
		}
	}
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

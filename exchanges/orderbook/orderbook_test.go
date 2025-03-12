package orderbook

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	p := currency.NewPair(currency.BTC, currency.USD)

	b := Base{
		Pair:     p,
		Asset:    asset.Spot,
		Exchange: "SubscribeToExchangeOrderbooks",
		Bids:     []Tranche{{Price: 100, Amount: 1}, {Price: 99, Amount: 1}},
	}

	require.NoError(t, b.Process(), "process must not error")

	_, err = SubscribeToExchangeOrderbooks("SubscribeToExchangeOrderbooks")
	assert.NoError(t, err, "SubscribeToExchangeOrderbooks should not error")
}

func TestVerify(t *testing.T) {
	t.Parallel()
	b := Base{
		Exchange:        "TestExchange",
		Asset:           asset.Spot,
		Pair:            currency.NewPair(currency.BTC, currency.USD),
		VerifyOrderbook: true,
	}

	err := b.Verify()
	if err != nil {
		t.Fatalf("expecting %v error but received %v", nil, err)
	}

	b.Asks = []Tranche{{ID: 1337, Price: 99, Amount: 1}, {ID: 1337, Price: 100, Amount: 1}}
	err = b.Verify()
	if !errors.Is(err, errIDDuplication) {
		t.Fatalf("expecting %s error but received %v", errIDDuplication, err)
	}

	b.Asks = []Tranche{{Price: 100, Amount: 1}, {Price: 100, Amount: 1}}
	err = b.Verify()
	if !errors.Is(err, errDuplication) {
		t.Fatalf("expecting %s error but received %v", errDuplication, err)
	}

	b.Asks = []Tranche{{Price: 100, Amount: 1}, {Price: 99, Amount: 1}}
	b.IsFundingRate = true
	err = b.Verify()
	if !errors.Is(err, errPeriodUnset) {
		t.Fatalf("expecting %s error but received %v", errPeriodUnset, err)
	}
	b.IsFundingRate = false

	err = b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("expecting %s error but received %v", errPriceOutOfOrder, err)
	}

	b.Asks = []Tranche{{Price: 100, Amount: 1}, {Price: 100, Amount: 0}}
	err = b.Verify()
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("expecting %s error but received %v", errAmountInvalid, err)
	}

	b.Asks = []Tranche{{Price: 100, Amount: 1}, {Price: 0, Amount: 100}}
	err = b.Verify()
	if !errors.Is(err, errPriceNotSet) {
		t.Fatalf("expecting %s error but received %v", errPriceNotSet, err)
	}

	b.Bids = []Tranche{{ID: 1337, Price: 100, Amount: 1}, {ID: 1337, Price: 99, Amount: 1}}
	err = b.Verify()
	if !errors.Is(err, errIDDuplication) {
		t.Fatalf("expecting %s error but received %v", errIDDuplication, err)
	}

	b.Bids = []Tranche{{Price: 100, Amount: 1}, {Price: 100, Amount: 1}}
	err = b.Verify()
	if !errors.Is(err, errDuplication) {
		t.Fatalf("expecting %s error but received %v", errDuplication, err)
	}

	b.Bids = []Tranche{{Price: 99, Amount: 1}, {Price: 100, Amount: 1}}
	b.IsFundingRate = true
	err = b.Verify()
	if !errors.Is(err, errPeriodUnset) {
		t.Fatalf("expecting %s error but received %v", errPeriodUnset, err)
	}
	b.IsFundingRate = false

	err = b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("expecting %s error but received %v", errPriceOutOfOrder, err)
	}

	b.Bids = []Tranche{{Price: 100, Amount: 1}, {Price: 100, Amount: 0}}
	err = b.Verify()
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("expecting %s error but received %v", errAmountInvalid, err)
	}

	b.Bids = []Tranche{{Price: 100, Amount: 1}, {Price: 0, Amount: 100}}
	err = b.Verify()
	if !errors.Is(err, errPriceNotSet) {
		t.Fatalf("expecting %s error but received %v", errPriceNotSet, err)
	}
}

func TestCalculateTotalBids(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Pair:        curr,
		Bids:        []Tranche{{Price: 100, Amount: 10}},
		LastUpdated: time.Now(),
	}

	a, b := base.TotalBidsAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCalculateTotalBids expected a = 10 and b = 1000")
	}
}

func TestCalculateTotalAsks(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Pair: curr,
		Asks: []Tranche{{Price: 100, Amount: 10}},
	}

	a, b := base.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCalculateTotalAsks expected a = 10 and b = 1000")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	c, err := currency.NewPairFromStrings("BTC", "USD")
	require.NoError(t, err, "NewPairFromStrings must not error")

	base := &Base{
		Pair:     c,
		Asks:     []Tranche{{Price: 100, Amount: 10}},
		Bids:     []Tranche{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	require.NoError(t, base.Process(), "Process must not error")

	result, err := Get("Exchange", c, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, result.Pair.Equal(c))

	_, err = Get("nonexistent", c, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	c.Base = currency.NewCode("blah")
	_, err = Get("Exchange", c, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	newCurrency, err := currency.NewPairFromStrings("BTC", "AUD")
	require.NoError(t, err, "NewPairFromStrings must not error")

	_, err = Get("Exchange", newCurrency, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	base.Pair = newCurrency
	require.NoError(t, base.Process(), "Process must not error")

	got, err := Get("Exchange", newCurrency, asset.Spot)
	require.NoError(t, err, "Get must not error")
	assert.True(t, got.Pair.Equal(newCurrency))
}

func TestGetDepth(t *testing.T) {
	t.Parallel()

	c, err := currency.NewPairFromStrings("BTC", "USD")
	require.NoError(t, err, "NewPairFromStrings must not error")

	base := &Base{
		Pair:     c,
		Asks:     []Tranche{{Price: 100, Amount: 10}},
		Bids:     []Tranche{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	require.NoError(t, base.Process(), "Process must not error")

	result, err := GetDepth("Exchange", c, asset.Spot)
	require.NoError(t, err, "GetDepth must not error")
	assert.True(t, result.pair.Equal(c))

	_, err = GetDepth("nonexistent", c, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	c.Base = currency.NewCode("blah")
	_, err = GetDepth("Exchange", c, asset.Spot)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	newCurrency, err := currency.NewPairFromStrings("BTC", "DOGE")
	require.NoError(t, err, "NewPairFromStrings must not error")

	_, err = GetDepth("Exchange", newCurrency, asset.Futures)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	base.Pair = newCurrency
	require.NoError(t, base.Process(), "Process must not error")

	_, err = GetDepth("Exchange", newCurrency, asset.Empty)
	assert.ErrorIs(t, err, ErrOrderbookNotFound)
}

func TestBaseGetDepth(t *testing.T) {
	t.Parallel()

	c, err := currency.NewPairFromStrings("BTC", "UST")
	require.NoError(t, err, "NewPairFromStrings must not error")

	base := &Base{
		Pair:     c,
		Asks:     []Tranche{{Price: 100, Amount: 10}},
		Bids:     []Tranche{{Price: 200, Amount: 10}},
		Exchange: "Exchange",
		Asset:    asset.Spot,
	}

	_, err = base.GetDepth()
	assert.ErrorIs(t, err, ErrOrderbookNotFound)

	require.NoError(t, base.Process(), "Process must not error")

	result, err := base.GetDepth()
	require.NoError(t, err, "GetDepth must not error")
	assert.True(t, result.pair.Equal(c))
}

func TestDeployDepth(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	require.NoError(t, err)
	_, err = DeployDepth("", c, asset.Spot)
	require.ErrorIs(t, err, errExchangeNameUnset)
	_, err = DeployDepth("test", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, errPairNotSet)
	_, err = DeployDepth("test", c, asset.Empty)
	require.ErrorIs(t, err, errAssetTypeNotSet)
	d, err := DeployDepth("test", c, asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, d)
	_, err = DeployDepth("test", c, asset.Spot)
	require.NoError(t, err)
}

func TestCreateNewOrderbook(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := &Base{
		Pair:     c,
		Asks:     []Tranche{{Price: 100, Amount: 10}},
		Bids:     []Tranche{{Price: 200, Amount: 10}},
		Exchange: "testCreateNewOrderbook",
		Asset:    asset.Spot,
	}

	err = base.Process()
	if err != nil {
		t.Fatal(err)
	}

	result, err := Get("testCreateNewOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestCreateNewOrderbook failed to create new orderbook", err)
	}

	if !result.Pair.Equal(c) {
		t.Fatal("TestCreateNewOrderbook result pair is incorrect")
	}

	a, b := result.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCreateNewOrderbook CalculateTotalAsks value is incorrect")
	}

	a, b = result.TotalBidsAmount()
	if a != 10 && b != 2000 {
		t.Fatal("TestCreateNewOrderbook CalculateTotalBids value is incorrect")
	}
}

func TestProcessOrderbook(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Asks:     []Tranche{{Price: 100, Amount: 10}},
		Bids:     []Tranche{{Price: 200, Amount: 10}},
		Exchange: "ProcessOrderbook",
	}

	// test for empty pair
	base.Pair = currency.EMPTYPAIR
	err = base.Process()
	if err == nil {
		t.Error("empty pair should throw an err")
	}

	// test for empty asset type
	base.Pair = c
	err = base.Process()
	if err == nil {
		t.Error("empty asset type should throw an err")
	}

	// now process a valid orderbook
	base.Asset = asset.Spot
	err = base.Process()
	if err != nil {
		t.Error("unexpected result: ", err)
	}
	result, err := Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to create new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair with a different quote currency
	c, err = currency.NewPairFromStrings("BTC", "GBP")
	if err != nil {
		t.Fatal(err)
	}
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair which has a different base currency
	c, err = currency.NewPairFromStrings("LTC", "GBP")
	if err != nil {
		t.Fatal(err)
	}
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	base.Asks = []Tranche{{Price: 200, Amount: 200}}
	base.Asset = asset.Spot
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}

	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}

	a, b := result.TotalAsksAmount()
	if a != 200 && b != 40000 {
		t.Fatal("TestProcessOrderbook CalculateTotalsAsks incorrect values")
	}

	base.Bids = []Tranche{{Price: 420, Amount: 200}}
	base.Exchange = "Blah"
	base.Asset = asset.CoinMarginedFutures
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}

	_, err = Get("Blah", c, asset.CoinMarginedFutures)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to create new orderbook")
	}

	if a != 200 && b != 84000 {
		t.Fatal("TestProcessOrderbook CalculateTotalsBids incorrect values")
	}

	type quick struct {
		Name string
		P    currency.Pair
		Bids []Tranche
		Asks []Tranche
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
		wg.Add(1)
		go func() {
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10) //nolint:gosec // no need to import crypo/rand for testing
			newPairs := currency.NewPair(currency.NewCode("BTC"+strconv.FormatInt(rand.Int63(), 10)),
				currency.NewCode("USD"+strconv.FormatInt(rand.Int63(), 10))) //nolint:gosec // no need to import crypo/rand for testing

			asks := []Tranche{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypo/rand for testing
			bids := []Tranche{{Price: rand.Float64(), Amount: rand.Float64()}} //nolint:gosec // no need to import crypo/rand for testing
			base := &Base{
				Pair:     newPairs,
				Asks:     asks,
				Bids:     bids,
				Exchange: newName,
				Asset:    asset.Spot,
			}

			m.Lock()
			err = base.Process()
			if err != nil {
				t.Error(err)
				catastrophicFailure = true
				m.Unlock()
				wg.Done()
				return
			}
			testArray = append(testArray, quick{Name: newName, P: newPairs, Bids: bids, Asks: asks})
			m.Unlock()
			wg.Done()
		}()
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

func deployUnorderedSlice() Tranches {
	ts := make([]Tranche, 1000)
	for x := range 1000 {
		ts[x] = Tranche{Amount: 1, Price: rand.Float64(), ID: rand.Int63()} //nolint:gosec // Not needed in tests
	}
	return ts
}

func TestSorting(t *testing.T) {
	var b Base
	b.VerifyOrderbook = true

	b.Asks = deployUnorderedSlice()
	err := b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("error expected %v received %v", errPriceOutOfOrder, err)
	}

	b.Asks.SortAsks()
	err = b.Verify()
	if err != nil {
		t.Fatal(err)
	}

	b.Bids = deployUnorderedSlice()
	err = b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("error expected %v received %v", errPriceOutOfOrder, err)
	}

	b.Bids.SortBids()
	err = b.Verify()
	if err != nil {
		t.Fatal(err)
	}
}

func deploySliceOrdered() Tranches {
	ts := make([]Tranche, 1000)
	for i := range 1000 {
		ts[i] = Tranche{Amount: 1, Price: float64(i + 1), ID: rand.Int63()} //nolint:gosec // Not needed in tests
	}
	return ts
}

func TestReverse(t *testing.T) {
	var b Base
	b.VerifyOrderbook = true

	if b.Bids = deploySliceOrdered(); len(b.Bids) != 1000 {
		t.Fatal("incorrect length")
	}

	err := b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("error expected %v received %v", errPriceOutOfOrder, err)
	}

	b.Bids.Reverse()
	err = b.Verify()
	if err != nil {
		t.Fatal(err)
	}

	b.Asks = append(b.Bids[:0:0], b.Bids...) //nolint:gocritic //  Short hand
	err = b.Verify()
	if !errors.Is(err, errPriceOutOfOrder) {
		t.Fatalf("error expected %v received %v", errPriceOutOfOrder, err)
	}

	b.Asks.Reverse()
	err = b.Verify()
	if err != nil {
		t.Fatal(err)
	}
}

// 705985	      1856 ns/op	       0 B/op	       0 allocs/op
func BenchmarkReverse(b *testing.B) {
	s := deploySliceOrdered()
	if len(s) != 1000 {
		b.Fatal("incorrect length")
	}

	for b.Loop() {
		s.Reverse()
	}
}

// 361266	      3556 ns/op	      24 B/op	       1 allocs/op (old)
// 385783	      3000 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksDecending(b *testing.B) {
	s := deploySliceOrdered()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortAsks()
	}
}

// 266998	      4292 ns/op	      40 B/op	       2 allocs/op (old)
// 372396	      3001 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsAscending(b *testing.B) {
	s := deploySliceOrdered()
	s.Reverse()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortBids()
	}
}

// 22119	     46532 ns/op	      35 B/op	       1 allocs/op (old)
// 16233	     76951 ns/op	     167 B/op	       3 allocs/op (new)
func BenchmarkSortAsksStandard(b *testing.B) {
	s := deployUnorderedSlice()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortAsks()
	}
}

// 19504	     62518 ns/op	      53 B/op	       2 allocs/op (old)
// 15698	     72859 ns/op	     168 B/op	       3 allocs/op (new)
func BenchmarkSortBidsStandard(b *testing.B) {
	s := deployUnorderedSlice()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortBids()
	}
}

// 376708	      3559 ns/op	      24 B/op 		   1 allocs/op (old)
// 377113	      3020 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksAscending(b *testing.B) {
	s := deploySliceOrdered()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortAsks()
	}
}

// 262874	      4364 ns/op	      40 B/op	       2 allocs/op (old)
// 401788	      3348 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsDescending(b *testing.B) {
	s := deploySliceOrdered()
	s.Reverse()
	bucket := make(Tranches, len(s))
	for b.Loop() {
		copy(bucket, s)
		bucket.SortBids()
	}
}

func TestCheckAlignment(t *testing.T) {
	t.Parallel()
	itemWithFunding := Tranches{{Amount: 1337, Price: 0, Period: 1337}}
	err := checkAlignment(itemWithFunding, true, true, false, false, dsc, "Bitfinex")
	if err != nil {
		t.Error(err)
	}
	err = checkAlignment(itemWithFunding, false, true, false, false, dsc, "Bitfinex")
	if !errors.Is(err, errPriceNotSet) {
		t.Fatalf("received: %v but expected: %v", err, errPriceNotSet)
	}
	err = checkAlignment(itemWithFunding, true, true, false, false, dsc, "Binance")
	if !errors.Is(err, errPriceNotSet) {
		t.Fatalf("received: %v but expected: %v", err, errPriceNotSet)
	}

	itemWithFunding[0].Price = 1337
	err = checkAlignment(itemWithFunding, true, true, false, true, dsc, "Binance")
	if !errors.Is(err, errChecksumStringNotSet) {
		t.Fatalf("received: %v but expected: %v", err, errChecksumStringNotSet)
	}

	itemWithFunding[0].StrAmount = "1337.0000000"
	itemWithFunding[0].StrPrice = "1337.0000000"
	err = checkAlignment(itemWithFunding, true, true, false, true, dsc, "Binance")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

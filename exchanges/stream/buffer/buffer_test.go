package buffer

import (
	"errors"
	"math/rand"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	itemArray = [][]orderbook.Tranche{
		{{Price: 1000, Amount: 1, ID: 1000}},
		{{Price: 2000, Amount: 1, ID: 2000}},
		{{Price: 3000, Amount: 1, ID: 3000}},
		{{Price: 3000, Amount: 2, ID: 4000}},
		{{Price: 4000, Amount: 0, ID: 6000}},
		{{Price: 5000, Amount: 1, ID: 5000}},
	}
	offset = common.Counter{}
)

const exchangeName = "exchangeTest"

// getExclusivePair returns a currency pair with a unique ID for testing as books are centralised and changes will affect other tests
func getExclusivePair() (currency.Pair, error) {
	return currency.NewPairFromStrings(currency.BTC.String(), currency.USDT.String()+strconv.FormatInt(offset.IncrementAndGet(), 10))
}

func createSnapshot(pair currency.Pair, bookVerifiy ...bool) (holder *Orderbook, asks, bids orderbook.Tranches, err error) {
	asks = orderbook.Tranches{{Price: 4000, Amount: 1, ID: 6}}
	bids = orderbook.Tranches{{Price: 4000, Amount: 1, ID: 6}}

	book := &orderbook.Base{
		Exchange:         exchangeName,
		Asks:             asks,
		Bids:             bids,
		Asset:            asset.Spot,
		Pair:             pair,
		PriceDuplication: true,
		LastUpdated:      time.Now(),
		VerifyOrderbook:  len(bookVerifiy) > 0 && bookVerifiy[0],
	}

	newBook := make(map[key.PairAsset]*orderbookHolder)

	ch := make(chan any)
	go func(<-chan any) { // reader
		for range ch {
			continue
		}
	}(ch)
	holder = &Orderbook{
		exchangeName: exchangeName,
		dataHandler:  ch,
		ob:           newBook,
	}
	err = holder.LoadSnapshot(book)
	return holder, asks, bids, err
}

func bidAskGenerator() []orderbook.Tranche {
	response := make([]orderbook.Tranche, 100)
	for i := range 100 {
		price := float64(rand.Intn(1000)) //nolint:gosec // no need to import crypo/rand for testing
		if price == 0 {
			price = 1
		}
		response[i] = orderbook.Tranche{
			Amount: float64(rand.Intn(10)), //nolint:gosec // no need to import crypo/rand for testing
			Price:  price,
			ID:     int64(i),
		}
	}
	return response
}

func BenchmarkUpdateBidsByPrice(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	ob, _, _, err := createSnapshot(cp)
	require.NoError(b, err)

	for b.Loop() {
		bidAsks := bidAskGenerator()
		update := &orderbook.Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
		require.NoError(b, holder.updateByPrice(update))
	}
}

func BenchmarkUpdateAsksByPrice(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	ob, _, _, err := createSnapshot(cp)
	require.NoError(b, err)

	for b.Loop() {
		bidAsks := bidAskGenerator()
		update := &orderbook.Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
		require.NoError(b, holder.updateByPrice(update))
	}
}

// BenchmarkBufferPerformance demonstrates buffer more performant than multi
// process calls
// 890016	      1688 ns/op	     416 B/op	       3 allocs/op
func BenchmarkBufferPerformance(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	holder, asks, bids, err := createSnapshot(cp)
	require.NoError(b, err)

	holder.bufferEnabled = true
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	for b.Loop() {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		require.NoError(b, holder.Update(update))
	}
}

// BenchmarkBufferSortingPerformance benchmark
//
//	613964	      2093 ns/op	     440 B/op	       4 allocs/op
func BenchmarkBufferSortingPerformance(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	holder, asks, bids, err := createSnapshot(cp)
	require.NoError(b, err)

	holder.bufferEnabled = true
	holder.sortBuffer = true
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	for b.Loop() {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		require.NoError(b, holder.Update(update))
	}
}

// BenchmarkBufferSortingPerformance benchmark
// 914500	      1599 ns/op	     440 B/op	       4 allocs/op
func BenchmarkBufferSortingByIDPerformance(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	holder, asks, bids, err := createSnapshot(cp)
	require.NoError(b, err)

	holder.bufferEnabled = true
	holder.sortBuffer = true
	holder.sortBufferByUpdateIDs = true
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}

	for b.Loop() {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		require.NoError(b, holder.Update(update))
	}
}

// BenchmarkNoBufferPerformance demonstrates orderbook process more performant
// than buffer
//   122659	     12792 ns/op	     972 B/op	       7 allocs/op PRIOR
//  1225924	      1028 ns/op	     240 B/op	       2 allocs/op CURRENT

func BenchmarkNoBufferPerformance(b *testing.B) {
	cp, err := getExclusivePair()
	require.NoError(b, err)

	obl, asks, bids, err := createSnapshot(cp)
	require.NoError(b, err)

	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}

	for b.Loop() {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		require.NoError(b, obl.Update(update))
	}
}

func TestUpdates(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	err = book.updateByPrice(&orderbook.Update{
		Bids:       itemArray[5],
		Asks:       itemArray[5],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	assert.NoError(t, err)

	err = book.updateByPrice(&orderbook.Update{
		Bids:       itemArray[0],
		Asks:       itemArray[0],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	assert.NoError(t, err)

	askLen, err := book.ob.GetAskLength()
	require.NoError(t, err)
	assert.Equal(t, 3, askLen)
}

// TestHittingTheBuffer logic test
func TestHittingTheBuffer(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	holder.bufferEnabled = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		})
		require.NoError(t, err)
	}

	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	require.NoError(t, err)
	assert.Equal(t, 3, askLen)

	bidLen, err := book.ob.GetBidLength()
	require.NoError(t, err)
	assert.Equal(t, 3, bidLen)
}

// TestInsertWithIDs logic test
func TestInsertWithIDs(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	holder.bufferEnabled = true
	holder.updateEntriesByID = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		if asks[0].Amount <= 0 {
			continue
		}
		bids := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
			Action:     orderbook.UpdateInsert,
		})
		require.NoError(t, err)
	}

	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	require.NoError(t, err)
	assert.Equal(t, 6, askLen)

	bidLen, err := book.ob.GetBidLength()
	require.NoError(t, err)
	assert.Equal(t, 6, bidLen)

	cp, err = getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err = createSnapshot(cp, true)
	require.NoError(t, err)

	holder.checksum = nil
	holder.updateIDProgression = false
	err = holder.Update(&orderbook.Update{
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
		Asks:       []orderbook.Tranche{{Price: 999999}},
		Pair:       cp,
	})
	require.NoError(t, err)
}

// TestSortIDs logic test
func TestSortIDs(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	holder.bufferEnabled = true
	holder.sortBufferByUpdateIDs = true
	holder.sortBuffer = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateID:   int64(i),
			Asset:      asset.Spot,
			UpdateTime: time.Now(),
		})
		require.NoError(t, err)
	}
	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	require.NoError(t, err)
	assert.Equal(t, 3, askLen)

	bidLen, err := book.ob.GetBidLength()
	require.NoError(t, err)
	assert.Equal(t, 3, bidLen)
}

// TestOutOfOrderIDs logic test
func TestOutOfOrderIDs(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	outOFOrderIDs := []int64{2, 1, 5, 3, 4, 6, 7}
	assert.Equal(t, 1000., itemArray[0][0].Price)

	holder.bufferEnabled = true
	holder.sortBuffer = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Asks:       asks,
			Pair:       cp,
			UpdateID:   outOFOrderIDs[i],
			Asset:      asset.Spot,
			UpdateTime: time.Now(),
		})
		require.NoError(t, err)
	}
	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	cpy, err := book.ob.Retrieve()
	require.NoError(t, err)
	// Index 1 since index 0 is price 7000
	assert.Equal(t, 2000., cpy.Asks[1].Price)
}

func TestOrderbookLastUpdateID(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	assert.Equal(t, 1000., itemArray[0][0].Price)

	holder.checksum = func(*orderbook.Base, uint32) error { return errors.New("testerino") }

	// this update invalidates the book
	err = holder.Update(&orderbook.Update{
		Asks:       []orderbook.Tranche{{Price: 999999}},
		Pair:       cp,
		UpdateID:   -1,
		Asset:      asset.Spot,
		UpdateTime: time.Now(),
	})
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)

	cp, err = getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err = createSnapshot(cp)
	require.NoError(t, err)

	holder.checksum = func(*orderbook.Base, uint32) error { return nil }
	holder.updateIDProgression = true

	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Asks:       asks,
			Pair:       cp,
			UpdateID:   int64(i) + 1,
			Asset:      asset.Spot,
			UpdateTime: time.Now(),
		})
		require.NoError(t, err)
	}

	// out of order
	err = holder.Update(&orderbook.Update{
		Asks:     []orderbook.Tranche{{Price: 999999}},
		Pair:     cp,
		UpdateID: 1,
		Asset:    asset.Spot,
	})
	require.NoError(t, err)

	ob, err := holder.GetOrderbook(cp, asset.Spot)
	require.NoError(t, err)
	assert.Equal(t, int64(len(itemArray)), ob.LastUpdateID)
}

// TestRunUpdateWithoutSnapshot logic test
func TestRunUpdateWithoutSnapshot(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var holder Orderbook
	asks := []orderbook.Tranche{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Tranche{{Price: 5999, Amount: 1, ID: 8}, {Price: 4000, Amount: 1, ID: 9}}
	holder.exchangeName = exchangeName
	err = holder.Update(&orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	require.ErrorIs(t, err, errDepthNotFound)
}

// TestRunUpdateWithoutAnyUpdates logic test
func TestRunUpdateWithoutAnyUpdates(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var obl Orderbook
	obl.exchangeName = exchangeName
	err = obl.Update(&orderbook.Update{
		Bids:       []orderbook.Tranche{},
		Asks:       []orderbook.Tranche{},
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	require.ErrorIs(t, err, errUpdateNoTargets)
}

// TestRunSnapshotWithNoData logic test
func TestRunSnapshotWithNoData(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var obl Orderbook
	obl.ob = make(map[key.PairAsset]*orderbookHolder)
	obl.dataHandler = make(chan any, 1)
	var snapShot1 orderbook.Base
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.Exchange = "test"
	obl.exchangeName = "test"
	snapShot1.LastUpdated = time.Now()
	require.NoError(t, obl.LoadSnapshot(&snapShot1))
}

// TestLoadSnapshot logic test
func TestLoadSnapshot(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var obl Orderbook
	obl.dataHandler = make(chan any, 100)
	obl.ob = make(map[key.PairAsset]*orderbookHolder)
	var snapShot1 orderbook.Base
	snapShot1.Exchange = "SnapshotWithOverride"
	asks := []orderbook.Tranche{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Tranche{{Price: 4000, Amount: 1, ID: 9}}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.LastUpdated = time.Now()
	require.NoError(t, obl.LoadSnapshot(&snapShot1))
}

// TestFlushBuffer logic test
func TestFlushBuffer(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	obl, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	assert.NotEmpty(t, obl.ob)
	obl.FlushBuffer()
	assert.Empty(t, obl.ob)
}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var holder Orderbook
	holder.dataHandler = make(chan any, 100)
	holder.ob = make(map[key.PairAsset]*orderbookHolder)
	var snapShot1 orderbook.Base
	snapShot1.Exchange = "WSORDERBOOKTEST1"
	asks := []orderbook.Tranche{
		{Price: 6000, Amount: 1, ID: 1},
		{Price: 6001, Amount: 0.5, ID: 2},
		{Price: 6002, Amount: 2, ID: 3},
		{Price: 6003, Amount: 3, ID: 4},
		{Price: 6004, Amount: 5, ID: 5},
		{Price: 6005, Amount: 2, ID: 6},
		{Price: 6006, Amount: 1.5, ID: 7},
		{Price: 6007, Amount: 0.5, ID: 8},
		{Price: 6008, Amount: 23, ID: 9},
		{Price: 6009, Amount: 9, ID: 10},
		{Price: 6010, Amount: 7, ID: 11},
	}

	bids := []orderbook.Tranche{
		{Price: 5999, Amount: 1, ID: 12},
		{Price: 5998, Amount: 0.5, ID: 13},
		{Price: 5997, Amount: 2, ID: 14},
		{Price: 5996, Amount: 3, ID: 15},
		{Price: 5995, Amount: 5, ID: 16},
		{Price: 5994, Amount: 2, ID: 17},
		{Price: 5993, Amount: 1.5, ID: 18},
		{Price: 5992, Amount: 0.5, ID: 19},
		{Price: 5991, Amount: 23, ID: 20},
		{Price: 5990, Amount: 9, ID: 21},
		{Price: 5989, Amount: 7, ID: 22},
	}

	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.LastUpdated = time.Now()
	require.NoError(t, holder.LoadSnapshot(&snapShot1))

	var snapShot2 orderbook.Base
	snapShot2.Exchange = "WSORDERBOOKTEST2"
	asks = []orderbook.Tranche{
		{Price: 51, Amount: 1, ID: 1},
		{Price: 52, Amount: 0.5, ID: 2},
		{Price: 53, Amount: 2, ID: 3},
		{Price: 54, Amount: 3, ID: 4},
		{Price: 55, Amount: 5, ID: 5},
		{Price: 56, Amount: 2, ID: 6},
		{Price: 57, Amount: 1.5, ID: 7},
		{Price: 58, Amount: 0.5, ID: 8},
		{Price: 59, Amount: 23, ID: 9},
		{Price: 50, Amount: 9, ID: 10},
		{Price: 60, Amount: 7, ID: 11},
	}

	bids = []orderbook.Tranche{
		{Price: 49, Amount: 1, ID: 12},
		{Price: 48, Amount: 0.5, ID: 13},
		{Price: 47, Amount: 2, ID: 14},
		{Price: 46, Amount: 3, ID: 15},
		{Price: 45, Amount: 5, ID: 16},
		{Price: 44, Amount: 2, ID: 17},
		{Price: 43, Amount: 1.5, ID: 18},
		{Price: 42, Amount: 0.5, ID: 19},
		{Price: 41, Amount: 23, ID: 20},
		{Price: 40, Amount: 9, ID: 21},
		{Price: 39, Amount: 7, ID: 22},
	}

	snapShot2.Asks = asks
	snapShot2.Asks.SortAsks()
	snapShot2.Bids = bids
	snapShot2.Bids.SortBids()
	snapShot2.Asset = asset.Spot
	snapShot2.Pair, err = getExclusivePair()
	require.NoError(t, err)

	snapShot2.LastUpdated = time.Now()
	require.NoError(t, holder.LoadSnapshot(&snapShot2))

	var snapShot3 orderbook.Base
	snapShot3.Exchange = "WSORDERBOOKTEST3"
	asks = []orderbook.Tranche{
		{Price: 511, Amount: 1, ID: 1},
		{Price: 52, Amount: 0.5, ID: 2},
		{Price: 53, Amount: 2, ID: 3},
		{Price: 54, Amount: 3, ID: 4},
		{Price: 55, Amount: 5, ID: 5},
		{Price: 56, Amount: 2, ID: 6},
		{Price: 57, Amount: 1.5, ID: 7},
		{Price: 58, Amount: 0.5, ID: 8},
		{Price: 59, Amount: 23, ID: 9},
		{Price: 50, Amount: 9, ID: 10},
		{Price: 60, Amount: 7, ID: 11},
	}

	bids = []orderbook.Tranche{
		{Price: 49, Amount: 1, ID: 12},
		{Price: 48, Amount: 0.5, ID: 13},
		{Price: 47, Amount: 2, ID: 14},
		{Price: 46, Amount: 3, ID: 15},
		{Price: 45, Amount: 5, ID: 16},
		{Price: 44, Amount: 2, ID: 17},
		{Price: 43, Amount: 1.5, ID: 18},
		{Price: 42, Amount: 0.5, ID: 19},
		{Price: 41, Amount: 23, ID: 20},
		{Price: 40, Amount: 9, ID: 21},
		{Price: 39, Amount: 7, ID: 22},
	}

	snapShot3.Asks = asks
	snapShot3.Asks.SortAsks()
	snapShot3.Bids = bids
	snapShot3.Bids.SortBids()
	snapShot3.Asset = asset.Futures
	snapShot3.Pair, err = getExclusivePair()
	require.NoError(t, err)

	snapShot3.LastUpdated = time.Now()
	require.NoError(t, holder.LoadSnapshot(&snapShot3))

	ob, err := holder.GetOrderbook(snapShot1.Pair, snapShot1.Asset)
	require.NoError(t, err)
	assert.Equal(t, snapShot1.Asks[0], ob.Asks[0])

	ob, err = holder.GetOrderbook(snapShot2.Pair, snapShot2.Asset)
	require.NoError(t, err)
	assert.Equal(t, snapShot2.Asks[0], ob.Asks[0])

	ob, err = holder.GetOrderbook(snapShot3.Pair, snapShot3.Asset)
	require.NoError(t, err)
	assert.Equal(t, snapShot3.Asks[0], ob.Asks[0])
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	ob, err := holder.GetOrderbook(cp, asset.Spot)
	require.NoError(t, err)

	bufferOb := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	b, err := bufferOb.ob.Retrieve()
	require.NoError(t, err)

	askLen, err := bufferOb.ob.GetAskLength()
	require.NoError(t, err)

	bidLen, err := bufferOb.ob.GetBidLength()
	require.NoError(t, err)

	if askLen != len(ob.Asks) ||
		bidLen != len(ob.Bids) ||
		b.Asset != ob.Asset ||
		b.Exchange != ob.Exchange ||
		b.LastUpdateID != ob.LastUpdateID ||
		b.PriceDuplication != ob.PriceDuplication ||
		b.Pair != ob.Pair {
		t.Fatal("data on both books should be the same")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	w := Orderbook{}
	err := w.Setup(nil, nil, nil)
	require.ErrorIs(t, err, errExchangeConfigNil)

	exchangeConfig := &config.Exchange{}
	err = w.Setup(exchangeConfig, nil, nil)
	require.ErrorIs(t, err, errBufferConfigNil)

	bufferConf := &Config{}
	err = w.Setup(exchangeConfig, bufferConf, nil)
	require.ErrorIs(t, err, errUnsetDataHandler)

	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	err = w.Setup(exchangeConfig, bufferConf, make(chan any))
	require.ErrorIs(t, err, errIssueBufferEnabledButNoLimit)

	exchangeConfig.Orderbook.WebsocketBufferLimit = 1337
	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	exchangeConfig.Name = "test"
	bufferConf.SortBuffer = true
	bufferConf.SortBufferByUpdateIDs = true
	bufferConf.UpdateEntriesByID = true
	err = w.Setup(exchangeConfig, bufferConf, make(chan any))
	require.NoError(t, err)

	if w.obBufferLimit != 1337 ||
		!w.bufferEnabled ||
		!w.sortBuffer ||
		!w.sortBufferByUpdateIDs ||
		!w.updateEntriesByID ||
		w.exchangeName != "test" {
		t.Errorf("Setup incorrectly loaded %s", w.exchangeName)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	w := Orderbook{}
	err := w.validate(nil)
	require.ErrorIs(t, err, errUpdateIsNil)
	err = w.validate(&orderbook.Update{})
	require.ErrorIs(t, err, errUpdateNoTargets)
}

func TestEnsureMultipleUpdatesViaPrice(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	asks := bidAskGenerator()
	book := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	err = book.updateByPrice(&orderbook.Update{
		Bids:       asks,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	require.NoError(t, err)

	askLen, err := book.ob.GetAskLength()
	require.NoError(t, err)
	assert.LessOrEqual(t, 3, askLen)
}

func deploySliceOrdered(size int) orderbook.Tranches {
	items := make([]orderbook.Tranche, size)
	for i := range size {
		items[i] = orderbook.Tranche{Amount: 1, Price: rand.Float64() + float64(i), ID: rand.Int63()} //nolint:gosec // Not needed for tests
	}
	return items
}

func TestUpdateByIDAndAction(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	asks := deploySliceOrdered(100)
	bids := slices.Clone(asks)
	bids.Reverse()

	book, err := orderbook.DeployDepth("test", cp, asset.Spot)
	require.NoError(t, err)

	err = book.LoadSnapshot(slices.Clone(bids), slices.Clone(asks), 0, time.Now(), time.Now(), true)
	require.NoError(t, err)

	ob, err := book.Retrieve()
	require.NoError(t, err)

	require.NoError(t, ob.Verify())

	holder := orderbookHolder{ob: book}
	err = holder.updateByIDAndAction(&orderbook.Update{})
	require.ErrorIs(t, err, errInvalidAction)

	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Amend,
		Bids:   []orderbook.Tranche{{Price: 100, ID: 6969}},
	})
	require.ErrorIs(t, err, errAmendFailure)

	err = book.LoadSnapshot(slices.Clone(bids), slices.Clone(asks), 0, time.Now(), time.Now(), true)
	require.NoError(t, err)

	// append to slice
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action:     orderbook.UpdateInsert,
		Bids:       []orderbook.Tranche{{Price: 0, ID: 1337, Amount: 1}},
		Asks:       []orderbook.Tranche{{Price: 100, ID: 1337, Amount: 1}},
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cpy, err := book.Retrieve()
	require.NoError(t, err)
	require.Equal(t, 0., cpy.Bids[len(cpy.Bids)-1].Price)
	require.Equal(t, 100., cpy.Asks[len(cpy.Asks)-1].Price)

	// Change amount
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action:     orderbook.UpdateInsert,
		Bids:       []orderbook.Tranche{{Price: 0, ID: 1337, Amount: 100}},
		Asks:       []orderbook.Tranche{{Price: 100, ID: 1337, Amount: 100}},
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cpy, err = book.Retrieve()
	require.NoError(t, err)
	require.Equal(t, 100., cpy.Bids[len(cpy.Bids)-1].Amount)
	require.Equal(t, 100., cpy.Asks[len(cpy.Asks)-1].Amount)

	// Change price level
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action:     orderbook.UpdateInsert,
		Bids:       []orderbook.Tranche{{Price: 100, ID: 1337, Amount: 99}},
		Asks:       []orderbook.Tranche{{Price: 0, ID: 1337, Amount: 99}},
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cpy, err = book.Retrieve()
	require.NoError(t, err)

	require.Equal(t, 99., cpy.Bids[0].Amount)
	require.Equal(t, 100., cpy.Bids[0].Price)
	require.Equal(t, 99., cpy.Asks[0].Amount)
	require.Equal(t, 0., cpy.Asks[0].Price)

	err = book.LoadSnapshot(slices.Clone(bids), slices.Clone(asks), 0, time.Now(), time.Now(), true)
	require.NoError(t, err)
	// Delete - not found
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Delete,
		Asks:   []orderbook.Tranche{{Price: 0, ID: 1337, Amount: 99}},
	})
	require.ErrorIs(t, err, errDeleteFailure)

	err = book.LoadSnapshot(slices.Clone(bids), slices.Clone(asks), 0, time.Now(), time.Now(), true)
	require.NoError(t, err)

	// Delete - found
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action:     orderbook.Delete,
		Asks:       []orderbook.Tranche{asks[0]},
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	askLen, err := book.GetAskLength()
	require.NoError(t, err)
	require.Equal(t, 99, askLen)

	// Apply update
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Amend,
		Asks:   []orderbook.Tranche{{ID: 123456}},
	})
	require.ErrorIs(t, err, errAmendFailure)

	err = book.LoadSnapshot(slices.Clone(bids), slices.Clone(bids), 0, time.Now(), time.Now(), true)
	require.NoError(t, err)

	ob, err = book.Retrieve()
	require.NoError(t, err)
	require.NotEmpty(t, ob.Asks)
	require.NotEmpty(t, ob.Bids)

	update := ob.Asks[0]
	update.Amount = 1337

	err = holder.updateByIDAndAction(&orderbook.Update{
		Action:     orderbook.Amend,
		Asks:       []orderbook.Tranche{update},
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	ob, err = book.Retrieve()
	require.NoError(t, err)
	require.Equal(t, 1337., ob.Asks[0].Amount)
}

func TestFlushOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	w := &Orderbook{}
	err = w.Setup(&config.Exchange{Name: "test"}, &Config{}, make(chan any, 2))
	require.NoError(t, err)

	var snapShot1 orderbook.Base
	snapShot1.Exchange = "Snapshooooot"
	asks := []orderbook.Tranche{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Tranche{{Price: 4000, Amount: 1, ID: 9}}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.LastUpdated = time.Now()

	err = w.FlushOrderbook(cp, asset.Spot)
	if err == nil {
		t.Fatal("book not loaded error cannot be nil")
	}

	_, err = w.GetOrderbook(cp, asset.Spot)
	require.ErrorIs(t, err, errDepthNotFound)

	require.NoError(t, w.LoadSnapshot(&snapShot1))
	require.NoError(t, w.FlushOrderbook(cp, asset.Spot))

	_, err = w.GetOrderbook(cp, asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)
}

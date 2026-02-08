package buffer

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	itemArray = [][]orderbook.Level{
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

func createSnapshot(pair currency.Pair) (holder *Orderbook, asks, bids orderbook.Levels, err error) {
	asks = orderbook.Levels{{Price: 4000, Amount: 1, ID: 6}}
	bids = orderbook.Levels{{Price: 4000, Amount: 1, ID: 6}}

	book := &orderbook.Book{
		Exchange:          exchangeName,
		Asks:              asks,
		Bids:              bids,
		Asset:             asset.Spot,
		Pair:              pair,
		PriceDuplication:  true,
		LastUpdated:       time.Now(),
		LastUpdateID:      69420,
		ValidateOrderbook: true,
	}

	newBook := make(map[key.PairAsset]*orderbookHolder)

	relay := stream.NewRelay(10)
	go func(relay *stream.Relay) { // reader
		for range relay.C {
			continue
		}
	}(relay)
	holder = &Orderbook{
		exchangeName: exchangeName,
		dataHandler:  relay,
		ob:           newBook,
	}
	err = holder.LoadSnapshot(book)
	return holder, asks, bids, err
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
			Action:     orderbook.UpdateOrInsertAction,
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

	holder.obBufferLimit = 1
	err = holder.Update(&orderbook.Update{
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
		Pair:       cp,
	})
	assert.ErrorIs(t, err, orderbook.ErrEmptyUpdate)
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

	// this update invalidates the book
	err = holder.Update(&orderbook.Update{
		Asks:             orderbook.Levels{{Price: 999999}},
		Pair:             cp,
		UpdateID:         -1,
		Asset:            asset.Spot,
		UpdateTime:       time.Now(),
		ExpectedChecksum: 1337,
		GenerateChecksum: func(*orderbook.Book) uint32 { return 1336 },
	})
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)

	cp, err = getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err = createSnapshot(cp)
	require.NoError(t, err)

	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Asks:                       asks,
			Pair:                       cp,
			UpdateID:                   int64(i) + 1 + 69420,
			Asset:                      asset.Spot,
			UpdateTime:                 time.Now(),
			SkipOutOfOrderLastUpdateID: true,
			ExpectedChecksum:           1337,
			GenerateChecksum:           func(*orderbook.Book) uint32 { return 1337 },
		})
		require.NoError(t, err)
	}

	// out of order
	err = holder.Update(&orderbook.Update{
		Asks:                       orderbook.Levels{{Price: 999999}},
		Pair:                       cp,
		UpdateID:                   1,
		Asset:                      asset.Spot,
		SkipOutOfOrderLastUpdateID: true,
	})
	require.NoError(t, err, "Out of sequence Update must not error")

	ob, err := holder.GetOrderbook(cp, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, int64(len(itemArray)+69420), ob.LastUpdateID, "Out of sequence Update should not change LastUpdateID")
}

// TestRunUpdateWithoutSnapshot logic test
func TestRunUpdateWithoutSnapshot(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var holder Orderbook
	asks := []orderbook.Level{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Level{{Price: 5999, Amount: 1, ID: 8}, {Price: 4000, Amount: 1, ID: 9}}
	holder.exchangeName = exchangeName
	err = holder.Update(&orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)
}

// TestRunUpdateWithoutAnyUpdates logic test
func TestRunUpdateWithoutAnyUpdates(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	holder.exchangeName = exchangeName
	err = holder.Update(&orderbook.Update{
		Bids:       orderbook.Levels{},
		Asks:       orderbook.Levels{},
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	require.ErrorIs(t, err, orderbook.ErrEmptyUpdate)
}

// TestRunSnapshotWithNoData logic test
func TestRunSnapshotWithNoData(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var obl Orderbook
	obl.ob = make(map[key.PairAsset]*orderbookHolder)
	obl.dataHandler = stream.NewRelay(1)
	var snapShot1 orderbook.Book
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
	obl.dataHandler = stream.NewRelay(100)
	obl.ob = make(map[key.PairAsset]*orderbookHolder)

	err = obl.LoadSnapshot(&orderbook.Book{Asks: orderbook.Levels{{Amount: 1}}, ValidateOrderbook: true})
	require.ErrorIs(t, err, orderbook.ErrPriceZero)

	err = obl.LoadSnapshot(&orderbook.Book{Asks: orderbook.Levels{{Amount: 1}}})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	err = obl.LoadSnapshot(&orderbook.Book{Asks: orderbook.Levels{{Amount: 1}}, Exchange: "test", Pair: cp, Asset: asset.Spot})
	require.ErrorIs(t, err, orderbook.ErrLastUpdatedNotSet)

	var snapShot1 orderbook.Book
	snapShot1.Exchange = "SnapshotWithOverride"
	asks := []orderbook.Level{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Level{{Price: 4000, Amount: 1, ID: 9}}
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
	require.NoError(t, err, "createSnapshot must not error")
	require.NotEmpty(t, obl.ob, "createSnapshot must not return empty")

	k := key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}
	holder, ok := obl.ob[k]
	require.Truef(t, ok, "createSnapshot must return a orderbook for %v", k)

	holder.buffer = make([]orderbook.Update, 0, 10)
	holder.buffer = append(holder.buffer, orderbook.Update{})

	obl.FlushBuffer()
	assert.Empty(t, holder.buffer, "FlushBuffer should empty buffer")
	assert.Equal(t, 10, cap(holder.buffer), "FlushBuffer should leave the buffer cap to avoid reallocs")
}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	var holder Orderbook
	holder.dataHandler = stream.NewRelay(100)
	holder.ob = make(map[key.PairAsset]*orderbookHolder)
	var snapShot1 orderbook.Book
	snapShot1.Exchange = "WSORDERBOOKTEST1"
	asks := []orderbook.Level{
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

	bids := []orderbook.Level{
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

	var snapShot2 orderbook.Book
	snapShot2.Exchange = "WSORDERBOOKTEST2"
	asks = []orderbook.Level{
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

	bids = []orderbook.Level{
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

	var snapShot3 orderbook.Book
	snapShot3.Exchange = "WSORDERBOOKTEST3"
	asks = []orderbook.Level{
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

	bids = []orderbook.Level{
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

	_, err = holder.GetOrderbook(currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = holder.GetOrderbook(cp, 0)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	ob, err := holder.GetOrderbook(cp, asset.Spot)
	require.NoError(t, err)

	bufferOb := holder.ob[key.PairAsset{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	b, err := bufferOb.ob.Retrieve()
	require.NoError(t, err)

	askLen, err := bufferOb.ob.GetAskLength()
	require.NoError(t, err)

	bidLen, err := bufferOb.ob.GetBidLength()
	require.NoError(t, err)

	assert.Equal(t, askLen, len(ob.Asks), "ask length mismatch")
	assert.Equal(t, bidLen, len(ob.Bids), "bid length mismatch")
	assert.Equal(t, b.Asset, ob.Asset, "asset mismatch")
	assert.Equal(t, b.Exchange, ob.Exchange, "exchange name mismatch")
	assert.Equal(t, b.LastUpdateID, ob.LastUpdateID, "last update ID mismatch")
	assert.Equal(t, b.PriceDuplication, ob.PriceDuplication, "price duplication mismatch")
	assert.Equal(t, b.Pair, ob.Pair, "pair mismatch")
}

func TestLastUpdateID(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	holder, _, _, err := createSnapshot(cp)
	require.NoError(t, err)

	_, err = holder.LastUpdateID(currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = holder.LastUpdateID(cp, 0)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	_, err = holder.LastUpdateID(cp, asset.FutureCombo)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	ob, err := holder.LastUpdateID(cp, asset.Spot)
	require.NoError(t, err)
	require.Equal(t, int64(69420), ob)
}

func TestSetup(t *testing.T) {
	t.Parallel()
	w := Orderbook{}
	err := w.Setup(nil, nil, nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	exchangeConfig := &config.Exchange{}
	err = w.Setup(exchangeConfig, nil, nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	bufferConf := &Config{}
	err = w.Setup(exchangeConfig, bufferConf, nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	err = w.Setup(exchangeConfig, bufferConf, stream.NewRelay(1))
	require.ErrorIs(t, err, errIssueBufferEnabledButNoLimit)

	exchangeConfig.Orderbook.WebsocketBufferLimit = 1337
	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	exchangeConfig.Name = "test"
	bufferConf.SortBuffer = true
	bufferConf.SortBufferByUpdateIDs = true
	err = w.Setup(exchangeConfig, bufferConf, stream.NewRelay(1))
	require.NoError(t, err)

	require.Equal(t, 1337, w.obBufferLimit)
	require.True(t, w.bufferEnabled)
	require.True(t, w.sortBuffer)
	require.True(t, w.sortBufferByUpdateIDs)
	require.Equal(t, "test", w.exchangeName)
}

func TestInvalidateOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := getExclusivePair()
	require.NoError(t, err)

	w := &Orderbook{}
	err = w.Setup(&config.Exchange{Name: "test"}, &Config{}, stream.NewRelay(2))
	require.NoError(t, err)

	var snapShot1 orderbook.Book
	snapShot1.Exchange = "Snapshooooot"
	asks := []orderbook.Level{{Price: 4000, Amount: 1, ID: 8}}
	bids := []orderbook.Level{{Price: 4000, Amount: 1, ID: 9}}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.LastUpdated = time.Now()

	err = w.InvalidateOrderbook(cp, asset.Spot)
	if err == nil {
		t.Fatal("book not loaded error cannot be nil")
	}

	_, err = w.GetOrderbook(cp, asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	require.NoError(t, w.LoadSnapshot(&snapShot1))
	require.NoError(t, w.InvalidateOrderbook(cp, asset.Spot))

	_, err = w.GetOrderbook(cp, asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)
}

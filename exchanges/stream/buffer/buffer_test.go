package buffer

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var (
	itemArray = [][]orderbook.Item{
		{{Price: 1000, Amount: 1, ID: 1000}},
		{{Price: 2000, Amount: 1, ID: 2000}},
		{{Price: 3000, Amount: 1, ID: 3000}},
		{{Price: 3000, Amount: 2, ID: 4000}},
		{{Price: 4000, Amount: 0, ID: 6000}},
		{{Price: 5000, Amount: 1, ID: 5000}},
	}

	cp, _ = currency.NewPairFromString("BTCUSD")
)

const (
	exchangeName = "exchangeTest"
)

func createSnapshot() (holder *Orderbook, asks, bids orderbook.Items, err error) {
	asks = orderbook.Items{{Price: 4000, Amount: 1, ID: 6}}
	bids = orderbook.Items{{Price: 4000, Amount: 1, ID: 6}}

	book := &orderbook.Base{
		Exchange:         exchangeName,
		Asks:             asks,
		Bids:             bids,
		Asset:            asset.Spot,
		Pair:             cp,
		PriceDuplication: true,
	}

	newBook := make(map[Key]*orderbookHolder)

	ch := make(chan interface{})
	go func(<-chan interface{}) { // reader
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

func bidAskGenerator() []orderbook.Item {
	var response []orderbook.Item
	randIterator := 100
	for i := 0; i < randIterator; i++ {
		price := float64(rand.Intn(1000)) //nolint:gosec // no need to import crypo/rand for testing
		if price == 0 {
			price = 1
		}
		response = append(response, orderbook.Item{
			Amount: float64(rand.Intn(10)), //nolint:gosec // no need to import crypo/rand for testing
			Price:  price,
			ID:     int64(i),
		})
	}
	return response
}

func BenchmarkUpdateBidsByPrice(b *testing.B) {
	ob, _, _, err := createSnapshot()
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bidAsks := bidAskGenerator()
		update := &orderbook.Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
		holder.updateByPrice(update)
	}
}

func BenchmarkUpdateAsksByPrice(b *testing.B) {
	ob, _, _, err := createSnapshot()
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bidAsks := bidAskGenerator()
		update := &orderbook.Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
		holder.updateByPrice(update)
	}
}

// BenchmarkBufferPerformance demonstrates buffer more performant than multi
// process calls
// 890016	      1688 ns/op	     416 B/op	       3 allocs/op
func BenchmarkBufferPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	holder.bufferEnabled = true
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = holder.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferSortingPerformance benchmark
//
//	613964	      2093 ns/op	     440 B/op	       4 allocs/op
func BenchmarkBufferSortingPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	holder.bufferEnabled = true
	holder.sortBuffer = true
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = holder.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferSortingPerformance benchmark
// 914500	      1599 ns/op	     440 B/op	       4 allocs/op
func BenchmarkBufferSortingByIDPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = holder.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNoBufferPerformance demonstrates orderbook process more performant
// than buffer
//   122659	     12792 ns/op	     972 B/op	       7 allocs/op PRIOR
//  1225924	      1028 ns/op	     240 B/op	       2 allocs/op CURRENT

func BenchmarkNoBufferPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	update := &orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) //nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = obl.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestUpdates(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Error(err)
	}

	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	book.updateByPrice(&orderbook.Update{
		Bids:       itemArray[5],
		Asks:       itemArray[5],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	book.updateByPrice(&orderbook.Update{
		Bids:       itemArray[0],
		Asks:       itemArray[0],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	askLen, err := book.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 3 {
		t.Error("Did not update")
	}
}

// TestHittingTheBuffer logic test
func TestHittingTheBuffer(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
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
		if err != nil {
			t.Fatal(err)
		}
	}

	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 3 {
		t.Errorf("expected 3 entries, received: %v", askLen)
	}

	bidLen, err := book.ob.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if bidLen != 3 {
		t.Errorf("expected 3 entries, received: %v", bidLen)
	}
}

// TestInsertWithIDs logic test
func TestInsertWithIDs(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
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
		if err != nil {
			t.Fatal(err)
		}
	}

	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if askLen != 6 {
		t.Errorf("expected 6 entries, received: %v", askLen)
	}

	bidLen, err := book.ob.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if bidLen != 6 {
		t.Errorf("expected 6 entries, received: %v", bidLen)
	}
}

// TestSortIDs logic test
func TestSortIDs(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	holder.bufferEnabled = true
	holder.sortBufferByUpdateIDs = true
	holder.sortBuffer = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Bids:     bids,
			Asks:     asks,
			Pair:     cp,
			UpdateID: int64(i),
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	askLen, err := book.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if askLen != 3 {
		t.Errorf("expected 3 entries, received: %v", askLen)
	}

	bidLen, err := book.ob.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if bidLen != 3 {
		t.Errorf("expected 3 entries, received: %v", bidLen)
	}
}

// TestOutOfOrderIDs logic test
func TestOutOfOrderIDs(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	outOFOrderIDs := []int64{2, 1, 5, 3, 4, 6, 7}
	if itemArray[0][0].Price != 1000 {
		t.Errorf("expected sorted price to be 3000, received: %v",
			itemArray[1][0].Price)
	}
	holder.bufferEnabled = true
	holder.sortBuffer = true
	holder.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: outOFOrderIDs[i],
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	cpy, err := book.ob.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	// Index 1 since index 0 is price 7000
	if cpy.Asks[1].Price != 2000 {
		t.Errorf("expected sorted price to be 2000, received: %v", cpy.Asks[1].Price)
	}
}

func TestOrderbookLastUpdateID(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if exp := float64(1000); itemArray[0][0].Price != exp {
		t.Errorf("expected sorted price to be %f, received: %v",
			exp, itemArray[1][0].Price)
	}

	holder.checksum = func(state *orderbook.Base, checksum uint32) error { return errors.New("testerino") }

	// this update invalidates the book
	err = holder.Update(&orderbook.Update{
		Asks:     []orderbook.Item{{Price: 999999}},
		Pair:     cp,
		UpdateID: -1,
		Asset:    asset.Spot,
	})
	if !errors.Is(err, orderbook.ErrOrderbookInvalid) {
		t.Fatalf("received: %v but expected: %v", err, orderbook.ErrOrderbookInvalid)
	}

	holder, _, _, err = createSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	holder.checksum = func(state *orderbook.Base, checksum uint32) error { return nil }
	holder.updateIDProgression = true

	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&orderbook.Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: int64(i) + 1,
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// out of order
	err = holder.Update(&orderbook.Update{
		Asks:     []orderbook.Item{{Price: 999999}},
		Pair:     cp,
		UpdateID: 1,
		Asset:    asset.Spot,
	})
	if err != nil {
		t.Fatal(err)
	}

	ob, err := holder.GetOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if exp := len(itemArray); ob.LastUpdateID != int64(exp) {
		t.Errorf("expected last update id to be %d, received: %v", exp, ob.LastUpdateID)
	}
}

// TestRunUpdateWithoutSnapshot logic test
func TestRunUpdateWithoutSnapshot(t *testing.T) {
	t.Parallel()
	var holder Orderbook
	var snapShot1 orderbook.Base
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 5999, Amount: 1, ID: 8},
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	holder.exchangeName = exchangeName
	err := holder.Update(&orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if !errors.Is(err, errDepthNotFound) {
		t.Fatalf("expected %v but received %v", errDepthNotFound, err)
	}
}

// TestRunUpdateWithoutAnyUpdates logic test
func TestRunUpdateWithoutAnyUpdates(t *testing.T) {
	t.Parallel()
	var obl Orderbook
	var snapShot1 orderbook.Base
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	obl.exchangeName = exchangeName
	err := obl.Update(&orderbook.Update{
		Bids:       snapShot1.Asks,
		Asks:       snapShot1.Bids,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if !errors.Is(err, errUpdateNoTargets) {
		t.Fatalf("expected %v but received %v", errUpdateNoTargets, err)
	}
}

// TestRunSnapshotWithNoData logic test
func TestRunSnapshotWithNoData(t *testing.T) {
	t.Parallel()
	var obl Orderbook
	obl.ob = make(map[Key]*orderbookHolder)
	obl.dataHandler = make(chan interface{}, 1)
	var snapShot1 orderbook.Base
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	snapShot1.Exchange = "test"
	obl.exchangeName = "test"
	err := obl.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestLoadSnapshot logic test
func TestLoadSnapshot(t *testing.T) {
	t.Parallel()
	var obl Orderbook
	obl.dataHandler = make(chan interface{}, 100)
	obl.ob = make(map[Key]*orderbookHolder)
	var snapShot1 orderbook.Base
	snapShot1.Exchange = "SnapshotWithOverride"
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp
	err := obl.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Error(err)
	}
}

// TestFlushBuffer logic test
func TestFlushBuffer(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if obl.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}] == nil {
		t.Error("expected ob to have ask entries")
	}
	obl.FlushBuffer()
	if obl.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}] != nil {
		t.Error("expected ob be flushed")
	}
}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	t.Parallel()
	var holder Orderbook
	holder.dataHandler = make(chan interface{}, 100)
	holder.ob = make(map[Key]*orderbookHolder)
	var snapShot1 orderbook.Base
	snapShot1.Exchange = "WSORDERBOOKTEST1"
	asks := []orderbook.Item{
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

	bids := []orderbook.Item{
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
	err := holder.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot2 orderbook.Base
	snapShot2.Exchange = "WSORDERBOOKTEST2"
	asks = []orderbook.Item{
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

	bids = []orderbook.Item{
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
	snapShot2.Pair, err = currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	err = holder.LoadSnapshot(&snapShot2)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot3 orderbook.Base
	snapShot3.Exchange = "WSORDERBOOKTEST3"
	asks = []orderbook.Item{
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

	bids = []orderbook.Item{
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
	snapShot3.Pair, err = currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	err = holder.LoadSnapshot(&snapShot3)
	if err != nil {
		t.Fatal(err)
	}
	ob, err := holder.GetOrderbook(snapShot1.Pair, snapShot1.Asset)
	if err != nil {
		t.Fatal(err)
	}
	if ob.Asks[0] != snapShot1.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot1.Asks[0],
			ob.Asks[0])
	}
	ob, err = holder.GetOrderbook(snapShot2.Pair, snapShot2.Asset)
	if err != nil {
		t.Fatal(err)
	}
	if ob.Asks[0] != snapShot2.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot2.Asks[0],
			ob.Asks[0])
	}
	ob, err = holder.GetOrderbook(snapShot3.Pair, snapShot3.Asset)
	if err != nil {
		t.Fatal(err)
	}
	if ob.Asks[0] != snapShot3.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot3.Asks[0],
			ob.Asks[0])
	}
}

func TestGetOrderbook(t *testing.T) {
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	ob, err := holder.GetOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	bufferOb := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	b, err := bufferOb.ob.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	askLen, err := bufferOb.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	bidLen, err := bufferOb.ob.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
	if !errors.Is(err, errExchangeConfigNil) {
		t.Fatalf("expected error %v but received %v", errExchangeConfigNil, err)
	}

	exchangeConfig := &config.Exchange{}
	err = w.Setup(exchangeConfig, nil, nil)
	if !errors.Is(err, errBufferConfigNil) {
		t.Fatalf("expected error %v but received %v", errBufferConfigNil, err)
	}

	bufferConf := &Config{}
	err = w.Setup(exchangeConfig, bufferConf, nil)
	if !errors.Is(err, errUnsetDataHandler) {
		t.Fatalf("expected error %v but received %v", errUnsetDataHandler, err)
	}

	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	err = w.Setup(exchangeConfig, bufferConf, make(chan interface{}))
	if !errors.Is(err, errIssueBufferEnabledButNoLimit) {
		t.Fatalf("expected error %v but received %v", errIssueBufferEnabledButNoLimit, err)
	}

	exchangeConfig.Orderbook.WebsocketBufferLimit = 1337
	exchangeConfig.Orderbook.WebsocketBufferEnabled = true
	exchangeConfig.Name = "test"
	bufferConf.SortBuffer = true
	bufferConf.SortBufferByUpdateIDs = true
	bufferConf.UpdateEntriesByID = true
	err = w.Setup(exchangeConfig, bufferConf, make(chan interface{}))
	if err != nil {
		t.Fatal(err)
	}
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
	if !errors.Is(err, errUpdateIsNil) {
		t.Fatalf("expected error %v but received %v", errUpdateIsNil, err)
	}

	err = w.validate(&orderbook.Update{})
	if !errors.Is(err, errUpdateNoTargets) {
		t.Fatalf("expected error %v but received %v", errUpdateNoTargets, err)
	}
}

func TestEnsureMultipleUpdatesViaPrice(t *testing.T) {
	t.Parallel()
	holder, _, _, err := createSnapshot()
	if err != nil {
		t.Error(err)
	}

	asks := bidAskGenerator()
	book := holder.ob[Key{Base: cp.Base.Item, Quote: cp.Quote.Item, Asset: asset.Spot}]
	book.updateByPrice(&orderbook.Update{
		Bids:       asks,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	askLen, err := book.ob.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen <= 3 {
		t.Errorf("Insufficient updates")
	}
}

func deploySliceOrdered(size int) orderbook.Items {
	var items []orderbook.Item
	for i := 0; i < size; i++ {
		items = append(items, orderbook.Item{Amount: 1, Price: rand.Float64() + float64(i), ID: rand.Int63()}) //nolint:gosec // Not needed for tests
	}
	return items
}

func TestUpdateByIDAndAction(t *testing.T) {
	t.Parallel()
	holder := orderbookHolder{}

	asks := deploySliceOrdered(100)
	//nolint: gocritic
	bids := append(asks[:0:0], asks...)
	bids.Reverse()

	book, err := orderbook.DeployDepth("test", cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(asks[:0:0], asks...), 0, time.Time{}, true)

	ob, err := book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = ob.Verify()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	holder.ob = book

	err = holder.updateByIDAndAction(&orderbook.Update{})
	if !errors.Is(err, errInvalidAction) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAction)
	}

	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Amend,
		Bids: []orderbook.Item{
			{
				Price: 100,
				ID:    6969,
			},
		},
	})
	if !strings.Contains(err.Error(), errAmendFailure.Error()) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAmendFailure)
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(asks[:0:0], asks...), 0, time.Time{}, true)
	// append to slice
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.UpdateInsert,
		Bids: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 1,
			},
		},
		Asks: []orderbook.Item{
			{
				Price:  100,
				ID:     1337,
				Amount: 1,
			},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	cpy, err := book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if cpy.Bids[len(cpy.Bids)-1].Price != 0 {
		t.Fatal("did not append bid item")
	}
	if cpy.Asks[len(cpy.Asks)-1].Price != 100 {
		t.Fatal("did not append ask item")
	}

	// Change amount
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.UpdateInsert,
		Bids: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 100,
			},
		},
		Asks: []orderbook.Item{
			{
				Price:  100,
				ID:     1337,
				Amount: 100,
			},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	cpy, err = book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if cpy.Bids[len(cpy.Bids)-1].Amount != 100 {
		t.Fatal("did not update bid amount", cpy.Bids[len(cpy.Bids)-1].Amount)
	}

	if cpy.Asks[len(cpy.Asks)-1].Amount != 100 {
		t.Fatal("did not update ask amount")
	}

	// Change price level
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.UpdateInsert,
		Bids: []orderbook.Item{
			{
				Price:  100,
				ID:     1337,
				Amount: 99,
			},
		},
		Asks: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 99,
			},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	cpy, err = book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if cpy.Bids[0].Amount != 99 && cpy.Bids[0].Price != 100 {
		t.Fatal("did not adjust bid item placement and details")
	}

	if cpy.Asks[0].Amount != 99 && cpy.Asks[0].Amount != 0 {
		t.Fatal("did not adjust ask item placement and details")
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(asks[:0:0], asks...), 0, time.Time{}, true) //nolint:gocritic
	// Delete - not found
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Delete,
		Asks: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 99,
			},
		},
	})
	if !strings.Contains(err.Error(), errDeleteFailure.Error()) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDeleteFailure)
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(asks[:0:0], asks...), 0, time.Time{}, true) //nolint:gocritic
	// Delete - found
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Delete,
		Asks: []orderbook.Item{
			asks[0],
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	askLen, err := book.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 99 {
		t.Fatal("element not deleted")
	}

	// Apply update
	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Amend,
		Asks: []orderbook.Item{
			{ID: 123456},
		},
	})
	if !strings.Contains(err.Error(), errAmendFailure.Error()) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAmendFailure)
	}

	book.LoadSnapshot(bids, bids, 0, time.Time{}, true)

	ob, err = book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(ob.Asks) == 0 || len(ob.Bids) == 0 {
		t.Fatal("expected a valid orderbook")
	}

	update := ob.Asks[0]
	update.Amount = 1337

	err = holder.updateByIDAndAction(&orderbook.Update{
		Action: orderbook.Amend,
		Asks: []orderbook.Item{
			update,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ob, err = book.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob.Asks[0].Amount != 1337 {
		t.Fatal("element not updated")
	}
}

func TestFlushOrderbook(t *testing.T) {
	t.Parallel()
	w := &Orderbook{}
	err := w.Setup(&config.Exchange{Name: "test"}, &Config{}, make(chan interface{}, 2))
	if err != nil {
		t.Fatal(err)
	}

	var snapShot1 orderbook.Base
	snapShot1.Exchange = "Snapshooooot"
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.Asset = asset.Spot
	snapShot1.Pair = cp

	err = w.FlushOrderbook(cp, asset.Spot)
	if err == nil {
		t.Fatal("book not loaded error cannot be nil")
	}

	_, err = w.GetOrderbook(cp, asset.Spot)
	if !errors.Is(err, errDepthNotFound) {
		t.Fatalf("expected: %v but received: %v", errDepthNotFound, err)
	}

	err = w.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Fatal(err)
	}

	err = w.FlushOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.GetOrderbook(cp, asset.Spot)
	if !errors.Is(err, orderbook.ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, orderbook.ErrOrderbookInvalid)
	}
}

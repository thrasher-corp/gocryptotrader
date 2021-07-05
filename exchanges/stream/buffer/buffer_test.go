package buffer

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var itemArray = [][]orderbook.Item{
	{{Price: 1000, Amount: 1, ID: 1000}},
	{{Price: 2000, Amount: 1, ID: 2000}},
	{{Price: 3000, Amount: 1, ID: 3000}},
	{{Price: 3000, Amount: 2, ID: 4000}},
	{{Price: 4000, Amount: 0, ID: 6000}},
	{{Price: 5000, Amount: 1, ID: 5000}},
}

var cp, _ = currency.NewPairFromString("BTCUSD")

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

	newBook := make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)

	holder = &Orderbook{
		exchangeName: exchangeName,
		dataHandler:  make(chan interface{}, 100),
		ob:           newBook,
	}
	err = holder.LoadSnapshot(book)
	return
}

func bidAskGenerator() []orderbook.Item {
	var response []orderbook.Item
	randIterator := 100
	for i := 0; i < randIterator; i++ {
		price := float64(rand.Intn(1000)) // nolint:gosec // no need to import crypo/rand for testing
		if price == 0 {
			price = 1
		}
		response = append(response, orderbook.Item{
			Amount: float64(rand.Intn(10)), // nolint:gosec // no need to import crypo/rand for testing
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
		update := &Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[cp.Base][cp.Quote][asset.Spot]
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
		update := &Update{
			Bids:       bidAsks,
			Asks:       bidAsks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
		}
		holder := ob.ob[cp.Base][cp.Quote][asset.Spot]
		holder.updateByPrice(update)
	}
}

// BenchmarkBufferPerformance demonstrates buffer more performant than multi
// process calls
// 4219518	       287 ns/op	     176 B/op	       1 allocs/op
func BenchmarkBufferPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	holder.bufferEnabled = true
	update := &Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) // nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = holder.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferSortingPerformance benchmark
// 2693391	       467 ns/op	     208 B/op	       2 allocs/op
func BenchmarkBufferSortingPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	holder.bufferEnabled = true
	holder.sortBuffer = true
	update := &Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) // nolint:gosec // no need to import crypo/rand for testing
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = holder.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferSortingPerformance benchmark
// 1000000	      1019 ns/op	     208 B/op	       2 allocs/op
func BenchmarkBufferSortingByIDPerformance(b *testing.B) {
	holder, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	holder.bufferEnabled = true
	holder.sortBuffer = true
	holder.sortBufferByUpdateIDs = true
	update := &Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) // nolint:gosec // no need to import crypo/rand for testing
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
// 9516966	       141 ns/op	       0 B/op	       0 allocs/op
func BenchmarkNoBufferPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	update := &Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(4) // nolint:gosec // no need to import crypo/rand for testing
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

	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	book.updateByPrice(&Update{
		Bids:       itemArray[5],
		Asks:       itemArray[5],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	book.updateByPrice(&Update{
		Bids:       itemArray[0],
		Asks:       itemArray[0],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	if book.ob.GetAskLength() != 3 {
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
		err = holder.Update(&Update{
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

	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	if book.ob.GetAskLength() != 3 {
		t.Errorf("expected 3 entries, received: %v", book.ob.GetAskLength())
	}
	if book.ob.GetBidLength() != 3 {
		t.Errorf("expected 3 entries, received: %v", book.ob.GetBidLength())
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
		err = holder.Update(&Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
			Action:     UpdateInsert,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	if book.ob.GetAskLength() != 6 {
		t.Errorf("expected 5 entries, received: %v", book.ob.GetAskLength())
	}
	if book.ob.GetBidLength() != 6 {
		t.Errorf("expected 5 entries, received: %v", book.ob.GetBidLength())
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
		err = holder.Update(&Update{
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
	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	if book.ob.GetAskLength() != 3 {
		t.Errorf("expected 3 entries, received: %v", book.ob.GetAskLength())
	}
	if book.ob.GetAskLength() != 3 {
		t.Errorf("expected 3 entries, received: %v", book.ob.GetAskLength())
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
		err = holder.Update(&Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: outOFOrderIDs[i],
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	cpy := book.ob.Retrieve()
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

	for i := range itemArray {
		asks := itemArray[i]
		err = holder.Update(&Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: int64(i) + 1,
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
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
	err := holder.Update(&Update{
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
	err := obl.Update(&Update{
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
	obl.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
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
	obl.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
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
	if obl.ob[cp.Base][cp.Quote][asset.Spot] == nil {
		t.Error("expected ob to have ask entries")
	}
	obl.FlushBuffer()
	if obl.ob[cp.Base][cp.Quote][asset.Spot] != nil {
		t.Error("expected ob be flushed")
	}
}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	t.Parallel()
	var holder Orderbook
	holder.dataHandler = make(chan interface{}, 100)
	holder.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
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
	bufferOb := holder.ob[cp.Base][cp.Quote][asset.Spot]
	b := bufferOb.ob.Retrieve()
	if bufferOb.ob.GetAskLength() != len(ob.Asks) ||
		bufferOb.ob.GetBidLength() != len(ob.Bids) ||
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
	err := w.Setup(0, false, false, false, false, true, "", nil)
	if !errors.Is(err, errUnsetExchangeName) {
		t.Fatalf("expected error %v but received %v", errUnsetExchangeName, err)
	}

	err = w.Setup(0, false, false, false, false, false, "test", nil)
	if !errors.Is(err, errUnsetDataHandler) {
		t.Fatalf("expected error %v but received %v", errUnsetDataHandler, err)
	}

	err = w.Setup(0, true, false, false, false, true, "test", make(chan interface{}))
	if !errors.Is(err, errIssueBufferEnabledButNoLimit) {
		t.Fatalf("expected error %v but received %v", errIssueBufferEnabledButNoLimit, err)
	}

	err = w.Setup(1337, true, true, true, true, false, "test", make(chan interface{}))
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

	err = w.validate(&Update{})
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
	book := holder.ob[cp.Base][cp.Quote][asset.Spot]
	book.updateByPrice(&Update{
		Bids:       asks,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	if book.ob.GetAskLength() <= 3 {
		t.Errorf("Insufficient updates")
	}
}

func deploySliceOrdered(size int) orderbook.Items {
	rand.Seed(time.Now().UnixNano())
	var items []orderbook.Item
	for i := 0; i < size; i++ {
		items = append(items, orderbook.Item{Amount: 1, Price: rand.Float64() + float64(i), ID: rand.Int63()}) // nolint:gosec // Not needed for tests
	}
	return items
}

func TestUpdateByIDAndAction(t *testing.T) {
	t.Parallel()
	holder := orderbookHolder{}

	asks := deploySliceOrdered(100)
	bids := append(asks[:0:0], asks...)
	bids.Reverse()

	book, err := orderbook.DeployDepth("test", cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(asks[:0:0], asks...))

	err = book.Retrieve().Verify()
	if err != nil {
		t.Fatal(err)
	}

	holder.ob = book

	err = holder.updateByIDAndAction(&Update{})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = holder.updateByIDAndAction(&Update{
		Action: Amend,
		Bids: []orderbook.Item{
			{
				Price: 100,
				ID:    6969,
			},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	// append to slice
	err = holder.updateByIDAndAction(&Update{
		Action: UpdateInsert,
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
	if err != nil {
		t.Fatal(err)
	}

	cpy := book.Retrieve()

	if cpy.Bids[len(cpy.Bids)-1].Price != 0 {
		t.Fatal("did not append bid item")
	}
	if cpy.Asks[len(cpy.Asks)-1].Price != 100 {
		t.Fatal("did not append ask item")
	}

	// Change amount
	err = holder.updateByIDAndAction(&Update{
		Action: UpdateInsert,
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
	if err != nil {
		t.Fatal(err)
	}

	cpy = book.Retrieve()

	if cpy.Bids[len(cpy.Bids)-1].Amount != 100 {
		t.Fatal("did not update bid amount", cpy.Bids[len(cpy.Bids)-1].Amount)
	}

	if cpy.Asks[len(cpy.Asks)-1].Amount != 100 {
		t.Fatal("did not update ask amount")
	}

	// Change price level
	err = holder.updateByIDAndAction(&Update{
		Action: UpdateInsert,
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
	if err != nil {
		t.Fatal(err)
	}

	cpy = book.Retrieve()

	if cpy.Bids[0].Amount != 99 && cpy.Bids[0].Price != 100 {
		t.Fatal("did not adjust bid item placement and details")
	}

	if cpy.Asks[0].Amount != 99 && cpy.Asks[0].Amount != 0 {
		t.Fatal("did not adjust ask item placement and details")
	}

	book.LoadSnapshot(append(bids[:0:0], bids...), append(bids[:0:0], bids...)) // nolint:gocritic

	// Delete - not found
	err = holder.updateByIDAndAction(&Update{
		Action: Delete,
		Asks: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 99,
			},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = holder.updateByIDAndAction(&Update{
		Action: Delete,
		Bids: []orderbook.Item{
			{
				Price:  0,
				ID:     1337,
				Amount: 99,
			},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	// Delete - found
	err = holder.updateByIDAndAction(&Update{
		Action: Delete,
		Asks: []orderbook.Item{
			asks[0],
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if book.GetAskLength() != 99 {
		t.Fatal("element not deleted")
	}

	// Apply update
	err = holder.updateByIDAndAction(&Update{
		Action: Amend,
		Asks: []orderbook.Item{
			{ID: 123456},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	update := book.Retrieve().Asks[0]
	update.Amount = 1337

	err = holder.updateByIDAndAction(&Update{
		Action: Amend,
		Asks: []orderbook.Item{
			update,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if book.Retrieve().Asks[0].Amount != 1337 {
		t.Fatal("element not updated")
	}
}

func TestFlushOrderbook(t *testing.T) {
	t.Parallel()
	w := &Orderbook{}
	err := w.Setup(5, false, false, false, false, false, "test", make(chan interface{}, 2))
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

	o, err := w.GetOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if len(o.Bids) != 0 || len(o.Asks) != 0 {
		t.Fatal("orderbook items not flushed")
	}
}

package buffer

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var itemArray = [][]orderbook.Item{
	{{Price: 1000, Amount: 1, ID: 1}},
	{{Price: 2000, Amount: 1, ID: 2}},
	{{Price: 3000, Amount: 1, ID: 3}},
	{{Price: 3000, Amount: 2, ID: 4}},
	{{Price: 4000, Amount: 0, ID: 6}},
	{{Price: 5000, Amount: 1, ID: 5}},
}

var cp, _ = currency.NewPairFromString("BTCUSD")

const (
	exchangeName = "exchangeTest"
)

func createSnapshot() (obl *Orderbook, asks, bids []orderbook.Item, err error) {
	var snapShot1 orderbook.Base
	snapShot1.ExchangeName = exchangeName
	asks = []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	bids = []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	obl = &Orderbook{exchangeName: exchangeName, dataHandler: make(chan interface{}, 100)}
	err = obl.LoadSnapshot(&snapShot1)
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
		ob.updateBidsByPrice(ob.ob[cp][asset.Spot], update)
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
		ob.updateAsksByPrice(ob.ob[cp][asset.Spot], update)
	}
}

// BenchmarkBufferPerformance demonstrates buffer more performant than multi
// process calls
func BenchmarkBufferPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.bufferEnabled = true
	// This is to ensure we do not send in zero orderbook info to our main book
	// in orderbook.go, orderbooks should not be zero even after an update.
	dummyItem := orderbook.Item{
		Amount: 1333337,
		Price:  1337.1337,
		ID:     1337,
	}
	obl.ob[cp][asset.Spot].Bids = append(obl.ob[cp][asset.Spot].Bids, dummyItem)
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

// BenchmarkBufferSortingPerformance benchmark
func BenchmarkBufferSortingPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.bufferEnabled = true
	obl.sortBuffer = true
	// This is to ensure we do not send in zero orderbook info to our main book
	// in orderbook.go, orderbooks should not be zero even after an update.
	dummyItem := orderbook.Item{
		Amount: 1333337,
		Price:  1337.1337,
		ID:     1337,
	}
	obl.ob[cp][asset.Spot].Bids = append(obl.ob[cp][asset.Spot].Bids, dummyItem)
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

// BenchmarkBufferSortingPerformance benchmark
func BenchmarkBufferSortingByIDPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.bufferEnabled = true
	obl.sortBuffer = true
	obl.sortBufferByUpdateIDs = true
	// This is to ensure we do not send in zero orderbook info to our main book
	// in orderbook.go, orderbooks should not be zero even after an update.
	dummyItem := orderbook.Item{
		Amount: 1333337,
		Price:  1337.1337,
		ID:     1337,
	}
	obl.ob[cp][asset.Spot].Bids = append(obl.ob[cp][asset.Spot].Bids, dummyItem)
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

// BenchmarkNoBufferPerformance demonstrates orderbook process less performant
// than buffer
func BenchmarkNoBufferPerformance(b *testing.B) {
	obl, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	// This is to ensure we do not send in zero orderbook info to our main book
	// in orderbook.go, orderbooks should not be zero even after an update.
	dummyItem := orderbook.Item{
		Amount: 1333337,
		Price:  1337.1337,
		ID:     1337,
	}
	obl.ob[cp][asset.Spot].Bids = append(obl.ob[cp][asset.Spot].Bids, dummyItem)
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
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Error(err)
	}

	obl.updateAsksByPrice(obl.ob[cp][asset.Spot], &Update{
		Bids:       itemArray[5],
		Asks:       itemArray[5],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	obl.updateAsksByPrice(obl.ob[cp][asset.Spot], &Update{
		Bids:       itemArray[0],
		Asks:       itemArray[0],
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	if len(obl.ob[cp][asset.Spot].Asks) != 3 {
		t.Error("Did not update")
	}
}

// TestHittingTheBuffer logic test
func TestHittingTheBuffer(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.bufferEnabled = true
	obl.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&Update{
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
	if len(obl.ob[cp][asset.Spot].Asks) != 3 {
		t.Log(obl.ob[cp][asset.Spot])
		t.Errorf("expected 3 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Asks))
	}
	if len(obl.ob[cp][asset.Spot].Bids) != 3 {
		t.Errorf("expected 3 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Bids))
	}
}

// TestInsertWithIDs logic test
func TestInsertWithIDs(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.bufferEnabled = true
	obl.updateEntriesByID = true
	obl.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
			Action:     "insert",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[cp][asset.Spot].Asks) != 6 {
		t.Errorf("expected 6 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Asks))
	}
	if len(obl.ob[cp][asset.Spot].Bids) != 6 {
		t.Errorf("expected 6 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Bids))
	}
}

// TestSortIDs logic test
func TestSortIDs(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.bufferEnabled = true
	obl.sortBufferByUpdateIDs = true
	obl.sortBuffer = true
	obl.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&Update{
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
	if len(obl.ob[cp][asset.Spot].Asks) != 3 {
		t.Errorf("expected 3 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Asks))
	}
	if len(obl.ob[cp][asset.Spot].Bids) != 3 {
		t.Errorf("expected 3 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Bids))
	}
}

// TestDeleteWithIDs logic test
func TestDeleteWithIDs(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	// This is to ensure we do not send in zero orderbook info to our main book
	// in orderbook.go, orderbooks should not be zero even after an update.
	dummyItem := orderbook.Item{
		Amount: 1333337,
		Price:  1337.1337,
		ID:     1337,
	}
	obl.ob[cp][asset.Spot].Bids = append(obl.ob[cp][asset.Spot].Bids, dummyItem)
	obl.ob[cp][asset.Spot].Asks = append(obl.ob[cp][asset.Spot].Asks,
		itemArray[2][0])
	obl.ob[cp][asset.Spot].Asks = append(obl.ob[cp][asset.Spot].Asks,
		itemArray[1][0])

	obl.updateEntriesByID = true
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
			Action:     "delete",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(obl.ob[cp][asset.Spot].Asks) != 0 {
		t.Errorf("expected 0 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Asks))
	}
	if len(obl.ob[cp][asset.Spot].Bids) != 1 {
		t.Errorf("expected 1 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Bids))
	}
}

// TestUpdateWithIDs logic test
func TestUpdateWithIDs(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.updateEntriesByID = true
	for i := range itemArray {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateTime: time.Now(),
			Asset:      asset.Spot,
			Action:     "update",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[cp][asset.Spot].Asks) != 1 {
		t.Log(obl.ob[cp][asset.Spot])
		t.Errorf("expected 1 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Asks))
	}
	if len(obl.ob[cp][asset.Spot].Bids) != 1 {
		t.Errorf("expected 1 entries, received: %v",
			len(obl.ob[cp][asset.Spot].Bids))
	}
}

// TestOutOfOrderIDs logic test
func TestOutOfOrderIDs(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	outOFOrderIDs := []int64{2, 1, 5, 3, 4, 6, 7}
	if itemArray[0][0].Price != 1000 {
		t.Errorf("expected sorted price to be 3000, received: %v",
			itemArray[1][0].Price)
	}
	obl.bufferEnabled = true
	obl.sortBuffer = true
	obl.obBufferLimit = 5
	for i := range itemArray {
		asks := itemArray[i]
		err = obl.Update(&Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: outOFOrderIDs[i],
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Index 1 since index 0 is price 7000
	if obl.ob[cp][asset.Spot].Asks[1].Price != 2000 {
		t.Errorf("expected sorted price to be 3000, received: %v",
			obl.ob[cp][asset.Spot].Asks[1].Price)
	}
}

func TestOrderbookLastUpdateID(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if exp := float64(1000); itemArray[0][0].Price != exp {
		t.Errorf("expected sorted price to be %f, received: %v",
			exp, itemArray[1][0].Price)
	}

	for i := range itemArray {
		asks := itemArray[i]
		err = obl.Update(&Update{
			Asks:     asks,
			Pair:     cp,
			UpdateID: int64(i) + 1,
			Asset:    asset.Spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	ob := obl.GetOrderbook(cp, asset.Spot)
	if exp := len(itemArray); ob.LastUpdateID != int64(exp) {
		t.Errorf("expected last update id to be %d, received: %v", exp, ob.LastUpdateID)
	}
}

// TestRunUpdateWithoutSnapshot logic test
func TestRunUpdateWithoutSnapshot(t *testing.T) {
	var obl Orderbook
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
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	obl.exchangeName = exchangeName
	err := obl.Update(&Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != "ob.Base could not be found for Exchange exchangeTest CurrencyPair: BTCUSD AssetType: spot" {
		t.Fatal(err)
	}
}

// TestRunUpdateWithoutAnyUpdates logic test
func TestRunUpdateWithoutAnyUpdates(t *testing.T) {
	var obl Orderbook
	var snapShot1 orderbook.Base
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	obl.exchangeName = exchangeName
	err := obl.Update(&Update{
		Bids:       snapShot1.Asks,
		Asks:       snapShot1.Bids,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != fmt.Sprintf("%v cannot have bids and ask targets both nil",
		exchangeName) {
		t.Fatal("expected nil asks and bids error")
	}
}

// TestRunSnapshotWithNoData logic test
func TestRunSnapshotWithNoData(t *testing.T) {
	var obl Orderbook
	var snapShot1 orderbook.Base
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	snapShot1.ExchangeName = "test"
	obl.exchangeName = "test"
	err := obl.LoadSnapshot(&snapShot1)
	if err == nil {
		t.Fatal("expected an error loading a snapshot")
	}
	if err.Error() != "test snapshot ask and bids are nil" {
		t.Fatal(err)
	}
}

// TestLoadSnapshot logic test
func TestLoadSnapshot(t *testing.T) {
	var obl Orderbook
	obl.dataHandler = make(chan interface{}, 100)
	var snapShot1 orderbook.Base
	snapShot1.ExchangeName = "SnapshotWithOverride"
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	err := obl.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Error(err)
	}
}

// TestFlushbuffer logic test
func TestFlushbuffer(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if obl.ob[cp][asset.Spot] == nil {
		t.Error("expected ob to have ask entries")
	}
	obl.FlushBuffer()
	if obl.ob[cp][asset.Spot] != nil {
		t.Error("expected ob be flushed")
	}
}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	var obl Orderbook
	obl.dataHandler = make(chan interface{}, 100)
	var snapShot1 orderbook.Base
	snapShot1.ExchangeName = "WSORDERBOOKTEST1"
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
	snapShot1.AssetType = asset.Spot
	snapShot1.Pair = cp
	err := obl.LoadSnapshot(&snapShot1)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot2 orderbook.Base
	snapShot2.ExchangeName = "WSORDERBOOKTEST2"
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
	snapShot2.Bids = bids
	snapShot2.AssetType = asset.Spot
	snapShot2.Pair, err = currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	err = obl.LoadSnapshot(&snapShot2)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot3 orderbook.Base
	snapShot3.ExchangeName = "WSORDERBOOKTEST3"
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
	snapShot3.Bids = bids
	snapShot3.AssetType = "FUTURES"
	snapShot3.Pair, err = currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	err = obl.LoadSnapshot(&snapShot3)
	if err != nil {
		t.Fatal(err)
	}
	if obl.ob[snapShot1.Pair][snapShot1.AssetType].Asks[0] != snapShot1.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot1.Asks[0],
			obl.ob[snapShot1.Pair][snapShot1.AssetType].Asks[0])
	}
	if obl.ob[snapShot2.Pair][snapShot2.AssetType].Asks[0] != snapShot2.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot2.Asks[0],
			obl.ob[snapShot2.Pair][snapShot2.AssetType].Asks[0])
	}
	if obl.ob[snapShot3.Pair][snapShot3.AssetType].Asks[0] != snapShot3.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v",
			snapShot3.Asks[0],
			obl.ob[snapShot3.Pair][snapShot3.AssetType].Asks[0])
	}
}

func TestGetOrderbook(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	ob := obl.GetOrderbook(cp, asset.Spot)
	if obl.ob[cp][asset.Spot] != ob {
		t.Error("Failed to get orderbook")
	}
}

func TestSetup(t *testing.T) {
	w := Orderbook{}
	w.Setup(1, true, true, true, true, "hi", make(chan interface{}))
	if w.obBufferLimit != 1 ||
		!w.bufferEnabled ||
		!w.sortBuffer ||
		!w.sortBufferByUpdateIDs ||
		!w.updateEntriesByID ||
		w.exchangeName != "hi" {
		t.Errorf("Setup incorrectly loaded %s", w.exchangeName)
	}
}

func TestEnsureMultipleUpdatesViaPrice(t *testing.T) {
	obl, _, _, err := createSnapshot()
	if err != nil {
		t.Error(err)
	}

	asks := bidAskGenerator()
	obl.updateAsksByPrice(obl.ob[cp][asset.Spot], &Update{
		Bids:       asks,
		Asks:       asks,
		Pair:       cp,
		UpdateTime: time.Now(),
		Asset:      asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}

	if len(obl.ob[cp][asset.Spot].Asks) <= 3 {
		t.Errorf("Insufficient updates")
	}
}

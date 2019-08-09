package wsorderbook

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
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

const (
	exchangeName = "exchangeTest"
	spot         = orderbook.Spot
)

func createSnapshot() (obl *WebsocketOrderbookLocal, curr currency.Pair, asks, bids []orderbook.Item, err error) {
	var snapShot1 orderbook.Base
	curr = currency.NewPairFromString("BTCUSD")
	asks = []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	bids = []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	obl = &WebsocketOrderbookLocal{}
	err = obl.LoadSnapshot(&snapShot1, false)
	return
}

// BenchmarkBufferPerformance demonstrates buffer more performant than multi process calls
func BenchmarkBufferPerformance(b *testing.B) {
	obl, curr, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.sortBuffer = true
	update := &WebsocketOrderbookUpdate{
		Bids:         bids,
		Asks:         asks,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		AssetType:    spot,
	}
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(5)
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
	obl, curr, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.sortBuffer = true
	obl.bufferEnabled = true
	obl.obBufferLimit = 5
	update := &WebsocketOrderbookUpdate{
		Bids:         bids,
		Asks:         asks,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		AssetType:    spot,
	}
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(5)
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = obl.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNoBufferPerformance demonstrates orderbook process less performant than buffer
func BenchmarkNoBufferPerformance(b *testing.B) {
	obl, curr, asks, bids, err := createSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	obl.exchangeName = exchangeName
	update := &WebsocketOrderbookUpdate{
		Bids:         bids,
		Asks:         asks,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		AssetType:    spot,
	}
	for i := 0; i < b.N; i++ {
		randomIndex := rand.Intn(5)
		update.Asks = itemArray[randomIndex]
		update.Bids = itemArray[randomIndex]
		err = obl.Update(update)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestHittingTheBuffer logic test
func TestHittingTheBuffer(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.bufferEnabled = true
	obl.obBufferLimit = 5
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			AssetType:    spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[curr][spot].Asks) != 3 {
		t.Log(obl.ob[curr][spot])
		t.Errorf("expected 3 entries, received: %v", len(obl.ob[curr][spot].Asks))
	}
	if len(obl.ob[curr][spot].Bids) != 3 {
		t.Errorf("expected 3 entries, received: %v", len(obl.ob[curr][spot].Bids))
	}
}

// TestInsertWithIDs logic test
func TestInsertWithIDs(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.bufferEnabled = true
	obl.updateEntriesByID = true
	obl.obBufferLimit = 5
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			AssetType:    spot,
			Action:       "insert",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[curr][spot].Asks) != 6 {
		t.Errorf("expected 6 entries, received: %v", len(obl.ob[curr][spot].Asks))
	}
	if len(obl.ob[curr][spot].Bids) != 6 {
		t.Errorf("expected 6 entries, received: %v", len(obl.ob[curr][spot].Bids))
	}
}

// TestSortIDs logic test
func TestSortIDs(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.bufferEnabled = true
	obl.sortBufferByUpdateIDs = true
	obl.sortBuffer = true
	obl.obBufferLimit = 5
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateID:     int64(i),
			AssetType:    spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[curr][spot].Asks) != 3 {
		t.Errorf("expected 6 entries, received: %v", len(obl.ob[curr][spot].Asks))
	}
	if len(obl.ob[curr][spot].Bids) != 3 {
		t.Errorf("expected 6 entries, received: %v", len(obl.ob[curr][spot].Bids))
	}
}

// TestDeleteWithIDs logic test
func TestDeleteWithIDs(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.updateEntriesByID = true
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			AssetType:    spot,
			Action:       "delete",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[curr][spot].Asks) != 0 {
		t.Errorf("expected 0 entries, received: %v", len(obl.ob[curr][spot].Asks))
	}
	if len(obl.ob[curr][spot].Bids) != 0 {
		t.Errorf("expected 0 entries, received: %v", len(obl.ob[curr][spot].Bids))
	}
}

// TestUpdateWithIDs logic test
func TestUpdateWithIDs(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	obl.exchangeName = exchangeName
	obl.updateEntriesByID = true
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		bids := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			AssetType:    spot,
			Action:       "update",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(obl.ob[curr][spot].Asks) != 1 {
		t.Log(obl.ob[curr][spot])
		t.Errorf("expected 1 entries, received: %v", len(obl.ob[curr][spot].Asks))
	}
	if len(obl.ob[curr][spot].Bids) != 1 {
		t.Errorf("expected 1 entries, received: %v", len(obl.ob[curr][spot].Bids))
	}
}

// TestOutOfOrderIDs logic test
func TestOutOfOrderIDs(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	outOFOrderIDs := []int64{2, 1, 5, 3, 4, 6}
	if itemArray[0][0].Price != 1000 {
		t.Errorf("expected sorted price to be 3000, received: %v", itemArray[1][0].Price)
	}
	obl.exchangeName = exchangeName
	obl.bufferEnabled = true
	obl.sortBuffer = true
	obl.obBufferLimit = 5
	for i := 0; i < len(itemArray); i++ {
		asks := itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Asks:         asks,
			CurrencyPair: curr,
			UpdateID:     outOFOrderIDs[i],
			AssetType:    spot,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Index 1 since index 0 is price 7000
	if obl.ob[curr][spot].Asks[1].Price != 2000 {
		t.Errorf("expected sorted price to be 3000, received: %v", obl.ob[curr][spot].Asks[1].Price)
	}
}

// TestRunUpdateWithoutSnapshot logic test
func TestRunUpdateWithoutSnapshot(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 5999, Amount: 1, ID: 8},
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	obl.exchangeName = exchangeName
	err := obl.Update(&WebsocketOrderbookUpdate{
		Bids:         bids,
		Asks:         asks,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		AssetType:    spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != "ob.Base could not be found for Exchange exchangeTest CurrencyPair: BTCUSD AssetType: SPOT" {
		t.Fatal(err)
	}
}

// TestRunUpdateWithoutAnyUpdates logic test
func TestRunUpdateWithoutAnyUpdates(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	obl.exchangeName = exchangeName
	err := obl.Update(&WebsocketOrderbookUpdate{
		Bids:         snapShot1.Asks,
		Asks:         snapShot1.Bids,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		AssetType:    spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != fmt.Sprintf("%v cannot have bids and ask targets both nil", exchangeName) {
		t.Fatal("expected nil asks and bids error")
	}
}

// TestRunSnapshotWithNoData logic test
func TestRunSnapshotWithNoData(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1,
		false)
	if err == nil {
		t.Fatal("expected an error loading a snapshot")
	}
	if err.Error() != "snapshot ask and bids are nil" {
		t.Fatal(err)
	}
}

// TestLoadSnapshotWithOverride logic test
func TestLoadSnapshotWithOverride(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 8},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 9},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, false)
	if err != nil {
		t.Error(err)
	}
	err = obl.LoadSnapshot(&snapShot1, false)
	if err == nil {
		t.Error("expected error: 'snapshot instance already found'")
	}
	err = obl.LoadSnapshot(&snapShot1, true)
	if err != nil {
		t.Error(err)
	}
}

// TestInsertWithIDs logic test
func TestFlushCache(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if obl.ob[curr][spot] == nil {
		t.Error("expected ob to have ask entries")
	}
	obl.FlushCache()
	if obl.ob[curr][spot] != nil {
		t.Error("expected ob be flushed")
	}

}

// TestInsertingSnapShots logic test
func TestInsertingSnapShots(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
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
	snapShot1.AssetType = spot
	snapShot1.Pair = currency.NewPairFromString("BTCUSD")
	err := obl.LoadSnapshot(&snapShot1, false)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot2 orderbook.Base
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
	snapShot2.AssetType = spot
	snapShot2.Pair = currency.NewPairFromString("LTCUSD")
	err = obl.LoadSnapshot(&snapShot2, false)
	if err != nil {
		t.Fatal(err)
	}
	var snapShot3 orderbook.Base
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
	snapShot3.Pair = currency.NewPairFromString("LTCUSD")
	err = obl.LoadSnapshot(&snapShot3, false)
	if err != nil {
		t.Fatal(err)
	}
	if obl.ob[snapShot1.Pair][snapShot1.AssetType].Asks[0] != snapShot1.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v", snapShot1.Asks[0], obl.ob[snapShot1.Pair][snapShot1.AssetType].Asks[0])
	}
	if obl.ob[snapShot2.Pair][snapShot2.AssetType].Asks[0] != snapShot2.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v", snapShot2.Asks[0], obl.ob[snapShot2.Pair][snapShot2.AssetType].Asks[0])
	}
	if obl.ob[snapShot3.Pair][snapShot3.AssetType].Asks[0] != snapShot3.Asks[0] {
		t.Errorf("loaded data mismatch. Expected %v, received %v", snapShot3.Asks[0], obl.ob[snapShot3.Pair][snapShot3.AssetType].Asks[0])
	}
}

func TestGetOrderbook(t *testing.T) {
	obl, curr, _, _, err := createSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	ob := obl.GetOrderbook(curr, spot)
	if obl.ob[curr][spot] != ob {
		t.Error("Failed to get orderbook")
	}
}

func TestSetup(t *testing.T) {
	w := WebsocketOrderbookLocal{}
	w.Setup(1, true, true, true, true, "hi")
	if w.obBufferLimit != 1 || !w.bufferEnabled || !w.sortBuffer || !w.sortBufferByUpdateIDs || !w.updateEntriesByID || w.exchangeName != "hi" {
		t.Errorf("Setup incorrectly loaded %v", w)
	}
}

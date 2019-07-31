package ob

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

var itemArray = [][]orderbook.Item{
	[]orderbook.Item{{Price: 1000, Amount: 1}},
	[]orderbook.Item{{Price: 2000, Amount: 1}},
	[]orderbook.Item{{Price: 3000, Amount: 1}},
	[]orderbook.Item{{Price: 3000, Amount: 2}},
	[]orderbook.Item{{Price: 4000, Amount: 0, ID: 6}},
	[]orderbook.Item{{Price: 5000, Amount: 1}},
}

const (
	exchangeName = "exchangeTest"
	spot         = "SPOT"
	futures      = "FUTURES"
)

func TestHittingTheBuffer(t *testing.T) {
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
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		bids = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:          bids,
			Asks:          asks,
			CurrencyPair:  curr,
			UpdateTime:    time.Now(),
			ExchangeName:  exchangeName,
			AssetType:     spot,
			BufferEnabled: true,
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

func TestInsertWithIDs(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		bids = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:          bids,
			Asks:          asks,
			CurrencyPair:  curr,
			UpdateTime:    time.Now(),
			ExchangeName:  exchangeName,
			AssetType:     spot,
			Action:        "insert",
			UpdateByIDs:   true,
			BufferEnabled: true,
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

func TestSortIDs(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		bids = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:             bids,
			Asks:             asks,
			CurrencyPair:     curr,
			UpdateID:         int64(i),
			ExchangeName:     exchangeName,
			AssetType:        spot,
			OrderByUpdateIDs: true,
			BufferEnabled:    true,
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

func TestOutOfOrderIDs(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 7000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = asks
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	outOFOrderIDs := []int64{2, 1, 5, 3, 4, 6}
	if itemArray[0][0].Price != 1000 {
		t.Errorf("expected sorted price to be 3000, received: %v", itemArray[1][0].Price)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Asks:             asks,
			CurrencyPair:     curr,
			UpdateID:         outOFOrderIDs[i],
			ExchangeName:     exchangeName,
			AssetType:        spot,
			OrderByUpdateIDs: true,
			BufferEnabled:    true,
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

func TestDeleteWithIDs(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		bids = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			ExchangeName: exchangeName,
			AssetType:    spot,
			Action:       "delete",
			UpdateByIDs:  true,
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

func TestUpdateWithIDs(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	asks := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	bids := []orderbook.Item{
		{Price: 4000, Amount: 1, ID: 6},
	}
	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(itemArray); i++ {
		asks = itemArray[i]
		bids = itemArray[i]
		err = obl.Update(&WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: curr,
			UpdateTime:   time.Now(),
			ExchangeName: exchangeName,
			AssetType:    spot,
			Action:       "update",
			UpdateByIDs:  true,
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
	err := obl.Update(&WebsocketOrderbookUpdate{
		Bids:         bids,
		Asks:         asks,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		ExchangeName: exchangeName,
		AssetType:    spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != "ob.Base could not be found for Exchange exchangeTest CurrencyPair: BTCUSD AssetType: SPOT" {
		t.Fatal(err)
	}
}

func TestRunUpdateWithoutAnyUpdatesLol(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.Update(&WebsocketOrderbookUpdate{
		Bids:         snapShot1.Asks,
		Asks:         snapShot1.Bids,
		CurrencyPair: curr,
		UpdateTime:   time.Now(),
		ExchangeName: exchangeName,
		AssetType:    spot,
	})
	if err == nil {
		t.Fatal("expected an error running update with no snapshot loaded")
	}
	if err.Error() != "cannot have bids and ask targets both nil" {
		t.Fatal(err)
	}
}

func TestRunSnapshotWithNoDataLmao(t *testing.T) {
	var obl WebsocketOrderbookLocal
	var snapShot1 orderbook.Base
	curr := currency.NewPairFromString("BTCUSD")
	snapShot1.Asks = []orderbook.Item{}
	snapShot1.Bids = []orderbook.Item{}
	snapShot1.AssetType = spot
	snapShot1.Pair = curr
	err := obl.LoadSnapshot(&snapShot1,
		exchangeName,
		false)
	if err == nil {
		t.Fatal("expected an error loading a snapshot")
	}
	if err.Error() != "snapshot ask and bids are nil" {
		t.Fatal(err)
	}
}

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
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Error(err)
	}
	err = obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err == nil {
		t.Error("expected error: 'snapshot instance already found'")
	}
	err = obl.LoadSnapshot(&snapShot1, exchangeName, true)
	if err != nil {
		t.Error(err)
	}
}

func TestFlushCache(t *testing.T) {
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
	err := obl.LoadSnapshot(&snapShot1, exchangeName, false)
	if err != nil {
		t.Error(err)
	}
	if obl.ob[curr][spot] == nil {
		t.Error("expected ob to have ask entries")
	}
	obl.FlushCache()
	if obl.ob[curr][spot] != nil {
		t.Error("expected ob be flushed")
	}

}

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

	obl.LoadSnapshot(&snapShot1, exchangeName, false)

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

	obl.LoadSnapshot(&snapShot2, exchangeName, false)

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

	obl.LoadSnapshot(&snapShot3, exchangeName, false)
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

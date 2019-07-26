package orderbook

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

var obl WebsocketOrderbookLocal

func TestInsertingSnapShots(t *testing.T) {
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
	snapShot1.AssetType = "SPOT"
	snapShot1.Pair = currency.NewPairFromString("BTCUSD")

	obl.LoadSnapshot(&snapShot1, "ExchangeTest", false)

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
	snapShot2.AssetType = "SPOT"
	snapShot2.Pair = currency.NewPairFromString("LTCUSD")

	obl.LoadSnapshot(&snapShot2, "ExchangeTest", false)

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

	obl.LoadSnapshot(&snapShot3, "ExchangeTest", false)
	if obl.orderbook[snapShot1.Pair][snapShot1.AssetType].Asks[0] != snapShot1.Asks[0] {
		t.Error("test failed - inserting orderbook data")
	}
	if obl.orderbook[snapShot2.Pair][snapShot2.AssetType].Asks[0] != snapShot2.Asks[0] {
		t.Error("test failed - inserting orderbook data")
	}
	if obl.orderbook[snapShot3.Pair][snapShot3.AssetType].Asks[0] != snapShot3.Asks[0] {
		t.Error("test failed - inserting orderbook data")
	}
}

func TestUpdate(t *testing.T) {
	LTCUSDPAIR := currency.NewPairFromString("LTCUSD")
	BTCUSDPAIR := currency.NewPairFromString("BTCUSD")

	bidTargets := []orderbook.Item{
		{Price: 1, Amount: 24},  // Amend
		{Price: 2, Amount: 0},   // Delete
		{Price: 3, Amount: 100}, // Append
		{Price: 4, Amount: 0},   // Ghost delete
	}

	askTargets := []orderbook.Item{
		{Price: 5, Amount: 24},  // Amend
		{Price: 6, Amount: 0},   // Delete
		{Price: 7, Amount: 100}, // Append
		{Price: 8, Amount: 0},   // Ghost delete
	}
	err := obl.Update(bidTargets,
		askTargets,
		LTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"SPOT")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}
	if obl.orderbookBuffer[LTCUSDPAIR]["SPOT"][0].Bids[0].Price != bidTargets[0].Price {
		t.Error(obl.orderbookBuffer)
	}

	err = obl.Update(bidTargets,
		askTargets,
		LTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"FUTURES")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}

	bidTargets = []orderbook.Item{
		{Price: 5999, Amount: 24},  // Amend
		{Price: 5998, Amount: 0},   // Delete
		{Price: 1337, Amount: 100}, // Append
		{Price: 1336, Amount: 0},   // Ghost delete
	}

	askTargets = []orderbook.Item{
		{Price: 6000, Amount: 24},  // Amend
		{Price: 6001, Amount: 0},   // Delete
		{Price: 1337, Amount: 100}, // Append
		{Price: 1336, Amount: 0},   // Ghost delete
	}

	err = obl.Update(bidTargets,
		askTargets,
		BTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"SPOT")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}
}

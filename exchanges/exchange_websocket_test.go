package exchange

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

var wsTest Base

func TestWebsocketInit(t *testing.T) {
	if wsTest.Websocket != nil {
		t.Error("test failed - WebsocketInit() error")
	}

	wsTest.WebsocketInit()

	if wsTest.Websocket == nil {
		t.Error("test failed - WebsocketInit() error")
	}
}

func TestWebsocket(t *testing.T) {
	if err := wsTest.Websocket.SetProxyAddress("testProxy"); err != nil {
		t.Error("test failed - SetProxyAddress", err)
	}

	wsTest.WebsocketSetup(func() error { return nil },
		"testName",
		true,
		"testDefaultURL",
		"testRunningURL")

	// Test variable setting and retreival
	if wsTest.Websocket.GetName() != "testName" {
		t.Error("test failed - WebsocketSetup")
	}

	if !wsTest.Websocket.IsEnabled() {
		t.Error("test failed - WebsocketSetup")
	}

	if wsTest.Websocket.GetProxyAddress() != "testProxy" {
		t.Error("test failed - WebsocketSetup")
	}

	if wsTest.Websocket.GetDefaultURL() != "testDefaultURL" {
		t.Error("test failed - WebsocketSetup")
	}

	if wsTest.Websocket.GetWebsocketURL() != "testRunningURL" {
		t.Error("test failed - WebsocketSetup")
	}

	// Test websocket connect and shutdown functions
	comms := make(chan struct{}, 1)
	go func() {
		var count int
		for {
			if count == 4 {
				close(comms)
				return
			}
			select {
			case <-wsTest.Websocket.Connected:
				count++
			case <-wsTest.Websocket.Disconnected:
				count++
			}
		}
	}()

	// -- Not connected shutdown
	err := wsTest.Websocket.Shutdown()
	if err == nil {
		t.Fatal("test failed - should not be connected to able to shut down")
	}

	// -- Normal connect
	err = wsTest.Websocket.Connect()
	if err != nil {
		t.Fatal("test failed - WebsocketSetup", err)
	}

	// -- Already connected connect
	err = wsTest.Websocket.Connect()
	if err == nil {
		t.Fatal("test failed - should not connect, already connected")
	}

	wsTest.Websocket.SetWebsocketURL("")

	// -- Set true when already true
	err = wsTest.Websocket.SetEnabled(true)
	if err == nil {
		t.Fatal("test failed - setting enabled should not work")
	}

	// -- Set false normal
	err = wsTest.Websocket.SetEnabled(false)
	if err != nil {
		t.Fatal("test failed - setting enabled should not work")
	}

	// -- Set true normal
	err = wsTest.Websocket.SetEnabled(true)
	if err != nil {
		t.Fatal("test failed - setting enabled should not work")
	}

	// -- Normal shutdown
	err = wsTest.Websocket.Shutdown()
	if err != nil {
		t.Fatal("test failed - WebsocketSetup", err)
	}

	timer := time.NewTimer(5 * time.Second)
	select {
	case <-comms:
	case <-timer.C:
		t.Fatal("test failed - WebsocketSetup - timeout")
	}
}

func TestInsertingSnapShots(t *testing.T) {
	var snapShot1 orderbook.Base
	asks := []orderbook.Item{
		orderbook.Item{Price: 6000, Amount: 1, ID: 1},
		orderbook.Item{Price: 6001, Amount: 0.5, ID: 2},
		orderbook.Item{Price: 6002, Amount: 2, ID: 3},
		orderbook.Item{Price: 6003, Amount: 3, ID: 4},
		orderbook.Item{Price: 6004, Amount: 5, ID: 5},
		orderbook.Item{Price: 6005, Amount: 2, ID: 6},
		orderbook.Item{Price: 6006, Amount: 1.5, ID: 7},
		orderbook.Item{Price: 6007, Amount: 0.5, ID: 8},
		orderbook.Item{Price: 6008, Amount: 23, ID: 9},
		orderbook.Item{Price: 6009, Amount: 9, ID: 10},
		orderbook.Item{Price: 6010, Amount: 7, ID: 11},
	}

	bids := []orderbook.Item{
		orderbook.Item{Price: 5999, Amount: 1, ID: 12},
		orderbook.Item{Price: 5998, Amount: 0.5, ID: 13},
		orderbook.Item{Price: 5997, Amount: 2, ID: 14},
		orderbook.Item{Price: 5996, Amount: 3, ID: 15},
		orderbook.Item{Price: 5995, Amount: 5, ID: 16},
		orderbook.Item{Price: 5994, Amount: 2, ID: 17},
		orderbook.Item{Price: 5993, Amount: 1.5, ID: 18},
		orderbook.Item{Price: 5992, Amount: 0.5, ID: 19},
		orderbook.Item{Price: 5991, Amount: 23, ID: 20},
		orderbook.Item{Price: 5990, Amount: 9, ID: 21},
		orderbook.Item{Price: 5989, Amount: 7, ID: 22},
	}

	snapShot1.Asks = asks
	snapShot1.Bids = bids
	snapShot1.AssetType = "SPOT"
	snapShot1.CurrencyPair = "BTCUSD"
	snapShot1.LastUpdated = time.Now()
	snapShot1.Pair = pair.NewCurrencyPairFromString("BTCUSD")

	wsTest.Websocket.Orderbook.LoadSnapshot(snapShot1, "ExchangeTest")

	var snapShot2 orderbook.Base
	asks = []orderbook.Item{
		orderbook.Item{Price: 51, Amount: 1, ID: 1},
		orderbook.Item{Price: 52, Amount: 0.5, ID: 2},
		orderbook.Item{Price: 53, Amount: 2, ID: 3},
		orderbook.Item{Price: 54, Amount: 3, ID: 4},
		orderbook.Item{Price: 55, Amount: 5, ID: 5},
		orderbook.Item{Price: 56, Amount: 2, ID: 6},
		orderbook.Item{Price: 57, Amount: 1.5, ID: 7},
		orderbook.Item{Price: 58, Amount: 0.5, ID: 8},
		orderbook.Item{Price: 59, Amount: 23, ID: 9},
		orderbook.Item{Price: 50, Amount: 9, ID: 10},
		orderbook.Item{Price: 60, Amount: 7, ID: 11},
	}

	bids = []orderbook.Item{
		orderbook.Item{Price: 49, Amount: 1, ID: 12},
		orderbook.Item{Price: 48, Amount: 0.5, ID: 13},
		orderbook.Item{Price: 47, Amount: 2, ID: 14},
		orderbook.Item{Price: 46, Amount: 3, ID: 15},
		orderbook.Item{Price: 45, Amount: 5, ID: 16},
		orderbook.Item{Price: 44, Amount: 2, ID: 17},
		orderbook.Item{Price: 43, Amount: 1.5, ID: 18},
		orderbook.Item{Price: 42, Amount: 0.5, ID: 19},
		orderbook.Item{Price: 41, Amount: 23, ID: 20},
		orderbook.Item{Price: 40, Amount: 9, ID: 21},
		orderbook.Item{Price: 39, Amount: 7, ID: 22},
	}

	snapShot2.Asks = asks
	snapShot2.Bids = bids
	snapShot2.AssetType = "SPOT"
	snapShot2.CurrencyPair = "LTCUSD"
	snapShot2.LastUpdated = time.Now()
	snapShot2.Pair = pair.NewCurrencyPairFromString("LTCUSD")

	wsTest.Websocket.Orderbook.LoadSnapshot(snapShot2, "ExchangeTest")

	var snapShot3 orderbook.Base
	asks = []orderbook.Item{
		orderbook.Item{Price: 51, Amount: 1, ID: 1},
		orderbook.Item{Price: 52, Amount: 0.5, ID: 2},
		orderbook.Item{Price: 53, Amount: 2, ID: 3},
		orderbook.Item{Price: 54, Amount: 3, ID: 4},
		orderbook.Item{Price: 55, Amount: 5, ID: 5},
		orderbook.Item{Price: 56, Amount: 2, ID: 6},
		orderbook.Item{Price: 57, Amount: 1.5, ID: 7},
		orderbook.Item{Price: 58, Amount: 0.5, ID: 8},
		orderbook.Item{Price: 59, Amount: 23, ID: 9},
		orderbook.Item{Price: 50, Amount: 9, ID: 10},
		orderbook.Item{Price: 60, Amount: 7, ID: 11},
	}

	bids = []orderbook.Item{
		orderbook.Item{Price: 49, Amount: 1, ID: 12},
		orderbook.Item{Price: 48, Amount: 0.5, ID: 13},
		orderbook.Item{Price: 47, Amount: 2, ID: 14},
		orderbook.Item{Price: 46, Amount: 3, ID: 15},
		orderbook.Item{Price: 45, Amount: 5, ID: 16},
		orderbook.Item{Price: 44, Amount: 2, ID: 17},
		orderbook.Item{Price: 43, Amount: 1.5, ID: 18},
		orderbook.Item{Price: 42, Amount: 0.5, ID: 19},
		orderbook.Item{Price: 41, Amount: 23, ID: 20},
		orderbook.Item{Price: 40, Amount: 9, ID: 21},
		orderbook.Item{Price: 39, Amount: 7, ID: 22},
	}

	snapShot3.Asks = asks
	snapShot3.Bids = bids
	snapShot3.AssetType = "FUTURES"
	snapShot3.CurrencyPair = "LTCUSD"
	snapShot3.LastUpdated = time.Now()
	snapShot3.Pair = pair.NewCurrencyPairFromString("LTCUSD")

	wsTest.Websocket.Orderbook.LoadSnapshot(snapShot3, "ExchangeTest")

	if len(wsTest.Websocket.Orderbook.ob) != 3 {
		t.Error("test failed - inserting orderbook data")
	}
}

func TestUpdate(t *testing.T) {
	LTCUSDPAIR := pair.NewCurrencyPairFromString("LTCUSD")
	BTCUSDPAIR := pair.NewCurrencyPairFromString("BTCUSD")

	bidTargets := []orderbook.Item{
		orderbook.Item{Price: 49, Amount: 24},    // Ammend
		orderbook.Item{Price: 48, Amount: 0},     // Delete
		orderbook.Item{Price: 1337, Amount: 100}, // Append
		orderbook.Item{Price: 1336, Amount: 0},   // Ghost delete
	}

	askTargets := []orderbook.Item{
		orderbook.Item{Price: 51, Amount: 24},    // Ammend
		orderbook.Item{Price: 52, Amount: 0},     // Delete
		orderbook.Item{Price: 1337, Amount: 100}, // Append
		orderbook.Item{Price: 1336, Amount: 0},   // Ghost delete
	}

	err := wsTest.Websocket.Orderbook.Update(bidTargets,
		askTargets,
		LTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"SPOT")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}

	err = wsTest.Websocket.Orderbook.Update(bidTargets,
		askTargets,
		LTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"FUTURES")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}

	bidTargets = []orderbook.Item{
		orderbook.Item{Price: 5999, Amount: 24},  // Ammend
		orderbook.Item{Price: 5998, Amount: 0},   // Delete
		orderbook.Item{Price: 1337, Amount: 100}, // Append
		orderbook.Item{Price: 1336, Amount: 0},   // Ghost delete
	}

	askTargets = []orderbook.Item{
		orderbook.Item{Price: 6000, Amount: 24},  // Ammend
		orderbook.Item{Price: 6001, Amount: 0},   // Delete
		orderbook.Item{Price: 1337, Amount: 100}, // Append
		orderbook.Item{Price: 1336, Amount: 0},   // Ghost delete
	}

	err = wsTest.Websocket.Orderbook.Update(bidTargets,
		askTargets,
		BTCUSDPAIR,
		time.Now(),
		"ExchangeTest",
		"SPOT")

	if err != nil {
		t.Error("test failed - OrderbookUpdate error", err)
	}
}

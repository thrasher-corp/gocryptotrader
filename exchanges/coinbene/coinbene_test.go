package coinbene

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
	spotTestPair            = "BTC/USDT"
	swapTestPair            = "BTCUSDT"
)

var c Coinbene

func TestMain(m *testing.M) {
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	coinbeneConfig, err := cfg.GetExchangeConfig("Coinbene")
	if err != nil {
		log.Fatal(err)
	}
	coinbeneConfig.API.AuthenticatedWebsocketSupport = true
	coinbeneConfig.API.AuthenticatedSupport = true
	coinbeneConfig.API.Credentials.Secret = testAPISecret
	coinbeneConfig.API.Credentials.Key = testAPIKey

	err = c.Setup(coinbeneConfig)
	if err != nil {
		log.Fatal(err)
	}
	c.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	c.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return c.AllowAuthenticatedRequest()
}

func TestGetAllPairs(t *testing.T) {
	t.Parallel()
	_, err := c.GetAllPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := c.GetPairInfo(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderbook(spotTestPair, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetTicker(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetTrades(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAcounntBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountAssetBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountAssetBalance(currency.BTC.String())
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.PlaceSpotOrder(
		1,
		1,
		spotTestPair,
		order.Buy.Lower(),
		order.Limit.Lower(),
		"Sup3rAw3s0m3Cl13ntiDH",
		0,
	)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.PlaceSpotOrders(
		[]PlaceOrderRequest{
			{
				1,
				1,
				spotTestPair,
				order.Buy.Lower(),
				order.Limit.Lower(),
				"Sup3rAw3s0m3Cl13ntiDH",
				0,
			},
		})
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOpenSpotOrders(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchClosedOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchClosedOrders(spotTestPair, "")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchSpotOrderInfo("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSpotOrderFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSpotOrderFills("1912131427156307968")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSpotOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.CancelSpotOrder("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSpotOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSpotOrders([]string{"578639816552972288", "578639902896914432"})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = c.UpdateTicker(cp, asset.PerpetualSwap)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.UpdateAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = c.UpdateOrderbook(cp, asset.PerpetualSwap)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTickers(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTickers()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTicker(swapTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapOrderbook(swapTestPair, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapKlines(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapKlines(swapTestPair,
		"1573184608",
		"1573184808",
		"1")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTrades(swapTestPair, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSwapAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSwapPositions(swapTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceSwapOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.PlaceSwapOrder(swapTestPair,
		order.Buy.Lower(),
		"limit",
		"fixed",
		"12345",
		1,
		1,
		2)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSwapOrder("1337")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOpenOrders(swapTestPair, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOpenOrdersByPage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOpenOrdersByPage(swapTestPair, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderInfo("1337")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderHistory("", "", swapTestPair, 1, 10, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistoryByOrderID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderHistoryByOrderID("", "", swapTestPair, "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSwapOrders([]string{"578639816552972288", "578639902896914432"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderFills(swapTestPair, "5807143157122003", 580714315825905664)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapFundingRates(1, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubscribe(t *testing.T) {
	pressXToJSON := []byte(`{"event":"subscribe","topic":"orderBook.BTCUSDT.10"}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{"event":"unsubscribe","topic":"tradeList.BTCUSDT"}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsLogin(t *testing.T) {
	pressXToJSON := []byte(`{"event":"login","success":true}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{"event":"login","success":false}`)
	err = c.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "orderBook.BTCUSDT", 
    "action": "insert",
    "data": [{
        "asks": [
            ["5621.7", "58", "2"], 
            ["5621.8", "125", "5"],
            ["5621.9", "100", "9"],
            ["5622", "84", "20"],
            ["5623.5", "90", "12"],
            ["5624.2", "1540", "15"],
            ["5625.1", "300",  "20"],
            ["5625.9", "350", "1"],
            ["5629.3", "200", "1"],
            ["5650", "1000", "8"]
        ],
        "bids": [
            ["5621.3", "287","8"],
            ["5621.2", "41","1"],
            ["5621.1", "2","1"],
            ["5621", "26","2"],
            ["5620.8", "194","2"],
            ["5620", "2", "1"],
            ["5618.8", "204","2"],
            ["5618.4", "30", "9"],
            ["5617.2", "2","1"],
            ["5609.9", "100", "12"]
        ],
        "version":1,
        "timestamp": "2019-07-04T02:21:08Z"
    }]
 }`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "topic": "orderBook.BTCUSDT", 
    "action": "update", 
    "data": [{
        "asks": [
            ["5621.7", "50", "2"],
            ["5621.8", "0", "0"],
            ["5621.9", "30", "5"]
        ],
        "bids": [
            ["5621.3", "10","1"],
            ["5621.2", "20","1"],
            ["5621.1", "80","5"],
            ["5621", "0","0"],
            ["5620.8", "10","1"]
        ],
        "version":2,
        "timestamp": "2019-07-04T02:21:09Z"
    }]
 }`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "tradeList.BTCUSDT",
    "data": [  
      [
        "8600.0000", 
        "s", 
        "100", 
        "2019-05-21T08:25:22.735Z"
      ]
    ]
 }`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "ticker.BTCUSDT",
    "data": [
        {
          "symbol": "BTCUSDT",
          "lastPrice": "8548.0", 
          "markPrice": "8548.0", 
          "bestAskPrice": "8601.0", 
          "bestBidPrice": "8600.0",
          "bestAskVolume": "1222", 
          "bestBidVolume": "56505",
          "high24h": "8600.0000", 
          "low24h": "242.4500", 
          "volume24h": "4994", 
          "timestamp": "2019-05-06T06:45:56.716Z"
        }
    ]
 }`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKLine(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "kline.BTCUSDT",
    "data": [
        [
          "BTCUSDT",
          1557428280,
          "5794",
          "5794",
          "5794",
          "5794",
          "0",
          "0",
          "0",
          "0"
        ]
    ]
 }`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUserAccount(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "user.account",
    "data": [{
        "asset": "BTC",
        "availableBalance": "20.3859", 
        "frozenBalance": "0.7413",
        "balance": "21.1272", 
        "timestamp": "2019-05-22T03:11:22.0Z"
    }]
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUserPosition(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "user.position",
    "data": [{
      "availableQuantity": "100", 
      "avgPrice": "7778.1", 
      "leverage": "20", 
      "liquidationPrice": "5441.0", 
      "markPrice": "8086.5", 
      "positionMargin": "0.0285",  
      "quantity": "507", 
      "realisedPnl": "0.0069", 
      "side": "long", 
      "symbol": "BTCUSDT",
      "marginMode": "1",
      "createTime": "2019-05-22T03:11:22.0Z"
    }]
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUserOrder(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "user.order",
    "data": [{
      "orderId": "580721369818955776", 
      "direction": "openLong", 
      "leverage": "20", 
      "symbol": "BTCUSDT", 
      "orderType": "limit", 
      "quantity": "7", 
      "orderPrice": "146.30", 
      "orderValue": "0.0010", 
      "fee": "0.0000", 
      "filledQuantity": "0", 
      "averagePrice": "0.00", 
      "orderTime": "2019-05-22T03:39:24.0Z", 
      "status": "new",
      "lastFillQuantity": "0",
      "lastFillPrice": "0",
      "lastFillTime": ""
    }]
}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

package coinbene

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
	spotTestPair            = "BTC/USDT"
	swapTestPair            = "BTC-SWAP"
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
	c.Websocket = sharedtestvalues.NewTestWebsocket()
	err = c.Setup(coinbeneConfig)
	if err != nil {
		log.Fatal(err)
	}
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
	_, err := c.GetTrades(spotTestPair, 100)
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
	cp, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	cp, err = currency.NewPairFromString(swapTestPair)
	if err != nil {
		t.Fatal(err)
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
	_, err := c.UpdateAccountInfo(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	cp, err = currency.NewPairFromString(swapTestPair)
	if err != nil {
		t.Fatal(err)
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

func TestGetKlines(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetKlines(p.String(),
		time.Now().Add(-time.Hour*1), time.Now(), "1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSwapKlines(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromString(swapTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetSwapKlines(p.String(),
		time.Now().Add(-time.Hour*1), time.Now(), "1")
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

func TestGetSwapInstruments(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapInstruments()
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
	if err == nil {
		t.Error("error cannot be nil as this will initiate an auth subscription")
	}

	pressXToJSON = []byte(`{"event":"login","success":false}`)
	err = c.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{"topic":"spot/orderBook.BTCUSDT","action":"insert","data":[{"bids":[["0.00000015","174215.91"],["0.00000012","600000.00"],["0.00000010","10000.00"],["0.00000006","33333.00"],["0.00000004","50000.00"],["0.00000003","2000000.00"],["0.00000002","100000.00"],["0.00000001","1100000.00"]],"asks":[["0.00000262","5152.79"],["0.00000263","44626.00"],["0.00000340","2649.85"],["0.00000398","20056.93"],["0.00000400","1420385.54"],["0.00000790","8594.85"],["0.00000988","42380.97"],["0.00000997","43850.97"],["0.00001398","10541.59"],["0.00001400","3409.29"],["0.00002636","52.11"],["0.00002810","2543.66"],["0.00003200","1018.36"],["0.00004999","19.81"],["0.00005000","400.00"],["0.00005898","4060.56"],["0.00006498","3302.60"],["0.00006668","4060.56"],["0.00008000","400.00"]],"version":4915,"timestamp":1598529668288}]}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{"topic":"spot/orderBook.BTCUSDT","action":"update","data":[{"bids":[["2.983","8696"]],"asks":[["3.113","0"]],"version":34600866,"timestamp":1598587478738}]}`)
	err = c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{"data":[["0.00000050","s","37.00",1598500505000],["0.00000060","s","10.00",1598499782000],["0.00000066","s","1.00",1598499782000],["0.00000067","s","1.00",1598499782000],["0.00000068","s","1.00",1598499745000],["0.00000080","b","1.00",1598262728000],["0.00000089","b","5592.81",1597738441000],["0.00000072","b","1.00",1597693134000],["0.00000069","s","21739.13",1597378140000],["0.00000069","s","1.00",1597378140000],["0.00000074","b","1.00",1597354497000],["0.00000079","b","1.00",1597325675000],["0.00000082","b","1.00",1597162162000],["0.00000089","b","1.00",1597084892000],["0.00000073","b","109404.43",1597015827000],["0.00000070","b","1067.00",1597015827000],["0.00000070","b","1.00",1594732841000],["0.00000070","b","10.00",1592178569000],["0.00000065","b","194.76",1592178545000],["0.00000064","b","2.37",1592105641000],["0.00000064","b","3.00",1592087828000],["0.00000045","b","5.00",1592087828000],["0.00000045","b","5.00",1592004274000],["0.00000030","s","100.00",1591931268000],["0.00000020","b","138.12",1591928623000],["0.00000020","b","55027.66",1591928623000],["0.00000020","b","59880.11",1591572812000],["0.00000021","s","138.12",1590413750000],["0.00000021","s","5.37",1590413750000],["0.00000056","s","1.00",1589567228000],["0.00000065","b","1.00",1589567217000],["0.00000060","b","84890.64",1589407481000],["0.00000060","b","17.13",1589407433000],["0.00000060","b","9148.70",1589389270000],["0.00000059","b","9010.00",1589389159000],["0.00000055","b","3876.00",1589389098000],["0.00000055","b","30000.00",1588899981000],["0.00000055","b","5724.00",1588891192000],["0.00000050","b","400.00",1588891192000],["0.00000048","b","26874.64",1588891129000],["0.00000048","b","2.00",1588891129000],["0.00000049","b","7547.75",1585279296000],["0.00000049","b","12180.30",1584932828000],["0.00000049","b","8256.95",1584932828000],["0.00000053","b","500.42",1583351500000],["0.00000053","b","500.00",1583351484000],["0.00000053","b","400.00",1583351470000],["0.00000053","b","394.62",1583351455000],["0.00000053","b","1.99",1583343633000],["0.00000018","s","250.00",1583338813000]],"topic":"spot/tradeList.BTCUSDT","action":"insert"}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{"topic": "spot/ticker.BTCUSDT","action":"insert","data": [{"symbol":"BTCUSDT","lastPrice":"23.3746","bestAskPrice":"23.3885","bestBidPrice":"23.3603","high24h":"23.5773","open24h":"22.1961","openPrice":"22.5546","low24h":"21.8077","volume24h":"3784807.9709","timestamp":1598587472634}]}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKLine(t *testing.T) {
	pressXToJSON := []byte(`{"topic": "spot/kline.BTCUSDT.1h","action":"insert","data": [{"t":1594990800,"o":1.1e-07,"h":1.1e-07,"l":1.1e-07,"c":1.1e-07,"v":0}]}`)
	err := c.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUserAccount(t *testing.T) {
	pressXToJSON := []byte(`{
    "topic": "btc/user.account",
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

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	_, err = c.GetHistoricCandles(currencyPair, asset.Spot, startTime, time.Now(), kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}

	currencyPairSwap, err := currency.NewPairFromString(swapTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetHistoricCandles(currencyPairSwap, asset.PerpetualSwap, startTime, time.Now(), kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 2)
	_, err = c.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, time.Now(), kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"1",
		},
		{
			"OneHour",
			kline.OneHour,
			"60",
		},
		{
			"OneDay",
			kline.OneDay,
			"D",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"W",
		},
		{
			"OneMonth",
			kline.OneMonth,
			"M",
		},
		{
			"AllOther",
			kline.TwoWeek,
			"",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := c.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestInferAssetFromTopic(t *testing.T) {
	a := inferAssetFromTopic("spot/orderBook.BSVBTC")
	if a != asset.Spot {
		t.Error("expected spot")
	}
	a = inferAssetFromTopic("btc/orderBook.BSVBTC")
	if a != asset.PerpetualSwap {
		t.Error("expected PerpetualSwap")
	}
	a = inferAssetFromTopic("usdt/orderBook.BSVBTC")
	if a != asset.PerpetualSwap {
		t.Error("expected PerpetualSwap")
	}
	a = inferAssetFromTopic("orderBook.BSVBTC")
	if a != asset.PerpetualSwap {
		t.Error("expected PerpetualSwap")
	}
	a = inferAssetFromTopic("")
	if a != asset.PerpetualSwap {
		t.Error("expected PerpetualSwap")
	}
}

func TestGetCurrencyFromWsTopic(t *testing.T) {
	p, err := c.getCurrencyFromWsTopic(asset.Spot, "spot/orderbook.BTCUSDT")
	if err != nil {
		t.Error(err)
	}
	if p.Base.String() != "BTC" && p.Quote.String() != "USDT" {
		t.Errorf("unexpected currency, wanted BTCUSD, received %v", p.String())
	}

	_, err = c.getCurrencyFromWsTopic(asset.Spot, "fake")
	if err != nil && err.Error() != "no currency found in topic fake" {
		t.Error(err)
	}

	_, err = c.getCurrencyFromWsTopic(asset.Spot, "hello.moto")
	if err != nil && err.Error() != "currency moto not found in supplied pairs" {
		t.Error(err)
	}

	_, err = c.getCurrencyFromWsTopic(asset.Spot, "spot/kline.GOM2USDT.1h")
	if err != nil && err.Error() != "currency moto not found in enabled pairs" {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(spotTestPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

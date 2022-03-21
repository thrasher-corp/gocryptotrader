package bybit

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey    = ""
	apiSecret = ""
)

var by Bybit

func TestMain(m *testing.M) {
	by.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = false
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	by.Websocket = sharedtestvalues.NewTestWebsocket()
	err = by.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	by.Verbose = true

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return by.ValidateAPICredentials()
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := by.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = by.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

// test cases for SPOT

func TestGetAllPairs(t *testing.T) {
	t.Parallel()
	_, err := by.GetAllPairs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := by.GetOrderBook(context.Background(), "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := by.GetTrades(context.Background(), "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := by.GetKlines(context.Background(), "BTCUSDT", "5m", 2000, time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet24HrsChange(t *testing.T) {
	t.Parallel()
	_, err := by.Get24HrsChange(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.Get24HrsChange(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLastTradedPrice(t *testing.T) {
	t.Parallel()
	_, err := by.GetLastTradedPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLastTradedPrice(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetBestBidAskPrice(t *testing.T) {
	t.Parallel()
	_, err := by.GetBestBidAskPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetBestBidAskPrice(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePostOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.CreatePostOrder(context.Background(), &PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Quantity:    1,
		Side:        "BUY",
		TradeType:   "LIMIT",
		TimeInForce: "GTC",
		Price:       100,
		OrderLinkID: "linkID",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.QueryOrder(context.Background(), "0", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.CancelExistingOrder(context.Background(), "", "linkID")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.BatchCancelOrder(context.Background(), "", "BUY", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.ListOpenOrders(context.Background(), "BTCUSDT", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestListPastOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.ListPastOrders(context.Background(), "BTCUSDT", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetTradeHistory(context.Background(), "", 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetWalletBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

// test cases for WS SPOT

func TestWsSubscription(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
		"symbol": "BTCUSDT",
		"event": "sub",
		"topic": "trade",
		"params": {
			"binary": false
		}
	}`)
	err := by.wsHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
		"symbol":"BTCUSDT",
		"event": "cancel",
		"topic":"trade",
		"params": {
			"binary": false
		}
	}`)
	err := by.wsHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsTrade(t *testing.T) {
	t.Parallel()
	by.SetSaveTradeDataStatus(true)

	pressXToJSON := []byte(`{
		"topic": "trade",
		"params": {
			"symbol": "BTCUSDT",
			"binary": "false",
			"symbolName": "BTCUSDT"
		},
		"data": {
			"v": "564265886622695424",
			"t": 1582001735462,
			"p": "9787.5",
			"q": "0.195009",
			"m": true
		}
	}`)
	err := by.wsHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsOrderbook(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
		"topic": "depth",
		"params": {
		  "symbol": "BTCUSDT",
		  "binary": "false",
		  "symbolName": "BTCUSDT"
		},
		"data": {
			"s": "BTCUSDT",
			"t": 1582001376853,
			"v": "13850022_2",
			"b": [
				[
					"9780.79",
					"0.01"
				],
				[
					"9780.5",
					"0.1"
				],
				[
					"9780.4",
					"0.517813"
				]
			],
			"a": [
				[
					"9781.21",
					"0.042842"
				],
				[
					"9782",
					"0.3"
				],
				[
					"9782.1",
					"0.226"
				]
			]
		}
	}`)
	err := by.wsHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsTicker(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
		"topic": "bookTicker",
		"params": {
			"symbol": "BTCUSDT",
			"binary": "false",
			"symbolName": "BTCUSDT"
		},
		"data": {
			"symbol": "BTCUSDT",
			"bidPrice": "9797.79",
			"bidQty": "0.177976",
			"askPrice": "9799",
			"askQty": "0.65",
			"time": 1582001830346
		}
	}`)
	err := by.wsHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

// test cases for CoinMarginedFutures

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = by.GetFuturesOrderbook(context.Background(), pair)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = by.GetFuturesKlineData(context.Background(), pair, "M", 5, time.Time{})
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetFuturesKlineData(context.Background(), pair, "60", 5, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetFuturesSymbolPriceTicker(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetPublicTrades(context.Background(), pair, 0, 0)
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetPublicTrades(context.Background(), pair, 0, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSymbolsInfo(t *testing.T) {
	t.Parallel()
	_, err := by.GetSymbolsInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetMarkPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetIndexPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetPremiumIndexPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetPremiumIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetOpenInterest(context.Background(), pair, "5min", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestBigDeal(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLatestBigDeal(context.Background(), pair, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountRatio(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetAccountRatio(context.Background(), pair, "1d", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRiskLimit(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetRiskLimit(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastFundingRate(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLastFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := by.GetFuturesServerTime(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAnnouncement(t *testing.T) {
	t.Parallel()
	_, err := by.GetAnnouncement(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateCoinFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateCoinFuturesOrder(context.Background(), pair, "Buy", "Limit", "GTC", "", "", "", 1, 1, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveCoinFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetActiveCoinFuturesOrders(context.Background(), pair, "Filled", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelActiveCoinFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllActiveCoinFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceActiveCoinFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveRealtimeCoinOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveRealtimeCoinOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalCoinFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateConditionalCoinFuturesOrder(context.Background(), pair, "Buy", "Limit", "GTC", "", "", "", "", 1, 0.5, 0, 0, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalCoinFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelConditionalCoinFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllConditionalCoinFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceConditionalCoinFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalRealtimeCoinOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalRealtimeCoinOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetCoinPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetCoinMargin(context.Background(), pair, "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinTradingAndStop(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetCoinTradingAndStop(context.Background(), pair, 0, 0, 0, 0, 0, 0, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetCoinLeverage(context.Background(), pair, 10, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinTradeRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetCoinTradeRecords(context.Background(), pair, "", "", 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedCoinTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetClosedCoinTrades(context.Background(), pair, "", 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeCoinMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ChangeCoinMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeCoinMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	err = by.ChangeCoinMargin(context.Background(), pair, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetCoinRiskLimit(context.Background(), pair, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinLastFundingFee(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetCoinLastFundingFee(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinPredictedFundingRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = by.GetCoinPredictedFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetAPIKeyInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetLCPInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLiquidityContributionPointsInfo(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureWalletBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetFutureWalletBalance(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletFundRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetWalletFundRecords(context.Background(), "2021-09-11", "2021-10-09", "ETH", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletWithdrawalRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetWalletWithdrawalRecords(context.Background(), "2021-09-11", "2021-10-09", "ETH", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetExchangeRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := by.GetAssetExchangeRecords(context.Background(), "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

// test cases for USDTMarginedFutures

func TestGetUSDTFuturesKlineData(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = by.GetUSDTFuturesKlineData(context.Background(), pair, "M", 5, time.Time{})
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetUSDTFuturesKlineData(context.Background(), pair, "60", 5, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTPublicTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTPublicTrades(context.Background(), pair, 1000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTMarkPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTMarkPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTIndexPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTPremiumIndexPriceKline(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTPremiumIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTLastFundingRate(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTLastFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTRiskLimit(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTRiskLimit(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUSDTFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateUSDTFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", 1, 1, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveUSDTFuturesOrders(context.Background(), pair, "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetActiveUSDTFuturesOrders(context.Background(), pair, "Filled", "", "", "", 0, 50)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelActiveUSDTFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllActiveUSDTFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceActiveUSDTFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveUSDTRealtimeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveUSDTRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalUSDTFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateConditionalUSDTFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", "", 1, 0.5, 0, 0, 1, 1, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalUSDTFuturesOrders(context.Background(), pair, "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelConditionalUSDTFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllConditionalUSDTFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceConditionalUSDTFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalUSDTRealtimeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalUSDTRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetAutoAddMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = by.SetAutoAddMargin(context.Background(), pair, true, "Sell")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeUSDTMarginn(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = by.ChangeUSDTMargin(context.Background(), pair, 1, 1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeUSDTMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ChangeUSDTMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetUSDTMargin(context.Background(), pair, "Buy", "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = by.SetUSDTLeverage(context.Background(), pair, 10, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTTradingAndStop(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = by.SetUSDTTradingAndStop(context.Background(), pair, 0, 0, 0, 0, 0, "Buy", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTTradeRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetUSDTTradeRecords(context.Background(), pair, "", 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedUSDTTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetClosedUSDTTrades(context.Background(), pair, "", 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetUSDTRiskLimit(context.Background(), pair, "Buy", 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPredictedUSDTFundingRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = by.GetPredictedUSDTFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastUSDTFundingFee(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLastUSDTFundingFee(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

// test cases for Futures

func TestCreateFuturesOrderr(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateFuturesOrder(context.Background(), 1, pair, "Buy", "Market", "GoodTillCancel", "", "", "", 10, 1, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = by.GetActiveFuturesOrders(context.Background(), pair, "Filled", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelActiveFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllActiveFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceActiveFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveRealtimeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetActiveRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CreateConditionalFuturesOrder(context.Background(), 0, pair, "Buy", "Limit", "GTC", "", "", "", "", 1, 0.5, 0, 0, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelConditionalFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.CancelAllConditionalFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ReplaceConditionalFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalRealtimeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetConditionalRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetMargin(context.Background(), 0, pair, "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetTradingAndStop(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetTradingAndStop(context.Background(), 0, pair, 0, 0, 0, 0, 0, 0, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetLeverage(context.Background(), pair, 10, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	err = by.ChangePositionMode(context.Background(), pair, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.ChangeMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	err = by.ChangeMargin(context.Background(), pair, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetTradeRecords(context.Background(), pair, "", "", 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetClosedTrades(context.Background(), pair, "", 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	pair, err := currency.NewPairFromString("BTCUSDH22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.SetRiskLimit(context.Background(), pair, 2, 0)
	if err != nil {
		t.Error(err)
	}
}

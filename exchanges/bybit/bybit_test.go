package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b = &Bybit{}

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	if apiKey != "" && apiSecret != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
		b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	b.Websocket = sharedtestvalues.NewTestWrapperWebsocket()
	request.MaxRequestJobs = 200
	err = b.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	err = b.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		log.Fatal(err)
	}

	// Turn on all pairs for testing
	supportedAssets := b.GetAssetTypes(false)
	for x := range supportedAssets {
		avail, err := b.GetAvailablePairs(supportedAssets[x])
		if err != nil {
			log.Fatal(err)
		}

		err = b.CurrencyPairs.StorePairs(supportedAssets[x], avail, true)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := b.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = b.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

// test cases for SPOT

func TestGetAllSpotPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllSpotPairs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(context.Background(), "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMergedOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetMergedOrderBook(context.Background(), "BTCUSDT", 2, 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(context.Background(), "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetKlines(context.Background(), "BTCUSDT", "5m", 2000, time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet24HrsChange(t *testing.T) {
	t.Parallel()
	_, err := b.Get24HrsChange(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.Get24HrsChange(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLastTradedPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetLastTradedPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetLastTradedPrice(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetBestBidAskPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetBestBidAskPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetBestBidAskPrice(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePostOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.CreatePostOrder(context.Background(), &PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Quantity:    1,
		Side:        "Buy",
		TradeType:   "LIMIT",
		TimeInForce: "GTC",
		Price:       100,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders) // Note: here !canManipulateRealOrders added as we don't have orderID

	_, err := b.QueryOrder(context.Background(), "0", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.CancelExistingOrder(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.BatchCancelOrder(context.Background(), "", "Buy", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFastCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.FastCancelExistingOrder(context.Background(), "BTCUSDT", "889208273689997824", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.FastCancelExistingOrder(context.Background(), "BTCUSDT", "", "162081160171552")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchFastCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.BatchFastCancelOrder(context.Background(), "BTCUSDT", "Buy", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.BatchFastCancelOrder(context.Background(), "BTCUSDT", "", "Limit")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchCancelOrderByIDs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.BatchCancelOrderByIDs(context.Background(), []string{"889208273689997824", "889208273689997825"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestListOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.ListOpenOrders(context.Background(), "BTCUSDT", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPastOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetPastOrders(context.Background(), "BTCUSDT", "", 0, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetTradeHistory(context.Background(), 0, "", "", "", "", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWalletBalance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSpotServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotServerTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetDepositAddressForCurrency(context.Background(), currency.BTC.String())
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithdrawFund(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.WithdrawFund(context.Background(), currency.ETH.String(), currency.ETH.String(), "0xEA13A385BcB74e631AAF1B424d7a01c61bF27Fe0", "", "10")
	if err != nil && err.Error() != "Withdraw address chain or destination tag are not equal" {
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
	err := b.wsSpotHandleData(pressXToJSON)
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
	err := b.wsSpotHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsTrade(t *testing.T) {
	t.Parallel()
	b.SetSaveTradeDataStatus(true)

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
	err := b.wsSpotHandleData(pressXToJSON)
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
	err := b.wsSpotHandleData(pressXToJSON)
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
	err := b.wsSpotHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsKline(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
		"topic": "kline",
		"params": {
		 	"symbol": "BTCUSDT",
		  	"binary": "false",
		  	"klineType": "1m",
		  	"symbolName": "BTCUSDT"
		},
		"data": {
		  	"t": 1582001880000,
		  	"s": "BTCUSDT",
		  	"sn": "BTCUSDT",
		  	"c": "9799.4",
		  	"h": "9801.4",
		  	"l": "9798.91",
		  	"o": "9799.4",
		  	"v": "15.917433"
		}
	}`)
	err := b.wsSpotHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWSAccountInfo(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`[{
		"e":"outboundAccountInfo",
		"E":"1629969654753",
		"T":true,
		"W":true,
		"D":true,
		"B":[{
			"a":"BTC",
			"f":"10000000097.1982823144",
			"l":"0"
		}]
	}]`)
	err := b.wsSpotHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWSOrderExecution(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`[{
		"e": "executionReport",
		"E": "1499405658658",
		"s": "BTCUSDT",
		"c": "1000087761",
		"S": "BUY",
		"o": "LIMIT",
		"f": "GTC",
		"q": "1.00000000",
		"p": "0.10264410",
		"X": "NEW",
		"i": "4293153",
		"M": "0",
		"l": "0.00000000",
		"z": "0.00000000",
		"L": "0.00000000",
		"n": "0",
		"N": "BTC",
		"u": true,
		"w": true,
		"m": false,
		"O": "1499405658657",
		"Z": "473.199",
		"A": "0",
		"C": false,
		"v": "0"
	}]`)
	err := b.wsSpotHandleData(pressXToJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWSTickerInfo(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`[{
		"e":"ticketInfo",
		"E":"1621912542359",
		"s":"BTCUSDT",
		"q":"0.001639",
		"t":"1621912542314",
		"p":"61000.0",
		"T":"899062000267837441",
		"o":"899048013515737344",
		"c":"1621910874883",
		"O":"899062000118679808",
		"a":"10043",
		"A":"10024",
		"m":true,
		"S":"BUY"
	}]`)
	err := b.wsSpotHandleData(pressXToJSON)
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
	_, err = b.GetFuturesOrderbook(context.Background(), pair)
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
	_, err = b.GetFuturesKlineData(context.Background(), pair, "M", 5, time.Time{})
	if !errors.Is(err, errInvalidStartTime) {
		t.Errorf("received: %s, expected: %s", err, errInvalidStartTime)
	}

	_, err = b.GetFuturesKlineData(context.Background(), pair, "60", 5, time.Unix(1577836800, 0))
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
	_, err = b.GetFuturesSymbolPriceTicker(context.Background(), pair)
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

	_, err = b.GetPublicTrades(context.Background(), pair, 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetPublicTrades(context.Background(), pair, 10000)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSymbolsInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetSymbolsInfo(context.Background())
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

	_, err = b.GetMarkPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetPremiumIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetOpenInterest(context.Background(), pair, "5min", 0)
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

	_, err = b.GetLatestBigDeal(context.Background(), pair, 0)
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

	_, err = b.GetAccountRatio(context.Background(), pair, "1d", 0)
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

	_, err = b.GetRiskLimit(context.Background(), pair)
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

	_, err = b.GetLastFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesServerTime(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAnnouncement(t *testing.T) {
	t.Parallel()
	_, err := b.GetAnnouncement(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateCoinFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateCoinFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", 1, 20000, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveCoinFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetActiveCoinFuturesOrders(context.Background(), pair, "Filled", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelActiveCoinFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllActiveCoinFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceActiveCoinFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveRealtimeCoinOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveRealtimeCoinOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalCoinFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateConditionalCoinFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", "", 1, 20000, 0, 0, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalCoinFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelConditionalCoinFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllConditionalCoinFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalCoinFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceConditionalCoinFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalRealtimeCoinOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalRealtimeCoinOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetCoinPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetCoinMargin(context.Background(), pair, "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinTradingAndStop(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetCoinTradingAndStop(context.Background(), pair, 0, 0, 0, 0, 0, 0, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSetCoinLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetCoinLeverage(context.Background(), pair, 10, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinTradeRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetCoinTradeRecords(context.Background(), pair, "", "", 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedCoinTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetClosedCoinTrades(context.Background(), pair, "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeCoinMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ChangeCoinMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeCoinMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	err = b.ChangeCoinMargin(context.Background(), pair, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingFeeRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	feeRate, err := b.GetTradingFeeRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}

	if feeRate.MakerFeeRate == 0 && feeRate.TakerFeeRate == 0 {
		t.Error("expected fee rate")
	}

	if feeRate.UserID == 0 {
		t.Error("expected user id")
	}
}

func TestSetCoinRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetCoinRiskLimit(context.Background(), pair, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinLastFundingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetCoinLastFundingFee(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinPredictedFundingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = b.GetCoinPredictedFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetAPIKeyInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetLCPInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetLiquidityContributionPointsInfo(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetFutureWalletBalance(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletFundRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWalletFundRecords(context.Background(), "2021-09-11", "2021-10-09", "ETH", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletWithdrawalRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWalletWithdrawalRecords(context.Background(), "2021-09-11", "2021-10-09", "", currency.ETH, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetExchangeRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetAssetExchangeRecords(context.Background(), "", 0, 0)
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
	_, err = b.GetUSDTFuturesKlineData(context.Background(), pair, "M", 5, time.Time{})
	if !errors.Is(err, errInvalidStartTime) {
		t.Errorf("received: %s, expected: %s", err, errInvalidStartTime)
	}

	_, err = b.GetUSDTFuturesKlineData(context.Background(), pair, "60", 5, time.Unix(1577836800, 0))
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

	_, err = b.GetUSDTPublicTrades(context.Background(), pair, 1000)
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

	_, err = b.GetUSDTMarkPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetUSDTIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetUSDTPremiumIndexPriceKline(context.Background(), pair, "D", 0, time.Unix(1577836800, 0))
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

	_, err = b.GetUSDTLastFundingRate(context.Background(), pair)
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

	_, err = b.GetUSDTRiskLimit(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUSDTFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateUSDTFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", 1, 10000, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveUSDTFuturesOrders(context.Background(), pair, "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetActiveUSDTFuturesOrders(context.Background(), pair, "Filled", "", "", "", 0, 50)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelActiveUSDTFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllActiveUSDTFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceActiveUSDTFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveUSDTRealtimeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveUSDTRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalUSDTFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateConditionalUSDTFuturesOrder(context.Background(), pair, "Buy", "Limit", "GoodTillCancel", "", "", "", "", 1, 0.5, 0, 0, 1, 1, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalUSDTFuturesOrders(context.Background(), pair, "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelConditionalUSDTFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllConditionalUSDTFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalUSDTFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceConditionalUSDTFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalUSDTRealtimeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalUSDTRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}

	expectedErr := "Order not exists"
	_, err = b.GetConditionalUSDTRealtimeOrders(context.Background(), pair, "1234", "")
	if err != nil && err.Error() != expectedErr {
		t.Error(err)
	}

	_, err = b.GetConditionalUSDTRealtimeOrders(context.Background(), pair, "", "1234")
	if err != nil && err.Error() != expectedErr {
		t.Error(err)
	}
}

func TestGetUSDTPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDTPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetUSDTPositions(context.Background(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestSetAutoAddMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetAutoAddMargin(context.Background(), pair, true, "Sell")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeUSDTMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.ChangeUSDTMargin(context.Background(), pair, 1, 1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.SwitchPositionMode(context.Background(), pair, "BothSide")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeUSDTMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ChangeUSDTMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetUSDTMargin(context.Background(), pair, "Buy", "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetUSDTLeverage(context.Background(), pair, 10, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTTradingAndStop(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetUSDTTradingAndStop(context.Background(), pair, 0, 0, 0, 0, 0, "Buy", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDTTradeRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDTTradeRecords(context.Background(), pair, "", 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedUSDTTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetClosedUSDTTrades(context.Background(), pair, "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDTRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetUSDTRiskLimit(context.Background(), pair, "Buy", 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPredictedUSDTFundingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = b.GetPredictedUSDTFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastUSDTFundingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetLastUSDTFundingFee(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

// test cases for Futures

func TestCreateFuturesOrderr(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateFuturesOrder(context.Background(), 1, pair, "Buy", "Market", "GoodTillCancel", "", "", "", 10, 1, 0, 0, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetActiveFuturesOrders(context.Background(), pair, "Filled", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelActiveFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllActiveFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceActiveFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceActiveFuturesOrders(context.Background(), pair, "3bd1844f-f3c0-4e10-8c25-10fea03763f6", "", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveRealtimeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateConditionalFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CreateConditionalFuturesOrder(context.Background(), 0, pair, "Buy", "Limit", "GoodTillCancel", "", "", "", "", 1, 0.5, 0, 0, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalFuturesOrders(context.Background(), pair, "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelConditionalFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllConditionalFuturesOrders(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceConditionalFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ReplaceConditionalFuturesOrders(context.Background(), pair, "c1025629-e85b-4c26-b4f3-76e86ad9f8c", "", "", "", 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetConditionalRealtimeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetConditionalRealtimeOrders(context.Background(), pair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetPositions(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetMargin(context.Background(), 0, pair, "10")
	if err != nil {
		t.Error(err)
	}
}

func TestSetTradingAndStop(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetTradingAndStop(context.Background(), 0, pair, 0, 0, 0, 0, 0, 0, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetLeverage(context.Background(), pair, 10, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	err = b.ChangePositionMode(context.Background(), pair, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ChangeMode(context.Background(), pair, "Partial")
	if err != nil {
		t.Error(err)
	}
}

func TestChangeMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	err = b.ChangeMargin(context.Background(), pair, 1, 1, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetTradeRecords(context.Background(), pair, "", "", 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetClosedTrades(context.Background(), pair, "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetRiskLimit(context.Background(), pair, 2, 0)
	if err != nil {
		t.Error(err)
	}
}

// Miscellaneous

func TestTimeSecUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tInSec := time.Now().Unix()

	var ts bybitTime
	err := ts.UnmarshalJSON([]byte(strconv.FormatInt(tInSec, 10)))
	if err != nil {
		t.Fatal(err)
	}

	if !time.Unix(tInSec, 0).Equal(ts.Time()) {
		t.Errorf("TestTimeSecUnmarshalJSON failed")
	}
}

func TestTimeMilliSecUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tInMilliSec := time.Now().UnixMilli()

	var tms bybitTime
	err := tms.UnmarshalJSON([]byte(strconv.FormatInt(tInMilliSec, 10)))
	if err != nil {
		t.Fatal(err)
	}

	if !time.UnixMilli(tInMilliSec).Equal(tms.Time()) {
		t.Errorf("TestTimeMilliSecUnmarshalJSON failed")
	}
}

func TestTimeNanoSecUnmarshalJSON(t *testing.T) {
	t.Parallel()
	bybitTimeInst := &struct {
		Timestamp bybitTime `json:"ts"`
	}{}
	data1 := `{ "ts" : "1685523612777"}`
	resultTime := time.UnixMilli(1685523612777)
	err := json.Unmarshal([]byte(data1), bybitTimeInst)
	if err != nil {
		t.Fatal(err)
	} else if !bybitTimeInst.Timestamp.Time().Equal(resultTime) {
		t.Errorf("found %v, but expected %v", bybitTimeInst.Timestamp.Time(), resultTime)
	}

	data2 := `{ "ts" : "1685523612"}`
	resultTime = time.Unix(1685523612, 0)
	err = json.Unmarshal([]byte(data2), bybitTimeInst)
	if err != nil {
		t.Fatal(err)
	} else if !bybitTimeInst.Timestamp.Time().Equal(resultTime) {
		t.Errorf("found %v, but expected %v", bybitTimeInst.Timestamp.Time(), resultTime)
	}
	data3 := `{ "ts" : ""}`
	resultTime = time.Time{}
	err = json.Unmarshal([]byte(data3), bybitTimeInst)
	if err != nil {
		t.Fatal(err)
	} else if !bybitTimeInst.Timestamp.Time().Equal(resultTime) {
		t.Errorf("found %v, but expected %v", bybitTimeInst.Timestamp.Time(), resultTime)
	}
	data4 := `{ "ts" : "1685523612781790000"}`
	resultTime = time.Unix((int64(1685523612781790000) / 1e9), int64(1685523612781790000)%1e9)
	err = json.Unmarshal([]byte(data4), bybitTimeInst)
	if err != nil {
		t.Fatal(err)
	} else if bybitTimeInst.Timestamp.Time().UnixMilli() != resultTime.UnixMilli() {
		t.Errorf("found %v, but expected %v", bybitTimeInst.Timestamp.Time(), resultTime)
	}
	data5 := `{ "ts" : 1685523612777}`
	resultTime = time.UnixMilli(1685523612777)
	err = json.Unmarshal([]byte(data5), bybitTimeInst)
	if err != nil {
		t.Fatal(err)
	} else if !bybitTimeInst.Timestamp.Time().Equal(resultTime) {
		t.Errorf("found %v, but expected %v", bybitTimeInst.Timestamp.Time(), resultTime.String())
	}
	data6 := `{ "ts" : "abcdef"}`
	err = json.Unmarshal([]byte(data6), bybitTimeInst)
	if err == nil {
		t.Errorf("expecting an error, but got nil")
	}
}

// test cases for Wrapper
func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateTicker(context.Background(), pair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), pair1, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	// Futures update dynamically, so fetch the available tradable futures for this test
	availPairs, err := b.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	// Needs to be set before calling extractCurrencyPair
	if err = b.SetPairs(availPairs, asset.Futures, true); err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), availPairs[0], asset.Futures)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), pair3, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.Futures)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair1, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchTradablePairs(context.Background(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchTradablePairs(context.Background(), asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchTradablePairs(context.Background(), asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		t.Error(err)
	}

	err = b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair1, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair1, asset.Futures)
	if err != nil {
		t.Error(err)
	}

	pair2, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair2, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	end := time.Now()
	start := end.AddDate(0, 0, -3)

	_, err = b.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), pair, asset.USDTMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), pair1, asset.CoinMarginedFutures, kline.OneHour, start, end)
	if err != nil {
		t.Error(err)
	}

	enabled, err := b.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), enabled[0], asset.Futures, kline.OneHour, start, end)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), pair3, asset.USDCMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair, asset.USDTMarginedFutures, kline.OneMin, startTime, end)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair1, asset.CoinMarginedFutures, kline.OneHour, startTime, end)
	if err != nil {
		t.Error(err)
	}

	enabled, err := b.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), enabled[0], asset.Futures, kline.OneDay, startTime, end)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair3, asset.USDCMarginedFutures, kline.FiveMin, startTime, end)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchAccountInfo(context.Background(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchAccountInfo(context.Background(), asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchAccountInfo(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.FetchAccountInfo(context.Background(), asset.USDCMarginedFutures)
	if err != nil && err.Error() != "System error. Please try again later." {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	var oSpot = &order.Submit{
		Exchange: "Bybit",
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     0.0001,
		Amount:    10,
		ClientID:  "newOrder",
		AssetType: asset.Spot,
	}
	_, err := b.SubmitOrder(context.Background(), oSpot)
	if err != nil {
		if strings.TrimSpace(err.Error()) != "Balance insufficient" {
			t.Error(err)
		}
	}

	var oCMF = &order.Submit{
		Exchange: "Bybit",
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     10000,
		Amount:    1,
		ClientID:  "newOrder",
		AssetType: asset.CoinMarginedFutures,
	}
	_, err = b.SubmitOrder(context.Background(), oCMF)
	if err == nil {
		t.Error("SubmitOrder() Expected error")
	}

	var oUMF = &order.Submit{
		Exchange: "Bybit",
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USDT,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     10000,
		Amount:    1,
		ClientID:  "newOrder",
		AssetType: asset.USDTMarginedFutures,
	}
	_, err = b.SubmitOrder(context.Background(), oUMF)
	if err == nil {
		t.Error("SubmitOrder() Expected error")
	}

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	var oFutures = &order.Submit{
		Exchange:  "Bybit",
		Pair:      pair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     10000,
		Amount:    1,
		ClientID:  "newOrder",
		AssetType: asset.Futures,
	}
	_, err = b.SubmitOrder(context.Background(), oFutures)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	var oUSDC = &order.Submit{
		Exchange:  "Bybit",
		Pair:      pair1,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     10000,
		Amount:    1,
		ClientID:  "newOrder",
		AssetType: asset.USDCMarginedFutures,
	}
	_, err = b.SubmitOrder(context.Background(), oUSDC)
	if err != nil && err.Error() != "margin account not exist" {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.ModifyOrder(context.Background(), &order.Modify{
		Exchange: "Bybit",
		OrderID:  "1337",
		Price:    10000,
		Amount:   10,
		Side:     order.Sell,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		AssetType: asset.CoinMarginedFutures,
	})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	err := b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "Bybit",
		AssetType: asset.Spot,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		OrderID: "1234",
	})
	if err == nil {
		t.Error("CancelOrder() Spot Expected error")
	}

	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "Bybit",
		AssetType: asset.CoinMarginedFutures,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		OrderID: "1234",
	})
	if err == nil {
		t.Error("CancelOrder() CMF Expected error")
	}

	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "Bybit",
		AssetType: asset.USDTMarginedFutures,
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USDT,
		},
		OrderID: "1234",
	})
	if err == nil {
		t.Error("CancelOrder() USDT Expected error")
	}

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "Bybit",
		AssetType: asset.Futures,
		Pair:      pair,
		OrderID:   "1234",
	})
	if err == nil {
		t.Error("CancelOrder() Futures Expected error")
	}

	pair1, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  "Bybit",
		AssetType: asset.Futures,
		Pair:      pair1,
		OrderID:   "1234",
	})
	if err == nil {
		t.Error("CancelOrder() USDC Expected error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.CancelAllOrders(context.Background(),
		&order.Cancel{AssetType: asset.Spot})
	if err != nil {
		t.Error(err)
	}

	_, err = b.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "Bybit",
			AssetType: asset.CoinMarginedFutures,
			Pair: currency.Pair{
				Delimiter: "-",
				Base:      currency.BTC,
				Quote:     currency.USD,
			},
		})
	if err != nil {
		t.Error(err)
	}

	_, err = b.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "Bybit",
			AssetType: asset.USDTMarginedFutures,
			Pair: currency.Pair{
				Delimiter: "-",
				Base:      currency.BTC,
				Quote:     currency.USDT,
			},
		})
	if err != nil {
		t.Error(err)
	}

	pair, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "Bybit",
			AssetType: asset.Futures,
			Pair:      pair,
		})
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelAllOrders(context.Background(),
		&order.Cancel{
			Exchange:  "Bybit",
			AssetType: asset.USDCMarginedFutures,
			Pair:      pair1,
		})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetOrderInfo(context.Background(),
		"12234", pair, asset.Spot)
	if err == nil {
		t.Error("GetOrderInfo() Spot Expected error")
	}

	_, err = b.GetOrderInfo(context.Background(),
		"12234", pair, asset.USDTMarginedFutures)
	if err == nil {
		t.Error("GetOrderInfo() USDT Expected error")
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetOrderInfo(context.Background(),
		"12234", pair1, asset.CoinMarginedFutures)
	if err == nil {
		t.Error("GetOrderInfo() CMF Expected error")
	}

	pair2, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetOrderInfo(context.Background(),
		"12234", pair2, asset.Futures)
	if err == nil {
		t.Error("GetOrderInfo() Futures Expected error")
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetOrderInfo(context.Background(),
		"12234", pair3, asset.USDCMarginedFutures)
	if err == nil {
		t.Error("GetOrderInfo() USDC Expected error")
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestSpot = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}

	var getOrdersRequestUMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair},
		AssetType: asset.USDTMarginedFutures,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestCMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair1},
		AssetType: asset.CoinMarginedFutures,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestCMF)
	if err != nil {
		t.Error(err)
	}

	pair2, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestFutures = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair2},
		AssetType: asset.Futures,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestUSDC = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair3},
		AssetType: asset.USDCMarginedFutures,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestUSDC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestSpot = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}

	var getOrdersRequestUMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair},
		AssetType: asset.USDTMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestCMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair1},
		AssetType: asset.CoinMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestCMF)
	if err != nil {
		t.Error(err)
	}

	pair2, err := currency.NewPairFromString("BTCUSDZ22")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestFutures = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair2},
		AssetType: asset.Futures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequestUSDC = order.MultiOrderRequest{
		Pairs:     currency.Pairs{pair3},
		AssetType: asset.USDCMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestUSDC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err == nil {
		t.Error("GetWithdrawalsHistory() Spot Expected error")
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()

	_, err := b.GetServerTime(context.Background(), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetServerTime(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetDepositAddress(context.Background(), currency.USDT, "", currency.ETH.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange: "Bybit",
		Amount:   10,
		Currency: currency.LTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		}})
	if err != nil && err.Error() != "Withdraw address chain or destination tag are not equal" {
		t.Fatal(err)
	}
}

// test cases for USDCMarginedFutures

func TestGetUSDCFuturesOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCFuturesOrderbook(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCContracts(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCContracts(context.Background(), pair, "next", 1500)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetUSDCContracts(context.Background(), currency.EMPTYPAIR, "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCSymbols(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCSymbols(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCKlines(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCMarkPriceKlines(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCMarkPriceKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCIndexPriceKlines(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCIndexPriceKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCPremiumIndexKlines(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCPremiumIndexKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCOpenInterest(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCOpenInterest(context.Background(), pair, "1d", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCLargeOrders(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCLargeOrders(context.Background(), pair, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCAccountRatio(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCAccountRatio(context.Background(), pair, "1d", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCLatestTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCLatestTrades(context.Background(), pair, "PERPETUAL", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceUSDCOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.PlaceUSDCOrder(context.Background(), pair, "Limit", "Order", "Buy", "", "", 10000, 1, 0, 0, 0, 0, 0, 0, false, false, false)
	if err != nil {
		t.Error(err)
	}

	_, err = b.PlaceUSDCOrder(context.Background(), pair, "Market", "StopOrder", "Buy", "ImmediateOrCancel", "", 0, 64300, 0, 0, 0, 0, 1000, 0, false, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyUSDCOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.ModifyUSDCOrder(context.Background(), pair, "Order", "", "orderLinkID", 0, 0, 0, 0, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelUSDCOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.CancelUSDCOrder(context.Background(), pair, "Order", "", "orderLinkID")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllActiveUSDCOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	err = b.CancelAllActiveUSDCOrder(context.Background(), pair, "Order")
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveUSDCOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetActiveUSDCOrder(context.Background(), pair, "PERPETUAL", "", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCOrderHistory(context.Background(), pair, "PERPETUAL", "", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCTradeHistory(context.Background(), pair, "PERPETUAL", "", "orderLinkID", "", "", 50, time.Now().Add(-time.Hour))
	if err == nil { // order with link ID "orderLinkID" not present
		t.Error("GetUSDCTradeHistory() Expected error")
	}
}

func TestGetUSDCTransactionLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetUSDCTransactionLog(context.Background(), time.Time{}, time.Time{}, "TRADE", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetUSDCWalletBalance(context.Background())
	if err != nil && err.Error() != "System error. Please try again later." {
		t.Error(err)
	}
}

func TestGetUSDCAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetUSDCAssetInfo(context.Background(), "")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetUSDCAssetInfo(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCMarginInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetUSDCMarginInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCPosition(context.Background(), pair, "PERPETUAL", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDCLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetUSDCLeverage(context.Background(), pair, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCSettlementHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCSettlementHistory(context.Background(), pair, "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCRiskLimit(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCRiskLimit(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestSetUSDCRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.SetUSDCRiskLimit(context.Background(), pair, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCLastFundingRate(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetUSDCLastFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUSDCPredictedFundingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	pair, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = b.GetUSDCPredictedFundingRate(context.Background(), pair)
	if err != nil {
		t.Error(err)
	}
}

const (
	futuresTradeConnectionPushData              = `{"topic": "trade.ETHUSD", "data": [ { "trade_time_ms": 1683308606936, "timestamp": "2023-05-05T17:43:26.000Z", "symbol": "ETHUSD", "side": "Buy", "size": 1, "price": 1995.05, "tick_direction": "ZeroPlusTick", "trade_id": "14bbac52-f716-5cd0-af1a-236a2f963be5", "cross_seq": 14563860127, "is_block_trade": "false" }, { "trade_time_ms": 1683308606941, "timestamp": "2023-05-05T17:43:26.000Z", "symbol": "ETHUSD", "side": "Sell", "size": 1, "price": 1995, "tick_direction": "MinusTick", "trade_id": "5050a170-03d4-5984-9542-1d7f62c3c45e", "cross_seq": 14563860131, "is_block_trade": "false" } ] }`
	futuresOrderbookL2Depth25CDeltaPushData     = `{"topic": "orderBookL2_25.BTCUSD", "type": "delta", "data": { "delete": [ { "price": "29597.00", "symbol": "BTCUSD", "id": 295970000, "side": "Buy" } ], "update": [ { "price": "29606.50", "symbol": "BTCUSD", "id": 296065000, "side": "Buy", "size": 17500 } ], "insert": [ { "price": "29607.50", "symbol": "BTCUSD", "id": 296075000, "side": "Buy", "size": 299 } ], "transactTimeE6": 0 }, "cross_seq": 21538613235, "timestamp_e6": 1683307211621933 }`
	futuresOrderbookL2Depth25CSnapshootPushData = `{"topic": "orderBookL2_25.BTCUSD", "type": "snapshot", "data": [ { "price": "29517.00", "symbol": "BTCUSD", "id": 295170000, "side": "Buy", "size": 32586 }, { "price": "29517.50", "symbol": "BTCUSD", "id": 295175000, "side": "Buy", "size": 53029 }, { "price": "29518.00", "symbol": "BTCUSD", "id": 295180000, "side": "Buy", "size": 107133 }, { "price": "29518.50", "symbol": "BTCUSD", "id": 295185000, "side": "Buy", "size": 50425 }, { "price": "29519.00", "symbol": "BTCUSD", "id": 295190000, "side": "Buy", "size": 74309 }, { "price": "29519.50", "symbol": "BTCUSD", "id": 295195000, "side": "Buy", "size": 88387 }, { "price": "29520.00", "symbol": "BTCUSD", "id": 295200000, "side": "Buy", "size": 47011 }, { "price": "29520.50", "symbol": "BTCUSD", "id": 295205000, "side": "Buy", "size": 38408 }, { "price": "29521.00", "symbol": "BTCUSD", "id": 295210000, "side": "Buy", "size": 632 }, { "price": "29521.50", "symbol": "BTCUSD", "id": 295215000, "side": "Buy", "size": 39657 }, { "price": "29522.00", "symbol": "BTCUSD", "id": 295220000, "side": "Buy", "size": 51845 }, { "price": "29524.00", "symbol": "BTCUSD", "id": 295240000, "side": "Buy", "size": 57 }, { "price": "29525.00", "symbol": "BTCUSD", "id": 295250000, "side": "Buy", "size": 7542 }, { "price": "29525.50", "symbol": "BTCUSD", "id": 295255000, "side": "Buy", "size": 5 }, { "price": "29526.00", "symbol": "BTCUSD", "id": 295260000, "side": "Buy", "size": 2013 }, { "price": "29526.50", "symbol": "BTCUSD", "id": 295265000, "side": "Buy", "size": 45460 }, { "price": "29527.00", "symbol": "BTCUSD", "id": 295270000, "side": "Sell", "size": 228514 }, { "price": "29527.50", "symbol": "BTCUSD", "id": 295275000, "side": "Sell", "size": 103019 }, { "price": "29528.00", "symbol": "BTCUSD", "id": 295280000, "side": "Sell", "size": 123285 }, { "price": "29529.50", "symbol": "BTCUSD", "id": 295295000, "side": "Sell", "size": 915 }, { "price": "29530.00", "symbol": "BTCUSD", "id": 295300000, "side": "Sell", "size": 2 }, { "price": "29530.50", "symbol": "BTCUSD", "id": 295305000, "side": "Sell", "size": 15000 }, { "price": "29531.00", "symbol": "BTCUSD", "id": 295310000, "side": "Sell", "size": 15199 }, { "price": "29531.50", "symbol": "BTCUSD", "id": 295315000, "side": "Sell", "size": 299 }, { "price": "29532.00", "symbol": "BTCUSD", "id": 295320000, "side": "Sell", "size": 40706 }, { "price": "29532.50", "symbol": "BTCUSD", "id": 295325000, "side": "Sell", "size": 17500 }, { "price": "29533.00", "symbol": "BTCUSD", "id": 295330000, "side": "Sell", "size": 40506 }, { "price": "29533.50", "symbol": "BTCUSD", "id": 295335000, "side": "Sell", "size": 60462 }, { "price": "29534.00", "symbol": "BTCUSD", "id": 295340000, "side": "Sell", "size": 100769 }, { "price": "29534.50", "symbol": "BTCUSD", "id": 295345000, "side": "Sell", "size": 598 }, { "price": "29535.00", "symbol": "BTCUSD", "id": 295350000, "side": "Sell", "size": 106183 }, { "price": "29535.50", "symbol": "BTCUSD", "id": 295355000, "side": "Sell", "size": 840 }, { "price": "29536.00", "symbol": "BTCUSD", "id": 295360000, "side": "Sell", "size": 105389 }, { "price": "29536.50", "symbol": "BTCUSD", "id": 295365000, "side": "Sell", "size": 40881 }, { "price": "29537.00", "symbol": "BTCUSD", "id": 295370000, "side": "Sell", "size": 32297 }, { "price": "29537.50", "symbol": "BTCUSD", "id": 295375000, "side": "Sell", "size": 67333 }, { "price": "29538.00", "symbol": "BTCUSD", "id": 295380000, "side": "Sell", "size": 16950 }, { "price": "29538.50", "symbol": "BTCUSD", "id": 295385000, "side": "Sell", "size": 102518 }, { "price": "29539.00", "symbol": "BTCUSD", "id": 295390000, "side": "Sell", "size": 103458 }, { "price": "29539.50", "symbol": "BTCUSD", "id": 295395000, "side": "Sell", "size": 92849 }, { "price": "29540.00", "symbol": "BTCUSD", "id": 295400000, "side": "Sell", "size": 133414 } ], "cross_seq": 21541553659, "timestamp_e6": 1683320123862497 }`
	futuresOrderbookDepth200SnapshootPushData   = `{"topic": "orderBook_200.100ms.BTCUSD", "type": "snapshot", "data": [ { "price": "29486.00", "symbol": "BTCUSD", "id": 294860000, "side": "Buy", "size": 50 }, { "price": "29487.00", "symbol": "BTCUSD", "id": 294870000, "side": "Buy", "size": 6 }, { "price": "29487.50", "symbol": "BTCUSD", "id": 294875000, "side": "Buy", "size": 130 }, { "price": "29488.00", "symbol": "BTCUSD", "id": 294880000, "side": "Buy", "size": 10000 }, { "price": "29489.50", "symbol": "BTCUSD", "id": 294895000, "side": "Buy", "size": 7 }, { "price": "29490.00", "symbol": "BTCUSD", "id": 294900000, "side": "Buy", "size": 1686 }, { "price": "29490.50", "symbol": "BTCUSD", "id": 294905000, "side": "Buy", "size": 39 }, { "price": "29491.00", "symbol": "BTCUSD", "id": 294910000, "side": "Buy", "size": 36 }, { "price": "29491.50", "symbol": "BTCUSD", "id": 294915000, "side": "Buy", "size": 528 }, { "price": "29492.00", "symbol": "BTCUSD", "id": 294920000, "side": "Buy", "size": 23 }, { "price": "29604.50", "symbol": "BTCUSD", "id": 296045000, "side": "Buy", "size": 54459 }, { "price": "29605.00", "symbol": "BTCUSD", "id": 296050000, "side": "Buy", "size": 190668 }, { "price": "29605.50", "symbol": "BTCUSD", "id": 296055000, "side": "Buy", "size": 137083 }, { "price": "29606.00", "symbol": "BTCUSD", "id": 296060000, "side": "Buy", "size": 48455 }, { "price": "29606.50", "symbol": "BTCUSD", "id": 296065000, "side": "Buy", "size": 73097 }, { "price": "29607.00", "symbol": "BTCUSD", "id": 296070000, "side": "Buy", "size": 58459 }, { "price": "29607.50", "symbol": "BTCUSD", "id": 296075000, "side": "Buy", "size": 130149 }, { "price": "29608.00", "symbol": "BTCUSD", "id": 296080000, "side": "Buy", "size": 89916 }, { "price": "29608.50", "symbol": "BTCUSD", "id": 296085000, "side": "Buy", "size": 100335 }, { "price": "29609.00", "symbol": "BTCUSD", "id": 296090000, "side": "Buy", "size": 92727 }, { "price": "29609.50", "symbol": "BTCUSD", "id": 296095000, "side": "Buy", "size": 30668 }, { "price": "29610.00", "symbol": "BTCUSD", "id": 296100000, "side": "Buy", "size": 92324 }, { "price": "29610.50", "symbol": "BTCUSD", "id": 296105000, "side": "Buy", "size": 107466 }, { "price": "29611.00", "symbol": "BTCUSD", "id": 296110000, "side": "Buy", "size": 45974 } ], "cross_seq": 21538712631, "timestamp_e6": 1683307635861843 }`
	futuresOrderbookDepth200UpdatePushData      = `{"topic": "orderBook_200.100ms.BTCUSD", "type": "delta", "data": { "delete": [ { "price": "29484.50", "symbol": "BTCUSD", "id": 294845000, "side": "Buy" } ], "update": [ { "price": "29628.50", "symbol": "BTCUSD", "id": 296285000, "side": "Sell", "size": 95285 }, { "price": "29628.00", "symbol": "BTCUSD", "id": 296280000, "side": "Sell", "size": 88728 } ], "insert": [ { "price": "29617.00", "symbol": "BTCUSD", "id": 296170000, "side": "Buy", "size": 64100 } ], "transactTimeE6": 0 }, "cross_seq": 21538712722, "timestamp_e6": 1683307636462153 }`
	futuresUnsubscribePushData                  = `{"success": true, "ret_msg": "", "conn_id": "7bb4d5b1-0b98-4c61-9979-5a7abc9d5028", "request": { "op": "unsubscribe", "args": [ "orderBook_200.100ms.BTCUSD" ] } }`
	futuresInsurancePushData                    = `{"topic": "insurance.ETH", "data": [ { "currency": "ETH", "timestamp": "2023-05-04T20:00:00Z", "wallet_balance": 5067222494036 } ] }`
	futuresInstrumentInfoPushData               = `{"topic": "instrument_info.100ms.BTCUSD", "type": "snapshot", "data": { "id": 1, "symbol": "BTCUSD", "last_price_e4": 295245000, "last_price": "29524.50", "bid1_price_e4": 295240000, "bid1_price": "29524.00", "ask1_price_e4": 295245000, "ask1_price": "29524.50", "last_tick_direction": "PlusTick", "prev_price_24h_e4": 288165000, "prev_price_24h": "28816.50", "price_24h_pcnt_e6": 24569, "high_price_24h_e4": 297000000, "high_price_24h": "29700.00", "low_price_24h_e4": 286705000, "low_price_24h": "28670.50", "prev_price_1h_e4": 295195000, "prev_price_1h": "29519.50", "price_1h_pcnt_e6": 169, "mark_price_e4": 295210000, "mark_price": "29521.00", "index_price_e4": 295289500, "index_price": "29528.95", "open_interest": 508778809, "open_value_e8": 0, "total_turnover_e8": 106089738482568, "turnover_24h_e8": 2666832075738, "total_volume": 30329789303, "volume_24h": 778214476, "funding_rate_e6": 100, "predicted_funding_rate_e6": 100, "cross_seq": 21539067996, "created_at": "2018-11-14T16:33:26Z", "updated_at": "2023-05-05T17:49:57Z", "next_funding_time": "2023-05-06T00:00:00Z", "countdown_hour": 7, "funding_rate_interval": 8, "settle_time_e9": 0, "delisting_status": "0" }, "cross_seq": 21539068089, "timestamp_e6": 1683308997862627 }`
	futuresKlinePushData                        = `{"topic": "klineV2.1.BTCUSD", "data": [ { "start": 1683309060, "end": 1683309120, "open": 29529, "close": 29541, "high": 29541, "low": 29528.5, "volume": 184045, "turnover": 6.2314151, "confirm": false, "cross_seq": 21539090874, "timestamp": 1683309102077887 } ], "timestamp_e6": 1683309102077887 }`
	futuresSubscriptionResponsePushData         = `{"success": true, "ret_msg": "", "conn_id": "7bb4d5b1-0b98-4c61-9979-5a7abc9d5028", "request": { "op": "subscribe", "args": [ "liquidation" ] } }`
	futuresPositionPushData                     = `{"topic": "position", "data": [ { "user_id": 533285, "symbol": "BTCUSD", "size": 200, "side": "Buy", "position_value": "0.0099975", "entry_price": "20005.00125031", "liq_price": "489", "bust_price": "489", "leverage": "5", "order_margin": "0", "position_margin": "0.39929535", "available_balance": "0.39753405", "take_profit": "0", "stop_loss": "0", "realised_pnl": "0.00055631", "trailing_stop": "0", "trailing_active": "0", "wallet_balance": "0.40053971", "risk_id": 1, "occ_closing_fee": "0.0002454", "occ_funding_fee": "0", "auto_add_margin": 1, "cum_realised_pnl": "0.00055105", "position_status": "Normal", "position_seq": 0, "Isolated": false, "mode": 0, "position_idx": 0, "tp_sl_mode": "Partial", "tp_order_num": 0, "sl_order_num": 0, "tp_free_size_x": 200, "sl_free_size_x": 200 } ] }`
	futuresExecutionPushData                    = `{"topic": "execution", "data": [ { "symbol": "BTCUSD", "side": "Buy", "order_id": "xxxxxxxx-xxxx-xxxx-9a8f-4a973eb5c418", "exec_id": "xxxxxxxx-xxxx-xxxx-8b66-c3d2fcd352f6", "order_link_id": "", "price": "8300", "order_qty": 1, "exec_type": "Trade", "exec_qty": 1, "exec_fee": "0.00000009", "leaves_qty": 0, "is_maker": false, "trade_time": "2020-01-14T14:07:23.629Z" } ] }`
	futuresOrderPushData                        = `{"topic": "order", "data": [ { "order_id": "1640b725-75e9-407d-bea9-aae4fc666d33", "order_link_id": "IPBTC00005", "symbol": "BTCUSD", "side": "Sell", "order_type": "Market", "price": "20564", "qty": 200, "time_in_force": "ImmediateOrCancel", "create_type": "CreateByUser", "cancel_type": "", "order_status": "Filled", "leaves_qty": 0, "cum_exec_qty": 200, "cum_exec_value": "0.00943552", "cum_exec_fee": "0.00000567", "timestamp": "2022-06-21T07:35:56.505Z", "take_profit": "18500", "tp_trigger_by": "LastPrice", "stop_loss": "22000", "sl_trigger_by": "LastPrice", "trailing_stop": "0", "last_exec_price": "21196.5", "reduce_only": false, "close_on_trigger": false } ] }`
	futuresStopOrderPushData                    = `{"topic": "stop_order", "data": [ { "order_id": "xxxxxxxx-xxxx-xxxx-98fb-335aaa6c613b", "order_link_id": "", "user_id": 1, "symbol": "BTCUSD", "side": "Buy", "order_type": "Limit", "price": "8584.5", "qty": 1, "time_in_force": "ImmediateOrCancel", "create_type": "CreateByStopOrder", "cancel_type": "", "order_status": "Untriggered", "stop_order_type": "Stop", "trigger_by": "LastPrice", "trigger_price": "8584.5", "close_on_trigger": false, "timestamp": "2020-01-14T14:11:22.062Z", "take_profit": 10000, "stop_loss": 7500 } ] }`
	futuresWalletPushData                       = `{"topic": "wallet", "data": [ { "user_id": 738713, "coin": "BTC", "available_balance": "1.50121026", "wallet_balance": "1.50121261" } ] }`
)

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	err := b.wsFuturesHandleData([]byte(futuresOrderbookL2Depth25CSnapshootPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresOrderbookL2Depth25CDeltaPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresOrderbookDepth200SnapshootPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresTradeConnectionPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresOrderbookDepth200UpdatePushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresUnsubscribePushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresInsurancePushData))
	if err != nil {
		t.Error(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresInstrumentInfoPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresKlinePushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresSubscriptionResponsePushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresPositionPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresExecutionPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresOrderPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresStopOrderPushData))
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsFuturesHandleData([]byte(futuresWalletPushData))
	if err != nil {
		t.Fatal(err)
	}
}
func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	supportedAssets := b.GetAssetTypes(false)
	ctx := context.Background()
	for x := range supportedAssets {
		err := b.UpdateTickers(ctx, supportedAssets[x])
		if err != nil {
			t.Fatalf("%v %v\n", supportedAssets[x], err)
		}

		avail, err := b.GetAvailablePairs(supportedAssets[x])
		if err != nil {
			t.Fatalf("%v %v\n", supportedAssets[x], err)
		}

		for y := range avail {
			_, err = ticker.GetTicker(b.GetName(), avail[y], supportedAssets[x])
			if err != nil {
				t.Fatalf("%v %v %v\n", avail[y], supportedAssets[x], err)
			}
		}
	}
}

func TestGetTickersV5(t *testing.T) {
	t.Parallel()

	_, err := b.GetTickersV5(context.Background(), "bruh", "", "")
	if err != nil && err.Error() != "Illegal category" {
		t.Error(err)
	}

	_, err = b.GetTickersV5(context.Background(), "option", "", "")
	if !errors.Is(err, errBaseNotSet) {
		t.Fatalf("expected: %v, received: %v", errBaseNotSet, err)
	}

	_, err = b.GetTickersV5(context.Background(), "spot", "", "")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTickersV5(context.Background(), "option", "", "BTC")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTickersV5(context.Background(), "inverse", "", "")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTickersV5(context.Background(), "linear", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	err := b.UpdateOrderExecutionLimits(context.Background(), asset.USDCMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v expected: %v", err, asset.ErrNotSupported)
	}

	err = b.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	if err != nil {
		t.Error("Okx UpdateOrderExecutionLimits() error", err)
	}

	avail, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal("Okx GetAvailablePairs() error", err)
	}

	for x := range avail {
		limits, err := b.GetOrderExecutionLimits(asset.Spot, avail[x])
		if err != nil {
			t.Fatal("Okx GetOrderExecutionLimits() error", err)
		}
		if limits == (order.MinMaxLevel{}) {
			t.Fatal("Okx GetOrderExecutionLimits() error cannot be nil")
		}
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()

	_, err := b.GetFeeRate(context.Background(), "", "", "")
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("received %v but expected %v", err, errCategoryNotSet)
	}

	_, err = b.GetFeeRate(context.Background(), "bruh", "", "")
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("received %v but expected %v", err, errInvalidCategory)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err = b.GetFeeRate(context.Background(), "spot", "", "")
	if !errors.Is(err, nil) {
		t.Errorf("received %v but expected %v", err, nil)
	}

	_, err = b.GetFeeRate(context.Background(), "linear", "", "")
	if !errors.Is(err, nil) {
		t.Errorf("received %v but expected %v", err, nil)
	}

	_, err = b.GetFeeRate(context.Background(), "inverse", "", "")
	if !errors.Is(err, nil) {
		t.Errorf("received %v but expected %v", err, nil)
	}

	_, err = b.GetFeeRate(context.Background(), "option", "", "ETH")
	if !errors.Is(err, nil) {
		t.Errorf("received %v but expected %v", err, nil)
	}
}

func TestForceFileStandard(t *testing.T) {
	t.Parallel()
	err := sharedtestvalues.ForceFileStandard(t, sharedtestvalues.EmptyStringPotentialPattern)
	if err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.Fatal("Please use convert.StringToFloat64 type instead of `float64` and remove `,string` as strings can be empty in unmarshal process. Then call the Float64() method.")
	}
}

var orderbookMarshalingTestData = []byte(`{"topic":"orderBook_200.100ms.DOTUSD","type":"snapshot","data":[{"price":"4.020","symbol":"DOTUSD","id":40200,"side":"Buy","size":1},{"price":"4.025","symbol":"DOTUSD","id":40250,"side":"Buy","size":98},{"price":"4.040","symbol":"DOTUSD","id":40400,"side":"Buy","size":50},{"price":"4.045","symbol":"DOTUSD","id":40450,"side":"Buy","size":15},{"price":"4.050","symbol":"DOTUSD","id":40500,"side":"Buy","size":1924},{"price":"4.055","symbol":"DOTUSD","id":40550,"side":"Buy","size":112},{"price":"4.080","symbol":"DOTUSD","id":40800,"side":"Buy","size":66},{"price":"4.100","symbol":"DOTUSD","id":41000,"side":"Buy","size":1622},{"price":"4.110","symbol":"DOTUSD","id":41100,"side":"Buy","size":17},{"price":"4.120","symbol":"DOTUSD","id":41200,"side":"Buy","size":20},{"price":"4.125","symbol":"DOTUSD","id":41250,"side":"Buy","size":1},{"price":"4.150","symbol":"DOTUSD","id":41500,"side":"Buy","size":415},{"price":"4.200","symbol":"DOTUSD","id":42000,"side":"Buy","size":483},{"price":"4.215","symbol":"DOTUSD","id":42150,"side":"Buy","size":1},{"price":"4.220","symbol":"DOTUSD","id":42200,"side":"Buy","size":90},{"price":"4.225","symbol":"DOTUSD","id":42250,"side":"Buy","size":1},{"price":"4.245","symbol":"DOTUSD","id":42450,"side":"Buy","size":150},{"price":"4.250","symbol":"DOTUSD","id":42500,"side":"Buy","size":131},{"price":"4.260","symbol":"DOTUSD","id":42600,"side":"Buy","size":14},{"price":"4.270","symbol":"DOTUSD","id":42700,"side":"Buy","size":100},{"price":"4.275","symbol":"DOTUSD","id":42750,"side":"Buy","size":31},{"price":"4.290","symbol":"DOTUSD","id":42900,"side":"Buy","size":1},{"price":"4.300","symbol":"DOTUSD","id":43000,"side":"Buy","size":127},{"price":"4.315","symbol":"DOTUSD","id":43150,"side":"Buy","size":15},{"price":"4.320","symbol":"DOTUSD","id":43200,"side":"Buy","size":474},{"price":"4.330","symbol":"DOTUSD","id":43300,"side":"Buy","size":50},{"price":"4.355","symbol":"DOTUSD","id":43550,"side":"Buy","size":2500},{"price":"4.360","symbol":"DOTUSD","id":43600,"side":"Buy","size":500},{"price":"4.365","symbol":"DOTUSD","id":43650,"side":"Buy","size":4},{"price":"4.370","symbol":"DOTUSD","id":43700,"side":"Buy","size":464},{"price":"4.375","symbol":"DOTUSD","id":43750,"side":"Buy","size":61},{"price":"4.385","symbol":"DOTUSD","id":43850,"side":"Buy","size":20},{"price":"4.390","symbol":"DOTUSD","id":43900,"side":"Buy","size":15},{"price":"4.400","symbol":"DOTUSD","id":44000,"side":"Buy","size":4496},{"price":"4.420","symbol":"DOTUSD","id":44200,"side":"Buy","size":30},{"price":"4.440","symbol":"DOTUSD","id":44400,"side":"Buy","size":2},{"price":"4.450","symbol":"DOTUSD","id":44500,"side":"Buy","size":1000},{"price":"4.460","symbol":"DOTUSD","id":44600,"side":"Buy","size":1000},{"price":"4.470","symbol":"DOTUSD","id":44700,"side":"Buy","size":26},{"price":"4.485","symbol":"DOTUSD","id":44850,"side":"Buy","size":2},{"price":"4.500","symbol":"DOTUSD","id":45000,"side":"Buy","size":5137},{"price":"4.510","symbol":"DOTUSD","id":45100,"side":"Buy","size":15},{"price":"4.515","symbol":"DOTUSD","id":45150,"side":"Buy","size":65},{"price":"4.520","symbol":"DOTUSD","id":45200,"side":"Buy","size":500},{"price":"4.525","symbol":"DOTUSD","id":45250,"side":"Buy","size":40},{"price":"4.530","symbol":"DOTUSD","id":45300,"side":"Buy","size":1},{"price":"4.535","symbol":"DOTUSD","id":45350,"side":"Buy","size":25},{"price":"4.545","symbol":"DOTUSD","id":45450,"side":"Buy","size":75},{"price":"4.550","symbol":"DOTUSD","id":45500,"side":"Buy","size":754},{"price":"4.555","symbol":"DOTUSD","id":45550,"side":"Buy","size":40},{"price":"4.560","symbol":"DOTUSD","id":45600,"side":"Buy","size":25},{"price":"4.590","symbol":"DOTUSD","id":45900,"side":"Buy","size":2},{"price":"4.595","symbol":"DOTUSD","id":45950,"side":"Buy","size":1},{"price":"4.600","symbol":"DOTUSD","id":46000,"side":"Buy","size":6570},{"price":"4.605","symbol":"DOTUSD","id":46050,"side":"Buy","size":40},{"price":"4.610","symbol":"DOTUSD","id":46100,"side":"Buy","size":25},{"price":"4.615","symbol":"DOTUSD","id":46150,"side":"Buy","size":230},{"price":"4.625","symbol":"DOTUSD","id":46250,"side":"Buy","size":91},{"price":"4.630","symbol":"DOTUSD","id":46300,"side":"Buy","size":15},{"price":"4.635","symbol":"DOTUSD","id":46350,"side":"Buy","size":400},{"price":"4.640","symbol":"DOTUSD","id":46400,"side":"Buy","size":464},{"price":"4.650","symbol":"DOTUSD","id":46500,"side":"Buy","size":395},{"price":"4.655","symbol":"DOTUSD","id":46550,"side":"Buy","size":235},{"price":"4.660","symbol":"DOTUSD","id":46600,"side":"Buy","size":75},{"price":"4.665","symbol":"DOTUSD","id":46650,"side":"Buy","size":41},{"price":"4.670","symbol":"DOTUSD","id":46700,"side":"Buy","size":66},{"price":"4.675","symbol":"DOTUSD","id":46750,"side":"Buy","size":1},{"price":"4.685","symbol":"DOTUSD","id":46850,"side":"Buy","size":41},{"price":"4.695","symbol":"DOTUSD","id":46950,"side":"Buy","size":205},{"price":"4.700","symbol":"DOTUSD","id":47000,"side":"Buy","size":5184},{"price":"4.710","symbol":"DOTUSD","id":47100,"side":"Buy","size":16680},{"price":"4.720","symbol":"DOTUSD","id":47200,"side":"Buy","size":8},{"price":"4.725","symbol":"DOTUSD","id":47250,"side":"Buy","size":248},{"price":"4.735","symbol":"DOTUSD","id":47350,"side":"Buy","size":40},{"price":"4.740","symbol":"DOTUSD","id":47400,"side":"Buy","size":641},{"price":"4.750","symbol":"DOTUSD","id":47500,"side":"Buy","size":500},{"price":"4.755","symbol":"DOTUSD","id":47550,"side":"Buy","size":195},{"price":"4.770","symbol":"DOTUSD","id":47700,"side":"Buy","size":89},{"price":"4.780","symbol":"DOTUSD","id":47800,"side":"Buy","size":66},{"price":"4.785","symbol":"DOTUSD","id":47850,"side":"Buy","size":159},{"price":"4.790","symbol":"DOTUSD","id":47900,"side":"Buy","size":23},{"price":"4.800","symbol":"DOTUSD","id":48000,"side":"Buy","size":4372},{"price":"4.805","symbol":"DOTUSD","id":48050,"side":"Buy","size":61},{"price":"4.815","symbol":"DOTUSD","id":48150,"side":"Buy","size":2},{"price":"4.820","symbol":"DOTUSD","id":48200,"side":"Buy","size":700},{"price":"4.830","symbol":"DOTUSD","id":48300,"side":"Buy","size":169},{"price":"4.835","symbol":"DOTUSD","id":48350,"side":"Buy","size":100},{"price":"4.845","symbol":"DOTUSD","id":48450,"side":"Buy","size":60},{"price":"4.850","symbol":"DOTUSD","id":48500,"side":"Buy","size":315},{"price":"4.865","symbol":"DOTUSD","id":48650,"side":"Buy","size":83},{"price":"4.880","symbol":"DOTUSD","id":48800,"side":"Buy","size":5524},{"price":"4.885","symbol":"DOTUSD","id":48850,"side":"Buy","size":1},{"price":"4.890","symbol":"DOTUSD","id":48900,"side":"Buy","size":4},{"price":"4.895","symbol":"DOTUSD","id":48950,"side":"Buy","size":55},{"price":"4.900","symbol":"DOTUSD","id":49000,"side":"Buy","size":822},{"price":"4.910","symbol":"DOTUSD","id":49100,"side":"Buy","size":2500},{"price":"4.920","symbol":"DOTUSD","id":49200,"side":"Buy","size":500},{"price":"4.925","symbol":"DOTUSD","id":49250,"side":"Buy","size":40},{"price":"4.930","symbol":"DOTUSD","id":49300,"side":"Buy","size":8},{"price":"4.935","symbol":"DOTUSD","id":49350,"side":"Buy","size":73},{"price":"4.940","symbol":"DOTUSD","id":49400,"side":"Buy","size":330},{"price":"4.945","symbol":"DOTUSD","id":49450,"side":"Buy","size":17},{"price":"4.950","symbol":"DOTUSD","id":49500,"side":"Buy","size":901},{"price":"4.955","symbol":"DOTUSD","id":49550,"side":"Buy","size":43},{"price":"4.960","symbol":"DOTUSD","id":49600,"side":"Buy","size":3},{"price":"4.965","symbol":"DOTUSD","id":49650,"side":"Buy","size":24432},{"price":"4.970","symbol":"DOTUSD","id":49700,"side":"Buy","size":150},{"price":"4.975","symbol":"DOTUSD","id":49750,"side":"Buy","size":2},{"price":"4.980","symbol":"DOTUSD","id":49800,"side":"Buy","size":65},{"price":"4.985","symbol":"DOTUSD","id":49850,"side":"Buy","size":53},{"price":"4.990","symbol":"DOTUSD","id":49900,"side":"Buy","size":62},{"price":"4.995","symbol":"DOTUSD","id":49950,"side":"Buy","size":126},{"price":"5.000","symbol":"DOTUSD","id":50000,"side":"Buy","size":4695},{"price":"5.005","symbol":"DOTUSD","id":50050,"side":"Buy","size":96},{"price":"5.010","symbol":"DOTUSD","id":50100,"side":"Buy","size":1},{"price":"5.015","symbol":"DOTUSD","id":50150,"side":"Buy","size":40},{"price":"5.020","symbol":"DOTUSD","id":50200,"side":"Buy","size":22},{"price":"5.025","symbol":"DOTUSD","id":50250,"side":"Buy","size":107},{"price":"5.030","symbol":"DOTUSD","id":50300,"side":"Buy","size":642},{"price":"5.040","symbol":"DOTUSD","id":50400,"side":"Buy","size":905},{"price":"5.045","symbol":"DOTUSD","id":50450,"side":"Buy","size":94},{"price":"5.050","symbol":"DOTUSD","id":50500,"side":"Buy","size":2479},{"price":"5.055","symbol":"DOTUSD","id":50550,"side":"Buy","size":1030},{"price":"5.065","symbol":"DOTUSD","id":50650,"side":"Buy","size":2},{"price":"5.070","symbol":"DOTUSD","id":50700,"side":"Buy","size":29},{"price":"5.075","symbol":"DOTUSD","id":50750,"side":"Buy","size":116},
{"price":"5.080","symbol":"DOTUSD","id":50800,"side":"Buy","size":773},{"price":"5.085","symbol":"DOTUSD","id":50850,"side":"Buy","size":531},{"price":"5.090","symbol":"DOTUSD","id":50900,"side":"Buy","size":49},{"price":"5.095","symbol":"DOTUSD","id":50950,"side":"Buy","size":15455},{"price":"5.100","symbol":"DOTUSD","id":51000,"side":"Buy","size":360},{"price":"5.110","symbol":"DOTUSD","id":51100,"side":"Buy","size":51102},{"price":"5.115","symbol":"DOTUSD","id":51150,"side":"Buy","size":52},{"price":"5.120","symbol":"DOTUSD","id":51200,"side":"Buy","size":421},{"price":"5.125","symbol":"DOTUSD","id":51250,"side":"Buy","size":1042},{"price":"5.130","symbol":"DOTUSD","id":51300,"side":"Buy","size":220},{"price":"5.135","symbol":"DOTUSD","id":51350,"side":"Buy","size":9368},{"price":"5.145","symbol":"DOTUSD","id":51450,"side":"Buy","size":16},{"price":"5.150","symbol":"DOTUSD","id":51500,"side":"Buy","size":635},{"price":"5.155","symbol":"DOTUSD","id":51550,"side":"Buy","size":2003},
{"price":"5.160","symbol":"DOTUSD","id":51600,"side":"Buy","size":1124},{"price":"5.165","symbol":"DOTUSD","id":51650,"side":"Buy","size":20},{"price":"5.170","symbol":"DOTUSD","id":51700,"side":"Buy","size":7797},{"price":"5.175","symbol":"DOTUSD","id":51750,"side":"Buy","size":130},{"price":"5.180","symbol":"DOTUSD","id":51800,"side":"Buy","size":2},{"price":"5.185","symbol":"DOTUSD","id":51850,"side":"Buy","size":1562},{"price":"5.190","symbol":"DOTUSD","id":51900,"side":"Buy","size":2},{"price":"5.200","symbol":"DOTUSD","id":52000,"side":"Buy","size":3116},{"price":"5.205","symbol":"DOTUSD","id":52050,"side":"Buy","size":881},{"price":"5.210","symbol":"DOTUSD","id":52100,"side":"Buy","size":1022},{"price":"5.220","symbol":"DOTUSD","id":52200,"side":"Buy","size":328},{"price":"5.225","symbol":"DOTUSD","id":52250,"side":"Buy","size":51},{"price":"5.230","symbol":"DOTUSD","id":52300,"side":"Buy","size":1012},{"price":"5.235","symbol":"DOTUSD","id":52350,"side":"Buy","size":323},{"price":"5.240","symbol":"DOTUSD","id":52400,"side":"Buy","size":265},{"price":"5.245","symbol":"DOTUSD","id":52450,"side":"Buy","size":2},{"price":"5.250","symbol":"DOTUSD","id":52500,"side":"Buy","size":2862},{"price":"5.255","symbol":"DOTUSD","id":52550,"side":"Buy","size":1580},{"price":"5.260","symbol":"DOTUSD","id":52600,"side":"Buy","size":3},{"price":"5.265","symbol":"DOTUSD","id":52650,"side":"Buy","size":16},{"price":"5.270","symbol":"DOTUSD","id":52700,"side":"Buy","size":190},{"price":"5.275","symbol":"DOTUSD","id":52750,"side":"Buy","size":208},{"price":"5.280","symbol":"DOTUSD","id":52800,"side":"Buy","size":794},{"price":"5.285","symbol":"DOTUSD","id":52850,"side":"Buy","size":226},{"price":"5.290","symbol":"DOTUSD","id":52900,"side":"Buy","size":726},{"price":"5.295","symbol":"DOTUSD","id":52950,"side":"Buy","size":231},{"price":"5.300","symbol":"DOTUSD","id":53000,"side":"Buy","size":9815},{"price":"5.305","symbol":"DOTUSD","id":53050,"side":"Buy","size":1},{"price":"5.310","symbol":"DOTUSD","id":53100,"side":"Buy","size":825},{"price":"5.315","symbol":"DOTUSD","id":53150,"side":"Buy","size":5},{"price":"5.320","symbol":"DOTUSD","id":53200,"side":"Buy","size":277},{"price":"5.325","symbol":"DOTUSD","id":53250,"side":"Buy","size":35104},{"price":"5.335","symbol":"DOTUSD","id":53350,"side":"Buy","size":32596},{"price":"5.340","symbol":"DOTUSD","id":53400,"side":"Buy","size":1921},{"price":"5.345","symbol":"DOTUSD","id":53450,"side":"Buy","size":38862},{"price":"5.350","symbol":"DOTUSD","id":53500,"side":"Buy","size":2908},{"price":"5.355","symbol":"DOTUSD","id":53550,"side":"Buy","size":38777},{"price":"5.360","symbol":"DOTUSD","id":53600,"side":"Buy","size":9150},{"price":"5.365","symbol":"DOTUSD","id":53650,"side":"Buy","size":23718},{"price":"5.370","symbol":"DOTUSD","id":53700,"side":"Buy","size":2483},{"price":"5.375","symbol":"DOTUSD","id":53750,"side":"Buy","size":26824},{"price":"5.380","symbol":"DOTUSD","id":53800,"side":"Buy","size":35163},{"price":"5.385","symbol":"DOTUSD","id":53850,"side":"Buy","size":23214},{"price":"5.390","symbol":"DOTUSD","id":53900,"side":"Buy","size":3051},{"price":"5.395","symbol":"DOTUSD","id":53950,"side":"Buy","size":33167},{"price":"5.400","symbol":"DOTUSD","id":54000,"side":"Buy","size":8339},{"price":"5.405","symbol":"DOTUSD","id":54050,"side":"Buy","size":23668},{"price":"5.410","symbol":"DOTUSD","id":54100,"side":"Buy","size":2967},{"price":"5.415","symbol":"DOTUSD","id":54150,"side":"Buy","size":37829},{"price":"5.420","symbol":"DOTUSD","id":54200,"side":"Buy","size":2989},{"price":"5.425","symbol":"DOTUSD","id":54250,"side":"Buy","size":2869},{"price":"5.430","symbol":"DOTUSD","id":54300,"side":"Buy","size":3006},{"price":"5.435","symbol":"DOTUSD","id":54350,"side":"Buy","size":3807},{"price":"5.440","symbol":"DOTUSD","id":54400,"side":"Buy","size":5311},{"price":"5.445","symbol":"DOTUSD","id":54450,"side":"Buy","size":237522},{"price":"5.450","symbol":"DOTUSD","id":54500,"side":"Buy","size":17524},{"price":"5.455","symbol":"DOTUSD","id":54550,"side":"Buy","size":24051},{"price":"5.460","symbol":"DOTUSD","id":54600,"side":"Buy","size":13226},{"price":"5.465","symbol":"DOTUSD","id":54650,"side":"Buy","size":52388},{"price":"5.470","symbol":"DOTUSD","id":54700,"side":"Buy","size":24361},{"price":"5.475","symbol":"DOTUSD","id":54750,"side":"Sell","size":18877},{"price":"5.480","symbol":"DOTUSD","id":54800,"side":"Sell","size":30629},{"price":"5.485","symbol":"DOTUSD","id":54850,"side":"Sell","size":23514},{"price":"5.490","symbol":"DOTUSD","id":54900,"side":"Sell","size":15846},{"price":"5.495","symbol":"DOTUSD","id":54950,"side":"Sell","size":193599},{"price":"5.500","symbol":"DOTUSD","id":55000,"side":"Sell","size":42172},{"price":"5.505","symbol":"DOTUSD","id":55050,"side":"Sell","size":17718},{"price":"5.510","symbol":"DOTUSD","id":55100,"side":"Sell","size":2467},{"price":"5.515","symbol":"DOTUSD","id":55150,"side":"Sell","size":4974},{"price":"5.520","symbol":"DOTUSD","id":55200,"side":"Sell","size":2502},{"price":"5.525","symbol":"DOTUSD","id":55250,"side":"Sell","size":3928},{"price":"5.530","symbol":"DOTUSD","id":55300,"side":"Sell","size":22733},{"price":"5.535","symbol":"DOTUSD","id":55350,"side":"Sell","size":2467},{"price":"5.540","symbol":"DOTUSD","id":55400,"side":"Sell","size":27088},{"price":"5.545","symbol":"DOTUSD","id":55450,"side":"Sell","size":2468},{"price":"5.550","symbol":"DOTUSD","id":55500,"side":"Sell","size":18106},{"price":"5.555","symbol":"DOTUSD","id":55550,"side":"Sell","size":3438},{"price":"5.560","symbol":"DOTUSD","id":55600,"side":"Sell","size":47962},{"price":"5.565","symbol":"DOTUSD","id":55650,"side":"Sell","size":2468},{"price":"5.570","symbol":"DOTUSD","id":55700,"side":"Sell","size":38260},{"price":"5.575","symbol":"DOTUSD","id":55750,"side":"Sell","size":2503},{"price":"5.580","symbol":"DOTUSD","id":55800,"side":"Sell","size":23690},{"price":"5.585","symbol":"DOTUSD","id":55850,"side":"Sell","size":2467},{"price":"5.590","symbol":"DOTUSD","id":55900,"side":"Sell","size":17300},{"price":"5.595","symbol":"DOTUSD","id":55950,"side":"Sell","size":2467},{"price":"5.600","symbol":"DOTUSD","id":56000,"side":"Sell","size":15195},{"price":"5.605","symbol":"DOTUSD","id":56050,"side":"Sell","size":550},{"price":"5.610","symbol":"DOTUSD","id":56100,"side":"Sell","size":17474},{"price":"5.620","symbol":"DOTUSD","id":56200,"side":"Sell","size":28297},{"price":"5.625","symbol":"DOTUSD","id":56250,"side":"Sell","size":441},{"price":"5.640","symbol":"DOTUSD","id":56400,"side":"Sell","size":6667},{"price":"5.650","symbol":"DOTUSD","id":56500,"side":"Sell","size":515},{"price":"5.655","symbol":"DOTUSD","id":56550,"side":"Sell","size":22},{"price":"5.660","symbol":"DOTUSD","id":56600,"side":"Sell","size":61},{"price":"5.675","symbol":"DOTUSD","id":56750,"side":"Sell","size":201},{"price":"5.680","symbol":"DOTUSD","id":56800,"side":"Sell","size":464},{"price":"5.695","symbol":"DOTUSD","id":56950,"side":"Sell","size":992},{"price":"5.700","symbol":"DOTUSD","id":57000,"side":"Sell","size":710},{"price":"5.710","symbol":"DOTUSD","id":57100,"side":"Sell","size":7},{"price":"5.715","symbol":"DOTUSD","id":57150,"side":"Sell","size":1},{"price":"5.720","symbol":"DOTUSD","id":57200,"side":"Sell","size":1},{"price":"5.735","symbol":"DOTUSD","id":57350,"side":"Sell","size":500},{"price":"5.740","symbol":"DOTUSD","id":57400,"side":"Sell","size":150},{"price":"5.745","symbol":"DOTUSD","id":57450,"side":"Sell","size":542},{"price":"5.750","symbol":"DOTUSD","id":57500,"side":"Sell","size":1675},{"price":"5.760","symbol":"DOTUSD","id":57600,"side":"Sell","size":5},{"price":"5.765","symbol":"DOTUSD","id":57650,"side":"Sell","size":8261},{"price":"5.775","symbol":"DOTUSD","id":57750,"side":"Sell","size":4},{"price":"5.790","symbol":"DOTUSD","id":57900,"side":"Sell","size":1},{"price":"5.795","symbol":"DOTUSD","id":57950,"side":"Sell","size":1160},{"price":"5.800","symbol":"DOTUSD","id":58000,"side":"Sell","size":501},{"price":"5.840","symbol":"DOTUSD","id":58400,"side":"Sell","size":500},{"price":"5.845","symbol":"DOTUSD","id":58450,"side":"Sell","size":100},{"price":"5.850","symbol":"DOTUSD","id":58500,"side":"Sell","size":501},{"price":"5.855","symbol":"DOTUSD","id":58550,"side":"Sell","size":10},{"price":"5.860","symbol":"DOTUSD","id":58600,"side":"Sell","size":10},{"price":"5.865","symbol":"DOTUSD","id":58650,"side":"Sell","size":41},{"price":"5.880","symbol":"DOTUSD","id":58800,"side":"Sell","size":255},{"price":"5.885","symbol":"DOTUSD","id":58850,"side":"Sell","size":40},{"price":"5.890","symbol":"DOTUSD","id":58900,"side":"Sell","size":115},{"price":"5.895","symbol":"DOTUSD","id":58950,"side":"Sell","size":115},{"price":"5.900","symbol":"DOTUSD","id":59000,"side":"Sell","size":1175},{"price":"5.910","symbol":"DOTUSD","id":59100,"side":"Sell","size":1},{"price":"5.920","symbol":"DOTUSD","id":59200,"side":"Sell","size":500},
{"price":"5.925","symbol":"DOTUSD","id":59250,"side":"Sell","size":1},{"price":"5.930","symbol":"DOTUSD","id":59300,"side":"Sell","size":310},{"price":"5.940","symbol":"DOTUSD","id":59400,"side":"Sell","size":941},{"price":"5.945","symbol":"DOTUSD","id":59450,"side":"Sell","size":101},{"price":"5.950","symbol":"DOTUSD","id":59500,"side":"Sell","size":501},{"price":"5.955","symbol":"DOTUSD","id":59550,"side":"Sell","size":1},{"price":"5.980","symbol":"DOTUSD","id":59800,"side":"Sell","size":10},{"price":"5.995","symbol":"DOTUSD","id":59950,"side":"Sell","size":1100},{"price":"6.000","symbol":"DOTUSD","id":60000,"side":"Sell","size":797},{"price":"6.015","symbol":"DOTUSD","id":60150,"side":"Sell","size":1},{"price":"6.030","symbol":"DOTUSD","id":60300,"side":"Sell","size":100},{"price":"6.035","symbol":"DOTUSD","id":60350,"side":"Sell","size":142},{"price":"6.045","symbol":"DOTUSD","id":60450,"side":"Sell","size":100},{"price":"6.050","symbol":"DOTUSD","id":60500,"side":"Sell","size":60},{"price":"6.055","symbol":"DOTUSD","id":60550,"side":"Sell","size":2},{"price":"6.065","symbol":"DOTUSD","id":60650,"side":"Sell","size":61},{"price":"6.090","symbol":"DOTUSD","id":60900,"side":"Sell","size":1},{"price":"6.095","symbol":"DOTUSD","id":60950,"side":"Sell","size":100},{"price":"6.100","symbol":"DOTUSD","id":61000,"side":"Sell","size":2010},{"price":"6.120","symbol":"DOTUSD","id":61200,"side":"Sell","size":5},{"price":"6.125","symbol":"DOTUSD","id":61250,"side":"Sell","size":1},{"price":"6.140","symbol":"DOTUSD","id":61400,"side":"Sell","size":1500},{"price":"6.145","symbol":"DOTUSD","id":61450,"side":"Sell","size":100},{"price":"6.150","symbol":"DOTUSD","id":61500,"side":"Sell","size":100},{"price":"6.190","symbol":"DOTUSD","id":61900,"side":"Sell","size":1},{"price":"6.195","symbol":"DOTUSD","id":61950,"side":"Sell","size":100},{"price":"6.200","symbol":"DOTUSD","id":62000,"side":"Sell","size":60},{"price":"6.205","symbol":"DOTUSD","id":62050,"side":"Sell","size":40},{"price":"6.210","symbol":"DOTUSD","id":62100,"side":"Sell","size":15},{"price":"6.220","symbol":"DOTUSD","id":62200,"side":"Sell","size":10},{"price":"6.245","symbol":"DOTUSD","id":62450,"side":"Sell","size":100},{"price":"6.250","symbol":"DOTUSD","id":62500,"side":"Sell","size":5},{"price":"6.290","symbol":"DOTUSD","id":62900,"side":"Sell","size":15},{"price":"6.295","symbol":"DOTUSD","id":62950,"side":"Sell","size":1100},{"price":"6.300","symbol":"DOTUSD","id":63000,"side":"Sell","size":100},{"price":"6.315","symbol":"DOTUSD","id":63150,"side":"Sell","size":1},{"price":"6.340","symbol":"DOTUSD","id":63400,"side":"Sell","size":10},{"price":"6.345","symbol":"DOTUSD","id":63450,"side":"Sell","size":110},{"price":"6.375","symbol":"DOTUSD","id":63750,"side":"Sell","size":1},{"price":"6.380","symbol":"DOTUSD","id":63800,"side":"Sell","size":5},{"price":"6.395","symbol":"DOTUSD","id":63950,"side":"Sell","size":100},{"price":"6.400","symbol":"DOTUSD","id":64000,"side":"Sell","size":100},{"price":"6.445","symbol":"DOTUSD","id":64450,"side":"Sell","size":100},{"price":"6.450","symbol":"DOTUSD","id":64500,"side":"Sell","size":250},{"price":"6.460","symbol":"DOTUSD","id":64600,"side":"Sell","size":10},{"price":"6.490","symbol":"DOTUSD","id":64900,"side":"Sell","size":300},{"price":"6.495","symbol":"DOTUSD","id":64950,"side":"Sell","size":1100},{"price":"6.510","symbol":"DOTUSD","id":65100,"side":"Sell","size":5},{"price":"6.525","symbol":"DOTUSD","id":65250,"side":"Sell","size":15},{"price":"6.545","symbol":"DOTUSD","id":65450,"side":"Sell","size":100},{"price":"6.550","symbol":"DOTUSD","id":65500,"side":"Sell","size":500},{"price":"6.565","symbol":"DOTUSD","id":65650,"side":"Sell","size":1},{"price":"6.580","symbol":"DOTUSD","id":65800,"side":"Sell","size":10},{"price":"6.595","symbol":"DOTUSD","id":65950,"side":"Sell","size":110},{"price":"6.600","symbol":"DOTUSD","id":66000,"side":"Sell","size":5},{"price":"6.610","symbol":"DOTUSD","id":66100,"side":"Sell","size":1},{"price":"6.645","symbol":"DOTUSD","id":66450,"side":"Sell","size":100},{"price":"6.650","symbol":"DOTUSD","id":66500,"side":"Sell","size":5},{"price":"6.660","symbol":"DOTUSD","id":66600,"side":"Sell","size":200},{"price":"6.680","symbol":"DOTUSD","id":66800,"side":"Sell","size":2},{"price":"6.695","symbol":"DOTUSD","id":66950,"side":"Sell","size":100},{"price":"6.700","symbol":"DOTUSD","id":67000,"side":"Sell","size":10},{"price":"6.745","symbol":"DOTUSD","id":67450,"side":"Sell","size":100},{"price":"6.750","symbol":"DOTUSD","id":67500,"side":"Sell","size":200},{"price":"6.790","symbol":"DOTUSD","id":67900,"side":"Sell","size":5},{"price":"6.795","symbol":"DOTUSD","id":67950,"side":"Sell","size":100},{"price":"6.800","symbol":"DOTUSD","id":68000,"side":"Sell","size":2500},{"price":"6.815","symbol":"DOTUSD","id":68150,"side":"Sell","size":1},{"price":"6.820","symbol":"DOTUSD","id":68200,"side":"Sell","size":10},{"price":"6.830","symbol":"DOTUSD","id":68300,"side":"Sell","size":2},{"price":"6.840","symbol":"DOTUSD","id":68400,"side":"Sell","size":15},{"price":"6.845","symbol":"DOTUSD","id":68450,"side":"Sell","size":100},{"price":"6.850","symbol":"DOTUSD","id":68500,"side":"Sell","size":15},{"price":"6.875","symbol":"DOTUSD","id":68750,"side":"Sell","size":1},{"price":"6.895","symbol":"DOTUSD","id":68950,"side":"Sell","size":100},{"price":"6.900","symbol":"DOTUSD","id":69000,"side":"Sell","size":1627},{"price":"6.930","symbol":"DOTUSD","id":69300,"side":"Sell","size":755},{"price":"6.940","symbol":"DOTUSD","id":69400,"side":"Sell","size":11},{"price":"6.945","symbol":"DOTUSD","id":69450,"side":"Sell","size":100},{"price":"6.950","symbol":"DOTUSD","id":69500,"side":"Sell","size":20},{"price":"6.990","symbol":"DOTUSD","id":69900,"side":"Sell","size":10},{"price":"6.995","symbol":"DOTUSD","id":69950,"side":"Sell","size":100},{"price":"7.000","symbol":"DOTUSD","id":70000,"side":"Sell","size":111},{"price":"7.035","symbol":"DOTUSD","id":70350,"side":"Sell","size":200},{"price":"7.045","symbol":"DOTUSD","id":70450,"side":"Sell","size":100},{"price":"7.060","symbol":"DOTUSD","id":70600,"side":"Sell","size":10},{"price":"7.070","symbol":"DOTUSD","id":70700,"side":"Sell","size":5},{"price":"7.080","symbol":"DOTUSD","id":70800,"side":"Sell","size":17},{"price":"7.095","symbol":"DOTUSD","id":70950,"side":"Sell","size":100},{"price":"7.125","symbol":"DOTUSD","id":71250,"side":"Sell","size":1},{"price":"7.145","symbol":"DOTUSD","id":71450,"side":"Sell","size":100},{"price":"7.160","symbol":"DOTUSD","id":71600,"side":"Sell","size":15},{"price":"7.170","symbol":"DOTUSD","id":71700,"side":"Sell","size":1000},{"price":"7.180","symbol":"DOTUSD","id":71800,"side":"Sell","size":10},{"price":"7.195","symbol":"DOTUSD","id":71950,"side":"Sell","size":100},{"price":"7.220","symbol":"DOTUSD","id":72200,"side":"Sell","size":128},{"price":"7.245","symbol":"DOTUSD","id":72450,"side":"Sell","size":100},{"price":"7.295","symbol":"DOTUSD","id":72950,"side":"Sell","size":100},{"price":"7.300","symbol":"DOTUSD","id":73000,"side":"Sell","size":10},{"price":"7.315","symbol":"DOTUSD","id":73150,"side":"Sell","size":1},{"price":"7.345","symbol":"DOTUSD","id":73450,"side":"Sell","size":100},{"price":"7.350","symbol":"DOTUSD","id":73500,"side":"Sell","size":2110},{"price":"7.355","symbol":"DOTUSD","id":73550,"side":"Sell","size":794},{"price":"7.370","symbol":"DOTUSD","id":73700,"side":"Sell","size":6},{"price":"7.375","symbol":"DOTUSD","id":73750,"side":"Sell","size":1},{"price":"7.380","symbol":"DOTUSD","id":73800,"side":"Sell","size":100},{"price":"7.395","symbol":"DOTUSD","id":73950,"side":"Sell","size":100},{"price":"7.400","symbol":"DOTUSD","id":74000,"side":"Sell","size":100},{"price":"7.420","symbol":"DOTUSD","id":74200,"side":"Sell","size":11},{"price":"7.445","symbol":"DOTUSD","id":74450,"side":"Sell","size":100},{"price":"7.470","symbol":"DOTUSD","id":74700,"side":"Sell","size":1},{"price":"7.490","symbol":"DOTUSD","id":74900,"side":"Sell","size":30},{"price":"7.495","symbol":"DOTUSD","id":74950,"side":"Sell","size":100},{"price":"7.500","symbol":"DOTUSD","id":75000,"side":"Sell","size":10},{"price":"7.520","symbol":"DOTUSD","id":75200,"side":"Sell","size":6},{"price":"7.540","symbol":"DOTUSD","id":75400,"side":"Sell","size":10},{"price":"7.545","symbol":"DOTUSD","id":75450,"side":"Sell","size":100},{"price":"7.570","symbol":"DOTUSD","id":75700,"side":"Sell","size":1},{"price":"7.575","symbol":"DOTUSD","id":75750,"side":"Sell","size":1},{"price":"7.580","symbol":"DOTUSD","id":75800,"side":"Sell","size":250},{"price":"7.620","symbol":"DOTUSD","id":76200,"side":"Sell","size":1},{"price":"7.660","symbol":"DOTUSD","id":76600,"side":"Sell","size":10},{"price":"7.670","symbol":"DOTUSD","id":76700,"side":"Sell","size":1},{"price":"7.680","symbol":"DOTUSD","id":76800,"side":"Sell","size":5},{"price":"7.700","symbol":"DOTUSD","id":77000,"side":"Sell","size":36336},{"price":"7.720","symbol":"DOTUSD","id":77200,"side":"Sell","size":1},
{"price":"7.770","symbol":"DOTUSD","id":77700,"side":"Sell","size":756},{"price":"7.780","symbol":"DOTUSD","id":77800,"side":"Sell","size":10},{"price":"7.790","symbol":"DOTUSD","id":77900,"side":"Sell","size":60},{"price":"7.820","symbol":"DOTUSD","id":78200,"side":"Sell","size":1},{"price":"7.830","symbol":"DOTUSD","id":78300,"side":"Sell","size":43},{"price":"7.840","symbol":"DOTUSD","id":78400,"side":"Sell","size":5},{"price":"7.845","symbol":"DOTUSD","id":78450,"side":"Sell","size":60},{"price":"7.850","symbol":"DOTUSD","id":78500,"side":"Sell","size":50},{"price":"7.870","symbol":"DOTUSD","id":78700,"side":"Sell","size":1},{"price":"7.875","symbol":"DOTUSD","id":78750,"side":"Sell","size":1}],"cross_seq":1394439647,"timestamp_e6":1689445779707233}`)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	var resp WsUSDTOrderbook
	err := json.Unmarshal(orderbookMarshalingTestData, &resp)
	if err != nil {
		t.Error(err)
	}
}

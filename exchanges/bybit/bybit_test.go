package bybit

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	s := time.Now().Add(-time.Hour)
	e := time.Now()
	if mockTests {
		s = time.Unix(1691897100, 0).Round(kline.FiveMin.Duration())
		e = time.Unix(1691907100, 0).Round(kline.FiveMin.Duration())
	}
	_, err := b.GetKlines(context.Background(), "BTCUSDT", "5m", 2000, s, e)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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
	err := b.wsHandleData(pressXToJSON)
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

	var ts bybitTimeSec
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

	var tms bybitTimeMilliSec
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
	tInNanoSec := time.Now().UnixNano()

	var tns bybitTimeNanoSec
	err := tns.UnmarshalJSON([]byte(strconv.FormatInt(tInNanoSec, 10)))
	if err != nil {
		t.Fatal(err)
	}

	if !time.Unix(0, tInNanoSec).Equal(tns.Time()) {
		t.Errorf("TestTimeNanoSecUnmarshalJSON failed")
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

	var pairs currency.Pairs
	if mockTests {
		var pair2 currency.Pair
		pair2, err = currency.NewPairFromString("BTCUSD-U23")
		if err != nil {
			t.Fatal(err)
		}
		pairs = pairs.Add(pair2)
	} else {
		// Futures update dynamically, so fetch the available tradable futures for this test
		pairs, err = b.FetchTradablePairs(context.Background(), asset.Futures)
		if err != nil {
			t.Fatal(err)
		}
		// Needs to be set before calling extractCurrencyPair
		if err = b.SetPairs(pairs, asset.Futures, true); err != nil {
			t.Fatal(err)
		}
	}

	_, err = b.UpdateTicker(context.Background(), pairs[0], asset.Futures)
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
	err := b.UpdateTradablePairs(context.Background(), true)
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
	e := time.Now()
	s := e.AddDate(0, 0, -3)
	if mockTests {
		s = time.Unix(1691897100, 0).Truncate(kline.OneDay.Duration())
		e = time.Unix(1692007100, 0).Truncate(kline.OneDay.Duration())
	}

	_, err = b.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, s, e)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), pair, asset.USDTMarginedFutures, kline.OneDay, s, e)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricCandles(context.Background(), pair1, asset.CoinMarginedFutures, kline.OneHour, s, e)
	if err != nil {
		t.Error(err)
	}

	enabled, err := b.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	var pair2 currency.Pair
	if mockTests {
		pair2, err = currency.NewPairFromString("BTCUSD-U23")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		pair2 = enabled[0]
	}

	_, err = b.GetHistoricCandles(context.Background(), pair2, asset.Futures, kline.OneHour, s, e)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), pair3, asset.USDCMarginedFutures, kline.OneDay, s, e)
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
	e := time.Now()
	s := e.AddDate(0, 0, -3)
	if mockTests {
		s = time.Unix(1691897100, 0).Truncate(kline.OneDay.Duration())
		e = time.Unix(1692007100, 0).Truncate(kline.OneDay.Duration())
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneDay, s, e)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair, asset.USDTMarginedFutures, kline.OneDay, s, e)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricCandlesExtended(context.Background(), pair1, asset.CoinMarginedFutures, kline.OneHour, s, e)
	if err != nil {
		t.Error(err)
	}

	enabled, err := b.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	var pair2 currency.Pair
	if mockTests {
		pair2, err = currency.NewPairFromString("BTCUSD-U23")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		pair2 = enabled[0]
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair2, asset.Futures, kline.OneHour, s, e)
	if err != nil {
		t.Error(err)
	}

	pair3, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(context.Background(), pair3, asset.USDCMarginedFutures, kline.OneDay, s, e)
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
	s := time.Now().Add(-time.Hour)
	if mockTests {
		s = time.Unix(1691897100, 0)
	}
	_, err = b.GetUSDCKlines(context.Background(), pair, "5", s, 0)
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
	s := time.Now().Add(-time.Hour)
	if mockTests {
		s = time.Unix(1691897100, 0)
	}
	_, err = b.GetUSDCMarkPriceKlines(context.Background(), pair, "5", s, 0)
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
	s := time.Now().Add(-time.Hour)
	if mockTests {
		s = time.Unix(1691897100, 0)
	}
	_, err = b.GetUSDCIndexPriceKlines(context.Background(), pair, "5", s, 0)
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
	s := time.Now().Add(-time.Hour)
	if mockTests {
		s = time.Unix(1692077100, 0)
	}

	_, err = b.GetUSDCPremiumIndexKlines(context.Background(), pair, "5", s, 0)
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
	if mockTests {
		t.Skip("test it not relevant in a mock setting")
	}
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

	var err error
	if !mockTests {
		_, err = b.GetTickersV5(context.Background(), "bruh", "", "")
		if err != nil && err.Error() != "Illegal category" {
			t.Error(err)
		}
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

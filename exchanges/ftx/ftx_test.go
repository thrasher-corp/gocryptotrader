package ftx

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	spotPair                = "FTT/BTC"
	futuresPair             = "LEO-0327"
	testToken               = "ADAMOON"
)

var f FTX

func TestMain(m *testing.M) {
	f.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("FTX")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = f.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return f.ValidateAPICredentials()
}

// Implement tests for API endpoints below

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := f.GetMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarket(t *testing.T) {
	t.Parallel()
	_, err := f.GetMarket(spotPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := f.GetOrderbook(spotPair, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := f.GetTrades(spotPair, time.Time{}, time.Time{}, 5)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTrades(spotPair, time.Unix(1559881511, 0), time.Unix(1559901511, 0), 5)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTrades(spotPair, time.Unix(1559901511, 0), time.Unix(1559881511, 0), 5)
	if err == nil {
		t.Error(err)
	}
}

func TestGetHistoricalData(t *testing.T) {
	t.Parallel()
	_, err := f.GetHistoricalData(spotPair, "86400", "5", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetHistoricalData(spotPair, "86400", "5", time.Unix(1559881511, 0), time.Unix(1559901511, 0))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetHistoricalData(spotPair, "86400", "5", time.Unix(1559901511, 0), time.Unix(1559881511, 0))
	if err == nil {
		t.Error(err)
	}
}

func TestGetFutures(t *testing.T) {
	t.Parallel()
	_, err := f.GetFutures()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuture(t *testing.T) {
	t.Parallel()
	_, err := f.GetFuture(futuresPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := f.GetFutureStats(futuresPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	_, err := f.GetFundingRates()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetPositions()
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAccountLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	err := f.ChangeAccountLeverage(50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetCoins()
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllWalletBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAllWalletBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchDepositAddress("TUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchDepositHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchWithdrawalHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.Withdraw("BTC", "38eyTMFHvo5UjPR91zwYYKuCtdF2uhtdxS", "", "", "957378", 0.01)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOpenOrders(spotPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchOrderHistory(spotPair, time.Time{}, time.Time{}, "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(spotPair, time.Unix(1559881511, 0), time.Unix(1559901511, 0), "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(spotPair, time.Unix(1559901511, 0), time.Unix(1559881511, 0), "2")
	if err == nil {
		t.Error(err)
	}
}

func TestGetOpenTriggerOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOpenTriggerOrders(spotPair, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderTriggers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTriggerOrderTriggers("alkdjfkajdsf")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTriggerOrderHistory(spotPair, time.Time{}, time.Time{}, order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(spotPair, time.Unix(1559881511, 0), time.Unix(1559901511, 0), order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(spotPair, time.Unix(1559901511, 0), time.Unix(1559881511, 0), order.Buy.Lower(), "stop", "1")
	if err == nil {
		t.Error(err)
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.Order(spotPair, order.Buy.Lower(), "limit", "", "", "", "", 0.0001, 500)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.TriggerOrder(spotPair, order.Buy.Lower(), order.Stop.Lower(), "", "", 500, 0.0004, 0.0001, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.DeleteOrder("testing123")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.DeleteOrderByClientID("clientID123")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.DeleteTriggerOrder("triggerOrder123")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetFills(spotPair, "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(spotPair, "", time.Unix(1559881511, 0), time.Unix(1559901511, 0))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(spotPair, "", time.Unix(1559901511, 0), time.Unix(1559881511, 0))
	if err == nil {
		t.Error(err)
	}
}

func TestGetFundingPayments(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetFundingPayments(time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(time.Unix(1559881511, 0), time.Unix(1559901511, 0), "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(time.Unix(1559901511, 0), time.Unix(1559881511, 0), "")
	if err == nil {
		t.Error(err)
	}
}

func TestListLeveragedTokens(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLeveragedTokens()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTokenInfo(t *testing.T) {
	f.Verbose = true
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTokenInfo("")
	if err != nil {
		t.Error(err)
	}
}

func TestListLTBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestListLTCreations(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTCreations()
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTCreation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.RequestLTCreation(testToken, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTRedemptions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTRedemptions(testToken, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuoteRequests(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetQuoteRequests()
	if err != nil {
		t.Error(err)
	}
}

func TestGetYourQuoteRequests(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetYourQuoteRequests()
	if err != nil {
		t.Error(err)
	}
}

func TestCreateQuoteRequest(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.CreateQuoteRequest(strings.ToUpper(currency.BTC.String()), "call", order.Buy.Lower(), strconv.FormatInt(time.Now().AddDate(0, 0, 3).UnixNano()/1000000, 10), "", 0.1, 10, 5, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.DeleteQuote("testing123")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotesForYourQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetQuotesForYourQuote("testing123")
	if err != nil {
		t.Error(err)
	}
}

func TestMakeQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.MakeQuote("testing123", "5")
	if err != nil {
		t.Error(err)
	}
}

func TestMyQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.MyQuotes()
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMyQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.DeleteMyQuote("testing123")
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.AcceptQuote("testing123")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountOptionsInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAccountOptionsInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetOptionsPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOptionsPositions()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicOptionsTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetPublicOptionsTrades(time.Time{}, time.Time{}, "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetPublicOptionsTrades(time.Unix(1559881511, 0), time.Unix(1559901511, 0), "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetPublicOptionsTrades(time.Unix(1559901511, 0), time.Unix(1559881511, 0), "5")
	if err == nil {
		t.Error(err)
	}
}

func TestGetOptionsFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOptionsFills(time.Time{}, time.Time{}, "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOptionsFills(time.Unix(1559881511, 0), time.Unix(1559901511, 0), "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOptionsFills(time.Unix(1559901511, 0), time.Unix(1559881511, 0), "5")
	if err == nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	_, err := f.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	_, err := f.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	var orderReq order.GetOrdersRequest
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	orderReq.Pairs = append(orderReq.Pairs, cp)
	_, err := f.GetActiveOrders(&orderReq)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	var orderReq order.GetOrdersRequest
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	orderReq.Pairs = append(orderReq.Pairs, cp)
	_, err := f.GetOrderHistory(&orderReq)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAccountHoldings(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.UpdateAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.FetchAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	var x exchange.FeeBuilder
	x.PurchasePrice = 10
	x.Amount = 1
	x.IsMaker = true
	var a float64
	var err error
	if areTestAPIKeysSet() {
		a, err = f.GetFee(&x)
		if err != nil {
			t.Error(err)
		}
		if a != 0.0039 {
			t.Errorf("incorrect maker fee value")
		}
	}
	x.IsMaker = false
	if areTestAPIKeysSet() {
		a, err = f.GetFee(&x)
		if err != nil {
			t.Error(err)
		}
		if a != 0.00865 {
			t.Errorf("incorrect taker fee value")
		}
	}
	x.FeeType = exchange.OfflineTradeFee
	a, err = f.GetFee(&x)
	if err != nil {
		t.Error(err)
	}
	if a != 0.007 {
		t.Errorf("incorrect offline taker fee value")
	}
	x.IsMaker = true
	a, err = f.GetFee(&x)
	if err != nil {
		t.Error(err)
	}
	if a != 0.002 {
		t.Errorf("incorrect offline maker fee value")
	}
}

func TestGetOfflineTradingFee(t *testing.T) {
	var f exchange.FeeBuilder
	f.PurchasePrice = 10
	f.Amount = 1
	f.IsMaker = true
	fee := getOfflineTradeFee(&f)
	if fee != 0.002 {
		t.Errorf("incorrect offline maker fee")
	}
	f.IsMaker = false
	fee = getOfflineTradeFee(&f)
	if fee != 0.007 {
		t.Errorf("incorrect offline taker fee")
	}
}

func TestGetOrderStatus(t *testing.T) {
	_, err := f.GetOrderStatus("testID")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderStatusByClientID(t *testing.T) {
	_, err := f.GetOrderStatusByClientID("testID")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTRedemption(t *testing.T) {
	_, err := f.RequestLTRedemption("ADA-PERP", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	var request withdraw.Request
	request.Amount = 5
	request.Currency = currency.NewCode("FTT")
	request.Crypto.Address = "testaddress123"
	request.Crypto.AddressTag = "testtag123"
	request.OneTimePassword = 123456
	request.TradePassword = "incorrectTradePassword"
	_, err := f.WithdrawCryptocurrencyFunds(&request)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetDepositAddress(currency.NewCode("FTT"), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := f.GetFundingHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	currencyPair := currency.NewPairFromString(spotPair)
	start := time.Date(2019, 11, 12, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 5)
	_, err := f.GetHistoricCandles(currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParsingWSFillData(t *testing.T) {
	t.Parallel()
	f.Verbose = true
	data := []byte(`{
		  "channel": "fills",
		  "data": {
			"fee": 78.05799225,
			"feeRate": 0.0014,
			"future": "BTC-PERP",
			"id": 7828307,
			"liquidity": "taker",
			"market": "BTC-PERP",
			"orderId": 38065410,
			"tradeId": 19129310,
			"price": 3723.75,
			"side": "buy",
			"size": 14.973,
			"time": "2019-05-07T16:40:58.358438+00:00",
			"type": "order"
		  },
		  "type": "update"
		}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestParsingWSTradesData(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"channel": "trades",
		"market": "BTC-PERP",
		"type": "update",
		"data": [
			{
				"id": 44200173,
				"price": 9761.0,
				"size": 0.0008,
				"side": "buy",
				"liquidation": false,
				"time": "2020-05-15T01:10:04.369194+00:00"
			}
		]
	}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestParsingWSTickerData(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"channel": "ticker", 
		"market": "BTC-PERP", 
		"type": "update", 
		"data": {
			"bid": 9760.5, 
			"ask": 9761.0, 
			"bidSize": 3.36, 
			"askSize": 71.8484, 
			"last": 9761.0, 
			"time": 1589505004.4237103
		}
	}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestParsingWSOrdersData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	data := []byte(`{
		"channel": "orders",
		"data": {
		  "id": 24852229,
		  "clientId": null,
		  "market": "BTC-PERP",
		  "type": "limit",
		  "side": "buy",
		  "size": 42353.0,
		  "price": 0.2977,
		  "reduceOnly": false,
		  "ioc": false,
		  "postOnly": false,
		  "status": "closed",
		  "filledSize": 0.0,
		  "remainingSize": 0.0,
		  "avgFillPrice": 0.2978
		},
		"type": "update"
	  }`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestCalcPartialOBChecksum(t *testing.T) {
	var ob WsOrderbookData
	b := [][2]float64{[2]float64{5000.5, 10}, [2]float64{4995.0, 5}}
	a := [][2]float64{[2]float64{5001.0, 6}}
	ob.Asks = a
	ob.Bids = b
	fmt.Println(ob.Asks)
	fmt.Println(ob.Bids)
	output := f.CalcPartialOBChecksum(&ob)
	fmt.Println(output)
}

func TestCalcOBChecksum(t *testing.T) {
	var ob orderbook.Base
	a := []orderbook.Item{orderbook.Item{Price: 5001.0, Amount: 6}}
	b := []orderbook.Item{orderbook.Item{Price: 5000.5, Amount: 10},
		orderbook.Item{Price: 4995.0, Amount: 5}}
	ob.Asks = a
	ob.Bids = b
	retVal := f.CalcOBChecksum(&ob)
	fmt.Println(retVal)
}

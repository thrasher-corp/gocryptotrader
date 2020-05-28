package ftx

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/core"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
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
	btcusd                  = "BTC/USD"
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
	f.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	f.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
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
	_, err := f.Withdraw("BTC", core.BitcoinDonationAddress, "", "", "957378", 0.0009)
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
	_, err := f.GetTriggerOrderTriggers("1031")
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
	_, err := f.DeleteOrder("1031")
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
	_, err := f.DeleteTriggerOrder("1031")
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
	_, err := f.GetFundingPayments(time.Unix(1559881511, 0), time.Unix(1559901511, 0), "")
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
	_, err := f.DeleteQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotesForYourQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetQuotesForYourQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestMakeQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.MakeQuote("1031", "5")
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
	_, err := f.DeleteMyQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.AcceptQuote("1031")
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
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOrderStatus("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderStatusByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOrderStatusByClientID("testID")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTRedemption(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
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
	var request = new(withdraw.Request)
	request.Amount = 5
	request.Currency = currency.NewCode("FTT")
	var cryptoData withdraw.CryptoRequest
	cryptoData.Address = "testaddress123"
	cryptoData.AddressTag = "testtag123"
	request.Crypto = &cryptoData
	request.OneTimePassword = 123456
	request.TradePassword = "incorrectTradePassword"
	_, err := f.WithdrawCryptocurrencyFunds(request)
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
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
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

func TestParsingWSOBData(t *testing.T) {
	data := []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "partial", "data": {"time": 1589855831.4606245, "checksum": 225973019, "bids": [[9602.0, 3.2903], [9601.5, 3.11], [9601.0, 2.1356], [9600.5, 3.0991], [9600.0, 8.014], [9599.5, 4.1571], [9599.0, 79.1846], [9598.5, 3.099], [9598.0, 3.985], [9597.5, 3.999], [9597.0, 16.4335], [9596.5, 4.006], [9596.0, 3.2596], [9595.0, 6.334], [9594.0, 3.5685], [9593.0, 14.2717], [9592.5, 0.5], [9591.0, 2.181], [9590.5, 40.4246], [9590.0, 1.0], [9589.0, 1.357], [9588.5, 0.4738], [9587.5, 0.15], [9587.0, 16.811], [9586.5, 1.2], [9586.0, 0.2], [9585.5, 1.0], [9584.5, 0.002], [9584.0, 1.51], [9583.5, 0.01], [9583.0, 1.4], [9582.5, 0.1], [9582.0, 24.7921], [9581.0, 2.087], [9580.5, 2.0], [9580.0, 0.1], [9579.0, 1.1588], [9578.0, 0.9477], [9577.5, 22.216], [9576.0, 0.2], [9574.0, 22.0], [9573.5, 1.0], [9572.0, 0.203], [9570.0, 0.1026], [9565.5, 5.5332], [9565.0, 27.5243], [9563.5, 2.6], [9562.0, 0.0175], [9561.0, 2.0085], [9552.0, 1.6], [9550.5, 27.3399], [9550.0, 0.1046], [9548.0, 0.0175], [9544.0, 4.8197], [9542.5, 26.5754], [9542.0, 0.003], [9541.0, 0.0549], [9540.0, 0.1984], [9537.5, 0.0008], [9535.5, 0.0105], [9535.0, 1.514], [9534.5, 36.5858], [9532.5, 4.7798], [9531.0, 40.6564], [9525.0, 0.001], [9523.5, 1.6], [9522.0, 0.0894], [9521.0, 0.315], [9520.5, 5.4525], [9520.0, 0.07], [9518.0, 0.034], [9517.5, 4.0], [9513.0, 0.0175], [9512.5, 15.6016], [9512.0, 32.7882], [9511.5, 0.0482], [9510.5, 0.0482], [9510.0, 0.2999], [9509.0, 2.0], [9508.5, 0.0482], [9506.0, 0.0416], [9505.5, 0.0492], [9505.0, 0.2], [9502.5, 0.01], [9502.0, 0.01], [9501.5, 0.0592], [9501.0, 0.001], [9500.0, 3.4913], [9499.5, 39.8683], [9498.0, 4.6108], [9497.0, 0.0481], [9492.0, 41.3559], [9490.0, 1.1104], [9488.0, 0.0105], [9486.0, 5.4443], [9485.5, 0.0482], [9484.0, 4.0], [9482.0, 0.25], [9481.5, 2.0], [9481.0, 8.1572]], "asks": [[9602.5, 3.0], [9603.0, 2.8979], [9603.5, 54.49], [9604.0, 5.9982], [9604.5, 3.028], [9605.0, 4.657], [9606.5, 5.2512], [9607.0, 4.003], [9607.5, 4.011], [9608.0, 13.7505], [9608.5, 3.994], [9609.0, 2.974], [9609.5, 3.002], [9612.0, 10.298], [9612.5, 13.455], [9613.5, 3.013], [9614.0, 2.02], [9614.5, 3.359], [9615.0, 21.2429], [9616.0, 0.5], [9616.5, 0.01], [9617.0, 2.182], [9617.5, 23.0223], [9618.0, 0.0623], [9618.5, 1.5795], [9619.0, 0.3065], [9620.0, 3.9], [9621.0, 1.5], [9622.0, 1.5], [9622.5, 1.216], [9625.0, 1.0], [9625.5, 0.9477], [9626.0, 0.05], [9628.5, 1.1588], [9629.0, 1.4], [9630.0, 4.2332], [9630.5, 1.228], [9631.0, 1.5], [9631.5, 0.0104], [9632.5, 26.7529], [9633.0, 0.25], [9638.0, 1.0], [9640.0, 0.2], [9641.0, 1.001], [9642.0, 0.0175], [9643.0, 0.25], [9643.5, 1.6], [9644.0, 31.4166], [9646.5, 41.6609], [9649.5, 0.2], [9653.5, 1.5], [9656.5, 1.6], [9657.0, 0.2], [9658.0, 1.5], [9659.5, 4.7804], [9660.5, 43.3405], [9665.5, 40.6564], [9670.0, 0.1034], [9671.5, 4.9098], [9674.0, 0.25], [9678.0, 15.6016], [9678.5, 1.5], [9681.0, 34.9683], [9683.0, 0.2], [9683.5, 5.3845], [9684.5, 5.087], [9685.0, 0.1032], [9686.5, 0.0075], [9689.0, 1.6], [9691.0, 34.7472], [9692.0, 0.001], [9694.0, 0.5], [9695.0, 0.0109], [9696.5, 4.825], [9700.0, 1.0595], [9701.5, 2.0], [9702.0, 0.011], [9702.5, 0.01], [9706.0, 1.2], [9708.0, 0.0175], [9710.0, 39.153], [9712.0, 48.6163], [9712.5, 1.5], [9713.0, 8.1572], [9715.5, 0.5021], [9716.5, 2.0], [9719.0, 0.0245], [9721.0, 0.5], [9724.0, 0.251], [9726.0, 0.12], [9727.5, 0.5075], [9730.0, 0.015], [9732.0, 58.5394], [9733.0, 0.001], [9734.0, 20.0], [9743.0, 0.06], [9750.0, 9.5], [9755.0, 52.4404], [9757.0, 48.6121], [9764.0, 0.015]], "action": "partial"}}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	// data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "update", "data": {"time": 1589855831.5128105, "checksum": 365946911, "bids": [[9596.0, 4.2656], [9512.0, 32.7912]], "asks": [[9613.5, 4.012], [9702.0, 0.021]], "action": "update"}}`)
	// err = f.wsHandleData([]byte(data))
	// if err != nil {
	// 	t.Error(err)
	// }
}

func TestGetOTCQuoteStatus(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOTCQuoteStatus(btcusd, "1")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestForQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.RequestForQuotes("BTC", "USD", 0.5)
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := f.AcceptOTCQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

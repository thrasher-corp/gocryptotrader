package ftx

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	subaccount              = ""
	canManipulateRealOrders = false
	spotPair                = "FTT/BTC"
	futuresPair             = "DOGE-PERP"
	testLeverageToken       = "ADAMOON"

	validFTTBTCStartTime   = 1565445600           // Sat Aug 10 2019 14:00:00 GMT+0000
	validFTTBTCEndTime     = 1565532000           // Sat Aug 10 2019 14:00:00 GMT+0000
	invalidFTTBTCStartTime = 1559881511           // Fri Jun 07 2019 04:25:11 GMT+0000
	invalidFTTBTCEndTime   = 1559901511           // Fri Jun 07 2019 09:58:31 GMT+0000
	authStartTime          = validFTTBTCStartTime // Adjust these to test auth requests
	authEndTime            = validFTTBTCEndTime
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

	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.Subaccount = subaccount
	if apiKey != "" && apiSecret != "" {
		// Only set auth to true when keys present as fee online calculation requires authentication
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	f.Websocket = sharedtestvalues.NewTestWebsocket()
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

func TestGetHistoricalIndex(t *testing.T) {
	t.Parallel()
	_, err := f.GetHistoricalIndex("BTC", 3600, time.Now().Add(-time.Hour*2), time.Now().Add(-time.Hour*1))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetHistoricalIndex("BTC", 3600, time.Time{}, time.Time{})
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
	// test empty market
	_, err := f.GetTrades("", 0, 0, 200)
	if err == nil {
		t.Error("empty market should return an error")
	}
	_, err = f.GetTrades(spotPair, validFTTBTCEndTime, validFTTBTCStartTime, 5)
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	// test optional params
	var trades []TradeData
	trades, err = f.GetTrades(spotPair, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 20 {
		t.Error("default limit should return 20 items")
	}
	trades, err = f.GetTrades(spotPair, validFTTBTCStartTime, validFTTBTCEndTime, 5)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	trades, err = f.GetTrades(spotPair, invalidFTTBTCStartTime, invalidFTTBTCEndTime, 5)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 0 {
		t.Error("invalid time range should return 0 items")
	}
}

func TestGetHistoricalData(t *testing.T) {
	t.Parallel()
	// test empty market
	_, err := f.GetHistoricalData("", 86400, 5, time.Time{}, time.Time{})
	if err == nil {
		t.Error("empty market should return an error")
	}
	// test empty resolution
	_, err = f.GetHistoricalData(spotPair, 0, 5, time.Time{}, time.Time{})
	if err == nil {
		t.Error("empty resolution should return an error")
	}
	_, err = f.GetHistoricalData(spotPair, 86400, 5, time.Unix(validFTTBTCEndTime, 0), time.Unix(validFTTBTCStartTime, 0))
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	var o []OHLCVData
	o, err = f.GetHistoricalData(spotPair, 86400, 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(o) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	o, err = f.GetHistoricalData(spotPair, 86400, 5, time.Unix(invalidFTTBTCStartTime, 0), time.Unix(invalidFTTBTCEndTime, 0))
	if err != nil {
		t.Error(err)
	}
	if len(o) != 0 {
		t.Error("invalid time range should return 0 items")
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
	_, err := f.GetFutureStats("BTC-PERP")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	// optional params
	_, err := f.GetFundingRates(time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingRates(time.Now().Add(-time.Hour), time.Now(), "BTC-PERP")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	// t.Parallel()
	// if !areTestAPIKeysSet() {
	// 	t.Skip()
	// }
	f.Verbose = true
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

func TestChangeAccountLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
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

func TestGetMarginBorrowRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginBorrowRates()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginLendingRates()
	if err != nil {
		t.Error(err)
	}
}

func TestMarginDailyBorrowedAmounts(t *testing.T) {
	t.Parallel()
	_, err := f.MarginDailyBorrowedAmounts()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginMarketInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginMarketInfo("BTC_USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginBorrowHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginBorrowHistory(tmNow.AddDate(0, 0, 1), tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err = f.GetMarginBorrowHistory(tmNow.AddDate(0, 0, -1), tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginMarketLendingHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginMarketLendingHistory(currency.USD, tmNow.AddDate(0, 0, 1), tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	_, err = f.GetMarginMarketLendingHistory(currency.USD, tmNow.AddDate(0, 0, -1), tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginLendingHistory(currency.USD, tmNow.AddDate(0, 0, 1), tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err = f.GetMarginLendingHistory(currency.USD, tmNow.AddDate(0, 0, -1), tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingOffers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginLendingOffers()
	if err != nil {
		t.Error(err)
	}
}

func TestGetLendingInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetLendingInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitLendingOffer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}
	if err := f.SubmitLendingOffer(currency.NewCode("bTc"), 0.1, 500); err != nil {
		t.Error(err)
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchDepositAddress(currency.NewCode("tUsD"))
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
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.Withdraw(currency.NewCode("bTc"), core.BitcoinDonationAddress, "", "", "957378", 0.0009)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOpenOrders("")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOpenOrders(spotPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchOrderHistory("", time.Time{}, time.Time{}, "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(spotPair, time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(spotPair, time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), "2")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestGetOpenTriggerOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	// optional params
	_, err := f.GetOpenTriggerOrders("", "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOpenTriggerOrders(spotPair, "")
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
	_, err := f.GetTriggerOrderHistory("", time.Time{}, time.Time{}, "", "", "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(spotPair, time.Time{}, time.Time{}, order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(spotPair, time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(spotPair, time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), order.Buy.Lower(), "stop", "1")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.Order(spotPair, order.Buy.Lower(), "limit", false, false, false, "", 0.0001, 500)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set correctly")
	}

	currencyPair, err := currency.NewPairFromString(spotPair)
	if err != nil {
		t.Fatal(err)
	}

	var orderSubmission = &order.Submit{
		Pair:          currencyPair,
		Side:          order.Sell,
		Type:          order.Limit,
		Price:         100000,
		Amount:        1,
		AssetType:     asset.Spot,
		ClientOrderID: "order12345679$$$$$",
	}
	_, err = f.SubmitOrder(orderSubmission)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.TriggerOrder(spotPair, order.Buy.Lower(), order.Stop.Lower(), "", "", 500, 0.0004, 0.0001, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set correctly")
	}

	currencyPair, err := currency.NewPairFromString(spotPair)
	if err != nil {
		t.Fatal(err)
	}

	c := order.Cancel{
		ID:        "12366984218",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}
	if err := f.CancelOrder(&c); err != nil {
		t.Error(err)
	}

	c.ClientOrderID = "1337"
	if err := f.CancelOrder(&c); err != nil {
		t.Error(err)
	}
}

func TestDeleteOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteOrder("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteOrderByClientID("clientID123")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
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
	// optional params
	_, err := f.GetFills("", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(spotPair, "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(spotPair, "", time.Unix(authStartTime, 0), time.Unix(authEndTime, 0))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(spotPair, "", time.Unix(authEndTime, 0), time.Unix(authStartTime, 0))
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestGetFundingPayments(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	// optional params
	_, err := f.GetFundingPayments(time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), futuresPair)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), futuresPair)
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
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
	_, err := f.RequestLTCreation(testLeverageToken, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTRedemptions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTRedemptions()
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
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.CreateQuoteRequest(currency.BTC, "call", order.Buy.Lower(), 1593140400, "", 10, 10, 5, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
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
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
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
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteMyQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
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
	// test optional params
	result, err := f.GetPublicOptionsTrades(time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	if len(result) != 20 {
		t.Error("default limit should have returned 20 items")
	}
	tmNow := time.Now()
	result, err = f.GetPublicOptionsTrades(tmNow.AddDate(0, 0, -1), tmNow, "5")
	if err != nil {
		t.Error(err)
	}
	if len(result) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	_, err = f.GetPublicOptionsTrades(time.Unix(validFTTBTCEndTime, 0), time.Unix(validFTTBTCStartTime, 0), "5")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
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
	_, err = f.GetOptionsFills(time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOptionsFills(time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), "5")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
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

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := f.UpdateTickers(asset.Spot)
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
	orderReq.AssetType = asset.Spot
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
	orderReq.AssetType = asset.Spot
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
	_, err := f.UpdateAccountInfo(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	// t.Parallel()
	// if !areTestAPIKeysSet() {
	// 	t.Skip("API keys required but not set, skipping test")
	// }
	f.Verbose = true
	_, err := f.FetchAccountInfo(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	fees, err := f.GetFee(10, 1, true)
	if err != nil {
		t.Error(err)
	}
	if fees <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	fees, err = f.GetFee(10, 1, false)
	if err != nil {
		t.Error(err)
	}
	if fees <= 0 {
		t.Errorf("incorrect maker fee value")
	}
}

func TestGetOfflineTradingFee(t *testing.T) {
	t.Parallel()
	fees, err := f.Fees.GetMakerTotalOffline(10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if fees != 0.002 {
		t.Errorf("incorrect offline maker fee")
	}
	fees, err = f.Fees.GetTakerTotalOffline(10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if fees != 0.007 {
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
	_, err := f.RequestLTRedemption("ETHBULL", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	var request = new(withdraw.Request)
	request.Amount = 5
	request.Currency = currency.NewCode("FTT")
	var cryptoData withdraw.CryptoRequest
	cryptoData.Address = "testaddress123"
	cryptoData.AddressTag = "testtag123"
	request.Crypto = cryptoData
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
	currencyPair, err := currency.NewPairFromString("BTC/USD")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2019, 11, 12, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 2)
	_, err = f.GetHistoricCandles(currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC/USD")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2019, 11, 12, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 2)
	_, err = f.GetHistoricCandlesExtended(currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOTCQuoteStatus(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOTCQuoteStatus(spotPair, "1")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestForQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.RequestForQuotes(currency.NewCode("BtC"), currency.NewCode("UsD"), 0.5)
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	err := f.AcceptOTCQuote("1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	assets := f.GetAssetTypes(false)
	for i := range assets {
		enabledPairs, err := f.GetEnabledPairs(assets[i])
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.GetHistoricTrades(enabledPairs.GetRandomPair(),
			assets[i],
			time.Now().Add(-time.Minute*15),
			time.Now())
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	assets := f.GetAssetTypes(false)
	for i := range assets {
		enabledPairs, err := f.GetEnabledPairs(assets[i])
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.GetRecentTrades(enabledPairs.GetRandomPair(), assets[i])
		if err != nil {
			t.Error(err)
		}
	}
}

func TestTimestampFromFloat64(t *testing.T) {
	t.Parallel()
	constTime := 1592697600.0
	checkTime := time.Date(2020, time.June, 21, 0, 0, 0, 0, time.UTC)
	timeConst := timestampFromFloat64(constTime)
	if timeConst != checkTime {
		t.Error("invalid time conversion")
	}
}

func TestCompatibleOrderVars(t *testing.T) {
	t.Parallel()
	orderVars, err := f.compatibleOrderVars(
		"buy",
		"closed",
		"limit",
		0.5,
		0.5,
		9500)
	if err != nil {
		t.Error(err)
	}
	if orderVars.Side != order.Buy {
		t.Errorf("received %v expected %v", orderVars.Side, order.Buy)
	}
	if orderVars.OrderType != order.Limit {
		t.Errorf("received %v expected %v", orderVars.OrderType, order.Limit)
	}
	if orderVars.Status != order.Filled {
		t.Errorf("received %v expected %v", orderVars.Status, order.Filled)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"closed",
		"limit",
		0,
		0,
		9500)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if orderVars.Status != order.Cancelled {
		t.Errorf("received %v expected %v", orderVars.Status, order.Cancelled)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"closed",
		"limit",
		0.5,
		0.2,
		9500)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if orderVars.Status != order.PartiallyCancelled {
		t.Errorf("received %v expected %v", orderVars.Status, order.PartiallyCancelled)
	}

	orderVars, err = f.compatibleOrderVars(
		"sell",
		"closed",
		"limit",
		1337,
		1337,
		9500)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if orderVars.Status != order.Filled {
		t.Errorf("received %v expected %v", orderVars.Status, order.Filled)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"closed",
		"limit",
		0.1,
		0.2,
		9500)
	if !errors.Is(err, errInvalidOrderAmounts) {
		t.Errorf("received %v expected %v", err, errInvalidOrderAmounts)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"fake",
		"limit",
		0.3,
		0.2,
		9500)
	if !errors.Is(err, errUnrecognisedOrderStatus) {
		t.Errorf("received %v expected %v", err, errUnrecognisedOrderStatus)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"new",
		"limit",
		0.3,
		0.2,
		9500)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if orderVars.Status != order.New {
		t.Errorf("received %v expected %v", orderVars.Status, order.New)
	}

	orderVars, err = f.compatibleOrderVars(
		"buy",
		"open",
		"limit",
		0.3,
		0.2,
		9500)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if orderVars.Status != order.Open {
		t.Errorf("received %v expected %v", orderVars.Status, order.Open)
	}
}

func TestGetIndexWeights(t *testing.T) {
	t.Parallel()
	_, err := f.GetIndexWeights("SHIT")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyPlacedOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyPlacedOrder("1234", "", -0.1, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyOrderByClientID("1234", "", -0.1, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyTriggerOrder("1234", "stop", -0.1, 0.1, 0.02, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubaccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetSubaccounts()
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubaccount(t *testing.T) {
	t.Parallel()
	_, err := f.CreateSubaccount("")
	if !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %v, but received: %s", errSubaccountNameMustBeSpecified, err)
	}

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err = f.CreateSubaccount("subzero")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.DeleteSubaccount("subzero"); err != nil {
		t.Error(err)
	}
}

func TestUpdateSubaccountName(t *testing.T) {
	t.Parallel()
	_, err := f.UpdateSubaccountName("", "")
	if !errors.Is(err, errSubaccountUpdateNameInvalid) {
		t.Errorf("expected %v, but received: %s", errSubaccountUpdateNameInvalid, err)
	}

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err = f.CreateSubaccount("subzero")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.UpdateSubaccountName("subzero", "bizzlebot")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSubaccount("bizzlebot"); err != nil {
		t.Error(err)
	}
}

func TestDeleteSubaccountName(t *testing.T) {
	t.Parallel()
	if err := f.DeleteSubaccount(""); !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %v, but received: %s", errSubaccountNameMustBeSpecified, err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.CreateSubaccount("subzero")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSubaccount("subzero"); err != nil {
		t.Error(err)
	}
}

func TestSubaccountBalances(t *testing.T) {
	t.Parallel()
	_, err := f.SubaccountBalances("")
	if !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %s, but received: %s", errSubaccountNameMustBeSpecified, err)
	}
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err = f.SubaccountBalances("non-existent")
	if err == nil {
		t.Error("expecting non-existent subaccount to return an error")
	}
	_, err = f.CreateSubaccount("subzero")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.SubaccountBalances("subzero")
	if err != nil {
		t.Error(err)
	}
	if err := f.DeleteSubaccount("subzero"); err != nil {
		t.Error(err)
	}
}

func TestSubaccountTransfer(t *testing.T) {
	tt := []struct {
		Coin        currency.Code
		Source      string
		Destination string
		Size        float64
		ErrExpected error
	}{
		{ErrExpected: errCoinMustBeSpecified},
		{Coin: currency.BTC, ErrExpected: errSubaccountTransferSizeGreaterThanZero},
		{Coin: currency.BTC, Size: 420, ErrExpected: errSubaccountTransferSourceDestinationMustNotBeEqual},
	}
	for x := range tt {
		_, err := f.SubaccountTransfer(tt[x].Coin, tt[x].Source, tt[x].Destination, tt[x].Size)
		if !errors.Is(err, tt[x].ErrExpected) {
			t.Errorf("expected %s, but received: %s", tt[x].ErrExpected, err)
		}
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.SubaccountTransfer(currency.BTC, "", "test", 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakes(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakes()
	if err != nil {
		t.Error(err)
	}
}

func TestGetUnstakeRequests(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetUnstakeRequests()
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakeBalances(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakeBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestUnstakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	r, err := f.UnstakeRequest(currency.FTT, 0.1)
	if err != nil {
		t.Fatal(err)
	}

	success, err := f.CancelUnstakeRequest(r.ID)
	if err != nil || !success {
		t.Errorf("unable to cancel unstaking request: %s", err)
	}
}

func TestCancelUnstakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.CancelUnstakeRequest(74351)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakingRewards(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakingRewards()
	if err != nil {
		t.Error(err)
	}
}

func TestStakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}

	// WARNING: This will lock up your funds for 14 days
	_, err := f.StakeRequest(currency.FTT, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	err := f.UpdateOrderExecutionLimits("")
	if err != nil {
		t.Fatal(err)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	limit, err := f.GetOrderExecutionLimits(asset.Spot, cp)
	if err != nil {
		t.Fatal(err)
	}

	err = limit.Conforms(33000, 0.00001, order.Limit)
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Fatalf("expected error %v but received %v",
			order.ErrAmountBelowMin,
			err)
	}

	err = limit.Conforms(33000, 0.0001, order.Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v",
			nil,
			err)
	}
}

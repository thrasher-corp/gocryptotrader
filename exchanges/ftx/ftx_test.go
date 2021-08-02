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
	if areTestAPIKeysSet() {
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
	_, err := f.Order(spotPair, order.Buy.Lower(), "limit", "", "", "", "", 0.0001, 500)
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
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.FetchAccountInfo(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := &exchange.FeeBuilder{
		PurchasePrice: 10,
		Amount:        1,
		IsMaker:       true,
	}
	fee, err := f.GetFee(feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.IsMaker = false
	if fee, err = f.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.FeeType = exchange.OfflineTradeFee
	fee, err = f.GetFee(feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.IsMaker = true
	fee, err = f.GetFee(feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}
}

func TestGetOfflineTradingFee(t *testing.T) {
	t.Parallel()
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

func TestParsingOrders(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		  "channel": "fills",
		  "data": {
			"id": 24852229,
			"clientId": null,
			"market": "XRP-PERP",
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

func TestParsingMarketsData(t *testing.T) {
	t.Parallel()
	data := []byte(`{"channel": "markets",
	 	"type": "partial",
		"data": {
			"ADA-0626": {
			"name": "ADA-0626",
			"enabled": true,
			"priceIncrement": 5e-06,
			"sizeIncrement": 1.0,
			"type": "future",
			"baseCurrency": null,
			"quoteCurrency": null,
			"restricted": false,
			"underlying": "ADA",
			"future": {
				"name": "ADA-0626",
				"underlying": "ADA",
				"description": "Cardano June 2020 Futures",
				"type": "future", "expiry": "2020-06-26T003:00:00+00:00", 
				"perpetual": false, 
				"expired": false, 
				"enabled": true, 
				"postOnly": false, 
				"imfFactor": 4e-05, 
				"underlyingDescription": "Cardano", 
				"expiryDescription": "June 2020", 
				"moveStart": null, "positionLimitWeight": 10.0, 
				"group": "quarterly"}}},
		"action": "partial"
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
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "update", "data": {"time": 1589855831.5128105, "checksum": 365946911, "bids": [[9596.0, 4.2656], [9512.0, 32.7912]], "asks": [[9613.5, 4.012], [9702.0, 0.021]], "action": "update"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
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

func TestParsingWSOBData2(t *testing.T) {
	t.Parallel()
	data := []byte(`{"channel": "orderbook", "market": "PRIVBEAR/USD", "type": "partial", "data": {"time": 1593498757.0915809, "checksum": 87356415, "bids": [[1389.5, 5.1019], [1384.5, 16.6318], [1371.5, 23.5531], [1365.5, 23.3001], [1354.0, 26.758], [1352.5, 24.6891], [1337.5, 30.3091], [1333.5, 24.9583], [1323.0, 30.9597], [1302.0, 40.9241], [1282.5, 38.0319], [1272.5, 39.1436], [1084.5, 1.8934], [1080.0, 2.0595], [1075.0, 2.0527], [1069.0, 1.8077], [1053.5, 1.855], [1.0, 2.0]], "asks": [[1403.5, 6.8077], [1407.5, 17.6482], [1417.0, 14.6401], [1418.5, 22.6664], [1426.0, 20.3936], [1430.5, 34.2797], [1435.0, 30.6073], [1443.0, 20.2036], [1471.5, 35.5789], [1494.5, 29.2815], [1505.0, 30.9842], [1511.5, 39.4325], [1799.5, 1.7529], [1810.5, 2.0379], [1813.5, 2.0423], [1817.5, 2.0393], [1821.0, 1.7148], [86347.5, 9e-05], [94982.5, 0.0001], [104480.0, 0.0001], [114930.0, 0.00011], [126420.0, 0.00011], [139065.0, 0.00011], [152970.0, 0.00012], [168267.5, 0.00012], [185092.5, 0.00012], [223962.5, 0.00013], [246360.0, 0.00014], [270995.0, 0.00017], [1203602.5, 0.00013]], "action": "partial"}}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "DOGE-PERP", "type": "partial", "data": {"time": 1593395710.072698, "checksum": 2591057682, "bids": [[0.0023085, 507742.0], [0.002308, 7000.0], [0.0023075, 100000.0], [0.0023065, 324770.0], [0.002305, 46000.0], [0.0023035, 879600.0], [0.002303, 49000.0], [0.0023025, 1076421.0], [0.002296, 30511800.0], [0.002293, 3006300.0], [0.0022925, 1256349.0], [0.0022895, 11855700.0], [0.0022855, 1008960.0], [0.0022775, 1047578.0], [0.0022745, 3070200.0], [0.00227, 2939100.0], [0.002269, 1599711.0], [0.00226, 1671504.0], [0.00225, 1957119.0], [0.00224, 5225404.0], [0.0022395, 250.0], [0.002233, 2994000.0], [0.002229, 2336857.0], [0.002218, 2144227.0], [0.002205, 2101662.0], [0.0021985, 7406099.0], [0.0021915, 2470187.0], [0.0021775, 2690545.0], [0.0021755, 250.0], [0.002162, 2997201.0], [0.00215, 11464856.0], [0.002148, 16178857.0], [0.0021255, 11063510.0], [0.002119, 164239.0], [0.0020435, 19124572.0], [0.0020395, 18376430.0], [0.0020125, 1250.0], [0.0019655, 50.0], [0.001958, 97012.0], [0.001942, 50000.0], [0.001899, 50000.0], [0.001895, 1250.0], [0.001712, 2500.0], [0.0012075, 70190.0], [0.00112, 22321.0], [1.65e-05, 31889.0]], "asks": [[0.0023145, 359557.0], [0.0023155, 222497.0], [0.0023175, 40000.0], [0.002319, 879600.0], [0.0023195, 50000.0], [0.0023205, 1067334.0], [0.0023215, 45000.0], [0.002326, 33518100.0], [0.0023265, 1113997.0], [0.0023285, 1170756.0], [0.002331, 11855700.0], [0.002336, 1105442.0], [0.002344, 1244804.0], [0.002348, 3070200.0], [0.0023525, 1546561.0], [0.0023555, 2939100.0], [0.0023575, 2928000.0], [0.002362, 1509707.0], [0.0023725, 1786697.0], [0.002374, 5710.0], [0.0023795, 151098.0], [0.0023835, 1747428.0], [0.002385, 2994000.0], [0.002395, 1721532.0], [0.0024015, 5710.0], [0.002408, 2552142.0], [0.002422, 2188855.0], [0.002429, 5710.0], [0.0024295, 8441953.0], [0.002437, 2196750.0], [0.002445, 122574.0], [0.002454, 1974273.0], [0.0024565, 5710.0], [0.0024715, 2864643.0], [0.00248, 15238408.0], [0.002484, 5710.0], [0.002497, 16343646.0], [0.0025025, 12177084.0], [0.0025115, 5710.0], [0.002539, 5710.0], [0.002566, 16643688.0], [0.0025665, 5710.0], [0.002594, 5710.0], [0.002617, 50.0], [0.002623, 10.0], [0.0027685, 20825893.0], [0.003178, 50000.0], [0.003811, 68952.0], [0.0074, 41460.0]], "action": "partial"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "partial", "data": {"time": 1589855831.4606245, "checksum": 225973019, "bids": [[9602.0, 3.2903], [9601.5, 3.11], [9601.0, 2.1356], [9600.5, 3.0991], [9600.0, 8.014], [9599.5, 4.1571], [9599.0, 79.1846], [9598.5, 3.099], [9598.0, 3.985], [9597.5, 3.999], [9597.0, 16.4335], [9596.5, 4.006], [9596.0, 3.2596], [9595.0, 6.334], [9594.0, 3.5685], [9593.0, 14.2717], [9592.5, 0.5], [9591.0, 2.181], [9590.5, 40.4246], [9590.0, 1.0], [9589.0, 1.357], [9588.5, 0.4738], [9587.5, 0.15], [9587.0, 16.811], [9586.5, 1.2], [9586.0, 0.2], [9585.5, 1.0], [9584.5, 0.002], [9584.0, 1.51], [9583.5, 0.01], [9583.0, 1.4], [9582.5, 0.1], [9582.0, 24.7921], [9581.0, 2.087], [9580.5, 2.0], [9580.0, 0.1], [9579.0, 1.1588], [9578.0, 0.9477], [9577.5, 22.216], [9576.0, 0.2], [9574.0, 22.0], [9573.5, 1.0], [9572.0, 0.203], [9570.0, 0.1026], [9565.5, 5.5332], [9565.0, 27.5243], [9563.5, 2.6], [9562.0, 0.0175], [9561.0, 2.0085], [9552.0, 1.6], [9550.5, 27.3399], [9550.0, 0.1046], [9548.0, 0.0175], [9544.0, 4.8197], [9542.5, 26.5754], [9542.0, 0.003], [9541.0, 0.0549], [9540.0, 0.1984], [9537.5, 0.0008], [9535.5, 0.0105], [9535.0, 1.514], [9534.5, 36.5858], [9532.5, 4.7798], [9531.0, 40.6564], [9525.0, 0.001], [9523.5, 1.6], [9522.0, 0.0894], [9521.0, 0.315], [9520.5, 5.4525], [9520.0, 0.07], [9518.0, 0.034], [9517.5, 4.0], [9513.0, 0.0175], [9512.5, 15.6016], [9512.0, 32.7882], [9511.5, 0.0482], [9510.5, 0.0482], [9510.0, 0.2999], [9509.0, 2.0], [9508.5, 0.0482], [9506.0, 0.0416], [9505.5, 0.0492], [9505.0, 0.2], [9502.5, 0.01], [9502.0, 0.01], [9501.5, 0.0592], [9501.0, 0.001], [9500.0, 3.4913], [9499.5, 39.8683], [9498.0, 4.6108], [9497.0, 0.0481], [9492.0, 41.3559], [9490.0, 1.1104], [9488.0, 0.0105], [9486.0, 5.4443], [9485.5, 0.0482], [9484.0, 4.0], [9482.0, 0.25], [9481.5, 2.0], [9481.0, 8.1572]], "asks": [[9602.5, 3.0], [9603.0, 2.8979], [9603.5, 54.49], [9604.0, 5.9982], [9604.5, 3.028], [9605.0, 4.657], [9606.5, 5.2512], [9607.0, 4.003], [9607.5, 4.011], [9608.0, 13.7505], [9608.5, 3.994], [9609.0, 2.974], [9609.5, 3.002], [9612.0, 10.298], [9612.5, 13.455], [9613.5, 3.013], [9614.0, 2.02], [9614.5, 3.359], [9615.0, 21.2429], [9616.0, 0.5], [9616.5, 0.01], [9617.0, 2.182], [9617.5, 23.0223], [9618.0, 0.0623], [9618.5, 1.5795], [9619.0, 0.3065], [9620.0, 3.9], [9621.0, 1.5], [9622.0, 1.5], [9622.5, 1.216], [9625.0, 1.0], [9625.5, 0.9477], [9626.0, 0.05], [9628.5, 1.1588], [9629.0, 1.4], [9630.0, 4.2332], [9630.5, 1.228], [9631.0, 1.5], [9631.5, 0.0104], [9632.5, 26.7529], [9633.0, 0.25], [9638.0, 1.0], [9640.0, 0.2], [9641.0, 1.001], [9642.0, 0.0175], [9643.0, 0.25], [9643.5, 1.6], [9644.0, 31.4166], [9646.5, 41.6609], [9649.5, 0.2], [9653.5, 1.5], [9656.5, 1.6], [9657.0, 0.2], [9658.0, 1.5], [9659.5, 4.7804], [9660.5, 43.3405], [9665.5, 40.6564], [9670.0, 0.1034], [9671.5, 4.9098], [9674.0, 0.25], [9678.0, 15.6016], [9678.5, 1.5], [9681.0, 34.9683], [9683.0, 0.2], [9683.5, 5.3845], [9684.5, 5.087], [9685.0, 0.1032], [9686.5, 0.0075], [9689.0, 1.6], [9691.0, 34.7472], [9692.0, 0.001], [9694.0, 0.5], [9695.0, 0.0109], [9696.5, 4.825], [9700.0, 1.0595], [9701.5, 2.0], [9702.0, 0.011], [9702.5, 0.01], [9706.0, 1.2], [9708.0, 0.0175], [9710.0, 39.153], [9712.0, 48.6163], [9712.5, 1.5], [9713.0, 8.1572], [9715.5, 0.5021], [9716.5, 2.0], [9719.0, 0.0245], [9721.0, 0.5], [9724.0, 0.251], [9726.0, 0.12], [9727.5, 0.5075], [9730.0, 0.015], [9732.0, 58.5394], [9733.0, 0.001], [9734.0, 20.0], [9743.0, 0.06], [9750.0, 9.5], [9755.0, 52.4404], [9757.0, 48.6121], [9764.0, 0.015]], "action": "partial"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "update", "data": {"time": 1589855831.5128105, "checksum": 365946911, "bids": [[9596.0, 4.2656], [9512.0, 32.7912]], "asks": [[9613.5, 4.012], [9702.0, 0.021]], "action": "update"}}`)
	err = f.wsHandleData(data)
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

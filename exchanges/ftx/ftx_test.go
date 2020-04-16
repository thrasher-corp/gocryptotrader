package ftx

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var f Ftx

func TestMain(m *testing.M) {
	f.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Ftx")
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
	a, err := f.GetMarkets()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarket(t *testing.T) {
	a, err := f.GetMarket("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	a, err := f.GetOrderbook("FTT/BTC", 5)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	a, err := f.GetTrades("FTT/BTC", "10234032", "5234343433", 5)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalData(t *testing.T) {
	a, err := f.GetHistoricalData("FTT/BTC", "86400", "5", "", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutures(t *testing.T) {
	a, err := f.GetFutures()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuture(t *testing.T) {
	a, err := f.GetFuture("LEO-0327")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureStats(t *testing.T) {
	a, err := f.GetFutureStats("LEO-0327")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	a, err := f.GetFundingRates()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetAccountInfo()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetPositions()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAccountLeverage(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	f.Verbose = true
	err := f.ChangeAccountLeverage(50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoins(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetCoins()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalances(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	a, err := f.GetBalances()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllWalletBalances(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	a, err := f.GetAllWalletBalances()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositAddress(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	a, err := f.FetchDepositAddress("TUSD")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositHistory(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	a, err := f.FetchDepositHistory()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchWithdrawalHistory(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	a, err := f.FetchWithdrawalHistory()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.Withdraw("BTC", "38eyTMFHvo5UjPR91zwYYKuCtdF2uhtdxS", "", "", "957378", 0.01)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetOpenOrders("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderHistory(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.FetchOrderHistory("FTT/BTC", "", "", "2")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenTriggerOrders(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetOpenTriggerOrders("FTT/BTC", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderTriggers(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetTriggerOrderTriggers("alkdjfkajdsf")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetTriggerOrderHistory("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestOrder(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.Order("FTT/BTC", "buy", "limit", "", "", "", "", 0.0001, 500)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerOrder(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.TriggerOrder("FTT/BTC", "buy", stopOrderType, "", "", 500, 0.0004, 0.0001, 0)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrder(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.DeleteOrder("testing123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrderByClientID(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.DeleteOrderByClientID("clientID123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTriggerOrder(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	a, err := f.DeleteTriggerOrder("triggerOrder123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetFills("FTT/BTC", "", "", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingPayments(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetFundingPayments("", "", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestListLeveragedTokens(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.ListLeveragedTokens()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTokenInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetTokenInfo("ADAMOON")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTBalances(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.ListLTBalances()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTCreations(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.ListLTCreations()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTCreation(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.RequestLTCreation("ADAMOON", 1)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTRedemptions(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.ListLTRedemptions("ADAMOON", 5)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuoteRequests(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetQuoteRequests()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetYourQuoteRequests(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetYourQuoteRequests()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateQuoteRequest(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.CreateQuoteRequest("BTC", "call", "buy", int64(time.Now().UnixNano()/1000000), 0.1, 1, 0, 0, 0, false)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteQuote(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.DeleteQuote("testing123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotesForYourQuote(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetQuotesForYourQuote()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestMakeQuote(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.MakeQuote("testing123", "5")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestMyQuotes(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.MyQuotes()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMyQuote(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.DeleteMyQuote("testing123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptQuote(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.AcceptQuote("testing123")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountOptionsInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetAccountOptionsInfo()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOptionsPositions(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetOptionsPositions()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicOptionsTrades(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	a, err := f.GetPublicOptionsTrades("", "", "5")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

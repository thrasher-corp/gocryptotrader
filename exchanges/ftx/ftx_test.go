package ftx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	subaccount              = ""
	canManipulateRealOrders = false
	spotPair                = "FTT/BTC"
	futuresPair             = "DOGE-PERP"
	testLeverageToken       = "ADAMOON"

	validFTTBTCStartTime   = 1565445600           // Sat Aug 10 2019 14:00:00 GMT+0000
	validFTTBTCEndTime     = 1565532000           // Sat Aug 11 2019 14:00:00 GMT+0000
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

func TestStart(t *testing.T) {
	t.Parallel()
	err := f.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = f.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := f.GetMarkets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalIndex(t *testing.T) {
	t.Parallel()
	_, err := f.GetHistoricalIndex(context.Background(),
		"BTC", 3600, time.Now().Add(-time.Hour*2), time.Now().Add(-time.Hour*1))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetHistoricalIndex(context.Background(),
		"BTC", 3600, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarket(t *testing.T) {
	t.Parallel()
	_, err := f.GetMarket(context.Background(), spotPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := f.GetOrderbook(context.Background(), spotPair, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	// test empty market
	_, err := f.GetTrades(context.Background(), "", 0, 0, 200)
	if err == nil {
		t.Error("empty market should return an error")
	}
	_, err = f.GetTrades(context.Background(),
		spotPair, validFTTBTCEndTime, validFTTBTCStartTime, 5)
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	// test optional params
	var trades []TradeData
	trades, err = f.GetTrades(context.Background(), spotPair, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 20 {
		t.Error("default limit should return 20 items")
	}
	trades, err = f.GetTrades(context.Background(),
		spotPair, validFTTBTCStartTime, validFTTBTCEndTime, 5)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	trades, err = f.GetTrades(context.Background(),
		spotPair, invalidFTTBTCStartTime, invalidFTTBTCEndTime, 5)
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
	_, err := f.GetHistoricalData(context.Background(),
		"", 86400, 5, time.Time{}, time.Time{})
	if err == nil {
		t.Error("empty market should return an error")
	}
	// test empty resolution
	_, err = f.GetHistoricalData(context.Background(),
		spotPair, 0, 5, time.Time{}, time.Time{})
	if err == nil {
		t.Error("empty resolution should return an error")
	}
	_, err = f.GetHistoricalData(context.Background(),
		spotPair, 86400, 5, time.Unix(validFTTBTCEndTime, 0),
		time.Unix(validFTTBTCStartTime, 0))
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	var o []OHLCVData
	o, err = f.GetHistoricalData(context.Background(),
		spotPair, 86400, 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(o) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	o, err = f.GetHistoricalData(context.Background(),
		spotPair, 86400, 5, time.Unix(invalidFTTBTCStartTime, 0),
		time.Unix(invalidFTTBTCEndTime, 0))
	if err != nil {
		t.Error(err)
	}
	if len(o) != 0 {
		t.Error("invalid time range should return 0 items")
	}
}

func TestGetFutures(t *testing.T) {
	t.Parallel()
	_, err := f.GetFutures(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuture(t *testing.T) {
	t.Parallel()
	_, err := f.GetFuture(context.Background(), futuresPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := f.GetFutureStats(context.Background(), "BTC-PERP")
	if err != nil {
		t.Error(err)
	}

	future, err := f.GetFutureStats(context.Background(), "BTC-MOVE-2021Q4")
	if err != nil {
		t.Error(err)
	}

	if future.Greeks == nil {
		t.Fatal("no greeks returned for futures contract")
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	// optional params
	_, err := f.GetFundingRates(context.Background(), time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingRates(context.Background(),
		time.Now().Add(-time.Hour), time.Now(), "BTC-PERP")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAccountInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	f.Verbose = true
	_, err := f.GetPositions(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllWalletBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAllWalletBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAccountLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	err := f.ChangeAccountLeverage(context.Background(), 50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetCoins(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginBorrowRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginBorrowRates(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginLendingRates(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestMarginDailyBorrowedAmounts(t *testing.T) {
	t.Parallel()
	_, err := f.MarginDailyBorrowedAmounts(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginMarketInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginMarketInfo(context.Background(), "BTC_USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginBorrowHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginBorrowHistory(context.Background(),
		tmNow.AddDate(0, 0, 1),
		tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err = f.GetMarginBorrowHistory(context.Background(),
		tmNow.AddDate(0, 0, -1),
		tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginMarketLendingHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginMarketLendingHistory(context.Background(),
		currency.USD, tmNow.AddDate(0, 0, 1), tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err = f.GetMarginMarketLendingHistory(context.Background(),
		currency.USD, tmNow.AddDate(0, 0, -1), tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingHistory(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	_, err := f.GetMarginLendingHistory(context.Background(),
		currency.USD, tmNow.AddDate(0, 0, 1), tmNow)
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("expected %s, got %s", errStartTimeCannotBeAfterEndTime, err)
	}

	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err = f.GetMarginLendingHistory(context.Background(),
		currency.USD, tmNow.AddDate(0, 0, -1), tmNow)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLendingOffers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetMarginLendingOffers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetLendingInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetLendingInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitLendingOffer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}
	if err := f.SubmitLendingOffer(context.Background(),
		currency.NewCode("bTc"), 0.1, 500); err != nil {
		t.Error(err)
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	r, err := f.FetchDepositAddress(context.Background(), currency.NewCode("UsDt"), "trx")
	if err != nil {
		t.Fatal(err)
	}
	if r.Method != "trx" {
		t.Error("expected trx method")
	}
}

func TestFetchDepositHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchDepositHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestFetchWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchWithdrawalHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.Withdraw(context.Background(),
		currency.NewCode("UsDT"),
		"TJU9piX2WA8WTvxVKMqpvTzZGhvXQAZKSY",
		"",
		"",
		"trx",
		"715913",
		-1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOpenOrders(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOpenOrders(context.Background(), spotPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.FetchOrderHistory(context.Background(),
		"", time.Time{}, time.Time{}, "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(context.Background(),
		spotPair, time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(context.Background(),
		spotPair, time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), "2")
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
	_, err := f.GetOpenTriggerOrders(context.Background(), "", "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOpenTriggerOrders(context.Background(), spotPair, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderTriggers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTriggerOrderTriggers(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTriggerOrderHistory(context.Background(),
		"", time.Time{}, time.Time{}, "", "", "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(context.Background(),
		spotPair, time.Time{}, time.Time{}, order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(context.Background(),
		spotPair,
		time.Unix(authStartTime, 0),
		time.Unix(authEndTime, 0),
		order.Buy.Lower(),
		"stop",
		"1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(context.Background(),
		spotPair,
		time.Unix(authEndTime, 0),
		time.Unix(authStartTime, 0),
		order.Buy.Lower(),
		"stop",
		"1")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.Order(context.Background(),
		spotPair,
		order.Buy.Lower(),
		"limit",
		false, false, false,
		"", 0.0001, 500)
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
	_, err = f.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.TriggerOrder(context.Background(),
		spotPair,
		order.Buy.Lower(),
		order.Stop.Lower(),
		"", "",
		500, 0.0004, 0.0001, 0)
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
	if err := f.CancelOrder(context.Background(), &c); err != nil {
		t.Error(err)
	}

	c.ClientOrderID = "1337"
	if err := f.CancelOrder(context.Background(), &c); err != nil {
		t.Error(err)
	}
}

func TestDeleteOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteOrder(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteOrderByClientID(context.Background(), "clientID123")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteTriggerOrder(context.Background(), "1031")
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
	_, err := f.GetFills(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(context.Background(), spotPair, "", time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(context.Background(),
		spotPair, "", time.Unix(authStartTime, 0), time.Unix(authEndTime, 0))
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFills(context.Background(),
		spotPair, "", time.Unix(authEndTime, 0), time.Unix(authStartTime, 0))
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
	_, err := f.GetFundingPayments(context.Background(),
		time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(context.Background(),
		time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), futuresPair)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetFundingPayments(context.Background(),
		time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), futuresPair)
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestListLeveragedTokens(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLeveragedTokens(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTokenInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetTokenInfo(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestListLTBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestListLTCreations(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTCreations(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTCreation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.RequestLTCreation(context.Background(), testLeverageToken, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestListLTRedemptions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.ListLTRedemptions(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuoteRequests(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetQuoteRequests(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetYourQuoteRequests(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetYourQuoteRequests(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateQuoteRequest(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.CreateQuoteRequest(context.Background(),
		currency.BTC, "call", order.Buy.Lower(), 1593140400, "", 10, 10, 5, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteQuote(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotesForYourQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetQuotesForYourQuote(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestMakeQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.MakeQuote(context.Background(), "1031", "5")
	if err != nil {
		t.Error(err)
	}
}

func TestMyQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.MyQuotes(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMyQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.DeleteMyQuote(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.AcceptQuote(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountOptionsInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetAccountOptionsInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOptionsPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOptionsPositions(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPublicOptionsTrades(t *testing.T) {
	t.Parallel()
	// test optional params
	result, err := f.GetPublicOptionsTrades(context.Background(),
		time.Time{}, time.Time{}, "")
	if err != nil {
		t.Error(err)
	}
	if len(result) != 20 {
		t.Error("default limit should have returned 20 items")
	}
	tmNow := time.Now()
	result, err = f.GetPublicOptionsTrades(context.Background(),
		tmNow.AddDate(0, 0, -7), tmNow, "5")
	if err != nil {
		t.Error(err)
	}
	if len(result) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	_, err = f.GetPublicOptionsTrades(context.Background(),
		time.Unix(validFTTBTCEndTime, 0), time.Unix(validFTTBTCStartTime, 0), "5")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestGetOptionsFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetOptionsFills(context.Background(), time.Time{}, time.Time{}, "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOptionsFills(context.Background(),
		time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), "5")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetOptionsFills(context.Background(),
		time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), "5")
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	_, err := f.UpdateOrderbook(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
	_, err := f.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := f.UpdateTickers(context.Background(), asset.Spot)
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
	_, err := f.GetActiveOrders(context.Background(), &orderReq)
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
	_, err := f.GetOrderHistory(context.Background(), &orderReq)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAccountHoldings(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.FetchAccountInfo(context.Background(), asset.Spot)
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
	fee, err := f.GetFee(context.Background(), feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.IsMaker = false
	if fee, err = f.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.FeeType = exchange.OfflineTradeFee
	fee, err = f.GetFee(context.Background(), feeBuilder)
	if err != nil {
		t.Error(err)
	}
	if fee <= 0 {
		t.Errorf("incorrect maker fee value")
	}

	feeBuilder.IsMaker = true
	fee, err = f.GetFee(context.Background(), feeBuilder)
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
	_, err := f.GetOrderStatus(context.Background(), "1031")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderStatusByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOrderStatusByClientID(context.Background(), "testID")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestLTRedemption(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.RequestLTRedemption(context.Background(), "ETHBULL", 5)
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
	_, err := f.WithdrawCryptocurrencyFunds(context.Background(), request)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetDepositAddress(context.Background(), currency.NewCode("FTT"), "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetFundingHistory(context.Background())
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
	_, err = f.GetHistoricCandles(context.Background(),
		currencyPair, asset.Spot, start, end, kline.OneDay)
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
	_, err = f.GetHistoricCandlesExtended(context.Background(),
		currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOTCQuoteStatus(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := f.GetOTCQuoteStatus(context.Background(), spotPair, "1")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestForQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.RequestForQuotes(context.Background(),
		currency.NewCode("BtC"), currency.NewCode("UsD"), 0.5)
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	err := f.AcceptOTCQuote(context.Background(), "1031")
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
		_, err = f.GetHistoricTrades(context.Background(),
			enabledPairs.GetRandomPair(),
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
		_, err = f.GetRecentTrades(context.Background(),
			enabledPairs.GetRandomPair(), assets[i])
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
	orderVars, err := f.compatibleOrderVars(context.Background(),
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

	orderVars, err = f.compatibleOrderVars(context.Background(),
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

	orderVars, err = f.compatibleOrderVars(context.Background(),
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

	orderVars, err = f.compatibleOrderVars(context.Background(),
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

	_, err = f.compatibleOrderVars(context.Background(),
		"buy",
		"closed",
		"limit",
		0.1,
		0.2,
		9500)
	if !errors.Is(err, errInvalidOrderAmounts) {
		t.Errorf("received %v expected %v", err, errInvalidOrderAmounts)
	}

	_, err = f.compatibleOrderVars(context.Background(),
		"buy",
		"fake",
		"limit",
		0.3,
		0.2,
		9500)
	if !errors.Is(err, errUnrecognisedOrderStatus) {
		t.Errorf("received %v expected %v", err, errUnrecognisedOrderStatus)
	}

	orderVars, err = f.compatibleOrderVars(context.Background(),
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

	orderVars, err = f.compatibleOrderVars(context.Background(),
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
	_, err := f.GetIndexWeights(context.Background(), "SHIT")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyPlacedOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyPlacedOrder(context.Background(), "1234", "", -0.1, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyOrderByClientID(context.Background(), "1234", "", -0.1, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyTriggerOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := f.ModifyTriggerOrder(context.Background(),
		"1234", "stop", -0.1, 0.1, 0.02, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubaccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetSubaccounts(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubaccount(t *testing.T) {
	t.Parallel()
	_, err := f.CreateSubaccount(context.Background(), "")
	if !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %v, but received: %s", errSubaccountNameMustBeSpecified, err)
	}

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err = f.CreateSubaccount(context.Background(), "subzero")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.DeleteSubaccount(context.Background(), "subzero"); err != nil {
		t.Error(err)
	}
}

func TestUpdateSubaccountName(t *testing.T) {
	t.Parallel()
	_, err := f.UpdateSubaccountName(context.Background(), "", "")
	if !errors.Is(err, errSubaccountUpdateNameInvalid) {
		t.Errorf("expected %v, but received: %s", errSubaccountUpdateNameInvalid, err)
	}

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err = f.CreateSubaccount(context.Background(), "subzero")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.UpdateSubaccountName(context.Background(), "subzero", "bizzlebot")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSubaccount(context.Background(), "bizzlebot"); err != nil {
		t.Error(err)
	}
}

func TestDeleteSubaccountName(t *testing.T) {
	t.Parallel()
	if err := f.DeleteSubaccount(context.Background(), ""); !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %v, but received: %s", errSubaccountNameMustBeSpecified, err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.CreateSubaccount(context.Background(), "subzero")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSubaccount(context.Background(), "subzero"); err != nil {
		t.Error(err)
	}
}

func TestSubaccountBalances(t *testing.T) {
	t.Parallel()
	_, err := f.SubaccountBalances(context.Background(), "")
	if !errors.Is(err, errSubaccountNameMustBeSpecified) {
		t.Errorf("expected %s, but received: %s", errSubaccountNameMustBeSpecified, err)
	}
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err = f.SubaccountBalances(context.Background(), "non-existent")
	if err == nil {
		t.Error("expecting non-existent subaccount to return an error")
	}
	_, err = f.CreateSubaccount(context.Background(), "subzero")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.SubaccountBalances(context.Background(), "subzero")
	if err != nil {
		t.Error(err)
	}
	if err := f.DeleteSubaccount(context.Background(), "subzero"); err != nil {
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
		_, err := f.SubaccountTransfer(context.Background(),
			tt[x].Coin, tt[x].Source, tt[x].Destination, tt[x].Size)
		if !errors.Is(err, tt[x].ErrExpected) {
			t.Errorf("expected %s, but received: %s", tt[x].ErrExpected, err)
		}
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.SubaccountTransfer(context.Background(),
		currency.BTC, "", "test", 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakes(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakes(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUnstakeRequests(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetUnstakeRequests(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakeBalances(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakeBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestUnstakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	r, err := f.UnstakeRequest(context.Background(), currency.FTT, 0.1)
	if err != nil {
		t.Fatal(err)
	}

	success, err := f.CancelUnstakeRequest(context.Background(), r.ID)
	if err != nil || !success {
		t.Errorf("unable to cancel unstaking request: %s", err)
	}
}

func TestCancelUnstakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.CancelUnstakeRequest(context.Background(), 74351)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakingRewards(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakingRewards(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestStakeRequest(t *testing.T) {
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}

	// WARNING: This will lock up your funds for 14 days
	_, err := f.StakeRequest(context.Background(), currency.FTT, 0.1)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	err := f.UpdateOrderExecutionLimits(context.Background(), "")
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

func TestCalculatePNLFromOrders(t *testing.T) {
	pressXToJSON := `{"success":true,"result":[
	{"id":102360167561,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":49318.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T04:31:09.246156+00:00","future":"BTC-1231"},
	{"id":102359914432,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0002,"status":"closed","filledSize":0.0002,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49311.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T04:30:02.546539+00:00","future":"BTC-1231"},
	{"id":102359875376,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49289.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T04:29:48.461182+00:00","future":"BTC-1231"},
	{"id":102357575195,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0009,"status":"closed","filledSize":0.0009,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":49205.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T04:17:35.686416+00:00","future":"BTC-1231"},
	{"id":102356768854,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0009,"status":"closed","filledSize":0.0009,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49248.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T04:13:28.891973+00:00","future":"BTC-1231"},
	{"id":102334270248,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":49068.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T02:20:41.551614+00:00","future":"BTC-1231"},
	{"id":102334231437,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0004,"status":"closed","filledSize":0.0004,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49050.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T02:20:27.615861+00:00","future":"BTC-1231"},
	{"id":102333965786,"clientId":null,"market":"BTC-1231","type":"limit","side":"buy","price":49072.0,"size":0.0003,"status":"closed","filledSize":0.0003,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49030.0,"postOnly":false,"ioc":false,"createdAt":"2021-12-06T02:19:04.483189+00:00","future":"BTC-1231"},
	{"id":102333762284,"clientId":null,"market":"BTC-1231","type":"limit","side":"buy","price":49072.0,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49061.0,"postOnly":false,"ioc":false,"createdAt":"2021-12-06T02:18:07.703362+00:00","future":"BTC-1231"},
	{"id":102333732631,"clientId":null,"market":"BTC-1231","type":"limit","side":"buy","price":49072.0,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":49063.0,"postOnly":false,"ioc":false,"createdAt":"2021-12-06T02:18:02.044371+00:00","future":"BTC-1231"},
	{"id":102321616290,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":48637.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T01:28:35.419340+00:00","future":"BTC-1231"},
	{"id":102321498868,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":48624.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T01:28:07.428170+00:00","future":"BTC-1231"},
	{"id":102321095447,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":48633.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T01:26:34.077562+00:00","future":"BTC-1231"},
	{"id":102320767608,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":48720.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-06T01:25:32.920109+00:00","future":"BTC-1231"},
	{"id":101062637645,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0002,"status":"closed","filledSize":0.0002,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":57487.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-02T01:45:32.636504+00:00","future":"BTC-1231"},
	{"id":101062454431,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":57561.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-02T01:44:44.588077+00:00","future":"BTC-1231"},
	{"id":101062444084,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":57561.0,"postOnly":false,"ioc":true,"createdAt":"2021-12-02T01:44:40.982596+00:00","future":"BTC-1231"},
	{"id":92221164777,"clientId":null,"market":"BTC-1231","type":"market","side":"buy","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":true,"liquidation":false,"avgFillPrice":61679.0,"postOnly":false,"ioc":true,"createdAt":"2021-11-01T02:25:43.913874+00:00","future":"BTC-1231"},
	{"id":92186217479,"clientId":null,"market":"BTC-1231","type":"market","side":"sell","price":null,"size":0.0001,"status":"closed","filledSize":0.0001,"remainingSize":0.0,"reduceOnly":false,"liquidation":false,"avgFillPrice":62902.0,"postOnly":false,"ioc":true,"createdAt":"2021-10-31T23:38:26.914224+00:00","future":"BTC-1231"}
],"hasMoreData":false}`
	resp := struct {
		Data []OrderData `json:"result"`
	}{}
	err := json.Unmarshal([]byte(pressXToJSON), &resp)
	if err != nil {
		t.Fatal()
	}
	result := resp.Data
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	pair := currency.NewPair(currency.BTC, currency.NewCode("1231"))
	var orders []order.Detail
	for i := range result {
		price := result[i].AvgFillPrice
		side, err := order.StringToOrderSide(result[i].Side)
		if err != nil {
			t.Error(err)
		}
		orders = append(orders, order.Detail{
			Side:      side,
			Pair:      pair,
			ID:        fmt.Sprintf("%v", result[i].ID),
			Price:     price,
			Amount:    result[i].Size,
			Status:    order.Status(result[i].Status),
			AssetType: asset.Futures,
			Exchange:  f.Name,
			Date:      result[i].CreatedAt,
		})
	}

	exch := f.Name
	item := asset.Futures
	setup := &order.PositionControllerSetup{
		Exchange:      exch,
		Asset:         item,
		Pair:          pair,
		Underlying:    pair.Base,
		PNLCalculator: &f,
	}
	p, err := order.SetupPositionController(setup)
	if err != nil {
		t.Error(err)
	}
	for i := range orders {
		err = p.TrackNewOrder(&orders[i])
		if err != nil {
			t.Error(err)
		}
	}
	pos := p.GetPositions()
	if len(pos) != 6 {
		t.Fatal("expected 6 positions")
	}
	if pnl, err := pos[0].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(0.1223)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected 0.1223, received '%v'", err, pnl)
	}
	if pnl, err := pos[1].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(0.0148)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected 0.0148, received '%v'", err, pnl)
	}
	if pnl, err := pos[2].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(0.0092)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected 0.0092, received '%v'", err, pnl)
	}
	if pnl, err := pos[3].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(-0.0054)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected -0.0054, received '%v'", err, pnl)
	}
	if pnl, err := pos[4].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(0.0387)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected 0.0387, received '%v'", err, pnl)
	}
	if pnl, err := pos[5].GetRealisedPNL(); !pnl.Equal(decimal.NewFromFloat(-0.0029)) || err != nil {
		t.Errorf("expected nil err, received '%v', expected -0.0029, received '%v'", err, pnl)
	}
}

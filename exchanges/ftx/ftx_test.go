package ftx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
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
	spotPairStr             = "FTT/BTC"
	futuresPair             = "DOGE-PERP"
	testLeverageToken       = "ADAMOON"

	validFTTBTCStartTime   = 1565445600           // Sat Aug 10 2019 14:00:00 GMT+0000
	validFTTBTCEndTime     = 1565532000           // Sat Aug 11 2019 14:00:00 GMT+0000
	invalidFTTBTCStartTime = 1559881511           // Fri Jun 07 2019 04:25:11 GMT+0000
	invalidFTTBTCEndTime   = 1559901511           // Fri Jun 07 2019 09:58:31 GMT+0000
	authStartTime          = validFTTBTCStartTime // Adjust these to test auth requests
	authEndTime            = validFTTBTCEndTime
)

var (
	f        FTX
	spotPair = currency.NewPair(currency.FTT, currency.BTC)
)

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
	return f.ValidateAPICredentials(f.GetDefaultCredentials()) == nil
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
	_, err := f.GetMarket(context.Background(), spotPairStr)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := f.GetOrderbook(context.Background(), spotPairStr, 5)
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
		spotPairStr, validFTTBTCEndTime, validFTTBTCStartTime, 5)
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	// test optional params
	var trades []TradeData
	trades, err = f.GetTrades(context.Background(), spotPairStr, 0, 0, 0)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 20 {
		t.Error("default limit should return 20 items")
	}
	trades, err = f.GetTrades(context.Background(),
		spotPairStr, validFTTBTCStartTime, validFTTBTCEndTime, 5)
	if err != nil {
		t.Error(err)
	}
	if len(trades) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	trades, err = f.GetTrades(context.Background(),
		spotPairStr, invalidFTTBTCStartTime, invalidFTTBTCEndTime, 5)
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
		spotPairStr, 0, 5, time.Time{}, time.Time{})
	if err == nil {
		t.Error("empty resolution should return an error")
	}
	_, err = f.GetHistoricalData(context.Background(),
		spotPairStr, 86400, 5, time.Unix(validFTTBTCEndTime, 0),
		time.Unix(validFTTBTCStartTime, 0))
	if err != errStartTimeCannotBeAfterEndTime {
		t.Errorf("should have thrown errStartTimeCannotBeAfterEndTime, got %v", err)
	}
	var o []OHLCVData
	o, err = f.GetHistoricalData(context.Background(),
		spotPairStr, 86400, 5, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	if len(o) != 5 {
		t.Error("limit of 5 should return 5 items")
	}
	o, err = f.GetHistoricalData(context.Background(),
		spotPairStr, 86400, 5, time.Unix(invalidFTTBTCStartTime, 0),
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
	_, err := f.GetBalances(context.Background(), false, false)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetBalances(context.Background(), true, false)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetBalances(context.Background(), false, true)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetBalances(context.Background(), true, true)
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
	_, err = f.GetOpenOrders(context.Background(), spotPairStr)
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
		spotPairStr, time.Unix(authStartTime, 0), time.Unix(authEndTime, 0), "2")
	if err != nil {
		t.Error(err)
	}
	_, err = f.FetchOrderHistory(context.Background(),
		spotPairStr, time.Unix(authEndTime, 0), time.Unix(authStartTime, 0), "2")
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
	_, err = f.GetOpenTriggerOrders(context.Background(), spotPairStr, "")
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
		spotPairStr, time.Time{}, time.Time{}, order.Buy.Lower(), "stop", "1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(context.Background(),
		spotPairStr,
		time.Unix(authStartTime, 0),
		time.Unix(authEndTime, 0),
		order.Buy.Lower(),
		"stop",
		"1")
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetTriggerOrderHistory(context.Background(),
		spotPairStr,
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
		spotPairStr,
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

	currencyPair, err := currency.NewPairFromString(spotPairStr)
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
		spotPairStr,
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

	currencyPair, err := currency.NewPairFromString(spotPairStr)
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
	_, err := f.GetFills(context.Background(),
		currency.Pair{}, asset.Futures, time.Now().Add(time.Hour*24*365), time.Now())
	if !errors.Is(err, errStartTimeCannotBeAfterEndTime) {
		t.Errorf("received '%v' expected '%v'", err, errStartTimeCannotBeAfterEndTime)
	}

	_, err = f.GetFills(context.Background(),
		currency.Pair{}, asset.Futures, time.Time{}, time.Time{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = f.GetFills(context.Background(),
		currency.Pair{}, asset.Futures, time.Now().Add(-time.Hour*24*365), time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = f.GetFills(context.Background(),
		spotPair, asset.Spot, time.Now().Add(-time.Hour*24*365), time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
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
		tmNow.AddDate(0, -1, 0), tmNow, "5")
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

	cp = currency.NewPairWithDelimiter("ALGOBEAR", currency.USD.String(), "/")
	_, err = f.UpdateOrderbook(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	cp = currency.NewPairWithDelimiter("ASDBEAR", currency.USD.String(), "/")
	_, err = f.UpdateOrderbook(context.Background(), cp, asset.Spot)
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
	_, err := f.GetOTCQuoteStatus(context.Background(), spotPairStr, "1")
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
	t.Parallel()
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
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakes(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUnstakeRequests(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetUnstakeRequests(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakeBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakeBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestUnstakeRequest(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, err := f.CancelUnstakeRequest(context.Background(), 74351)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStakingRewards(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	_, err := f.GetStakingRewards(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestStakeRequest(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestScaleCollateral(t *testing.T) {
	t.Parallel()

	result, err := f.ScaleCollateral(
		context.Background(),
		&order.CollateralCalculator{
			CollateralCurrency: currency.USDT,
			Asset:              asset.Spot,
			Side:               order.Buy,
			CalculateOffline:   true,
			FreeCollateral:     decimal.NewFromInt(100000),
			USDPrice:           decimal.NewFromFloat(1.0003),
		})
	if err != nil {
		t.Error(err)
	}
	expectedUSDValue := decimal.NewFromFloat(97529.25)
	if !result.CollateralContribution.Equal(expectedUSDValue) {
		t.Errorf("received %v expected %v", result.CollateralContribution, expectedUSDValue)
	}

	if !areTestAPIKeysSet() {
		return
	}
	accountInfo, err := f.GetAccountInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
	walletInfo, err := f.GetAllWalletBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
	localScaling := 0.0
	providedUSDValue := 0.0
	for _, v := range walletInfo {
		for v2 := range v {
			coin := v[v2].Coin
			if coin.Equal(currency.USD) {
				localScaling += v[v2].Total
				providedUSDValue += v[v2].USDValue
				continue
			}
			var tick MarketData
			tick, err = f.GetMarket(context.Background(), currency.NewPairWithDelimiter(coin.String(), "usd", "/").String())
			if err != nil {
				// not all markets exist like this, skip
				continue
			}
			_, err = f.ScaleCollateral(
				context.Background(),
				&order.CollateralCalculator{
					CollateralCurrency: coin,
					Asset:              asset.Spot,
					Side:               order.Buy,
					FreeCollateral:     decimal.NewFromFloat(v[v2].Total),
					USDPrice:           decimal.NewFromFloat(tick.Price),
					CalculateOffline:   true,
				})
			if err != nil {
				if errors.Is(err, errCollateralCurrencyNotFound) ||
					errors.Is(err, order.ErrUSDValueRequired) {
					continue
				}
				t.Error(err)
			}
			providedUSDValue += v[v2].USDValue
			_, err = f.ScaleCollateral(context.Background(),
				&order.CollateralCalculator{
					CollateralCurrency: coin,
					Asset:              asset.Spot,
					Side:               order.Buy,
					FreeCollateral:     decimal.NewFromFloat(v[v2].Total),
					USDPrice:           decimal.NewFromFloat(tick.Price),
					IsForNewPosition:   true,
					CalculateOffline:   true,
				})
			if err != nil {
				t.Error(err)
			}
			_, err = f.ScaleCollateral(context.Background(),
				&order.CollateralCalculator{
					CollateralCurrency: coin,
					Asset:              asset.Spot,
					Side:               order.Buy,
					FreeCollateral:     decimal.NewFromFloat(v[v2].Total),
					USDPrice:           decimal.Zero,
					IsLiquidating:      true,
					CalculateOffline:   true,
				})
			if !errors.Is(err, order.ErrUSDValueRequired) {
				t.Errorf("received '%v' exepected '%v'", err, order.ErrUSDValueRequired)
			}

			_, err = f.ScaleCollateral(
				context.Background(),
				&order.CollateralCalculator{
					CollateralCurrency: coin,
					Asset:              asset.Spot,
					Side:               order.Buy,
				})
			if err != nil {
				t.Error(err)
			}
		}
	}
	if accountInfo.Collateral == 0 {
		return
	}
	if (math.Abs((localScaling-accountInfo.Collateral)/accountInfo.Collateral) * 100) > 5 {
		t.Errorf("collateral scaling less than 95%% accurate, received '%v' expected roughly '%v'", localScaling, accountInfo.Collateral)
	}
}

func TestCalculateTotalCollateral(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	walletInfo, err := f.GetAllWalletBalances(context.Background())
	if err != nil {
		t.Error(err)
	}
	var scales []order.CollateralCalculator
	for _, v := range walletInfo {
		for v2 := range v {
			coin := v[v2].Coin
			if coin.Equal(currency.USD) {
				total := decimal.NewFromFloat(v[v2].Total)
				scales = append(scales, order.CollateralCalculator{
					CollateralCurrency: coin,
					Asset:              asset.Spot,
					Side:               order.Buy,
					FreeCollateral:     total,
					USDPrice:           total,
					CalculateOffline:   true,
				})
				continue
			}
			var tick MarketData
			tick, err = f.GetMarket(context.Background(), currency.NewPairWithDelimiter(coin.String(), "usd", "/").String())
			if err != nil {
				// some assumed markets don't exist, just don't process them
				t.Log(err)
				continue
			}
			if tick.Price == 0 {
				continue
			}
			scales = append(scales, order.CollateralCalculator{
				CollateralCurrency: coin,
				Asset:              asset.Spot,
				Side:               order.Buy,
				FreeCollateral:     decimal.NewFromFloat(v[v2].Total),
				USDPrice:           decimal.NewFromFloat(tick.Price),
				CalculateOffline:   true,
			})
		}
	}
	calc := &order.TotalCollateralCalculator{
		CollateralAssets: scales,
		FetchPositions:   false,
		CalculateOffline: true,
	}
	total, err := f.CalculateTotalCollateral(context.Background(), calc)
	if err != nil {
		t.Fatal(err)
	}
	localScaling := total.AvailableCollateral.InexactFloat64()
	accountInfo, err := f.GetAccountInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
	if accountInfo.Collateral != 0 && (math.Abs((localScaling-accountInfo.Collateral)/accountInfo.Collateral)*100) > 5 {
		t.Errorf("collateral scaling less than 95%% accurate, received '%v' expected roughly '%v'", localScaling, accountInfo.Collateral)
	}

	for i := range scales {
		scales[i].CalculateOffline = false
	}
	calc.CalculateOffline = false
	_, err = f.CalculateTotalCollateral(context.Background(), calc)
	if err != nil {
		t.Error(err)
	}
}

func TestCalculateTotalCollateralOnline(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	// nil data
	_, err := f.calculateTotalCollateralOnline(context.Background(), nil, nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilPointer)
	}
	calc := &order.TotalCollateralCalculator{}
	// no currency data
	_, err = f.calculateTotalCollateralOnline(context.Background(), calc, nil)
	if !errors.Is(err, errCollateralCurrencyNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errCollateralCurrencyNotFound)
	}
	calc.CalculateOffline = true
	calc.CollateralAssets = []order.CollateralCalculator{
		{
			CollateralCurrency: currency.BTC,
		},
		{
			CollateralCurrency: currency.USD,
		},
	}
	// offline true
	_, err = f.calculateTotalCollateralOnline(context.Background(), calc, nil)
	if !errors.Is(err, order.ErrOfflineCalculationSet) {
		t.Errorf("received '%v' expected '%v'", err, order.ErrOfflineCalculationSet)
	}

	calc.CalculateOffline = false
	calc.CollateralAssets[0].CalculateOffline = true
	// offline true for individual currency
	_, err = f.calculateTotalCollateralOnline(context.Background(), calc, nil)
	if !errors.Is(err, order.ErrOfflineCalculationSet) {
		t.Errorf("received '%v' expected '%v'", err, order.ErrOfflineCalculationSet)
	}
	// successful run
	calc.CollateralAssets[0].CalculateOffline = false
	result, err := f.calculateTotalCollateralOnline(context.Background(), calc, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !result.CollateralCurrency.Equal(currency.USD) {
		t.Error("expected USD collateral currency")
	}
	curr, err := currency.NewPairFromString("BTC-PERP")
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	// with position data
	pos := []PositionData{
		{
			CollateralUsed: 5,
			Future:         curr,
			UnrealizedPNL:  10,
		},
	}
	_, err = f.calculateTotalCollateralOnline(context.Background(), calc, pos)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	calc.CollateralAssets = []order.CollateralCalculator{
		{
			CollateralCurrency: currency.BURST,
		},
	}
	// irrelevant currency
	result, err = f.calculateTotalCollateralOnline(context.Background(), calc, pos)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !result.UnrealisedPNL.IsZero() {
		t.Error("expected zero")
	}
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	pair := currency.NewPair(currency.BTC, currency.NewCode("20211231"))
	positions, err := f.GetFuturesPositions(context.Background(), asset.Futures, pair, time.Date(2021, 1, 6, 4, 28, 0, 0, time.UTC), time.Date(2021, 12, 31, 4, 32, 0, 0, time.UTC))
	if err != nil {
		t.Error(err)
	}
	var orders []order.Detail
	for i := range positions {
		orders = append(orders, order.Detail{
			Side:      positions[i].Side,
			Pair:      pair,
			ID:        fmt.Sprintf("%v", positions[i].ID),
			Price:     positions[i].Price,
			Amount:    positions[i].Amount,
			AssetType: asset.Futures,
			Exchange:  f.Name,
			Fee:       positions[i].Fee,
			Date:      positions[i].Date,
		})
	}

	exch := f.Name
	item := asset.Futures
	setup := &order.MultiPositionTrackerSetup{
		Exchange:                  exch,
		Asset:                     item,
		Pair:                      pair,
		Underlying:                pair.Base,
		UseExchangePNLCalculation: true,
		ExchangePNLCalculation:    &f,
	}
	p, err := order.SetupMultiPositionTracker(setup)
	if err != nil {
		t.Error(err)
	}
	for i := range orders {
		err = p.TrackNewOrder(&orders[i])
		if err != nil {
			t.Error(err)
		}
	}
	results := p.GetPositions()
	if len(orders) > 0 && len(results) == 0 {
		t.Error("expected position(s) to be generated")
	}
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test, api keys not set")
	}
	cp := currency.NewPair(currency.BTC, currency.NewCode("20211231"))
	start := time.Now().Add(-time.Hour * 24 * 365)
	end := time.Now()
	a := asset.Futures
	_, err := f.GetFuturesPositions(context.Background(), a, cp, start, end)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadCollateralWeightings(t *testing.T) {
	t.Parallel()
	ff := FTX{}
	err := ff.LoadCollateralWeightings(context.Background())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(ff.collateralWeight) == 0 {
		t.Fatal("expected some weight")
	}
	if !ff.collateralWeight.hasData() {
		t.Error("expected loaded weight")
	}
	if !areTestAPIKeysSet() {
		return
	}
	err = f.LoadCollateralWeightings(context.Background())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(f.collateralWeight) == 0 {
		t.Fatal("expected some weight")
	}
}

func TestLoadTotalIMF(t *testing.T) {
	t.Parallel()
	c := CollateralWeightHolder{}
	c.loadTotal("BTC", 1)
	if _, ok := c[currency.BTC.Item]; !ok {
		t.Error("expected entry")
	}
	c.loadInitialMarginFraction("btc", 1)
	cw, ok := c[currency.BTC.Item]
	if !ok {
		t.Error("expected entry")
	}
	if cw.Total != 1 {
		t.Errorf("expected '1', received '%v'", cw.Total)
	}
	if cw.InitialMarginFractionFactor != 1 {
		t.Errorf("expected '1', received '%v'", cw.InitialMarginFractionFactor)
	}
}

func TestLoadCollateralWeight(t *testing.T) {
	t.Parallel()
	c := CollateralWeightHolder{}
	c.load("DOGE", 1, 2, 3)
	cw, ok := c[currency.DOGE.Item]
	if !ok {
		t.Fatal("expected loaded collateral weight")
	}
	if cw.Total != 1 {
		t.Errorf("expected '1', received '%v'", cw.Total)
	}
	if cw.Initial != 2 {
		t.Errorf("expected '2', received '%v'", cw.Initial)
	}
	if cw.InitialMarginFractionFactor != 3 {
		t.Errorf("expected '3', received '%v'", cw.InitialMarginFractionFactor)
	}
}

func TestCollateralWeightHasData(t *testing.T) {
	t.Parallel()
	c := CollateralWeightHolder{}
	if c.hasData() {
		t.Error("expected false")
	}
	c.load("test", 1, 2, 3)
	if !c.hasData() {
		t.Error("expected true")
	}
}

func TestGetExpiredFutures(t *testing.T) {
	t.Parallel()
	_, err := f.GetExpiredFutures(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetExpiredFuture(t *testing.T) {
	t.Parallel()
	_, err := f.GetExpiredFuture(context.Background(), currency.NewPairWithDelimiter("BTC", "20211231", "-"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateral(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := f.GetCollateral(context.Background(), false)
	if err != nil {
		t.Error(err)
	}
	_, err = f.GetCollateral(context.Background(), true)
	if err != nil {
		t.Error(err)
	}
}

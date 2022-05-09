package binanceus

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	bi              Binanceus
	testPairMapping = currency.NewPair(currency.BTC, currency.USDT)
	// this lock guards against orderbook tests race
	binanceOrderBookLock = &sync.Mutex{}
)

func TestMain(m *testing.M) {
	bi.SetDefaults()
	bi.validLimits = []int{5, 10, 20, 50, 100, 500, 1000}
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Binanceus load config error", err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Binanceus")
	if err != nil {
		log.Fatal(err)
	}
	bi.SkipAuthCheck = true

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	bi.Websocket = sharedtestvalues.NewTestWebsocket()
	err = bi.Setup(exchCfg)
	if err != nil {
		log.Fatal("Binanceus TestMain()", err)
	}
	// This method instantiates the Order Book Manager of Binanceus
	bi.setupOrderbookManager()

	os.Exit(m.Run())
}

/*  For the websocket testing  */

// This is for testing the wait group
func TestStart(t *testing.T) {
	t.Parallel()
	err := bi.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("%s received: '%v' but expected: '%v'", bi.Name, err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = bi.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

/* End */

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Binanceus); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return bi.ValidateAPICredentials(bi.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.GetExchangeInfo(context.Background())
	if err != nil {
		println("DERR: ", err.Error())
		t.Error(err)
	}
}

/************************************************************************/

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	r, err := bi.UpdateTicker(context.Background(), testPairMapping, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if r.Pair.Base != currency.BTC && r.Pair.Quote != currency.USDT {
		t.Error("invalid pair values")
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := bi.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderBook(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Error("Binanceus UpdateOrderBook() error", err)
	}
	_, er := bi.UpdateOrderbook(context.Background(), currencyPair, asset.Spot)
	if er != nil {
		t.Error("Binanceus UpdateOrderBook() error", er)
	}
}

// TestFetchTradablePairs .. testint the tradable pairs for spot asset types.
func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error("Binanceus FetchTradablePairs() error", err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := bi.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		t.Error("Binanceus UpdateTradablePairs() error", err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()
	if _, err := bi.FetchAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Binanceus FetchAccountInfo() error", err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()
	_, err := bi.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error("Binanceus UpdateAccountInfo() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair := currency.Pair{Base: currency.BTC, Quote: currency.USD}
	_, err := bi.GetRecentTrades(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error("Binanceus GetRecentTrades() error", err)
	}
}

// TestGetHistoricTrades
func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	pair := currency.Pair{Base: currency.BTC, Quote: currency.USD}
	_, err := bi.GetHistoricTrades(context.Background(), pair, asset.Spot, time.Time{}, time.Time{})
	if err != nil {
		t.Error("Binanceus GetHistoricTrades() error", err)
	}
}

func TestGetFeeByType(t *testing.T) {
	// I have not implemented the method GetFeeByType yet
	t.SkipNow()
}

func TestGetFundingHistory(t *testing.T) {
	// This Method is not implemented yet
	t.SkipNow()
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.XRP,
			Quote: currency.USD,
		},
		AssetType: asset.Spot,
		Side:      order.Sell,
		Type:      order.Limit,
		Price:     1000,
		Amount:    20,
		ClientID:  "binanceSamOrder",
	}
	response, err := bi.SubmitOrder(context.Background(), orderSubmission)

	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not place order: %v", err)
	}
	if areTestAPIKeysSet() && !response.IsOrderPlaced {
		t.Error("Order not placed")
	}
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	pair := currency.NewPair(currency.XPR, currency.USD)
	var cancellationOrder = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          pair,
		AssetType:     asset.Spot,
	}
	err := bi.CancelOrder(context.Background(), cancellationOrder)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Binanceus CancelExchangeOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus CancelExchangeOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Binanceus Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	_, err := bi.CancelAllOrders(context.Background(), orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelAllExchangeOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("CancelAllExchangeOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() error", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("Binanceus GetOrderInfo() skipping test: api keys not set")
	}
	tradablePairs, err := bi.FetchTradablePairs(context.Background(),
		asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("Binanceus GetOrderInfo() no tradable pairs")
	}
	cp, err := currency.NewPairFromString(tradablePairs[0])
	if err != nil {
		t.Error("Binanceus GetOrderInfo() error", err)
	}
	_, err = bi.GetOrderInfo(context.Background(),
		"123", cp, asset.Spot)
	if !strings.Contains(err.Error(), "Order does not exist.") {
		t.Error("Binanceus GetOrderInfo() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddress(context.Background(), currency.USDT, "", currency.BNB.String())
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Binanceus GetDepositAddress() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus GetDepositAddress() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Binanceus Mock GetDepositAddress() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	// This method is not implemented yet.
	t.SkipNow()
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("Binanceus API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := bi.GetWithdrawalsHistory(context.Background(), currency.ETH)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Binanceus GetWithdrawalsHistory() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus GetWithdrawalsHistory() expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	// t.Parallel()
	// This method is not yet implemented.
	t.SkipNow()
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
		// Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
	}
	orders, err := bi.GetActiveOrders(context.Background(), &getOrdersRequest)
	t.Logf("Binanceus : %d Orders found", len(orders))
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetActiveOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() error", err)
	}
}

// TODO: this test is not completed yet.
func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("Binanceus API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    bi.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
			Chain:   "BSC",
		},
	}

	_, err := bi.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)

	switch {
	case areTestAPIKeysSet() && err != nil:
		if strings.Contains(err.Error(), "amount must be greater than zero") {
			return
		}
		t.Error("Binanceus Withdraw() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus Withdraw() expecting an error when no keys are set")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
	val, er := bi.GetFeeByType(context.Background(), feeBuilder)
	if er != nil {
		t.Fatal("Binanceus GetFeeByType() error", er)
	}
	println(val)

}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, er := bi.GetHistoricCandles(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*5))
	if !strings.Contains(er.Error(), "interval not supported") {
		t.Errorf("Binanceus GetHistoricCandles() expected %s, but found %v", "interval not supported", er)
	}
	// startTime = time.Unix(time.Now().Unix()-int64(time.Hour*30), 0)
	// endTime = time.Now()
	_, er = bi.GetHistoricCandles(context.Background(), pair, asset.Spot, time.Time{}, time.Time{}, kline.Interval(time.Hour*4))
	if er != nil {
		t.Error("Binanceus GetHistoricCandles() error", er)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, er := bi.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*5))
	if !strings.Contains(er.Error(), "interval not supported") {
		t.Errorf("Binanceus GetHistoricCandlesExtended() expected %s, but found %v", "interval not supported", er)
	}
	startTime = time.Unix(time.Now().Unix()-int64(time.Hour*30), 0)
	endTime = time.Now()
	_, er = bi.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*4))
	if er != nil {
		t.Error("Binanceus GetHistoricCandlesExtended() error", er)
	}
}

/************************************************************************/

// TestGetMostRecentTrades -- test most recent trades end-point
func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMostRecentTrades(context.Background(), RecentTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  15,
	})
	if err != nil {
		t.Error("Binanceus GetMostRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalTrades(context.Background(), HistoricalTradeParams{
		Symbol: "BTCUSDT",
		Limit:  5,
		FromID: 0,
	})
	if err != nil {
		t.Errorf("Binanceus GetHistoricalTrades() error: %v", err)
	}
}

func TestGetAggregateTrades(t *testing.T) {
	t.Parallel()
	// _, err := bi.GetAggregateTrades(context.Background(),
	// 	&AggregatedTradeRequestParams{
	// 		Symbol: currency.NewPair(currency.BTC, currency.USDT),
	// 		Limit:  1001,
	// 	})
	// if err != nil {
	// 	t.Error("Binanceus GetAggregateTrades() error", err)
	// }
	_, err := bi.GetAggregateTrades(context.Background(),
		&AggregatedTradeRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  5,
		})
	if err != nil {
		t.Error("Binanceus GetAggregateTrades() error", err)
	}
	// _, err = bi.GetAggregateTrades(context.Background(),
	// 	&AggregatedTradeRequestParams{
	// 		Symbol:  currency.NewPair(currency.BTC, currency.USDT),
	// 		Limit:   5,
	// 		EndTime: uint64(time.Now().UnixMilli()),
	// 	})
	// if err != nil {
	// 	t.Error("Binanceus GetAggregateTrades() error", err)
	// }
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOrderBookDepth(context.TODO(), &OrderBookDataRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  1000,
	})
	if er != nil {
		t.Error("Binanceus GetOrderBook() error", er)
	}
}

func TestGetCandlestickData(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSpotKline(context.Background(), &KlinesRequestParams{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: time.Unix(1577836800, 0),
		EndTime:   time.Unix(1580515200, 0),
	})
	if er != nil {
		t.Error("Binanceus GetSpotKline() error", er)
	}
}

func TestGetPriceDatas(t *testing.T) {
	t.Parallel()
	_, er := bi.GetPriceDatas(context.TODO())
	if er != nil {
		t.Error("Binanceus GetPriceDatas() error", er)
	}
}

func TestGetSinglePriceData(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSinglePriceData(context.Background(), currency.Pair{
		Base:  currency.BTC,
		Quote: currency.USDT,
	})
	if er != nil {
		t.Error("Binanceus GetSinglePriceData() error", er)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := bi.GetAveragePrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetAveragePrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := bi.GetBestPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binanceus GetBestPrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPriceChangeStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	_, err := bi.GetTickers(context.Background())
	if err != nil {
		t.Error("Binance TestGetTickers error", err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	_, er := bi.GetAccount(context.Background())
	if er != nil {
		t.Error("Binanceus GetAccount() error", er)
	}
}

func TestGetUserAccountStatus(t *testing.T) {
	t.Parallel()
	res, er := bi.GetUserAccountStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAccountStatus() error", er)
	}
	val, _ := json.Marshal(res)
	println("\n", string(val))
}

func TestGetUserAPITradingStatus(t *testing.T) {
	t.Parallel()
	_, er := bi.GetUserAPITradingStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAPITradingStatus() error", er)
	}
}
func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	_, er := bi.GetTradeFee(context.Background(), 3000, "BTCUSTD")
	if er != nil {
		t.Error("Binanceus GetTradeFee() error", er)
	}
}

func TestGetAssetDistributionHistory(t *testing.T) {
	t.Parallel()
	_, er := bi.GetAssetDistributionHistory(context.Background(), "", 0, 0, 3000)
	if er != nil {
		t.Error("Binanceus GetAssetDistributionHistory() error", er)
	}
}

func TestGetSubaccountInformation(t *testing.T) {
	t.Parallel()
	t.Skip("meanwhile, ther is no sub account information available ")
	_, er := bi.GetSubaccountInformation(context.Background(), 1, 100, "", "")
	if er != nil {
		t.Error("Binanceus GetSubaccountInformation() error", er)
	}
}

func TestGetSubaccountTransferHistory(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSubaccountTransferhistory(context.Background(), "", 0, 0, 0, 0)
	if !errors.Is(er, errNotValidEmailAddress) {
		t.Errorf("Binanceus GetSubaccountTransferhistory() expected %v, but received: %s", errNotValidEmailAddress, er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus GetSubaccountTransferhistory() skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, er = bi.GetSubaccountTransferhistory(context.Background(), "example@golang.org", 0, 0, 0, 0)
	if !errors.Is(er, errNotValidEmailAddress) {
		t.Fatalf("Binanceus GetSubaccountTransferhistory() error %v", er)
	}
}

func TestExecuteSubAccountTransfer(t *testing.T) {
	t.Parallel()
	_, er := bi.ExecuteSubAccountTransfer(context.Background(), &SubaccountTransferRequestParams{
		// FromEmail: "fromemail@thrasher.io",
		// ToEmail:   "toemail@threasher.io",
		// Asset:     "BTC",
		// Amount:    0.000005,
	})
	if !errors.Is(er, errUnacceptableSenderEmail) {
		t.Errorf("binanceus error: expected %v, but found %v", errUnacceptableSenderEmail, er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus GetSubaccountTransferhistory() skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, er = bi.ExecuteSubAccountTransfer(context.Background(), &SubaccountTransferRequestParams{
		FromEmail: "fromemail@thrasher.io",
		ToEmail:   "toemail@threasher.io",
		Asset:     "BTC",
		Amount:    0.000005,
	})
	if er != nil && !strings.Contains(er.Error(), "You don't have permission.") {
		t.Fatalf("Binanceus GetSubaccountTransferhistory() error %v", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus GetSubaccountTransferhistory() skipping test, either api keys or canManipulateRealOrders isn't set")
	}

}

func TestGetSubaccountAssets(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSubaccountAssets(context.Background(), "")
	if !errors.Is(er, errNotValidEmailAddress) {
		t.Errorf("Binanceus GetSubaccountAssets() expected %v, but found %v", er, errNotValidEmailAddress)
	}
	_, er = bi.GetSubaccountAssets(context.Background(), "subaccount@thrasher.io")
	if er != nil && !strings.Contains(er.Error(), "Illegal request.") {
		t.Fatal("Binanceus GetSubaccountAssets() error", er)
	}
}

func TestGetOrderRateLimits(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOrderRateLimits(context.Background(), 0)
	if er != nil {
		t.Error("Binanceus GetOrderRateLimits() error", er)
	}
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()

	req := &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   BinanceRequestParamsOrderLimit,
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}
	_, err := bi.NewOrderTest(context.Background(), req)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Binanceus NewOrderTest() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus NewOrderTest() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Binanceus Mock NewOrderTest() error", err)
		// default:
		// 	t.Error("Binanceus NewOrderTest() error", err)
	}
	req = &NewOrderRequest{
		Symbol:        currency.NewPair(currency.LTC, currency.BTC),
		Side:          order.Sell.String(),
		TradeType:     BinanceRequestParamsOrderMarket,
		Price:         0.0045,
		QuoteOrderQty: 10,
	}

	result, err := bi.NewOrderTest(context.Background(), req)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("NewOrderTest() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("NewOrderTest() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock NewOrderTest() error", err)
	}
	re, _ := json.Marshal(result)
	println(string(re))
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, er := bi.GetOrder(context.Background(), OrderRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus GetOrder() error expecting %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.GetOrder(context.Background(), OrderRequestParams{
		Symbol:            "BTCUSDT",
		OrigClientOrderId: "something",
	})
	// You can check the existance of an order using a valid Symbol and OrigClient Order ID
	if er != nil && !strings.Contains(er.Error(), "Order does not exist.") {
		t.Error("Binanceus GetOrder() error", er)
	}
}

func TestGetAllOpenOrders(t *testing.T) {
	t.Parallel()
	orders, er := bi.GetAllOpenOrders(context.Background(), "")
	if er != nil {
		t.Error("Binanceus GetAllOpenOrders() error", er)
	}
	ordersString, _ := json.Marshal(orders)
	println(string(ordersString))
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, er := bi.CancelExistingOrder(context.Background(), CancelOrderRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CancelExistingOrder() error expecting %v, but found %v", errIncompleteArguments, er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus CancelExistingOrder() skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, er = bi.CancelExistingOrder(context.Background(), CancelOrderRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
	})
	if er != nil {
		t.Error(er)
	}
}

//
func TestCancelOpenOrdersForSymbol(t *testing.T) {
	t.Parallel()
	_, er := bi.CancelOpenOrdersForSymbol(context.Background(), "")
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CancelOpenOrdersForSymbol() error expecting %v, but found %v", errIncompleteArguments, er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus CancelOpenOrdersForSymbol() skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, er = bi.CancelOpenOrdersForSymbol(context.Background(), "BTCUSDT")
	if er != nil && !strings.Contains(er.Error(), "Unknown order sent") {
		t.Error(er)
	}
}

// TestGetTrades test for fetching the list of
// trades attached with this account.
func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, er := bi.GetTrades(context.Background(), GetTradesParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf(" Binanceus GetTrades() expecting error %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.GetTrades(context.Background(), GetTradesParams{Symbol: "BTCUSDT"})
	if er != nil {
		t.Error("Binanceus GetTrades() error", er)
	}
}

func TestCreateNewOCOOrder(t *testing.T) {
	t.Parallel()
	_, er := bi.CreateNewOCOOrder(context.Background(),
		OCOOrderInputParams{
			// Symbol:    "BTCUSDT",
			StopPrice: 1000,
			Side:      "BUY",
			Quantity:  0.0000001,
			Price:     1232334.00,
		})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CreatenewOCOOrder() error expected %v, but found %v", errIncompleteArguments, er)
	}
	// TODO: Incomplete yet
}

func TestGetOCOOrder(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOCOOrder(context.Background(), GetOCOPrderRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus GetOCOOrder() error  expecting %v, but found %v", errIncompleteArguments, er)
	}
	// TODO:
}

func TestGetAllOCOOrder(t *testing.T) {
	t.Parallel()
	_, er := bi.GetAllOCOOrder(context.Background(), OCOOrdersRequestParams{})
	if er != nil {
		t.Error("Binanceus GetAllOCOOrder() error", er)
	}
}

func TestGetOpenOCOOrders(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOpenOCOOrders(context.Background(), 0)
	if er != nil {
		t.Error("Binanceus GetOpenOCOOrders() error", er)
	}
}

func TestCancelOCOOrder(t *testing.T) {
	t.Parallel()
	//
	_, er := bi.CancelOCOOrder(context.Background(), OCOOrdersDeleteRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CancelOCOOrder() error expected %v, but found %v", errIncompleteArguments, er)
	}
}

// OTC end Points test code.
func TestGetSupportedCoinPairs(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSupportedCoinPairs(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT})
	if er != nil {
		t.Error("Binanceus GetSupportedCoinPairs() error", er)
	}
}

func TestRequestForQuote(t *testing.T) {
	t.Parallel()
	_, er := bi.RequestForQuote(context.Background(), RequestQuoteParams{FromCoin: "ETH", ToCoin: "BTC", RequestCoin: "USDT", RequestAmount: 0.000000001})
	if er != nil && !strings.Contains(er.Error(), "illegal parameter") {
		t.Error("Binanceus RequestForQuote() error", er)
	} else {
		t.Skip("Binanceus RequestForQuote() error", "illegal parameter")
	}
}

var testPlaceOTCTradeOrderJSON = `{
    "orderId": "10002349",
    "createTime": 1641906714,
    "orderStatus": "PROCESS"
}
`

func TestPlaceOTCTradeOrder(t *testing.T) {
	t.Parallel()
	var res OTCTradeOrderResponse
	er := json.Unmarshal([]byte(testPlaceOTCTradeOrderJSON), &res)
	if er != nil {
		t.Error("Binanceus PlaceOTCTradeOrder() error", er)
	}
	_, er = bi.PlaceOTCTradeOrder(context.Background(), "")
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus PlaceOTCTradeOrder()  expecting %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.PlaceOTCTradeOrder(context.Background(), "4e5446f2cc6f44ab86ab02abf19a2fd2")
	if er != nil && !strings.Contains(er.Error(), "execute quote fail") {
		t.Error("Binanceus  PlaceOTCTradeOrder() error", er)
	}
}

var testGetOTCTradeOrderJSON = `{
    "quoteId": "4e5446f2cc6f44ab86ab02abf19a2fd2",
    "orderId": "10002349", 
    "orderStatus": "SUCCESS",
    "fromCoin": "BTC",
    "fromAmount": 1,
    "toCoin": "USDT",
    "toAmount": 50550.26,
    "ratio": 50550.26,
    "inverseRatio": 0.00001978,
    "createTime": 1641806714
}
`

func TestGetOTCTradeOrder(t *testing.T) {
	t.Parallel()
	var val OTCTradeOrder
	er := json.Unmarshal([]byte(testGetOTCTradeOrderJSON), &val)
	if er != nil {
		t.Error("Binanceus JSON GetOTCTradeOrder() error", er)
	}
	_, er = bi.GetOTCTradeOrder(context.Background(), 10002349)
	if er != nil && !strings.Contains(er.Error(), "order not found") {
		t.Error("Binanceus GetOTCTradeOrder() error ", er)
	}
}

var getAllOTCTradeOrders = `[
    {
        "quoteId": "4e5446f2cc6f44ab86ab02abf19a2fd2",
        "orderId": "10002349", 
        "orderStatus": "SUCCESS",
        "fromCoin": "BTC",
        "fromAmount": 1,
        "toCoin": "USDT",
        "toAmount": 50550.26,
        "ratio": 50550.26,
        "inverseRatio": 0.00001978,
        "createTime": 1641806714
    },
    {
        "quoteId": "15848645308",
        "orderId": "10002380", 
        "orderStatus": "PROCESS",
        "fromCoin": "SHIB",
        "fromAmount": 10000,
        "toCoin": "KSHIB",
        "toAmount": 10,
        "ratio": 0.001,
        "inverseRatio": 1000,
        "createTime": 1641916714
    }
]
`

func TestGetAllOTCTradeOrders(t *testing.T) {
	t.Parallel()
	// --------------------------------------------------------------------------------------------
	var orders []*OTCTradeOrder
	er := json.Unmarshal([]byte(getAllOTCTradeOrders), &orders)
	if er != nil {
		t.Error(er)
	}
	_, er = bi.GetAllOTCTradeOrders(context.Background(), OTCTradeOrderRequestParams{})
	if er != nil {
		t.Error("Binanceus GetAllOTCTradeOrders() error", er)
	}
}

func TestGetAssetFeesAndWalletStatus(t *testing.T) {
	t.Parallel()
	_, er := bi.GetAssetFeesAndWalletStatus(context.Background())
	if er != nil {
		t.Error("Binanceus GetAssetFeesAndWalletStatus()  error", er)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, er := bi.WithdrawCrypto(context.Background(), WithdrawalRequestParam{})
	// if !errors.Is(er, errIncompleteArguments) {
	// 	t.Errorf("Binanceus error %v, but found %v", errIncompleteArguments, er)
	// }
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("Binanceus CancelExistingOrder() skipping test, either api keys or canManipulateRealOrders isn't set")
	}
	_, er = bi.WithdrawCrypto(context.Background(), WithdrawalRequestParam{})
	if er != nil {
		t.Error("Binanceus WithdrawCrypto() error", er)
	}
}

// WEBSOCKET support testing
// Since both binance and Binance US has same websocket functions,
// the tests functions are also simmilar

// TestWebsocketStreamKey .. this test mmethod handles the
// creating, updating, and deleting of user stream key or "listenKey"
// all the three methods in one test methods.

func TestWebsocketStreamKey(t *testing.T) {
	t.Parallel()

	lnKey, er := bi.GetWsAuthStreamKey(context.Background())
	if er != nil {
		t.Error("Binanceus GetWsAuthStreamKey() error", er)
	}
	log.Println(lnKey)
	er = bi.MaintainWsAuthStreamKey(context.Background())
	if er != nil {
		t.Error("Binanceus MaintainWsAuthStreamKey() error", er)
	}
	log.Println(lnKey)
	er = bi.CloseUserDataStream(context.Background())
	if er != nil {
		t.Error("Binanceus CloseUserDataStream() error", er)
	}
}

var subscriptionRequestString = `{
	"method": "SUBSCRIBE",
	"params": [
	  "btcusdt@aggTrade",
	  "btcusdt@depth"
	],
	"id": 1
  }`

func TestWebsocketSubscriptionHandling(t *testing.T) {
	t.Parallel()
	rawData := []byte(subscriptionRequestString)
	err := bi.wsHandleData(rawData)
	if err != nil {
		t.Error("Binanceus wsHandleData() error", err)
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	subscriptions := []stream.ChannelSubscription{
		{
			Channel: "btcusdt@depth",
		},
		{
			Channel: "ltcusdt@aggTrade",
		},
		{
			Channel: "btcltc@depth",
		},
	}
	bi.Subscribe(subscriptions)
}
func TestWebsocketUnsubscriptionHandling(t *testing.T) {
	pressXToJSON := []byte(`{
  "method": "UNSUBSCRIBE",
  "params": [
    "btcusdt@depth"
  ],
  "id": 312
}`)
	err := bi.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestUnsubscription(t *testing.T) {
	t.Parallel()
	unsubscriptions := []stream.ChannelSubscription{
		{
			Channel: "btcusdt@depth",
		},
		{
			Channel: "ltcusdt@aggTrade",
		},
		{
			Channel: "btcltc@depth",
		},
	}
	bi.Unsubscribe(unsubscriptions)
}

func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubscriptions()
	if err != nil {
		t.Error("Binanceus GetSubscriptions() error", err)
	}
}

var ticker24hourChangeStream = `{
	"stream":"btcusdt@ticker",
	"data" :{
		"e": "24hrTicker",  
		"E": 123456789,     
		"s": "BNBBTC",      
		"p": "0.0015",      
		"P": "250.00",      
		"w": "0.0018",      
		"x": "0.0009",      
		"c": "0.0025",      
		"Q": "10",          
		"b": "0.0024",       
		"B": "10",           
		"a": "0.0026",       
		"A": "100",          
		"o": "0.0010",      
		"h": "0.0025",      
		"l": "0.0010",      
		"v": "10000",        
		"q": "18",           
		"O": 0,             
		"C": 86400000,      
		"F": 0,             
		"L": 18150,         
		"n": 18151           
  }
}
`

func TestWebsocketTickerUpdate(t *testing.T) {
	t.Parallel()
	err := bi.wsHandleData([]byte(ticker24hourChangeStream))
	if err != nil {
		t.Error("Binanceus wsHandleData() for Ticker 24h Change Stream", err)
	}
}

func TestWebsocketKlineUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`
	{
		"stream":"btcusdt@kline_1m",
		"data":{
			"e": "kline",     
			"E": 123456789,   
			"s": "BNBBTC",    
			"k": {
				"t": 123400000, 
				"T": 123460000, 
				"s": "BNBBTC",  
				"i": "1m",      
				"f": 100,       
				"L": 200,       
				"o": "0.0010",  
				"c": "0.0020",  
				"h": "0.0025",  
				"l": "0.0015",  
				"v": "1000",    
				"n": 100,       
				"x": false,     
				"q": "1.0000",  
				"V": "500",     
				"Q": "0.500",   
				"B": "123456"   
	  			}
			}
		}`)
	err := bi.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error("Binanceus wsHandleData() btcusdt@kline_1m stream data conversion ", err)
	}
}

func TestWebsocketStreamTradeUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"stream":"btcusdt@trade","data":{
	  "e": "trade",     
	  "E": 123456789,   
	  "s": "BNBBTC",    
	  "t": 12345,       
	  "p": "0.001",     
	  "q": "100",
	  "b": 88,        
	  "a": 50,          
	  "T": 123456785,
	  "m": true,        
	  "M": true         
	}}`)
	err := bi.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error("Binanceus wsHandleData() error", err)
	}
}

// TestWsDepthUpdate copied from the Binance Test
func TestWebsocketDepthUpdate(t *testing.T) {
	binanceOrderBookLock.Lock()
	defer binanceOrderBookLock.Unlock()
	bi.setupOrderbookManager()
	seedLastUpdateID := int64(161)
	book := OrderBook{
		Asks: []OrderbookItem{
			{Price: 6621.80000000, Quantity: 0.00198100},
			{Price: 6622.14000000, Quantity: 4.00000000},
			{Price: 6622.46000000, Quantity: 2.30000000},
			{Price: 6622.47000000, Quantity: 1.18633300},
			{Price: 6622.64000000, Quantity: 4.00000000},
			{Price: 6622.73000000, Quantity: 0.02900000},
			{Price: 6622.76000000, Quantity: 0.12557700},
			{Price: 6622.81000000, Quantity: 2.08994200},
			{Price: 6622.82000000, Quantity: 0.01500000},
			{Price: 6623.17000000, Quantity: 0.16831300},
		},
		Bids: []OrderbookItem{
			{Price: 6621.55000000, Quantity: 0.16356700},
			{Price: 6621.45000000, Quantity: 0.16352600},
			{Price: 6621.41000000, Quantity: 0.86091200},
			{Price: 6621.25000000, Quantity: 0.16914100},
			{Price: 6621.23000000, Quantity: 0.09193600},
			{Price: 6621.22000000, Quantity: 0.00755100},
			{Price: 6621.13000000, Quantity: 0.08432000},
			{Price: 6621.03000000, Quantity: 0.00172000},
			{Price: 6620.94000000, Quantity: 0.30506700},
			{Price: 6620.93000000, Quantity: 0.00200000},
		},
		LastUpdateID: seedLastUpdateID,
	}

	update1 := []byte(`{"stream":"btcusdt@depth","data":{
	  "e": "depthUpdate", 
	  "E": 123456788,     
	  "s": "BTCUSDT",      
	  "U": 157,           
	  "u": 160,           
	  "b": [              
		["6621.45", "0.3"]
	  ],
	  "a": [              
		["6622.46", "1.5"]
	  ]
	}}`)

	p := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	if err := bi.SeedLocalCacheWithBook(p, &book); err != nil {
		t.Error(err)
	}

	if err := bi.wsHandleData(update1); err != nil {
		t.Error(err)
	}

	bi.obm.state[currency.BTC][currency.USDT][asset.Spot].fetchingBook = false

	ob, err := bi.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if exp, got := seedLastUpdateID, ob.LastUpdateID; got != exp {
		t.Fatalf("Unexpected Last update id of orderbook for old update. Exp: %d, got: %d", exp, got)
	}
	if exp, got := 2.3, ob.Asks[2].Amount; got != exp {
		t.Fatalf("Ask altered by outdated update. Exp: %f, got %f", exp, got)
	}
	if exp, got := 0.163526, ob.Bids[1].Amount; got != exp {
		t.Fatalf("Bid altered by outdated update. Exp: %f, got %f", exp, got)
	}
	update2 := []byte(`{"stream":"btcusdt@depth","data":{
	  "e": "depthUpdate", 
	  "E": 123456789,     
	  "s": "BTCUSDT",      
	  "U": 161,           
	  "u": 165,           
	  "b": [           
		["6621.45", "0.163526"]
	  ],
	  "a": [             
		["6622.46", "2.3"], 
		["6622.47", "1.9"]
	  ]
	}}`)

	if err = bi.wsHandleData(update2); err != nil {
		t.Error(err)
	}

	ob, err = bi.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if exp, got := int64(165), ob.LastUpdateID; got != exp {
		t.Fatalf("Binanceus Unexpected Last update id of orderbook for new update. Exp: %d, got: %d", exp, got)
	}
	if exp, got := 2.3, ob.Asks[2].Amount; got != exp {
		t.Fatalf("Binanceus Unexpected Ask amount. Exp: %f, got %f", exp, got)
	}
	if exp, got := 1.9, ob.Asks[3].Amount; got != exp {
		t.Fatalf("Binanceus Unexpected Ask amount. Exp: %f, got %f", exp, got)
	}
	if exp, got := 0.163526, ob.Bids[1].Amount; got != exp {
		t.Fatalf("Binanceus Unexpected Bid amount. Exp: %f, got %f", exp, got)
	}
	bi.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

var balanceUpdateInputJSON = `
{
	"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc",
	"data":{
		"e": "balanceUpdate",         
		"E": 1573200697110,           
		"a": "BTC",                   
		"d": "100.00000000",          
		"T": 1573200697068            
  }
}`

func TestWebsocketBalanceUpdate(t *testing.T) {
	t.Parallel()
	thejson := []byte(balanceUpdateInputJSON)
	err := bi.wsHandleData(thejson)
	if err != nil {
		t.Error(err)
	}
}

var listStatusUserDataStreamPayload = `
{
	"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc",
	"data":{
		"e": "listStatus",                
		"E": 1564035303637,               
		"s": "ETHBTC",                    
		"g": 2,                           
		"c": "OCO",                       
		"l": "EXEC_STARTED",              
		"L": "EXECUTING",                 
		"r": "NONE",                      
		"C": "F4QN4G8DlFATFlIUQ0cjdD",    
		"T": 1564035303625,               
		"O": [                            
			{
				"s": "ETHBTC",                
				"i": 17,                      
				"c": "AJYsMjErWJesZvqlJCTUgL" 
			},
			{
				"s": "ETHBTC",
				"i": 18,
				"c": "bfYPSQdLoqAJeNrOr9adzq"
			}
		]
	}
}`

func TestWebsocketListStatus(t *testing.T) {
	t.Parallel()
	err := bi.wsHandleData([]byte(listStatusUserDataStreamPayload))
	if err != nil {
		t.Error(err)
	}
}

// TestExecutionTypeToOrderStatus ..
func TestExecutionTypeToOrderStatus(t *testing.T) {
	// directly copied from binance
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "NEW", Result: order.New},
		{Case: "PARTIALLY_FILLED", Result: order.PartiallyFilled},
		{Case: "FILLED", Result: order.Filled},
		{Case: "CANCELED", Result: order.Cancelled},
		{Case: "PENDING_CANCEL", Result: order.PendingCancel},
		{Case: "REJECTED", Result: order.Rejected},
		{Case: "EXPIRED", Result: order.Expired},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Binanceus Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

var websocketDepthUpdate = []byte(
	`{
		"e": "depthUpdate",
		"E": 123456789,    
		"s": "BNBBTC",     
		"U": 157,          
		"u": 160,          
		"b": [             
		  [
			"0.0024",      
			"10"           
		  ]
		],
		"a": [             
		  [
			"0.0026",      
			"100"          
		  ]
		]
	  }
	`)

func TestProcessUpdate(t *testing.T) {
	t.Parallel()
	binanceOrderBookLock.Lock()
	defer binanceOrderBookLock.Unlock()
	p := currency.NewPair(currency.BTC, currency.USDT)
	var depth WebsocketDepthStream
	err := json.Unmarshal(websocketDepthUpdate, &depth)
	if err != nil {
		t.Fatal(err)
	}
	err = bi.obm.stageWsUpdate(&depth, p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = bi.obm.fetchBookViaREST(p)
	if err != nil {
		t.Fatal(err)
	}
	err = bi.obm.cleanup(p)
	if err != nil {
		t.Fatal(err)
	}
	bi.obm.state[currency.BTC][currency.USDT][asset.Spot].lastUpdateID = 0
}

func TestWebsocketOrderExecutionReport(t *testing.T) {
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616627567900,"s":"BTCUSDT","c":"c4wyKsIhoAaittTYlIVLqk","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028400","p":"52789.10000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":5340845958,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":"BTC","T":1616627567900,"t":-1,"I":11388173160,"w":true,"m":false,"M":false,"O":1616627567900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}}`)
	expRes := order.Detail{
		Price:                52789.1,
		Amount:               0.00028400,
		AverageExecutedPrice: 0,
		QuoteAmount:          0,
		ExecutedAmount:       0,
		RemainingAmount:      0.00028400,
		Cost:                 0,
		CostAsset:            currency.USDT,
		Fee:                  0,
		FeeAsset:             currency.BTC,
		Exchange:             "Binanceus",
		ID:                   "5340845958",
		ClientOrderID:        "c4wyKsIhoAaittTYlIVLqk",
		Type:                 order.Limit,
		Side:                 order.Buy,
		Status:               order.New,
		AssetType:            asset.Spot,
		Date:                 time.UnixMilli(1616627567900),
		LastUpdated:          time.UnixMilli(1616627567900),
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
	}
	for len(bi.Websocket.DataHandler) > 0 {
		<-bi.Websocket.DataHandler
	}
	err := bi.wsHandleData(payload)
	if err != nil {
		t.Fatal(err)
	}
	res := <-bi.Websocket.DataHandler
	switch r := res.(type) {
	case *order.Detail:
		if !reflect.DeepEqual(expRes, *r) {
			t.Errorf("Results do not match:\nexpected: %v\nreceived: %v", expRes, *r)
		}
	default:
		t.Fatalf("expected type order.Detail, found %T", res)
	}
	payload = []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"executionReport","E":1616633041556,"s":"BTCUSDT","c":"YeULctvPAnHj5HXCQo9Mob","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00028600","p":"52436.85000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":5341783271,"l":"0.00028600","z":"0.00028600","L":"52436.85000000","n":"0.00000029","N":"BTC","T":1616633041555,"t":726946523,"I":11390206312,"w":false,"m":false,"M":true,"O":1616633041555,"Z":"14.99693910","Y":"14.99693910","Q":"0.00000000"}}`)
	err = bi.wsHandleData(payload)
	if err != nil {
		t.Fatal("Binanceus OrderExecutionReport json conversion error", err)
	}
}

func TestWebsocketOutboundAccountPosition(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"stream":"jTfvpakT2yT0hVIo5gYWVihZhdM2PrBgJUZ5PyfZ4EVpCkx4Uoxk5timcrQc","data":{"e":"outboundAccountPosition","E":1616628815745,"u":1616628815745,"B":[{"a":"BTC","f":"0.00225109","l":"0.00123000"},{"a":"BNB","f":"0.00000000","l":"0.00000000"},{"a":"USDT","f":"54.43390661","l":"0.00000000"}]}}`)
	if err := bi.wsHandleData(payload); err != nil {
		t.Fatal("Binanceus testing \"outboundAccountPosition\" data conversion error", err)
	}
}

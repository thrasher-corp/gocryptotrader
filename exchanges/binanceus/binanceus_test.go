package binanceus

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
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
)

var (
	bi              Binanceus
	testPairMapping = currency.NewPair(currency.BTC, currency.USDT)
)

func TestMain(m *testing.M) {
	bi.SetDefaults()
	bi.validLimits = []int{5, 10, 20, 50, 100, 500, 1000}
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Binanceus")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = bi.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

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
	// println(info)
	// if mockTests {
	// 	serverTime := time.Date(2022, 2, 25, 3, 50, 40, int(601*time.Millisecond), time.UTC)
	// 	if !info.Servertime.Equal(serverTime) {
	// 		t.Errorf("Expected %v, got %v", serverTime, info.Servertime)
	// 	}
	// }
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
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := bi.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Binanceus Withdraw() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Binanceus Withdraw() expecting an error when no keys are set")
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

//

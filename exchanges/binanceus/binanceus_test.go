package binanceus

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	reflects "reflect"
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
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to test authenticated endpoints
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	bi              = &Binanceus{}
	testPairMapping = currency.NewPair(currency.BTC, currency.USDT)
	// this lock guards against orderbook tests race
	binanceusOrderBookLock = &sync.Mutex{}
)

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Binanceus load config error", err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Binanceus")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	bi.SetDefaults()
	bi.Websocket = sharedtestvalues.NewTestWebsocket()
	bi.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	err = bi.Setup(exchCfg)
	if err != nil {
		log.Fatal("Binanceus TestMain()", err)
	}
	bi.setupOrderbookManager()
	err = bi.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		log.Fatalf("%s received: '%v' but expected: '%v'", bi.Name, err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = bi.Start(context.Background(), &testWg)
	if err != nil {
		log.Fatal("Binanceus Starting error ", err)
	}
	os.Exit(m.Run())
}

func TestServerTime(t *testing.T) {
	t.Parallel()
	if _, er := bi.GetServerTime(context.Background(), asset.Spot); er != nil {
		t.Error("Binanceus SystemTime() error", er)
	}
}

func TestServerStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetSystemStatus(context.Background()); er != nil {
		t.Error("Binanceus GetSystemStatus() error", er)
	}
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.GetExchangeInfo(context.Background())
	if err != nil {
		t.Error("Binanceus GetExchangeInfo() error", err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	r, err := bi.UpdateTicker(context.Background(), testPairMapping, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if r.Pair.Base != currency.BTC && r.Pair.Quote != currency.USDT {
		t.Error("Binanceus UpdateTicker() invalid pair values")
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
	_, er := bi.UpdateOrderbook(context.Background(), testPairMapping, asset.Spot)
	if er != nil {
		t.Error("Binanceus UpdateOrderBook() error", er)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
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
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, err := bi.FetchAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Binanceus FetchAccountInfo() error", err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
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

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	pair := currency.Pair{Base: currency.BTC, Quote: currency.USD}
	_, err := bi.GetHistoricTrades(context.Background(), pair, asset.Spot, time.Time{}, time.Time{})
	if err != nil {
		t.Error("Binanceus GetHistoricTrades() error", err)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    currency.NewPair(currency.USD, currency.BTC),
		FeeType: exchange.CryptocurrencyTradeFee,
	}); er != nil {
		t.Error("Binanceus GetFeeByType() error", er)
	}
	if _, er := bi.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    currency.NewPair(currency.USD, currency.BTC),
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	}); er != nil {
		t.Error("Binanceus GetFeeByType() error", er)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
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
		Exchange:  bi.Name,
	}
	response, err := bi.SubmitOrder(context.Background(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(bi) && err != nil && strings.Contains(err.Error(), "{\"code\":-1013,\"msg\":\"Market is closed.\""):
		t.Skip("Binanceus SubmitOrder() Market is Closed")
	case sharedtestvalues.AreAPICredentialsSet(bi) && err != nil:
		t.Errorf("Binanceus SubmitOrder() Could not place order: %v", err)
	case sharedtestvalues.AreAPICredentialsSet(bi) && response.Status != order.Filled:
		t.Error("Binanceus SubmitOrder() Order not placed")
	case !sharedtestvalues.AreAPICredentialsSet(bi) && err == nil:
		t.Error("Binanceus SubmitOrder() Expecting an error when no keys are set")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	pair := currency.NewPair(currency.BTC, currency.USD)
	err := bi.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.Spot,
		OrderID:   "1337",
	})
	if err != nil && !errors.Is(err, errMissingCurrencySymbol) {
		t.Error("Binanceus CancelOrder() error", err)
	}
	err = bi.CancelOrder(context.Background(), &order.Cancel{
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
	})
	if err != nil && !(errors.Is(err, errEitherOrderIDOrClientOrderIDIsRequired) || strings.Contains(err.Error(), "ID not set")) {
		t.Errorf("Binanceus CancelOrder() expecting %v, but found %v", errEitherOrderIDOrClientOrderIDIsRequired, err)
	}
	var cancellationOrder = &order.Cancel{
		OrderID:   "1",
		Pair:      pair,
		AssetType: asset.Spot,
	}
	err = bi.CancelOrder(context.Background(), cancellationOrder)
	if err != nil && !strings.Contains(err.Error(), "Unknown order sent.") {
		t.Error("Binanceus CancelOrder() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	}
	if _, err := bi.CancelAllOrders(context.Background(), orderCancellation); err != nil {
		t.Error("Binanceus CancelAllOrders() error", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tradablePairs, err := bi.FetchTradablePairs(context.Background(),
		asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("Binanceus GetOrderInfo() no tradable pairs")
	}
	_, err = bi.GetOrderInfo(context.Background(),
		"123",
		tradablePairs[0],
		asset.Spot)
	if !strings.Contains(err.Error(), "Order does not exist.") {
		t.Error("Binanceus GetOrderInfo() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetDepositAddress(context.Background(), currency.EMPTYCODE, "", currency.BNB.String())
	if err != nil && !errors.Is(err, errMissingRequiredArgumentCoin) {
		t.Errorf("Binanceus GetDepositAddress() expecting %v, but found %v", errMissingRequiredArgumentCoin, err)
	}
	if _, err := bi.GetDepositAddress(context.Background(), currency.USDT, "", currency.BNB.String()); err != nil {
		t.Error("Binanceus GetDepositAddress() error", err)
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	_, err := bi.GetWithdrawalsHistory(context.Background(), currency.ETH, asset.Spot)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(bi) && err != nil:
		t.Error("Binanceus GetWithdrawalsHistory() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(bi) && err == nil:
		t.Error("Binanceus GetWithdrawalsHistory() expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	if _, er := bi.WithdrawFiat(context.Background(), &WithdrawFiatRequestParams{
		PaymentChannel: "SILVERGATE",
		PaymentAccount: "myaccount",
		PaymentMethod:  "SEN",
		Amount:         1,
	}); er != nil && !strings.Contains(er.Error(), "You are not authorized to execute this request.") {
		t.Error("Binanceus WithdrawFiat() error", er)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := bi.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error("Binanceus GetActiveOrders() error", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
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
	_, err := bi.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	if err != nil && !strings.EqualFold(errAmountValueMustBeGreaterThan0.Error(), err.Error()) {
		t.Errorf("Binanceus Withdraw() expecting %v, but found %v", errAmountValueMustBeGreaterThan0, err)
	} else if !sharedtestvalues.AreAPICredentialsSet(bi) && err == nil {
		t.Error("Binanceus Withdraw() expecting an error when no keys are set")
	}
	withdrawCryptoRequest.Amount = 1
	_, err = bi.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	if err != nil && !strings.Contains(err.Error(), "You are not authorized to execute this request.") {
		t.Error("Binanceus WithdrawCryptocurrencyFunds() error", err)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	var feeBuilder = &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
	_, er := bi.GetFeeByType(context.Background(), feeBuilder)
	if er != nil {
		t.Fatal("Binanceus GetFeeByType() error", er)
	}
	var withdrawalFeeBuilder = &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyWithdrawalFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
	_, er = bi.GetFeeByType(context.Background(), withdrawalFeeBuilder)
	if er != nil {
		t.Fatal("Binanceus GetFeeByType() error", er)
	}
	var offlineFeeTradeBuilder = &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.OfflineTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
	_, er = bi.GetFeeByType(context.Background(), offlineFeeTradeBuilder)
	if er != nil {
		t.Fatal("Binanceus GetFeeByType() error", er)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)

	_, err := bi.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.Interval(time.Hour*5), startTime, endTime)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Fatalf("received: '%v', but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}

	_, err = bi.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, startTime, endTime)
	if err != nil {
		t.Error("Binanceus GetHistoricCandles() error", err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)

	_, err := bi.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneDay, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}

	startTime = time.Now().Add(-time.Hour * 30)
	endTime = time.Now()

	_, err = bi.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.FourHour, startTime, endTime)
	if err != nil {
		t.Error("Binanceus GetHistoricCandlesExtended() error", err)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
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
	_, err := bi.GetAggregateTrades(context.Background(),
		&AggregatedTradeRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  5,
		})
	if err != nil {
		t.Error("Binanceus GetAggregateTrades() error", err)
	}
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOrderBookDepth(context.Background(), &OrderBookDataRequestParams{
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
	_, er := bi.GetPriceDatas(context.Background())
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetAccount(context.Background())
	if er != nil {
		t.Error("Binanceus GetAccount() error", er)
	}
}

func TestGetUserAccountStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetUserAccountStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAccountStatus() error", er)
	}
}

func TestGetUserAPITradingStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetUserAPITradingStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAPITradingStatus() error", er)
	}
}
func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetTradeFee(context.Background(), 3000, "BTC-USDT")
	if er != nil {
		t.Error("Binanceus GetTradeFee() error", er)
	}
}

func TestGetAssetDistributionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetAssetDistributionHistory(context.Background(), "", 0, 0, 3000)
	if er != nil {
		t.Error("Binanceus GetAssetDistributionHistory() error", er)
	}
}

func TestGetMasterAccountTotalUSDValue(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetMasterAccountTotalUSDValue(context.Background(), "", 0, 0); er != nil && !strings.Contains(er.Error(), "Sub-account function is not enabled.") {
		t.Errorf("Binanceus GetMasterAccountTotalUSDValue() expecting %s, but found %v", "Sub-account function is not enabled.", er)
	}
}

func TestGetSubaccountStatusList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetSubaccountStatusList(context.Background(), ""); er != nil && !errors.Is(er, errMissingSubAccountEmail) {
		t.Errorf("Binanceus GetSubaccountStatusList() expecting %v, but found %v", errMissingSubAccountEmail, er)
	}
	if _, er := bi.GetSubaccountStatusList(context.Background(), "someone@thrasher.corp"); er != nil && !strings.Contains(er.Error(), "Sub-account function is not enabled.") {
		t.Errorf("Binanceus GetSubaccountStatusList() expecting %s, but found %v", "Sub-account function is not enabled.", er)
	}
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetSubAccountDepositAddress(context.Background(), SubAccountDepositAddressRequestParams{}); er != nil && !errors.Is(er, errMissingSubAccountEmail) {
		t.Errorf("Binanceus GetSubAccountDepositAddress() %v, but found %v", errMissingSubAccountEmail, er)
	}
	if _, er := bi.GetSubAccountDepositAddress(context.Background(), SubAccountDepositAddressRequestParams{
		Email: "someone@thrasher.io",
	}); er != nil && !errors.Is(er, errMissingCurrencyCoin) {
		t.Errorf("Binanceus GetSubAccountDepositAddress() %v, but found %v", errMissingCurrencyCoin, er)
	}
	if _, er := bi.GetSubAccountDepositAddress(context.Background(), SubAccountDepositAddressRequestParams{
		Email: "someone@thrasher.io",
		Coin:  currency.BTC,
	}); er != nil && !strings.Contains(er.Error(), "This parent sub have no relation") {
		t.Errorf("Binanceus GetSubAccountDepositAddress() %v, but found %v", errMissingCurrencyCoin, er)
	}
}

var subAccountDepositHistoryItemJSON = `{
	"amount": "9.9749",
	"coin": "BTC", 
	"network": "btc",
	"status": 4, 
	"address": "bc1qxurvdd7tzn09agdvg3j8xpm3f7e978y07wg83s",
	"addressTag": "",
	"txId": "0x1b4b8c8090d15e3c1b0476b1c19118b1f00066e01de567cd7bc5b6e9c100193f",
	"insertTime": 1652942429211,
	"transferType": 0,
	"confirmTimes": "0/0"
}`

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	var resp SubAccountDepositItem
	if er := json.Unmarshal([]byte(subAccountDepositHistoryItemJSON), &resp); er != nil {
		t.Error("Binanceus Decerializing to SubAccountDepositItem error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetSubAccountDepositHistory(context.Background(), "", currency.BTC, 1, time.Time{}, time.Time{}, 0, 0); er != nil && !errors.Is(er, errMissingSubAccountEmail) {
		t.Errorf("Binanceus GetSubAccountDepositHistory() expecting %v, but found %v", errMissingSubAccountEmail, er)
	}
	if _, er := bi.GetSubAccountDepositHistory(context.Background(), "someone@thrasher.io", currency.BTC, 1, time.Time{}, time.Time{}, 0, 0); er != nil && !strings.Contains(er.Error(), "This parent sub have no relation") {
		t.Errorf("Binanceus GetSubAccountDepositHistory() expecting %s, but found %v", "This parent sub have no relation", er)
	}
}

var subaccountItemJSON = `{
	"email": "123@test.com",
	"status": "enabled",
	"activated": true,
	"mobile": "91605290",
	"gAuth": true,
	"createTime": 1544433328000
}`

func TestGetSubaccountInformation(t *testing.T) {
	t.Parallel()
	var resp SubAccount
	if er := json.Unmarshal([]byte(subaccountItemJSON), &resp); er != nil {
		t.Error("Binanceus decerializing to SubAccount error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetSubaccountInformation(context.Background(), 1, 100, "", "")
	if er != nil && !strings.Contains(er.Error(), "Sub-account function is not enabled.") {
		t.Error("Binanceus GetSubaccountInformation() error", er)
	}
}

var referalRewardHistoryResponse = `{
    "total": 1,
    "rows": [
        {
            "userId": 350991652,
            "rewardAmount": "8",
            "receiveDateTime": 1651131084091,
            "rewardType": "USD"
        }
    ]
}`

func TestGetReferralRewardHistory(t *testing.T) {
	t.Parallel()
	var resp ReferralRewardHistoryResponse
	if er := json.Unmarshal([]byte(referalRewardHistoryResponse), &resp); er != nil {
		t.Error("Binanceus decerializing to ReferalRewardHistoryResponse error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetReferralRewardHistory(context.Background(), 9, 5, 50); !errors.Is(er, errInvalidUserBusinessType) {
		t.Errorf("Binanceus GetReferralRewardHistory() expecting %v, but found %v", errInvalidUserBusinessType, er)
	}
	if _, er := bi.GetReferralRewardHistory(context.Background(), 1, 0, 50); !errors.Is(er, errMissingPageNumber) {
		t.Errorf("Binanceus GetReferralRewardHistory() expecting %v, but found %v", errMissingPageNumber, er)
	}
	if _, er := bi.GetReferralRewardHistory(context.Background(), 1, 5, 0); !errors.Is(er, errInvalidRowNumber) {
		t.Errorf("Binanceus GetReferralRewardHistory() expecting %v, but found %v", errInvalidRowNumber, er)
	}
	if _, er := bi.GetReferralRewardHistory(context.Background(), 1, 5, 50); er != nil {
		t.Error("Binanceus GetReferralRewardHistory() error", er)
	}
}

func TestGetSubaccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetSubaccountTransferHistory(context.Background(), "", 0, 0, 0, 0)
	if !errors.Is(er, errNotValidEmailAddress) {
		t.Errorf("Binanceus GetSubaccountTransferHistory() expected %v, but received: %s", errNotValidEmailAddress, er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er = bi.GetSubaccountTransferHistory(context.Background(), "example@golang.org", 0, 0, 0, 0)
	if er != nil && !(errors.Is(er, errNotValidEmailAddress) || strings.Contains(er.Error(), "Sub-account function is not enabled.")) {
		t.Fatalf("Binanceus GetSubaccountTransferHistory() error %v", er)
	}
}

func TestExecuteSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.ExecuteSubAccountTransfer(context.Background(), &SubAccountTransferRequestParams{})
	if !errors.Is(er, errUnacceptableSenderEmail) {
		t.Errorf("binanceus error: expected %v, but found %v", errUnacceptableSenderEmail, er)
	}
	_, er = bi.ExecuteSubAccountTransfer(context.Background(), &SubAccountTransferRequestParams{
		FromEmail: "fromemail@thrasher.io",
		ToEmail:   "toemail@threasher.io",
		Asset:     "BTC",
		Amount:    0.000005,
	})
	if er != nil && !strings.Contains(er.Error(), "You are not authorized to execute this request.") {
		t.Errorf("Binanceus GetSubaccountTransferHistory() error %v", er)
	}
}

func TestGetSubaccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetSubaccountAssets(context.Background(), "")
	if !errors.Is(er, errNotValidEmailAddress) {
		t.Errorf("Binanceus GetSubaccountAssets() expected %v, but found %v", er, errNotValidEmailAddress)
	}
	_, er = bi.GetSubaccountAssets(context.Background(), "subaccount@thrasher.io")
	if er != nil && !strings.Contains(er.Error(), "This account does not exist.") {
		t.Fatal("Binanceus GetSubaccountAssets() error", er)
	}
}

func TestGetOrderRateLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetOrderRateLimits(context.Background(), 0)
	if er != nil {
		t.Error("Binanceus GetOrderRateLimits() error", er)
	}
}

var testNewOrderResponseJSON = `{
	"symbol": "BTCUSDT",
	"orderId": 28,
	"orderListId": -1,
	"clientOrderId": "6gCrw2kRUAF9CvJDGP16IP",
	"transactTime": 1507725176595,
	"price": "0.00000000",
	"origQty": "10.00000000",
	"executedQty": "10.00000000",
	"cummulativeQuoteQty": "10.00000000",
	"status": "FILLED",
	"timeInForce": "GTC",
	"type": "MARKET",
	"side": "SELL"
  }`

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	var resp NewOrderResponse
	if er := json.Unmarshal([]byte(testNewOrderResponseJSON), &resp); er != nil {
		t.Error("Binanceus decerializing to Order error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req := &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   BinanceRequestParamsOrderLimit,
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}
	_, err := bi.NewOrderTest(context.Background(), req)
	if err != nil {
		t.Error("Binanceus NewOrderTest() error", err)
	}
	req = &NewOrderRequest{
		Symbol:        currency.NewPair(currency.LTC, currency.BTC),
		Side:          order.Sell.String(),
		TradeType:     BinanceRequestParamsOrderMarket,
		Price:         0.0045,
		QuoteOrderQty: 10,
	}
	_, err = bi.NewOrderTest(context.Background(), req)
	if err != nil {
		t.Error("NewOrderTest() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req := &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		TradeType:   BinanceRequestParamsOrderLimit,
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}
	if _, err := bi.NewOrder(context.Background(), req); err != nil && !strings.Contains(err.Error(), "Account has insufficient balance for requested action") {
		t.Error("Binanceus NewOrder() error", err)
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetOrder(context.Background(), &OrderRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus GetOrder() error expecting %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.GetOrder(context.Background(), &OrderRequestParams{
		Symbol:            "BTCUSDT",
		OrigClientOrderID: "something",
	})
	// You can check the existence of an order using a valid Symbol and OrigClient Order ID
	if er != nil && !strings.Contains(er.Error(), "Order does not exist.") {
		t.Error("Binanceus GetOrder() error", er)
	}
}

var openOrdersItemJSON = `{
    "symbol": "LTCBTC",
    "orderId": 1,
    "orderListId": -1,
    "clientOrderId": "myOrder1",
    "price": "0.1",
    "origQty": "1.0",
    "executedQty": "0.0",
    "cummulativeQuoteQty": "0.0",
    "status": "NEW",
    "timeInForce": "GTC",
    "type": "LIMIT",
    "side": "BUY",
    "stopPrice": "0.0",
    "icebergQty": "0.0",
    "time": 1499827319559,
    "updateTime": 1499827319559,
    "isWorking": true,
    "origQuoteOrderQty": "0.000000"
  }`

func TestGetAllOpenOrders(t *testing.T) {
	t.Parallel()
	var resp Order
	if er := json.Unmarshal([]byte(openOrdersItemJSON), &resp); er != nil {
		t.Error("Binanceus decerializing to Order error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)

	_, er := bi.GetAllOpenOrders(context.Background(), "")
	if er != nil {
		t.Error("Binanceus GetAllOpenOrders() error", er)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.CancelExistingOrder(context.Background(), &CancelOrderRequestParams{Symbol: currency.NewPair(currency.BTC, currency.USDT)})
	if er != nil && !errors.Is(er, errEitherOrderIDOrClientOrderIDIsRequired) {
		t.Errorf("Binanceus CancelExistingOrder() error expecting %v, but found %v", errEitherOrderIDOrClientOrderIDIsRequired, er)
	}
	_, er = bi.CancelExistingOrder(context.Background(), &CancelOrderRequestParams{
		Symbol:                currency.NewPair(currency.BTC, currency.USDT),
		ClientSuppliedOrderID: "1234",
	})
	if er != nil && !strings.Contains(er.Error(), "Unknown order sent.") {
		t.Error("Binanceus CancelExistingorder() error", er)
	}
}

func TestCancelOpenOrdersForSymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.CancelOpenOrdersForSymbol(context.Background(), "")
	if !errors.Is(er, errMissingCurrencySymbol) {
		t.Errorf("Binanceus CancelOpenOrdersForSymbol() error expecting %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.CancelOpenOrdersForSymbol(context.Background(), "BTCUSDT")
	if er != nil && !strings.Contains(er.Error(), "Unknown order sent") {
		t.Error("Binanceus CancelOpenOrdersForSymbol() error", er)
	}
}

// TestGetTrades test for fetching the list of
// trades attached with this account.
func TestGetTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetTrades(context.Background(), &GetTradesParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf(" Binanceus GetTrades() expecting error %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.GetTrades(context.Background(), &GetTradesParams{Symbol: "BTCUSDT"})
	if er != nil {
		t.Error("Binanceus GetTrades() error", er)
	}
}

func TestCreateNewOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.CreateNewOCOOrder(context.Background(),
		&OCOOrderInputParams{
			StopPrice: 1000,
			Side:      order.Buy.String(),
			Quantity:  0.0000001,
			Price:     1232334.00,
		})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CreatenewOCOOrder() error expected %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.CreateNewOCOOrder(
		context.Background(),
		&OCOOrderInputParams{
			Symbol:               "XTZUSD",
			Price:                100,
			StopPrice:            3,
			StopLimitPrice:       2.5,
			Side:                 order.Buy.String(),
			Quantity:             1,
			StopLimitTimeInForce: "GTC",
			RecvWindow:           6000,
		})
	if er != nil && !strings.Contains(er.Error(), "Precision is over the maximum defined for this asset.") {
		t.Error("Binanceus CreateNewOCOOrder() error", er)
	}
}

var ocoOrderJSON = `{
	"orderListId": 27,
	"contingencyType": "OCO",
	"listStatusType": "EXEC_STARTED",
	"listOrderStatus": "EXECUTING",
	"listClientOrderId": "h2USkA5YQpaXHPIrkd96xE",
	"transactionTime": 1565245656253,
	"symbol": "LTCBTC",
	"orders": [
	  {
		"symbol": "LTCBTC",
		"orderId": 4,
		"clientOrderId": "qD1gy3kc3Gx0rihm9Y3xwS"
	  },
	  {
		"symbol": "LTCBTC",
		"orderId": 5,
		"clientOrderId": "ARzZ9I00CPM8i3NhmU9Ega"
	  }
	]
  }`

func TestGetOCOOrder(t *testing.T) {
	t.Parallel()
	var resp OCOOrderResponse
	if er := json.Unmarshal([]byte(ocoOrderJSON), &resp); er != nil {
		t.Error("Binanceus decerializing OCOOrderResponse error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetOCOOrder(context.Background(), &GetOCOOrderRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus GetOCOOrder() error  expecting %v, but found %v", errIncompleteArguments, er)
	}
	_, er = bi.GetOCOOrder(context.Background(), &GetOCOOrderRequestParams{
		OrderListID: "123445",
	})
	if er != nil && !strings.Contains(er.Error(), "Order list does not exist.") {
		t.Error("Binanceus GetOCOOrder() error", er)
	}
}

func TestGetAllOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetAllOCOOrder(context.Background(), &OCOOrdersRequestParams{})
	if er != nil {
		t.Error("Binanceus GetAllOCOOrder() error", er)
	}
}

func TestGetOpenOCOOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetOpenOCOOrders(context.Background(), 0)
	if er != nil {
		t.Error("Binanceus GetOpenOCOOrders() error", er)
	}
}

func TestCancelOCOOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.CancelOCOOrder(context.Background(), &OCOOrdersDeleteRequestParams{})
	if !errors.Is(er, errIncompleteArguments) {
		t.Errorf("Binanceus CancelOCOOrder() error expected %v, but found %v", errIncompleteArguments, er)
	}
}

// OTC end Points test code.
func TestGetSupportedCoinPairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetSupportedCoinPairs(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT})
	if er != nil {
		t.Error("Binanceus GetSupportedCoinPairs() error", er)
	}
}

func TestRequestForQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.RequestForQuote(context.Background(), &RequestQuoteParams{ToCoin: "BTC", RequestCoin: "USDT", RequestAmount: 1})
	if er != nil && !errors.Is(er, errMissingFromCoinName) {
		t.Errorf("Binanceus RequestForQuote() expecting %v, but found %v", errMissingFromCoinName, er)
	}
	_, er = bi.RequestForQuote(context.Background(), &RequestQuoteParams{FromCoin: "ETH", RequestCoin: "USDT", RequestAmount: 1})
	if er != nil && !errors.Is(er, errMissingToCoinName) {
		t.Errorf("Binanceus RequestForQuote() expecting %v, but found %v", errMissingToCoinName, er)
	}
	_, er = bi.RequestForQuote(context.Background(), &RequestQuoteParams{FromCoin: "ETH", ToCoin: "BTC", RequestCoin: "USDT"})
	if er != nil && !errors.Is(er, errMissingRequestAmount) {
		t.Errorf("Binanceus RequestForQuote() expecting %v, but found %v", errMissingRequestAmount, er)
	}
	_, er = bi.RequestForQuote(context.Background(), &RequestQuoteParams{FromCoin: "ETH", ToCoin: "BTC", RequestAmount: 1})
	if er != nil && !errors.Is(er, errMissingRequestCoin) {
		t.Errorf("Binanceus RequestForQuote() expecting %v, but found %v", errMissingRequestCoin, er)
	}
	_, er = bi.RequestForQuote(context.Background(), &RequestQuoteParams{FromCoin: "BTC", ToCoin: "USDT", RequestCoin: "BTC", RequestAmount: 1})
	if er != nil {
		t.Error("Binanceus RequestForQuote() error", er)
	}
}

var testPlaceOTCTradeOrderJSON = `{
    "orderId": "10002349",
    "createTime": 1641906714,
    "orderStatus": "PROCESS"
}`

func TestPlaceOTCTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	var res OTCTradeOrderResponse
	er := json.Unmarshal([]byte(testPlaceOTCTradeOrderJSON), &res)
	if er != nil {
		t.Error("Binanceus PlaceOTCTradeOrder() error", er)
	}
	_, er = bi.PlaceOTCTradeOrder(context.Background(), "")
	if !errors.Is(er, errMissingQuoteID) {
		t.Errorf("Binanceus PlaceOTCTradeOrder()  expecting %v, but found %v", errMissingQuoteID, er)
	}
	_, er = bi.PlaceOTCTradeOrder(context.Background(), "15848701022")
	if er != nil && !strings.Contains(er.Error(), "-9000") {
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
}`

func TestGetOTCTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	var val OTCTradeOrder
	er := json.Unmarshal([]byte(testGetOTCTradeOrderJSON), &val)
	if er != nil {
		t.Error("Binanceus JSON GetOTCTradeOrder() error", er)
	}
	_, er = bi.GetOTCTradeOrder(context.Background(), 10002349)
	if er != nil && !strings.Contains(er.Error(), "status code: 400") {
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	var orders []OTCTradeOrder
	er := json.Unmarshal([]byte(getAllOTCTradeOrders), &orders)
	if er != nil {
		t.Error(er)
	}
	_, er = bi.GetAllOTCTradeOrders(context.Background(), &OTCTradeOrderRequestParams{})
	if er != nil {
		t.Error("Binanceus GetAllOTCTradeOrders() error", er)
	}
}

var ocbsTradeOrderJSON = `
{
  "quoteId": "4e5446f2cc6f44ab86ab02abf19abvd",
  "orderId": "1000238000", 
  "orderStatus": "FAIL",
  "fromCoin": "USD",
  "fromAmount": 1000.5,
  "toCoin": "ETH",
  "toAmount": 0.5,
  "feeCoin": "USD",
  "feeAmount": 0.5,
  "ratio": 2000,
  "createTime": 1641916714
}`

func TestGetAllOCBSTradeOrders(t *testing.T) {
	t.Parallel()
	var orderDetail OCBSOrder
	if er := json.Unmarshal([]byte(ocbsTradeOrderJSON), &orderDetail); er != nil {
		t.Error("Binanceus decerializing to OCBSOrder error", er)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetAllOCBSTradeOrders(context.Background(), OCBSOrderRequestParams{}); er != nil {
		t.Error("Binanceus GetAllOCBSTradeOrders() error", er)
	}
}

func TestGetAssetFeesAndWalletStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetAssetFeesAndWalletStatus(context.Background())
	if er != nil {
		t.Error("Binanceus GetAssetFeesAndWalletStatus()  error", er)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, er := bi.WithdrawCrypto(context.Background(), &withdraw.Request{})
	if !errors.Is(er, errMissingRequiredArgumentCoin) {
		t.Errorf("Binanceus WithdrawCrypto() error expecting %v, but found %v", errMissingRequiredArgumentCoin, er)
	}
	if _, er = bi.WithdrawCrypto(context.Background(), &withdraw.Request{
		Currency: currency.BTC,
	}); !errors.Is(er, errMissingRequiredArgumentNetwork) {
		t.Errorf("Binanceus WithdrawCrypto() expecting %v, but found %v", errMissingRequiredArgumentNetwork, er)
	}
	params := &withdraw.Request{
		Currency: currency.BTC,
	}
	params.Crypto.Chain = "BSC"
	if _, er = bi.WithdrawCrypto(context.Background(), params); !errors.Is(er, errMissingRequiredParameterAddress) {
		t.Errorf("Binanceus WithdrawCrypto() expecting %v, but found %v", errMissingRequiredParameterAddress, er)
	}
	params.Crypto.Address = "1234567"
	if _, er = bi.WithdrawCrypto(context.Background(), params); !errors.Is(er, errAmountValueMustBeGreaterThan0) {
		t.Errorf("Binanceus WithdrawCrypto() expecting %v, but found %v", errAmountValueMustBeGreaterThan0, er)
	}
	params.Amount = 1
	if _, er = bi.WithdrawCrypto(context.Background(), params); er != nil && !strings.Contains(er.Error(), "You are not authorized to execute this request.") {
		t.Error("Binanceus WithdrawCrypto() error", er)
	}
}

func TestFiatWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.FiatWithdrawalHistory(context.Background(), &FiatWithdrawalRequestParams{
		FiatCurrency: "USDT",
	})
	if er != nil {
		t.Errorf("%s FiatWithdrawalHistory() error %v", bi.Name, er)
	}
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.DepositHistory(context.Background(), currency.USD, 1, time.Time{}, time.Time{}, 0, 100)
	if er != nil {
		t.Error("Binanceus DepositHistory() error", er)
	}
}
func TestFiatDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.FiatDepositHistory(context.Background(), &FiatWithdrawalRequestParams{})
	if er != nil {
		t.Error("Binanceus FiatDepositHistory() error", er)
	}
}

// WEBSOCKET support testing
// Since both binance and Binance US has same websocket functions,
// the tests functions are also similar

// TestWebsocketStreamKey  this test mmethod handles the
// creating, updating, and deleting of user stream key or "listenKey"
// all the three methods in one test methods.
func TestWebsocketStreamKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, er := bi.GetWsAuthStreamKey(context.Background())
	if er != nil {
		t.Error("Binanceus GetWsAuthStreamKey() error", er)
	}
	er = bi.MaintainWsAuthStreamKey(context.Background())
	if er != nil {
		t.Error("Binanceus MaintainWsAuthStreamKey() error", er)
	}
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	rawData := []byte(subscriptionRequestString)
	err := bi.wsHandleData(rawData)
	if err != nil {
		t.Error("Binanceus wsHandleData() error", err)
	}
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

func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := bi.GetSubscriptions(); err != nil {
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
}`

func TestWebsocketTickerUpdate(t *testing.T) {
	t.Parallel()
	if err := bi.wsHandleData([]byte(ticker24hourChangeStream)); err != nil {
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
	if err := bi.wsHandleData(pressXToJSON); err != nil {
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
	if err := bi.wsHandleData(pressXToJSON); err != nil {
		t.Error("Binanceus wsHandleData() error", err)
	}
}

// TestWsDepthUpdate copied from the Binance Test
func TestWebsocketOrderBookDepthDiffStream(t *testing.T) {
	binanceusOrderBookLock.Lock()
	defer binanceusOrderBookLock.Unlock()
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
	update2 := []byte(`{
		"stream":"btcusdt@depth","data":{
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
		}
	}`)
	if err = bi.wsHandleData(update2); err != nil {
		t.Error("Binanceus wshandlerData error", err)
	}
	ob, err = bi.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
	if err != nil {
		t.Fatal("Binanceus GetOrderBook error", err)
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

// TestWebsocketPartialOrderBookDepthStream copied from the Binance Test
func TestWebsocketPartialOrderBookDepthStream(t *testing.T) {
	t.Parallel()
	update1 := []byte(`{"stream":"btcusdt@depth5","data":
	{
		"lastUpdateId": 160,
		"bids": [           
		  [
			"0.0024",       
			"10"            
		  ]
		],
		"asks": [           
		  [
			"0.0026",       
			"100"           
		  ]
		]
	  }}`)
	var err error
	if err = bi.wsHandleData(update1); err != nil {
		t.Error("Binanceus Partial Order Book Depth Sream error", err)
	}
	update2 := []byte(`{
		"stream":"btcusdt@depth10",
		"data":{
			"lastUpdateId": 160, 
			"bids": [            
					[
						"0.0024",        
						"10"             
					]
			],
			"asks": [            
				[
					"0.0026",        
					"100"            
				]
			]
		}
	  }`)
	if err = bi.wsHandleData(update2); err != nil {
		t.Error("Binanceus Partial Order Book Depth Sream error", err)
	}
}

func TestWebsocketBookTicker(t *testing.T) {
	t.Parallel()
	var bookTickerJSON = []byte(
		`{
		"stream": "btcusdt@bookTicker",
		"data": {
			"u":400900217,   
			"s":"BNBUSDT",  
			"b":"25.35190000",
			"B":"31.21000000",
			"a":"25.36520000",
			"A":"40.66000000" 
		}
	  }`)
	if err := bi.wsHandleData(bookTickerJSON); err != nil {
		t.Error("Binanceus Book Ticker error", err)
	}
	var bookTickerForAllSymbols = []byte(`
	{
		"stream" : "!bookTicker",
		"data":{
			"u":400900217,    
			"s":"BNBUSDT",    
			"b":"25.35190000",
			"B":"31.21000000",
			"a":"25.36520000",
			"A":"40.66000000" 
		}
	}`)
	if err := bi.wsHandleData(bookTickerForAllSymbols); err != nil {
		t.Error("Binanceus Web socket Book ticker for all symbols error", err)
	}
}

func TestWebsocketAggTrade(t *testing.T) {
	t.Parallel()
	var aggTradejson = []byte(
		`{  
			"stream":"btcusdt@aggTrade", 
			"data": {
				"e": "aggTrade",  
				"E": 123456789,   
				"s": "BNBBTC",    
				"a": 12345,       
				"p": "0.001",     
				"q": "100",   
				"f": 100,     
				"l": 105,   
				"T": 123456785,
				"m": true,
				"M": true         
			}
	   }`)
	if err := bi.wsHandleData(aggTradejson); err != nil {
		t.Error("Binanceus Aggregated Trade Order Json() error", err)
	}
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
	if err := bi.wsHandleData(thejson); err != nil {
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
	if err := bi.wsHandleData([]byte(listStatusUserDataStreamPayload)); err != nil {
		t.Error(err)
	}
}

func TestExecutionTypeToOrderStatus(t *testing.T) {
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
			t.Errorf("Binanceus expected: %v, received: %v", testCases[i].Result, result)
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
	binanceusOrderBookLock.Lock()
	defer binanceusOrderBookLock.Unlock()
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
	expectedResult := order.Detail{
		Price:           52789.1,
		Amount:          0.00028400,
		RemainingAmount: 0.00028400,
		CostAsset:       currency.USDT,
		FeeAsset:        currency.BTC,
		Exchange:        "Binanceus",
		OrderID:         "5340845958",
		ClientOrderID:   "c4wyKsIhoAaittTYlIVLqk",
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.New,
		AssetType:       asset.Spot,
		Date:            time.UnixMilli(1616627567900),
		LastUpdated:     time.UnixMilli(1616627567900),
		Pair:            currency.NewPair(currency.BTC, currency.USDT),
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
		if !reflects.DeepEqual(expectedResult, *r) {
			t.Errorf("Binanceus Results do not match:\nexpected: %v\nreceived: %v", expectedResult, *r)
		}
	default:
		t.Fatalf("Binanceus expected type order.Detail, found %T", res)
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

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetAvailableTransferChains(context.Background(), currency.BTC); er != nil {
		t.Error("Binanceus GetAvailableTransferChains() error", er)
	}
}

func TestQuickEnableCryptoWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if er := bi.QuickEnableCryptoWithdrawal(context.Background()); er != nil && !strings.Contains(er.Error(), "unexpected end of JSON input") {
		t.Errorf("Binanceus QuickEnableCryptoWithdrawal() expecting %s, but found %v", "unexpected end of JSON input", er)
	}
}
func TestQuickDisableCryptoWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if er := bi.QuickDisableCryptoWithdrawal(context.Background()); er != nil && !strings.Contains(er.Error(), "unexpected end of JSON input") {
		t.Errorf("Binanceus QuickDisableCryptoWithdrawal() expecting %s, but found %v", "unexpected end of JSON input", er)
	}
}

func TestGetUsersSpotAssetSnapshot(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	if _, er := bi.GetUsersSpotAssetSnapshot(context.Background(), time.Time{}, time.Time{}, 10, 6); er != nil {
		t.Error("Binanceus GetUsersSpotAssetSnapshot() error", er)
	}
}

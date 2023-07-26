package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b = &Bybit{}

var spotTradablePair, linearTradablePair, inverseTradablePair, optionsTradablePair currency.Pair

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.Enabled = true
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = false
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	request.MaxRequestJobs = 100
	err = b.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	// Turn on all pairs for testing
	supportedAssets := b.GetAssetTypes(false)
	for x := range supportedAssets {
		avail, err := b.GetAvailablePairs(supportedAssets[x])
		if err != nil {
			log.Fatal(err)
		}

		err = b.CurrencyPairs.StorePairs(supportedAssets[x], avail, true)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = instantiateTradablePairs()
	if err != nil {
		log.Fatalf("%s %v", b.Name, err)
	}
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := b.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = b.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

// test cases for SPOT

func TestGetInstrumentInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetInstruments(context.Background(), "spot", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetInstruments(context.Background(), "linear", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetInstruments(context.Background(), "inverse", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetInstruments(context.Background(), "option", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := b.GetKlines(context.Background(), "spot", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetKlines(context.Background(), "linear", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetKlines(context.Background(), "inverse", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetKlines(context.Background(), "option", "BTC-7JUL23-27000-C", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err == nil {
		t.Fatalf("expected 'params error: Category is invalid', but found nil")
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkPriceKline(context.Background(), "linear", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetMarkPriceKline(context.Background(), "inverse", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetMarkPriceKline(context.Background(), "option", "BTC-7JUL23-27000-C", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err == nil {
		t.Fatalf("expected 'params error: Category is invalid', but found nil")
	}
}

func TestGetIndexPriceKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndexPriceKline(context.Background(), "linear", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetIndexPriceKline(context.Background(), "inverse", "BTCUSDT", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetIndexPriceKline(context.Background(), "option", "BTC-7JUL23-27000-C", kline.FiveMin, time.Now().Add(-time.Hour*1), time.Now(), 5)
	if err == nil {
		t.Fatalf("expected 'params error: Category is invalid', but found nil")
	}
}

// func TestGetAllSpotPairs(t *testing.T) {
// 	t.Parallel()
// 	_, err := b.GetAllSpotPairs(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(context.Background(), "spot", "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(context.Background(), "linear", "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(context.Background(), "inverse", "BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(context.Background(), "option", "BTC-7JUL23-27000-C", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRiskLimit(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRiskLimit(context.Background(), "linear", pair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(context.Background(), "inverse", "BTCUSDM23")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(context.Background(), "option", "BTCUSDM23")
	if !errors.Is(err, errInvalidCategory) {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(context.Background(), "spot", "BTCUSDM23")
	if !errors.Is(err, errInvalidCategory) {
		t.Error(err)
	}
}

// test cases for Wrapper
func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.UpdateTicker(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(context.Background(), pair, asset.Linear)
	if err != nil {
		t.Error(err)
	}
	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), pair1, asset.Inverse)
	if err != nil {
		t.Error(err)
	}

	// Futures update dynamically, so fetch the available tradable futures for this test
	availPairs, err := b.FetchTradablePairs(context.Background(), asset.Options)
	if err != nil {
		t.Fatal(err)
	}

	// Needs to be set before calling extractCurrencyPair
	if err = b.SetPairs(availPairs, asset.Futures, true); err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(context.Background(), availPairs[0], asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair, asset.Futures)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateOrderbook(context.Background(), pair1, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	end := time.Now()
	start := end.AddDate(0, 0, -3)
	_, err := b.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), linearTradablePair, asset.Linear, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), inverseTradablePair, asset.Inverse, kline.OneHour, start, end)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.OneHour, start, end)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, got %v", err, asset.ErrNotSupported)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)
	_, err := b.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(context.Background(), inverseTradablePair, asset.Inverse, kline.OneHour, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(context.Background(), linearTradablePair, asset.Linear, kline.OneDay, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(context.Background(), optionsTradablePair, asset.Options, kline.FiveMin, startTime, end)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("found %v, expected %v", err, asset.ErrNotSupported)
	}
}

// func TestFetchAccountInfo(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err := b.FetchAccountInfo(context.Background(), asset.Spot)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.FetchAccountInfo(context.Background(), asset.CoinMarginedFutures)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.FetchAccountInfo(context.Background(), asset.USDTMarginedFutures)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.FetchAccountInfo(context.Background(), asset.Futures)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.FetchAccountInfo(context.Background(), asset.USDCMarginedFutures)
// 	if err != nil && err.Error() != "System error. Please try again later." {
// 		t.Error(err)
// 	}
// }

// func TestSubmitOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	var oSpot = &order.Submit{
// 		Exchange: "Bybit",
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.LTC,
// 			Quote:     currency.BTC,
// 		},
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     0.0001,
// 		Amount:    10,
// 		ClientID:  "newOrder",
// 		AssetType: asset.Spot,
// 	}
// 	_, err := b.SubmitOrder(context.Background(), oSpot)
// 	if err != nil {
// 		if strings.TrimSpace(err.Error()) != "Balance insufficient" {
// 			t.Error(err)
// 		}
// 	}

// 	var oCMF = &order.Submit{
// 		Exchange: "Bybit",
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USD,
// 		},
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     10000,
// 		Amount:    1,
// 		ClientID:  "newOrder",
// 		AssetType: asset.CoinMarginedFutures,
// 	}
// 	_, err = b.SubmitOrder(context.Background(), oCMF)
// 	if err == nil {
// 		t.Error("SubmitOrder() Expected error")
// 	}

// 	var oUMF = &order.Submit{
// 		Exchange: "Bybit",
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USDT,
// 		},
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     10000,
// 		Amount:    1,
// 		ClientID:  "newOrder",
// 		AssetType: asset.USDTMarginedFutures,
// 	}
// 	_, err = b.SubmitOrder(context.Background(), oUMF)
// 	if err == nil {
// 		t.Error("SubmitOrder() Expected error")
// 	}

// 	pair, err := currency.NewPairFromString("BTCUSDZ22")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var oFutures = &order.Submit{
// 		Exchange:  "Bybit",
// 		Pair:      pair,
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     10000,
// 		Amount:    1,
// 		ClientID:  "newOrder",
// 		AssetType: asset.Futures,
// 	}
// 	_, err = b.SubmitOrder(context.Background(), oFutures)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	pair1, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var oUSDC = &order.Submit{
// 		Exchange:  "Bybit",
// 		Pair:      pair1,
// 		Side:      order.Buy,
// 		Type:      order.Limit,
// 		Price:     10000,
// 		Amount:    1,
// 		ClientID:  "newOrder",
// 		AssetType: asset.USDCMarginedFutures,
// 	}
// 	_, err = b.SubmitOrder(context.Background(), oUSDC)
// 	if err != nil && err.Error() != "margin account not exist" {
// 		t.Error(err)
// 	}
// }

// func TestModifyOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	_, err := b.ModifyOrder(context.Background(), &order.Modify{
// 		Exchange: "Bybit",
// 		OrderID:  "1337",
// 		Price:    10000,
// 		Amount:   10,
// 		Side:     order.Sell,
// 		Pair: currency.Pair{
// 			Delimiter: "-",
// 			Base:      currency.BTC,
// 			Quote:     currency.USD,
// 		},
// 		AssetType: asset.CoinMarginedFutures,
// 	})
// 	if err == nil {
// 		t.Error("ModifyOrder() Expected error")
// 	}
// }

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.Spot,
		Pair:      spotTradablePair,
		OrderID:   "1234"})
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.Linear,
		Pair:      linearTradablePair,
		OrderID:   "1234"})
	if err != nil {
		t.Error(err)
	}

	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.Inverse,
		Pair:      inverseTradablePair,
		OrderID:   "1234"})
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(context.Background(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		OrderID:   "1234"})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelAllOrders(context.Background(), &order.Cancel{AssetType: asset.Spot})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(context.Background(), &order.Cancel{Exchange: b.Name, AssetType: asset.Linear, Pair: linearTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(context.Background(), &order.Cancel{Exchange: b.Name, AssetType: asset.Inverse, Pair: inverseTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(context.Background(), &order.Cancel{Exchange: b.Name, AssetType: asset.Options, Pair: optionsTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(context.Background(), &order.Cancel{Exchange: b.Name, AssetType: asset.Futures})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, but found %v", asset.ErrNotSupported, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	results, err := b.GetOrderInfo(context.Background(),
		"12234", spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	} else {
		val, _ := json.Marshal(results)
		println(string(val))
	}
	_, err = b.GetOrderInfo(context.Background(),
		"12234", linearTradablePair, asset.Linear)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo(context.Background(),
		"12234", inverseTradablePair, asset.Inverse)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo(context.Background(),
		"12234", optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	var getOrdersRequestSpot = order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}
	_, err := b.GetActiveOrders(context.Background(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestLinear = order.MultiOrderRequest{Pairs: currency.Pairs{linearTradablePair}, AssetType: asset.Linear, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestLinear)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestInverse = order.MultiOrderRequest{Pairs: currency.Pairs{inverseTradablePair}, AssetType: asset.Inverse, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestInverse)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestFutures = order.MultiOrderRequest{Pairs: currency.Pairs{optionsTradablePair}, AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	var getOrdersRequestSpot = order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := b.GetOrderHistory(context.Background(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestUMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{linearTradablePair},
		AssetType: asset.Linear,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestCMF = order.MultiOrderRequest{
		Pairs:     currency.Pairs{inverseTradablePair},
		AssetType: asset.Inverse,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestCMF)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequestFutures = order.MultiOrderRequest{
		Pairs:     currency.Pairs{optionsTradablePair},
		AssetType: asset.Options,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetDepositAddress(context.Background(), currency.USDT, "", currency.ETH.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange: "Bybit",
		Amount:   10,
		Currency: currency.LTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		}})
	if err != nil && err.Error() != "Withdraw address chain or destination tag are not equal" {
		t.Fatal(err)
	}
}

// // test cases for USDCMarginedFutures

// func TestGetUSDCFuturesOrderbook(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCFuturesOrderbook(context.Background(), pair)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCContracts(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCContracts(context.Background(), pair, "next", 1500)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.GetUSDCContracts(context.Background(), currency.EMPTYPAIR, "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCSymbols(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCSymbols(context.Background(), pair)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCKlines(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCMarkPriceKlines(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCMarkPriceKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCIndexPriceKlines(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCIndexPriceKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCPremiumIndexKlines(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCPremiumIndexKlines(context.Background(), pair, "5", time.Now().Add(-time.Hour), 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCOpenInterest(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCOpenInterest(context.Background(), pair, "1d", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCLargeOrders(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCLargeOrders(context.Background(), pair, 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCAccountRatio(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCAccountRatio(context.Background(), pair, "1d", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCLatestTrades(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCLatestTrades(context.Background(), pair, "PERPETUAL", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestPlaceUSDCOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.PlaceUSDCOrder(context.Background(), pair, "Limit", "Order", "Buy", "", "", 10000, 1, 0, 0, 0, 0, 0, 0, false, false, false)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.PlaceUSDCOrder(context.Background(), pair, "Market", "StopOrder", "Buy", "ImmediateOrCancel", "", 0, 64300, 0, 0, 0, 0, 1000, 0, false, false, false)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestModifyUSDCOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.ModifyUSDCOrder(context.Background(), pair, "Order", "", "orderLinkID", 0, 0, 0, 0, 0, 0, 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestCancelUSDCOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.CancelUSDCOrder(context.Background(), pair, "Order", "", "orderLinkID")
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestCancelAllActiveUSDCOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = b.CancelAllActiveUSDCOrder(context.Background(), pair, "Order")
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetActiveUSDCOrder(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetActiveUSDCOrder(context.Background(), pair, "PERPETUAL", "", "", "", "", "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCOrderHistory(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCOrderHistory(context.Background(), pair, "PERPETUAL", "", "", "", "", "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCTradeHistory(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCTradeHistory(context.Background(), pair, "PERPETUAL", "", "orderLinkID", "", "", 50, time.Now().Add(-time.Hour))
// 	if err == nil { // order with link ID "orderLinkID" not present
// 		t.Error("GetUSDCTradeHistory() Expected error")
// 	}
// }

// func TestGetUSDCTransactionLog(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err := b.GetUSDCTransactionLog(context.Background(), time.Time{}, time.Time{}, "TRADE", "", "", "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCWalletBalance(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err := b.GetUSDCWalletBalance(context.Background())
// 	if err != nil && err.Error() != "System error. Please try again later." {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCAssetInfo(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err := b.GetUSDCAssetInfo(context.Background(), "")
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = b.GetUSDCAssetInfo(context.Background(), "BTC")
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCMarginInfo(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err := b.GetUSDCMarginInfo(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCPositions(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCPosition(context.Background(), pair, "PERPETUAL", "", "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestSetUSDCLeverage(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.SetUSDCLeverage(context.Background(), pair, 2)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCSettlementHistory(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCSettlementHistory(context.Background(), pair, "", "", 0)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCRiskLimit(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCRiskLimit(context.Background(), pair)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestSetUSDCRiskLimit(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.SetUSDCRiskLimit(context.Background(), pair, 2)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCLastFundingRate(t *testing.T) {
// 	t.Parallel()
// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = b.GetUSDCLastFundingRate(context.Background(), pair)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetUSDCPredictedFundingRate(t *testing.T) {
// 	t.Parallel()
// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	pair, err := currency.NewPairFromString("BTCPERP")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, _, err = b.GetUSDCPredictedFundingRate(context.Background(), pair)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := b.UpdateTickers(ctx, asset.Spot)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Spot, err)
	}
	err = b.UpdateTickers(ctx, asset.Linear)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Linear, err)
	}
	err = b.UpdateTickers(ctx, asset.Inverse)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Inverse, err)
	}
	err = b.UpdateTickers(ctx, asset.Options)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Options, err)
	}
}

func TestGetTickersV5(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickers(context.Background(), "bruh", "", "", time.Time{})
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetTickers(context.Background(), "option", "BTCUSDT", "", time.Time{})
	if !errors.Is(err, errBaseNotSet) {
		t.Fatalf("expected: %v, received: %v", errBaseNotSet, err)
	}
	_, err = b.GetTickers(context.Background(), "spot", "", "", time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTickers(context.Background(), "option", "", "BTC", time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTickers(context.Background(), "inverse", "", "", time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTickers(context.Background(), "linear", "", "", time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingRateHistory(context.Background(), "bruh", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetFundingRateHistory(context.Background(), "spot", "BTCUSDT", time.Time{}, time.Time{}, 100)
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetFundingRateHistory(context.Background(), "linear", "BTCUSDT", time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRateHistory(context.Background(), "inverse", "BTCUSDT", time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRateHistory(context.Background(), "option", "BTC-7JUL23-27000-C", time.Time{}, time.Time{}, 100)
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
}

func TestGetPublicTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetPublicTradingHistory(context.Background(), "spot", "BTCUSDT", "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(context.Background(), "linear", "BTCUSDT", "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(context.Background(), "inverse", "BTCUSDM23", "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(context.Background(), "option", "BTC-7JUL23-27000-C", "BTC", "", 30)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterest(context.Background(), "spot", "BTCUSDT", "5min", time.Time{}, time.Time{}, 0, "")
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetOpenInterest(context.Background(), "linear", "BTCUSDT", "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterest(context.Background(), "inverse", "BTCUSDM23", "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterest(context.Background(), "option", "BTC-7JUL23-27000-C", "5min", time.Time{}, time.Time{}, 0, "")
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
}

func TestGetHistoricalValatility(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalValatility(context.Background(), "option", "", 123, time.Now().Add(-time.Hour*30*24), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricalValatility(context.Background(), "spot", "", 123, time.Now().Add(-time.Hour*30*24), time.Now())
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, but found %v", errInvalidCategory, err)
	}
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	_, err := b.GetInsurance(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetDeliveryPrice(context.Background(), "spot", "BTCUSDT", "", "", 200)
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, but found %v", errInvalidCategory, err)
	}
	_, err = b.GetDeliveryPrice(context.Background(), "linear", "BTCUSDT", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetDeliveryPrice(context.Background(), "inverse", "BTCUSDT", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetDeliveryPrice(context.Background(), "option", "BTC-7JUL23-27000-C", "", "", 200)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(context.Background(), asset.USDCMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v expected: %v", err, asset.ErrNotSupported)
	}
	err = b.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	if err != nil {
		t.Error("Bybit UpdateOrderExecutionLimits() error", err)
	}
	avail, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal("Bybit GetAvailablePairs() error", err)
	}
	for x := range avail {
		limits, err := b.GetOrderExecutionLimits(asset.Spot, avail[x])
		if err != nil {
			t.Fatal("Bybit GetOrderExecutionLimits() error", err)
		}
		if limits == (order.MinMaxLevel{}) {
			t.Fatal("Bybit GetOrderExecutionLimits() error cannot be nil")
		}
	}
}

// func TestGetFeeRate(t *testing.T) {
// 	t.Parallel()

// 	_, err := b.GetFeeRate(context.Background(), "", "", "")
// 	if !errors.Is(err, errCategoryNotSet) {
// 		t.Fatalf("received %v but expected %v", err, errCategoryNotSet)
// 	}

// 	_, err = b.GetFeeRate(context.Background(), "bruh", "", "")
// 	if !errors.Is(err, errInvalidCategory) {
// 		t.Fatalf("received %v but expected %v", err, errInvalidCategory)
// 	}

// 	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

// 	_, err = b.GetFeeRate(context.Background(), "spot", "", "")
// 	if !errors.Is(err, nil) {
// 		t.Errorf("received %v but expected %v", err, nil)
// 	}

// 	_, err = b.GetFeeRate(context.Background(), "linear", "", "")
// 	if !errors.Is(err, nil) {
// 		t.Errorf("received %v but expected %v", err, nil)
// 	}

// 	_, err = b.GetFeeRate(context.Background(), "inverse", "", "")
// 	if !errors.Is(err, nil) {
// 		t.Errorf("received %v but expected %v", err, nil)
// 	}

// 	_, err = b.GetFeeRate(context.Background(), "option", "", "ETH")
// 	if !errors.Is(err, nil) {
// 		t.Errorf("received %v but expected %v", err, nil)
// 	}
// }

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	ctx := context.Background()
	_, err := b.PlaceOrder(ctx, nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "my-category",
	})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
	})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
		Symbol:   currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Fatalf("expected %v, got %v", order.ErrSideIsInvalid, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
		Symbol:   currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
		Side:     "buy",
	})
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Fatalf("expected %v, got %v", order.ErrTypeIsInvalid, err)
	}
	// order.ErrAmountBelowMin
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category:  "spot",
		Symbol:    currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
		Side:      "buy",
		OrderType: "limit",
	})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Fatalf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category:         "spot",
		Symbol:           currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
		Side:             "buy",
		OrderType:        "limit",
		OrderQuantity:    1,
		TriggerDirection: 3,
	})
	if !errors.Is(err, errInvalidTriggerDirection) {
		t.Fatalf("expected %v, got %v", errInvalidTriggerDirection, err)
	}
	_, err = b.PlaceOrder(context.Background(), &PlaceOrderParams{
		Category:         "spot",
		Symbol:           currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
		Side:             "buy",
		OrderType:        "limit",
		OrderQuantity:    1,
		Price:            31431.48,
		TriggerDirection: 2,
	})
	if err != nil {
		t.Error(err)
	}

	// Spot post only normal order
	arg := &PlaceOrderParams{Category: "spot", Symbol: currency.Pair{Base: currency.BTC, Quote: currency.USDT}, Side: "Buy", OrderType: "Limit", OrderQuantity: 0.1, Price: 15600, TimeInForce: "PostOnly", OrderLinkID: "spot-test-01", IsLeverage: 0, OrderFilter: "Order"}
	_, err = b.PlaceOrder(context.Background(), arg)
	if err != nil {
		t.Error(err)
	}

	// // Spot TP/SL order
	arg = &PlaceOrderParams{Category: "spot", Symbol: currency.Pair{Base: currency.BTC, Quote: currency.USDT},
		Side: "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TriggerPrice: 15000,
		TimeInForce: "GTC", OrderLinkID: "spot-test-02", IsLeverage: 0, OrderFilter: "tpslOrder"}
	_, err = b.PlaceOrder(context.Background(), arg)
	if err != nil {
		t.Error(err)
	}

	// Spot margin normal order (UTA)
	arg = &PlaceOrderParams{Category: "spot", Symbol: currency.Pair{Base: currency.BTC, Quote: currency.USDT}, Side: "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TimeInForce: "IOC", OrderLinkID: "spot-test-limit", IsLeverage: 1, OrderFilter: "Order"}
	_, err = b.PlaceOrder(context.Background(), arg)
	if err != nil {
		t.Error(err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.AmendOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.AmendOrder(context.Background(), &AmendOrderParams{})
	if !errors.Is(err, errEitherOrderIDOROrderLinkIDRequired) {
		t.Fatalf("expected %v, got %v", errEitherOrderIDOROrderLinkIDRequired, err)
	}
	_, err = b.AmendOrder(context.Background(), &AmendOrderParams{
		OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa",
	})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.AmendOrder(context.Background(), &AmendOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "mycat"})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.AmendOrder(context.Background(), &AmendOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option"})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	_, err = b.AmendOrder(context.Background(), &AmendOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "spot", Symbol: currency.Pair{Base: currency.BTC, Quote: currency.USD},
		TriggerPrice:    1145,
		OrderQuantity:   0.15,
		Price:           1050,
		TakeProfitPrice: 0,
		StopLossPrice:   0})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelTradeOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CancelTradeOrder(context.Background(), &CancelOrderParams{})
	if !errors.Is(err, errEitherOrderIDOROrderLinkIDRequired) {
		t.Fatalf("expected %v, got %v", errEitherOrderIDOROrderLinkIDRequired, err)
	}
	_, err = b.CancelTradeOrder(context.Background(), &CancelOrderParams{
		OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa",
	})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.CancelTradeOrder(context.Background(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "mycat"})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.CancelTradeOrder(context.Background(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option"})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyPairEmpty, err)
	}
	cp, err := currency.NewPairFromString("BTC-7JUL23-27000-C")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CancelTradeOrder(context.Background(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option",
		Symbol:   cp,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOpenOrders(context.Background(), "", "", "", "", "", "", "", "", 0, 100)
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.GetOpenOrders(context.Background(), "spot", "", "", "", "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelAllTradeOrders(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CancelAllTradeOrders(context.Background(), &CancelAllOrdersParam{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.CancelAllTradeOrders(context.Background(), &CancelAllOrdersParam{Category: "option"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTradeOrderHistory(context.Background(), "", "", "", "", "", "", "", "", "", time.Now().Add(-time.Hour*24*20), time.Now(), 100)
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.GetTradeOrderHistory(context.Background(), "spot", "BTCUSDT", "", "", "BTC", "", "StopOrder", "", "", time.Now().Add(-time.Hour*24*20), time.Now(), 100)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.PlaceBatchOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.PlaceBatchOrder(context.Background(), &PlaceBatchOrderParam{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.PlaceBatchOrder(context.Background(), &PlaceBatchOrderParam{
		Category: "spot",
	})
	if !errors.Is(err, errNoOrderPassed) {
		t.Fatalf("expected %v, got %v", errNoOrderPassed, err)
	}
	cp, err := currency.NewPairFromString("BTC-7JUL23-27000-C")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.PlaceBatchOrder(context.Background(), &PlaceBatchOrderParam{
		Category: "option",
		Request: []BatchOrderItemParam{
			{
				Symbol:        cp,
				OrderType:     "Limit",
				Side:          "Buy",
				OrderQuantity: 1,
				OrderIv:       6,
				TimeInForce:   "GTC",
				OrderLinkID:   "option-test-001",
				Mmp:           false,
				ReduceOnly:    false,
			},
			{
				Symbol:        cp,
				OrderType:     "Limit",
				Side:          "Sell",
				OrderQuantity: 2,
				Price:         700,
				TimeInForce:   "GTC",
				OrderLinkID:   "option-test-001",
				Mmp:           false,
				ReduceOnly:    false,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.BatchAmendOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.BatchAmendOrder(context.Background(), &BatchAmendOrderParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.BatchAmendOrder(context.Background(), &BatchAmendOrderParams{Category: "spot"})
	if !errors.Is(err, errNoOrderPassed) {
		t.Fatalf("expected %v, got %v", errNoOrderPassed, err)
	}
	cp, err := currency.NewPairFromString("BTC-7JUL23-27000-C")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.BatchAmendOrder(context.Background(), &BatchAmendOrderParams{
		Category: "option",
		Request: []BatchAmendOrderParamItem{
			{
				Symbol:                 cp,
				OrderImpliedVolatility: "6.8",
				OrderID:                "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
			},
			{
				Symbol:  cp,
				Price:   650,
				OrderID: "fa6a595f-1a57-483f-b9d3-30e9c8235a52",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelBatchOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CancelBatchOrder(context.Background(), &CancelBatchOrder{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.CancelBatchOrder(context.Background(), &CancelBatchOrder{Category: "spot"})
	if !errors.Is(err, errNoOrderPassed) {
		t.Fatalf("expected %v, got %v", errNoOrderPassed, err)
	}
	cp, err := currency.NewPairFromString("BTC-7JUL23-27000-C")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CancelBatchOrder(context.Background(), &CancelBatchOrder{
		Category: "option",
		Request: []CancelOrderParams{
			{
				Symbol:  cp,
				OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
			},
			{
				Symbol:  cp,
				OrderID: "fa6a595f-1a57-483f-b9d3-30e9c8235a52",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetBorrowQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBorrowQuota(context.Background(), "", "BTCUSDT", "Buy")
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.GetBorrowQuota(context.Background(), "spot", "", "Buy")
	if !errors.Is(err, errSymbolMissing) {
		t.Fatalf("expected %v, got %v", errSymbolMissing, err)
	}
	_, err = b.GetBorrowQuota(context.Background(), "spot", "BTCUSDT", "")
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Error(err)
	}
	_, err = b.GetBorrowQuota(context.Background(), "spot", "BTCUSDT", "Buy")
	if err != nil {
		t.Error(err)
	}
}

func TestSetDisconnectCancelAll(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetDisconnectCancelAll(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	err = b.SetDisconnectCancelAll(context.Background(), &SetDCPParams{TimeWindow: 300})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPositionInfo(context.Background(), "", "", "", "", "", 20)
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.GetPositionInfo(context.Background(), "spot", "", "", "", "", 20)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetPositionInfo(context.Background(), "linear", "BTCUSDT", "", "", "", 20)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPositionInfo(context.Background(), "option", "BTC-7JUL23-27000-C", "BTC", "", "", 20)
	if err != nil {
		t.Error(err)
	}
}
func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetLeverage(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	err = b.SetLeverage(context.Background(), &SetLeverageParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	err = b.SetLeverage(context.Background(), &SetLeverageParams{Category: "spot"})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	err = b.SetLeverage(context.Background(), &SetLeverageParams{Category: "linear"})
	if !errors.Is(err, errSymbolMissing) {
		t.Fatalf("expected %v, got %v", errSymbolMissing, err)
	}
	err = b.SetLeverage(context.Background(), &SetLeverageParams{Category: "linear", Symbol: "BTCUSDT"})
	if !errors.Is(err, errInvalidLeverage) {
		t.Fatalf("expected %v, got %v", errInvalidLeverage, err)
	}
	err = b.SetLeverage(context.Background(), &SetLeverageParams{Category: "linear", Symbol: "BTCUSDT", SellLeverage: 3, BuyLeverage: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchTradeMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SwitchTradeMode(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{Category: "spot"})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{Category: "linear"})
	if !errors.Is(err, errSymbolMissing) {
		t.Fatalf("expected %v, got %v", errSymbolMissing, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{Category: "linear", Symbol: "BTCUSDT"})
	if !errors.Is(err, errInvalidLeverage) {
		t.Fatalf("expected %v, got %v", errInvalidLeverage, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{Category: "linear", Symbol: "BTCUSDT", SellLeverage: 3, BuyLeverage: 3, TradeMode: 2})
	if !errors.Is(err, errInvalidTradeModeValue) {
		t.Fatalf("expected %v, got %v", errInvalidTradeModeValue, err)
	}
	err = b.SwitchTradeMode(context.Background(), &SwitchTradeModeParams{Category: "linear", Symbol: "BTCUSDT", SellLeverage: 3, BuyLeverage: 3, TradeMode: 1})
	if err != nil {
		t.Error(err)
	}
}

func TestSetTakeProfitStopLossMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetTakeProfitStopLossMode(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{
		Category: "spot",
	})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{Category: "spot"})
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{Category: "linear"})
	if !errors.Is(err, errSymbolMissing) {
		t.Fatalf("expected %v, got %v", errSymbolMissing, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{Category: "linear", Symbol: "BTCUSDT"})
	if !errors.Is(err, errTakeProfitOrStopLossModeMissing) {
		t.Fatalf("expected %v, got %v", errTakeProfitOrStopLossModeMissing, err)
	}
	_, err = b.SetTakeProfitStopLossMode(context.Background(), &TPSLModeParams{Category: "linear", Symbol: "BTCUSDT", TpslMode: "Partial"})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SwitchPositionMode(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	err = b.SwitchPositionMode(context.Background(), &SwitchPositionModeParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	err = b.SwitchPositionMode(context.Background(), &SwitchPositionModeParams{Category: "linear"})
	if !errors.Is(err, errEitherSymbolOrCoinRequired) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	cp, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	err = b.SwitchPositionMode(context.Background(), &SwitchPositionModeParams{Category: "linear", Symbol: cp, PositionMode: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSetRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetRiskLimit(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.SetRiskLimit(context.Background(), &SetRiskLimitParam{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Errorf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.SetRiskLimit(context.Background(), &SetRiskLimitParam{Category: "linear", PositionMode: -2})
	if !errors.Is(err, errInvalidPositionMode) {
		t.Errorf("expected %v, got %v", errInvalidPositionMode, err)
	}
	_, err = b.SetRiskLimit(context.Background(), &SetRiskLimitParam{Category: "linear"})
	if !errors.Is(err, errSymbolMissing) {
		t.Errorf("expected %v, got %v", errSymbolMissing, err)
	}
	cp, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.SetRiskLimit(context.Background(), &SetRiskLimitParam{
		Category:     "linear",
		RiskID:       1234,
		Symbol:       cp,
		PositionMode: 0,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetTradingStop(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetTradingStop(context.Background(), &TradingStopParams{})
	if !errors.Is(err, errCategoryNotSet) {
		t.Errorf("expected %v, got %v", errCategoryNotSet, err)
	}
	err = b.SetTradingStop(context.Background(), &TradingStopParams{Category: "spot"})
	if !errors.Is(err, errInvalidCategory) {
		t.Errorf("expected %v, got %v", errInvalidCategory, err)
	}
	err = b.SetTradingStop(context.Background(), &TradingStopParams{
		Category:                 "linear",
		Symbol:                   currency.NewPair(currency.XRP, currency.USDT),
		TakeProfit:               "0.5",
		StopLoss:                 "0.2",
		TakeProfitTriggerType:    "MarkPrice",
		StopLossTriggerType:      "IndexPrice",
		TakeProfitOrStopLossMode: "Partial",
		TakeProfitOrderType:      "Limit",
		StopLossOrderType:        "Limit",
		TakeProfitSize:           50,
		StopLossSize:             50,
		TakeProfitLimitPrice:     0.49,
		StopLossLimitPrice:       0.21,
		PositionIndex:            0,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetAutoAddMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetAutoAddMargin(context.Background(), &AddRemoveMarginParams{
		Category:      "inverse",
		Symbol:        pair,
		AutoAddmargin: 0,
		PositionMode:  2,
	})
	if err != nil {
		t.Error(err)
	}
}
func TestAddOrReduceMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.AddOrReduceMargin(context.Background(), &AddRemoveMarginParams{
		Category:      "inverse",
		Symbol:        pair,
		AutoAddmargin: 0,
		PositionMode:  2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetExecution(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetExecution(context.Background(), "spot", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetClosedPnL(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetClosedPnL(context.Background(), "spot", "", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", err, errInvalidCategory)
	}
	_, err = b.GetClosedPnL(context.Background(), "linear", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPreUpgradeOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeOrderHistory(context.Background(), "", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 100)
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("expected %v, got %v", errCategoryNotSet, err)
	}
	_, err = b.GetPreUpgradeOrderHistory(context.Background(), "option", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errBaseNotSet) {
		t.Fatalf("expected %v, got %v", errBaseNotSet, err)
	}
	_, err = b.GetPreUpgradeOrderHistory(context.Background(), "linear", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeTradeHistory(context.Background(), "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errCategoryNotSet) {
		t.Fatalf("found %v, expected %v", err, errCategoryNotSet)
	}
	_, err = b.GetPreUpgradeTradeHistory(context.Background(), "option", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("found %v, expected %v", err, errInvalidCategory)
	}
	_, err = b.GetPreUpgradeTradeHistory(context.Background(), "linear", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeClosedPnL(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeClosedPnL(context.Background(), "option", "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetPreUpgradeClosedPnL(context.Background(), "linear", "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTransactionLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeTransactionLog(context.Background(), "option", "", "", "", time.Time{}, time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("found %v, expected %v", err, errInvalidCategory)
	}
	_, err = b.GetPreUpgradeTransactionLog(context.Background(), "linear", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeOptionDeliveryRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeOptionDeliveryRecord(context.Background(), "linear", "", "", time.Time{}, 0)
	if !errors.Is(err, errInvalidCategory) {
		t.Error(err)
	}
	_, err = b.GetPreUpgradeOptionDeliveryRecord(context.Background(), "option", "", "", time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeUSDCSessionSettlement(context.Background(), "option", "", "", 10)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetPreUpgradeUSDCSessionSettlement(context.Background(), "linear", "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	result, err := b.GetWalletBalance(context.Background(), "UNIFIED", "")
	if err != nil {
		t.Fatal(err)
	} else {
		val, _ := json.Marshal(result)
		println(string(val))
	}
}

func TestUpgradeToUnifiedAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UpgradeToUnifiedAccount(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBorrowHistory(context.Background(), "BTC", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetCollateralInfo(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetCoinGreeks(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetFeeRate(context.Background(), "something", "", "BTC")
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetFeeRate(context.Background(), "linear", "", "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTransactionLog(context.Background(), "option", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTransactionLog(context.Background(), "linear", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMarginMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetMarginMode(context.Background(), "PORTFOLIO_MARGIN")
	if err != nil {
		t.Error(err)
	}
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetMMP(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("found %v, expected %v", err, errNilArgument)
	}
	b.Verbose = true
	err = b.SetMMP(context.Background(), &MMPRequestParam{
		BaseCoin:           "ETH",
		TimeWindowMS:       5000,
		FrozenPeriod:       100000,
		TradeQuantityLimit: 50,
		DeltaLimit:         20,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.ResetMMP(context.Background(), "USDT")
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("found %v, expected %v", err, errNilArgument)
	}
	err = b.ResetMMP(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMMPState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	b.Verbose = true
	_, err := b.GetMMPState(context.Background(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinExchangeRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetCoinExchangeRecords(context.Background(), "", "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDeliveryRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetDeliveryRecord(context.Background(), "spot", "", "", time.Now().Add(time.Hour*40), 20)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatal(err)
	}
	_, err = b.GetDeliveryRecord(context.Background(), "linear", "", "", time.Now().Add(time.Hour*40), 20)
	if err != nil {
		t.Error(err)
	}
}
func TestGetUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUSDCSessionSettlement(context.Background(), "option", "", "", 10)
	if !errors.Is(err, errInvalidCategory) {
		t.Fatalf("expected %v, got %v", errInvalidCategory, err)
	}
	_, err = b.GetUSDCSessionSettlement(context.Background(), "linear", "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAssetInfo(context.Background(), "", "BTC")
	if !errors.Is(err, errMissingAccountType) {
		t.Fatal(err)
	}
	_, err = b.GetAssetInfo(context.Background(), "SPOT", "BTC")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetAllCoinBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAllCoinBalance(context.Background(), "", "", "", 0)
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.GetAllCoinBalance(context.Background(), "SPOT", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSingleCoinBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSingleCoinBalance(context.Background(), "", "", "", 0, 0)
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.GetSingleCoinBalance(context.Background(), "SPOT", currency.BTC.String(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTransferableCoin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTransferableCoin(context.Background(), "SPOT", "OPTION")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateInternalTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CreateInternalTransfer(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{})
	if !errors.Is(err, errMissingTransferID) {
		t.Fatalf("expected %v, got %v", errMissingTransferID, err)
	}
	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{TransferID: transferID})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	if !errors.Is(err, order.ErrAmountIsInvalid) {
		t.Fatalf("expected %v, got %v", order.ErrAmountIsInvalid, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED"})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateInternalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetInternalTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetInternalTransferRecords(context.Background(), transferID.String(), currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubUID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubUID(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestEnableUniversalTransferForSubUID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.EnableUniversalTransferForSubUID(context.Background())
	if !errors.Is(err, errMembersIDsNotSet) {
		t.Fatalf("expected %v, got %v", errMembersIDsNotSet, err)
	}
	transferID1, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	transferID2, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	err = b.EnableUniversalTransferForSubUID(context.Background(), transferID1.String(), transferID2.String())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.CreateUniversalTransfer(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{})
	if !errors.Is(err, errMissingTransferID) {
		t.Fatalf("expected %v, got %v", errMissingTransferID, err)
	}
	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{TransferID: transferID})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	if !errors.Is(err, order.ErrAmountIsInvalid) {
		t.Fatalf("expected %v, got %v", order.ErrAmountIsInvalid, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED"})
	if !errors.Is(err, errMissingAccountType) {
		t.Fatalf("expected %v, got %v", errMissingAccountType, err)
	}
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{TransferID: transferID,
		Coin: currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED"})
	if !errors.Is(err, errMemberIDRequired) {
		t.Fatalf("expected %v, got %v", errMemberIDRequired, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateUniversalTransfer(context.Background(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED",
		FromMemberID:    123,
		ToMemberID:      456,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetUniversalTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetUniversalTransferRecords(context.Background(), transferID.String(), currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllowedDepositCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAllowedDepositCoinInfo(context.Background(), "BTC", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetDepositAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.SetDepositAccount(context.Background(), "SPOT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetDepositRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubDepositRecords(context.Background(), "12345", "", "nextPageCursor", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestInternalDepositRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInternalDepositRecordsOffChain(context.Background(), currency.ETH.String(), "", time.Time{}, time.Time{}, 8)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMasterDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetMasterDepositAddress(context.Background(), currency.LTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubDepositAddress(context.Background(), currency.LTC, "LTC", "12345")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetCoinInfo(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawalRecords(context.Background(), currency.LTC, "", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawableAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawableAmount(context.Background(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.WithdrawCurrency(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.WithdrawCurrency(context.Background(), &WithdrawalParam{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.WithdrawCurrency(context.Background(), &WithdrawalParam{Coin: currency.BTC})
	if !errors.Is(err, errMissingChainInformation) {
		t.Fatalf("expected %v, got %v", errMissingChainInformation, err)
	}
	_, err = b.WithdrawCurrency(context.Background(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC"})
	if !errors.Is(err, errMissingAddressInfo) {
		t.Fatalf("expected %v, got %v", errMissingAddressInfo, err)
	}
	_, err = b.WithdrawCurrency(context.Background(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234"})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Fatalf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.WithdrawCurrency(context.Background(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234", Amount: 123})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelWithdrawal(context.Background(), "")
	if !errors.Is(err, errMissingWithdrawalID) {
		t.Fatalf("expected %v, got %v", errMissingWithdrawalID, err)
	}
	_, err = b.CancelWithdrawal(context.Background(), "12314")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewSubUserID(t *testing.T) {
	t.Parallel()
	_, err := b.CreateNewSubUserID(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CreateNewSubUserID(context.Background(), &CreateSubUserParams{MemberType: 1, Switch: 1, Note: "test"})
	if !errors.Is(err, errMissingusername) {
		t.Fatalf("expected %v, got %v", errMissingusername, err)
	}
	_, err = b.CreateNewSubUserID(context.Background(), &CreateSubUserParams{Username: "Sami", Switch: 1, Note: "test"})
	if !errors.Is(err, errInvalidMemberType) {
		t.Fatalf("expected %v, got %v", errInvalidMemberType, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateNewSubUserID(context.Background(), &CreateSubUserParams{Username: "sami", MemberType: 1, Switch: 1, Note: "test"})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubUIDAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.CreateSubUIDAPIKey(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.CreateSubUIDAPIKey(context.Background(), &SubUIDAPIKeyParam{})
	if !errors.Is(err, errMissingUserID) {
		t.Fatalf("expected %v, got %v", errMissingUserID, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateSubUIDAPIKey(context.Background(), &SubUIDAPIKeyParam{
		Subuid:      53888000,
		Note:        "testxxx",
		ReadOnly:    0,
		Permissions: map[string][]string{"Wallet": {"AccountTransfer"}},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubUIDList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubUIDList(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestFreezeSubUID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.FreezeSubUID(context.Background(), "1234", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAPIKeyInformation(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUIDWalletType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUIDWalletType(context.Background(), "234234")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyMasterAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifyMasterAPIKey(context.Background(), &SubUIDAPIKeyUpdateParam{})
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.ModifyMasterAPIKey(context.Background(), &SubUIDAPIKeyUpdateParam{
		ReadOnly: 0,
		IPs:      []string{"*"},
		Permissions: map[string][]string{
			"ContractTrade": {"Order", "Position"},
			"Spot":          {"SpotTrade"},
			"Wallet":        {"AccountTransfer", "SubMemberTransfer"},
			"Options":       {"OptionsTrade"},
			"Derivatives":   {"DerivativesTrade"},
			"CopyTrading":   {"CopyTrading"},
			"BlockTrade":    {},
			"Exchange":      {"ExchangeHistory"},
			"NFT":           {"NFTQueryProductList"}},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestModifySubAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifySubAPIKey(context.Background(), &SubUIDAPIKeyUpdateParam{})
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.ModifySubAPIKey(context.Background(), &SubUIDAPIKeyUpdateParam{
		ReadOnly: 0,
		IPs:      []string{"*"},
		Permissions: map[string][]string{
			"ContractTrade": {},
			"Spot":          {"SpotTrade"},
			"Wallet":        {"AccountTransfer"},
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMasterAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DeleteMasterAPIKey(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteSubAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DeleteSubAccountAPIKey(context.Background(), "12434")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateUserInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAffiliateUserInfo(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeverageTokenInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLeverageTokenInfo(context.Background(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeveragedTokenMarket(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLeveragedTokenMarket(context.Background(), currency.EMPTYCODE)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.GetLeveragedTokenMarket(context.Background(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestPurchaseLeverageToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.PurchaseLeverageToken(context.Background(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestRedeemLeverageToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.RedeemLeverageToken(context.Background(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPurchaseAndRedemptionRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPurchaseAndRedemptionRecords(context.Background(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestToggleMarginTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ToggleMarginTrade(context.Background(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestSetSpotMarginTradeLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetSpotMarginTradeLeverage(context.Background(), 3)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
}

func TestGetMarginCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetMarginCoinInfo(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowableCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBorrowableCoinInfo(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInterestAndQuota(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInterestAndQuota(context.Background(), currency.EMPTYCODE)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.GetInterestAndQuota(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLoanAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLoanAccountInfo(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestBorrow(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.Borrow(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.Borrow(context.Background(), &LendArgument{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.Borrow(context.Background(), &LendArgument{Coin: currency.BTC})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.Borrow(context.Background(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}
func TestRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.Repay(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, got %v", errNilArgument, err)
	}
	_, err = b.Repay(context.Background(), &LendArgument{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.Repay(context.Background(), &LendArgument{Coin: currency.BTC})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.Repay(context.Background(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBorrowOrderDetail(context.Background(), time.Time{}, time.Time{}, currency.BTC, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRepaymentOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetRepaymentOrderDetail(context.Background(), time.Time{}, time.Time{}, currency.BTC, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestToggleMarginTradeNormal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.ToggleMarginTradeNormal(context.Background(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetProductInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetProductInfo(context.Background(), "78")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLengingMarginCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInstitutionalLengingMarginCoinInfo(context.Background(), "123")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInstitutionalLoanOrders(context.Background(), "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalRepayOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInstitutionalRepayOrders(context.Background(), time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLTV(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLTV(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingCoinInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingCoinInfo(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestC2CDepositFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.C2CDepositFunds(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Error(err)
	}
	_, err = b.C2CDepositFunds(context.Background(), &C2CLendingFundsParams{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.C2CDepositFunds(context.Background(), &C2CLendingFundsParams{Coin: currency.BTC})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.C2CDepositFunds(context.Background(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestC2CRedeemFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.C2CRedeemFunds(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Error(err)
	}
	_, err = b.C2CRedeemFunds(context.Background(), &C2CLendingFundsParams{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, got %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = b.C2CRedeemFunds(context.Background(), &C2CLendingFundsParams{Coin: currency.BTC})
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Errorf("expected %v, got %v", order.ErrAmountBelowMin, err)
	}
	_, err = b.C2CRedeemFunds(context.Background(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingOrderRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingOrderRecords(context.Background(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingAccountInfo(context.Background(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBrokerEarning(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBrokerEarning(context.Background(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func instantiateTradablePairs() error {
	err := b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		return err
	}
	tradables, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	spotTradablePair = tradables[0]
	tradables, err = b.GetEnabledPairs(asset.Linear)
	if err != nil {
		return err
	}
	linearTradablePair = tradables[0]
	tradables, err = b.GetEnabledPairs(asset.Inverse)
	if err != nil {
		return err
	}
	inverseTradablePair = tradables[0]
	tradables, err = b.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	optionsTradablePair = tradables[0]
	return nil
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.CoinMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, got %v", asset.ErrNotSupported, err)
	}
	results, err := b.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error("GetWithdrawalsHistory()", err)
	} else {
		val, _ := json.Marshal(results)
		println(string(val))
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair, asset.Options)
	if err != nil {
		t.Error(err)
	}

	pair1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair1, asset.Inverse)
	if err != nil {
		t.Error(err)
	}

	pair2, err := currency.NewPairFromString("BTCPERP")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetRecentTrades(context.Background(), pair2, asset.Linear)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRecentTrades(context.Background(), pair1, asset.CoinMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestGetBybitServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetBybitServerTime(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(context.Background(), asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricTrades(context.Background(), spotTradablePair, asset.Spot, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(context.Background(), linearTradablePair, asset.Linear, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(context.Background(), inverseTradablePair, asset.Inverse, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(context.Background(), optionsTradablePair, asset.Options, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	var orderCancellationParams = []order.Cancel{{
		OrderID:   "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot}, {
		OrderID:   "1",
		Pair:      linearTradablePair,
		AssetType: asset.Linear}}
	_, err := b.CancelBatchOrders(context.Background(), orderCancellationParams)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, got %v", asset.ErrNotSupported, err)
	}
	orderCancellationParams = []order.Cancel{{
		OrderID:   "1",
		AccountID: "1",
		Pair:      optionsTradablePair,
		AssetType: asset.Options}, {
		OrderID:   "2",
		Pair:      optionsTradablePair,
		AssetType: asset.Options}}
	b.Verbose = true
	_, err = b.CancelBatchOrders(context.Background(), orderCancellationParams)
	if err != nil {
		t.Error(err)
	}
}

// func TestForceFileStandard(t *testing.T) {
// 	t.Parallel()
// 	err := sharedtestvalues.ForceFileStandard(t, sharedtestvalues.EmptyStringPotentialPattern)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if t.Failed() {
// 		t.Fatal("Please use convert.StringToFloat64 type instead of `float64` and remove `,string` as strings can be empty in unmarshal process. Then call the Float64() method.")
// 	}
// }

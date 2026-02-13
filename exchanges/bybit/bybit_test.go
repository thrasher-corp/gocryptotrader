package bybit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false

	skipAuthenticatedFunctionsForMockTesting = "skipping authenticated function for mock testing"
)

var (
	e *Exchange

	spotTradablePair, usdcMarginedTradablePair, usdtMarginedTradablePair, inverseTradablePair, optionsTradablePair currency.Pair
)

func TestGetInstrumentInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentInfo(t.Context(), cSpot, "", "", "", "", 0)
	require.NoError(t, err)
	_, err = e.GetInstrumentInfo(t.Context(), cLinear, "", "", "", "", 0)
	require.NoError(t, err)
	_, err = e.GetInstrumentInfo(t.Context(), cInverse, "", "", "", "", 0)
	require.NoError(t, err)
	_, err = e.GetInstrumentInfo(t.Context(), cOption, "", "", "", "", 0)
	require.NoError(t, err)
	payload, err := e.GetInstrumentInfo(t.Context(), cLinear, "10000000AIDOGEUSDT", "", "", "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, payload.List)
	require.NotZero(t, payload.List[0].LotSizeFilter.MinNotionalValue)
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()
	if mockTests {
		startTime = time.Unix(1691897100, 0).Round(kline.FiveMin.Duration())
		endTime = time.Unix(1691907100, 0).Round(kline.FiveMin.Duration())
	}
	for _, tc := range []struct {
		category   string
		pair       currency.Pair
		reqLimit   uint64
		expRespLen int
		expError   error
	}{
		{cSpot, spotTradablePair, 100, 34, nil}, // TODO: Update expected limit when mock data is updated
		{cLinear, usdtMarginedTradablePair, 5, 5, nil},
		{cLinear, usdcMarginedTradablePair, 5, 5, nil},
		{cInverse, inverseTradablePair, 5, 5, nil},
		{cOption, optionsTradablePair, 5, 5, errInvalidCategory},
	} {
		t.Run(fmt.Sprintf("%s-%s", tc.category, tc.pair), func(t *testing.T) {
			t.Parallel()
			r, err := e.GetKlines(t.Context(), tc.category, tc.pair.String(), kline.FiveMin, startTime, endTime, tc.reqLimit)
			if tc.expError != nil {
				require.ErrorIs(t, err, tc.expError)
				return
			}
			require.NoError(t, err)
			if mockTests {
				require.Equal(t, tc.expRespLen, len(r))

				switch tc.category {
				case cSpot:
					assert.Equal(t, KlineItem{StartTime: types.Time(endTime), Open: 29393.99, High: 29399.76, Low: 29393.98, Close: 29399.76, TradeVolume: 1.168988, Turnover: 34363.5346739}, r[0])
				case cLinear:
					if tc.pair == usdtMarginedTradablePair {
						assert.Equal(t, KlineItem{StartTime: types.Time(endTime), Open: 0.0003, High: 0.0003, Low: 0.0002995, Close: 0.0003, TradeVolume: 55102100, Turnover: 16506.2427}, r[0])
						return
					}
					assert.Equal(t, KlineItem{StartTime: types.Time(endTime), Open: 239.7, High: 239.7, Low: 239.7, Close: 239.7}, r[0])
				case cInverse:
					assert.Equal(t, KlineItem{StartTime: types.Time(endTime), Open: 0.2908, High: 0.2912, Low: 0.2908, Close: 0.2912, TradeVolume: 5131, Turnover: 17626.40000346}, r[0])
				}
			} else {
				assert.NotEmpty(t, r)
			}
		})
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 1)
	endTime := time.Now()
	if mockTests {
		startTime = time.UnixMilli(1693077167971)
		endTime = time.UnixMilli(1693080767971)
	}
	_, err := e.GetMarkPriceKline(t.Context(), cLinear, usdtMarginedTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetMarkPriceKline(t.Context(), cLinear, usdcMarginedTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetMarkPriceKline(t.Context(), cInverse, inverseTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetMarkPriceKline(t.Context(), cOption, optionsTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err == nil {
		t.Fatalf("expected 'params error: Category is invalid', but found nil")
	}
}

func TestGetIndexPriceKline(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 1)
	endTime := time.Now()
	if mockTests {
		startTime = time.UnixMilli(1693077165571)
		endTime = time.UnixMilli(1693080765571)
	}
	_, err := e.GetIndexPriceKline(t.Context(), cLinear, usdtMarginedTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetIndexPriceKline(t.Context(), cLinear, usdcMarginedTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetIndexPriceKline(t.Context(), cInverse, inverseTradablePair.String(), kline.FiveMin, startTime, endTime, 5)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), cSpot, spotTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetOrderBook(t.Context(), cLinear, usdtMarginedTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetOrderBook(t.Context(), cLinear, usdcMarginedTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetOrderBook(t.Context(), cInverse, inverseTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetOrderBook(t.Context(), cOption, optionsTradablePair.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetRiskLimit(t.Context(), cLinear, usdtMarginedTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetRiskLimit(t.Context(), cLinear, usdcMarginedTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetRiskLimit(t.Context(), cInverse, inverseTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetRiskLimit(t.Context(), cOption, optionsTradablePair.String())
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetRiskLimit(t.Context(), cSpot, spotTradablePair.String())
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateTicker(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateTicker(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateTicker(t.Context(), inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateTicker(t.Context(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	var err error
	_, err = e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateOrderbook(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateOrderbook(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = e.UpdateOrderbook(t.Context(), inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.UpdateOrderbook(t.Context(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderSubmission := &order.Submit{
		Exchange:      e.GetName(),
		Pair:          spotTradablePair,
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         1,
		Amount:        1,
		ClientOrderID: "1234",
		AssetType:     asset.Spot,
	}
	_, err := e.SubmitOrder(t.Context(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      e.GetName(),
		AssetType:     asset.Options,
		Pair:          optionsTradablePair,
		Side:          order.Sell,
		Type:          order.Market,
		Price:         1,
		Amount:        1,
		Leverage:      1234,
		ClientOrderID: "1234",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ModifyOrder(t.Context(), &order.Modify{
		OrderID:      "1234",
		Type:         order.Limit,
		Side:         order.Buy,
		AssetType:    asset.Options,
		Pair:         spotTradablePair,
		Price:        1234,
		Amount:       0.15,
		TriggerPrice: 1145,
		RiskManagementModes: order.RiskManagementModes{
			StopLoss: order.RiskManagement{
				Price: 0,
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	end := time.Now()
	start := end.AddDate(0, 0, -3)
	if mockTests {
		start = time.UnixMilli(1692748800000)
		end = time.UnixMilli(1693094400000)
	}
	_, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandles(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandles(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandles(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, kline.OneHour, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandles(t.Context(), optionsTradablePair, asset.Options, kline.OneHour, start, end)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)
	if mockTests {
		startTime = time.UnixMilli(1692889428738)
		end = time.UnixMilli(1693145028738)
	}
	_, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandlesExtended(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, kline.OneHour, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandlesExtended(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, kline.OneDay, time.UnixMilli(1692889428738), time.UnixMilli(1693145028738))
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricCandlesExtended(t.Context(), optionsTradablePair, asset.Options, kline.FiveMin, startTime, end)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  e.Name,
		AssetType: asset.Spot,
		Pair:      spotTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
	err = e.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  e.Name,
		AssetType: asset.USDTMarginedFutures,
		Pair:      usdtMarginedTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}

	err = e.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  e.Name,
		AssetType: asset.CoinMarginedFutures,
		Pair:      inverseTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
	err = e.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  e.Name,
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: spotTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.USDTMarginedFutures, Pair: usdtMarginedTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.CoinMarginedFutures, Pair: inverseTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Options, Pair: optionsTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Futures, Pair: spotTradablePair})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOrderInfo(t.Context(),
		"12234", spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOrderInfo(t.Context(),
		"12234", usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOrderInfo(t.Context(),
		"12234", inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOrderInfo(t.Context(),
		"12234", optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequestSpot := order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}
	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestLinear := order.MultiOrderRequest{Pairs: currency.Pairs{usdtMarginedTradablePair}, AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType}
	_, err = e.GetActiveOrders(t.Context(), &getOrdersRequestLinear)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestInverse := order.MultiOrderRequest{Pairs: currency.Pairs{inverseTradablePair}, AssetType: asset.CoinMarginedFutures, Side: order.AnySide, Type: order.AnyType}
	_, err = e.GetActiveOrders(t.Context(), &getOrdersRequestInverse)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestFutures := order.MultiOrderRequest{Pairs: currency.Pairs{optionsTradablePair}, AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err = e.GetActiveOrders(t.Context(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
	pairs, err := currency.NewPairsFromStrings([]string{"BTC_USDT", "BTC_ETH", "BTC_USDC"})
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequestSpot = order.MultiOrderRequest{Pairs: pairs, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
	_, err = e.GetActiveOrders(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequestSpot := order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestUMF := order.MultiOrderRequest{
		Pairs:     currency.Pairs{usdtMarginedTradablePair},
		AssetType: asset.USDTMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestUMF.Pairs = currency.Pairs{usdcMarginedTradablePair}
	getOrdersRequestUMF.AssetType = asset.USDCMarginedFutures
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestCMF := order.MultiOrderRequest{
		Pairs:     currency.Pairs{inverseTradablePair},
		AssetType: asset.CoinMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequestCMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestFutures := order.MultiOrderRequest{
		Pairs:     currency.Pairs{optionsTradablePair},
		AssetType: asset.Options,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", currency.ETH.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: "Bybit",
		Amount:   -0.1,
		Currency: currency.LTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		},
	})
	if err != nil && err.Error() != "Withdraw address chain or destination tag are not equal" {
		t.Fatal(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	err := e.UpdateTickers(ctx, asset.Spot)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Spot, err)
	}
	err = e.UpdateTickers(ctx, asset.USDTMarginedFutures)
	if err != nil {
		t.Fatalf("%v %v\n", asset.USDTMarginedFutures, err)
	}
	err = e.UpdateTickers(ctx, asset.CoinMarginedFutures)
	if err != nil {
		t.Fatalf("%v %v\n", asset.CoinMarginedFutures, err)
	}
	err = e.UpdateTickers(ctx, asset.Options)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Options, err)
	}
}

func TestGetTickersV5(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), "bruh", "", "", time.Time{})
	require.ErrorIs(t, err, errInvalidCategory)
	_, err = e.GetTickers(t.Context(), cOption, "BTC-26NOV24-92000-C", "", time.Time{})
	require.NoError(t, err)
	_, err = e.GetTickers(t.Context(), cSpot, "", "", time.Time{})
	require.NoError(t, err)
	_, err = e.GetTickers(t.Context(), cInverse, "", "", time.Time{})
	require.NoError(t, err)
	_, err = e.GetTickers(t.Context(), cLinear, "", "", time.Time{})
	require.NoError(t, err)
	_, err = e.GetTickers(t.Context(), cOption, "", "BTC", time.Time{})
	require.NoError(t, err)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateHistory(t.Context(), "bruh", "", time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetFundingRateHistory(t.Context(), cSpot, spotTradablePair.String(), time.Time{}, time.Time{}, 100)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetFundingRateHistory(t.Context(), cLinear, usdtMarginedTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetFundingRateHistory(t.Context(), cLinear, usdcMarginedTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetFundingRateHistory(t.Context(), cInverse, inverseTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetFundingRateHistory(t.Context(), cOption, optionsTradablePair.String(), time.Time{}, time.Time{}, 100)
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestGetPublicTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicTradingHistory(t.Context(), cSpot, spotTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetPublicTradingHistory(t.Context(), cLinear, usdtMarginedTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetPublicTradingHistory(t.Context(), cLinear, usdcMarginedTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetPublicTradingHistory(t.Context(), cInverse, inverseTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetPublicTradingHistory(t.Context(), cOption, optionsTradablePair.String(), "BTC", "", 30)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestData(t.Context(), cSpot, spotTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetOpenInterestData(t.Context(), cLinear, usdtMarginedTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOpenInterestData(t.Context(), cLinear, usdcMarginedTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOpenInterestData(t.Context(), cInverse, inverseTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetOpenInterestData(t.Context(), cOption, optionsTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-time.Hour * 30 * 24)
	end := time.Now()
	if mockTests {
		end = time.UnixMilli(1693080759395)
		start = time.UnixMilli(1690488759395)
	}
	_, err := e.GetHistoricalVolatility(t.Context(), cOption, "", 123, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricalVolatility(t.Context(), cSpot, "", 123, start, end)
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	_, err := e.GetInsurance(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPrice(t.Context(), cSpot, spotTradablePair.String(), "", "", 200)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetDeliveryPrice(t.Context(), cLinear, "", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetDeliveryPrice(t.Context(), cInverse, "", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetDeliveryPrice(t.Context(), cOption, "", "BTC", "", 200)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, true)
			require.NoError(t, err, "GetPairs must not error")

			for _, p := range pairs {
				t.Run(p.String(), func(t *testing.T) {
					t.Parallel()
					l, err := e.GetOrderExecutionLimits(a, p)
					require.NoError(t, err, "GetOrderExecutionLimits must not error")
					assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")

					if !l.Delisted.IsZero() {
						assert.NotZero(t, l.Delisting, "Delisting should be set for Delisted coins")
					}

					pair := l.Key.Pair()
					require.True(t, pair.Equal(p), "Pair must be equal to input")
					require.Greater(t, len(pair.String()), 3, "pair string length must be > 3 to check for 1xxx rule")
					require.Equal(t, e.Name, l.Key.Exchange, "Exchange must be equal to input")
					require.Equal(t, a, l.Key.Asset, "Asset must be equal to input")

					assert.Positive(t, l.PriceDivisor, "PriceDivisor should be positive")
					if pair.String()[:2] == "10" {
						assert.Greater(t, l.PriceDivisor, 1.0, "PriceDivisor for 1xxx pairs should be > 1.0")
					}

					if a == asset.USDTMarginedFutures && !pair.Quote.Equal(currency.USDT) {
						assert.NotZero(t, l.Expiry, "Expiry should be set for USDT margined non-USDT pairs")
					}
				})
			}
		})
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{
		Category:         cSpot,
		Symbol:           spotTradablePair,
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
	arg := &PlaceOrderRequest{Category: cSpot, Symbol: spotTradablePair, Side: "Buy", OrderType: "Limit", OrderQuantity: 0.1, Price: 15600, TimeInForce: "PostOnly", OrderLinkID: "spot-test-01", IsLeverage: 0, OrderFilter: "Order"}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// Spot TP/SL order
	arg = &PlaceOrderRequest{
		Category: cSpot,
		Symbol:   spotTradablePair,
		Side:     "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TriggerPrice: 15000,
		TimeInForce: "GTC", OrderLinkID: "spot-test-02", IsLeverage: 0, OrderFilter: "tpslOrder",
	}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// Spot margin normal order (UTA)
	arg = &PlaceOrderRequest{
		Category: cSpot, Symbol: spotTradablePair, Side: "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TimeInForce: "IOC", OrderLinkID: "spot-test-limit", IsLeverage: 1, OrderFilter: "Order",
	}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	arg = &PlaceOrderRequest{
		Category: cSpot,
		Symbol:   spotTradablePair,
		Side:     "Buy", OrderType: "Market", OrderQuantity: 200,
		TimeInForce: "IOC", OrderLinkID: "spot-test-04",
		IsLeverage: 0, OrderFilter: "Order",
	}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// USDT Perp open long position (one-way mode)
	arg = &PlaceOrderRequest{
		Category: cLinear,
		Symbol:   usdcMarginedTradablePair, Side: "Buy", OrderType: "Limit", OrderQuantity: 1, Price: 25000, TimeInForce: "GTC", PositionIdx: 0, OrderLinkID: "usdt-test-01", ReduceOnly: false, TakeProfitPrice: 28000, StopLossPrice: 20000, TpslMode: "Partial", TpOrderType: "Limit", SlOrderType: "Limit", TpLimitPrice: 27500, SlLimitPrice: 20500,
	}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// USDT Perp close long position (one-way mode)
	arg = &PlaceOrderRequest{
		Category: cLinear, Symbol: usdtMarginedTradablePair, Side: "Sell",
		OrderType: "Limit", OrderQuantity: 1, Price: 3000, TimeInForce: "GTC", PositionIdx: 0, OrderLinkID: "usdt-test-02", ReduceOnly: true,
	}
	_, err = e.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := e.AmendOrder(t.Context(), &AmendOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{
		OrderID:       "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category:      cSpot,
		Symbol:        spotTradablePair,
		TriggerPrice:  1145,
		OrderQuantity: 0.15,
		Price:         1050,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()

	_, err := e.CancelTradeOrder(t.Context(), &CancelOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.CancelTradeOrder(t.Context(), &CancelOrderRequest{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: cOption,
		Symbol:   optionsTradablePair,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetOpenOrders(t.Context(), "", "", "", "", "", "", "", "", 0, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetOpenOrders(t.Context(), cSpot, "", "", "", "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllTradeOrders(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CancelAllTradeOrders(t.Context(), &CancelAllOrdersParam{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.CancelAllTradeOrders(t.Context(), &CancelAllOrdersParam{Category: cOption})
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeOrderHistory(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-time.Hour * 24 * 6)
	end := time.Now()
	if mockTests {
		end = time.UnixMilli(1700058627109)
		start = time.UnixMilli(1699540227109)
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetTradeOrderHistory(t.Context(), "", "", "", "", "", "", "", "", "", start, end, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetTradeOrderHistory(t.Context(), cSpot, spotTradablePair.String(), "", "", "BTC", "", "StopOrder", "", "", start, end, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: cLinear,
	})
	require.ErrorIs(t, err, errNoOrderPassed)

	_, err = e.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: cOption,
		Request: []BatchOrderItemParam{
			{
				Symbol:                optionsTradablePair,
				OrderType:             "Limit",
				Side:                  "Buy",
				OrderQuantity:         1,
				OrderIv:               6,
				TimeInForce:           "GTC",
				OrderLinkID:           "option-test-001",
				MarketMakerProtection: false,
				ReduceOnly:            false,
			},
			{
				Symbol:                optionsTradablePair,
				OrderType:             "Limit",
				Side:                  "Sell",
				OrderQuantity:         2,
				Price:                 700,
				TimeInForce:           "GTC",
				OrderLinkID:           "option-test-001",
				MarketMakerProtection: false,
				ReduceOnly:            false,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: cLinear,
		Request: []BatchOrderItemParam{
			{
				Symbol:                optionsTradablePair,
				OrderType:             "Limit",
				Side:                  "Buy",
				OrderQuantity:         1,
				OrderIv:               6,
				TimeInForce:           "GTC",
				OrderLinkID:           "linear-test-001",
				MarketMakerProtection: false,
				ReduceOnly:            false,
			},
			{
				Symbol:                optionsTradablePair,
				OrderType:             "Limit",
				Side:                  "Sell",
				OrderQuantity:         2,
				Price:                 700,
				TimeInForce:           "GTC",
				OrderLinkID:           "linear-test-001",
				MarketMakerProtection: false,
				ReduceOnly:            false,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchAmendOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.BatchAmendOrder(t.Context(), cLinear, nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.BatchAmendOrder(t.Context(), "", []BatchAmendOrderParamItem{
		{
			Symbol:                 optionsTradablePair,
			OrderImpliedVolatility: "6.8",
			OrderID:                "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
		},
	})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.BatchAmendOrder(t.Context(), cOption, []BatchAmendOrderParamItem{
		{
			Symbol:                 optionsTradablePair,
			OrderImpliedVolatility: "6.8",
			OrderID:                "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
		},
		{
			Symbol:  optionsTradablePair,
			Price:   650,
			OrderID: "fa6a595f-1a57-483f-b9d3-30e9c8235a52",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelBatchOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CancelBatchOrder(t.Context(), &CancelBatchOrder{})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.CancelBatchOrder(t.Context(), &CancelBatchOrder{Category: cOption})
	require.ErrorIs(t, err, errNoOrderPassed)

	_, err = e.CancelBatchOrder(t.Context(), &CancelBatchOrder{
		Category: cOption,
		Request: []CancelOrderRequest{
			{
				Symbol:  optionsTradablePair,
				OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
			},
			{
				Symbol:  optionsTradablePair,
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
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetBorrowQuota(t.Context(), "", "BTCUSDT", "Buy")
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetBorrowQuota(t.Context(), cSpot, "", "Buy")
	require.ErrorIs(t, err, errSymbolMissing)

	_, err = e.GetBorrowQuota(t.Context(), cSpot, spotTradablePair.String(), "")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.GetBorrowQuota(t.Context(), cSpot, spotTradablePair.String(), "Buy")
	if err != nil {
		t.Error(err)
	}
}

func TestSetDisconnectCancelAll(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetDisconnectCancelAll(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = e.SetDisconnectCancelAll(t.Context(), &SetDCPParams{TimeWindow: 300})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPositionInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetPositionInfo(t.Context(), "", "", "", "", "", 20)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetPositionInfo(t.Context(), cSpot, "", "", "", "", 20)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPositionInfo(t.Context(), cLinear, "BTCUSDT", "", "", "", 20)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetPositionInfo(t.Context(), cOption, "BTC-26NOV24-92000-C", "BTC", "", "", 20)
	if err != nil {
		t.Error(err)
	}
}

func TestSetLeverageLevel(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetLeverageLevel(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	err = e.SetLeverageLevel(t.Context(), &SetLeverageParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = e.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: cSpot})
	require.ErrorIs(t, err, errInvalidCategory)

	err = e.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing)

	err = e.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: cLinear, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errInvalidLeverage)

	err = e.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: cLinear, Symbol: "BTCUSDT", SellLeverage: 3, BuyLeverage: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchTradeMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SwitchTradeMode(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: cSpot})
	require.ErrorIs(t, err, errInvalidCategory)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: cLinear, Symbol: usdtMarginedTradablePair.String()})
	require.ErrorIs(t, err, errInvalidLeverage)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: cLinear, Symbol: usdcMarginedTradablePair.String(), SellLeverage: 3, BuyLeverage: 3, TradeMode: 2})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	err = e.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: cLinear, Symbol: usdtMarginedTradablePair.String(), SellLeverage: 3, BuyLeverage: 3, TradeMode: 1})
	if err != nil {
		t.Error(err)
	}
}

func TestSetTakeProfitStopLossMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.SetTakeProfitStopLossMode(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{
		Category: cSpot,
	})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: cSpot})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: cLinear, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errTakeProfitOrStopLossModeMissing)

	_, err = e.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: cLinear, Symbol: "BTCUSDT", TpslMode: "Partial"})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SwitchPositionMode(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = e.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = e.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{Category: cLinear})
	require.ErrorIs(t, err, errEitherSymbolOrCoinRequired)

	err = e.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{Category: cLinear, Symbol: usdtMarginedTradablePair, PositionMode: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSetRiskLimit(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.SetRiskLimit(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.SetRiskLimit(t.Context(), &SetRiskLimitParam{})
	assert.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.SetRiskLimit(t.Context(), &SetRiskLimitParam{Category: cLinear, PositionMode: -2})
	assert.ErrorIs(t, err, errInvalidPositionMode)

	_, err = e.SetRiskLimit(t.Context(), &SetRiskLimitParam{Category: cLinear})
	assert.ErrorIs(t, err, errSymbolMissing)

	_, err = e.SetRiskLimit(t.Context(), &SetRiskLimitParam{
		Category:     cLinear,
		RiskID:       1234,
		Symbol:       usdtMarginedTradablePair,
		PositionMode: 0,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetTradingStop(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetTradingStop(t.Context(), &TradingStopParams{})
	assert.ErrorIs(t, err, errCategoryNotSet)

	err = e.SetTradingStop(t.Context(), &TradingStopParams{Category: cSpot})
	assert.ErrorIs(t, err, errInvalidCategory)

	err = e.SetTradingStop(t.Context(), &TradingStopParams{
		Category:                 cLinear,
		Symbol:                   usdtMarginedTradablePair,
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
	err = e.SetTradingStop(t.Context(), &TradingStopParams{
		Category:                 cLinear,
		Symbol:                   usdcMarginedTradablePair,
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
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetAutoAddMargin(t.Context(), &AutoAddMarginParam{
		Category:      cInverse,
		Symbol:        inverseTradablePair,
		AutoAddmargin: 0,
		PositionIndex: 2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAddOrReduceMargin(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.AddOrReduceMargin(t.Context(), &AddOrReduceMarginParam{
		Category:      cInverse,
		Symbol:        inverseTradablePair,
		Margin:        -10,
		PositionIndex: 2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetExecution(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetExecution(t.Context(), cSpot, "", "", "", "", "Trade", "tpslOrder", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetClosedPnL(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetClosedPnL(t.Context(), cSpot, "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetClosedPnL(t.Context(), cLinear, "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfirmNewRiskLimit(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.ConfirmNewRiskLimit(t.Context(), cLinear, "BTCUSDT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeOrderHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeOrderHistory(t.Context(), "", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetPreUpgradeOrderHistory(t.Context(), cOption, "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errBaseNotSet)

	_, err = e.GetPreUpgradeOrderHistory(t.Context(), cLinear, "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTradeHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeTradeHistory(t.Context(), "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = e.GetPreUpgradeTradeHistory(t.Context(), cOption, "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPreUpgradeTradeHistory(t.Context(), cLinear, "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeClosedPnL(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeClosedPnL(t.Context(), cOption, "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPreUpgradeClosedPnL(t.Context(), cLinear, "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTransactionLog(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeTransactionLog(t.Context(), cOption, "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPreUpgradeTransactionLog(t.Context(), cLinear, "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeOptionDeliveryRecord(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeOptionDeliveryRecord(t.Context(), cLinear, "", "", time.Time{}, 0)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPreUpgradeOptionDeliveryRecord(t.Context(), cOption, "", "", time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPreUpgradeUSDCSessionSettlement(t.Context(), cOption, "", "", 10)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetPreUpgradeUSDCSessionSettlement(t.Context(), cLinear, "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	r, err := e.GetWalletBalance(t.Context(), "UNIFIED", "")
	require.NoError(t, err, "GetWalletBalance must not error")
	require.NotNil(t, r, "GetWalletBalance must return a result")

	if mockTests {
		require.Len(t, r.List, 1, "GetWalletBalance must return a single list result")
		assert.Equal(t, types.Number(0.1997), r.List[0].AccountIMRate, "AccountIMRate should be correct")
		assert.Equal(t, types.Number(0.4996), r.List[0].AccountLTV, "AccountLTV should be correct")
		assert.Equal(t, types.Number(0.0399), r.List[0].AccountMMRate, "AccountMMRate should be correct")
		assert.Equal(t, "UNIFIED", r.List[0].AccountType, "AccountType should be correct")
		assert.Equal(t, types.Number(24616.49915805), r.List[0].TotalAvailableBalance, "TotalAvailableBalance should be correct")
		assert.Equal(t, types.Number(41445.9203332), r.List[0].TotalEquity, "TotalEquity should be correct")
		assert.Equal(t, types.Number(6144.46796478), r.List[0].TotalInitialMargin, "TotalInitialMargin should be correct")
		assert.Equal(t, types.Number(1228.89359295), r.List[0].TotalMaintenanceMargin, "TotalMaintenanceMargin should be correct")
		assert.Equal(t, types.Number(30760.96712284), r.List[0].TotalMarginBalance, "TotalMarginBalance should be correct")
		assert.Equal(t, types.Number(0.0), r.List[0].TotalPerpUPL, "TotalPerpUPL should be correct")
		assert.Equal(t, types.Number(30760.96712284), r.List[0].TotalWalletBalance, "TotalWalletBalance should be correct")
		require.Len(t, r.List[0].Coin, 3, "GetWalletBalance must return 3 coins")

		for x := range r.List[0].Coin {
			switch x {
			case 0:
				assert.Equal(t, types.Number(0.21976631), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should be correct")
				assert.Equal(t, types.Number(30723.630216383714), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should be correct")
				assert.Equal(t, currency.USDC, r.List[0].Coin[x].Coin, "Coin should be correct")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should be correct")
				assert.Equal(t, types.Number(-30723.63021638), r.List[0].Coin[x].Equity, "Equity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should be correct")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should be correct")
				assert.Equal(t, types.Number(-30722.33982391), r.List[0].Coin[x].USDValue, "USDValue should be correct")
				assert.Equal(t, types.Number(-30723.63021638), r.List[0].Coin[x].WalletBalance, "WalletBalance should be correct")
			case 1:
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should be correct")
				assert.Equal(t, types.Number(1005.79191187), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should be correct")
				assert.Equal(t, currency.AVAX, r.List[0].Coin[x].Coin, "Coin should be correct")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should be correct")
				assert.Equal(t, types.Number(2473.9), r.List[0].Coin[x].Equity, "Equity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should be correct")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should be correct")
				assert.Equal(t, types.Number(71233.0214024), r.List[0].Coin[x].USDValue, "USDValue should be correct")
				assert.Equal(t, types.Number(2473.9), r.List[0].Coin[x].WalletBalance, "WalletBalance should be correct")
			case 2:
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should be correct")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should be correct")
				assert.Equal(t, currency.USDT, r.List[0].Coin[x].Coin, "Coin should be correct")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should be correct")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].Equity, "Equity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should be correct")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should be correct")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should be correct")
				assert.Equal(t, types.Number(935.23875471), r.List[0].Coin[x].USDValue, "USDValue should be correct")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].WalletBalance, "WalletBalance should be correct")
			}
		}
	}
}

func TestUpgradeToUnifiedAccount(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.UpgradeToUnifiedAccount(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetBorrowHistory(t.Context(), "BTC", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetCollateralCoin(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetCollateralCoin(t.Context(), currency.BTC, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetCollateralInfo(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinGreeks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetCoinGreeks(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetFeeRate(t.Context(), "something", "", "BTC")
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetFeeRate(t.Context(), cLinear, "", "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAccountInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetTransactionLog(t.Context(), cOption, "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetTransactionLog(t.Context(), cLinear, "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMarginMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.SetMarginMode(t.Context(), "PORTFOLIO_MARGIN")
	if err != nil {
		t.Error(err)
	}
}

func TestSetSpotHedging(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetSpotHedging(t.Context(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountALLAPIKeys(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountAllAPIKeys(t.Context(), "", "", 10)
	assert.ErrorIs(t, err, errMemberIDRequired)

	_, err = e.GetSubAccountAllAPIKeys(t.Context(), "1234", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.SetMMP(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = e.SetMMP(t.Context(), &MMPRequestParam{
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
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.ResetMMP(t.Context(), "USDT")
	require.ErrorIs(t, err, errNilArgument)

	err = e.ResetMMP(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMMPState(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetMMPState(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinExchangeRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetCoinExchangeRecords(t.Context(), "", "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDeliveryRecord(t *testing.T) {
	t.Parallel()
	expiryTime := time.Now().Add(time.Hour * 40)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	} else {
		expiryTime = time.UnixMilli(1700216290093)
	}
	_, err := e.GetDeliveryRecord(t.Context(), cSpot, "", "", expiryTime, 20)
	assert.ErrorIs(t, err, errInvalidCategory)
	_, err = e.GetDeliveryRecord(t.Context(), cLinear, "", "", expiryTime, 20)
	assert.NoError(t, err, "GetDeliveryRecord should not error for linear category")
}

func TestGetUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetUSDCSessionSettlement(t.Context(), cOption, "", "", 10)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = e.GetUSDCSessionSettlement(t.Context(), cLinear, "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAssetInfo(t.Context(), "", "BTC")
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.GetAssetInfo(t.Context(), "SPOT", "BTC")
	assert.NoError(t, err, "GetAssetInfo should not error for SPOT account type")
}

func TestGetAllCoinBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAllCoinBalance(t.Context(), "", "", "", 0)
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.GetAllCoinBalance(t.Context(), "FUND", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSingleCoinBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSingleCoinBalance(t.Context(), "", "", "", 0, 0)
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.GetSingleCoinBalance(t.Context(), "SPOT", currency.BTC.String(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTransferableCoin(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetTransferableCoin(t.Context(), "SPOT", "OPTION")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateInternalTransfer(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CreateInternalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{})
	require.ErrorIs(t, err, errMissingTransferID)

	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{TransferID: transferID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetInternalTransferRecords(t *testing.T) {
	t.Parallel()
	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	transferIDString := transferID.String()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	} else {
		transferIDString = "018bd458-dba0-728b-b5b6-ecd5bd296528"
	}
	_, err = e.GetInternalTransferRecords(t.Context(), transferIDString, currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubUID(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSubUID(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestEnableUniversalTransferForSubUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.EnableUniversalTransferForSubUID(t.Context())
	require.ErrorIs(t, err, errMembersIDsNotSet)

	transferID1, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	transferID2, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	err = e.EnableUniversalTransferForSubUID(t.Context(), transferID1.String(), transferID2.String())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CreateUniversalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{})
	require.ErrorIs(t, err, errMissingTransferID)

	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{TransferID: transferID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMemberIDRequired)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateUniversalTransfer(t.Context(), &TransferParams{
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
	var transferIDString string
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		transferID, err := uuid.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		transferIDString = transferID.String()
	} else {
		transferIDString = "018bd461-cb9c-75ce-94d4-0d3f4d84c339"
	}
	_, err := e.GetUniversalTransferRecords(t.Context(), transferIDString, currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllowedDepositCoinInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAllowedDepositCoinInfo(t.Context(), "BTC", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetDepositAccount(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.SetDepositAccount(t.Context(), "FUND")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetDepositRecords(t.Context(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubDepositRecords(t.Context(), "12345", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestInternalDepositRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetInternalDepositRecordsOffChain(t.Context(), currency.ETH.String(), "", time.Time{}, time.Time{}, 8)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMasterDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetMasterDepositAddress(t.Context(), currency.LTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositAddress(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubDepositAddress(t.Context(), currency.LTC, "LTC", "12345")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetWithdrawalRecords(t.Context(), currency.LTC, "", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawableAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetWithdrawableAmount(t.Context(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.WithdrawCurrency(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.BTC})
	require.ErrorIs(t, err, errMissingChainInformation)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC"})
	require.ErrorIs(t, err, errMissingAddressInfo)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234", Amount: -0.1})
	require.NoError(t, err)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID)

	_, err = e.CancelWithdrawal(t.Context(), "12314")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewSubUserID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateNewSubUserID(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CreateNewSubUserID(t.Context(), &CreateSubUserParams{MemberType: 1, Switch: 1, Note: "test"})
	require.ErrorIs(t, err, errMissingUsername)

	_, err = e.CreateNewSubUserID(t.Context(), &CreateSubUserParams{Username: "Sami", Switch: 1, Note: "test"})
	require.ErrorIs(t, err, errInvalidMemberType)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateNewSubUserID(t.Context(), &CreateSubUserParams{Username: "sami", MemberType: 1, Switch: 1, Note: "test"})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubUIDAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubUIDAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CreateSubUIDAPIKey(t.Context(), &SubUIDAPIKeyParam{})
	require.ErrorIs(t, err, errMissingUserID)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateSubUIDAPIKey(t.Context(), &SubUIDAPIKeyParam{
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
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubUIDList(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestFreezeSubUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.FreezeSubUID(t.Context(), "1234", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyInformation(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAPIKeyInformation(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUIDWalletType(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUIDWalletType(t.Context(), "234234")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyMasterAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ModifyMasterAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.ModifyMasterAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{
		ReadOnly: 0,
		IPs:      "*",
		Permissions: PermissionsList{
			ContractTrade: []string{"Order", "Position"},
			Spot:          []string{"SpotTrade"},
			Wallet:        []string{"AccountTransfer", "SubMemberTransfer"},
			Options:       []string{"OptionsTrade"},
			CopyTrading:   []string{"CopyTrading"},
			Exchange:      []string{"ExchangeHistory"},
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestModifySubAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ModifySubAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.ModifySubAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{
		APIKey:   "lnqQ8ACaoMLi4168He",
		ReadOnly: 0,
		IPs:      "*",
		Permissions: PermissionsList{
			ContractTrade: []string{"Order", "Position"},
			Spot:          []string{"SpotTrade"},
			Wallet:        []string{"AccountTransfer", "SubMemberTransfer"},
			Options:       []string{"OptionsTrade"},
			CopyTrading:   []string{"CopyTrading"},
			Exchange:      []string{"ExchangeHistory"},
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteSubUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.DeleteSubUID(t.Context(), "")
	assert.ErrorIs(t, err, errMemberIDRequired)

	err = e.DeleteSubUID(t.Context(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMasterAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.DeleteMasterAPIKey(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteSubAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.DeleteSubAccountAPIKey(t.Context(), "12434")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateUserInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAffiliateUserInfo(t.Context(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeverageTokenInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLeverageTokenInfo(t.Context(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeveragedTokenMarket(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLeveragedTokenMarket(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetLeveragedTokenMarket(t.Context(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestPurchaseLeverageToken(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PurchaseLeverageToken(t.Context(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestRedeemLeverageToken(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.RedeemLeverageToken(t.Context(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPurchaseAndRedemptionRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPurchaseAndRedemptionRecords(t.Context(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestToggleMarginTrade(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ToggleMarginTrade(t.Context(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestSetSpotMarginTradeLeverage(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.SetSpotMarginTradeLeverage(t.Context(), 3)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetVIPMarginData(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPMarginData(t.Context(), "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowableCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowableCoinInfo(t.Context(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInterestAndQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetInterestAndQuota(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err = e.GetInterestAndQuota(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetLoanAccountInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLoanAccountInfo(t.Context())
	assert.NoError(t, err)
}

func TestBorrow(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.Borrow(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.Borrow(t.Context(), &LendArgument{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.Borrow(t.Context(), &LendArgument{Coin: currency.BTC})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.Borrow(t.Context(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}

func TestRepay(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.Repay(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.Repay(t.Context(), &LendArgument{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.Repay(t.Context(), &LendArgument{Coin: currency.BTC})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.Repay(t.Context(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowOrderDetail(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetBorrowOrderDetail(t.Context(), time.Time{}, time.Time{}, currency.BTC, 0, 0)
	assert.NoError(t, err)
}

func TestGetRepaymentOrderDetail(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetRepaymentOrderDetail(t.Context(), time.Time{}, time.Time{}, currency.BTC, 0)
	assert.NoError(t, err)
}

func TestToggleMarginTradeNormal(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.ToggleMarginTradeNormal(t.Context(), true)
	assert.NoError(t, err)
}

func TestGetProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetProductInfo(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLengingMarginCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstitutionalLengingMarginCoinInfo(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLoanOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetInstitutionalLoanOrders(t.Context(), "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetInstitutionalRepayOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetInstitutionalRepayOrders(t.Context(), time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetLTV(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLTV(t.Context())
	assert.NoError(t, err)
}

func TestBindOrUnbindUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.BindOrUnbindUID(t.Context(), "12234", "0")
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingCoinInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetC2CLendingCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestC2CDepositFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.C2CDepositFunds(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestC2CRedeemFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.C2CRedeemFunds(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = e.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingOrderRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetC2CLendingOrderRecords(t.Context(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingAccountInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetC2CLendingAccountInfo(t.Context(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBrokerEarning(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetBrokerEarning(t.Context(), "DERIVATIVES", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	e := testInstance()

	subAccts, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.NotEmpty(t, subAccts, "UpdateAccountBalances must return account info")

	if mockTests {
		require.Len(t, subAccts, 1, "Accounts must have 1 item")
		require.Len(t, subAccts[0].Balances, 3, "Accounts currencies must have 3 currency items")

		for _, curr := range []currency.Code{currency.USDC, currency.AVAX, currency.USDT} {
			t.Run(curr.String(), func(t *testing.T) {
				t.Parallel()
				require.Contains(t, subAccts[0].Balances, curr, "Balances must contain currency")
				bal := subAccts[0].Balances[curr]
				assert.Equal(t, curr, bal.Currency, "Balance Currency should be set")
				switch curr {
				case currency.USDC:
					assert.Equal(t, -30723.63021638, bal.Total, "Total amount should be correct")
					assert.Zero(t, bal.Hold, "Hold amount should be zero")
					assert.Equal(t, 30723.630216383711792744, bal.Borrowed, "Borrowed amount should be correct")
					assert.Zero(t, bal.Free, "Free amount should be zero")
					assert.Zero(t, bal.AvailableWithoutBorrow, "AvailableWithoutBorrow amount should be zero")
				case currency.AVAX:
					assert.Equal(t, 2473.9, bal.Total, "Total amount should be correct")
					assert.Zero(t, bal.Hold, "Hold amount should be zero")
					assert.Zero(t, bal.Borrowed, "Borrowed amount should be zero")
					assert.Equal(t, 2473.9, bal.Free, "Free amount should be correct")
					assert.Equal(t, 1005.79191187, bal.AvailableWithoutBorrow, "AvailableWithoutBorrow amount should be correct")
				case currency.USDT:
					assert.Equal(t, 935.1415, bal.Total, "Total amount should be correct")
					assert.Zero(t, bal.Borrowed, "Borrowed amount should be zero")
					assert.Zero(t, bal.Hold, "Hold amount should be zero")
					assert.Equal(t, 935.1415, bal.Free, "Free amount should be correct")
					assert.Equal(t, 935.1415, bal.AvailableWithoutBorrow, "AvailableWithoutBorrow amount should be correct")
				}
			})
		}
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error("GetWithdrawalsHistory()", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a asset.Item
		p currency.Pair
	}{
		{asset.Spot, spotTradablePair},
		{asset.Options, optionsTradablePair},
		{asset.CoinMarginedFutures, inverseTradablePair},
		{asset.USDTMarginedFutures, usdtMarginedTradablePair},
		{asset.USDCMarginedFutures, usdcMarginedTradablePair},
	} {
		_, err := e.GetRecentTrades(t.Context(), tt.p, tt.a)
		assert.NoErrorf(t, err, "GetRecentTrades should not error for %s asset", tt.a)
	}

	_, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetBybitServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetBybitServerTime(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricTrades(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricTrades(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricTrades(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetHistoricTrades(t.Context(), optionsTradablePair, asset.Options, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellationParams := []order.Cancel{{
		OrderID:   "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	}, {
		OrderID:   "1",
		Pair:      usdtMarginedTradablePair,
		AssetType: asset.USDTMarginedFutures,
	}}
	_, err := e.CancelBatchOrders(t.Context(), orderCancellationParams)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	orderCancellationParams = []order.Cancel{{
		OrderID:   "1",
		AccountID: "1",
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
	}, {
		OrderID:   "2",
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
	}}
	_, err = e.CancelBatchOrders(t.Context(), orderCancellationParams)
	if err != nil {
		t.Error(err)
	}
}

type FixtureConnection struct {
	dialError                         error
	sendMessageReturnResponseOverride []byte
	match                             *websocket.Match
	websocket.Connection
}

func (d *FixtureConnection) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {}
func (d *FixtureConnection) Dial(context.Context, *gws.Dialer, http.Header) error          { return d.dialError }

func (d *FixtureConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	if d.sendMessageReturnResponseOverride != nil {
		return d.sendMessageReturnResponseOverride, nil
	}
	return []byte(`{"success":true,"ret_msg":"subscribe","conn_id":"5758770c-8152-4545-a84f-dae089e56499","req_id":"1","op":"subscribe"}`), nil
}

func (d *FixtureConnection) SendJSONMessage(context.Context, request.EndpointLimit, any) error {
	return nil
}

func (d *FixtureConnection) RequireMatchWithData(signature any, data []byte) error {
	return d.match.RequireMatchWithData(signature, data)
}

func (d *FixtureConnection) IncomingWithData(signature any, data []byte) bool {
	return d.match.IncomingWithData(signature, data)
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := e.WsConnect(t.Context(), &FixtureConnection{dialError: nil})
	require.NoError(t, err)
	exp := errors.New("dial error")
	err = e.WsConnect(t.Context(), &FixtureConnection{dialError: exp})
	require.ErrorIs(t, err, exp)
}

var pushDataMap = map[string]string{
	"Orderbook Snapshot":   `{"topic":"orderbook.50.BTCUSDT","ts":1731035685326,"type":"snapshot","data":{"s":"BTCUSDT","b":[["75848.74","0.067669"],["75848.63","0.004772"],["75848.61","0.00659"],["75848.05","0.000329"],["75847.68","0.00159"],["75846.88","0.00159"],["75845.97","0.026366"],["75845.87","0.013185"],["75845.41","0.077259"],["75845.4","0.132228"],["75844.61","0.00159"],["75844.44","0.026367"],["75844.2","0.013185"],["75844","0.00039"],["75843.13","0.00159"],["75843.07","0.013185"],["75842.33","0.00159"],["75841.99","0.006"],["75841.75","0.019538"],["75841.74","0.04"],["75841.71","0.031817"],["75841.36","0.017336"],["75841.33","0.000072"],["75841.16","0.001872"],["75841.11","0.172641"],["75841.04","0.029772"],["75841","0.000065"],["75840.93","0.015244"],["75840.86","0.00159"],["75840.79","0.000072"],["75840.38","0.043333"],["75840.32","0.092539"],["75840.3","0.132228"],["75840.2","0.054966"],["75840.06","0.00159"],["75840","0.20726"],["75839.64","0.003744"],["75839.29","0.006592"],["75838.58","0.00159"],["75838.52","0.049778"],["75838.14","0.003955"],["75838","0.000065"],["75837.78","0.00159"],["75837.75","0.000587"],["75837.53","0.322245"],["75837.52","0.593323"],["75837.37","0.00384"],["75837.29","0.044335"],["75837.24","0.119228"],["75837.13","0.152844"]],"a":[["75848.75","0.747137"],["75848.89","0.060306"],["75848.9","0.1"],["75851.43","0.00159"],["75851.44","0.080754"],["75852.23","0.00159"],["75852.54","0.131067"],["75852.65","0.003955"],["75853.71","0.00159"],["75853.86","0.003955"],["75854.43","0.015684"],["75854.5","0.130389"],["75854.51","0.00159"],["75855.21","0.031168"],["75855.23","0.271494"],["75855.73","0.042698"],["75855.98","0.00159"],["75856.04","0.01346"],["75856.33","0.001872"],["75856.78","0.00159"],["75857.15","0.000072"],["75857.17","0.015127"],["75857.8","0.043322"],["75857.81","0.045305"],["75857.85","0.003792"],["75858.09","0.026344"],["75858.26","0.00159"],["75859.06","0.031618"],["75859.07","0.025"],["75859.1","0.006592"],["75859.98","0.013183"],["75860.12","0.00384"],["75860.54","0.00159"],["75860.74","0.051204"],["75860.75","0.065861"],["75861.18","0.031222"],["75861.33","0.00159"],["75861.64","0.003888"],["75861.96","0.042213"],["75862.28","0.000777"],["75862.79","0.013184"],["75862.81","0.00159"],["75862.84","0.027959"],["75863.16","0.003888"],["75863.51","0.043628"],["75863.52","0.002525"],["75863.61","0.00159"],["75864.2","0.003955"],["75864.76","0.000072"],["75864.81","0.002018"]],"u":2876700,"seq":47474967795},"cts":1731035685323}`,
	"Orderbook Update":     `{"topic":"orderbook.50.BTCUSDT","ts":1731035685345,"type":"delta","data":{"s":"BTCUSDT","b":[["75848.62","0.014895"],["75837.13","0"]],"a":[["75848.89","0.088149"],["75851.44","0.078379"],["75852.65","0"],["75855.23","0.260219"],["75857.74","0.049778"]],"u":2876701,"seq":47474967823},"cts":1731035685342}`,
	"Public Trade":         `{"topic":"publicTrade.BTCUSDT","ts":1690720953113,"type":"snapshot","data":[{"i":"2200000000067341890","T":1690720953111,"p":"3.6279","v":"1.3637","S":"Sell","s":"BTCUSDT","BT":false}]}`,
	"Public Kline":         `{ "topic": "kline.5.BTCUSDT", "data": [ { "start": 1672324800000, "end": 1672325099999, "interval": "5", "open": "16649.5", "close": "16677", "high": "16677", "low": "16608", "volume": "2.081", "turnover": "34666.4005", "confirm": false, "timestamp": 1672324988882} ], "ts": 1672324988882,"type": "snapshot"}`,
	"Public Liquidiation":  `{ "data": { "price": "0.03803", "side": "Buy", "size": "1637", "symbol": "GALAUSDT", "updatedTime": 1673251091822}, "topic": "liquidation.GALAUSDT", "ts": 1673251091822, "type": "snapshot" }`,
	"Public LT Kline":      `{ "type": "snapshot", "topic": "kline_lt.5.BTCUSDT", "data": [ { "start": 1672325100000, "end": 1672325399999, "interval": "5", "open": "0.416039541212402799", "close": "0.41477848043290448", "high": "0.416039541212402799", "low": "0.409734237314911206", "confirm": false, "timestamp": 1672325322393} ], "ts": 1672325322393}`,
	"Public LT Ticker":     `{ "topic": "tickers_lt.BTCUSDT", "ts": 1672325446847, "type": "snapshot", "data": { "symbol": "BTCUSDT", "lastPrice": "0.41477848043290448", "highPrice24h": "0.435285472510871305", "lowPrice24h": "0.394601507960931382", "prevPrice24h": "0.431502290172376349", "price24hPcnt": "-0.0388" } }`,
	"Public LT Navigation": `{ "topic": "lt.EOS3LUSDT", "ts": 1672325564669, "type": "snapshot", "data": { "symbol": "BTCUSDT", "time": 1672325564554, "nav": "0.413517419653406162", "basketPosition": "1.261060779498318641", "leverage": "2.656197506416192150", "basketLoan": "-0.684866519289629374", "circulation": "72767.309468460367138199", "basket": "91764.000000292013277472" } }`,
	"pong":                 `{"op":"pong","args":["1753340040127"],"conn_id":"d157a7favkf4mm3ibuvg-14toog"}`,
	"unhandled":            `{"topic": "unhandled"}`,
}

func TestWSHandleData(t *testing.T) {
	t.Parallel()

	e := testInstance()

	keys := slices.Collect(maps.Keys(pushDataMap))
	slices.Sort(keys)
	for x := range keys {
		err := e.wsHandleData(t.Context(), nil, asset.Spot, []byte(pushDataMap[keys[x]]))
		if keys[x] == "unhandled" {
			assert.ErrorIs(t, err, errUnhandledStreamData, "wsHandleData should error correctly for unhandled topics")
		} else {
			assert.NoError(t, err, "wsHandleData should not error")
		}
	}
}

func TestWSHandleAuthenticatedData(t *testing.T) {
	t.Parallel()

	err := e.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"op":"pong","args":["1753340040127"],"conn_id":"d157a7favkf4mm3ibuvg-14toog"}`))
	require.NoError(t, err, "wsHandleAuthenticatedData must not error for pong message")

	err = e.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic": "unhandled"}`))
	require.ErrorIs(t, err, errUnhandledStreamData, "wsHandleAuthenticatedData must error for unhandled stream data")

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials("test", "test", "", "", "", "")
	fErrs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsAuth.json", func(ctx context.Context, r []byte) error {
		if bytes.Contains(r, []byte("%s")) {
			r = fmt.Appendf(nil, string(r), optionsTradablePair.String())
		}
		if bytes.Contains(r, []byte("FANGLE-ACCOUNTS")) {
			hold := e.Accounts
			e.Accounts = nil
			defer func() { e.Accounts = hold }()
		}
		return e.wsHandleAuthenticatedData(ctx, &FixtureConnection{match: websocket.NewMatch()}, r)
	})
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 6, "Should see correct number of messages")
	require.Len(t, fErrs, 1, "Must get exactly one error message")
	assert.ErrorContains(t, fErrs[0].Err, "cannot save holdings: nil pointer: *accounts.Accounts")

	i := 0
	for data := range e.Websocket.DataHandler.C {
		i++
		switch v := data.Data.(type) {
		case WsPositions:
			require.Len(t, v, 1, "must see 1 position")
			assert.Zero(t, v[0].PositionIdx, "PositionIdx should be 0")
			assert.Zero(t, v[0].TradeMode, "TradeMode should be 0")
			assert.Equal(t, int64(41), v[0].RiskID, "RiskID should be correct")
			assert.Equal(t, 200000.0, v[0].RiskLimitValue.Float64(), "RiskLimitValue should be correct")
			assert.Equal(t, "XRPUSDT", v[0].Symbol, "Symbol should be correct")
			assert.Equal(t, "Buy", v[0].Side, "Side should be correct")
			assert.Equal(t, 75.0, v[0].Size.Float64(), "Size should be correct")
			assert.Equal(t, 0.3615, v[0].EntryPrice.Float64(), "Entry price should be correct")
			assert.Equal(t, 10.0, v[0].Leverage.Float64(), "Leverage should be correct")
			assert.Equal(t, 27.1125, v[0].PositionValue.Float64(), "Position value should be correct")
			assert.Zero(t, v[0].PositionBalance.Float64(), "Position balance should be 0")
			assert.Equal(t, 0.3374, v[0].MarkPrice.Float64(), "Mark price should be correct")
			assert.Equal(t, 2.72589075, v[0].PositionIM.Float64(), "Position IM should be correct")
			assert.Equal(t, 0.28576575, v[0].PositionMM.Float64(), "Position MM should be correct")
			assert.Zero(t, v[0].TakeProfit.Float64(), "Take profit should be 0")
			assert.Zero(t, v[0].StopLoss.Float64(), "Stop loss should be 0")
			assert.Zero(t, v[0].TrailingStop.Float64(), "Trailing stop should be 0")
			assert.Equal(t, -1.8075, v[0].UnrealisedPnl.Float64(), "Unrealised PnL should be correct")
			assert.Equal(t, 0.64782276, v[0].CumRealisedPnl.Float64(), "Cum realised PnL should be correct")
			assert.Equal(t, time.UnixMilli(1672121182216), v[0].CreatedTime.Time(), "Creation time should be correct")
			assert.Equal(t, time.UnixMilli(1672364174449), v[0].UpdatedTime.Time(), "Updated time should be correct")
			assert.Equal(t, "Full", v[0].TpslMode, "TPSL mode should be correct")
			assert.Zero(t, v[0].LiqPrice.Float64(), "Liq price should be 0")
			assert.Zero(t, v[0].BustPrice.Float64(), "Bust price should be 0")
			assert.Equal(t, cLinear, v[0].Category, "Category should be correct")
			assert.Equal(t, "Normal", v[0].PositionStatus, "Position status should be correct")
			assert.Equal(t, int64(2), v[0].AdlRankIndicator, "ADL Rank Indicator should be correct")
		case []order.Detail:
			if i == 6 {
				require.Len(t, v, 1)
				assert.Equal(t, "c1956690-b731-4191-97c0-94b00422231b", v[0].OrderID)
				assert.Equal(t, "BTC_USDT", v[0].Pair.String())
				assert.Equal(t, order.Sell, v[0].Side)
				assert.Equal(t, order.Filled, v[0].Status)
				assert.Equal(t, 1.7, v[0].Amount)
				assert.Equal(t, 4.033, v[0].Price)
				assert.Equal(t, 4.24, v[0].AverageExecutedPrice)
				assert.Equal(t, 0.0, v[0].RemainingAmount)
				assert.Equal(t, asset.USDTMarginedFutures, v[0].AssetType)
				continue
			}
			require.Len(t, v, 1, "must see 1 order")
			assert.True(t, optionsTradablePair.Equal(v[0].Pair), "Pair should match")
			assert.Equal(t, "5cf98598-39a7-459e-97bf-76ca765ee020", v[0].OrderID, "Order ID should be correct")
			assert.Equal(t, order.Sell, v[0].Side, "Side should be correct")
			assert.Equal(t, order.Market, v[0].Type, "Order type should be correct")
			assert.Equal(t, 72.5, v[0].Price, "Price should be correct")
			assert.Equal(t, 1.0, v[0].Amount, "Amount should be correct")
			assert.Equal(t, order.ImmediateOrCancel, v[0].TimeInForce, "Time in force should be correct")
			assert.Equal(t, order.Filled, v[0].Status, "Order status should be correct")
			assert.Empty(t, v[0].ClientOrderID, "client order ID should be empty")
			assert.False(t, v[0].ReduceOnly, "Reduce only should be false")
			assert.Equal(t, 1.0, v[0].ExecutedAmount, "executed amount should be correct")
			assert.Equal(t, 75.0, v[0].AverageExecutedPrice, "Avg price should be correct")
			assert.Equal(t, 0.358635, v[0].Fee, "fee should be correct")
			assert.Equal(t, time.UnixMilli(1672364262444), v[0].Date, "Created time should be correct")
			assert.Equal(t, time.UnixMilli(1672364262457), v[0].LastUpdated, "Updated time should be correct")
		case accounts.SubAccounts:
			require.Len(t, v, 1, "Must have correct number of SubAccounts")
			assert.Equal(t, asset.Spot, v[0].AssetType, "Asset type should be correct")
			exp := accounts.CurrencyBalances{}
			exp.Set(currency.ETH, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
			})
			exp.Set(currency.USDT, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
				Total:     11728.54414904,
				Free:      11728.54414904,
			})
			exp.Set(currency.EOS3L, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
				Total:     215.0570412,
				Free:      215.0570412,
			})
			exp.Set(currency.BIT, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
				Total:     1.82,
				Free:      1.82,
			})
			exp.Set(currency.USDC, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
				Total:     201.34882644,
				Free:      201.34882644,
			})
			exp.Set(currency.BTC, accounts.Balance{
				UpdatedAt: time.UnixMilli(1672364262482),
				Total:     0.06488393,
				Free:      0.06488393,
			})
			assert.Equal(t, exp, v[0].Balances, "Balances should be correct")
		case *GreeksResponse:
			assert.Equal(t, "592324fa945a30-2603-49a5-b865-21668c29f2a6", v.ID, "ID should be correct")
			assert.Equal(t, "greeks", v.Topic, "Topic should be correct")
			assert.Equal(t, time.UnixMilli(1672364262482), v.CreationTime.Time(), "Creation time should be correct")
			require.Len(t, v.Data, 1, "must see 1 greek")
			assert.Equal(t, "ETH", v.Data[0].BaseCoin.String(), "Base coin should be correct")
			assert.Equal(t, 0.06999986, v.Data[0].TotalDelta.Float64(), "Total delta should be correct")
			assert.Equal(t, -0.00000001, v.Data[0].TotalGamma.Float64(), "Total gamma should be correct")
			assert.Equal(t, -0.00000024, v.Data[0].TotalVega.Float64(), "Total vega should be correct")
			assert.Equal(t, 0.00001314, v.Data[0].TotalTheta.Float64(), "Total theta should be correct")
		case []fill.Data:
			require.Len(t, v, 1, "must see 1 fill")
			assert.Equal(t, "7e2ae69c-4edf-5800-a352-893d52b446aa", v[0].ID, "ID should be correct")
			assert.Equal(t, time.UnixMilli(1672364174443), v[0].Timestamp, "time should be correct")
			assert.Equal(t, e.Name, v[0].Exchange, "Exchange name should be correct")
			assert.Equal(t, asset.USDTMarginedFutures, v[0].AssetType, "Asset type should be correct")
			assert.Equal(t, "XRP_USDT", v[0].CurrencyPair.String(), "Symbol should be correct")
			assert.Equal(t, order.Sell, v[0].Side, "Side should be correct")
			assert.Equal(t, "f6e324ff-99c2-4e89-9739-3086e47f9381", v[0].OrderID, "Order ID should be correct")
			assert.Empty(t, v[0].ClientOrderID, "Client order ID should be empty")
			assert.Empty(t, v[0].TradeID, "Trade ID should be empty")
			assert.Equal(t, 0.3374, v[0].Price, "price should be correct")
			assert.Equal(t, 25.0, v[0].Amount, "amount should be correct")
		default:
			t.Errorf("Unexpected data received: %T %v", v, v)
		}
	}
}

func TestWsTicker(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	assetRouting := []asset.Item{
		asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDTMarginedFutures,
		asset.USDCMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.CoinMarginedFutures,
	}
	testexch.FixtureToDataHandler(t, "testdata/wsTicker.json", func(_ context.Context, r []byte) error {
		defer slices.Delete(assetRouting, 0, 1)
		return e.wsHandleData(t.Context(), nil, assetRouting[0], r)
	})
	e.Websocket.DataHandler.Close()
	expected := 8
	require.Len(t, e.Websocket.DataHandler.C, expected, "Should see correct number of tickers")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case *ticker.Price:
			assert.Equal(t, e.Name, v.ExchangeName, "ExchangeName should be correct")
			switch expected - len(e.Websocket.DataHandler.C) {
			case 1: // Spot
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.USDT, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 21109.77, v.Last, "Last should be correct")
				assert.Equal(t, 21426.99, v.High, "High should be correct")
				assert.Equal(t, 20575.00, v.Low, "Low should be correct")
				assert.Equal(t, 6780.866843, v.Volume, "Volume should be correct")
				assert.Equal(t, "BTC_USDT", v.Pair.String(), "Pair should be correct")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715742949283), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 2: // Option
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, 3565.00, v.Last, "Last should be correct")
				assert.Equal(t, 3715.00, v.High, "High should be correct")
				assert.Equal(t, 3555.00, v.Low, "Low should be correct")
				assert.Equal(t, 1.62, v.Volume, "Volume should be correct")
				assert.Equal(t, 3475.00, v.Bid, "Bid should be correct")
				assert.Equal(t, 10.14, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 3520.00, v.Ask, "Ask should be correct")
				assert.Equal(t, 2.5, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 3502.0715721, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61912.8, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 29.35, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, "BTC-28JUN24-60000-P", v.Pair.String(), "Pair should be correct")
				assert.Equal(t, asset.Options, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715742949283), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 3: // USDTMargined snapshot
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.USDT, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61874.00, v.Last, "Last should be correct")
				assert.Equal(t, 62752.90, v.High, "High should be correct")
				assert.Equal(t, 61000.10, v.Low, "Low should be correct")
				assert.Equal(t, 98430.1050, v.Volume, "Volume should be correct")
				assert.Equal(t, 61873.9, v.Bid, "Bid should be correct")
				assert.Equal(t, 3.783, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61874.00, v.Ask, "Ask should be correct")
				assert.Equal(t, 16.278, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61875.25, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61903.73, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 58117.022, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.USDTMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715748762463), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 4: // USDTMargined partial
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.USDT, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61874.00, v.Last, "Last should be correct")
				assert.Equal(t, 62752.90, v.High, "High should be correct")
				assert.Equal(t, 61000.10, v.Low, "Low should be correct")
				assert.Equal(t, 98430.1050, v.Volume, "Volume should be correct")
				assert.Equal(t, 61873.90, v.Bid, "Bid should be correct")
				assert.Equal(t, 3.543, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61874.00, v.Ask, "Ask should be correct")
				assert.Equal(t, 16.278, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61875.06, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61903.59, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 58117.022, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.USDTMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715748763063), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 5: // USDCMargined snapshot
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.PERP, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61945.70, v.Last, "Last should be correct")
				assert.Equal(t, 62242.2, v.High, "High should be correct")
				assert.Equal(t, 61059.1, v.Low, "Low should be correct")
				assert.Equal(t, 427.375, v.Volume, "Volume should be correct")
				assert.Equal(t, 61909.2, v.Bid, "Bid should be correct")
				assert.Equal(t, 0.035, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61909.60, v.Ask, "Ask should be correct")
				assert.Equal(t, 0.082, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61943.58, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61942.85, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 526.806, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.USDCMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715756612118), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 6: // USDCMargined partial
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.PERP, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61945.70, v.Last, "Last should be correct")
				assert.Equal(t, 62242.2, v.High, "High should be correct")
				assert.Equal(t, 61059.1, v.Low, "Low should be correct")
				assert.Equal(t, 427.375, v.Volume, "Volume should be correct")
				assert.Equal(t, 61909.5, v.Bid, "Bid should be correct")
				assert.Equal(t, 0.035, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61909.60, v.Ask, "Ask should be correct")
				assert.Equal(t, 0.082, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61943.58, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61942.85, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 526.806, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.USDCMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715756612210), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 7: // CoinMargined snapshot
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.USD, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61894.0, v.Last, "Last should be correct")
				assert.Equal(t, 62265.5, v.High, "High should be correct")
				assert.Equal(t, 61029.5, v.Low, "Low should be correct")
				assert.Equal(t, 391976479.0, v.Volume, "Volume should be correct")
				assert.Equal(t, 61891.5, v.Bid, "Bid should be correct")
				assert.Equal(t, 12667.0, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61892.0, v.Ask, "Ask should be correct")
				assert.Equal(t, 60953.0, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61894.0, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61923.36, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 931760496.0, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.CoinMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715757637952), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			case 8: // CoinMargined partial
				assert.Equal(t, currency.BTC, v.Pair.Base, "Pair base should be correct")
				assert.Equal(t, currency.USD, v.Pair.Quote, "Pair quote should be correct")
				assert.Equal(t, 61894.0, v.Last, "Last should be correct")
				assert.Equal(t, 62265.5, v.High, "High should be correct")
				assert.Equal(t, 61029.5, v.Low, "Low should be correct")
				assert.Equal(t, 391976479.0, v.Volume, "Volume should be correct")
				assert.Equal(t, 61891.5, v.Bid, "Bid should be correct")
				assert.Equal(t, 27634.0, v.BidSize, "BidSize should be correct")
				assert.Equal(t, 61892.0, v.Ask, "Ask should be correct")
				assert.Equal(t, 60953.0, v.AskSize, "AskSize should be correct")
				assert.Equal(t, 61894.0, v.MarkPrice, "MarkPrice should be correct")
				assert.Equal(t, 61923.36, v.IndexPrice, "IndexPrice should be correct")
				assert.Equal(t, 931760496.0, v.OpenInterest, "OpenInterest should be correct")
				assert.Equal(t, asset.CoinMarginedFutures, v.AssetType, "AssetType should be correct")
				assert.Equal(t, int64(1715757638152), v.LastUpdated.UnixMilli(), "LastUpdated should be correct")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                spotTradablePair,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = optionsTradablePair
	_, err = e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = usdtMarginedTradablePair
	_, err = e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = inverseTradablePair
	_, err = e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ctx := t.Context()
	err := e.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Multi, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}
	err = e.SetLeverage(ctx, asset.USDCMarginedFutures, usdcMarginedTradablePair, margin.Multi, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}

	err = e.SetLeverage(ctx, asset.CoinMarginedFutures, inverseTradablePair, margin.Isolated, 5, order.UnknownSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	err = e.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Isolated, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}

	err = e.SetLeverage(ctx, asset.CoinMarginedFutures, inverseTradablePair, margin.Isolated, 5, order.Sell)
	if err != nil {
		t.Error(err)
	}

	err = e.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Isolated, 5, order.CouldNotBuy)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	err = e.SetLeverage(ctx, asset.Spot, inverseTradablePair, margin.Multi, 5, order.UnknownSide)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	assert.NoError(t, err)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDCMarginedFutures)
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.FetchTradablePairs(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.FetchTradablePairs(t.Context(), asset.USDCMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.FetchTradablePairs(t.Context(), asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.FetchTradablePairs(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestDeltaUpdateOrderbook(t *testing.T) {
	t.Parallel()
	data := []byte(`{"topic":"orderbook.50.WEMIXUSDT","ts":1697573183768,"type":"snapshot","data":{"s":"WEMIXUSDT","b":[["0.9511","260.703"],["0.9677","0"]],"a":[],"u":3119516,"seq":14126848493},"cts":1728966699481}`)
	err := e.wsHandleData(t.Context(), nil, asset.Spot, data)
	require.NoError(t, err, "wsHandleData must not error")
	update := []byte(`{"topic":"orderbook.50.WEMIXUSDT","ts":1697573183768,"type":"delta","data":{"s":"WEMIXUSDT","b":[["0.9511","260.703"],["0.9677","0"]],"a":[],"u":3119516,"seq":14126848493},"cts":1728966699481}`)
	var wsResponse WebsocketResponse
	err = json.Unmarshal(update, &wsResponse)
	require.NoError(t, err, "Unmarshal must not error")
	err = e.wsProcessOrderbook(asset.Spot, &wsResponse)
	require.NoError(t, err, "wsProcessOrderbook must not error")
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetLongShortRatio(t.Context(), cLinear, "BTCUSDT", kline.FiveMin, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetLongShortRatio(t.Context(), cInverse, "BTCUSDT", kline.FiveMin, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetLongShortRatio(t.Context(), cSpot, "BTCUSDT", kline.FiveMin, 0)
	require.ErrorIs(t, err, errInvalidCategory)
}

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	input := []struct {
		OrderStatus string
		Expectation order.Status
	}{
		{
			OrderStatus: "",
			Expectation: order.UnknownStatus,
		},
		{
			OrderStatus: "UNKNOWN",
			Expectation: order.UnknownStatus,
		},
		{
			OrderStatus: "Cancelled",
			Expectation: order.Cancelled,
		},
		{
			OrderStatus: "ACTIVE",
			Expectation: order.Active,
		},
		{
			OrderStatus: "NEW",
			Expectation: order.New,
		},
		{
			OrderStatus: "FILLED",
			Expectation: order.Filled,
		},
		{
			OrderStatus: "UNTRIGGERED",
			Expectation: order.Pending,
		},
	}
	var oStatus order.Status
	for x := range input {
		oStatus = StringToOrderStatus(input[x].OrderStatus)
		if oStatus != input[x].Expectation {
			t.Fatalf("expected %v, got %v", input[x].Expectation, oStatus)
		}
	}
}

func TestFetchAccountType(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	val, err := e.FetchAccountType(t.Context())
	require.NoError(t, err)
	require.NotZero(t, val)
}

func TestAccountTypeString(t *testing.T) {
	t.Parallel()
	require.Equal(t, "unset", AccountType(0).String())
	require.Equal(t, "unified", accountTypeUnified.String())
	require.Equal(t, "normal", accountTypeNormal.String())
	require.Equal(t, "unknown", AccountType(3).String())
}

func TestRequiresUnifiedAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	err := e.RequiresUnifiedAccount(t.Context())
	require.NoError(t, err)
	b := &Exchange{}
	b.account.accountType = accountTypeNormal
	err = b.RequiresUnifiedAccount(t.Context())
	require.ErrorIs(t, err, errAPIKeyIsNotUnified)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  usdtMarginedTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Spot,
		Pair:  spotTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Options,
		Pair:  optionsTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDCMarginedFutures,
		Pair:  usdcMarginedTradablePair,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestConstructOrderDetails(t *testing.T) {
	t.Parallel()
	const data = `[	{"orderId": "fd4300ae-7847-404e-b947-b46980a4d140","orderLinkId": "test-000005","blockTradeId": "","symbol": "ETHUSDT","price": "1600.00","qty": "0.10","side": "Buy","isLeverage": "","positionIdx": 1,"orderStatus": "New","cancelType": "UNKNOWN","rejectReason": "EC_NoError","avgPrice": "0","leavesQty": "0.10","leavesValue": "160","cumExecQty": "0.00","cumExecValue": "0","cumExecFee": "0","timeInForce": "GTC","orderType": "Limit","stopOrderType": "UNKNOWN","orderIv": "","triggerPrice": "0.00","takeProfit": "2500.00","stopLoss": "1500.00","tpTriggerBy": "LastPrice","slTriggerBy": "LastPrice","triggerDirection": 0,"triggerBy": "UNKNOWN","lastPriceOnCreated": "","reduceOnly": false,"closeOnTrigger": false,"smpType": "None",		"smpGroup": 0,"smpOrderId": "","tpslMode": "Full","tpLimitPrice": "","slLimitPrice": "","placeType": "","createdTime": "1684738540559","updatedTime": "1684738540561"}]`
	var response []TradeOrder
	err := json.Unmarshal([]byte(data), &response)
	if err != nil {
		t.Fatal(err)
	}
	orders, err := e.ConstructOrderDetails(response, asset.Spot, currency.Pair{Base: currency.BTC, Quote: currency.USDT}, currency.Pairs{})
	if err != nil {
		t.Fatal(err)
	} else if len(orders) > 0 {
		t.Errorf("expected order with length 0, got %d", len(orders))
	}
	orders, err = e.ConstructOrderDetails(response, asset.Spot, currency.EMPTYPAIR, currency.Pairs{})
	if err != nil {
		t.Fatal(err)
	} else if len(orders) != 1 {
		t.Errorf("expected order with length 1, got %d", len(orders))
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  usdcMarginedTradablePair.Base.Item,
		Quote: usdcMarginedTradablePair.Quote.Item,
		Asset: asset.USDCMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  usdtMarginedTradablePair.Base.Item,
		Quote: usdtMarginedTradablePair.Quote.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  inverseTradablePair.Base.Item,
		Quote: inverseTradablePair.Quote.Item,
		Asset: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()

	is, err := e.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	assert.NoError(t, err)
	assert.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, inverseTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.CoinMarginedFutures, inverseTradablePair)

	is, err = e.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, usdtMarginedTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.USDTMarginedFutures, usdtMarginedTradablePair)

	is, err = e.IsPerpetualFutureCurrency(asset.USDCMarginedFutures, usdcMarginedTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.USDCMarginedFutures, usdcMarginedTradablePair)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

// TestGenerateSubscriptions exercises generateSubscriptions
func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	for _, s := range e.Features.Subscriptions {
		for _, a := range e.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: ""})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if isSymbolChannel(channelName(s)) {
				for i, p := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					switch s.Channel {
					case subscription.CandlesChannel:
						s.QualifiedChannel = fmt.Sprintf("%s.%.f.%s", channelName(s), s.Interval.Duration().Minutes(), p)
					case subscription.OrderbookChannel:
						s.QualifiedChannel = fmt.Sprintf("%s.%d.%s", channelName(s), s.Levels, p)
					default:
						s.QualifiedChannel = channelName(s) + "." + p.String()
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = channelName(s)
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestAuthSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.authSubscribe(t.Context(), &FixtureConnection{}, subscription.List{}))

	authsubs, err := e.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	require.Empty(t, authsubs, "generateAuthSubscriptions must not return subs")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	authsubs, err = e.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	require.NotEmpty(t, authsubs, "generateAuthSubscriptions must return subs")

	require.NoError(t, e.authSubscribe(t.Context(), &FixtureConnection{}, authsubs))
	require.NoError(t, e.authUnsubscribe(t.Context(), &FixtureConnection{}, authsubs))
}

func TestWebsocketAuthenticatePrivateConnection(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))

	err := e.WebsocketAuthenticatePrivateConnection(t.Context(), &FixtureConnection{})
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "dummy", Secret: "dummy"})
	err = e.WebsocketAuthenticatePrivateConnection(ctx, &FixtureConnection{})
	require.NoError(t, err)
	err = e.WebsocketAuthenticatePrivateConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"success":false,"ret_msg":"failed auth","conn_id":"5758770c-8152-4545-a84f-dae089e56499","req_id":"1","op":"subscribe"}`)})
	require.Error(t, err)
}

func TestWebsocketAuthenticateTradeConnection(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))

	err := e.WebsocketAuthenticateTradeConnection(t.Context(), &FixtureConnection{})
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "dummy", Secret: "dummy"})
	err = e.WebsocketAuthenticateTradeConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":0,"retMsg":"OK","op":"auth","connId":"d2a641kgcg7ab33b7mdg-4x6a"}`)})
	require.NoError(t, err)
	err = e.WebsocketAuthenticateTradeConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":10004,"retMsg":"Invalid sign","op":"auth","connId":"d2a63t6p49kk82nefh90-4ye8"}`)})
	require.Error(t, err)
}

func TestTransformSymbol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		symbol         string
		baseCoin       string
		contractType   string
		item           asset.Item
		expectedSymbol string
	}{
		{
			symbol:         "POPCATUSDT",
			baseCoin:       "POPCAT",
			item:           asset.Spot,
			expectedSymbol: "POPCAT_USDT",
		},
		{
			symbol:         "BTC26SEP25-300000-P",
			item:           asset.Options,
			baseCoin:       "BTC",
			expectedSymbol: "BTC-26SEP25-300000-P",
		},
		{
			symbol:         "1000000BABYDOGEUSDT",
			item:           asset.USDTMarginedFutures,
			baseCoin:       "1000000BABYDOGE",
			expectedSymbol: "1000000BABYDOGE-USDT",
		},
		{
			symbol:         "BTC-06DEC24",
			item:           asset.USDCMarginedFutures,
			expectedSymbol: "BTC-06DEC24",
			contractType:   "LinearFutures",
		},
		{
			symbol:         "1000PEPEPERP",
			baseCoin:       "1000PEPE",
			item:           asset.USDCMarginedFutures,
			expectedSymbol: "1000PEPE-PERP",
		},
		{
			symbol:         "BTCUSD",
			baseCoin:       "BTC",
			item:           asset.CoinMarginedFutures,
			expectedSymbol: "BTC_USD",
		},
		{
			symbol:         "nothingHappens",
			item:           asset.CrossMargin,
			expectedSymbol: "nothingHappens",
		},
	}
	for i := range tests {
		t.Run(tests[i].symbol+" "+tests[i].item.String(), func(t *testing.T) {
			t.Parallel()
			ii := InstrumentInfo{
				Symbol:       tests[i].symbol,
				ContractType: tests[i].contractType,
				BaseCoin:     tests[i].baseCoin,
			}
			assert.Equal(t, tests[i].expectedSymbol, ii.transformSymbol(tests[i].item), "expected symbols to match")
		})
	}
}

func TestMatchPairAssetFromResponse(t *testing.T) {
	t.Parallel()

	noDelim := currency.PairFormat{Uppercase: true}
	for _, tc := range []struct {
		pair          string
		category      string
		expectedAsset asset.Item
		expectedPair  currency.Pair
		err           error
	}{
		{pair: noDelim.Format(spotTradablePair), category: cSpot, expectedAsset: asset.Spot, expectedPair: spotTradablePair},
		{pair: noDelim.Format(usdtMarginedTradablePair), category: cLinear, expectedAsset: asset.USDTMarginedFutures, expectedPair: usdtMarginedTradablePair},
		{pair: noDelim.Format(usdcMarginedTradablePair), category: cLinear, expectedAsset: asset.USDCMarginedFutures, expectedPair: usdcMarginedTradablePair},
		{pair: noDelim.Format(inverseTradablePair), category: cInverse, expectedAsset: asset.CoinMarginedFutures, expectedPair: inverseTradablePair},
		{pair: optionsTradablePair.String(), category: cOption, expectedAsset: asset.Options, expectedPair: optionsTradablePair},
		{pair: optionsTradablePair.String(), category: "silly", err: errUnsupportedCategory, expectedAsset: 0},
		{pair: "bad pair", category: cSpot, err: currency.ErrPairNotFound},
	} {
		t.Run(fmt.Sprintf("pair: %s, category: %s", tc.pair, tc.category), func(t *testing.T) {
			t.Parallel()
			p, a, err := e.matchPairAssetFromResponse(tc.category, tc.pair)
			require.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.expectedAsset, a)
			assert.True(t, tc.expectedPair.Equal(p))
		})
	}
}

func TestHandleNoTopicWebsocketResponse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		operation string
		requestID string
		error     error
	}{
		{operation: "subscribe"},
		{operation: "unsubscribe"},
		{operation: "auth"},
		{operation: "auth", requestID: "noMatch", error: websocket.ErrSignatureNotMatched},
		{operation: "ping"},
		{operation: "pong"},
	} {
		t.Run(fmt.Sprintf("operation: %s, requestID: %s", tc.operation, tc.requestID), func(t *testing.T) {
			t.Parallel()
			err := e.handleNoTopicWebsocketResponse(t.Context(), &FixtureConnection{match: websocket.NewMatch()}, &WebsocketResponse{Operation: tc.operation, RequestID: tc.requestID}, nil)
			assert.ErrorIs(t, err, tc.error, "handleNoTopicWebsocketResponse should return expected error")
		})
	}
}

package bybit

import (
	"context"
	"errors"
	"fmt"
	"maps"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false

	skipAuthenticatedFunctionsForMockTesting = "skipping authenticated function for mock testing"
	skippingWebsocketFunctionsForMockTesting = "skipping websocket function for mock testing"
)

var (
	b = &Bybit{}

	spotTradablePair, usdcMarginedTradablePair, usdtMarginedTradablePair, inverseTradablePair, optionsTradablePair currency.Pair
)

func TestGetInstrumentInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetInstrumentInfo(t.Context(), "spot", "", "", "", "", 0)
	require.NoError(t, err)
	_, err = b.GetInstrumentInfo(t.Context(), "linear", "", "", "", "", 0)
	require.NoError(t, err)
	_, err = b.GetInstrumentInfo(t.Context(), "inverse", "", "", "", "", 0)
	require.NoError(t, err)
	_, err = b.GetInstrumentInfo(t.Context(), "option", "", "", "", "", 0)
	require.NoError(t, err)
	payload, err := b.GetInstrumentInfo(t.Context(), "linear", "10000000AIDOGEUSDT", "", "", "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, payload.List)
	require.NotZero(t, payload.List[0].LotSizeFilter.MinNotionalValue)
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	s := time.Now().Add(-time.Hour)
	e := time.Now()
	if mockTests {
		s = time.Unix(1691897100, 0).Round(kline.FiveMin.Duration())
		e = time.Unix(1691907100, 0).Round(kline.FiveMin.Duration())
	}
	for _, tc := range []struct {
		category   string
		pair       currency.Pair
		reqLimit   uint64
		expRespLen int
		expError   error
	}{
		{"spot", spotTradablePair, 100, 34, nil}, // TODO: Update expected limit when mock data is updated
		{"linear", usdtMarginedTradablePair, 5, 5, nil},
		{"linear", usdcMarginedTradablePair, 5, 5, nil},
		{"inverse", inverseTradablePair, 5, 5, nil},
		{"option", optionsTradablePair, 5, 5, errInvalidCategory},
	} {
		t.Run(fmt.Sprintf("%s-%s", tc.category, tc.pair), func(t *testing.T) {
			t.Parallel()
			r, err := b.GetKlines(t.Context(), tc.category, tc.pair.String(), kline.FiveMin, s, e, tc.reqLimit)
			if tc.expError != nil {
				require.ErrorIs(t, err, tc.expError)
				return
			}
			require.NoError(t, err)
			if mockTests {
				require.Equal(t, tc.expRespLen, len(r))

				switch tc.category {
				case "spot":
					assert.Equal(t, KlineItem{StartTime: types.Time(e), Open: 29393.99, High: 29399.76, Low: 29393.98, Close: 29399.76, TradeVolume: 1.168988, Turnover: 34363.5346739}, r[0])
				case "linear":
					if tc.pair == usdtMarginedTradablePair {
						assert.Equal(t, KlineItem{StartTime: types.Time(e), Open: 0.0003, High: 0.0003, Low: 0.0002995, Close: 0.0003, TradeVolume: 55102100, Turnover: 16506.2427}, r[0])
						return
					}
					assert.Equal(t, KlineItem{StartTime: types.Time(e), Open: 239.7, High: 239.7, Low: 239.7, Close: 239.7}, r[0])
				case "inverse":
					assert.Equal(t, KlineItem{StartTime: types.Time(e), Open: 0.2908, High: 0.2912, Low: 0.2908, Close: 0.2912, TradeVolume: 5131, Turnover: 17626.40000346}, r[0])
				}
			} else {
				assert.NotEmpty(t, r)
			}
		})
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	s := time.Now().Add(-time.Hour * 1)
	e := time.Now()
	if mockTests {
		s = time.UnixMilli(1693077167971)
		e = time.UnixMilli(1693080767971)
	}
	_, err := b.GetMarkPriceKline(t.Context(), "linear", usdtMarginedTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetMarkPriceKline(t.Context(), "linear", usdcMarginedTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetMarkPriceKline(t.Context(), "inverse", inverseTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetMarkPriceKline(t.Context(), "option", optionsTradablePair.String(), kline.FiveMin, s, e, 5)
	if err == nil {
		t.Fatalf("expected 'params error: Category is invalid', but found nil")
	}
}

func TestGetIndexPriceKline(t *testing.T) {
	t.Parallel()
	s := time.Now().Add(-time.Hour * 1)
	e := time.Now()
	if mockTests {
		s = time.UnixMilli(1693077165571)
		e = time.UnixMilli(1693080765571)
	}
	_, err := b.GetIndexPriceKline(t.Context(), "linear", usdtMarginedTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetIndexPriceKline(t.Context(), "linear", usdcMarginedTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetIndexPriceKline(t.Context(), "inverse", inverseTradablePair.String(), kline.FiveMin, s, e, 5)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(t.Context(), "spot", spotTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(t.Context(), "linear", usdtMarginedTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(t.Context(), "linear", usdcMarginedTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(t.Context(), "inverse", inverseTradablePair.String(), 100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetOrderBook(t.Context(), "option", optionsTradablePair.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := b.GetRiskLimit(t.Context(), "linear", usdtMarginedTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(t.Context(), "linear", usdcMarginedTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(t.Context(), "inverse", inverseTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetRiskLimit(t.Context(), "option", optionsTradablePair.String())
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetRiskLimit(t.Context(), "spot", spotTradablePair.String())
	assert.ErrorIs(t, err, errInvalidCategory)
}

// test cases for Wrapper
func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := b.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(t.Context(), inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateTicker(t.Context(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	var err error
	_, err = b.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}

	_, err = b.UpdateOrderbook(t.Context(), inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.UpdateOrderbook(t.Context(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	orderSubmission := &order.Submit{
		Exchange:      b.GetName(),
		Pair:          spotTradablePair,
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         1,
		Amount:        1,
		ClientOrderID: "1234",
		AssetType:     asset.Spot,
	}
	_, err := b.SubmitOrder(t.Context(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
	_, err = b.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      b.GetName(),
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifyOrder(t.Context(), &order.Modify{
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
	_, err := b.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandles(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandles(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures, kline.OneDay, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandles(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, kline.OneHour, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandles(t.Context(), optionsTradablePair, asset.Options, kline.OneHour, start, end)
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
	_, err := b.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, kline.OneHour, startTime, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, kline.OneDay, time.UnixMilli(1692889428738), time.UnixMilli(1693145028738))
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricCandlesExtended(t.Context(), optionsTradablePair, asset.Options, kline.FiveMin, startTime, end)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.Spot,
		Pair:      spotTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.USDTMarginedFutures,
		Pair:      usdtMarginedTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}

	err = b.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  b.Name,
		AssetType: asset.CoinMarginedFutures,
		Pair:      inverseTradablePair,
		OrderID:   "1234",
	})
	if err != nil {
		t.Error(err)
	}
	err = b.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  b.Name,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: spotTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(t.Context(), &order.Cancel{Exchange: b.Name, AssetType: asset.USDTMarginedFutures, Pair: usdtMarginedTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(t.Context(), &order.Cancel{Exchange: b.Name, AssetType: asset.CoinMarginedFutures, Pair: inverseTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(t.Context(), &order.Cancel{Exchange: b.Name, AssetType: asset.Options, Pair: optionsTradablePair})
	if err != nil {
		t.Error(err)
	}
	_, err = b.CancelAllOrders(t.Context(), &order.Cancel{Exchange: b.Name, AssetType: asset.Futures, Pair: spotTradablePair})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOrderInfo(t.Context(),
		"12234", spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo(t.Context(),
		"12234", usdtMarginedTradablePair, asset.USDTMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo(t.Context(),
		"12234", inverseTradablePair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrderInfo(t.Context(),
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	getOrdersRequestSpot := order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Type:      order.AnyType,
	}
	_, err := b.GetActiveOrders(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestLinear := order.MultiOrderRequest{Pairs: currency.Pairs{usdtMarginedTradablePair}, AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(t.Context(), &getOrdersRequestLinear)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestInverse := order.MultiOrderRequest{Pairs: currency.Pairs{inverseTradablePair}, AssetType: asset.CoinMarginedFutures, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(t.Context(), &getOrdersRequestInverse)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestFutures := order.MultiOrderRequest{Pairs: currency.Pairs{optionsTradablePair}, AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(t.Context(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
	pairs, err := currency.NewPairsFromStrings([]string{"BTC_USDT", "BTC_ETH", "BTC_USDC"})
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequestSpot = order.MultiOrderRequest{Pairs: pairs, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
	_, err = b.GetActiveOrders(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	getOrdersRequestSpot := order.MultiOrderRequest{
		Pairs:     currency.Pairs{spotTradablePair},
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := b.GetOrderHistory(t.Context(), &getOrdersRequestSpot)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestUMF := order.MultiOrderRequest{
		Pairs:     currency.Pairs{usdtMarginedTradablePair},
		AssetType: asset.USDTMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(t.Context(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestUMF.Pairs = currency.Pairs{usdcMarginedTradablePair}
	getOrdersRequestUMF.AssetType = asset.USDCMarginedFutures
	_, err = b.GetOrderHistory(t.Context(), &getOrdersRequestUMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestCMF := order.MultiOrderRequest{
		Pairs:     currency.Pairs{inverseTradablePair},
		AssetType: asset.CoinMarginedFutures,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(t.Context(), &getOrdersRequestCMF)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequestFutures := order.MultiOrderRequest{
		Pairs:     currency.Pairs{optionsTradablePair},
		AssetType: asset.Options,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err = b.GetOrderHistory(t.Context(), &getOrdersRequestFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetDepositAddress(t.Context(), currency.USDT, "", currency.ETH.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: "Bybit",
		Amount:   10,
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
	err := b.UpdateTickers(ctx, asset.Spot)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Spot, err)
	}
	err = b.UpdateTickers(ctx, asset.USDTMarginedFutures)
	if err != nil {
		t.Fatalf("%v %v\n", asset.USDTMarginedFutures, err)
	}
	err = b.UpdateTickers(ctx, asset.CoinMarginedFutures)
	if err != nil {
		t.Fatalf("%v %v\n", asset.CoinMarginedFutures, err)
	}
	err = b.UpdateTickers(ctx, asset.Options)
	if err != nil {
		t.Fatalf("%v %v\n", asset.Options, err)
	}
}

func TestGetTickersV5(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickers(t.Context(), "bruh", "", "", time.Time{})
	require.ErrorIs(t, err, errInvalidCategory)
	_, err = b.GetTickers(t.Context(), "option", "BTC-26NOV24-92000-C", "", time.Time{})
	require.NoError(t, err)
	_, err = b.GetTickers(t.Context(), "spot", "", "", time.Time{})
	require.NoError(t, err)
	_, err = b.GetTickers(t.Context(), "inverse", "", "", time.Time{})
	require.NoError(t, err)
	_, err = b.GetTickers(t.Context(), "linear", "", "", time.Time{})
	require.NoError(t, err)
	_, err = b.GetTickers(t.Context(), "option", "", "BTC", time.Time{})
	require.NoError(t, err)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingRateHistory(t.Context(), "bruh", "", time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetFundingRateHistory(t.Context(), "spot", spotTradablePair.String(), time.Time{}, time.Time{}, 100)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetFundingRateHistory(t.Context(), "linear", usdtMarginedTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRateHistory(t.Context(), "linear", usdcMarginedTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRateHistory(t.Context(), "inverse", inverseTradablePair.String(), time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetFundingRateHistory(t.Context(), "option", optionsTradablePair.String(), time.Time{}, time.Time{}, 100)
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestGetPublicTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetPublicTradingHistory(t.Context(), "spot", spotTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(t.Context(), "linear", usdtMarginedTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(t.Context(), "linear", usdcMarginedTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(t.Context(), "inverse", inverseTradablePair.String(), "", "", 30)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPublicTradingHistory(t.Context(), "option", optionsTradablePair.String(), "BTC", "", 30)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterestData(t.Context(), "spot", spotTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetOpenInterestData(t.Context(), "linear", usdtMarginedTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterestData(t.Context(), "linear", usdcMarginedTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterestData(t.Context(), "inverse", inverseTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOpenInterestData(t.Context(), "option", optionsTradablePair.String(), "5min", time.Time{}, time.Time{}, 0, "")
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
	_, err := b.GetHistoricalVolatility(t.Context(), "option", "", 123, start, end)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricalVolatility(t.Context(), "spot", "", 123, start, end)
	assert.ErrorIs(t, err, errInvalidCategory)
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	_, err := b.GetInsurance(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetDeliveryPrice(t.Context(), "spot", spotTradablePair.String(), "", "", 200)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetDeliveryPrice(t.Context(), "linear", "", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetDeliveryPrice(t.Context(), "inverse", "", "", "", 200)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetDeliveryPrice(t.Context(), "option", "", "BTC", "", 200)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = b.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	assert.NoError(t, err)
	err = b.UpdateOrderExecutionLimits(t.Context(), asset.USDCMarginedFutures)
	assert.NoError(t, err)
	err = b.UpdateOrderExecutionLimits(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)

	err = b.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)
	availablePairs, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal("Bybit GetAvailablePairs() error", err)
	}
	for x := range availablePairs {
		var limits order.MinMaxLevel
		limits, err = b.GetOrderExecutionLimits(asset.Spot, availablePairs[x])
		require.NoError(t, err)
		if limits == (order.MinMaxLevel{}) {
			t.Fatal("Bybit GetOrderExecutionLimits() error cannot be nil")
		}
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	ctx := t.Context()
	_, err := b.PlaceOrder(ctx, nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "my-category",
	})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
		Symbol:   currency.Pair{Delimiter: "", Base: currency.BTC, Quote: currency.USDT},
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category: "spot",
		Symbol:   spotTradablePair,
		Side:     "buy",
	})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category:  "spot",
		Symbol:    spotTradablePair,
		Side:      "buy",
		OrderType: "limit",
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.PlaceOrder(ctx, &PlaceOrderParams{
		Category:         "spot",
		Symbol:           spotTradablePair,
		Side:             "buy",
		OrderType:        "limit",
		OrderQuantity:    1,
		TriggerDirection: 3,
	})
	require.ErrorIs(t, err, errInvalidTriggerDirection)

	_, err = b.PlaceOrder(t.Context(), &PlaceOrderParams{
		Category:         "spot",
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
	arg := &PlaceOrderParams{Category: "spot", Symbol: spotTradablePair, Side: "Buy", OrderType: "Limit", OrderQuantity: 0.1, Price: 15600, TimeInForce: "PostOnly", OrderLinkID: "spot-test-01", IsLeverage: 0, OrderFilter: "Order"}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// Spot TP/SL order
	arg = &PlaceOrderParams{
		Category: "spot",
		Symbol:   spotTradablePair,
		Side:     "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TriggerPrice: 15000,
		TimeInForce: "GTC", OrderLinkID: "spot-test-02", IsLeverage: 0, OrderFilter: "tpslOrder",
	}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// Spot margin normal order (UTA)
	arg = &PlaceOrderParams{
		Category: "spot", Symbol: spotTradablePair, Side: "Buy", OrderType: "Limit",
		OrderQuantity: 0.1, Price: 15600, TimeInForce: "IOC", OrderLinkID: "spot-test-limit", IsLeverage: 1, OrderFilter: "Order",
	}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	arg = &PlaceOrderParams{
		Category: "spot",
		Symbol:   spotTradablePair,
		Side:     "Buy", OrderType: "Market", OrderQuantity: 200,
		TimeInForce: "IOC", OrderLinkID: "spot-test-04",
		IsLeverage: 0, OrderFilter: "Order",
	}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// USDT Perp open long position (one-way mode)
	arg = &PlaceOrderParams{
		Category: "linear",
		Symbol:   usdcMarginedTradablePair, Side: "Buy", OrderType: "Limit", OrderQuantity: 1, Price: 25000, TimeInForce: "GTC", PositionIdx: 0, OrderLinkID: "usdt-test-01", ReduceOnly: false, TakeProfitPrice: 28000, StopLossPrice: 20000, TpslMode: "Partial", TpOrderType: "Limit", SlOrderType: "Limit", TpLimitPrice: 27500, SlLimitPrice: 20500,
	}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
	// USDT Perp close long position (one-way mode)
	arg = &PlaceOrderParams{
		Category: "linear", Symbol: usdtMarginedTradablePair, Side: "Sell",
		OrderType: "Limit", OrderQuantity: 1, Price: 3000, TimeInForce: "GTC", PositionIdx: 0, OrderLinkID: "usdt-test-02", ReduceOnly: true,
	}
	_, err = b.PlaceOrder(t.Context(), arg)
	if err != nil {
		t.Error(err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.AmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.AmendOrder(t.Context(), &AmendOrderParams{})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	_, err = b.AmendOrder(t.Context(), &AmendOrderParams{
		OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa",
	})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.AmendOrder(t.Context(), &AmendOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "mycat",
	})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.AmendOrder(t.Context(), &AmendOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.AmendOrder(t.Context(), &AmendOrderParams{
		OrderID:         "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category:        cSpot,
		Symbol:          spotTradablePair,
		TriggerPrice:    1145,
		OrderQuantity:   0.15,
		Price:           1050,
		TakeProfitPrice: 0,
		StopLossPrice:   0,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelTradeOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CancelTradeOrder(t.Context(), &CancelOrderParams{})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired)

	_, err = b.CancelTradeOrder(t.Context(), &CancelOrderParams{
		OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa",
	})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.CancelTradeOrder(t.Context(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "mycat",
	})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.CancelTradeOrder(t.Context(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.CancelTradeOrder(t.Context(), &CancelOrderParams{
		OrderID:  "c6f055d9-7f21-4079-913d-e6523a9cfffa",
		Category: "option",
		Symbol:   optionsTradablePair,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetOpenOrders(t.Context(), "", "", "", "", "", "", "", "", 0, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetOpenOrders(t.Context(), "spot", "", "", "", "", "", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelAllTradeOrders(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CancelAllTradeOrders(t.Context(), &CancelAllOrdersParam{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.CancelAllTradeOrders(t.Context(), &CancelAllOrdersParam{Category: "option"})
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetTradeOrderHistory(t.Context(), "", "", "", "", "", "", "", "", "", start, end, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetTradeOrderHistory(t.Context(), "spot", spotTradablePair.String(), "", "", "BTC", "", "StopOrder", "", "", start, end, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.PlaceBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: "linear",
	})
	require.ErrorIs(t, err, errNoOrderPassed)

	_, err = b.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: "option",
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
	_, err = b.PlaceBatchOrder(t.Context(), &PlaceBatchOrderParam{
		Category: "linear",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.BatchAmendOrder(t.Context(), "linear", nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.BatchAmendOrder(t.Context(), "", []BatchAmendOrderParamItem{
		{
			Symbol:                 optionsTradablePair,
			OrderImpliedVolatility: "6.8",
			OrderID:                "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
		},
	})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.BatchAmendOrder(t.Context(), "option", []BatchAmendOrderParamItem{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CancelBatchOrder(t.Context(), &CancelBatchOrder{})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.CancelBatchOrder(t.Context(), &CancelBatchOrder{Category: cOption})
	require.ErrorIs(t, err, errNoOrderPassed)

	_, err = b.CancelBatchOrder(t.Context(), &CancelBatchOrder{
		Category: "option",
		Request: []CancelOrderParams{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetBorrowQuota(t.Context(), "", "BTCUSDT", "Buy")
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetBorrowQuota(t.Context(), "spot", "", "Buy")
	require.ErrorIs(t, err, errSymbolMissing)

	_, err = b.GetBorrowQuota(t.Context(), "spot", spotTradablePair.String(), "")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = b.GetBorrowQuota(t.Context(), "spot", spotTradablePair.String(), "Buy")
	if err != nil {
		t.Error(err)
	}
}

func TestSetDisconnectCancelAll(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetDisconnectCancelAll(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = b.SetDisconnectCancelAll(t.Context(), &SetDCPParams{TimeWindow: 300})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPositionInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetPositionInfo(t.Context(), "", "", "", "", "", 20)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetPositionInfo(t.Context(), "spot", "", "", "", "", 20)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPositionInfo(t.Context(), "linear", "BTCUSDT", "", "", "", 20)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetPositionInfo(t.Context(), "option", "BTC-26NOV24-92000-C", "BTC", "", "", 20)
	if err != nil {
		t.Error(err)
	}
}

func TestSetLeverageLevel(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetLeverageLevel(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	err = b.SetLeverageLevel(t.Context(), &SetLeverageParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = b.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: "spot"})
	require.ErrorIs(t, err, errInvalidCategory)

	err = b.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: "linear"})
	require.ErrorIs(t, err, errSymbolMissing)

	err = b.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: "linear", Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errInvalidLeverage)

	err = b.SetLeverageLevel(t.Context(), &SetLeverageParams{Category: "linear", Symbol: "BTCUSDT", SellLeverage: 3, BuyLeverage: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchTradeMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SwitchTradeMode(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: "spot"})
	require.ErrorIs(t, err, errInvalidCategory)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: "linear"})
	require.ErrorIs(t, err, errSymbolMissing)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: "linear", Symbol: usdtMarginedTradablePair.String()})
	require.ErrorIs(t, err, errInvalidLeverage)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: "linear", Symbol: usdcMarginedTradablePair.String(), SellLeverage: 3, BuyLeverage: 3, TradeMode: 2})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	err = b.SwitchTradeMode(t.Context(), &SwitchTradeModeParams{Category: "linear", Symbol: usdtMarginedTradablePair.String(), SellLeverage: 3, BuyLeverage: 3, TradeMode: 1})
	if err != nil {
		t.Error(err)
	}
}

func TestSetTakeProfitStopLossMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetTakeProfitStopLossMode(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{
		Category: "spot",
	})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: "spot"})
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: "linear"})
	require.ErrorIs(t, err, errSymbolMissing)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: "linear", Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errTakeProfitOrStopLossModeMissing)

	_, err = b.SetTakeProfitStopLossMode(t.Context(), &TPSLModeParams{Category: "linear", Symbol: "BTCUSDT", TpslMode: "Partial"})
	if err != nil {
		t.Error(err)
	}
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SwitchPositionMode(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = b.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{})
	require.ErrorIs(t, err, errCategoryNotSet)

	err = b.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{Category: "linear"})
	require.ErrorIs(t, err, errEitherSymbolOrCoinRequired)

	err = b.SwitchPositionMode(t.Context(), &SwitchPositionModeParams{Category: "linear", Symbol: usdtMarginedTradablePair, PositionMode: 3})
	if err != nil {
		t.Error(err)
	}
}

func TestSetRiskLimit(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetRiskLimit(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.SetRiskLimit(t.Context(), &SetRiskLimitParam{})
	assert.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.SetRiskLimit(t.Context(), &SetRiskLimitParam{Category: "linear", PositionMode: -2})
	assert.ErrorIs(t, err, errInvalidPositionMode)

	_, err = b.SetRiskLimit(t.Context(), &SetRiskLimitParam{Category: "linear"})
	assert.ErrorIs(t, err, errSymbolMissing)

	_, err = b.SetRiskLimit(t.Context(), &SetRiskLimitParam{
		Category:     "linear",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetTradingStop(t.Context(), &TradingStopParams{})
	assert.ErrorIs(t, err, errCategoryNotSet)

	err = b.SetTradingStop(t.Context(), &TradingStopParams{Category: "spot"})
	assert.ErrorIs(t, err, errInvalidCategory)

	err = b.SetTradingStop(t.Context(), &TradingStopParams{
		Category:                 "linear",
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
	err = b.SetTradingStop(t.Context(), &TradingStopParams{
		Category:                 "linear",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetAutoAddMargin(t.Context(), &AutoAddMarginParam{
		Category:      "inverse",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.AddOrReduceMargin(t.Context(), &AddOrReduceMarginParam{
		Category:      "inverse",
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetExecution(t.Context(), "spot", "", "", "", "", "Trade", "tpslOrder", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetClosedPnL(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetClosedPnL(t.Context(), "spot", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetClosedPnL(t.Context(), "linear", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfirmNewRiskLimit(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.ConfirmNewRiskLimit(t.Context(), "linear", "BTCUSDT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeOrderHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeOrderHistory(t.Context(), "", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetPreUpgradeOrderHistory(t.Context(), "option", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errBaseNotSet)

	_, err = b.GetPreUpgradeOrderHistory(t.Context(), "linear", "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTradeHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeTradeHistory(t.Context(), "", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errCategoryNotSet)

	_, err = b.GetPreUpgradeTradeHistory(t.Context(), "option", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPreUpgradeTradeHistory(t.Context(), "linear", "", "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeClosedPnL(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeClosedPnL(t.Context(), "option", "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPreUpgradeClosedPnL(t.Context(), "linear", "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeTransactionLog(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeTransactionLog(t.Context(), "option", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPreUpgradeTransactionLog(t.Context(), "linear", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeOptionDeliveryRecord(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeOptionDeliveryRecord(t.Context(), "linear", "", "", time.Time{}, 0)
	assert.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPreUpgradeOptionDeliveryRecord(t.Context(), "option", "", "", time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPreUpgradeUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPreUpgradeUSDCSessionSettlement(t.Context(), "option", "", "", 10)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetPreUpgradeUSDCSessionSettlement(t.Context(), "linear", "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}

	r, err := b.GetWalletBalance(t.Context(), "UNIFIED", "")
	require.NoError(t, err, "GetWalletBalance must not error")
	require.NotNil(t, r, "GetWalletBalance must return a result")

	if mockTests {
		require.Len(t, r.List, 1, "GetWalletBalance must return a single list result")
		assert.Equal(t, types.Number(0.1997), r.List[0].AccountIMRate, "AccountIMRate should match")
		assert.Equal(t, types.Number(0.4996), r.List[0].AccountLTV, "AccountLTV should match")
		assert.Equal(t, types.Number(0.0399), r.List[0].AccountMMRate, "AccountMMRate should match")
		assert.Equal(t, "UNIFIED", r.List[0].AccountType, "AccountType should match")
		assert.Equal(t, types.Number(24616.49915805), r.List[0].TotalAvailableBalance, "TotalAvailableBalance should match")
		assert.Equal(t, types.Number(41445.9203332), r.List[0].TotalEquity, "TotalEquity should match")
		assert.Equal(t, types.Number(6144.46796478), r.List[0].TotalInitialMargin, "TotalInitialMargin should match")
		assert.Equal(t, types.Number(1228.89359295), r.List[0].TotalMaintenanceMargin, "TotalMaintenanceMargin should match")
		assert.Equal(t, types.Number(30760.96712284), r.List[0].TotalMarginBalance, "TotalMarginBalance should match")
		assert.Equal(t, types.Number(0.0), r.List[0].TotalPerpUPL, "TotalPerpUPL should match")
		assert.Equal(t, types.Number(30760.96712284), r.List[0].TotalWalletBalance, "TotalWalletBalance should match")
		require.Len(t, r.List[0].Coin, 3, "GetWalletBalance must return 3 coins")

		for x := range r.List[0].Coin {
			switch x {
			case 0:
				assert.Equal(t, types.Number(0.21976631), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should match")
				assert.Equal(t, types.Number(30723.630216383711792744), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should match")
				assert.Equal(t, currency.USDC, r.List[0].Coin[x].Coin, "Coin should match")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should match")
				assert.Equal(t, types.Number(-30723.63021638), r.List[0].Coin[x].Equity, "Equity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should match")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should match")
				assert.Equal(t, types.Number(-30722.33982391), r.List[0].Coin[x].USDValue, "USDValue should match")
				assert.Equal(t, types.Number(-30723.63021638), r.List[0].Coin[x].WalletBalance, "WalletBalance should match")
			case 1:
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should match")
				assert.Equal(t, types.Number(1005.79191187), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should match")
				assert.Equal(t, currency.AVAX, r.List[0].Coin[x].Coin, "Coin should match")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should match")
				assert.Equal(t, types.Number(2473.9), r.List[0].Coin[x].Equity, "Equity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should match")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should match")
				assert.Equal(t, types.Number(71233.0214024), r.List[0].Coin[x].USDValue, "USDValue should match")
				assert.Equal(t, types.Number(2473.9), r.List[0].Coin[x].WalletBalance, "WalletBalance should match")
			case 2:
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AccruedInterest, "AccruedInterest should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].AvailableToBorrow, "AvailableToBorrow should match")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].AvailableToWithdraw, "AvailableToWithdraw should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Bonus, "Bonus should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].BorrowAmount, "BorrowAmount should match")
				assert.Equal(t, currency.USDT, r.List[0].Coin[x].Coin, "Coin should match")
				assert.True(t, r.List[0].Coin[x].CollateralSwitch, "CollateralSwitch should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].CumulativeRealisedPNL, "CumulativeRealisedPNL should match")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].Equity, "Equity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].Locked, "Locked should match")
				assert.True(t, r.List[0].Coin[x].MarginCollateral, "MarginCollateral should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].SpotHedgingQuantity, "SpotHedgingQuantity should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalOrderIM, "TotalOrderIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionIM, "TotalPositionIM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].TotalPositionMM, "TotalPositionMM should match")
				assert.Equal(t, types.Number(0), r.List[0].Coin[x].UnrealisedPNL, "UnrealisedPNL should match")
				assert.Equal(t, types.Number(935.23875471), r.List[0].Coin[x].USDValue, "USDValue should match")
				assert.Equal(t, types.Number(935.1415), r.List[0].Coin[x].WalletBalance, "WalletBalance should match")
			}
		}
	}
}

func TestUpgradeToUnifiedAccount(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.UpgradeToUnifiedAccount(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetBorrowHistory(t.Context(), "BTC", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetCollateralCoin(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetCollateralCoin(t.Context(), currency.BTC, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetCollateralInfo(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinGreeks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetCoinGreeks(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetFeeRate(t.Context(), "something", "", "BTC")
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetFeeRate(t.Context(), "linear", "", "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetAccountInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetTransactionLog(t.Context(), "option", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTransactionLog(t.Context(), "linear", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMarginMode(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetMarginMode(t.Context(), "PORTFOLIO_MARGIN")
	if err != nil {
		t.Error(err)
	}
}

func TestSetSpotHedging(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetSpotHedging(t.Context(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountALLAPIKeys(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubAccountAllAPIKeys(t.Context(), "", "", 10)
	assert.ErrorIs(t, err, errMemberIDRequired)

	_, err = b.GetSubAccountAllAPIKeys(t.Context(), "1234", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.SetMMP(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	err = b.SetMMP(t.Context(), &MMPRequestParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.ResetMMP(t.Context(), "USDT")
	require.ErrorIs(t, err, errNilArgument)

	err = b.ResetMMP(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMMPState(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetMMPState(t.Context(), "BTC")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinExchangeRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetCoinExchangeRecords(t.Context(), "", "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDeliveryRecord(t *testing.T) {
	t.Parallel()
	expiryTime := time.Now().Add(time.Hour * 40)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	} else {
		expiryTime = time.UnixMilli(1700216290093)
	}
	_, err := b.GetDeliveryRecord(t.Context(), "spot", "", "", expiryTime, 20)
	assert.ErrorIs(t, err, errInvalidCategory)
	_, err = b.GetDeliveryRecord(t.Context(), "linear", "", "", expiryTime, 20)
	assert.NoError(t, err, "GetDeliveryRecord should not error for linear category")
}

func TestGetUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetUSDCSessionSettlement(t.Context(), "option", "", "", 10)
	require.ErrorIs(t, err, errInvalidCategory)

	_, err = b.GetUSDCSessionSettlement(t.Context(), "linear", "", "", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetAssetInfo(t.Context(), "", "BTC")
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.GetAssetInfo(t.Context(), "SPOT", "BTC")
	assert.NoError(t, err, "GetAssetInfo should not error for SPOT account type")
}

func TestGetAllCoinBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetAllCoinBalance(t.Context(), "", "", "", 0)
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.GetAllCoinBalance(t.Context(), "FUND", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSingleCoinBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetSingleCoinBalance(t.Context(), "", "", "", 0, 0)
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.GetSingleCoinBalance(t.Context(), "SPOT", currency.BTC.String(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTransferableCoin(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetTransferableCoin(t.Context(), "SPOT", "OPTION")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateInternalTransfer(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CreateInternalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{})
	require.ErrorIs(t, err, errMissingTransferID)

	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{TransferID: transferID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateInternalTransfer(t.Context(), &TransferParams{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	} else {
		transferIDString = "018bd458-dba0-728b-b5b6-ecd5bd296528"
	}
	_, err = b.GetInternalTransferRecords(t.Context(), transferIDString, currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubUID(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetSubUID(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestEnableUniversalTransferForSubUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.EnableUniversalTransferForSubUID(t.Context())
	require.ErrorIs(t, err, errMembersIDsNotSet)

	transferID1, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	transferID2, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	err = b.EnableUniversalTransferForSubUID(t.Context(), transferID1.String(), transferID2.String())
	if err != nil {
		t.Error(err)
	}
}

func TestCreateUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := b.CreateUniversalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{})
	require.ErrorIs(t, err, errMissingTransferID)

	transferID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{TransferID: transferID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC,
		Amount:     123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456, FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMissingAccountType)

	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
		TransferID: transferID,
		Coin:       currency.BTC, Amount: 123.456,
		ToAccountType:   "CONTRACT",
		FromAccountType: "UNIFIED",
	})
	require.ErrorIs(t, err, errMemberIDRequired)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateUniversalTransfer(t.Context(), &TransferParams{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
		transferID, err := uuid.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		transferIDString = transferID.String()
	} else {
		transferIDString = "018bd461-cb9c-75ce-94d4-0d3f4d84c339"
	}
	_, err := b.GetUniversalTransferRecords(t.Context(), transferIDString, currency.BTC.String(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllowedDepositCoinInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetAllowedDepositCoinInfo(t.Context(), "BTC", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestSetDepositAccount(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.SetDepositAccount(t.Context(), "FUND")
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetDepositRecords(t.Context(), "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubDepositRecords(t.Context(), "12345", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestInternalDepositRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetInternalDepositRecordsOffChain(t.Context(), currency.ETH.String(), "", time.Time{}, time.Time{}, 8)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMasterDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetMasterDepositAddress(t.Context(), currency.LTC, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubDepositAddress(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubDepositAddress(t.Context(), currency.LTC, "LTC", "12345")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetWithdrawalRecords(t.Context(), currency.LTC, "", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawableAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	_, err := b.GetWithdrawableAmount(t.Context(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.WithdrawCurrency(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.WithdrawCurrency(t.Context(), &WithdrawalParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.BTC})
	require.ErrorIs(t, err, errMissingChainInformation)

	_, err = b.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC"})
	require.ErrorIs(t, err, errMissingAddressInfo)

	_, err = b.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234"})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.WithdrawCurrency(t.Context(), &WithdrawalParam{Coin: currency.LTC, Chain: "LTC", Address: "234234234", Amount: 123})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID)

	_, err = b.CancelWithdrawal(t.Context(), "12314")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewSubUserID(t *testing.T) {
	t.Parallel()
	_, err := b.CreateNewSubUserID(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CreateNewSubUserID(t.Context(), &CreateSubUserParams{MemberType: 1, Switch: 1, Note: "test"})
	require.ErrorIs(t, err, errMissingUsername)

	_, err = b.CreateNewSubUserID(t.Context(), &CreateSubUserParams{Username: "Sami", Switch: 1, Note: "test"})
	require.ErrorIs(t, err, errInvalidMemberType)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateNewSubUserID(t.Context(), &CreateSubUserParams{Username: "sami", MemberType: 1, Switch: 1, Note: "test"})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubUIDAPIKey(t *testing.T) {
	t.Parallel()
	_, err := b.CreateSubUIDAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.CreateSubUIDAPIKey(t.Context(), &SubUIDAPIKeyParam{})
	require.ErrorIs(t, err, errMissingUserID)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err = b.CreateSubUIDAPIKey(t.Context(), &SubUIDAPIKeyParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetSubUIDList(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestFreezeSubUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.FreezeSubUID(t.Context(), "1234", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyInformation(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAPIKeyInformation(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUIDWalletType(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUIDWalletType(t.Context(), "234234")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyMasterAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifyMasterAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.ModifyMasterAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifySubAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = b.ModifySubAPIKey(t.Context(), &SubUIDAPIKeyUpdateParam{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.DeleteSubUID(t.Context(), "")
	assert.ErrorIs(t, err, errMemberIDRequired)

	err = b.DeleteSubUID(t.Context(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteMasterAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DeleteMasterAPIKey(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteSubAPIKey(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	err := b.DeleteSubAccountAPIKey(t.Context(), "12434")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAffiliateUserInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAffiliateUserInfo(t.Context(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeverageTokenInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLeverageTokenInfo(t.Context(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestGetLeveragedTokenMarket(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLeveragedTokenMarket(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.GetLeveragedTokenMarket(t.Context(), currency.NewCode("BTC3L"))
	if err != nil {
		t.Error(err)
	}
}

func TestPurchaseLeverageToken(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.PurchaseLeverageToken(t.Context(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestRedeemLeverageToken(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.RedeemLeverageToken(t.Context(), currency.BTC3L, 100, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPurchaseAndRedemptionRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetPurchaseAndRedemptionRecords(t.Context(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestToggleMarginTrade(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ToggleMarginTrade(t.Context(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestSetSpotMarginTradeLeverage(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	err := b.SetSpotMarginTradeLeverage(t.Context(), 3)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetVIPMarginData(t *testing.T) {
	t.Parallel()
	_, err := b.GetVIPMarginData(t.Context(), "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowableCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetBorrowableCoinInfo(t.Context(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInterestAndQuota(t *testing.T) {
	t.Parallel()
	_, err := b.GetInterestAndQuota(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err = b.GetInterestAndQuota(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetLoanAccountInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLoanAccountInfo(t.Context())
	assert.NoError(t, err)
}

func TestBorrow(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.Borrow(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.Borrow(t.Context(), &LendArgument{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.Borrow(t.Context(), &LendArgument{Coin: currency.BTC})
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.Borrow(t.Context(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}

func TestRepay(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.Repay(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.Repay(t.Context(), &LendArgument{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.Repay(t.Context(), &LendArgument{Coin: currency.BTC})
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.Repay(t.Context(), &LendArgument{Coin: currency.BTC, AmountToBorrow: 0.1})
	if err != nil {
		t.Error(err)
	}
}

func TestGetBorrowOrderDetail(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBorrowOrderDetail(t.Context(), time.Time{}, time.Time{}, currency.BTC, 0, 0)
	assert.NoError(t, err)
}

func TestGetRepaymentOrderDetail(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetRepaymentOrderDetail(t.Context(), time.Time{}, time.Time{}, currency.BTC, 0)
	assert.NoError(t, err)
}

func TestToggleMarginTradeNormal(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.ToggleMarginTradeNormal(t.Context(), true)
	assert.NoError(t, err)
}

func TestGetProductInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetProductInfo(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLengingMarginCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetInstitutionalLengingMarginCoinInfo(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstitutionalLoanOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInstitutionalLoanOrders(t.Context(), "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetInstitutionalRepayOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInstitutionalRepayOrders(t.Context(), time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetLTV(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetLTV(t.Context())
	assert.NoError(t, err)
}

func TestBindOrUnbindUID(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.BindOrUnbindUID(t.Context(), "12234", "0")
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingCoinInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingCoinInfo(t.Context(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestC2CDepositFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.C2CDepositFunds(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC})
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.C2CDepositFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestC2CRedeemFunds(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.C2CRedeemFunds(t.Context(), nil)
	assert.ErrorIs(t, err, errNilArgument)

	_, err = b.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = b.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC})
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = b.C2CRedeemFunds(t.Context(), &C2CLendingFundsParams{Coin: currency.BTC, Quantity: 1232})
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingOrderRecords(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingOrderRecords(t.Context(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetC2CLendingAccountInfo(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetC2CLendingAccountInfo(t.Context(), currency.LTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBrokerEarning(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBrokerEarning(t.Context(), "DERIVATIVES", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}

	r, err := b.UpdateAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountInfo must not error")
	require.NotEmpty(t, r, "UpdateAccountInfo must return account info")

	if mockTests {
		require.Len(t, r.Accounts, 1, "Accounts must have 1 item")
		require.Len(t, r.Accounts[0].Currencies, 3, "Accounts currencies must have 3 currency items")

		for x := range r.Accounts[0].Currencies {
			switch x {
			case 0:
				assert.Equal(t, currency.USDC, r.Accounts[0].Currencies[x].Currency, "Currency should be USDC")
				assert.Equal(t, -30723.63021638, r.Accounts[0].Currencies[x].Total, "Total amount should match")
				assert.Equal(t, -30723.63021638, r.Accounts[0].Currencies[x].Hold, "Hold amount should match")
				assert.Equal(t, 30723.630216383714, r.Accounts[0].Currencies[x].Borrowed, "Borrowed amount should match")
				assert.Equal(t, 0.0, r.Accounts[0].Currencies[x].Free, "Free amount should match")
			case 1:
				assert.Equal(t, currency.AVAX, r.Accounts[0].Currencies[x].Currency, "Currency should be AVAX")
				assert.Equal(t, 2473.9, r.Accounts[0].Currencies[x].Total, "Total amount should match")
				assert.Equal(t, 1468.10808813, r.Accounts[0].Currencies[x].Hold, "Hold amount should match")
				assert.Equal(t, 0.0, r.Accounts[0].Currencies[x].Borrowed, "Borrowed amount should match")
				assert.Equal(t, 1005.79191187, r.Accounts[0].Currencies[x].Free, "Free amount should match")
			case 2:
				assert.Equal(t, currency.USDT, r.Accounts[0].Currencies[x].Currency, "Currency should be USDT")
				assert.Equal(t, 935.1415, r.Accounts[0].Currencies[x].Total, "Total amount should match")
				assert.Equal(t, 0.0, r.Accounts[0].Currencies[x].Borrowed, "Borrowed amount should match")
				assert.Equal(t, 0.0, r.Accounts[0].Currencies[x].Hold, "Hold amount should match")
				assert.Equal(t, 935.1415, r.Accounts[0].Currencies[x].Free, "Free amount should match")
			}
		}
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
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
		_, err := b.GetRecentTrades(t.Context(), tt.p, tt.a)
		assert.NoErrorf(t, err, "GetRecentTrades should not error for %s asset", tt.a)
	}

	_, err := b.GetRecentTrades(t.Context(), spotTradablePair, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetBybitServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetBybitServerTime(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(t.Context(), asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(t.Context(), usdtMarginedTradablePair, asset.USDTMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(t.Context(), usdcMarginedTradablePair, asset.USDCMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(t.Context(), inverseTradablePair, asset.CoinMarginedFutures, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricTrades(t.Context(), optionsTradablePair, asset.Options, time.Time{}, time.Time{})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	orderCancellationParams := []order.Cancel{{
		OrderID:   "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	}, {
		OrderID:   "1",
		Pair:      usdtMarginedTradablePair,
		AssetType: asset.USDTMarginedFutures,
	}}
	_, err := b.CancelBatchOrders(t.Context(), orderCancellationParams)
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
	_, err = b.CancelBatchOrders(t.Context(), orderCancellationParams)
	if err != nil {
		t.Error(err)
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skippingWebsocketFunctionsForMockTesting)
	}
	err := b.WsConnect()
	if err != nil {
		t.Error(err)
	}
}

func TestWsLinearConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skippingWebsocketFunctionsForMockTesting)
	}
	err := b.WsLinearConnect()
	assert.Truef(t, errors.Is(err, websocket.ErrWebsocketNotEnabled) || err == nil, "WsLinerConnect should not error: %s", err)
}

func TestWsInverseConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skippingWebsocketFunctionsForMockTesting)
	}
	err := b.WsInverseConnect()
	assert.Truef(t, errors.Is(err, websocket.ErrWebsocketNotEnabled) || err == nil, "WsInverseConnect should not error: %s", err)
}

func TestWsOptionsConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(skippingWebsocketFunctionsForMockTesting)
	}
	err := b.WsOptionsConnect()
	assert.Truef(t, errors.Is(err, websocket.ErrWebsocketNotEnabled) || err == nil, "WsOptionsConnect should not error: %s", err)
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
	"Private Position":     `{"id": "59232430b58efe-5fc5-4470-9337-4ce293b68edd", "topic": "position", "creationTime": 1672364174455, "data": [ { "positionIdx": 0, "tradeMode": 0, "riskId": 41, "riskLimitValue": "200000", "symbol": "XRPUSDT", "side": "Buy", "size": "75", "entryPrice": "0.3615", "leverage": "10", "positionValue": "27.1125", "positionBalance": "0", "markPrice": "0.3374", "positionIM": "2.72589075", "positionMM": "0.28576575", "takeProfit": "0", "stopLoss": "0", "trailingStop": "0", "unrealisedPnl": "-1.8075", "cumRealisedPnl": "0.64782276", "createdTime": "1672121182216", "updatedTime": "1672364174449", "tpslMode": "Full", "liqPrice": "", "bustPrice": "", "category": "linear","positionStatus":"Normal","adlRankIndicator":2}]}`,
	"Private Order":        `{ "id": "5923240c6880ab-c59f-420b-9adb-3639adc9dd90", "topic": "order", "creationTime": 1672364262474, "data": [ { "symbol": "BTCUSDT", "orderId": "5cf98598-39a7-459e-97bf-76ca765ee020", "side": "Sell", "orderType": "Market", "cancelType": "UNKNOWN", "price": "72.5", "qty": "1", "orderIv": "", "timeInForce": "IOC", "orderStatus": "Filled", "orderLinkId": "", "lastPriceOnCreated": "", "reduceOnly": false, "leavesQty": "", "leavesValue": "", "cumExecQty": "1", "cumExecValue": "75", "avgPrice": "75", "blockTradeId": "", "positionIdx": 0, "cumExecFee": "0.358635", "createdTime": "1672364262444", "updatedTime": "1672364262457", "rejectReason": "EC_NoError", "stopOrderType": "", "tpslMode": "", "triggerPrice": "", "takeProfit": "", "stopLoss": "", "tpTriggerBy": "", "slTriggerBy": "", "tpLimitPrice": "", "slLimitPrice": "", "triggerDirection": 0, "triggerBy": "", "closeOnTrigger": false, "category": "option", "placeType": "price", "smpType": "None", "smpGroup": 0, "smpOrderId": "" } ] }`,
	"Private Wallet":       `{ "id": "5923242c464be9-25ca-483d-a743-c60101fc656f", "topic": "wallet", "creationTime": 1672364262482, "data": [ { "accountIMRate": "0.016", "accountMMRate": "0.003", "totalEquity": "12837.78330098", "totalWalletBalance": "12840.4045924", "totalMarginBalance": "12837.78330188", "totalAvailableBalance": "12632.05767702", "totalPerpUPL": "-2.62129051", "totalInitialMargin": "205.72562486", "totalMaintenanceMargin": "39.42876721", "coin": [ { "coin": "USDC", "equity": "200.62572554", "usdValue": "200.62572554", "walletBalance": "201.34882644", "availableToWithdraw": "0", "availableToBorrow": "1500000", "borrowAmount": "0", "accruedInterest": "0", "totalOrderIM": "0", "totalPositionIM": "202.99874213", "totalPositionMM": "39.14289747", "unrealisedPnl": "74.2768991", "cumRealisedPnl": "-209.1544627", "bonus": "0" }, { "coin": "BTC", "equity": "0.06488393", "usdValue": "1023.08402268", "walletBalance": "0.06488393", "availableToWithdraw": "0.06488393", "availableToBorrow": "2.5", "borrowAmount": "0", "accruedInterest": "0", "totalOrderIM": "0", "totalPositionIM": "0", "totalPositionMM": "0", "unrealisedPnl": "0", "cumRealisedPnl": "0", "bonus": "0" }, { "coin": "ETH", "equity": "0", "usdValue": "0", "walletBalance": "0", "availableToWithdraw": "0", "availableToBorrow": "26", "borrowAmount": "0", "accruedInterest": "0", "totalOrderIM": "0", "totalPositionIM": "0", "totalPositionMM": "0", "unrealisedPnl": "0", "cumRealisedPnl": "0", "bonus": "0" }, { "coin": "USDT", "equity": "11726.64664904", "usdValue": "11613.58597018", "walletBalance": "11728.54414904", "availableToWithdraw": "11723.92075829", "availableToBorrow": "2500000", "borrowAmount": "0", "accruedInterest": "0", "totalOrderIM": "0", "totalPositionIM": "2.72589075", "totalPositionMM": "0.28576575", "unrealisedPnl": "-1.8975", "cumRealisedPnl": "0.64782276", "bonus": "0" }, { "coin": "EOS3L", "equity": "215.0570412", "usdValue": "0", "walletBalance": "215.0570412", "availableToWithdraw": "215.0570412", "availableToBorrow": "0", "borrowAmount": "0", "accruedInterest": "", "totalOrderIM": "0", "totalPositionIM": "0", "totalPositionMM": "0", "unrealisedPnl": "0", "cumRealisedPnl": "0", "bonus": "0" }, { "coin": "BIT", "equity": "1.82", "usdValue": "0.48758257", "walletBalance": "1.82", "availableToWithdraw": "1.82", "availableToBorrow": "0", "borrowAmount": "0", "accruedInterest": "", "totalOrderIM": "0", "totalPositionIM": "0", "totalPositionMM": "0", "unrealisedPnl": "0", "cumRealisedPnl": "0", "bonus": "0" } ], "accountType": "UNIFIED", "accountLTV": "0.017" } ] }`,
	"Private Greek":        `{ "id": "592324fa945a30-2603-49a5-b865-21668c29f2a6", "topic": "greeks", "creationTime": 1672364262482, "data": [ { "baseCoin": "ETH", "totalDelta": "0.06999986", "totalGamma": "-0.00000001", "totalVega": "-0.00000024", "totalTheta": "0.00001314" } ] }`,
	"Execution":            `{"id": "592324803b2785-26fa-4214-9963-bdd4727f07be", "topic": "execution", "creationTime": 1672364174455, "data": [ { "category": "linear", "symbol": "XRPUSDT", "execFee": "0.005061", "execId": "7e2ae69c-4edf-5800-a352-893d52b446aa", "execPrice": "0.3374", "execQty": "25", "execType": "Trade", "execValue": "8.435", "isMaker": false, "feeRate": "0.0006", "tradeIv": "", "markIv": "", "blockTradeId": "", "markPrice": "0.3391", "indexPrice": "", "underlyingPrice": "", "leavesQty": "0", "orderId": "f6e324ff-99c2-4e89-9739-3086e47f9381", "orderLinkId": "", "orderPrice": "0.3207", "orderQty":"25","orderType":"Market","stopOrderType":"UNKNOWN","side":"Sell","execTime":"1672364174443","isLeverage": "0","closedSize": "","seq":4688002127}]}`,
}

func TestPushData(t *testing.T) {
	t.Parallel()

	keys := slices.Collect(maps.Keys(pushDataMap))
	slices.Sort(keys)

	for x := range keys {
		err := b.wsHandleData(t.Context(), asset.Spot, []byte(pushDataMap[keys[x]]))
		assert.NoError(t, err, "wsHandleData should not error")
	}
}

func TestWsTicker(t *testing.T) {
	t.Parallel()
	b := new(Bybit) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	assetRouting := []asset.Item{
		asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDTMarginedFutures,
		asset.USDCMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.CoinMarginedFutures,
	}
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsTicker.json", func(_ context.Context, r []byte) error {
		defer slices.Delete(assetRouting, 0, 1)
		return b.wsHandleData(t.Context(), assetRouting[0], r)
	})
	close(b.Websocket.DataHandler)
	expected := 8
	require.Len(t, b.Websocket.DataHandler, expected, "Should see correct number of tickers")
	for resp := range b.Websocket.DataHandler {
		switch v := resp.(type) {
		case *ticker.Price:
			assert.Equal(t, b.Name, v.ExchangeName, "ExchangeName should be correct")
			switch expected - len(b.Websocket.DataHandler) {
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
	_, err := b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = optionsTradablePair
	_, err = b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = usdtMarginedTradablePair
	_, err = b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	feeBuilder.Pair = inverseTradablePair
	_, err = b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	ctx := t.Context()
	err := b.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Multi, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}
	err = b.SetLeverage(ctx, asset.USDCMarginedFutures, usdcMarginedTradablePair, margin.Multi, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}

	err = b.SetLeverage(ctx, asset.CoinMarginedFutures, inverseTradablePair, margin.Isolated, 5, order.UnknownSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	err = b.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Isolated, 5, order.Buy)
	if err != nil {
		t.Error(err)
	}

	err = b.SetLeverage(ctx, asset.CoinMarginedFutures, inverseTradablePair, margin.Isolated, 5, order.Sell)
	if err != nil {
		t.Error(err)
	}

	err = b.SetLeverage(ctx, asset.USDTMarginedFutures, usdtMarginedTradablePair, margin.Isolated, 5, order.CouldNotBuy)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	err = b.SetLeverage(ctx, asset.Spot, inverseTradablePair, margin.Multi, 5, order.UnknownSide)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	assert.NoError(t, err)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.NoError(t, err)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.USDCMarginedFutures)
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.FetchTradablePairs(t.Context(), asset.CoinMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.FetchTradablePairs(t.Context(), asset.USDCMarginedFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.FetchTradablePairs(t.Context(), asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.FetchTradablePairs(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestDeltaUpdateOrderbook(t *testing.T) {
	t.Parallel()
	data := []byte(`{"topic":"orderbook.50.WEMIXUSDT","ts":1697573183768,"type":"snapshot","data":{"s":"WEMIXUSDT","b":[["0.9511","260.703"],["0.9677","0"]],"a":[],"u":3119516,"seq":14126848493},"cts":1728966699481}`)
	err := b.wsHandleData(t.Context(), asset.Spot, data)
	if err != nil {
		t.Fatal(err)
	}
	update := []byte(`{"topic":"orderbook.50.WEMIXUSDT","ts":1697573183768,"type":"delta","data":{"s":"WEMIXUSDT","b":[["0.9511","260.703"],["0.9677","0"]],"a":[],"u":3119516,"seq":14126848493},"cts":1728966699481}`)
	var wsResponse WebsocketResponse
	err = json.Unmarshal(update, &wsResponse)
	if err != nil {
		t.Fatal(err)
	}
	err = b.wsProcessOrderbook(asset.Spot, &wsResponse)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := b.GetLongShortRatio(t.Context(), "linear", "BTCUSDT", kline.FiveMin, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetLongShortRatio(t.Context(), "inverse", "BTCUSDT", kline.FiveMin, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetLongShortRatio(t.Context(), "spot", "BTCUSDT", kline.FiveMin, 0)
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	val, err := b.FetchAccountType(t.Context())
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	}
	err := b.RequiresUnifiedAccount(t.Context())
	require.NoError(t, err)
	b := &Bybit{} //nolint:govet // Intentional shadow to avoid future copy/paste mistakes. Also stops race below.
	b.account.accountType = accountTypeNormal
	err = b.RequiresUnifiedAccount(t.Context())
	require.ErrorIs(t, err, errAPIKeyIsNotUnified)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  usdtMarginedTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Spot,
		Pair:  spotTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Options,
		Pair:  optionsTradablePair,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.USDTMarginedFutures,
	})
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
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
	orders, err := b.ConstructOrderDetails(response, asset.Spot, currency.Pair{Base: currency.BTC, Quote: currency.USDT}, currency.Pairs{})
	if err != nil {
		t.Fatal(err)
	} else if len(orders) > 0 {
		t.Errorf("expected order with length 0, got %d", len(orders))
	}
	orders, err = b.ConstructOrderDetails(response, asset.Spot, currency.EMPTYPAIR, currency.Pairs{})
	if err != nil {
		t.Fatal(err)
	} else if len(orders) != 1 {
		t.Errorf("expected order with length 1, got %d", len(orders))
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  usdcMarginedTradablePair.Base.Item,
		Quote: usdcMarginedTradablePair.Quote.Item,
		Asset: asset.USDCMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  usdtMarginedTradablePair.Base.Item,
		Quote: usdtMarginedTradablePair.Quote.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  inverseTradablePair.Base.Item,
		Quote: inverseTradablePair.Quote.Item,
		Asset: asset.CoinMarginedFutures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()

	is, err := b.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	assert.NoError(t, err)
	assert.False(t, is)

	is, err = b.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, inverseTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.CoinMarginedFutures, inverseTradablePair)

	is, err = b.IsPerpetualFutureCurrency(asset.USDTMarginedFutures, usdtMarginedTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.USDTMarginedFutures, usdtMarginedTradablePair)

	is, err = b.IsPerpetualFutureCurrency(asset.USDCMarginedFutures, usdcMarginedTradablePair)
	assert.NoError(t, err)
	assert.Truef(t, is, "%s %s should be a perp", asset.USDCMarginedFutures, usdcMarginedTradablePair)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

// TestGenerateSubscriptions exercises generateSubscriptions
func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	b := new(Bybit)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")

	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	for _, s := range b.Features.Subscriptions {
		for _, a := range b.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := b.GetEnabledPairs(a)
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
				categoryName := getCategoryName(a)
				if isCategorisedChannel(s.QualifiedChannel) && categoryName != "" {
					s.QualifiedChannel += "." + categoryName
				}

				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	b := new(Bybit)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	subs, err := b.Features.Subscriptions.ExpandTemplates(b)
	require.NoError(t, err, "ExpandTemplates must not error")
	b.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, b)
	err = b.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
}

func TestAuthSubscribe(t *testing.T) {
	t.Parallel()
	b := new(Bybit)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := b.Features.Subscriptions.ExpandTemplates(b)
	require.NoError(t, err, "ExpandTemplates must not error")
	b.Features.Subscriptions = subscription.List{}
	success := true
	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req SubscriptionArgument
		require.NoError(tb, json.Unmarshal(msg, &req), "Unmarshal must not error")
		require.Equal(tb, "subscribe", req.Operation)
		msg, err = json.Marshal(SubscriptionResponse{
			Success:   success,
			RetMsg:    "Mock Resp Error",
			RequestID: req.RequestID,
			Operation: req.Operation,
		})
		require.NoError(tb, err, "Marshal must not error")
		return w.WriteMessage(gws.TextMessage, msg)
	}
	b = testexch.MockWsInstance[Bybit](t, testws.CurryWsMockUpgrader(t, mock))
	b.Websocket.AuthConn = b.Websocket.Conn
	err = b.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	success = false
	err = b.Subscribe(subs)
	assert.ErrorContains(t, err, "Mock Resp Error", "Subscribe should error containing the returned RetMsg")
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

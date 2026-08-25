package htx

import (
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const emptySuccessResponse = `{"status":"ok","code":200,"msg":"","data":null}`

func newHTTPTestExchange(t *testing.T, endpoint exchange.URL, method, path, response string, check func(*http.Request)) *Exchange {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path != "/v5/position/mode" && r.URL.Path == "/v5/position/mode" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":200,"data":{"position_mode":"single_side"}}`))
			return
		}
		assert.Equal(t, method, r.Method, "HTTP method should match")
		assert.Equal(t, path, r.URL.Path, "endpoint path should match")
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(endpoint.String(), server.URL), "running endpoint must be set")
	return h
}

// setV5PositionModeEndpoint configures the position-mode response required by V5 order tests.
func setV5PositionModeEndpoint(t *testing.T, h *Exchange, mode string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/position/mode", r.URL.Path, "position mode endpoint should match")
		_, _ = w.Write([]byte(`{"code":200,"data":{"position_mode":"` + mode + `"}}`))
	}))
	t.Cleanup(server.Close)
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "position mode endpoint must be set")
}

func TestNewHTTPTestExchange(t *testing.T) {
	t.Parallel()
	var checked atomic.Bool
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode", `{"code":200,"data":{"asset_mode":1}}`, func(r *http.Request) {
		checked.Store(true)
		assert.NotEmpty(t, r.URL.Query().Get("Signature"), "authenticated request should include a signature")
	})
	response, err := h.GetV5AssetMode(t.Context())
	require.NoError(t, err, "newHTTPTestExchange must provide a usable authenticated exchange")
	assert.True(t, checked.Load(), "request check should run")
	assert.Equal(t, uint64(1), response.Data.AssetMode, "response should decode")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	h.SetDefaults()
	assert.Equal(t, "HTX", h.Name, "exchange name should match")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.FundingRateFetching, "websocket funding rates should be supported")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.SubmitOrder, "websocket order submission should be supported")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.SubmitOrders, "websocket batch order submission should be supported")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.CancelOrder, "websocket order cancellation should be supported")
	assert.True(t, h.Features.TradingRequirements.SpotMarketBuyQuotation, "spot market buys should require quote amount")
	assert.True(t, h.Features.TradingRequirements.SpotMarketSellBase, "spot market sells should require base amount")
	assert.Len(t, h.Features.Subscriptions, 38, "default subscriptions should cover public and private spot and derivatives channels")
	for _, sub := range h.Features.Subscriptions {
		if sub.Authenticated && sub.Asset != asset.Spot {
			assert.False(t, sub.Enabled, "private derivative subscriptions should default to disabled")
		}
	}
	for _, tt := range []struct {
		endpoint exchange.URL
		want     string
	}{
		{endpoint: exchange.WebsocketFuturesPrivate, want: wsFuturesPrivateURL},
		{endpoint: exchange.WebsocketCoinMarginedPrivate, want: wsCoinMarginedPrivateURL},
		{endpoint: exchange.WebsocketUSDTMarginedPrivate, want: wsUSDTMarginedPrivateURL},
		{endpoint: exchange.WebsocketTrade, want: wsUSDTMarginedTradeURL},
	} {
		got, err := h.API.Endpoints.GetURL(tt.endpoint)
		require.NoError(t, err, "private derivative endpoint must be configured")
		assert.Equal(t, tt.want, got, "private derivative endpoint should match HTX documentation")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	assert.NotNil(t, h.Websocket, "websocket manager should be configured")
	assert.True(t, h.SupportsAsset(asset.Futures), "delivery futures should remain enabled")
	assert.True(t, h.SupportsAsset(asset.CoinMarginedFutures), "coin-margined futures should remain enabled")
	assert.True(t, h.SupportsAsset(asset.USDTMarginedFutures), "USDT-margined futures should remain enabled")
	tradeSetup := &websocket.ConnectionSetup{
		URL:                      wsUSDTMarginedTradeURL,
		Connector:                h.wsConnect,
		Handler:                  h.wsHandleData,
		MessageFilter:            exchange.WebsocketTrade,
		SubscriptionsNotRequired: true,
	}
	require.Error(t, h.Websocket.SetupNewConnection(tradeSetup), "trade websocket must be configured for runtime authentication gating")

	authenticated := new(Exchange)
	authenticated.SetDefaults()
	require.NoError(t, authenticated.Setup(&config.Exchange{
		Name:                    "HTX",
		Enabled:                 true,
		WebsocketTrafficTimeout: time.Second,
		API: config.APIConfig{
			AuthenticatedWebsocketSupport: true,
		},
	}), "Setup must configure authenticated websocket support")
	authenticatedTradeSetup := *tradeSetup
	authenticatedTradeSetup.Connector = authenticated.wsConnect
	authenticatedTradeSetup.Handler = authenticated.wsHandleData
	err := authenticated.Websocket.SetupNewConnection(&authenticatedTradeSetup)
	require.Error(t, err, "trade websocket must already be configured when authenticated websocket support is enabled")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/reference/currencies", `{
		"code":200,
		"data":[
			{"currency":"btc","instStatus":"normal","chains":[{"depositStatus":"allowed","withdrawStatus":"prohibited"},{"depositStatus":"prohibited","withdrawStatus":"allowed"},null]},
			{"currency":"bad","instStatus":"delisted","chains":[]}
		]
	}`, nil)
	require.NoError(t, h.UpdateCurrencyStates(t.Context(), asset.Spot), "UpdateCurrencyStates must not error")
	assert.NoError(t, h.CanTrade(currency.BTC, asset.Spot), "BTC trading should be enabled")
	assert.NoError(t, h.CanDeposit(currency.BTC, asset.Spot), "BTC deposits should be enabled when one chain allows deposits")
	assert.NoError(t, h.CanWithdraw(currency.BTC, asset.Spot), "BTC withdrawals should be enabled when one chain allows withdrawals")
	assert.Error(t, h.CanTrade(currency.NewCode("BAD"), asset.Spot), "delisted currency trading should be disabled")
	assert.Error(t, h.CanDeposit(currency.NewCode("BAD"), asset.Spot), "currency without chains should have deposits disabled")
	assert.Error(t, h.CanWithdraw(currency.NewCode("BAD"), asset.Spot), "currency without chains should have withdrawals disabled")
	require.ErrorIs(t, h.UpdateCurrencyStates(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateCurrencyStates must reject non-spot assets")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "-"), asset.Spot)
	assert.ErrorContains(t, err, "invalid symbol")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	require.NoError(t, err)
}

func TestUpdateTickerCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "_"), asset.CoinMarginedFutures)
	assert.ErrorContains(t, err, "symbol data error")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", "_"), asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateTickerFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTickerUSDTMarginedFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
}

func TestUpdateOrderbookCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookFuture(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookUSDTMarginedFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookUnsupportedAsset(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject unsupported assets")
}

func TestUpdateOrderbookWithLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","tick":{"ts":1604312615051,"bids":[[10,2],[9,3]],"asks":[[11,2],[12,3]]}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
	book, err := h.UpdateOrderbookWithLimit(t.Context(), btcusdtPair, asset.Spot, 1)
	require.NoError(t, err, "UpdateOrderbookWithLimit must not error")
	assert.Len(t, book.Bids, 1, "bid depth should be capped")
	assert.Len(t, book.Asks, 1, "ask depth should be capped")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		item       asset.Item
		pair       currency.Pair
		path       string
		contract   string
		extraEntry map[string]any
	}{
		{
			name:     "coin margined futures",
			item:     asset.CoinMarginedFutures,
			pair:     btcusdPair,
			path:     "/swap-api/v3/swap_hisorders",
			contract: "BTC-USD",
		},
		{
			name:     "delivery futures",
			item:     asset.Futures,
			pair:     btccwPair,
			path:     fOrderHistory,
			contract: "BTC_CW",
			extraEntry: map[string]any{
				"create_date": time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path, "history endpoint path should match")
				payload, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "request body should be readable")
				var requestBody map[string]any
				assert.NoError(t, json.Unmarshal(payload, &requestBody), "request body should decode")
				assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "history pagination should move forwards")
				cursor, _ := requestBody["from_id"].(float64)
				entries := make([]map[string]any, 0, 50)
				switch int64(cursor) {
				case 0:
					for queryID := int64(1); queryID <= 50; queryID++ {
						entry := map[string]any{
							"query_id":         queryID,
							"order_id":         queryID,
							"order_id_str":     strconv.FormatInt(queryID, 10),
							"contract_code":    tc.contract,
							"direction":        "buy",
							"order_price_type": "limit",
							"status":           6,
						}
						maps.Copy(entry, tc.extraEntry)
						entries = append(entries, entry)
					}
				case 50:
					entry := map[string]any{
						"query_id":         int64(51),
						"order_id":         int64(51),
						"order_id_str":     "51",
						"contract_code":    tc.contract,
						"direction":        "buy",
						"order_price_type": "limit",
						"status":           6,
					}
					maps.Copy(entry, tc.extraEntry)
					entries = append(entries, entry)
				}
				response, err := json.Marshal(map[string]any{"code": 200, "data": entries})
				assert.NoError(t, err, "response body should encode")
				_, _ = w.Write(response)
				calls.Add(1)
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
			startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tc.pair},
				AssetType: tc.item,
				Side:      order.AnySide,
				StartTime: startTime,
				EndTime:   startTime.Add(24 * time.Hour),
			})
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Len(t, orders, 51, "all cursor pages should be returned")
			assert.Equal(t, int64(2), calls.Load(), "pagination should stop after the short final page")
		})
	}
	t.Run("USDT margined futures", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v5/trade/order/history", r.URL.Path, "history endpoint path should match")
			assert.Equal(t, "next", r.URL.Query().Get("direct"), "history pagination should move forwards")
			start, err := strconv.ParseInt(r.URL.Query().Get("start_time"), 10, 64)
			if !assert.NoError(t, err, "history start time should be valid") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			end, err := strconv.ParseInt(r.URL.Query().Get("end_time"), 10, 64)
			if !assert.NoError(t, err, "history end time should be valid") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.LessOrEqual(t, end-start, int64(48*time.Hour/time.Millisecond), "V5 history windows should not exceed 48 hours")
			calls.Add(1)
			if r.URL.Query().Get("margin_mode") == "cross" {
				_, _ = w.Write([]byte(`{"code":200,"data":[{"id":"1","contract_code":"BTC-USDT","order_id":"1","side":"buy","type":"limit","state":"filled","time_in_force":"gtc","margin_mode":"cross","price":"100","volume":"2","trade_volume":"1"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
		startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
		orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			Type:      order.AnyType,
			Pairs:     currency.Pairs{btcusdtPair},
			AssetType: asset.USDTMarginedFutures,
			Side:      order.AnySide,
			StartTime: startTime,
			EndTime:   startTime.Add(49 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error")
		require.Len(t, orders, 1, "one historical order must be returned")
		assert.Equal(t, "1", orders[0].OrderID, "order ID should match")
		assert.Equal(t, int64(4), calls.Load(), "each history window should query cross and isolated margin modes")
	})
}

func TestGetV3HistoryWindows(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(5 * 24 * time.Hour)
	windows, err := getV3HistoryWindows(startTime, endTime)
	require.NoError(t, err, "getV3HistoryWindows must not error")
	require.Len(t, windows, 3, "five days must be split into three windows")
	for x := range windows {
		assert.LessOrEqual(t, windows[x].end.Sub(windows[x].start), 48*time.Hour, "each window should be at most 48 hours")
	}
	assert.Equal(t, startTime, windows[0].start, "first window should preserve the requested start")
	assert.Equal(t, endTime, windows[len(windows)-1].end, "last window should preserve the requested end")
	assert.Equal(t, time.Millisecond, windows[1].start.Sub(windows[0].end), "adjacent millisecond ranges should not overlap")

	windows, err = getV3HistoryWindows(time.Time{}, time.Time{})
	require.NoError(t, err, "getV3HistoryWindows must accept an unspecified interval")
	require.Len(t, windows, 1, "an unspecified interval must produce one request")
	assert.True(t, windows[0].start.IsZero(), "unspecified start should remain zero")
	assert.True(t, windows[0].end.IsZero(), "unspecified end should remain zero")

	_, err = getV3HistoryWindows(endTime, startTime)
	require.ErrorIs(t, err, errStartTimeAfterEndTime, "getV3HistoryWindows must reject reversed intervals")
	_, err = getV3HistoryWindows(startTime, startTime.Add(91*24*time.Hour))
	require.ErrorIs(t, err, errInvalidCreateDate, "getV3HistoryWindows must reject intervals over 90 days")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		endpoint exchange.URL
		path     string
		cancel   order.Cancel
		response string
	}{
		{
			name: "spot", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchCancelOpenOrders",
			cancel:   order.Cancel{AccountID: "1", Pair: btcusdtPair, AssetType: asset.Spot},
			response: `{"status":"ok","data":{"success-count":1,"failed-count":0}}`,
		},
		{
			name: "coin margined futures", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancelall",
			cancel:   order.Cancel{Pair: btcusdPair, AssetType: asset.CoinMarginedFutures},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`,
		},
		{
			name: "USDT margined futures", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_all_orders",
			cancel:   order.Cancel{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures},
			response: `{"code":200,"data":[{"order_id":"1"}]}`,
		},
		{
			name: "delivery futures", endpoint: exchange.RestFutures, path: fCancelAllOrders,
			cancel:   order.Cancel{Pair: btccwPair, AssetType: asset.Futures},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, tt.response, nil)
			resp, err := h.CancelAllOrders(t.Context(), &tt.cancel)
			require.NoError(t, err, "CancelAllOrders must not error")
			if tt.cancel.AssetType != asset.Spot {
				assert.Equal(t, htxStatusSuccess, resp.Status["1"], "cancellation status should indicate success")
			}
		})
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	require.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetServerTime should return a time")
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/order/openOrders", `{
		"status":"ok",
		"data":[{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-limit","state":"submitted","filled-amount":"1","filled-fees":"0.1"}]
	}`, func(r *http.Request) {
		assert.Equal(t, "buy", r.URL.Query().Get("side"), "spot order-side filter should be forwarded")
	})
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		Side:      order.Buy,
	}

	orders, err := h.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, orders, 1, "one active order must be returned")
	assert.Equal(t, "1", orders[0].OrderID, "order ID should match")

	var crossRequests, isolatedRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/order/opens", r.URL.Path, "V5 open-order path should match")
		marginMode := r.URL.Query().Get("margin_mode")
		switch marginMode {
		case "cross":
			crossRequests.Add(1)
		case "isolated":
			isolatedRequests.Add(1)
		default:
			assert.Failf(t, "unexpected margin mode", "received %q", marginMode)
		}
		_, _ = w.Write([]byte(`{"code":200,"data":[{"id":"1","contract_code":"BTC-USDT","order_id":"1","side":"buy","type":"limit","state":"submitted","time_in_force":"gtc","margin_mode":"` + marginMode + `","price":"100","volume":"2","trade_volume":"1"}]}`))
	}))
	t.Cleanup(server.Close)
	h = new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT endpoint must be set")
	orders, err = h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.USDTMarginedFutures,
		Type:      order.AnyType,
		Pairs:     currency.Pairs{btcusdtPair},
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must query both margin modes")
	require.Len(t, orders, 2, "cross and isolated active orders must be returned")
	assert.Equal(t, int32(1), crossRequests.Load(), "cross mode should be queried once")
	assert.Equal(t, int32(1), isolatedRequests.Load(), "isolated mode should be queried once")
}

func TestGetActiveOrdersValidation(t *testing.T) {
	t.Parallel()

	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.ErrorContains(t, err, "currency must be supplied", "GetActiveOrders must require a currency pair for spot")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tt.pair},
				AssetType: tt.asset,
				Side:      order.AnySide,
			})
			require.ErrorIs(t, err, tt.expectedErr, "GetActiveOrders must return the expected branch error")
		})
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	var requests uint64
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/query/deposit-withdraw",
		`{"status":"ok","data":[{"id":1,"type":"deposit","currency":"btc","tx-hash":"tx-1","chain":"btc","amount":2,"address":"address-1","fee":0,"state":"safe","created-at":1612261330443},{"id":2,"type":"withdraw","currency":"usdt","tx-hash":"tx-2","chain":"trc20usdt","amount":3,"address":"address-2","fee":0.1,"state":"confirmed","error-message":"","created-at":1612261389250}]}`,
		func(r *http.Request) {
			assert.Equal(t, "next", r.URL.Query().Get("direct"), "query direction should request the newest records")
			assert.Equal(t, "500", r.URL.Query().Get("size"), "query size should use HTX's maximum page size")
			switch r.URL.Query().Get("type") {
			case "deposit", "withdraw":
			default:
				require.Failf(t, "unexpected funding history type", "type %q must be deposit or withdraw", r.URL.Query().Get("type"))
			}
			atomic.AddUint64(&requests, 1)
		})
	h.Name = "HTX"
	history, err := h.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	require.Len(t, history, 4, "funding history must include both endpoint responses")
	assert.Equal(t, uint64(2), atomic.LoadUint64(&requests), "funding history should make two requests")
	assert.Equal(t, exchange.FundingHistory{
		ExchangeName:    "HTX",
		Status:          "safe",
		TransferID:      "1",
		Timestamp:       time.UnixMilli(1612261330443),
		Currency:        "BTC",
		Amount:          2,
		TransferType:    "deposit",
		CryptoToAddress: "address-1",
		CryptoTxID:      "tx-1",
		CryptoChain:     "btc",
	}, history[0], "deposit history should be normalised")
	assert.Equal(t, "withdraw", history[1].TransferType, "withdrawal type should be normalised")
	assert.Equal(t, 0.1, history[1].Fee, "withdrawal fee should be retained")
}

func TestGetAccountID(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetAccountID(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetAccountID must require credentials")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		endpoint   exchange.URL
		path       string
		submission order.Submit
		response   string
		status     order.Status
		check      func(*testing.T, *http.Request)
	}{
		{
			name: "spot", endpoint: exchange.RestSpot, path: "/v1/order/orders/place",
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, ClientID: "1"},
			response:   `{"status":"ok","data":"1"}`,
			status:     order.New,
		},
		{
			name: "spot market buy quote amount", endpoint: exchange.RestSpot, path: "/v1/order/orders/place",
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientID: "1", ClientOrderID: "client-1"},
			response:   `{"status":"ok","data":"1"}`,
			status:     order.New,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "spot market request must decode")
				assert.Equal(t, "25", req["amount"], "spot market buy should submit the quote amount")
				assert.Equal(t, "client-1", req["client-order-id"], "spot client order ID should be submitted")
			},
		},
		{
			name: "coin margined futures", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_order",
			submission: order.Submit{Pair: btcusdPair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5},
			response:   `{"status":"ok","data":{"order_id":1,"order_id_str":"1"}}`,
			status:     order.New,
		},
		{
			name: "USDT margined futures", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/order",
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5, TimeInForce: order.GoodTillCancel | order.PostOnly},
			response:   `{"code":200,"data":{"order_id":"1"}}`,
			status:     order.New,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req V5OrderRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "USDT-margined order request must decode")
				assert.Equal(t, orderPriceTypePostOnly, req.Type, "post-only protection should be preserved")
			},
		},
		{
			name: "delivery futures", endpoint: exchange.RestFutures, path: fOrder,
			submission: order.Submit{Pair: btccwPair, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5},
			response:   `{"status":"ok","data":{"order_id":1,"order_id_str":"1"}}`,
			status:     order.New,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var check func(*http.Request)
			if tt.check != nil {
				check = func(r *http.Request) { tt.check(t, r) }
			}
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, tt.response, check)
			tt.submission.Exchange = h.Name
			response, err := h.SubmitOrder(t.Context(), &tt.submission)
			require.NoError(t, err, "SubmitOrder must not error")
			assert.Equal(t, "1", response.OrderID, "order ID should match")
			assert.Equal(t, tt.status, response.Status, "response status should match")
		})
	}
}

func TestSubmitOrderValidation(t *testing.T) {
	t.Parallel()

	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject nil submissions")

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange: e.Name, Pair: btcusdtPair, AssetType: asset.Spot,
		Side: order.Buy, Type: order.Market, Amount: 1, ClientID: "1",
	})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "SubmitOrder must require quote amount for spot market buys")

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "invalid",
	})
	require.Error(t, err, "SubmitOrder must reject invalid spot account IDs")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.SubmitOrder(t.Context(), &order.Submit{
				Exchange:  h.Name,
				Pair:      tt.pair,
				AssetType: tt.asset,
				Side:      order.Buy,
				Type:      order.Limit,
				Price:     1,
				Amount:    1,
				Leverage:  1,
			})
			require.ErrorIs(t, err, tt.expectedErr, "SubmitOrder must return the expected branch error")
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelOrder must reject nil cancellations")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "invalid",
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
	})
	require.Error(t, err, "CancelOrder must reject non-numeric spot order IDs")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		Pair:      btcusdtPair,
		AssetType: asset.Binary,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelOrder must reject unsupported assets")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := h.CancelOrder(t.Context(), &order.Cancel{
				OrderID:   "1",
				Pair:      tt.pair,
				AssetType: tt.asset,
			})
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelOrder must require credentials")
		})
	}
	for _, tt := range []struct {
		name          string
		endpoint      exchange.URL
		path          string
		cancel        order.Cancel
		expectedField string
		expectedValue any
		response      string
		expectedErr   error
	}{
		{
			name: "spot response", endpoint: exchange.RestSpot, path: "/v1/order/orders/42/submitcancel",
			cancel: order.Cancel{OrderID: "42", Pair: btcusdtPair, AssetType: asset.Spot},
		},
		{
			name: "spot client order ID", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.Spot},
			expectedField: "client-order-ids", expectedValue: []any{"client-42"},
			response: `{"status":"ok","data":{"success":["client-42"],"failed":[]}}`,
		},
		{
			name: "spot client order ID failure", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.Spot},
			expectedField: "client-order-ids", expectedValue: []any{"client-42"},
			response:    `{"status":"ok","data":{"success":[],"failed":[{"client-order-id":"client-42","err-code":"order-orderstate-error","err-msg":"Incorrect order state"}]}}`,
			expectedErr: htxError("Incorrect order state"),
		},
		{
			name: "coin margined client order ID", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdPair, AssetType: asset.CoinMarginedFutures},
			expectedField: "client_order_id", expectedValue: "client-42",
		},
		{
			name: "USDT margined client order ID", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_order",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures},
			expectedField: "client_order_id", expectedValue: "client-42",
		},
		{
			name: "delivery futures order ID", endpoint: exchange.RestFutures, path: fCancelOrder,
			cancel:        order.Cancel{OrderID: "42", Pair: btccwPair, AssetType: asset.Futures},
			expectedField: "order_id", expectedValue: "42",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := tt.response
			if response == "" {
				response = emptySuccessResponse
			}
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, response, func(r *http.Request) {
				if tt.expectedField == "" {
					return
				}
				payload, err := io.ReadAll(r.Body)
				require.NoError(t, err, "request body must be readable")
				var body map[string]any
				require.NoError(t, json.Unmarshal(payload, &body), "request body must decode")
				assert.Equal(t, tt.expectedValue, body[tt.expectedField], "order identifier should match")
			})
			err := h.CancelOrder(t.Context(), &tt.cancel)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "CancelOrder must return the expected error")
			} else {
				require.NoError(t, err, "CancelOrder must not error")
			}
		})
	}
}

func TestCancelBatchOrdersValidation(t *testing.T) {
	t.Parallel()

	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject empty requests")

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must require an order ID")

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{
		{OrderID: "1", AssetType: asset.Spot},
		{OrderID: "2", AssetType: asset.Futures},
	})
	require.ErrorIs(t, err, errBatchAssetMismatch, "CancelBatchOrders must reject mixed assets")

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{
		{OrderID: "1", AssetType: asset.Futures, Pair: btccwPair},
		{OrderID: "2", AssetType: asset.Futures, Pair: currency.NewPair(currency.ETH, currency.NewCode("CW"))},
	})
	require.ErrorIs(t, err, errBatchPairMismatch, "CancelBatchOrders must reject mixed derivative pairs")

	tooMany := make([]order.Cancel, 11)
	for i := range tooMany {
		tooMany[i] = order.Cancel{OrderID: strconv.Itoa(i + 1), AssetType: asset.USDTMarginedFutures, Pair: btcusdtPair}
	}
	_, err = e.CancelBatchOrders(t.Context(), tooMany)
	require.ErrorIs(t, err, errBatchOrderLimitExceeded, "CancelBatchOrders must enforce the V5 ten-order limit")
}

func TestCancelAllOrdersValidation(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject nil cancellations")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.CancelAllOrders(t.Context(), &order.Cancel{
				OrderID:   "1",
				Pair:      tt.pair,
				AssetType: tt.asset,
			})
			require.ErrorIs(t, err, tt.expectedErr, "CancelAllOrders must return the expected branch error")
		})
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account/accounts":
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"type":"spot","state":"working"}]}`))
		case "/v1/account/accounts/1/balance":
			_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"type":"spot","state":"working","list":[{"currency":"btc","type":"trade","balance":"2"},{"currency":"btc","type":"frozen","balance":"3"},{"currency":"btc","type":"loan","balance":"7"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
	subAccounts, err := h.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "one spot subaccount must be returned")
	balance := subAccounts[0].Balances[currency.BTC]
	assert.Equal(t, 5.0, balance.Total, "total should include free and held balances")
	assert.Equal(t, 2.0, balance.Free, "free balance should include trade funds")
	assert.Equal(t, 3.0, balance.Hold, "held balance should include frozen funds")

	h = newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance",
		`{"code":200,"data":{"details":[{"currency":"USDT","equity":"10","available":"4","isolated_available":"2"}]}}`, nil)
	subAccounts, err = h.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "USDT-margined UpdateAccountBalances must not error")
	balance = subAccounts[0].Balances[currency.USDT]
	assert.Equal(t, 6.0, balance.Free, "free balance should include cross and isolated availability")
	assert.Equal(t, 4.0, balance.Hold, "held balance should exclude all available funds")
}

func TestUpdateAccountBalancesValidation(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.UpdateAccountBalances(t.Context(), tt.asset)
			require.ErrorIs(t, err, tt.expectedErr, "UpdateAccountBalances must return the expected branch error")
		})
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder must return unsupported")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.AuthenticateWebsocket(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "AuthenticateWebsocket must require credentials")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.ValidateAPICredentials(t.Context(), asset.Spot)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "ValidateAPICredentials must report authentication errors")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/account/deposit/address", `{
		"code":200,
		"data":[{"currency":"usdt","address":"address","addressTag":"","chain":"usdterc20"}]
	}`, nil)
	address, err := h.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.Equal(t, "address", address.Address, "deposit address should match")
}

func TestGetDepositAddressAuthentication(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetDepositAddress must require credentials")
}

func TestFormatV5OrderDetail(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	h.SetDefaults()
	_, err := h.formatV5OrderDetail(nil, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, common.ErrNilPointer, "formatV5OrderDetail must reject nil data")
	detail, err := h.formatV5OrderDetail(&V5OrderData{
		ContractCode:      "BTC-USDT",
		OrderID:           "1",
		ClientOrderID:     "2",
		Side:              "buy",
		Type:              orderPriceTypePostOnly,
		State:             "filled",
		TimeInForce:       "gtc",
		MarginMode:        "cross",
		Price:             100,
		Volume:            2,
		TradeVolume:       1,
		TradeTurnover:     100,
		TradeAveragePrice: 100,
		Fee:               0.1,
		FeeCurrency:       "USDT",
		LeverageRate:      5,
		ReduceOnly:        true,
	}, asset.USDTMarginedFutures)
	require.NoError(t, err, "formatV5OrderDetail must not error")
	assert.Equal(t, "1", detail.OrderID, "order ID should match")
	assert.Equal(t, order.Buy, detail.Side, "side should match")
	assert.Equal(t, order.Limit, detail.Type, "type should match")
	assert.Equal(t, order.Filled, detail.Status, "status should match")
	assert.Equal(t, order.PostOnly, detail.TimeInForce, "post-only order should retain its time in force")
	assert.Equal(t, margin.Multi, detail.MarginType, "margin type should match")
	assert.Equal(t, 1.0, detail.RemainingAmount, "remaining amount should match")

	detail, err = h.formatV5OrderDetail(&V5OrderData{
		ContractCode: "BTC-USDT",
		Side:         "sell",
		Type:         "limit",
		State:        "partially_canceled",
		TimeInForce:  "gtc",
		MarginMode:   "isolated",
	}, asset.USDTMarginedFutures)
	require.NoError(t, err, "formatV5OrderDetail must accept documented partially-cancelled states")
	assert.Equal(t, order.PartiallyCancelled, detail.Status, "partially-cancelled state should be retained")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()

	_, err := e.GetOrderInfo(t.Context(), "", btcusdtPair, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")

	_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject empty pairs")

	_, err = e.GetOrderInfo(t.Context(), "1", btcusdtPair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject unsupported assets")

	_, err = e.GetOrderInfo(t.Context(), "invalid", btcusdtPair, asset.Spot)
	require.Error(t, err, "GetOrderInfo must reject non-numeric spot order IDs")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetOrderInfo(t.Context(), "1", tt.pair, tt.asset)
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetOrderInfo must require credentials")
		})
	}
	for _, tt := range []struct {
		name               string
		endpoint           exchange.URL
		method             string
		path               string
		pair               currency.Pair
		asset              asset.Item
		response           string
		expectedMarginMode string
		expectedType       order.Type
		expectedTIF        order.TimeInForce
	}{
		{
			name: "spot maker response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-limit-maker","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.PostOnly,
		},
		{
			name: "spot stop limit response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-stop-limit","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.Limit,
		},
		{
			name: "spot limit FOK response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"sell-limit-fok","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.FillOrKill,
		},
		{
			name: "spot stop limit FOK response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-stop-limit-fok","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.FillOrKill,
		},
		{
			name: "coin margined response", endpoint: exchange.RestFutures, method: http.MethodPost, path: "/swap-api/v1/swap_order_info", pair: btcusdPair, asset: asset.CoinMarginedFutures,
			response: `{"status":"ok","data":[{"contract_code":"BTC-USD","volume":2,"price":100,"order_price_type":"limit","direction":"buy","offset":"close","lever_rate":5,"order_id":1,"order_id_str":"1","client_order_id":2,"trade_volume":1,"trade_turnover":100,"fee":0.1,"trade_avg_price":100,"status":6,"fee_asset":"BTC"}]}`,
		},
		{
			name: "USDT margined response", endpoint: exchange.RestUSDTMargined, method: http.MethodGet, path: "/v5/trade/order", pair: btcusdtPair, asset: asset.USDTMarginedFutures,
			response:           `{"code":200,"data":{"id":"1","contract_code":"BTC-USDT","order_id":"1","client_order_id":"2","side":"buy","type":"limit","state":"filled","time_in_force":"gtc","margin_mode":"isolated","price":"100","volume":"2","trade_volume":"1","trade_turnover":"100","trade_avg_price":"100","fee":"0.1","fee_currency":"USDT","lever_rate":"5","reduce_only":true}}`,
			expectedMarginMode: "cross",
		},
		{
			name: "delivery futures response", endpoint: exchange.RestFutures, method: http.MethodPost, path: fOrderInfo, pair: btccwPair, asset: asset.Futures,
			response: `{"status":"ok","data":[{"contract_code":"BTC_CW","volume":2,"price":100,"order_price_type":"limit","direction":"buy","offset":"close","lever_rate":5,"order_id":1,"order_id_str":"1","client_order_id":2,"trade_volume":1,"trade_turnover":100,"fee":0.1,"trade_avg_price":100,"status":6,"fee_asset":"BTC"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var check func(*http.Request)
			if tt.expectedMarginMode != "" {
				check = func(r *http.Request) {
					assert.Equal(t, tt.expectedMarginMode, r.URL.Query().Get("margin_mode"), "GetOrderInfo should query a concrete margin mode")
				}
			}
			h := newHTTPTestExchange(t, tt.endpoint, tt.method, tt.path, tt.response, check)
			detail, err := h.GetOrderInfo(t.Context(), "1", tt.pair, tt.asset)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, "1", detail.OrderID, "order ID should match")
			assert.Equal(t, 2.0, detail.Amount, "amount should match")
			assert.Equal(t, 1.0, detail.ExecutedAmount, "executed amount should match")
			assert.Equal(t, tt.asset, detail.AssetType, "asset type should match")
			if tt.expectedType != order.UnknownType {
				assert.Equal(t, tt.expectedType, detail.Type, "order type should match")
				assert.Equal(t, tt.expectedTIF, detail.TimeInForce, "time in force should match")
			}
		})
	}
	t.Run("USDT isolated fallback after cross error", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v5/trade/order", r.URL.Path, "order endpoint should match")
			calls.Add(1)
			if r.URL.Query().Get("margin_mode") == "cross" {
				_, _ = w.Write([]byte(`{"code":1048,"message":"order not found"}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"contract_code":"BTC-USDT","order_id":"1","side":"buy","type":"limit","state":"filled","time_in_force":"gtc","margin_mode":"isolated","price":"100","volume":"2","trade_volume":"1"}}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

		detail, err := h.GetOrderInfo(t.Context(), "1", btcusdtPair, asset.USDTMarginedFutures)
		require.NoError(t, err, "GetOrderInfo must try isolated margin after a cross-margin lookup error")
		assert.Equal(t, margin.Isolated, detail.MarginType, "isolated order should retain its margin type")
		assert.Equal(t, int64(2), calls.Load(), "both margin modes should be queried")
	})
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		interval kline.Interval
		output   string
	}{
		{kline.OneMin, "1min"},
		{kline.FourHour, "4hour"},
		{kline.OneDay, "1day"},
		{kline.OneWeek, "1week"},
		{kline.OneMonth, "1mon"},
		{kline.OneYear, "1year"},
		{kline.TwoWeek, ""},
	} {
		assert.Equalf(t, tt.output, e.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correctly for %s", tt.output)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), btcusdtPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetV5PositionModeName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		response    string
		expected    string
		expectedErr error
		wantErr     bool
	}{
		{name: "success", response: `{"code":200,"data":{"position_mode":"dual_side"}}`, expected: "dual_side"},
		{name: "empty response", response: `null`, expectedErr: errEmptyResult},
		{name: "empty mode", response: `{"code":200,"data":{}}`, expectedErr: errEmptyResult},
		{name: "invalid mode", response: `{"code":200,"data":{"position_mode":"invalid"}}`, expectedErr: errInvalidPositionMode},
		{name: "decode error", response: `{`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/mode", tc.response, nil)
			mode, err := h.getV5PositionModeName(t.Context())
			if tc.wantErr || tc.expectedErr != nil {
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr, "getV5PositionModeName must return the expected error")
				} else {
					require.Error(t, err, "getV5PositionModeName must return an endpoint error")
				}
				return
			}
			require.NoError(t, err, "getV5PositionModeName must not error")
			assert.Equal(t, tc.expected, mode, "position mode should match")
		})
	}
}

func TestFormatV5OrderRequest(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	_, err := h.formatV5OrderRequest(nil, "dual_side")
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "formatV5OrderRequest must reject nil submissions")

	for _, tc := range []struct {
		name             string
		submit           order.Submit
		expectedType     string
		expectedTIF      string
		expectedMode     string
		expectedPosition string
	}{
		{
			name: "isolated post only",
			submit: order.Submit{
				Pair:          btcusdtPair,
				AssetType:     asset.USDTMarginedFutures,
				Side:          order.Buy,
				Type:          order.Limit,
				Price:         100,
				Amount:        2,
				MarginType:    margin.Isolated,
				TimeInForce:   order.GoodTillCancel | order.PostOnly,
				ReduceOnly:    true,
				ClientOrderID: "client-id",
			},
			expectedType:     orderPriceTypePostOnly,
			expectedTIF:      "gtc",
			expectedMode:     "isolated",
			expectedPosition: "long",
		},
		{
			name: "cross market FOK",
			submit: order.Submit{
				Pair:        btcusdtPair,
				AssetType:   asset.USDTMarginedFutures,
				Side:        order.Sell,
				Type:        order.Market,
				Amount:      2,
				MarginType:  margin.Multi,
				TimeInForce: order.FillOrKill,
			},
			expectedType:     "market",
			expectedTIF:      "fok",
			expectedMode:     "cross",
			expectedPosition: "short",
		},
		{
			name: "default GTC",
			submit: order.Submit{
				Pair:      btcusdtPair,
				AssetType: asset.USDTMarginedFutures,
				Side:      order.Buy,
				Type:      order.Limit,
				Price:     100,
				Amount:    2,
			},
			expectedType:     "limit",
			expectedTIF:      "gtc",
			expectedMode:     "cross",
			expectedPosition: "long",
		},
		{
			name: "IOC",
			submit: order.Submit{
				Pair:        btcusdtPair,
				AssetType:   asset.USDTMarginedFutures,
				Side:        order.Buy,
				Type:        order.Limit,
				Price:       100,
				Amount:      2,
				TimeInForce: order.ImmediateOrCancel,
			},
			expectedType:     "limit",
			expectedTIF:      "ioc",
			expectedMode:     "cross",
			expectedPosition: "long",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := h.formatV5OrderRequest(&tc.submit, "dual_side")
			require.NoError(t, err, "formatV5OrderRequest must not error")
			assert.Equal(t, "BTC-USDT", got.ContractCode, "contract code should be formatted")
			assert.Equal(t, tc.expectedType, got.Type, "order type should match")
			assert.Equal(t, tc.expectedTIF, got.TimeInForce, "time in force should match")
			assert.Equal(t, tc.expectedMode, got.MarginMode, "margin mode should match")
			assert.Equal(t, tc.expectedPosition, got.PositionSide, "position side should match")
		})
	}

	for _, tc := range []struct {
		name        string
		submit      order.Submit
		expectedErr error
	}{
		{name: "empty pair", submit: order.Submit{Side: order.Buy, Type: order.Limit}, expectedErr: currency.ErrCurrencyPairEmpty},
		{name: "unsupported margin", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Limit, MarginType: margin.NoMargin}, expectedErr: margin.ErrMarginTypeUnsupported},
		{name: "invalid side", submit: order.Submit{Pair: btcusdtPair, Type: order.Limit}, expectedErr: order.ErrSideIsInvalid},
		{name: "unsupported type", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Stop}, expectedErr: order.ErrUnsupportedOrderType},
		{name: "unsupported time in force", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.TimeInForce(1 << 15)}, expectedErr: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.formatV5OrderRequest(&tc.submit, "dual_side")
			require.ErrorIs(t, err, tc.expectedErr, "formatV5OrderRequest must return the expected validation error")
		})
	}
	t.Run("unconfigured pair format", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).formatV5OrderRequest(&order.Submit{
			Pair: btcusdtPair,
			Side: order.Buy,
			Type: order.Limit,
		}, "dual_side")
		require.Error(t, err, "formatV5OrderRequest must return pair-format errors")
	})

	t.Run("single-side mode", func(t *testing.T) {
		t.Parallel()
		got, err := h.formatV5OrderRequest(&order.Submit{
			Pair: btcusdtPair,
			Side: order.Buy,
			Type: order.Market,
		}, "single_side")
		require.NoError(t, err, "formatV5OrderRequest must support single-side mode")
		assert.Equal(t, "both", got.PositionSide, "single-side mode should target both positions")
	})
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	setV5PositionModeEndpoint(t, h, "single_side")
	submission := &order.Submit{
		Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures,
		Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1,
	}
	resp, err := h.WebsocketSubmitOrder(t.Context(), submission)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	assert.Equal(t, "1", resp.OrderID, "order ID should match")
	assert.Equal(t, "2", resp.ClientOrderID, "client order ID should match")
	assert.Equal(t, order.New, resp.Status, "order status should be new")

	submission.AssetType = asset.Spot
	_, err = h.WebsocketSubmitOrder(t.Context(), submission)
	require.ErrorIs(t, err, asset.ErrNotSupported, "WebsocketSubmitOrder must reject unsupported assets")
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	setV5PositionModeEndpoint(t, h, "single_side")
	_, err := h.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WebsocketSubmitOrders must reject empty submissions")
	responses, err := h.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
		Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures,
		Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1,
	}})
	require.NoError(t, err, "WebsocketSubmitOrders must not error")
	require.Len(t, responses, 1, "one response must be returned")
	assert.Equal(t, "1", responses[0].OrderID, "order ID should match")
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	require.ErrorIs(t, h.WebsocketCancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "WebsocketCancelOrder must reject nil cancellations")
	require.NoError(t, h.WebsocketCancelOrder(t.Context(), &order.Cancel{
		Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, ClientOrderID: "2",
	}), "WebsocketCancelOrder must accept a client order ID")
	require.ErrorIs(t, h.WebsocketCancelOrder(t.Context(), &order.Cancel{
		Pair: btcusdtPair, AssetType: asset.Spot, OrderID: "1",
	}), asset.ErrNotSupported, "WebsocketCancelOrder must reject unsupported assets")
}

func TestGetOrderHistoryValidation(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.ErrorContains(t, err, "currency must be supplied", "GetOrderHistory must require a currency pair for spot")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tt.pair},
				AssetType: tt.asset,
				Side:      order.AnySide,
				StartTime: time.Now().AddDate(0, 0, -1),
				EndTime:   time.Now(),
			})
			require.ErrorIs(t, err, tt.expectedErr, "GetOrderHistory must return the expected branch error")
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	c, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	require.Greater(t, len(c), 2, "Must get more than 2 chains")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/query/deposit-withdraw", `{
		"status":"ok",
		"data":[{"type":"withdraw","currency":"btc","tx-hash":"tx","chain":"btc","amount":1,"address":"address","fee":0.1,"state":"confirmed","created-at":1772779491561}]
	}`, nil)
	history, err := h.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	require.Len(t, history, 1, "one withdrawal must be returned")
	assert.Equal(t, "tx", history[0].TransferID, "transfer ID should match")
	assert.Equal(t, "confirmed", history[0].Status, "status should match")
}

func TestGetWithdrawalsHistoryUnsupportedAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject unsupported assets")
}

func TestCompatibleVars(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		side           string
		orderPriceType string
		status         int64
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
		expectedTIF    order.TimeInForce
		wantErr        bool
	}{
		{name: "buy limit active", side: "buy", orderPriceType: "limit", status: 3, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Active},
		{name: "sell market filled", side: "sell", orderPriceType: "opponent", status: 6, expectedSide: order.Sell, expectedType: order.Market, expectedStatus: order.Filled},
		{name: "post only cancelled", side: "buy", orderPriceType: "post_only", status: 7, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Cancelled, expectedTIF: order.PostOnly},
		{name: "invalid side", side: "hold", orderPriceType: "limit", status: 3, wantErr: true},
		{name: "invalid order price type", side: "buy", orderPriceType: "stop", status: 3, wantErr: true},
		{name: "invalid status", side: "buy", orderPriceType: "limit", status: 99, wantErr: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := compatibleVars(tt.side, tt.orderPriceType, tt.status)
			if tt.wantErr {
				require.Error(t, err, "compatibleVars must reject invalid input")
				return
			}
			require.NoError(t, err, "compatibleVars must not error")
			assert.Equal(t, tt.expectedSide, resp.Side, "side should match")
			assert.Equal(t, tt.expectedType, resp.OrderType, "order type should match")
			assert.Equal(t, tt.expectedStatus, resp.Status, "status should match")
			assert.Equal(t, tt.expectedTIF, resp.TimeInForce, "time in force should match")
		})
	}
}

func TestSetOrderSideStatusAndType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		state          string
		requestType    string
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
	}{
		{
			name:           "buy market",
			state:          "filled",
			requestType:    string(SpotNewOrderRequestTypeBuyMarket),
			expectedSide:   order.Buy,
			expectedType:   order.Market,
			expectedStatus: order.Filled,
		},
		{
			name:           "sell market",
			state:          "partial-filled",
			requestType:    string(SpotNewOrderRequestTypeSellMarket),
			expectedSide:   order.Sell,
			expectedType:   order.Market,
			expectedStatus: order.PartiallyFilled,
		},
		{
			name:           "buy limit",
			state:          "submitted",
			requestType:    string(SpotNewOrderRequestTypeBuyLimit),
			expectedSide:   order.Buy,
			expectedType:   order.Limit,
			expectedStatus: order.New,
		},
		{
			name:           "sell limit",
			state:          "canceled",
			requestType:    string(SpotNewOrderRequestTypeSellLimit),
			expectedSide:   order.Sell,
			expectedType:   order.Limit,
			expectedStatus: order.Cancelled,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detail := &order.Detail{Exchange: e.Name}
			setOrderSideStatusAndType(tt.state, tt.requestType, detail)
			assert.Equal(t, tt.expectedSide, detail.Side, "side should match")
			assert.Equal(t, tt.expectedType, detail.Type, "order type should match")
			assert.Equal(t, tt.expectedStatus, detail.Status, "status should match")
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	resp, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")
	updatePairsOnce(t, e)
	err := e.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, currency.NewBTCUSDT())
	if err != nil {
		require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)
	}

	resp, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "USDT-margined predicted funding rates must be rejected")
	assert.Empty(t, resp, "unsupported predicted funding rates should not be returned")

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	err = e.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, currency.NewBTCUSD())
	require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.True(t, is)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		endpoint exchange.URL
		path     string
		response string
	}{
		{
			name: "spot", item: asset.Spot, pair: btcusdtPair, endpoint: exchange.RestSpot, path: htxMarketTickers,
			response: `{"status":"ok","data":[{"symbol":"btcusdt","open":1,"high":3,"low":1,"close":2,"amount":4,"vol":8,"bid":1.9,"bidSize":2,"ask":2.1,"askSize":3}]}`,
		},
		{
			name: "coin margined futures", item: asset.CoinMarginedFutures, pair: btcusdPair, endpoint: exchange.RestFutures, path: htxBatchCoinMarginSwapContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USD","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
		{
			name: "USDT margined futures", item: asset.USDTMarginedFutures, pair: btcusdtPair, endpoint: exchange.RestFutures, path: htxBatchLinearSwapContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USDT","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
		{
			name: "delivery futures", item: asset.Futures, pair: btccwPair, endpoint: exchange.RestFutures, path: htxBatchContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC_CW","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tc.endpoint, http.MethodGet, tc.path, tc.response, nil)
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, false), "available pair must be set")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, true), "enabled pair must be set")
			require.NoError(t, h.UpdateTickers(t.Context(), tc.item), "UpdateTickers must not error")
		})
	}

	for _, tc := range []struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		path     string
		response string
	}{
		{
			name: "coin margined empty bid", item: asset.CoinMarginedFutures, pair: btcusdPair, path: htxBatchCoinMarginSwapContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USD","bid":[],"ask":[2.1,3]}]}`,
		},
		{
			name: "USDT margined empty ask", item: asset.USDTMarginedFutures, pair: btcusdtPair, path: htxBatchLinearSwapContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USDT","bid":[1.9,2],"ask":[]}]}`,
		},
		{
			name: "delivery futures empty bid", item: asset.Futures, pair: btccwPair, path: htxBatchContracts,
			response: `{"status":"ok","ticks":[{"contract_code":"BTC_CW","bid":[],"ask":[2.1,3]}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, tc.path, tc.response, nil)
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, false), "available pair must be set")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, true), "enabled pair must be set")
			err := h.UpdateTickers(t.Context(), tc.item)
			if strings.Contains(tc.name, "ask") {
				require.ErrorIs(t, err, errInvalidAskData, "UpdateTickers must reject an empty ask")
			} else {
				require.ErrorIs(t, err, errInvalidBidData, "UpdateTickers must reject an empty bid")
			}
		})
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  btccwPair.Base.Item,
		Quote: btccwPair.Quote.Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			require.NotEmpty(t, pairs, "GetPairs must return pairs")
			for _, p := range pairs {
				l, err := e.GetOrderExecutionLimits(a, p)
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
				assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
				assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			}
		})
	}
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported)
	})
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	c, err := e.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.True(t, c, "Bootstrap should return true to continue")

	e.futureContractCodes = nil
	e.Features.Enabled.AutoPairUpdates = false
	_, err = e.Bootstrap(t.Context())
	require.NoError(t, err)
	require.NotNil(t, e.futureContractCodes)
}

var (
	updatePairsMutex         sync.Mutex
	futureContractCodesCache map[string]currency.Code
)

// updatePairsOnce updates the pairs once, and ensures a future dated contract is enabled

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		endpoint exchange.URL
		path     string
		orders   []order.Cancel
		response string
		statusID string
	}{
		{
			name: "spot", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel",
			orders:   []order.Cancel{{ClientOrderID: "client-1", AssetType: asset.Spot, Pair: btcusdtPair}},
			response: `{"status":"ok","data":{"success":["client-1"],"failed":[]}}`, statusID: "client-1",
		},
		{
			name: "coin margined futures", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancel",
			orders:   []order.Cancel{{OrderID: "1", AssetType: asset.CoinMarginedFutures, Pair: btcusdPair}},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`, statusID: "1",
		},
		{
			name: "USDT margined futures", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_batch_orders",
			orders:   []order.Cancel{{OrderID: "1", AssetType: asset.USDTMarginedFutures, Pair: btcusdtPair}},
			response: `{"code":200,"data":[{"code":200,"order_id":"1"}]}`, statusID: "1",
		},
		{
			name: "delivery futures", endpoint: exchange.RestFutures, path: fCancelOrder,
			orders:   []order.Cancel{{OrderID: "1", AssetType: asset.Futures, Pair: btccwPair}},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`, statusID: "1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, tt.response, nil)
			resp, err := h.CancelBatchOrders(t.Context(), tt.orders)
			require.NoError(t, err, "CancelBatchOrders must not error")
			assert.Equal(t, htxStatusSuccess, resp.Status[tt.statusID], "cancellation status should indicate success")
		})
	}
}

func TestGetHistoricCandlesUSDTMargined(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Minute)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapKline, r.URL.Path, "USDT-margined kline path should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":` + strconv.FormatInt(now.Add(-time.Minute).Unix(), 10) + `,"open":1,"high":2,"low":0.5,"close":1.5,"vol":10}]}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, false), "available pair must be set")
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, true), "enabled pair must be set")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

	got, err := h.GetHistoricCandles(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
	require.NoError(t, err, "GetHistoricCandles must not error for USDT-margined futures")
	require.NotEmpty(t, got.Candles, "candles must be returned")

	got, err = h.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for USDT-margined futures")
	require.NotEmpty(t, got.Candles, "extended candles must be returned")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject nil fee builder")

	_, err = h.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "fee type should fall back when credentials are not valid")
}

package htx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetLinearSwapMarkets(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapMarkets, r.URL.Path, "market endpoint path should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":[{"contract_code":"BTC-USDT"}]}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetLinearSwapMarkets(t.Context(), btcusdtPair, "cross", "swap", "futures")
	require.NoError(t, err, "GetLinearSwapMarkets must not error")
	require.Len(t, resp, 1, "decoded markets must be returned")
	assert.Equal(t, "BTC-USDT", resp[0].ContractCode, "contract code should decode")
}

func TestGetLinearSwapMarketDepth(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapMarketDepth, r.URL.Path, "market depth endpoint path should match")
		_, _ = w.Write([]byte(`{"status":"ok","tick":{"id":123}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetLinearSwapMarketDepth(t.Context(), btcusdtPair, "step0")
	require.NoError(t, err, "GetLinearSwapMarketDepth must not error")
	assert.Equal(t, int64(123), resp.Tick.ID, "depth ID should decode")
}

func TestGetLinearSwapMarketOverview(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapMarketOverview, r.URL.Path, "market overview endpoint path should match")
		_, _ = w.Write([]byte(`{"status":"ok","ch":"market.BTC-USDT.detail","tick":{"id":123}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetLinearSwapMarketOverview(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetLinearSwapMarketOverview must not error")
	assert.Equal(t, int64(123), resp.Tick.ID, "overview ID should decode")
}

func TestGetLinearSwapKlineData(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapKline, r.URL.Path, "kline endpoint path should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1604312615,"close":1}]}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetLinearSwapKlineData(t.Context(), btcusdtPair, "1min", 10, time.Time{}, time.Time{})
	require.NoError(t, err, "GetLinearSwapKlineData must not error")
	require.Len(t, resp.Data, 1, "decoded candles must be returned")
	assert.Equal(t, float64(1), resp.Data[0].Close, "close price should decode")
}

func TestGetLinearSwapBatchTrades(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapBatchTrades, r.URL.Path, "batch trades endpoint path should match")
		_, _ = w.Write([]byte(`{"status":"ok","id":123,"data":[{"id":1,"price":10}]}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetLinearSwapBatchTrades(t.Context(), btcusdtPair, 10)
	require.NoError(t, err, "GetLinearSwapBatchTrades must not error")
	assert.Equal(t, int64(123), resp.ID, "batch ID should decode")
}

func TestGetV5OpenInterest(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/market/open_interest", r.URL.Path, "open interest endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"success":true,"data":{"contract_code":"BTC-USDT"}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetV5OpenInterest(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetV5OpenInterest must not error")
	require.NotNil(t, resp, "decoded open interest must be returned")
	assert.Equal(t, "BTC-USDT", resp.Data.ContractCode, "contract code should decode")
}

func TestGetV5AccountBalance(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/account/balance", r.URL.Path, "balance endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":{"state":"working"}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetV5AccountBalance(t.Context())
	require.NoError(t, err, "GetV5AccountBalance must not error")
	require.NotNil(t, resp, "decoded balance must be returned")
	assert.Equal(t, "working", resp.Data.State, "account state should decode")
}

func TestPlaceV5Order(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/order", r.URL.Path, "place order endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":{"order_id":"123"}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.PlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceV5Order must reject nil request")
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.PlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "limit", Volume: types.Number(1)})
	require.NoError(t, err, "PlaceV5Order must not error")
	require.NotNil(t, resp, "decoded order response must be returned")
	assert.Equal(t, "123", resp.Data.OrderID, "order ID should decode")
}

func TestCancelV5Order(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/cancel_order", r.URL.Path, "cancel order endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":{"order_id":"123"}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.CancelV5Order(t.Context(), btcusdtPair, "1", "")
	require.NoError(t, err, "CancelV5Order must not error")
	require.NotNil(t, resp, "decoded cancellation must be returned")
	assert.Equal(t, "123", resp.Data.OrderID, "order ID should decode")
}

func TestCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/cancel_all_orders", r.URL.Path, "cancel all endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":[{"order_id":"123"}]}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.CancelAllV5Orders(t.Context(), btcusdtPair, "buy", "long")
	require.NoError(t, err, "CancelAllV5Orders must not error")
	require.NotNil(t, resp, "decoded cancellations must be returned")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "123", resp.Data[0].OrderID, "order ID should decode")
}

func TestGetV5Order(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/order", r.URL.Path, "get order endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":{"order_id":"123"}}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetV5Order(t.Context(), btcusdtPair, "cross", "1", "")
	require.NoError(t, err, "GetV5Order must not error")
	require.NotNil(t, resp, "decoded order must be returned")
	assert.Equal(t, "123", resp.Data.OrderID, "order ID should decode")
}

func TestGetV5OpenOrders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/trade/order/opens", r.URL.Path, "open orders endpoint path should match")
		_, _ = w.Write([]byte(`{"code":200,"data":[{"order_id":"123"}]}`))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	resp, err := h.GetV5OpenOrders(t.Context(), btcusdtPair, "cross", "", "", 1, 10, "next")
	require.NoError(t, err, "GetV5OpenOrders must not error")
	require.NotNil(t, resp, "decoded open orders must be returned")
	require.Len(t, resp.Data, 1, "one open order must decode")
	assert.Equal(t, "123", resp.Data[0].OrderID, "order ID should decode")
}

package htx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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

func setupV5HTTPTest(t *testing.T, method, path, response string, check func(*http.Request)) *Exchange {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	return h
}

func TestUSDTFuturesEndpointPaths(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/linear-swap-api/v1/swap_contract_info", linearSwapMarkets, "linear swap contract info endpoint should match HTX docs")
}

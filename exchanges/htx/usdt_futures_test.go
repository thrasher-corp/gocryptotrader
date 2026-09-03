package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestGetLinearSwapMarkets(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, linearSwapMarkets, `{"status":"ok","data":[{"contract_code":"BTC-USDT"}]}`, nil)
	resp, err := h.GetLinearSwapMarkets(t.Context(), btcusdtPair, "cross", "swap", "futures")
	require.NoError(t, err, "GetLinearSwapMarkets must not error")
	require.Len(t, resp, 1, "decoded markets must be returned")
	assert.Equal(t, "BTC-USDT", resp[0].ContractCode, "contract code should decode")
}

func TestGetLinearSwapMarketDepth(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, linearSwapMarketDepth, `{"status":"ok","tick":{"id":123}}`, nil)
	resp, err := h.GetLinearSwapMarketDepth(t.Context(), btcusdtPair, "step0")
	require.NoError(t, err, "GetLinearSwapMarketDepth must not error")
	assert.Equal(t, int64(123), resp.Tick.ID, "depth ID should decode")
}

func TestGetLinearSwapMarketOverview(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, linearSwapMarketOverview, `{"status":"ok","ch":"market.BTC-USDT.detail","tick":{"id":123}}`, nil)
	resp, err := h.GetLinearSwapMarketOverview(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetLinearSwapMarketOverview must not error")
	assert.Equal(t, int64(123), resp.Tick.ID, "overview ID should decode")
}

func TestGetLinearSwapKlineData(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, linearSwapKline, `{"status":"ok","data":[{"id":1604312615,"close":1}]}`, nil)
	resp, err := h.GetLinearSwapKlineData(t.Context(), btcusdtPair, "1min", 10, time.Time{}, time.Time{})
	require.NoError(t, err, "GetLinearSwapKlineData must not error")
	require.Len(t, resp.Data, 1, "decoded candles must be returned")
	assert.Equal(t, float64(1), resp.Data[0].Close, "close price should decode")
}

func TestGetLinearSwapBatchTrades(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, linearSwapBatchTrades, `{"status":"ok","id":123,"data":[{"id":1,"price":10}]}`, nil)
	resp, err := h.GetLinearSwapBatchTrades(t.Context(), btcusdtPair, 10)
	require.NoError(t, err, "GetLinearSwapBatchTrades must not error")
	assert.Equal(t, int64(123), resp.ID, "batch ID should decode")
}

func TestUSDTFuturesEndpointPaths(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/linear-swap-api/v1/swap_contract_info", linearSwapMarkets, "linear swap contract info endpoint should match HTX docs")
}

package okx

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestTradeScopeFromInstrumentID(t *testing.T) {
	t.Parallel()

	require.Empty(t, tradeScopeFromInstrumentID(""))
	require.Equal(t, "BTC-USDT", tradeScopeFromInstrumentID("btc-usdt"))
	require.Equal(t, "BTC-USD", tradeScopeFromInstrumentID("BTC-USD-241227-50000-C"))
}

func TestIsOptionInstrumentID(t *testing.T) {
	t.Parallel()

	require.True(t, isOptionInstrumentID("BTC-USD-241227-50000-C"), "dash-delimited option ID must be detected")
	require.True(t, isOptionInstrumentID("BTC_USD_241227_50000_C"), "underscore-delimited option ID must be detected")
	require.False(t, isOptionInstrumentID("BTC-USDT"), "spot-style instrument ID must not be detected as option")
}

func TestTradeScopeCountsFromPlaceOrders(t *testing.T) {
	t.Parallel()

	args := []PlaceOrderRequestParam{
		{InstrumentID: "BTC-USDT"},
		{InstrumentID: "BTC-USDT"},
		{InstrumentID: "ETH-USDT"},
		{InstrumentID: "BTC-USD-241227-50000-C"},
		{InstrumentID: "BTC-USD-241227-45000-P"},
	}
	got := tradeScopeCountsFromPlaceOrders(args)
	require.Equal(t, 2, got["BTC-USDT"])
	require.Equal(t, 1, got["ETH-USDT"])
	require.Equal(t, 2, got["BTC-USD"])
}

func TestTradeScopeCountsFromCancelOrders(t *testing.T) {
	t.Parallel()

	args := []CancelOrderRequestParam{
		{InstrumentID: "SOL-USDT"},
		{InstrumentID: "SOL-USDT"},
		{InstrumentID: "SOL-USD-241227-100-P"},
	}
	got := tradeScopeCountsFromCancelOrders(args)
	require.Equal(t, 2, got["SOL-USDT"])
	require.Equal(t, 1, got["SOL-USD"])
}

func TestTradeScopeCountsFromAmendOrders(t *testing.T) {
	t.Parallel()

	args := []AmendOrderRequestParams{
		{InstrumentID: "XRP-USDT"},
		{InstrumentID: "XRP-USDT"},
		{InstrumentID: "XRP-USDT"},
	}
	got := tradeScopeCountsFromAmendOrders(args)
	require.Equal(t, 3, got["XRP-USDT"])
}

func TestCountByOrder(t *testing.T) {
	t.Parallel()

	require.Zero(t, countByOrder([]string(nil)))
	require.Equal(t, 3, countByOrder([]int{1, 2, 3}))
}

func TestToRateLimitWeight(t *testing.T) {
	t.Parallel()

	require.Zero(t, toRateLimitWeight(0))
	require.Zero(t, toRateLimitWeight(-1))
	require.Equal(t, uint8(12), toRateLimitWeight(12))
	require.Equal(t, uint8(math.MaxUint8), toRateLimitWeight(math.MaxInt32))
}

func TestGetOrCreateTradeScopedLimiter(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	first := ex.getOrCreateTradeScopedLimiter(tradeRateLimitPlaceSingle, " btc-usdt ")
	second := ex.getOrCreateTradeScopedLimiter(tradeRateLimitPlaceSingle, "BTC-USDT")
	require.Same(t, first, second, "scoped limiter must be cached by normalised key")

	batch := ex.getOrCreateTradeScopedLimiter(tradeRateLimitPlaceBatch, "BTC-USDT")
	require.NotNil(t, batch, "batch limiter must be created")

	ex.tradeScopedLimiters.Store("place-single|BTC-USDT", "not-a-limiter")
	recovered := ex.getOrCreateTradeScopedLimiter(tradeRateLimitPlaceSingle, "BTC-USDT")
	require.NotNil(t, recovered, "limiter must be recreated if stored type is invalid")
}

func TestApplyTradeScopeRateLimit(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, ex.applyTradeScopeRateLimit(t.Context(), tradeRateLimitPlaceSingle, nil), "empty scope map must not error")
	require.NoError(t, ex.applyTradeScopeRateLimit(t.Context(), tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 0}), "non-positive scope weights must be ignored")
	require.NoError(t, ex.applyTradeScopeRateLimit(t.Context(), tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 1}), "valid scope weight must not error")
}

func TestApplyTradeSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, ex.applyTradeSubAccountRateLimit(t.Context(), 0), "zero order count must not error")
	require.NoError(t, ex.applyTradeSubAccountRateLimit(t.Context(), 1), "single order count must not error")

	ex.tradeSubAccountLimiter.Store("structural-subaccount-limit", "bad-type")
	err := ex.applyTradeSubAccountRateLimit(t.Context(), 1)
	require.Error(t, err, "invalid stored limiter type must error")
	assert.Contains(t, err.Error(), "invalid subaccount limiter type", "error should mention invalid limiter type")

	ex.tradeSubAccountLimiter = sync.Map{}
	ex.tradeSubAccountLimiter.Store("structural-subaccount-limit", request.NewRateLimitWithWeight(twoSecondsInterval, 1000, 1))
	require.NoError(t, ex.applyTradeSubAccountRateLimit(t.Context(), 3), "valid stored limiter must allow weighted call")
}

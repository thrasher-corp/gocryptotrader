package okx

import (
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

func TestToRateLimitWeight(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { toRateLimitWeight(0) }, "zero weight must panic")
	require.Panics(t, func() { toRateLimitWeight(-1) }, "negative weight must panic")
	require.Equal(t, uint8(12), toRateLimitWeight(12))
	require.Equal(t, uint8(255), toRateLimitWeight(300))
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

func TestTradeScopeRateLimits(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	limiters, weights := ex.tradeScopeRateLimits(tradeRateLimitPlaceSingle, nil)
	require.Empty(t, limiters, "empty scope map must not return limiters")
	require.Empty(t, weights, "empty scope map must not return weights")

	limiters, weights = ex.tradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 0})
	require.Empty(t, limiters, "non-positive scope weights must be ignored")
	require.Empty(t, weights, "non-positive scope weights must be ignored")

	limiters, weights = ex.tradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 2})
	require.Len(t, limiters, 1, "valid scope weight must return one limiter")
	require.Equal(t, []request.Weight{2}, weights, "valid scope weight must return one weight")
}

func TestTradeSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	limit, err := ex.tradeSubAccountRateLimit(0)
	require.NoError(t, err, "zero order count must not error")
	assert.Nil(t, limit, "zero order count should not return a limit")

	limit, err = ex.tradeSubAccountRateLimit(1)
	require.NoError(t, err, "single order count must not error")
	assert.NotNil(t, limit, "limiter should be set")

	ex.tradeSubAccountLimiter.Store("structural-subaccount-limit", "bad-type")
	_, err = ex.tradeSubAccountRateLimit(1)
	require.Error(t, err, "invalid stored limiter type must error")
	assert.Contains(t, err.Error(), "invalid subaccount limiter type", "error should mention invalid limiter type")

	ex.tradeSubAccountLimiter = sync.Map{}
	ex.tradeSubAccountLimiter.Store("structural-subaccount-limit", request.NewRateLimitWithWeight(twoSecondsInterval, 1000, 1))
	limit, err = ex.tradeSubAccountRateLimit(3)
	require.NoError(t, err, "valid stored limiter must return a weighted limit")
	assert.NotNil(t, limit, "limiter should be set")
}

func TestTradeRateLimits(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	limits, err := ex.tradeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 2}, 3)
	require.NoError(t, err, "valid trade rate limits must not error")
	require.Len(t, limits.limiters, 2, "valid trade rate limits must return scoped and subaccount limiters")
	require.Equal(t, []request.Weight{2, 3}, limits.weights, "valid trade rate limits must return matching weights")

	ex.tradeSubAccountLimiter.Store("structural-subaccount-limit", "bad-type")
	_, err = ex.tradeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 2}, 3)
	require.Error(t, err, "invalid subaccount limiter type must error")
}

package okx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	tradeRateLimitBTCUSDT          = "BTC-USDT"
	tradeRateLimitBTCUSDOptionCall = "BTC-USD-241227-50000-C"
	tradeRateLimitBTCUSDOptionPut  = "BTC-USD-241227-45000-P"
)

func TestTradeScopeFromInstrumentID(t *testing.T) {
	t.Parallel()

	require.Empty(t, tradeScopeFromInstrumentID(""))
	require.Equal(t, tradeRateLimitBTCUSDT, tradeScopeFromInstrumentID("btc-usdt"))
	require.Equal(t, "BTC-USD", tradeScopeFromInstrumentID(tradeRateLimitBTCUSDOptionCall))
}

func TestIsOptionInstrumentID(t *testing.T) {
	t.Parallel()

	require.True(t, isOptionInstrumentID(tradeRateLimitBTCUSDOptionCall), "dash-delimited option ID must be detected")
	require.True(t, isOptionInstrumentID("BTC_USD_241227_50000_C"), "underscore-delimited option ID must be detected")
	require.False(t, isOptionInstrumentID(tradeRateLimitBTCUSDT), "spot-style instrument ID must not be detected as option")
}

func TestTradeScopeCountsFromPlaceOrders(t *testing.T) {
	t.Parallel()

	args := []PlaceOrderRequestParam{
		{InstrumentID: tradeRateLimitBTCUSDT},
		{InstrumentID: tradeRateLimitBTCUSDT},
		{InstrumentID: "ETH-USDT"},
		{InstrumentID: tradeRateLimitBTCUSDOptionCall},
		{InstrumentID: tradeRateLimitBTCUSDOptionPut},
	}
	got := tradeScopeCountsFromPlaceOrders(args)
	require.Equal(t, 2, got[tradeRateLimitBTCUSDT])
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

func TestRateLimitWeight(t *testing.T) {
	t.Parallel()

	require.Zero(t, boundRateLimitWeight(0), "zero weight must be ignored")
	require.Zero(t, boundRateLimitWeight(-1), "negative weight must be ignored")
	require.Equal(t, uint8(12), boundRateLimitWeight(12), "positive weight must be preserved")
	require.Equal(t, uint8(255), boundRateLimitWeight(300), "large weight must clamp to uint8 max")
}

func TestValidateOKXBatchOrderCount(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateOKXBatchOrderCount(maxOKXBatchOrders), "maximum batch order count must be valid")
	require.ErrorIs(t, validateOKXBatchOrderCount(maxOKXBatchOrders+1), errExceedLimit, "oversized batch order count must return limit error")
}

func TestTradeRateLimiterGetOrCreateScopedLimiter(t *testing.T) {
	t.Parallel()

	limiter := new(tradeRateLimiter)
	first := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, " btc-usdt ")
	second := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
	require.Same(t, first, second, "scoped limiter must be cached by normalised key")

	batch := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceBatch, tradeRateLimitBTCUSDT)
	require.NotNil(t, batch, "batch limiter must be created")
	require.NotSame(t, first, batch, "different limiter classes must use different buckets")
}

func TestTradeRateLimiterGetOrCreateScopedLimiterEvictsOldest(t *testing.T) {
	t.Parallel()

	limiter := new(tradeRateLimiter)
	for i := range maxTradeScopedLimiters {
		limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, fmt.Sprintf("BTC-USDT-%d", i))
	}

	firstKey := string(tradeRateLimitPlaceSingle) + "|BTC-USDT-0"
	require.Contains(t, limiter.scopedLimiters, firstKey, "oldest limiter must remain before capacity is exceeded")

	limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "BTC-USDT-overflow")
	require.NotContains(t, limiter.scopedLimiters, firstKey, "oldest limiter must be evicted when capacity is exceeded")
	require.Len(t, limiter.scopedLimiters, maxTradeScopedLimiters, "scoped limiter cache must remain bounded")
}

func TestTradeRateLimiterAdditionalTradeScopeRateLimits(t *testing.T) {
	t.Parallel()

	limiter := new(tradeRateLimiter)
	additionalRateLimits := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, nil)
	require.Empty(t, additionalRateLimits, "empty scope map must not return additional rate limits")

	additionalRateLimits = limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 0})
	require.Empty(t, additionalRateLimits, "non-positive scope weights must be ignored")

	additionalRateLimits = limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 2})
	require.Len(t, additionalRateLimits, 1, "valid scope weight must return one additional rate limit")
	require.NotNil(t, additionalRateLimits[0].Limiter, "valid scope limit must include limiter")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].Weight, "valid scope weight must return one weight")
}

func TestTradeRateLimiterSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	limiter := new(tradeRateLimiter)
	limit, ok := limiter.subAccountRateLimit(0)
	assert.False(t, ok, "zero order count should not return a limit")
	assert.Empty(t, limit, "zero order count should not return a limit")

	limit, ok = limiter.subAccountRateLimit(1)
	require.True(t, ok, "single order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(1), limit.Weight, "weight should match order count")

	limit, ok = limiter.subAccountRateLimit(3)
	require.True(t, ok, "positive order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(3), limit.Weight, "weight should match order count")
}

func TestTradeRateLimiterAdditionalTradeRateLimits(t *testing.T) {
	t.Parallel()

	limiter := new(tradeRateLimiter)
	additionalRateLimits := limiter.additionalTradeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 2}, 3)
	require.Len(t, additionalRateLimits, 2, "valid trade rate limits must return scoped and subaccount limiters")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].Weight, "valid trade rate limits must return scope weight")
	require.Equal(t, request.Weight(3), additionalRateLimits[1].Weight, "valid trade rate limits must return subaccount weight")
}

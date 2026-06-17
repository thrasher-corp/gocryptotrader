package okx

import (
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
	got, err := tradeScopeCountsFromPlaceOrders(args)
	require.NoError(t, err, "tradeScopeCountsFromPlaceOrders must not error")
	require.Equal(t, 2, got[tradeRateLimitBTCUSDT])
	require.Equal(t, 1, got["ETH-USDT"])
	require.Equal(t, 2, got["BTC-USD"])

	_, err = tradeScopeCountsFromPlaceOrders([]PlaceOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
}

func TestTradeScopeCountsFromCancelOrders(t *testing.T) {
	t.Parallel()

	args := []CancelOrderRequestParam{
		{InstrumentID: "SOL-USDT"},
		{InstrumentID: "SOL-USDT"},
		{InstrumentID: "SOL-USD-241227-100-P"},
	}
	got, err := tradeScopeCountsFromCancelOrders(args)
	require.NoError(t, err, "tradeScopeCountsFromCancelOrders must not error")
	require.Equal(t, 2, got["SOL-USDT"])
	require.Equal(t, 1, got["SOL-USD"])

	_, err = tradeScopeCountsFromCancelOrders([]CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
}

func TestTradeScopeCountsFromAmendOrders(t *testing.T) {
	t.Parallel()

	args := []AmendOrderRequestParams{
		{InstrumentID: "XRP-USDT"},
		{InstrumentID: "XRP-USDT"},
		{InstrumentID: "XRP-USDT"},
	}
	got, err := tradeScopeCountsFromAmendOrders(args)
	require.NoError(t, err, "tradeScopeCountsFromAmendOrders must not error")
	require.Equal(t, 3, got["XRP-USDT"])

	_, err = tradeScopeCountsFromAmendOrders([]AmendOrderRequestParams{{}})
	require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
}

func TestClampWeight(t *testing.T) {
	t.Parallel()

	require.Zero(t, clampWeight(0), "zero weight must be ignored")
	require.Zero(t, clampWeight(-1), "negative weight must be ignored")
	require.Equal(t, request.Weight(12), clampWeight(12), "positive weight must be preserved")
	require.Equal(t, request.Weight(255), clampWeight(300), "large weight must clamp to uint8 max")
}

func TestValidateBatchOrderCount(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateBatchOrderCount(maxBatchOrders), "maximum batch order count must be valid")
	require.ErrorIs(t, validateBatchOrderCount(maxBatchOrders+1), errExceedLimit, "oversized batch order count must return limit error")
}

func TestTradeRateLimiterGetOrCreateScopedLimiter(t *testing.T) {
	t.Parallel()

	limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
	first, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, " btc-usdt ")
	require.NoError(t, err, "getOrCreateScopedLimiter must not error")
	second, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
	require.NoError(t, err, "getOrCreateScopedLimiter must not error")
	require.Same(t, first, second, "scoped limiter must be cached by normalised key")

	batch, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceBatch, tradeRateLimitBTCUSDT)
	require.NoError(t, err, "getOrCreateScopedLimiter must not error")
	require.NotNil(t, batch, "batch limiter must be created")
	require.NotSame(t, first, batch, "different limiter classes must use different buckets")

	_, err = limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "")
	require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty scope must return missing scope error")

	_, err = new(tradeRateLimiter).getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
	require.ErrorIs(t, err, errTradeRateLimiterNotInitialised, "uninitialised limiter must return expected error")
}

func TestTradeRateLimiterAdditionalTradeScopeRateLimits(t *testing.T) {
	t.Parallel()

	limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
	additionalRateLimits, err := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, nil)
	require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty scope map must return missing scope error")
	require.Empty(t, additionalRateLimits, "empty scope map must not return additional rate limits")

	additionalRateLimits, err = limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 0})
	require.ErrorIs(t, err, errInvalidTradeRateLimitWeight, "non-positive scope weights must return invalid weight error")
	require.Empty(t, additionalRateLimits, "non-positive scope weights must not return additional rate limits")

	additionalRateLimits, err = limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 2})
	require.NoError(t, err, "valid scope weight must not error")
	require.Len(t, additionalRateLimits, 1, "valid scope weight must return one additional rate limit")
	require.NotNil(t, additionalRateLimits[0].Limiter, "valid scope limit must include limiter")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].WeightOverride, "valid scope weight must return one weight")
}

func TestTradeRateLimiterSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	limiter := &tradeRateLimiter{subAccountLimiter: newTradeSubAccountRateLimiter()}
	limit, ok, err := limiter.subAccountRateLimit(0)
	require.NoError(t, err, "zero order count must not error")
	assert.False(t, ok, "zero order count should not return a limit")
	assert.Empty(t, limit, "zero order count should not return a limit")

	limit, ok, err = limiter.subAccountRateLimit(1)
	require.NoError(t, err, "single order count must not error")
	require.True(t, ok, "single order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(1), limit.WeightOverride, "weight should match order count")

	limit, ok, err = limiter.subAccountRateLimit(3)
	require.NoError(t, err, "positive order count must not error")
	require.True(t, ok, "positive order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(3), limit.WeightOverride, "weight should match order count")

	_, _, err = new(tradeRateLimiter).subAccountRateLimit(1)
	require.ErrorIs(t, err, errTradeRateLimiterNotInitialised, "uninitialised limiter must return expected error")
}

func TestTradeRateLimiterAdditionalTradeRateLimits(t *testing.T) {
	t.Parallel()

	limiter := &tradeRateLimiter{
		scopedLimiters:    make(map[tradeRateLimitKey]*request.RateLimiterWithWeight),
		subAccountLimiter: newTradeSubAccountRateLimiter(),
	}
	additionalRateLimits, err := limiter.additionalTradeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 2}, 3)
	require.NoError(t, err, "valid trade rate limits must not error")
	require.Len(t, additionalRateLimits, 2, "valid trade rate limits must return scoped and subaccount limiters")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].WeightOverride, "valid trade rate limits must return scope weight")
	require.Equal(t, request.Weight(3), additionalRateLimits[1].WeightOverride, "valid trade rate limits must return subaccount weight")
}

func TestNewTradeSubAccountRateLimiter(t *testing.T) {
	t.Parallel()

	require.NotNil(t, newTradeSubAccountRateLimiter(), "newTradeSubAccountRateLimiter must return a limiter")
}

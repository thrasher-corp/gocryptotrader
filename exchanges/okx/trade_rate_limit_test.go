package okx

import (
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

func TestAdditionalTradeScopeRateLimits(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	additionalRateLimits := ex.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, nil)
	require.Empty(t, additionalRateLimits, "empty scope map must not return additional rate limits")

	additionalRateLimits = ex.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 0})
	require.Empty(t, additionalRateLimits, "non-positive scope weights must be ignored")

	additionalRateLimits = ex.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 2})
	require.Len(t, additionalRateLimits, 1, "valid scope weight must return one additional rate limit")
	require.NotNil(t, additionalRateLimits[0].Limiter, "valid scope limit must include limiter")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].Weight, "valid scope weight must return one weight")
}

func TestTradeSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	limit, ok := ex.tradeSubAccountRateLimit(0)
	assert.False(t, ok, "zero order count should not return a limit")
	assert.Empty(t, limit, "zero order count should not return a limit")

	limit, ok = ex.tradeSubAccountRateLimit(1)
	require.True(t, ok, "single order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(1), limit.Weight, "weight should match order count")

	limit, ok = ex.tradeSubAccountRateLimit(3)
	require.True(t, ok, "positive order count must return a limit")
	assert.NotNil(t, limit.Limiter, "limiter should be set")
	assert.Equal(t, request.Weight(3), limit.Weight, "weight should match order count")
}

func TestAdditionalTradeRateLimits(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	additionalRateLimits := ex.additionalTradeRateLimits(tradeRateLimitPlaceSingle, map[string]int{"BTC-USDT": 2}, 3)
	require.Len(t, additionalRateLimits, 2, "valid trade rate limits must return scoped and subaccount limiters")
	require.Equal(t, request.Weight(2), additionalRateLimits[0].Weight, "valid trade rate limits must return scope weight")
	require.Equal(t, request.Weight(3), additionalRateLimits[1].Weight, "valid trade rate limits must return subaccount weight")
}

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

	testCases := []struct {
		name          string
		instrumentID  string
		expectedScope string
		expectedError error
	}{
		{
			name:          "empty instrument ID",
			expectedError: errMissingTradeRateLimitScope,
		},
		{
			name:          "blank instrument ID",
			instrumentID:  " ",
			expectedError: errMissingTradeRateLimitScope,
		},
		{
			name:          "spot instrument ID",
			instrumentID:  tradeRateLimitBTCUSDT,
			expectedScope: tradeRateLimitBTCUSDT,
		},
		{
			name:          "instrument ID casing remains unchanged",
			instrumentID:  "btc-usdt",
			expectedScope: "btc-usdt",
		},
		{
			name:          "option instrument ID",
			instrumentID:  tradeRateLimitBTCUSDOptionCall,
			expectedScope: "BTC-USD",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scope, err := tradeScopeFromInstrumentID(tc.instrumentID)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "tradeScopeFromInstrumentID must return expected error")
				assert.Empty(t, scope, "error response should not return a scope")
				return
			}
			require.NoError(t, err, "tradeScopeFromInstrumentID must not error")
			assert.Equal(t, tc.expectedScope, scope, "trade scope should preserve the exchange identifier")
		})
	}
}

func TestOptionInstrumentFamily(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		instrumentID  string
		expected      string
		expectedError error
	}{
		{name: "dash-delimited", instrumentID: "BTC-USD-240329-70000-C", expected: "BTC-USD"},
		{name: "underscore-delimited", instrumentID: "ETH_USD_240329_3500_P", expected: "ETH_USD"},
		{name: "invalid", instrumentID: "INVALID", expectedError: errMissingTradeRateLimitScope},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			family, err := optionInstrumentFamily(tc.instrumentID)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "optionInstrumentFamily must return expected error")
				assert.Empty(t, family, "error response should not return a family")
				return
			}
			require.NoError(t, err, "optionInstrumentFamily must not error")
			assert.Equal(t, tc.expected, family, "option family should match")
		})
	}
}

func TestIsOptionInstrumentID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		instrumentID string
		expected     bool
	}{
		{name: "dash-delimited option", instrumentID: tradeRateLimitBTCUSDOptionCall, expected: true},
		{name: "underscore-delimited option", instrumentID: "BTC_USD_241227_50000_C", expected: true},
		{name: "spot", instrumentID: tradeRateLimitBTCUSDT},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isOptionInstrumentID(tc.instrumentID), "option instrument detection should match")
		})
	}
}

func TestTradeScopeCountsFromPlaceOrders(t *testing.T) {
	t.Parallel()

	t.Run("valid orders", func(t *testing.T) {
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
		assert.Equal(t, 2, got[tradeRateLimitBTCUSDT], "BTC-USDT count should match")
		assert.Equal(t, 1, got["ETH-USDT"], "ETH-USDT count should match")
		assert.Equal(t, 2, got["BTC-USD"], "BTC-USD option family count should match")
	})

	t.Run("missing instrument ID", func(t *testing.T) {
		t.Parallel()

		_, err := tradeScopeCountsFromPlaceOrders([]PlaceOrderRequestParam{{}})
		require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
	})
}

func TestTradeScopeCountsFromCancelOrders(t *testing.T) {
	t.Parallel()

	t.Run("valid orders", func(t *testing.T) {
		t.Parallel()

		args := []CancelOrderRequestParam{
			{InstrumentID: "SOL-USDT"},
			{InstrumentID: "SOL-USDT"},
			{InstrumentID: "SOL-USD-241227-100-P"},
		}
		got, err := tradeScopeCountsFromCancelOrders(args)
		require.NoError(t, err, "tradeScopeCountsFromCancelOrders must not error")
		assert.Equal(t, 2, got["SOL-USDT"], "SOL-USDT count should match")
		assert.Equal(t, 1, got["SOL-USD"], "SOL-USD option family count should match")
	})

	t.Run("missing instrument ID", func(t *testing.T) {
		t.Parallel()

		_, err := tradeScopeCountsFromCancelOrders([]CancelOrderRequestParam{{}})
		require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
	})
}

func TestTradeScopeCountsFromAmendOrders(t *testing.T) {
	t.Parallel()

	t.Run("valid orders", func(t *testing.T) {
		t.Parallel()

		args := []AmendOrderRequestParams{
			{InstrumentID: "XRP-USDT"},
			{InstrumentID: "XRP-USDT"},
			{InstrumentID: "XRP-USDT"},
		}
		got, err := tradeScopeCountsFromAmendOrders(args)
		require.NoError(t, err, "tradeScopeCountsFromAmendOrders must not error")
		assert.Equal(t, 3, got["XRP-USDT"], "XRP-USDT count should match")
	})

	t.Run("missing instrument ID", func(t *testing.T) {
		t.Parallel()

		_, err := tradeScopeCountsFromAmendOrders([]AmendOrderRequestParams{{}})
		require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty instrument ID must return missing scope error")
	})
}

func TestTradeScopeCounts(t *testing.T) {
	t.Parallel()

	counts, err := tradeScopeCounts([]string{tradeRateLimitBTCUSDT, tradeRateLimitBTCUSDT, "ETH-USDT"}, func(instrumentID string) string {
		return instrumentID
	})
	require.NoError(t, err, "tradeScopeCounts must not error for valid instrument IDs")
	assert.Equal(t, 2, counts[tradeRateLimitBTCUSDT], "tradeScopeCounts should aggregate duplicate scopes")
	assert.Equal(t, 1, counts["ETH-USDT"], "tradeScopeCounts should retain distinct scopes")
}

func TestBatchOrderWeight(t *testing.T) {
	t.Parallel()

	t.Run("maximum", func(t *testing.T) {
		t.Parallel()
		weight, err := batchOrderWeight(maxBatchOrders)
		require.NoError(t, err, "maximum batch order count must be valid")
		assert.Equal(t, request.Weight(maxBatchOrders), weight, "batch weight should match order count")
	})

	t.Run("over maximum", func(t *testing.T) {
		t.Parallel()
		_, err := batchOrderWeight(maxBatchOrders + 1)
		require.ErrorIs(t, err, errExceedLimit, "oversized batch order count must return limit error")
	})
}

func TestRateLimitWeight(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		count         int
		expected      request.Weight
		expectedError error
	}{
		{name: "negative", count: -1, expectedError: errInvalidTradeRateLimitWeight},
		{name: "zero", expectedError: errInvalidTradeRateLimitWeight},
		{name: "valid", count: 20, expected: 20},
		{name: "maximum", count: 255, expected: 255},
		{name: "over maximum", count: 256, expectedError: errInvalidTradeRateLimitWeight},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			weight, err := rateLimitWeight(tc.count)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "rateLimitWeight must return expected error")
				return
			}
			require.NoError(t, err, "rateLimitWeight must not error")
			assert.Equal(t, tc.expected, weight, "rate-limit weight should match")
		})
	}
}

func TestTradeRateLimiterGetOrCreateScopedLimiter(t *testing.T) {
	t.Parallel()

	t.Run("cached exact key", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		first, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		second, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		assert.Same(t, first, second, "scoped limiter should be cached by exact key")
	})

	t.Run("case-sensitive scope", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		upper, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		lower, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "btc-usdt")
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		assert.NotSame(t, upper, lower, "differently cased instrument IDs should use distinct keys")
	})

	t.Run("limiter class", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		single, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		batch, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceBatch, tradeRateLimitBTCUSDT)
		require.NoError(t, err, "getOrCreateScopedLimiter must not error")
		assert.NotSame(t, single, batch, "different limiter classes should use different buckets")
	})

	t.Run("invalid class", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		_, err := limiter.getOrCreateScopedLimiter("unknown", tradeRateLimitBTCUSDT)
		require.ErrorIs(t, err, errInvalidTradeRateLimitClass, "unknown class must return expected error")
	})

	t.Run("empty scope", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		_, err := limiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "")
		require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty scope must return missing scope error")
	})

	t.Run("uninitialised", func(t *testing.T) {
		t.Parallel()

		_, err := new(tradeRateLimiter).getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, tradeRateLimitBTCUSDT)
		require.ErrorIs(t, err, errTradeRateLimiterNotInitialised, "uninitialised limiter must return expected error")
	})
}

func TestTradeRateLimiterAdditionalTradeScopeRateLimits(t *testing.T) {
	t.Parallel()

	t.Run("missing scope", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, nil)
		require.ErrorIs(t, err, errMissingTradeRateLimitScope, "empty scope map must return missing scope error")
		assert.Empty(t, additionalRateLimits, "empty scope map should not return additional rate limits")
		assert.Zero(t, orderCount, "empty scope map should not return an order count")
	})

	t.Run("invalid class", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits("unknown", map[string]int{tradeRateLimitBTCUSDT: 1})
		require.ErrorIs(t, err, errInvalidTradeRateLimitClass, "unknown class must return expected error")
		assert.Empty(t, additionalRateLimits, "unknown class should not return additional rate limits")
		assert.Zero(t, orderCount, "unknown class should not return an order count")
	})

	t.Run("invalid weight", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 0})
		require.ErrorIs(t, err, errInvalidTradeRateLimitWeight, "non-positive scope weights must return invalid weight error")
		assert.Empty(t, additionalRateLimits, "non-positive scope weights should not return additional rate limits")
		assert.Zero(t, orderCount, "non-positive scope weights should not return an order count")
	})

	t.Run("oversized weight", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 256})
		require.ErrorIs(t, err, errInvalidTradeRateLimitWeight, "oversized scope weights must return invalid weight error")
		assert.Empty(t, additionalRateLimits, "oversized scope weights should not return additional rate limits")
		assert.Zero(t, orderCount, "oversized scope weights should not return an order count")
	})

	t.Run("valid weight", func(t *testing.T) {
		t.Parallel()

		limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
		additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits(tradeRateLimitPlaceSingle, map[string]int{tradeRateLimitBTCUSDT: 2})
		require.NoError(t, err, "valid scope weight must not error")
		require.Len(t, additionalRateLimits, 1, "valid scope weight must return one additional rate limit")
		assert.NotNil(t, additionalRateLimits[0].Limiter, "valid scope limit should include limiter")
		assert.Equal(t, request.Weight(2), additionalRateLimits[0].WeightOverride, "valid scope weight should return one weight")
		assert.Equal(t, 2, orderCount, "order count should match the scope weight")
	})

	batchClasses := []struct {
		name          string
		class         tradeRateLimitClass
		expectedClass tradeRateLimitClass
	}{
		{name: "place", class: tradeRateLimitPlaceBatch, expectedClass: tradeRateLimitPlaceSingle},
		{name: "cancel", class: tradeRateLimitCancelBatch, expectedClass: tradeRateLimitCancelSingle},
		{name: "amend", class: tradeRateLimitAmendBatch, expectedClass: tradeRateLimitAmendSingle},
	}
	for _, tc := range batchClasses {
		t.Run("single order batch uses "+tc.name+" single bucket", func(t *testing.T) {
			t.Parallel()

			limiter := &tradeRateLimiter{scopedLimiters: make(map[tradeRateLimitKey]*request.RateLimiterWithWeight)}
			additionalRateLimits, orderCount, err := limiter.additionalTradeScopeRateLimits(tc.class, map[string]int{tradeRateLimitBTCUSDT: 1})
			require.NoError(t, err, "one-order batch must not error")
			require.Len(t, additionalRateLimits, 1, "one-order batch must return one scoped limiter")
			assert.Equal(t, 1, orderCount, "one-order batch should return one order")
			assert.Contains(t, limiter.scopedLimiters, tradeRateLimitKey{class: tc.expectedClass, scope: tradeRateLimitBTCUSDT}, "one-order batch should use the single-order bucket")
			assert.NotContains(t, limiter.scopedLimiters, tradeRateLimitKey{class: tc.class, scope: tradeRateLimitBTCUSDT}, "one-order batch should not use the batch bucket")
		})
	}
}

func TestTradeRateLimiterSubAccountRateLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		initialised    bool
		orderCount     int
		expectedOK     bool
		expectedWeight request.Weight
		expectedError  error
	}{
		{name: "zero orders", initialised: true},
		{name: "single order", initialised: true, orderCount: 1, expectedOK: true, expectedWeight: 1},
		{name: "multiple orders", initialised: true, orderCount: 3, expectedOK: true, expectedWeight: 3},
		{name: "oversized order count", initialised: true, orderCount: 256, expectedError: errInvalidTradeRateLimitWeight},
		{name: "uninitialised", orderCount: 1, expectedError: errTradeRateLimiterNotInitialised},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			limiter := new(tradeRateLimiter)
			if tc.initialised {
				limiter.subAccountLimiter = request.NewRateLimitWithWeight(twoSecondsInterval, subAccountTradeRateLimitActions, 1)
			}
			limit, ok, err := limiter.subAccountRateLimit(tc.orderCount)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "subAccountRateLimit must return expected error")
				return
			}
			require.NoError(t, err, "subAccountRateLimit must not error")
			assert.Equal(t, tc.expectedOK, ok, "limit availability should match")
			assert.Equal(t, tc.expectedWeight, limit.WeightOverride, "weight should match order count")
			if tc.expectedOK {
				assert.NotNil(t, limit.Limiter, "limiter should be set")
			}
		})
	}
}

func TestTradeRateLimiterAdditionalTradeRateLimits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		class           tradeRateLimitClass
		counts          map[string]int
		expectedLimits  int
		expectedWeights []request.Weight
		expectedError   error
	}{
		{
			name:            "place includes scoped and subaccount limits",
			class:           tradeRateLimitPlaceSingle,
			counts:          map[string]int{tradeRateLimitBTCUSDT: 2},
			expectedLimits:  2,
			expectedWeights: []request.Weight{2, 2},
		},
		{
			name:            "amend includes scoped and subaccount limits",
			class:           tradeRateLimitAmendBatch,
			counts:          map[string]int{tradeRateLimitBTCUSDT: 3},
			expectedLimits:  2,
			expectedWeights: []request.Weight{3, 3},
		},
		{
			name:            "cancel excludes subaccount limit",
			class:           tradeRateLimitCancelSingle,
			counts:          map[string]int{tradeRateLimitBTCUSDT: 1},
			expectedLimits:  1,
			expectedWeights: []request.Weight{1},
		},
		{
			name:          "invalid class",
			class:         "unknown",
			counts:        map[string]int{tradeRateLimitBTCUSDT: 1},
			expectedError: errInvalidTradeRateLimitClass,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			limiter := &tradeRateLimiter{
				scopedLimiters:    make(map[tradeRateLimitKey]*request.RateLimiterWithWeight),
				subAccountLimiter: request.NewRateLimitWithWeight(twoSecondsInterval, subAccountTradeRateLimitActions, 1),
			}
			additionalRateLimits, err := limiter.additionalTradeRateLimits(tc.class, tc.counts)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "additionalTradeRateLimits must return expected error")
				return
			}
			require.NoError(t, err, "additionalTradeRateLimits must not error")
			require.Len(t, additionalRateLimits, tc.expectedLimits, "additionalTradeRateLimits must return expected limits")
			for i := range tc.expectedWeights {
				assert.Equal(t, tc.expectedWeights[i], additionalRateLimits[i].WeightOverride, "rate-limit weight should match")
			}
		})
	}
}

func TestTradeRateLimiterExchangeInstances(t *testing.T) {
	t.Parallel()

	first := new(Exchange)
	first.SetDefaults()
	second := new(Exchange)
	second.SetDefaults()

	firstScopedLimiter, err := first.tradeLimiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "INSTANCE-TEST")
	require.NoError(t, err, "first Exchange must create scoped limiter")
	secondScopedLimiter, err := second.tradeLimiter.getOrCreateScopedLimiter(tradeRateLimitPlaceSingle, "INSTANCE-TEST")
	require.NoError(t, err, "second Exchange must create scoped limiter")
	assert.NotSame(t, firstScopedLimiter, secondScopedLimiter, "Exchange instances should not share request-scoped buckets")
}

func TestRateLimitsSharedAcrossExchangeInstances(t *testing.T) {
	t.Parallel()

	first := new(Exchange)
	first.SetDefaults()
	second := new(Exchange)
	second.SetDefaults()

	tradeEndpoints := []struct {
		name     string
		endpoint request.EndpointLimit
	}{
		{name: "place order", endpoint: placeOrderEPL},
		{name: "place multiple orders", endpoint: placeMultipleOrdersEPL},
		{name: "cancel order", endpoint: cancelOrderEPL},
		{name: "cancel multiple orders", endpoint: cancelMultipleOrdersEPL},
		{name: "amend order", endpoint: amendOrderEPL},
		{name: "amend multiple orders", endpoint: amendMultipleOrdersEPL},
	}
	for _, tc := range tradeEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.NotNil(t, rateLimits[tc.endpoint], "trade endpoint should have a standard limiter definition")
			assert.Same(t, first.Requester.GetRateLimiterDefinitions()[tc.endpoint], second.Requester.GetRateLimiterDefinitions()[tc.endpoint], "Exchange instances should use the same static endpoint limiter")
		})
	}
	t.Run("distinct endpoints", func(t *testing.T) {
		t.Parallel()

		assert.NotSame(t, rateLimits[placeOrderEPL], rateLimits[cancelOrderEPL], "distinct endpoints should retain distinct limiter instances")
	})
}

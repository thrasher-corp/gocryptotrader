package coinmarketcap

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetRateLimits(t *testing.T) {
	t.Parallel()

	first := getRateLimits()
	second := getRateLimits()
	require.Len(t, first, 6, "getRateLimits must return all tier definitions")
	require.Len(t, second, 6, "getRateLimits must return all tier definitions")

	seen := make(map[*request.RateLimiterWithWeight]struct{}, len(first))
	for _, key := range []request.EndpointLimit{
		basicEPL,
		builderEPL,
		startupEPL,
		growthEPL,
		professionalEPL,
		enterpriseEPL,
	} {
		require.NotNil(t, first[key], "getRateLimits must return each tier limiter")
		require.NotNil(t, second[key], "getRateLimits must return each tier limiter")
		require.NotContains(t, seen, first[key], "getRateLimits must return independent tier limiters")
		seen[first[key]] = struct{}{}
		assert.NotSame(t, first[key], second[key], "getRateLimits should return independent per-client tier limiters")
	}

	for _, tc := range []struct {
		name        string
		rateLimit   request.EndpointLimit
		requestRate int
	}{
		{name: "basic", rateLimit: basicEPL, requestRate: basicRequestRate},
		{name: "builder", rateLimit: builderEPL, requestRate: builderRequestRate},
		{name: "startup", rateLimit: startupEPL, requestRate: startupRequestRate},
		{name: "growth", rateLimit: growthEPL, requestRate: growthRequestRate},
		{name: "professional", rateLimit: professionalEPL, requestRate: professionalRequestRate},
		{name: "enterprise", rateLimit: enterpriseEPL, requestRate: enterpriseRequestRate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				limiter := getRateLimits()[tc.rateLimit]
				require.NotNil(t, limiter, "getRateLimits must return the tier limiter")
				start := time.Now()
				require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the first request")
				require.NoError(t, limiter.RateLimit(t.Context()), "RateLimit must allow the second request")
				assert.Equal(t, rateInterval/time.Duration(tc.requestRate), time.Since(start), "getRateLimits should configure the correct tier rate")
			})
		})
	}
}

func TestPlanRateLimit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		plan      uint8
		rateLimit request.EndpointLimit
	}{
		{name: "unset", rateLimit: basicEPL},
		{name: "basic", plan: Basic, rateLimit: basicEPL},
		{name: "builder", plan: Builder, rateLimit: builderEPL},
		{name: "startup", plan: Startup, rateLimit: startupEPL},
		{name: "growth", plan: Growth, rateLimit: growthEPL},
		{name: "professional", plan: Professional, rateLimit: professionalEPL},
		{name: "enterprise", plan: Enterprise, rateLimit: enterpriseEPL},
		{name: "combined", plan: Basic | Builder, rateLimit: basicEPL},
		{name: "unknown", plan: 255, rateLimit: basicEPL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.rateLimit, planRateLimit(tc.plan), "planRateLimit should return the correct tier limiter")
		})
	}
}

package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		epl         request.EndpointLimit
		perInterval float64
	}{
		{defaultEPL, 200},
		{orderEPL, 500},
		{contractEPL, 200},
	} {
		require.Containsf(t, rateLimits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
		assert.InDeltaf(t, tc.perInterval/rateInterval.Seconds(), float64(rateLimits[tc.epl].Limit()), 1e-9, "rateLimits should refill endpoint limit %d at its documented rate", tc.epl)
		assert.Equalf(t, request.Weight(1), rateLimits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d as one request", tc.epl)
	}
	assert.Len(t, rateLimits, 3, "rateLimits should define a limit for every endpoint group and no more")
	for a, limitA := range rateLimits {
		for b, limitB := range rateLimits {
			if a != b {
				assert.Falsef(t, limitA.SharesBudgetWith(limitB), "Endpoint limits %v and %v should not share a budget, as LBank limits orders apart from other requests", a, b)
			}
		}
	}
}

// TestRateLimitsSharedAcrossInstances checks that instances built by SetDefaults draw each endpoint on the same budget,
// since instances carrying limiters of their own would together exceed LBank's limits
func TestRateLimitsSharedAcrossInstances(t *testing.T) {
	t.Parallel()
	first, second := new(Exchange), new(Exchange)
	first.SetDefaults()
	second.SetDefaults()
	firstLimits := first.Requester.GetRateLimiterDefinitions()
	secondLimits := second.Requester.GetRateLimiterDefinitions()
	require.Len(t, firstLimits, len(rateLimits), "SetDefaults must install every endpoint limit")
	require.Len(t, secondLimits, len(rateLimits), "SetDefaults must install every endpoint limit")
	for epl, limiter := range firstLimits {
		assert.Truef(t, limiter.SharesBudgetWith(secondLimits[epl]), "Endpoint limit %d should draw on the same budget in both instances", epl)
	}
}

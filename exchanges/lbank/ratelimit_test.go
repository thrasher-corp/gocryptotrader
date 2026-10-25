package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetRateLimits(t *testing.T) {
	t.Parallel()
	limits := GetRateLimits()
	for _, tc := range []struct {
		epl         request.EndpointLimit
		perInterval float64
	}{
		{defaultEPL, 200},
		{orderEPL, 500},
		{contractEPL, 200},
	} {
		require.Containsf(t, limits, tc.epl, "GetRateLimits must define endpoint limit %d", tc.epl)
		assert.InDeltaf(t, tc.perInterval/rateInterval.Seconds(), float64(limits[tc.epl].Limit()), 1e-9, "GetRateLimits should refill endpoint limit %d at its documented rate", tc.epl)
		assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "GetRateLimits should weigh endpoint limit %d as one request", tc.epl)
	}
	assert.Len(t, limits, 3, "GetRateLimits should define a limit for every endpoint group and no more")
	for a, limitA := range limits {
		for b, limitB := range limits {
			if a != b {
				assert.Falsef(t, limitA.SharesBudgetWith(limitB), "Endpoint limits %v and %v should not share a budget, as LBank limits orders apart from other requests", a, b)
			}
		}
	}
}

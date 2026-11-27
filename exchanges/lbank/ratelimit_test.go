package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/request"
)

func TestRateLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		epl         request.EndpointLimit
		perInterval float64
	}{
		{"defaultEPL", defaultEPL, 200},
		{"orderEPL", orderEPL, 500},
		{"contractEPL", contractEPL, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Containsf(t, rateLimits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
			assert.InDeltaf(t, tc.perInterval/rateInterval.Seconds(), float64(rateLimits[tc.epl].Limit()), 1e-9, "rateLimits should refill endpoint limit %d at its documented rate", tc.epl)
			assert.Equalf(t, request.Weight(1), rateLimits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d as one request", tc.epl)
		})
	}
	t.Run("no other endpoint limits", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, rateLimits, 3, "rateLimits should define a limit for every endpoint group and no more")
	})
	t.Run("separate budgets", func(t *testing.T) {
		t.Parallel()
		for a, limitA := range rateLimits {
			for b, limitB := range rateLimits {
				if a != b {
					assert.Falsef(t, limitA.SharesBudgetWith(limitB), "Endpoint limits %v and %v should not share a budget, as LBank limits orders apart from other requests", a, b)
				}
			}
		}
	})
}

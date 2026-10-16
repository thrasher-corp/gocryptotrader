package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

func TestGetRateLimits(t *testing.T) {
	t.Parallel()
	limits := GetRateLimits()
	for _, tc := range []struct {
		epl  request.EndpointLimit
		rate rate.Limit
	}{
		{epl: nonMatchingEPL, rate: 20},
		{epl: matchingEPL, rate: 5},
		{epl: getInstrumentsEPL, rate: 1},
		{epl: subscribeEPL, rate: 10.0 / 3},
		{epl: movePositionsEPL, rate: 6.0 / 60},
		{epl: transactionLogEPL, rate: 1},
		{epl: massQuoteEPL, rate: 10},
		{epl: simulatePortfolioEPL, rate: 1},
	} {
		require.Containsf(t, limits, tc.epl, "GetRateLimits must define endpoint limit %d", tc.epl)
		assert.InDeltaf(t, float64(tc.rate), float64(limits[tc.epl].Limit()), 1e-9, "GetRateLimits should refill endpoint limit %d at its documented rate", tc.epl)
		assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "GetRateLimits should weigh endpoint limit %d as one request", tc.epl)
	}
	assert.Len(t, limits, 8, "GetRateLimits should define only the documented limits")
}

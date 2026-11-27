package cryptodotcom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimits(t *testing.T) {
	t.Parallel()
	for epl := publicEPL; epl <= privateEPL; epl++ {
		assert.NotNilf(t, rateLimits[epl], "endpoint limit %d should have a limiter", epl)
	}
	a, b := new(Exchange), new(Exchange)
	a.SetDefaults()
	b.SetDefaults()
	aLimits, bLimits := a.Requester.GetRateLimiterDefinitions(), b.Requester.GetRateLimiterDefinitions()
	require.Len(t, aLimits, len(rateLimits), "the first Exchange must hold every endpoint limit")
	require.Len(t, bLimits, len(rateLimits), "the second Exchange must hold every endpoint limit")
	for epl, limiter := range rateLimits {
		assert.Samef(t, limiter, aLimits[epl], "the first Exchange should share endpoint limit %d's limiter", epl)
		assert.Samef(t, limiter, bLimits[epl], "the second Exchange should share endpoint limit %d's limiter", epl)
	}
}

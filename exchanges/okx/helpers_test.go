package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetAssetsFromInstrumentIDWithCheck(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	_, err := ex.getAssetsFromInstrumentIDWithCheck("", false)
	require.ErrorIs(t, err, errMissingInstrumentID, "getAssetsFromInstrumentIDWithCheck must error for empty instrument IDs")

	pair := ex.CurrencyPairs.Pairs[asset.Spot].Enabled[0]
	availableAssets, err := ex.getAssetsFromInstrumentIDWithCheck(pair.String(), false)
	require.NoError(t, err, "getAssetsFromInstrumentIDWithCheck must not error for available spot pairs")
	assert.Contains(t, availableAssets, asset.Spot, "getAssetsFromInstrumentIDWithCheck should include spot for available lookups")

	require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Spot, pair), "DisablePair must not error")

	enabledAssets, err := ex.getAssetsFromInstrumentIDWithCheck(pair.String(), true)
	require.NoError(t, err, "getAssetsFromInstrumentIDWithCheck must not error for enabled spot lookups")
	assert.NotContains(t, enabledAssets, asset.Spot, "getAssetsFromInstrumentIDWithCheck should exclude disabled spot pairs for enabled lookups")
}

func TestPairMatchesRequirement(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	pair := ex.CurrencyPairs.Pairs[asset.Spot].Enabled[0]
	matches, err := ex.pairMatchesRequirement(pair, asset.Spot, false)
	require.NoError(t, err, "pairMatchesRequirement must not error for available lookups")
	assert.True(t, matches, "pairMatchesRequirement should match available pairs")

	require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Spot, pair), "DisablePair must not error")

	matches, err = ex.pairMatchesRequirement(pair, asset.Spot, false)
	require.NoError(t, err, "pairMatchesRequirement must not error for available lookups after disable")
	assert.True(t, matches, "pairMatchesRequirement should still match available pairs when disabled")

	matches, err = ex.pairMatchesRequirement(pair, asset.Spot, true)
	require.NoError(t, err, "pairMatchesRequirement must not error for enabled lookups after disable")
	assert.False(t, matches, "pairMatchesRequirement should not match disabled pairs for enabled lookups")
}

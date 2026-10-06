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

	for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "setup must succeed")
			pairs, err := ex.GetAvailablePairs(a)
			require.NoError(t, err, "pairs must load")
			require.NotEmpty(t, pairs, "fixture must have pairs")
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "asset must disable")
			got, err := ex.getAssetsFromInstrumentIDWithCheck(pairs[0].String())
			require.Error(t, err, "disabled asset must not resolve")
			assert.Empty(t, got, "disabled asset should be excluded")
		})
	}

	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run("disabled "+a.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "setup must succeed")
			pairs, err := ex.GetAvailablePairs(a)
			require.NoError(t, err, "available pairs must load")
			require.NotEmpty(t, pairs, "available pairs must exist")
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "asset must disable")
			got, err := ex.getAssetsFromInstrumentIDWithCheck(pairs[0].String())
			if err != nil {
				require.ErrorIs(t, err, asset.ErrNotSupported, "unmatched instrument must report no asset")
			}
			assert.NotContains(t, got, a, "disabled asset should never resolve through the two-part branch")
		})
	}

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	_, err := ex.getAssetsFromInstrumentIDWithCheck("")
	require.ErrorIs(t, err, errMissingInstrumentID, "getAssetsFromInstrumentIDWithCheck must error for empty instrument IDs")

	pair := ex.CurrencyPairs.Pairs[asset.Spot].Enabled[0]
	availableAssets, err := ex.getAssetsFromInstrumentIDWithCheck(pair.String())
	require.NoError(t, err, "getAssetsFromInstrumentIDWithCheck must not error for available spot pairs")
	assert.Contains(t, availableAssets, asset.Spot, "getAssetsFromInstrumentIDWithCheck should include spot for available lookups")

	require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Spot, pair), "DisablePair must not error")

	availableAssets, err = ex.getAssetsFromInstrumentIDWithCheck(pair.String())
	require.NoError(t, err, "getAssetsFromInstrumentIDWithCheck must not error after disabling the spot pair")
	assert.Contains(t, availableAssets, asset.Spot, "available lookup should include disabled pairs")
}

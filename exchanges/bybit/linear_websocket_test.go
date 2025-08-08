package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGenerateLinearDefaultSubscriptions(t *testing.T) {
	t.Parallel()

	_, err := e.GenerateLinearDefaultSubscriptions(asset.OptionCombo)
	assert.ErrorIs(t, err, asset.ErrInvalidAsset)

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")
	for i := range subs {
		assert.Equal(t, asset.USDTMarginedFutures, subs[i].Asset, "Asset type should be USDTMarginedFutures")
	}

	err = e.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false)
	require.NoError(t, err, "SetAssetEnabled must not error")

	subs, err = e.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.Empty(t, subs, "Subscriptions should be empty when asset is disabled")

	subs, err = e.GenerateLinearDefaultSubscriptions(asset.USDCMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")
	for i := range subs {
		assert.Equal(t, asset.USDCMarginedFutures, subs[i].Asset, "Asset type should be USDCMarginedFutures")
	}
}

func TestLinearSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")

	err = e.LinearSubscribe(t.Context(), &FixtureConnection{}, asset.OptionCombo, subs)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	err = e.LinearSubscribe(t.Context(), &FixtureConnection{}, asset.USDTMarginedFutures, subs)
	require.NoError(t, err, "LinearSubscribe must not error")
}

func TestLinearUnsubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")

	err = e.LinearSubscribe(t.Context(), &FixtureConnection{}, asset.USDTMarginedFutures, subs)
	require.NoError(t, err, "LinearSubscribe must not error")

	err = e.LinearUnsubscribe(t.Context(), &FixtureConnection{}, asset.OptionCombo, subs)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	err = e.LinearUnsubscribe(t.Context(), &FixtureConnection{}, asset.USDTMarginedFutures, subs)
	require.NoError(t, err, "LinearUnsubscribe must not error")
}

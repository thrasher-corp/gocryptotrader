package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")
	for i := range subs {
		assert.Equal(t, asset.Options, subs[i].Asset, "Asset type should be Options")
	}

	err = e.CurrencyPairs.SetAssetEnabled(asset.Options, false)
	require.NoError(t, err, "SetAssetEnabled must not error")

	subs, err = e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.Empty(t, subs, "Subscriptions should be empty when asset is disabled")
}

func TestOptionSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")

	err = e.OptionsSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsSubscribe must not error")
}

func TestOptionsUnsubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")

	err = e.OptionsSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsSubscribe must not error")

	err = e.OptionsUnsubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "OptionsUnsubscribe must not error")
}

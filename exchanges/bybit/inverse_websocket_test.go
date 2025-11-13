package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGenerateInverseDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	subs, err := e.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")
	assert.NotEmpty(t, subs, "Subscriptions should not be empty")
	for i := range subs {
		assert.Equal(t, asset.CoinMarginedFutures, subs[i].Asset, "Asset type should be CoinMarginedFutures")
	}

	err = e.CurrencyPairs.SetAssetEnabled(asset.CoinMarginedFutures, false)
	require.NoError(t, err, "SetAssetEnabled must not error")

	subs, err = e.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")
	assert.Empty(t, subs, "Subscriptions should be empty when asset is disabled")
}

func TestInverseSubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")

	err = e.InverseSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "InverseSubscribe must not error")
}

func TestInverseUnsubscribe(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	subs, err := e.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")

	err = e.InverseSubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "InverseSubscribe must not error")

	err = e.InverseUnsubscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "InverseUnsubscribe must not error")
}

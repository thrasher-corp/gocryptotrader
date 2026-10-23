package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.DeliveryFutures)
	assert.NoError(t, err)
}

func TestGenerateDeliveryFuturesDefaultSubscriptionsSkipsUnsupportedAsset(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()

	got, err := ex.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.Futures)
	require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error for unsupported delivery asset")
	require.Empty(t, got, "unsupported delivery asset must not generate subscriptions")
}

func TestGenerateDeliveryFuturesDefaultSubscriptionsUsesContextCredentials(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	_, err := ex.GenerateDeliveryFuturesDefaultSubscriptions(t.Context(), asset.DeliveryFutures)
	require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error without credentials")
	require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "GenerateDeliveryFuturesDefaultSubscriptions must disable authenticated endpoints without credentials")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	_, err = ex.GenerateDeliveryFuturesDefaultSubscriptions(ctx, asset.DeliveryFutures)
	require.NoError(t, err, "GenerateDeliveryFuturesDefaultSubscriptions must not error with context credentials")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "GenerateDeliveryFuturesDefaultSubscriptions should keep authenticated endpoints when its context carries credentials")
}

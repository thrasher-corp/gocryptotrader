package v18_test

import (
	"testing"

	"github.com/buger/jsonparser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Okx"}, new(v18.Version).Exchanges(), "migration should apply only to OKX")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, input, expected string }{
		{"defaults", `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com:8443/ws/v5/public","WebsocketPrivateURL":"wss://ws.okx.com:8443/ws/v5/private","WebsocketSpotSupplementaryURL":"wss://ws.okx.com:8443/ws/v5/business","RestSpotURL":"https://www.okx.com/api/v5/"}},"enabled":false}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public","WebsocketPrivateURL":"wss://ws.okx.com/ws/v5/private","WebsocketSpotSupplementaryURL":"wss://ws.okx.com/ws/v5/business","RestSpotURL":"https://www.okx.com/api/v5/"}},"enabled":false}`},
		{"escaped default", `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss:\/\/ws.okx.com:8443\/ws\/v5\/public"}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}`},
		{"missing", `{}`, `{}`},
		{"null", `{"api":{"urlEndpoints":null}}`, `{"api":{"urlEndpoints":null}}`},
		{"custom", `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://custom.example:8443/ws/v5/public","WebsocketPrivateURL":"wss://ws.okx.com:8443/custom","WebsocketSpotSupplementaryURL":null}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://custom.example:8443/ws/v5/public","WebsocketPrivateURL":"wss://ws.okx.com:8443/custom","WebsocketSpotSupplementaryURL":null}}}`},
		{"current", `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "migration must succeed")
			assert.JSONEq(t, tc.expected, string(got), "migration should replace only legacy defaults")
		})
	}
	t.Run("invalid endpoint value", func(t *testing.T) {
		t.Parallel()
		_, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":}}}`))
		require.ErrorIs(t, err, jsonparser.UnknownValueTypeError, "invalid JSON must fail with its parser error")
	})
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}`)
	got, err := new(v18.Version).DowngradeExchange(t.Context(), input)
	require.NoError(t, err, "downgrade must succeed")
	assert.Equal(t, input, got, "downgrade should retain the compatible port")
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":17,"exchanges":[{"name":"Okx","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com:8443/ws/v5/public"}}},{"name":"Other","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com:8443/ws/v5/public"}}}]}`)
	got, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "registered upgrade must succeed")
	assert.JSONEq(t, `{"version":18,"exchanges":[{"name":"Okx","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}},{"name":"Other","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com:8443/ws/v5/public"}}}]}`, string(got), "only OKX defaults should migrate")
	down, err := versions.Manager.Deploy(t.Context(), got, 17)
	require.NoError(t, err, "registered downgrade must succeed")
	again, err := versions.Manager.Deploy(t.Context(), down, 18)
	require.NoError(t, err, "repeated upgrade must succeed")
	assert.JSONEq(t, string(got), string(again), "migration round trip should preserve usable endpoints")
}

package v14_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v14 "github.com/thrasher-corp/gocryptotrader/config/versions/v14"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Gemini"}, new(v14.Version).Exchanges(), "Exchanges should target Gemini")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, want string
		invalid           bool
	}{
		{name: "legacy", input: `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com"}}}`, want: `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com/v2/marketdata"}}}`},
		{name: "custom", input: `{"api":{"urlEndpoints":{"WebsocketSpotURL":"ws://localhost/custom"}}}`},
		{name: "current", input: `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com/v2/marketdata"}}}`},
		{name: "missing", input: `{}`},
		{name: "invalid type", input: `{"api":{"urlEndpoints":{"WebsocketSpotURL":42}}}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v14.Version).UpgradeExchange(t.Context(), []byte(tc.input))
			if tc.invalid {
				require.Error(t, err, "UpgradeExchange must reject an invalid URL type")
				return
			}
			require.NoError(t, err, "UpgradeExchange must not error")
			want := tc.want
			if want == "" {
				want = tc.input
			}
			assert.JSONEq(t, want, string(out), "UpgradeExchange should preserve custom settings")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com/v2/marketdata"}}}`)
	out, err := new(v14.Version).DowngradeExchange(t.Context(), input)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.Equal(t, input, out, "DowngradeExchange should retain a working endpoint")
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	out, err := versions.Manager.Deploy(t.Context(), []byte(`{"version":13,"exchanges":[{"name":"Gemini","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com"}}}]}`), 14)
	require.NoError(t, err, "Deploy must apply the registered upgrade")
	assert.JSONEq(t, `{"version":14,"exchanges":[{"name":"Gemini","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://api.gemini.com/v2/marketdata"}}}]}`, string(out), "Deploy should update the legacy endpoint")
}

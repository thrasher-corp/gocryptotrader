package v18_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GateIO"}, new(v18.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "old defaults are removed and the unchanged testnet default is kept",
			in:   `{"RestSpotURL":"https://api.gateio.ws/api/v4/","RestFuturesURL":"https://fx-api.gateio.ws/api/v4/","RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
			exp:  `{"RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
		},
		{
			name: "custom endpoints lose the version prefix",
			in:   `{"RestSpotURL":"https://proxy.example/gate/api/v4/","RestFuturesURL":"https://proxy.example/gate-fx/api/v4","RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws/api/v4/"}`,
			exp:  `{"RestSpotURL":"https://proxy.example/gate","RestFuturesURL":"https://proxy.example/gate-fx","RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws"}`,
		},
		{
			name: "custom endpoints without the prefix are kept",
			in:   `{"RestSpotURL":"https://proxy.example","RestFuturesURL":"https://proxy.example/v4/","RestSpotSupplementaryURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
			exp:  `{"RestSpotURL":"https://proxy.example","RestFuturesURL":"https://proxy.example/v4/","RestSpotSupplementaryURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
		},
		{
			name: "unrelated endpoints are kept",
			in:   `{"EdgeCase1URL":"https://www.gate.com/apiw/v2/api/v4/","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
			exp:  `{"EdgeCase1URL":"https://www.gate.com/apiw/v2/api/v4/","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"name":"GateIO","api":{"urlEndpoints":`+tc.in+`}}`))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, `{"name":"GateIO","api":{"urlEndpoints":`+tc.exp+`}}`, string(out), "UpgradeExchange should migrate the endpoints")
		})
	}

	in := []byte(`{"name":"GateIO","api":{}}`)
	out, err := new(v18.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err, "UpgradeExchange must not error when urlEndpoints is missing")
	assert.Equal(t, string(in), string(out), "UpgradeExchange should not change a config without urlEndpoints")

	_, err = new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"name":"GateIO","api":{"urlEndpoints":{"RestSpotURL":42}}}`))
	assert.ErrorContains(t, err, "Value is not a string", "UpgradeExchange should error on a non-string endpoint")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "new defaults are removed and the unchanged testnet default is kept",
			in:   `{"RestSpotURL":"https://api.gateio.ws","RestFuturesURL":"https://fx-api.gateio.ws","RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
			exp:  `{"RestSpotSupplementaryURL":"https://fx-api-testnet.gateio.ws","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}`,
		},
		{
			name: "custom endpoints regain the version prefix",
			in:   `{"RestSpotURL":"https://proxy.example/gate","RestFuturesURL":"https://proxy.example/gate-fx/","RestSpotSupplementaryURL":"https://testnet-proxy.example"}`,
			exp:  `{"RestSpotURL":"https://proxy.example/gate/api/v4/","RestFuturesURL":"https://proxy.example/gate-fx/api/v4/","RestSpotSupplementaryURL":"https://testnet-proxy.example/api/v4/"}`,
		},
		{
			name: "endpoints already carrying the prefix and placeholders are kept",
			in:   `{"RestSpotURL":"https://proxy.example/api/v4/","RestFuturesURL":"","RestSpotSupplementaryURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
			exp:  `{"RestSpotURL":"https://proxy.example/api/v4/","RestFuturesURL":"","RestSpotSupplementaryURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"name":"GateIO","api":{"urlEndpoints":`+tc.in+`}}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, `{"name":"GateIO","api":{"urlEndpoints":`+tc.exp+`}}`, string(out), "DowngradeExchange should migrate the endpoints")
		})
	}

	_, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"name":"GateIO","api":{"urlEndpoints":{"RestFuturesURL":true}}}`))
	assert.ErrorContains(t, err, "Value is not a string", "DowngradeExchange should error on a non-string endpoint")
}

// TestRoundTrip checks a custom endpoint survives an upgrade and downgrade unchanged
func TestRoundTrip(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"GateIO","api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/gate/api/v4/"}}}`)
	up, err := new(v18.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err, "UpgradeExchange must not error")
	down, err := new(v18.Version).DowngradeExchange(t.Context(), up)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.JSONEq(t, string(in), string(down), "a custom endpoint should round trip unchanged")
}

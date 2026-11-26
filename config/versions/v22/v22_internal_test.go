package v22

import (
	"cmp"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGateIO(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, config string
		expected     bool
	}{
		{name: "GateIO", config: `{"name":"GateIO"}`, expected: true},
		{name: "lower case", config: `{"name":"gateio"}`, expected: true},
		{name: "upper case", config: `{"name":"GATEIO"}`, expected: true},
		{name: "mixed case", config: `{"name":"gAtEiO"}`, expected: true},
		{name: "Gate", config: `{"name":"Gate"}`},
		{name: "longer name", config: `{"name":"GateIOX"}`},
		{name: "other exchange", config: `{"name":"Binance"}`},
		{name: "missing name", config: `{"enabled":true}`},
		{name: "non-string name", config: `{"name":42}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isGateIO([]byte(tc.config)), "isGateIO should match only GateIO's name, in any case")
		})
	}
}

func TestIsBareHost(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		endpoint string
		expected bool
	}{
		{endpoint: "https://proxy.example", expected: true},
		{endpoint: "https://proxy.example:8443", expected: true},
		{endpoint: "http://localhost:8080", expected: true},
		{endpoint: "https://proxy.example:8443/gateway/"},
		{endpoint: "https://proxy.example/"},
		{endpoint: "https://proxy.example//"},
		{endpoint: "https://proxy.example/gateway"},
		{endpoint: "https://proxy.example/gateway/"},
		{endpoint: "https://proxy.example/gate/api/v4/"},
		{endpoint: "proxy.example"},
		{endpoint: "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"},
		{endpoint: ""},
	} {
		t.Run(cmp.Or(tc.endpoint, "empty endpoint"), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isBareHost(tc.endpoint), "isBareHost should report whether the endpoint is a scheme and host alone")
		})
	}
}

func TestSetEndpoint(t *testing.T) {
	t.Parallel()
	got, err := setEndpoint([]byte(`{"api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/api/v4/","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}}}`), "RestSpotURL", `https://proxy.example/"gate"`)
	require.NoError(t, err, "setEndpoint must not error")
	assert.JSONEq(t, `{"api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/\"gate\"","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}}}`, string(got), "setEndpoint should replace only the endpoint, encoded as a JSON string")

	_, err = setEndpoint([]byte(`not json`), "RestSpotURL", "https://proxy.example")
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "setEndpoint should reject a document it cannot set the endpoint in")
}

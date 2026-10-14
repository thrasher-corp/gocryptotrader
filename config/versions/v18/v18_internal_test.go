package v18

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetEndpoint(t *testing.T) {
	t.Parallel()
	got, err := setEndpoint([]byte(`{"api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/api/v4/","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}}}`), "RestSpotURL", `https://proxy.example/"gate"`)
	require.NoError(t, err, "setEndpoint must not error")
	assert.JSONEq(t, `{"api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/\"gate\"","WebsocketSpotURL":"wss://api.gateio.ws/ws/v4/"}}}`, string(got), "setEndpoint should replace only the endpoint, encoded as a JSON string")

	_, err = setEndpoint([]byte(`not json`), "RestSpotURL", "https://proxy.example")
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "setEndpoint should reject a document it cannot set the endpoint in")
}

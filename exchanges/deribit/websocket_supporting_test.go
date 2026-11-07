package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIGetTime(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetTime(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(mockServerTime), result, "WebsocketAPI.GetTime should decode the server time")
	} else {
		assert.WithinDuration(t, time.Now(), result, time.Minute, "WebsocketAPI.GetTime should return the current time")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTime(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTime should error without a websocket connection")
}

func TestWebsocketAPIHello(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().Hello(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.Hello must reject a nil request")
	_, err = ex.WebsocketAPI().Hello(t.Context(), &HelloRequest{ClientVersion: testClientVersion})
	require.ErrorIs(t, err, errClientNameRequired, "WebsocketAPI.Hello must reject an empty client name")
	_, err = ex.WebsocketAPI().Hello(t.Context(), &HelloRequest{ClientName: testClientName})
	require.ErrorIs(t, err, errClientVersionRequired, "WebsocketAPI.Hello must reject an empty client version")

	result, err := ex.WebsocketAPI().Hello(t.Context(), &HelloRequest{ClientName: testClientName, ClientVersion: testClientVersion})
	require.NoError(t, err, "WebsocketAPI.Hello must not error")
	if mockTests {
		assert.Equal(t, &HelloResponse{Version: "1.2.26"}, result, "WebsocketAPI.Hello should decode the API version")
	} else {
		assert.NotEmpty(t, result.Version, "WebsocketAPI.Hello should return the API version")
	}

	_, err = newTestExchange(t).WebsocketAPI().Hello(t.Context(), &HelloRequest{ClientName: testClientName, ClientVersion: testClientVersion})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Hello should error without a websocket connection")
}

func TestWebsocketAPIGetStatus(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetStatus(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetStatus must not error")
	if mockTests {
		assert.Equal(t, &StatusResponse{Locked: "false"}, result, "WebsocketAPI.GetStatus should decode the lock status")
	} else {
		assert.Contains(t, []string{"true", "partial", "false"}, result.Locked, "WebsocketAPI.GetStatus should return a documented lock status")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetStatus(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetStatus should error without a websocket connection")
}

func TestWebsocketAPITestConnection(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().TestConnection(t.Context(), "bogus")
	require.ErrorIs(t, err, errInvalidExpectedResult, "WebsocketAPI.TestConnection must reject an undocumented expected result")

	result, err := ex.WebsocketAPI().TestConnection(t.Context(), "")
	require.NoError(t, err, "WebsocketAPI.TestConnection must not error")
	if mockTests {
		assert.Equal(t, &TestConnectionResponse{Version: "1.2.26"}, result, "WebsocketAPI.TestConnection should decode the API version")
		requests := venue.received("public/test")
		require.NotEmpty(t, requests, "WebsocketAPI.TestConnection must send a public/test request")
		assert.Empty(t, requests[len(requests)-1].Params, "WebsocketAPI.TestConnection should leave out an empty expected result")
	} else {
		assert.NotEmpty(t, result.Version, "WebsocketAPI.TestConnection should return the API version")
	}

	_, err = ex.WebsocketAPI().TestConnection(t.Context(), testExpectedResultException)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "WebsocketAPI.TestConnection must return the error Deribit is asked for")
	assert.Equal(t, int64(11094), apiErr.Code, "WebsocketAPI.TestConnection should decode the requested error")

	_, err = newTestExchange(t).WebsocketAPI().TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.TestConnection should error without a websocket connection")
}

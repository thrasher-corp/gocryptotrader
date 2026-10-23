package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPISetHeartbeat(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	err := ex.WebsocketAPI().SetHeartbeat(t.Context(), 9*time.Second)
	require.ErrorIs(t, err, errHeartbeatIntervalTooShort, "WebsocketAPI.SetHeartbeat must reject an interval under 10 seconds")

	require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), 30*time.Second), "WebsocketAPI.SetHeartbeat must not error")
	if mockTests {
		requests := venue.received("public/set_heartbeat")
		require.Len(t, requests, 2, "the connection and WebsocketAPI.SetHeartbeat must each set the heartbeat")
		assert.JSONEq(t, `{"interval":15}`, string(requests[0].Params), "connecting should ask for a heartbeat every 15 seconds")
		assert.JSONEq(t, `{"interval":30}`, string(requests[1].Params), "WebsocketAPI.SetHeartbeat should send the interval in seconds")
	}

	err = newTestExchange(t).WebsocketAPI().SetHeartbeat(t.Context(), 30*time.Second)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.SetHeartbeat should error without a websocket connection")
}

func TestWebsocketAPIDisableHeartbeat(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().DisableHeartbeat(t.Context()), "WebsocketAPI.DisableHeartbeat must not error")
	if !mockTests {
		// The live connection is shared, so its heartbeat is restored for the other tests
		require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), websocketHeartbeatInterval), "WebsocketAPI.SetHeartbeat must restore the heartbeat")
	}

	err := newTestExchange(t).WebsocketAPI().DisableHeartbeat(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.DisableHeartbeat should error without a websocket connection")
}

func TestWebsocketAPIEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	err := ex.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), "bogus")
	require.ErrorIs(t, err, errInvalidCancelOnDisconnectScope, "WebsocketAPI.EnableCancelOnDisconnect must reject an undocumented scope")

	require.NoError(t, ex.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), CancelOnDisconnectScopeConnection), "WebsocketAPI.EnableCancelOnDisconnect must not error")
	if mockTests {
		requests := venue.received("private/enable_cancel_on_disconnect")
		require.Len(t, requests, 1, "WebsocketAPI.EnableCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{"scope":"connection"}`, string(requests[0].Params), "WebsocketAPI.EnableCancelOnDisconnect should send the scope")
	} else {
		require.NoError(t, ex.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), CancelOnDisconnectScopeConnection), "WebsocketAPI.DisableCancelOnDisconnect must restore the connection's setting")
	}

	unauthenticated := newTestExchange(t)
	err = unauthenticated.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EnableCancelOnDisconnect should error without an authenticated connection")
}

func TestWebsocketAPIDisableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	err := ex.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), "bogus")
	require.ErrorIs(t, err, errInvalidCancelOnDisconnectScope, "WebsocketAPI.DisableCancelOnDisconnect must reject an undocumented scope")

	require.NoError(t, ex.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), ""), "WebsocketAPI.DisableCancelOnDisconnect must not error")
	if mockTests {
		requests := venue.received("private/disable_cancel_on_disconnect")
		require.Len(t, requests, 1, "WebsocketAPI.DisableCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{}`, string(requests[0].Params), "WebsocketAPI.DisableCancelOnDisconnect should leave out an empty scope, so Deribit applies the connection scope")
	}

	unauthenticated := newTestExchange(t)
	err = unauthenticated.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.DisableCancelOnDisconnect should error without an authenticated connection")
}

func TestWebsocketAPIGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().GetCancelOnDisconnect(t.Context(), "bogus")
	require.ErrorIs(t, err, errInvalidCancelOnDisconnectScope, "WebsocketAPI.GetCancelOnDisconnect must reject an undocumented scope")

	result, err := ex.WebsocketAPI().GetCancelOnDisconnect(t.Context(), CancelOnDisconnectScopeConnection)
	require.NoError(t, err, "WebsocketAPI.GetCancelOnDisconnect must not error")
	if mockTests {
		assert.Equal(t, &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}, result, "WebsocketAPI.GetCancelOnDisconnect should decode the connection's setting")
	} else {
		assert.Equal(t, CancelOnDisconnectScopeConnection, result.Scope, "WebsocketAPI.GetCancelOnDisconnect should return the connection scope")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetCancelOnDisconnect should error without an authenticated connection")
}

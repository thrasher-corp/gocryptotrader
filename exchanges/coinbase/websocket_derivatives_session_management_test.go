package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIDisableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	err := e.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), "subaccount")
	require.ErrorIs(t, err, errDerivativesInvalidCancelOnDisconnectScope, "WebsocketAPI.DisableCancelOnDisconnect must refuse an undocumented scope")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newDerivativesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), ""), "WebsocketAPI.DisableCancelOnDisconnect must not error")
	if mockTests {
		requests := venue.requestsFor("private/disable_cancel_on_disconnect")
		require.Len(t, requests, 1, "WebsocketAPI.DisableCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{}`, string(requests[0].Params), "WebsocketAPI.DisableCancelOnDisconnect should leave an empty scope out, so the gateway applies the connection scope")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	err = unauthenticated.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.DisableCancelOnDisconnect should refuse an unauthenticated connection")
}

func TestWebsocketAPIEnableCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	err := e.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), "subaccount")
	require.ErrorIs(t, err, errDerivativesInvalidCancelOnDisconnectScope, "WebsocketAPI.EnableCancelOnDisconnect must refuse an undocumented scope")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newDerivativesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), DerivativesCancelOnDisconnectScopeConnection), "WebsocketAPI.EnableCancelOnDisconnect must not error")
	if mockTests {
		requests := venue.requestsFor("private/enable_cancel_on_disconnect")
		require.Len(t, requests, 1, "WebsocketAPI.EnableCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{"scope":"connection"}`, string(requests[0].Params), "WebsocketAPI.EnableCancelOnDisconnect should send the scope")
	} else {
		require.NoError(t, ex.WebsocketAPI().DisableCancelOnDisconnect(t.Context(), DerivativesCancelOnDisconnectScopeConnection), "WebsocketAPI.DisableCancelOnDisconnect must restore the connection's setting")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	err = unauthenticated.WebsocketAPI().EnableCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.EnableCancelOnDisconnect should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().GetCancelOnDisconnect(t.Context(), "subaccount")
	require.ErrorIs(t, err, errDerivativesInvalidCancelOnDisconnectScope, "WebsocketAPI.GetCancelOnDisconnect must refuse an undocumented scope")

	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetCancelOnDisconnect(t.Context(), DerivativesCancelOnDisconnectScopeConnection)
	require.NoError(t, err, "WebsocketAPI.GetCancelOnDisconnect must not error")
	if mockTests {
		exp := &DerivativesCancelOnDisconnectResponse{Scope: "connection"}
		assert.Equal(t, exp, resp, "WebsocketAPI.GetCancelOnDisconnect should decode every field")
	} else {
		assert.Equal(t, DerivativesCancelOnDisconnectScopeConnection, resp.Scope, "WebsocketAPI.GetCancelOnDisconnect should return the connection scope")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetCancelOnDisconnect(t.Context(), "")
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetCancelOnDisconnect should refuse an unauthenticated connection")
}

func TestWebsocketAPIDisableHeartbeat(t *testing.T) {
	t.Parallel()
	ex, venue := newDerivativesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().DisableHeartbeat(t.Context()), "WebsocketAPI.DisableHeartbeat must not error")
	if mockTests {
		requests := venue.requestsFor("public/disable_heartbeat")
		require.Len(t, requests, 1, "WebsocketAPI.DisableHeartbeat must send one request")
		assert.Empty(t, requests[0].Params, "WebsocketAPI.DisableHeartbeat should send no params")
	} else {
		// The live connection is shared, so its heartbeats are restored for the other tests
		require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), derivativesHeartbeatInterval), "WebsocketAPI.SetHeartbeat must restore the connection's heartbeats")
	}

	err := newTestExchange(t).WebsocketAPI().DisableHeartbeat(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.DisableHeartbeat should error without a websocket connection")
}

func TestWebsocketAPISetHeartbeat(t *testing.T) {
	t.Parallel()
	ex, venue := newDerivativesWebsocketTestExchange(t)
	err := ex.WebsocketAPI().SetHeartbeat(t.Context(), derivativesMinimumHeartbeatInterval-time.Second)
	require.ErrorIs(t, err, errDerivativesHeartbeatIntervalTooShort, "WebsocketAPI.SetHeartbeat must refuse an interval under 10 seconds, which the gateway refuses")
	err = ex.WebsocketAPI().SetHeartbeat(t.Context(), 12500*time.Millisecond)
	require.ErrorIs(t, err, errDerivativesHeartbeatIntervalFraction, "WebsocketAPI.SetHeartbeat must refuse a fraction of a second, which the gateway refuses")

	require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), 30*time.Second), "WebsocketAPI.SetHeartbeat must not error")
	if mockTests {
		// The gateway's public streams and its private connection ask for heartbeats themselves once they connect, so
		// their requests may arrive after this one
		require.Eventually(t, func() bool { return len(venue.requestsFor("public/set_heartbeat")) == 3 }, 30*time.Second, time.Millisecond, "each gateway connection and WebsocketAPI.SetHeartbeat must set heartbeats")
		intervals := make([]float64, 0, 3)
		for _, req := range venue.requestsFor("public/set_heartbeat") {
			var params DerivativesHeartbeatWireRequest
			require.NoError(t, json.Unmarshal(req.Params, &params), "the heartbeat params must decode")
			intervals = append(intervals, params.Interval)
		}
		assert.ElementsMatch(t, []float64{derivativesHeartbeatInterval.Seconds(), derivativesHeartbeatInterval.Seconds(), 30}, intervals, "WebsocketAPI.SetHeartbeat should send the interval in seconds")
	} else {
		// The live connection is shared, so its heartbeats are restored for the other tests
		require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), derivativesHeartbeatInterval), "WebsocketAPI.SetHeartbeat must restore the connection's heartbeats")
	}

	err = newTestExchange(t).WebsocketAPI().SetHeartbeat(t.Context(), 30*time.Second)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.SetHeartbeat should error without a websocket connection")
}

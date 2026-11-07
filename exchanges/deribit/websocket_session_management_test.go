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
		requests := venue.receivedOn(requestVenueConnection, "public/set_heartbeat")
		require.Len(t, requests, 2, "the request connection and WebsocketAPI.SetHeartbeat must each set its heartbeat")
		assert.JSONEq(t, `{"interval":15}`, string(requests[0].Params), "connecting should ask for a heartbeat every 15 seconds")
		assert.JSONEq(t, `{"interval":30}`, string(requests[1].Params), "WebsocketAPI.SetHeartbeat should send the interval in seconds")
	}

	err = newTestExchange(t).WebsocketAPI().SetHeartbeat(t.Context(), 30*time.Second)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.SetHeartbeat should error without a websocket connection")

	t.Run("interval of 10 seconds", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("an interval at the limit is checked against the mock websocket, which records the request")
		}
		ex, venue := newWebsocketTestExchange(t)
		// testdata/http.json holds no answer for this interval, so the mock gives Deribit's answer to the others
		venue.answerNext("public/set_heartbeat", `{"result":"ok"}`)
		require.NoError(t, ex.WebsocketAPI().SetHeartbeat(t.Context(), 10*time.Second), "WebsocketAPI.SetHeartbeat must accept an interval of 10 seconds")
		requests := venue.receivedOn(requestVenueConnection, "public/set_heartbeat")
		require.Len(t, requests, 2, "the request connection and WebsocketAPI.SetHeartbeat must each set its heartbeat")
		assert.JSONEq(t, `{"interval":10}`, string(requests[1].Params), "WebsocketAPI.SetHeartbeat should send an interval of 10 seconds")
	})
	// The last interval is too large for float64 seconds to hold its nanosecond, so a check made in float seconds fails it
	for _, interval := range []time.Duration{10500 * time.Millisecond, 10*time.Second + 999*time.Microsecond, 10*time.Second + time.Nanosecond, (1<<24)*time.Second + time.Nanosecond} {
		t.Run("interval of "+interval.String(), func(t *testing.T) {
			t.Parallel()
			err := newTestExchange(t).WebsocketAPI().SetHeartbeat(t.Context(), interval)
			assert.ErrorIs(t, err, errHeartbeatIntervalNotWholeSeconds, "WebsocketAPI.SetHeartbeat should reject an interval that is not whole seconds")
		})
	}
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

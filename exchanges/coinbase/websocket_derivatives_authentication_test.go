package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPILogout(t *testing.T) {
	t.Parallel()
	t.Run("logs out of its connection", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex, venue := newDedicatedDerivativesWebsocketTestExchange(t)
		// A session acts for the account of its API key alone, so another key's logout is refused
		other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "organizations/other/apiKeys/other", Secret: "other"})
		assert.ErrorIs(t, ex.WebsocketAPI().Logout(other, true), errDerivativesWebsocketOtherAccount, "WebsocketAPI.Logout should refuse another API key's logout")
		if mockTests {
			assert.Empty(t, venue.requestsFor("private/logout"), "WebsocketAPI.Logout should not send another API key's logout")
		}
		conn, err := ex.Websocket.GetConnection(derivativesPrivateConnection)
		require.NoError(t, err, "GetConnection must return the gateway's private connection")
		require.True(t, isConnected(conn), "the private connection must be connected before the logout")
		require.NoError(t, ex.WebsocketAPI().Logout(t.Context(), false), "WebsocketAPI.Logout must not error")
		// The gateway closes the connection in place of a response; the manager dials any reconnection afresh, so the
		// connection logged out of stays closed
		require.Eventually(t, func() bool { return !isConnected(conn) }, 30*time.Second, 10*time.Millisecond, "the gateway must close the connection logged out of")
		if mockTests {
			requests := venue.requestsFor("private/logout")
			require.Len(t, requests, 1, "WebsocketAPI.Logout must send one logout request")
			assert.JSONEq(t, `{"invalidate_token":false}`, string(requests[0].Params), "WebsocketAPI.Logout should send a false invalidate_token, which the gateway would default to true")
		}
	})
	t.Run("refuses an unauthenticated connection", func(t *testing.T) {
		t.Parallel()
		unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
		err := unauthenticated.WebsocketAPI().Logout(t.Context(), true)
		assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.Logout should refuse an unauthenticated connection")
	})
	t.Run("errors without a websocket connection", func(t *testing.T) {
		t.Parallel()
		disconnected := newTestExchange(t)
		disconnected.Websocket.SetCanUseAuthenticatedEndpoints(true)
		err := disconnected.WebsocketAPI().Logout(t.Context(), true)
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Logout should error without a websocket connection")
	})
}

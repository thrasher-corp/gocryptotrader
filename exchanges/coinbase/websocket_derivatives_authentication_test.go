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
	if !mockTests {
		t.Skip("WebsocketAPI.Logout closes the connection the live tests share")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	// A session acts for the account of its API key alone, so another key's logout is refused
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "organizations/other/apiKeys/other", Secret: "other"})
	assert.ErrorIs(t, ex.WebsocketAPI().Logout(other, true), errDerivativesWebsocketOtherAccount, "WebsocketAPI.Logout should refuse another API key's logout")
	assert.Empty(t, venue.requestsFor("private/logout"), "WebsocketAPI.Logout should not send another API key's logout")
	require.NoError(t, ex.WebsocketAPI().Logout(t.Context(), false), "WebsocketAPI.Logout must not error")
	// The gateway closes the connection in place of a response, so the request is only seen once the mock receives it
	require.Eventually(t, func() bool { return len(venue.requestsFor("private/logout")) == 1 }, 5*time.Second, 10*time.Millisecond, "WebsocketAPI.Logout must send one logout request")
	assert.JSONEq(t, `{"invalidate_token":false}`, string(venue.requestsFor("private/logout")[0].Params), "WebsocketAPI.Logout should send a false invalidate_token, which the gateway would default to true")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	err := unauthenticated.WebsocketAPI().Logout(t.Context(), true)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.Logout should refuse an unauthenticated connection")

	disconnected := newTestExchange(t)
	disconnected.Websocket.SetCanUseAuthenticatedEndpoints(true)
	err = disconnected.WebsocketAPI().Logout(t.Context(), true)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Logout should error without a websocket connection")
}

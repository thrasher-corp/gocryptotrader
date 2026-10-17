package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIAuthenticate(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, venue := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().Authenticate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.Authenticate must reject a nil request")

	result, err := ex.WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
	require.NoError(t, err, "WebsocketAPI.Authenticate must not error")
	if mockTests {
		assert.Equal(t, testAccessToken("connection mainaccount"), result, "WebsocketAPI.Authenticate should decode the token")
		// The connection authenticated itself first, with a client signature
		requests := venue.received("public/auth")
		require.Len(t, requests, 2, "the connection and WebsocketAPI.Authenticate must each authenticate")
		var params map[string]any
		require.NoError(t, json.Unmarshal(requests[0].Params, &params), "the connection's authentication params must decode")
		assert.Equal(t, GrantTypeClientSignature, params["grant_type"], "the connection should authenticate with a client signature")
		assert.NotContains(t, params, "client_secret", "the connection should not send the client secret")
	} else {
		assert.NotEmpty(t, result.AccessToken, "WebsocketAPI.Authenticate should return an access token")
	}

	_, err = newTestExchange(t).WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Authenticate should error without a websocket connection")
}

func TestWebsocketAPIExchangeToken(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ExchangeToken needs a refresh token and a subaccount, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().ExchangeToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.ExchangeToken must reject a nil request")

	result, err := ex.WebsocketAPI().ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken, SubjectID: mockSubjectID})
	require.NoError(t, err, "WebsocketAPI.ExchangeToken must not error")
	exp := testAccessToken("session:named_session mainaccount")
	exp.EnabledFeatures = nil
	assert.Equal(t, exp, result, "WebsocketAPI.ExchangeToken should decode the token")
}

func TestWebsocketAPIForkToken(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ForkToken needs the refresh token of a session scoped token, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().ForkToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.ForkToken must reject a nil request")

	result, err := ex.WebsocketAPI().ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken, SessionName: mockSessionName})
	require.NoError(t, err, "WebsocketAPI.ForkToken must not error")
	exp := testAccessToken("session:named_session mainaccount")
	exp.EnabledFeatures = nil
	assert.Equal(t, exp, result, "WebsocketAPI.ForkToken should decode the token")
}

func TestWebsocketAPILogout(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.Logout closes the connection the live tests share")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().Logout(t.Context(), false), "WebsocketAPI.Logout must not error")
	assert.Eventually(t, func() bool {
		return len(venue.received("private/logout")) == 1
	}, 5*time.Second, 10*time.Millisecond, "WebsocketAPI.Logout should send one logout request")
	requests := venue.received("private/logout")
	if assert.Len(t, requests, 1, "WebsocketAPI.Logout should send one logout request") {
		assert.JSONEq(t, `{"invalidate_token":false}`, string(requests[0].Params), "WebsocketAPI.Logout should send a false invalidate_token, which Deribit would otherwise default to true")
	}

	err := newTestExchange(t).WebsocketAPI().Logout(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Logout should error without an authenticated connection")
}

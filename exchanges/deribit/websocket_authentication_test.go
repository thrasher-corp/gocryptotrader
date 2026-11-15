package deribit

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestWebsocketAPIAuthenticate(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().Authenticate(t.Context(), nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.Authenticate must reject a nil request")
		result, err := ex.WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
		require.NoError(t, err, "WebsocketAPI.Authenticate must not error")
		assert.NotEmpty(t, result.AccessToken, "WebsocketAPI.Authenticate should return an access token")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, venue := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().Authenticate(t.Context(), nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.Authenticate must reject a nil request")
		result, err := ex.WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
		require.NoError(t, err, "WebsocketAPI.Authenticate must not error")
		assert.Equal(t, testAccessToken("connection mainaccount"), result, "WebsocketAPI.Authenticate should decode the token")
		// The manager logged the request connection in first, with a client signature
		requests := venue.receivedOn(requestVenueConnection, "public/auth")
		require.Len(t, requests, 2, "the manager and WebsocketAPI.Authenticate must each log the request connection in")
		var params map[string]any
		require.NoError(t, json.Unmarshal(requests[0].Params, &params), "the connection's authentication params must decode")
		assert.Equal(t, GrantTypeClientSignature, params["grant_type"], "the connection should authenticate with a client signature")
		assert.NotContains(t, params, "client_secret", "the connection should not send the client secret")

		venue.answerNext("public/auth", `{"result":{"token_type":"bearer"}}`)
		_, err = ex.WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
		assert.ErrorIs(t, err, errNoAccessToken, "WebsocketAPI.Authenticate should refuse a result without an access token")
	})

	arg := &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientID: testClientID, ClientSecret: testClientSecret}
	_, err := newTestExchange(t).WebsocketAPI().Authenticate(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Authenticate should error without a websocket connection")

	t.Run("authenticates an unauthenticated connection", func(t *testing.T) {
		t.Parallel()
		ex, venue := newUnauthenticatedWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().GetSubaccounts(t.Context(), true)
		require.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetSubaccounts must not be sent before the connection authenticates")
		_, err = ex.WebsocketAPI().Authenticate(t.Context(), testAuthenticateRequest(t))
		require.NoError(t, err, "WebsocketAPI.Authenticate must not error")
		assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "WebsocketAPI.Authenticate should mark the connection authenticated")
		result, err := ex.WebsocketAPI().GetSubaccounts(t.Context(), true)
		require.NoError(t, err, "WebsocketAPI.GetSubaccounts must not error once the connection authenticates")
		assert.Equal(t, testSubaccounts(), result, "WebsocketAPI.GetSubaccounts should decode every field")
		assert.Len(t, venue.received("private/get_subaccounts"), 1, "only the request made once the connection authenticated should be sent")
	})
}

func TestAuthenticateConnection(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newDedicatedWebsocketTestExchange(t, nil)
		result, err := ex.authenticateConnection(t.Context(), requestConnection(t, ex), testAuthenticateRequest(t))
		require.NoError(t, err, "authenticateConnection must not error")
		assert.NotEmpty(t, result.AccessToken, "authenticateConnection should return an access token")
		assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticateConnection should mark the connection authenticated")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		for _, tc := range []struct {
			name   string
			arg    *AuthenticateRequest
			answer string
			sent   bool
			err    error
			code   int64
		}{
			{name: "a token", arg: testAuthenticateRequest(t), sent: true},
			{name: "a nil request", err: common.ErrNilPointer},
			{name: "an undocumented grant type", arg: &AuthenticateRequest{GrantType: "password"}, err: errInvalidGrantType},
			{name: "a result without an access token", arg: testAuthenticateRequest(t), answer: `{"result":{"token_type":"bearer"}}`, sent: true, err: errNoAccessToken},
			{name: "a refusal", arg: testAuthenticateRequest(t), answer: `{"error":{"code":13004,"message":"invalid_credentials"}}`, sent: true, code: 13004},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, venue := newUnauthenticatedWebsocketTestExchange(t)
				if tc.answer != "" {
					venue.answerNext("public/auth", tc.answer)
				}
				result, err := ex.authenticateConnection(t.Context(), requestConnection(t, ex), tc.arg)
				switch {
				case tc.code != 0:
					var apiErr *APIError
					require.ErrorAsf(t, err, &apiErr, "authenticateConnection must return Deribit's error for %s", tc.name)
					assert.Equalf(t, tc.code, apiErr.Code, "authenticateConnection should return Deribit's code for %s", tc.name)
				case tc.err != nil:
					assert.ErrorIsf(t, err, tc.err, "authenticateConnection should refuse %s", tc.name)
				default:
					require.NoErrorf(t, err, "authenticateConnection must not error for %s", tc.name)
					assert.Equalf(t, testAccessToken("connection mainaccount"), result, "authenticateConnection should return the token for %s", tc.name)
				}
				assert.Equalf(t, tc.err == nil && tc.code == 0, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticateConnection should mark the connection authenticated only once it returns a token, for %s", tc.name)
				expSent := 0
				if tc.sent {
					expSent = 1
				}
				assert.Lenf(t, venue.receivedOn(requestVenueConnection, "public/auth"), expSent, "authenticateConnection should send the request over the connection given only once it is valid, for %s", tc.name)
			})
		}
	})
}

func TestWebsocketAPIExchangeToken(t *testing.T) {
	t.Parallel()
	_, err := newTestExchange(t).WebsocketAPI().ExchangeToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.ExchangeToken must reject a nil request")

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		auth, err := e.Authenticate(t.Context(), testAuthenticateRequest(t))
		require.NoError(t, err, "Authenticate must not error")
		// A websocket of its own, as a token exchange may switch the connection it is sent on to another subject
		ex, _ := newDedicatedWebsocketTestExchange(t, nil)
		_, err = ex.WebsocketAPI().ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: auth.RefreshToken, SubjectID: liveForeignID})
		requireRefusedOnMerits(t, err, "WebsocketAPI.ExchangeToken", "subject_id")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken, SubjectID: mockSubjectID})
		require.NoError(t, err, "WebsocketAPI.ExchangeToken must not error")
		exp := testAccessToken("session:named_session mainaccount")
		exp.EnabledFeatures = nil
		assert.Equal(t, exp, result, "WebsocketAPI.ExchangeToken should decode the token")
	})
}

func TestWebsocketAPIForkToken(t *testing.T) {
	t.Parallel()
	_, err := newTestExchange(t).WebsocketAPI().ForkToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.ForkToken must reject a nil request")

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		arg := testAuthenticateRequest(t)
		arg.Scope = "session:" + liveSessionName
		auth, err := e.Authenticate(t.Context(), arg)
		require.NoError(t, err, "Authenticate must not error for a session scoped token")
		// A websocket of its own, as a fork may move the connection it is sent on to the new session
		ex, _ := newDedicatedWebsocketTestExchange(t, nil)
		result, err := ex.WebsocketAPI().ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: auth.RefreshToken, SessionName: liveSessionName + "-fork"})
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.ForkToken") {
			assert.NotEmpty(t, result.AccessToken, "WebsocketAPI.ForkToken should return an access token")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken, SessionName: mockSessionName})
		require.NoError(t, err, "WebsocketAPI.ForkToken must not error")
		exp := testAccessToken("session:named_session mainaccount")
		exp.EnabledFeatures = nil
		assert.Equal(t, exp, result, "WebsocketAPI.ForkToken should decode the token")
	})
}

func TestWebsocketAPILogout(t *testing.T) {
	t.Parallel()
	err := newTestExchange(t).WebsocketAPI().Logout(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Logout should error without an authenticated connection")

	unconnected := newTestExchange(t)
	unconnected.Websocket.SetCanUseAuthenticatedEndpoints(true)
	err = unconnected.WebsocketAPI().Logout(t.Context(), true)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Logout should error without a connection")

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		// A websocket of its own, as a logout closes the connection it is sent on
		ex, _ := newDedicatedWebsocketTestExchange(t, nil)
		require.NoError(t, ex.WebsocketAPI().Logout(t.Context(), false), "WebsocketAPI.Logout must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, venue := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().Logout(t.Context(), false), "WebsocketAPI.Logout must not error")
		assert.Eventually(t, func() bool {
			return len(venue.received("private/logout")) == 1
		}, 5*time.Second, 10*time.Millisecond, "WebsocketAPI.Logout should send one logout request")
		requests := venue.received("private/logout")
		if assert.Len(t, requests, 1, "WebsocketAPI.Logout should send one logout request") {
			assert.JSONEq(t, `{"invalidate_token":false}`, string(requests[0].Params), "WebsocketAPI.Logout should send a false invalidate_token, which Deribit would otherwise default to true")
			assert.Equal(t, requestVenueConnection, requests[0].connection, "WebsocketAPI.Logout should close the request connection")
		}
		// The manager connects the websocket again once Deribit closes the request connection, which the test waits for,
		// so no request of the new connections is under way when it ends
		require.Eventually(t, func() bool {
			return ex.Websocket.IsConnected() && venue.connectionCount() == 4 && len(venue.received("public/set_heartbeat")) == 4
		}, 10*time.Second, 10*time.Millisecond, "the manager must connect the websocket again after a logout")
		for _, connection := range []int{3, 4} {
			t.Run("connection "+strconv.Itoa(connection), func(t *testing.T) {
				t.Parallel()
				assert.Lenf(t, venue.receivedOn(connection, "public/auth"), 1, "the manager should log new connection %d in after a logout", connection)
			})
		}

		// An aborted rate limit barrier fails the logout's rate limit, which the limiters every Exchange shares cannot be
		// made to do for one test
		contexts, err := request.NewRateLimitBarrierContexts(t.Context(), 2)
		require.NoError(t, err, "NewRateLimitBarrierContexts must not error")
		request.AbortRateLimitBarrier(contexts[0])
		assert.ErrorIs(t, ex.WebsocketAPI().Logout(contexts[1], false), request.ErrRateLimitBarrierRejected, "WebsocketAPI.Logout should return a rate limit failure")
		assert.Len(t, venue.received("private/logout"), 1, "WebsocketAPI.Logout should send nothing once its rate limit fails")
	})
}

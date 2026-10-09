package okx

import (
	"context"
	"sync/atomic"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestCheckWebsocketOrderCredentials(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		creds   accounts.Credentials
		missing bool
	}{
		{name: "same account", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}},
		{name: "different key", creds: accounts.Credentials{Key: "other", Secret: "secret", ClientID: "passphrase"}},
		{name: "different secret", creds: accounts.Credentials{Key: "key", Secret: "other", ClientID: "passphrase"}},
		{name: "different passphrase", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "other"}},
		{name: "different subaccount", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase", SubAccount: "other"}},
		{name: "unconfirmed login", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}, missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
			conn, err := ex.Websocket.GetConnection(privateConnection)
			require.NoError(t, err, "private connection must be available")
			loggedIn := accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}
			ex.SetCredentials(&loggedIn)
			ex.setWebsocketOrderCredentials(conn, &loggedIn)
			if tc.missing {
				ex.setWebsocketOrderCredentials(conn, nil)
			}
			store := &accounts.ContextCredentialsStore{}
			store.Load(&tc.creds)
			ctx := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store)
			err = ex.checkWebsocketOrderCredentials(ctx, conn)
			if tc.missing || tc.creds != loggedIn {
				assert.ErrorIs(t, err, errWebsocketOrderCredentialsMismatch, "unconfirmed or different credentials should be rejected")
			} else {
				assert.NoError(t, err, "confirmed matching credentials should be accepted")
			}
			ex.SkipAuthCheck, ex.API.AuthenticatedSupport, ex.API.AuthenticatedWebsocketSupport = false, false, false
			assert.Error(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), "unavailable credentials should return the credential error")
		})
	}
}

func TestSetWebsocketOrderCredentials(t *testing.T) {
	t.Parallel()
	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
	conn, err := ex.Websocket.GetConnection(privateConnection)
	require.NoError(t, err, "private connection must be available")
	ex.websocketOrderCredentials = nil
	creds := accounts.Credentials{Key: "first"}
	ex.setWebsocketOrderCredentials(conn, &creds)
	creds.Key = "mutated"
	assert.Equal(t, "first", ex.websocketOrderCredentials[conn].Key, "login identity should retain an independent credential snapshot")
	ex.setWebsocketOrderCredentials(conn, &creds)
	assert.Equal(t, "mutated", ex.websocketOrderCredentials[conn].Key, "a subsequent confirmed login should replace the snapshot")
	ex.setWebsocketOrderCredentials(conn, nil)
	assert.Empty(t, ex.websocketOrderCredentials, "invalidated login should remove the identity")
	for range maxWebsocketOrderCredentials + 1 {
		newConn, err := ex.Websocket.CreateTestConnection(privateConnection)
		require.NoError(t, err, "replacement connection must be created")
		ex.setWebsocketOrderCredentials(newConn, &creds)
		assert.LessOrEqual(t, len(ex.websocketOrderCredentials), maxWebsocketOrderCredentials, "retired connection identities should remain bounded")
		assert.Equal(t, creds, ex.websocketOrderCredentials[newConn], "the latest confirmed connection should retain its identity")
	}
}

func TestWsAuthenticateConnection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		response string
		success  bool
	}{
		{name: "confirmed", response: `{"event":"login","code":"0"}`, success: true},
		{name: "rejected", response: `{"event":"login","code":"60009","msg":"rejected"}`},
		{name: "malformed code", response: `{"event":"login","code":"invalid"}`},
		{name: "missing code", response: `{"event":"login"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var orders atomic.Int64
			ex := connectOKXWithMockedWebsocket(t, func(tb testing.TB, payload []byte, conn *gws.Conn) error {
				tb.Helper()
				var req WebsocketAuthLogin
				if err := json.Unmarshal(payload, &req); err != nil {
					return err
				}
				if req.Operation == operationLogin {
					assert.Equal(tb, "account-a", req.Arguments[0].APIKey, "login should use the requested account")
					assert.Equal(tb, "passphrase", req.Arguments[0].Passphrase, "login should use the requested passphrase")
					return conn.WriteMessage(gws.TextMessage, []byte(tc.response))
				}
				orders.Add(1)
				return okxOrderWsMock(tb, payload, conn)
			})
			creds := accounts.Credentials{Key: "account-a", Secret: "secret", ClientID: "passphrase"}
			ex.SetCredentials(&creds)
			conn, err := ex.Websocket.GetConnection(privateConnection)
			require.NoError(t, err, "private connection must be available")
			ex.setWebsocketOrderCredentials(conn, &creds)
			err = ex.wsAuthenticateConnection(t.Context(), conn)
			if !tc.success {
				require.ErrorIs(t, err, request.ErrAuthRequestFailed, "failed login must retain the authentication sentinel")
				assert.ErrorIs(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), errWebsocketOrderCredentialsMismatch, "failed login should remove the prior account identity")
				return
			}
			require.NoError(t, err, "real mocked login acknowledgement must authenticate")
			assert.NoError(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), "confirmed login should register its account")
			other := creds
			other.Key = "account-b"
			store := &accounts.ContextCredentialsStore{}
			store.Load(&other)
			ctx := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store)
			for _, operation := range []string{"order", "batch-orders"} {
				assert.ErrorIs(t, ex.SendAuthenticatedWebsocketRequest(ctx, placeOrderEPL, ex.MessageID(), operation, []PlaceOrderRequestParam{{}}, nil), errWebsocketOrderCredentialsMismatch, "another account's order should be rejected before sending")
			}
			assert.Zero(t, orders.Load(), "mismatched account requests should never reach the socket")
			var response any
			require.NoError(t, ex.SendAuthenticatedWebsocketRequest(t.Context(), placeOrderEPL, ex.MessageID(), "order", []PlaceOrderRequestParam{{}}, &response), "matching account order must use the authenticated socket")
			assert.EqualValues(t, 1, orders.Load(), "matching account order should reach the socket")
			newConn, err := ex.Websocket.CreateTestConnection(privateConnection)
			require.NoError(t, err, "replacement connection must be created")
			assert.ErrorIs(t, ex.checkWebsocketOrderCredentials(t.Context(), newConn), errWebsocketOrderCredentialsMismatch, "replacement connection should require its own successful login")
		})
	}
}

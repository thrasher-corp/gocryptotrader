package btse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type authenticationTestConnection struct {
	websocket.Connection
	response []byte
	request  wsSub
}

func (c *authenticationTestConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, payload any) ([]byte, error) {
	var ok bool
	c.request, ok = payload.(wsSub)
	if !ok {
		return nil, common.GetTypeAssertError("wsSub", payload, "websocket authentication request")
	}
	return c.response, nil
}

func TestWsAuthenticate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		response string
		err      error
	}{
		{name: "accepted", response: `{"event":"login","success":true}`},
		{name: "rejected", response: `{"event":"login","success":false}`, err: errWebsocketAuthenticationFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
			ex.SkipAuthCheck = true
			ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
			conn := &authenticationTestConnection{response: []byte(tc.response)}
			assert.ErrorIs(t, ex.wsAuthenticate(t.Context(), conn), tc.err, "authentication should reflect the acknowledgement")
			assert.Equal(t, "authKeyExpires", conn.request.Operation, "authentication should retain its wire operation")
			require.Len(t, conn.request.Arguments, 3, "authentication must include key, nonce and signature")
			assert.Equal(t, "test-key", conn.request.Arguments[0], "authentication should use configured credentials")
		})
	}
}

func TestWsHandleDataConnectionMatching(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	other := testexch.GetMockConn(t, ex, "wss://other.example")
	conn := testexch.GetMockConn(t, ex, "wss://target.example")
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	otherResponses, err := other.MatchReturnResponses(ctx, "login", 1)
	require.NoError(t, err, "competing matcher must register first")
	responses, err := conn.MatchReturnResponses(ctx, "login", 1)
	require.NoError(t, err, "target matcher must register")
	require.NoError(t, ex.wsHandleData(ctx, conn, []byte(`{"event":"login","success":true}`)), "login response must be handled")
	select {
	case <-responses:
	case <-ctx.Done():
		t.Fatal("response must reach its physical connection")
	}
	select {
	case <-otherResponses:
		t.Fatal("response must not reach a different connection")
	default:
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	t.Run("disabled exchange", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		ex.SetEnabled(false)
		assert.ErrorIs(t, ex.WsConnect(), websocket.ErrWebsocketNotEnabled, "disabled exchange should not connect")
	})
	t.Run("managed lifecycle", func(t *testing.T) {
		t.Parallel()
		operations := make(chan string, 8)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
			if !assert.NoError(t, err, "mock websocket upgrade should succeed") {
				return
			}
			defer conn.Close()
			for {
				var req wsSub
				if err := conn.ReadJSON(&req); err != nil {
					return
				}
				operations <- req.Operation
				if req.Operation == "authKeyExpires" {
					if err := conn.WriteJSON(WsLoginAcknowledgement{Event: "login", Success: true}); err != nil {
						return
					}
				}
			}
		}))
		t.Cleanup(server.Close)
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		ex.SkipAuthCheck = true
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.API.AuthenticatedWebsocketSupport = true
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.MyTradesChannel, Asset: asset.Spot, Authenticated: true}}
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "mock endpoint must be configured")
		t.Cleanup(func() {
			_ = ex.Websocket.Disable()
			_ = ex.Websocket.Shutdown()
		})
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		var previous websocket.Connection
		for range 2 {
			require.NoError(t, ex.Websocket.Connect(ctx), "managed connection must start")
			conn, err := ex.Websocket.GetConnection(asset.Spot)
			require.NoError(t, err, "managed route must exist")
			assert.NotSame(t, previous, conn, "reconnect should replace the physical connection")
			assert.Len(t, conn.Subscriptions().List(), 1, "connection should own its subscription")
			for _, operation := range []string{"authKeyExpires", "subscribe"} {
				select {
				case got := <-operations:
					assert.Equal(t, operation, got, "authentication should precede subscription")
				case <-ctx.Done():
					t.Fatal("expected websocket operation must arrive")
				}
			}
			require.NoError(t, ex.Websocket.Shutdown(), "managed connection must shut down")
			assert.Empty(t, ex.Websocket.GetSubscriptions(), "shutdown should release subscription ownership")
			previous = conn
		}
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		testexch.SkipTestIfCannotUseAuthenticatedWebsocket(t, e)
		testexch.SetupWs(t, e)
		assert.True(t, e.Websocket.IsConnected(), "live websocket should connect")
	})
}

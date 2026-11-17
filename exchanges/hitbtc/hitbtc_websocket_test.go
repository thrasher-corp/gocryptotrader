package hitbtc

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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsConnect(t *testing.T) {
	t.Run("public entry uses manager lifecycle", func(t *testing.T) {
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		require.False(t, ex.Websocket.IsEnabled(), "fixture websocket must start disabled")
		assert.ErrorIs(t, ex.WsConnect(), websocket.ErrWebsocketNotEnabled, "public entry should return manager lifecycle errors")
		assert.Nil(t, ex.Websocket.Conn, "public entry should not allocate a legacy connection")
	})
	t.Run("managed lifecycle", func(t *testing.T) {
		methods := make(chan string, 20)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var req struct {
					Method string `json:"method"`
					ID     string `json:"id"`
				}
				if err := json.Unmarshal(data, &req); err != nil {
					return
				}
				methods <- req.Method
				result := any(true)
				if req.Method == "getSymbol" {
					result = map[string]string{"id": "BTCUSD", "baseCurrency": "BTC", "quoteCurrency": "USD"}
				}
				if err := conn.WriteJSON(map[string]any{"jsonrpc": rpcVersion, "id": req.ID, "result": result}); err != nil {
					return
				}
			}
		}))
		defer server.Close()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "mock URL must be configured")
		public := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSD()}}
		private := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true}
		ex.Features.Subscriptions = subscription.List{public, private}
		t.Cleanup(func() {
			assert.NoError(t, ex.Websocket.Disable(), "manager should disable")
			err := ex.Websocket.Shutdown()
			if err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected, "manager should shut down or already be disconnected")
			}
		})
		nextMethod := func(want string) {
			t.Helper()
			select {
			case got := <-methods:
				require.Equal(t, want, got, "wire operation must preserve ordering")
			case <-time.After(5 * time.Second):
				t.Fatalf("wire operation %s must arrive", want)
			}
		}
		require.NoError(t, ex.Websocket.Enable(t.Context()), "manager must enable, connect and subscribe")
		nextMethod("login")
		// Subscription operations run concurrently, so their order is not prescribed.
		received := make([]string, 0, 2)
		for range 2 {
			select {
			case method := <-methods:
				received = append(received, method)
			case <-time.After(5 * time.Second):
				t.Fatal("subscriptions must arrive")
			}
		}
		assert.ElementsMatch(t, []string{"subscribeTicker", "subscribeReports"}, received, "both subscriptions should be sent after login")
		conn, err := ex.Websocket.GetConnection(hitbtcWebsocketAddress)
		require.NoError(t, err, "managed connection must be routable")
		assert.Nil(t, ex.Websocket.Conn, "legacy connection should not be allocated")
		assert.Len(t, conn.Subscriptions().List(), 2, "connection should own both subscriptions")
		response, err := ex.wsGetSymbols(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "request response must use the managed reader and matcher")
		require.NotNil(t, response, "symbol response must be returned")
		nextMethod("getSymbol")
		ex.Features.Subscriptions = subscription.List{public}
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "flush must remove private subscription")
		nextMethod("unsubscribeReports")
		assert.Len(t, conn.Subscriptions().List(), 1, "connection should retain only the public subscription")
		require.NoError(t, ex.Websocket.Shutdown(), "manager must shut down its socket and reader")
		require.NoError(t, ex.WsConnect(), "public entry must reconnect through the manager")
		nextMethod("login")
		nextMethod("subscribeTicker")
		replacement, err := ex.Websocket.GetConnection(hitbtcWebsocketAddress)
		require.NoError(t, err, "replacement connection must be routable")
		assert.NotSame(t, conn, replacement, "reconnection should allocate a new socket")
		assert.Len(t, replacement.Subscriptions().List(), 1, "reconnected socket should own current subscriptions")
		ex.Features.Subscriptions = nil
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "removing all streams must preserve direct requests")
		nextMethod("unsubscribeTicker")
		assert.Empty(t, replacement.Subscriptions().List(), "request-only socket should own no subscriptions")
		_, err = ex.wsGetSymbols(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "direct requests must remain available without streams")
		nextMethod("getSymbol")
	})
}

func TestWsGetTableName(t *testing.T) {
	t.Parallel()
	t.Run("response belongs to receiving connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		conn := &responseTestConnection{Connection: testexch.GetMockConn(t, ex, hitbtcWebsocketAddress)}
		response := []byte(`{"id":"request-1","result":true}`)
		method, err := ex.wsGetTableName(t.Context(), conn, response)
		require.NoError(t, err, "response matching must succeed")
		assert.Empty(t, method, "matched responses should not be processed as notifications")
		assert.Equal(t, "request-1", conn.signature, "response should use the receiving connection matcher")
		assert.Equal(t, response, conn.response, "matcher should receive the complete response")
	})
}

type responseTestConnection struct {
	websocket.Connection
	signature any
	response  []byte
}

func (c *responseTestConnection) IncomingWithData(signature any, response []byte) bool {
	c.signature, c.response = signature, response
	return true
}

func TestWsSendRequest(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	_, err := ex.wsSendRequest(context.Background(), "request-1", &WsRequest{Method: "getSymbol"})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "offline request should return the managed routing error")
}

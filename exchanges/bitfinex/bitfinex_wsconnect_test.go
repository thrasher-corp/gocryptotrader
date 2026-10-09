package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

type wsConnectFixtureConnection struct {
	websocket.Connection
	dialErr     error
	sendErr     error
	responseRaw []byte
	respond     func(context.Context, any, any) ([]byte, error)
	sent        []any

	dialCalls     atomic.Int32
	readCalls     atomic.Int32
	sendJSONCalls atomic.Int32
}

func (f *wsConnectFixtureConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *wsConnectFixtureConnection) ReadMessage() websocket.Response {
	f.readCalls.Add(1)
	return websocket.Response{}
}

func (f *wsConnectFixtureConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	f.sendJSONCalls.Add(1)
	f.sent = append(f.sent, payload)
	return f.sendErr
}

func (f *wsConnectFixtureConnection) SendMessageReturnResponse(ctx context.Context, _ request.EndpointLimit, signature, payload any) ([]byte, error) {
	if f.respond != nil {
		return f.respond(ctx, signature, payload)
	}
	f.sent = append(f.sent, payload)
	return f.responseRaw, f.sendErr
}

func TestWsConnect(t *testing.T) {
	t.Parallel()

	t.Run("success configures connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{}

		require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must not error")
		assert.Equal(t, int32(1), conn.dialCalls.Load(), "connection should dial once")
		assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "wsConnect should configure the connection")
	})

	t.Run("dial failure skips configuration", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		errDialFailed := errors.New("dial failed")
		conn := &wsConnectFixtureConnection{dialErr: errDialFailed}

		err := ex.wsConnect(t.Context(), conn)
		require.ErrorIs(t, err, errDialFailed, "wsConnect must return the dial error")
		assert.Equal(t, int32(1), conn.dialCalls.Load(), "connection should dial once")
		assert.Equal(t, int32(0), conn.sendJSONCalls.Load(), "ConfigureWS should not send when dial fails")
	})
}

func TestConfigureWS(t *testing.T) {
	t.Parallel()

	errSend := errors.New("send failure")
	conn := &wsConnectFixtureConnection{sendErr: errSend}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	assert.ErrorIs(t, ex.ConfigureWS(t.Context(), conn), errSend, "ConfigureWS should return the send failure")
	assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "ConfigureWS should send one request")
}

func TestGeneratePublicSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	subs, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	require.NotEmpty(t, subs, "generatePublicSubscriptions must return subscriptions")
	for _, sub := range subs {
		assert.False(t, sub.Authenticated, "generatePublicSubscriptions should return only public subscriptions")
	}
}

func TestSubscribeToChan(t *testing.T) {
	t.Parallel()

	t.Run("requires one subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.ErrorIs(t, ex.subscribeToChan(t.Context(), testexch.GetMockConn(t, ex, ""), nil), subscription.ErrNotSinglePair, "subscribeToChan should require one subscription")
	})

	t.Run("rejects invalid qualified channel", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.subscribeToChan(t.Context(), testexch.GetMockConn(t, ex, ""), subscription.List{{QualifiedChannel: "{"}})
		require.Error(t, err, "subscribeToChan must reject invalid JSON")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{
			Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
			responseRaw: []byte(`{"event":"subscribed"}`),
		}
		sub := &subscription.Subscription{QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`}

		require.NoError(t, ex.subscribeToChan(t.Context(), conn, subscription.List{sub}), "subscribeToChan must not error")
		assert.NotNil(t, ex.Websocket.GetSubscription(sub.Key), "subscribeToChan should store the temporary subscription")
	})
}

func TestUnsubscribeFromChan(t *testing.T) {
	t.Parallel()

	t.Run("rejects batching", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.ErrorIs(t, ex.unsubscribeFromChan(t.Context(), testexch.GetMockConn(t, ex, ""), nil), subscription.ErrBatchingNotSupported, "unsubscribeFromChan should reject batching")
	})

	t.Run("rejects an invalid key type", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.unsubscribeFromChan(t.Context(), testexch.GetMockConn(t, ex, ""), subscription.List{{Key: "invalid"}})
		require.Error(t, err, "unsubscribeFromChan must reject a key that is not a websocketChannelKey")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{
			Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
			responseRaw: []byte(`{"event":"unsubscribed"}`),
		}
		sub := &subscription.Subscription{Key: websocketChannelKey{conn, 42}}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")

		require.NoError(t, ex.unsubscribeFromChan(t.Context(), conn, subscription.List{sub}), "unsubscribeFromChan must not error")
		assert.Nil(t, ex.Websocket.GetSubscription(websocketChannelKey{conn, 42}), "unsubscribeFromChan should remove the subscription")
	})
}

func TestSubscribeForConnection(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &wsConnectFixtureConnection{
		Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
		responseRaw: []byte(`{"event":"subscribed"}`),
	}
	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		Pairs:            currency.Pairs{currency.NewBTCUSD()},
		QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`,
	}

	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{sub}), "subscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "subscribeForConnection should send one request")
	assert.NotNil(t, ex.Websocket.GetSubscription(sub.Key), "subscribeForConnection should store a temporary subscription")
}

func TestUnsubscribeForConnection(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &wsConnectFixtureConnection{
		Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
		responseRaw: []byte(`{"event":"unsubscribed"}`),
	}
	sub := &subscription.Subscription{Key: websocketChannelKey{conn, 42}, QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")

	require.NoError(t, ex.unsubscribeForConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "unsubscribeForConnection should send one request")
	assert.Nil(t, ex.Websocket.GetSubscription(websocketChannelKey{conn, 42}), "unsubscribeForConnection should remove the subscription")
}

func TestWsSendAuthConn(t *testing.T) {
	t.Parallel()
	errSend := errors.New("auth send failure")
	for _, tc := range []struct {
		name    string
		sendErr error
	}{
		{name: "pending acknowledgement"},
		{name: "send failure", sendErr: errSend},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.API.AuthenticatedWebsocketSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			conn := &wsConnectFixtureConnection{sendErr: tc.sendErr}
			err := ex.wsSendAuthConn(t.Context(), conn)
			if tc.sendErr != nil {
				assert.ErrorIs(t, err, tc.sendErr, "authentication should retain the send failure")
			} else {
				require.NoError(t, err, "wsSendAuthConn must not error")
			}
			assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "wsSendAuthConn should send one request")
			assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "private requests should wait for a successful acknowledgement even when sending succeeds")
		})
	}
}

func TestResubOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("full capacity recovers twice", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		ex.Websocket.MaxSubscriptionsPerConnection = 1
		conn := &wsConnectFixtureConnection{Connection: testexch.GetMockConn(t, ex, publicBitfinexWebsocketEndpoint)}
		require.NoError(t, ex.Websocket.TrackTestConnection(asset.Spot, conn), "connection must be tracked")
		sub := &subscription.Subscription{Key: websocketChannelKey{conn, 1}, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, QualifiedChannel: `{"channel":"book","symbol":"tBTCUSD"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "initial subscription must register")
		require.NoError(t, conn.Subscriptions().Add(sub), "initial subscription must occupy its connection slot")
		var acknowledged atomic.Int32
		conn.respond = func(ctx context.Context, signature, payload any) ([]byte, error) {
			req, ok := payload.(map[string]any)
			if !ok {
				return nil, common.ErrTypeAssertFailure
			}
			if req["event"] == "unsubscribe" {
				return []byte(`{"event":"unsubscribed"}`), nil
			}
			responses, err := conn.MatchReturnResponses(ctx, signature, 1)
			if err != nil {
				return nil, err
			}
			id := acknowledged.Load() + 2
			raw, err := json.Marshal(map[string]any{"event": "subscribed", "channel": "book", "chanId": id, "subId": req["subId"]})
			if err != nil {
				return nil, err
			}
			if err := ex.handleWSSubscribed(conn, raw); err != nil {
				return nil, err
			}
			select {
			case matched := <-responses:
				acknowledged.Add(1)
				return matched.Responses[0], matched.Err
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return nil, context.DeadlineExceeded
			}
		}
		for i := int32(1); i <= 2; i++ {
			require.NoError(t, ex.resubOrderbook(t.Context(), conn, sub), "recovery must start")
			require.Eventually(t, func() bool { return acknowledged.Load() == i && sub.State() == subscription.SubscribedState }, time.Second, time.Millisecond, "recovery must complete")
			assert.Samef(t, sub, ex.Websocket.GetSubscription(websocketChannelKey{conn, int(i + 1)}), "recovery %d should track the new channel key", i)
			assert.Equal(t, 1, conn.Subscriptions().Len(), "recovery should retain exactly one slot")
			assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "manager should retain exactly one subscription")
		}
	})

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.ErrorIs(t, ex.resubOrderbook(t.Context(), testexch.GetMockConn(t, ex, ""), nil), common.ErrNilPointer, "resubOrderbook should reject a nil subscription")
	assert.ErrorIs(t, ex.resubOrderbook(t.Context(), testexch.GetMockConn(t, ex, ""), &subscription.Subscription{}), subscription.ErrNotSinglePair, "resubOrderbook should require exactly one pair")
}

func TestWsDisconnected(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must succeed")
	a := testexch.GetMockConn(t, ex, "ws://a")
	b := testexch.GetMockConn(t, ex, "ws://b")
	ka, kb := websocketChannelKey{a, 1}, websocketChannelKey{b, 1}
	cMtx.Lock()
	checksumStore[ka] = new(checksum)
	checksumStore[kb] = new(checksum)
	cMtx.Unlock()
	t.Cleanup(func() { ex.wsDisconnected(a); ex.wsDisconnected(b) })
	ex.wsDisconnected(a)
	cMtx.Lock()
	_, hasA := checksumStore[ka]
	_, hasB := checksumStore[kb]
	cMtx.Unlock()
	assert.False(t, hasA, "disconnected checksum should be removed")
	assert.True(t, hasB, "other connection checksum should survive")
}

func TestHandleWSSubscribedConnectionIsolation(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must succeed")
	a, err := ex.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "connection must be created")
	b, err := ex.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "connection must be created")
	for _, tc := range []struct {
		conn websocket.Connection
		id   string
	}{{a, "a"}, {b, "b"}} {
		require.NoError(t, ex.Websocket.TrackTestConnection(asset.Spot, tc.conn), "connection must be tracked")
		sub := &subscription.Subscription{Key: tc.id, Channel: subscription.TickerChannel}
		require.NoError(t, ex.Websocket.AddSubscriptions(tc.conn, sub), "AddSubscriptions must succeed")
		_, err := tc.conn.MatchReturnResponses(t.Context(), "subscribe:"+tc.id, 1)
		require.NoError(t, err, "matcher must register")
		require.NoError(t, ex.handleWSSubscribed(tc.conn, []byte(`{"event":"subscribed","channel":"ticker","chanId":1,"subId":"`+tc.id+`"}`)), "acknowledgement must succeed")
		assert.Same(t, sub, ex.Websocket.GetSubscription(websocketChannelKey{tc.conn, 1}), "channel ID should belong to its connection")
	}
	assert.NotSame(t, ex.Websocket.GetSubscription(websocketChannelKey{a, 1}), ex.Websocket.GetSubscription(websocketChannelKey{b, 1}), "identical channel IDs should remain distinct")
}

func TestCanUseWebsocketOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                       string
		privateDialFails, disableAuth, pendingAuth bool
		authResponse                               string
		want                                       bool
	}{
		{name: "private route", want: true},
		{name: "private dial failure", privateDialFails: true},
		{name: "pending authentication", pendingAuth: true},
		{name: "authentication disabled", disableAuth: true},
		{name: "authentication rejected", authResponse: `{"event":"auth","status":"ERROR","code":10100}`},
		{name: "authentication malformed", authResponse: `{"event":"auth"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var dials atomic.Int32
			upgrade := mockws.CurryWsMockUpgrader(t, func(_ testing.TB, raw []byte, c *gws.Conn) error {
				var req map[string]any
				if err := json.Unmarshal(raw, &req); err != nil {
					return err
				}
				switch req["event"] {
				case "auth":
					if tc.pendingAuth {
						return nil
					}
					return c.WriteMessage(gws.TextMessage, []byte(`{"event":"auth","status":"OK"}`))
				case "conf":
					return c.WriteMessage(gws.TextMessage, []byte(`{"event":"conf","status":"OK"}`))
				case "subscribe":
					return c.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"event":"subscribed","channel":"ticker","chanId":1,"subId":%q}`, req["subId"]))
				}
				return nil
			})
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if dials.Add(1) > 1 && tc.privateDialFails {
					http.Error(w, "private endpoint unavailable", http.StatusServiceUnavailable)
					return
				}
				upgrade(w, r)
			}))
			server.Start() // The connector uses its own dialler and requires a network socket.
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			ex.Websocket.SetAuthenticatedSupport(true)
			ex.API.AuthenticatedWebsocketSupport = true
			ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSD()}}}
			connectErr := ex.Websocket.Connect(t.Context())
			t.Cleanup(func() {
				if err := ex.Websocket.Disable(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should only report an already disabled monitor")
				}
				if err := ex.Websocket.Shutdown(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only report already closed mock connections")
				}
			})
			if tc.privateDialFails {
				require.ErrorIs(t, connectErr, websocket.ErrNotConnected, "Connect must report the private dial failure")
				require.True(t, ex.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "authentication must stay enabled when only the private dial fails")
			} else {
				require.NoError(t, connectErr, "Connect must establish both routes")
				if !tc.pendingAuth {
					require.Eventually(t, ex.canUseWebsocketOrders, time.Second, time.Millisecond, "private route must wait for a successful auth acknowledgement")
				} else {
					_, err := ex.Websocket.GetConnection("auth")
					require.NoError(t, err, "pending authentication must have a connected private socket")
					assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a connected socket should remain unauthenticated until acknowledgement")
				}
			}
			if tc.disableAuth {
				ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
			}
			if tc.authResponse != "" {
				conn, err := ex.Websocket.GetConnection("auth")
				require.NoError(t, err, "private connection must remain dialled to reproduce rejected authentication")
				require.Error(t, ex.handleWSEvent(t.Context(), conn, []byte(tc.authResponse)), "invalid authentication must report an error")
				assert.False(t, ex.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "invalid authentication should disable private requests")
			}
			assert.Equal(t, tc.want, ex.canUseWebsocketOrders(), "orders should require an authenticated connected private route")
			if tc.want {
				return
			}
			var cancellations atomic.Int32
			restServer := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cancellations.Add(1)
				assert.True(t, strings.HasSuffix(r.URL.Path, "order/cancel/all"), "cancellation should use the REST route")
				_, err := w.Write([]byte(`{"result":"success"}`))
				assert.NoError(t, err, "cancellation response should write")
			}))
			ex.API.AuthenticatedSupport = true
			ex.SkipAuthCheck = true
			require.NoError(t, ex.SetHTTPClient(restServer.Client()), "mock HTTP client must configure")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), restServer.URL+"/"), "REST endpoint must configure")
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{})
			require.NoError(t, err, "cancellation must fall back to REST")
			assert.Equal(t, int32(1), cancellations.Load(), "one cancellation should reach REST rather than an unavailable or unauthenticated socket")
		})
	}
}

package binanceus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type websocketTestConnection struct {
	websocket.Connection
	dialErr error
	sent    []WebsocketPayload
	sendErr error
	ping    websocket.PingHandler
}

func (c *websocketTestConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return c.dialErr
}

func (c *websocketTestConnection) SetupPingHandler(_ request.EndpointLimit, ping websocket.PingHandler) {
	c.ping = ping
}

func (c *websocketTestConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	p, ok := payload.(WebsocketPayload)
	if !ok {
		return errors.New("unexpected websocket payload type")
	}
	p.Params = append([]any(nil), p.Params...)
	c.sent = append(c.sent, p)
	return c.sendErr
}

type websocketRoundTripper func(*http.Request) (*http.Response, error)

func (f websocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newWebsocketTestConnection(t *testing.T, ex *Exchange) *websocketTestConnection {
	t.Helper()
	conn, err := ex.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "managed connection setup must exist")
	return &websocketTestConnection{Connection: conn}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	for _, disableExchange := range []bool{false, true} {
		name := "websocket disabled"
		if disableExchange {
			name = "exchange disabled"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
			if disableExchange {
				ex.SetEnabled(false)
			} else {
				require.NoError(t, ex.Websocket.Disable(), "websocket must disable")
			}
			assert.ErrorIs(t, ex.WsConnect(), websocket.ErrWebsocketNotEnabled, "exported wrapper should preserve the disabled error")
			assert.False(t, ex.wsSessionActive, "disabled connection should not start a session")
		})
	}

	t.Run("public lifecycle", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		conn := newWebsocketTestConnection(t, ex)
		require.NoError(t, ex.wsConnect(t.Context(), conn), "connector must succeed")
		assert.True(t, conn.ping.UseGorillaHandler, "server ping handler should be installed")
		require.NotNil(t, ex.obm, "orderbook workers must have state before reader starts")
		competing := newWebsocketTestConnection(t, ex)
		assert.ErrorIs(t, ex.wsConnect(t.Context(), competing), errWebsocketSessionActive, "second combined stream should be rejected without resetting workers")
		assert.True(t, ex.wsSessionActive, "rejected stream should preserve the active session")
		shutdown := ex.Websocket.ShutdownC
		ex.Websocket.ShutdownC = make(chan struct{})
		close(shutdown)
		waitWebsocketWorkers(t, ex)
		assert.False(t, ex.wsSessionActive, "disconnect should remove the session")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		require.NoError(t, ex.wsConnect(ctx, competing), "replacement stream must start after prior workers stop")
		cancel()
		waitWebsocketWorkers(t, ex)
	})
	t.Run("dial failure", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
		conn := newWebsocketTestConnection(t, ex)
		conn.dialErr = errors.New("dial failed")
		assert.ErrorIs(t, ex.wsConnect(t.Context(), conn), conn.dialErr, "connector should preserve the dial error")
		assert.False(t, ex.wsSessionActive, "failed dial should not start workers")
	})
	t.Run("listen keys remain with their exchange", func(t *testing.T) {
		t.Parallel()
		for _, key := range []string{"first-key", "second-key", "failed-key"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
				ex.API.AuthenticatedWebsocketSupport = true
				ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
				ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
				require.NoError(t, ex.Requester.DisableRateLimiter(), "test rate limiter must disable")
				var mu sync.Mutex
				var requests []string
				require.NoError(t, ex.SetHTTPClient(&http.Client{Transport: websocketRoundTripper(func(r *http.Request) (*http.Response, error) {
					mu.Lock()
					requests = append(requests, r.Method+":"+r.URL.Query().Get("listenKey"))
					mu.Unlock()
					body := `{}`
					if r.Method == http.MethodPost {
						body = `{"listenKey":"` + key + `"}`
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})}), "test client must install")
				conn := newWebsocketTestConnection(t, ex)
				if key == "failed-key" {
					conn.dialErr = errors.New("dial failed")
					assert.ErrorIs(t, ex.wsConnect(t.Context(), conn), conn.dialErr, "authenticated dial should preserve its error")
					assert.False(t, ex.wsSessionActive, "failed authenticated dial should not start workers")
					assert.Empty(t, ex.listenKey, "failed dial cleanup should clear the closed key")
					require.NoError(t, ex.MaintainWsAuthStreamKey(t.Context()), "maintenance must acquire a fresh key after failed dial cleanup")
					mu.Lock()
					assert.Equal(t, []string{"POST:", "DELETE:" + key, "POST:"}, requests, "failed dial should release its key and later maintenance should reacquire it")
					mu.Unlock()
					return
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				require.NoError(t, ex.wsConnect(ctx, conn), "authenticated connector must succeed")
				assert.Equal(t, binanceusDefaultWebsocketURL+"?streams="+key, conn.GetURL(), "only this connection should receive its listen key")
				require.NoError(t, ex.MaintainWsAuthStreamKey(t.Context()), "renewal must succeed")
				cancel()
				waitWebsocketWorkers(t, ex)
				assert.Empty(t, ex.listenKey, "disconnect should clear the closed key")
				mu.Lock()
				assert.Equal(t, []string{"POST:", "PUT:" + key, "DELETE:" + key}, requests, "creation, renewal and cleanup should use this exchange's key")
				mu.Unlock()
			})
		}
	})
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	first := newWebsocketTestConnection(t, ex)
	second := newWebsocketTestConnection(t, ex)
	require.NoError(t, ex.Websocket.TrackTestConnection(asset.Spot, first), "first route must register")
	require.NoError(t, ex.Websocket.TrackTestConnection(asset.Spot, second), "second route must register")
	sub := &subscription.Subscription{Channel: "btcusdt@ticker", Asset: asset.Spot}
	require.NoError(t, ex.Subscribe(t.Context(), second, subscription.List{sub}), "subscription must succeed")
	assert.Empty(t, first.sent, "other socket should not send the request")
	require.Len(t, second.sent, 1, "target socket must send one request")
	assert.Equal(t, "SUBSCRIBE", second.sent[0].Method, "wire operation should remain unchanged")
	assert.Same(t, sub, ex.Websocket.GetSubscription(sub), "successful subscription should be tracked")
	require.NoError(t, ex.Unsubscribe(t.Context(), second, subscription.List{sub}), "unsubscription must succeed")
	assert.Equal(t, "UNSUBSCRIBE", second.sent[1].Method, "unsubscription should use the target socket")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "unsubscribed channel should be removed")
}

func waitWebsocketWorkers(t *testing.T, ex *Exchange) {
	t.Helper()
	done := make(chan struct{})
	go func() { ex.Websocket.Wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("disconnect must join background workers")
	}
}

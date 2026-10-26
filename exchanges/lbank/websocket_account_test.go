package lbank

import (
	"cmp"
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const (
	testSubscribeKey = "4e9958623e6006bd7b13ff9f36c03b36132f0f8da37f70b14ff2c4eab1fe0c97"
	// replacementSubscribeKey is the key a test server creates in place of an expired one
	replacementSubscribeKey = "9301ef1ca6cafbef2df4a1430dc8b53879ea68c595a142eac311d8d590fbd60a"
)

// subscribeKeyServer counts a test server's subscribe key requests by path, answering each as LBank does. It creates
// key, or testSubscribeKey when key is empty, after getDelay unless failGet is set, refreshes a key after refreshDelay,
// and refuses to refresh the expired key
type subscribeKeyServer struct {
	mu           sync.Mutex
	requests     map[string][]string
	failGet      bool
	getDelay     time.Duration
	refreshDelay time.Duration
	key          string
	expired      string
}

func (s *subscribeKeyServer) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm(), "the request body should parse as a form") {
			return
		}
		key := r.PostForm.Get("subscribeKey")
		s.mu.Lock()
		if s.requests == nil {
			s.requests = make(map[string][]string)
		}
		s.requests[r.URL.Path] = append(s.requests[r.URL.Path], key)
		failGet, getDelay, refreshDelay, created, expired := s.failGet, s.getDelay, s.refreshDelay, cmp.Or(s.key, testSubscribeKey), s.expired
		s.mu.Unlock()
		switch r.URL.Path {
		case "/v2/subscribe/get_key.do":
			time.Sleep(getDelay)
		case "/v2/subscribe/refresh_key.do":
			time.Sleep(refreshDelay)
		}
		switch {
		case r.URL.Path == "/v2/subscribe/get_key.do" && failGet:
			writeResponse(t, w, `{"result":"false","msg":"API Key permission denied","error_code":10022,"ts":1}`)
		case r.URL.Path == "/v2/subscribe/get_key.do":
			writeResponse(t, w, `{"result":true,"data":"`+created+`","error_code":0,"ts":1}`)
		case r.URL.Path == "/v2/subscribe/refresh_key.do" && expired != "" && key == expired:
			writeResponse(t, w, `{"result":"false","msg":"Not existing or expired subscribeKey","error_code":10003,"ts":1}`)
		default:
			writeResponse(t, w, `{"result":"true"}`)
		}
	}
}

func (s *subscribeKeyServer) sent(path string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests[path]...)
}

// waitForRenewers waits for an exchange's subscribe key renewers to stop, failing the test if they don't
func waitForRenewers(t *testing.T, ex *Exchange) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		ex.Websocket.Wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the subscribe key renewer must stop")
	}
}

// orderUpdateMessage decodes an order update frame
func orderUpdateMessage(t *testing.T, frame string) *websocketMessage {
	t.Helper()
	msg := new(websocketMessage)
	require.NoError(t, json.Unmarshal([]byte(frame), msg), "Unmarshal must decode the order update")
	return msg
}

// accountSubscriptions returns new subscriptions to the account channels, expanded as generateAccountSubscriptions
// expands them
func accountSubscriptions() subscription.List {
	return subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsOrderUpdateChannel},
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true, QualifiedChannel: wsAssetUpdateChannel},
	}
}

// subscribeAccountChannels subscribes a connection holding testSubscribeKey to new subscriptions to the account
// channels, as wsSubscribe does, returning them; the requests it sends are cleared from the connection's messages
func subscribeAccountChannels(t *testing.T, ex *Exchange, conn *recordingConnection) subscription.List {
	t.Helper()
	ex.subscribeKeys.set(conn, testSubscribeKey)
	subs := accountSubscriptions()
	require.NoError(t, ex.wsSubscribe(t.Context(), conn, subs), "wsSubscribe must not error")
	conn.clearMessages()
	return subs
}

// holdAccountSubscriptions stores new subscriptions to the account channels as subscribed on a connection, as
// wsSubscribe records them, returning them
func holdAccountSubscriptions(t *testing.T, ex *Exchange, conn websocket.Connection) subscription.List {
	t.Helper()
	subs := accountSubscriptions()
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
	return subs
}

// accountSubscribeRequests returns the requests subscribing to the account channels with a subscribe key
func accountSubscribeRequests(key string) []any {
	return []any{
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: wsOrderUpdateChannel, Pair: "all", SubscribeKey: key},
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: wsAssetUpdateChannel, SubscribeKey: key},
	}
}

func TestCreateSubscribeKey(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.CreateSubscribeKey(t.Context())
	require.NoError(t, err, "CreateSubscribeKey must not error")
	if mockTests {
		assert.Equal(t, testSubscribeKey, result, "CreateSubscribeKey should decode the key")
		return
	}
	assert.NotEmpty(t, result, "CreateSubscribeKey should return a key")
	assert.NoError(t, e.CloseSubscribeKey(t.Context(), result), "CloseSubscribeKey should close the key")
}

func TestRefreshSubscribeKey(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.RefreshSubscribeKey(t.Context(), ""), errSubscribeKeyRequired, "RefreshSubscribeKey must require a key")
	if !mockTests {
		t.Skip("RefreshSubscribeKey needs a key the account created")
	}
	assert.NoError(t, e.RefreshSubscribeKey(t.Context(), testSubscribeKey), "RefreshSubscribeKey should not error")
}

func TestCloseSubscribeKey(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CloseSubscribeKey(t.Context(), ""), errSubscribeKeyRequired, "CloseSubscribeKey must require a key")
	if !mockTests {
		t.Skip("CloseSubscribeKey needs a key the account created")
	}
	assert.NoError(t, e.CloseSubscribeKey(t.Context(), testSubscribeKey), "CloseSubscribeKey should not error")
}

func TestSubscribeKeyStore(t *testing.T) {
	t.Parallel()
	var s subscribeKeyStore
	a, b := &recordingConnection{}, &recordingConnection{}
	assert.Empty(t, s.get(a), "get should return no key for a connection without one")
	s.set(a, "key-a")
	s.set(b, "key-b")
	assert.Equal(t, "key-a", s.get(a), "get should return the connection's own key")
	s.remove(a, "stale")
	assert.Equal(t, "key-a", s.get(a), "remove should keep a key the connection was given since")
	s.markSent(a, wsOrderUpdateChannel, "key-a")
	s.markSent(b, wsOrderUpdateChannel, "key-b")
	key, ok := s.sentWith(a, wsOrderUpdateChannel)
	assert.True(t, ok, "sentWith should report a channel subscribed on the connection")
	assert.Equal(t, "key-a", key, "sentWith should return the key the channel was subscribed with")
	_, ok = s.sentWith(a, wsAssetUpdateChannel)
	assert.False(t, ok, "sentWith should report a channel never subscribed on the connection")
	s.remove(a, "key-a")
	assert.Empty(t, s.get(a), "remove should forget the connection's key")
	_, ok = s.sentWith(a, wsOrderUpdateChannel)
	assert.False(t, ok, "remove should forget the channels subscribed with the connection's key")
	assert.Equal(t, "key-b", s.get(b), "remove should leave other connections' keys")
	s.forgetSent(b, wsOrderUpdateChannel)
	_, ok = s.sentWith(b, wsOrderUpdateChannel)
	assert.False(t, ok, "forgetSent should forget a channel unsubscribed")
	s.forgetSent(a, wsOrderUpdateChannel)
	s.markSent(a, wsOrderUpdateChannel, "key-a")
	_, ok = s.sentWith(a, wsOrderUpdateChannel)
	assert.False(t, ok, "markSent should record nothing for a connection whose key is gone")
	s.markSent(b, wsAssetUpdateChannel, "stale")
	_, ok = s.sentWith(b, wsAssetUpdateChannel)
	assert.False(t, ok, "markSent should record nothing sent with a key the connection no longer holds")
	_, ok = s.sent[a]
	assert.False(t, ok, "markSent should leave no record of a connection whose key is gone")
}

func TestConnectionGone(t *testing.T) {
	t.Parallel()
	assert.False(t, connectionGone(&recordingConnection{connected: true}), "connectionGone should report a connected connection as live")
	assert.True(t, connectionGone(&recordingConnection{}), "connectionGone should report a shut down connection as gone")
	assert.False(t, connectionGone(struct{ websocket.Connection }{}), "connectionGone should take a connection that can't tell as live")
}

func TestSubscribeKeyTimingsNextRenewal(t *testing.T) {
	t.Parallel()
	assert.Equal(t, subscribeKeyRefreshInterval, defaultSubscribeKeyTimings.nextRenewal(nil), "nextRenewal should wait the refresh interval after a renewal")
	assert.Equal(t, subscribeKeyRetryInterval, defaultSubscribeKeyTimings.nextRenewal(errRequestFailed), "nextRenewal should retry sooner after a failed renewal")
	short := subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: time.Minute, lifetime: time.Hour}
	assert.Equal(t, 10*time.Millisecond, short.nextRenewal(errRequestFailed), "nextRenewal should never wait longer than the refresh interval")
}

func TestStartSubscribeKey(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	ctx, cancel := context.WithCancel(t.Context())
	timings := subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: 10 * time.Millisecond, lifetime: time.Hour}
	require.NoError(t, ex.startSubscribeKey(ctx, conn, timings), "startSubscribeKey must not error")
	assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "startSubscribeKey should give the connection its key")
	assert.Equal(t, []string{""}, server.sent("/v2/subscribe/get_key.do"), "startSubscribeKey should create one key")
	cancel()
	refreshed := len(server.sent("/v2/subscribe/refresh_key.do"))
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/refresh_key.do")) >= refreshed+2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must go on refreshing the key once the request that created it is done")
	assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "the renewer should keep the connection's key once the request that created it is done")
	assert.Empty(t, relayedPayloads(ex), "the renewer's refreshes should succeed once the request that created the key is done")
	close(ex.Websocket.ShutdownC)
	waitForRenewers(t, ex)
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the key when the websocket shuts down")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")

	refused := &subscribeKeyServer{failGet: true}
	ex = newTestServerExchange(t, refused.handler(t))
	conn = &recordingConnection{connected: true}
	require.ErrorIs(t, ex.startSubscribeKey(t.Context(), conn, defaultSubscribeKeyTimings), request.ErrAuthRequestFailed, "startSubscribeKey must report a key it couldn't create")
	assert.Empty(t, ex.subscribeKeys.get(conn), "startSubscribeKey should give the connection no key it couldn't create")
	waitForRenewers(t, ex)

	server = &subscribeKeyServer{}
	ex = newTestServerExchange(t, server.handler(t))
	gone := &recordingConnection{}
	require.ErrorIs(t, ex.startSubscribeKey(t.Context(), gone, defaultSubscribeKeyTimings), websocket.ErrNotConnected, "startSubscribeKey must report a connection shut down while its key was created")
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "startSubscribeKey should close the key of a connection shut down meanwhile")
	assert.Empty(t, ex.subscribeKeys.get(gone), "startSubscribeKey should give a connection shut down meanwhile no key")
	waitForRenewers(t, ex)
}

func TestWebsocketSubscribeAccountKey(t *testing.T) {
	t.Parallel()
	t.Run("created with the first account subscription", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		subs := accountSubscriptions()
		require.NoError(t, ex.wsSubscribe(t.Context(), conn, subs), "wsSubscribe must not error")
		assert.Equal(t, accountSubscribeRequests(testSubscribeKey), conn.messages(), "wsSubscribe should subscribe the account channels with the key it created")
		assert.Equal(t, []string{""}, server.sent("/v2/subscribe/get_key.do"), "wsSubscribe should create one key for the connection")
		for _, s := range subs {
			assert.Equalf(t, subscription.SubscribedState, s.State(), "wsSubscribe should record subscription %s", s)
		}
		close(ex.Websocket.ShutdownC)
		waitForRenewers(t, ex)
		assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the key's renewer should close it when the websocket shuts down")
	})
	t.Run("created once for concurrent subscriptions", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{getDelay: 200 * time.Millisecond}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		var wg sync.WaitGroup
		for _, s := range accountSubscriptions() {
			wg.Go(func() {
				assert.NoErrorf(t, ex.wsSubscribe(t.Context(), conn, subscription.List{s}), "wsSubscribe should not error for %s", s)
			})
		}
		wg.Wait()
		assert.Equal(t, []string{""}, server.sent("/v2/subscribe/get_key.do"), "wsSubscribe should create one key for a connection subscribed to concurrently")
		assert.ElementsMatch(t, accountSubscribeRequests(testSubscribeKey), conn.messages(), "wsSubscribe should subscribe each account channel with the connection's one key")
		close(ex.Websocket.ShutdownC)
		waitForRenewers(t, ex)
		assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "one renewer should close the connection's key")
	})
	t.Run("reused while the connection holds one", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		ex.subscribeKeys.set(conn, replacementSubscribeKey)
		require.NoError(t, ex.wsSubscribe(t.Context(), conn, accountSubscriptions()), "wsSubscribe must not error")
		assert.Equal(t, accountSubscribeRequests(replacementSubscribeKey), conn.messages(), "wsSubscribe should subscribe with the connection's key")
		assert.Empty(t, server.sent("/v2/subscribe/get_key.do"), "wsSubscribe should create no key for a connection holding one")
		waitForRenewers(t, ex)
	})
	t.Run("not created for market data", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		tick := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}, QualifiedChannel: "tick:btc_usdt"}
		require.NoError(t, ex.wsSubscribe(t.Context(), conn, subscription.List{tick}), "wsSubscribe must not error")
		assert.Equal(t, []any{&websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"}}, conn.messages(), "wsSubscribe should subscribe the ticker")
		assert.Empty(t, server.sent("/v2/subscribe/get_key.do"), "wsSubscribe should create no key for market data")
		assert.Empty(t, ex.subscribeKeys.get(conn), "wsSubscribe should give a market data connection no key")
	})
	t.Run("refused", func(t *testing.T) {
		t.Parallel()
		refused := &subscribeKeyServer{failGet: true}
		ex := newTestServerExchange(t, refused.handler(t))
		conn := &recordingConnection{connected: true}
		tick := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}, QualifiedChannel: "tick:btc_usdt"}
		subs := append(accountSubscriptions(), tick)
		err := ex.wsSubscribe(t.Context(), conn, subs)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "wsSubscribe must report a key it couldn't create")
		assert.ErrorIs(t, err, errSubscribeKeyRequired, "wsSubscribe should fail the account channels without a key")
		assert.Equal(t, []any{&websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"}}, conn.messages(), "wsSubscribe should still subscribe the market data")
		assert.Equal(t, subscription.SubscribedState, tick.State(), "wsSubscribe should record the market data subscription")
		for _, s := range subs[:2] {
			assert.Equalf(t, subscription.InactiveState, s.State(), "wsSubscribe should not record subscription %s", s)
		}
		assert.Empty(t, ex.subscribeKeys.get(conn), "wsSubscribe should give the connection no key it couldn't create")
		waitForRenewers(t, ex)
	})
}

func TestKeepSubscribeKeyAlive(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	timings := subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: 10 * time.Millisecond, lifetime: 200 * time.Millisecond}
	start := time.Now()
	shutdown := make(chan struct{})
	ex.subscribeKeys.set(conn, testSubscribeKey)
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, timings)
	// The connection outlives the key's lifetime, which each refresh extends
	require.Eventually(t, func() bool {
		return time.Since(start) > 5*timings.lifetime/2 && len(server.sent("/v2/subscribe/refresh_key.do")) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must refresh the key while the connection lives")

	conn.setConnected(false)
	waitForRenewers(t, ex)
	for _, key := range server.sent("/v2/subscribe/refresh_key.do") {
		assert.Equal(t, testSubscribeKey, key, "the renewer should refresh the connection's own key")
	}
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the key once the connection has gone")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")
	assert.Empty(t, server.sent("/v2/subscribe/get_key.do"), "the renewer should not replace a key its refreshes keep alive")
}

func TestKeepSubscribeKeyAliveShutdown(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	shutdown := make(chan struct{})
	ex.subscribeKeys.set(conn, testSubscribeKey)
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, defaultSubscribeKeyTimings)
	close(shutdown)
	waitForRenewers(t, ex)
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the key when the websocket shuts down")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")
}

func TestKeepSubscribeKeyAliveRefreshFailure(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{expired: testSubscribeKey}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	shutdown := make(chan struct{})
	ex.subscribeKeys.set(conn, testSubscribeKey)
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: 10 * time.Millisecond, lifetime: time.Hour})
	var relayed []error
	require.Eventually(t, func() bool {
		for len(ex.Websocket.DataHandler.C) > 0 {
			if err, ok := (<-ex.Websocket.DataHandler.C).Data.(error); ok {
				relayed = append(relayed, err)
			}
		}
		return len(relayed) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must keep retrying a refresh it couldn't make, relaying each failure")
	close(shutdown)
	waitForRenewers(t, ex)
	assert.ErrorIs(t, relayed[0], errRequestFailed, "the renewer should relay LBank's refusal")
	assert.Empty(t, server.sent("/v2/subscribe/get_key.do"), "the renewer should not replace a key before it expires")
}

func TestKeepSubscribeKeyAliveRelayFailure(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{expired: testSubscribeKey}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	// A full data handler refuses the failures the renewer relays, which it logs instead
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must fill the data handler")
	shutdown := make(chan struct{})
	ex.subscribeKeys.set(conn, testSubscribeKey)
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: 10 * time.Millisecond, lifetime: time.Hour})
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/refresh_key.do")) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must keep retrying a refresh it couldn't relay the failure of")
	close(shutdown)
	waitForRenewers(t, ex)
}

func TestKeepSubscribeKeyAliveReplacesExpiredKey(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{key: replacementSubscribeKey, expired: testSubscribeKey}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	subscribeAccountChannels(t, ex, conn)
	shutdown := make(chan struct{})
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, subscribeKeyTimings{refresh: 20 * time.Millisecond, retry: 20 * time.Millisecond, lifetime: 200 * time.Millisecond})
	exp := accountSubscribeRequests(replacementSubscribeKey)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Subset(c, conn.messages(), exp, "the renewer should subscribe the account channels again under the new key")
		assert.Contains(c, server.sent("/v2/subscribe/refresh_key.do"), replacementSubscribeKey, "the renewer should go on to refresh the new key")
	}, 5*time.Second, 10*time.Millisecond, "the renewer must replace a key it couldn't refresh for as long as the key lasts")
	assert.Subset(t, exp, conn.messages(), "the renewer should send only the account subscriptions under the new key")
	assert.Equal(t, replacementSubscribeKey, ex.subscribeKeys.get(conn), "the renewer should give the connection the new key")
	close(shutdown)
	waitForRenewers(t, ex)
	assert.Equal(t, []string{replacementSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the new key when the websocket shuts down")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")
}

func TestKeepSubscribeKeyAliveReplacementFailure(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{failGet: true, expired: testSubscribeKey}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	subscribeAccountChannels(t, ex, conn)
	shutdown := make(chan struct{})
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, shutdown, subscribeKeyTimings{refresh: 10 * time.Millisecond, retry: 10 * time.Millisecond, lifetime: 30 * time.Millisecond})
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/get_key.do")) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must keep retrying a replacement it couldn't create")
	assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "the renewer should keep the connection's key until it creates another")
	assert.Empty(t, conn.messages(), "the renewer should subscribe nothing without a new key")
	close(shutdown)
	waitForRenewers(t, ex)
}

func TestRenewSubscribeKey(t *testing.T) {
	t.Parallel()
	t.Run("refresh", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		subscribeAccountChannels(t, ex, conn)
		renewed := time.Now().Add(-time.Minute)
		r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: renewed}
		require.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey must refresh a live key")
		assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/refresh_key.do"), "renewSubscribeKey should refresh the connection's key")
		assert.True(t, r.renewed.After(renewed), "renewSubscribeKey should extend a refreshed key's lifetime")
		assert.Empty(t, server.sent("/v2/subscribe/get_key.do"), "renewSubscribeKey should not replace a live key")
		assert.Empty(t, conn.messages(), "renewSubscribeKey should subscribe nothing under a refreshed key")

		server.mu.Lock()
		server.expired = testSubscribeKey
		server.mu.Unlock()
		r.renewed = renewed
		require.ErrorIs(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), errRequestFailed, "renewSubscribeKey must report a refresh LBank refuses")
		assert.Equal(t, renewed, r.renewed, "renewSubscribeKey should not extend a key it couldn't refresh")
		assert.Empty(t, conn.messages(), "renewSubscribeKey should subscribe nothing under a key it couldn't refresh")
	})
	t.Run("replacement refused", func(t *testing.T) {
		t.Parallel()
		refused := &subscribeKeyServer{failGet: true}
		ex := newTestServerExchange(t, refused.handler(t))
		conn := &recordingConnection{connected: true}
		subscribeAccountChannels(t, ex, conn)
		r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: time.Now().Add(-2 * time.Hour)}
		require.ErrorIs(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), request.ErrAuthRequestFailed, "renewSubscribeKey must report a replacement it couldn't create")
		assert.Equal(t, testSubscribeKey, r.key, "renewSubscribeKey should keep the expired key until it creates another")
		assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "renewSubscribeKey should keep the connection's expired key until it creates another")
		assert.Empty(t, conn.messages(), "renewSubscribeKey should subscribe nothing without a new key")
	})
	t.Run("replacement", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{key: replacementSubscribeKey}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		subscribeAccountChannels(t, ex, conn)
		conn.setSendFailures(nil, wsAssetUpdateChannel)
		r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: time.Now().Add(-2 * time.Hour)}
		require.ErrorIs(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), errSendFailed, "renewSubscribeKey must report a resubscription it couldn't send")
		assert.Equal(t, replacementSubscribeKey, r.key, "renewSubscribeKey should keep the new key a resubscription failed under")
		assert.Equal(t, replacementSubscribeKey, ex.subscribeKeys.get(conn), "renewSubscribeKey should give the connection the new key")
		assert.Equal(t, accountSubscribeRequests(replacementSubscribeKey)[:1], conn.messages(), "renewSubscribeKey should subscribe the orders channel with the new key")

		conn.setSendFailures(nil, "")
		require.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey must retry a channel it couldn't subscribe")
		assert.Equal(t, accountSubscribeRequests(replacementSubscribeKey), conn.messages(), "renewSubscribeKey should subscribe again only the channel it couldn't before")
		assert.Equal(t, []string{""}, server.sent("/v2/subscribe/get_key.do"), "renewSubscribeKey should keep the new key while subscribing with it")
		require.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey must refresh the new key")
		assert.Len(t, conn.messages(), 2, "renewSubscribeKey should subscribe nothing once every channel carries the new key")
	})
	t.Run("channels changed meanwhile", func(t *testing.T) {
		t.Parallel()
		server := &subscribeKeyServer{key: replacementSubscribeKey}
		ex := newTestServerExchange(t, server.handler(t))
		conn := &recordingConnection{connected: true}
		subs := subscribeAccountChannels(t, ex, conn)
		conn.setSendFailures(errSendFailed, "")
		r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: time.Now().Add(-2 * time.Hour)}
		require.ErrorIs(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), errSendFailed, "renewSubscribeKey must report the channels it couldn't subscribe")
		conn.setSendFailures(nil, "")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, accountSubscriptions()[1:]), "wsUnsubscribe must unsubscribe through an equivalent subscription")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs[:1]), "wsUnsubscribe must not error")
		require.NoError(t, ex.wsSubscribe(t.Context(), conn, subs[:1]), "wsSubscribe must subscribe the same subscription again")
		require.Len(t, ex.subscribedAccountSubscriptions(), 1, "only the orders subscription must be held")
		conn.clearMessages()
		require.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey must refresh the new key")
		assert.Empty(t, conn.messages(), "renewSubscribeKey should not subscribe again a channel unsubscribed, or subscribed since with the new key")
	})
}

func TestRenewSubscribeKeyLocking(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{key: replacementSubscribeKey, refreshDelay: time.Second}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	subs := subscribeAccountChannels(t, ex, conn)
	refreshed := make(chan struct{})
	go func() {
		defer close(refreshed)
		r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: time.Now()}
		assert.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey should refresh the key")
	}()
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/refresh_key.do")) == 1
	}, 5*time.Second, 5*time.Millisecond, "the refresh must be under way")
	start := time.Now()
	require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs[:1]), "wsUnsubscribe must not error during a refresh")
	assert.Less(t, time.Since(start), 500*time.Millisecond, "the account subscriptions should not wait for a slow refresh")
	<-refreshed

	ex.subscribeKeys.account.Lock()
	replaced := make(chan struct{})
	r := &subscribeKeyRenewal{key: testSubscribeKey, renewed: time.Now().Add(-2 * time.Hour)}
	go func() {
		defer close(replaced)
		assert.NoError(t, ex.renewSubscribeKey(t.Context(), conn, r, time.Hour), "renewSubscribeKey should replace the expired key")
	}()
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/get_key.do")) == 1
	}, 5*time.Second, 5*time.Millisecond, "the replacement key must be created while the account subscriptions change")
	sent := len(conn.messages())
	assert.Never(t, func() bool { return ex.subscribeKeys.get(conn) != testSubscribeKey || len(conn.messages()) > sent }, 100*time.Millisecond, 10*time.Millisecond, "the replacement should wait for the account subscriptions to change before it is installed")
	ex.subscribeKeys.account.Unlock()
	<-replaced
	assert.Equal(t, replacementSubscribeKey, ex.subscribeKeys.get(conn), "the replacement should be installed once the account subscriptions have changed")
	assert.Equal(t, accountSubscribeRequests(replacementSubscribeKey)[1:], conn.messages()[sent:], "the replacement should subscribe again the account channel still subscribed")
}

func TestSubscribedAccountSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{connected: true}
	assert.Empty(t, ex.subscribedAccountSubscriptions(), "subscribedAccountSubscriptions should return nothing before any subscription")
	subs := holdAccountSubscriptions(t, ex, conn)
	tick := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}, QualifiedChannel: "tick:btc_usdt"}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, tick), "AddSuccessfulSubscriptions must not error")
	assert.ElementsMatch(t, subs, ex.subscribedAccountSubscriptions(), "subscribedAccountSubscriptions should return the account subscriptions only")
	require.NoError(t, subs[1].SetState(subscription.ResubscribingState), "SetState must not error")
	assert.Equal(t, subs[:1], ex.subscribedAccountSubscriptions(), "subscribedAccountSubscriptions should leave out a subscription not subscribed")
}

func TestResubscribeAccountChannels(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn, other := &recordingConnection{connected: true}, &recordingConnection{connected: true}
	subs := subscribeAccountChannels(t, ex, conn)
	require.NoError(t, ex.resubscribeAccountChannels(t.Context(), conn), "resubscribeAccountChannels must not error")
	assert.Empty(t, conn.messages(), "resubscribeAccountChannels should subscribe nothing carrying the connection's key")

	ex.subscribeKeys.set(other, replacementSubscribeKey)
	require.NoError(t, ex.resubscribeAccountChannels(t.Context(), other), "resubscribeAccountChannels must not error")
	assert.Empty(t, other.messages(), "resubscribeAccountChannels should subscribe nothing a connection doesn't carry")

	ex.subscribeKeys.set(conn, "")
	assert.ErrorIs(t, ex.resubscribeAccountChannels(t.Context(), conn), errSubscribeKeyRequired, "resubscribeAccountChannels should report a connection without a key")

	ex.subscribeKeys.set(conn, replacementSubscribeKey)
	conn.setSendFailures(errSendFailed, "")
	assert.ErrorIs(t, ex.resubscribeAccountChannels(t.Context(), conn), errSendFailed, "resubscribeAccountChannels should report a channel it couldn't subscribe")
	conn.setSendFailures(nil, "")
	require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, subs[1]), "RemoveSubscriptions must not error")
	require.NoError(t, ex.resubscribeAccountChannels(t.Context(), conn), "resubscribeAccountChannels must not error")
	assert.Equal(t, accountSubscribeRequests(replacementSubscribeKey)[:1], conn.messages(), "resubscribeAccountChannels should subscribe again with the new key only the channels still subscribed")
	require.NoError(t, ex.resubscribeAccountChannels(t.Context(), conn), "resubscribeAccountChannels must not error")
	assert.Len(t, conn.messages(), 1, "resubscribeAccountChannels should subscribe a channel again only once")

	require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs[:1]), "wsUnsubscribe must not error")
	require.NoError(t, ex.wsSubscribe(t.Context(), other, accountSubscriptions()[:1]), "wsSubscribe must not error")
	conn.clearMessages()
	ex.subscribeKeys.set(conn, testSubscribeKey)
	require.NoError(t, ex.resubscribeAccountChannels(t.Context(), conn), "resubscribeAccountChannels must not error")
	assert.Empty(t, conn.messages(), "resubscribeAccountChannels should not subscribe a channel the connection has unsubscribed from")
}

func TestWebsocketAccountSubscriptionsSerialised(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{connected: true}
	ex.subscribeKeys.set(conn, testSubscribeKey)
	tick := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}, QualifiedChannel: "tick:btc_usdt"}
	subs := accountSubscriptions()
	ex.subscribeKeys.account.Lock()
	require.NoError(t, ex.wsSubscribe(t.Context(), conn, subscription.List{tick}), "wsSubscribe must not wait to subscribe market data")
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, ex.wsSubscribe(t.Context(), conn, subs), "wsSubscribe should subscribe the account channels")
		assert.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs[:1]), "wsUnsubscribe should unsubscribe an account channel")
	}()
	assert.Never(t, func() bool { return len(conn.messages()) > 1 }, 100*time.Millisecond, 10*time.Millisecond, "the account channels should wait for a key's renewal")
	ex.subscribeKeys.account.Unlock()
	<-done
	exp := append([]any{&websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"}}, accountSubscribeRequests(testSubscribeKey)...)
	exp = append(exp, &websocketSubscribeRequest{Action: "unsubscribe", Subscribe: wsOrderUpdateChannel, Pair: "all", SubscribeKey: testSubscribeKey})
	assert.Equal(t, exp, conn.messages(), "the account channels should be subscribed and unsubscribed once the renewal ends")

	ex.subscribeKeys.account.Lock()
	done = make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs[1:]), "wsUnsubscribe should unsubscribe an account channel")
	}()
	assert.Never(t, func() bool { return len(conn.messages()) > len(exp) }, 100*time.Millisecond, 10*time.Millisecond, "an account unsubscription should wait for a key's renewal")
	ex.subscribeKeys.account.Unlock()
	<-done
}

func TestWebsocketAccountSubscribe(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	}
	ex, venue := newWebsocketTestExchange(t, subs)
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 2, "both account subscriptions must be stored")
	for _, s := range stored {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "subscription %s should be subscribed", s)
	}
	conn, err := ex.Websocket.GetConnection(wsAccountConnection)
	require.NoError(t, err, "GetConnection must find the account connection")
	require.NotEmpty(t, ex.subscribeKeys.get(conn), "the account connection must hold a subscribe key")
	if !mockTests {
		// LBank refuses an unknown or expired subscribe key with an error frame shortly after the subscription
		assert.Never(t, func() bool { return len(recordedErrors(t, ex)) > 0 }, 5*time.Second, 100*time.Millisecond, "LBank should accept the connection's subscribe key")
		return
	}
	exp := []websocketSubscribeRequest{
		{Action: "subscribe", Subscribe: "orderUpdate", Pair: "all", SubscribeKey: testSubscribeKey},
		{Action: "subscribe", Subscribe: "assetUpdate", SubscribeKey: testSubscribeKey},
	}
	// LBank confirms no subscription, so the mock may still be reading the last requests
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.ElementsMatch(c, exp, venue.received("subscribe"), "each account subscription should carry the key LBank created for the connection")
	}, 30*time.Second, 50*time.Millisecond, "the mock should receive each account subscription")
	// The mock follows the balance subscription with the recorded balance update, which is stored
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		balance, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
		if assert.NoError(c, err, "GetBalance should find the streamed balance") {
			assert.InDelta(c, 114548.31881315, balance.Total, 1e-8, "the streamed balance should be stored")
		}
	}, 30*time.Second, 50*time.Millisecond, "the account channels must deliver updates")
	assert.Empty(t, recordedErrors(t, ex), "the account updates should be handled without error")
}

func TestWebsocketConnectSubscribeKeyRefused(t *testing.T) {
	t.Parallel()
	refused := &subscribeKeyServer{failGet: true}
	ex := newTestServerExchange(t, refused.handler(t))
	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	}
	_, err := dialWebsocketTestExchange(t, ex, subs)
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Connect must report the account subscriptions it couldn't make")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "Connect should report LBank's refusal of the subscribe key")
	assert.True(t, ex.Websocket.IsConnected(), "a refused subscribe key should leave the websocket connected")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := ticker.GetTicker(ex.Name, spotTestPair, asset.Spot)
		assert.NoError(c, err, "GetTicker should find the subscribed ticker")
	}, 30*time.Second, 50*time.Millisecond, "the market data must stream while the account channels are refused")
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 1, "only the market data subscription must be stored")
	assert.Equal(t, subscription.TickerChannel, stored[0].Channel, "the ticker subscription should be stored")
	assert.Equal(t, []string{""}, refused.sent("/v2/subscribe/get_key.do"), "Connect should ask for one subscribe key")

	refused.mu.Lock()
	refused.failGet = false
	refused.mu.Unlock()
	flushCtx, cancelFlush := context.WithCancel(t.Context())
	require.NoError(t, ex.Websocket.FlushChannels(flushCtx), "FlushChannels must subscribe the account channels once LBank creates a key")
	cancelFlush()
	assert.Len(t, ex.Websocket.GetSubscriptions(), 3, "FlushChannels should store the account subscriptions with the market data")
	conn, err := ex.Websocket.GetConnection(wsAccountConnection)
	require.NoError(t, err, "GetConnection must find the account connection")
	assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "FlushChannels should give the account connection the key LBank created")
	assert.Never(t, func() bool { return ex.subscribeKeys.get(conn) == "" }, 100*time.Millisecond, 10*time.Millisecond, "the account connection should keep its key once the flush is done")
	assert.Empty(t, refused.sent("/v2/subscribe/destroy_key.do"), "the key should stay open once the flush is done")
}

func TestWebsocketAccountHandlers(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	testexch.FixtureToDataHandler(t, "testdata/wsAccount.json", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, conn, b)
	})
	pair, err := ex.MatchSymbolWithAvailablePairs("btc_usdt", asset.Spot, true)
	require.NoError(t, err, "MatchSymbolWithAvailablePairs must find btc_usdt")

	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 4, "the handlers must relay both orders, the fill and the balance")
	balances, ok := payloads[3].(accounts.SubAccounts)
	require.True(t, ok, "the asset handler must relay the balance")
	require.Len(t, balances, 1, "the asset handler must relay one sub-account")
	// LBank sends its coins in lower case, which the codes it relays keep
	coin := currency.NewCode("usdt")
	usdt, ok := balances[0].Balances[coin]
	require.True(t, ok, "the asset handler must relay the coin's balance")
	require.NotZero(t, usdt.UpdatedAt, "the asset handler must stamp the balance")
	usdt.UpdatedAt = time.Time{}
	balances[0].Balances[coin] = usdt

	exp := []any{
		&order.Detail{
			Exchange: ex.Name, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell", Pair: pair, AssetType: asset.Spot,
			Side: order.Sell, Status: order.PartiallyFilled, Price: 80000, Amount: 0.005, ExecutedAmount: 0.003, RemainingAmount: 0.002,
			AverageExecutedPrice: 80000.5, Cost: 240.0015, LastUpdated: time.UnixMilli(1792888874775),
		},
		[]fill.Data{{
			ID: "da88f354d5e04c4ea12128aa5bdcb301", TradeID: "da88f354d5e04c4ea12128aa5bdcb301", Exchange: ex.Name, AssetType: asset.Spot,
			CurrencyPair: pair, Side: order.Sell, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell",
			Price: 80000.75, Amount: 0.002, Timestamp: time.UnixMilli(1792888874775),
		}},
		&order.Detail{
			Exchange: ex.Name, OrderID: "8241275e-8666-4ab8-a84d-2e87f82e12ff", Pair: pair, AssetType: asset.Spot,
			Side: order.Buy, Status: order.Active, Price: 50000, Amount: 0.001, RemainingAmount: 0.001, LastUpdated: time.UnixMilli(1792888874999),
		},
		accounts.SubAccounts{{
			AssetType: asset.Spot,
			Balances: accounts.CurrencyBalances{
				coin: {Currency: coin, Total: 114548.31881315, Free: 97430.6739041, Hold: 17117.64490905},
			},
		}},
	}
	assert.Equal(t, exp, payloads, "the account handlers should relay the orders, the fill and the balance")

	stored, err := ex.Accounts.CurrencyBalances(nil, asset.Spot)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, 114548.31881315, stored[currency.USDT].Total, "the asset handler should store the balance it relays")
}

// TestWebsocketOrderUpdateType checks order updates are relayed with the type LBank pushes, as in a market buy's update
// ccxt recorded live
func TestWebsocketOrderUpdateType(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	lbk := currency.NewPairWithDelimiter("lbk", "usdt", currency.UnderscoreDelimiter)
	available, err := ex.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err, "GetPairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, append(available, lbk), false), "StorePairs must not error")
	captured := `{"SERVER":"V2","orderUpdate":{"accAmt":"0","amount":"0","avgPrice":"0","customerID":"","orderAmt":"5","orderPrice":"0.009834","orderStatus":0,"price":"0.009834","remainAmt":"5","role":"taker","symbol":"lbk_usdt","type":"buy_market","updateTime":1705676718532,"uuid":"9b94ab2d-a510-4abe-a784-44a9d9c38ec7","volumePrice":"0"},"type":"orderUpdate","pair":"lbk_usdt","TS":"2024-01-19T23:05:18.548"}`
	require.NoError(t, ex.wsProcessOrderUpdate(t.Context(), orderUpdateMessage(t, captured)), "wsProcessOrderUpdate must relay a market buy's update")
	exp := []any{&order.Detail{
		Exchange: ex.Name, OrderID: "9b94ab2d-a510-4abe-a784-44a9d9c38ec7", Pair: lbk, AssetType: asset.Spot, Side: order.Buy, Type: order.Market,
		Status: order.Active, Price: 0.009834, QuoteAmount: 5, LastUpdated: time.UnixMilli(1705676718532),
	}}
	assert.Equal(t, exp, relayedPayloads(ex), "wsProcessOrderUpdate should relay the market buy with its quote amount")

	pair, err := ex.MatchSymbolWithAvailablePairs("btc_usdt", asset.Spot, true)
	require.NoError(t, err, "MatchSymbolWithAvailablePairs must find btc_usdt")
	partial := `{"orderUpdate":{"accAmt":"0.003","amount":"0.002","avgPrice":"80000.5","symbol":"btc_usdt","type":"sell","orderAmt":"0.005","orderStatus":1,"orderPrice":"80000","price":"80000.75","role":"maker","remainAmt":"0.002","updateTime":1792888874775,"uuid":"d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9","txUuid":"da88f354d5e04c4ea12128aa5bdcb301","volumePrice":"160.0015","customerID":"gct-sell"},"pair":"btc_usdt","type":"orderUpdate","SERVER":"V2","TS":"2026-10-25T08:41:14.816"}`
	for orderType, tc := range map[string]struct {
		side      order.Side
		orderType order.Type
		tif       order.TimeInForce
	}{
		"buy_market":  {side: order.Buy, orderType: order.Market},
		"sell_market": {side: order.Sell, orderType: order.Market},
		"buy_maker":   {side: order.Buy, orderType: order.Limit, tif: order.PostOnly},
		"sell_maker":  {side: order.Sell, orderType: order.Limit, tif: order.PostOnly},
		"buy_ioc":     {side: order.Buy, orderType: order.Limit, tif: order.ImmediateOrCancel},
		"sell_ioc":    {side: order.Sell, orderType: order.Limit, tif: order.ImmediateOrCancel},
		"buy_fok":     {side: order.Buy, orderType: order.Limit, tif: order.FillOrKill},
		"sell_fok":    {side: order.Sell, orderType: order.Limit, tif: order.FillOrKill},
	} {
		frame := strings.Replace(partial, `"type":"sell"`, `"type":"`+orderType+`"`, 1)
		require.NoErrorf(t, ex.wsProcessOrderUpdate(t.Context(), orderUpdateMessage(t, frame)), "wsProcessOrderUpdate must relay a %s update", orderType)
		detail := &order.Detail{
			Exchange: ex.Name, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell", Pair: pair, AssetType: asset.Spot,
			Side: tc.side, Type: tc.orderType, TimeInForce: tc.tif, Status: order.PartiallyFilled, Price: 80000, Amount: 0.005,
			ExecutedAmount: 0.003, RemainingAmount: 0.002, AverageExecutedPrice: 80000.5, Cost: 240.0015, LastUpdated: time.UnixMilli(1792888874775),
		}
		// A market buy's order amount is the quote amount it spends, which leaves its remaining amount unknown
		if orderType == "buy_market" {
			detail.Amount, detail.QuoteAmount, detail.RemainingAmount = 0, 0.005, 0
		}
		fills := []fill.Data{{
			ID: "da88f354d5e04c4ea12128aa5bdcb301", TradeID: "da88f354d5e04c4ea12128aa5bdcb301", Exchange: ex.Name, AssetType: asset.Spot,
			CurrencyPair: pair, Side: tc.side, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell",
			Price: 80000.75, Amount: 0.002, Timestamp: time.UnixMilli(1792888874775),
		}}
		assert.Equalf(t, []any{detail, fills}, relayedPayloads(ex), "wsProcessOrderUpdate should relay the %s order and its fill", orderType)
	}
}

// TestWebsocketOrderUpdateExecution checks a limit buy's updates, from partly filled to filled, carry its remaining
// amount and cumulative cost
func TestWebsocketOrderUpdateExecution(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	pair, err := ex.MatchSymbolWithAvailablePairs("btc_usdt", asset.Spot, true)
	require.NoError(t, err, "MatchSymbolWithAvailablePairs must find btc_usdt")
	frames := []string{
		`{"orderUpdate":{"accAmt":"0.003","amount":"0.003","avgPrice":"80000.5","symbol":"btc_usdt","type":"buy","orderAmt":"0.005","orderStatus":1,"orderPrice":"80001","price":"80000.5","role":"maker","remainAmt":"160.002","updateTime":1792888874775,"uuid":"o1","txUuid":"t1","volumePrice":"240.0015"},"pair":"btc_usdt","type":"orderUpdate","SERVER":"V2","TS":"2026-10-25T08:41:14.816"}`,
		`{"orderUpdate":{"accAmt":"0.005","amount":"0.002","avgPrice":"80000.3","symbol":"btc_usdt","type":"buy","orderAmt":"0.005","orderStatus":2,"orderPrice":"80001","price":"80000","role":"maker","remainAmt":"0","updateTime":1792888875775,"uuid":"o1","txUuid":"t2","volumePrice":"160"},"pair":"btc_usdt","type":"orderUpdate","SERVER":"V2","TS":"2026-10-25T08:41:15.816"}`,
	}
	for _, frame := range frames {
		require.NoError(t, ex.wsProcessOrderUpdate(t.Context(), orderUpdateMessage(t, frame)), "wsProcessOrderUpdate must relay the order's update")
	}
	exp := []any{
		&order.Detail{
			Exchange: ex.Name, OrderID: "o1", Pair: pair, AssetType: asset.Spot, Side: order.Buy, Status: order.PartiallyFilled, Price: 80001,
			Amount: 0.005, ExecutedAmount: 0.003, RemainingAmount: 0.002, AverageExecutedPrice: 80000.5, Cost: 240.0015,
			LastUpdated: time.UnixMilli(1792888874775),
		},
		[]fill.Data{{
			ID: "t1", TradeID: "t1", Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: pair, Side: order.Buy, OrderID: "o1",
			Price: 80000.5, Amount: 0.003, Timestamp: time.UnixMilli(1792888874775),
		}},
		&order.Detail{
			Exchange: ex.Name, OrderID: "o1", Pair: pair, AssetType: asset.Spot, Side: order.Buy, Status: order.Filled, Price: 80001,
			Amount: 0.005, ExecutedAmount: 0.005, AverageExecutedPrice: 80000.3, Cost: 400.0015, LastUpdated: time.UnixMilli(1792888875775),
		},
		[]fill.Data{{
			ID: "t2", TradeID: "t2", Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: pair, Side: order.Buy, OrderID: "o1",
			Price: 80000, Amount: 0.002, Timestamp: time.UnixMilli(1792888875775),
		}},
	}
	assert.Equal(t, exp, relayedPayloads(ex), "wsProcessOrderUpdate should relay each update with the order's remaining amount and cost")

	require.ErrorIs(t, ex.wsProcessOrderUpdate(t.Context(), &websocketMessage{}), common.ErrNoResponse, "wsProcessOrderUpdate must report a frame without its update")
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must fill the data handler")
	assert.Error(t, ex.wsProcessOrderUpdate(t.Context(), orderUpdateMessage(t, frames[0])), "wsProcessOrderUpdate should report an update it couldn't relay")
}

func TestWebsocketAccountHandlerErrors(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	for _, tc := range []struct {
		name  string
		frame string
		err   error
	}{
		{name: "unknown pair", frame: `{"orderUpdate":{"symbol":"nope_usdt","type":"buy","orderStatus":0},"type":"orderUpdate"}`, err: currency.ErrPairNotFound},
		{name: "unknown side", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"hold","orderStatus":0},"type":"orderUpdate"}`, err: order.ErrSideIsInvalid},
		{name: "unknown side and type", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"hold_market","orderStatus":0},"type":"orderUpdate"}`, err: order.ErrSideIsInvalid},
		{name: "unknown type", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"buy_stop","orderStatus":0},"type":"orderUpdate"}`, err: order.ErrUnsupportedOrderType},
		{name: "unknown status", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"buy","orderStatus":9},"type":"orderUpdate"}`, err: order.ErrUnsupportedStatusType},
		{name: "balance without a coin", frame: `{"data":{"asset":"1","free":"1","freeze":"0"},"type":"assetUpdate"}`, err: currency.ErrCurrencyCodeEmpty},
	} {
		err := ex.wsHandleData(t.Context(), conn, []byte(tc.frame))
		assert.ErrorIsf(t, err, tc.err, "wsHandleData should reject an update with an %s", tc.name)
	}
	assert.Empty(t, relayedPayloads(ex), "wsHandleData should relay nothing it rejected")

	ex.Websocket.Fills.Setup(false, ex.Websocket.DataHandler)
	frame := `{"orderUpdate":{"accAmt":"1","amount":"1","symbol":"btc_usdt","type":"buy","orderStatus":2,"price":"1","txUuid":"t1","uuid":"o1","updateTime":1792888874775},"type":"orderUpdate"}`
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must relay an order's trade without the fills feed")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 1, "wsHandleData must relay only the order without the fills feed")
	assert.IsType(t, &order.Detail{}, payloads[0], "wsHandleData should relay the order")
}

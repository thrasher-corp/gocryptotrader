package binanceus

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Request securities a WebSocket API fixture records: a NONE request carries neither, a USER_STREAM request the API key
// alone and a SIGNED request the API key, timestamp and signature
const (
	securityNone   = ""
	securityAPIKey = "apiKey"
	securitySigned = "signed"
)

// websocketAPIFixture is a WebSocket API method's recorded response to the parameters that request it
type websocketAPIFixture struct {
	Security string            `json:"security"`
	Params   map[string]string `json:"params"`
	Status   int64             `json:"status"`
	Result   json.RawMessage   `json:"result"`
	Error    json.RawMessage   `json:"error"`
}

var loadWebsocketAPIFixtures = sync.OnceValues(func() (map[string][]websocketAPIFixture, error) {
	raw, err := os.ReadFile("testdata/websocket_api.json")
	if err != nil {
		return nil, err
	}
	var fixtures map[string][]websocketAPIFixture
	return fixtures, json.Unmarshal(raw, &fixtures)
})

// websocketAPIMock answers each WebSocket API request with the fixture whose security and parameters equal the
// request's, after checking that a SIGNED request carries the placeholder API key, a timestamp, the default receive
// window and a valid signature over its other parameters, and that a USER_STREAM request carries the placeholder API key
func websocketAPIMock(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	fixtures, err := loadWebsocketAPIFixtures()
	require.NoError(tb, err, "loadWebsocketAPIFixtures must not error")
	var req WebsocketAPIRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	params := maps.Clone(req.Params)
	if params == nil {
		params = map[string]string{}
	}
	security := securityNone
	if signature, ok := params["signature"]; ok {
		security = securitySigned
		delete(params, "signature")
		mac := hmac.New(sha256.New, []byte("mock-api-secret"))
		mac.Write([]byte(signaturePayload(params)))
		assert.Equalf(tb, hex.EncodeToString(mac.Sum(nil)), signature, "%s signature should cover every other parameter", req.Method)
		assert.NotEmptyf(tb, params["timestamp"], "%s should send a timestamp", req.Method)
		assert.Equalf(tb, "5000", params["recvWindow"], "%s should send the default receive window", req.Method)
		delete(params, "timestamp")
		delete(params, "recvWindow")
	} else if _, ok := params["apiKey"]; ok {
		security = securityAPIKey
	}
	if security != securityNone {
		assert.Equalf(tb, "mock-api-key", params["apiKey"], "%s should send the API key", req.Method)
		delete(params, "apiKey")
	}
	for _, f := range fixtures[req.Method] {
		fixtureParams := f.Params
		if fixtureParams == nil {
			fixtureParams = map[string]string{}
		}
		if f.Security != security || !maps.Equal(fixtureParams, params) {
			continue
		}
		resp := map[string]any{"id": req.ID, "status": 200}
		if f.Status != 0 {
			resp["status"] = f.Status
		}
		if len(f.Error) != 0 {
			resp["error"] = f.Error
		} else {
			resp["result"] = f.Result
		}
		data, err := json.Marshal(resp)
		require.NoError(tb, err, "response must encode")
		return w.WriteMessage(gws.TextMessage, data)
	}
	tb.Errorf("no %s fixture matches security %q parameters %v", req.Method, security, params)
	return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":400,"error":{"code":-1,"msg":"no fixture"}}`, req.ID))
}

// websocketAPIFixtureResult decodes the result of a method's fixture into the type the method returns
func websocketAPIFixtureResult[T any](tb testing.TB, method string, index int) T {
	tb.Helper()
	fixtures, err := loadWebsocketAPIFixtures()
	require.NoError(tb, err, "loadWebsocketAPIFixtures must not error")
	require.Lessf(tb, index, len(fixtures[method]), "%s must have fixture %d", method, index)
	var result T
	require.NoErrorf(tb, json.Unmarshal(fixtures[method][index].Result, &result), "%s fixture %d result must decode", method, index)
	return result
}

// newWebsocketTestExchange returns an exchange named after the test, so tests writing the shared ticker, order book and
// account stores cannot collide, with the test pair enabled and credentials: the live ones in live tests that have
// them, and placeholders otherwise, which tests against local servers and handlers sign with. Mock tests are not rate
// limited. Its configuration takes the name too, for a websocket set up from it, whose order book manager looks books
// up under the configuration's name
func newWebsocketTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	ex.Config.Name = ex.Name
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	if !mockTests && apiCredentials.Key != "" && apiCredentials.Secret != "" {
		ex.SetCredentials(apiCredentials)
	} else {
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	}
	if mockTests {
		require.NoError(tb, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	}
	pair := currency.NewBTCUSDT()
	require.NoError(tb, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair, currency.NewPair(currency.ETH, currency.USDT)}, false), "StorePairs must not error")
	require.NoError(tb, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, true), "StorePairs must not error")
	return ex
}

// useMockWebsocket serves a connection's setup from a local server answering each message with mock
func useMockWebsocket(tb testing.TB, ex *Exchange, setup *websocket.ConnectionSetup, mock mockws.WsMockFunc) {
	tb.Helper()
	s := httptest.NewTestServer(tb, mockws.CurryWsMockUpgrader(tb, mock))
	s.Start()
	setup.URL = "ws" + strings.TrimPrefix(s.URL, "http")
	useWebsocket(tb, ex, setup)
}

// useWebsocket sets an exchange's websocket up with one connection, and shuts it down when the test ends
func useWebsocket(tb testing.TB, ex *Exchange, setup *websocket.ConnectionSetup) {
	tb.Helper()
	ex.Config.ConnectionMonitorDelay = time.Hour
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(tb, ex.Websocket.Setup(ex.websocketManagerSetup(ex.Config)), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(tb, ex.Websocket.SetupNewConnection(setup), "SetupNewConnection must not error")
	tb.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(tb, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
	})
}

// newTwoServerTestExchange returns an exchange whose websocket is connected to a local stream server answering as
// streamsMock does and a local WebSocket API server answering with apiMock, counting each server's connections, with a
// connection monitor checking every 50ms and a traffic timeout no test reaches
func newTwoServerTestExchange(t *testing.T, apiMock mockws.WsMockFunc, authenticated bool) (ex *Exchange, streamDials, apiDials *atomic.Int32) {
	t.Helper()
	ex = newWebsocketTestExchange(t)
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	streamDials, apiDials = new(atomic.Int32), new(atomic.Int32)
	serve := func(dials *atomic.Int32, mock mockws.WsMockFunc) string {
		s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dials.Add(1)
			mockws.WsMockUpgrader(t, w, r, mock)
		}))
		s.Start()
		return "ws" + strings.TrimPrefix(s.URL, "http")
	}
	ex.Config.ConnectionMonitorDelay = 50 * time.Millisecond
	ex.Config.WebsocketTrafficTimeout = time.Hour
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel}}
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Websocket.Setup(ex.websocketManagerSetup(ex.Config)), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
	streams := ex.streamConnectionSetup(ex.Config)
	streams.URL = serve(streamDials, streamsMock)
	require.NoError(t, ex.Websocket.SetupNewConnection(streams), "SetupNewConnection must not error for the streams")
	api := ex.websocketAPIConnectionSetup(ex.Config)
	api.URL = serve(apiDials, apiMock)
	require.NoError(t, ex.Websocket.SetupNewConnection(api), "SetupNewConnection must not error for the WebSocket API")
	t.Cleanup(func() {
		// Disabling stops the connection monitor connecting again once the servers close; a shutdown it started may have
		// taken the websocket down already
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error unless the websocket is down")
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	return ex, streamDials, apiDials
}

// rejectKey answers a WebSocket API request as Binance.US answers one whose API key it refuses: with -2015, then by
// closing the connection
func rejectKey(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WebsocketAPIRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	if err := w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","status":401,"error":{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}}`)); err != nil {
		return err
	}
	if err := w.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(gws.ClosePolicyViolation, "disconnected"), time.Now().Add(time.Second)); err != nil {
		return err
	}
	return w.Close()
}

// newWebsocketAPITestExchange returns an exchange connected only to the WebSocket API, which the fixtures serve in mock
// tests and Binance.US in live tests. Live tests of SIGNED and USER_STREAM methods skip before connecting without
// credentials, or without permission to manipulate orders when canManipulateOrders is passed
func newWebsocketAPITestExchange(tb testing.TB, authenticated bool, canManipulateOrders ...bool) *Exchange {
	tb.Helper()
	if authenticated && !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(tb, e, canManipulateOrders...)
	}
	ex := newWebsocketTestExchange(tb)
	setup := ex.websocketAPIConnectionSetup(ex.Config)
	if mockTests {
		useMockWebsocket(tb, ex, setup, websocketAPIMock)
	} else {
		useWebsocket(tb, ex, setup)
	}
	require.NoError(tb, ex.Websocket.Connect(tb.Context()), "Connect must not error")
	return ex
}

// fakeConnection is a websocket connection whose state, responses and matched messages a test controls; calls to
// methods it does not implement panic
type fakeConnection struct {
	websocket.Connection
	mu        sync.Mutex
	connected bool
	response  []byte
	sendErr   error
	sent      []any
	limits    []request.EndpointLimit
	matched   map[string][]byte
	shutdowns int
}

func newFakeConnection() *fakeConnection {
	return &fakeConnection{connected: true, matched: make(map[string][]byte)}
}

func (f *fakeConnection) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connected
}

func (f *fakeConnection) SendMessageReturnResponse(_ context.Context, epl request.EndpointLimit, _, payload any) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, payload)
	f.limits = append(f.limits, epl)
	return f.response, f.sendErr
}

func (f *fakeConnection) RequireMatchWithData(signature any, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := signature.(string)
	if !ok {
		return common.ErrTypeAssertFailure
	}
	f.matched[id] = data
	return nil
}

func (f *fakeConnection) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shutdowns++
	f.connected = false
	return nil
}

// closed reports how many times the connection was shut down
func (f *fakeConnection) closed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.shutdowns
}

func TestWebsocketAPI(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.Same(t, ex, ex.WebsocketAPI().e, "WebsocketAPI should call the exchange's methods")
}

func TestWebsocketAPIConnectionSet(t *testing.T) {
	t.Parallel()
	var w websocketAPIConnection
	first, second := newFakeConnection(), newFakeConnection()
	w.set(first, time.Second)
	_ = w.retire(first)
	require.Nil(t, w.get(), "get must return nil for a retired connection")
	w.set(second, 2*time.Second)
	assert.Same(t, second, w.get(), "set should keep the new connection for requests, retired or not before")
	assert.Equal(t, 2*time.Second, w.responseLimit, "set should keep the new response limit")
	w.set(nil, 0)
	assert.Nil(t, w.kept(), "set should forget the connection")
}

func TestWebsocketAPIConnectionGet(t *testing.T) {
	t.Parallel()
	var w websocketAPIConnection
	assert.Nil(t, w.get(), "get should return nil before a connection is set")
	conn := newFakeConnection()
	w.set(conn, time.Second)
	assert.Same(t, conn, w.get(), "get should return the connection set")
	_ = w.retire(conn)
	assert.Nil(t, w.get(), "get should return nil once the connection is retired")
}

func TestWebsocketAPIConnectionKept(t *testing.T) {
	t.Parallel()
	var w websocketAPIConnection
	conn := newFakeConnection()
	w.set(conn, time.Second)
	_ = w.retire(conn)
	assert.Same(t, conn, w.kept(), "kept should return the connection even once it is retired")
}

func TestWebsocketAPIConnectionAcquire(t *testing.T) {
	t.Parallel()
	var w websocketAPIConnection
	conn, release := w.acquire()
	assert.Nil(t, conn, "acquire should return no connection before one is set")
	assert.Nil(t, release, "acquire should return no release before a connection is set")

	fake := newFakeConnection()
	w.set(fake, time.Hour)
	conn, release = w.acquire()
	require.Same(t, fake, conn, "acquire must return the connection set")
	require.NotNil(t, release, "acquire must return a release")
	wait := w.retire(fake)
	returned := make(chan struct{})
	go func() {
		wait()
		close(returned)
	}()
	select {
	case <-returned:
		require.Fail(t, "retire's wait must wait for a request in flight")
	case <-time.After(50 * time.Millisecond):
	}
	release()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		require.Fail(t, "retire's wait must return once the request in flight is released")
	}
	conn, _ = w.acquire()
	assert.Nil(t, conn, "acquire should return no connection once it is retired")
}

func TestWebsocketAPIConnectionRetire(t *testing.T) {
	t.Parallel()
	var w websocketAPIConnection
	conn, other := newFakeConnection(), newFakeConnection()
	w.set(conn, 10*time.Millisecond)
	w.retire(other)()
	assert.Same(t, conn, w.get(), "retire should leave the connection kept when another is retired")
	w.retire(nil)()
	assert.Same(t, conn, w.get(), "retire should leave the connection kept when none is retired")

	_, release := w.acquire()
	defer release()
	start := time.Now()
	w.retire(conn)()
	assert.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond, "retire's wait should wait for the response limit while a request is in flight")
	assert.Nil(t, w.get(), "retire should stop the connection taking requests")
	w.retire(conn)()
	assert.Nil(t, w.get(), "retiring the connection again should keep it retired")
}

func TestWebsocketAPIConnectionSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: time.Second}
	setup := ex.websocketAPIConnectionSetup(exch)
	require.NotNil(t, setup, "websocketAPIConnectionSetup must return a setup")
	assert.Equal(t, websocketAPIURL, setup.URL, "URL should be the WebSocket API's")
	assert.Equal(t, websocketAPIMessageFilter, setup.MessageFilter, "MessageFilter should name the WebSocket API connection")
	assert.True(t, setup.SubscriptionsNotRequired, "SubscriptionsNotRequired should be set, as the connection takes no stream subscriptions")
	assert.Equal(t, time.Second, setup.ResponseCheckTimeout, "ResponseCheckTimeout should be the configured one")
	assert.Equal(t, websocketAPIResponseMaxLimit, setup.ResponseMaxLimit, "ResponseMaxLimit should be at least the WebSocket API's")
	assert.NotNil(t, setup.Connector, "Connector should be set")
	assert.NotNil(t, setup.Authenticate, "Authenticate should be set")
	assert.NotNil(t, setup.Handler, "Handler should be set")
	assert.Nil(t, setup.RateLimit, "RateLimit should be unset, as requests draw on the weight pools they share with REST")
}

func TestWebsocketAPIResponseLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, websocketAPIResponseMaxLimit, websocketAPIResponseLimit(&config.Exchange{WebsocketResponseMaxLimit: time.Second}), "a shorter configured limit should give way to the WebSocket API's")
	assert.Equal(t, time.Minute, websocketAPIResponseLimit(&config.Exchange{WebsocketResponseMaxLimit: time.Minute}), "a longer configured limit should be kept")
}

func TestWebsocketConnectAPI(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	conn := ex.wsAPI.get()
	require.NotNil(t, conn, "WebsocketConnectAPI must keep the connection for requests")
	assert.True(t, isConnected(conn), "the connection kept should be up")

	ex.userData.expect("stale", &accounts.Credentials{Key: "k"})
	ex.Config.HTTPTimeout = time.Second
	err := ex.WebsocketConnectAPI(t.Context(), &fakeDialConnection{fakeConnection: newFakeConnection(), dialErr: errDialFailed})
	require.ErrorIs(t, err, errDialFailed, "WebsocketConnectAPI must return a failed dial")
	assert.Nil(t, ex.wsAPI.kept(), "a failed dial should leave no connection kept for requests")
	assert.Empty(t, ex.userData.pending, "connecting should forget the subscription requests of the earlier connection")
}

var errDialFailed = errors.New("dial failed")

// fakeDialConnection is a fakeConnection whose dial returns dialErr
type fakeDialConnection struct {
	*fakeConnection
	dialErr error
}

func (f *fakeDialConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return f.dialErr
}

func TestWebsocketHandleAPIData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := newFakeConnection()

	ex.userData.expect("1", &accounts.Credentials{Key: "k"})
	require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, []byte(`{"id":"1","status":200,"result":{"subscriptionId":3}}`)), "wsHandleAPIData must not error for a response")
	assert.Contains(t, conn.matched, "1", "a response should be matched to its request")
	_, ok := ex.userData.credentials(3)
	assert.True(t, ok, "a subscription response should record the subscription")

	err := ex.wsHandleAPIData(t.Context(), conn, []byte(`{"status":200}`))
	assert.ErrorIs(t, err, errUnhandledMessage, "a message without an id or event should be unhandled")

	require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, []byte(`{"subscriptionId":3,"event":{"e":"balanceUpdate","E":1573200697110,"a":"BTC","d":"100.00000000","T":1573200697068}}`)), "wsHandleAPIData must not error for a user data event")
	_, ok = relayed(t, ex).(*WebsocketBalanceUpdate)
	assert.True(t, ok, "a user data event should be passed to its handler")

	wrapped, raw := `{"event":{"e":"serverShutdown","E":1794466395877}}`, `{"e":"serverShutdown","E":1794466395877}`
	for _, tc := range []struct {
		name       string
		frame      string
		subscribed bool
	}{
		{name: "wrapped", frame: wrapped},
		{name: "raw", frame: raw},
		{name: "wrapped with a User Data Stream subscribed", frame: wrapped, subscribed: true},
		{name: "raw with a User Data Stream subscribed", frame: raw, subscribed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			shutdown := newWebsocketTestExchange(t)
			if tc.subscribed {
				shutdown.userData.expect("1", &accounts.Credentials{Key: "k"})
				require.NoError(t, shutdown.wsHandleAPIData(t.Context(), newFakeConnection(), []byte(`{"id":"1","status":200,"result":{"subscriptionId":3}}`)), "wsHandleAPIData must not error for a response")
				_, ok := shutdown.userData.credentials(3)
				require.True(t, ok, "the User Data Stream must be subscribed before the serverShutdown event")
			}
			shutdownConn := newFakeConnection()
			shutdown.wsAPI.set(shutdownConn, 10*time.Millisecond)
			require.NoError(t, shutdown.wsHandleAPIData(t.Context(), shutdownConn, []byte(tc.frame)), "wsHandleAPIData must not error for a serverShutdown event")
			assert.Nil(t, shutdown.wsAPI.get(), "a serverShutdown event should retire the connection")
		})
	}

	down := newFakeConnection()
	down.connected = false
	ex.wsAPI.set(down, 10*time.Millisecond)
	require.NoError(t, ex.wsHandleAPIData(t.Context(), down, []byte(`{"e":"serverShutdown","E":1794466395877}`)), "wsHandleAPIData must not error for a disconnected connection's serverShutdown event")
	assert.Nil(t, ex.wsAPI.get(), "a disconnected connection's serverShutdown event should retire it")
	assert.Never(t, func() bool { return down.closed() > 0 }, 100*time.Millisecond, 10*time.Millisecond, "a serverShutdown event should leave a disconnected connection to the websocket's reconnection")

	configured := configuredCredentials(t, ex)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.userData.expect("other", &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"})
	refusal := `{"id":"%s","status":401,"error":{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}}`
	require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, fmt.Appendf(nil, refusal, "other")), "wsHandleAPIData must not error for a refused subscription")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a refusal of another account's key should leave authenticated websocket use on")
	ex.userData.expect("configured", configured)
	require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, fmt.Appendf(nil, refusal, "configured")), "wsHandleAPIData must not error for a refused subscription")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a refusal of the configured account's key should turn authenticated websocket use off")
	assert.Contains(t, conn.matched, "configured", "a refused subscription should be matched to its request")

	for _, tc := range []struct {
		name          string
		frame         string
		authenticated bool
	}{
		{name: "a wrapped serverShutdown event has the websocket connect a new WebSocket API connection", frame: wrapped},
		{name: "a raw serverShutdown event has the websocket connect a new WebSocket API connection", frame: raw},
		{name: "a wrapped serverShutdown event has the websocket connect a new WebSocket API connection and subscribe its User Data Stream again", frame: wrapped, authenticated: true},
		{name: "a raw serverShutdown event has the websocket connect a new WebSocket API connection and subscribe its User Data Stream again", frame: raw, authenticated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var subscriptions atomic.Int32
			ex, _, apiDials := newTwoServerTestExchange(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				var req WebsocketAPIRequest
				require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
				switch req.Method {
				case "userDataStream.subscribe.signature":
					return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":200,"result":{"subscriptionId":%d}}`, req.ID, subscriptions.Add(1)))
				case "ping":
					if err := w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","status":200,"result":{}}`)); err != nil {
						return err
					}
					return w.WriteMessage(gws.TextMessage, []byte(tc.frame))
				}
				return websocketAPIMock(tb, msg, w)
			}, tc.authenticated)
			if tc.authenticated {
				require.Eventually(t, func() bool {
					_, ok := ex.userData.credentials(1)
					return ok
				}, 10*time.Second, 10*time.Millisecond, "the User Data Stream must be subscribed before the serverShutdown event")
			}
			require.NoError(t, ex.WebsocketAPI().TestConnectivity(t.Context()), "TestConnectivity must not error")
			assert.Eventually(t, func() bool {
				return apiDials.Load() == 2 && ex.WebsocketAPI().IsConnected()
			}, 10*time.Second, 10*time.Millisecond, "a serverShutdown event should have the websocket connect a new WebSocket API connection")
			if tc.authenticated {
				assert.Eventually(t, func() bool {
					_, ok := ex.userData.credentials(2)
					return ok && subscriptions.Load() == 2
				}, 10*time.Second, 10*time.Millisecond, "a serverShutdown event should have the new WebSocket API connection subscribe the User Data Stream again")
			}
		})
	}

	t.Run("a refused key turns authenticated websocket use off rather than reconnecting over and over", func(t *testing.T) {
		t.Parallel()
		var subscriptions atomic.Int32
		ex, streamDials, _ := newTwoServerTestExchange(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			subscriptions.Add(1)
			return rejectKey(tb, msg, w)
		}, true)
		assert.Eventually(t, func() bool {
			return streamDials.Load() == 2 && ex.Websocket.IsConnected()
		}, 10*time.Second, 10*time.Millisecond, "the websocket should connect again once Binance.US closes the connection")
		assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a refused subscription should turn authenticated websocket use off")
		assert.Never(t, func() bool { return streamDials.Load() > 2 }, 500*time.Millisecond, 10*time.Millisecond, "a refused key should not reconnect the websocket again")
		assert.Equal(t, int32(1), subscriptions.Load(), "a refused key should be sent one subscription")
	})
}

func TestAcquireWebsocketAPIConnection(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	_, _, err := ex.acquireWebsocketAPIConnection()
	require.ErrorIs(t, err, websocket.ErrNotConnected, "acquireWebsocketAPIConnection must error while the websocket is disconnected")

	ex = newWebsocketAPITestExchange(t, false)
	conn, release, err := ex.acquireWebsocketAPIConnection()
	require.NoError(t, err, "acquireWebsocketAPIConnection must not error while the WebSocket API is connected")
	assert.Same(t, ex.wsAPI.get(), conn, "acquireWebsocketAPIConnection should return the connection kept for requests")
	release()

	down := newFakeConnection()
	down.connected = false
	ex.wsAPI.set(down, time.Second)
	_, _, err = ex.acquireWebsocketAPIConnection()
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "acquireWebsocketAPIConnection should error for a connection that is down")

	ex.wsAPI.set(nil, 0)
	_, _, err = ex.acquireWebsocketAPIConnection()
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "acquireWebsocketAPIConnection should error without a connection kept")
}

func TestIsConnected(t *testing.T) {
	t.Parallel()
	conn := newFakeConnection()
	assert.True(t, isConnected(conn), "isConnected should report a connection that is up")
	conn.connected = false
	assert.False(t, isConnected(conn), "isConnected should report a connection that is down")
	assert.True(t, isConnected(struct{ websocket.Connection }{}), "isConnected should take a connection that cannot tell as up")
}

func TestWebsocketAPIIsConnected(t *testing.T) {
	t.Parallel()
	assert.False(t, newWebsocketTestExchange(t).WebsocketAPI().IsConnected(), "IsConnected should report a disconnected WebSocket API")
	assert.True(t, newWebsocketAPITestExchange(t, false).WebsocketAPI().IsConnected(), "IsConnected should report a connected WebSocket API")
}

func TestSendWebsocketAPIRequest(t *testing.T) {
	t.Parallel()
	err := newWebsocketTestExchange(t).SendWebsocketAPIRequest(t.Context(), defaultRate, "time", nil, nil)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "SendWebsocketAPIRequest must error while the websocket is disconnected")

	ex := newWebsocketAPITestExchange(t, false)
	var resp *ServerTimeResponse
	require.NoError(t, ex.SendWebsocketAPIRequest(t.Context(), defaultRate, "time", nil, &resp), "SendWebsocketAPIRequest must not error")
	require.NotNil(t, resp, "SendWebsocketAPIRequest must decode the result")
	assert.NotZero(t, resp.ServerTime.Time(), "SendWebsocketAPIRequest should decode the server time")
}

func TestSendAPIKeyWebsocketAPIRequest(t *testing.T) {
	t.Parallel()
	err := newWebsocketTestExchange(t).SendAPIKeyWebsocketAPIRequest(t.Context(), defaultRate, "userDataStream.start", nil, nil)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "SendAPIKeyWebsocketAPIRequest must error while the websocket is disconnected")

	ex := newWebsocketAPITestExchange(t, true)
	var resp *ListenKeyResponse
	require.NoError(t, ex.SendAPIKeyWebsocketAPIRequest(t.Context(), defaultRate, "userDataStream.start", nil, &resp), "SendAPIKeyWebsocketAPIRequest must not error")
	require.NotNil(t, resp, "SendAPIKeyWebsocketAPIRequest must decode the result")
	assert.NotEmpty(t, resp.ListenKey, "SendAPIKeyWebsocketAPIRequest should decode the listen key")

	ex.SetCredentials(&accounts.Credentials{})
	ex.API.AuthenticatedSupport = true
	ex.SkipAuthCheck = false
	err = ex.SendAPIKeyWebsocketAPIRequest(t.Context(), defaultRate, "userDataStream.start", nil, nil)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAPIKeyWebsocketAPIRequest should error without credentials")

	t.Run("a refused key sends the wrapper's requests over REST", func(t *testing.T) {
		t.Parallel()
		ex, streamDials, _ := newTwoServerTestExchange(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			if strings.Contains(string(msg), `"userDataStream.start"`) {
				return rejectKey(tb, msg, w)
			}
			return websocketAPIMock(tb, msg, w)
		}, false)
		err := ex.SendAPIKeyWebsocketAPIRequest(t.Context(), defaultRate, "userDataStream.start", nil, nil)
		assert.ErrorIs(t, err, errAPIResponse, "SendAPIKeyWebsocketAPIRequest should return the refusal")
		assert.True(t, ex.websocketAPIKeyRejected.Load(), "a refused key should send the wrapper's requests over REST")
		assert.Eventually(t, func() bool {
			return streamDials.Load() == 2 && ex.Websocket.IsConnected()
		}, 10*time.Second, 10*time.Millisecond, "the websocket should connect again once Binance.US closes the connection")
	})
}

func TestSendSignedWebsocketAPIRequest(t *testing.T) {
	t.Parallel()
	err := newWebsocketTestExchange(t).SendSignedWebsocketAPIRequest(t.Context(), defaultRate, "account.status", nil, nil)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "SendSignedWebsocketAPIRequest must error while the websocket is disconnected")

	ex := newWebsocketAPITestExchange(t, true)
	var resp *AccountResponse
	require.NoError(t, ex.SendSignedWebsocketAPIRequest(t.Context(), accountInformationRate, "account.status", nil, &resp), "SendSignedWebsocketAPIRequest must not error")
	require.NotNil(t, resp, "SendSignedWebsocketAPIRequest must decode the result")

	if mockTests {
		params := url.Values{}
		params.Set("symbol", "BTCUSDT")
		params.Set("orderId", "1")
		err = ex.SendSignedWebsocketAPIRequest(t.Context(), queryOrderRate, "order.status", params, nil)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "SendSignedWebsocketAPIRequest should map Binance.US's error codes")
		assert.ErrorIs(t, err, errWebsocketAPIRequestFailed, "SendSignedWebsocketAPIRequest should report a failed request")
	}

	t.Run("a refused key sends the wrapper's requests over REST and keeps the User Data Stream", func(t *testing.T) {
		t.Parallel()
		ex, streamDials, _ := newTwoServerTestExchange(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			var req WebsocketAPIRequest
			require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
			switch req.Method {
			case "userDataStream.subscribe.signature":
				return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","status":200,"result":{"subscriptionId":0}}`))
			case "account.status":
				return rejectKey(tb, msg, w)
			}
			return websocketAPIMock(tb, msg, w)
		}, true)
		require.True(t, ex.canUseWebsocketAPIForOrders(t.Context()), "orders must go over the WebSocket API before a refusal")
		err := ex.SendSignedWebsocketAPIRequest(t.Context(), accountInformationRate, "account.status", nil, nil)
		assert.ErrorIs(t, err, errAPIResponse, "SendSignedWebsocketAPIRequest should return the refusal")
		require.Eventually(t, func() bool {
			_, ok := ex.userData.credentials(0)
			return streamDials.Load() == 2 && ok && ex.WebsocketAPI().IsConnected()
		}, 10*time.Second, 10*time.Millisecond, "the websocket must connect again and subscribe the User Data Stream")
		assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a refused request should leave authenticated websocket use on")
		assert.False(t, ex.canUseWebsocketAPIForOrders(t.Context()), "orders should go over REST once the WebSocket API refused the configured account's key")
	})
}

func TestWebsocketAPIParams(t *testing.T) {
	t.Parallel()
	assert.Empty(t, websocketAPIParams(nil), "websocketAPIParams should return no parameters for none")
	params := websocketAPIParams(url.Values{"symbol": {"BTCUSDT"}, "symbols": {`["BTCUSDT","ETHUSDT"]`}, "limit": {"5", "6"}})
	assert.Equal(t, map[string]string{"symbol": "BTCUSDT", "symbols": `["BTCUSDT","ETHUSDT"]`, "limit": "5"}, params, "websocketAPIParams should send each parameter's first value as text")
}

func TestRateLimitAndSignWebsocketAPIParams(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	params := map[string]string{"symbol": "BTCUSDT"}
	require.NoError(t, ex.rateLimitAndSignWebsocketAPIParams(t.Context(), orderRate, params), "rateLimitAndSignWebsocketAPIParams must not error")
	assert.NotEmpty(t, params["signature"], "rateLimitAndSignWebsocketAPIParams should sign the parameters")

	limited := newWebsocketTestExchange(t)
	if mockTests {
		require.NoError(t, limited.EnableRateLimiter(), "EnableRateLimiter must not error")
	}
	params = map[string]string{"symbol": "BTCUSDT"}
	err := limited.rateLimitAndSignWebsocketAPIParams(t.Context(), request.Unset, params)
	assert.ErrorIs(t, err, common.ErrNilPointer, "rateLimitAndSignWebsocketAPIParams should return the error of a limit without a limiter")
	assert.NotContains(t, params, "signature", "rateLimitAndSignWebsocketAPIParams should not sign a request it cannot rate limit")
}

func TestSignWebsocketAPIParams(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	params := map[string]string{"symbol": "BTCUSDT", "side": "SELL", "price": "52000.00", "symbols": `["BTCUSDT","ETHUSDT"]`}
	require.NoError(t, ex.signWebsocketAPIParams(t.Context(), params), "signWebsocketAPIParams must not error")
	assert.Equal(t, "mock-api-key", params["apiKey"], "signWebsocketAPIParams should send the API key")
	assert.Equal(t, "5000", params["recvWindow"], "signWebsocketAPIParams should send the default receive window")
	require.NotEmpty(t, params["timestamp"], "signWebsocketAPIParams must send a timestamp")
	payload := "apiKey=mock-api-key&price=52000.00&recvWindow=5000&side=SELL&symbol=BTCUSDT" +
		`&symbols=["BTCUSDT","ETHUSDT"]&timestamp=` + params["timestamp"]
	mac := hmac.New(sha256.New, []byte("mock-api-secret"))
	mac.Write([]byte(payload))
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), params["signature"], "signWebsocketAPIParams should sign the parameters sorted by name, unencoded")

	// Signing again, as a retry does, must not sign the earlier signature, and keeps a receive window the caller set
	params["recvWindow"] = "100"
	require.NoError(t, ex.signWebsocketAPIParams(t.Context(), params), "signWebsocketAPIParams must not error signing again")
	assert.Equal(t, "100", params["recvWindow"], "signWebsocketAPIParams should keep a receive window the caller set")
	unsigned := maps.Clone(params)
	delete(unsigned, "signature")
	mac = hmac.New(sha256.New, []byte("mock-api-secret"))
	mac.Write([]byte(signaturePayload(unsigned)))
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), params["signature"], "signing again should sign every parameter but the signature")

	ex.SkipAuthCheck = false
	ex.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, ex.signWebsocketAPIParams(t.Context(), map[string]string{}), exchange.ErrCredentialsAreEmpty, "signWebsocketAPIParams should error without credentials")
}

func TestSignaturePayload(t *testing.T) {
	t.Parallel()
	assert.Empty(t, signaturePayload(nil), "signaturePayload should be empty for no parameters")
	assert.Equal(t, "a=1&b=x y&c=[\"A\",\"B\"]", signaturePayload(map[string]string{"c": `["A","B"]`, "a": "1", "b": "x y"}), "signaturePayload should join the parameters sorted by name without encoding them")
}

func TestSendOnWebsocketAPIConnection(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	req := &WebsocketAPIRequest{ID: "1", Method: "time"}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	conn := newFakeConnection()
	require.ErrorIs(t, ex.sendOnWebsocketAPIConnection(ctx, conn, defaultRate, req, nil), context.Canceled, "sendOnWebsocketAPIConnection must not send a cancelled request")
	assert.Empty(t, conn.sent, "a cancelled request should not be sent")

	for _, tc := range []struct {
		name     string
		response string
		sendErr  error
		err      error
	}{
		{name: "send error", sendErr: errDialFailed, err: errDialFailed},
		{name: "null", response: `null`, err: common.ErrNoResponse},
		{name: "error", response: `{"id":"1","status":400,"error":{"code":-2013,"msg":"Order does not exist."}}`, err: order.ErrOrderNotFound},
		{name: "status without error", response: `{"id":"1","status":418}`, err: errWebsocketAPIRequestFailed},
		{name: "null result", response: `{"id":"1","status":200,"result":null}`, err: common.ErrNoResponse},
		{name: "missing result", response: `{"id":"1","status":200}`, err: common.ErrNoResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			failing := newFakeConnection()
			failing.response, failing.sendErr = []byte(tc.response), tc.sendErr
			var resp *ServerTimeResponse
			assert.ErrorIsf(t, ex.sendOnWebsocketAPIConnection(t.Context(), failing, defaultRate, req, &resp), tc.err, "sendOnWebsocketAPIConnection should error for %s", tc.name)
		})
	}
	conn = newFakeConnection()
	conn.response = []byte(`{`)
	assert.Error(t, ex.sendOnWebsocketAPIConnection(t.Context(), conn, defaultRate, req, nil), "sendOnWebsocketAPIConnection should error for a response that is not JSON")

	conn = newFakeConnection()
	conn.response = []byte(`{"id":"1","status":200,"result":{"serverTime":1794466395877}}`)
	var resp *ServerTimeResponse
	require.NoError(t, ex.sendOnWebsocketAPIConnection(t.Context(), conn, defaultRate, req, &resp), "sendOnWebsocketAPIConnection must not error")
	assert.Equal(t, &ServerTimeResponse{ServerTime: types.Time(time.UnixMilli(1794466395877))}, resp, "sendOnWebsocketAPIConnection should decode the result")
	require.Len(t, conn.sent, 1, "sendOnWebsocketAPIConnection must send the request once")
	assert.Same(t, req, conn.sent[0], "sendOnWebsocketAPIConnection should send the request")

	conn.response = []byte(`{"id":"1","status":200,"result":{}}`)
	assert.NoError(t, ex.sendOnWebsocketAPIConnection(t.Context(), conn, defaultRate, req, nil), "sendOnWebsocketAPIConnection should not decode a result nobody wants")
}

func TestIsRejectedKey(t *testing.T) {
	t.Parallel()
	assert.True(t, isRejectedKey([]byte(`{"id":"1","status":401,"error":{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}}`)), "isRejectedKey should report a refused key")
	assert.False(t, isRejectedKey([]byte(`{"id":"1","status":400,"error":{"code":-1100,"msg":"Illegal characters found in parameter 'apiKey'."}}`)), "isRejectedKey should not report another error")
	assert.False(t, isRejectedKey([]byte(`{"id":"1","status":200,"result":{}}`)), "isRejectedKey should not report a success")
}

func TestStopAuthenticatedWebsocket(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.stopAuthenticatedWebsocket()
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "stopAuthenticatedWebsocket should turn authenticated websocket use off")
}

func TestNoteRejectedKey(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	configured := configuredCredentials(t, ex)
	rejected := fmt.Errorf("%w: %w", errWebsocketAPIRequestFailed, &APIError{Code: rejectedKeyCode, Message: "Invalid API-key, IP, or permissions for action."})
	ex.noteRejectedKey(&accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"}, rejected)
	assert.False(t, ex.websocketAPIKeyRejected.Load(), "noteRejectedKey should ignore a refusal of another account's key")
	ex.noteRejectedKey(configured, &APIError{Code: -1100})
	assert.False(t, ex.websocketAPIKeyRejected.Load(), "noteRejectedKey should ignore another error")
	ex.noteRejectedKey(configured, nil)
	assert.False(t, ex.websocketAPIKeyRejected.Load(), "noteRejectedKey should ignore a success")
	ex.noteRejectedKey(configured, rejected)
	assert.True(t, ex.websocketAPIKeyRejected.Load(), "noteRejectedKey should record a refusal of the configured account's key")
}

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
	"testing/synctest"
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
// limited over REST, and draw on limits of their own over the WebSocket API. Its configuration takes the name too, for a
// websocket set up from it, whose order book manager looks books up under the configuration's name
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
		// A websocket set up from the exchange limits WebSocket API requests itself, so mock tests running in parallel
		// would otherwise queue on the limits every instance shares
		request.WithLimiter(newRateLimits())(ex.Requester)
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
		var disableErr, shutdownErr error
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			disableErr = ex.Websocket.Disable()
			if ex.Websocket.IsConnected() {
				shutdownErr = ex.Websocket.Shutdown()
			}
		}()
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			assert.Fail(t, "the websocket should shut down within 10 seconds")
			return
		}
		assert.NoError(t, disableErr, "Disable should not error")
		if shutdownErr != nil {
			assert.ErrorIs(t, shutdownErr, websocket.ErrNotConnected, "Shutdown should not error unless the websocket is down")
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

// unawaitedConnection is a fakeConnection on which no request awaits a response any longer
type unawaitedConnection struct {
	*fakeConnection
}

func (unawaitedConnection) RequireMatchWithData(signature any, _ []byte) error {
	return fmt.Errorf("%q: %w", signature, websocket.ErrSignatureNotMatched)
}

// closeCheckingConnection is a fakeConnection counting its IsConnected calls, which closing it for a serverShutdown event
// makes first
type closeCheckingConnection struct {
	*fakeConnection
	checks atomic.Int32
}

func (c *closeCheckingConnection) IsConnected() bool {
	c.checks.Add(1)
	return c.fakeConnection.IsConnected()
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

// waitForRetirement returns how long a retirement's wait took to return, failing the test when it has not returned
// within 5 seconds
func waitForRetirement(t *testing.T, wait func()) time.Duration {
	t.Helper()
	start := time.Now()
	returned := make(chan struct{})
	go func() {
		wait()
		close(returned)
	}()
	select {
	case <-returned:
		return time.Since(start)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a retirement's wait must return within 5 seconds")
		return 0
	}
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

	t.Run("a request released after its connection was retired and replaced counts against its own connection", func(t *testing.T) {
		t.Parallel()
		var w websocketAPIConnection
		old, replacement := newFakeConnection(), newFakeConnection()
		w.set(old, time.Millisecond)
		_, releaseOld := w.acquire()
		releaseOld = sync.OnceFunc(releaseOld)
		defer releaseOld()
		waitForRetirement(t, w.retire(old))
		w.set(nil, 0)
		w.set(replacement, 10*time.Millisecond)
		_, releaseReplacement := w.acquire()
		defer releaseReplacement()
		require.NotPanics(t, releaseOld, "releasing a request on the replaced connection must not panic")
		assert.GreaterOrEqual(t, waitForRetirement(t, w.retire(replacement)), 10*time.Millisecond, "retiring the replacement should wait for its own request in flight")
	})

	t.Run("a retirement waits for an earlier request in flight once a later one returns", func(t *testing.T) {
		t.Parallel()
		var w websocketAPIConnection
		conn := newFakeConnection()
		w.set(conn, 10*time.Millisecond)
		_, releaseEarlier := w.acquire()
		defer releaseEarlier()
		_, releaseLater := w.acquire()
		releaseLater()
		assert.GreaterOrEqual(t, waitForRetirement(t, w.retire(conn)), 10*time.Millisecond, "retiring the connection should wait for the earlier request still in flight")
	})

	t.Run("a retirement waits for a later request in flight once an earlier one returns", func(t *testing.T) {
		t.Parallel()
		var w websocketAPIConnection
		conn := newFakeConnection()
		w.set(conn, 10*time.Millisecond)
		_, releaseEarlier := w.acquire()
		_, releaseLater := w.acquire()
		defer releaseLater()
		releaseEarlier()
		assert.GreaterOrEqual(t, waitForRetirement(t, w.retire(conn)), 10*time.Millisecond, "retiring the connection should wait for the later request still in flight")
	})
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
	t.Cleanup(release)
	assert.GreaterOrEqual(t, waitForRetirement(t, w.retire(conn)), 10*time.Millisecond, "retire's wait should wait for the response limit while a request is in flight")
	assert.Nil(t, w.get(), "retire should stop the connection taking requests")
	assert.GreaterOrEqual(t, waitForRetirement(t, w.retire(conn)), 10*time.Millisecond, "retiring the connection again should wait for the same request in flight")
	assert.Nil(t, w.get(), "retiring the connection again should keep it retired")

	t.Run("each retirement waits exactly a long response limit, to the microsecond, while a request is in flight", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			var w websocketAPIConnection
			conn := newFakeConnection()
			limit := time.Minute + 750*time.Microsecond
			w.set(conn, limit)
			_, release := w.acquire()
			defer release()
			start := time.Now()
			w.retire(conn)()
			assert.Equal(t, limit, time.Since(start), "retire's wait should last the response limit while a request is in flight")
			start = time.Now()
			w.retire(conn)()
			assert.Equal(t, limit, time.Since(start), "retiring the connection again should wait the response limit again")
		})
	})

	t.Run("a retirement's wait counts its response limit from when it runs", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			var w websocketAPIConnection
			conn := newFakeConnection()
			w.set(conn, time.Minute)
			_, release := w.acquire()
			defer release()
			wait := w.retire(conn)
			time.Sleep(2 * time.Second)
			start := time.Now()
			wait()
			assert.Equal(t, time.Minute, time.Since(start), "a retirement's wait should last the response limit from when it runs")
		})
	})

	for _, tc := range []struct {
		name            string
		releaseRetired  bool
		expectedElapsed time.Duration
	}{
		{name: "a retirement waits its own connection's response limit once a replacement with a longer limit takes requests", expectedElapsed: time.Minute},
		{name: "a retirement returns once its own connection's request has, while a replacement's request is in flight", releaseRetired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				var w websocketAPIConnection
				old, replacement := newFakeConnection(), newFakeConnection()
				w.set(old, time.Minute)
				_, releaseOld := w.acquire()
				releaseOld = sync.OnceFunc(releaseOld)
				defer releaseOld()
				wait := w.retire(old)
				w.set(replacement, 2*time.Minute)
				_, releaseReplacement := w.acquire()
				defer releaseReplacement()
				if tc.releaseRetired {
					releaseOld()
				}
				start := time.Now()
				wait()
				assert.Equal(t, tc.expectedElapsed, time.Since(start), "a retirement should wait for the retired connection's own request, within its own response limit")
			})
		})
	}
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
	assert.True(t, conn.IsConnected(), "the connection kept should be up")

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

	t.Run("serverShutdown events retire each connection in turn", func(t *testing.T) {
		t.Parallel()
		shutdown := newWebsocketTestExchange(t)
		retire := func(id string, subscriptionID int, frame, which string) {
			t.Helper()
			apiConn := newFakeConnection()
			shutdown.wsAPI.set(apiConn, 10*time.Millisecond)
			shutdown.userData.expect(id, &accounts.Credentials{Key: "k"})
			require.NoErrorf(t, shutdown.wsHandleAPIData(t.Context(), apiConn, fmt.Appendf(nil, `{"id":%q,"status":200,"result":{"subscriptionId":%d}}`, id, subscriptionID)), "wsHandleAPIData must not error for the %s connection's subscription response", which)
			require.NoErrorf(t, shutdown.wsHandleAPIData(t.Context(), apiConn, fmt.Appendf(nil, `{"subscriptionId":%d,"event":{"e":"balanceUpdate","E":1573200697110,"a":"BTC","d":"100.00000000","T":1573200697068}}`, subscriptionID)), "wsHandleAPIData must not error for the %s connection's balance update", which)
			_, ok := relayed(t, shutdown).(*WebsocketBalanceUpdate)
			require.Truef(t, ok, "the %s connection's balance update must be passed to its handler", which)
			require.NoErrorf(t, shutdown.wsHandleAPIData(t.Context(), apiConn, []byte(frame)), "wsHandleAPIData must not error for the %s connection's serverShutdown event", which)
			assert.Nilf(t, shutdown.wsAPI.get(), "a serverShutdown event should retire the %s connection", which)
		}
		retire("1", 3, wrapped, "first")
		retire("2", 4, raw, "second")
		retire("3", 5, wrapped, "third")
	})

	for _, tc := range []struct {
		name     string
		apiFirst bool
	}{
		{name: "the stream server's notice first"},
		{name: "the WebSocket API server's notice first", apiFirst: true},
	} {
		t.Run("a WebSocket API connection both servers' serverShutdown notices retire is closed once its request in flight returns, "+tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				draining := new(Exchange)
				draining.Websocket = sharedtestvalues.NewTestWebsocket()
				apiConn := &closeCheckingConnection{fakeConnection: newFakeConnection()}
				streamConn := &closeCheckingConnection{fakeConnection: newFakeConnection()}
				draining.wsAPI.set(apiConn, time.Hour)
				_, release := draining.wsAPI.acquire()
				release = sync.OnceFunc(release)
				defer release()
				streamNotice := func() {
					require.NoError(t, draining.handleStreamsServerShutdown(streamConn, []byte(raw)), "handleStreamsServerShutdown must not error")
				}
				apiNotice := func() {
					require.NoError(t, draining.wsHandleAPIData(t.Context(), apiConn, []byte(wrapped)), "wsHandleAPIData must not error for the WebSocket API server's serverShutdown event")
				}
				if tc.apiFirst {
					apiNotice()
					streamNotice()
				} else {
					streamNotice()
					apiNotice()
				}
				synctest.Wait()
				assert.Zero(t, apiConn.checks.Load(), "the WebSocket API connection should not be closed while its request is in flight")
				assert.Zero(t, streamConn.checks.Load(), "the stream connection should not be closed while the WebSocket API connection's request is in flight")
				release()
				synctest.Wait()
				assert.Equal(t, int32(1), apiConn.checks.Load(), "the WebSocket API connection should be closed once its request returns")
				assert.Equal(t, int32(1), streamConn.checks.Load(), "the stream connection should be closed once the WebSocket API connection's request returns")
			})
		})
	}

	t.Run("a retired WebSocket API connection still delivers its responses and User Data Stream events", func(t *testing.T) {
		t.Parallel()
		draining := subscribedUserDataTestExchange(t)
		drainingConn := newFakeConnection()
		draining.wsAPI.set(drainingConn, time.Hour)
		_ = draining.wsAPI.retire(drainingConn)
		first, second := `{"id":"first","status":200,"result":{"serverTime":1794466395000}}`, `{"id":"second","status":200,"result":{}}`
		require.NoError(t, draining.wsHandleAPIData(t.Context(), drainingConn, []byte(first)), "wsHandleAPIData must not error for a response on a retired connection")
		require.NoError(t, draining.wsHandleAPIData(t.Context(), drainingConn, []byte(second)), "wsHandleAPIData must not error for a later response on a retired connection")
		assert.Equal(t, []byte(first), drainingConn.matched["first"], "a response on a retired connection should reach its request as received")
		assert.Equal(t, []byte(second), drainingConn.matched["second"], "a later response on a retired connection should reach its request as received")
		refusal := `{"id":"third","status":400,"error":{"code":-2010,"msg":"Account has insufficient balance for requested action."}}`
		require.NoError(t, draining.wsHandleAPIData(t.Context(), drainingConn, []byte(refusal)), "wsHandleAPIData must not error for an error response on a retired connection")
		assert.Equal(t, []byte(refusal), drainingConn.matched["third"], "an error response on a retired connection should reach its request as received")
		require.NoError(t, draining.wsHandleAPIData(t.Context(), drainingConn, []byte(`{"subscriptionId":0,"event":{"e":"balanceUpdate","E":1794466395010,"a":"BTC","d":"0.01000000","T":1794466395009}}`)), "wsHandleAPIData must not error for a User Data Stream event on a retired connection")
		_, ok := relayed(t, draining).(*WebsocketBalanceUpdate)
		assert.True(t, ok, "a User Data Stream event on a retired connection should be passed to its handler")
	})

	t.Run("a subscription response no request awaits any longer is still recorded, and its stream's events handled", func(t *testing.T) {
		t.Parallel()
		late := subscribedUserDataTestExchange(t)
		lateConn := unawaitedConnection{newFakeConnection()}
		late.userData.expect("late", &accounts.Credentials{Key: "other-account-key", Secret: "other-account-secret"})
		late.userData.returned("late")
		err := late.wsHandleAPIData(t.Context(), lateConn, []byte(`{"id":"late","status":200,"result":{"subscriptionId":1}}`))
		require.ErrorIs(t, err, websocket.ErrSignatureNotMatched, "wsHandleAPIData must return that no request awaits the response")
		_, ok := late.userData.credentials(1)
		require.True(t, ok, "a subscription response no request awaits must still record the subscription")
		require.NoError(t, late.wsHandleAPIData(t.Context(), lateConn, []byte(`{"subscriptionId":1,"event":{"e":"balanceUpdate","E":1794466395010,"a":"BTC","d":"0.01000000","T":1794466395009}}`)), "wsHandleAPIData must not error for the late subscription's event")
		_, ok = relayed(t, late).(*WebsocketBalanceUpdate)
		assert.True(t, ok, "the late subscription's event should be passed to its handler")
	})

	t.Run("two accounts' User Data Streams on one connection keep their own events and subscriptions", func(t *testing.T) {
		t.Parallel()
		shared := subscribedUserDataTestExchange(t)
		configured := configuredCredentials(t, shared)
		other := &accounts.Credentials{Key: "other-account-key", Secret: "other-account-secret"}
		sharedConn := newFakeConnection()
		send := func(subscriptionID uint64, event string) error {
			return shared.wsHandleAPIData(t.Context(), sharedConn, fmt.Appendf(nil, `{"subscriptionId":%d,"event":%s}`, subscriptionID, event))
		}
		shared.userData.reset()
		shared.userData.expect("configured", configured)
		require.NoError(t, shared.wsHandleAPIData(t.Context(), sharedConn, []byte(`{"id":"configured","status":200,"result":{"subscriptionId":0}}`)), "wsHandleAPIData must not error for the configured account's subscription response")
		shared.userData.expect("other", other)
		require.NoError(t, shared.wsHandleAPIData(t.Context(), sharedConn, []byte(`{"id":"other","status":200,"result":{"subscriptionId":1}}`)), "wsHandleAPIData must not error for another account's subscription response")
		position := func(subscriptionID uint64, free string) {
			t.Helper()
			require.NoErrorf(t, send(subscriptionID, `{"e":"outboundAccountPosition","E":1794466395402,"u":1794466395400,"B":[{"a":"BTC","f":"`+free+`","l":"0.00000000"}]}`), "wsHandleAPIData must not error for subscription %d's account position", subscriptionID)
			_, ok := relayed(t, shared).(accounts.SubAccounts)
			require.Truef(t, ok, "subscription %d's account position must be relayed as sub-accounts", subscriptionID)
		}
		position(0, "1.25000000")
		position(1, "2.75000000")
		balance := func(creds *accounts.Credentials, which string) float64 {
			t.Helper()
			stored, err := shared.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
			require.NoErrorf(t, err, "GetBalance must not error for the %s account", which)
			return stored.Total
		}
		assert.Equal(t, 1.25, balance(configured, "configured"), "the configured account's position should be stored as its own")
		assert.Equal(t, 2.75, balance(other, "other"), "the other account's position, updated in the same millisecond, should be stored as its own")

		require.NoError(t, send(0, `{"e":"balanceUpdate","E":1794466395410,"a":"BNB","d":"0.10000000","T":1794466395409}`), "wsHandleAPIData must not error for a balance update")
		deposit := relayed(t, shared)
		require.NoError(t, send(0, `{"e":"balanceUpdate","E":1794466395420,"a":"BNB","d":"-0.03000000","T":1794466395419}`), "wsHandleAPIData must not error for a later balance update")
		assert.Equal(t, &WebsocketBalanceUpdate{
			EventType: "balanceUpdate", EventTime: types.Time(time.UnixMilli(1794466395420)), Asset: currency.BNB, BalanceDelta: -0.03, ClearTime: types.Time(time.UnixMilli(1794466395419)),
		}, relayed(t, shared), "a later balance update should be relayed as its own")
		assert.Equal(t, &WebsocketBalanceUpdate{
			EventType: "balanceUpdate", EventTime: types.Time(time.UnixMilli(1794466395410)), Asset: currency.BNB, BalanceDelta: 0.1, ClearTime: types.Time(time.UnixMilli(1794466395409)),
		}, deposit, "a balance update should be left as it was relayed after a later one")
		require.NoError(t, send(0, `{"e":"externalLockUpdate","E":1794466395430,"a":"BTC","d":"0.01000000","T":1794466395429}`), "wsHandleAPIData must not error for an external lock update")
		lock := relayed(t, shared)
		require.NoError(t, send(0, `{"e":"externalLockUpdate","E":1794466395440,"a":"BTC","d":"-0.01000000","T":1794466395439}`), "wsHandleAPIData must not error for a later external lock update")
		assert.Equal(t, &WebsocketExternalLockUpdate{
			EventType: "externalLockUpdate", EventTime: types.Time(time.UnixMilli(1794466395440)), Asset: currency.BTC, Delta: -0.01, TransactionTime: types.Time(time.UnixMilli(1794466395439)),
		}, relayed(t, shared), "a later external lock update should be relayed as its own")
		assert.Equal(t, &WebsocketExternalLockUpdate{
			EventType: "externalLockUpdate", EventTime: types.Time(time.UnixMilli(1794466395430)), Asset: currency.BTC, Delta: 0.01, TransactionTime: types.Time(time.UnixMilli(1794466395429)),
		}, lock, "an external lock update should be left as it was relayed after a later one")
		require.NoError(t, send(1, `{"e":"externalLockUpdate","E":1794466395445,"a":"BTC","d":"0.01000000","T":1794466395444}`), "wsHandleAPIData must not error for the other account's external lock update")
		_, ok := relayed(t, shared).(*WebsocketExternalLockUpdate)
		assert.True(t, ok, "the other account's external lock update should be relayed")

		end := func(subscriptionID uint64, eventTime int64) {
			t.Helper()
			require.NoErrorf(t, send(subscriptionID, fmt.Sprintf(`{"e":"eventStreamTerminated","E":%d}`, eventTime)), "wsHandleAPIData must not error for the end of subscription %d's User Data Stream", subscriptionID)
			_, terminated := relayed(t, shared).(*WebsocketEventStreamTerminated)
			require.Truef(t, terminated, "the end of subscription %d's User Data Stream must be relayed", subscriptionID)
			_, subscribed := shared.userData.credentials(subscriptionID)
			assert.Falsef(t, subscribed, "the end of subscription %d's User Data Stream should end it", subscriptionID)
		}
		end(0, 1794466395450)
		_, ok = shared.userData.credentials(1)
		assert.True(t, ok, "the end of the configured account's User Data Stream should leave the other account's subscribed")
		require.NoError(t, send(1, `{"e":"balanceUpdate","E":1794466395460,"a":"BTC","d":"0.02000000","T":1794466395459}`), "wsHandleAPIData must not error for the other account's balance update after the configured account's stream ended")
		_, ok = relayed(t, shared).(*WebsocketBalanceUpdate)
		assert.True(t, ok, "the other account's balance update should be relayed after the configured account's stream ended")
		end(1, 1794466395470)
		assert.Empty(t, shared.Websocket.DataHandler.C, "each event should be relayed once")
	})

	t.Run("the end of a User Data Stream is handled on each connection, though a new connection's subscription takes an ended one's ID", func(t *testing.T) {
		t.Parallel()
		ended := subscribedUserDataTestExchange(t)
		end := func(which string, eventTime int64) {
			t.Helper()
			endedConn := newFakeConnection()
			ended.userData.reset()
			ended.wsAPI.set(endedConn, time.Second)
			ended.userData.expect(which, configuredCredentials(t, ended))
			require.NoErrorf(t, ended.wsHandleAPIData(t.Context(), endedConn, fmt.Appendf(nil, `{"id":%q,"status":200,"result":{"subscriptionId":0}}`, which)), "wsHandleAPIData must not error for the %s connection's subscription response", which)
			require.NoErrorf(t, ended.wsHandleAPIData(t.Context(), endedConn, fmt.Appendf(nil, `{"subscriptionId":0,"event":{"e":"eventStreamTerminated","E":%d}}`, eventTime)), "wsHandleAPIData must not error for the end of the %s connection's User Data Stream", which)
			_, ok := relayed(t, ended).(*WebsocketEventStreamTerminated)
			require.Truef(t, ok, "the end of the %s connection's User Data Stream must be relayed", which)
			_, ok = ended.userData.credentials(0)
			assert.Falsef(t, ok, "the end of the %s connection's User Data Stream should end its subscription", which)
		}
		end("first", 1728973001334)
		end("second", 1728973061334)
		assert.Empty(t, ended.Websocket.DataHandler.C, "the end of each User Data Stream should be relayed once")
	})

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

	t.Run("each serverShutdown event has the websocket connect a new WebSocket API connection, on the new connection too", func(t *testing.T) {
		t.Parallel()
		ex, _, apiDials := newTwoServerTestExchange(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			if err := websocketAPIMock(tb, msg, w); err != nil {
				return err
			}
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"e":"serverShutdown","E":%d}`, time.Now().UnixMilli()))
		}, false)
		reconnects := func(dials int32) {
			t.Helper()
			require.NoError(t, ex.WebsocketAPI().TestConnectivity(t.Context()), "TestConnectivity must not error")
			require.Eventuallyf(t, func() bool {
				return apiDials.Load() == dials && ex.WebsocketAPI().IsConnected()
			}, 10*time.Second, 10*time.Millisecond, "a serverShutdown event must have the websocket connect WebSocket API connection %d", dials)
		}
		reconnects(2)
		reconnects(3)
	})

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

	t.Run("a nil result", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			response string
			err      error
			msg      string
		}{
			{name: "empty object", response: `{"id":"1","status":200,"result":{}}`},
			{name: "empty array", response: `{"id":"1","status":200,"result":[]}`},
			{name: "null", response: `{"id":"1","status":200,"result":null}`},
			{name: "missing", response: `{"id":"1","status":200}`},
			{name: "data", response: `{"id":"1","status":200,"result":{"serverTime":1794466395877}}`, err: errUnexpectedResponseData, msg: "time: " + errUnexpectedResponseData.Error()},
			{name: "error", response: `{"id":"1","status":400,"error":{"code":-2013,"msg":"Order does not exist."}}`, err: order.ErrOrderNotFound, msg: "time status 400"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				conn := newFakeConnection()
				conn.response = []byte(tc.response)
				err := ex.sendOnWebsocketAPIConnection(t.Context(), conn, defaultRate, req, nil)
				assert.ErrorIs(t, err, tc.err, "sendOnWebsocketAPIConnection should return the expected error when no result is wanted")
				if tc.msg != "" {
					assert.ErrorContains(t, err, tc.msg, "sendOnWebsocketAPIConnection should name the method in its error")
				}
			})
		}
	})

	t.Run("a typed-nil result", func(t *testing.T) {
		t.Parallel()
		conn := newFakeConnection()
		conn.response = []byte(`{"id":"1","status":200,"result":{}}`)
		var result *ServerTimeResponse
		assert.Error(t, ex.sendOnWebsocketAPIConnection(t.Context(), conn, defaultRate, req, result), "sendOnWebsocketAPIConnection should error for a typed-nil result rather than take it for no result")
	})
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

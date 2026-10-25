package kucoin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// The recorded tokens the mock server answers the token requests with
const (
	mockPublicToken  = "2neAiuYvAU61ZDXANAGAsiL4-iAExhsBXZxftpOeh_55i3Ysy2q2LEsEWU64mdzUOPusi34M_wGoSf7iNyEWJ93g2WHl-j4M7tCZ_S21PuIByWXUJFDywtiYB9J6i9GjsxUuhPw3Blq6rhZlGykT3Vp1phUafnulOOpts-MEmEF-3bpfetLOAq79RbGaLlE4JBvJHl5Vs9Y=.ymP9jIr6v-vucrZr8761yA=="
	mockPrivateToken = "2neAiuYvAU737TOajb2U3uT8AEZqSWYe0fBD4LoHuXJDSC7gIzJiH4kNTWhCPISWo6nDpAe7aUaaHJ4fG8oRjFgMfUI2sM4IySWHrBceFocY8pKy2REU1HwZIngtMdMrjqPnP-biofFWbNaP1cl0X1pZc2SQ-33hDH1LgNP-yg80ZJv_Ctaj8sAFhTOZ8m1L.jut4vBQxXAseWKxODdGVGg=="
)

// wsFixture returns a recorded frame from a websocket fixture file in testdata
func wsFixture(tb testing.TB, file, key string) []byte {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	require.NoErrorf(tb, err, "reading fixture %s must not error", file)
	var frames map[string]json.RawMessage
	require.NoErrorf(tb, json.Unmarshal(data, &frames), "fixture %s must decode", file)
	frame, ok := frames[key]
	require.Truef(tb, ok, "fixture %s must contain %s", file, key)
	return frame
}

// wsFixtureKeys returns the keys of a websocket fixture file in testdata
func wsFixtureKeys(tb testing.TB, file string) []string {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	require.NoErrorf(tb, err, "reading fixture %s must not error", file)
	var frames map[string]json.RawMessage
	require.NoErrorf(tb, json.Unmarshal(data, &frames), "fixture %s must decode", file)
	keys := make([]string, 0, len(frames))
	for k := range frames {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// wsTestInstance returns an exchange named for the test, so the ticker and orderbook stores it writes stay apart from
// other tests
func wsTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := testInstance(t)
	ex.Name = t.Name()
	return ex
}

// accountTestInstance returns a test instance that can store balances, with placeholder credentials when live
// credentials are not set
func accountTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := wsTestInstance(t)
	if !ex.AreCredentialsValid(t.Context()) {
		ex.SkipAuthCheck = true
		ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret", ClientID: "test-passphrase"})
	}
	return ex
}

// relayed returns what an exchange's handler relayed, which it must have done before returning, as the handlers relay
// synchronously; failing at once rather than waiting keeps a handler that relays nothing from stalling the test
func relayed(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case p := <-ex.Websocket.DataHandler.C:
		return p.Data
	default:
		require.FailNow(t, "the handler must have relayed a message")
		return nil
	}
}

// received returns what ch sends, failing the test once five seconds pass without it rather than leaving the test
// waiting for go test's timeout
func received[T any](t *testing.T, ch <-chan T, msg string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		require.FailNow(t, msg)
	}
	var zero T
	return zero
}

// wsPush feeds a recorded frame through the router and returns what it relays
func wsPush(t *testing.T, ex *Exchange, file, key string) any {
	t.Helper()
	require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, file, key)), "wsHandleData must not error for %s", key)
	require.Lenf(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must relay one message for %s", key)
	return relayed(t, ex)
}

// liveWsData subscribes a fresh exchange to KuCoin and returns the first relayed message of type T; prepare, when set,
// readies the exchange before it connects
func liveWsData[T any](t *testing.T, prepare func(*Exchange), subs ...*subscription.Subscription) T {
	t.Helper()
	ex := liveWsConnect(t, prepare, subs...)
	timeout := time.After(time.Minute)
	for {
		select {
		case resp := <-ex.Websocket.DataHandler.C:
			if v, ok := resp.Data.(T); ok {
				return v
			}
			if err, ok := resp.Data.(error); ok {
				require.NoError(t, err, "the websocket must not relay an error")
			}
		case <-timeout:
			var zero T
			require.FailNowf(t, "no data", "a %T must arrive within a minute", zero)
		}
	}
}

// liveWsSubscribe subscribes a fresh exchange to KuCoin and checks every subscription is accepted, for topics that push
// only on account activity or during call auctions
func liveWsSubscribe(t *testing.T, subs ...*subscription.Subscription) {
	t.Helper()
	ex := liveWsConnect(t, nil, subs...)
	assert.Len(t, ex.Websocket.GetSubscriptions(), len(subs), "every subscription should be accepted")
}

// liveWsConnect connects a fresh exchange to KuCoin with the subscriptions
func liveWsConnect(t *testing.T, prepare func(*Exchange), subs ...*subscription.Subscription) *Exchange {
	t.Helper()
	ex := wsTestInstance(t)
	if prepare != nil {
		prepare(ex)
	}
	ex.Features.Subscriptions = subs
	connectTestWebsocket(t, ex)
	return ex
}

// connectTestWebsocket connects the exchange's websocket and disables it before shutting it down when the test ends,
// because the connection monitor keeps an enabled websocket connected and would redial with the ended test's context
func connectTestWebsocket(t *testing.T, ex *Exchange) {
	t.Helper()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() {
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should fail only when the connection monitor shut the websocket down first")
		}
	})
}

// ConnectionFixture is a websocket connection mock, this is used in test that previously used the actual websocket connection.
// The current design makes it difficult to connect and subscribe using the actual websocket connection in tests, so
// this mock is used to simulate the connection behaviour.
type ConnectionFixture struct {
	websocket.Connection
	messageResponse string
	messageError    error
	sendFn          func(any) ([]byte, error)
	sentRequests    []WebsocketSubscriptionRequest
}

func (c *ConnectionFixture) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, req any) ([]byte, error) {
	if input, ok := req.(*WebsocketSubscriptionRequest); ok {
		c.sentRequests = append(c.sentRequests, *input)
	}
	if c.messageError != nil {
		return nil, c.messageError
	}
	if c.sendFn != nil {
		return c.sendFn(req)
	}
	return []byte(c.messageResponse), nil
}

// fakeConnection is a websocket connection that records what a connector does with it and answers reads with scripted
// messages, in which {connectId} stands for the connect ID it was dialled with
type fakeConnection struct {
	websocket.Connection
	mu          sync.Mutex
	url         string
	dialValues  url.Values
	dialErr     error
	messages    []string
	pingHandler websocket.PingHandler
	rawSent     []string
	sendErr     error
	matched     map[any][]byte
	matchErr    error
	closeOnce   sync.Once
	closed      chan struct{}
	shutdowns   int
}

func newFakeConnection(messages ...string) *fakeConnection {
	return &fakeConnection{messages: messages, closed: make(chan struct{}), matched: make(map[any][]byte)}
}

func (c *fakeConnection) SetURL(u string) { c.url = u }

func (c *fakeConnection) GetURL() string { return c.url }

func (c *fakeConnection) Dial(_ context.Context, _ *gws.Dialer, _ http.Header, values url.Values) error {
	c.dialValues = values
	return c.dialErr
}

func (c *fakeConnection) ReadMessage() websocket.Response {
	c.mu.Lock()
	if len(c.messages) != 0 {
		m := strings.ReplaceAll(c.messages[0], "{connectId}", c.dialValues.Get("connectId"))
		c.messages = c.messages[1:]
		c.mu.Unlock()
		return websocket.Response{Type: gws.TextMessage, Raw: []byte(m)}
	}
	c.mu.Unlock()
	<-c.closed
	return websocket.Response{}
}

func (c *fakeConnection) SetupPingHandler(_ request.EndpointLimit, h websocket.PingHandler) {
	c.pingHandler = h
}

func (c *fakeConnection) SendRawMessage(_ context.Context, _ request.EndpointLimit, _ int, message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rawSent = append(c.rawSent, string(message))
	return c.sendErr
}

func (c *fakeConnection) RequireMatchWithData(signature any, incoming []byte) error {
	c.matched[signature] = incoming
	return c.matchErr
}

func (c *fakeConnection) Shutdown() error {
	c.shutdowns++
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func TestIntervalFromString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		interval string
		expected kline.Interval
		hasErr   bool
	}{
		{interval: "1min", expected: kline.OneMin},
		{interval: "15min", expected: kline.FifteenMin},
		{interval: "1hour", expected: kline.OneHour},
		{interval: "1day", expected: kline.OneDay},
		{interval: "1week", expected: kline.OneWeek},
		{interval: "bad", hasErr: true},
	}
	for _, tt := range tests {
		got, err := IntervalFromString(tt.interval)
		if tt.hasErr {
			require.ErrorIs(t, err, kline.ErrInvalidInterval)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tt.expected, got)
	}
}

func TestGetPublicToken(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicToken(t.Context())
	require.NoError(t, err, "GetPublicToken must not error")
	if mockTests {
		exp := &WebsocketTokenResponse{
			Token:           mockPublicToken,
			InstanceServers: []InstanceServer{{Endpoint: "wss://ws-api-spot.kucoin.com/", Encrypt: true, Protocol: "websocket", PingInterval: 18000, PingTimeout: 10000}},
		}
		assert.Equal(t, exp, result, "GetPublicToken should decode every field")
		return
	}
	assert.NotEmpty(t, result.Token, "GetPublicToken should return a token")
	require.NotEmpty(t, result.InstanceServers, "GetPublicToken must return a server")
	assert.Equal(t, "wss://ws-api-spot.kucoin.com/", result.InstanceServers[0].Endpoint, "GetPublicToken should return the spot server")
	assert.Positive(t, result.InstanceServers[0].PingInterval, "GetPublicToken should return a ping interval")
}

func TestGetPrivateToken(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPrivateToken(t.Context())
	require.NoError(t, err, "GetPrivateToken must not error")
	if mockTests {
		exp := &WebsocketTokenResponse{
			Token:           mockPrivateToken,
			InstanceServers: []InstanceServer{{Endpoint: "wss://ws-api-spot.kucoin.com/", Encrypt: true, Protocol: "websocket", PingInterval: 18000, PingTimeout: 10000}},
		}
		assert.Equal(t, exp, result, "GetPrivateToken should decode every field")
		return
	}
	assert.NotEmpty(t, result.Token, "GetPrivateToken should return a token")
	assert.NotEmpty(t, result.InstanceServers, "GetPrivateToken should return a server")
}

func TestGetFuturesPublicToken(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesPublicToken(t.Context())
	require.NoError(t, err, "GetFuturesPublicToken must not error")
	if mockTests {
		// The mock server serves the spot and futures hosts from one route, so this is the spot recording
		exp := &WebsocketTokenResponse{
			Token:           mockPublicToken,
			InstanceServers: []InstanceServer{{Endpoint: "wss://ws-api-spot.kucoin.com/", Encrypt: true, Protocol: "websocket", PingInterval: 18000, PingTimeout: 10000}},
		}
		assert.Equal(t, exp, result, "GetFuturesPublicToken should decode every field")
		return
	}
	assert.NotEmpty(t, result.Token, "GetFuturesPublicToken should return a token")
	require.NotEmpty(t, result.InstanceServers, "GetFuturesPublicToken must return a server")
	assert.Equal(t, "wss://ws-api-futures.kucoin.com/", result.InstanceServers[0].Endpoint, "GetFuturesPublicToken should return the futures server")
}

func TestGetFuturesPrivateToken(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPrivateToken(t.Context())
	require.NoError(t, err, "GetFuturesPrivateToken must not error")
	if mockTests {
		exp := &WebsocketTokenResponse{
			Token:           mockPrivateToken,
			InstanceServers: []InstanceServer{{Endpoint: "wss://ws-api-spot.kucoin.com/", Encrypt: true, Protocol: "websocket", PingInterval: 18000, PingTimeout: 10000}},
		}
		assert.Equal(t, exp, result, "GetFuturesPrivateToken should decode every field")
		return
	}
	assert.NotEmpty(t, result.Token, "GetFuturesPrivateToken should return a token")
	assert.NotEmpty(t, result.InstanceServers, "GetFuturesPrivateToken should return a server")
}

// TestGetPublicTokenErrors covers getPublicToken's failures against a local server
func TestGetPublicTokenErrors(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	_, err := ex.getPublicToken(t.Context(), exchange.RestUSDTMargined, getPublicTokenEPL)
	require.Error(t, err, "getPublicToken must reject an endpoint the exchange has no URL for")

	// The decoding error's text depends on the JSON backend, standard library or Sonic, so only its presence is checked
	for _, tc := range []struct {
		reply, errContains string
	}{
		{reply: `{"code":"400100","msg":"Invalid request"}`, errContains: "400100"},
		{reply: `not json`},
	} {
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "getPublicToken should post")
			assert.Equal(t, "/api/v1/bullet-public", r.URL.Path, "getPublicToken should post to the token path")
			_, _ = w.Write([]byte(tc.reply))
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		_, err = ex.getPublicToken(t.Context(), exchange.RestSpot, getPublicTokenEPL)
		require.ErrorContainsf(t, err, tc.errContains, "getPublicToken must return the error for %s", tc.reply)
	}
}

func TestNewSpotConnectionSetup(t *testing.T) {
	t.Parallel()
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: 2 * time.Second}
	s := e.newSpotConnectionSetup(exch, "wss://ws-api-spot.kucoin.com/")
	assert.Equal(t, "wss://ws-api-spot.kucoin.com/", s.URL, "URL should be the configured spot URL")
	assert.Equal(t, spotConnection, s.MessageFilter, "MessageFilter should name the spot connection")
	assert.False(t, s.Optional, "the spot connection should be required, so its failure is the connect's")
	assert.Equal(t, time.Second, s.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, 2*time.Second, s.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.NotNil(t, s.RateLimit, "RateLimit should be set")
	assert.NotNil(t, s.Connector, "Connector should be set")
	assert.NotNil(t, s.GenerateSubscriptions, "GenerateSubscriptions should be set")
	assert.NotNil(t, s.Subscriber, "Subscriber should be set")
	assert.NotNil(t, s.Unsubscriber, "Unsubscriber should be set")
	assert.NotNil(t, s.Handler, "Handler should be set")
}

func TestNewFuturesConnectionSetup(t *testing.T) {
	t.Parallel()
	exch := &config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: 2 * time.Second}
	s := e.newFuturesConnectionSetup(exch, kucoinFuturesWebsocketURL)
	assert.Equal(t, kucoinFuturesWebsocketURL, s.URL, "URL should be the futures URL")
	assert.Equal(t, futuresConnection, s.MessageFilter, "MessageFilter should name the futures connection")
	assert.True(t, s.Optional, "the futures connection should be optional, so its failure leaves spot up")
	assert.Equal(t, time.Second, s.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, 2*time.Second, s.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.NotNil(t, s.RateLimit, "RateLimit should be set")
	assert.NotNil(t, s.Connector, "Connector should be set")
	assert.NotNil(t, s.GenerateSubscriptions, "GenerateSubscriptions should be set")
	assert.NotNil(t, s.Subscriber, "Subscriber should be set")
	assert.NotNil(t, s.Unsubscriber, "Unsubscriber should be set")
	assert.NotNil(t, s.Handler, "Handler should be set")
}

// newClassicWebsocketServer serves KuCoin's public and private token endpoints and the Classic websocket they name on
// loopback, for a spot REST host at /spot/api and a futures one at /futures/api. The websocket welcomes each connection,
// acknowledges each subscription and pushes a level 5 futures orderbook once subscribed to one; a token endpoint answers
// with its given status instead, and the order API at /order refuses every sign-in, counting them
func newClassicWebsocketServer(t *testing.T, spotStatus, futuresStatus int) (s *httptest.Server, orderSignIns *atomic.Int32) {
	t.Helper()
	orderSignIns = new(atomic.Int32)
	upgrader := gws.Upgrader{}
	// The token names the server's own websocket, which is known once the server starts
	var endpoint atomic.Pointer[string]
	token := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			if status != http.StatusOK {
				http.Error(w, http.StatusText(status), status)
				return
			}
			_, err := io.WriteString(w, `{"code":"200000","data":{"token":"test-token","instanceServers":[{"endpoint":"`+*endpoint.Load()+`","protocol":"websocket","pingInterval":18000,"pingTimeout":10000}]}}`)
			assert.NoError(t, err, "writing the token should not error")
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /spot/api/v1/bullet-public", token(spotStatus))
	mux.HandleFunc("POST /futures/api/v1/bullet-public", token(futuresStatus))
	mux.HandleFunc("POST /spot/api/v1/bullet-private", token(spotStatus))
	mux.HandleFunc("POST /futures/api/v1/bullet-private", token(futuresStatus))
	mux.HandleFunc("/order", func(w http.ResponseWriter, _ *http.Request) {
		orderSignIns.Add(1)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		if err := c.WriteMessage(gws.TextMessage, []byte(`{"id":"`+r.URL.Query().Get("connectId")+`","type":"welcome"}`)); err != nil {
			return
		}
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var req WebsocketSubscriptionRequest
			if !assert.NoError(t, json.Unmarshal(data, &req), "a client message should decode") {
				return
			}
			if req.Type == "subscribe" || req.Type == "unsubscribe" {
				if err := c.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","type":"ack"}`)); err != nil {
					return
				}
			}
			if channel, symbols, _ := strings.Cut(req.Topic, ":"); req.Type == "subscribe" && channel == futuresOrderbookDepth5Channel {
				for symbol := range strings.SplitSeq(symbols, ",") {
					push := `{"topic":"` + channel + ":" + symbol + `","type":"message","subject":"level2","data":{"sequence":1,"asks":[["101",1]],"bids":[["99",1]],"timestamp":1791899571610,"ts":1791899571610}}`
					if err := c.WriteMessage(gws.TextMessage, []byte(push)); err != nil {
						return
					}
				}
			}
		}
	})
	s = httptest.NewTestServer(t, mux)
	s.Start()
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
	endpoint.Store(&wsURL)
	return s, orderSignIns
}

// TestOptionalConnectionsLeaveSpotUp connects the Classic spot, Classic futures and order API setups through the
// websocket manager against a local KuCoin: the futures and order API connections are optional, so their failures leave
// the spot connection up, while the spot connection is required, so its failure is the connect's
func TestOptionalConnectionsLeaveSpotUp(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		spotStatus, futuresCode int
		authenticated           bool
		spotUp, futuresUp       bool
		// orderDown says why the order API connection is down
		orderDown string
	}{
		{name: "futures token refused", spotStatus: http.StatusOK, futuresCode: http.StatusServiceUnavailable, spotUp: true, orderDown: "while authenticated use is not enabled"},
		{name: "spot token refused", spotStatus: http.StatusServiceUnavailable, futuresCode: http.StatusOK},
		{name: "order API sign-in refused", spotStatus: http.StatusOK, futuresCode: http.StatusOK, authenticated: true, spotUp: true, futuresUp: true, orderDown: "while its sign-in is refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, orderSignIns := newClassicWebsocketServer(t, tc.spotStatus, tc.futuresCode)
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret", ClientID: "test-passphrase"})
			ex.API.AuthenticatedSupport = tc.authenticated
			ex.API.AuthenticatedWebsocketSupport = tc.authenticated
			ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
			require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
			require.NoError(t, ex.SetHTTPClient(s.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), s.URL+"/spot/api"), "SetRunningURL must not error for spot")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), s.URL+"/futures/api"), "SetRunningURL must not error for futures")
			ex.Features.Subscriptions = subscription.List{
				{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}},
				{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresTestPair}},
			}
			wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
			newUTAWebsocket(t, ex,
				ex.newSpotConnectionSetup(ex.Config, wsURL+"/ws"),
				ex.newFuturesConnectionSetup(ex.Config, wsURL+"/ws"),
				ex.newOrderConnectionSetup(ex.Config, wsURL+"/order"),
			)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			err := ex.Websocket.Connect(t.Context())
			if !tc.spotUp {
				require.Error(t, err, "Connect must fail when the spot connection cannot be made")
				assert.False(t, ex.Websocket.IsConnected(), "the websocket should not be connected without its spot connection")
				return
			}
			require.NoError(t, err, "Connect must not error when only optional connections fail")
			assert.True(t, ex.Websocket.IsConnected(), "the websocket should be connected")
			_, err = ex.Websocket.GetConnection(spotConnection)
			assert.NoError(t, err, "the spot connection should be up")
			_, err = ex.Websocket.GetConnection(futuresConnection)
			if tc.futuresUp {
				assert.NoError(t, err, "the futures connection should be up")
			} else {
				assert.Error(t, err, "the futures connection should be down while its token is refused")
			}
			_, err = ex.Websocket.GetConnection(orderConnection)
			assert.Errorf(t, err, "the order API connection should be down %s", tc.orderDown)
			if tc.authenticated {
				assert.NotZero(t, orderSignIns.Load(), "the order API's sign-in should be tried and refused")
			} else {
				assert.Zero(t, orderSignIns.Load(), "the order API connection should not be dialled while authenticated use is off")
			}
			subs := ex.Websocket.GetSubscriptions()
			assert.True(t, slices.ContainsFunc(subs, func(s *subscription.Subscription) bool {
				return s.QualifiedChannel == marketTickerChannel+":BTC-USDT"
			}), "the spot ticker should be subscribed with the futures and order API connections down")
		})
	}
}

// loadTestOrderbook stores a valid orderbook for a pair and asset in the websocket's orderbook store
func loadTestOrderbook(t *testing.T, ex *Exchange, p currency.Pair, a asset.Item) {
	t.Helper()
	require.NoErrorf(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
		Exchange:     ex.Name,
		Pair:         p,
		Asset:        a,
		Bids:         orderbook.Levels{{Price: 99, Amount: 1}},
		Asks:         orderbook.Levels{{Price: 101, Amount: 1}},
		LastUpdated:  time.Now(),
		LastUpdateID: 1,
	}), "LoadSnapshot must not error for %s %s", a, p)
}

// assertOrderbookValid asserts whether the websocket's orderbook of a pair and asset is valid
func assertOrderbookValid(t *testing.T, ex *Exchange, p currency.Pair, a asset.Item, valid bool, msg string) {
	t.Helper()
	_, err := ex.Websocket.Orderbook.GetOrderbook(p, a)
	if valid {
		assert.NoErrorf(t, err, "%s: the %s %s orderbook should be valid", msg, a, p)
		return
	}
	assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "%s: the %s %s orderbook should be invalid", msg, a, p)
}

// TestFuturesConnectionInvalidatesOrderbooks checks the Classic futures connection's disconnect hook invalidates the
// orderbooks that connection's pushes fed, whatever its subscription store holds, as a connection can drop while still
// subscribing, and leaves the orderbooks other connections fed valid
func TestFuturesConnectionInvalidatesOrderbooks(t *testing.T) {
	t.Parallel()
	t.Run("snapshot pushes", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		// Both subscription stores are empty, as they are while a connection is still subscribing
		dropped, other := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
		require.NoError(t, ex.wsHandleData(t.Context(), dropped, wsFixture(t, "wsFutures.json", "level50")), "wsHandleData must not error for the level 50 push")
		ethusdtm := currency.NewPair(currency.ETH, currency.USDTM)
		level5 := func(symbol string, sequence int) []byte {
			return []byte(`{"topic":"/contractMarket/level2Depth5:` + symbol + `","type":"message","subject":"level2","data":{"sequence":` + strconv.Itoa(sequence) + `,"asks":[["2509",1]],"bids":[["2508",1]],"timestamp":1791899571610,"ts":1791899571610}}`)
		}
		require.NoError(t, ex.wsHandleData(t.Context(), other, level5("ETHUSDTM", 1)), "wsHandleData must not error for the level 5 push")
		// A push the dropped connection sends for that book which fails does not make it the book's feed
		require.Error(t, ex.wsHandleData(t.Context(), dropped, []byte(`{"topic":"/contractMarket/level2Depth5:ETHUSDTM","type":"message","subject":"level2","data":{"asks":{}}}`)), "wsHandleData must reject an undecodable level 5 push")
		// A book the dropped connection fed before another connection fed it is the other connection's
		solusdtm := currency.NewPair(currency.SOL, currency.USDTM)
		require.NoError(t, ex.wsHandleData(t.Context(), dropped, level5("SOLUSDTM", 1)), "wsHandleData must not error for the dropped connection's level 5 push")
		require.NoError(t, ex.wsHandleData(t.Context(), other, level5("SOLUSDTM", 2)), "wsHandleData must not error for the other connection's level 5 push")
		loadTestOrderbook(t, ex, spotTestPair, asset.Spot)
		// A book the store does not hold has nothing to invalidate
		require.NoError(t, ex.feedOrderbook(dropped, currency.NewPair(currency.LTC, currency.USDTM), asset.Futures, 1, orderbookIncrement, func() error { return nil }), "feedOrderbook must not error for an increment queued for a book the store does not hold")
		ex.newFuturesConnectionSetup(ex.Config, kucoinFuturesWebsocketURL).OnDisconnect(t.Context(), dropped)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "the orderbook the dropped connection's pushes fed")
		assertOrderbookValid(t, ex, ethusdtm, asset.Futures, true, "an orderbook another connection fed")
		assertOrderbookValid(t, ex, solusdtm, asset.Futures, true, "an orderbook another connection fed after the dropped one")
		assertOrderbookValid(t, ex, spotTestPair, asset.Spot, true, "an orderbook no futures connection fed")
		ex.orderbookFeedsMu.RLock()
		_, droppedKept := ex.orderbookFeeds[dropped]
		_, otherKept := ex.orderbookFeeds[other]
		ex.orderbookFeedsMu.RUnlock()
		assert.False(t, droppedKept, "the hook should forget the orderbooks the dropped connection fed")
		assert.True(t, otherKept, "the hook should keep the orderbooks another connection fed")
	})

	t.Run("increment pushes", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		ex.wsOBUpdateMgr = newTestUpdateManager(ex, 1748392084690)
		dropped := &utaFakeConnection{subs: subscription.NewStore()}
		require.NoError(t, ex.wsHandleData(t.Context(), dropped, wsFixture(t, "wsFutures.json", "increment")), "wsHandleData must not error for the increment push")
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			_, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
			assert.NoError(c, err, "the increment should reach the futures book")
		}, time.Second, 10*time.Millisecond, "the increment must reach the futures book")
		ex.newFuturesConnectionSetup(ex.Config, kucoinFuturesWebsocketURL).OnDisconnect(t.Context(), dropped)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "the orderbook the dropped connection's increments fed")
	})
}

// TestOrderbookFeedOwnership checks a disconnect hook invalidates a book holding its connection's push, or one its
// connection queued a resync of, whatever the handler returned, and leaves a book holding another connection's push
func TestOrderbookFeedOwnership(t *testing.T) {
	t.Parallel()
	for _, feed := range []struct {
		name     string
		snapshot func(t *testing.T, ex *Exchange, conn websocket.Connection, sequence int, stamped bool) error
	}{
		{
			name: "Classic",
			snapshot: func(t *testing.T, ex *Exchange, conn websocket.Connection, sequence int, stamped bool) error {
				t.Helper()
				var timestamp string
				if stamped {
					timestamp = `,"timestamp":1791899571610`
				}
				return ex.wsHandleData(t.Context(), conn, []byte(`{"topic":"/contractMarket/level2Depth50:XBTUSDTM","type":"message","subject":"level2","data":{"sequence":`+strconv.Itoa(sequence)+`,"asks":[["101",1]],"bids":[["99",1]]`+timestamp+`}}`))
			},
		},
		{
			name: "UTA",
			snapshot: func(t *testing.T, ex *Exchange, conn websocket.Connection, sequence int, stamped bool) error {
				t.Helper()
				var timestamp string
				if stamped {
					timestamp = `,"M":1791901029947000000`
				}
				return ex.processUTAOrderbook(t.Context(), conn, utaMessage(t, `{"T":"obu.FUTURES","dp":"5","t":"snapshot","P":1791901029951262471,"d":{"a":[["101","1"]],"b":[["99","1"]],"s":"XBTUSDTM","O":`+strconv.Itoa(sequence)+`,"C":`+strconv.Itoa(sequence)+timestamp+`}}`), "FUTURES")
			},
		},
	} {
		t.Run(feed.name, func(t *testing.T) {
			t.Parallel()
			t.Run("latest push", func(t *testing.T) {
				t.Parallel()
				ex := wsTestInstance(t)
				earlier, latest := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
				require.NoError(t, feed.snapshot(t, ex, earlier, 100, true), "the earlier snapshot must be stored")
				require.NoError(t, feed.snapshot(t, ex, latest, 101, true), "the latest snapshot must be stored")
				ex.invalidateOrderbooks(t.Context(), earlier)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding another connection's push")
				ex.invalidateOrderbooks(t.Context(), latest)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's push")
				require.NoError(t, feed.snapshot(t, ex, earlier, 102, true), "the next snapshot must be stored")
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book the remaining connection pushed to again")
			})
			t.Run("push stored before its relay failed", func(t *testing.T) {
				t.Parallel()
				ex := wsTestInstance(t)
				earlier, latest := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
				require.NoError(t, feed.snapshot(t, ex, earlier, 100, true), "the earlier snapshot must be stored")
				for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
					require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must not error while the relay has room")
				}
				require.Error(t, feed.snapshot(t, ex, latest, 101, true), "the latest snapshot's relay must fail on a full relay")
				lastUpdateID, err := ex.Websocket.Orderbook.LastUpdateID(futuresTestPair, asset.Futures)
				require.NoError(t, err, "LastUpdateID must not error")
				require.Equal(t, int64(101), lastUpdateID, "the latest snapshot must be stored before its relay")
				ex.invalidateOrderbooks(t.Context(), latest)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's unrelayed push")
			})
			for _, sequence := range []int{101, 100} {
				t.Run("rejected push with update ID "+strconv.Itoa(sequence), func(t *testing.T) {
					t.Parallel()
					ex := wsTestInstance(t)
					fed, rejected := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
					require.NoError(t, feed.snapshot(t, ex, fed, 100, true), "the snapshot must be stored")
					require.ErrorIs(t, feed.snapshot(t, ex, rejected, sequence, false), orderbook.ErrLastUpdatedNotSet, "a snapshot without a timestamp must be rejected")
					ex.invalidateOrderbooks(t.Context(), rejected)
					assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book the dropped connection's rejected push left")
					ex.invalidateOrderbooks(t.Context(), fed)
					assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's push")
				})
			}
			t.Run("push with the stored update ID", func(t *testing.T) {
				t.Parallel()
				ex := wsTestInstance(t)
				earlier, latest := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
				require.NoError(t, feed.snapshot(t, ex, earlier, 100, true), "the earlier snapshot must be stored")
				require.NoError(t, feed.snapshot(t, ex, latest, 100, true), "a snapshot with the stored update ID must be stored")
				ex.invalidateOrderbooks(t.Context(), earlier)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding another connection's push with the same update ID")
				ex.invalidateOrderbooks(t.Context(), latest)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's push")
			})
			t.Run("push with the stored update ID stored before its relay failed", func(t *testing.T) {
				t.Parallel()
				ex := wsTestInstance(t)
				earlier, latest := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
				require.NoError(t, feed.snapshot(t, ex, earlier, 100, true), "the earlier snapshot must be stored")
				for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
					require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must not error while the relay has room")
				}
				require.ErrorIs(t, feed.snapshot(t, ex, latest, 100, true), stream.ErrChannelBufferFull, "the latest snapshot's relay must fail on a full relay")
				ex.invalidateOrderbooks(t.Context(), earlier)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding another connection's unrelayed push with the same update ID")
				ex.invalidateOrderbooks(t.Context(), latest)
				assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's unrelayed push")
			})
		})
	}

	t.Run("disconnect while another connection's push is stored", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		dropped, live := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
		require.NoError(t, ex.wsHandleData(t.Context(), dropped, []byte(`{"topic":"/contractMarket/level2Depth50:XBTUSDTM","type":"message","subject":"level2","data":{"sequence":100,"asks":[["101",1]],"bids":[["99",1]],"timestamp":1791899571610}}`)), "wsHandleData must not error")
		stored, release, fed, invalidated := make(chan struct{}), make(chan struct{}), make(chan error, 1), make(chan struct{})
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		go func() {
			fed <- ex.feedOrderbook(live, futuresTestPair, asset.Futures, 101, orderbookSnapshot, func() error {
				err := ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
					Exchange: ex.Name, Pair: futuresTestPair, Asset: asset.Futures, LastUpdateID: 101, LastUpdated: time.Now(),
					Bids: orderbook.Levels{{Price: 99, Amount: 1}}, Asks: orderbook.Levels{{Price: 101, Amount: 1}},
				})
				close(stored)
				<-release
				return err
			})
		}()
		received(t, stored, "the push must be stored")
		go func() {
			ex.invalidateOrderbooks(t.Context(), dropped)
			close(invalidated)
		}()
		select {
		case <-invalidated:
			assert.Fail(t, "the disconnect hook should wait for a push being stored")
		case <-time.After(100 * time.Millisecond):
		}
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, received(t, fed, "feedOrderbook must return once released"), "feedOrderbook must not error")
		received(t, invalidated, "the disconnect hook must return once the push is stored")
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding the push another connection stored as the dropped connection's hook ran")
	})

	t.Run("another connection's push while one is stored", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		first, second := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
		stored, release, firstFed, secondFed := make(chan struct{}), make(chan struct{}), make(chan error, 1), make(chan error, 1)
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		go func() {
			firstFed <- ex.feedOrderbook(first, futuresTestPair, asset.Futures, 100, orderbookSnapshot, func() error {
				err := ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
					Exchange: ex.Name, Pair: futuresTestPair, Asset: asset.Futures, LastUpdateID: 100, LastUpdated: time.Now(),
					Bids: orderbook.Levels{{Price: 99, Amount: 1}}, Asks: orderbook.Levels{{Price: 101, Amount: 1}},
				})
				close(stored)
				<-release
				return err
			})
		}()
		received(t, stored, "the push must be stored")
		// The second push has the same update ID, as two feeds of one book can
		go func() {
			secondFed <- ex.wsHandleData(t.Context(), second, []byte(`{"topic":"/contractMarket/level2Depth50:XBTUSDTM","type":"message","subject":"level2","data":{"sequence":100,"asks":[["101",1]],"bids":[["99",1]],"timestamp":1791899571610}}`))
		}()
		waited := true
		select {
		case <-secondFed:
			waited = false
		case <-time.After(100 * time.Millisecond):
		}
		assert.True(t, waited, "another connection's push should wait for a push being stored")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, received(t, firstFed, "feedOrderbook must return once released"), "feedOrderbook must not error")
		if waited {
			require.NoError(t, received(t, secondFed, "wsHandleData must return once the first push is stored"), "wsHandleData must not error")
		}
		ex.invalidateOrderbooks(t.Context(), first)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding another connection's later push with the same update ID")
		ex.invalidateOrderbooks(t.Context(), second)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's push")
	})

	t.Run("UTA deltas", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		fed, skipped, stored := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
		push := func(conn websocket.Connection, updateType string, start, end int) error {
			return ex.processUTAOrderbook(t.Context(), conn, utaMessage(t, `{"T":"obu.FUTURES","dp":"increment@10ms","t":"`+updateType+`","P":1791901029951262471,"d":{"a":[["101","1"]],"b":[["99","1"]],"s":"XBTUSDTM","O":`+strconv.Itoa(start)+`,"C":`+strconv.Itoa(end)+`,"M":1791901029947000000}}`), "FUTURES")
		}
		require.NoError(t, push(fed, "snapshot", 100, 100), "processUTAOrderbook must not error for the snapshot")
		for _, end := range []int{99, 100} {
			require.NoErrorf(t, push(skipped, "delta", end, end), "processUTAOrderbook must not error for a delta ending at %d, which the book holds", end)
		}
		ex.invalidateOrderbooks(t.Context(), skipped)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book the dropped connection's skipped delta left")
		require.NoError(t, push(stored, "delta", 101, 105), "processUTAOrderbook must not error for the next delta")
		ex.invalidateOrderbooks(t.Context(), fed)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, true, "a book holding another connection's later delta")
		ex.invalidateOrderbooks(t.Context(), stored)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's delta")
	})

	t.Run("Classic increment queued", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		fed, queued := &utaFakeConnection{subs: subscription.NewStore()}, &utaFakeConnection{subs: subscription.NewStore()}
		require.NoError(t, ex.wsHandleData(t.Context(), fed, []byte(`{"topic":"/contractMarket/level2Depth50:XBTUSDTM","type":"message","subject":"level2","data":{"sequence":100,"asks":[["101",1]],"bids":[["99",1]],"timestamp":1791899571610}}`)), "wsHandleData must not error for the snapshot")
		fetching, fetched := make(chan struct{}), make(chan struct{})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		ex.wsOBUpdateMgr = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
			FetchDeadline:      time.Minute,
			CheckPendingUpdate: checkPendingUpdate,
			Orderbook:          &ex.Websocket.Orderbook,
			FetchOrderbook: func(ctx context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Book, error) {
				defer close(fetched)
				close(fetching)
				<-ctx.Done()
				return nil, ctx.Err()
			},
		})
		require.NoError(t, ex.processFuturesOrderbookIncrement(ctx, queued, []byte(`{"sequence":101,"change":"99,buy,2","timestamp":1791899571611}`), "XBTUSDTM"), "processFuturesOrderbookIncrement must not error")
		select {
		case <-fetching:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the update manager must fetch a snapshot for the queued increment")
		}
		ex.invalidateOrderbooks(t.Context(), fed)
		assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "a book holding the dropped connection's push while another connection's resync is under way")
		cancel()
		received(t, fetched, "the snapshot fetch must return once its context is cancelled")
	})
}

// TestOptionalConnectionLossInvalidatesOrderbooks connects the Classic spot and futures setups through the websocket
// manager against a local KuCoin pushing a futures orderbook, and shuts them down: the futures orderbook the futures
// connection fed is invalidated, while the spot orderbook, which the required spot connection's loss leaves to the
// reconnection, is left alone
func TestOptionalConnectionLossInvalidatesOrderbooks(t *testing.T) {
	t.Parallel()
	s, _ := newClassicWebsocketServer(t, http.StatusOK, http.StatusOK)
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	require.NoError(t, ex.SetHTTPClient(s.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), s.URL+"/spot/api"), "SetRunningURL must not error for spot")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), s.URL+"/futures/api"), "SetRunningURL must not error for futures")
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}},
		{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresTestPair}},
	}
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
	newUTAWebsocket(t, ex, ex.newSpotConnectionSetup(ex.Config, wsURL), ex.newFuturesConnectionSetup(ex.Config, wsURL))
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	require.Eventually(t, func() bool {
		_, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
		return err == nil
	}, 5*time.Second, 10*time.Millisecond, "the futures connection must feed the futures orderbook")
	loadTestOrderbook(t, ex, spotTestPair, asset.Spot)
	require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
	assertOrderbookValid(t, ex, futuresTestPair, asset.Futures, false, "the orderbook of the futures connection shut down")
	assertOrderbookValid(t, ex, spotTestPair, asset.Spot, true, "an orderbook the futures connection did not feed")
}

// testWsConnector runs a Classic connector against a fake connection, publicly and then with a private token
func testWsConnector(t *testing.T, connect func(context.Context, *Exchange, websocket.Connection) error, publicEndpoint string) {
	t.Helper()
	ex := testInstance(t)
	conn := newFakeConnection(`{"id":"{connectId}","type":"welcome"}`)
	require.NoError(t, connect(t.Context(), ex, conn), "the connector must not error")
	connectID := conn.dialValues.Get("connectId")
	assert.NotEmpty(t, connectID, "the connector should dial with a connect ID")
	ping, err := json.Marshal(&WebsocketPingMessage{ID: connectID, Type: "ping"})
	require.NoError(t, err, "the ping must encode")
	assert.Equal(t, websocket.PingHandler{Delay: 9 * time.Second, Message: ping, MessageType: gws.TextMessage}, conn.pingHandler, "the connector should ping with the connect ID at half the ping interval")
	if mockTests {
		assert.Equal(t, mockPublicToken, conn.dialValues.Get("token"), "the connector should dial with the public token")
		assert.Equal(t, "wss://ws-api-spot.kucoin.com/", conn.url, "the connector should dial the token's server")
	} else {
		assert.NotEmpty(t, conn.dialValues.Get("token"), "the connector should dial with the public token")
		assert.Equal(t, publicEndpoint, conn.url, "the connector should dial the token's server")
	}

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.API.AuthenticatedSupport = true
	saved := ex.GetDefaultCredentials()
	skipAuthCheck := ex.SkipAuthCheck
	ex.SkipAuthCheck = false
	ex.SetCredentials(&accounts.Credentials{})
	err = connect(t.Context(), ex, newFakeConnection())
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "the connector must ask for a private token when authenticated")
	ex.SkipAuthCheck = skipAuthCheck

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ex.SetCredentials(saved)
	conn = newFakeConnection(`{"id":"{connectId}","type":"welcome"}`)
	require.NoError(t, connect(t.Context(), ex, conn), "the connector must not error when authenticated")
	if mockTests {
		assert.Equal(t, mockPrivateToken, conn.dialValues.Get("token"), "the connector should dial with the private token when authenticated")
	}
}

func TestWebsocketConnectSpot(t *testing.T) {
	t.Parallel()
	testWsConnector(t, func(ctx context.Context, ex *Exchange, conn websocket.Connection) error {
		return ex.WebsocketConnectSpot(ctx, conn)
	}, "wss://ws-api-spot.kucoin.com/")
}

func TestWebsocketConnectFutures(t *testing.T) {
	t.Parallel()
	testWsConnector(t, func(ctx context.Context, ex *Exchange, conn websocket.Connection) error {
		return ex.WebsocketConnectFutures(ctx, conn)
	}, "wss://ws-api-futures.kucoin.com/")
}

func TestWebsocketDialer(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Config.HTTPTimeout = 3 * time.Second
	d := ex.websocketDialer()
	assert.Equal(t, 3*time.Second, d.HandshakeTimeout, "the dialer should time out the handshake after the HTTP timeout")
	assert.NotNil(t, d.Proxy, "the dialer should honour the environment's proxy")
}

// welcomeServer returns a websocket server that welcomes each connection with the given frame, in which {connectId}
// stands for the connection's connect ID, and records every message it receives
func welcomeServer(t *testing.T, welcome string, received chan<- string) (*httptest.Server, *gws.Dialer) {
	t.Helper()
	upgrader := gws.Upgrader{}
	return testws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/endpoint" {
			http.NotFound(w, r)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		if welcome != "" {
			msg := strings.ReplaceAll(welcome, "{connectId}", r.URL.Query().Get("connectId"))
			if !assert.NoError(t, c.WriteMessage(gws.TextMessage, []byte(msg)), "writing the welcome should not error") {
				return
			}
		}
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			select {
			case received <- string(msg):
			default:
			}
		}
	}))
}

func TestConnectWebsocket(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	err := ex.connectWebsocket(t.Context(), newFakeConnection(), nil, &gws.Dialer{})
	require.ErrorIs(t, err, common.ErrNilPointer, "connectWebsocket must reject a nil token")
	err = ex.connectWebsocket(t.Context(), newFakeConnection(), &WebsocketTokenResponse{}, &gws.Dialer{})
	require.ErrorIs(t, err, errNoInstanceServer, "connectWebsocket must reject a token without a server")

	received := make(chan string, 8)
	server, dialer := welcomeServer(t, `{"id":"{connectId}","type":"welcome"}`, received)
	token := &WebsocketTokenResponse{Token: "secret/token=", InstanceServers: []InstanceServer{{Endpoint: "ws" + strings.TrimPrefix(server.URL, "http") + "/endpoint", PingInterval: 100}}}
	connected, err := ex.Websocket.CreateTestConnection(spotConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.connectWebsocket(t.Context(), connected, token, dialer), "connectWebsocket must not error")
	t.Cleanup(func() { assert.NoError(t, connected.Shutdown(), "Shutdown should not error") })
	select {
	case ping := <-received:
		var msg WebsocketPingMessage
		require.NoError(t, json.Unmarshal([]byte(ping), &msg), "the ping must decode")
		assert.Equal(t, "ping", msg.Type, "the connection should ping")
		assert.NotEmpty(t, msg.ID, "the ping should carry the connect ID")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "no ping arrived")
	}

	token.InstanceServers[0].Endpoint = "ws" + strings.TrimPrefix(server.URL, "http") + "/missing"
	conn, err := ex.Websocket.CreateTestConnection(spotConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	err = ex.connectWebsocket(t.Context(), conn, token, dialer)
	require.ErrorIs(t, err, errWebsocketDialFailed, "connectWebsocket must return a dial failure")
	assert.ErrorIs(t, err, gws.ErrBadHandshake, "a dial failure should keep the dial error")
	assert.NotContains(t, err.Error(), url.QueryEscape(token.Token), "a dial failure should not repeat the token")

	wrongWelcome, wrongDialer := welcomeServer(t, `{"id":"another","type":"welcome"}`, received)
	token.InstanceServers[0].Endpoint = "ws" + strings.TrimPrefix(wrongWelcome.URL, "http") + "/endpoint"
	conn, err = ex.Websocket.CreateTestConnection(spotConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	err = ex.connectWebsocket(t.Context(), conn, token, wrongDialer)
	require.ErrorIs(t, err, errWelcomeNotReceived, "connectWebsocket must reject a welcome for another connection")
	requireOpenUntilShutdown(t, conn)

	// An interval too long for a time.Duration would wrap to a short delay, so it gets the default too
	for interval, exp := range map[uint64]time.Duration{0: defaultPingDelay, 18000: 9 * time.Second, 9223372036854777808: defaultPingDelay} {
		fake := newFakeConnection(`{"id":"{connectId}","type":"welcome"}`)
		require.NoErrorf(t, ex.connectWebsocket(t.Context(), fake, &WebsocketTokenResponse{InstanceServers: []InstanceServer{{Endpoint: "wss://ws-api-spot.kucoin.com/", PingInterval: interval}}}, &gws.Dialer{}), "connectWebsocket must not error for a ping interval of %d", interval)
		assert.Equalf(t, exp, fake.pingHandler.Delay, "a server with a ping interval of %d should be pinged every %s", interval, exp)
	}

	silent, silentDialer := welcomeServer(t, "", received)
	token.InstanceServers[0].Endpoint = "ws" + strings.TrimPrefix(silent.URL, "http") + "/endpoint"
	ex.WebsocketResponseMaxLimit = 50 * time.Millisecond
	conn, err = ex.Websocket.CreateTestConnection(spotConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	err = ex.connectWebsocket(t.Context(), conn, token, silentDialer)
	require.ErrorIs(t, err, context.DeadlineExceeded, "connectWebsocket must give up when no welcome arrives")
	requireOpenUntilShutdown(t, conn)
}

// requireOpenUntilShutdown checks a connector left open a connection it dialled but failed to set up, as the websocket
// manager closes a connection that fails to connect, then shuts the connection down
func requireOpenUntilShutdown(t *testing.T, conn websocket.Connection) {
	t.Helper()
	status, ok := conn.(interface{ IsConnected() bool })
	require.True(t, ok, "the test connection must report whether it is connected")
	assert.True(t, status.IsConnected(), "a connector should leave closing a connection it failed to set up to the manager")
	require.NoError(t, conn.Shutdown(), "Shutdown must not error")
}

func TestReadWebsocketMessage(t *testing.T) {
	t.Parallel()
	conn := newFakeConnection(`{"id":"1","type":"welcome"}`)
	got, err := readWebsocketMessage(t.Context(), conn, time.Second)
	require.NoError(t, err, "readWebsocketMessage must not error")
	assert.JSONEq(t, `{"id":"1","type":"welcome"}`, string(got), "readWebsocketMessage should return the message")

	_, err = readWebsocketMessage(t.Context(), conn, 10*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded, "readWebsocketMessage must give up after the timeout")
	require.NoError(t, conn.Shutdown(), "Shutdown must not error")
	_, err = readWebsocketMessage(t.Context(), conn, 0)
	require.ErrorIs(t, err, errWelcomeNotReceived, "readWebsocketMessage must report a closed connection")
}

func TestCheckWelcome(t *testing.T) {
	t.Parallel()
	require.NoError(t, checkWelcome([]byte(`{"id":"abc","type":"welcome"}`), "abc"), "checkWelcome must accept the connection's welcome")
	require.ErrorIs(t, checkWelcome([]byte(`{"id":"xyz","type":"welcome"}`), "abc"), errWelcomeNotReceived, "checkWelcome must reject another connection's welcome")
	require.ErrorIs(t, checkWelcome([]byte(`{"id":"abc","type":"error","code":401,"data":"token is invalid"}`), "abc"), errWelcomeNotReceived, "checkWelcome must reject an error")
	require.Error(t, checkWelcome([]byte(`{"id":`), "abc"), "checkWelcome must reject an undecodable message")
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	conn := newFakeConnection()
	for _, frame := range []string{`{"id":"sub-1","type":"ack"}`, `{"id":"sub-2","type":"error","code":404,"data":"topic does not exist"}`} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must not error for %s", frame)
	}
	assert.Equal(t, map[any][]byte{"sub-1": []byte(`{"id":"sub-1","type":"ack"}`), "sub-2": []byte(`{"id":"sub-2","type":"error","code":404,"data":"topic does not exist"}`)}, conn.matched, "acks and errors should be matched to their requests")
	conn.matchErr = websocket.ErrSignatureNotMatched
	require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":"sub-3","type":"ack"}`)), websocket.ErrSignatureNotMatched, "wsHandleData must return an unmatched reply")

	for _, frame := range []string{`{"id":"1","type":"welcome"}`, `{"id":"1","type":"pong","timestamp":1791899445488782}`} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must not error for %s", frame)
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "welcome and pong frames should not be relayed")

	require.NoError(t, ex.wsHandleData(t.Context(), conn, wsFixture(t, "wsFutures.json", "orderMargin")), "wsHandleData must route a push without a type")
	assert.IsType(t, &WebsocketFuturesOrderMargin{}, relayed(t, ex), "a push without a type should reach its topic's handler")

	require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":"message","data":{}}`)), errUnhandledMessageType, "wsHandleData must reject a push without a topic")
	require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":"1","type":"notice"}`)), errUnhandledMessageType, "wsHandleData must reject an unknown message type")
	require.Error(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":`)), "wsHandleData must reject an undecodable message")
}

func TestProcessPush(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	err := ex.processPush(t.Context(), nil, &WebsocketMessage{Topic: "/market/unknown:BTC-USDT", Type: "message"})
	require.ErrorIs(t, err, errUnhandledTopic, "processPush must reject a topic it does not handle")
	assert.ErrorContains(t, err, "/market/unknown:BTC-USDT", "processPush should name the topic")

	got, ok := wsPush(t, ex, "wsSpot.json", "allTickers").([]ticker.Price)
	require.True(t, ok, "an All Tickers push must relay tickers")
	require.Len(t, got, 1, "an All Tickers push must relay the ticker of the symbol its subject names")
	assert.Equal(t, "ETH-USDT", got[0].Pair.String(), "an All Tickers push should be read for the symbol in its subject")
}

func TestProcessPayload(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	require.NoError(t, processPayload[WebsocketFuturesLeverage](t.Context(), ex, []byte(`{"leverage":"5"}`)), "processPayload must not error")
	assert.Equal(t, &WebsocketFuturesLeverage{Leverage: 5}, relayed(t, ex), "processPayload should relay the decoded push")
	require.Error(t, processPayload[WebsocketFuturesLeverage](t.Context(), ex, []byte(`{"leverage":[]}`)), "processPayload must reject an undecodable push")
}

func TestUnhandledSubject(t *testing.T) {
	t.Parallel()
	err := unhandledSubject(&WebsocketMessage{Topic: "/contract/instrument:XBTUSDTM", Subject: "new.subject"})
	require.ErrorIs(t, err, errUnhandledSubject, "unhandledSubject must return errUnhandledSubject")
	assert.ErrorContains(t, err, `"new.subject" on topic "/contract/instrument:XBTUSDTM"`, "unhandledSubject should name the subject and topic")
}

func TestMergeTicker(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	pair := currency.NewPair(currency.NewCode("MERGE"+strconv.FormatInt(time.Now().UnixNano(), 36)), currency.USDT)
	got, err := ex.mergeTicker(pair, asset.Spot, func(p *ticker.Price) { p.Bid = 1 })
	require.NoError(t, err, "mergeTicker must not error without a stored ticker")
	assert.Equal(t, &ticker.Price{ExchangeName: ex.Name, Pair: pair, AssetType: asset.Spot, Bid: 1}, got, "mergeTicker should start a new ticker when none is stored")

	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: pair, AssetType: asset.Spot, Last: 2, High: 3}), "seeding the stored ticker must not error")
	got, err = ex.mergeTicker(pair, asset.Spot, func(p *ticker.Price) { p.Bid = 1 })
	require.NoError(t, err, "mergeTicker must not error with a stored ticker")
	got.LastUpdated = time.Time{}
	assert.Equal(t, &ticker.Price{ExchangeName: ex.Name, Pair: pair, AssetType: asset.Spot, Last: 2, High: 3, Bid: 1}, got, "mergeTicker should keep the stored fields the push does not set")

	ex.Name = ""
	_, err = ex.mergeTicker(pair, asset.Spot, func(*ticker.Price) {})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet, "mergeTicker must return a store error")
}

func TestProcessMergedTicker(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	pair := currency.NewBTCUSDT()
	require.NoError(t, ex.processMergedTicker(t.Context(), pair, asset.Spot, func(p *ticker.Price) { p.Bid, p.Ask = 1, 2 }), "processMergedTicker must not error")
	got, ok := relayed(t, ex).(*ticker.Price)
	require.True(t, ok, "processMergedTicker must relay a ticker")
	stored, err := ticker.GetTicker(ex.Name, pair, asset.Spot)
	require.NoError(t, err, "processMergedTicker must store the ticker")
	assert.Equal(t, stored, got, "processMergedTicker should relay the stored ticker")

	require.ErrorIs(t, ex.processMergedTicker(t.Context(), pair, asset.Spot, func(p *ticker.Price) { p.Bid, p.Ask = 2, 2 }), ticker.ErrBidEqualsAsk, "processMergedTicker must not relay a ticker the store rejects")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processMergedTicker should not relay a rejected ticker")

	// A pair's pushes can arrive on several connections at once, so merges must not undo each other
	futuresPair := currency.NewPair(currency.XBT, currency.USDTM)
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: futuresPair, AssetType: asset.Futures}), "seeding the stored ticker must not error")
	const mergers, merges = 8, 250
	ex.Websocket.DataHandler = stream.NewRelay(mergers * merges)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range mergers {
		wg.Go(func() {
			<-start
			for range merges {
				assert.NoError(t, ex.processMergedTicker(t.Context(), futuresPair, asset.Futures, func(p *ticker.Price) { p.Last++ }), "processMergedTicker should not error")
			}
		})
	}
	close(start)
	merged := make(chan struct{})
	go func() {
		wg.Wait()
		close(merged)
	}()
	// Generous, as the merges take a fraction of a second, so that merges stuck on each other fail rather than hang
	select {
	case <-merged:
	case <-time.After(time.Minute):
		require.FailNow(t, "every merge must finish")
	}
	stored, err = ticker.GetTicker(ex.Name, futuresPair, asset.Futures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, float64(mergers*merges), stored.Last, "every concurrent merge should be kept")

	ex.Name = ""
	require.ErrorIs(t, ex.processMergedTicker(t.Context(), pair, asset.Spot, func(*ticker.Price) {}), common.ErrExchangeNameNotSet, "processMergedTicker must return a store error")
}

func TestFuturesPair(t *testing.T) {
	t.Parallel()
	pair, err := e.futuresPair("XBTUSDTM")
	require.NoError(t, err, "futuresPair must not error")
	assert.True(t, pair.Equal(currency.NewPair(currency.XBT, currency.USDTM)), "futuresPair should return the contract's pair")
	_, err = e.futuresPair("NOTACONTRACT")
	require.ErrorIs(t, err, currency.ErrPairNotFound, "futuresPair must reject an unknown symbol")
}

// TestPushData feeds every recorded frame through the router
func TestPushData(t *testing.T) {
	t.Parallel()
	ex := accountTestInstance(t)
	ex.wsOBUpdateMgr = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		Orderbook:          &ex.Websocket.Orderbook,
		CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (bool, error) { return false, nil },
		FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook: func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			return &orderbook.Book{Pair: p, Asset: a, Exchange: ex.Name, Bids: orderbook.Levels{{Amount: 1, Price: 1}}, Asks: orderbook.Levels{{Amount: 1, Price: 1e6}}, LastUpdated: time.Now()}, nil
		},
	})
	for _, file := range []string{"wsSpot.json", "wsMargin.json", "wsFutures.json"} {
		for _, key := range wsFixtureKeys(t, file) {
			assert.NoErrorf(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, file, key)), "wsHandleData should not error for %s %s", file, key)
		}
	}
}

func TestGenerateSpotSubscriptions(t *testing.T) {
	t.Parallel()
	ku := testInstance(t)
	for _, authenticated := range []bool{false, true} {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
		subs, err := ku.generateSpotSubscriptions()
		require.NoErrorf(t, err, "generateSpotSubscriptions must not error with authentication=%t", authenticated)
		require.NotEmpty(t, subs, "generateSpotSubscriptions must generate the default spot and margin subscriptions")
		for _, s := range subs {
			assert.Falsef(t, isFuturesSubscription(s), "%s should not be on the spot connection", s.QualifiedChannel)
		}
	}
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.CandlesChannel, Asset: asset.Futures, Interval: kline.SixHour}}
	_, err := ku.generateSpotSubscriptions()
	require.NoError(t, err, "generateSpotSubscriptions must not fail on a futures candle interval, which only the futures connection carries")
	ku.CurrencyPairs.Pairs[asset.Spot].RequestFormat = nil
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}}
	_, err = ku.generateSpotSubscriptions()
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "generateSpotSubscriptions must return a generation error")
}

func TestGenerateFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	ku := testInstance(t)
	for _, authenticated := range []bool{false, true} {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
		subs, err := ku.generateFuturesSubscriptions()
		require.NoErrorf(t, err, "generateFuturesSubscriptions must not error with authentication=%t", authenticated)
		require.NotEmpty(t, subs, "generateFuturesSubscriptions must generate the default futures subscriptions")
		for _, s := range subs {
			assert.Truef(t, isFuturesSubscription(s), "%s should be on the futures connection", s.QualifiedChannel)
		}
	}
	ku.CurrencyPairs.Pairs[asset.Futures].RequestFormat = nil
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Futures}}
	_, err := ku.generateFuturesSubscriptions()
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "generateFuturesSubscriptions must return a generation error")
}

func TestIsFuturesSubscription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp bool
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures}, true},
		{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}, false},
		{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin}, false},
		{&subscription.Subscription{Channel: accountBalanceChannel}, false},
		{&subscription.Subscription{Channel: futuresAccountBalanceEventChannel}, true},
		{&subscription.Subscription{Channel: futuresMarginModeChannel}, true},
	} {
		assert.Equalf(t, tc.exp, isFuturesSubscription(tc.sub), "isFuturesSubscription should be %t for %s %s", tc.exp, tc.sub.Channel, tc.sub.Asset)
	}
}

func TestAddFuturesTickerCompanions(t *testing.T) {
	t.Parallel()
	xbt := currency.NewPair(currency.XBT, currency.USDTM)
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}, QualifiedChannel: futuresTickerV2Channel + ":XBTUSDTM"}
	spotTicker := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: marketTickerChannel + ":BTC-USDT"}
	got := addFuturesTickerCompanions(subscription.List{tickerSub, spotTicker})
	exp := subscription.List{
		tickerSub,
		spotTicker,
		{Channel: futuresTickerV1Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}, QualifiedChannel: futuresTickerV1Channel + ":XBTUSDTM"},
		{Channel: futuresSnapshotChannel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}, QualifiedChannel: futuresSnapshotChannel + ":XBTUSDTM"},
	}
	testsubs.EqualLists(t, exp, got)

	snapshot := &subscription.Subscription{Channel: futuresSnapshotChannel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}, QualifiedChannel: futuresSnapshotChannel + ":XBTUSDTM"}
	got = addFuturesTickerCompanions(subscription.List{tickerSub, snapshot})
	assert.Len(t, got, 3, "a companion topic another subscription carries should not be added again")
}

// subscriptionTemplateContext is the context subscription.ExpandTemplates executes a subscription template with
type subscriptionTemplateContext struct {
	S              *subscription.Subscription
	AssetPairs     map[asset.Item]currency.Pairs
	PairSeparator  string
	AssetSeparator string
	BatchSize      string
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	btcUSDT := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", "-")
	trxBTC := currency.NewPairWithDelimiter("TRX", "BTC", "-")
	// The futures pair is given in the config format, which the topic must not use
	xbtUSDTM := currency.NewPairWithDelimiter("XBT", "USDTM", "_")
	bases := []string{"ADA", "AVAX", "BTC", "DOGE", "DOT", "ETH", "LINK", "LTC", "SOL", "TRX", "XRP"}
	elevenPairs := make(currency.Pairs, 0, len(bases))
	for _, base := range bases {
		elevenPairs = append(elevenPairs, currency.NewPairWithDelimiter(base, "USDT", "-"))
	}
	// The separators and the batch size directive are visible here; ExpandTemplates uses control characters
	for _, tc := range []struct {
		name  string
		sub   *subscription.Subscription
		pairs map[asset.Item]currency.Pairs
		exp   string
		err   error
	}{
		{
			name:  "one topic per pair",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: {ethUSDT, btcUSDT}},
			exp:   "/market/ticker:BTC-USDT|/market/ticker:ETH-USDT|#BS1;",
		},
		{
			name:  "all tickers",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: elevenPairs},
			exp:   "/market/ticker:all#BS11;",
		},
		{
			name:  "candle interval",
			sub:   &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneHour},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: {btcUSDT}},
			exp:   "/market/candles:BTC-USDT_1hour|#BS1;",
		},
		{
			name:  "futures request format",
			sub:   &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures},
			pairs: map[asset.Item]currency.Pairs{asset.Futures: {xbtUSDTM}},
			exp:   "/contractMarket/level2Depth5:XBTUSDTM|#BS1;",
		},
		{
			name:  "all assets share the spot feed",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: {btcUSDT}, asset.Margin: {trxBTC}, asset.Futures: {xbtUSDTM}},
			exp:   "/market/ticker:BTC-USDT|/market/ticker:TRX-BTC|#BS1;#BS1;/contractMarket/tickerV2:XBTUSDTM|#BS1;",
		},
		{
			name:  "channel without a symbol",
			sub:   &subscription.Subscription{Channel: accountBalanceChannel, Authenticated: true},
			pairs: map[asset.Item]currency.Pairs{asset.Empty: nil},
			exp:   "/account/balance",
		},
		{
			name:  "asset without a pair format",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Options},
			pairs: map[asset.Item]currency.Pairs{asset.Options: {btcUSDT}},
			err:   asset.ErrNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Features.Subscriptions = subscription.List{tc.sub}
			tpl, err := ku.GetSubscriptionTemplate(tc.sub)
			require.NoError(t, err, "GetSubscriptionTemplate must not error")
			require.NotNil(t, tpl, "GetSubscriptionTemplate must return a template")
			var out strings.Builder
			err = tpl.Execute(&out, &subscriptionTemplateContext{S: tc.sub, AssetPairs: tc.pairs, PairSeparator: "|", AssetSeparator: ";", BatchSize: "#BS"})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "the template must fail for pairs it cannot format")
				return
			}
			require.NoError(t, err, "the template must execute")
			assert.Equal(t, tc.exp, strings.TrimSpace(out.String()), "the template should name each topic KuCoin serves")
		})
	}
}

func TestGenerateSubscriptionsFuturesCandleInterval(t *testing.T) {
	t.Parallel()
	ku := testInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.All, Interval: kline.SixHour},
		{Channel: subscription.TickerChannel, Asset: asset.Futures},
	}
	subs, err := ku.generateFuturesSubscriptions()
	require.ErrorIs(t, err, errUnsupportedCandleInterval, "generateFuturesSubscriptions must report a candle interval the futures websocket does not push")
	require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateFuturesSubscriptions must report the dropped candles as a partial failure")
	require.NotEmpty(t, subs, "generateFuturesSubscriptions must keep the other futures subscriptions")
	for _, s := range subs {
		assert.NotEqual(t, subscription.CandlesChannel, s.Channel, "generateFuturesSubscriptions should leave out the six hour candles")
	}
	spotSubs, err := ku.generateSpotSubscriptions()
	require.NoError(t, err, "generateSpotSubscriptions must not error for six hour candles, which spot pushes")
	assert.True(t, slices.ContainsFunc(spotSubs, func(s *subscription.Subscription) bool { return strings.HasSuffix(s.QualifiedChannel, "_6hour") }), "spot six hour candles should be kept")
	pairs, err := ku.GetEnabledPairs(asset.Futures)
	require.NoError(t, err, "GetEnabledPairs must not error")
	for _, channel := range []string{subscription.CandlesChannel, futuresLimitCandleChannel} {
		ku.Features.Subscriptions = subscription.List{
			{Channel: channel, Asset: asset.Futures, Interval: kline.FourHour},
			{Channel: channel, Asset: asset.Futures, Interval: kline.SixHour},
		}
		subs, err = ku.generateFuturesSubscriptions()
		require.ErrorIsf(t, err, errUnsupportedCandleInterval, "generateFuturesSubscriptions must report six hour candles on %s", channel)
		require.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "generateFuturesSubscriptions must report six hour candles on %s as a partial failure", channel)
		require.Lenf(t, subs, len(pairs), "generateFuturesSubscriptions must keep the four hour candles of each futures pair on %s", channel)
		for _, s := range subs {
			assert.Truef(t, strings.HasPrefix(s.QualifiedChannel, futuresLimitCandleChannel+":"), "futures candles on %s should use the futures candle topic", channel)
			assert.Truef(t, strings.HasSuffix(s.QualifiedChannel, "_4hour"), "only the four hour futures candles on %s should be kept", channel)
		}
	}
}

func TestIsSymbolChannel(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{spotOrderV2Channel, spotOrderV1Channel, accountBalanceChannel, spotMarketAdvancedChannel, marginPositionChannel, futuresAnnouncementChannel, futuresTradeOrderChannel, futuresAccountBalanceEventChannel, futuresPositionAllChannel, futuresMarginModeChannel, futuresCrossLeverageChannel, futuresStopOrdersLifecycleEventChannel} {
		assert.Falsef(t, isSymbolChannel(&subscription.Subscription{Channel: channel}), "%s should not take a symbol", channel)
	}
	for _, channel := range []string{subscription.TickerChannel, isolatedMarginPositionChannel, futuresPositionChannel, futuresTickerV1Channel, callAuctionDataChannel} {
		assert.Truef(t, isSymbolChannel(&subscription.Subscription{Channel: channel}), "%s should take a symbol", channel)
	}
}

func TestChannelInterval(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1hour", channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneHour}), "spot candles should carry the interval")
	assert.Equal(t, "1hour", channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Futures, Interval: kline.OneHour}), "futures candles should carry the interval")
	assert.Empty(t, channelInterval(&subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds}), "an interval KuCoin does not name should be left out")
	assert.Empty(t, channelInterval(&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Interval: kline.OneHour}), "other channels should not carry an interval")
}

func TestSameSharedFeed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		a, b *subscription.Subscription
		exp  bool
	}{
		{name: "different channels", a: &subscription.Subscription{Channel: subscription.TickerChannel}, b: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: false},
		{name: "orderbooks of any depth and interval", a: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5, Interval: kline.HundredMilliseconds}, b: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 50, Interval: kline.OneMin}, exp: true},
		{name: "same candles on different assets", a: &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneMin}, b: &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Margin, Interval: kline.OneMin}, exp: true},
		{name: "candles of different intervals", a: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMin}, b: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FiveMin}, exp: false},
		{name: "different levels", a: &subscription.Subscription{Channel: marketOrderbookDepth5Channel, Levels: 5}, b: &subscription.Subscription{Channel: marketOrderbookDepth5Channel, Levels: 50}, exp: false},
		{name: "intervals KuCoin does not name", a: &subscription.Subscription{Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds}, b: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: true},
	} {
		assert.Equalf(t, tc.exp, sameSharedFeed(tc.a, tc.b), "sameSharedFeed should be %t for %s", tc.exp, tc.name)
	}
}

func TestJoinPairsWithInterval(t *testing.T) {
	t.Parallel()
	pairs := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-"), currency.NewPairWithDelimiter("ETH", "USDT", "-")}
	assert.Equal(t, "BTC-USDT_1hour,ETH-USDT_1hour", joinPairsWithInterval(pairs, &subscription.Subscription{Interval: kline.OneHour}), "each symbol should carry the interval KuCoin names")
	assert.Equal(t, "BTC-USDT,ETH-USDT", joinPairsWithInterval(pairs, &subscription.Subscription{Interval: kline.HundredMilliseconds}), "an interval KuCoin does not name should be left out")
	assert.Empty(t, joinPairsWithInterval(nil, &subscription.Subscription{Interval: kline.OneHour}), "no pairs should join to nothing")
}

// subscriptionTestInstance returns an instance with the enabled pairs of the test config, which the subscription tests
// are written against; the mock harness also enables the pairs its fixtures record
func subscriptionTestInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := testInstance(tb)
	if mockTests {
		require.NoError(tb, ex.CurrencyPairs.DisablePair(asset.Margin, marginTradablePair), "DisablePair must not error for margin")
		require.NoError(tb, ex.CurrencyPairs.DisablePair(asset.Futures, futuresTradablePair), "DisablePair must not error for futures")
	}
	return ex
}

func expectedPerPairSubscriptions(channel string, a asset.Item, pairs currency.Pairs, qualifiedPrefix string, interval kline.Interval, suffixFn func(currency.Pair) string) subscription.List {
	if suffixFn == nil {
		suffixFn = func(pair currency.Pair) string {
			return pair.String()
		}
	}

	resp := make(subscription.List, 0, len(pairs))
	for _, pair := range pairs {
		resp = append(resp, &subscription.Subscription{
			Channel:          channel,
			Asset:            a,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: qualifiedPrefix + ":" + suffixFn(pair),
			Interval:         interval,
		})
	}
	return resp
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)

	// Pairs overlap for spot/margin tests:
	// Only in Spot: BTC-USDT, ETH-USDT
	// In Both: ETH-BTC, LTC-USDT
	// Only in Margin: TRX-BTC, SOL-USDC
	pairs := map[string]currency.Pairs{}
	for a, ss := range map[string][]string{
		"spot":    {"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"},
		"margin":  {"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"},
		"futures": {"ETHUSDCM", "SOLUSDTM", "XBTUSDCM"},
	} {
		for _, s := range ss {
			p, err := currency.NewPairFromString(s)
			require.NoError(t, err, "NewPairFromString must not error")
			pairs[a] = pairs[a].Add(p)
		}
	}
	pairs["both"] = common.SortStrings(pairs["spot"].Add(pairs["margin"]...))

	exp := append(subscription.List{}, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Spot, pairs["both"], marketTickerChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Futures, pairs["futures"], futuresTickerV2Channel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.OrderbookChannel, asset.Spot, pairs["both"], marketOrderbookDepth5Channel, kline.HundredMilliseconds, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.OrderbookChannel, asset.Futures, pairs["futures"], futuresOrderbookDepth5Channel, kline.HundredMilliseconds, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.AllTradesChannel, asset.Spot, pairs["both"], marketMatchChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(futuresTickerV1Channel, asset.Futures, pairs["futures"], futuresTickerV1Channel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(futuresSnapshotChannel, asset.Futures, pairs["futures"], futuresSnapshotChannel, 0, nil)...)

	publicSubs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, publicSubs)

	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	exp = append(subscription.List{}, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Spot, pairs["both"], marketTickerChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Futures, pairs["futures"], futuresTickerV2Channel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(marketOrderbookChannel, asset.Spot, pairs["both"], marketOrderbookChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(futuresOrderbookChannel, asset.Futures, pairs["futures"], futuresOrderbookChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.AllTradesChannel, asset.Spot, pairs["both"], marketMatchChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(futuresTickerV1Channel, asset.Futures, pairs["futures"], futuresTickerV1Channel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(futuresSnapshotChannel, asset.Futures, pairs["futures"], futuresSnapshotChannel, 0, nil)...)
	exp = append(exp, subscription.List{
		{Asset: asset.Futures, Channel: futuresTradeOrderChannel, QualifiedChannel: "/contractMarket/tradeOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresStopOrdersLifecycleEventChannel, QualifiedChannel: "/contractMarket/advancedOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresAccountBalanceEventChannel, QualifiedChannel: "/contractAccount/wallet", Pairs: pairs["futures"]},
		{Asset: asset.Margin, Channel: marginPositionChannel, QualifiedChannel: "/margin/position", Pairs: pairs["margin"]},
		{Channel: accountBalanceChannel, QualifiedChannel: "/account/balance"},
	}...)

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)

	// UTA channels belong to the UTA connections, so the Classic list is unchanged by them
	ku.Features.Subscriptions = append(
		ku.Features.Subscriptions.Clone(),
		&subscription.Subscription{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot},
		&subscription.Subscription{Enabled: true, Channel: UTABalanceChannel, Authenticated: true},
	)
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with UTA channels configured must not error")
	testsubs.EqualLists(t, exp, subs)

	t.Run("expansion error", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}}
		ku.CurrencyPairs.Pairs[asset.Spot].RequestFormat = nil
		_, err := ku.generateSubscriptions()
		require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "generateSubscriptions must return the error expanding its templates")
	})

	t.Run("UTA generation error", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		generic := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds}
		ku.Features.Subscriptions = subscription.List{generic}
		exp, err := ku.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		ku.Features.Subscriptions = subscription.List{
			generic,
			{Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 2},
			{Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 2},
		}
		for _, connection := range []string{utaSpotPublicConnection, utaFuturesPublicConnection} {
			_, err = ku.generateUTASubscriptions(connection)
			require.ErrorIsf(t, err, subscription.ErrInvalidLevel, "generateUTASubscriptions must reject the orderbook depth for %s", connection)
		}
		subs, err := ku.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not return the errors the UTA connections report")
		testsubs.EqualLists(t, exp, subs)
	})

	t.Run("UTA partial generation", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.Features.Subscriptions = subscription.List{
			{Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds},
			{Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 5, Pairs: currency.Pairs{spotTestPair}},
			// Margin shares the spot connection, where this depth fails alone
			{Channel: UTAOrderbookChannel, Asset: asset.Margin, Levels: 2},
		}
		_, err := ku.generateUTASubscriptions(utaSpotPublicConnection)
		require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateUTASubscriptions must leave out the margin orderbook's depth as a partial failure")
		subs, err := ku.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not return the errors the UTA connections report")
		var spotBooks int
		for _, sub := range subs {
			if sub.Channel != subscription.OrderbookChannel || sub.Asset == asset.Futures {
				continue
			}
			_, symbols, _ := strings.Cut(sub.QualifiedChannel, ":")
			spotBooks++
			assert.NotContainsf(t, strings.Split(symbols, ","), "BTC-USDT", "the generic orderbook %s should leave BTC-USDT to its UTA orderbook while another UTA orderbook fails", sub.QualifiedChannel)
		}
		assert.NotZero(t, spotBooks, "the generic orderbook should still feed the other spot pairs")
	})

	// Expansion leaves a pre-expanded orderbook's asset as configured, and every book but a futures one is fed by the spot
	// realtime topic
	t.Run("pre-expanded realtime orderbook without an asset", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.All, asset.Empty} {
			ku := subscriptionTestInstance(t)
			ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ku.Features.Subscriptions = subscription.List{{
				Channel: subscription.OrderbookChannel, Asset: a, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "_")},
				QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT",
			}}
			subs, err := ku.generateSubscriptions()
			require.NoErrorf(t, err, "generateSubscriptions must not error for asset %q", a)
			require.Lenf(t, subs, 1, "generateSubscriptions must keep the orderbook for asset %q", a)
			assert.Equalf(t, marketOrderbookChannel+":BTC-USDT", subs[0].QualifiedChannel, "the orderbook for asset %q should take the spot realtime topic and spot request format", a)
		}
	})

	t.Run("pre-expanded realtime orderbook for an asset without a pair format", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
		ku.Features.Subscriptions = subscription.List{{
			Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{currency.NewBTCUSDT()},
			QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT",
		}}
		_, err := ku.generateSubscriptions()
		require.ErrorIs(t, err, asset.ErrNotSupported, "generateSubscriptions must reject a realtime orderbook it cannot spell the pairs of")
	})
}

func TestGenerateRealtimeOrderbookIntervals(t *testing.T) {
	t.Parallel()
	for assetType, topics := range map[asset.Item][]string{
		asset.Spot:    {"/market/level2:BTC-USDT", "/market/level2:ETH-BTC", "/market/level2:ETH-USDT", "/market/level2:LTC-USDT"},
		asset.Margin:  {"/market/level2:ETH-BTC", "/market/level2:LTC-USDT", "/market/level2:SOL-USDC", "/market/level2:TRX-BTC"},
		asset.Futures: {"/contractMarket/level2:ETHUSDCM", "/contractMarket/level2:SOLUSDTM", "/contractMarket/level2:XBTUSDCM"},
	} {
		for _, interval := range []kline.Interval{kline.OneMin, kline.FourHour} {
			t.Run(fmt.Sprintf("%s/%s", assetType, interval), func(t *testing.T) {
				t.Parallel()
				ku := subscriptionTestInstance(t)
				ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
				ku.Features.Subscriptions = subscription.List{{Channel: subscription.OrderbookChannel, Asset: assetType, Interval: interval, Levels: 50}}
				subs, err := ku.generateSubscriptions()
				require.NoError(t, err, "realtime subscriptions must generate")
				require.NotEmpty(t, subs, "realtime subscriptions must include enabled pairs")
				gotTopics := make([]string, 0, len(subs))
				for _, sub := range subs {
					gotTopics = append(gotTopics, sub.QualifiedChannel)
					assert.Zero(t, sub.Interval, "realtime subscriptions should not retain an interval")
					assert.Zero(t, sub.Levels, "realtime subscriptions should not retain a depth limit")
				}
				assert.ElementsMatch(t, topics, gotTopics, "realtime topics should contain the expected pair symbols without interval suffixes")
			})
		}
	}
}

func TestGenerateRealtimeOrderbooksPreservesConfiguredSubscriptions(t *testing.T) {
	t.Parallel()
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprintf("mixed=%t", mixed), func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			pair, err := currency.NewPairFromString("ETH-BTC")
			require.NoError(t, err, "pre-expanded pair must parse")
			configured := &subscription.Subscription{
				Channel: subscription.OrderbookChannel, Asset: asset.Spot,
				Pairs: currency.Pairs{pair}, Interval: kline.OneMin, Levels: 50,
				QualifiedChannel: marketOrderbookDepth5Channel + ":ETH-BTC_1min",
			}
			original := configured.Clone()
			ku.Features.Subscriptions = subscription.List{configured}
			if mixed {
				ku.Features.Subscriptions = append(ku.Features.Subscriptions, &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot})
			}
			ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
			subs, err := ku.generateSubscriptions()
			require.NoError(t, err, "mixed and qualified subscriptions must generate")
			require.NotEmpty(t, subs, "generated subscriptions must contain the orderbook")
			assert.NotSame(t, configured, subs[0], "rewritten orderbook should not alias its configuration")
			assert.Equal(t, original, configured, "generation should preserve the configured subscription")
			assert.Equal(t, "/market/level2:ETH-BTC", subs[0].QualifiedChannel, "realtime topic should omit the candle suffix")
			assert.Zero(t, subs[0].Levels, "realtime subscriptions should not retain the configured depth limit")
			ku.Websocket.SetCanUseAuthenticatedEndpoints(false)
			public, err := ku.generateSubscriptions()
			require.NoError(t, err, "public subscriptions must generate after authentication is disabled")
			require.NotEmpty(t, public, "public subscriptions must contain the orderbook")
			assert.Equal(t, original, public[0], "disabling authentication should restore the configured public subscription")
		})
	}
}

func TestGenerateRealtimeOrderbooksFormatsPreExpandedPairs(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{{
		Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()},
		QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT",
	}}
	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "pre-expanded realtime orderbook must generate")
	require.Len(t, subs, 1, "pre-expanded realtime orderbook must remain singular")
	assert.Equal(t, marketOrderbookChannel+":BTC-USDT", subs[0].QualifiedChannel, "realtime topic should retain request-formatted pair spelling")
}

func TestGenerateOrderbooksSkipsEmptyAsset(t *testing.T) {
	t.Parallel()
	for _, authenticated := range []bool{false, true} {
		ku := subscriptionTestInstance(t)
		ku.Features.Subscriptions = subscription.List{{Channel: subscription.OrderbookChannel}}
		ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
		subs, err := ku.generateSubscriptions()
		require.NoErrorf(t, err, "assetless orderbook must not fail with authentication=%t", authenticated)
		assert.Empty(t, subs, "assetless orderbook should not generate subscriptions")
	}
}

func TestGenerateRealtimeOrderbooksCoalescesNormalisedKeys(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		interval kline.Interval
		levels   int
	}{
		{name: "levels", interval: kline.HundredMilliseconds, levels: 50},
		{name: "interval", interval: kline.OneMin},
		{name: "identical", interval: kline.HundredMilliseconds},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Config.Features.Subscriptions = subscription.List{
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds},
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: test.interval, Levels: test.levels},
				{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot},
			}
			saved, err := json.Marshal(ku.Config)
			require.NoError(t, err, "configured subscriptions must serialise")
			restored := new(config.Exchange)
			require.NoError(t, json.Unmarshal(saved, restored), "configured subscriptions must deserialise")
			restarted := new(Exchange)
			restarted.SetDefaults()
			require.NoError(t, restarted.Setup(restored), "configured subscriptions must survive setup")
			for _, authenticated := range []bool{false, true} {
				restarted.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				subs, genErr := restarted.generateSubscriptions()
				require.NoError(t, genErr, "distinct configured subscriptions must generate")
				_, err = subscription.NewStoreFromList(subs)
				require.NoError(t, err, "normalised subscriptions must fit the connection store")
				var tickers int
				served := make(map[string]int)
				for _, sub := range subs {
					if sub.Channel == subscription.TickerChannel {
						tickers++
						continue
					}
					// KuCoin never serves a suffixed orderbook topic, so only unsuffixed symbols count as coverage.
					if _, symbol, _ := strings.Cut(sub.QualifiedChannel, ":"); !strings.Contains(symbol, "_") {
						served[symbol]++
					}
				}
				assert.Equal(t, 4, tickers, "unrelated ticker subscriptions should remain intact")
				assert.Lenf(t, served, 9, "every enabled pair should keep a served orderbook topic with authentication=%t", authenticated)
				for symbol, count := range served {
					assert.Equalf(t, 1, count, "%s should be served by one orderbook subscription with authentication=%t", symbol, authenticated)
				}
			}
		})
	}
}

func TestGenerateRealtimeOrderbooksCoalescesSameAssetSettings(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds},
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds, Levels: 50},
	}
	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "same-asset orderbooks that differ only in depth must coalesce")
	assert.Len(t, subs, 4, "each spot pair should be subscribed once after normalisation")
}

func TestGenerateOrderbooksRejectsConfiguredDuplicates(t *testing.T) {
	t.Parallel()
	for _, authenticated := range []bool{false, true} {
		t.Run(fmt.Sprintf("authenticated=%t", authenticated), func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Features.Subscriptions = subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
			}
			ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
			subs, err := ku.generateSubscriptions()
			if err == nil {
				_, err = subscription.NewStoreFromList(subs)
			}
			require.ErrorIs(t, err, subscription.ErrDuplicate, "pre-existing duplicate subscriptions must remain invalid")
		})
	}
}

func TestFormatSubscriptionPairs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		channel string
		asset   asset.Item
		pair    string
		want    string
		err     error
	}{
		{name: "spot", channel: subscription.OrderbookChannel, asset: asset.Spot, pair: "BTC_USDT", want: "BTC-USDT"},
		{name: "margin", channel: subscription.OrderbookChannel, asset: asset.Margin, pair: "ETH_BTC", want: "ETH-BTC"},
		{name: "futures", channel: subscription.OrderbookChannel, asset: asset.Futures, pair: "ETH_USDCM", want: "ETHUSDCM"},
		{name: "empty asset", channel: subscription.OrderbookChannel, asset: asset.Empty, pair: "BTC_USDT", want: "BTC_USDT"},
		{name: "spot ticker", channel: subscription.TickerChannel, asset: asset.Spot, pair: "BTC_USDT", want: "BTC-USDT"},
		{name: "futures ticker", channel: subscription.TickerChannel, asset: asset.Futures, pair: "ETH_USDCM", want: "ETHUSDCM"},
		{name: "asset without a pair format", channel: subscription.TickerChannel, asset: asset.Options, pair: "BTC_USDT", err: asset.ErrNotSupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			pair, err := currency.NewPairFromString(test.pair)
			require.NoError(t, err, "configured pair must parse")
			original := currency.Pairs{pair}
			pairs := map[asset.Item]currency.Pairs{test.asset: original}
			output, err := ku.formatSubscriptionPairs(&subscription.Subscription{Channel: test.channel}, pairs)
			if test.err != nil {
				require.ErrorIs(t, err, test.err, "formatSubscriptionPairs must reject an asset it has no request format for")
				return
			}
			require.NoError(t, err, "subscription pairs must format")
			assert.Empty(t, output, "formatting should not emit template content")
			assert.Equal(t, test.want, pairs[test.asset].Join(), "expanded pairs should use the request format")
			assert.Equal(t, test.pair, original.Join(), "formatting should preserve the input slice")
		})
	}
}

func TestGenerateTickerAllSub(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)
	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:11], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}}
	exp := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, QualifiedChannel: "/market/ticker:all", Pairs: avail[:11]},
	}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)
}

// TestGenerateOtherSubscriptions exercises non-default subscriptions
func TestGenerateOtherSubscriptions(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)
	spotPairs, err := ku.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	spotPairs = common.SortStrings(spotPairs)
	interval, err := IntervalToString(kline.FourHour)
	require.NoError(t, err, "IntervalToString must not error")

	subs := subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.FourHour},
		{Channel: marketSnapshotChannel, Asset: asset.Spot},
	}

	for _, s := range subs {
		ku.Features.Subscriptions = subscription.List{s}
		got, err := ku.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")

		var exp subscription.List
		switch s.Channel {
		case subscription.CandlesChannel:
			exp = expectedPerPairSubscriptions(subscription.CandlesChannel, asset.Spot, spotPairs, marketCandlesChannel, kline.FourHour, func(pair currency.Pair) string {
				return pair.String() + "_" + interval
			})
		case marketSnapshotChannel:
			exp = expectedPerPairSubscriptions(marketSnapshotChannel, asset.Spot, spotPairs, marketSnapshotChannel, 0, nil)
		default:
			t.Fatalf("unexpected test channel %s", s.Channel)
		}

		testsubs.EqualLists(t, exp, got)
	}
}

// TestGenerateMarginSubscriptions is a regression test for #1755 and ensures margin subscriptions work without spot subs
func TestGenerateMarginSubscriptions(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)

	spotAvail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error for spot pairs")
	spotAvail = common.SortStrings(spotAvail)
	marginAvail, err := ku.GetAvailablePairs(asset.Margin)
	require.NoError(t, err, "GetAvailablePairs must not error for margin pairs")
	marginAvail = common.SortStrings(marginAvail)
	require.GreaterOrEqual(t, len(marginAvail), 6, "Margin available pairs must include at least 6 pairs")
	require.GreaterOrEqual(t, len(spotAvail), 3, "Spot available pairs must include at least 3 pairs")

	err = ku.CurrencyPairs.StorePairs(asset.Margin, marginAvail[:6], true)
	require.NoError(t, err, "StorePairs must not error storing margin pairs")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, spotAvail[:3], true)
	require.NoError(t, err, "StorePairs must not error storing spot pairs")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin}}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Margin, marginAvail[:6], marketTickerChannel, 0, nil), subs)

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, err, "SetAssetEnabled must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Margin is disabled")
	require.NotEmpty(t, subs, "generateSubscriptions must return some subs")

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "SetAssetEnabled Margin must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled Futures must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Spot is disabled")
	require.NotEmpty(t, subs, "generateSubscriptions must return some subs")
}

func TestGenerateSharedSpotMarginFeeds(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
	ethBTC := currency.NewPair(currency.ETH, currency.BTC)
	dogeUSDT := currency.NewPair(currency.DOGE, currency.USDT)
	for _, test := range []struct {
		name        string
		subs        subscription.List
		disable     asset.Item
		publicCount int
		authCount   int
		threshold   bool
	}{
		{
			name: "restricted spot subscriptions with margin",
			subs: subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{ethUSDT}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{ethBTC}},
			},
			publicCount: 3,
			authCount:   3,
		},
		{
			name: "all assets with margin",
			subs: subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.All},
				{Channel: subscription.OrderbookChannel, Asset: asset.Margin},
			},
			publicCount: 9,
			authCount:   9,
		},
		{
			name: "authenticated spot filtered",
			subs: subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Authenticated: true},
				{Channel: subscription.OrderbookChannel, Asset: asset.Margin},
			},
			publicCount: 4,
			authCount:   5,
		},
		{
			name: "spot disabled",
			subs: subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{ethBTC}},
			},
			disable:     asset.Spot,
			publicCount: 1,
			authCount:   1,
		},
		{
			name: "margin disabled",
			subs: subscription.List{
				{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{dogeUSDT}},
			},
			disable:     asset.Margin,
			publicCount: 1,
			authCount:   1,
		},
		{
			name: "restricted ticker stays below all threshold",
			subs: subscription.List{
				{Channel: subscription.TickerChannel, Asset: asset.Spot},
				{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{ethBTC}},
				{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: currency.Pairs{dogeUSDT}},
			},
			publicCount: 12,
			authCount:   12,
			threshold:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			if test.disable != asset.Empty {
				require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(test.disable, false), "test asset must be disabled")
			}
			ku.Features.Subscriptions = test.subs.Clone()
			if test.threshold {
				available, err := ku.GetAvailablePairs(asset.Spot)
				require.NoError(t, err, "available spot pairs must load")
				require.GreaterOrEqual(t, len(available), 10, "test must have at least ten available spot pairs")
				ku.Features.Subscriptions[0].Pairs = available[:10]
			}
			for _, authenticated := range []bool{false, true} {
				ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				got, err := ku.generateSubscriptions()
				require.NoErrorf(t, err, "shared feeds must generate with authentication=%t", authenticated)
				wantCount := test.publicCount
				if authenticated {
					wantCount = test.authCount
				}
				assert.Len(t, got, wantCount, "shared feeds should retain the expected pair coverage")
				seen := make(map[string]bool, len(got))
				for _, sub := range got {
					assert.Falsef(t, seen[sub.QualifiedChannel], "topic %s should only be generated once", sub.QualifiedChannel)
					if test.threshold {
						assert.NotEqual(t, marketTickerChannel+":all", sub.QualifiedChannel, "restricted ticker subscriptions should not widen to ticker:all")
					}
					seen[sub.QualifiedChannel] = true
				}
				_, err = subscription.NewStoreFromList(got)
				require.NoError(t, err, "shared feeds must fit the subscription store")
			}
		})
	}
}

func TestGenerateFuturesOrderbookOverlap(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.All},
		{Channel: subscription.OrderbookChannel, Asset: asset.Futures},
	}
	for _, authenticated := range []bool{false, true} {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
		subs, err := ku.generateSubscriptions()
		require.NoErrorf(t, err, "overlapping futures orderbooks must generate with authentication=%t", authenticated)
		var symbols []string
		for _, sub := range subs {
			if sub.Asset == asset.Futures {
				_, symbol, _ := strings.Cut(sub.QualifiedChannel, ":")
				symbols = append(symbols, symbol)
			}
		}
		assert.ElementsMatchf(t, []string{"ETHUSDCM", "SOLUSDTM", "XBTUSDCM"}, symbols, "each futures pair should be subscribed once with authentication=%t", authenticated)
	}
}

func TestGenerateRealtimeOrderbooksLeavesDepthFedPairs(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds},
		{Enabled: true, Channel: futuresOrderbookDepth5Channel, Asset: asset.Futures},
		{Enabled: true, Channel: marketOrderbookDepth5Channel, Asset: asset.Spot},
		{
			Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot,
			Pairs:            currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-"), currency.NewPairWithDelimiter("DOGE", "USDT", "-")},
			QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT,DOGE-USDT",
		},
	}
	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	feeds := make(map[string][]string)
	for _, sub := range subs {
		channel, symbols, _ := strings.Cut(sub.QualifiedChannel, ":")
		for symbol := range strings.SplitSeq(symbols, ",") {
			feeds[symbol] = append(feeds[symbol], channel)
		}
	}
	for symbol, want := range map[string][]string{
		"BTC-USDT":  {marketOrderbookDepth5Channel},
		"XBTUSDCM":  {futuresOrderbookDepth5Channel},
		"SOL-USDC":  {marketOrderbookChannel},
		"DOGE-USDT": {marketOrderbookChannel},
	} {
		assert.Equalf(t, want, feeds[symbol], "%s should be fed by exactly one topic", symbol)
	}
	for symbol, channels := range feeds {
		realtime := slices.Contains(channels, marketOrderbookChannel) || slices.Contains(channels, futuresOrderbookChannel)
		depth := slices.ContainsFunc(channels, func(channel string) bool { return strings.Contains(channel, "Depth") })
		assert.Falsef(t, realtime && depth, "%s should not be fed by both a depth and a realtime topic: %v", symbol, channels)
	}
}

// TestGenerateDepthTopicsFormatConfigPairs covers a depth subscription listing a pair in the config format: the topic
// spells it in the request format, which KuCoin serves, so the realtime upgrade leaves the pair to the depth topic
func TestGenerateDepthTopicsFormatConfigPairs(t *testing.T) {
	t.Parallel()
	configFormat, err := currency.NewPairFromString("XBT_USDCM")
	require.NoError(t, err, "config-format futures pair must parse")
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds},
		{Channel: futuresOrderbookDepth5Channel, Asset: asset.Futures, Pairs: currency.Pairs{configFormat}},
	}
	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	topics := make([]string, 0, len(subs))
	for _, sub := range subs {
		topics = append(topics, sub.QualifiedChannel)
	}
	assert.Contains(t, topics, futuresOrderbookDepth5Channel+":XBTUSDCM", "the depth topic should spell the pair in the request format")
	assert.NotContains(t, topics, futuresOrderbookChannel+":XBTUSDCM", "the realtime upgrade should leave a depth fed pair to its depth topic")
	assert.Contains(t, topics, futuresOrderbookChannel+":ETHUSDCM", "other futures pairs should keep their realtime feed")
}

// TestGenerateRealtimeOrderbooksLeavesLevel1FedPairs covers Orderbook - Level 1 feeding the book of its pairs, so the
// realtime upgrade must leave those pairs to it as it does for the depth 5 and 50 channels
func TestGenerateRealtimeOrderbooksLeavesLevel1FedPairs(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds},
		{Channel: marketOrderbookDepth1Channel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-")}},
	}
	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	topics := make([]string, 0, len(subs))
	for _, sub := range subs {
		topics = append(topics, sub.QualifiedChannel)
	}
	assert.Contains(t, topics, marketOrderbookDepth1Channel+":BTC-USDT", "the level 1 topic should be generated")
	assert.NotContains(t, topics, marketOrderbookChannel+":BTC-USDT", "a level 1 fed pair should not get a realtime feed")
	assert.Contains(t, topics, marketOrderbookChannel+":ETH-USDT", "other pairs should keep their realtime feed")
}

// orderbookWriters returns the topics writing each book, keyed by market and symbol: every spot market topic loads
// both the spot and the margin book of its symbol
func orderbookWriters(t *testing.T, ex *Exchange) (writers map[string][]string, partial error) {
	t.Helper()
	writers = make(map[string][]string)
	for _, generate := range []func() (subscription.List, error){ex.generateSpotSubscriptions, ex.generateFuturesSubscriptions} {
		subs, err := generate()
		if err != nil {
			require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generating Classic subscriptions must not fail outright")
			partial = common.AppendError(partial, err)
		}
		for _, s := range subs {
			channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
			var market string
			switch channel {
			case marketOrderbookDepth1Channel, marketOrderbookDepth5Channel, marketOrderbookDepth50Channel, marketOrderbookChannel:
				market = "spot "
			case futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel, futuresOrderbookChannel:
				market = "futures "
			default:
				continue
			}
			for symbol := range strings.SplitSeq(symbols, ",") {
				writers[market+symbol] = append(writers[market+symbol], s.QualifiedChannel)
			}
		}
	}
	for _, connection := range []string{utaSpotPublicConnection, utaFuturesPublicConnection} {
		subs, err := ex.generateUTASubscriptions(connection)
		require.NoError(t, err, "generating UTA subscriptions must not error")
		for _, s := range subs {
			if s.Channel != UTAOrderbookChannel {
				continue
			}
			market := "spot "
			if s.Asset == asset.Futures {
				market = "futures "
			}
			_, symbol, _ := strings.Cut(s.QualifiedChannel, ":")
			writers[market+symbol] = append(writers[market+symbol], s.QualifiedChannel)
		}
	}
	return writers, partial
}

// TestGenerateOrderbooksOneWriterPerBook covers the generic orderbook giving way on every pair another orderbook channel
// feeds, Classic or UTA, at any depth and with or without websocket authentication, as each would load the same book
func TestGenerateOrderbooksOneWriterPerBook(t *testing.T) {
	t.Parallel()
	btc := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	xbt := currency.NewPairWithDelimiter("XBT", "USDCM", "_")
	generic := &subscription.Subscription{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds}
	for _, tc := range []struct {
		name   string
		sub    *subscription.Subscription
		book   string
		writer string
	}{
		{name: "level 1", sub: &subscription.Subscription{Enabled: true, Channel: marketOrderbookDepth1Channel, Asset: asset.Spot, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: marketOrderbookDepth1Channel + ":BTC-USDT"},
		{name: "spot depth 50", sub: &subscription.Subscription{Enabled: true, Channel: marketOrderbookDepth50Channel, Asset: asset.Spot, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: marketOrderbookDepth50Channel + ":BTC-USDT"},
		{name: "futures depth 50", sub: &subscription.Subscription{Enabled: true, Channel: futuresOrderbookDepth50Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}}, book: "futures XBTUSDCM", writer: futuresOrderbookDepth50Channel + ":XBTUSDCM"},
		{name: "UTA spot", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 500, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: "obu:BTC-USDT"},
		{name: "UTA margin", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Margin, Levels: 5}, book: "spot SOL-USDC", writer: "obu:SOL-USDC"},
		{name: "UTA futures", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 500, Pairs: currency.Pairs{xbt}}, book: "futures XBTUSDCM", writer: "obu:XBTUSDCM"},
		{name: "UTA spot depth 1", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 1, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: "obu:BTC-USDT"},
		{name: "UTA futures depth 1", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 1, Pairs: currency.Pairs{xbt}}, book: "futures XBTUSDCM", writer: "obu:XBTUSDCM"},
		{name: "UTA spot depth 5", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 5, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: "obu:BTC-USDT"},
		{name: "UTA futures depth 5", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 5, Pairs: currency.Pairs{xbt}}, book: "futures XBTUSDCM", writer: "obu:XBTUSDCM"},
		{name: "UTA spot depth 50", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 50, Pairs: currency.Pairs{btc}}, book: "spot BTC-USDT", writer: "obu:BTC-USDT"},
		{name: "UTA futures depth 50", sub: &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 50, Pairs: currency.Pairs{xbt}}, book: "futures XBTUSDCM", writer: "obu:XBTUSDCM"},
	} {
		for _, authenticated := range []bool{false, true} {
			ex := subscriptionTestInstance(t)
			ex.Features.Subscriptions = subscription.List{generic, tc.sub}
			ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
			writers, partial := orderbookWriters(t, ex)
			require.NoErrorf(t, partial, "%s with authentication %t: generating subscriptions must leave nothing out", tc.name, authenticated)
			assert.Equalf(t, enabledBooks(t, ex), slices.Sorted(maps.Keys(writers)), "%s with authentication %t: the book of every enabled pair should have a writer", tc.name, authenticated)
			assert.Equalf(t, []string{tc.writer}, writers[tc.book], "%s with authentication %t: the %s book should be written by the channel configured for it alone", tc.name, authenticated, tc.book)
			for book, topics := range writers {
				assert.Lenf(t, topics, 1, "%s with authentication %t: the %s book should have one writer, got %v", tc.name, authenticated, book, topics)
			}
		}
	}
}

// TestGenerateOrderbooksExplicitDepthGivesWayToUTA covers an explicit Classic depth channel and a UTA orderbook
// configured for one book: the depth channel's snapshots would reset the UTA orderbook's sequence, so the depth channel
// leaves the pair to the UTA orderbook, reported as a partial failure, and keeps the pairs it alone feeds
func TestGenerateOrderbooksExplicitDepthGivesWayToUTA(t *testing.T) {
	t.Parallel()
	btc, eth := currency.NewPairWithDelimiter("BTC", "USDT", "-"), currency.NewPairWithDelimiter("ETH", "USDT", "-")
	sol := currency.NewPairWithDelimiter("SOL", "USDC", "-")
	xbt := currency.NewPairWithDelimiter("XBT", "USDCM", "_")
	for _, tc := range []struct {
		name       string
		depth, uta *subscription.Subscription
		book       string
		writer     string
		kept       string
	}{
		{
			name: "spot", book: "spot BTC-USDT", writer: "obu:BTC-USDT",
			depth: &subscription.Subscription{Enabled: true, Channel: marketOrderbookDepth50Channel, Asset: asset.Spot, Pairs: currency.Pairs{btc}},
			uta:   &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 5, Pairs: currency.Pairs{btc}},
		},
		{
			name: "margin", book: "spot SOL-USDC", writer: "obu:SOL-USDC",
			depth: &subscription.Subscription{Enabled: true, Channel: marketOrderbookDepth5Channel, Asset: asset.Margin, Pairs: currency.Pairs{sol}},
			uta:   &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Margin, Levels: 5},
		},
		{
			name: "futures", book: "futures XBTUSDCM", writer: "obu:XBTUSDCM",
			depth: &subscription.Subscription{Enabled: true, Channel: futuresOrderbookDepth50Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}},
			uta:   &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Levels: 500, Pairs: currency.Pairs{xbt}},
		},
		{
			// A pre-expanded depth channel naming several pairs keeps the pair the UTA orderbook does not feed
			name: "pre-expanded spot batch", book: "spot BTC-USDT", writer: "obu:BTC-USDT", kept: marketOrderbookDepth5Channel + ":ETH-USDT",
			depth: &subscription.Subscription{Enabled: true, Channel: marketOrderbookDepth5Channel, Asset: asset.All, Pairs: currency.Pairs{btc, eth}, QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT,ETH-USDT"},
			uta:   &subscription.Subscription{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 5, Pairs: currency.Pairs{btc}},
		},
	} {
		for _, authenticated := range []bool{false, true} {
			ex := subscriptionTestInstance(t)
			ex.Features.Subscriptions = subscription.List{tc.depth, tc.uta}
			ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
			writers, partial := orderbookWriters(t, ex)
			assert.ErrorIsf(t, partial, errOrderbookFedByUTA, "%s with authentication %t: leaving the pair out should be reported", tc.name, authenticated)
			assert.Equalf(t, []string{tc.writer}, writers[tc.book], "%s with authentication %t: the %s book should be written by the UTA orderbook alone", tc.name, authenticated, tc.book)
			if tc.kept == "" {
				continue
			}
			subs, err := ex.generateSpotSubscriptions()
			require.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "%s with authentication %t: generateSpotSubscriptions must report a partial failure", tc.name, authenticated)
			var depth subscription.List
			for _, s := range subs {
				if s.Channel == tc.depth.Channel {
					depth = append(depth, s)
				}
			}
			require.Lenf(t, depth, 1, "%s with authentication %t: the depth channel must keep its other pair", tc.name, authenticated)
			assert.Equalf(t, tc.kept, depth[0].QualifiedChannel, "%s with authentication %t: the depth channel should keep the topic of its other pair", tc.name, authenticated)
			assert.Equalf(t, currency.Pairs{eth}, depth[0].Pairs, "%s with authentication %t: the depth channel should keep the pair it still feeds", tc.name, authenticated)
		}
	}
}

// TestGenerateOrderbooksPreExpandedBatch covers a pre-expanded generic orderbook naming several pairs, of which only the
// pair a UTA orderbook feeds gives way to it, with or without websocket authentication
func TestGenerateOrderbooksPreExpandedBatch(t *testing.T) {
	t.Parallel()
	btc, eth := currency.NewPairWithDelimiter("BTC", "USDT", "-"), currency.NewPairWithDelimiter("ETH", "USDT", "-")
	for _, tc := range []struct {
		authenticated bool
		exp           string
	}{
		{exp: marketOrderbookDepth5Channel + ":ETH-USDT"},
		{authenticated: true, exp: marketOrderbookChannel + ":ETH-USDT"},
	} {
		ex := subscriptionTestInstance(t)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{btc, eth}, QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT,ETH-USDT"},
			{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Levels: 5, Pairs: currency.Pairs{btc}},
		}
		subs, err := ex.generateSpotSubscriptions()
		require.NoErrorf(t, err, "generateSpotSubscriptions must not error with authentication %t", tc.authenticated)
		var books subscription.List
		for _, sub := range subs {
			if channel, _, _ := strings.Cut(sub.QualifiedChannel, ":"); channel == marketOrderbookDepth5Channel || channel == marketOrderbookChannel {
				books = append(books, sub)
			}
		}
		require.Lenf(t, books, 1, "generateSpotSubscriptions must keep the generic orderbook for its other pair with authentication %t", tc.authenticated)
		assert.Equalf(t, tc.exp, books[0].QualifiedChannel, "the generic orderbook should leave BTC-USDT to its UTA orderbook with authentication %t", tc.authenticated)
		assert.Equalf(t, currency.Pairs{eth}, books[0].Pairs, "the generic orderbook should keep the pair it still feeds with authentication %t", tc.authenticated)
	}
}

// TestGenerateSubscriptionsExplicitFuturesTicker covers an explicit tickerV2 subscription beside the generic futures
// ticker: KuCoin keeps one subscription per topic, so the generic ticker leaves the explicit one's pairs to it, keeping
// its companion topics, rather than subscribing the same topic twice, which removing either would silence
func TestGenerateSubscriptionsExplicitFuturesTicker(t *testing.T) {
	t.Parallel()
	xbt, eth := currency.NewPair(currency.XBT, currency.USDCM), currency.NewPair(currency.ETH, currency.USDCM)
	topics := func(subs subscription.List) map[string][]string {
		byTopic := make(map[string][]string, len(subs))
		for _, s := range subs {
			byTopic[s.QualifiedChannel] = append(byTopic[s.QualifiedChannel], s.Channel)
		}
		return byTopic
	}
	ex := subscriptionTestInstance(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Futures},
		{Enabled: true, Channel: futuresTickerV2Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}},
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	got := topics(subs)
	assert.Equal(t, []string{futuresTickerV2Channel}, got[futuresTickerV2Channel+":XBTUSDCM"], "the explicit tickerV2 subscription should hold its topic alone")
	assert.Equal(t, []string{futuresTickerV1Channel}, got[futuresTickerV1Channel+":XBTUSDCM"], "the generic ticker should keep its last fill companion of the pair")
	assert.Equal(t, []string{futuresSnapshotChannel}, got[futuresSnapshotChannel+":XBTUSDCM"], "the generic ticker should keep its snapshot companion of the pair")
	for _, symbol := range []string{"ETHUSDCM", "SOLUSDTM"} {
		assert.Equalf(t, []string{subscription.TickerChannel}, got[futuresTickerV2Channel+":"+symbol], "the generic ticker should keep the tickerV2 topic of %s", symbol)
	}

	// A pre-expanded generic ticker naming several symbols gives way on the explicitly fed one alone
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{xbt, eth}, QualifiedChannel: futuresTickerV2Channel + ":XBTUSDCM,ETHUSDCM"},
		{Enabled: true, Channel: futuresTickerV2Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}},
	}
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error for a pre-expanded ticker")
	var generic subscription.List
	for _, s := range subs {
		if s.Channel == subscription.TickerChannel {
			generic = append(generic, s)
		}
	}
	require.Len(t, generic, 1, "generateSubscriptions must keep the pre-expanded ticker for its other pair")
	assert.Equal(t, futuresTickerV2Channel+":ETHUSDCM", generic[0].QualifiedChannel, "the pre-expanded ticker should leave XBTUSDCM to the explicit subscription")
	assert.Equal(t, currency.Pairs{eth}, generic[0].Pairs, "the pre-expanded ticker should keep the pair it still feeds")
	assert.Equal(t, []string{futuresTickerV2Channel}, topics(subs)[futuresTickerV2Channel+":XBTUSDCM"], "the explicit tickerV2 subscription should hold its topic alone")
}

// enabledBooks returns the book of every enabled pair, keyed by market and symbol as orderbookWriters keys them
func enabledBooks(t *testing.T, ex *Exchange) []string {
	t.Helper()
	var books []string
	for _, a := range ex.GetAssetTypes(true) {
		pairs, err := ex.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
		format, err := ex.GetPairFormat(a, true)
		require.NoErrorf(t, err, "GetPairFormat must not error for %s", a)
		market := "spot "
		if a == asset.Futures {
			market = "futures "
		}
		for _, p := range pairs {
			books = append(books, market+p.Format(format).String())
		}
	}
	slices.Sort(books)
	return slices.Compact(books)
}

func TestGenerateOrderbooksAuthFlipKeepsSharedTopics(t *testing.T) {
	t.Parallel()
	generic := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds}
	depth := &subscription.Subscription{Channel: marketOrderbookDepth5Channel, Asset: asset.Spot}
	batch := &subscription.Subscription{
		Channel: marketOrderbookDepth5Channel, Asset: asset.Spot,
		Pairs:            currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-"), currency.NewPairWithDelimiter("ETH", "BTC", "-")},
		QualifiedChannel: marketOrderbookDepth5Channel + ":BTC-USDT,ETH-BTC",
	}
	candle := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.OneMin}
	topics := func(l subscription.List) map[string]bool {
		out := make(map[string]bool)
		for _, sub := range l {
			channel, symbols, _ := strings.Cut(sub.QualifiedChannel, ":")
			for symbol := range strings.SplitSeq(symbols, ",") {
				out[channel+":"+symbol] = true
			}
		}
		return out
	}
	for _, test := range []struct {
		name string
		subs subscription.List
	}{
		{name: "explicit depth", subs: subscription.List{generic, depth}},
		{name: "pre-expanded depth batch", subs: subscription.List{generic, batch}},
		{name: "candle interval entry", subs: subscription.List{generic, candle, depth}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Features.Subscriptions = test.subs.Clone()
			for _, authenticated := range []bool{false, true} {
				ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				before, err := ku.generateSubscriptions()
				require.NoErrorf(t, err, "subscriptions must generate with authentication=%t", authenticated)
				store, err := subscription.NewStoreFromList(before)
				require.NoErrorf(t, err, "subscriptions must fit the store with authentication=%t", authenticated)
				ku.Websocket.SetCanUseAuthenticatedEndpoints(!authenticated)
				after, err := ku.generateSubscriptions()
				require.NoErrorf(t, err, "subscriptions must generate with authentication=%t", !authenticated)
				added, removed := store.Diff(after)
				kept := slices.DeleteFunc(slices.Clone(after), func(sub *subscription.Subscription) bool { return slices.Contains(added, sub) })
				stays := topics(kept)
				for topic := range topics(removed) {
					assert.Falsef(t, stays[topic], "unsubscribing %s should not silence a subscription that stays", topic)
				}
			}
		})
	}
}

func TestGeneratePublicOrderbooksKeepServedTopics(t *testing.T) {
	t.Parallel()
	generic := func(a asset.Item, interval kline.Interval, pairs ...currency.Pair) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: a, Interval: interval, Pairs: pairs}
	}
	for _, test := range []struct {
		name  string
		subs  subscription.List
		topic string
	}{
		{name: "spot", subs: subscription.List{generic(asset.All, kline.HundredMilliseconds), generic(asset.Spot, kline.OneMin)}, topic: marketOrderbookDepth5Channel + ":BTC-USDT"},
		{name: "pinned spot", subs: subscription.List{generic(asset.All, kline.HundredMilliseconds), generic(asset.Spot, kline.OneMin, currency.NewBTCUSDT())}, topic: marketOrderbookDepth5Channel + ":BTC-USDT"},
		{name: "futures", subs: subscription.List{generic(asset.All, kline.HundredMilliseconds), generic(asset.Futures, kline.OneMin)}, topic: futuresOrderbookDepth5Channel + ":XBTUSDCM"},
		{name: "margin", subs: subscription.List{generic(asset.Spot, kline.HundredMilliseconds), generic(asset.Margin, kline.OneMin)}, topic: marketOrderbookDepth5Channel + ":SOL-USDC"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Features.Subscriptions = test.subs
			subs, err := ku.generateSubscriptions()
			require.NoError(t, err, "generateSubscriptions must not error")
			for _, sub := range subs {
				assert.NotContainsf(t, sub.QualifiedChannel, "_1min", "orderbook topic %s should not carry a candle suffix", sub.QualifiedChannel)
			}
			assert.True(t, slices.ContainsFunc(subs, func(sub *subscription.Subscription) bool {
				return sub.QualifiedChannel == test.topic
			}), "a candle interval should not take a pair off a topic KuCoin serves")
		})
	}
}

func TestGeneratePublicOrderbooksKeepRankAmongSuffixedEntries(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.OneMin},
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.OneMin, Levels: 5, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
	}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	topics := make(map[string]int)
	for _, sub := range subs {
		topics[sub.QualifiedChannel]++
	}
	for topic, count := range topics {
		assert.Equalf(t, 1, count, "%s should be generated once", topic)
	}
}

func TestCalculateAssets(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	for _, tc := range []struct {
		name  string
		topic string
		pair  string
		exp   []asset.Item
	}{
		{name: "futures topic", topic: futuresTickerV2Channel, pair: "XBT-USDCM", exp: []asset.Item{asset.Futures}},
		{name: "USDT margined contract", topic: marketTickerChannel, pair: "SOL-USDTM", exp: []asset.Item{asset.Futures}},
		{name: "margin topic", topic: marginPositionChannel, pair: "ETH-BTC", exp: []asset.Item{asset.Margin}},
		{name: "spot and margin pair", topic: marketTickerChannel, pair: "ETH-BTC", exp: []asset.Item{asset.Spot, asset.Margin}},
		{name: "spot pair", topic: marketMatchChannel, pair: "ETH-USDT", exp: []asset.Item{asset.Spot}},
		{name: "margin pair", topic: marketMatchChannel, pair: "TRX-BTC", exp: []asset.Item{asset.Margin}},
		{name: "pair enabled for neither", topic: marketMatchChannel, pair: "DOGE-USDT", exp: []asset.Item{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pair, err := currency.NewPairFromString(tc.pair)
			require.NoError(t, err, "pair must parse")
			got, err := ku.CalculateAssets(tc.topic, pair)
			require.NoError(t, err, "CalculateAssets must not error")
			assert.Equal(t, tc.exp, got, "CalculateAssets should return the assets the topic and pair belong to")
		})
	}

	unready := new(Exchange)
	for _, topic := range []string{futuresTickerV2Channel, marginPositionChannel} {
		_, err := unready.CalculateAssets(topic, currency.NewBTCUSDT())
		assert.ErrorIsf(t, err, currency.ErrPairManagerNotInitialised, "CalculateAssets should return the asset lookup error for %s", topic)
	}

	broken := subscriptionTestInstance(t)
	broken.CurrencyPairs.Pairs[asset.Margin] = nil
	ethBTC := currency.NewPairWithDelimiter("ETH", "BTC", "-")
	_, errLookup := broken.CurrencyPairs.IsPairEnabled(ethBTC, asset.Margin)
	require.Error(t, errLookup, "a nil margin pair store must fail the margin lookup")
	_, err := broken.CalculateAssets(marketTickerChannel, ethBTC)
	assert.Equal(t, errLookup, err, "CalculateAssets should return the margin lookup error once the spot lookup succeeds")
}

func TestCalculateAssetsToleratesUnsupportedAsset(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		asset asset.Item
		topic string
		pair  string
	}{
		{name: "futures", asset: asset.Futures, topic: futuresOrderbookChannel, pair: "XBT-USDTM"},
		{name: "margin", asset: asset.Margin, topic: marginPositionChannel, pair: "ETH-BTC"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			delete(ku.CurrencyPairs.Pairs, test.asset)
			pair, err := currency.NewPairFromString(test.pair)
			require.NoError(t, err, "pair must parse")
			assets, err := ku.CalculateAssets(test.topic, pair)
			require.NoError(t, err, "CalculateAssets must tolerate an unsupported asset")
			assert.Empty(t, assets, "an unsupported asset should not be returned")
		})
	}
}

// TestMergeMarginPairs covers sharing pairs out on KuCoin's spot and margin feeds, which serve both assets from one
// topic per pair, so each pair must go to one subscription; others are the other configured subscriptions
func TestMergeMarginPairs(t *testing.T) {
	t.Parallel()
	btcUSDT := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethBTC := currency.NewPairWithDelimiter("ETH", "BTC", "-")
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", "-")
	trxBTC := currency.NewPairWithDelimiter("TRX", "BTC", "-")
	xbtUSDCM := currency.NewPairWithDelimiter("XBT", "USDCM", "_")
	ethUSDCM := currency.NewPairWithDelimiter("ETH", "USDCM", "_")
	for _, tc := range []struct {
		name   string
		sub    *subscription.Subscription
		others subscription.List
		pairs  map[asset.Item]currency.Pairs
		exp    map[asset.Item]string
	}{
		{
			name:  "margin channel",
			sub:   &subscription.Subscription{Channel: marginPositionChannel, Asset: asset.Margin},
			pairs: map[asset.Item]currency.Pairs{asset.Margin: {trxBTC, ethBTC}},
			exp:   map[asset.Item]string{asset.Margin: "TRX-BTC,ETH-BTC"},
		},
		{
			name:  "all assets",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: {ethBTC, btcUSDT}, asset.Margin: {trxBTC, ethBTC}, asset.Futures: {xbtUSDCM}},
			exp:   map[asset.Item]string{asset.Spot: "BTC-USDT,ETH-BTC,TRX-BTC", asset.Margin: "", asset.Futures: "XBT_USDCM"},
		},
		{
			name:  "all assets without margin",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			pairs: map[asset.Item]currency.Pairs{asset.Spot: {ethUSDT, btcUSDT}},
			exp:   map[asset.Item]string{asset.Spot: "BTC-USDT,ETH-USDT"},
		},
		{
			name:  "all assets without spot",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			pairs: map[asset.Item]currency.Pairs{asset.Margin: {trxBTC, ethBTC}},
			exp:   map[asset.Item]string{asset.Margin: "ETH-BTC,TRX-BTC"},
		},
		{
			name:  "all assets without spot or margin",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			pairs: map[asset.Item]currency.Pairs{asset.Futures: {xbtUSDCM}},
			exp:   map[asset.Item]string{asset.Futures: "XBT_USDCM"},
		},
		{
			name:   "spot carries the margin feed",
			sub:    &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot},
			others: subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin}},
			pairs:  map[asset.Item]currency.Pairs{asset.Spot: {ethUSDT, btcUSDT}},
			exp:    map[asset.Item]string{asset.Spot: "BTC-USDT,ETH-BTC,ETH-USDT,LTC-USDT,SOL-USDC,TRX-BTC"},
		},
		{
			name:   "spot with its own pairs",
			sub:    &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{btcUSDT}},
			others: subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin}},
			pairs:  map[asset.Item]currency.Pairs{asset.Spot: {btcUSDT}},
			exp:    map[asset.Item]string{asset.Spot: "BTC-USDT"},
		},
		{
			name:   "margin gives way to spot",
			sub:    &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin},
			others: subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}},
			pairs:  map[asset.Item]currency.Pairs{asset.Margin: {trxBTC, ethBTC}},
			exp:    map[asset.Item]string{asset.Margin: ""},
		},
		{
			name:  "futures",
			sub:   &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures},
			pairs: map[asset.Item]currency.Pairs{asset.Futures: {xbtUSDCM, ethUSDCM}},
			exp:   map[asset.Item]string{asset.Futures: "XBT_USDCM,ETH_USDCM"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Features.Subscriptions = append(subscription.List{tc.sub}, tc.others...)
			assert.Empty(t, ku.mergeMarginPairs(tc.sub, tc.pairs), "mergeMarginPairs should add nothing to the template output")
			got := make(map[asset.Item]string, len(tc.pairs))
			for a, pairs := range tc.pairs {
				got[a] = pairs.Join()
			}
			assert.Equal(t, tc.exp, got, "mergeMarginPairs should give each pair to one subscription")
		})
	}
}

func TestMergeMarginPairsWithSpotDisabled(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{subscription.OrderbookChannel, subscription.TickerChannel, subscription.AllTradesChannel} {
		t.Run(channel, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "spot asset must be disabled")
			marginChannel := channel
			if channel == subscription.OrderbookChannel {
				marginChannel = marketOrderbookChannel
			}
			ku.Config.Features.Subscriptions = subscription.List{
				{Enabled: true, Channel: channel, Asset: asset.Spot, Interval: kline.HundredMilliseconds},
				{Enabled: true, Channel: marginChannel, Asset: asset.Margin, Interval: kline.HundredMilliseconds},
			}
			ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
			require.NoError(t, ku.checkSubscriptions(), "margin subscription migration must succeed")
			for _, authenticated := range []bool{false, true} {
				ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				subs, err := ku.generateSubscriptions()
				require.NoError(t, err, "margin-only feeds must generate")
				symbols := make([]string, 0, len(subs))
				for _, sub := range subs {
					assert.Equal(t, asset.Margin, sub.Asset, "disabled spot should not consume margin pairs")
					_, symbol, ok := strings.Cut(sub.QualifiedChannel, ":")
					require.True(t, ok, "margin topics must contain a symbol")
					symbols = append(symbols, symbol)
				}
				assert.ElementsMatch(t, []string{"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"}, symbols, "all margin pairs should remain subscribed")
			}
		})
	}
}

func TestFilterAssetFeedPairs(t *testing.T) {
	t.Parallel()
	xbtUSDCM := currency.NewPairWithDelimiter("XBT", "USDCM", "_")
	solUSDTM := currency.NewPairWithDelimiter("SOL", "USDTM", "_")
	ethUSDCM := currency.NewPairWithDelimiter("ETH", "USDCM", "_")
	generic := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}
	futures := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures}
	pinned := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{xbtUSDCM}}
	ku := subscriptionTestInstance(t)
	// Each entry after the first three would claim SOL_USDTM from the futures orderbook if it took part
	ku.Features.Subscriptions = subscription.List{
		generic, futures, pinned,
		{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{solUSDTM}, QualifiedChannel: futuresOrderbookDepth5Channel + ":SOLUSDTM"},
		{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{solUSDTM}, Authenticated: true},
		{Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{solUSDTM}},
		{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{solUSDTM}},
	}
	pairs := currency.Pairs{xbtUSDCM, solUSDTM, ethUSDCM}
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp []string
	}{
		{sub: generic, exp: []string{}},
		{sub: futures, exp: []string{"ETH_USDCM", "SOL_USDTM"}},
		{sub: pinned, exp: []string{"ETH_USDCM", "SOL_USDTM", "XBT_USDCM"}},
	} {
		got := ku.filterAssetFeedPairs(tc.sub, asset.Futures, pairs)
		assert.Equalf(t, tc.exp, got.Strings(), "filterAssetFeedPairs should leave a %s orderbook with %v the futures pairs no higher ranked orderbook claims", tc.sub.Asset, tc.sub.Pairs)
	}
	assert.Equal(t, "XBT_USDCM,SOL_USDTM,ETH_USDCM", pairs.Join(), "filterAssetFeedPairs should not change the pairs it is given")

	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.Equal(t, []string{"ETH_USDCM"}, ku.filterAssetFeedPairs(futures, asset.Futures, pairs).Strings(), "an authenticated orderbook should claim its pairs once websocket authentication is available")
}

func TestSubscriptionPairsForAsset(t *testing.T) {
	t.Parallel()
	ku := subscriptionTestInstance(t)
	pinned := currency.Pairs{currency.NewPairWithDelimiter("XBT", "USDCM", "_")}
	assert.Equal(t, pinned, ku.subscriptionPairsForAsset(&subscription.Subscription{Pairs: pinned}, asset.Futures), "a subscription's own pairs should be returned")
	assert.ElementsMatch(t, []string{"ETH_USDCM", "SOL_USDTM", "XBT_USDCM"}, ku.subscriptionPairsForAsset(&subscription.Subscription{}, asset.Futures).Strings(), "a subscription without pairs should cover the asset's enabled pairs")
	assert.Nil(t, ku.subscriptionPairsForAsset(&subscription.Subscription{}, asset.Options), "a subscription for an asset without pairs should cover none")
}

func TestFilterSharedFeedPairs(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", "-")
	spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}
	pinnedSpot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{ethUSDT}}
	margin := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin}
	all := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}
	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		spot, pinnedSpot, margin, all,
		{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-")}, Interval: kline.OneMin},
	}
	// DOGE-USDT is enabled for neither spot nor margin, so no subscription without pairs claims it
	pairs := currency.Pairs{currency.NewPairWithDelimiter("LTC", "USDT", "-"), currency.NewPairWithDelimiter("BTC", "USDT", "-"), ethUSDT, currency.NewPairWithDelimiter("DOGE", "USDT", "-")}
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  []string
	}{
		{name: "spot gives way to a spot subscription with pairs", sub: spot, exp: []string{"BTC-USDT", "DOGE-USDT", "LTC-USDT"}},
		{name: "a subscription with pairs outranks the rest", sub: pinnedSpot, exp: []string{"BTC-USDT", "DOGE-USDT", "ETH-USDT", "LTC-USDT"}},
		{name: "margin gives way to spot", sub: margin, exp: []string{"DOGE-USDT"}},
		{name: "all assets give way to every asset", sub: all, exp: []string{"DOGE-USDT"}},
	} {
		got := ku.filterSharedFeedPairs(tc.sub, pairs)
		assert.Equalf(t, tc.exp, got.Strings(), "filterSharedFeedPairs should leave the pairs no higher ranked subscription claims (%s)", tc.name)
	}
	assert.Equal(t, "LTC-USDT,BTC-USDT,ETH-USDT,DOGE-USDT", pairs.Join(), "filterSharedFeedPairs should not change the pairs it is given")
}

func TestSharedFeedCandidate(t *testing.T) {
	t.Parallel()
	reference := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}
	for _, tc := range []struct {
		name          string
		candidate     *subscription.Subscription
		disable       []asset.Item
		authenticated bool
		exp           bool
	}{
		{name: "spot", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}, exp: true},
		{name: "margin", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin}, exp: true},
		{name: "disabled margin", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin}, disable: []asset.Item{asset.Margin}, exp: false},
		{name: "all assets", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}, exp: true},
		{name: "all assets with margin alone enabled", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}, disable: []asset.Item{asset.Spot}, exp: true},
		{name: "all assets with spot and margin disabled", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}, disable: []asset.Item{asset.Spot, asset.Margin}, exp: false},
		{name: "futures", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures}, exp: false},
		{name: "another feed", candidate: &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Spot}, exp: false},
		{name: "pre-expanded", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, QualifiedChannel: marketTickerChannel + ":BTC-USDT"}, exp: false},
		{name: "authenticated without websocket authentication", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Authenticated: true}, exp: false},
		{name: "authenticated with websocket authentication", candidate: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Authenticated: true}, authenticated: true, exp: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			for _, a := range tc.disable {
				require.NoErrorf(t, ku.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
			}
			ku.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			assert.Equal(t, tc.exp, ku.sharedFeedCandidate(tc.candidate, reference), "sharedFeedCandidate should only accept a subscription that will be generated on the same shared feed")
		})
	}
}

func TestSharedFeedPairs(t *testing.T) {
	t.Parallel()
	dogeUSDT := currency.Pairs{currency.NewPairWithDelimiter("DOGE", "USDT", "-")}
	for _, tc := range []struct {
		name    string
		sub     *subscription.Subscription
		others  subscription.List
		disable asset.Item
		exp     []string
	}{
		{name: "own pairs", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: dogeUSDT}, exp: []string{"DOGE-USDT"}},
		{name: "all assets", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}, exp: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT", "SOL-USDC", "TRX-BTC"}},
		{name: "margin", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin}, exp: []string{"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"}},
		{
			name: "spot carries the margin subscriptions on its feed",
			sub:  &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot},
			others: subscription.List{
				{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: dogeUSDT},
				{Channel: subscription.AllTradesChannel, Asset: asset.Margin, Pairs: currency.Pairs{currency.NewPairWithDelimiter("XRP", "USDT", "-")}},
				{Channel: subscription.TickerChannel, Asset: asset.All, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ADA", "USDT", "-")}},
			},
			exp: []string{"BTC-USDT", "DOGE-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"},
		},
		{
			name:    "spot disabled",
			sub:     &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot},
			others:  subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: dogeUSDT}},
			disable: asset.Spot,
			exp:     []string{"DOGE-USDT"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			if tc.disable != asset.Empty {
				require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(tc.disable, false), "SetAssetEnabled must not error")
			}
			ku.Features.Subscriptions = append(subscription.List{tc.sub}, tc.others...)
			assert.ElementsMatch(t, tc.exp, ku.sharedFeedPairs(tc.sub).Strings(), "sharedFeedPairs should return the pairs the subscription carries on its feed")
		})
	}
}

func TestSharedFeedPriority(t *testing.T) {
	t.Parallel()
	btcUSDT := currency.Pairs{currency.NewBTCUSDT()}
	for _, tc := range []struct {
		name          string
		sub           *subscription.Subscription
		authenticated bool
		exp           int
	}{
		{name: "all assets", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}, exp: 0},
		{name: "one asset", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}, exp: 1},
		{name: "all assets with pairs", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All, Pairs: btcUSDT}, exp: 2},
		{name: "one asset with pairs", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: btcUSDT}, exp: 3},
		{name: "candle suffixed public orderbook", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btcUSDT, Interval: kline.OneMin}, exp: -1},
		{name: "candle interval on an authenticated orderbook", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btcUSDT, Interval: kline.OneMin}, authenticated: true, exp: 3},
		{name: "public orderbook push interval", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds}, exp: 1},
		{name: "public candles", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneMin}, exp: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			assert.Equal(t, tc.exp, ku.sharedFeedPriority(tc.sub), "sharedFeedPriority should rank pairs above assets and a topic KuCoin does not serve below every served one")
		})
	}
}

func TestSharedAssetPriority(t *testing.T) {
	t.Parallel()
	assert.Greater(t, sharedAssetPriority(asset.Spot), sharedAssetPriority(asset.Margin), "spot should outrank margin on a shared feed")
	assert.Positive(t, sharedAssetPriority(asset.Margin), "margin should outrank other assets on a shared feed")
	for _, a := range []asset.Item{asset.All, asset.Futures, asset.Empty} {
		assert.Zerof(t, sharedAssetPriority(a), "%q should have no shared feed priority", a)
	}
}

func TestPingDelay(t *testing.T) {
	t.Parallel()
	maxInterval := uint64(math.MaxInt64 / int64(time.Millisecond))
	for interval, exp := range map[uint64]time.Duration{
		0:     defaultPingDelay,
		18000: 9 * time.Second,
		1:     500 * time.Microsecond,
		// The longest interval a time.Duration holds, and those past it, which would wrap to a short or negative delay
		maxInterval:         time.Duration(maxInterval) * time.Millisecond / 2,
		maxInterval + 1:     defaultPingDelay,
		9223372036854777808: defaultPingDelay,
		math.MaxUint64:      defaultPingDelay,
	} {
		assert.Equalf(t, exp, pingDelay(interval), "pingDelay should return %s for an interval of %d milliseconds", exp, interval)
	}
}

// TestCheckSubscriptions ensures checkSubscriptions upgrades user config correctly
func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()

	ku := &Exchange{
		Config: &config.Exchange{
			Features: &config.FeaturesConfig{
				Subscriptions: subscription.List{
					{Enabled: true, Channel: "ticker"},
					{Enabled: true, Channel: "allTrades"},
					{Enabled: true, Channel: "orderbook", Interval: kline.HundredMilliseconds},
					{Enabled: true, Channel: "/contractMarket/tickerV2:%s"},
					{Enabled: true, Channel: "/contractMarket/level2Depth50:%s"},
					{Enabled: true, Channel: "/margin/fundingBook:%s", Authenticated: true},
					{Enabled: true, Channel: "/account/balance", Authenticated: true},
					{Enabled: true, Channel: "/margin/position", Authenticated: true},
					{Enabled: true, Channel: "/margin/loan:%s", Authenticated: true},
					{Enabled: true, Channel: "/contractMarket/tradeOrders", Authenticated: true},
					{Enabled: true, Channel: "/contractMarket/advancedOrders", Authenticated: true},
					{Enabled: true, Channel: "/contractAccount/wallet", Authenticated: true},
					{Enabled: true, Channel: "/contractMarket/level2", Asset: asset.Futures},
					{Enabled: true, Channel: "/market/level2", Asset: asset.Spot, Authenticated: true},
				},
			},
		},
		Features: exchange.Features{},
	}

	require.NoError(t, ku.checkSubscriptions(), "subscription migration must succeed")
	testsubs.EqualLists(t, defaultSubscriptions, ku.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, ku.Config.Features.Subscriptions)

	// UTA channels are not Classic legacy entries, so a configuration of them alone is left as it is
	uta := subscription.List{
		{Enabled: true, Channel: UTABalanceChannel, Authenticated: true},
		{Enabled: true, Channel: UTATickerChannel, Asset: asset.Spot},
	}
	ku = &Exchange{Config: &config.Exchange{Features: &config.FeaturesConfig{Subscriptions: uta.Clone()}}}
	require.NoError(t, ku.checkSubscriptions(), "checkSubscriptions must not error for UTA channels")
	testsubs.EqualLists(t, uta, ku.Config.Features.Subscriptions)
	assert.Empty(t, ku.Features.Subscriptions, "checkSubscriptions should not report an upgrade for UTA channels")
}

// TestCheckSubscriptionsKeepsExplicitFuturesChannels ensures the legacy cleanup removes only the entries the generic
// ticker and orderbook replaced, so a futures subscription to the depth-50 or tickerV2 channel keeps its feed
func TestCheckSubscriptionsKeepsExplicitFuturesChannels(t *testing.T) {
	t.Parallel()
	xbt := currency.NewPair(currency.XBT, currency.USDCM)
	explicit := subscription.List{
		{Enabled: true, Channel: futuresOrderbookDepth50Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}},
		{Enabled: true, Channel: futuresTickerV2Channel, Asset: asset.Futures, Pairs: currency.Pairs{xbt}},
	}
	ku := subscriptionTestInstance(t)
	ku.Config.Features.Subscriptions = explicit.Clone()
	ku.Features.Subscriptions = explicit.Clone()
	require.NoError(t, ku.checkSubscriptions(), "checkSubscriptions must not error for explicit futures channels")
	testsubs.EqualLists(t, explicit, ku.Config.Features.Subscriptions)

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error for explicit futures channels")
	exp := expectedPerPairSubscriptions(futuresOrderbookDepth50Channel, asset.Futures, currency.Pairs{xbt}, futuresOrderbookDepth50Channel, 0, nil)
	exp = append(exp, expectedPerPairSubscriptions(futuresTickerV2Channel, asset.Futures, currency.Pairs{xbt}, futuresTickerV2Channel, 0, nil)...)
	testsubs.EqualLists(t, exp, subs)

	// Entries on the same channels for other assets carried no futures feed, so they are still removed
	ku = &Exchange{Config: &config.Exchange{Features: &config.FeaturesConfig{Subscriptions: append(subscription.List{
		{Enabled: true, Channel: futuresOrderbookDepth50Channel + ":%s"},
		{Enabled: true, Channel: futuresTickerV2Channel, Asset: asset.All},
		{Enabled: true, Channel: futuresOrderbookDepth50Channel, Asset: asset.Spot},
	}, explicit.Clone()...)}}}
	require.NoError(t, ku.checkSubscriptions(), "checkSubscriptions must not error for legacy futures channels")
	testsubs.EqualLists(t, explicit, ku.Config.Features.Subscriptions)
	testsubs.EqualLists(t, explicit, ku.Features.Subscriptions)
}

func TestCheckSubscriptionsPreservesRealtimeOrderbooks(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		generic     *subscription.Subscription
		pinned      subscription.List
		wantAssets  []asset.Item
		wantSymbols []string
		realtime    subscription.List
		wantBooks   bool
	}{
		{name: "disabled generic", generic: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}, {Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Futures}},
		{name: "missing generic spot", realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot}},
		{name: "missing generic futures", realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Futures}},
		{name: "enabled generic", generic: &subscription.Subscription{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.HundredMilliseconds}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true},
		{name: "disabled realtime", generic: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}, realtime: subscription.List{{Channel: marketOrderbookChannel, Asset: asset.Spot}, {Channel: futuresOrderbookChannel, Asset: asset.Futures}}},
		{name: "disabled realtime without generic", realtime: subscription.List{{Channel: marketOrderbookChannel, Asset: asset.Spot}}},
		{name: "no realtime", generic: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}},
		{name: "pinned spot", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"}},
		{name: "pinned spot and futures", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}, {Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Futures}},
		{name: "pinned spot with disabled all", generic: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}, pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot}},
		{name: "pinned spot with realtime futures", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Futures}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT", "ETHUSDCM", "SOLUSDTM", "XBTUSDCM"}},
		{name: "pinned margin with realtime spot", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Margin}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Margin, asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT", "SOL-USDC", "TRX-BTC"}},
		{name: "pinned spot with realtime futures and margin", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}, {Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Margin}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}},
		{name: "pinned spot with duplicate realtime futures", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}, {Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Futures}},
		{name: "pinned spot with assetless realtime", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}},
		{name: "pinned spot with all assets realtime", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.All}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}},
		{name: "pinned spot with disabled all and realtime futures", generic: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}, pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Futures}},
		{name: "pinned margin settings with realtime spot", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Margin, Interval: kline.HundredMilliseconds, Levels: 50}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Margin, asset.Spot}},
		{name: "assetless realtime without generic", realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}},
		{name: "all assets realtime without generic", realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.All}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}},
		{name: "restricted generic spot with unrestricted legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"}},
		{name: "restricted generic margin with unrestricted legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.BTC)}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Margin}}, wantBooks: true, wantAssets: []asset.Item{asset.Margin}, wantSymbols: []string{"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"}},
		{name: "restricted legacy without generic", realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot}, wantSymbols: []string{"BTC-USDT"}},
		{name: "restricted legacy with futures generic", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}, wantBooks: true, wantAssets: []asset.Item{asset.Futures, asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETHUSDCM", "SOLUSDTM", "XBTUSDCM"}},
		{name: "restricted generic futures with unrestricted legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDCM)}}}, realtime: subscription.List{{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}}, wantBooks: true, wantAssets: []asset.Item{asset.Futures}, wantSymbols: []string{"ETHUSDCM", "SOLUSDTM", "XBTUSDCM"}},
		{name: "restricted spot with unrestricted margin legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.BTC)}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Margin}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin}, wantSymbols: []string{"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"}},
		{name: "restricted margin with unrestricted spot legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.BTC)}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}}, wantBooks: true, wantAssets: []asset.Item{asset.Margin, asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"}},
		{name: "restricted spot with unrestricted all-assets legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.BTC)}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.All}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Margin, asset.Futures}, wantSymbols: []string{"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT", "SOL-USDC", "TRX-BTC", "ETHUSDCM", "SOLUSDTM", "XBTUSDCM"}},
		{name: "generic pair union covers legacy", pinned: subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}, {Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}}}, realtime: subscription.List{{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}}}, wantBooks: true, wantAssets: []asset.Item{asset.Spot, asset.Spot}, wantSymbols: []string{"BTC-USDT", "ETH-USDT"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Config.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.All}}
			if test.generic != nil {
				ku.Config.Features.Subscriptions = append(ku.Config.Features.Subscriptions, test.generic.Clone())
			}
			ku.Config.Features.Subscriptions = append(ku.Config.Features.Subscriptions, test.realtime.Clone()...)
			ku.Config.Features.Subscriptions = append(ku.Config.Features.Subscriptions, test.pinned.Clone()...)
			ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
			require.NoError(t, ku.checkSubscriptions(), "subscription migration must succeed")
			for _, pinned := range test.pinned {
				assert.Contains(t, ku.Config.Features.Subscriptions, pinned, "existing pinned choices should remain unchanged")
			}
			for _, sub := range ku.Config.Features.Subscriptions {
				assert.NotContains(t, []string{marketOrderbookChannel, futuresOrderbookChannel}, sub.Channel, "migration should remove obsolete realtime entries")
			}
			wantAssets := test.wantAssets
			if test.wantBooks && len(wantAssets) == 0 {
				wantAssets = []asset.Item{asset.All}
			}
			saved, err := json.Marshal(ku.Config)
			require.NoError(t, err, "migrated config must serialise")
			require.NoError(t, ku.checkSubscriptions(), "repeated subscription migration must succeed")
			again, err := json.Marshal(ku.Config)
			require.NoError(t, err, "repeated migration must serialise")
			assert.Equal(t, saved, again, "migration should be idempotent")
			restored := new(config.Exchange)
			require.NoError(t, json.Unmarshal(saved, restored), "saved config must deserialise")
			restarted := new(Exchange)
			restarted.SetDefaults()
			require.NoError(t, restarted.Setup(restored), "fresh setup must accept migrated config")
			for _, instance := range []*Exchange{ku, restarted} {
				for _, authenticated := range []bool{false, true} {
					instance.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
					subs, err := instance.generateSubscriptions()
					require.NoError(t, err, "migrated subscriptions must generate before and after restart")
					generatedAssets := make(map[asset.Item]bool)
					topics := make(map[string]bool)
					var generatedSymbols []string
					for _, sub := range subs {
						if sub.Channel != subscription.OrderbookChannel && sub.Channel != marketOrderbookChannel && sub.Channel != futuresOrderbookChannel {
							continue
						}
						generatedAsset := sub.Asset
						if generatedAsset == asset.Margin {
							generatedAsset = asset.Spot
						}
						generatedAssets[generatedAsset] = true
						assert.Falsef(t, topics[sub.QualifiedChannel], "orderbook topic %s should not be duplicated", sub.QualifiedChannel)
						topics[sub.QualifiedChannel] = true
						_, symbols, ok := strings.Cut(sub.QualifiedChannel, ":")
						require.True(t, ok, "orderbook topic must contain symbols")
						generatedSymbols = append(generatedSymbols, strings.Split(symbols, ",")...)
					}
					expectedAssets := make(map[asset.Item]bool)
					for _, assetType := range wantAssets {
						switch assetType {
						case asset.All:
							expectedAssets[asset.Spot], expectedAssets[asset.Futures] = true, true
						case asset.Margin:
							expectedAssets[asset.Spot] = true
						default:
							expectedAssets[assetType] = true
						}
					}
					assert.Equal(t, expectedAssets, generatedAssets, "generated orderbooks should retain asset coverage before and after restart")
					if len(test.wantSymbols) > 0 {
						assert.ElementsMatch(t, test.wantSymbols, generatedSymbols, "generated orderbooks should retain pair coverage before and after restart")
					}
				}
			}
		})
	}
}

func TestCheckSubscriptionsMigrationEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("disabled asset", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "spot asset must be disabled")
		ku.Config.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Interval: kline.HundredMilliseconds},
			{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
		}
		ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
		require.NoError(t, ku.checkSubscriptions(), "disabled asset migration must not stop exchange setup")
	})

	t.Run("restricted all-assets coverage", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.Config.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Interval: kline.HundredMilliseconds},
			{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
		}
		ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
		require.NoError(t, ku.checkSubscriptions(), "restricted all-assets migration must succeed")
		for _, authenticated := range []bool{false, true} {
			ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
			subs, err := ku.generateSubscriptions()
			require.NoErrorf(t, err, "migrated subscriptions must generate with authentication=%t", authenticated)
			_, err = subscription.NewStoreFromList(subs)
			require.NoErrorf(t, err, "migrated subscriptions must not contain duplicate topics with authentication=%t: %v", authenticated, subs)
		}
	})

	t.Run("authenticated coverage unavailable", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.API.AuthenticatedWebsocketSupport = false
		ku.Config.Features.Subscriptions = subscription.List{
			{Enabled: true, Authenticated: true, Channel: subscription.OrderbookChannel, Asset: asset.Futures},
			{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures},
		}
		ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
		require.NoError(t, ku.checkSubscriptions(), "migration without websocket authentication must succeed")
		require.Len(t, ku.Config.Features.Subscriptions, 1, "legacy coverage must consolidate with the authenticated generic entry")
		assert.False(t, ku.Config.Features.Subscriptions[0].Authenticated, "replacement coverage should generate without authentication")
		subs, err := ku.generateSubscriptions()
		require.NoError(t, err, "replacement coverage must generate without authentication")
		require.Len(t, subs, 3, "replacement coverage must retain all enabled futures pairs")
	})

	t.Run("candle interval coverage unavailable", func(t *testing.T) {
		t.Parallel()
		for _, test := range []struct {
			name     string
			generic  *subscription.Subscription
			legacy   *subscription.Subscription
			depth    string
			realtime string
		}{
			{name: "all-assets generic with spot legacy", generic: &subscription.Subscription{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.OneMin}, legacy: &subscription.Subscription{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}, depth: marketOrderbookDepth5Channel, realtime: marketOrderbookChannel},
			{name: "all-assets generic with futures legacy", generic: &subscription.Subscription{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Interval: kline.OneMin}, legacy: &subscription.Subscription{Enabled: true, Channel: futuresOrderbookChannel, Asset: asset.Futures}, depth: futuresOrderbookDepth5Channel, realtime: futuresOrderbookChannel},
			{name: "authenticated spot generic with spot legacy", generic: &subscription.Subscription{Enabled: true, Authenticated: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.OneMin}, legacy: &subscription.Subscription{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot}, depth: marketOrderbookDepth5Channel, realtime: marketOrderbookChannel},
		} {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				// Websocket authentication can be unavailable after a migration saved with it, so every combination must
				// keep the legacy pairs on a served topic.
				for _, auth := range []struct{ migrated, running bool }{{false, false}, {true, true}, {true, false}} {
					ku := subscriptionTestInstance(t)
					ku.API.AuthenticatedWebsocketSupport = auth.migrated
					ku.Config.Features.Subscriptions = subscription.List{test.generic.Clone(), test.legacy.Clone()}
					ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
					require.NoErrorf(t, ku.checkSubscriptions(), "migration must succeed with websocket authentication=%t", auth.migrated)
					ku.Websocket.SetCanUseAuthenticatedEndpoints(auth.running)
					subs, err := ku.generateSubscriptions()
					require.NoErrorf(t, err, "migrated subscriptions must generate with authentication=%t", auth.running)
					channel := test.depth
					if auth.running {
						channel = test.realtime
					}
					pairs, err := ku.GetEnabledPairs(test.legacy.Asset)
					require.NoError(t, err, "enabled pairs must load")
					format, err := ku.GetPairFormat(test.legacy.Asset, true)
					require.NoError(t, err, "request format must load")
					for _, pair := range pairs.Format(format) {
						assert.Truef(t, slices.ContainsFunc(subs, func(sub *subscription.Subscription) bool {
							return sub.QualifiedChannel == channel+":"+pair.String()
						}), "%s should keep a served orderbook topic when migrated with websocket authentication=%t and running with %t", pair, auth.migrated, auth.running)
					}
				}
			})
		}
	})

	t.Run("wildcard remains dynamic", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		ku.Config.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
			{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
		}
		ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
		require.NoError(t, ku.checkSubscriptions(), "wildcard migration must succeed")
		assert.Contains(t, ku.Config.Features.Subscriptions, &subscription.Subscription{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds}, "unrestricted legacy coverage should remain a wildcard")
		pairs, err := ku.GetEnabledPairs(asset.Spot)
		require.NoError(t, err, "enabled spot pairs must load")
		dogeUSDT := currency.NewPair(currency.DOGE, currency.USDT)
		require.NoError(t, ku.CurrencyPairs.StorePairs(asset.Spot, pairs.Add(dogeUSDT), true), "new spot pair must be enabled")
		subs, err := ku.generateSubscriptions()
		require.NoError(t, err, "widened wildcard must generate")
		assert.True(t, slices.ContainsFunc(subs, func(sub *subscription.Subscription) bool {
			return sub.QualifiedChannel == marketOrderbookDepth5Channel+":DOGE-USDT"
		}), "widened wildcard should track newly enabled pairs")
	})

	t.Run("legacy asset without a pair format", func(t *testing.T) {
		t.Parallel()
		ku := subscriptionTestInstance(t)
		// The generic entry covers some of the legacy pairs, which are compared in the legacy asset's request format
		ku.Config.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
			{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}},
		}
		ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
		require.ErrorIs(t, ku.checkSubscriptions(), asset.ErrNotSupported, "migration must fail when it cannot compare a legacy orderbook's pairs")
	})
}

func TestCheckSubscriptionsOverlappingCoverage(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
	for _, test := range []struct {
		name        string
		subs        subscription.List
		publicCount int
		authCount   int
	}{
		{
			name: "multiple restricted spot subscriptions",
			subs: subscription.List{
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{ethUSDT}},
				{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
			},
			publicCount: 4,
			authCount:   4,
		},
		{
			name: "authenticated restricted coverage unavailable",
			subs: subscription.List{
				{Enabled: true, Authenticated: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{ethUSDT}},
				{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
			},
			publicCount: 4,
			authCount:   4,
		},
		{
			name: "authenticated all-assets coverage unavailable",
			subs: subscription.List{
				{Enabled: true, Authenticated: true, Channel: subscription.OrderbookChannel, Asset: asset.All},
				{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
			},
			publicCount: 4,
			authCount:   9,
		},
		{
			name: "disabled authenticated all-assets fallback",
			subs: subscription.List{
				{Authenticated: true, Channel: subscription.OrderbookChannel, Asset: asset.All},
				{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Spot},
			},
			publicCount: 4,
			authCount:   4,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.API.AuthenticatedWebsocketSupport = false
			ku.Config.Features.Subscriptions = test.subs.Clone()
			ku.Features.Subscriptions = ku.Config.Features.Subscriptions.Enabled()
			require.NoError(t, ku.checkSubscriptions(), "overlapping migration must succeed")
			for _, authenticated := range []bool{false, true} {
				ku.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				got, err := ku.generateSubscriptions()
				require.NoErrorf(t, err, "migrated coverage must generate with authentication=%t", authenticated)
				wantCount := test.publicCount
				if authenticated {
					wantCount = test.authCount
				}
				assert.Len(t, got, wantCount, "migration should retain all covered pairs")
				seen := make(map[string]bool, len(got))
				for _, sub := range got {
					assert.Falsef(t, seen[sub.QualifiedChannel], "topic %s should only be generated once", sub.QualifiedChannel)
					seen[sub.QualifiedChannel] = true
				}
				_, err = subscription.NewStoreFromList(got)
				require.NoError(t, err, "migrated coverage must fit the subscription store")
			}
		})
	}
}

// TestSubscribeBatches ensures that endpoints support batching, contrary to kucoin api docs
func TestSubscribeBatches(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)
	ku.Features.Subscriptions = subscription.List{
		{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Asset: asset.Futures, Channel: subscription.TickerChannel},
		{Asset: asset.Spot, Channel: marketSnapshotChannel},
	}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	spotPairs, err := ku.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error for spot pairs")
	futuresPairs, err := ku.GetEnabledPairs(asset.Futures)
	require.NoError(t, err, "GetEnabledPairs must not error for futures pairs")
	// A futures ticker carries its Ticker V1 and snapshot companions
	require.Len(t, subs, len(spotPairs)+3*len(futuresPairs)+len(spotPairs), "generateSubscriptions must return one subscription per enabled pair and topic")

	conn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
	err = ku.Subscribe(t.Context(), conn, subs)
	require.NoError(t, err, "Subscribe must not error for small batches")

	expectedOutbound := collapseSubscriptionList(subs)
	require.Len(t, conn.sentRequests, len(expectedOutbound), "Subscribe must send one request per collapsed batch")
	expectedTopics := make(map[string]struct{}, len(expectedOutbound))
	for _, outbound := range expectedOutbound {
		expectedTopics[outbound.QualifiedChannel] = struct{}{}
	}
	for _, req := range conn.sentRequests {
		_, ok := expectedTopics[req.Topic]
		assert.Truef(t, ok, "Request topic should match a collapsed subscription: %s", req.Topic)
	}
	assert.Len(t, ku.Websocket.GetSubscriptions(), len(subs), "Subscribe should track each original subscription")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		ch  string
		exp string
	}{
		{asset.Futures, futuresOrderbookDepth50Channel, futuresOrderbookDepth50Channel},
		{asset.Futures, subscription.OrderbookChannel, futuresOrderbookDepth5Channel},
		{asset.Futures, subscription.CandlesChannel, futuresLimitCandleChannel},
		{asset.Futures, subscription.TickerChannel, futuresTickerV2Channel},
		{asset.Futures, subscription.AllTradesChannel, futuresExecutionChannel},
		{asset.Spot, subscription.OrderbookChannel, marketOrderbookDepth5Channel},
		{asset.Spot, subscription.AllTradesChannel, marketMatchChannel},
		{asset.Spot, subscription.CandlesChannel, marketCandlesChannel},
		{asset.Spot, subscription.TickerChannel, marketTickerChannel},
	} {
		assert.Equal(t, tt.exp, channelName(&subscription.Subscription{Channel: tt.ch}, tt.a))
	}
}

// TestSubscribeBatchLimit exercises the kucoin batch limits of 400 per connection
// Ensures batching of 100 pairs and the connection symbol limit is still 400 at Kucoin's end
func TestSubscribeBatchLimit(t *testing.T) {
	t.Parallel()

	const expectedLimit = 400

	setupSubs := func(tb testing.TB, total int) (*Exchange, subscription.List) {
		tb.Helper()
		ku := subscriptionTestInstance(tb)
		avail, err := ku.GetAvailablePairs(asset.Spot)
		require.NoError(tb, err, "GetAvailablePairs must not error")
		require.GreaterOrEqual(tb, len(avail), total, "GetAvailablePairs must include enough spot pairs")

		err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:total], true)
		require.NoError(tb, err, "StorePairs must not error")

		ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.AllTradesChannel}}
		subs, err := ku.generateSubscriptions()
		require.NoError(tb, err, "generateSubscriptions must not error")
		return ku, subs
	}

	t.Run("within_limit", func(t *testing.T) {
		t.Parallel()

		ku, subs := setupSubs(t, expectedLimit)
		require.Len(t, subs, expectedLimit, "generateSubscriptions must return one subscription per enabled pair")

		conn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
		err := ku.Subscribe(t.Context(), conn, subs)
		require.NoError(t, err, "Subscribe must not error at KuCoin's session limit")
		require.Len(t, conn.sentRequests, 4, "Subscribe must send four batched requests at KuCoin's session limit")
		assert.Len(t, ku.Websocket.GetSubscriptions(), expectedLimit, "Subscribe should track each original subscription at KuCoin's session limit")

		unsubConn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
		err = ku.Unsubscribe(t.Context(), unsubConn, subs)
		require.NoError(t, err, "Unsubscribe must not error at KuCoin's session limit")
		require.Len(t, unsubConn.sentRequests, 4, "Unsubscribe must send four batched requests at KuCoin's session limit")
		assert.Empty(t, ku.Websocket.GetSubscriptions(), "Unsubscribe should remove all tracked subscriptions")
	})

	t.Run("over_limit", func(t *testing.T) {
		t.Parallel()

		ku, subs := setupSubs(t, expectedLimit+20)
		require.Len(t, subs, expectedLimit+20, "generateSubscriptions must return one subscription per enabled pair")
		expectedOutbound := collapseSubscriptionList(subs)
		batchSizes := make(map[string]int, len(expectedOutbound))
		for assoc, outbound := range expectedOutbound {
			batchSizes[outbound.QualifiedChannel] = len(*assoc)
		}

		conn := &ConnectionFixture{}
		conn.sendFn = func(_ any) ([]byte, error) {
			if len(conn.sentRequests) == 5 {
				return []byte(`{"id":"019ae22f-4718-7da4-846d-999b085cc24a","type":"error","code":509,"data":"exceed max subscription count limitation of 400 per session"}`), nil
			}
			return []byte(`{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`), nil
		}

		err := ku.Subscribe(t.Context(), conn, subs)
		require.Error(t, err, "Subscribe must error once KuCoin's session limit is exceeded")
		require.Len(t, conn.sentRequests, 5, "Subscribe must send five batched requests when more than 400 pairs are enabled")
		assert.ErrorContains(t, err, "exceed max subscription count limitation of 400 per session", "Subscribe should return KuCoin's session limit error")
		failedBatchSize := batchSizes[conn.sentRequests[len(conn.sentRequests)-1].Topic]
		assert.Len(t, ku.Websocket.GetSubscriptions(), len(subs)-failedBatchSize, "Only acknowledged subscriptions should be tracked after KuCoin rejects one batch")
	})
}

// TestSubscribeTickerAll ensures that ticker subscriptions switch to using all and it works
func TestSubscribeTickerAll(t *testing.T) {
	t.Parallel()

	ku := subscriptionTestInstance(t)
	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")

	require.GreaterOrEqual(t, len(avail), 500)
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:500], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel}}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "Must generate one subscription")
	assert.Equal(t, "/market/ticker:all", subs[0].QualifiedChannel, "QualifiedChannel should be correct")

	err = ku.Subscribe(t.Context(), &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}, subs)
	assert.NoError(t, err, "Subscribe to should not error")
}

func TestManageSubscriptions(t *testing.T) {
	t.Parallel()

	baseSub := func() *subscription.Subscription {
		pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
		return &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: marketTickerChannel + ":" + pair.String(),
		}
	}

	t.Run("error_paths", func(t *testing.T) {
		t.Parallel()
		t.Run("send_message", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			errExpected := errors.New("send failure")

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageError: errExpected}, subscription.List{baseSub()}, "subscribe")
			require.ErrorIs(t, err, errExpected)
		})

		t.Run("invalid_json", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":`}, subscription.List{baseSub()}, "subscribe")
			require.Error(t, err)
		})

		t.Run("null_reply", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `null`}, subscription.List{baseSub()}, "subscribe")
			require.ErrorIs(t, err, common.ErrNoResponse)
		})

		t.Run("error_payload_not_string", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"error","code":509,"data":{}}`}, subscription.List{baseSub()}, "subscribe")
			require.Error(t, err)
			assert.ErrorContains(t, err, "unknown error (509)")
		})

		t.Run("unexpected_message_type", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"mystery","code":1}`}, subscription.List{baseSub()}, "subscribe")
			require.ErrorIs(t, err, errInvalidMsgType)
			assert.ErrorContains(t, err, "mystery")
		})

		t.Run("subscribe_ack_add_error", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)
			ku.Verbose = true
			sub := baseSub()
			require.NoError(t, sub.SetState(subscription.SubscribedState), "SetState must not error")

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"ack"}`}, subscription.List{sub}, "subscribe")
			require.ErrorIs(t, err, subscription.ErrInStateAlready)
		})

		t.Run("unsubscribe_ack_remove_error", func(t *testing.T) {
			t.Parallel()
			ku := subscriptionTestInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"ack"}`}, subscription.List{baseSub()}, "unsubscribe")
			require.ErrorIs(t, err, subscription.ErrNotFound)
		})
	})
}

func TestCollapseSubscriptionList(t *testing.T) {
	t.Parallel()

	const totalSubscriptions = 101

	marketSubs := make(subscription.List, 0, totalSubscriptions)
	expectedPairs := make(currency.Pairs, 0, totalSubscriptions)
	expectedSuffixes := make([]string, 0, totalSubscriptions)
	for i := range totalSubscriptions {
		base := fmt.Sprintf("COIN%03d", i)
		suffix := base + "-USDT"
		pair := currency.NewPairWithDelimiter(base, "USDT", "-")
		marketSubs = append(marketSubs, &subscription.Subscription{
			Channel:          subscription.AllTradesChannel,
			Asset:            asset.Spot,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: marketMatchChannel + ":" + suffix,
		})
		expectedPairs = append(expectedPairs, pair)
		expectedSuffixes = append(expectedSuffixes, suffix)
	}

	accountSub := &subscription.Subscription{
		Channel:          accountBalanceChannel,
		Authenticated:    true,
		QualifiedChannel: accountBalanceChannel,
	}

	subs := append(subscription.List{}, marketSubs...)
	subs = append(subs, accountSub)

	collapsed := collapseSubscriptionList(subs)
	require.Len(t, collapsed, 3, "collapseSubscriptionList must create three collapsed batches")

	type batchResult struct {
		assoc *subscription.List
		sub   *subscription.Subscription
	}

	var hundredBatch *batchResult
	var singleBatch *batchResult
	var accountBatch *batchResult

	for assoc, sub := range collapsed {
		switch {
		case sub.QualifiedChannel == accountBalanceChannel:
			accountBatch = &batchResult{assoc: assoc, sub: sub}
		case strings.HasPrefix(sub.QualifiedChannel, marketMatchChannel+":"):
			switch len(*assoc) {
			case 100:
				hundredBatch = &batchResult{assoc: assoc, sub: sub}
			case 1:
				singleBatch = &batchResult{assoc: assoc, sub: sub}
			default:
				t.Fatalf("unexpected market batch size: %d", len(*assoc))
			}
		default:
			t.Fatalf("unexpected collapsed channel: %s", sub.QualifiedChannel)
		}
	}

	require.NotNil(t, hundredBatch, "the 100-item market batch must be present")
	require.NotNil(t, singleBatch, "the single-item market batch must be present")
	require.NotNil(t, accountBatch, "the pairless account batch must be present")

	assertCollapsedBatch(t, marketSubs[:100], expectedPairs[:100], expectedSuffixes[:100], hundredBatch.assoc, hundredBatch.sub)
	assertCollapsedBatch(t, marketSubs[100:], expectedPairs[100:], expectedSuffixes[100:], singleBatch.assoc, singleBatch.sub)

	require.Len(t, *accountBatch.assoc, 1, "the pairless account batch must preserve one original subscription")
	assert.Same(t, accountSub, (*accountBatch.assoc)[0], "the pairless account batch should preserve the original subscription pointer")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.Channel, "the pairless account batch should preserve the original channel")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.QualifiedChannel, "the pairless account batch should preserve the qualified channel")
	assert.Empty(t, accountBatch.sub.Pairs, "the pairless account batch should not gain pairs")
	assert.True(t, accountBatch.sub.Authenticated, "the pairless account batch should preserve authentication state")

	assert.Len(t, marketSubs[0].Pairs, 1, "the source subscription should remain unchanged after collapsing")
	assert.Equal(t, marketMatchChannel+":"+expectedSuffixes[0], marketSubs[0].QualifiedChannel, "the source subscription should keep its original qualified channel")
}

func assertCollapsedBatch(t *testing.T, expectedOriginal subscription.List, expectedPairs currency.Pairs, expectedSuffixes []string, assoc *subscription.List, got *subscription.Subscription) {
	t.Helper()

	require.NotNil(t, assoc, "the associated subscription list must not be nil")
	require.NotNil(t, got, "the collapsed subscription must not be nil")
	require.Len(t, *assoc, len(expectedOriginal), "the associated subscription list must preserve the original subscriptions")

	for i := range expectedOriginal {
		assert.Samef(t, expectedOriginal[i], (*assoc)[i], "associated subscription %d should match the original pointer", i)
	}

	assert.Equal(t, subscription.AllTradesChannel, got.Channel, "the collapsed subscription should preserve the channel")
	assert.Equal(t, asset.Spot, got.Asset, "the collapsed subscription should preserve the asset")
	assert.Equal(t, expectedPairs, got.Pairs, "the collapsed subscription should merge pairs in order")
	assert.Equal(t, marketMatchChannel+":"+strings.Join(expectedSuffixes, ","), got.QualifiedChannel, "the collapsed subscription should join the qualified channel suffixes")
	assert.False(t, got.Authenticated, "the collapsed market subscription should remain public")
}

package bingx

import (
	"bufio"
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
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
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const testListenKey = "test-listen-key"

// testAccountRESTPaths are the REST endpoints serving each asset's account connections
var testAccountRESTPaths = map[asset.Item]exchange.URL{
	asset.Spot:                exchange.RestSpot,
	asset.USDTMarginedFutures: exchange.RestUSDTMargined,
	asset.CoinMarginedFutures: exchange.RestCoinMargined,
}

var (
	testSpotPair         = currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter)
	testSwapPair         = currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter)
	testCoinMarginedPair = currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	errTestDial          = errors.New("test dial failure")
)

// newWebsocketTestExchange returns an exchange named after the test, set up through the real Setup from the test
// configuration as configure changes it
func newWebsocketTestExchange(tb testing.TB, configure ...func(*config.Exchange)) *Exchange {
	tb.Helper()
	root, err := testutils.RootPathFromCWD()
	require.NoError(tb, err, "RootPathFromCWD must not error")
	cfg := &config.Config{}
	require.NoError(tb, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	ex := new(Exchange)
	ex.SetDefaults()
	exchConf, err := cfg.GetExchangeConfig(ex.Name)
	require.NoError(tb, err, "GetExchangeConfig must not error")
	for _, f := range configure {
		f(exchConf)
	}
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(tb, ex.Setup(exchConf), "Setup must not error")
	ex.Accounts = accounts.MustNewAccounts(ex)
	ex.Name = tb.Name()
	return ex
}

// testWebsocketAuthenticated turns authenticated websocket use on with placeholder credentials
func testWebsocketAuthenticated(cfg *config.Exchange) {
	cfg.API.AuthenticatedSupport = true
	cfg.API.AuthenticatedWebsocketSupport = true
	cfg.API.Credentials.Key = "test-api-key"
	cfg.API.Credentials.Secret = "test-api-secret"
}

// testWebsocketAssets enables the assets given and disables the others
func testWebsocketAssets(assets ...asset.Item) func(*config.Exchange) {
	return func(cfg *config.Exchange) {
		for a, ps := range cfg.CurrencyPairs.Pairs {
			ps.AssetEnabled = slices.Contains(assets, a)
		}
	}
}

// testWebsocketSubscriptions configures the subscriptions given
func testWebsocketSubscriptions(subs ...*subscription.Subscription) func(*config.Exchange) {
	return func(cfg *config.Exchange) {
		cfg.Features.Subscriptions = subs
	}
}

// testWebsocketPairs makes the pairs given an asset's available and enabled pairs
func testWebsocketPairs(a asset.Item, pairs currency.Pairs) func(*config.Exchange) {
	return func(cfg *config.Exchange) {
		cfg.CurrencyPairs.Pairs[a].Available = pairs
		cfg.CurrencyPairs.Pairs[a].Enabled = pairs
	}
}

// testWebsocketConnection is a connection without a socket, recording what is sent on it, with the budget each send drew
// on, and the replies routed to it
type testWebsocketConnection struct {
	websocket.Connection
	mu       sync.Mutex
	sent     []testWebsocketMessage
	matched  []any
	matchErr error
	reply    func(payload []byte) ([]byte, error)
	down     atomic.Bool
}

// testWebsocketMessage is a message sent on a testWebsocketConnection
type testWebsocketMessage struct {
	limit   request.EndpointLimit
	message string
}

func (c *testWebsocketConnection) SendRawMessage(_ context.Context, epl request.EndpointLimit, _ int, message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, testWebsocketMessage{limit: epl, message: string(message)})
	return nil
}

func (c *testWebsocketConnection) SendMessageReturnResponse(_ context.Context, epl request.EndpointLimit, _, payload any) ([]byte, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.sent = append(c.sent, testWebsocketMessage{limit: epl, message: string(encoded)})
	reply := c.reply
	c.mu.Unlock()
	if reply == nil {
		return nil, errTestDial
	}
	return reply(encoded)
}

func (c *testWebsocketConnection) RequireMatchWithData(signature any, _ []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.matched = append(c.matched, signature)
	return c.matchErr
}

func (c *testWebsocketConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return errTestDial
}

func (c *testWebsocketConnection) IsConnected() bool {
	return !c.down.Load()
}

func (c *testWebsocketConnection) GetURL() string {
	return "wss://bingx.invalid/market"
}

func (c *testWebsocketConnection) Shutdown() error {
	c.down.Store(true)
	return nil
}

func (c *testWebsocketConnection) messages() []testWebsocketMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.sent)
}

func (c *testWebsocketConnection) matches() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.matched)
}

// testSession is a websocket session a test ends
type testSession struct {
	ch   chan struct{}
	once sync.Once
}

// end tears the session down, once
func (s *testSession) end() {
	s.once.Do(func() { close(s.ch) })
}

// token returns the session as the exchange holds it
func (s *testSession) token() <-chan struct{} {
	return s.ch
}

// receiveOnly returns a channel as the exchange holds a session
func receiveOnly(ch chan struct{}) <-chan struct{} {
	return ch
}

// newTestSession returns a session ended, with its watcher and keepers stopped, when the test ends
func newTestSession(tb testing.TB, ex *Exchange) *testSession {
	tb.Helper()
	s := &testSession{ch: make(chan struct{})}
	tb.Cleanup(func() {
		s.end()
		waitForSessionEnd(tb, ex, s.ch)
	})
	return s
}

// waitForSessionEnd waits for the watcher and the keepers of a torn down session to stop
func waitForSessionEnd(tb testing.TB, ex *Exchange, session <-chan struct{}) {
	tb.Helper()
	assert.Eventually(tb, func() bool {
		ex.wsState.mu.Lock()
		defer ex.wsState.mu.Unlock()
		if ex.wsState.sessions[session] {
			return false
		}
		for _, k := range ex.wsState.keepers {
			if k.session == session {
				return false
			}
		}
		return true
	}, 5*time.Second, time.Millisecond, "the session's watcher and keepers should stop")
}

// trackTestConnection tracks a connection as a Connector does, in a session ended with the test
func trackTestConnection(tb testing.TB, ex *Exchange, conn websocket.Connection, serialRequests bool) (*websocketConnection, *testSession) {
	tb.Helper()
	session := newTestSession(tb, ex)
	return ex.wsTrackConnection(conn, session.ch, serialRequests, asset.Empty, nil), session
}

// newWebsocketMockConnection dials a connection of the setup filter names to a mock server answering with mock. It is
// tracked as a Connector tracks it and read by handler until the test ends
func newWebsocketMockConnection(tb testing.TB, ex *Exchange, filter string, serialRequests bool, handler func(context.Context, websocket.Connection, []byte) error, mock mockws.WsMockFunc) (websocket.Connection, *testSession) {
	tb.Helper()
	s := httptest.NewTestServer(tb, mockws.CurryWsMockUpgrader(tb, mock))
	s.Start()
	require.NoError(tb, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	conn, err := ex.Websocket.CreateTestConnection(filter)
	require.NoError(tb, err, "CreateTestConnection must not error")
	require.NoError(tb, conn.Dial(tb.Context(), &gws.Dialer{}, http.Header{}, nil), "Dial must not error")
	session := &testSession{ch: make(chan struct{})}
	ex.wsTrackConnection(conn, session.ch, serialRequests, asset.Empty, nil)
	read := make(chan struct{})
	ex.Websocket.Wg.Add(1)
	go func() {
		defer close(read)
		ex.Websocket.Reader(context.Background(), conn, handler)
	}()
	tb.Cleanup(func() {
		assert.NoError(tb, conn.Shutdown(), "Shutdown should not error")
		select {
		case <-read:
		case <-time.After(5 * time.Second):
			assert.Fail(tb, "the reader should stop once the connection is closed")
		}
		session.end()
		waitForSessionEnd(tb, ex, session.ch)
	})
	return conn, session
}

// testWebsocketRequest decodes a subscription request a mock server received
func testWebsocketRequest(tb testing.TB, msg []byte) *WebsocketSubscriptionRequest {
	tb.Helper()
	var req *WebsocketSubscriptionRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	return req
}

// testWebsocketAck returns the reply to a request in the shape of a product's servers
func testWebsocketAck(a asset.Item, id string, code int64, msg string) []byte {
	switch {
	case a == asset.USDTMarginedFutures:
		return fmt.Appendf(nil, `{"id":%q,"code":%d,"msg":%q,"dataType":"","data":null}`, id, code, msg)
	case a == asset.CoinMarginedFutures && code != 0:
		return fmt.Appendf(nil, `{"code":%d,"msg":%q,"timestamp":1794390798083}`, code, msg)
	default:
		return fmt.Appendf(nil, `{"code":%d,"id":%q,"msg":%q,"timestamp":1794390797881}`, code, id, msg)
	}
}

// testGzip compresses a frame as BingX sends them; writing to memory cannot fail
func testGzip(frame []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(frame)
	_ = w.Close()
	return b.Bytes()
}

// fixtureLines returns the lines of a fixture
func fixtureLines(tb testing.TB, path string) [][]byte {
	tb.Helper()
	f, err := os.Open(path)
	require.NoErrorf(tb, err, "opening %s must not error", path)
	defer func() { assert.NoError(tb, f.Close(), "closing the fixture should not error") }()
	var lines [][]byte
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		lines = append(lines, slices.Clone(s.Bytes()))
	}
	require.NoError(tb, s.Err(), "scanning the fixture must not error")
	return lines
}

// fixtureLine returns the first line of a fixture containing substr
func fixtureLine(tb testing.TB, path, substr string) []byte {
	tb.Helper()
	for _, line := range fixtureLines(tb, path) {
		if bytes.Contains(line, []byte(substr)) {
			return line
		}
	}
	require.FailNowf(tb, "fixture line must exist", "%s has no line containing %q", path, substr)
	return nil
}

// websocketPayloads drains what has been relayed so far
func websocketPayloads(ex *Exchange) []any {
	var got []any
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			got = append(got, p.Data)
		default:
			return got
		}
	}
}

// waitForWebsocketPayload returns the first payload of type T relayed that match accepts, failing the test after five
// seconds
func waitForWebsocketPayload[T any](tb testing.TB, ex *Exchange, match func(T) bool) T {
	tb.Helper()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			if v, ok := p.Data.(T); ok && (match == nil || match(v)) {
				return v
			}
		case <-timeout.C:
			var zero T
			require.FailNowf(tb, "payload must be relayed", "no %T relayed", zero)
			return zero
		}
	}
}

// testBook returns a one level book of a pair and asset
func testBook(ex *Exchange, pair currency.Pair, a asset.Item, price float64) *orderbook.Book {
	return &orderbook.Book{
		Bids:              orderbook.Levels{{Price: price - 1, Amount: 1}},
		Asks:              orderbook.Levels{{Price: price + 1, Amount: 1}},
		Exchange:          ex.Name,
		Pair:              pair,
		Asset:             a,
		LastUpdated:       time.Now(),
		ValidateOrderbook: true,
	}
}

// testOrderbookSubscription returns an order book subscription of a pair and asset streaming dataType
func testOrderbookSubscription(pair currency.Pair, a asset.Item, levels int) *subscription.Subscription {
	return &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            a,
		Pairs:            currency.Pairs{pair},
		Levels:           levels,
		QualifiedChannel: pair.String() + "@depth" + strconv.Itoa(levels),
	}
}

// claimAndLoadBook makes conn the feed of a book and loads a snapshot into it
func claimAndLoadBook(tb testing.TB, ex *Exchange, conn websocket.Connection, s *subscription.Subscription) {
	tb.Helper()
	state, err := ex.wsConnection(conn)
	require.NoError(tb, err, "wsConnection must not error")
	require.NoError(tb, ex.wsClaimBook(conn, state.generation, s), "wsClaimBook must not error")
	require.NoError(tb, ex.wsLoadBook(tb.Context(), conn, s.QualifiedChannel, testBook(ex, s.Pairs[0], s.Asset, 100)), "wsLoadBook must not error")
}

// bookError returns the error retrieving a stored book returns, nil for a valid book
func bookError(ex *Exchange, pair currency.Pair, a asset.Item) error {
	_, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
	return err
}

// testListenKeyServer serves the listen key endpoints, numbering the keys it creates and counting extensions, and the
// perpetual balance endpoints, answering with testBalanceReplies and counting the requests
type testListenKeyServer struct {
	*httptest.Server
	mu          sync.Mutex
	created     int
	extended    int
	createReply string
	extendReply string
	// failExtensions is how many extensions to refuse before replying with extendReply
	failExtensions  int
	balanceRequests map[string]int
	// apiKeys holds the API key of each request, by method and path
	apiKeys map[string][]string
}

// testBalanceReplies are the documented REST balance replies of the perpetual products, by path
var testBalanceReplies = map[string]string{
	"/openApi/swap/v3/user/balance":  `{"code":0,"msg":"","data":[{"userId":"116***295","asset":"USDT","balance":"194.8212","equity":"196.7431","unrealizedProfit":"1.9219","realisedProfit":"-109.2504","availableMargin":"193.7609","usedMargin":"1.0602","freezedMargin":"0.0000"}]}`,
	"/openApi/cswap/v1/user/balance": `{"code":0,"msg":"","timestamp":1718421301887,"data":[{"asset":"BTC","balance":"0.14438227","equity":"0.14428116","unrealizedProfit":"-0.0001011","availableMargin":"0.14317271","usedMargin":"0.00110845","freezedMargin":"0","shortUid":"12345678"}]}`,
}

// balanceRequestCount returns how many balance requests the server answered on a path
func (s *testListenKeyServer) balanceRequestCount(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.balanceRequests[path]
}

// newTestListenKeyServer starts a listen key server for the test
func newTestListenKeyServer(tb testing.TB) *testListenKeyServer {
	tb.Helper()
	s := &testListenKeyServer{}
	s.Server = httptest.NewTestServer(tb, s)
	s.Start()
	return s
}

func (s *testListenKeyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.apiKeys == nil {
		s.apiKeys = make(map[string][]string)
	}
	route := r.Method + " " + r.URL.Path
	s.apiKeys[route] = append(s.apiKeys[route], r.Header.Get(keyHeader))
	s.mu.Unlock()
	if reply, ok := testBalanceReplies[r.URL.Path]; ok && r.Method == http.MethodGet {
		s.mu.Lock()
		if s.balanceRequests == nil {
			s.balanceRequests = make(map[string]int)
		}
		s.balanceRequests[r.URL.Path]++
		s.mu.Unlock()
		_, _ = w.Write([]byte(reply))
		return
	}
	if r.URL.Path != "/openApi/user/auth/userDataStream" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Method {
	case http.MethodPost:
		s.created++
		if s.createReply != "" {
			_, _ = w.Write([]byte(s.createReply))
			return
		}
		_, _ = fmt.Fprintf(w, `{"listenKey":"%s-%d"}`, testListenKey, s.created)
	case http.MethodPut:
		s.extended++
		reply := cmpOrString(s.extendReply, `{}`)
		if s.failExtensions > 0 {
			s.failExtensions--
			reply = `{"code":100400,"msg":"try again"}`
		}
		_, _ = w.Write([]byte(reply))
	default:
		http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
	}
}

// requestKeys returns the API key of each request the server received on a route, such as
// "PUT /openApi/user/auth/userDataStream"
func (s *testListenKeyServer) requestKeys(route string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.apiKeys[route])
}

func (s *testListenKeyServer) counts() (created, extended int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.created, s.extended
}

func (s *testListenKeyServer) setReplies(createReply, extendReply string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createReply, s.extendReply = createReply, extendReply
}

// refuseExtensions makes the server refuse the next n extensions
func (s *testListenKeyServer) refuseExtensions(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failExtensions = n
}

// cmpOrString returns s, or fallback when s is empty
func cmpOrString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// testRESTEndpoints points every REST endpoint at a server
func testRESTEndpoints(tb testing.TB, ex *Exchange, serverURL string) {
	tb.Helper()
	for _, ePath := range []exchange.URL{exchange.RestSpot, exchange.RestUSDTMargined, exchange.RestCoinMargined} {
		require.NoErrorf(tb, ex.API.Endpoints.SetRunningURL(ePath.String(), serverURL), "SetRunningURL must not error for %s", ePath)
	}
}

func TestWebsocketConnectionSetups(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	_, err := ex.wsConnectionSetups(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "wsConnectionSetups should reject a nil configuration")

	setups, err := ex.wsConnectionSetups(ex.Config)
	require.NoError(t, err, "wsConnectionSetups must not error")
	require.Len(t, setups, 6, "wsConnectionSetups must return a market and an account setup for each product")
	for i, exp := range []struct {
		filter, url   string
		subscriptions int
	}{
		{websocketSpotFilter, spotWebsocketURL, 4},
		{websocketSpotAccountFilter, spotWebsocketURL, 0},
		{websocketSwapFilter, usdtMarginedWebsocketURL, 4},
		{websocketSwapAccountFilter, usdtMarginedWebsocketURL, 0},
		{websocketCoinMarginedFilter, coinMarginedWebsocketURL, 4},
		{websocketCoinMarginedAccountFilter, coinMarginedWebsocketURL, 0},
	} {
		t.Run(exp.filter, func(t *testing.T) {
			t.Parallel()
			s := setups[i]
			assert.Equalf(t, exp.filter, s.MessageFilter, "setup %d should have the expected message filter", i)
			assert.Equalf(t, exp.url, s.URL, "setup %d should connect to its product's endpoint", i)
			assert.Equalf(t, ex.Config.WebsocketResponseMaxLimit, s.ResponseMaxLimit, "setup %d should take the configured response limit", i)
			assert.Equalf(t, ex.Config.WebsocketResponseCheckTimeout, s.ResponseCheckTimeout, "setup %d should take the configured response check timeout", i)
			for name, fn := range map[string]any{"Connector": s.Connector, "GenerateSubscriptions": s.GenerateSubscriptions, "Subscriber": s.Subscriber, "Unsubscriber": s.Unsubscriber, "Handler": s.Handler} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					assert.NotNilf(t, fn, "setup %d should have a %s", i, name)
				})
			}
			subs, err := s.GenerateSubscriptions()
			require.NoErrorf(t, err, "setup %d GenerateSubscriptions must not error", i)
			assert.Lenf(t, subs, exp.subscriptions, "setup %d should generate its enabled pair's market subscriptions, and no account ones without authentication", i)
		})
	}

	t.Run("configured endpoint", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.WebsocketUSDTMargined.String(), "wss://vst-open-api-ws.bingx.com/swap-market"), "SetRunningURL must not error")
		setups, err := ex.wsConnectionSetups(ex.Config)
		require.NoError(t, err, "wsConnectionSetups must not error")
		assert.Equal(t, "wss://vst-open-api-ws.bingx.com/swap-market", setups[2].URL, "the USDT-M market setup should take the configured endpoint")
		assert.Equal(t, "wss://vst-open-api-ws.bingx.com/swap-market", setups[3].URL, "the USDT-M account setup should take the configured endpoint")
	})

	t.Run("missing endpoint", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		ex.API.Endpoints = ex.NewEndpoints()
		_, err := ex.wsConnectionSetups(ex.Config)
		assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "wsConnectionSetups should error without the websocket endpoints")
	})
}

func TestWebsocketDialer(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	d := ex.wsDialer()
	assert.Equal(t, ex.Config.HTTPTimeout, d.HandshakeTimeout, "the handshake should be bounded by the configured HTTP timeout")
	assert.NotNil(t, d.Proxy, "the dialer should use the environment's proxy")
}

func TestWebsocketConnectMarket(t *testing.T) { //nolint:tparallel // the cases track their connections on one exchange
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, mockws.EchoHandler))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	t.Cleanup(func() {
		// The manager never connects, so the session the connections were tracked under is ended here
		close(ex.Websocket.ShutdownC)
		waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
	})
	for _, serialRequests := range []bool{false, true} {
		t.Run("serial requests "+strconv.FormatBool(serialRequests), func(t *testing.T) {
			conn, err := ex.Websocket.CreateTestConnection(websocketCoinMarginedFilter)
			require.NoError(t, err, "CreateTestConnection must not error")
			require.NoErrorf(t, ex.wsConnectMarket(t.Context(), conn, serialRequests), "wsConnectMarket must not error with serial requests %t", serialRequests)
			t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
			state, err := ex.wsConnection(conn)
			require.NoError(t, err, "wsConnectMarket must track the connection")
			assert.Equalf(t, serialRequests, state.gate != nil, "a connection should have a request gate only for serial requests %t", serialRequests)
			assert.Equal(t, receiveOnly(ex.Websocket.ShutdownC), state.session, "the connection should belong to the manager's session")
		})
	}

	err := ex.wsConnectMarket(t.Context(), &testWebsocketConnection{}, false)
	assert.ErrorIs(t, err, errTestDial, "wsConnectMarket should return the dial error")
}

func TestWebsocketConnectAccount(t *testing.T) {
	t.Parallel()
	rest := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, rest.URL)
	var dialledKeys []string
	var mu sync.Mutex
	s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dialledKeys = append(dialledKeys, r.URL.Query().Get("listenKey"))
		mu.Unlock()
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	t.Cleanup(func() {
		close(ex.Websocket.ShutdownC)
		waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
	})

	spot, err := ex.Websocket.CreateTestConnection(websocketSpotAccountFilter)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.wsConnectAccount(t.Context(), spot, asset.Spot, exchange.RestSpot), "wsConnectAccount must not error")
	t.Cleanup(func() { assert.NoError(t, spot.Shutdown(), "Shutdown should not error") })
	swap, err := ex.Websocket.CreateTestConnection(websocketSwapAccountFilter)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.wsConnectAccount(t.Context(), swap, asset.USDTMarginedFutures, exchange.RestUSDTMargined), "wsConnectAccount must not error")
	t.Cleanup(func() { assert.NoError(t, swap.Shutdown(), "Shutdown should not error") })

	created, _ := rest.counts()
	assert.Equal(t, 1, created, "account connections to one REST host should share one listen key")
	mu.Lock()
	assert.Equal(t, []string{testListenKey + "-1", testListenKey + "-1"}, dialledKeys, "each account connection should be dialled with the listen key")
	mu.Unlock()
	assert.NotContains(t, spot.GetURL(), "listenKey", "the listen key should not be part of the connection's URL")
	ex.wsState.mu.Lock()
	keeper := ex.wsState.keepers[rest.URL]
	require.NotNil(t, keeper, "a keeper must keep the listen key alive")
	assert.Equal(t, []websocket.Connection{spot, swap}, keeper.conns, "the keeper should know the connections using its key")
	ex.wsState.mu.Unlock()
	for _, tc := range []struct {
		a    asset.Item
		conn websocket.Connection
	}{{asset.Spot, spot}, {asset.USDTMarginedFutures, swap}} {
		t.Run(tc.a.String()+" tracked", func(t *testing.T) {
			t.Parallel()
			state, err := ex.wsConnection(tc.conn)
			require.NoError(t, err, "wsConnectAccount must track the connection")
			assert.Equal(t, tc.a, state.account, "the connection should stream the account events of its asset's product")
		})
	}
	testWaitBackground(t, ex)
	assert.Equal(t, []string{"test-api-key"}, rest.requestKeys("GET /openApi/swap/v2/trade/openOrders"), "the USDT-M account connection should read its account's open orders once up")
	assert.Empty(t, rest.requestKeys("GET /openApi/spot/v1/trade/openOrders"), "the spot account connection should read no open orders")

	t.Run("dial failure", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		refusing := httptest.NewTestServer(t, http.NotFoundHandler())
		refusing.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(refusing.URL, "http")), "SetAllConnectionURLs must not error")
		t.Cleanup(func() {
			close(ex.Websocket.ShutdownC)
			waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
		})
		conn, err := ex.Websocket.CreateTestConnection(websocketSpotAccountFilter)
		require.NoError(t, err, "CreateTestConnection must not error")
		err = ex.wsConnectAccount(t.Context(), conn, asset.Spot, exchange.RestSpot)
		require.ErrorIs(t, err, errWebsocketAccountDial, "wsConnectAccount must report the failed dial")
		assert.NotContains(t, err.Error(), testListenKey, "the error should not carry the listen key")
		assert.Contains(t, err.Error(), "[redacted]", "the error should show where the listen key was")
	})

	t.Run("listen key failure", func(t *testing.T) {
		t.Parallel()
		failing := newTestListenKeyServer(t)
		failing.setReplies(`{"code":100500,"msg":"internal error"}`, "")
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, failing.URL)
		err := ex.wsConnectAccount(t.Context(), &testWebsocketConnection{}, asset.Spot, exchange.RestSpot)
		assert.ErrorIs(t, err, errWebsocketListenKeyUnavailable, "wsConnectAccount should report a listen key it could not get")
	})

	t.Run("balance refresh", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		ex.balances.interval = time.Millisecond
		s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		s.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
		t.Cleanup(func() {
			close(ex.Websocket.ShutdownC)
			waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
		})
		connect := func(a asset.Item, filter string, ePath exchange.URL) {
			conn, err := ex.Websocket.CreateTestConnection(filter)
			require.NoError(t, err, "CreateTestConnection must not error")
			require.NoErrorf(t, ex.wsConnectAccount(t.Context(), conn, a, ePath), "wsConnectAccount must not error for %s", a)
			t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
		}
		connect(asset.Spot, websocketSpotAccountFilter, exchange.RestSpot)
		connect(asset.USDTMarginedFutures, websocketSwapAccountFilter, exchange.RestUSDTMargined)
		connect(asset.CoinMarginedFutures, websocketCoinMarginedAccountFilter, exchange.RestCoinMargined)
		testWaitBackground(t, ex)
		assert.Equal(t, 1, rest.balanceRequestCount("/openApi/swap/v3/user/balance"), "wsConnectAccount should refresh the USDT-M balances once the account connection is up")
		assert.Equal(t, 1, rest.balanceRequestCount("/openApi/cswap/v1/user/balance"), "wsConnectAccount should refresh the coin-M balances once the account connection is up")
		assert.Zero(t, ex.balances.tickets[testAccountBalanceKey(t.Context(), t, ex, asset.Spot)], "wsConnectAccount should not refresh spot balances, which spot's account events carry")
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		_, err = ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		assert.NoError(t, err, "wsConnectAccount should store the refreshed balances")
	})

	t.Run("refresh the previous session's end cut off", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		var abandoned atomic.Bool
		arrived, handled := make(chan struct{}), make(chan struct{})
		rest := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/openApi/user/auth/userDataStream":
				_, _ = io.WriteString(w, `{"listenKey":"`+testListenKey+`"}`)
			case "/openApi/swap/v3/user/balance":
				if requests.Add(1) == 1 {
					defer close(handled)
					close(arrived)
					// The server notices the client giving up, which ends the request's context, once the body is read
					_, err := io.Copy(io.Discard, r.Body)
					assert.NoError(t, err, "reading the request body should not error")
					select {
					case <-r.Context().Done():
						abandoned.Store(true)
					case <-time.After(5 * time.Second):
					}
					return
				}
				_, _ = io.WriteString(w, testBalanceReplies[r.URL.Path])
			default:
				http.NotFound(w, r)
			}
		}))
		rest.Start()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		ex.balances.interval = time.Hour
		s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		s.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
		// An account event of the previous session asks for a refresh, which the session's end cuts off
		previous := make(chan struct{})
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, previous)
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the previous session's balance request must arrive")
		}
		close(previous)
		select {
		case <-handled:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "the previous session's balance request must be answered")
		}
		require.True(t, abandoned.Load(), "the previous session's end must cut its balance request off")
		testWaitBackground(t, ex)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		_, err = ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		require.ErrorIs(t, err, accounts.ErrNoBalances, "the balances the cut off refresh requested must not be stored")

		t.Cleanup(func() {
			close(ex.Websocket.ShutdownC)
			waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
			testWaitBackground(t, ex)
		})
		conn, err := ex.Websocket.CreateTestConnection(websocketSwapAccountFilter)
		require.NoError(t, err, "CreateTestConnection must not error")
		require.NoError(t, ex.wsConnectAccount(t.Context(), conn, asset.USDTMarginedFutures, exchange.RestUSDTMargined), "wsConnectAccount must not error")
		t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
		assert.Eventually(t, func() bool {
			_, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
			return err == nil
		}, 5*time.Second, 10*time.Millisecond, "wsConnectAccount should refresh the balances the previous session's end cut off once the next account connection is up")
		assert.Equal(t, int32(2), requests.Load(), "wsConnectAccount should refresh the balances once for the next session")
	})

	t.Run("credentials changing while connected", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		ex.balances.interval = time.Millisecond
		ex.wsState.mu.Lock()
		ex.wsState.keepaliveInterval = 5 * time.Millisecond
		ex.wsState.mu.Unlock()
		s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		s.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
		t.Cleanup(func() {
			close(ex.Websocket.ShutdownC)
			waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
			testWaitBackground(t, ex)
		})
		connect := func(a asset.Item, filter string, ePath exchange.URL) websocket.Connection {
			conn, err := ex.Websocket.CreateTestConnection(filter)
			require.NoError(t, err, "CreateTestConnection must not error")
			require.NoErrorf(t, ex.wsConnectAccount(t.Context(), conn, a, ePath), "wsConnectAccount must not error for %s", a)
			t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
			return conn
		}
		first, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		spot := connect(asset.Spot, websocketSpotAccountFilter, exchange.RestSpot)
		second := &accounts.Credentials{Key: "second-key", Secret: "second-secret"}
		ex.SetCredentials(second)
		extensions := len(rest.requestKeys("PUT /openApi/user/auth/userDataStream"))
		// The perpetual connections take the listen key created for the first credentials, so they stream their account too
		swap := connect(asset.USDTMarginedFutures, websocketSwapAccountFilter, exchange.RestUSDTMargined)
		connect(asset.CoinMarginedFutures, websocketCoinMarginedAccountFilter, exchange.RestCoinMargined)
		require.NoError(t, ex.wsHandleSpotAccountData(t.Context(), spot, fixtureLine(t, "testdata/wsSpotAccount.jsonl", "ACCOUNT_UPDATE")), "wsHandleSpotAccountData must not error")
		require.NoError(t, ex.wsHandleSwapAccountData(t.Context(), swap, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_UPDATE")), "wsHandleSwapAccountData must not error")
		require.Eventually(t, func() bool {
			return len(rest.requestKeys("PUT /openApi/user/auth/userDataStream")) >= extensions+2
		}, 5*time.Second, time.Millisecond, "the keeper must extend the listen key after the default credentials change")
		testWaitBackground(t, ex)
		assert.Len(t, rest.requestKeys("GET /openApi/swap/v3/user/balance"), 2, "the USDT-M balances should be refreshed once the connection is up and once after its balance event")
		assert.Len(t, rest.requestKeys("GET /openApi/swap/v2/trade/openOrders"), 1, "the USDT-M open orders should be read once the connection is up")
		assert.Len(t, rest.requestKeys("GET /openApi/cswap/v1/trade/openOrders"), 1, "the coin-M open orders should be read once the connection is up")
		for _, tc := range []struct {
			name, route string
		}{
			{"listen key creation", "POST /openApi/user/auth/userDataStream"},
			{"listen key extensions", "PUT /openApi/user/auth/userDataStream"},
			{"balance refreshes", "GET /openApi/swap/v3/user/balance"},
			{"open order reads", "GET /openApi/swap/v2/trade/openOrders"},
			{"coin-M balance refreshes", "GET /openApi/cswap/v1/user/balance"},
			{"coin-M open order reads", "GET /openApi/cswap/v1/trade/openOrders"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				for _, apiKey := range rest.requestKeys(tc.route) {
					assert.Equalf(t, first.Key, apiKey, "%s should be sent for the account the listen key was created for", tc.route)
				}
			})
		}
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				_, err := ex.Accounts.GetBalance("", first, a, currency.USDT)
				assert.NoErrorf(t, err, "the %s balances should be stored for the account the listen key was created for", a)
				_, err = ex.Accounts.GetBalance("", second, a, currency.USDT)
				assert.ErrorIsf(t, err, accounts.ErrNoBalances, "no %s balances should be stored for the exchange's new default credentials", a)
			})
		}
	})

	t.Run("order events held while the open orders are read", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			a              asset.Item
			filter, symbol string
			ePath          exchange.URL
			openOrders     string
		}{
			{asset.USDTMarginedFutures, websocketSwapAccountFilter, "BTC-USDT", exchange.RestUSDTMargined, "/openApi/swap/v2/trade/openOrders"},
			{asset.CoinMarginedFutures, websocketCoinMarginedAccountFilter, "BTC-USD", exchange.RestCoinMargined, "/openApi/cswap/v1/trade/openOrders"},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				arrived, release := make(chan struct{}), make(chan struct{})
				rest := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/openApi/user/auth/userDataStream":
						_, _ = io.WriteString(w, `{"listenKey":"`+testListenKey+`"}`)
					case tc.openOrders:
						close(arrived)
						<-release
						_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[{"symbol":"`+tc.symbol+`","orderId":42,"side":"BUY","type":"LIMIT","origQty":"3","price":"10","executedQty":"2","avgPrice":"10","commission":"-0.23","status":"PARTIALLY_FILLED"}]}}`)
					default:
						_, _ = io.WriteString(w, testBalanceReplies[r.URL.Path])
					}
				}))
				rest.Start()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				testRESTEndpoints(t, ex, rest.URL)
				require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
				s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
				}))
				s.Start()
				require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
				t.Cleanup(func() {
					close(ex.Websocket.ShutdownC)
					waitForSessionEnd(t, ex, ex.Websocket.ShutdownC)
					testWaitBackground(t, ex)
				})
				conn, err := ex.Websocket.CreateTestConnection(tc.filter)
				require.NoError(t, err, "CreateTestConnection must not error")
				t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
				var releaseOnce sync.Once
				t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
				require.NoError(t, ex.wsConnectAccount(t.Context(), conn, tc.a, tc.ePath), "wsConnectAccount must not wait for the open order read")
				select {
				case <-arrived:
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the connection must read its account's open orders")
				}
				state, err := ex.wsConnection(conn)
				require.NoError(t, err, "wsConnectAccount must track the connection")
				state.heldOrders.Lock()
				assert.True(t, state.holding, "the connection's order events should be held while its open orders are read")
				state.heldOrders.Unlock()
				releaseOnce.Do(func() { close(release) })
				assert.Eventually(t, func() bool {
					state.heldOrders.Lock()
					defer state.heldOrders.Unlock()
					return !state.holding
				}, 5*time.Second, time.Millisecond, "the connection should stop holding order events once the read ends")
				state.fees.mu.Lock()
				defer state.fees.mu.Unlock()
				if total, ok := state.fees.orders["42"]; assert.True(t, ok, "the read should start the open order's total") {
					assert.True(t, decimal.MustFromString("0.23").Equal(total.fee), "the total should hold the fees the order had paid")
				}
			})
		}
	})
}

func TestWebsocketRedact(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "dial wss://h/market?listenKey=[redacted]: bad handshake", wsRedact("dial wss://h/market?listenKey=k1: bad handshake", "k1"), "wsRedact should replace the secret")
	assert.Equal(t, "unchanged", wsRedact("unchanged", ""), "wsRedact should leave a message alone without a secret")
}

func TestWebsocketTrackConnection(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	session := newTestSession(t, ex)
	a, b := &testWebsocketConnection{}, &testWebsocketConnection{}
	stateA := ex.wsTrackConnection(a, session.ch, false, asset.Empty, nil)
	stateB := ex.wsTrackConnection(b, session.ch, true, asset.Empty, nil)
	assert.Less(t, stateA.generation, stateB.generation, "each connection should have a later generation")
	assert.Nil(t, stateA.gate, "a connection without serial requests should have no request gate")
	assert.NotNil(t, stateB.gate, "a connection with serial requests should have a request gate")
	assert.Nil(t, stateA.credentials, "a market connection should have no credentials")
	assert.Equal(t, asset.Empty, stateA.account, "a market connection should stream no account")
	creds := &accounts.Credentials{Key: "account-key", Secret: "account-secret"}
	account := ex.wsTrackConnection(&testWebsocketConnection{}, session.ch, false, asset.CoinMarginedFutures, creds)
	assert.Same(t, creds, account.credentials, "an account connection should keep the credentials of its listen key")
	assert.Equal(t, asset.CoinMarginedFutures, account.account, "an account connection should keep the asset whose account events it streams")
	ex.wsState.mu.Lock()
	assert.Len(t, ex.wsState.sessions, 1, "a session should be watched once")
	ex.wsState.mu.Unlock()

	sub := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	claimAndLoadBook(t, ex, a, sub)
	require.NoError(t, bookError(ex, testSpotPair, asset.Spot), "the book must be valid")
	a.down.Store(true)
	ex.wsTrackConnection(&testWebsocketConnection{}, session.ch, false, asset.Empty, nil)
	_, err := ex.wsConnection(a)
	assert.ErrorIs(t, err, errWebsocketConnectionUnknown, "a connection no longer connected should be forgotten when another is tracked")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "the book a forgotten connection fed should be invalidated")

	again := ex.wsTrackConnection(b, session.ch, true, asset.Empty, nil)
	assert.Greater(t, again.generation, stateB.generation, "a connection tracked again should have a new generation")
}

func TestWebsocketEndSession(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ended, kept := newTestSession(t, ex), newTestSession(t, ex)
	a, b := &testWebsocketConnection{}, &testWebsocketConnection{}
	ex.wsTrackConnection(a, ended.ch, false, asset.Empty, nil)
	ex.wsTrackConnection(b, kept.ch, false, asset.Empty, nil)
	claimAndLoadBook(t, ex, a, testOrderbookSubscription(testSpotPair, asset.Spot, 20))
	claimAndLoadBook(t, ex, b, testOrderbookSubscription(testCoinMarginedPair, asset.CoinMarginedFutures, 20))

	ex.wsEndSession(make(chan struct{}))
	assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "ending an unknown session should leave the books alone")

	ended.end()
	require.Eventually(t, func() bool {
		return errors.Is(bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid)
	}, 5*time.Second, time.Millisecond, "the book a torn down session's connection fed must be invalidated")
	assert.NoError(t, bookError(ex, testCoinMarginedPair, asset.CoinMarginedFutures), "the book another session feeds should stay valid")
	_, err := ex.wsConnection(a)
	assert.ErrorIs(t, err, errWebsocketConnectionUnknown, "a torn down session's connections should be forgotten")
	_, err = ex.wsConnection(b)
	assert.NoError(t, err, "another session's connections should be kept")
}

func TestWebsocketConnection(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	_, err := ex.wsConnection(conn)
	assert.ErrorIs(t, err, errWebsocketConnectionUnknown, "wsConnection should error for a connection no Connector dialled")
	tracked, _ := trackTestConnection(t, ex, conn, false)
	state, err := ex.wsConnection(conn)
	require.NoError(t, err, "wsConnection must not error for a tracked connection")
	assert.Same(t, tracked, state, "wsConnection should return the connection's state")
}

func TestWebsocketIsConnected(t *testing.T) {
	t.Parallel()
	conn := &testWebsocketConnection{}
	assert.True(t, wsIsConnected(conn), "wsIsConnected should report a connected connection")
	conn.down.Store(true)
	assert.False(t, wsIsConnected(conn), "wsIsConnected should report a closed connection")
	assert.True(t, wsIsConnected(struct{ websocket.Connection }{}), "wsIsConnected should take a connection that cannot tell as connected")
}

func TestWebsocketAccountConnections(t *testing.T) {
	t.Parallel()
	creds := &accounts.Credentials{Key: "account-key", Secret: "account-secret"}
	for _, tc := range []struct {
		name        string
		account     asset.Item
		credentials *accounts.Credentials
		down        bool
		exp         bool
	}{
		{name: "the account's connection on the product", account: asset.USDTMarginedFutures, credentials: creds, exp: true},
		{name: "the account's connection on another product", account: asset.CoinMarginedFutures, credentials: creds},
		{name: "another account's connection on the product", account: asset.USDTMarginedFutures, credentials: &accounts.Credentials{Key: "other-key", Secret: "other-secret"}},
		{name: "a market connection", account: asset.Empty},
		{name: "the account's connection down", account: asset.USDTMarginedFutures, credentials: creds, down: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			conn := &testWebsocketConnection{}
			conn.down.Store(tc.down)
			state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, tc.account, tc.credentials)
			got := ex.wsAccountConnections(asset.USDTMarginedFutures, creds)
			if tc.exp {
				assert.Equal(t, []*websocketConnection{state}, got, "wsAccountConnections should return the account's up connection of the product")
			} else {
				assert.Empty(t, got, "wsAccountConnections should return only up connections of the product for the account")
			}
		})
	}
	t.Run("several connections", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		session := newTestSession(t, ex).ch
		exp := make([]*websocketConnection, 0, 3)
		for range 3 {
			exp = append(exp, ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.USDTMarginedFutures, creds))
			ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.CoinMarginedFutures, creds)
		}
		assert.ElementsMatch(t, exp, ex.wsAccountConnections(asset.USDTMarginedFutures, creds), "wsAccountConnections should return every up connection of the product for the account")
	})
	t.Run("no connection", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, newWebsocketTestExchange(t).wsAccountConnections(asset.USDTMarginedFutures, creds), "wsAccountConnections should return nothing without a connection")
	})
}

func TestWebsocketBookKey(t *testing.T) {
	t.Parallel()
	k, err := wsBookKey(testOrderbookSubscription(testSpotPair, asset.Spot, 20))
	require.NoError(t, err, "wsBookKey must not error for one pair")
	assert.Equal(t, testSpotPair.Base.Item, k.Base, "the key should hold the base")
	assert.Equal(t, testSpotPair.Quote.Item, k.Quote, "the key should hold the quote")
	assert.Equal(t, asset.Spot, k.Asset, "the key should hold the asset")
	for _, pairs := range []currency.Pairs{nil, {testSpotPair, currency.NewPairWithDelimiter("ETH", "USDT", "-")}} {
		t.Run(strconv.Itoa(len(pairs))+" pairs", func(t *testing.T) {
			t.Parallel()
			_, err := wsBookKey(&subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: pairs})
			assert.ErrorIsf(t, err, subscription.ErrNotSinglePair, "wsBookKey should reject %d pairs", len(pairs))
		})
	}
}

func TestWebsocketClaimBook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	a, b, c := &testWebsocketConnection{}, &testWebsocketConnection{}, &testWebsocketConnection{}
	stateA, _ := trackTestConnection(t, ex, a, false)
	stateB, _ := trackTestConnection(t, ex, b, false)
	stateC, _ := trackTestConnection(t, ex, c, false)
	depth20 := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	depth50 := testOrderbookSubscription(testSpotPair, asset.Spot, 50)

	require.NoError(t, ex.wsClaimBook(a, stateA.generation, depth20), "wsClaimBook must not error for a book without a feed")
	err := ex.wsClaimBook(b, stateB.generation, depth50)
	assert.ErrorIs(t, err, errWebsocketOrderbookSubscriptionConflict, "wsClaimBook should refuse a book another live stream feeds")
	assert.NoError(t, ex.wsClaimBook(b, stateB.generation, depth20), "wsClaimBook should let the same stream move to another connection")
	ex.wsState.booksMu.Lock()
	feed := ex.wsState.books[key.PairAsset{Base: testSpotPair.Base.Item, Quote: testSpotPair.Quote.Item, Asset: asset.Spot}]
	ex.wsState.booksMu.Unlock()
	assert.Equal(t, websocketBookFeed{conn: b, generation: stateB.generation, stream: depth20.QualifiedChannel}, feed, "the book's feed should be the last claim")

	b.down.Store(true)
	assert.NoError(t, ex.wsClaimBook(c, stateC.generation, depth50), "wsClaimBook should replace a feed whose connection is closed")
	assert.NoError(t, ex.wsClaimBook(a, stateA.generation, testOrderbookSubscription(testSpotPair, asset.USDTMarginedFutures, 20)), "wsClaimBook should let another asset's book be fed")

	err = ex.wsClaimBook(a, stateA.generation, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot})
	assert.ErrorIs(t, err, subscription.ErrNotSinglePair, "wsClaimBook should reject a subscription without one pair")
}

func TestWebsocketReleaseBook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	a, b := &testWebsocketConnection{}, &testWebsocketConnection{}
	trackTestConnection(t, ex, a, false)
	trackTestConnection(t, ex, b, false)
	depth20 := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	claimAndLoadBook(t, ex, a, depth20)

	ex.wsReleaseBook(b, depth20)
	assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "a release by another connection should leave the book alone")
	ex.wsReleaseBook(a, testOrderbookSubscription(testSpotPair, asset.Spot, 50))
	assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "a release of another stream should leave the book alone")
	ex.wsReleaseBook(a, &subscription.Subscription{Channel: subscription.OrderbookChannel})
	assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "a release without a pair should leave the book alone")
	ex.wsReleaseBook(a, depth20)
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "releasing the book's stream should invalidate the book")
	ex.wsState.booksMu.Lock()
	assert.Empty(t, ex.wsState.books, "releasing the book's stream should forget its feed")
	ex.wsState.booksMu.Unlock()

	unloaded := testOrderbookSubscription(testCoinMarginedPair, asset.CoinMarginedFutures, 20)
	stateA, err := ex.wsConnection(a)
	require.NoError(t, err, "wsConnection must not error")
	require.NoError(t, ex.wsClaimBook(a, stateA.generation, unloaded), "wsClaimBook must not error")
	ex.wsReleaseBook(a, unloaded)
	assert.ErrorIs(t, bookError(ex, testCoinMarginedPair, asset.CoinMarginedFutures), orderbook.ErrDepthNotFound, "releasing a book never loaded should not create it")
}

func TestWebsocketInvalidateBooks(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	a, b := &testWebsocketConnection{}, &testWebsocketConnection{}
	stateA, _ := trackTestConnection(t, ex, a, false)
	trackTestConnection(t, ex, b, false)
	claimAndLoadBook(t, ex, a, testOrderbookSubscription(testSpotPair, asset.Spot, 20))
	claimAndLoadBook(t, ex, b, testOrderbookSubscription(testSwapPair, asset.USDTMarginedFutures, 20))

	ex.wsInvalidateBooks(map[uint64]bool{stateA.generation: true})
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "the generation's book should be invalidated")
	assert.NoError(t, bookError(ex, testSwapPair, asset.USDTMarginedFutures), "another generation's book should stay valid")
}

func TestWebsocketLoadBook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	a, b := &testWebsocketConnection{}, &testWebsocketConnection{}
	stateA, session := trackTestConnection(t, ex, a, false)
	depth20 := testOrderbookSubscription(testSpotPair, asset.Spot, 20)

	stream := depth20.QualifiedChannel
	require.NoError(t, ex.wsLoadBook(t.Context(), b, stream, testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error for a connection no Connector dialled")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrDepthNotFound, "a push from a connection no Connector dialled should not load a book")
	require.NoError(t, ex.wsLoadBook(t.Context(), a, stream, testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error for a book without a feed")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrDepthNotFound, "a push of a book without a feed should not load it")

	require.NoError(t, ex.wsClaimBook(a, stateA.generation, depth20), "wsClaimBook must not error")
	ex.wsTrackConnection(b, session.ch, false, asset.Empty, nil)
	require.NoError(t, ex.wsLoadBook(t.Context(), b, stream, testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error for another connection")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrDepthNotFound, "a push from a connection not feeding the book should not load it")
	require.NoError(t, ex.wsLoadBook(t.Context(), a, "BTC-USDT@depth5", testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error for another stream")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrDepthNotFound, "a push of a stream not feeding the book should not load it")

	require.NoError(t, ex.wsLoadBook(t.Context(), a, stream, testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error for the book's feed")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
	require.NoError(t, err, "the feed's push must load the book")
	assert.InDelta(t, 99, book.Bids[0].Price, 1e-9, "the book should hold the push's levels")

	ex.wsTrackConnection(a, session.ch, false, asset.Empty, nil)
	require.NoError(t, ex.wsLoadBook(t.Context(), a, stream, testBook(ex, testSpotPair, asset.Spot, 200)), "wsLoadBook must not error for a stale generation")
	book, err = ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.InDelta(t, 99, book.Bids[0].Price, 1e-9, "a push from a connection's older generation should not reload the book")

	state, err := ex.wsConnection(a)
	require.NoError(t, err, "wsConnection must not error")
	require.NoError(t, ex.wsClaimBook(a, state.generation, depth20), "wsClaimBook must not error")
	invalid := testBook(ex, testSpotPair, asset.Spot, 100)
	invalid.Bids[0].Amount = 0
	assert.Error(t, ex.wsLoadBook(t.Context(), a, stream, invalid), "wsLoadBook should return a book that fails validation")
}

func TestWebsocketSortLevels(t *testing.T) {
	t.Parallel()
	bids := orderbook.Levels{{Price: 1, Amount: 1}, {Price: 3, Amount: 3}, {Price: 2, Amount: 2}}
	asks := orderbook.Levels{{Price: 6, Amount: 6}, {Price: 4, Amount: 4}, {Price: 5, Amount: 5}}
	wsSortLevels(bids, asks)
	assert.Equal(t, orderbook.Levels{{Price: 3, Amount: 3}, {Price: 2, Amount: 2}, {Price: 1, Amount: 1}}, bids, "bids should be sorted highest first")
	assert.Equal(t, orderbook.Levels{{Price: 4, Amount: 4}, {Price: 5, Amount: 5}, {Price: 6, Amount: 6}}, asks, "asks should be sorted lowest first")
}

func TestWebsocketFrameTime(t *testing.T) {
	t.Parallel()
	assert.Equal(t, time.UnixMilli(1794390762865), wsFrameTime([]byte(`{"ts":1794390762865}`), "ts"), "wsFrameTime should return the frame's time")
	assert.Equal(t, time.UnixMilli(1794390763070), wsFrameTime([]byte(`{"data":{"timestamp":1794390763070}}`), "data", "timestamp"), "wsFrameTime should return a nested time")
	for _, tc := range []struct {
		name, frame string
	}{
		{"missing time", `{}`},
		{"zero time", `{"ts":0}`},
		{"malformed time", `{"ts":"x"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.WithinDurationf(t, time.Now(), wsFrameTime([]byte(tc.frame), "ts"), time.Minute, "wsFrameTime should return the receipt time for %s", tc.frame)
		})
	}
}

func TestWebsocketGenerateMarketSubscriptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		exp   []string
	}{
		{asset.Spot, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1min", "BTC-USDT@depth50"}},
		{asset.USDTMarginedFutures, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1m", "BTC-USDT@depth50"}},
		{asset.CoinMarginedFutures, []string{"BTC-USD@ticker", "BTC-USD@trade", "BTC-USD@kline_1m", "BTC-USD@depth50"}},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			subs, err := ex.wsGenerateMarketSubscriptions(tc.asset)
			require.NoError(t, err, "wsGenerateMarketSubscriptions must not error")
			assert.ElementsMatch(t, tc.exp, subs.QualifiedChannels(), "wsGenerateMarketSubscriptions should name each default stream of the enabled pair")
			for _, s := range subs {
				assert.Equal(t, tc.asset, s.Asset, "each subscription should be of the asset")
				assert.Lenf(t, s.Pairs, 1, "%s should have one pair", s.QualifiedChannel)
				assert.False(t, s.Authenticated, "no market subscription should need authentication")
			}
		})
	}

	t.Run("disabled asset", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAssets(asset.Spot))
		subs, err := ex.wsGenerateMarketSubscriptions(asset.CoinMarginedFutures)
		require.NoError(t, err, "wsGenerateMarketSubscriptions must not error for a disabled asset")
		assert.Empty(t, subs, "a disabled asset should generate nothing")
	})

	t.Run("all assets and duplicates", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketSubscriptions(
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Levels: 5},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
		))
		subs, err := ex.wsGenerateMarketSubscriptions(asset.Spot)
		require.NoError(t, err, "wsGenerateMarketSubscriptions must not error")
		require.Len(t, subs, 1, "subscriptions to one stream must be kept once")
		assert.Equal(t, "BTC-USDT@ticker", subs[0].QualifiedChannel, "the all-assets subscription should expand to the asset's stream")
		assert.Equal(t, asset.Spot, subs[0].Asset, "the all-assets subscription should take the asset")
		swap, err := ex.wsGenerateMarketSubscriptions(asset.USDTMarginedFutures)
		require.NoError(t, err, "wsGenerateMarketSubscriptions must not error")
		assert.Equal(t, []string{"BTC-USDT@ticker"}, swap.QualifiedChannels(), "the all-assets subscription should expand on every asset")
	})

	t.Run("order book conflict", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketSubscriptions(
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		))
		subs, err := ex.wsGenerateMarketSubscriptions(asset.Spot)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "a conflicting order book subscription should be reported as partial generation")
		assert.ErrorIs(t, err, errWebsocketOrderbookSubscriptionConflict, "the conflict should be reported")
		assert.Equal(t, []string{"BTC-USDT@depth20", "BTC-USDT@ticker"}, subs.QualifiedChannels(), "the first order book subscription and the others should be kept")
	})

	t.Run("unsupported subscriptions", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketSubscriptions(
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Levels: 7},
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: WebsocketMarkPriceChannel},
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
		))
		subs, err := ex.wsGenerateMarketSubscriptions(asset.Spot)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "unsupported subscriptions should be reported as partial generation")
		assert.ErrorIs(t, err, errWebsocketUnsupportedLevels, "unsupported levels should be reported")
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "a spot mark price subscription should be reported")
		assert.Equal(t, []string{"BTC-USDT@ticker"}, subs.QualifiedChannels(), "the supported subscriptions should be kept")
		swap, err := ex.wsGenerateMarketSubscriptions(asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, errWebsocketUnsupportedLevels, "unsupported levels should be reported for every asset")
		assert.ElementsMatch(t, []string{"BTC-USDT@markPrice", "BTC-USDT@ticker"}, swap.QualifiedChannels(), "mark prices should stream on the perpetual connections")
	})

	t.Run("capacity", func(t *testing.T) {
		t.Parallel()
		pairs := make(currency.Pairs, 50)
		for i := range pairs {
			pairs[i] = currency.NewPairWithDelimiter("C"+strconv.Itoa(i), "USDT", currency.DashDelimiter)
		}
		channels := []*subscription.Subscription{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20},
			{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
		}
		ex := newWebsocketTestExchange(t, testWebsocketPairs(asset.Spot, pairs), testWebsocketSubscriptions(channels...))
		subs, err := ex.wsGenerateMarketSubscriptions(asset.Spot)
		require.NoError(t, err, "wsGenerateMarketSubscriptions must not error")
		assert.Len(t, subs, websocketMaxSubscriptionsPerConnection, "duplicates should be removed before the subscriptions are counted against a connection")
	})
}

func TestWebsocketGenerateAccountSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for a, ePath := range testAccountRESTPaths {
		t.Run(a.String()+" without authentication", func(t *testing.T) {
			t.Parallel()
			subs, err := ex.wsGenerateAccountSubscriptions(a, ePath)
			require.NoErrorf(t, err, "wsGenerateAccountSubscriptions must not error for %s without authentication", a)
			assert.Emptyf(t, subs, "%s should generate no account subscriptions without authenticated websocket use", a)
		})
	}

	for _, tc := range []struct {
		asset asset.Item
		exp   []string
	}{
		{asset.Spot, []string{"spot.executionReport", "ACCOUNT_UPDATE"}},
		{asset.USDTMarginedFutures, []string{"ORDER_TRADE_UPDATE", "ACCOUNT_UPDATE"}},
		{asset.CoinMarginedFutures, []string{"ORDER_TRADE_UPDATE", "ACCOUNT_UPDATE"}},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			subs, err := ex.wsGenerateAccountSubscriptions(tc.asset, testAccountRESTPaths[tc.asset])
			require.NoError(t, err, "wsGenerateAccountSubscriptions must not error")
			assert.ElementsMatch(t, tc.exp, subs.QualifiedChannels(), "the account streams should be generated")
			for _, s := range subs {
				assert.True(t, s.Authenticated, "account subscriptions should need authentication")
				assert.Empty(t, s.Pairs, "account subscriptions should have no pairs")
				assert.Equal(t, tc.asset, s.Asset, "account subscriptions should be of the asset")
			}
		})
	}

	t.Run("disabled asset", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated, testWebsocketAssets(asset.Spot))
		subs, err := ex.wsGenerateAccountSubscriptions(asset.USDTMarginedFutures, exchange.RestUSDTMargined)
		require.NoError(t, err, "wsGenerateAccountSubscriptions must not error")
		assert.Empty(t, subs, "a disabled asset should generate nothing")
	})

	t.Run("listen key requests waiting", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), "https://open-api-vst.bingx.com"), "SetRunningURL must not error")
		host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		ex.wsRecordListenKeyFailure(host)
		for a, exp := range map[asset.Item]int{asset.Spot: 0, asset.USDTMarginedFutures: 0, asset.CoinMarginedFutures: 2} {
			// The cases read the wait the failure above started, which the test ends below, so they run in turn
			t.Run(a.String(), func(t *testing.T) {
				subs, err := ex.wsGenerateAccountSubscriptions(a, testAccountRESTPaths[a])
				require.NoErrorf(t, err, "wsGenerateAccountSubscriptions must not error for %s", a)
				assert.Lenf(t, subs, exp, "%s should generate its account subscriptions only when its REST host's listen key requests are not waiting", a)
			})
		}
		ex.wsState.mu.Lock()
		ex.wsState.keyRetries[host] = websocketListenKeyRetry{wait: time.Second, next: time.Now().Add(-time.Millisecond)}
		ex.wsState.mu.Unlock()
		subs, err := ex.wsGenerateAccountSubscriptions(asset.Spot, exchange.RestSpot)
		require.NoError(t, err, "wsGenerateAccountSubscriptions must not error")
		assert.Len(t, subs, 2, "the account subscriptions should be generated once the wait has ended")
	})

	t.Run("credentials missing", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		ex.SetCredentials(&accounts.Credentials{})
		subs, err := ex.wsGenerateAccountSubscriptions(asset.Spot, exchange.RestSpot)
		require.NoError(t, err, "wsGenerateAccountSubscriptions must not error")
		assert.Empty(t, subs, "nothing should be generated without credentials")
	})

	t.Run("unsupported and duplicate subscriptions", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated, testWebsocketSubscriptions(
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		))
		subs, err := ex.wsGenerateAccountSubscriptions(asset.Spot, exchange.RestSpot)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "an unsupported account channel should be reported as partial generation")
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "the unsupported channel should be reported")
		assert.Equal(t, []string{"ACCOUNT_UPDATE"}, subs.QualifiedChannels(), "an account stream should be generated once")
	})
}

func TestWebsocketAccountStreamName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset   asset.Item
		channel string
		exp     string
	}{
		{asset.Spot, subscription.MyOrdersChannel, "spot.executionReport"},
		{asset.Spot, subscription.MyAccountChannel, "ACCOUNT_UPDATE"},
		{asset.USDTMarginedFutures, subscription.MyOrdersChannel, "ORDER_TRADE_UPDATE"},
		{asset.CoinMarginedFutures, subscription.MyAccountChannel, "ACCOUNT_UPDATE"},
	} {
		t.Run(tc.asset.String()+" "+tc.channel, func(t *testing.T) {
			t.Parallel()
			got, err := wsAccountStreamName(tc.asset, tc.channel)
			require.NoErrorf(t, err, "wsAccountStreamName must not error for %s %s", tc.asset, tc.channel)
			assert.Equalf(t, tc.exp, got, "wsAccountStreamName should name the %s %s stream", tc.asset, tc.channel)
		})
	}
	_, err := wsAccountStreamName(asset.Spot, subscription.MyTradesChannel)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "wsAccountStreamName should reject an unsupported channel")
}

func TestWebsocketDistinctSubscriptions(t *testing.T) {
	t.Parallel()
	first := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "BTC-USDT@ticker"}
	second := &subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "BTC-USDT@trade"}
	duplicate := &subscription.Subscription{Channel: subscription.TickerChannel, Levels: 5, QualifiedChannel: "BTC-USDT@ticker"}
	got := wsDistinctSubscriptions(subscription.List{first, second, duplicate})
	require.Len(t, got, 2, "wsDistinctSubscriptions must keep one subscription a stream")
	assert.Same(t, first, got[0], "the first subscription to a stream should be kept")
	assert.Same(t, second, got[1], "the order should be kept")
	assert.Empty(t, wsDistinctSubscriptions(nil), "wsDistinctSubscriptions should return nothing for nothing")
}

func TestWebsocketOneOrderbookSubscription(t *testing.T) {
	t.Parallel()
	depth20 := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	depth50 := testOrderbookSubscription(testSpotPair, asset.Spot, 50)
	swapDepth50 := testOrderbookSubscription(testSwapPair, asset.USDTMarginedFutures, 50)
	tickers := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: "BTC-USDT@ticker"}
	noPair := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot}
	kept, err := wsOneOrderbookSubscription(subscription.List{depth20, tickers, depth50, swapDepth50, noPair})
	assert.ErrorIs(t, err, errWebsocketOrderbookSubscriptionConflict, "a second order book subscription to a book should be reported")
	assert.ErrorIs(t, err, subscription.ErrNotSinglePair, "an order book subscription without one pair should be reported")
	assert.Equal(t, subscription.List{depth20, tickers, swapDepth50}, kept, "the first subscription to each book and the others should be kept")
	kept, err = wsOneOrderbookSubscription(subscription.List{tickers})
	assert.NoError(t, err, "wsOneOrderbookSubscription should not error without a conflict")
	assert.Equal(t, subscription.List{tickers}, kept, "subscriptions without a conflict should be kept")
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tpl, err := e.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	require.NotNil(t, tpl, "GetSubscriptionTemplate must return a template")

	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	ex := newWebsocketTestExchange(t, testWebsocketPairs(asset.Spot, currency.Pairs{testSpotPair, ethUSDT}))
	subs, err := subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour}}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.Len(t, subs, 2, "the template must name a stream for each pair")
	for _, s := range subs {
		require.Len(t, s.Pairs, 1, "each stream must be of one pair")
		assert.Equal(t, s.Pairs[0].String()+"@kline_60min", s.QualifiedChannel, "the stream should be named for its pair and the spot interval")
	}
}

func TestWebsocketStreamName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		a    asset.Item
		exp  string
		err  error
	}{
		{"ticker", &subscription.Subscription{Channel: subscription.TickerChannel}, asset.Spot, "ticker", nil},
		{"trades", &subscription.Subscription{Channel: subscription.AllTradesChannel}, asset.CoinMarginedFutures, "trade", nil},
		{"spot candles", &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMin}, asset.Spot, "kline_1min", nil},
		{"perpetual candles", &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth}, asset.USDTMarginedFutures, "kline_1M", nil},
		{"unsupported interval", &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, asset.Spot, "", errUnsupportedInterval},
		{"order book", &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 100}, asset.CoinMarginedFutures, "depth100", nil},
		{"order book levels", &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 7}, asset.Spot, "", errWebsocketUnsupportedLevels},
		{"order book without levels", &subscription.Subscription{Channel: subscription.OrderbookChannel}, asset.Spot, "", errWebsocketUnsupportedLevels},
		{"last price", &subscription.Subscription{Channel: WebsocketLastPriceChannel}, asset.Spot, "lastPrice", nil},
		{"book ticker", &subscription.Subscription{Channel: WebsocketBookTickerChannel}, asset.USDTMarginedFutures, "bookTicker", nil},
		{"perpetual mark price", &subscription.Subscription{Channel: WebsocketMarkPriceChannel}, asset.CoinMarginedFutures, "markPrice", nil},
		{"spot mark price", &subscription.Subscription{Channel: WebsocketMarkPriceChannel}, asset.Spot, "", subscription.ErrNotSupported},
		{"unsupported channel", &subscription.Subscription{Channel: subscription.MyTradesChannel}, asset.Spot, "", subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := wsStreamName(tc.sub, tc.a)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "wsStreamName should return the expected error")
				return
			}
			require.NoError(t, err, "wsStreamName must not error")
			assert.Equal(t, tc.exp, got, "wsStreamName should name the stream")
		})
	}
}

func TestWebsocketParseInterval(t *testing.T) {
	t.Parallel()
	for _, interval := range websocketKlineIntervals {
		for _, f := range []struct {
			streams string
			format  func(kline.Interval) (string, error)
		}{{"perpetual", intervalToString}, {"spot", spotWebsocketIntervalToString}} {
			t.Run(f.streams+" "+interval.Word(), func(t *testing.T) {
				t.Parallel()
				name, err := f.format(interval)
				require.NoErrorf(t, err, "formatting %s must not error", interval)
				got, err := wsParseInterval(name, f.format)
				require.NoErrorf(t, err, "wsParseInterval must not error for %s", name)
				assert.Equalf(t, interval, got, "wsParseInterval should parse %s", name)
			})
		}
	}
	_, err := wsParseInterval("1min", intervalToString)
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "wsParseInterval should reject an interval in another product's format")
}

func TestWebsocketUTCPlus8PeriodStart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 11, 14, 12, 34, 666e6, time.UTC)
	for interval, exp := range map[kline.Interval]time.Time{
		kline.OneMin:     time.Date(2026, 11, 11, 14, 12, 0, 0, time.UTC),
		kline.OneHour:    time.Date(2026, 11, 11, 14, 0, 0, 0, time.UTC),
		kline.FourHour:   time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC),
		kline.TwelveHour: time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC),
		kline.OneDay:     time.Date(2026, 11, 10, 16, 0, 0, 0, time.UTC),
		kline.ThreeDay:   time.Date(2026, 11, 9, 16, 0, 0, 0, time.UTC),
		kline.OneWeek:    time.Date(2026, 11, 8, 16, 0, 0, 0, time.UTC),
		kline.OneMonth:   time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC),
	} {
		t.Run(interval.Word(), func(t *testing.T) {
			t.Parallel()
			got := wsUTCPlus8PeriodStart(at, interval)
			assert.Equalf(t, exp, got, "%s periods should start as the spot candle stream starts them", interval)
			assert.Equalf(t, exp, wsUTCPlus8PeriodStart(got, interval), "a %s period's start should be its own period's start", interval)
		})
	}
	assert.Equal(t, time.Date(1969, 12, 31, 16, 0, 0, 0, time.UTC), wsUTCPlus8PeriodStart(time.Unix(0, 0).UTC(), kline.OneDay), "a time before the first period should fall in the period before it")
	assert.Equal(t, time.Date(2026, 11, 10, 16, 0, 0, 0, time.UTC).Local(), wsUTCPlus8PeriodStart(at.Local(), kline.OneDay), "the period start should be in the time's location")
	assert.Equal(t, time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC).Local(), wsUTCPlus8PeriodStart(at.Local(), kline.OneMonth), "the month's start should be in the time's location")
}

func TestWebsocketCandleValidation(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1794390720000)
	end := start.Add(time.Minute)
	assert.Equal(t, kline.PartialCandle, wsCandleValidation(end.Add(-time.Millisecond), start, kline.OneMin, time.UTC), "a candle before its end should be partial")
	assert.Empty(t, wsCandleValidation(end, start, kline.OneMin, time.UTC), "a candle at its end should not be partial")
	assert.Empty(t, wsCandleValidation(end.Add(time.Second), start, kline.OneMin, time.UTC), "a candle after its end should not be partial")
	month := time.UnixMilli(1793462400000)
	assert.Equal(t, kline.PartialCandle, wsCandleValidation(time.UnixMilli(1794406336378), month, kline.OneMonth, websocketUTCPlus8), "a month candle should be partial until the month has passed")

	december := time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC)
	nextMonth := time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC)
	assert.Equal(t, kline.PartialCandle, wsCandleValidation(nextMonth.Add(-time.Hour), december.Local(), kline.OneMonth, websocketUTCPlus8), "a 31 day month's candle should be partial on its last day")
	assert.Empty(t, wsCandleValidation(nextMonth, december.Local(), kline.OneMonth, websocketUTCPlus8), "a month candle should not be partial once the next month starts in the stream's time zone")
	utcDecember := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, kline.PartialCandle, wsCandleValidation(nextMonth, utcDecember, kline.OneMonth, time.UTC), "a month candle should be partial until the next month starts in UTC")
	assert.Empty(t, wsCandleValidation(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), utcDecember, kline.OneMonth, time.UTC), "a month candle should not be partial once the next month starts in UTC")
}

// testSubscriptionMock answers requests in a product's reply shape, refusing the dataTypes in reject
func testSubscriptionMock(a asset.Item, reject ...string) (mock mockws.WsMockFunc, received func() []WebsocketSubscriptionRequest) {
	var mu sync.Mutex
	var requests []WebsocketSubscriptionRequest
	return func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			req := testWebsocketRequest(tb, msg)
			mu.Lock()
			requests = append(requests, *req)
			mu.Unlock()
			if slices.Contains(reject, req.DataType) {
				code := int64(100400)
				if a == asset.USDTMarginedFutures {
					code = 80015
				}
				return w.WriteMessage(gws.BinaryMessage, testGzip(testWebsocketAck(a, req.ID, code, req.DataType+" dataType is not supported")))
			}
			return w.WriteMessage(gws.BinaryMessage, testGzip(testWebsocketAck(a, req.ID, 0, "SUCCESS")))
		}, func() []WebsocketSubscriptionRequest {
			mu.Lock()
			defer mu.Unlock()
			return slices.Clone(requests)
		}
}

func TestWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset   asset.Item
		filter  string
		serial  bool
		handler func(*Exchange, context.Context, websocket.Connection, []byte) error
		pair    currency.Pair
	}{
		{asset.Spot, websocketSpotFilter, false, (*Exchange).wsHandleSpotData, testSpotPair},
		{asset.USDTMarginedFutures, websocketSwapFilter, false, (*Exchange).wsHandleSwapData, testSwapPair},
		{asset.CoinMarginedFutures, websocketCoinMarginedFilter, true, (*Exchange).wsHandleCoinMarginedData, testCoinMarginedPair},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			rejected := tc.pair.String() + "@trade"
			mock, received := testSubscriptionMock(tc.asset, rejected)
			handler := func(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
				return tc.handler(ex, ctx, conn, respRaw)
			}
			conn, _ := newWebsocketMockConnection(t, ex, tc.filter, tc.serial, handler, mock)
			subs := subscription.List{
				{Channel: subscription.TickerChannel, Asset: tc.asset, Pairs: currency.Pairs{tc.pair}, QualifiedChannel: tc.pair.String() + "@ticker"},
				{Channel: subscription.AllTradesChannel, Asset: tc.asset, Pairs: currency.Pairs{tc.pair}, QualifiedChannel: rejected},
				testOrderbookSubscription(tc.pair, tc.asset, 20),
			}
			err := ex.wsSubscribe(t.Context(), conn, subs)
			assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "a refused subscription should be reported")
			assert.ErrorIs(t, err, errWebsocketRequestRejected, "the refusal should be reported")
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr, "the refusal must carry BingX's error")
			assert.Contains(t, apiErr.Message, rejected, "the refusal should carry BingX's message")
			assert.Equal(t, subscription.SubscribedState, subs[0].State(), "the other subscriptions should be subscribed")
			assert.Equal(t, subscription.SubscribedState, subs[2].State(), "the other subscriptions should be subscribed")
			assert.ElementsMatch(t, []string{subs[0].QualifiedChannel, subs[2].QualifiedChannel}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "only the accepted subscriptions should be stored")
			reqs := received()
			require.Len(t, reqs, 3, "each subscription must have a request of its own")
			for _, req := range reqs {
				assert.Equal(t, WebsocketRequestTypeSubscribe, req.RequestType, "each request should subscribe")
				assert.NotEmpty(t, req.ID, "each request should have an ID")
			}
			ex.wsState.booksMu.Lock()
			assert.Len(t, ex.wsState.books, 1, "the order book subscription should feed its book")
			ex.wsState.booksMu.Unlock()
		})
	}
}

func TestWebsocketSubscribeOne(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	mock, received := testSubscriptionMock(asset.Spot, "BTC-USDT@depth50")
	conn, _ := newWebsocketMockConnection(t, ex, websocketSpotFilter, false, ex.wsHandleSpotData, mock)

	err := ex.wsSubscribeOne(t.Context(), &testWebsocketConnection{}, testOrderbookSubscription(testSpotPair, asset.Spot, 20))
	assert.ErrorIs(t, err, errWebsocketConnectionUnknown, "wsSubscribeOne should refuse a connection no Connector dialled")

	other := &testWebsocketConnection{}
	otherState, _ := trackTestConnection(t, ex, other, false)
	require.NoError(t, ex.wsClaimBook(other, otherState.generation, testOrderbookSubscription(testSpotPair, asset.Spot, 5)), "wsClaimBook must not error")
	conflicting := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	err = ex.wsSubscribeOne(t.Context(), conn, conflicting)
	assert.ErrorIs(t, err, errWebsocketOrderbookSubscriptionConflict, "wsSubscribeOne should refuse a book another stream feeds")
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "the refusal should be a subscription failure")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "a refused subscription should not be stored")
	assert.Empty(t, received(), "a refused subscription should send nothing")

	other.down.Store(true)
	require.NoError(t, ex.wsSubscribeOne(t.Context(), conn, conflicting), "wsSubscribeOne must not error once the other feed's connection is closed")
	assert.Equal(t, subscription.SubscribedState, conflicting.State(), "the subscription should be subscribed")

	err = ex.wsSubscribeOne(t.Context(), conn, testOrderbookSubscription(testSpotPair, asset.Spot, 20))
	assert.ErrorIs(t, err, subscription.ErrDuplicate, "wsSubscribeOne should refuse a subscription the store holds")
	ex.wsState.booksMu.Lock()
	assert.Len(t, ex.wsState.books, 1, "the stored subscription should still feed its book")
	ex.wsState.booksMu.Unlock()

	refused := testOrderbookSubscription(testSwapPair, asset.USDTMarginedFutures, 50)
	refused.QualifiedChannel = "BTC-USDT@depth50"
	err = ex.wsSubscribeOne(t.Context(), conn, refused)
	assert.ErrorIs(t, err, errWebsocketRequestRejected, "wsSubscribeOne should return the refusal")
	ex.wsState.booksMu.Lock()
	assert.Len(t, ex.wsState.books, 1, "a refused order book subscription should release its book")
	ex.wsState.booksMu.Unlock()
	assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "a refused subscription should be removed from the store")
	ex.wsState.mu.Lock()
	recovering := ex.wsState.recovering
	ex.wsState.mu.Unlock()
	assert.Nil(t, recovering, "wsSubscribeOne should keep the connection after a refusal")

	for _, tc := range []struct {
		name    string
		a       asset.Item
		pair    currency.Pair
		err     error
		expired bool
	}{
		{"spot signature timeout", asset.Spot, testSpotPair, websocket.ErrSignatureTimeout, false},
		{"spot cancellation", asset.Spot, testSpotPair, context.Canceled, false},
		{"USDT-M deadline", asset.USDTMarginedFutures, testSwapPair, context.DeadlineExceeded, true},
		{"USDT-M network write timeout", asset.USDTMarginedFutures, testSwapPair, &net.OpError{Op: "write", Net: "tcp", Err: os.ErrDeadlineExceeded}, false},
		{"spot network write failure", asset.Spot, testSpotPair, &net.OpError{Op: "write", Net: "tcp", Err: errors.New("broken pipe")}, false},
	} {
		for _, s := range []*subscription.Subscription{
			testOrderbookSubscription(tc.pair, tc.a, 20),
			{Channel: subscription.TickerChannel, Asset: tc.a, Pairs: currency.Pairs{tc.pair}, QualifiedChannel: tc.pair.String() + "@ticker"},
		} {
			t.Run(s.Channel+" unanswered after "+tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t)
				conn := &testWebsocketConnection{}
				_, session := trackTestConnection(t, ex, conn, false)
				conn.reply = func([]byte) ([]byte, error) { return nil, tc.err }
				ctx := t.Context()
				if tc.expired {
					ctx = testExpiredContext(t)
				}
				err := ex.wsSubscribeOne(ctx, conn, s)
				assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsSubscribeOne should report the failed subscription")
				assert.ErrorIs(t, err, tc.err, "wsSubscribeOne should return why the subscription failed")
				assert.Empty(t, ex.Websocket.GetSubscriptions(), "wsSubscribeOne should remove the unanswered subscription from the store")
				ex.wsState.mu.Lock()
				recovering := ex.wsState.recovering
				ex.wsState.mu.Unlock()
				assert.Equal(t, session.token(), recovering, "wsSubscribeOne should retire the connection, so the websocket reconnects")
				testWaitBackground(t, ex)
			})
		}
	}

	t.Run("refused by the rate limiter before sending", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		trackTestConnection(t, ex, conn, false)
		conn.reply = func([]byte) ([]byte, error) { return nil, errTestRateLimitRefusal }
		s := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: testSpotPair.String() + "@ticker"}
		err := ex.wsSubscribeOne(testDeadlineContext(t), conn, s)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsSubscribeOne should report the failed subscription")
		assert.ErrorIs(t, err, context.DeadlineExceeded, "wsSubscribeOne should return why the subscription failed")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "wsSubscribeOne should remove a subscription it never sent")
		ex.wsState.mu.Lock()
		recovering := ex.wsState.recovering
		ex.wsState.mu.Unlock()
		assert.Nil(t, recovering, "wsSubscribeOne should keep the connection after a request it never sent")
	})
}

func TestWebsocketReleaseBookClaim(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	depth := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, depth), "wsClaimBook must not error")
	ex.wsReleaseBookClaim(conn, &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: depth.QualifiedChannel})
	ex.wsState.booksMu.Lock()
	assert.Len(t, ex.wsState.books, 1, "a subscription of another channel should release nothing")
	ex.wsState.booksMu.Unlock()
	ex.wsReleaseBookClaim(conn, depth)
	ex.wsState.booksMu.Lock()
	assert.Empty(t, ex.wsState.books, "an order book subscription should release its book")
	ex.wsState.booksMu.Unlock()
}

func TestWebsocketUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	mock, received := testSubscriptionMock(asset.Spot, "BTC-USDT@trade")
	conn, _ := newWebsocketMockConnection(t, ex, websocketSpotFilter, false, ex.wsHandleSpotData, mock)
	depth := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
	trades := &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: "BTC-USDT@trade"}
	require.NoError(t, ex.wsSubscribe(t.Context(), conn, subscription.List{depth}), "wsSubscribe must not error")
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, trades), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.wsLoadBook(t.Context(), conn, depth.QualifiedChannel, testBook(ex, testSpotPair, asset.Spot, 100)), "wsLoadBook must not error")

	err := ex.wsUnsubscribe(t.Context(), conn, subscription.List{depth, trades})
	assert.ErrorIs(t, err, errWebsocketRequestRejected, "a refused unsubscription should be reported")
	assert.Equal(t, []string{"BTC-USDT@trade"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "only the accepted unsubscription should be removed")
	assert.Equal(t, subscription.UnsubscribedState, depth.State(), "the unsubscribed subscription should be unsubscribed")
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "unsubscribing a book's stream should invalidate the book")
	var unsubscribed int
	for _, req := range received() {
		if req.RequestType == WebsocketRequestTypeUnsubscribe {
			unsubscribed++
		}
	}
	assert.Equal(t, 2, unsubscribed, "each unsubscription should have a request of its own")
}

func TestWebsocketUnsubscribeOne(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	err := ex.wsUnsubscribeOne(t.Context(), &testWebsocketConnection{}, &subscription.Subscription{})
	assert.ErrorIs(t, err, errWebsocketConnectionUnknown, "wsUnsubscribeOne should refuse a connection no Connector dialled")

	conn := &testWebsocketConnection{}
	trackTestConnection(t, ex, conn, false)
	conn.reply = func(payload []byte) ([]byte, error) {
		req := testWebsocketRequest(t, payload)
		return testWebsocketAck(asset.Spot, req.ID, 0, "SUCCESS"), nil
	}
	s := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: "BTC-USDT@ticker"}
	require.NoError(t, s.SetState(subscription.UnsubscribingState), "SetState must not error")
	assert.ErrorIs(t, ex.wsUnsubscribeOne(t.Context(), conn, s), subscription.ErrInStateAlready, "wsUnsubscribeOne should refuse a subscription already unsubscribing")

	s = &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, QualifiedChannel: "BTC-USDT@ticker"}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.wsUnsubscribeOne(t.Context(), conn, s), "wsUnsubscribeOne must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "the subscription should be removed")
	msgs := conn.messages()
	require.Len(t, msgs, 1, "one request must be sent")
	assert.Equal(t, websocketSubscriptionRate, msgs[0].limit, "unsubscriptions should draw on the subscription budget")
	assert.Equal(t, WebsocketRequestTypeUnsubscribe, testWebsocketRequest(t, []byte(msgs[0].message)).RequestType, "the request should unsubscribe")

	t.Run("refused", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		trackTestConnection(t, ex, conn, false)
		var refuse atomic.Bool
		refuse.Store(true)
		conn.reply = func(payload []byte) ([]byte, error) {
			req := testWebsocketRequest(t, payload)
			if refuse.Load() {
				return testWebsocketAck(asset.Spot, req.ID, 100400, "request parse error"), nil
			}
			return testWebsocketAck(asset.Spot, req.ID, 0, "SUCCESS"), nil
		}
		book := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, book), "AddSuccessfulSubscriptions must not error")
		claimAndLoadBook(t, ex, conn, book)
		err := ex.wsUnsubscribeOne(t.Context(), conn, book)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsUnsubscribeOne should report the failed unsubscription")
		assert.ErrorIs(t, err, errWebsocketRequestRejected, "wsUnsubscribeOne should return why the unsubscription failed")
		assert.Equal(t, subscription.SubscribedState, book.State(), "wsUnsubscribeOne should leave a subscription it failed to unsubscribe subscribed")
		assert.Equal(t, []string{book.QualifiedChannel}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "wsUnsubscribeOne should keep the subscription stored")
		assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "wsUnsubscribeOne should keep the book its stream still feeds")
		assert.NoError(t, ex.wsLoadBook(t.Context(), conn, book.QualifiedChannel, testBook(ex, testSpotPair, asset.Spot, 101)), "wsLoadBook should still load the stream's pushes")
		book2, err := ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
		require.NoError(t, err, "GetOrderbook must not error")
		assert.Equal(t, 100.0, book2.Bids[0].Price, "wsUnsubscribeOne should keep the book's claim with the subscription, so its pushes load")
		ex.wsState.mu.Lock()
		recovering := ex.wsState.recovering
		ex.wsState.mu.Unlock()
		assert.Nil(t, recovering, "wsUnsubscribeOne should keep the connection after a refusal")

		refuse.Store(false)
		require.NoError(t, ex.wsUnsubscribeOne(t.Context(), conn, book), "wsUnsubscribeOne must unsubscribe a subscription whose unsubscription failed before")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "wsUnsubscribeOne should remove the subscription once unsubscribed")
		assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "wsUnsubscribeOne should invalidate the book once unsubscribed")
	})

	for _, tc := range []struct {
		name    string
		a       asset.Item
		pair    currency.Pair
		err     error
		expired bool
	}{
		{"spot signature timeout", asset.Spot, testSpotPair, websocket.ErrSignatureTimeout, false},
		{"spot cancellation", asset.Spot, testSpotPair, context.Canceled, false},
		{"USDT-M deadline", asset.USDTMarginedFutures, testSwapPair, context.DeadlineExceeded, true},
		{"USDT-M network write timeout", asset.USDTMarginedFutures, testSwapPair, &net.OpError{Op: "write", Net: "tcp", Err: os.ErrDeadlineExceeded}, false},
		{"spot network write failure", asset.Spot, testSpotPair, &net.OpError{Op: "write", Net: "tcp", Err: errors.New("broken pipe")}, false},
	} {
		t.Run("unanswered after "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			conn := &testWebsocketConnection{}
			_, session := trackTestConnection(t, ex, conn, false)
			conn.reply = func([]byte) ([]byte, error) { return nil, tc.err }
			book := testOrderbookSubscription(tc.pair, tc.a, 20)
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, book), "AddSuccessfulSubscriptions must not error")
			claimAndLoadBook(t, ex, conn, book)
			ctx := t.Context()
			if tc.expired {
				ctx = testExpiredContext(t)
			}
			err := ex.wsUnsubscribeOne(ctx, conn, book)
			assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsUnsubscribeOne should report the failed unsubscription")
			assert.ErrorIs(t, err, tc.err, "wsUnsubscribeOne should return why the unsubscription failed")
			assert.Equal(t, subscription.SubscribedState, book.State(), "wsUnsubscribeOne should leave an unanswered subscription subscribed until the reconnection")
			assert.Equal(t, []string{book.QualifiedChannel}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "wsUnsubscribeOne should keep the subscription stored until the reconnection")
			assert.ErrorIs(t, bookError(ex, tc.pair, tc.a), orderbook.ErrOrderbookInvalid, "wsUnsubscribeOne should invalidate the book of a stream that may have stopped")
			require.NoError(t, ex.wsLoadBook(t.Context(), conn, book.QualifiedChannel, testBook(ex, tc.pair, tc.a, 101)), "wsLoadBook must not error")
			assert.ErrorIs(t, bookError(ex, tc.pair, tc.a), orderbook.ErrOrderbookInvalid, "wsUnsubscribeOne should leave the book invalid until the reconnection")
			ex.wsState.mu.Lock()
			recovering := ex.wsState.recovering
			ex.wsState.mu.Unlock()
			assert.Equal(t, session.token(), recovering, "wsUnsubscribeOne should retire the connection, so the websocket reconnects")
			testWaitBackground(t, ex)
		})
	}

	t.Run("refused by the rate limiter before sending", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		trackTestConnection(t, ex, conn, false)
		conn.reply = func([]byte) ([]byte, error) { return nil, errTestRateLimitRefusal }
		book := testOrderbookSubscription(testSpotPair, asset.Spot, 20)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, book), "AddSuccessfulSubscriptions must not error")
		claimAndLoadBook(t, ex, conn, book)
		err := ex.wsUnsubscribeOne(testDeadlineContext(t), conn, book)
		assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "wsUnsubscribeOne should report the failed unsubscription")
		assert.ErrorIs(t, err, context.DeadlineExceeded, "wsUnsubscribeOne should return why the unsubscription failed")
		assert.Equal(t, subscription.SubscribedState, book.State(), "wsUnsubscribeOne should leave a subscription it never unsubscribed subscribed")
		assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "wsUnsubscribeOne should keep the book of a stream it never stopped")
		ex.wsState.mu.Lock()
		recovering := ex.wsState.recovering
		ex.wsState.mu.Unlock()
		assert.Nil(t, recovering, "wsUnsubscribeOne should keep the connection after a request it never sent")
	})

	t.Run("refusal through the manager", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		tickers := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}
		ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, false, testWebsocketSubscriptions(tickers,
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20}))
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		require.Eventually(t, func() bool { return bookError(ex, testSpotPair, asset.Spot) == nil }, 5*time.Second, time.Millisecond, "the book must load")
		ex.wsState.mu.Lock()
		conns := slices.Collect(maps.Keys(ex.wsState.connections))
		ex.wsState.mu.Unlock()
		require.Len(t, conns, 1, "one market connection must be dialled")
		conn := conns[0]
		book := ex.Websocket.GetSubscription(subscription.ExactKey{Subscription: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, Levels: 20}})
		require.NotNil(t, book, "the book's subscription must be stored")

		servers.ws[asset.Spot].rejectUnsubscribe.Store(true)
		ex.Features.Subscriptions = subscription.List{tickers}
		assert.ErrorIs(t, ex.Websocket.FlushChannels(t.Context()), websocket.ErrSubscriptionFailure, "FlushChannels should report the refused unsubscription")
		assert.Equal(t, subscription.SubscribedState, book.State(), "wsUnsubscribeOne should leave the subscription subscribed")
		assert.Same(t, book, ex.Websocket.GetSubscription(book), "wsUnsubscribeOne should keep the subscription in the manager's store")
		assert.Same(t, book, conn.Subscriptions().Get(book), "wsUnsubscribeOne should keep the subscription in the connection's store")
		assert.NoError(t, bookError(ex, testSpotPair, asset.Spot), "wsUnsubscribeOne should keep the book its stream still feeds")

		servers.ws[asset.Spot].rejectUnsubscribe.Store(false)
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must unsubscribe once BingX accepts the unsubscription")
		assert.Nil(t, ex.Websocket.GetSubscription(book), "the subscription should leave the manager's store")
		assert.Nil(t, conn.Subscriptions().Get(book), "the subscription should leave the connection's store")
		assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "the book should be invalidated once unsubscribed")
	})

	t.Run("unanswered through the manager", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		tickers := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}
		ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, false, testWebsocketSubscriptions(tickers,
			&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20}), func(cfg *config.Exchange) {
			cfg.ConnectionMonitorDelay = 50 * time.Millisecond
			cfg.WebsocketResponseMaxLimit = 200 * time.Millisecond
		})
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		require.Eventually(t, func() bool { return bookError(ex, testSpotPair, asset.Spot) == nil }, 5*time.Second, time.Millisecond, "the book must load")
		first := testConnectedSession(t, ex)

		servers.ws[asset.Spot].ignoreUnsubscribe.Store(true)
		ex.Features.Subscriptions = subscription.List{tickers}
		assert.ErrorIs(t, ex.Websocket.FlushChannels(t.Context()), websocket.ErrSignatureTimeout, "FlushChannels should report the unanswered unsubscription")
		assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "the book of a stream that may have stopped should be invalidated")
		require.Eventually(t, func() bool {
			select {
			case <-first:
				return true
			default:
				return false
			}
		}, 5*time.Second, time.Millisecond, "the unanswered unsubscription must retire the connection's session")
		require.Eventually(t, func() bool {
			return servers.ws[asset.Spot].marketDials() == 2 && ex.Websocket.IsConnected() && len(ex.Websocket.GetSubscriptions()) == 1
		}, 5*time.Second, 10*time.Millisecond, "the connection monitor must connect again")
		testWaitBackground(t, ex)
		assert.Equal(t, []string{"BTC-USDT@ticker"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "the reconnection should subscribe to what is still wanted alone")
		assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "the book should stay invalid once nothing streams it")
	})
}

func TestWebsocketTrackPushedSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	subs := subscription.List{
		{Channel: subscription.MyOrdersChannel, Asset: asset.USDTMarginedFutures, Authenticated: true, QualifiedChannel: "ORDER_TRADE_UPDATE"},
		{Channel: subscription.MyAccountChannel, Asset: asset.USDTMarginedFutures, Authenticated: true, QualifiedChannel: "ACCOUNT_UPDATE"},
	}
	require.NoError(t, ex.wsTrackPushedSubscriptions(t.Context(), conn, subs), "wsTrackPushedSubscriptions must not error")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "the subscriptions should be stored")
	for _, s := range subs {
		t.Run(s.QualifiedChannel, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, subscription.SubscribedState, s.State(), "the subscriptions should be subscribed")
		})
	}
	assert.Empty(t, conn.messages(), "nothing should be sent")
}

func TestWebsocketUntrackPushedSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	subs := subscription.List{{Channel: subscription.MyOrdersChannel, Asset: asset.CoinMarginedFutures, Authenticated: true, QualifiedChannel: "ORDER_TRADE_UPDATE"}}
	require.NoError(t, ex.wsTrackPushedSubscriptions(t.Context(), conn, subs), "wsTrackPushedSubscriptions must not error")
	require.NoError(t, ex.wsUntrackPushedSubscriptions(t.Context(), conn, subs), "wsUntrackPushedSubscriptions must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "the subscriptions should be removed")
	assert.Equal(t, subscription.UnsubscribedState, subs[0].State(), "the subscriptions should be unsubscribed")
	assert.Empty(t, conn.messages(), "nothing should be sent")
}

func TestWebsocketSendRequest(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, tc := range []struct {
		name  string
		reply func(id string) string
		err   error
		code  int64
	}{
		{"spot and coin-M reply", func(id string) string { return string(testWebsocketAck(asset.Spot, id, 0, "SUCCESS")) }, nil, 0},
		{"USDT-M reply", func(id string) string { return string(testWebsocketAck(asset.USDTMarginedFutures, id, 0, "")) }, nil, 0},
		{"spot refusal", func(id string) string {
			return fmt.Sprintf(`{"code":100400,"id":%q,"msg":"dataType is error: BTC-USDT@nonsense","timestamp":1794390797881}`, id)
		}, errWebsocketRequestRejected, 100400},
		{"USDT-M refusal", func(id string) string {
			return fmt.Sprintf(`{"id":%q,"code":80015,"msg":"dataType not support","dataType":"","data":null}`, id)
		}, errWebsocketRequestRejected, 80015},
		{"malformed reply", func(id string) string { return fmt.Sprintf(`{"id":%q,"code":"x"}`, id) }, nil, -1},
		{"null reply", func(string) string { return "null" }, errWebsocketRequestRejected, -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conn := &testWebsocketConnection{}
			state, _ := trackTestConnection(t, ex, conn, false)
			conn.reply = func(payload []byte) ([]byte, error) {
				return []byte(tc.reply(testWebsocketRequest(t, payload).ID)), nil
			}
			err := ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USDT@ticker")
			switch {
			case tc.code == -1:
				assert.Error(t, err, "wsSendRequest should error for a reply it cannot decode")
			case tc.code == -2:
				assert.ErrorIs(t, err, tc.err, "wsSendRequest should reject an empty reply")
			case tc.err != nil:
				assert.ErrorIs(t, err, tc.err, "wsSendRequest should return the refusal")
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "the refusal must carry BingX's error")
				assert.Equal(t, tc.code, apiErr.Code, "the refusal should carry BingX's code")
			default:
				assert.NoError(t, err, "wsSendRequest should not error for an accepted request")
			}
			msgs := conn.messages()
			require.Len(t, msgs, 1, "one request must be sent")
			assert.Equal(t, websocketSubscriptionRate, msgs[0].limit, "requests should draw on the subscription budget")
			req := testWebsocketRequest(t, []byte(msgs[0].message))
			assert.Equal(t, WebsocketRequestTypeSubscribe, req.RequestType, "the request should have its type")
			assert.Equal(t, "BTC-USDT@ticker", req.DataType, "the request should name its stream")
		})
	}

	t.Run("send failure", func(t *testing.T) {
		t.Parallel()
		conn := &testWebsocketConnection{}
		state, _ := trackTestConnection(t, ex, conn, false)
		assert.ErrorIs(t, ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USDT@ticker"), errTestDial, "wsSendRequest should return a failed send")
	})
}

func TestWebsocketSendSerialRequest(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t, func(cfg *config.Exchange) { cfg.WebsocketResponseMaxLimit = 100 * time.Millisecond })
	var writes sync.Mutex
	var received []string
	lateErrorSent := make(chan struct{})
	conn, session := newWebsocketMockConnection(t, ex, websocketCoinMarginedFilter, true, ex.wsHandleCoinMarginedData, func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		req := testWebsocketRequest(tb, msg)
		writes.Lock()
		defer writes.Unlock()
		received = append(received, req.DataType)
		switch req.DataType {
		case "BTC-USD@nonsense":
			return w.WriteMessage(gws.BinaryMessage, testGzip([]byte(`{"code":100400,"msg":"code:100400:nonsense dataType is not supported","timestamp":1794390762853}`)))
		case "BTC-USD@depth20":
			// Answered after the request has timed out
			go func() {
				defer close(lateErrorSent)
				time.Sleep(300 * time.Millisecond)
				writes.Lock()
				defer writes.Unlock()
				assert.NoError(tb, w.WriteMessage(gws.BinaryMessage, testGzip([]byte(`{"code":100400,"msg":"code:100400: symbol is not found: BTC_USD","timestamp":1794390798083}`))), "the late reply should be written")
			}()
			return nil
		default:
			return w.WriteMessage(gws.BinaryMessage, testGzip(testWebsocketAck(asset.CoinMarginedFutures, req.ID, 0, "SUCCESS")))
		}
	})
	state, err := ex.wsConnection(conn)
	require.NoError(t, err, "wsConnection must not error")

	assert.NoError(t, ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USD@trade"), "an acknowledged request should not error")
	err = ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USD@nonsense")
	assert.ErrorIs(t, err, errWebsocketRequestRejected, "an error reply naming the request in flight should be its refusal")

	err = ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USD@depth20")
	assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "an unanswered request should time out")
	state.mu.Lock()
	assert.True(t, state.retired, "an unanswered request should retire its connection")
	assert.Empty(t, state.inFlightID, "nothing should be in flight after the timeout")
	state.mu.Unlock()
	ex.wsState.mu.Lock()
	assert.Equal(t, session.token(), ex.wsState.recovering, "an unanswered request should reconnect its session")
	ex.wsState.mu.Unlock()

	select {
	case <-lateErrorSent:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the late reply must be sent")
	}
	late := waitForWebsocketPayload(t, ex, func(err error) bool { return errors.Is(err, errWebsocketErrorMessage) })
	assert.Contains(t, late.Error(), "BTC_USD", "the late reply should be reported rather than taken for another request's")

	err = ex.wsSendRequest(t.Context(), conn, state, WebsocketRequestTypeSubscribe, "BTC-USD@ticker")
	assert.ErrorIs(t, err, errWebsocketConnectionRetired, "a retired connection should refuse later requests at once")
	writes.Lock()
	assert.Equal(t, []string{"BTC-USD@trade", "BTC-USD@nonsense", "BTC-USD@depth20"}, received, "a retired connection should send nothing more")
	writes.Unlock()
	testWaitBackground(t, ex)

	t.Run("cancelled while another request is in flight", func(t *testing.T) {
		t.Parallel()
		conn := &testWebsocketConnection{}
		state, _ := trackTestConnection(t, ex, conn, true)
		state.gate <- struct{}{}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := ex.wsSendSerialRequest(ctx, conn, state, &WebsocketSubscriptionRequest{ID: "1", RequestType: WebsocketRequestTypeSubscribe, DataType: "BTC-USD@trade"})
		assert.ErrorIs(t, err, context.Canceled, "a request waiting for the one in flight should end with its context")
		assert.Empty(t, conn.messages(), "a request that never went out should send nothing")
		state.mu.Lock()
		assert.False(t, state.retired, "a request that never went out should not retire the connection")
		state.mu.Unlock()
	})

	t.Run("sent request ending at its context's deadline", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		state, session := trackTestConnection(t, ex, conn, true)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		conn.reply = func([]byte) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		_, err := ex.wsSendSerialRequest(ctx, conn, state, &WebsocketSubscriptionRequest{ID: "1", RequestType: WebsocketRequestTypeSubscribe, DataType: "BTC-USD@trade"})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "a sent request should end with its context's deadline")
		state.mu.Lock()
		assert.True(t, state.retired, "a sent request left without a reply should retire the connection")
		state.mu.Unlock()
		ex.wsState.mu.Lock()
		assert.Equal(t, session.token(), ex.wsState.recovering, "a sent request left without a reply should reconnect its session")
		ex.wsState.mu.Unlock()
		testWaitBackground(t, ex)
	})

	t.Run("refused by the rate limiter before sending", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		state, _ := trackTestConnection(t, ex, conn, true)
		conn.reply = func([]byte) ([]byte, error) { return nil, errTestRateLimitRefusal }
		_, err := ex.wsSendSerialRequest(testDeadlineContext(t), conn, state, &WebsocketSubscriptionRequest{ID: "1", RequestType: WebsocketRequestTypeSubscribe, DataType: "BTC-USD@trade"})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "a refused request should return the rate limiter's error")
		state.mu.Lock()
		assert.False(t, state.retired, "a request the rate limiter refused should not retire the connection")
		state.mu.Unlock()
		ex.wsState.mu.Lock()
		assert.Nil(t, ex.wsState.recovering, "a request the rate limiter refused should not reconnect its session")
		ex.wsState.mu.Unlock()
	})
}

func TestWebsocketRouteCoinMarginedError(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	frame := []byte(`{"code":100400,"msg":"code:100400: symbol is not found: NOPE_USD","timestamp":1794390798083}`)

	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, true)
	state.inFlightID, state.inFlightDataType = "req-1", "NOPE-USD@trade"
	require.NoError(t, ex.wsRouteCoinMarginedError(conn, frame, 100400), "an error naming the request in flight must be routed to it")
	assert.Equal(t, []any{"req-1"}, conn.matches(), "the error should be matched to the request in flight")

	state.inFlightDataType = "BTC-USD@trade"
	err := ex.wsRouteCoinMarginedError(conn, frame, 100400)
	assert.ErrorIs(t, err, errWebsocketErrorMessage, "an error naming another request should be reported")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "the report must carry BingX's error")
	assert.Equal(t, int64(100400), apiErr.Code, "the report should carry BingX's code")

	state.inFlightID, state.inFlightDataType = "", ""
	assert.ErrorIs(t, ex.wsRouteCoinMarginedError(conn, frame, 100400), errWebsocketErrorMessage, "an error with nothing in flight should be reported")

	other := &testWebsocketConnection{}
	otherState, _ := trackTestConnection(t, ex, other, false)
	otherState.inFlightID, otherState.inFlightDataType = "req-2", "NOPE-USD@trade"
	assert.ErrorIs(t, ex.wsRouteCoinMarginedError(other, frame, 100400), errWebsocketErrorMessage, "an error on a connection without serial requests should be reported")
	assert.ErrorIs(t, ex.wsRouteCoinMarginedError(&testWebsocketConnection{}, frame, 100400), errWebsocketErrorMessage, "an error on an unknown connection should be reported")
	assert.Empty(t, other.matches(), "nothing should be matched on a connection without serial requests")
}

func TestWebsocketErrorNamesRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, msg, dataType string
		exp                 bool
	}{
		{"symbol with an underscore", "code:100400: symbol is not found: BTC_USD", "BTC-USD@trade", true},
		{"symbol with a dash", "symbol is not found: BTC-USD", "BTC-USD@trade", true},
		{"stream", "code:100400:nonsense dataType is not supported", "BTC-USD@nonsense", true},
		{"candle interval", "code:100400:peroid is not supported: 1min", "BTC-USD@kline_1min", true}, // codespell:ignore peroid // BingX's spelling in its reply
		{"another request's stream", "code:100400:nonsense dataType is not supported", "ETH-USD@trade", false},
		{"empty dataType", "code:100400:something else", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, wsErrorNamesRequest(tc.msg, tc.dataType), "wsErrorNamesRequest should report %t for %q and %q", tc.exp, tc.msg, tc.dataType)
		})
	}
}

// testExpiredContext returns a context whose deadline has passed, as a request's has once its wait for a reply ends at
// the deadline
func testExpiredContext(tb testing.TB) context.Context {
	tb.Helper()
	ctx, cancel := context.WithDeadline(tb.Context(), time.Now().Add(-time.Second))
	tb.Cleanup(cancel)
	return ctx
}

// testDeadlineContext returns a context whose deadline is an hour away, as a request's is when the rate limiter refuses
// to wait past it
func testDeadlineContext(tb testing.TB) context.Context {
	tb.Helper()
	ctx, cancel := context.WithTimeout(tb.Context(), time.Hour)
	tb.Cleanup(cancel)
	return ctx
}

// errTestRateLimitRefusal is the error the shared connection returns when its rate limiter refuses, before sending, to
// wait past the request's deadline
var errTestRateLimitRefusal = fmt.Errorf("BingX websocket connection: rate limit error: %w", fmt.Errorf("rate limit delay of 1s will exceed deadline: %w", context.DeadlineExceeded))

func TestWebsocketRequestUnanswered(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		err     error
		expired bool
	}{
		{"signature timeout", websocket.ErrSignatureTimeout, false},
		{"cancelled", context.Canceled, false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped signature timeout", fmt.Errorf("%w: BTC-USDT@depth20", websocket.ErrSignatureTimeout), false},
		{"network write timeout", &net.OpError{Op: "write", Net: "tcp", Err: os.ErrDeadlineExceeded}, false},
		{"wrapped network write failure", fmt.Errorf("sending: %w", &net.OpError{Op: "write", Net: "tcp", Err: errors.New("broken pipe")}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if tc.expired {
				ctx = testExpiredContext(t)
			}
			assert.Truef(t, wsRequestUnanswered(ctx, tc.err), "wsRequestUnanswered should report a request ending with %v as unanswered", tc.err)
		})
	}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"no error", nil},
		{"rejected", errWebsocketRequestRejected},
		{"connection retired", errWebsocketConnectionRetired},
		{"not connected", websocket.ErrNotConnected},
		{"rate limiter refusal before sending", errTestRateLimitRefusal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Falsef(t, wsRequestUnanswered(testDeadlineContext(t), tc.err), "wsRequestUnanswered should not report a request ending with %v as unanswered", tc.err)
		})
	}
}

func TestWebsocketRequestRecovery(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.False(t, ex.wsRequestRecovery(nil, errTestDial), "wsRequestRecovery should do nothing without a session")
	session := newTestSession(t, ex)
	assert.True(t, ex.wsRequestRecovery(session.ch, errTestDial), "wsRequestRecovery should recover a session")
	assert.False(t, ex.wsRequestRecovery(session.ch, errTestDial), "wsRequestRecovery should recover a session once")
	testWaitBackground(t, ex)

	t.Run("connected websocket", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, false, func(cfg *config.Exchange) {
			cfg.ConnectionMonitorDelay = 50 * time.Millisecond
		})
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		first := testConnectedSession(t, ex)
		assert.True(t, ex.wsRequestRecovery(first, errTestDial), "wsRequestRecovery should recover the connected session")
		require.Eventually(t, func() bool {
			select {
			case <-first:
				return true
			default:
				return false
			}
		}, 5*time.Second, time.Millisecond, "the recovery must tear the session down")
		require.Eventually(t, ex.Websocket.IsConnected, 5*time.Second, 10*time.Millisecond, "the connection monitor must connect again")
		testWaitBackground(t, ex)
		assert.Equal(t, 2, servers.ws[asset.Spot].marketDials(), "the market connection should be dialled again")

		second := testConnectedSession(t, ex)
		assert.True(t, ex.wsRequestRecovery(first, errTestDial), "a request for a session torn down since its recovery should start")
		testWaitBackground(t, ex)
		assert.True(t, ex.Websocket.IsConnected(), "a recovery of a session already torn down should leave the new one connected")
		select {
		case <-second:
			assert.Fail(t, "the new session should not be torn down")
		default:
		}
	})
}

// testConnectedSession returns the session of the websocket's connections
func testConnectedSession(tb testing.TB, ex *Exchange) <-chan struct{} {
	tb.Helper()
	ex.wsState.mu.Lock()
	defer ex.wsState.mu.Unlock()
	for _, state := range ex.wsState.connections {
		return state.session
	}
	require.FailNow(tb, "a connection must be tracked")
	return nil
}

func TestWebsocketSession(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	assert.Nil(t, ex.wsSession(conn), "wsSession should return nothing for an unknown connection")
	_, session := trackTestConnection(t, ex, conn, false)
	assert.Equal(t, session.token(), ex.wsSession(conn), "wsSession should return the connection's session")
}

func TestWebsocketListenKey(t *testing.T) {
	t.Parallel()
	rest := newTestListenKeyServer(t)
	vst := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, rest.URL)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), vst.URL), "SetRunningURL must not error")
	session := newTestSession(t, ex)

	spot, err := ex.wsListenKey(t.Context(), exchange.RestSpot, session.ch)
	require.NoError(t, err, "wsListenKey must not error")
	assert.Equal(t, testListenKey+"-1", spot.key, "wsListenKey should create a key")
	assert.Equal(t, exchange.RestSpot, spot.ePath, "the keeper should extend through the endpoint that created its key")
	swap, err := ex.wsListenKey(t.Context(), exchange.RestUSDTMargined, session.ch)
	require.NoError(t, err, "wsListenKey must not error")
	assert.Same(t, spot, swap, "products on one REST host should share a keeper")
	coinMargined, err := ex.wsListenKey(t.Context(), exchange.RestCoinMargined, session.ch)
	require.NoError(t, err, "wsListenKey must not error")
	assert.NotSame(t, spot, coinMargined, "another REST host should have its own keeper")
	created, _ := rest.counts()
	assert.Equal(t, 1, created, "one key should be created for a host")
	created, _ = vst.counts()
	assert.Equal(t, 1, created, "the other host should create its own key")

	next := newTestSession(t, ex)
	renewed, err := ex.wsListenKey(t.Context(), exchange.RestSpot, next.ch)
	require.NoError(t, err, "wsListenKey must not error")
	assert.Equal(t, testListenKey+"-2", renewed.key, "a new session should take a new key")

	t.Run("failure waits", func(t *testing.T) {
		t.Parallel()
		failing := newTestListenKeyServer(t)
		failing.setReplies(`{"code":100500,"msg":"internal error"}`, "")
		waiting := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		waiting.wsState.retryDelay = time.Hour
		testRESTEndpoints(t, waiting, failing.URL)
		waitingSession := newTestSession(t, waiting)
		t.Cleanup(waiting.wsState.background.Wait)
		_, err := waiting.wsListenKey(t.Context(), exchange.RestSpot, waitingSession.ch)
		assert.ErrorIs(t, err, errWebsocketListenKeyUnavailable, "a failed key request should be reported")
		host, err := waiting.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		waiting.wsState.mu.Lock()
		assert.Equal(t, waiting.wsState.keyRetries[host].next, waiting.wsState.reconnectAt, "a failure should schedule the account streams' return for when the wait ends")
		waiting.wsState.mu.Unlock()
		_, err = waiting.wsListenKey(t.Context(), exchange.RestUSDTMargined, waitingSession.ch)
		assert.ErrorIs(t, err, errWebsocketListenKeyBackoff, "a request soon after a failure should wait")
		created, _ := failing.counts()
		assert.Equal(t, 1, created, "a waiting request should not reach BingX")
		assert.True(t, waiting.Websocket.CanUseAuthenticatedEndpoints(), "a failure other than refused credentials should leave authenticated use on")
	})

	t.Run("credentials refused", func(t *testing.T) {
		t.Parallel()
		refusing := newTestListenKeyServer(t)
		refusing.setReplies(`{"code":100413,"msg":"Incorrect apiKey"}`, "")
		refused := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, refused, refusing.URL)
		refusedSession := newTestSession(t, refused)
		_, err := refused.wsListenKey(t.Context(), exchange.RestSpot, refusedSession.ch)
		assert.ErrorIs(t, err, errWebsocketListenKeyUnavailable, "a refused key request should be reported")
		assert.False(t, refused.Websocket.CanUseAuthenticatedEndpoints(), "refused credentials should turn authenticated websocket use off")
		refused.wsState.mu.Lock()
		assert.Empty(t, refused.wsState.keyRetries, "refused credentials should not make later requests wait")
		assert.Zero(t, refused.wsState.reconnectAt, "refused credentials should schedule nothing")
		refused.wsState.mu.Unlock()
	})

	t.Run("missing endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := ex.wsListenKey(t.Context(), exchange.RestFutures, session.ch)
		assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "wsListenKey should error for an unset endpoint")
	})

	t.Run("credentials", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		session := newTestSession(t, ex)
		first, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		k, err := ex.wsListenKey(t.Context(), exchange.RestSpot, session.ch)
		require.NoError(t, err, "wsListenKey must not error")
		assert.Equal(t, first, k.credentials, "the keeper should keep the credentials its key was created for")
		ex.SetCredentials(&accounts.Credentials{Key: "second-key", Secret: "second-secret"})
		shared, err := ex.wsListenKey(t.Context(), exchange.RestUSDTMargined, session.ch)
		require.NoError(t, err, "wsListenKey must not error")
		assert.Same(t, k, shared, "a request in the key's session should share its keeper whatever the default credentials become")
		assert.Equal(t, []string{first.Key}, rest.requestKeys("POST /openApi/user/auth/userDataStream"), "the key should be created for the credentials ctx named")

		contextCreds := &accounts.Credentials{Key: "context-key", Secret: "context-secret"}
		next := newTestSession(t, ex)
		renewed, err := ex.wsListenKey(accounts.DeployCredentialsToContext(t.Context(), contextCreds), exchange.RestSpot, next.ch)
		require.NoError(t, err, "wsListenKey must not error")
		assert.Equal(t, contextCreds, renewed.credentials, "a key created for context credentials should keep them")
		assert.Equal(t, "context-key", rest.requestKeys("POST /openApi/user/auth/userDataStream")[1], "a key should be created for the context's credentials")

		empty := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, empty, rest.URL)
		empty.wsState.retryDelay = time.Hour
		empty.SetCredentials(&accounts.Credentials{})
		t.Cleanup(empty.wsState.background.Wait)
		_, err = empty.wsListenKey(t.Context(), exchange.RestSpot, newTestSession(t, empty).ch)
		assert.ErrorIs(t, err, errWebsocketListenKeyUnavailable, "wsListenKey should report a key it could not get")
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "wsListenKey should return the credentials error")
		assert.Len(t, rest.requestKeys("POST /openApi/user/auth/userDataStream"), 2, "wsListenKey should request no key without credentials")
	})

	t.Run("concurrent requests share a keeper", func(t *testing.T) {
		t.Parallel()
		var created atomic.Int32
		bothRequested := make(chan struct{})
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
				return
			}
			n := created.Add(1)
			if n == 2 {
				close(bothRequested)
			}
			// Neither key is created before both requests arrive, so neither request finds the other's keeper first
			select {
			case <-bothRequested:
			case <-r.Context().Done():
				return
			case <-time.After(5 * time.Second):
				assert.Fail(t, "both listen key requests should arrive")
			}
			_, _ = fmt.Fprintf(w, `{"listenKey":"%s-%d"}`, testListenKey, n)
		}))
		server.Start()
		sharing := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		require.NoError(t, sharing.DisableRateLimiter(), "DisableRateLimiter must not error")
		testRESTEndpoints(t, sharing, server.URL)
		sharingSession := newTestSession(t, sharing)
		keepers := make([]*websocketListenKeyKeeper, 2)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i := range keepers {
			wg.Go(func() {
				keepers[i], errs[i] = sharing.wsListenKey(t.Context(), exchange.RestSpot, sharingSession.ch)
			})
		}
		wg.Wait()
		require.NoError(t, errs[0], "wsListenKey must not error")
		require.NoError(t, errs[1], "wsListenKey must not error")
		assert.Equal(t, int32(2), created.Load(), "each request should create a key")
		assert.Same(t, keepers[0], keepers[1], "requests in one session should share the keeper the first of them created")
		sharing.wsState.mu.Lock()
		assert.Len(t, sharing.wsState.keepers, 1, "one keeper should be kept for the REST host")
		sharing.wsState.mu.Unlock()
	})

	t.Run("credentials rotated before a scheduled reconnect", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		keys := &listenKeyRotationServer{}
		rotation := httptest.NewTestServer(t, keys)
		rotation.Start()
		rotated := newWebsocketIntegrationExchange(t, servers, asset.Spot, true, func(cfg *config.Exchange) {
			cfg.ConnectionMonitorDelay = 50 * time.Millisecond
			cfg.Features.Subscriptions = subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
			}
		})
		testRESTEndpoints(t, rotated, rotation.URL)
		// The monitor reconnects the public streams within the backoff, which turns its own listen key request away, so
		// only the reconnect the failure scheduled can connect the account stream
		rotated.wsState.retryDelay = 2 * time.Second
		assert.ErrorIs(t, rotated.Websocket.Connect(t.Context()), errWebsocketListenKeyUnavailable, "Connect should fail while the listen key is unavailable")
		rotated.SetCredentials(&accounts.Credentials{Key: "rotated-key", Secret: "rotated-secret"})
		keys.rotate()
		require.Eventually(t, rotated.Websocket.IsConnected, 5*time.Second, time.Millisecond, "the public streams must connect")
		require.Eventually(t, func() bool { return len(rotated.Websocket.GetSubscriptions()) == 2 }, 10*time.Second, 10*time.Millisecond, "the account stream must connect once the wait ends")
		testWaitBackground(t, rotated)
		created := keys.createdWith()
		require.NotEmpty(t, created, "a listen key must be requested")
		assert.Equal(t, "rotated-key", created[len(created)-1], "the scheduled reconnect should request the listen key with the credentials current when it runs")
		assert.True(t, rotated.Websocket.CanUseAuthenticatedEndpoints(), "authenticated websocket use should stay on while the current credentials are valid")
	})
}

// listenKeyRotationServer answers listen key creations with a busy error until rotated, then refuses the original test
// API key and accepts any other, recording the API key of each creation
type listenKeyRotationServer struct {
	mu      sync.Mutex
	rotated bool
	keys    []string
}

func (s *listenKeyRotationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/openApi/user/auth/userDataStream" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Method != http.MethodPost {
		_, _ = io.WriteString(w, `{}`)
		return
	}
	apiKey := r.Header.Get(keyHeader)
	s.keys = append(s.keys, apiKey)
	switch {
	case !s.rotated:
		_, _ = io.WriteString(w, `{"code":100500,"msg":"internal error"}`)
	case apiKey == "test-api-key":
		_, _ = io.WriteString(w, `{"code":100413,"msg":"Incorrect apiKey"}`)
	default:
		_, _ = io.WriteString(w, `{"listenKey":"rotated-listen-key"}`)
	}
}

func (s *listenKeyRotationServer) rotate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rotated = true
}

func (s *listenKeyRotationServer) createdWith() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.keys)
}

func TestWebsocketCredentialsRefused(t *testing.T) {
	t.Parallel()
	for _, code := range []int64{100001, 100412, 100413, 100419} {
		t.Run(strconv.FormatInt(code, 10), func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, wsCredentialsRefused(fmt.Errorf("wrapped: %w", &APIError{Code: code})), "code %d should be refused credentials", code)
		})
	}
	assert.False(t, wsCredentialsRefused(&APIError{Code: 100410}), "a rate limit error should not be refused credentials")
	assert.False(t, wsCredentialsRefused(request.ErrAuthRequestFailed), "the requester's authenticated request error should not be refused credentials")
}

func TestWebsocketRecordListenKeyFailure(t *testing.T) { //nolint:tparallel // each failure doubles the wait the failure before it left
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.wsState.retryDelay = time.Minute
	waits := make([]time.Duration, 0, 5)
	for i := range 5 {
		t.Run("failure "+strconv.Itoa(i+1), func(t *testing.T) {
			next := ex.wsRecordListenKeyFailure("host")
			ex.wsState.mu.Lock()
			r := ex.wsState.keyRetries["host"]
			ex.wsState.mu.Unlock()
			waits = append(waits, r.wait)
			assert.WithinDuration(t, time.Now().Add(r.wait), r.next, time.Second, "the next attempt should be the wait away")
			assert.Equal(t, r.next, next, "wsRecordListenKeyFailure should return when the wait ends")
		})
	}
	assert.Equal(t, []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, websocketListenKeyRetryMaxDelay, websocketListenKeyRetryMaxDelay}, waits, "each failure should double the wait up to the cap")

	ex.wsState.retryDelay = 0
	ex.wsRecordListenKeyFailure("other")
	ex.wsState.mu.Lock()
	assert.Equal(t, websocketListenKeyRetryDelay, ex.wsState.keyRetries["other"].wait, "the first wait should default to the retry delay")
	ex.wsState.mu.Unlock()
}

func TestWebsocketListenKeyWaiting(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.False(t, ex.wsListenKeyWaiting(exchange.RestSpot), "requests should not wait without a failure")
	host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	ex.wsRecordListenKeyFailure(host)
	assert.True(t, ex.wsListenKeyWaiting(exchange.RestSpot), "requests should wait after a failure")
	assert.True(t, ex.wsListenKeyWaiting(exchange.RestUSDTMargined), "requests to the same REST host should wait")
	assert.False(t, ex.wsListenKeyWaiting(exchange.RestFutures), "an unset endpoint should not wait")
	ex.wsState.mu.Lock()
	ex.wsState.keyRetries[host] = websocketListenKeyRetry{wait: time.Second, next: time.Now().Add(-time.Millisecond)}
	ex.wsState.mu.Unlock()
	assert.False(t, ex.wsListenKeyWaiting(exchange.RestSpot), "requests should not wait once the wait has ended")
}

func TestWebsocketListenKeyWaitEnd(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Zero(t, ex.wsListenKeyWaitEnd(), "nothing should wait without a failure")
	now := time.Now()
	ex.wsState.mu.Lock()
	ex.wsState.keyRetries = map[string]websocketListenKeyRetry{
		"later":   {next: now.Add(2 * time.Hour)},
		"earlier": {next: now.Add(time.Hour)},
		"ended":   {next: now.Add(-time.Minute)},
	}
	ex.wsState.mu.Unlock()
	assert.Equal(t, now.Add(time.Hour), ex.wsListenKeyWaitEnd(), "the earliest wait still running should end first")
}

func TestWebsocketScheduleAccountReconnect(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ctx, cancel := context.WithCancel(t.Context())
	now := time.Now()
	ex.wsScheduleAccountReconnect(ctx, now.Add(time.Hour))
	ex.wsScheduleAccountReconnect(ctx, now.Add(2*time.Hour))
	ex.wsState.mu.Lock()
	assert.Equal(t, now.Add(time.Hour), ex.wsState.reconnectAt, "a later flush should be absorbed by one due earlier")
	assert.Equal(t, uint64(1), ex.wsState.reconnects, "an absorbed flush should not be scheduled")
	ex.wsState.mu.Unlock()
	ex.wsScheduleAccountReconnect(ctx, now.Add(time.Minute))
	ex.wsState.mu.Lock()
	assert.Equal(t, now.Add(time.Minute), ex.wsState.reconnectAt, "an earlier flush should replace one due later")
	assert.Equal(t, uint64(2), ex.wsState.reconnects, "the earlier flush should be scheduled")
	ex.wsState.mu.Unlock()
	cancel()
	testWaitBackground(t, ex)
	ex.wsState.mu.Lock()
	assert.Zero(t, ex.wsState.reconnectAt, "a flush ended with its context should be forgotten")
	ex.wsState.mu.Unlock()

	t.Run("rescheduled while listen key requests still wait", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		ctx, cancel := context.WithCancel(t.Context())
		waitEnd := time.Now().Add(time.Hour)
		ex.wsState.mu.Lock()
		ex.wsState.keyRetries = map[string]websocketListenKeyRetry{"host": {wait: time.Hour, next: waitEnd}}
		ex.wsState.mu.Unlock()
		ex.wsScheduleAccountReconnect(ctx, time.Now())
		require.Eventually(t, func() bool {
			ex.wsState.mu.Lock()
			defer ex.wsState.mu.Unlock()
			return ex.wsState.reconnects == 2
		}, 5*time.Second, time.Millisecond, "a flush must schedule the next while listen key requests still wait")
		ex.wsState.mu.Lock()
		assert.Equal(t, waitEnd, ex.wsState.reconnectAt, "the next flush should be due when the wait still running ends")
		ex.wsState.mu.Unlock()
		cancel()
		testWaitBackground(t, ex)
	})

	t.Run("public streams connect while listen key requests wait", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		servers.rest.setReplies(`{"code":100500,"msg":"internal error"}`, "")
		ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, true, func(cfg *config.Exchange) {
			cfg.ConnectionMonitorDelay = 50 * time.Millisecond
			cfg.Features.Subscriptions = subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
			}
		})
		ex.wsState.retryDelay = 500 * time.Millisecond
		assert.ErrorIs(t, ex.Websocket.Connect(t.Context()), errWebsocketListenKeyUnavailable, "the first connection attempt should fail for want of a listen key")
		servers.rest.setReplies("", "")
		require.Eventually(t, ex.Websocket.IsConnected, 5*time.Second, time.Millisecond, "the public streams must connect while listen key requests wait")
		assert.Equal(t, []string{"BTC-USDT@ticker"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "the account stream should be left out while listen key requests wait")
		testWaitBackground(t, ex)
		assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "the account stream should connect once the wait ends")
		assert.Equal(t, []string{testListenKey + "-2"}, servers.ws[asset.Spot].accountKeys(), "the account connection should be dialled with the key taken after the wait")
		assert.Equal(t, 2, servers.ws[asset.Spot].marketDials(), "the market connection should not be dialled again when the account stream returns")
	})
}

func TestWebsocketAddKeeperConnection(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	closed, kept, added := &testWebsocketConnection{}, &testWebsocketConnection{}, &testWebsocketConnection{}
	closed.down.Store(true)
	k := &websocketListenKeyKeeper{conns: []websocket.Connection{closed, kept}}
	ex.wsAddKeeperConnection(k, added)
	assert.Equal(t, []websocket.Connection{kept, added}, k.conns, "the keeper should forget closed connections and add the new one")
}

func TestWebsocketKeeperInUse(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	k := &websocketListenKeyKeeper{}
	assert.False(t, ex.wsKeeperInUse(k), "a keeper without connections should not be in use")
	k.conns = []websocket.Connection{conn}
	assert.True(t, ex.wsKeeperInUse(k), "a keeper with a connected connection should be in use")
	conn.down.Store(true)
	assert.False(t, ex.wsKeeperInUse(k), "a keeper whose connections are closed should not be in use")
}

// startTestKeeper starts a keeper of a key for a test connection in a test session, extending every millisecond a key
// valid for 2 seconds, and trying a failed extension again after 50 milliseconds, then after twice the last wait
func startTestKeeper(tb testing.TB, ex *Exchange, conn websocket.Connection) (*websocketListenKeyKeeper, *testSession) {
	tb.Helper()
	session := newTestSession(tb, ex)
	host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(tb, err, "GetURL must not error")
	k := &websocketListenKeyKeeper{
		host:       host,
		ePath:      exchange.RestSpot,
		key:        testListenKey,
		interval:   time.Millisecond,
		validity:   2 * time.Second,
		retryDelay: 50 * time.Millisecond,
		session:    session.ch,
		conns:      []websocket.Connection{conn},
		done:       make(chan struct{}),
	}
	ex.wsState.mu.Lock()
	ex.wsState.keepers = map[string]*websocketListenKeyKeeper{host: k}
	ex.wsState.mu.Unlock()
	ex.Websocket.Wg.Go(func() { ex.wsKeepListenKeyAlive(tb.Context(), k) })
	return k, session
}

// waitForKeeper waits for a keeper to stop
func waitForKeeper(tb testing.TB, k *websocketListenKeyKeeper) {
	tb.Helper()
	select {
	case <-k.done:
	case <-time.After(5 * time.Second):
		require.FailNow(tb, "the keeper must stop")
	}
}

func TestWebsocketKeepListenKeyAlive(t *testing.T) {
	t.Parallel()
	t.Run("renewal and shutdown", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		k, session := startTestKeeper(t, ex, &testWebsocketConnection{})
		require.Eventually(t, func() bool {
			_, extended := rest.counts()
			return extended >= 3
		}, 5*time.Second, time.Millisecond, "the keeper must extend the key repeatedly")
		session.end()
		waitForKeeper(t, k)
		ex.wsState.mu.Lock()
		assert.Empty(t, ex.wsState.keepers, "a stopped keeper should be forgotten")
		assert.Nil(t, ex.wsState.recovering, "a keeper stopped with its session should not reconnect")
		ex.wsState.mu.Unlock()
		_, before := rest.counts()
		time.Sleep(20 * time.Millisecond)
		_, after := rest.counts()
		assert.Equal(t, before, after, "a stopped keeper should extend nothing more")
	})

	t.Run("connections closed", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		conn := &testWebsocketConnection{}
		conn.down.Store(true)
		k, _ := startTestKeeper(t, ex, conn)
		waitForKeeper(t, k)
		_, extended := rest.counts()
		assert.Zero(t, extended, "a keeper without a connection using its key should extend nothing")
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		rest.setReplies("", `{"code":100400,"msg":"listen key `+testListenKey+` does not exist"}`)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		start := time.Now()
		k, session := startTestKeeper(t, ex, &testWebsocketConnection{})
		waitForKeeper(t, k)
		ex.wsState.mu.Lock()
		assert.Equal(t, session.token(), ex.wsState.recovering, "a failed extension should reconnect the session once no attempt fits before the key expires")
		ex.wsState.mu.Unlock()
		_, extended := rest.counts()
		assert.Equal(t, 6, extended, "a failed extension should be tried again after 50, 100, 200, 400 and 800 milliseconds, each attempt coming at least 50 milliseconds before the key expires")
		assert.Less(t, time.Since(start), k.validity, "the websocket should reconnect before the key expires")
		testWaitBackground(t, ex)
	})

	t.Run("failure then success", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		rest.refuseExtensions(2)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		_, session := startTestKeeper(t, ex, &testWebsocketConnection{})
		require.Eventually(t, func() bool {
			_, extended := rest.counts()
			return extended >= 6
		}, 5*time.Second, time.Millisecond, "the keeper must keep extending the key after a retry succeeds")
		ex.wsState.mu.Lock()
		assert.Nil(t, ex.wsState.recovering, "an extension that succeeds on a retry should not reconnect")
		ex.wsState.mu.Unlock()
		session.end()
	})

	t.Run("failure reconnects with a new key", func(t *testing.T) {
		t.Parallel()
		servers := newTestWebsocketServers(t, asset.Spot)
		ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, true, func(cfg *config.Exchange) {
			cfg.ConnectionMonitorDelay = 50 * time.Millisecond
			cfg.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true}}
		})
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		require.Equal(t, []string{testListenKey + "-1"}, servers.ws[asset.Spot].accountKeys(), "the account connection must be dialled with the first key")
		servers.rest.setReplies("", `{"code":100400,"msg":"listen key does not exist"}`)
		// Long enough for the account connection to be dialled and recorded before the keeper first extends its key, and
		// a validity short enough for the failed extension's retries to give up soon after
		ex.wsState.mu.Lock()
		ex.wsState.keepaliveInterval = 200 * time.Millisecond
		ex.wsState.keyValidity = 400 * time.Millisecond
		ex.wsState.extensionRetryDelay = 20 * time.Millisecond
		ex.wsState.mu.Unlock()
		// The keeper took the interval when it started, so the session is ended to start a keeper that takes the new one
		require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
		require.Eventually(t, func() bool {
			return len(servers.ws[asset.Spot].accountKeys()) >= 3
		}, 10*time.Second, 5*time.Millisecond, "a failed extension must reconnect the account connection with a new key")
		keys := servers.ws[asset.Spot].accountKeys()
		assert.Equal(t, testListenKey+"-2", keys[1], "the reconnection should take a new key")
		assert.Equal(t, testListenKey+"-3", keys[2], "a failed extension should reconnect with a new key")
	})

	t.Run("extension ended with its session", func(t *testing.T) {
		t.Parallel()
		var extensions atomic.Int32
		extending := make(chan struct{})
		server := httptest.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if extensions.Add(1) == 1 {
				close(extending)
			}
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		}))
		server.Start()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, server.URL)
		session := newTestSession(t, ex)
		host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		k := &websocketListenKeyKeeper{
			host:     host,
			ePath:    exchange.RestSpot,
			key:      testListenKey,
			interval: time.Millisecond,
			// No attempt fits before the key expires, so an extension failing for any other reason would reconnect
			validity:   time.Minute,
			retryDelay: time.Minute,
			session:    session.ch,
			conns:      []websocket.Connection{&testWebsocketConnection{}},
			done:       make(chan struct{}),
		}
		ex.wsState.mu.Lock()
		ex.wsState.keepers = map[string]*websocketListenKeyKeeper{host: k}
		ex.wsState.mu.Unlock()
		go func() {
			select {
			case <-extending:
				session.end()
			case <-t.Context().Done():
			}
		}()
		ex.wsKeepListenKeyAlive(t.Context(), k)
		waitForKeeper(t, k)
		assert.Equal(t, int32(1), extensions.Load(), "the keeper should have tried one extension")
		ex.wsState.mu.Lock()
		assert.Nil(t, ex.wsState.recovering, "an extension ended with its session should not reconnect")
		assert.Empty(t, ex.wsState.keepers, "a stopped keeper should be forgotten")
		ex.wsState.mu.Unlock()
	})

	t.Run("credentials the key was created for", func(t *testing.T) {
		t.Parallel()
		rest := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, rest.URL)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		first, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		ex.SetCredentials(&accounts.Credentials{Key: "second-key", Secret: "second-secret"})
		session := newTestSession(t, ex)
		host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		k := &websocketListenKeyKeeper{
			host:        host,
			ePath:       exchange.RestSpot,
			key:         testListenKey,
			interval:    time.Millisecond,
			validity:    time.Minute,
			retryDelay:  time.Minute,
			session:     session.ch,
			credentials: first,
			conns:       []websocket.Connection{&testWebsocketConnection{}},
			done:        make(chan struct{}),
		}
		ex.wsState.mu.Lock()
		ex.wsState.keepers = map[string]*websocketListenKeyKeeper{host: k}
		ex.wsState.mu.Unlock()
		ex.Websocket.Wg.Go(func() { ex.wsKeepListenKeyAlive(t.Context(), k) })
		require.Eventually(t, func() bool {
			return len(rest.requestKeys("PUT /openApi/user/auth/userDataStream")) >= 2
		}, 5*time.Second, time.Millisecond, "the keeper must extend the key")
		session.end()
		waitForKeeper(t, k)
		for _, apiKey := range rest.requestKeys("PUT /openApi/user/auth/userDataStream") {
			assert.Equal(t, first.Key, apiKey, "the keeper should extend the key with the credentials it was created for, whatever the default credentials become")
		}
	})
}

func TestWebsocketExtensionRetryWait(t *testing.T) {
	t.Parallel()
	extended := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	now := extended.Add(websocketListenKeyKeepaliveInterval)
	var waits []time.Duration
	var lastWait time.Duration
	for {
		wait, retry := wsExtensionRetryWait(extended, now, lastWait, websocketListenKeyExtensionRetryDelay, websocketListenKeyValidity)
		if !retry {
			assert.Equal(t, 16*time.Minute, wait, "the wait that no longer fits should be twice the last")
			break
		}
		waits = append(waits, wait)
		now, lastWait = now.Add(wait), wait
	}
	assert.Equal(t, []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute}, waits, "a failure 30 minutes after an extension should be tried again after 1, 2, 4 and 8 minutes")
	assert.Equal(t, extended.Add(45*time.Minute), now, "the last attempt should come well before the key expires")

	_, retry := wsExtensionRetryWait(extended, extended.Add(58*time.Minute), 0, time.Minute, time.Hour)
	assert.False(t, retry, "an attempt less than the first wait before the key expires should not be made")
	wait, retry := wsExtensionRetryWait(extended, extended.Add(57*time.Minute), 0, time.Minute, time.Hour)
	assert.True(t, retry, "an attempt the first wait before the key expires should be made")
	assert.Equal(t, time.Minute, wait, "the first wait should follow the first failure")
}

func TestWebsocketStopKeeper(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	current := &websocketListenKeyKeeper{host: "host", done: make(chan struct{})}
	replaced := &websocketListenKeyKeeper{host: "host", done: make(chan struct{})}
	ex.wsState.keepers = map[string]*websocketListenKeyKeeper{"host": current}
	ex.wsStopKeeper(replaced)
	assert.Same(t, current, ex.wsState.keepers["host"], "stopping a replaced keeper should leave its successor")
	assert.NotPanics(t, func() { <-replaced.done }, "a stopped keeper should be done")
	ex.wsStopKeeper(current)
	assert.Empty(t, ex.wsState.keepers, "a stopped keeper should be forgotten")
	assert.NotPanics(t, func() { <-current.done }, "a stopped keeper should be done")
}

func TestWebsocketHandleControlFrame(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, tc := range []struct {
		name, frame, exp string
	}{
		{"USDT-M heartbeat", "Ping", "Pong"},
		{"spot heartbeat", `{"ping":"dfa87f932a004ec5a03e46e7a31171c0","time":"2026-11-11T17:53:38.549+0800"}`, `{"pong":"dfa87f932a004ec5a03e46e7a31171c0","time":"2026-11-11T17:53:38.549+0800"}`},
		{"coin-M heartbeat", `{"ping":"0c6bbddcbc384117842b04e335c8e092","time":"2026-11-11T17:52:45.233+0800"}`, `{"pong":"0c6bbddcbc384117842b04e335c8e092","time":"2026-11-11T17:52:45.233+0800"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conn := &testWebsocketConnection{}
			handled, err := ex.wsHandleControlFrame(t.Context(), conn, []byte(tc.frame))
			require.NoError(t, err, "wsHandleControlFrame must not error")
			assert.True(t, handled, "a heartbeat should be handled")
			assert.Equal(t, []testWebsocketMessage{{limit: websocketHeartbeatRate, message: tc.exp}}, conn.messages(), "the heartbeat should be answered on the heartbeat budget")
		})
	}

	t.Run("malformed heartbeat", func(t *testing.T) {
		t.Parallel()
		conn := &testWebsocketConnection{}
		handled, err := ex.wsHandleControlFrame(t.Context(), conn, []byte(`{"ping":"a","time":1}`))
		assert.True(t, handled, "a heartbeat should be handled")
		assert.Error(t, err, "a heartbeat that does not decode should error")
		assert.Empty(t, conn.messages(), "nothing should be answered")
	})

	t.Run("reply", func(t *testing.T) {
		t.Parallel()
		conn := &testWebsocketConnection{matchErr: websocket.ErrSignatureNotMatched}
		handled, err := ex.wsHandleControlFrame(t.Context(), conn, []byte(`{"code":0,"id":"probe-0","msg":"SUCCESS","timestamp":1794390762675}`))
		assert.True(t, handled, "a reply should be handled")
		assert.ErrorIs(t, err, websocket.ErrSignatureNotMatched, "a reply without a request should be reported")
		assert.Equal(t, []any{"probe-0"}, conn.matches(), "a reply should be routed to its request")
	})

	t.Run("data", func(t *testing.T) {
		t.Parallel()
		conn := &testWebsocketConnection{}
		handled, err := ex.wsHandleControlFrame(t.Context(), conn, fixtureLine(t, "testdata/wsSpot.jsonl", "@ticker"))
		assert.NoError(t, err, "wsHandleControlFrame should not error for data")
		assert.False(t, handled, "data should be left to the stream handlers")
		assert.Empty(t, conn.messages(), "nothing should be sent")
	})
}

func TestWebsocketMarketFrame(t *testing.T) {
	t.Parallel()
	symbol, stream, data, err := wsMarketFrame([]byte(`{"code":0,"dataType":"BTC-USDT@depth20@200ms","ts":1,"data":{"bids":[]}}`))
	require.NoError(t, err, "wsMarketFrame must not error")
	assert.Equal(t, "BTC-USDT", symbol, "the symbol should be split from the dataType")
	assert.Equal(t, "depth20@200ms", stream, "the stream should be the rest of the dataType")
	assert.JSONEq(t, `{"bids":[]}`, string(data), "the data should be returned")

	symbol, _, _, err = wsMarketFrame([]byte(`{"code":100400,"msg":"bad request"}`))
	assert.ErrorIs(t, err, errWebsocketErrorMessage, "an error frame should be reported")
	assert.Empty(t, symbol, "an error frame should name no symbol")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "the report must carry BingX's error")
	assert.Equal(t, &APIError{Code: 100400, Message: "bad request"}, apiErr, "the report should carry the code and message")
	for _, tc := range []struct {
		name, frame string
	}{
		{"missing dataType", `{"code":0}`},
		{"dataType without a stream", `{"code":0,"dataType":"BTC-USDT"}`},
		{"missing data", `{"code":0,"dataType":"BTC-USDT@ticker"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := wsMarketFrame([]byte(tc.frame))
			assert.ErrorIsf(t, err, errWebsocketUnhandledMessage, "wsMarketFrame should reject %s", tc.frame)
		})
	}
}

func TestWebsocketAccountEvent(t *testing.T) {
	t.Parallel()
	event, data, err := wsAccountEvent(fixtureLine(t, "testdata/wsSpotAccount.jsonl", "executionReport"))
	require.NoError(t, err, "wsAccountEvent must not error")
	assert.Equal(t, "executionReport", event, "the event should be read from the data member")
	assert.True(t, bytes.HasPrefix(data, []byte(`{"u":`)), "the data member should be returned")

	frame := fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_CONFIG_UPDATE")
	event, data, err = wsAccountEvent(frame)
	require.NoError(t, err, "wsAccountEvent must not error")
	assert.Equal(t, "ACCOUNT_CONFIG_UPDATE", event, "the event of a bare frame should be read")
	assert.Equal(t, frame, data, "a bare frame should be returned whole")

	_, _, err = wsAccountEvent([]byte(`{"code":100400,"msg":"bad request"}`))
	assert.ErrorIs(t, err, errWebsocketErrorMessage, "an error frame should be reported")
	_, _, err = wsAccountEvent([]byte(`{"listenKey":"secret-key","data":null}`))
	require.ErrorIs(t, err, errWebsocketUnhandledMessage, "a frame without an event must be reported")
	assert.NotContains(t, err.Error(), "secret-key", "the report should not quote the frame")
}

func TestWebsocketRelay(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, wsRelay[WebsocketMarkPrice](t.Context(), ex, []byte(`{"e":"markPriceUpdate","E":1794390763430,"s":"BTC-USDT","p":"73664.1"}`)), "wsRelay must not error")
	got := waitForWebsocketPayload[*WebsocketMarkPrice](t, ex, nil)
	assert.Equal(t, &WebsocketMarkPrice{EventType: "markPriceUpdate", EventTime: types.Time(time.UnixMilli(1794390763430)), Symbol: "BTC-USDT", Price: 73664.1}, got, "wsRelay should relay the decoded payload")
	assert.Error(t, wsRelay[WebsocketMarkPrice](t.Context(), ex, []byte(`{"p":true}`)), "wsRelay should error for a payload that does not decode")
}

func TestWebsocketProcessTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	p := &ticker.Price{Last: 1, Pair: testSpotPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.Now()}
	require.NoError(t, ex.wsProcessTicker(t.Context(), p), "wsProcessTicker must not error")
	stored, err := ticker.GetTicker(ex.Name, testSpotPair, asset.Spot)
	require.NoError(t, err, "the ticker must be stored")
	assert.InDelta(t, 1, stored.Last, 1e-9, "the stored ticker should be the one processed")
	assert.Same(t, p, waitForWebsocketPayload[*ticker.Price](t, ex, nil), "the processed ticker should be relayed")

	assert.Error(t, ex.wsProcessTicker(t.Context(), &ticker.Price{ExchangeName: ex.Name, AssetType: asset.Spot}), "wsProcessTicker should error for a ticker that fails processing")
	assert.Empty(t, websocketPayloads(ex), "a ticker that fails processing should not be relayed")
}

func TestWebsocketTakerSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Sell, wsTakerSide(true), "a buyer that was the maker should make the trade a sell")
	assert.Equal(t, order.Buy, wsTakerSide(false), "a buyer that was the taker should make the trade a buy")
}

func TestWebsocketSplitSymbol(t *testing.T) {
	t.Parallel()
	for symbol, exp := range map[string][2]string{"BTC-USDT": {"BTC", "USDT"}, "X-OLD-USDT": {"X-OLD", "USDT"}, "BTC-USD": {"BTC", "USD"}} {
		t.Run(symbol, func(t *testing.T) {
			t.Parallel()
			base, quote, err := wsSplitSymbol(symbol)
			require.NoErrorf(t, err, "wsSplitSymbol must not error for %s", symbol)
			assert.Equalf(t, exp, [2]string{base, quote}, "wsSplitSymbol should split %s at its last dash", symbol)
		})
	}
	for _, symbol := range []string{"BTCUSDT", "-USDT", "BTC-", ""} {
		t.Run(cmp.Or(symbol, "empty"), func(t *testing.T) {
			t.Parallel()
			_, _, err := wsSplitSymbol(symbol)
			assert.ErrorIsf(t, err, errWebsocketInvalidSymbol, "wsSplitSymbol should reject %q", symbol)
		})
	}
}

func TestWebsocketMatchPair(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	pair, enabled, err := ex.wsMatchPair("BTC-USDT", asset.Spot)
	require.NoError(t, err, "wsMatchPair must not error")
	assert.True(t, enabled, "an enabled pair should be reported enabled")
	assert.True(t, testSpotPair.Equal(pair), "wsMatchPair should return the pair")
	pair, enabled, err = ex.wsMatchPair("ETH-USDT", asset.Spot)
	require.NoError(t, err, "wsMatchPair must not error for an available pair")
	assert.False(t, enabled, "an available pair should be reported disabled")
	assert.Equal(t, "ETH-USDT", pair.String(), "wsMatchPair should return the available pair")
	_, _, err = ex.wsMatchPair("DOGE-USDT", asset.Spot)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsMatchPair should reject a pair that is not available")
	_, _, err = ex.wsMatchPair("BTCUSDT", asset.Spot)
	assert.ErrorIs(t, err, errWebsocketInvalidSymbol, "wsMatchPair should reject a symbol without a dash")
}

func TestWebsocketAccountPair(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	pair, err := ex.wsAccountPair("ETH-USD", asset.CoinMarginedFutures)
	require.NoError(t, err, "wsAccountPair must not error for an available pair")
	assert.Equal(t, "ETH-USD", pair.String(), "wsAccountPair should return the available pair")
	pair, err = ex.wsAccountPair("SUI-USD", asset.CoinMarginedFutures)
	require.NoError(t, err, "wsAccountPair must not error for a symbol listed since the pairs were updated")
	assert.Equal(t, currency.NewPairWithDelimiter("SUI", "USD", currency.DashDelimiter), pair, "wsAccountPair should return the pair the symbol names")
	_, err = ex.wsAccountPair("SUIUSD", asset.CoinMarginedFutures)
	assert.ErrorIs(t, err, errWebsocketInvalidSymbol, "wsAccountPair should reject a symbol without a dash")
	_, err = ex.wsAccountPair("SUI-USD", asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "wsAccountPair should reject an asset without a pair format")
}

func TestWebsocketProcessListenKeyExpired(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	_, session := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsProcessListenKeyExpired(t.Context(), conn, []byte(`{"e":"listenKeyExpired","E":1576653824250,"listenKey":"secret-key"}`)), "wsProcessListenKeyExpired must not error")
	got := waitForWebsocketPayload[*WebsocketListenKeyExpired](t, ex, nil)
	assert.Equal(t, &WebsocketListenKeyExpired{EventType: "listenKeyExpired", EventTime: types.Time(time.UnixMilli(1576653824250))}, got, "the expiry should be relayed without its key")
	ex.wsState.mu.Lock()
	assert.Equal(t, session.token(), ex.wsState.recovering, "an expired key should reconnect the connection's session")
	ex.wsState.mu.Unlock()
	testWaitBackground(t, ex)
	assert.Error(t, ex.wsProcessListenKeyExpired(t.Context(), conn, []byte(`{"E":"x"}`)), "wsProcessListenKeyExpired should error for an event that does not decode")
}

// testWebsocketServer is a mock of a product's BingX websocket server. Every connection is sent a heartbeat first. A
// market connection has each request answered in the product's reply shape, refused for a dataType in reject, or
// for any unsubscription while rejectUnsubscribe is set, unless it is an unsubscription left unanswered while
// ignoreUnsubscribe is set, and is pushed the product's fixture frames of each stream it subscribes to. An account
// connection must carry a listen key:
// spot account connections are pushed the documented event of each stream they subscribe to, and perpetual ones every
// documented event at once
type testWebsocketServer struct {
	*httptest.Server
	product           asset.Item
	reject            []string
	rejectUnsubscribe atomic.Bool
	ignoreUnsubscribe atomic.Bool
	marketFrames      [][]byte
	accountFrames     [][]byte
	mu                sync.Mutex
	marketCount       int
	accountClosed     int
	keysDialled       []string
	requests          []WebsocketSubscriptionRequest
	replies           []string
}

var testUpgrader = gws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// newTestWebsocketServer starts a product's mock websocket server for the test
func newTestWebsocketServer(tb testing.TB, product asset.Item, reject ...string) *testWebsocketServer {
	tb.Helper()
	fixtures := map[asset.Item][2]string{
		asset.Spot:                {"testdata/wsSpot.jsonl", "testdata/wsSpotAccount.jsonl"},
		asset.USDTMarginedFutures: {"testdata/wsSwap.jsonl", "testdata/wsSwapAccount.jsonl"},
		asset.CoinMarginedFutures: {"testdata/wsCoinMargined.jsonl", "testdata/wsCoinMarginedAccount.jsonl"},
	}[product]
	s := &testWebsocketServer{
		product:       product,
		reject:        reject,
		marketFrames:  fixtureLines(tb, fixtures[0]),
		accountFrames: fixtureLines(tb, fixtures[1]),
	}
	s.Server = httptest.NewTestServer(tb, s)
	s.Start()
	return s
}

// websocketURL returns the server's websocket URL
func (s *testWebsocketServer) websocketURL() string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func (s *testWebsocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	listenKey := r.URL.Query().Get("listenKey")
	c, err := testUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	s.mu.Lock()
	if listenKey == "" {
		s.marketCount++
	} else {
		s.keysDialled = append(s.keysDialled, listenKey)
	}
	s.mu.Unlock()
	heartbeat := []byte(`{"ping":"26f911f610e94c97a1b75ae4fbc966f7","time":"2026-11-11T17:53:19.200+0800"}`)
	if s.product == asset.USDTMarginedFutures {
		heartbeat = []byte(websocketSwapPing)
	}
	if c.WriteMessage(gws.BinaryMessage, testGzip(heartbeat)) != nil {
		return
	}
	if listenKey != "" {
		defer func() {
			s.mu.Lock()
			s.accountClosed++
			s.mu.Unlock()
		}()
	}
	if listenKey != "" && s.product != asset.Spot {
		for _, frame := range s.accountFrames {
			if c.WriteMessage(gws.BinaryMessage, testGzip(frame)) != nil {
				return
			}
		}
	}
	for {
		_, msg, err := c.ReadMessage()
		if err != nil || !s.handle(c, msg, listenKey != "") {
			return
		}
	}
}

// handle answers a message a connection sent, reporting whether the connection should be kept
func (s *testWebsocketServer) handle(c *gws.Conn, msg []byte, account bool) bool {
	if string(msg) == websocketSwapPong || bytes.HasPrefix(msg, []byte(`{"pong"`)) {
		s.mu.Lock()
		s.replies = append(s.replies, string(msg))
		s.mu.Unlock()
		return true
	}
	var req WebsocketSubscriptionRequest
	if json.Unmarshal(msg, &req) != nil {
		return false
	}
	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.mu.Unlock()
	if req.RequestType == WebsocketRequestTypeUnsubscribe && s.ignoreUnsubscribe.Load() {
		return true
	}
	if slices.Contains(s.reject, req.DataType) || req.RequestType == WebsocketRequestTypeUnsubscribe && s.rejectUnsubscribe.Load() {
		_, stream, _ := strings.Cut(req.DataType, "@")
		return c.WriteMessage(gws.BinaryMessage, testGzip(testWebsocketAck(s.product, req.ID, 100400, "code:100400:"+stream+" dataType is not supported"))) == nil
	}
	if c.WriteMessage(gws.BinaryMessage, testGzip(testWebsocketAck(s.product, req.ID, 0, "SUCCESS"))) != nil {
		return false
	}
	if req.RequestType != WebsocketRequestTypeSubscribe {
		return true
	}
	frames, match := s.marketFrames, []byte(`"dataType":"`+req.DataType+`"`)
	if account {
		frames, match = s.accountFrames, []byte(`"e":"`+strings.TrimPrefix(req.DataType, "spot.")+`"`)
	}
	for _, frame := range frames {
		if bytes.Contains(frame, match) && c.WriteMessage(gws.BinaryMessage, testGzip(frame)) != nil {
			return false
		}
	}
	return true
}

// marketDials returns how many market connections were dialled
func (s *testWebsocketServer) marketDials() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.marketCount
}

// accountKeys returns the listen key of each account connection dialled
func (s *testWebsocketServer) accountKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.keysDialled)
}

// accountCloses returns how many account connections have closed
func (s *testWebsocketServer) accountCloses() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accountClosed
}

// receivedRequests returns the subscription requests received
func (s *testWebsocketServer) receivedRequests() []WebsocketSubscriptionRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// heartbeatReplies returns the heartbeat replies received
func (s *testWebsocketServer) heartbeatReplies() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.replies)
}

// testWebsocketServers are the mock servers of an integration test: a REST host serving the listen keys and each
// product's websocket server
type testWebsocketServers struct {
	rest *testListenKeyServer
	ws   map[asset.Item]*testWebsocketServer
}

// newTestWebsocketServers starts the mock servers of an integration test, the product's websocket server refusing the
// dataTypes in reject
func newTestWebsocketServers(tb testing.TB, product asset.Item, reject ...string) *testWebsocketServers {
	tb.Helper()
	s := &testWebsocketServers{rest: newTestListenKeyServer(tb), ws: make(map[asset.Item]*testWebsocketServer)}
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		var refused []string
		if a == product {
			refused = reject
		}
		s.ws[a] = newTestWebsocketServer(tb, a, refused...)
	}
	return s
}

// newWebsocketIntegrationExchange returns an exchange set up through the real Setup with its websocket enabled, asset a
// alone enabled, authenticated websocket use when authenticated is set, and every REST and websocket endpoint served
// by servers. When the test ends the websocket is disabled and shut down
func newWebsocketIntegrationExchange(tb testing.TB, servers *testWebsocketServers, a asset.Item, authenticated bool, configure ...func(*config.Exchange)) *Exchange {
	tb.Helper()
	setup := []func(*config.Exchange){
		func(cfg *config.Exchange) {
			cfg.Features.Enabled.Websocket = true
			cfg.API.Endpoints = map[string]string{
				exchange.RestSpot.String():              servers.rest.URL,
				exchange.RestUSDTMargined.String():      servers.rest.URL,
				exchange.RestCoinMargined.String():      servers.rest.URL,
				exchange.WebsocketSpot.String():         servers.ws[asset.Spot].websocketURL(),
				exchange.WebsocketUSDTMargined.String(): servers.ws[asset.USDTMarginedFutures].websocketURL(),
				exchange.WebsocketCoinMargined.String(): servers.ws[asset.CoinMarginedFutures].websocketURL(),
			}
		},
		testWebsocketAssets(a),
	}
	if authenticated {
		setup = append(setup, testWebsocketAuthenticated)
	}
	ex := newWebsocketTestExchange(tb, append(setup, configure...)...)
	tb.Cleanup(func() { shutdownTestWebsocket(tb, ex) })
	return ex
}

// testLiveMarketStreams connects an asset's public market streams on BingX through the real Setup and waits for its
// enabled pair's ticker and order book
func testLiveMarketStreams(t *testing.T, a asset.Item, pair currency.Pair, levels int) {
	t.Helper()
	if mockTests {
		t.Skip("the live streams are tested with -tags mock_test_off")
	}
	ex := newWebsocketTestExchange(t, func(cfg *config.Exchange) { cfg.Features.Enabled.Websocket = true }, testWebsocketAssets(a), testWebsocketSubscriptions(
		&subscription.Subscription{Enabled: true, Asset: a, Channel: subscription.TickerChannel},
		&subscription.Subscription{Enabled: true, Asset: a, Channel: subscription.OrderbookChannel, Levels: levels},
	))
	t.Cleanup(func() { shutdownTestWebsocket(t, ex) })
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "both streams should be subscribed")
	require.Eventually(t, func() bool {
		_, tickerErr := ticker.GetTicker(ex.Name, pair, a)
		book, bookErr := ex.Websocket.Orderbook.GetOrderbook(pair, a)
		return tickerErr == nil && bookErr == nil && len(book.Bids) != 0 && len(book.Asks) != 0
	}, 30*time.Second, 50*time.Millisecond, "the live ticker and order book must arrive")
	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Greater(t, book.Asks[0].Price, book.Bids[0].Price, "the live book should not be crossed")
}

// shutdownTestWebsocket disables the websocket and shuts it down, waiting for a connection attempt or recovery in
// progress, as the connection monitor would otherwise connect again
func shutdownTestWebsocket(tb testing.TB, ex *Exchange) {
	tb.Helper()
	if ex.Websocket.IsEnabled() {
		assert.NoError(tb, ex.Websocket.Disable(), "Disable should not error")
	}
	assert.Eventually(tb, func() bool {
		err := ex.Websocket.Shutdown()
		return err == nil || errors.Is(err, websocket.ErrNotConnected)
	}, 10*time.Second, 10*time.Millisecond, "the websocket should shut down")
	testWaitBackground(tb, ex)
}

// testWaitBackground waits for the background work of ex's websocket state, the recoveries, scheduled flushes and
// balance refreshes, failing the test if it has not finished within ten seconds
func testWaitBackground(tb testing.TB, ex *Exchange) {
	tb.Helper()
	done := make(chan struct{})
	go func() {
		ex.wsState.background.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		require.FailNow(tb, "the websocket state's background work must finish")
	}
}

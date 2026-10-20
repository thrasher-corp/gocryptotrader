package kraken

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	// wsTestPair and wsTestDOGEPair are spot pairs in the test configuration's format
	wsTestPair     = currency.NewPairWithDelimiter("XBT", "USD", "_")
	wsTestDOGEPair = currency.NewPairWithDelimiter("XDG", "USD", "_")
)

// wsTestToken is the websocket token of the recorded GetWebSocketsToken response
const wsTestToken = "1Dwc4lzSwNWOAwkMdqhssNNFhs1ed606d1WcF3XfEMw"

// fixtureLines returns the lines of a websocket fixture file
func fixtureLines(tb testing.TB, path string) [][]byte {
	tb.Helper()
	f, err := os.Open(path)
	require.NoErrorf(tb, err, "Open must not error for %s", path)
	defer f.Close()
	var lines [][]byte
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		lines = append(lines, slices.Clone(s.Bytes()))
	}
	require.NoError(tb, s.Err(), "Scanner must not error")
	return lines
}

// useTestCredentials gives an exchange placeholder credentials, for tests that need credentials without sending them
func useTestCredentials(ex *Exchange) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: testSecret})
}

// newMockWebsocketURL starts an in-memory websocket server answering each message with mock, returning its URL. The
// keep-alive pings a spot connection sends every wsPingDelay are not passed to mock, since a test slowed by load can
// outlive the delay
func newMockWebsocketURL(tb testing.TB, mock mockws.WsMockFunc) string {
	tb.Helper()
	s := httptest.NewTestServer(tb, mockws.CurryWsMockUpgrader(tb, func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		if method, err := jsonparser.GetString(msg, "method"); err == nil && method == "ping" {
			return nil
		}
		return mock(tb, msg, w)
	}))
	s.Start()
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

// serveWebsocketToken answers REST requests with Get Websockets Token's reply. Subscribing to a channel that takes a
// token refreshes it over REST, which this serves so that live test runs request none from Kraken
func serveWebsocketToken(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/GetWebSocketsToken", r.URL.Path, "only a websocket token should be requested")
		_, err := w.Write([]byte(`{"error":[],"result":{"token":"` + wsTestToken + `","expires":900}}`))
		assert.NoError(t, err, "Write should not error")
	}
}

// useTestWebsocket replaces the exchange's websocket with one holding the connections the message filters name. A
// non-empty url replaces their URLs, such as with a mock server's. The connection monitor does not reconnect within the
// test
func useTestWebsocket(tb testing.TB, ex *Exchange, url string, filters ...string) {
	tb.Helper()
	ex.Config.ConnectionMonitorDelay = time.Hour
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(tb, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               ex.Config,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		TradeFeed:                    ex.Features.Enabled.TradeFeed,
		FillsFeed:                    ex.Features.Enabled.FillsFeed,
		UseMultiConnectionManagement: true,
	}), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(ex.API.AuthenticatedWebsocketSupport)
	setups, err := ex.connectionSetups(ex.Config)
	require.NoError(tb, err, "connectionSetups must not error")
	for _, s := range setups {
		if !slices.ContainsFunc(filters, func(f string) bool { return f == s.MessageFilter }) {
			continue
		}
		if url != "" {
			s.URL = url
		}
		require.NoError(tb, ex.Websocket.SetupNewConnection(s), "SetupNewConnection must not error")
	}
	tb.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise outlive the test
		if ex.Websocket.IsEnabled() {
			assert.NoError(tb, ex.Websocket.Disable(), "Disable should not error")
		}
		if ex.Websocket.IsConnected() {
			assert.NoError(tb, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
	})
}

// mockSubscriptionServer answers subscribe and unsubscribe requests as Kraken does, once for each symbol, and records
// each request without its req_id
type mockSubscriptionServer struct {
	m        sync.Mutex
	requests []string
	// rejected maps a symbol to the error its subscription is rejected with
	rejected map[string]string
	// unexpected answers with a symbol the request did not name
	unexpected bool
	// repeated answers each symbol of a request with the first
	repeated bool
	// replies, when set, answers each request with these replies as Kraken sent them, their req_id set to the request's
	replies []string
}

func (s *mockSubscriptionServer) handle(_ testing.TB, msg []byte, w *gws.Conn) error {
	var req struct {
		Method    string               `json:"method"`
		Params    wsSubscriptionParams `json:"params"`
		RequestID int64                `json:"req_id"`
	}
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	s.m.Lock()
	s.requests = append(s.requests, string(jsonparser.Delete(slices.Clone(msg), "req_id")))
	s.m.Unlock()
	for _, r := range s.replies {
		reply, err := jsonparser.Set([]byte(r), []byte(strconv.FormatInt(req.RequestID, 10)), "req_id")
		if err != nil {
			return err
		}
		if err := w.WriteMessage(gws.TextMessage, reply); err != nil {
			return err
		}
	}
	if len(s.replies) != 0 {
		return nil
	}
	symbols := req.Params.Symbols
	if len(symbols) == 0 {
		symbols = []string{""}
	}
	for _, symbol := range symbols {
		switch {
		case s.unexpected:
			symbol = "MOON/USD"
		case s.repeated:
			symbol = symbols[0]
		}
		reply := wsResponse{Method: req.Method, RequestID: req.RequestID, Success: true}
		if errText, ok := s.rejected[symbol]; ok {
			reply.Success, reply.Error, reply.Symbol = false, errText, symbol
		} else {
			result, err := json.Marshal(&wsSubscriptionResult{Channel: req.Params.Channel, Symbol: symbol})
			if err != nil {
				return err
			}
			reply.Result = result
		}
		b, err := json.Marshal(&reply)
		if err != nil {
			return err
		}
		if err := w.WriteMessage(gws.TextMessage, b); err != nil {
			return err
		}
	}
	return nil
}

// takeRequests returns and forgets the requests received
func (s *mockSubscriptionServer) takeRequests() []string {
	s.m.Lock()
	defer s.m.Unlock()
	r := s.requests
	s.requests = nil
	return r
}

// assertRequests asserts that the requests are the expected ones, each a JSON document, in any order
func assertRequests(tb testing.TB, exp, got []string) {
	tb.Helper()
	if !assert.Len(tb, got, len(exp), "requests should be the expected number") {
		return
	}
	remaining := slices.Clone(exp)
	for _, g := range got {
		i := slices.IndexFunc(remaining, func(e string) bool {
			return assert.ObjectsAreEqual(decodeJSON(tb, e), decodeJSON(tb, g))
		})
		if !assert.NotEqualf(tb, -1, i, "request %s should be expected", g) {
			continue
		}
		remaining = slices.Delete(remaining, i, i+1)
	}
}

func decodeJSON(tb testing.TB, s string) any {
	tb.Helper()
	var v any
	require.NoErrorf(tb, json.Unmarshal([]byte(s), &v), "Unmarshal must not error for %s", s)
	return v
}

// expectedSubscriptions returns the subscriptions the default subscriptions expand to for the test configuration
func expectedSubscriptions(ex *Exchange) subscription.List {
	pairs := currency.Pairs{wsTestPair}
	exp := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Pairs: pairs},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, QualifiedChannel: wsChannelTrade, Pairs: pairs},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Interval: kline.OneMin, Pairs: pairs},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Levels: 1000, Pairs: pairs},
	}
	if ex.Websocket.CanUseAuthenticatedEndpoints() && ex.AreCredentialsValid(context.Background()) {
		exp = append(exp, &subscription.Subscription{
			Enabled:          true,
			Channel:          subscription.MyOrdersChannel,
			Authenticated:    true,
			QualifiedChannel: wsChannelExecutions,
			Params:           map[string]any{"snap_orders": true, "snap_trades": true},
		})
	}
	return exp
}

func TestMockWebsocketSkipsKeepAlivePings(t *testing.T) {
	t.Parallel()
	ex := newWsTradingTestExchange(t,
		`{"method":"cancel_all","params":{"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"cancel_all","result":{"count":1,"warnings":["Untriggered orders cancelled"]},"success":true,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`,
	)
	conn, err := ex.Websocket.GetConnection(wsPrivateConnection)
	require.NoError(t, err, "GetConnection must not error")
	// A test slowed by load can outlive wsPingDelay, so its mock must not take the keep-alive ping for a request
	require.NoError(t, conn.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte(`{"method":"ping"}`)), "SendRawMessage must not error")
	_, err = ex.WebsocketCancelAllOrders(t.Context())
	assert.NoError(t, err, "WebsocketCancelAllOrders should not error after a keep-alive ping")
}

func TestConnectionSetups(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	setups, err := ex.connectionSetups(ex.Config)
	require.NoError(t, err, "connectionSetups must not error")
	require.Len(t, setups, 4, "connectionSetups must return the public, private, level 3 and futures connections")
	for i, exp := range []struct {
		filter, url string
	}{
		{wsPublicConnection, wsPublicURL},
		{wsPrivateConnection, wsPrivateURL},
		{wsLevel3Connection, wsLevel3URL},
		{wsFuturesConnection, wsFuturesURL},
	} {
		assert.Equalf(t, exp.filter, setups[i].MessageFilter, "connection %d should have the expected message filter", i)
		assert.Equalf(t, exp.url, setups[i].URL, "connection %d should use the configured websocket URL", i)
		assert.NotNilf(t, setups[i].Connector, "connection %d should have a connector", i)
		assert.NotNilf(t, setups[i].GenerateSubscriptions, "connection %d should generate subscriptions", i)
		assert.NotNilf(t, setups[i].Subscriber, "connection %d should have a subscriber", i)
		assert.NotNilf(t, setups[i].Unsubscriber, "connection %d should have an unsubscriber", i)
		assert.NotNilf(t, setups[i].Handler, "connection %d should have a handler", i)
		assert.NotNilf(t, setups[i].RateLimit, "connection %d should be rate limited", i)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	exp := expectedSubscriptions(ex)
	public, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	testsubs.EqualLists(t, exp.Public(), public)
	private, err := ex.generatePrivateSubscriptions()
	require.NoError(t, err, "generatePrivateSubscriptions must not error")
	testsubs.EqualLists(t, exp.Private(), private)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	private, err = ex.generatePrivateSubscriptions()
	require.NoError(t, err, "generatePrivateSubscriptions must not error without authenticated websocket use")
	assert.Empty(t, private, "generatePrivateSubscriptions should return nothing without authenticated websocket use")
}

func TestDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// Config version 18 migrates the previous default subscriptions to the defaults the config fixture holds
	testsubs.EqualLists(t, defaultSubscriptions.Clone(), ex.Config.Features.Subscriptions.Clone())
	useTestCredentials(ex)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	type feed struct {
		asset   asset.Item
		channel string
	}
	got := make(map[feed]bool)
	var all subscription.List
	for name, generate := range map[string]func() (subscription.List, error){
		"generatePublicSubscriptions":  ex.generatePublicSubscriptions,
		"generatePrivateSubscriptions": ex.generatePrivateSubscriptions,
		"generateLevel3Subscriptions":  ex.generateLevel3Subscriptions,
		"generateFuturesSubscriptions": ex.generateFuturesSubscriptions,
	} {
		subs, err := generate()
		require.NoErrorf(t, err, "%s must not error", name)
		for _, s := range subs {
			got[feed{s.Asset, s.QualifiedChannel}] = true
		}
		all = append(all, subs...)
	}
	assert.NoError(t, ex.ValidateSubscriptions(all), "the default subscriptions should be valid together")
	exp := map[feed]bool{
		{asset.Spot, wsChannelTicker}:            true,
		{asset.Spot, wsChannelTrade}:             true,
		{asset.Spot, wsChannelOHLC}:              true,
		{asset.Spot, wsChannelBook}:              true,
		{asset.Empty, wsChannelExecutions}:       true,
		{asset.Futures, wsFuturesFeedTicker}:     true,
		{asset.Futures, wsFuturesFeedTrade}:      true,
		{asset.Futures, wsFuturesFeedBook}:       true,
		{asset.Futures, wsFuturesFeedOpenOrders}: true,
		{asset.Futures, wsFuturesFeedFills}:      true,
	}
	assert.Equal(t, exp, got, "the default subscriptions should cover the spot and futures market and account feeds")
}

func TestGeneratePrivateSubscriptions(t *testing.T) {
	t.Parallel()
	myOrders := &subscription.Subscription{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}
	myTrades := &subscription.Subscription{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true}
	myWallet := &subscription.Subscription{Enabled: true, Channel: subscription.MyWalletChannel, Authenticated: true}
	executions := func(channel string, snapOrders, snapTrades bool) *subscription.Subscription {
		return &subscription.Subscription{
			Enabled:          true,
			Channel:          channel,
			Authenticated:    true,
			QualifiedChannel: wsChannelExecutions,
			Params:           map[string]any{"snap_orders": snapOrders, "snap_trades": snapTrades},
		}
	}
	balances := &subscription.Subscription{Enabled: true, Channel: subscription.MyWalletChannel, Authenticated: true, QualifiedChannel: wsChannelBalances}
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  subscription.List
	}{
		{"my orders", subscription.List{myOrders}, subscription.List{executions(subscription.MyOrdersChannel, true, false)}},
		{"my trades", subscription.List{myTrades}, subscription.List{executions(subscription.MyTradesChannel, false, true)}},
		{"my trades, my orders and my wallet", subscription.List{myTrades, myOrders, myWallet}, subscription.List{executions(subscription.MyOrdersChannel, true, true), balances}},
		{"public only", subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}, subscription.List{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			useTestCredentials(ex)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = tc.subs.Clone()
			got, err := ex.generatePrivateSubscriptions()
			require.NoError(t, err, "generatePrivateSubscriptions must not error")
			testsubs.EqualLists(t, tc.exp, got)
			for _, s := range ex.Features.Subscriptions {
				assert.Nilf(t, s.Params, "generatePrivateSubscriptions should not change the configured %s subscription", s.Channel)
			}
		})
	}
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	book := func(levels int, pairs ...currency.Pair) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: levels, Pairs: pairs}
	}
	ohlc := func(interval kline.Interval) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Asset: asset.Spot, Interval: interval, Pairs: currency.Pairs{wsTestPair}}
	}
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	assert.NoError(t, e.ValidateSubscriptions(subscription.List{tickerSub, book(1000, wsTestPair), book(1000, wsTestPair), book(10, wsTestDOGEPair), ohlc(kline.OneMin), ohlc(kline.FifteenDay)}), "ValidateSubscriptions should accept the depths and intervals Kraken streams")
	for _, depth := range []int{0, 20, 2000} {
		assert.ErrorIsf(t, e.ValidateSubscriptions(subscription.List{book(depth, wsTestPair)}), errInvalidDepth, "ValidateSubscriptions should reject a depth of %d", depth)
	}
	assert.ErrorIs(t, e.ValidateSubscriptions(subscription.List{book(10, wsTestPair), book(100, wsTestPair)}), subscription.ErrExclusiveSubscription, "ValidateSubscriptions should reject a pair subscribed to at two depths")
	level3 := func(levels int, pairs ...currency.Pair) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.AllOrdersChannel, QualifiedChannel: wsChannelLevel3, Asset: asset.Spot, Levels: levels, Pairs: pairs}
	}
	assert.NoError(t, e.ValidateSubscriptions(subscription.List{level3(10, wsTestPair), level3(10, wsTestPair), level3(100, wsTestDOGEPair), book(10, currency.NewPairWithDelimiter("ETH", "USD", "_"))}), "ValidateSubscriptions should accept the level 3 depths Kraken streams")
	for _, depth := range []int{0, 25, 500} {
		assert.ErrorIsf(t, e.ValidateSubscriptions(subscription.List{level3(depth, wsTestPair)}), errInvalidDepth, "ValidateSubscriptions should reject a level 3 depth of %d", depth)
	}
	assert.ErrorIs(t, e.ValidateSubscriptions(subscription.List{level3(10, wsTestPair), level3(100, wsTestPair)}), subscription.ErrExclusiveSubscription, "ValidateSubscriptions should reject a pair's level 3 order book subscribed to at two depths")
	assert.ErrorIs(t, e.ValidateSubscriptions(subscription.List{book(10, wsTestPair), level3(10, wsTestPair)}), subscription.ErrExclusiveSubscription, "ValidateSubscriptions should reject a pair's level 2 and level 3 order books, which share one book")
	for _, interval := range []kline.Interval{0, kline.ThirtySecond, kline.OneMin + kline.Interval(time.Nanosecond), kline.Interval(90 * time.Second), kline.ThreeMin, kline.TwoWeek} {
		assert.ErrorIsf(t, e.ValidateSubscriptions(subscription.List{ohlc(interval)}), kline.ErrUnsupportedInterval, "ValidateSubscriptions should reject an interval of %s", interval)
	}
}

func TestGroupSubscriptions(t *testing.T) {
	t.Parallel()
	ethUSD := currency.NewPairWithDelimiter("ETH", "USD", "_")
	tickerXBT := &subscription.Subscription{QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{wsTestPair}}
	tickerETH := &subscription.Subscription{QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{ethUSD}}
	book10 := &subscription.Subscription{QualifiedChannel: wsChannelBook, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	book100 := &subscription.Subscription{QualifiedChannel: wsChannelBook, Levels: 100, Pairs: currency.Pairs{ethUSD}}
	ohlc1 := &subscription.Subscription{QualifiedChannel: wsChannelOHLC, Interval: kline.OneMin, Pairs: currency.Pairs{wsTestPair}}
	ohlc5 := &subscription.Subscription{QualifiedChannel: wsChannelOHLC, Interval: kline.FiveMin, Pairs: currency.Pairs{wsTestPair}}
	executions := &subscription.Subscription{QualifiedChannel: wsChannelExecutions}
	balances := &subscription.Subscription{QualifiedChannel: wsChannelBalances}
	instruments := &subscription.Subscription{QualifiedChannel: wsChannelInstrument}
	instrumentsAgain := &subscription.Subscription{QualifiedChannel: wsChannelInstrument}
	got := groupSubscriptions(subscription.List{tickerXBT, book10, tickerETH, book100, ohlc1, ohlc5, executions, balances, instruments, instrumentsAgain})
	exp := []subscription.List{{tickerXBT, tickerETH}, {book10}, {book100}, {ohlc1}, {ohlc5}, {executions}, {balances}, {instruments}, {instrumentsAgain}}
	assert.Equal(t, exp, got, "groupSubscriptions should group only subscriptions that differ by pair, keeping their order")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	server := new(mockSubscriptionServer)
	var url string
	if mockTests {
		url = newMockWebsocketURL(t, server.handle)
	}
	useTestWebsocket(t, ex, url, wsPublicConnection, wsPrivateConnection)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
	exp := expectedSubscriptions(ex)
	subs := ex.Websocket.GetSubscriptions()
	testsubs.EqualLists(t, exp, subs.Clone())
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s)
	}
	assert.True(t, ex.assetNames.seeded(), "connecting should seed the asset names")
	depth, ok := ex.bookDepth(wsTestPair)
	assert.True(t, ok, "subscribing should record the order book depth")
	assert.Equal(t, 1000, depth, "the recorded order book depth should be the subscribed one")
	if mockTests {
		assertRequests(t, []string{
			`{"method":"subscribe","params":{"channel":"ticker","symbol":["BTC/USD"]}}`,
			`{"method":"subscribe","params":{"channel":"trade","symbol":["BTC/USD"]}}`,
			`{"method":"subscribe","params":{"channel":"ohlc","symbol":["BTC/USD"],"interval":1}}`,
			`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":1000}}`,
			`{"method":"subscribe","params":{"channel":"executions","snap_orders":true,"snap_trades":true,"token":"` + wsTestToken + `"}}`,
		}, server.takeRequests())
	}

	for _, filter := range []string{wsPublicConnection, wsPrivateConnection} {
		connSubs := slices.DeleteFunc(slices.Clone(subs), func(s *subscription.Subscription) bool {
			return s.Authenticated != (filter == wsPrivateConnection)
		})
		conn, err := ex.Websocket.GetConnection(filter)
		if len(connSubs) == 0 {
			assert.ErrorIsf(t, err, websocket.ErrNotConnected, "the %s connection should not connect without subscriptions", filter)
			continue
		}
		require.NoErrorf(t, err, "GetConnection must not error for the %s connection", filter)
		require.NoErrorf(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, connSubs), "UnsubscribeChannels must not error for the %s connection", filter)
	}
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "UnsubscribeChannels should remove every subscription")
	_, ok = ex.bookDepth(wsTestPair)
	assert.False(t, ok, "unsubscribing should forget the order book depth")
	if mockTests {
		assertRequests(t, []string{
			`{"method":"unsubscribe","params":{"channel":"ticker","symbol":["BTC/USD"]}}`,
			`{"method":"unsubscribe","params":{"channel":"trade","symbol":["BTC/USD"]}}`,
			`{"method":"unsubscribe","params":{"channel":"ohlc","symbol":["BTC/USD"],"interval":1}}`,
			`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":1000}}`,
			`{"method":"unsubscribe","params":{"channel":"executions","token":"` + wsTestToken + `"}}`,
		}, server.takeRequests())
	}
}

// newSubscriptionTestExchange returns an exchange whose public connection, connected without subscriptions, is served
// by server
func newSubscriptionTestExchange(t *testing.T, server *mockSubscriptionServer) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsPublicConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	return ex, conn
}

func TestSubscribeRejected(t *testing.T) {
	t.Parallel()
	server := &mockSubscriptionServer{rejected: map[string]string{"DOGE/USD": "Currency pair not supported DOGE/USD"}}
	ex, conn := newSubscriptionTestExchange(t, server)
	accepted := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	rejected := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestDOGEPair}}
	err := ex.subscribeForConnection(t.Context(), conn, subscription.List{accepted, rejected})
	require.ErrorIs(t, err, errAPIResponse, "a rejected subscription must error")
	assert.ErrorContains(t, err, "Currency pair not supported DOGE/USD", "the error should carry Kraken's reason")
	assertRequests(t, []string{`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD","DOGE/USD"],"depth":10}}`}, server.takeRequests())
	assert.NotNil(t, ex.Websocket.GetSubscription(accepted), "the accepted subscription should be stored")
	assert.Nil(t, ex.Websocket.GetSubscription(rejected), "the rejected subscription should not be stored")
	_, ok := ex.bookDepth(wsTestPair)
	assert.True(t, ok, "the accepted subscription's order book depth should be recorded")
	_, ok = ex.bookDepth(wsTestDOGEPair)
	assert.False(t, ok, "the rejected subscription's order book depth should be forgotten")
}

func TestSubscribeUnexpectedReply(t *testing.T) {
	t.Parallel()
	ex, conn := newSubscriptionTestExchange(t, &mockSubscriptionServer{unexpected: true})
	s := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	err := ex.subscribeForConnection(t.Context(), conn, subscription.List{s})
	assert.ErrorIs(t, err, errUnexpectedSubscriptionReply, "a reply for a symbol not requested should error")
	assert.Nil(t, ex.Websocket.GetSubscription(s), "the subscription should not be stored")
	_, ok := ex.bookDepth(wsTestPair)
	assert.False(t, ok, "the failed subscription's order book depth should be forgotten")
}

func TestSubscribeMissingReply(t *testing.T) {
	t.Parallel()
	ex, conn := newSubscriptionTestExchange(t, &mockSubscriptionServer{repeated: true})
	first := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	second := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestDOGEPair}}
	err := ex.subscribeForConnection(t.Context(), conn, subscription.List{first, second})
	assert.ErrorIs(t, err, errSubscriptionResponseMissing, "a subscription without a reply should error")
	assert.NotNil(t, ex.Websocket.GetSubscription(first), "the acknowledged subscription should be stored")
	assert.Nil(t, ex.Websocket.GetSubscription(second), "the unacknowledged subscription should not be stored")
}

func TestUnsubscribeRejected(t *testing.T) {
	t.Parallel()
	server := &mockSubscriptionServer{}
	ex, conn := newSubscriptionTestExchange(t, server)
	s := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{s}), "subscribeForConnection must not error")
	server.rejected = map[string]string{"BTC/USD": "Subscription Not Found"}
	err := ex.unsubscribeForConnection(t.Context(), conn, subscription.List{s})
	assert.ErrorIs(t, err, errAPIResponse, "a rejected unsubscription should error")
	assert.NotNil(t, ex.Websocket.GetSubscription(s), "a subscription Kraken did not unsubscribe should be kept")
}

func TestSubscribeTokenError(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"error":["EAPI:Invalid key"]}`))
		assert.NoError(t, err, "Write should not error")
	})
	seedTestAssetNames(ex)
	server := new(mockSubscriptionServer)
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsPrivateConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(wsPrivateConnection)
	require.NoError(t, err, "GetConnection must not error")
	s := &subscription.Subscription{Channel: subscription.MyOrdersChannel, QualifiedChannel: wsChannelExecutions, Authenticated: true}
	err = ex.subscribeForConnection(t.Context(), conn, subscription.List{s})
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "a token request failing should fail the subscription")
	assert.Empty(t, server.takeRequests(), "no subscription should be sent without a token")
}

func TestWebsocketToken(t *testing.T) {
	t.Parallel()
	var requests int
	tokens := []string{"first-token", "", "second-token"}
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		token := tokens[requests]
		requests++
		_, err := w.Write([]byte(`{"error":[],"result":{"token":"` + token + `","expires":900}}`))
		assert.NoError(t, err, "Write should not error")
	})
	token, err := ex.websocketToken(t.Context(), false)
	require.NoError(t, err, "websocketToken must not error")
	assert.Equal(t, "first-token", token, "websocketToken should fetch a token when none is held")
	token, err = ex.websocketToken(t.Context(), false)
	require.NoError(t, err, "websocketToken must not error")
	assert.Equal(t, "first-token", token, "websocketToken should reuse the token held")
	assert.Equal(t, 1, requests, "websocketToken should not fetch a token while one is held")
	_, err = ex.websocketToken(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketTokenEmpty, "websocketToken should reject an empty token")
	token, err = ex.websocketToken(t.Context(), true)
	require.NoError(t, err, "websocketToken must not error")
	assert.Equal(t, "second-token", token, "websocketToken should fetch a new token when refreshing")
	assert.Equal(t, 3, requests, "websocketToken should fetch a token on each refresh")
}

func TestChannelPayloads(t *testing.T) {
	t.Parallel()
	ts := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339Nano, s)
		require.NoErrorf(t, err, "Parse must not error for %s", s)
		return v
	}
	level := func(price, qty string) WebsocketBookLevel {
		p, err := types.NewPreciseNumberFromString(price)
		require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", price)
		q, err := types.NewPreciseNumberFromString(qty)
		require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", qty)
		return WebsocketBookLevel{Price: p, Quantity: q}
	}
	for _, tc := range []struct {
		fixture string
		exp     []any
	}{
		{"testdata/wsPublic.json", []any{
			&[]WebsocketStatus{{
				System:       "online",
				APIVersion:   "v2",
				ConnectionID: 13834774380200032777,
				Version:      "2.0.0",
				UpcomingMaintenance: []MaintenanceEvent{{
					EventID:            21,
					Title:              "Scheduled Maintenance",
					ExpectedStart:      ts("2025-08-12T02:00:00Z"),
					ExpectedEnd:        ts("2025-08-12T03:00:00Z"),
					TimeToStartSeconds: 1800,
					Phase:              "approaching_30m",
					AffectedServices:   []string{"futures_ws", "futures_trading"},
					OrderSubmission:    "discouraged",
					RecommendedAction:  "reduce_activity",
					CancelBefore:       ts("2025-08-12T01:55:00Z"),
					SourceURL:          "https://stspg.io/mt18zxvtzz00",
				}},
				Emergency: []EmergencyEvent{{
					EventID:          4821,
					Title:            "Spot Connectivity Incident",
					IncidentStatus:   "identified",
					Impact:           "major",
					AffectedServices: []string{"futures_ws"},
					StartedAt:        ts("2025-08-12T00:14:30Z"),
					NextSteps:        []EmergencyNextStep{{AppliesTo: []string{"futures_ws"}, Type: "expected_restart", ExpectedAt: ts("2025-08-12T00:45:00Z")}},
					SourceURL:        "https://status.kraken.com/incidents/4821",
				}},
			}},
			nil,
			&[]WebsocketTicker{{
				Symbol:                     "BTC/USD",
				Bid:                        81776.6,
				BidQuantity:                0.47200852,
				Ask:                        81776.7,
				AskQuantity:                0.00050857,
				Last:                       81776.6,
				Volume:                     4031.89061547,
				VolumeWeightedAveragePrice: 81871.6,
				Low:                        80328.6,
				High:                       83467.7,
				Change:                     -1448.7,
				ChangePercentage:           -1.74,
				Trades:                     197725,
				Timestamp:                  ts("2026-10-09T00:34:21.060761Z"),
			}},
			&[]WebsocketTrade{
				{Symbol: "BTC/USD", Side: "buy", Price: 81776.7, Quantity: 0.00005, OrderType: "limit", TradeID: 110763305, Timestamp: ts("2026-10-09T00:34:22.814908Z")},
				{Symbol: "BTC/USD", Side: "buy", Price: 81776.7, Quantity: 0.00000879, OrderType: "limit", TradeID: 110763306, Timestamp: ts("2026-10-09T00:34:22.814908Z")},
			},
			&[]WebsocketCandle{{
				Symbol:                     "BTC/USD",
				Open:                       81780.8,
				High:                       81780.8,
				Low:                        81767.3,
				Close:                      81776.7,
				VolumeWeightedAveragePrice: 81775.8,
				Trades:                     29,
				Volume:                     0.02155518,
				IntervalBegin:              ts("2026-10-09T00:34:00Z"),
				IntervalMinutes:            1,
			}},
			&[]WebsocketBook{{
				Symbol: "BTC/USD",
				Bids: []WebsocketBookLevel{
					level("81776.6", "0.47200852"), level("81776.1", "0.11790609"), level("81776.0", "1.84423887"), level("81775.6", "3.05714315"),
					level("81774.3", "3.05719335"), level("81773.2", "3.05723288"), level("81772.7", "0.00005100"), level("81768.7", "0.93754440"),
					level("81768.6", "3.05745897"), level("81768.4", "0.06320000"),
				},
				Asks: []WebsocketBookLevel{
					level("81776.7", "0.00050857"), level("81777.1", "0.00012500"), level("81780.7", "0.00005100"), level("81781.8", "0.02016285"),
					level("81781.9", "0.05474923"), level("81782.0", "0.18341452"), level("81783.1", "0.00012500"), level("81783.5", "0.05000000"),
					level("81783.7", "0.93145626"), level("81783.8", "3.05684048"),
				},
				Checksum:  2459006653,
				Timestamp: ts("2026-10-09T00:34:21.169750Z"),
			}},
			&[]WebsocketBook{{Symbol: "BTC/USD", Bids: []WebsocketBookLevel{}, Asks: []WebsocketBookLevel{level("81781.9", "0.00000000"), level("81784.8", "0.00005100")}, Checksum: 3437785338, Timestamp: ts("2026-10-09T00:34:21.904206Z")}},
			&[]WebsocketBook{{Symbol: "BTC/USD", Bids: []WebsocketBookLevel{}, Asks: []WebsocketBookLevel{level("81782.0", "0.00000000"), level("81784.9", "0.06320000")}, Checksum: 1758305755, Timestamp: ts("2026-10-09T00:34:21.910654Z")}},
			&WebsocketInstruments{
				Assets: []WebsocketInstrumentAsset{
					{ID: "BTC", Status: "enabled", Precision: 10, PrecisionDisplay: 5, Borrowable: true, CollateralValue: 0.99, Class: "currency", MarginRate: 0.01},
					{ID: "TSLAx", UnderlyingSymbol: "TSLA", Precision: 8, PrecisionDisplay: 8, CollateralValue: 0.8, MaximumCollateralUSDValue: 250000, Class: "tokenized_asset", Multiplier: 1},
				},
				Pairs: []WebsocketInstrumentPair{{
					Symbol:                         "BTC/USD",
					Base:                           "BTC",
					Quote:                          "USD",
					Status:                         "online",
					QuantityPrecision:              8,
					QuantityIncrement:              1e-08,
					PricePrecision:                 1,
					CostPrecision:                  5,
					MarginTradable:                 true,
					WebsocketDisplayPricePrecision: 1,
					CostMinimum:                    0.5,
					MarginInitial:                  0.1,
					PositionLimitLong:              350,
					PositionLimitShort:             250,
					PriceIncrement:                 0.1,
					QuantityMinimum:                5e-05,
				}},
			},
		}},
		{"testdata/wsPrivate.json", []any{
			&[]WebsocketExecution{{
				OrderID:                  "OAIYAU-LGI3M-PFM5VW",
				OrderUserReference:       100054,
				ClientOrderID:            "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
				Symbol:                   "DOGE/USD",
				Side:                     "buy",
				OrderType:                "stop-loss-limit",
				OrderQuantity:            1200,
				CashOrderQuantity:        252,
				LimitPrice:               0.2101,
				LimitPriceType:           "static",
				StopPrice:                0.21,
				TimeInForce:              "GTD",
				PostOnly:                 true,
				ReduceOnly:               true,
				Margin:                   true,
				NoMarketPriceProtection:  true,
				FeeCurrencyPreference:    "fcib",
				DisplayQuantity:          100,
				DisplayQuantityRemaining: 80,
				EffectiveTime:            ts("2026-10-09T01:00:00Z"),
				ExpireTime:               ts("2026-10-10T01:00:00Z"),
				Triggers: WebsocketExecutionTriggers{
					Reference:   "index",
					Price:       0.21,
					PriceType:   "static",
					ActualPrice: 0.21,
					PeakPrice:   0.2205,
					LastPrice:   0.2099,
					Status:      "triggered",
					Timestamp:   ts("2026-10-09T01:05:00Z"),
				},
				Contingent: WebsocketExecutionContingent{
					OrderType:        "take-profit-limit",
					TriggerPrice:     5,
					TriggerPriceType: "pct",
					LimitPrice:       4.5,
					LimitPriceType:   "pct",
				},
				ExecutionType:       "trade",
				OrderStatus:         "partially_filled",
				Amended:             true,
				Liquidated:          true,
				Reason:              "User requested",
				ExecutionID:         "TFEYXB-5OO6U-H7NFC5",
				TradeID:             91342277,
				LastQuantity:        20,
				LastPrice:           0.2101,
				LiquidityIndicator:  "m",
				Cost:                4.202,
				MarginBorrow:        true,
				CumulativeQuantity:  20,
				CumulativeCost:      4.202,
				AveragePrice:        0.2101,
				Fees:                []WebsocketExecutionFee{{Asset: "USD", Quantity: 0.0067}},
				FeeUSDEquivalent:    0.0067,
				PositionStatus:      "opened",
				OrderReferenceID:    "OLUMT4-UTEGU-ZYM7E9",
				ExternalOrderID:     "b9e0b9b0-6c6d-4a0f-9f0e-7d6f3f2b1a10",
				ExternalExecutionID: "5d3c9e1a-2b4f-4c8e-9a7d-1e2f3a4b5c6d",
				SenderSubID:         "trader-7",
				User:                "AA96N74GCGEFN8KI",
				RateCount:           12,
				Timestamp:           ts("2026-10-09T01:05:00.12Z"),
				Trigger:             "index",
				TriggeredPrice:      0.2099,
				CancelReason:        "User requested",
			}},
			&[]WebsocketExecution{{
				OrderID:               "OK4GJX-KSTLS-7DZZO5",
				OrderUserReference:    3,
				Symbol:                "BTC/USD",
				OrderQuantity:         0.005,
				TimeInForce:           "GTC",
				ExecutionType:         "pending_new",
				Side:                  "sell",
				OrderType:             "limit",
				LimitPriceType:        "static",
				LimitPrice:            26500,
				OrderStatus:           "pending_new",
				FeeCurrencyPreference: "fciq",
				Timestamp:             ts("2023-09-22T10:33:05.709950Z"),
			}},
			&[]WebsocketExecution{{Timestamp: ts("2023-09-22T10:33:05.709982Z"), OrderStatus: "new", ExecutionType: "new", OrderUserReference: 3, OrderID: "OK4GJX-KSTLS-7DZZO5"}},
			&[]WebsocketExecution{{
				OrderID:            "OK4GJX-KSTLS-7DZZO5",
				OrderUserReference: 3,
				ExecutionID:        "TGBB7L-HT5LX-J3BZ4A",
				ExecutionType:      "trade",
				TradeID:            62887576,
				Symbol:             "BTC/USD",
				Side:               "sell",
				LastQuantity:       0.005,
				LastPrice:          26599.9,
				LiquidityIndicator: "t",
				Cost:               132.9995,
				OrderType:          "limit",
				Timestamp:          ts("2023-09-22T10:33:05.709993Z"),
				OrderStatus:        "partially_filled",
				CumulativeQuantity: 0.005,
				CumulativeCost:     132.9995,
				AveragePrice:       26599.9,
				FeeUSDEquivalent:   0.3458,
				Fees:               []WebsocketExecutionFee{{Asset: "USD", Quantity: 0.3458}},
			}},
			&[]WebsocketBalance{
				{Asset: "BTC", AssetClass: "currency", Balance: 1.2, Wallets: []WebsocketWallet{{Type: "spot", ID: "main", Balance: 1.2}}},
				{Asset: "MATIC", AssetClass: "currency", Balance: 500, Wallets: []WebsocketWallet{{Type: "spot", ID: "main", Balance: 300}, {Type: "earn", ID: "flex", Balance: 200}}},
				{Asset: "USD", AssetClass: "currency", Balance: 80595.4943, Wallets: []WebsocketWallet{{Type: "spot", ID: "main", Balance: 80595.4943}}},
			},
			&[]WebsocketLedgerEntry{{
				LedgerID:    "ADKKFF-WEA5A-CNUBHG",
				ReferenceID: "AGBWUJRU-LAREZ-W3UFAN",
				Timestamp:   ts("2023-09-22T10:23:42.925034Z"),
				Type:        "deposit",
				Asset:       "BTC",
				AssetClass:  "currency",
				Category:    "deposit",
				WalletType:  "spot",
				WalletID:    "main",
				Amount:      0.01,
				Balance:     0.02,
			}},
			&[]WebsocketLedgerEntry{{
				LedgerID:    "LQZ4GN-2WD7E-OIQAB5",
				ReferenceID: "RUSB7W6-ESIXUX-K6PVTM",
				Timestamp:   ts("2023-09-22T10:41:00Z"),
				Type:        "transfer",
				Subtype:     "spottostaking",
				Asset:       "DOT",
				AssetClass:  "currency",
				Category:    "withdrawal",
				WalletType:  "spot",
				WalletID:    "main",
				Amount:      -10,
				Balance:     40,
				User:        "AA96N74GCGEFN8KI",
			}},
		}},
	} {
		lines := fixtureLines(t, tc.fixture)
		require.Lenf(t, lines, len(tc.exp), "%s must have one message for each expected payload", tc.fixture)
		for i, line := range lines {
			var msg wsChannelMessage
			require.NoErrorf(t, json.Unmarshal(line, &msg), "Unmarshal must not error for %s message %d", tc.fixture, i)
			if tc.exp[i] == nil {
				assert.Emptyf(t, msg.Data, "%s message %d should have no data", tc.fixture, i)
				continue
			}
			got := reflect.New(reflect.TypeOf(tc.exp[i]).Elem()).Interface()
			require.NoErrorf(t, json.Unmarshal(msg.Data, got), "Unmarshal must not error for %s message %d", tc.fixture, i)
			assert.Equalf(t, tc.exp[i], got, "%s message %d on %s should decode every field", tc.fixture, i, msg.Channel)
		}
	}
}

// relayed returns the payloads relayed so far
func relayed(ex *Exchange) []any {
	var payloads []any
	for len(ex.Websocket.DataHandler.C) > 0 {
		payloads = append(payloads, (<-ex.Websocket.DataHandler.C).Data)
	}
	return payloads
}

func bookLevel(price float64, strPrice string, amount float64, strAmount string) orderbook.Level {
	return orderbook.Level{Price: price, StrPrice: strPrice, Amount: amount, StrAmount: strAmount}
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// The fixture's book was recorded with a depth of 10
	ex.setBookDepth(wsTestPair, 10)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	testexch.FixtureToDataHandler(t, "testdata/wsPublic.json", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, nil, b)
	})
	got := relayed(ex)
	require.Len(t, got, 8, "wsHandleData must relay every payload")

	status, ok := got[0].(*WebsocketStatus)
	require.True(t, ok, "the first payload must be the status")
	assert.Equal(t, "online", status.System, "the status should be relayed")

	expTicker := &ticker.Price{
		ExchangeName:               t.Name(),
		AssetType:                  asset.Spot,
		Pair:                       wsTestPair,
		Last:                       81776.6,
		Close:                      81776.6,
		High:                       83467.7,
		Low:                        80328.6,
		Bid:                        81776.6,
		BidSize:                    0.47200852,
		Ask:                        81776.7,
		AskSize:                    0.00050857,
		BaseVolume:                 4031.89061547,
		VolumeWeightedAveragePrice: 81871.6,
		Open24Hour:                 83225.3,
		PercentChange24Hour:        -1.74,
		LastUpdated:                time.Date(2026, 10, 9, 0, 34, 21, 60761000, time.UTC),
	}
	assert.Equal(t, []ticker.Price{*expTicker}, got[1], "the ticker should be relayed as a batch")
	stored, err := ticker.GetTicker(t.Name(), wsTestPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, expTicker, stored, "the ticker should be stored as it was relayed")

	tradeTime := time.Date(2026, 10, 9, 0, 34, 22, 814908000, time.UTC)
	assert.Equal(t, []trade.Data{
		{Exchange: t.Name(), CurrencyPair: wsTestPair, AssetType: asset.Spot, TID: "110763305", Side: order.Buy, Price: 81776.7, Amount: 0.00005, Timestamp: tradeTime},
		{Exchange: t.Name(), CurrencyPair: wsTestPair, AssetType: asset.Spot, TID: "110763306", Side: order.Buy, Price: 81776.7, Amount: 0.00000879, Timestamp: tradeTime},
	}, got[2], "the trades should be relayed")

	assert.Equal(t, kline.Item{
		Exchange: t.Name(),
		Pair:     wsTestPair,
		Asset:    asset.Spot,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:             time.Date(2026, 10, 9, 0, 34, 0, 0, time.UTC),
			Open:             81780.8,
			High:             81780.8,
			Low:              81767.3,
			Close:            81776.7,
			Volume:           0.02155518,
			QuoteVolume:      1762.692088644,
			ValidationIssues: kline.PartialCandle,
		}},
	}, got[3], "the candle, whose interval had not ended, should be relayed as partial")

	for i := 4; i < 7; i++ {
		assert.IsTypef(t, &orderbook.Depth{}, got[i], "payload %d should be the order book", i)
	}
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{
		bookLevel(81776.6, "81776.6", 0.47200852, "0.47200852"), bookLevel(81776.1, "81776.1", 0.11790609, "0.11790609"),
		bookLevel(81776, "81776.0", 1.84423887, "1.84423887"), bookLevel(81775.6, "81775.6", 3.05714315, "3.05714315"),
		bookLevel(81774.3, "81774.3", 3.05719335, "3.05719335"), bookLevel(81773.2, "81773.2", 3.05723288, "3.05723288"),
		bookLevel(81772.7, "81772.7", 0.000051, "0.00005100"), bookLevel(81768.7, "81768.7", 0.9375444, "0.93754440"),
		bookLevel(81768.6, "81768.6", 3.05745897, "3.05745897"), bookLevel(81768.4, "81768.4", 0.0632, "0.06320000"),
	}, book.Bids, "the bids should be the snapshot's")
	assert.Equal(t, orderbook.Levels{
		bookLevel(81776.7, "81776.7", 0.00050857, "0.00050857"), bookLevel(81777.1, "81777.1", 0.000125, "0.00012500"),
		bookLevel(81780.7, "81780.7", 0.000051, "0.00005100"), bookLevel(81781.8, "81781.8", 0.02016285, "0.02016285"),
		bookLevel(81783.1, "81783.1", 0.000125, "0.00012500"), bookLevel(81783.5, "81783.5", 0.05, "0.05000000"),
		bookLevel(81783.7, "81783.7", 0.93145626, "0.93145626"), bookLevel(81783.8, "81783.8", 3.05684048, "3.05684048"),
		bookLevel(81784.8, "81784.8", 0.000051, "0.00005100"), bookLevel(81784.9, "81784.9", 0.0632, "0.06320000"),
	}, book.Asks, "the asks should have the updates applied")
	assert.Equal(t, time.Date(2026, 10, 9, 0, 34, 21, 910654000, time.UTC), book.LastUpdated, "the book should be as of the last update")

	instruments, ok := got[7].(*WebsocketInstruments)
	require.True(t, ok, "the last payload must be the instruments")
	assert.Len(t, instruments.Pairs, 1, "the instruments should be relayed")
}

// TestWebsocketBookSequence replays a recorded depth 10 order book snapshot and every update that followed it, each of
// which must pass its checksum
func TestWebsocketBookSequence(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.setBookDepth(wsTestPair, 10)
	testexch.FixtureToDataHandler(t, "testdata/wsBook.json", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, nil, b)
	})
	assert.Len(t, relayed(ex), 20, "every snapshot and update should relay the order book")
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{
		bookLevel(81776.6, "81776.6", 0.67200852, "0.67200852"), bookLevel(81776.1, "81776.1", 0.11790609, "0.11790609"),
		bookLevel(81776, "81776.0", 1.84423887, "1.84423887"), bookLevel(81775.6, "81775.6", 3.05714315, "3.05714315"),
		bookLevel(81774.3, "81774.3", 3.05719335, "3.05719335"), bookLevel(81773.2, "81773.2", 3.05723288, "3.05723288"),
		bookLevel(81772.7, "81772.7", 0.000051, "0.00005100"), bookLevel(81768.7, "81768.7", 0.9375444, "0.93754440"),
		bookLevel(81768.6, "81768.6", 3.05745897, "3.05745897"), bookLevel(81768.4, "81768.4", 0.0632, "0.06320000"),
	}, book.Bids, "the bids should have the updates applied")
	assert.Equal(t, orderbook.Levels{
		bookLevel(81776.7, "81776.7", 0.00050857, "0.00050857"), bookLevel(81777.1, "81777.1", 0.000125, "0.00012500"),
		bookLevel(81780.7, "81780.7", 0.000051, "0.00005100"), bookLevel(81783.1, "81783.1", 0.000125, "0.00012500"),
		bookLevel(81783.5, "81783.5", 0.05, "0.05000000"), bookLevel(81784.8, "81784.8", 0.000051, "0.00005100"),
		bookLevel(81785.7, "81785.7", 0.000365, "0.00036500"), bookLevel(81787.3, "81787.3", 0.0002, "0.00020000"),
		bookLevel(81788.9, "81788.9", 0.000051, "0.00005100"), bookLevel(81789.1, "81789.1", 0.00104194, "0.00104194"),
	}, book.Asks, "the asks should have the updates applied and be truncated to the subscribed depth")
	assert.Equal(t, time.Date(2026, 10, 9, 0, 34, 22, 115897000, time.UTC), book.LastUpdated, "the book should be as of the last update")
}

func TestWebsocketHandleDataPrivate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	useTestCredentials(ex)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	testexch.FixtureToDataHandler(t, "testdata/wsPrivate.json", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, nil, b)
	})
	got := relayed(ex)
	require.Len(t, got, 9, "wsHandleData must relay every payload")

	tradeTime := time.Date(2026, 10, 9, 1, 5, 0, 120000000, time.UTC)
	pendingTime := time.Date(2023, 9, 22, 10, 33, 5, 709950000, time.UTC)
	fillTime := time.Date(2023, 9, 22, 10, 33, 5, 709993000, time.UTC)
	exp := []any{
		&order.Detail{
			Exchange:             t.Name(),
			AssetType:            asset.Spot,
			OrderID:              "OAIYAU-LGI3M-PFM5VW",
			ClientOrderID:        "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
			Pair:                 wsTestDOGEPair,
			FeeAsset:             currency.USD,
			Side:                 order.Buy,
			Type:                 order.StopLimit,
			Status:               order.PartiallyFilled,
			TimeInForce:          order.GoodTillTime | order.PostOnly,
			Amount:               1200,
			ExecutedAmount:       20,
			AverageExecutedPrice: 0.2101,
			Cost:                 4.202,
			Price:                0.2101,
			TriggerPrice:         0.21,
			ReduceOnly:           true,
			LastUpdated:          tradeTime,
			Trades: []order.TradeHistory{{
				TID:       "TFEYXB-5OO6U-H7NFC5",
				Price:     0.2101,
				Amount:    20,
				Fee:       0.0067,
				FeeAsset:  "USD",
				Exchange:  t.Name(),
				Type:      order.StopLimit,
				Side:      order.Buy,
				IsMaker:   true,
				Timestamp: tradeTime,
				Total:     4.202,
			}},
		},
		[]fill.Data{{
			ID:            "TFEYXB-5OO6U-H7NFC5",
			Timestamp:     tradeTime,
			Exchange:      t.Name(),
			AssetType:     asset.Spot,
			CurrencyPair:  wsTestDOGEPair,
			Side:          order.Buy,
			OrderID:       "OAIYAU-LGI3M-PFM5VW",
			ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
			TradeID:       "91342277",
			Price:         0.2101,
			Amount:        20,
		}},
		&order.Detail{
			Exchange:    t.Name(),
			AssetType:   asset.Spot,
			OrderID:     "OK4GJX-KSTLS-7DZZO5",
			Pair:        wsTestPair,
			FeeAsset:    currency.USD,
			Side:        order.Sell,
			Type:        order.Limit,
			Status:      order.Pending,
			TimeInForce: order.GoodTillCancel,
			Amount:      0.005,
			Price:       26500,
			Date:        pendingTime,
			LastUpdated: pendingTime,
		},
		&order.Detail{
			Exchange:    t.Name(),
			AssetType:   asset.Spot,
			OrderID:     "OK4GJX-KSTLS-7DZZO5",
			Status:      order.New,
			LastUpdated: time.Date(2023, 9, 22, 10, 33, 5, 709982000, time.UTC),
		},
		&order.Detail{
			Exchange:             t.Name(),
			AssetType:            asset.Spot,
			OrderID:              "OK4GJX-KSTLS-7DZZO5",
			Pair:                 wsTestPair,
			FeeAsset:             currency.USD,
			Side:                 order.Sell,
			Type:                 order.Limit,
			Status:               order.PartiallyFilled,
			ExecutedAmount:       0.005,
			AverageExecutedPrice: 26599.9,
			Cost:                 132.9995,
			LastUpdated:          fillTime,
			Trades: []order.TradeHistory{{
				TID:       "TGBB7L-HT5LX-J3BZ4A",
				Price:     26599.9,
				Amount:    0.005,
				Fee:       0.3458,
				FeeAsset:  "USD",
				Exchange:  t.Name(),
				Type:      order.Limit,
				Side:      order.Sell,
				Timestamp: fillTime,
				Total:     132.9995,
			}},
		},
		[]fill.Data{{
			ID:           "TGBB7L-HT5LX-J3BZ4A",
			Timestamp:    fillTime,
			Exchange:     t.Name(),
			AssetType:    asset.Spot,
			CurrencyPair: wsTestPair,
			Side:         order.Sell,
			OrderID:      "OK4GJX-KSTLS-7DZZO5",
			TradeID:      "62887576",
			Price:        26599.9,
			Amount:       0.005,
		}},
	}
	for i := range exp {
		assert.Equalf(t, exp[i], got[i], "payload %d should be relayed", i)
	}

	balances, ok := got[6].([]WebsocketBalance)
	require.True(t, ok, "the balances snapshot must be relayed as its balances")
	require.Len(t, balances, 3, "the balances snapshot must relay each asset's balance")
	assert.Equal(t, WebsocketBalance{
		Asset:      "MATIC",
		AssetClass: "currency",
		Balance:    500,
		Wallets:    []WebsocketWallet{{Type: "spot", ID: "main", Balance: 300}, {Type: "earn", ID: "flex", Balance: 200}},
	}, balances[1], "the balances snapshot should be relayed as sent")

	entries, ok := got[7].([]WebsocketLedgerEntry)
	require.True(t, ok, "a balances update must be relayed as its ledger entries")
	assert.Equal(t, "ADKKFF-WEA5A-CNUBHG", entries[0].LedgerID, "the ledger entry should be relayed")
	assert.IsType(t, []WebsocketLedgerEntry{}, got[8], "the second balances update should be relayed")
}

func TestWebsocketHandleDataErrors(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	conn, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for _, tc := range []struct {
		name string
		msg  string
		err  error
	}{
		{"unmatched reply", `{"method":"subscribe","req_id":42,"success":true}`, websocket.ErrSignatureNotMatched},
		{"engine not online", `{"channel":"status","type":"update","data":[{"system":"maintenance"}]}`, errSystemNotOnline},
		{"ticker symbol", `{"channel":"ticker","type":"update","data":[{"symbol":"BTCUSD"}]}`, errInvalidSymbol},
		{"book symbol", `{"channel":"book","type":"snapshot","data":[{"symbol":"BTC"}]}`, errInvalidSymbol},
		{"book without a snapshot", `{"channel":"book","type":"update","data":[{"symbol":"DOGE/USD","bids":[{"price":0.2,"qty":1}],"asks":[],"checksum":1,"timestamp":"2026-10-09T00:00:00Z"}]}`, orderbook.ErrDepthNotFound},
		{"ohlc symbol", `{"channel":"ohlc","type":"update","data":[{"symbol":""}]}`, errInvalidSymbol},
		{"trade side", `{"channel":"trade","type":"update","data":[{"symbol":"BTC/USD","side":"up"}]}`, order.ErrSideIsInvalid},
		{"trade symbol", `{"channel":"trade","type":"update","data":[{"symbol":"BTC-USD","side":"buy"}]}`, errInvalidSymbol},
		{"execution symbol", `{"channel":"executions","type":"update","data":[{"symbol":"BTC"}]}`, errInvalidSymbol},
		{"execution side", `{"channel":"executions","type":"update","data":[{"side":"up"}]}`, order.ErrSideIsInvalid},
		{"execution order type", `{"channel":"executions","type":"update","data":[{"order_type":"moon"}]}`, errUnknownOrderType},
		{"execution order status", `{"channel":"executions","type":"update","data":[{"order_status":"moon"}]}`, errUnknownOrderStatus},
		{"execution time in force", `{"channel":"executions","type":"update","data":[{"time_in_force":"GTX"}]}`, order.ErrInvalidTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(tc.msg)), tc.err, "wsHandleData should return the expected error")
		})
	}
	for _, msg := range []string{
		`not json`,
		`{"channel":"status","data":"bad"}`,
		`{"channel":"ticker","data":"bad"}`,
		`{"channel":"book","data":"bad"}`,
		`{"channel":"ohlc","data":"bad"}`,
		`{"channel":"trade","data":"bad"}`,
		`{"channel":"instrument","data":"bad"}`,
		`{"channel":"executions","data":"bad"}`,
		`{"channel":"balances","type":"snapshot","data":"bad"}`,
		`{"channel":"balances","type":"update","data":"bad"}`,
	} {
		assert.Errorf(t, ex.wsHandleData(t.Context(), conn, []byte(msg)), "wsHandleData should reject %s", msg)
	}
}

func TestWebsocketProcessTickerBatch(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	batch := &wsChannelMessage{Channel: wsChannelTicker, Type: wsTypeSnapshot, Data: json.RawMessage(`[{"symbol":"BTCUSD"},{"symbol":"BTC/USD","bid":81776.6,"bid_qty":0.47200852,"ask":81776.7,"ask_qty":0.00050857,"last":81776.6,"volume":4031.89061547,"vwap":81871.6,"low":80328.6,"high":83467.7,"change":-1448.7,"change_pct":-1.74,"trades":197725,"timestamp":"2026-10-09T00:34:21.060761Z"}]`)}
	assert.ErrorIs(t, ex.wsProcessTicker(t.Context(), batch), errInvalidSymbol, "wsProcessTicker should report a ticker it cannot store")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessTicker must relay the tickers it stored")
	relayed, ok := (<-ex.Websocket.DataHandler.C).Data.([]ticker.Price)
	require.True(t, ok, "wsProcessTicker must relay the stored tickers as a batch")
	require.Len(t, relayed, 1, "the relayed batch must hold only the stored ticker")
	assert.Equal(t, wsTestPair, relayed[0].Pair, "the ticker after the one that failed should be relayed")
	stored, err := ticker.GetTicker(ex.Name, wsTestPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error for the ticker after the one that failed")
	assert.Equal(t, relayed[0], *stored, "the relayed ticker should be the stored one")

	assert.ErrorIs(t, ex.wsProcessTicker(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTCUSD"}]`)}), errInvalidSymbol, "wsProcessTicker should report a ticker it cannot store")
	assert.Empty(t, ex.Websocket.DataHandler.C, "wsProcessTicker should not relay an empty batch")
	assert.Error(t, ex.wsProcessTicker(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessTicker should reject data that is not a list")
}

// tickerMessage returns a ticker message sent at sent, holding a BTC/USD ticker stamped at stamped, or unstamped when
// stamped is zero
func tickerMessage(stamped, sent time.Time) *wsChannelMessage {
	var timestamp string
	if !stamped.IsZero() {
		timestamp = `,"timestamp":"` + stamped.Format(time.RFC3339Nano) + `"`
	}
	return &wsChannelMessage{Channel: wsChannelTicker, Type: "update", Timestamp: sent, Data: json.RawMessage(`[{"symbol":"BTC/USD","bid":81776.6,"bid_qty":0.47200852,` +
		`"ask":81776.7,"ask_qty":0.00050857,"last":81776.6,"volume":4031.89061547,"vwap":81871.6,"low":80328.6,"high":83467.7,"change":-1448.7,"change_pct":-1.74` + timestamp + `}]`)}
}

// tickerMessagePrice returns the price tickerMessage's ticker is relayed as, updated at updated
func tickerMessagePrice(exchangeName string, updated time.Time) []ticker.Price {
	return []ticker.Price{{
		ExchangeName:               exchangeName,
		AssetType:                  asset.Spot,
		Pair:                       wsTestPair,
		Last:                       81776.6,
		Close:                      81776.6,
		High:                       83467.7,
		Low:                        80328.6,
		Bid:                        81776.6,
		BidSize:                    0.47200852,
		Ask:                        81776.7,
		AskSize:                    0.00050857,
		BaseVolume:                 4031.89061547,
		VolumeWeightedAveragePrice: 81871.6,
		Open24Hour:                 83225.3,
		PercentChange24Hour:        -1.74,
		LastUpdated:                updated,
	}}
}

func TestWebsocketProcessTickerTime(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	stamped := time.Date(2026, 10, 20, 0, 0, 1, 103000000, time.UTC)
	for _, tc := range []struct {
		name          string
		stamped, sent time.Time
	}{
		{name: "stamped, in a message sent a second later", stamped: stamped, sent: stamped.Add(time.Second)},
		{name: "stamped, in a message sent the day before", stamped: stamped, sent: stamped.Add(-24 * time.Hour)},
		{name: "unstamped, in a message sent at that time", sent: stamped},
	} {
		require.NoErrorf(t, ex.wsProcessTicker(t.Context(), tickerMessage(tc.stamped, tc.sent)), "wsProcessTicker must not error for a ticker %s", tc.name)
		assert.Equalf(t, []any{tickerMessagePrice(ex.Name, stamped)}, relayed(ex), "a ticker %s should be relayed with its own time before its message's", tc.name)
	}
}

func TestWebsocketHandleDataUnhandled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for _, msg := range []string{`{"channel":"heartbeat"}`, `{"method":"pong","req_id":7}`, `{"method":"pong"}`} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(msg)), "wsHandleData must not error for %s", msg)
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "heartbeats and unmatched pongs should not be relayed")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"moon","type":"update","data":[]}`)), "wsHandleData must not error for an unhandled channel")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "an unhandled channel's message must be relayed")
	warning, ok := (<-ex.Websocket.DataHandler.C).Data.(websocket.UnhandledMessageWarning)
	require.True(t, ok, "an unhandled channel's message must be relayed as a warning")
	assert.Contains(t, warning.Message, "moon", "the warning should carry the message")
}

func TestWebsocketHandleDataTradesDisabled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = false
	ex.Features.Enabled.SaveTradeData = false
	require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(`{"channel":"trade","data":"not decoded"}`)), "wsHandleData must not decode trades nobody uses")
	assert.Empty(t, ex.Websocket.DataHandler.C, "trades should not be relayed when the trade feed is disabled")
}

func TestWebsocketHandleDataFillsDisabled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.Features.Enabled.FillsFeed = false
	msg := `{"channel":"executions","type":"update","data":[{"order_id":"OK4GJX-KSTLS-7DZZO5","exec_id":"TGBB7L-HT5LX-J3BZ4A","exec_type":"trade","trade_id":62887576,"symbol":"BTC/USD","side":"sell","last_qty":0.005,"last_price":26599.9,"timestamp":"2023-09-22T10:33:05.709993Z"}]}`
	require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(msg)), "wsHandleData must not error")
	got := relayed(ex)
	require.Len(t, got, 1, "only the order update must be relayed when the fills feed is disabled")
	assert.IsType(t, &order.Detail{}, got[0], "the order update should be relayed")
}

func TestWebsocketBookChecksumMismatch(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	snapshot := func(checksum string) []byte {
		return []byte(`{"channel":"book","type":"snapshot","data":[{"symbol":"BTC/USD","bids":[{"price":81776.6,"qty":0.47200852}],"asks":[{"price":81776.7,"qty":0.00050857}],"checksum":` + checksum + `,"timestamp":"2026-10-09T00:34:21.169750Z"}]}`)
	}
	err := ex.wsHandleData(t.Context(), nil, snapshot("1"))
	require.ErrorIs(t, err, errChecksumMismatch, "a snapshot failing its checksum must error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.Error(t, err, "a snapshot failing its checksum should invalidate the book")

	b := &orderbook.Book{
		Bids: orderbook.Levels{{Price: 81776.6, StrPrice: "81776.6", Amount: 0.47200852, StrAmount: "0.47200852"}},
		Asks: orderbook.Levels{{Price: 81776.7, StrPrice: "81776.7", Amount: 0.00050857, StrAmount: "0.00050857"}},
	}
	checksum := strconv.FormatUint(uint64(bookChecksum(b)), 10)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, snapshot(checksum)), "wsHandleData must load a snapshot passing its checksum")
	update := `{"channel":"book","type":"update","data":[{"symbol":"BTC/USD","bids":[{"price":81776.5,"qty":1}],"asks":[],"checksum":1,"timestamp":"2026-10-09T00:34:22Z"}]}`
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, []byte(update)), errChecksumMismatch, "an update failing its checksum must error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.Error(t, err, "an update failing its checksum should invalidate the book")
}

func TestResubscribeBook(t *testing.T) {
	t.Parallel()
	server := new(mockSubscriptionServer)
	ex, conn := newSubscriptionTestExchange(t, server)
	s := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{s}), "subscribeForConnection must not error")
	server.takeRequests()

	ex.resubscribeSpotBook(conn, wsChannelBook, wsTestDOGEPair)
	ex.resubscribeSpotBook(conn, wsChannelLevel3, wsTestPair)
	ex.resubscribeSpotBook(conn, wsChannelBook, wsTestPair)
	exp := []string{
		`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
		`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "resubscribeSpotBook must resubscribe the subscribed book")
	assert.Equal(t, exp, got, "resubscribeSpotBook should resubscribe only the book subscribed, unsubscribing first")
	require.Eventually(t, func() bool {
		stored := ex.Websocket.GetSubscription(s)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, testWaitTimeout, 10*time.Millisecond, "the book must be subscribed again")
}

func TestResubscribeBookChannelNames(t *testing.T) {
	t.Parallel()
	valid := bookChecksum(&orderbook.Book{
		Bids: orderbook.Levels{bookLevel(100, "100.0", 1, "1.0")},
		Asks: orderbook.Levels{bookLevel(101, "101.0", 1, "1.0")},
	})
	book := func(msgType string, checksum uint32) string {
		return `{"channel":"book","type":"` + msgType + `","data":[{"symbol":"BTC/USD","bids":[{"price":100.0,"qty":1.0}],"asks":[{"price":101.0,"qty":1.0}],"checksum":` +
			strconv.FormatUint(uint64(checksum), 10) + `,"timestamp":"2026-10-16T00:00:00Z"}]}`
	}
	level3Snapshot := `{"channel":"level3","type":"snapshot","data":[{"symbol":"BTC/USD","checksum":1,"bids":[],"asks":[],"timestamp":"2024-01-08T12:26:45.110524116Z"}]}`
	level3Update := `{"channel":"level3","type":"update","data":[{"symbol":"BTC/USD","checksum":1,"bids":[{"event":"add","order_id":"OQ5WAX-SNVPL-R7VY2E","limit_price":44939.4,"order_qty":0.5,"timestamp":"2024-01-08T12:26:46Z"}],"asks":[],"timestamp":"2024-01-08T12:26:46Z"}]}`
	bookParams := `"channel":"book","symbol":["BTC/USD"],"depth":10`
	level3Params := `"channel":"level3","symbol":["BTC/USD"],"depth":10,"token":"` + wsTestToken + `"`
	for _, tc := range []struct {
		name     string
		channel  string
		filter   string
		messages []string
		err      error
		params   string
	}{
		{"orderbook snapshot", subscription.OrderbookChannel, wsPublicConnection, []string{book(wsTypeSnapshot, 1)}, errChecksumMismatch, bookParams},
		{"book snapshot", wsChannelBook, wsPublicConnection, []string{book(wsTypeSnapshot, 1)}, errChecksumMismatch, bookParams},
		{"orderbook update", subscription.OrderbookChannel, wsPublicConnection, []string{book(wsTypeSnapshot, valid), book("update", 1)}, errChecksumMismatch, bookParams},
		{"book update", wsChannelBook, wsPublicConnection, []string{book(wsTypeSnapshot, valid), book("update", 1)}, errChecksumMismatch, bookParams},
		{"allOrders snapshot", subscription.AllOrdersChannel, wsLevel3Connection, []string{level3Snapshot}, errChecksumMismatch, level3Params},
		{"level3 snapshot", wsChannelLevel3, wsLevel3Connection, []string{level3Snapshot}, errChecksumMismatch, level3Params},
		{"allOrders update before a snapshot", subscription.AllOrdersChannel, wsLevel3Connection, []string{level3Update}, errLevel3OutOfSync, level3Params},
		{"level3 update before a snapshot", wsChannelLevel3, wsLevel3Connection, []string{level3Update}, errLevel3OutOfSync, level3Params},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := new(mockSubscriptionServer)
			ex := newHTTPTestExchange(t, serveWebsocketToken(t))
			seedTestAssetNames(ex)
			useTestCredentials(ex)
			ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: tc.channel, Levels: 10}}
			useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), tc.filter)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
			subs := ex.Websocket.GetSubscriptions()
			require.Len(t, subs, 1, "Connect must make one subscription")
			server.takeRequests()
			conn, err := ex.Websocket.GetConnection(tc.filter)
			require.NoError(t, err, "GetConnection must not error")

			last := len(tc.messages) - 1
			for _, msg := range tc.messages[:last] {
				require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(msg)), "wsHandleData must not error")
			}
			require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(tc.messages[last])), tc.err, "wsHandleData must reject the book")
			exp := []string{`{"method":"unsubscribe","params":{` + tc.params + `}}`, `{"method":"subscribe","params":{` + tc.params + `}}`}
			var got []string
			require.Eventually(t, func() bool {
				got = append(got, server.takeRequests()...)
				return len(got) >= len(exp)
			}, testWaitTimeout, 10*time.Millisecond, "the book must be resubscribed")
			assert.Equal(t, exp, got, "the book should be unsubscribed before being subscribed again")
			require.Eventually(t, func() bool {
				return subs[0].State() == subscription.SubscribedState && ex.Websocket.GetSubscription(subs[0]) == subs[0]
			}, testWaitTimeout, 10*time.Millisecond, "the subscription must be subscribed again")
		})
	}
}

func TestBookChecksum(t *testing.T) {
	t.Parallel()
	// Kraken's documented example, whose numbers are strings
	msg := `{"symbol":"BTC/USD","bids":[{"price":"45283.5","qty":"0.10000000"},{"price":"45283.4","qty":"1.54582015"},{"price":"45282.1","qty":"0.10000000"},{"price":"45281.0","qty":"0.10000000"},{"price":"45280.3","qty":"1.54592586"},{"price":"45279.0","qty":"0.07990000"},{"price":"45277.6","qty":"0.03310103"},{"price":"45277.5","qty":"0.30000000"},{"price":"45277.3","qty":"1.54602737"},{"price":"45276.6","qty":"0.15445238"}],"asks":[{"price":"45285.2","qty":"0.00100000"},{"price":"45286.4","qty":"1.54571953"},{"price":"45286.6","qty":"1.54571109"},{"price":"45289.6","qty":"1.54560911"},{"price":"45290.2","qty":"0.15890660"},{"price":"45291.8","qty":"1.54553491"},{"price":"45294.7","qty":"0.04454749"},{"price":"45296.1","qty":"0.35380000"},{"price":"45297.5","qty":"0.09945542"},{"price":"45299.5","qty":"0.18772827"}],"checksum":3310070434}`
	var book WebsocketBook
	require.NoError(t, json.Unmarshal([]byte(msg), &book), "Unmarshal must not error")
	b := &orderbook.Book{Bids: wsBookLevels(book.Bids), Asks: wsBookLevels(book.Asks)}
	assert.Equal(t, uint32(3310070434), bookChecksum(b), "bookChecksum should match Kraken's documented example")
	assert.Equal(t, book.Checksum, bookChecksum(b), "bookChecksum should match the example's checksum")

	// Only the top 10 levels of each side are covered
	b.Bids = append(b.Bids, orderbook.Level{StrPrice: "45276.5", StrAmount: "1"})
	b.Asks = append(b.Asks, orderbook.Level{StrPrice: "45299.6", StrAmount: "1"})
	assert.Equal(t, uint32(3310070434), bookChecksum(b), "bookChecksum should ignore levels below the top 10")
}

func TestChecksumDigits(t *testing.T) {
	t.Parallel()
	for n, exp := range map[string]string{
		"45285.2":     "452852",
		"0.00100000":  "100000",
		"0.000005307": "5307",
		"81776":       "81776",
		"0.0":         "",
	} {
		assert.Equalf(t, exp, checksumDigits(n), "checksumDigits should remove the decimal point and leading zeros of %s", n)
	}
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]order.Type{
		"limit":               order.Limit,
		"iceberg":             order.Limit,
		"market":              order.Market,
		"settle-position":     order.Market,
		"stop-loss":           order.Stop,
		"stop-loss-limit":     order.StopLimit,
		"take-profit":         order.TakeProfit,
		"take-profit-limit":   order.TakeProfit | order.Limit,
		"trailing-stop":       order.TrailingStop,
		"trailing-stop-limit": order.TrailingStopLimit,
	} {
		got, err := orderTypeFromString(s)
		require.NoErrorf(t, err, "orderTypeFromString must not error for %s", s)
		assert.Equalf(t, exp, got, "orderTypeFromString should convert %s", s)
	}
	_, err := orderTypeFromString("moon")
	assert.ErrorIs(t, err, errUnknownOrderType, "orderTypeFromString should reject an unknown order type")
}

func TestOrderStatusFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]order.Status{
		"pending":          order.Pending,
		"pending_new":      order.Pending,
		"new":              order.New,
		"open":             order.Open,
		"partially_filled": order.PartiallyFilled,
		"filled":           order.Filled,
		"closed":           order.Filled,
		"canceled":         order.Cancelled,
		"expired":          order.Expired,
	} {
		got, err := orderStatusFromString(s)
		require.NoErrorf(t, err, "orderStatusFromString must not error for %s", s)
		assert.Equalf(t, exp, got, "orderStatusFromString should convert %s", s)
	}
	_, err := orderStatusFromString("moon")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "orderStatusFromString should reject an unknown order status")
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]order.TimeInForce{
		"GTC": order.GoodTillCancel,
		"gtd": order.GoodTillTime,
		"IOC": order.ImmediateOrCancel,
		"FOK": order.FillOrKill,
	} {
		got, err := timeInForceFromString(s)
		require.NoErrorf(t, err, "timeInForceFromString must not error for %s", s)
		assert.Equalf(t, exp, got, "timeInForceFromString should convert %s", s)
	}
	_, err := timeInForceFromString("GTX")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "timeInForceFromString should reject an unknown time in force")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		a       asset.Item
		exp     string
	}{
		{subscription.TickerChannel, asset.Spot, wsChannelTicker},
		{subscription.OrderbookChannel, asset.Spot, wsChannelBook},
		{subscription.CandlesChannel, asset.Spot, wsChannelOHLC},
		{subscription.AllTradesChannel, asset.Spot, wsChannelTrade},
		{subscription.AllOrdersChannel, asset.Spot, wsChannelLevel3},
		{subscription.MyOrdersChannel, asset.Empty, wsChannelExecutions},
		{subscription.MyTradesChannel, asset.Empty, wsChannelExecutions},
		{subscription.MyWalletChannel, asset.Empty, wsChannelBalances},
		{wsChannelInstrument, asset.Empty, wsChannelInstrument},
		{subscription.TickerChannel, asset.Futures, wsFuturesFeedTicker},
		{subscription.OrderbookChannel, asset.Futures, wsFuturesFeedBook},
		{subscription.AllTradesChannel, asset.Futures, wsFuturesFeedTrade},
		{subscription.MyOrdersChannel, asset.Futures, wsFuturesFeedOpenOrders},
		{subscription.MyTradesChannel, asset.Futures, wsFuturesFeedFills},
		{subscription.MyWalletChannel, asset.Futures, wsFuturesFeedBalances},
		{subscription.HeartbeatChannel, asset.Futures, wsFuturesFeedHeartbeat},
		// A feed without a standard channel is named by its feed
		{wsFuturesFeedOpenPositions, asset.Futures, wsFuturesFeedOpenPositions},
	} {
		assert.Equalf(t, tc.exp, channelName(&subscription.Subscription{Channel: tc.channel}, tc.a), "channelName should convert %s for %s", tc.channel, tc.a)
	}
}

func TestPerPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		a       asset.Item
		exp     bool
	}{
		{subscription.TickerChannel, asset.Spot, true},
		{subscription.AllOrdersChannel, asset.Spot, true},
		{subscription.TickerChannel, asset.Futures, true},
		{subscription.OrderbookChannel, asset.Futures, true},
		{subscription.AllTradesChannel, asset.Futures, true},
		{subscription.MyOrdersChannel, asset.Futures, false},
		{subscription.MyWalletChannel, asset.Futures, false},
		{subscription.HeartbeatChannel, asset.Futures, false},
		{wsFuturesFeedNotifications, asset.Futures, false},
	} {
		assert.Equalf(t, tc.exp, perPair(&subscription.Subscription{Channel: tc.channel}, tc.a), "perPair should report whether %s %s is subscribed to for each pair", tc.a, tc.channel)
	}
}

// level3Order returns a level 3 order, or an update's event for one when event is set
func level3Order(t *testing.T, event, orderID, price, qty string) WebsocketLevel3Order {
	t.Helper()
	p, err := types.NewPreciseNumberFromString(price)
	require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", price)
	q, err := types.NewPreciseNumberFromString(qty)
	require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", qty)
	return WebsocketLevel3Order{Event: event, OrderID: orderID, LimitPrice: p, Quantity: q}
}

func TestLevel3Checksum(t *testing.T) {
	t.Parallel()
	// Kraken's documented example, whose numbers are strings
	lines := fixtureLines(t, "testdata/wsLevel3.json")
	var snapshot struct {
		Data []WebsocketLevel3Book `json:"data"`
	}
	require.NoError(t, json.Unmarshal(lines[0], &snapshot), "Unmarshal must not error")
	book := &level3Book{bids: snapshot.Data[0].Bids, asks: snapshot.Data[0].Asks}
	assert.Equal(t, uint32(1063832831), book.checksum(), "checksum should match Kraken's documented example")

	// Only the orders of the top 10 price levels of each side are covered
	book.bids = append(book.bids, level3Order(t, "", "ONEXTB-AAAAA-BBBBBB", "44900.0", "1.00000000"))
	book.asks = append(book.asks, level3Order(t, "", "ONEXTA-AAAAA-BBBBBB", "44990.0", "1.00000000"))
	assert.Equal(t, uint32(1063832831), book.checksum(), "checksum should ignore price levels below the top 10")
}

func TestApplyLevel3Events(t *testing.T) {
	t.Parallel()
	bid := func(id, price, qty string) WebsocketLevel3Order { return level3Order(t, "", id, price, qty) }
	held := []WebsocketLevel3Order{bid("OA", "100.0", "1.0"), bid("OB", "100.0", "2.0"), bid("OC", "99.5", "3.0")}
	better := func(price, other float64) bool { return price > other }
	for _, tc := range []struct {
		name   string
		events []WebsocketLevel3Order
		exp    []WebsocketLevel3Order
	}{
		{
			name:   "add queues behind the orders at its price",
			events: []WebsocketLevel3Order{level3Order(t, "add", "OD", "100.0", "4.0")},
			exp:    []WebsocketLevel3Order{bid("OA", "100.0", "1.0"), bid("OB", "100.0", "2.0"), bid("OD", "100.0", "4.0"), bid("OC", "99.5", "3.0")},
		},
		{
			name:   "add at a better price",
			events: []WebsocketLevel3Order{level3Order(t, "add", "OD", "100.5", "4.0")},
			exp:    []WebsocketLevel3Order{bid("OD", "100.5", "4.0"), bid("OA", "100.0", "1.0"), bid("OB", "100.0", "2.0"), bid("OC", "99.5", "3.0")},
		},
		{
			name:   "add at the worst price",
			events: []WebsocketLevel3Order{level3Order(t, "add", "OD", "99.0", "4.0")},
			exp:    []WebsocketLevel3Order{bid("OA", "100.0", "1.0"), bid("OB", "100.0", "2.0"), bid("OC", "99.5", "3.0"), bid("OD", "99.0", "4.0")},
		},
		{
			name:   "modify keeps the order's place",
			events: []WebsocketLevel3Order{level3Order(t, "modify", "OA", "100.0", "0.5")},
			exp:    []WebsocketLevel3Order{bid("OA", "100.0", "0.5"), bid("OB", "100.0", "2.0"), bid("OC", "99.5", "3.0")},
		},
		{
			name:   "modify to another price requeues the order",
			events: []WebsocketLevel3Order{level3Order(t, "modify", "OA", "99.5", "1.0")},
			exp:    []WebsocketLevel3Order{bid("OB", "100.0", "2.0"), bid("OC", "99.5", "3.0"), bid("OA", "99.5", "1.0")},
		},
		{
			name:   "delete",
			events: []WebsocketLevel3Order{level3Order(t, "delete", "OB", "100.0", "2.0")},
			exp:    []WebsocketLevel3Order{bid("OA", "100.0", "1.0"), bid("OC", "99.5", "3.0")},
		},
	} {
		got, err := applyLevel3Events(slices.Clone(held), tc.events, better)
		require.NoErrorf(t, err, "applyLevel3Events must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "applyLevel3Events should apply %s", tc.name)
	}
	for _, tc := range []struct {
		name  string
		event WebsocketLevel3Order
	}{
		{"an order added twice", level3Order(t, "add", "OA", "100.0", "1.0")},
		{"an unknown order modified", level3Order(t, "modify", "OZ", "100.0", "1.0")},
		{"an unknown order deleted", level3Order(t, "delete", "OZ", "100.0", "1.0")},
		{"an unknown event", level3Order(t, "moon", "OA", "100.0", "1.0")},
	} {
		_, err := applyLevel3Events(slices.Clone(held), []WebsocketLevel3Order{tc.event}, better)
		assert.ErrorIsf(t, err, errLevel3OutOfSync, "applyLevel3Events should reject %s", tc.name)
	}
}

func TestTruncateLevel3(t *testing.T) {
	t.Parallel()
	ask := func(id, price string) WebsocketLevel3Order { return level3Order(t, "", id, price, "1.0") }
	asks := []WebsocketLevel3Order{ask("OA", "100.0"), ask("OB", "100.0"), ask("OC", "100.5"), ask("OD", "101.0"), ask("OE", "101.0")}
	assert.Equal(t, asks, truncateLevel3(slices.Clone(asks), 0), "truncateLevel3 should keep every order without a depth")
	assert.Equal(t, asks, truncateLevel3(slices.Clone(asks), 3), "truncateLevel3 should keep every order within the depth")
	assert.Equal(t, asks[:3], truncateLevel3(slices.Clone(asks), 2), "truncateLevel3 should keep the orders of the first price levels")
	assert.Equal(t, asks[:2], truncateLevel3(slices.Clone(asks), 1), "truncateLevel3 should keep every order of a level")
}

// TestWebsocketProcessLevel3 replays Kraken's documented level 3 order book example and updates whose checksums an
// independent implementation of Kraken's documented algorithm gave
func TestWebsocketProcessLevel3(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.setBookDepth(wsTestPair, 10)
	testexch.FixtureToDataHandler(t, "testdata/wsLevel3.json", func(ctx context.Context, b []byte) error {
		var msg wsChannelMessage
		if err := json.Unmarshal(b, &msg); err != nil {
			return err
		}
		return ex.wsProcessLevel3(ctx, nil, &msg)
	})
	got := relayed(ex)
	require.Len(t, got, 8, "wsProcessLevel3 must relay the order book and the level 3 message for each message")
	for i := 0; i < len(got); i += 2 {
		assert.IsTypef(t, &orderbook.Depth{}, got[i], "payload %d should be the order book", i)
		l3, ok := got[i+1].(*WebsocketLevel3Book)
		require.Truef(t, ok, "payload %d must be the level 3 message", i+1)
		assert.Equalf(t, "BTC/USD", l3.Symbol, "payload %d should be the level 3 message", i+1)
	}
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	// Each level is an order, the orders of a price queued in time order
	assert.Equal(t, orderbook.Levels{
		bookLevel(44945.0, "44945.0", 0.12, "0.12000000"),
		bookLevel(44939.4, "44939.4", 0.88968699, "0.88968699"),
		bookLevel(44939.4, "44939.4", 0.4521, "0.45210000"),
		bookLevel(44939.4, "44939.4", 0.1, "0.10000000"),
		bookLevel(44939.4, "44939.4", 0.14296323, "0.14296323"),
		bookLevel(44939.4, "44939.4", 0.25, "0.25000000"),
		bookLevel(44939.4, "44939.4", 0.10292988, "0.10292988"),
		bookLevel(44939.4, "44939.4", 0.3388, "0.33880000"),
		bookLevel(44939.4, "44939.4", 1.2814086, "1.28140860"),
		bookLevel(44939.4, "44939.4", 0.05, "0.05000000"),
		bookLevel(44937.1, "44937.1", 0.03346877, "0.03346877"),
		bookLevel(44934.7, "44934.7", 0.3563, "0.35630000"),
		bookLevel(44930.2, "44930.2", 0.22734299, "0.22734299"),
		bookLevel(44930.2, "44930.2", 0.01, "0.01000000"),
		bookLevel(44930.2, "44930.2", 0.7, "0.70000000"),
		bookLevel(44930.2, "44930.2", 0.15, "0.15000000"),
		bookLevel(44928.0, "44928.0", 0.0010524, "0.00105240"),
		bookLevel(44919.6, "44919.6", 0.3387, "0.33870000"),
		bookLevel(44919.5, "44919.5", 0.0761, "0.07610000"),
		bookLevel(44912.0, "44912.0", 0.3563, "0.35630000"),
		bookLevel(44909.7, "44909.7", 0.0669, "0.06690000"),
	}, book.Bids, "the bids should have the updates applied and be truncated to the subscribed depth")
	assert.Equal(t, orderbook.Levels{
		bookLevel(44950.0, "44950.0", 0.10334926, "0.10334926"),
		bookLevel(44953.0, "44953.0", 0.00064537, "0.00064537"),
		bookLevel(44955.0, "44955.0", 0.0025, "0.00250000"),
		bookLevel(44959.6, "44959.6", 0.3563, "0.35630000"),
		bookLevel(44959.6, "44959.6", 0.3563, "0.35630000"),
		bookLevel(44960.1, "44960.1", 0.00338072, "0.00338072"),
		bookLevel(44960.2, "44960.2", 0.88967575, "0.88967575"),
		bookLevel(44967.0, "44967.0", 3.14392283, "3.14392283"),
		bookLevel(44978.5, "44978.5", 0.0677896, "0.06778960"),
		bookLevel(44979.2, "44979.2", 0.3563, "0.35630000"),
		bookLevel(44980.0, "44980.0", 0.25, "0.25000000"),
		bookLevel(44980.0, "44980.0", 1, "1.00000000"),
	}, book.Asks, "the asks should have the updates applied, the level below the depth added as the best level was taken")
	assert.Equal(t, time.Date(2024, 1, 8, 12, 26, 47, 118034770, time.UTC), book.LastUpdated, "the book should be as of the last update")
}

func TestWebsocketProcessLevel3Errors(t *testing.T) {
	t.Parallel()
	snapshot := func(checksum string) []byte {
		return []byte(`{"channel":"level3","type":"snapshot","data":[{"symbol":"BTC/USD","checksum":` + checksum + `,"bids":[{"order_id":"OTCFZG-YOE2Q-LQKNM3","limit_price":44939.4,"order_qty":0.88968699,"timestamp":"2024-01-08T12:26:39.526146327Z"}],"asks":[{"order_id":"OFVLAA-HRSSP-BK75KB","limit_price":44939.5,"order_qty":4.52308393,"timestamp":"2024-01-08T12:18:05.770906486Z"}],"timestamp":"2024-01-08T12:26:45.110524116Z"}]}`)
	}
	update := func(event, orderID string) []byte {
		return []byte(`{"channel":"level3","type":"update","data":[{"symbol":"BTC/USD","checksum":1,"bids":[{"event":"` + event + `","order_id":"` + orderID + `","limit_price":44939.4,"order_qty":0.5,"timestamp":"2024-01-08T12:26:46Z"}],"asks":[],"timestamp":"2024-01-08T12:26:46Z"}]}`)
	}
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, update("add", "OQ5WAX-SNVPL-R7VY2E")), errLevel3OutOfSync, "an update before a snapshot must error")

	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, snapshot("1")), errChecksumMismatch, "a snapshot failing its checksum must error")
	_, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "a snapshot failing its checksum should not load the book")

	book := &level3Book{bids: []WebsocketLevel3Order{level3Order(t, "", "OTCFZG-YOE2Q-LQKNM3", "44939.4", "0.88968699")}, asks: []WebsocketLevel3Order{level3Order(t, "", "OFVLAA-HRSSP-BK75KB", "44939.5", "4.52308393")}}
	require.NoError(t, ex.wsHandleData(t.Context(), nil, snapshot(strconv.FormatUint(uint64(book.checksum()), 10))), "wsHandleData must load a snapshot passing its checksum")
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, update("modify", "OTCFZG-YOE2Q-LQKNM3")), errChecksumMismatch, "an update failing its checksum must error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update failing its checksum should invalidate the book")
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, update("modify", "OTCFZG-YOE2Q-LQKNM3")), errLevel3OutOfSync, "an update after a failed update must error until a new snapshot")

	require.NoError(t, ex.wsHandleData(t.Context(), nil, snapshot(strconv.FormatUint(uint64(book.checksum()), 10))), "wsHandleData must load a new snapshot")
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, update("delete", "OZZZZZ-ZZZZZ-ZZZZZZ")), errLevel3OutOfSync, "an update naming an order the book does not hold must error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update naming an order the book does not hold should invalidate the book")

	for _, msg := range []string{`{"channel":"level3","data":"bad"}`, `{"channel":"level3","type":"snapshot","data":[{"symbol":"BTC"}]}`} {
		assert.Errorf(t, ex.wsHandleData(t.Context(), nil, []byte(msg)), "wsHandleData should reject %s", msg)
	}
}

func TestSubscribeLevel3(t *testing.T) {
	t.Parallel()
	server := new(mockSubscriptionServer)
	ex := newHTTPTestExchange(t, serveWebsocketToken(t))
	seedTestAssetNames(ex)
	useTestCredentials(ex)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.AllOrdersChannel, Levels: 100}}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsLevel3Connection)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
	s := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.AllOrdersChannel, QualifiedChannel: wsChannelLevel3, Levels: 100, Pairs: currency.Pairs{wsTestPair}}
	// EqualLists rekeys the subscriptions it compares, which would orphan the stored ones from their store entries
	testsubs.EqualLists(t, subscription.List{s}, ex.Websocket.GetSubscriptions().Clone())
	// The level3 channel takes a token, which a new connection's subscription refreshes
	assertRequests(t, []string{`{"method":"subscribe","params":{"channel":"level3","symbol":["BTC/USD"],"depth":100,"token":"` + wsTestToken + `"}}`}, server.takeRequests())
	depth, ok := ex.bookDepth(wsTestPair)
	require.True(t, ok, "subscribing must record the level 3 order book depth")
	assert.Equal(t, 100, depth, "the recorded depth should be the subscribed one")

	conn, err := ex.Websocket.GetConnection(wsLevel3Connection)
	require.NoError(t, err, "GetConnection must not error")
	// A snapshot failing its checksum resubscribes the book for a new one
	badSnapshot := `{"channel":"level3","type":"snapshot","data":[{"symbol":"BTC/USD","checksum":1,"bids":[],"asks":[],"timestamp":"2024-01-08T12:26:45.110524116Z"}]}`
	require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(badSnapshot)), errChecksumMismatch, "a snapshot failing its checksum must error")
	exp := []string{
		`{"method":"unsubscribe","params":{"channel":"level3","symbol":["BTC/USD"],"depth":100,"token":"` + wsTestToken + `"}}`,
		`{"method":"subscribe","params":{"channel":"level3","symbol":["BTC/USD"],"depth":100,"token":"` + wsTestToken + `"}}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "a level 3 snapshot failing its checksum must resubscribe the book")
	assertRequests(t, exp, got)
	require.Eventually(t, func() bool {
		stored := ex.Websocket.GetSubscription(s)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, testWaitTimeout, 10*time.Millisecond, "the level 3 book must be subscribed again")

	ex.level3BooksMu.Lock()
	ex.level3Books = map[key.PairAsset]*level3Book{{Base: wsTestPair.Base.Item, Quote: wsTestPair.Quote.Item, Asset: asset.Spot}: {}}
	ex.level3BooksMu.Unlock()
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, ex.Websocket.GetSubscriptions()), "UnsubscribeChannels must not error")
	assertRequests(t, []string{`{"method":"unsubscribe","params":{"channel":"level3","symbol":["BTC/USD"],"depth":100,"token":"` + wsTestToken + `"}}`}, server.takeRequests())
	_, ok = ex.bookDepth(wsTestPair)
	assert.False(t, ok, "unsubscribing should forget the depth")
	ex.level3BooksMu.Lock()
	assert.Empty(t, ex.level3Books, "unsubscribing should forget the level 3 order book")
	ex.level3BooksMu.Unlock()
}

func TestGenerateLevel3Subscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// The default order book subscription is XBT/USD's level 2 book, which the level 3 book cannot share
	ex.Features.Subscriptions = append(defaultSubscriptions.Clone(), &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.AllOrdersChannel, Levels: 10, Pairs: currency.Pairs{wsTestDOGEPair}})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	got, err := ex.generateLevel3Subscriptions()
	require.NoError(t, err, "generateLevel3Subscriptions must not error")
	assert.Empty(t, got, "generateLevel3Subscriptions should return nothing without authenticated websocket use")

	useTestCredentials(ex)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	got, err = ex.generateLevel3Subscriptions()
	require.NoError(t, err, "generateLevel3Subscriptions must not error")
	level3 := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.AllOrdersChannel, QualifiedChannel: wsChannelLevel3, Levels: 10, Pairs: currency.Pairs{wsTestDOGEPair}}
	testsubs.EqualLists(t, subscription.List{level3}, got)
	for name, generate := range map[string]func() (subscription.List, error){
		"generatePublicSubscriptions":  ex.generatePublicSubscriptions,
		"generatePrivateSubscriptions": ex.generatePrivateSubscriptions,
	} {
		subs, err := generate()
		require.NoErrorf(t, err, "%s must not error", name)
		assert.Falsef(t, slices.ContainsFunc(subs, func(s *subscription.Subscription) bool {
			return s.QualifiedChannel == wsChannelLevel3 || s.Asset == asset.Futures
		}), "%s should leave out the level 3 and futures subscriptions", name)
	}
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	// Unseeded assets leave only the pairs whose display names differ untranslated, so connecting goes on without them
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"error":["EService:Unavailable"]}`))
		assert.NoError(t, err, "Write should not error")
	})
	useTestWebsocket(t, ex, newMockWebsocketURL(t, func(testing.TB, []byte, *gws.Conn) error { return nil }), wsPublicConnection)
	conn, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must not error when the assets cannot be seeded")
	t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
	assert.False(t, ex.assetNames.seeded(), "wsConnect should leave the assets unseeded when seeding fails")

	useTestWebsocket(t, ex, "ws://127.0.0.1:1", wsPublicConnection)
	unreachable, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	assert.Error(t, ex.wsConnect(t.Context(), unreachable), "wsConnect should return a failed dial's error")
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tmpl, err := e.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	require.NotNil(t, tmpl, "GetSubscriptionTemplate must return a template")
	assert.Equal(t, "master.tmpl", tmpl.Name(), "GetSubscriptionTemplate should return the master template")
}

func TestGenerateSubscriptionsExpansion(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{wsTestPair}}}, subs)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 7}}
	_, err = ex.generateSubscriptions()
	assert.ErrorIs(t, err, errInvalidDepth, "generateSubscriptions should validate the subscriptions it expands")
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	server := &mockSubscriptionServer{rejected: map[string]string{"DOGE/USD": "Currency pair not supported DOGE/USD"}}
	ex, conn := newSubscriptionTestExchange(t, server)
	tickerBTC := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	tickerDOGE := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestDOGEPair}}
	trades := &subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: wsChannelTrade, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	err := ex.manageSubs(t.Context(), conn, wsMethodSubscribe, subscription.List{tickerBTC, tickerDOGE, trades})
	assert.ErrorIs(t, err, errAPIResponse, "manageSubs should report a subscription Kraken rejects")
	assertRequests(t, []string{
		`{"method":"subscribe","params":{"channel":"ticker","symbol":["BTC/USD","DOGE/USD"]}}`,
		`{"method":"subscribe","params":{"channel":"trade","symbol":["BTC/USD"]}}`,
	}, server.takeRequests())
	assert.Equal(t, subscription.SubscribedState, tickerBTC.State(), "manageSubs should record a subscription Kraken accepts")
	assert.Equal(t, subscription.SubscribedState, trades.State(), "manageSubs should record each group's subscriptions")
	assert.Nil(t, ex.Websocket.GetSubscription(tickerDOGE), "manageSubs should not record a subscription Kraken rejects")
}

func TestManageSubGroup(t *testing.T) {
	t.Parallel()
	server := new(mockSubscriptionServer)
	ex := newHTTPTestExchange(t, serveWebsocketToken(t))
	seedTestAssetNames(ex)
	useTestCredentials(ex)
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsPublicConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	ethUSD := currency.NewPairWithDelimiter("ETH", "USD", "_")
	books := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 25, Pairs: currency.Pairs{wsTestPair, wsTestDOGEPair}}
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  string
	}{
		{
			name: "candles",
			sub:  &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Asset: asset.Spot, Interval: kline.FiveMin, Pairs: currency.Pairs{wsTestPair}},
			exp:  `{"method":"subscribe","params":{"channel":"ohlc","symbol":["BTC/USD"],"interval":5}}`,
		},
		{name: "order books", sub: books, exp: `{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD","DOGE/USD"],"depth":25}}`},
		{
			name: "level 3 order books",
			sub:  &subscription.Subscription{Channel: subscription.AllOrdersChannel, QualifiedChannel: wsChannelLevel3, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{ethUSD}},
			exp:  `{"method":"subscribe","params":{"channel":"level3","symbol":["ETH/USD"],"depth":10,"token":"` + wsTestToken + `"}}`,
		},
		{
			name: "executions",
			sub:  &subscription.Subscription{Channel: subscription.MyOrdersChannel, QualifiedChannel: wsChannelExecutions, Authenticated: true, Params: map[string]any{"snap_orders": true, "snap_trades": false}},
			exp:  `{"method":"subscribe","params":{"channel":"executions","snap_orders":true,"snap_trades":false,"token":"` + wsTestToken + `"}}`,
		},
	} {
		require.NoErrorf(t, ex.manageSubGroup(t.Context(), conn, wsMethodSubscribe, subscription.List{tc.sub}), "manageSubGroup must not error subscribing to %s", tc.name)
		assertRequests(t, []string{tc.exp}, server.takeRequests())
		assert.Equalf(t, subscription.SubscribedState, tc.sub.State(), "manageSubGroup should record the %s subscription", tc.name)
	}
	for _, p := range books.Pairs {
		depth, ok := ex.bookDepth(p)
		assert.Truef(t, ok, "manageSubGroup should record %s's order book depth", p)
		assert.Equalf(t, 25, depth, "manageSubGroup should record %s's subscribed depth", p)
	}
	depth, recorded := ex.bookDepth(ethUSD)
	assert.True(t, recorded, "manageSubGroup should record a level 3 order book's depth")
	assert.Equal(t, 10, depth, "manageSubGroup should record a level 3 order book's subscribed depth")
	require.NoError(t, ex.manageSubGroup(t.Context(), conn, wsMethodUnsubscribe, subscription.List{books}), "manageSubGroup must not error unsubscribing")
	assertRequests(t, []string{`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD","DOGE/USD"],"depth":25}}`}, server.takeRequests())
	for _, p := range books.Pairs {
		_, ok := ex.bookDepth(p)
		assert.Falsef(t, ok, "manageSubGroup should forget %s's order book depth on unsubscribing", p)
	}
	assert.Nil(t, ex.Websocket.GetSubscription(books), "manageSubGroup should remove an unsubscribed subscription")

	refused := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/GetWebSocketsToken", r.URL.Path, "only a websocket token should be requested")
		_, _ = w.Write([]byte(`{"error":["EGeneral:Permission denied"]}`))
	})
	executions := &subscription.Subscription{Channel: subscription.MyOrdersChannel, QualifiedChannel: wsChannelExecutions, Authenticated: true}
	err = refused.manageSubGroup(t.Context(), conn, wsMethodSubscribe, subscription.List{executions})
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "manageSubGroup should fail a subscription whose token Kraken refuses")
	assert.Empty(t, server.takeRequests(), "manageSubGroup should send no subscription without a token")

	unconnected, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	unsent := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{solUSDPair}}
	assert.Error(t, ex.manageSubGroup(t.Context(), unconnected, wsMethodSubscribe, subscription.List{unsent}), "manageSubGroup should return a failed send's error")
	_, recorded = ex.bookDepth(solUSDPair)
	assert.False(t, recorded, "manageSubGroup should forget the order book depth of a subscription it could not send")

	// A reply that cannot be decoded fails the whole request
	for _, reply := range []string{`{"method":"subscribe","req_id":1,"success":"yes"}`, `{"method":"subscribe","req_id":1,"result":"bad","success":true}`} {
		undecoded, undecodedConn := newSubscriptionTestExchange(t, &mockSubscriptionServer{replies: []string{reply}})
		book := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
		assert.Errorf(t, undecoded.manageSubGroup(t.Context(), undecodedConn, wsMethodSubscribe, subscription.List{book}), "manageSubGroup should reject the reply %s", reply)
		assert.Nilf(t, undecoded.Websocket.GetSubscription(book), "manageSubGroup should not record a subscription whose reply %s it cannot decode", reply)
		_, ok := undecoded.bookDepth(wsTestPair)
		assert.Falsef(t, ok, "manageSubGroup should forget the order book depth of a subscription whose reply %s it cannot decode", reply)
	}

	unexpected := &mockSubscriptionServer{unexpected: true}
	ex, conn = newSubscriptionTestExchange(t, unexpected)
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}}
	err = ex.manageSubGroup(t.Context(), conn, wsMethodSubscribe, subscription.List{tickerSub})
	assert.ErrorIs(t, err, errUnexpectedSubscriptionReply, "manageSubGroup should reject a reply for a symbol it did not request")

	// Kraken's replies to a request for the 5 minute candles of BTC/USD, whose 1 minute candles the connection streamed,
	// and of ETH/USD: it streams one candle interval for a symbol on a connection, rejecting a second without naming the
	// symbol
	const (
		secondInterval    = `{"error":"Already subscribed to one ohlc interval on this symbol","method":"subscribe","req_id":2,"success":false,"time_in":"2026-10-19T07:50:24.425511Z","time_out":"2026-10-19T07:50:24.425560Z"}`
		ethAccepted       = `{"method":"subscribe","req_id":2,"result":{"channel":"ohlc","interval":5,"snapshot":false,"symbol":"ETH/USD","warnings":["timestamp is deprecated, use interval_begin"]},"success":true,"time_in":"2026-10-19T07:50:24.425511Z","time_out":"2026-10-19T07:50:24.425560Z"}`
		ethRejected       = `{"error":"Currency pair not supported ETH/USD","method":"subscribe","req_id":2,"success":false,"symbol":"ETH/USD","time_in":"2026-10-19T07:48:25.460238Z","time_out":"2026-10-19T07:48:25.460304Z"}`
		alreadySubscribed = "Already subscribed to one ohlc interval on this symbol"
	)
	for _, tc := range []struct {
		name    string
		replies []string
		err     error
		// reasons are the errors Kraken gave that the subscriptions no reply names fail with
		reasons []string
		// named is the reason of a rejection naming ETH/USD, which its subscription fails with
		named string
		// accepted is set when ETH/USD's subscription is recorded
		accepted bool
	}{
		{name: "a rejection naming no symbol", replies: []string{secondInterval, ethAccepted}, err: errAPIResponse, reasons: []string{alreadySubscribed}, accepted: true},
		{name: "a rejection naming a symbol and one naming none", replies: []string{ethRejected, secondInterval}, err: errAPIResponse, reasons: []string{alreadySubscribed}, named: "Currency pair not supported ETH/USD"},
		{name: "repeated rejections naming no symbol", replies: []string{secondInterval, secondInterval}, err: errAPIResponse, reasons: []string{alreadySubscribed}},
		{name: "distinct rejections naming no symbol", replies: []string{secondInterval, strings.Replace(secondInterval, alreadySubscribed, "Internal error", 1)}, err: errAPIResponse, reasons: []string{alreadySubscribed, "Internal error"}},
		{name: "a success naming no symbol", replies: []string{ethAccepted, strings.Replace(ethAccepted, `"symbol":"ETH/USD",`, "", 1)}, err: errUnexpectedSubscriptionReply},
		{name: "a reply naming a symbol not requested", replies: []string{secondInterval, strings.Replace(ethAccepted, "ETH/USD", "MOON/USD", 1)}, err: errUnexpectedSubscriptionReply},
	} {
		server := &mockSubscriptionServer{replies: tc.replies}
		ex, conn := newSubscriptionTestExchange(t, server)
		// Verbose logs the warnings of a subscription Kraken accepts
		ex.Verbose = true
		btc := &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Asset: asset.Spot, Interval: kline.FiveMin, Pairs: currency.Pairs{wsTestPair}}
		eth := &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Asset: asset.Spot, Interval: kline.FiveMin, Pairs: currency.Pairs{ethUSD}}
		err := ex.manageSubGroup(t.Context(), conn, wsMethodSubscribe, subscription.List{btc, eth})
		require.ErrorIsf(t, err, tc.err, "manageSubGroup must fail a request answered with %s", tc.name)
		assertRequests(t, []string{`{"method":"subscribe","params":{"channel":"ohlc","symbol":["BTC/USD","ETH/USD"],"interval":5}}`}, server.takeRequests())
		if tc.reasons != nil {
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "manageSubGroup must return Kraken's error for a request answered with %s", tc.name)
			assert.Equalf(t, tc.reasons, apiErr.Errors, "the subscriptions no reply names should fail with each reason Kraken gave in %s", tc.name)
		}
		if tc.named != "" {
			assert.ErrorContainsf(t, err, tc.named, "the subscription a rejection names should fail with its reason in %s", tc.name)
		}
		assert.Nilf(t, ex.Websocket.GetSubscription(btc), "a subscription no reply accepts should not be recorded after %s", tc.name)
		if tc.accepted {
			assert.Equalf(t, subscription.SubscribedState, eth.State(), "the subscription Kraken accepted alongside %s should be recorded", tc.name)
			assert.NotNilf(t, ex.Websocket.GetSubscription(eth), "the subscription Kraken accepted alongside %s should be stored", tc.name)
			continue
		}
		assert.Nilf(t, ex.Websocket.GetSubscription(eth), "no subscription should be recorded from a request answered with %s", tc.name)
	}

	// A subscription holding several pairs subscribes only when Kraken accepts every one of them
	btcAccepted := strings.Replace(ethAccepted, "ETH/USD", "BTC/USD", 1)
	for _, tc := range []struct {
		name    string
		replies []string
		err     error
		reasons []string
	}{
		{name: "both pairs accepted", replies: []string{btcAccepted, ethAccepted}},
		{name: "a rejection naming no symbol", replies: []string{secondInterval, ethAccepted}, err: errAPIResponse, reasons: []string{alreadySubscribed}},
		{name: "a rejection naming one of its symbols", replies: []string{ethRejected, btcAccepted}, err: errAPIResponse, reasons: []string{"Currency pair not supported ETH/USD"}},
		{name: "one symbol answered twice and the other not at all", replies: []string{btcAccepted, btcAccepted}, err: errSubscriptionResponseMissing},
	} {
		server := &mockSubscriptionServer{replies: tc.replies}
		multi, multiConn := newSubscriptionTestExchange(t, server)
		both := &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: wsChannelOHLC, Asset: asset.Spot, Interval: kline.FiveMin, Pairs: currency.Pairs{wsTestPair, ethUSD}}
		err := multi.manageSubGroup(t.Context(), multiConn, wsMethodSubscribe, subscription.List{both})
		assertRequests(t, []string{`{"method":"subscribe","params":{"channel":"ohlc","symbol":["BTC/USD","ETH/USD"],"interval":5}}`}, server.takeRequests())
		if tc.err == nil {
			require.NoErrorf(t, err, "manageSubGroup must not error for a subscription holding two pairs answered with %s", tc.name)
			assert.Equalf(t, subscription.SubscribedState, both.State(), "a subscription holding two pairs answered with %s should be recorded", tc.name)
			continue
		}
		require.ErrorIsf(t, err, tc.err, "manageSubGroup must fail a subscription holding two pairs answered with %s", tc.name)
		if tc.reasons != nil {
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "manageSubGroup must return Kraken's error for a subscription holding two pairs answered with %s", tc.name)
			assert.Equalf(t, tc.reasons, apiErr.Errors, "a subscription holding two pairs should fail with each reason Kraken gave in %s", tc.name)
		}
		assert.Nilf(t, multi.Websocket.GetSubscription(both), "a subscription holding a pair Kraken did not accept should not be recorded after %s", tc.name)
	}
}

func TestSubscriptionRepliesErr(t *testing.T) {
	t.Parallel()
	const (
		alreadySubscribed = "Already subscribed to one ohlc interval on this symbol"
		ethNotSupported   = "Currency pair not supported ETH/USD"
		internal          = "Internal error"
	)
	for _, tc := range []struct {
		name    string
		replies subscriptionReplies
		symbols []string
		// reasons are the reasons the subscription fails with, and missing is set when it fails as unanswered
		reasons []string
		missing bool
	}{
		{name: "every symbol accepted", replies: subscriptionReplies{accepted: map[string]bool{"BTC/USD": true, "ETH/USD": true}}, symbols: []string{"BTC/USD", "ETH/USD"}},
		{name: "a symbol rejected", replies: subscriptionReplies{accepted: map[string]bool{"BTC/USD": true}, rejected: map[string][]string{"ETH/USD": {ethNotSupported}}}, symbols: []string{"BTC/USD", "ETH/USD"}, reasons: []string{ethNotSupported}},
		{name: "a symbol both accepted and rejected", replies: subscriptionReplies{accepted: map[string]bool{"ETH/USD": true}, rejected: map[string][]string{"ETH/USD": {internal}}}, symbols: []string{"ETH/USD"}, reasons: []string{internal}},
		{name: "symbols rejected for the same and distinct reasons", replies: subscriptionReplies{rejected: map[string][]string{"BTC/USD": {internal}, "ETH/USD": {internal, ethNotSupported}}}, symbols: []string{"BTC/USD", "ETH/USD"}, reasons: []string{internal, ethNotSupported}},
		{name: "a symbol no reply names, with rejections naming none", replies: subscriptionReplies{accepted: map[string]bool{"BTC/USD": true}, unnamed: []string{alreadySubscribed}}, symbols: []string{"BTC/USD", "ETH/USD"}, reasons: []string{alreadySubscribed}},
		{name: "a symbol no reply names, without rejections naming none", replies: subscriptionReplies{accepted: map[string]bool{"BTC/USD": true}}, symbols: []string{"BTC/USD", "ETH/USD"}, missing: true},
		{name: "a symbol rejected and another no reply names, with rejections naming none", replies: subscriptionReplies{rejected: map[string][]string{"BTC/USD": {internal}}, unnamed: []string{alreadySubscribed, internal}}, symbols: []string{"BTC/USD", "ETH/USD"}, reasons: []string{internal, alreadySubscribed}},
		{name: "a symbol rejected and another no reply names, without rejections naming none", replies: subscriptionReplies{rejected: map[string][]string{"BTC/USD": {internal}}}, symbols: []string{"BTC/USD", "ETH/USD"}, reasons: []string{internal}, missing: true},
		{name: "rejections naming no symbol, with every symbol accepted", replies: subscriptionReplies{accepted: map[string]bool{"BTC/USD": true}, unnamed: []string{alreadySubscribed}}, symbols: []string{"BTC/USD"}},
	} {
		err := tc.replies.err(tc.symbols)
		if tc.reasons == nil && !tc.missing {
			assert.NoErrorf(t, err, "err should return nothing for %s", tc.name)
			continue
		}
		if tc.reasons != nil {
			var apiErr *APIError
			if assert.ErrorAsf(t, err, &apiErr, "err should return Kraken's reasons for %s", tc.name) {
				assert.Equalf(t, tc.reasons, apiErr.Errors, "err should return each reason once for %s", tc.name)
			}
		} else {
			assert.NotErrorIsf(t, err, errAPIResponse, "err should return no reason for %s", tc.name)
		}
		if tc.missing {
			assert.ErrorIsf(t, err, errSubscriptionResponseMissing, "err should report the missing reply for %s", tc.name)
		} else {
			assert.NotErrorIsf(t, err, errSubscriptionResponseMissing, "err should not report a missing reply for %s", tc.name)
		}
	}
}

func TestAppendReasons(t *testing.T) {
	t.Parallel()
	const alreadySubscribed = "Already subscribed to one ohlc interval on this symbol"
	assert.Equal(t, []string{"Internal error", alreadySubscribed}, appendReasons([]string{"Internal error"}, alreadySubscribed, "Internal error", alreadySubscribed), "appendReasons should append each reason not held, once")
	assert.Nil(t, appendReasons(nil), "appendReasons should return no reasons without any")
}

func TestFailSubGroup(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	book := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	ex.setBookDepth(wsTestPair, 10)
	assert.NoError(t, ex.failSubGroup(wsMethodUnsubscribe, subscription.List{book}, nil), "failSubGroup should return no error without one")
	_, ok := ex.bookDepth(wsTestPair)
	assert.True(t, ok, "failSubGroup should keep the depth of a failed unsubscription")
	err := ex.failSubGroup(wsMethodSubscribe, subscription.List{book}, errSubscriptionResponseMissing)
	assert.ErrorIs(t, err, errSubscriptionResponseMissing, "failSubGroup should return the failure")
	_, ok = ex.bookDepth(wsTestPair)
	assert.False(t, ok, "failSubGroup should forget the depth a failed subscription recorded")
}

func TestClearBookDepths(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.setBookDepth(wsTestPair, 10)
	ex.clearBookDepths(&subscription.Subscription{QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{wsTestPair}})
	_, ok := ex.bookDepth(wsTestPair)
	assert.True(t, ok, "clearBookDepths should keep the depth for a channel without a book")
	ex.clearBookDepths(&subscription.Subscription{QualifiedChannel: wsChannelBook, Pairs: currency.Pairs{wsTestPair}})
	_, ok = ex.bookDepth(wsTestPair)
	assert.False(t, ok, "clearBookDepths should forget an order book's depth")

	ex.setBookDepth(wsTestPair, 10)
	ex.level3BooksMu.Lock()
	ex.level3Books = map[key.PairAsset]*level3Book{{Base: wsTestPair.Base.Item, Quote: wsTestPair.Quote.Item, Asset: asset.Spot}: {}}
	ex.level3BooksMu.Unlock()
	ex.clearBookDepths(&subscription.Subscription{QualifiedChannel: wsChannelLevel3, Pairs: currency.Pairs{wsTestPair}})
	_, ok = ex.bookDepth(wsTestPair)
	assert.False(t, ok, "clearBookDepths should forget a level 3 order book's depth")
	ex.level3BooksMu.Lock()
	assert.Empty(t, ex.level3Books, "clearBookDepths should forget a level 3 order book")
	ex.level3BooksMu.Unlock()
}

func TestWebsocketProcessStatus(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsProcessStatus(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"system":"online","api_version":"v2","connection_id":12893948,"version":"2.0.9"}]`)}), "wsProcessStatus must not error for an online engine")
	assert.Equal(t, []any{&WebsocketStatus{System: "online", APIVersion: "v2", ConnectionID: 12893948, Version: "2.0.9"}}, relayed(ex), "wsProcessStatus should relay the status")
	err := ex.wsProcessStatus(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"system":"maintenance"}]`)})
	assert.ErrorIs(t, err, errSystemNotOnline, "wsProcessStatus should report an engine that is not online")
	assert.Equal(t, []any{&WebsocketStatus{System: "maintenance"}}, relayed(ex), "wsProcessStatus should relay a status that is not online")
	assert.Error(t, ex.wsProcessStatus(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"online"`)}), "wsProcessStatus should reject data that is not a list")
}

// bookMessage decodes a book fixture line, returning the message and its only order book
func bookMessage(t *testing.T, line []byte) (*wsChannelMessage, *WebsocketBook) {
	t.Helper()
	var msg wsChannelMessage
	require.NoError(t, json.Unmarshal(line, &msg), "Unmarshal must not error for the message")
	var books []WebsocketBook
	require.NoError(t, json.Unmarshal(msg.Data, &books), "Unmarshal must not error for the order books")
	require.Len(t, books, 1, "the message must hold one order book")
	return &msg, &books[0]
}

func TestWebsocketProcessBook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// The fixture's book was recorded with a depth of 10
	ex.setBookDepth(wsTestPair, 10)
	lines := fixtureLines(t, "testdata/wsBook.json")
	for i, line := range lines[:3] {
		msg, _ := bookMessage(t, line)
		require.NoErrorf(t, ex.wsProcessBook(t.Context(), nil, msg), "wsProcessBook must not error for line %d", i+1)
	}
	_, update := bookMessage(t, lines[3])
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Len(t, book.Bids, 10, "wsProcessBook should keep the subscribed depth of bids")
	assert.Len(t, book.Asks, 10, "wsProcessBook should keep the subscribed depth of asks")

	update.Checksum++
	data, err := json.Marshal([]WebsocketBook{*update})
	require.NoError(t, err, "Marshal must not error")
	err = ex.wsProcessBook(t.Context(), nil, &wsChannelMessage{Type: "update", Data: data})
	assert.ErrorIs(t, err, errChecksumMismatch, "wsProcessBook should reject an update failing its checksum")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsProcessBook should invalidate a book failing its checksum")
	assert.ErrorIs(t, ex.wsProcessBook(t.Context(), nil, &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[{"symbol":"BTCUSD"}]`)}), errInvalidSymbol, "wsProcessBook should reject an unknown symbol")
	assert.Error(t, ex.wsProcessBook(t.Context(), nil, &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessBook should reject data that is not a list")
}

func TestWebsocketLoadBookSnapshot(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.setBookDepth(wsTestPair, 10)
	_, snapshot := bookMessage(t, fixtureLines(t, "testdata/wsBook.json")[0])
	require.NoError(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, snapshot), "wsLoadBookSnapshot must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Len(t, book.Bids, 10, "wsLoadBookSnapshot should load each bid")
	assert.Equal(t, snapshot.Timestamp, book.LastUpdated, "wsLoadBookSnapshot should stamp the book with the snapshot's time")

	snapshot.Checksum++
	assert.ErrorIs(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, snapshot), errChecksumMismatch, "wsLoadBookSnapshot should reject a snapshot failing its checksum")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsLoadBookSnapshot should invalidate a snapshot failing its checksum")
}

func TestWebsocketLoadBookSnapshotChecksum(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	snapshot := func(checksum uint32) []byte {
		return []byte(`{"channel":"book","type":"snapshot","data":[{"symbol":"BTC/USD","bids":[{"price":100.0,"qty":1.0}],"asks":[{"price":101.0,"qty":1.0}],"checksum":` +
			strconv.FormatUint(uint64(checksum), 10) + `,"timestamp":"2026-10-13T00:00:00Z"}]}`)
	}
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, snapshot(1)), errChecksumMismatch, "wsHandleData must reject a snapshot failing its checksum")
	assert.Empty(t, relayed(ex), "a snapshot failing its checksum should not be relayed")
	_, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "a snapshot failing its checksum should not be stored")
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), nil, snapshot(0)), errChecksumMismatch, "wsHandleData should reject a snapshot whose checksum of zero does not match")

	valid := bookChecksum(&orderbook.Book{
		Bids: orderbook.Levels{bookLevel(100, "100.0", 1, "1.0")},
		Asks: orderbook.Levels{bookLevel(101, "101.0", 1, "1.0")},
	})
	require.NoError(t, ex.wsHandleData(t.Context(), nil, snapshot(valid)), "wsHandleData must load a snapshot passing its checksum")
	relayed(ex)
	// A full relay must not stop a later snapshot's checksum from being checked
	var relayErr error
	for relayErr == nil {
		relayErr = ex.Websocket.DataHandler.Send(t.Context(), nil)
	}
	require.ErrorIs(t, ex.wsHandleData(t.Context(), nil, snapshot(1)), errChecksumMismatch, "wsHandleData must reject a snapshot failing its checksum while the relay is full")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a snapshot failing its checksum should leave the stored book invalid")
}

func TestWebsocketUpdateBook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.setBookDepth(wsTestPair, 10)
	lines := fixtureLines(t, "testdata/wsBook.json")
	_, snapshot := bookMessage(t, lines[0])
	_, update := bookMessage(t, lines[1])
	assert.ErrorIs(t, ex.wsUpdateBook(t.Context(), wsTestPair, update), orderbook.ErrDepthNotFound, "wsUpdateBook should reject an update before a snapshot")
	require.NoError(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, snapshot), "wsLoadBookSnapshot must not error")
	require.NoError(t, ex.wsUpdateBook(t.Context(), wsTestPair, update), "wsUpdateBook must not error for an update passing its checksum")
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, update.Timestamp, book.LastUpdated, "wsUpdateBook should stamp the book with the update's time")

	_, next := bookMessage(t, lines[2])
	next.Checksum++
	assert.ErrorIs(t, ex.wsUpdateBook(t.Context(), wsTestPair, next), errChecksumMismatch, "wsUpdateBook should reject an update failing its checksum")
}

func TestWebsocketBookDepthChange(t *testing.T) {
	t.Parallel()
	// side returns n levels of quantity 1 from price, each a step further from the best
	side := func(price, step float64, n int) orderbook.Levels {
		l := make(orderbook.Levels, n)
		for i := range l {
			p := price + step*float64(i)
			l[i] = bookLevel(p, strconv.FormatFloat(p, 'f', 1, 64), 1, "1.0")
		}
		return l
	}
	levelsJSON := func(l orderbook.Levels) string {
		s := make([]string, len(l))
		for i := range l {
			s[i] = `{"price":` + l[i].StrPrice + `,"qty":` + l[i].StrAmount + `}`
		}
		return "[" + strings.Join(s, ",") + "]"
	}
	// message returns a book message sending bids and asks, with the checksum of Kraken's book after it: bookBids and
	// bookAsks
	message := func(msgType string, bids, asks, bookBids, bookAsks orderbook.Levels) []byte {
		checksum := strconv.FormatUint(uint64(bookChecksum(&orderbook.Book{Bids: bookBids, Asks: bookAsks})), 10)
		return []byte(`{"channel":"book","type":"` + msgType + `","data":[{"symbol":"BTC/USD","bids":` + levelsJSON(bids) + `,"asks":` + levelsJSON(asks) +
			`,"checksum":` + checksum + `,"timestamp":"2026-10-16T00:00:00Z"}]}`)
	}
	for _, tc := range []struct {
		name          string
		first, second int
		// updates holds the bids each update sends, and bids the bids Kraken's book holds after each
		updates, bids []orderbook.Levels
	}{
		{
			name:  "deeper",
			first: 10, second: 25,
			updates: []orderbook.Levels{
				{bookLevel(100, "100.0", 2, "2.0")},
				// Removing the best bid brings the 11th into the top 10, and the 26th into the subscribed depth
				{bookLevel(100, "100.0", 0, "0.0"), bookLevel(75, "75.0", 1, "1.0")},
			},
			bids: []orderbook.Levels{
				append(orderbook.Levels{bookLevel(100, "100.0", 2, "2.0")}, side(99, -1, 24)...),
				side(99, -1, 25),
			},
		},
		{
			name:  "shallower",
			first: 25, second: 10,
			updates: []orderbook.Levels{
				// A new best bid pushes the 10th out of the subscribed depth, which Kraken sends no update for
				{bookLevel(100.5, "100.5", 1, "1.0")},
				// The level pushed out has since left Kraken's book unseen, so removing the best bid brings the next, 90.0,
				// into the subscribed depth
				{bookLevel(100.5, "100.5", 0, "0.0"), bookLevel(90, "90.0", 1, "1.0")},
			},
			bids: []orderbook.Levels{
				append(orderbook.Levels{bookLevel(100.5, "100.5", 1, "1.0")}, side(100, -1, 9)...),
				append(side(100, -1, 9), bookLevel(90, "90.0", 1, "1.0")),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, conn := newSubscriptionTestExchange(t, new(mockSubscriptionServer))
			book := func(depth int) *subscription.Subscription {
				return &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: depth, Pairs: currency.Pairs{wsTestPair}}
			}
			first := book(tc.first)
			require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{first}), "subscribeForConnection must not error")
			bids, asks := side(100, -1, tc.first), side(101, 1, tc.first)
			require.NoError(t, ex.wsHandleData(t.Context(), conn, message(wsTypeSnapshot, bids, asks, bids, asks)), "wsHandleData must load the first snapshot")
			require.NoError(t, ex.unsubscribeForConnection(t.Context(), conn, subscription.List{first}), "unsubscribeForConnection must not error")

			require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{book(tc.second)}), "subscribeForConnection must not error")
			bids, asks = side(100, -1, tc.second), side(101, 1, tc.second)
			require.NoError(t, ex.wsHandleData(t.Context(), conn, message(wsTypeSnapshot, bids, asks, bids, asks)), "wsHandleData must load the second snapshot")
			for i := range tc.updates {
				require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, message("update", tc.updates[i], nil, tc.bids[i], asks)), "wsHandleData must apply update %d at the depth subscribed with", i+1)
			}
			stored, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
			require.NoError(t, err, "GetOrderbook must not error")
			assert.Equal(t, tc.bids[len(tc.bids)-1], stored.Bids, "the bids should be those of Kraken's book at the depth subscribed with")
			assert.Equal(t, asks, stored.Asks, "the asks should be those of Kraken's book at the depth subscribed with")
		})
	}
}

func TestWebsocketLevel3BookRoundTrip(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// The pair's book moves from the level3 channel to the book channel and back, at a depth of 10 on each
	ex.setBookDepth(wsTestPair, 10)
	level3 := level3Message(t, fixtureLines(t, "testdata/wsLevel3.json")[0])
	_, level2 := bookMessage(t, fixtureLines(t, "testdata/wsBook.json")[0])
	require.NoError(t, ex.wsLevel3Book(t.Context(), wsTestPair, true, level3), "wsLevel3Book must load the level 3 snapshot")
	require.NoError(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, level2), "wsLoadBookSnapshot must load the level 2 snapshot")
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, 10, book.MaxDepth, "the level 2 book should take its snapshot's depth")
	assert.False(t, book.PriceDuplication, "the level 2 book should not allow duplicate prices")

	require.NoError(t, ex.wsLevel3Book(t.Context(), wsTestPair, true, level3), "wsLevel3Book must load the level 3 snapshot again")
	book, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Len(t, book.Bids, len(level3.Bids), "the level 3 book should hold each bid order as a level")
	assert.Zero(t, book.MaxDepth, "the level 3 book should take its snapshot's depth")
	assert.True(t, book.PriceDuplication, "the level 3 book should allow duplicate prices")
	assert.NoError(t, book.Validate(), "the level 3 book should validate with the orders that share a price")
}

func TestWebsocketLoadBookSnapshotWithoutTime(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.setBookDepth(wsTestPair, 10)
	_, first := bookMessage(t, fixtureLines(t, "testdata/wsBook.json")[0])
	require.NoError(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, first), "wsLoadBookSnapshot must load the first snapshot")
	loaded, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")

	// The book is subscribed to again at another depth, and its snapshot passes its checksum but carries no time
	ex.setBookDepth(wsTestPair, 25)
	var second WebsocketBook
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTC/USD","bids":[{"price":100.0,"qty":1.0}],"asks":[{"price":101.0,"qty":1.0}]}`), &second), "Unmarshal must not error")
	second.Checksum = bookChecksum(&orderbook.Book{Bids: wsBookLevels(second.Bids), Asks: wsBookLevels(second.Asks)})
	require.ErrorIs(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, &second), orderbook.ErrLastUpdatedNotSet, "wsLoadBookSnapshot must refuse a snapshot without a time")
	stored, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, loaded, stored, "a refused snapshot should leave the previous book with its levels, options and time")
}

func TestWebsocketLoadBookSnapshotConsistentReads(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// Each snapshot holds as many levels a side as its depth and is timed by its depth, so a book read holding one
	// snapshot's levels with the other's depth or time is caught
	snapshots := make(map[int]*WebsocketBook, 2)
	for _, depth := range []int{10, 25} {
		bids, asks := make([]string, depth), make([]string, depth)
		for i := range depth {
			bids[i] = `{"price":` + strconv.Itoa(1000-i) + `.0,"qty":1.0}`
			asks[i] = `{"price":` + strconv.Itoa(1001+i) + `.0,"qty":1.0}`
		}
		b := new(WebsocketBook)
		require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTC/USD","bids":[`+strings.Join(bids, ",")+`],"asks":[`+strings.Join(asks, ",")+`]}`), b), "Unmarshal must not error")
		b.Timestamp = time.Unix(int64(depth), 0)
		b.Checksum = bookChecksum(&orderbook.Book{Bids: wsBookLevels(b.Bids), Asks: wsBookLevels(b.Asks)})
		snapshots[depth] = b
	}
	ex.setBookDepth(wsTestPair, 10)
	require.NoError(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, snapshots[10]), "wsLoadBookSnapshot must load the first snapshot")

	var mixed atomic.Int64
	done := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
				}
				b, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
				if !assert.NoError(t, err, "GetOrderbook should not error while snapshots load") {
					return
				}
				if len(b.Bids) != b.MaxDepth || len(b.Asks) != b.MaxDepth || !b.LastUpdated.Equal(time.Unix(int64(b.MaxDepth), 0)) {
					mixed.Add(1)
				}
			}
		})
	}
	for i := range 1000 {
		depth := 25
		if i%2 != 0 {
			depth = 10
		}
		ex.setBookDepth(wsTestPair, depth)
		if !assert.NoErrorf(t, ex.wsLoadBookSnapshot(t.Context(), wsTestPair, snapshots[depth]), "wsLoadBookSnapshot should load snapshot %d", i+1) {
			break
		}
		relayed(ex)
	}
	close(done)
	readers.Wait()
	assert.Zero(t, mixed.Load(), "no book read should hold one snapshot's levels with the other's depth or time")
}

func TestResubscribeOrderbook(t *testing.T) {
	t.Parallel()
	server := new(mockSubscriptionServer)
	ex, conn := newSubscriptionTestExchange(t, server)
	book := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{book}), "subscribeForConnection must not error")
	server.takeRequests()
	ex.resubscribeOrderbook(conn, &subscription.Subscription{QualifiedChannel: wsChannelBook, Asset: asset.Spot, Pairs: currency.Pairs{wsTestPair}, Levels: 10})
	exp := []string{
		`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
		`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "resubscribeOrderbook must resubscribe the book")
	assert.Equal(t, exp, got, "resubscribeOrderbook should unsubscribe the book before subscribing again")
	require.Eventually(t, func() bool {
		stored := ex.Websocket.GetSubscription(book)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, testWaitTimeout, 10*time.Millisecond, "the book must be subscribed again")

	ex.resubscribeOrderbook(conn, &subscription.Subscription{QualifiedChannel: wsChannelBook, Asset: asset.Spot, Pairs: currency.Pairs{wsTestDOGEPair}, Levels: 10})
	assert.Never(t, func() bool {
		got = append(got, server.takeRequests()...)
		return len(got) > len(exp)
	}, 100*time.Millisecond, 5*time.Millisecond, "resubscribeOrderbook should not resubscribe a book without a subscription")
}

func TestBookSubscriptionKey(t *testing.T) {
	t.Parallel()
	k := bookSubscriptionKey{&subscription.Subscription{QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}}
	for _, tc := range []struct {
		name  string
		sub   *subscription.Subscription
		match bool
	}{
		{"the standard channel name", &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}, true},
		{"Kraken's channel name", &subscription.Subscription{Channel: wsChannelBook, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}, true},
		{"another channel", &subscription.Subscription{Channel: subscription.AllOrdersChannel, QualifiedChannel: wsChannelLevel3, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}, false},
		{"another asset", &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Futures, Levels: 10, Pairs: currency.Pairs{wsTestPair}}, false},
		{"another pair", &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestDOGEPair}}, false},
		{"another depth", &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 25, Pairs: currency.Pairs{wsTestPair}}, false},
		{"a key without a subscription", nil, false},
	} {
		assert.Equalf(t, tc.match, k.Match(subscription.ExactKey{Subscription: tc.sub}), "Match should return %t for %s", tc.match, tc.name)
	}
}

// badChecksumBookSnapshot is a BTC/USD book snapshot failing its checksum
const badChecksumBookSnapshot = `{"channel":"book","type":"snapshot","data":[{"symbol":"BTC/USD","bids":[{"price":100.0,"qty":1.0}],"asks":[{"price":101.0,"qty":1.0}],"checksum":1,"timestamp":"2026-10-16T00:00:00Z"}]}`

func TestResubscribeOrderbookReceivingConnection(t *testing.T) {
	t.Parallel()
	// Two public connections each hold a subscription to the same book, one named by the standard channel and one by
	// Kraken's
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	useTestWebsocket(t, ex, "")
	setups, err := ex.connectionSetups(ex.Config)
	require.NoError(t, err, "connectionSetups must not error")
	public := slices.IndexFunc(setups, func(s *websocket.ConnectionSetup) bool { return s.MessageFilter == wsPublicConnection })
	require.NotEqual(t, -1, public, "connectionSetups must return the public connection")
	servers := []*mockSubscriptionServer{new(mockSubscriptionServer), new(mockSubscriptionServer)}
	subs := subscription.List{
		{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}},
		{Channel: wsChannelBook, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}},
	}
	filters := []string{"first", "second"}
	for i := range subs {
		s := *setups[public]
		s.URL, s.MessageFilter = newMockWebsocketURL(t, servers[i].handle), filters[i]
		s.GenerateSubscriptions = func() (subscription.List, error) { return subscription.List{subs[i]}, nil }
		require.NoErrorf(t, ex.Websocket.SetupNewConnection(&s), "SetupNewConnection must not error for the %s connection", filters[i])
	}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
	conn, err := ex.Websocket.GetConnection(filters[1])
	require.NoError(t, err, "GetConnection must not error")
	require.Same(t, subs[1], conn.Subscriptions().Get(subs[1]), "the second connection's store must hold its subscription")
	for _, server := range servers {
		server.takeRequests()
	}

	require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(badChecksumBookSnapshot)), errChecksumMismatch, "wsHandleData must reject the book")
	exp := []string{
		`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
		`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
	}
	var got []string
	require.Eventually(t, func() bool {
		got = append(got, servers[1].takeRequests()...)
		return len(got) >= len(exp)
	}, testWaitTimeout, 10*time.Millisecond, "the book must be resubscribed on the connection it failed on")
	assert.Equal(t, exp, got, "the book should be unsubscribed before being subscribed again")
	require.Eventually(t, func() bool {
		return subs[1].State() == subscription.SubscribedState && ex.Websocket.GetSubscription(subs[1]) == subs[1]
	}, testWaitTimeout, 10*time.Millisecond, "the second connection's subscription must be subscribed again")
	assert.Empty(t, servers[0].takeRequests(), "the first connection should not be resubscribed")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "the first connection's subscription should stay subscribed")
}

func TestResubscribeOrderbookStaleConnectionSubscription(t *testing.T) {
	t.Parallel()
	book := func() *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelBook, Asset: asset.Spot, Levels: 10, Pairs: currency.Pairs{wsTestPair}}
	}
	for _, tc := range []struct {
		name string
		// byEqual unsubscribes with a subscription equal to the one stored, which leaves the stored one's state
		// subscribed
		byEqual    bool
		staleState subscription.State
	}{
		{"unsubscribed itself", false, subscription.UnsubscribedState},
		{"unsubscribed by an equal subscription", true, subscription.SubscribedState},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := new(mockSubscriptionServer)
			ex := newTestExchange(t)
			seedTestAssetNames(ex)
			ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10}}
			useTestWebsocket(t, ex, newMockWebsocketURL(t, server.handle), wsPublicConnection)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
			conn, err := ex.Websocket.GetConnection(wsPublicConnection)
			require.NoError(t, err, "GetConnection must not error")
			subs := ex.Websocket.GetSubscriptions()
			require.Len(t, subs, 1, "Connect must make one subscription")
			stale := subs[0]
			// A library caller unsubscribes the book and subscribes to it again with a new subscription, which only the
			// manager records
			unsubscribe := stale
			if tc.byEqual {
				unsubscribe = book()
			}
			require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subscription.List{unsubscribe}), "UnsubscribeChannels must not error")
			live := book()
			require.NoError(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{live}), "SubscribeToChannels must not error")
			require.Same(t, live, ex.Websocket.GetSubscription(live), "the manager must hold the new subscription")
			require.Same(t, stale, conn.Subscriptions().Get(stale), "the connection's store must still hold the old subscription")
			server.takeRequests()

			require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(badChecksumBookSnapshot)), errChecksumMismatch, "wsHandleData must reject the book")
			exp := []string{
				`{"method":"unsubscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
				`{"method":"subscribe","params":{"channel":"book","symbol":["BTC/USD"],"depth":10}}`,
			}
			var got []string
			require.Eventually(t, func() bool {
				got = append(got, server.takeRequests()...)
				return len(got) >= len(exp)
			}, testWaitTimeout, 10*time.Millisecond, "the book must be resubscribed")
			assert.Equal(t, exp, got, "the book should be unsubscribed before being subscribed again")
			require.Eventually(t, func() bool {
				return live.State() == subscription.SubscribedState && ex.Websocket.GetSubscription(live) == live
			}, testWaitTimeout, 10*time.Millisecond, "the manager's subscription must be subscribed again")
			assert.Equal(t, tc.staleState, stale.State(), "the old subscription should not be resubscribed")
		})
	}
}

// level3Message decodes a level 3 fixture line's only order book
func level3Message(t *testing.T, line []byte) *WebsocketLevel3Book {
	t.Helper()
	var msg wsChannelMessage
	require.NoError(t, json.Unmarshal(line, &msg), "Unmarshal must not error for the message")
	var books []WebsocketLevel3Book
	require.NoError(t, json.Unmarshal(msg.Data, &books), "Unmarshal must not error for the level 3 order books")
	require.Len(t, books, 1, "the message must hold one level 3 order book")
	return &books[0]
}

func TestWebsocketLevel3Book(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.setBookDepth(wsTestPair, 10)
	lines := fixtureLines(t, "testdata/wsLevel3.json")
	assert.ErrorIs(t, ex.wsLevel3Book(t.Context(), wsTestPair, false, level3Message(t, lines[1])), errLevel3OutOfSync, "wsLevel3Book should reject an update before a snapshot")
	snapshot := level3Message(t, lines[0])
	require.NoError(t, ex.wsLevel3Book(t.Context(), wsTestPair, true, snapshot), "wsLevel3Book must load a snapshot")
	book, err := ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Len(t, book.Bids, len(snapshot.Bids), "wsLevel3Book should load each bid order as a level")
	assert.Len(t, book.Asks, len(snapshot.Asks), "wsLevel3Book should load each ask order as a level")
	assert.Equal(t, snapshot.Timestamp, book.LastUpdated, "wsLevel3Book should stamp the book with the snapshot's time")
	snapshot.Timestamp = time.Time{}
	require.NoError(t, ex.wsLevel3Book(t.Context(), wsTestPair, true, snapshot), "wsLevel3Book must load a snapshot without a timestamp")
	book, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.WithinDuration(t, time.Now(), book.LastUpdated, time.Minute, "wsLevel3Book should stamp a book without a timestamp when it is loaded")

	update := level3Message(t, lines[1])
	update.Checksum++
	assert.ErrorIs(t, ex.wsLevel3Book(t.Context(), wsTestPair, false, update), errChecksumMismatch, "wsLevel3Book should reject an update failing its checksum")
	_, err = ex.Websocket.Orderbook.GetOrderbook(wsTestPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsLevel3Book should invalidate a book failing its checksum")
	ex.level3BooksMu.Lock()
	assert.Empty(t, ex.level3Books, "wsLevel3Book should forget a book failing its checksum")
	ex.level3BooksMu.Unlock()
}

func TestLevel3BookApply(t *testing.T) {
	t.Parallel()
	b := &level3Book{
		bids: []WebsocketLevel3Order{level3Order(t, "", "B1", "100", "1"), level3Order(t, "", "B2", "99", "1")},
		asks: []WebsocketLevel3Order{level3Order(t, "", "A1", "101", "1"), level3Order(t, "", "A2", "102", "1")},
	}
	update := &WebsocketLevel3Book{
		Bids: []WebsocketLevel3Order{level3Order(t, "add", "B3", "100", "2")},
		Asks: []WebsocketLevel3Order{level3Order(t, "delete", "A1", "101", "1")},
	}
	require.NoError(t, b.apply(update, 1), "apply must not error")
	assert.Equal(t, []WebsocketLevel3Order{level3Order(t, "", "B1", "100", "1"), level3Order(t, "", "B3", "100", "2")}, b.bids, "apply should add an order behind its price's and keep the bids' first level")
	assert.Equal(t, []WebsocketLevel3Order{level3Order(t, "", "A2", "102", "1")}, b.asks, "apply should delete an order")
	assert.ErrorIs(t, b.apply(&WebsocketLevel3Book{Bids: []WebsocketLevel3Order{level3Order(t, "delete", "B9", "100", "1")}}, 1), errLevel3OutOfSync, "apply should reject a bid event for an order it does not hold")
	assert.ErrorIs(t, b.apply(&WebsocketLevel3Book{Asks: []WebsocketLevel3Order{level3Order(t, "modify", "A9", "102", "1")}}, 1), errLevel3OutOfSync, "apply should reject an ask event for an order it does not hold")
}

func TestInsertLevel3Order(t *testing.T) {
	t.Parallel()
	higher := func(price, other float64) bool { return price > other }
	var orders []WebsocketLevel3Order
	for _, o := range []WebsocketLevel3Order{
		level3Order(t, "", "B1", "100", "1"),
		level3Order(t, "", "B2", "99", "1"),
		level3Order(t, "", "B3", "100", "2"),
		level3Order(t, "", "B4", "101", "1"),
	} {
		orders = insertLevel3Order(orders, &o, higher)
	}
	ids := make([]string, len(orders))
	for i := range orders {
		ids[i] = orders[i].OrderID
	}
	assert.Equal(t, []string{"B4", "B1", "B3", "B2"}, ids, "insertLevel3Order should queue each order behind those at its price and better prices")
}

func TestLevel3Levels(t *testing.T) {
	t.Parallel()
	exp := orderbook.Levels{bookLevel(0.1, "0.10", 1.5, "1.50000000"), bookLevel(0.09, "0.09", 2, "2")}
	assert.Equal(t, exp, level3Levels([]WebsocketLevel3Order{level3Order(t, "", "B1", "0.10", "1.50000000"), level3Order(t, "", "B2", "0.09", "2")}), "level3Levels should convert each order to a level, keeping the digits sent")
	assert.Empty(t, level3Levels(nil), "level3Levels should return no levels without orders")
}

func TestWebsocketProcessCandles(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	msg := &wsChannelMessage{
		Timestamp: time.Date(2026, 10, 9, 0, 35, 0, 0, time.UTC),
		Data: json.RawMessage(`[{"symbol":"BTC/USD","open":81700,"high":81800,"low":81600,"close":81750,"vwap":81710,"trades":10,"volume":1.5,"interval_begin":"2026-10-09T00:34:00Z","interval":1},` +
			`{"symbol":"BTC/USD","open":81750,"high":81760,"low":81740,"close":81755,"vwap":81752,"trades":2,"volume":0.1,"interval_begin":"2026-10-09T00:35:00Z","interval":1}]`),
	}
	require.NoError(t, ex.wsProcessCandles(t.Context(), msg), "wsProcessCandles must not error")
	exp := []any{
		kline.Item{Exchange: t.Name(), Pair: wsTestPair, Asset: asset.Spot, Interval: kline.OneMin, Candles: []kline.Candle{
			{Time: time.Date(2026, 10, 9, 0, 34, 0, 0, time.UTC), Open: 81700, High: 81800, Low: 81600, Close: 81750, Volume: 1.5, QuoteVolume: 122565},
		}},
		kline.Item{Exchange: t.Name(), Pair: wsTestPair, Asset: asset.Spot, Interval: kline.OneMin, Candles: []kline.Candle{
			{Time: time.Date(2026, 10, 9, 0, 35, 0, 0, time.UTC), Open: 81750, High: 81760, Low: 81740, Close: 81755, Volume: 0.1, QuoteVolume: 8175.2, ValidationIssues: kline.PartialCandle},
		}},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessCandles should relay each candle with its quote volume, the volume at the volume weighted average price, marking one whose interval had not ended as partial")
	var relayErr error
	for relayErr == nil {
		relayErr = ex.Websocket.DataHandler.Send(t.Context(), nil)
	}
	assert.ErrorContains(t, ex.wsProcessCandles(t.Context(), msg), "channel buffer is full", "wsProcessCandles should return the error relaying a candle")
	relayed(ex)
	assert.ErrorIs(t, ex.wsProcessCandles(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTCUSD","interval":1}]`)}), errInvalidSymbol, "wsProcessCandles should reject an unknown symbol")
	assert.ErrorIs(t, ex.wsProcessCandles(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTC/USD","interval":2}]`)}), kline.ErrUnsupportedInterval, "wsProcessCandles should reject an interval Kraken does not stream")
	assert.Error(t, ex.wsProcessCandles(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessCandles should reject data that is not a list")
}

func TestWebsocketProcessTrades(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	msg := &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTC/USD","side":"sell","price":81776.6,"qty":0.0005,"ord_type":"market","trade_id":110760690,"timestamp":"2026-10-09T00:34:21.060761Z"}]`)}
	require.NoError(t, ex.wsProcessTrades(t.Context(), msg), "wsProcessTrades must not error")
	exp := []any{[]trade.Data{{Exchange: t.Name(), CurrencyPair: wsTestPair, AssetType: asset.Spot, TID: "110760690", Side: order.Sell, Price: 81776.6, Amount: 0.0005, Timestamp: time.Date(2026, 10, 9, 0, 34, 21, 60761000, time.UTC)}}}
	assert.Equal(t, exp, relayed(ex), "wsProcessTrades should relay the trades")
	assert.ErrorIs(t, ex.wsProcessTrades(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTC/USD","side":"up"}]`)}), order.ErrSideIsInvalid, "wsProcessTrades should reject an unknown side")
	assert.ErrorIs(t, ex.wsProcessTrades(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"symbol":"BTCUSD","side":"buy"}]`)}), errInvalidSymbol, "wsProcessTrades should reject an unknown symbol")
	assert.Error(t, ex.wsProcessTrades(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessTrades should reject data that is not a list")
	ex.Features.Enabled.TradeFeed = false
	assert.NoError(t, ex.wsProcessTrades(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessTrades should ignore trades with the trade feed and trade saving disabled")
	assert.Empty(t, relayed(ex), "wsProcessTrades should relay nothing with the trade feed and trade saving disabled")
}

func TestWebsocketProcessInstruments(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsProcessInstruments(t.Context(), &wsChannelMessage{Data: json.RawMessage(`{"assets":[],"pairs":[]}`)}), "wsProcessInstruments must not error")
	assert.Equal(t, []any{&WebsocketInstruments{Assets: []WebsocketInstrumentAsset{}, Pairs: []WebsocketInstrumentPair{}}}, relayed(ex), "wsProcessInstruments should relay the reference data")
	assert.Error(t, ex.wsProcessInstruments(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[]`)}), "wsProcessInstruments should reject data that is not an object")
}

func TestWebsocketProcessExecutions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	fillTime := time.Date(2023, 9, 22, 10, 33, 5, 709993000, time.UTC)
	const fillExecution = `{"order_id":"OK4GJX-KSTLS-7DZZO5","exec_id":"TGBB7L-HT5LX-J3BZ4A","exec_type":"trade","trade_id":62887576,"symbol":"BTC/USD","side":"sell","last_qty":0.005,"last_price":26599.9,"liquidity_ind":"t","cost":132.9995,"fees":[{"asset":"USD","qty":0.3458}],"timestamp":"2023-09-22T10:33:05.709993Z"}`
	msg := &wsChannelMessage{Data: json.RawMessage(`[` + fillExecution + `]`)}
	require.NoError(t, ex.wsProcessExecutions(t.Context(), msg), "wsProcessExecutions must not error")
	got := relayed(ex)
	require.Len(t, got, 2, "wsProcessExecutions must relay the order update and its fill")
	fillUpdate, ok := got[0].(*order.Detail)
	require.True(t, ok, "wsProcessExecutions must relay the order update first")
	assert.Equal(t, "OK4GJX-KSTLS-7DZZO5", fillUpdate.OrderID, "wsProcessExecutions should relay the order's update")
	exp := []fill.Data{{ID: "TGBB7L-HT5LX-J3BZ4A", Timestamp: fillTime, Exchange: t.Name(), AssetType: asset.Spot, CurrencyPair: wsTestPair, Side: order.Sell, OrderID: "OK4GJX-KSTLS-7DZZO5", TradeID: "62887576", Price: 26599.9, Amount: 0.005}}
	assert.Equal(t, exp, got[1], "wsProcessExecutions should relay the trade as a fill")

	require.NoError(t, ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"OK4GJX-KSTLS-7DZZO5","exec_type":"canceled","order_status":"canceled","timestamp":"2023-09-22T10:33:06Z"}]`)}), "wsProcessExecutions must not error for a cancellation")
	got = relayed(ex)
	require.Len(t, got, 1, "wsProcessExecutions must relay a cancellation's order update alone")
	assert.IsType(t, &order.Detail{}, got[0], "wsProcessExecutions should relay the cancelled order's update")
	assert.ErrorIs(t, ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"side":"up"}]`)}), order.ErrSideIsInvalid, "wsProcessExecutions should reject an execution it cannot convert")
	assert.Empty(t, relayed(ex), "wsProcessExecutions should relay nothing for an execution it cannot convert")
	assert.Error(t, ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`"bad"`)}), "wsProcessExecutions should reject data that is not a list")

	// An execution that cannot be converted does not hold back the rest of the message
	err := ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"OAIYAU-LGI3M-PFM5VW","side":"up"},` + fillExecution + `]`)})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "wsProcessExecutions should report an execution it cannot convert")
	got = relayed(ex)
	require.Len(t, got, 2, "wsProcessExecutions must relay the order update and fill after an execution it cannot convert")
	d, ok := got[0].(*order.Detail)
	require.True(t, ok, "wsProcessExecutions must relay the order update after an execution it cannot convert")
	assert.Equal(t, fillUpdate, d, "wsProcessExecutions should relay the same update after an execution it cannot convert")
	assert.Equal(t, exp, got[1], "wsProcessExecutions should relay the fill after an execution it cannot convert")

	t.Run("relay full", func(t *testing.T) {
		t.Parallel()
		server := &finishedSpotOrdersServer{replies: []string{queryOrdersReply(t, "closed", 10, 0.6916)}, hold: make(chan struct{})}
		ex := newHTTPTestExchange(t, server.handle(t))
		// Cleanups run last first, so a held read is let go before the server closes
		release := sync.OnceFunc(func() { close(server.hold) })
		t.Cleanup(release)
		seedTestAssetNames(ex)
		ex.spotOrders.interval = time.Millisecond
		useConnectedRefreshers(ex)
		ex.Features.Enabled.FillsFeed = true
		ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
		var relayErr error
		for relayErr == nil {
			relayErr = ex.Websocket.DataHandler.Send(t.Context(), nil)
		}
		// A fill of one order, then the event reporting another order filled
		msg := &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"OAIYAU-LGI3M-PFM5VW","exec_id":"TFEYXB-5OO6U-H7NFC5","exec_type":"trade","trade_id":91342277,"symbol":"DOGE/USD","side":"buy","last_qty":20,"last_price":0.2101,"order_status":"partially_filled","cum_qty":20,"timestamp":"2023-09-22T10:33:05.709993Z"},` +
			`{"order_id":"` + finishedSpotOrderID + `","exec_type":"filled","order_status":"filled","cum_qty":10,"timestamp":"2023-09-22T10:33:05.709994Z"}]`)}
		err := ex.wsProcessExecutions(t.Context(), msg)
		assert.EqualError(t, err, "channel buffer is full: failed to relay <*order.Detail>, channel buffer is full: failed to relay <*order.Detail>, channel buffer is full: failed to relay <[]fill.Data>",
			"wsProcessExecutions should return the error relaying each update and the fills")
		require.Eventually(t, func() bool { return len(server.readOrders()) == 1 }, testWaitTimeout, time.Millisecond, "the order reported filled must be read although its update was not relayed")
		relayed(ex)
		release()
		var fee float64
		assert.Eventually(t, func() bool {
			for _, payload := range relayed(ex) {
				if d, ok := payload.(*order.Detail); ok && d.OrderID == finishedSpotOrderID {
					fee = d.Fee
				}
			}
			return fee != 0
		}, testWaitTimeout, time.Millisecond, "the read should relay the order reported filled")
		assert.Equal(t, 0.6916, fee, "the read should relay the order's fee")
		assert.Equal(t, [][]string{{finishedSpotOrderID}}, server.readOrders(), "only the order reported filled should be read")
	})
}

func TestExecutionOrderDetail(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	tradeTime := time.Date(2026, 10, 9, 1, 5, 0, 120000000, time.UTC)
	execution := &WebsocketExecution{
		OrderID:            "OAIYAU-LGI3M-PFM5VW",
		ClientOrderID:      "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		Symbol:             "DOGE/USD",
		Side:               "buy",
		OrderType:          "stop-loss-limit",
		OrderStatus:        "partially_filled",
		TimeInForce:        "GTD",
		PostOnly:           true,
		ReduceOnly:         true,
		OrderQuantity:      1200,
		CumulativeQuantity: 20,
		AveragePrice:       0.2101,
		CumulativeCost:     4.202,
		LimitPrice:         0.2101,
		Triggers:           WebsocketExecutionTriggers{Price: 0.21},
		ExecutionType:      "trade",
		ExecutionID:        "TFEYXB-5OO6U-H7NFC5",
		LastPrice:          0.2101,
		LastQuantity:       20,
		LiquidityIndicator: "m",
		Cost:               4.202,
		Fees:               []WebsocketExecutionFee{{Asset: "USD", Quantity: 0.0067}},
		Timestamp:          tradeTime,
	}
	d, err := ex.executionOrderDetail(execution)
	require.NoError(t, err, "executionOrderDetail must not error")
	exp := &order.Detail{
		Exchange:             t.Name(),
		AssetType:            asset.Spot,
		OrderID:              "OAIYAU-LGI3M-PFM5VW",
		ClientOrderID:        "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		Pair:                 wsTestDOGEPair,
		FeeAsset:             currency.USD,
		Side:                 order.Buy,
		Type:                 order.StopLimit,
		Status:               order.PartiallyFilled,
		TimeInForce:          order.GoodTillTime | order.PostOnly,
		Amount:               1200,
		ExecutedAmount:       20,
		AverageExecutedPrice: 0.2101,
		Cost:                 4.202,
		Price:                0.2101,
		TriggerPrice:         0.21,
		ReduceOnly:           true,
		LastUpdated:          tradeTime,
		Trades: []order.TradeHistory{{
			TID:       "TFEYXB-5OO6U-H7NFC5",
			Price:     0.2101,
			Amount:    20,
			Fee:       0.0067,
			FeeAsset:  "USD",
			Exchange:  t.Name(),
			Type:      order.StopLimit,
			Side:      order.Buy,
			IsMaker:   true,
			Timestamp: tradeTime,
			Total:     4.202,
		}},
	}
	assert.Equal(t, exp, d, "executionOrderDetail should convert a trade with its fill")

	for _, priceType := range []string{"pct", "quote"} {
		d, err = ex.executionOrderDetail(&WebsocketExecution{OrderID: "OAIYAU-LGI3M-PFM5VW", ExecutionType: "status", Triggers: WebsocketExecutionTriggers{Reference: "last", Price: 5, PriceType: priceType, ActualPrice: 95000}})
		require.NoErrorf(t, err, "executionOrderDetail must not error for a %s trigger", priceType)
		assert.Equalf(t, 95000.0, d.TriggerPrice, "executionOrderDetail should report the trigger price a %s trigger works out to", priceType)
	}

	d, err = ex.executionOrderDetail(&WebsocketExecution{OrderID: "OAIYAU-LGI3M-PFM5VW", ExecutionType: "pending_new", OrderStatus: "pending_new", Timestamp: tradeTime})
	require.NoError(t, err, "executionOrderDetail must not error for a pending order")
	assert.Equal(t, tradeTime, d.Date, "executionOrderDetail should date a pending order by its event")
	for _, tc := range []struct {
		filled float64
		exp    order.Status
	}{{0, order.Cancelled}, {0.0001, order.PartiallyFilledCancelled}, {20, order.PartiallyFilledCancelled}} {
		d, err = ex.executionOrderDetail(&WebsocketExecution{OrderID: "OAIYAU-LGI3M-PFM5VW", ExecutionType: "canceled", OrderStatus: "canceled", CumulativeQuantity: tc.filled, Timestamp: tradeTime})
		require.NoErrorf(t, err, "executionOrderDetail must not error for a cancellation after %v filled", tc.filled)
		assert.Equalf(t, tc.exp, d.Status, "executionOrderDetail should report a cancellation after %v filled as %s", tc.filled, tc.exp)
	}
	for _, tc := range []struct {
		name      string
		execution *WebsocketExecution
		err       error
	}{
		{name: "symbol", execution: &WebsocketExecution{Symbol: "BTC"}, err: errInvalidSymbol},
		{name: "side", execution: &WebsocketExecution{Side: "up"}, err: order.ErrSideIsInvalid},
		{name: "order type", execution: &WebsocketExecution{OrderType: "moon"}, err: errUnknownOrderType},
		{name: "order status", execution: &WebsocketExecution{OrderStatus: "moon"}, err: errUnknownOrderStatus},
		{name: "time in force", execution: &WebsocketExecution{TimeInForce: "GTX"}, err: order.ErrInvalidTimeInForce},
	} {
		_, err := ex.executionOrderDetail(tc.execution)
		assert.ErrorIsf(t, err, tc.err, "executionOrderDetail should reject an execution by its %s", tc.name)
	}
}

func TestWebsocketProcessBalances(t *testing.T) {
	t.Parallel()
	server := &spotBalancesServer{reply: `{"error":[],"result":{"ZUSD":{"balance":"100","hold_trade":"10","credit":"50","credit_used":"20"}}}`}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotBalances.interval = time.Millisecond
	useConnectedRefreshers(ex)
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.XDG: {Total: 7, Free: 7}}}}, true), "Save must not error")

	snapshot := &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[{"asset":"USD","asset_class":"currency","balance":150,"wallets":[{"type":"spot","id":"main","balance":120},{"type":"earn","id":"flex","balance":30}]}]`)}
	require.NoError(t, ex.wsProcessBalances(t.Context(), snapshot), "wsProcessBalances must not error for a snapshot")
	expBalances := []WebsocketBalance{{Asset: "USD", AssetClass: "currency", Balance: 150, Wallets: []WebsocketWallet{{Type: "spot", ID: "main", Balance: 120}, {Type: "earn", ID: "flex", Balance: 30}}}}
	assert.Equal(t, []any{expBalances}, relayed(ex), "wsProcessBalances should relay a snapshot's balances")
	// The snapshot's spot wallet holds 120, but only Get Extended Balance reports the hold and the credit
	assertStoredSpotBalances(t, ex, accounts.CurrencyBalances{
		currency.USD: {Currency: currency.USD, Total: 100, Hold: 10, Free: 120, AvailableWithoutBorrow: 90, Borrowed: 20},
	}, "a snapshot should refresh the stored balances whole, the unused credit counted as free and a currency no longer listed removed")

	server.setReply(`{"error":[],"result":{"ZUSD":{"balance":"60","hold_trade":"0"},"XXBT":{"balance":"0.005","hold_trade":"0"}}}`)
	update := &wsChannelMessage{Type: "update", Data: json.RawMessage(`[{"ledger_id":"L4UESK-KG3EQ-UFO4T5","ref_id":"TJKLXX-PGMUI-4NTLXU","timestamp":"2023-09-22T10:33:05.709993Z","type":"trade","category":"trade","asset":"BTC","asset_class":"currency","wallet_type":"spot","wallet_id":"main","amount":-0.005,"fee":0,"balance":1.195}]`)}
	require.NoError(t, ex.wsProcessBalances(t.Context(), update), "wsProcessBalances must not error for an update")
	exp := []any{[]WebsocketLedgerEntry{{
		LedgerID:    "L4UESK-KG3EQ-UFO4T5",
		ReferenceID: "TJKLXX-PGMUI-4NTLXU",
		Timestamp:   time.Date(2023, 9, 22, 10, 33, 5, 709993000, time.UTC),
		Type:        "trade",
		Category:    "trade",
		Asset:       "BTC",
		AssetClass:  "currency",
		WalletType:  "spot",
		WalletID:    "main",
		Amount:      -0.005,
		Balance:     1.195,
	}}}
	assert.Equal(t, exp, relayed(ex), "wsProcessBalances should relay an update's ledger entries")
	assertStoredSpotBalances(t, ex, accounts.CurrencyBalances{
		currency.USD: {Currency: currency.USD, Total: 60, Free: 60, AvailableWithoutBorrow: 60},
		currency.XBT: {Currency: currency.XBT, Total: 0.005, Free: 0.005, AvailableWithoutBorrow: 0.005},
	}, "an update should refresh the stored balances")

	// The update's refresh reads the balances once more, so the refresher must be idle before the reads are counted
	require.Eventually(t, func() bool {
		ex.spotBalances.mu.Lock()
		defer ex.spotBalances.mu.Unlock()
		return !ex.spotBalances.running
	}, testWaitTimeout, time.Millisecond, "the refresher must finish the update's refresh")
	reads := server.readCount()
	for _, msg := range []*wsChannelMessage{{Type: wsTypeSnapshot, Data: json.RawMessage(`"bad"`)}, {Type: "update", Data: json.RawMessage(`"bad"`)}} {
		assert.Errorf(t, ex.wsProcessBalances(t.Context(), msg), "wsProcessBalances should reject a %s that is not a list", msg.Type)
	}
	assert.Never(t, func() bool { return server.readCount() != reads }, 50*time.Millisecond, time.Millisecond, "a message wsProcessBalances rejects should start no refresh")
}

func TestIntervalFromMinutes(t *testing.T) {
	t.Parallel()
	for minutes, exp := range map[uint64]kline.Interval{1: kline.OneMin, 60: kline.OneHour, 1440: kline.OneDay, 10080: kline.OneWeek, 21600: kline.FifteenDay} {
		interval, err := intervalFromMinutes(minutes)
		require.NoErrorf(t, err, "intervalFromMinutes must not error for %d minutes", minutes)
		assert.Equalf(t, exp, interval, "intervalFromMinutes should return the interval of %d minutes", minutes)
	}
	_, err := intervalFromMinutes(2)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "intervalFromMinutes should reject an interval Kraken does not stream")
}

func TestFormatTradeID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "110760685", formatTradeID(110760685), "formatTradeID should format the trade ID in decimal")
	assert.Equal(t, "0", formatTradeID(0), "formatTradeID should format a zero trade ID")
}

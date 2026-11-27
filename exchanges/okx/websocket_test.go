package okx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
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
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
	"golang.org/x/time/rate"
)

// websocketTestFrame holds a request a mock connection received
type websocketTestFrame struct {
	path      string // The path the connection dialled
	loggedIn  bool   // Whether the connection had logged in when the request arrived
	operation string
	arguments json.RawMessage
}

// websocketTestServer serves mock OKX websocket connections: it answers
// logins with loginCode, acknowledges each subscription argument, or rejects it
// with the error code reject returns, and sends the pushes push returns for
// each subscription, and answers order operations with reply
type websocketTestServer struct {
	loginCode string
	reject    func(path, operation string, argument json.RawMessage) (code string)
	reply     func(path, operation string, arguments json.RawMessage) (code string, data any)
	push      func(path string, argument json.RawMessage) []string

	mu     sync.Mutex
	dials  []string
	frames []websocketTestFrame
}

func (s *websocketTestServer) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	path := r.URL.Path
	s.mu.Lock()
	s.dials = append(s.dials, path)
	s.mu.Unlock()
	var loggedIn bool
	mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
		if string(msg) == "ping" {
			return c.WriteMessage(gws.TextMessage, pongMsg)
		}
		var req struct {
			ID        string          `json:"id"`
			Operation string          `json:"op"`
			Arguments json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		s.mu.Lock()
		s.frames = append(s.frames, websocketTestFrame{path: path, loggedIn: loggedIn, operation: req.Operation, arguments: req.Arguments})
		s.mu.Unlock()
		switch req.Operation {
		case operationLogin:
			code := cmp.Or(s.loginCode, "0")
			event := "error"
			if code == "0" {
				event, loggedIn = operationLogin, true
			}
			return c.WriteMessage(gws.TextMessage, []byte(`{"event":"`+event+`","code":"`+code+`","msg":"","connId":"a4d3ae55"}`))
		case operationSubscribe, operationUnsubscribe:
			var args []json.RawMessage
			if err := json.Unmarshal(req.Arguments, &args); err != nil {
				return err
			}
			for _, arg := range args {
				if s.reject != nil {
					if code := s.reject(path, req.Operation, arg); code != "" {
						if err := c.WriteMessage(gws.TextMessage, []byte(`{"event":"error","msg":"Subscribe failed, wrong URL or channel doesn't exist.","code":"`+code+`","connId":"a4d3ae55","id":"`+req.ID+`"}`)); err != nil {
							return err
						}
						continue
					}
				}
				if err := c.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","event":"`+req.Operation+`","arg":`+string(arg)+`,"connId":"a4d3ae55"}`)); err != nil {
					return err
				}
				if s.push == nil || req.Operation != operationSubscribe {
					continue
				}
				for _, p := range s.push(path, arg) {
					if err := c.WriteMessage(gws.TextMessage, []byte(p)); err != nil {
						return err
					}
				}
			}
			return nil
		}
		code, data := "0", any([]any{})
		if s.reply != nil {
			code, data = s.reply(path, req.Operation, req.Arguments)
		}
		reply, err := json.Marshal(map[string]any{"id": req.ID, "op": req.Operation, "code": code, "msg": "", "data": data, "inTime": "1795406820638123", "outTime": "1795406820639456"})
		if err != nil {
			return err
		}
		return c.WriteMessage(gws.TextMessage, reply)
	})
}

// recorded returns the paths dialled and the requests received so far
func (s *websocketTestServer) recorded() (dials []string, frames []websocketTestFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.dials), slices.Clone(s.frames)
}

// requests returns the requests received for an operation
func (s *websocketTestServer) requests(operation string) []websocketTestFrame {
	_, frames := s.recorded()
	return slices.DeleteFunc(frames, func(f websocketTestFrame) bool { return f.operation != operation })
}

// dialWebsocketTestServer dials an in-memory server on an unmanaged connection
// of the message filter, whose frames the exchange handles, and the server
// answers each request with the frames answer returns. In a synctest bubble
// the connection's waits run on the bubble's clock.
func dialWebsocketTestServer(t *testing.T, ex *Exchange, filter string, answer func(id, operation string, arguments []json.RawMessage) []string) websocket.Connection {
	t.Helper()
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			var req struct {
				ID        string            `json:"id"`
				Operation string            `json:"op"`
				Arguments []json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			for _, frame := range answer(req.ID, req.Operation, req.Arguments) {
				if err := c.WriteMessage(gws.TextMessage, []byte(frame)); err != nil {
					return err
				}
			}
			return nil
		})
	}))
	conn, err := ex.Websocket.CreateTestConnection(filter)
	require.NoError(t, err, "CreateTestConnection must not error")
	conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v5/public")
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
	read := make(chan struct{})
	go func() {
		defer close(read)
		for {
			resp := conn.ReadMessage()
			if resp.Raw == nil {
				return
			}
			if err := ex.wsHandleData(context.Background(), conn, resp.Raw); err != nil {
				assert.NoError(t, ex.Websocket.DataHandler.Send(context.Background(), err), "a frame's error should be relayed")
			}
		}
	}()
	t.Cleanup(func() {
		assert.NoError(t, conn.Shutdown(), "Shutdown should not error")
		select {
		case <-read:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the reader should stop once the connection closes")
		}
	})
	return conn
}

// subscriptionReply returns OKX's acknowledgement of an argument of a request
func subscriptionReply(id, operation string, argument json.RawMessage) string {
	return `{"id":"` + id + `","event":"` + operation + `","arg":` + string(argument) + `,"connId":"a4d3ae55"}`
}

// newWebsocketTestExchange returns an exchange whose websocket connections dial
// srv, each on its documented path. With credentials the connections which log
// in do so; with nil subscriptions every connection dials without subscribing.
// configure, if any, adjusts the exchange before its websocket is set up.
func newWebsocketTestExchange(t *testing.T, srv *websocketTestServer, credentials bool, subs subscription.List, configure ...func(*Exchange)) *Exchange {
	t.Helper()
	ex, err := connectWebsocketTestExchange(t, srv, credentials, subs, configure...)
	require.NoError(t, err, "Connect must not error")
	return ex
}

// newWebsocketTestExchangeWithLimits returns newWebsocketTestExchange's
// exchange, its requester and websocket drawing on limits when given
func newWebsocketTestExchangeWithLimits(t *testing.T, srv *websocketTestServer, credentials bool, subs subscription.List, limits request.RateLimitDefinitions) *Exchange {
	t.Helper()
	ex, err := connectWebsocketTestExchangeWithLimits(t, srv, credentials, subs, limits)
	require.NoError(t, err, "Connect must not error")
	return ex
}

// connectWebsocketTestExchange is newWebsocketTestExchange returning the error
// connecting returns
func connectWebsocketTestExchange(t *testing.T, srv *websocketTestServer, credentials bool, subs subscription.List, configure ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	return connectWebsocketTestExchangeWithLimits(t, srv, credentials, subs, nil, configure...)
}

// connectWebsocketTestExchangeWithLimits is connectWebsocketTestExchange with
// the requester and websocket drawing on limits when given
func connectWebsocketTestExchangeWithLimits(t *testing.T, srv *websocketTestServer, credentials bool, subs subscription.List, limits request.RateLimitDefinitions, configure ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	var handlers sync.WaitGroup
	s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.Add(1)
		defer handlers.Done()
		srv.serve(t, w, r)
	}))
	s.Start()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	if limits != nil {
		request.WithLimiter(limits)(ex.Requester)
	}
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	for endpoint, path := range map[exchange.URL]string{
		exchange.WebsocketSpot:              "/ws/v5/public",
		exchange.WebsocketPrivate:           "/ws/v5/private",
		exchange.WebsocketSpotSupplementary: "/ws/v5/business",
	} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint.String(), wsURL+path), "SetRunningURL must not error")
	}
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	ex.Features.Subscriptions = subs
	if credentials {
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	}
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	for _, f := range configure {
		f(ex)
	}
	require.NoError(t, ex.setupWebsocket(ex.Config), "setupWebsocket must not error")
	if subs == nil {
		ex.Websocket.SetSubscriptionsNotRequired()
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(credentials)
	t.Cleanup(func() {
		if err := ex.Websocket.Disable(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should not error")
		}
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error")
		}
		handlers.Wait()
	})
	return ex, ex.Websocket.Connect(context.Background())
}

func TestSetupWebsocket(t *testing.T) {
	t.Parallel()

	t.Run("dials every connection with its login", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		newWebsocketTestExchange(t, srv, true, nil)
		require.Eventually(t, func() bool { return len(srv.requests(operationLogin)) == 3 }, 5*time.Second, 10*time.Millisecond, "the three login connections must log in")
		dials, _ := srv.recorded()
		slices.Sort(dials)
		assert.Equal(t, []string{"/ws/v5/business", "/ws/v5/business", "/ws/v5/private", "/ws/v5/public", "/ws/v5/public"}, dials, "every connection should dial its documented path")
		frames := srv.requests(operationLogin)
		logins := make([]string, 0, len(frames))
		for _, f := range frames {
			logins = append(logins, f.path)
		}
		slices.Sort(logins)
		assert.Equal(t, []string{"/ws/v5/business", "/ws/v5/private", "/ws/v5/public"}, logins, "one connection of each path should log in")
	})

	t.Run("routes each channel to its connection", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		newWebsocketTestExchange(t, srv, true, subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: channelOrderBooksTBT, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spot, Authenticated: true},
			{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
			{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spread, Authenticated: true},
		})
		require.Eventually(t, func() bool { return len(srv.requests(operationSubscribe)) == 5 }, 5*time.Second, 10*time.Millisecond, "every subscription must reach the server")
		got := map[string]string{}
		for _, f := range srv.requests(operationSubscribe) {
			got[string(f.arguments)] = fmt.Sprintf("%s login=%t", f.path, f.loggedIn)
		}
		assert.Equal(t, map[string]string{
			`[{"channel":"tickers","instId":"BTC-USDT"}]`:      "/ws/v5/public login=false",
			`[{"channel":"books-l2-tbt","instId":"BTC-USDT"}]`: "/ws/v5/public login=true",
			`[{"channel":"orders","instType":"SPOT"}]`:         "/ws/v5/private login=true",
			`[{"channel":"candle1m","instId":"BTC-USDT"}]`:     "/ws/v5/business login=false",
			`[{"channel":"sprd-orders"}]`:                      "/ws/v5/business login=true",
		}, got, "each channel should be subscribed on the connection the docs serve it on")
	})

	t.Run("opens another connection above the subscription limit", func(t *testing.T) {
		t.Parallel()
		pairs := make(currency.Pairs, 31)
		for i := range pairs {
			pairs[i] = currency.NewPairWithDelimiter(fmt.Sprintf("TEST%d", i), "USDT", currency.DashDelimiter)
		}
		srv := new(websocketTestServer)
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: pairs}})
		require.Eventually(t, func() bool { return len(srv.requests(operationSubscribe)) == 2 }, 5*time.Second, 10*time.Millisecond, "both connections must subscribe")
		dials, _ := srv.recorded()
		assert.Equal(t, []string{"/ws/v5/public", "/ws/v5/public"}, dials, "31 subscriptions should need a second public connection")
		frames := srv.requests(operationSubscribe)
		counts := make([]int, 0, len(frames))
		for _, f := range frames {
			var args []SubscriptionArgument
			require.NoError(t, json.Unmarshal(f.arguments, &args), "subscription arguments must decode")
			counts = append(counts, len(args))
		}
		slices.Sort(counts)
		assert.Equal(t, []int{1, 30}, counts, "a connection should hold at most 30 subscriptions")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 31, "every subscription should be tracked")
	})

	t.Run("subscribes public business channels without credentials", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{push: func(_ string, arg json.RawMessage) []string {
			if strings.Contains(string(arg), `"candle1m"`) {
				return []string{`{"arg":{"channel":"candle1m","instId":"BTC-USDT"},"data":[["1795405200000","61502.4","61502.4","61489.3","61491.4","0.00161768","99.484871076","99.484871076","0"]]}`}
			}
			return nil
		}}
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{
			{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
			{Enabled: true, Channel: channelSpreadTickers, Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}},
			{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spread, Authenticated: true},
		})
		require.Eventually(t, func() bool { return len(srv.requests(operationSubscribe)) == 1 }, 5*time.Second, 10*time.Millisecond, "the public business subscriptions must reach the server")
		assert.Empty(t, srv.requests(operationLogin), "no connection should log in without credentials")
		dials, _ := srv.recorded()
		assert.Equal(t, []string{"/ws/v5/business"}, dials, "only the business connection should dial")
		f := srv.requests(operationSubscribe)[0]
		assert.Equal(t, "/ws/v5/business", f.path, "public business channels should be subscribed on the business connection")
		assert.False(t, f.loggedIn, "public business channels should be subscribed without a login")
		assert.JSONEq(t, `[{"channel":"candle1m","instId":"BTC-USDT"},{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"}]`, string(f.arguments), "both public business channels should be subscribed together")
		select {
		case resp := <-ex.Websocket.DataHandler.C:
			_, ok := resp.Data.(kline.Item)
			assert.Truef(t, ok, "the candlestick push should be relayed as a kline item, got %T", resp.Data)
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the candlestick push should be relayed")
		}
	})

	t.Run("rejected subscriptions are reported and left untracked", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reject: func(_, _ string, arg json.RawMessage) string {
			if strings.Contains(string(arg), `"trades"`) {
				return "60018"
			}
			return ""
		}}
		ex, err := connectWebsocketTestExchange(t, srv, false, subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
		})
		require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Connect must report the subscription failure")
		require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "Connect must report the subscription OKX did not acknowledge")
		require.ErrorIs(t, err, errWebsocketRequestRejected, "Connect must report OKX's rejection")
		assert.ErrorContains(t, err, `code "60018"`, "the error should carry OKX's code")
		assert.True(t, ex.Websocket.IsConnected(), "a rejected subscription should leave the websocket connected")
		subs := ex.Websocket.GetSubscriptions()
		require.Len(t, subs, 1, "only the acknowledged subscription must be tracked")
		assert.JSONEq(t, `{"channel":"tickers","instId":"BTC-USDT"}`, subs[0].QualifiedChannel, "the acknowledged subscription should be tracked")
		assert.Equal(t, subscription.SubscribedState, subs[0].State(), "the acknowledged subscription should be subscribed")
		frames := srv.requests(operationSubscribe)
		require.Len(t, frames, 1, "both subscriptions must be sent in one request")
		assert.JSONEq(t, `[{"channel":"tickers","instId":"BTC-USDT"},{"channel":"trades","instId":"BTC-USDT"}]`, string(frames[0].arguments), "both arguments should be sent")
	})

	t.Run("live public subscriptions", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("live only: the mocked connections are covered above")
		}
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ethPair := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
		okbPair := currency.NewPairWithDelimiter("OKB", "USDT", currency.DashDelimiter)
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: channelCandle + "1D", Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: channelMarkPriceCandle + "1M", Asset: asset.Spot, Pairs: currency.Pairs{ethPair}},
			{Enabled: true, Channel: channelIndexCandle + "1H", Asset: asset.Spot, Pairs: currency.Pairs{okbPair}},
		}
		t.Cleanup(func() {
			if err := ex.Websocket.Disable(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should not error")
			}
			if err := ex.Websocket.Shutdown(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error")
			}
		})
		require.NoError(t, ex.Websocket.Connect(context.Background()), "Connect must not error")
		seen := map[string]currency.Pair{}
		deadline := time.After(30 * time.Second)
		for len(seen) < 4 {
			select {
			case resp := <-ex.Websocket.DataHandler.C:
				switch v := resp.Data.(type) {
				case []ticker.Price:
					seen["tickers"] = v[0].Pair
				case kline.Item:
					seen["candles"] = v.Pair
				case *WebsocketMarkPriceCandlesticksPush:
					seen["mark price candles"] = v.Argument.InstrumentID
				case *WebsocketIndexCandlesticksPush:
					seen["index candles"] = v.Argument.InstrumentID
				case error:
					require.NoError(t, v, "the live stream must not error")
				}
			case <-deadline:
				require.FailNowf(t, "live pushes must arrive", "seen %v", seen)
			}
		}
		exp := map[string]currency.Pair{"tickers": mainPair, "candles": mainPair, "mark price candles": ethPair, "index candles": okbPair}
		for k, p := range exp {
			assert.Truef(t, p.Equal(seen[k]), "the %s push should be for its subscribed pair %s, got %s", k, p, seen[k])
		}
	})
}

// dialRecorderConnection records when it is dialled
type dialRecorderConnection struct {
	websocket.Connection
	dials []time.Time
}

func (c *dialRecorderConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	c.dials = append(c.dials, time.Now())
	return nil
}

func (c *dialRecorderConnection) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()

	t.Run("dials every connection", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		ex := newWebsocketTestExchange(t, srv, false, nil)
		dials, _ := srv.recorded()
		assert.Len(t, dials, 5, "wsConnect should dial every connection")
		conn, err := ex.Websocket.GetConnection(publicConnection)
		require.NoError(t, err, "GetConnection must not error")
		connected, ok := conn.(interface{ IsConnected() bool })
		require.True(t, ok, "the manager's connections must report their state")
		assert.True(t, connected.IsConnected(), "wsConnect should leave the connection connected")

		unreachable, err := ex.Websocket.CreateTestConnection(publicConnection)
		require.NoError(t, err, "CreateTestConnection must not error")
		unreachable.SetURL("ws://127.0.0.1:1/ws/v5/public")
		assert.Error(t, ex.wsConnect(t.Context(), unreachable), "wsConnect should report a failed dial")
	})

	t.Run("every connection draws on one dial budget", func(t *testing.T) {
		t.Parallel()
		var pools map[request.EndpointLimit]*rate.Limiter
		srv := new(websocketTestServer)
		newWebsocketTestExchange(t, srv, true, nil, func(ex *Exchange) { pools = chargeToPools(t, ex) })
		dials, _ := srv.recorded()
		require.Len(t, dials, 5, "every connection must dial")
		assert.Equal(t, map[request.EndpointLimit]float64{websocketConnectionEPL: 5}, takenTokens(pools), "each dial should take one token of the connection budget")
	})

	t.Run("paces dials within OKX's limit of three a second", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		synctest.Test(t, func(t *testing.T) {
			request.WithLimiter(newRateLimits())(ex.Requester)
			conn := new(dialRecorderConnection)
			for range 4 {
				require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must not error")
			}
			require.Len(t, conn.dials, 4, "every connection must dial")
			for i, dial := range conn.dials {
				assert.InDeltaf(t, float64(i)*float64(400*time.Millisecond), float64(dial.Sub(conn.dials[0])), float64(time.Millisecond), "dial %d should wait 400ms after the one before", i)
			}
		})
	})

	t.Run("refuses a dial the context cannot wait for", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		synctest.Test(t, func(t *testing.T) {
			request.WithLimiter(newRateLimits())(ex.Requester)
			conn := new(dialRecorderConnection)
			require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must not error")
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			require.ErrorIs(t, ex.wsConnect(ctx, conn), context.DeadlineExceeded, "wsConnect must refuse a dial its context ends before")
			assert.Len(t, conn.dials, 1, "the refused dial should not be attempted")
		})
	})
}

func TestWebsocketAuthenticateConnection(t *testing.T) {
	t.Parallel()

	t.Run("logs in", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		newWebsocketTestExchange(t, srv, true, nil)
		require.Eventually(t, func() bool { return len(srv.requests(operationLogin)) == 3 }, 5*time.Second, 10*time.Millisecond, "every login connection must log in")
		var args []WebsocketLoginArgument
		require.NoError(t, json.Unmarshal(srv.requests(operationLogin)[0].arguments, &args), "the login arguments must decode")
		require.Len(t, args, 1, "a login must carry one account")
		assert.Equal(t, "mock-api-key", args[0].APIKey, "the login should carry the API key")
		assert.Equal(t, "mock-passphrase", args[0].Passphrase, "the login should carry the passphrase")
		assert.NotZero(t, args[0].Timestamp, "the login should carry its Unix time")
		assert.NotEmpty(t, args[0].Sign, "the login should be signed")
	})

	t.Run("records the credentials each connection logged in with", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, new(websocketTestServer), true, nil)
		exp := newWebsocketLogin(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
		for _, connection := range []string{publicLoginConnection, privateConnection, businessLoginConnection} {
			conn, err := ex.Websocket.GetConnection(connection)
			require.NoErrorf(t, err, "the %s connection must be available", connection)
			ex.websocketLoginsMu.Lock()
			login, ok := ex.websocketLogins[conn]
			ex.websocketLoginsMu.Unlock()
			require.Truef(t, ok, "the %s connection's login must be recorded", connection)
			assert.Equalf(t, exp, login, "the %s connection's login should record the credentials it logged in with", connection)
		}
	})

	t.Run("a rejected login fails the connection", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{loginCode: "60009"}
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
		server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { srv.serve(t, w, r) }))
		conn, err := ex.Websocket.CreateTestConnection(privateConnection)
		require.NoError(t, err, "CreateTestConnection must not error")
		conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v5/private")
		require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
		t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
		go func() {
			for {
				resp := conn.ReadMessage()
				if resp.Raw == nil {
					return
				}
				if err := ex.wsHandleData(t.Context(), conn, resp.Raw); err != nil {
					return
				}
			}
		}()
		err = ex.wsAuthenticateConnection(t.Context(), conn)
		require.Error(t, err, "a rejected login must error")
		assert.ErrorContains(t, err, "60009", "the error should carry OKX's code")
		ex.websocketLoginsMu.Lock()
		defer ex.websocketLoginsMu.Unlock()
		assert.NotContains(t, ex.websocketLogins, conn, "a rejected login should not be recorded")
	})

	t.Run("credentials are required", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{})
		conn, err := ex.Websocket.CreateTestConnection(privateConnection)
		require.NoError(t, err, "CreateTestConnection must not error")
		assert.Error(t, ex.wsAuthenticateConnection(t.Context(), conn), "a login without credentials should error")
	})
}

// unreportedConnection is a connection which does not report whether it is connected
type unreportedConnection struct {
	websocket.Connection
}

func TestRecordWebsocketLogin(t *testing.T) {
	t.Parallel()
	own := &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"}
	other := &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"}
	ex := newWebsocketTestExchange(t, new(websocketTestServer), true, nil)
	live, err := ex.Websocket.GetConnection(privateConnection)
	require.NoError(t, err, "the private connection must be available")
	closed, err := ex.Websocket.CreateTestConnection(privateConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	unreported := &unreportedConnection{Connection: closed}
	recorded := func(conn websocket.Connection) (websocketLogin, bool) {
		ex.websocketLoginsMu.Lock()
		defer ex.websocketLoginsMu.Unlock()
		login, ok := ex.websocketLogins[conn]
		return login, ok
	}
	ex.recordWebsocketLogin(closed, own)
	_, ok := recorded(closed)
	require.True(t, ok, "a connection's login must be recorded")
	ex.recordWebsocketLogin(unreported, own)
	_, ok = recorded(closed)
	assert.False(t, ok, "a later login should drop the login of a connection which has closed")

	ex.recordWebsocketLogin(live, other)
	_, ok = recorded(unreported)
	assert.True(t, ok, "a connection which does not report its state should keep its login")
	login, ok := recorded(live)
	require.True(t, ok, "the live connection's login must stay recorded")
	assert.Equal(t, newWebsocketLogin(other), login, "a later login should replace the connection's recorded credentials")
	for _, connection := range []string{publicLoginConnection, businessLoginConnection} {
		conn, err := ex.Websocket.GetConnection(connection)
		require.NoErrorf(t, err, "the %s connection must be available", connection)
		login, ok := recorded(conn)
		require.Truef(t, ok, "the %s connection's login must stay recorded while it is connected", connection)
		assert.Equalf(t, newWebsocketLogin(own), login, "the %s connection's login should be unchanged", connection)
	}
}

func TestCheckWebsocketLogin(t *testing.T) {
	t.Parallel()
	own := &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase"}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SkipAuthCheck = true
	ex.SetCredentials(own)
	conn, err := ex.Websocket.CreateTestConnection(privateConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.ErrorIs(t, ex.checkWebsocketLogin(t.Context(), conn), errWebsocketOtherAccount, "checkWebsocketLogin must refuse a connection which has not logged in")
	ex.recordWebsocketLogin(conn, own)
	for _, tc := range []struct {
		name  string
		creds *accounts.Credentials
		err   error
	}{
		{name: "the exchange's own credentials"},
		{name: "the login's credentials", creds: &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub1", OneTimePassword: "123456"}},
		{name: "another key", creds: &accounts.Credentials{Key: "other-key", Secret: "own-secret", ClientID: "own-passphrase"}, err: errWebsocketOtherAccount},
		{name: "another secret", creds: &accounts.Credentials{Key: "own-key", Secret: "other-secret", ClientID: "own-passphrase"}, err: errWebsocketOtherAccount},
		{name: "another passphrase", creds: &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "other-passphrase"}, err: errWebsocketOtherAccount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if tc.creds != nil {
				ctx = accounts.DeployCredentialsToContext(ctx, tc.creds)
			}
			assert.ErrorIs(t, ex.checkWebsocketLogin(ctx, conn), tc.err, "checkWebsocketLogin should accept only the credentials the connection logged in with")
		})
	}
	unusable := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, "unusable")
	assert.ErrorIs(t, ex.checkWebsocketLogin(unusable, conn), common.ErrTypeAssertFailure, "checkWebsocketLogin should report credentials it cannot read")
}

func TestWebsocketLoginCredentials(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn, err := ex.Websocket.CreateTestConnection(privateConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	_, err = ex.websocketLoginCredentials(conn)
	require.ErrorIs(t, err, errUnknownPushAccount, "a connection which has not logged in must name no account")
	own := &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub1", OneTimePassword: "123456"}
	ex.recordWebsocketLogin(conn, own)
	got, err := ex.websocketLoginCredentials(conn)
	require.NoError(t, err, "websocketLoginCredentials must not error")
	assert.Equal(t, own, got, "websocketLoginCredentials should return every credential the connection logged in with")
	got.Key = "changed-key"
	again, err := ex.websocketLoginCredentials(conn)
	require.NoError(t, err, "websocketLoginCredentials must not error")
	assert.Equal(t, own, again, "changing the returned credentials should leave the login's record as it was")
}

func TestSendWebsocketRequest(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	var resp []*OrderActionResponse
	require.ErrorIs(t, ex.sendWebsocketRequest(t.Context(), privateConnection, placeOrderEPL, "", []string{}, &resp), errInvalidWebsocketRequest, "an empty operation must be rejected")
	require.ErrorIs(t, ex.sendWebsocketRequest(t.Context(), privateConnection, placeOrderEPL, "order", nil, &resp), errInvalidWebsocketRequest, "a nil payload must be rejected")
	require.ErrorIs(t, ex.sendWebsocketRequest(t.Context(), privateConnection, placeOrderEPL, "order", []string{}, &resp), websocket.ErrNotConnected, "a disconnected websocket must be reported")

	t.Run("acts only for the account its connection logged in as", func(t *testing.T) {
		t.Parallel()
		other := &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"}
		for _, tc := range []struct {
			name       string
			connection string
			ctx        func(context.Context) context.Context
			rotate     bool
			err        error
		}{
			{name: "the login's account", connection: privateConnection},
			{name: "the login's account on the business connection", connection: businessLoginConnection},
			{name: "another account's context credentials", connection: privateConnection, ctx: func(ctx context.Context) context.Context { return accounts.DeployCredentialsToContext(ctx, other) }, err: errWebsocketOtherAccount},
			{name: "another account's context credentials on the business connection", connection: businessLoginConnection, ctx: func(ctx context.Context) context.Context { return accounts.DeployCredentialsToContext(ctx, other) }, err: errWebsocketOtherAccount},
			{name: "the exchange's credentials changed after the login", connection: privateConnection, rotate: true, err: errWebsocketOtherAccount},
			{name: "the changed credentials in the context", connection: privateConnection, rotate: true, ctx: func(ctx context.Context) context.Context { return accounts.DeployCredentialsToContext(ctx, other) }, err: errWebsocketOtherAccount},
			{name: "a connection which never logged in", connection: publicConnection, err: errWebsocketOtherAccount},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
					return "0", []map[string]string{{"ordId": "12345", "sCode": "0"}}
				}}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				if tc.rotate {
					ex.SetCredentials(other)
				}
				ctx := t.Context()
				if tc.ctx != nil {
					ctx = tc.ctx(ctx)
				}
				var resp []*OrderActionResponse
				err := ex.sendWebsocketRequest(ctx, tc.connection, placeOrderEPL, "order", []PlaceOrderRequest{}, &resp)
				require.ErrorIs(t, err, tc.err, "sendWebsocketRequest must act only for the account the connection logged in as")
				if tc.err != nil {
					assert.Empty(t, srv.requests("order"), "a refused operation should not be sent")
				} else {
					assert.Len(t, srv.requests("order"), 1, "the operation should be sent")
				}
			})
		}
	})

	for _, tc := range []struct {
		name   string
		code   string
		expErr error
		rows   int
	}{
		{name: "success", code: "0", rows: 1},
		{name: "operation failed", code: "1", expErr: errOperationFailed, rows: 1},
		{name: "partial success", code: "2", expErr: errPartialSuccess, rows: 1},
		{name: "rejected", code: "60013", rows: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
				return tc.code, []map[string]string{{"ordId": "12345", "sCode": "51008", "sMsg": "Order failed"}}
			}}
			ex := newWebsocketTestExchange(t, srv, true, nil)
			var resp []*OrderActionResponse
			err := ex.sendWebsocketRequest(t.Context(), privateConnection, placeOrderEPL, "order", []PlaceOrderRequest{}, &resp)
			switch {
			case tc.expErr != nil:
				require.ErrorIs(t, err, tc.expErr, "sendWebsocketRequest must return the reply's outcome")
				assert.ErrorContains(t, err, "Order failed", "a failed row should report its message")
			case tc.code != "0":
				require.ErrorContains(t, err, tc.code, "sendWebsocketRequest must report the reply's code")
			default:
				require.NoError(t, err, "sendWebsocketRequest must not error")
			}
			assert.Len(t, resp, 1, "the reply's rows should decode whatever its code")
		})
	}
}

type mockHasError struct {
	err error
}

func (m *mockHasError) Error() error {
	return m.err
}

func TestParseWebsocketResponseErrors(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { _ = parseWebsocketResponseErrors(123, nil) }, "result must be a pointer")
	require.Panics(t, func() { _ = parseWebsocketResponseErrors(&mockHasError{}, nil) }, "result must be a slice")

	var emptySlice []*mockHasError
	require.NoError(t, parseWebsocketResponseErrors(&emptySlice, nil), "an empty reply without an error must not error")
	require.ErrorIs(t, parseWebsocketResponseErrors(&emptySlice, errOperationFailed), errOperationFailed, "the reply's error must be kept")

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	// A null row in a batch reply decodes to a nil element and is skipped.
	mockSlice := []*mockHasError{nil, {err: nil}, {err: err1}, {err: err2}}
	err := parseWebsocketResponseErrors(&mockSlice, errPartialSuccess)
	require.ErrorIs(t, err, errPartialSuccess, "the reply's error must be kept")
	require.ErrorIs(t, err, err1, "the first row error must be collected")
	require.ErrorIs(t, err, err2, "the second row error must be collected")
}

func TestSingleItem(t *testing.T) {
	t.Parallel()

	_, err := singleItem([]*any(nil))
	require.ErrorIs(t, err, common.ErrNoResponse, "an empty reply must be rejected")
	_, err = singleItem([]*mockHasError{{}, {}})
	require.ErrorIs(t, err, errMultipleItemsReturned, "a reply with more than one row must be rejected")
	_, err = singleItem([]*mockHasError{nil})
	require.ErrorIs(t, err, common.ErrInvalidResponse, "a null row must be rejected")

	got, err := singleItem([]*mockHasError{{}})
	require.NoError(t, err, "a single row must not error")
	require.NotNil(t, got, "the single row must be returned")
}

func TestWebsocketAPI(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.Same(t, ex, ex.WebsocketAPI().e, "WebsocketAPI should operate on its exchange")
}

// subscriptionRecorderConnection records the subscription requests sent on it
// and answers each with what reply returns, or else acknowledges every argument
type subscriptionRecorderConnection struct {
	websocket.Connection
	subscriptions *subscription.Store
	reply         func(req *WebsocketSubscriptionRequest) ([][]byte, error)
	mu            sync.Mutex
	requests      []WebsocketSubscriptionRequest
}

// SendJSONMessage records a subscription request and delivers the replies to
// it, failing the send when reply errors
func (c *subscriptionRecorderConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	req, ok := payload.(*WebsocketSubscriptionRequest)
	if !ok {
		return common.GetTypeAssertError("*WebsocketSubscriptionRequest", payload)
	}
	recorded := *req
	recorded.replies = nil
	c.mu.Lock()
	c.requests = append(c.requests, recorded)
	c.mu.Unlock()
	reply := acknowledgeArguments
	if c.reply != nil {
		reply = c.reply
	}
	replies, err := reply(req)
	if err != nil {
		return err
	}
	for _, r := range replies {
		req.deliver(r)
	}
	return nil
}

func (c *subscriptionRecorderConnection) Subscriptions() *subscription.Store { return c.subscriptions }

// Requests returns a snapshot of recorded subscription requests.
func (c *subscriptionRecorderConnection) Requests() []WebsocketSubscriptionRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.requests)
}

// acknowledgeArguments returns OKX's replies acknowledging each argument of a
// subscription request
func acknowledgeArguments(req *WebsocketSubscriptionRequest) ([][]byte, error) {
	replies := make([][]byte, len(req.Arguments))
	for i := range req.Arguments {
		arg, err := json.Marshal(&req.Arguments[i])
		if err != nil {
			return nil, err
		}
		replies[i] = []byte(`{"id":"` + req.ID + `","event":"` + req.Operation + `","arg":` + string(arg) + `,"connId":"a4d3ae55"}`)
	}
	return replies, nil
}

// rejectArguments returns a subscriptionRecorderConnection reply rejecting the
// arguments at indexes with OKX's error for a channel which does not exist, and
// acknowledging the others
func rejectArguments(indexes ...int) func(*WebsocketSubscriptionRequest) ([][]byte, error) {
	return func(req *WebsocketSubscriptionRequest) ([][]byte, error) {
		replies, err := acknowledgeArguments(req)
		if err != nil {
			return nil, err
		}
		for _, i := range indexes {
			replies[i] = []byte(`{"event":"error","msg":"Subscribe failed, wrong URL or channel doesn't exist.","code":"60018","connId":"a4d3ae55","id":"` + req.ID + `"}`)
		}
		return replies, nil
	}
}

func TestSubscriptionRecorderConnectionRequests(t *testing.T) {
	t.Parallel()

	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	req := &WebsocketSubscriptionRequest{ID: "1512", Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, replies: make(chan []byte, 2)}
	require.NoError(t, conn.SendJSONMessage(t.Context(), 0, req), "SendJSONMessage must not error")
	require.Len(t, req.replies, 1, "SendJSONMessage must deliver a reply for each argument")
	assert.JSONEq(t, `{"id":"1512","event":"subscribe","arg":{"channel":"tickers","instId":"BTC-USDT"},"connId":"a4d3ae55"}`, string(<-req.replies), "each argument should be acknowledged as OKX does")

	requests := conn.Requests()
	require.Equal(t, []WebsocketSubscriptionRequest{{ID: "1512", Operation: operationSubscribe, Arguments: req.Arguments}}, requests, "Requests must return the recorded request without its replies")
	requests[0].Operation = operationUnsubscribe
	require.Equal(t, operationSubscribe, conn.Requests()[0].Operation, "Requests must return an independent slice")

	conn.reply = func(*WebsocketSubscriptionRequest) ([][]byte, error) { return nil, errWebsocketRequestRejected }
	require.ErrorIs(t, conn.SendJSONMessage(t.Context(), 0, req), errWebsocketRequestRejected, "a reply's error must fail the send")
	assert.Empty(t, req.replies, "a failed send should deliver nothing")

	require.ErrorIs(t, conn.SendJSONMessage(t.Context(), 0, *req), common.ErrTypeAssertFailure, "a payload other than a subscription request must be refused")
	assert.Len(t, conn.Requests(), 2, "a refused payload should not be recorded")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()

	t.Run("sends one request and tracks the acknowledged subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		sub := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`}
		require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{sub}), "Subscribe must not error")
		requests := conn.Requests()
		require.Len(t, requests, 1, "Subscribe must send one request")
		assert.Len(t, requests[0].ID, 32, "the request should carry an ID of OKX's maximum length for its replies to echo")
		assert.Equal(t, []WebsocketSubscriptionRequest{{ID: requests[0].ID, Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: perpetualSwapPair}}, subs: subscription.List{sub}}}, requests, "Subscribe should send one subscribe request")
		assert.NotNil(t, ex.Websocket.GetSubscription(sub), "Subscribe should track the subscription")
	})

	t.Run("replies routed over a connection decide what is tracked", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reject: func(_, _ string, arg json.RawMessage) string {
			if strings.Contains(string(arg), `"ETH-USDT"`) {
				return "60018"
			}
			return ""
		}}
		ex := newWebsocketTestExchange(t, srv, false, nil)
		conn, err := ex.Websocket.GetConnection(publicConnection)
		require.NoError(t, err, "GetConnection must not error")
		accepted := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		rejected := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)}, QualifiedChannel: `{"channel":"tickers","instId":"ETH-USDT"}`}
		err = ex.Subscribe(t.Context(), conn, subscription.List{accepted, rejected})
		require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "Subscribe must report the rejected subscription")
		require.ErrorIs(t, err, errWebsocketRequestRejected, "Subscribe must report OKX's rejection")
		assert.ErrorContains(t, err, `subscribe {"channel":"tickers","instId":"ETH-USDT"}`, "the error should name the rejected argument")
		assert.ErrorContains(t, err, `code "60018"`, "the error should carry OKX's code")
		assert.Same(t, accepted, ex.Websocket.GetSubscription(accepted), "the acknowledged subscription should be tracked")
		assert.Equal(t, subscription.SubscribedState, accepted.State(), "the acknowledged subscription should be subscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(rejected), "the rejected subscription should not be tracked")
		frames := srv.requests(operationSubscribe)
		require.Len(t, frames, 1, "Subscribe must send one request")
		assert.JSONEq(t, `[{"channel":"tickers","instId":"BTC-USDT"},{"channel":"tickers","instId":"ETH-USDT"}]`, string(frames[0].arguments), "both arguments should be sent in one request")
	})

	t.Run("an expanded argument naming an instrument type its channel does not take is not sent", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		markets := &subscription.Subscription{Channel: channelEventContractMarkets, QualifiedChannel: `{"channel":"event-contract-markets","instType":"SPOT"}`}
		require.ErrorIs(t, ex.Subscribe(t.Context(), conn, subscription.List{markets}), errSubscriptionParameter, "Subscribe must refuse the instrument type OKX does not take")
		assert.Empty(t, conn.Requests(), "the refused argument should not be sent")
		assert.Nil(t, ex.Websocket.GetSubscription(markets), "the refused subscription should not be tracked")
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()

	t.Run("sends one request and stops tracking the acknowledged subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		sub := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, ex.Unsubscribe(t.Context(), conn, subscription.List{sub}), "Unsubscribe must not error")
		requests := conn.Requests()
		require.Len(t, requests, 1, "Unsubscribe must send one request")
		assert.NotEmpty(t, requests[0].ID, "the request should carry an ID for its replies to echo")
		assert.Equal(t, []WebsocketSubscriptionRequest{{ID: requests[0].ID, Operation: operationUnsubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: perpetualSwapPair}}, subs: subscription.List{sub}}}, requests, "Unsubscribe should send one unsubscribe request")
		assert.Nil(t, ex.Websocket.GetSubscription(sub), "Unsubscribe should stop tracking the subscription")
	})

	t.Run("an unacknowledged unsubscription stays tracked", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore(), reply: rejectArguments(0)}
		sub := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		err := ex.Unsubscribe(t.Context(), conn, subscription.List{sub})
		require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "Unsubscribe must report the unsubscription OKX refused")
		require.ErrorIs(t, err, errWebsocketRequestRejected, "Unsubscribe must report OKX's rejection")
		assert.Same(t, sub, ex.Websocket.GetSubscription(sub), "a subscription OKX did not unsubscribe should stay tracked")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "a subscription OKX did not unsubscribe should stay subscribed")
	})
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{channelInstruments, channelLiquidationOrders, channelOrders, subscription.MyOrdersChannel} {
		for _, batch := range []bool{false, true} {
			for _, withPairs := range []bool{false, true} {
				for _, first := range []asset.Item{asset.Spot, asset.Margin} {
					t.Run(fmt.Sprintf("%s/batch=%t/pairs=%t/first=%s", channel, batch, withPairs, first), func(t *testing.T) {
						t.Parallel()
						tracked := new(Exchange)
						require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
						second := asset.Margin
						if first == asset.Margin {
							second = asset.Spot
						}
						name, err := channelName(&subscription.Subscription{Channel: channel}, first)
						require.NoError(t, err, "channelName must not error")
						var subs subscription.List
						for _, a := range []asset.Item{first, second} {
							sub := &subscription.Subscription{
								Channel: channel, Asset: a,
								QualifiedChannel: fmt.Sprintf(`{"channel":%q,"instType":%q}`, name, GetInstrumentTypeFromAssetItem(a)),
							}
							if withPairs {
								sub.Pairs = currency.Pairs{currency.NewBTCUSDT()}
							}
							subs = append(subs, sub)
						}
						conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
						for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
							before := len(conn.Requests())
							if batch {
								require.NoError(t, tracked.handleSubscription(t.Context(), conn, operation, subs), "both assets must be sent together")
							} else {
								for _, sub := range subs {
									require.NoError(t, tracked.handleSubscription(t.Context(), conn, operation, subscription.List{sub}), "each asset must be sent independently")
								}
							}
							var arguments []SubscriptionArgument
							for _, req := range conn.Requests()[before:] {
								require.Equal(t, operation, req.Operation, "outbound operation must match")
								arguments = append(arguments, req.Arguments...)
							}
							require.Equal(t, []SubscriptionArgument{
								{Channel: name, InstrumentType: GetInstrumentTypeFromAssetItem(first)},
								{Channel: name, InstrumentType: GetInstrumentTypeFromAssetItem(second)},
							}, arguments, "outbound requests must retain both instrument types")
							for _, sub := range subs {
								if operation == operationSubscribe {
									require.NotNil(t, tracked.Websocket.GetSubscription(sub), "sent subscriptions must be tracked")
								} else {
									require.Nil(t, tracked.Websocket.GetSubscription(sub), "unsubscribed assets must be removed")
								}
							}
						}
					})
				}
			}
		}
	}

	t.Run("replies", func(t *testing.T) {
		t.Parallel()
		const swapTickers = `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`
		reply := func(event, arg string) func(*WebsocketSubscriptionRequest) ([][]byte, error) {
			return func(req *WebsocketSubscriptionRequest) ([][]byte, error) {
				return [][]byte{[]byte(`{"id":"` + req.ID + `","event":"` + event + `","arg":` + arg + `,"connId":"a4d3ae55"}`)}, nil
			}
		}
		for _, tc := range []struct {
			name    string
			sub     *subscription.Subscription
			reply   func(*WebsocketSubscriptionRequest) ([][]byte, error)
			expErrs []error
			expWait time.Duration
		}{
			{
				name:  "acknowledgement",
				sub:   &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply: reply(operationSubscribe, swapTickers),
			},
			{
				name:  "acknowledgement without the configuration or user ID",
				sub:   &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account","extraParams":"{\"updateInterval\":\"0\"}"}`},
				reply: reply(operationSubscribe, `{"channel":"account","uid":"44*********584"}`),
			},
			{
				name:    "rejection",
				sub:     &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply:   rejectArguments(0),
				expErrs: []error{errSubscriptionNotAcknowledged, errWebsocketRequestRejected},
			},
			{
				name:    "no reply within the response limit",
				sub:     &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply:   func(*WebsocketSubscriptionRequest) ([][]byte, error) { return nil, nil },
				expErrs: []error{errSubscriptionNotAcknowledged, websocket.ErrSignatureTimeout},
				expWait: websocketResponseMaxLimit,
			},
			{
				name:    "a request which cannot be sent",
				sub:     &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply:   func(*WebsocketSubscriptionRequest) ([][]byte, error) { return nil, common.ErrFatal },
				expErrs: []error{errSubscriptionNotAcknowledged, common.ErrFatal},
			},
			{
				name:    "acknowledgement of another argument",
				sub:     &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply:   reply(operationSubscribe, `{"channel":"tickers","instId":"ETH-USDT-SWAP"}`),
				expErrs: []error{errSubscriptionNotAcknowledged},
			},
			{
				name:    "another operation's event",
				sub:     &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply:   reply(operationUnsubscribe, swapTickers),
				expErrs: []error{errSubscriptionNotAcknowledged},
			},
			{
				name: "a reply which does not decode",
				sub:  &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: swapTickers},
				reply: func(*WebsocketSubscriptionRequest) ([][]byte, error) {
					return [][]byte{[]byte(`{"event":`)}, nil
				},
				expErrs: []error{errSubscriptionNotAcknowledged},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				synctest.Test(t, func(t *testing.T) {
					conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore(), reply: tc.reply}
					start := time.Now()
					err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{tc.sub})
					assert.Equal(t, tc.expWait, time.Since(start), "handleSubscription should wait for a missing reply for the response limit alone")
					require.Len(t, conn.Requests(), 1, "handleSubscription must send one request")
					if len(tc.expErrs) == 0 {
						require.NoError(t, err, "handleSubscription must not error")
						assert.Same(t, tc.sub, ex.Websocket.GetSubscription(tc.sub), "an acknowledged subscription should be tracked")
						return
					}
					for _, expErr := range tc.expErrs {
						require.ErrorIs(t, err, expErr, "handleSubscription must report the subscription OKX did not acknowledge")
					}
					assert.Nil(t, ex.Websocket.GetSubscription(tc.sub), "an unacknowledged subscription should not be tracked")
				})
			})
		}
	})

	t.Run("only acknowledged arguments are tracked", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore(), reply: rejectArguments(1)}
		accepted := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`}
		rejected := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDT-SWAP", currency.DashDelimiter)}, QualifiedChannel: `{"channel":"tickers","instId":"ETH-USDT-SWAP"}`}
		err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{accepted, rejected})
		require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "handleSubscription must report the rejected subscription")
		require.ErrorIs(t, err, errWebsocketRequestRejected, "handleSubscription must report OKX's rejection")
		assert.ErrorContains(t, err, `subscribe {"channel":"tickers","instId":"ETH-USDT-SWAP"}`, "the error should name the rejected argument")
		assert.NotContains(t, err.Error(), "BTC-USDT-SWAP", "the error should not name the acknowledged argument")
		assert.Len(t, conn.Requests(), 1, "both subscriptions should be sent in one request")
		assert.Same(t, accepted, ex.Websocket.GetSubscription(accepted), "the acknowledged subscription should be tracked")
		assert.Nil(t, ex.Websocket.GetSubscription(rejected), "the rejected subscription should not be tracked")
	})

	t.Run("spot and margin follow their shared argument", func(t *testing.T) {
		t.Parallel()
		for _, rejected := range []bool{false, true} {
			t.Run(fmt.Sprintf("rejected=%t", rejected), func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				if rejected {
					conn.reply = rejectArguments(0)
				}
				spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
				margin := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: spot.QualifiedChannel}
				err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{spot, margin})
				requests := conn.Requests()
				require.Len(t, requests, 1, "handleSubscription must send one request")
				require.Equal(t, []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, requests[0].Arguments, "spot and margin must share one argument")
				for _, sub := range []*subscription.Subscription{spot, margin} {
					if rejected {
						assert.Nilf(t, ex.Websocket.GetSubscription(sub), "the %s subscription should not be tracked when its argument is rejected", sub.Asset)
					} else {
						assert.Samef(t, sub, ex.Websocket.GetSubscription(sub), "the %s subscription should be tracked when its argument is acknowledged", sub.Asset)
					}
				}
				if rejected {
					require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "handleSubscription must report both subscriptions")
					return
				}
				require.NoError(t, err, "handleSubscription must not error")
			})
		}
	})

	t.Run("an equivalent of a tracked subscription needs no request", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, spot), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(spot), "the connection must hold the spot subscription")
		margin := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: spot.QualifiedChannel}
		require.NoError(t, ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{margin}), "handleSubscription must not error")
		assert.Empty(t, conn.Requests(), "a subscription sharing a tracked equivalent's should not be sent")
		assert.Same(t, margin, ex.Websocket.GetSubscription(margin), "a subscription sharing a tracked equivalent's should be tracked")
	})

	t.Run("replies received within the response limit decide their arguments", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		tickers := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		trades := &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"trades","instId":"BTC-USDT"}`}
		books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		synctest.Test(t, func(t *testing.T) {
			conn := dialWebsocketTestServer(t, ex, publicConnection, func(id, operation string, arguments []json.RawMessage) []string {
				return []string{subscriptionReply(id, operation, arguments[0]), `{"id":"` + id + `","event":"error","code":"60012","msg":"Invalid request","connId":"a4d3ae55"}`}
			})
			start := time.Now()
			err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{tickers, trades, books})
			assert.Equal(t, websocketResponseMaxLimit, time.Since(start), "the missing reply should be awaited for the response limit")
			require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "the arguments without an acknowledgement must be reported")
			require.ErrorIs(t, err, errWebsocketRequestRejected, "the rejection received must be reported")
			require.ErrorIs(t, err, websocket.ErrSignatureTimeout, "the missing reply must be reported")
			assert.ErrorContains(t, err, `subscribe {"channel":"trades","instId":"BTC-USDT"}, {"channel":"books","instId":"BTC-USDT"}`, "the error should name the arguments without an acknowledgement")
			assert.ErrorContains(t, err, `code "60012"`, "the error should carry the rejection's code")
			assert.Same(t, tickers, ex.Websocket.GetSubscription(tickers), "the acknowledged subscription should be tracked")
			assert.Nil(t, ex.Websocket.GetSubscription(trades), "the rejected subscription should not be tracked")
			assert.Nil(t, ex.Websocket.GetSubscription(books), "the unanswered subscription should not be tracked")
		})
	})

	t.Run("arguments acknowledged alike are sent in separate requests", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		account := &subscription.Subscription{Channel: subscription.MyAccountChannel, Params: map[string]any{"ccy": "USDT"}, QualifiedChannel: `{"channel":"account","ccy":"USDT"}`}
		configured := &subscription.Subscription{Channel: channelAccount, Params: map[string]any{"ccy": "USDT", "extraParams": `{"updateInterval":"0"}`}, QualifiedChannel: `{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}`}
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore(), reply: func(req *WebsocketSubscriptionRequest) ([][]byte, error) {
			replies := make([][]byte, len(req.Arguments))
			for i := range req.Arguments {
				replies[i] = []byte(`{"id":"` + req.ID + `","event":"subscribe","arg":{"channel":"account","ccy":"USDT"},"connId":"a4d3ae55"}`)
				if req.Arguments[i].ExtraParams != "" {
					replies[i] = []byte(`{"id":"` + req.ID + `","event":"error","code":"60012","msg":"Invalid request","connId":"a4d3ae55"}`)
				}
			}
			return replies, nil
		}}
		err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{account, configured})
		require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "the rejected configuration must be reported")
		require.ErrorIs(t, err, errWebsocketRequestRejected, "OKX's rejection must be reported")
		sent := make([][]SubscriptionArgument, 0, len(conn.Requests()))
		for _, req := range conn.Requests() {
			sent = append(sent, req.Arguments)
		}
		exp := [][]SubscriptionArgument{
			{{Channel: channelAccount, Currency: currency.USDT}},
			{{Channel: channelAccount, Currency: currency.USDT, ExtraParams: `{"updateInterval":"0"}`}},
		}
		assert.Equal(t, exp, sent, "each configuration should be sent in a request of its own")
		assert.Same(t, account, ex.Websocket.GetSubscription(account), "the acknowledged configuration should be tracked")
		assert.Nil(t, ex.Websocket.GetSubscription(configured), "the rejected configuration should not be tracked")
	})

	t.Run("an argument a subscription on the connection keeps is not unsubscribed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		margin := &subscription.Subscription{Channel: channelTickers, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: spot.QualifiedChannel}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, spot, margin), "AddSuccessfulSubscriptions must not error")
		for _, s := range []*subscription.Subscription{spot, margin} {
			require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold both subscriptions")
		}
		require.NoError(t, ex.handleSubscription(t.Context(), conn, operationUnsubscribe, subscription.List{spot}), "handleSubscription must not error")
		assert.Empty(t, conn.Requests(), "the argument the margin subscription keeps should not be unsubscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(spot), "the spot subscription should no longer be tracked")
		assert.Same(t, margin, ex.Websocket.GetSubscription(margin), "the margin subscription should stay tracked")

		require.NoError(t, conn.Subscriptions().Remove(spot), "the connection must release the spot subscription")
		require.NoError(t, ex.handleSubscription(t.Context(), conn, operationUnsubscribe, subscription.List{margin}), "handleSubscription must not error")
		requests := conn.Requests()
		require.Len(t, requests, 1, "the last subscription to the argument must unsubscribe it")
		assert.Equal(t, []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, requests[0].Arguments, "the shared argument should be unsubscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(margin), "the margin subscription should no longer be tracked")
	})

	t.Run("the last subscription to an argument removed by an equivalent unsubscribes it", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}}}
		tracked, err := ex.generateSubscriptions(t.Context(), publicConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		require.Len(t, tracked, 1, "generateSubscriptions must generate the ticker subscription")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, tracked[0]), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(tracked[0]), "the connection must hold the subscription")
		equivalent, err := ex.generateSubscriptions(t.Context(), publicConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		require.NotSame(t, tracked[0], equivalent[0], "the equivalent must be another subscription")
		require.NoError(t, ex.handleSubscription(t.Context(), conn, operationUnsubscribe, equivalent), "handleSubscription must not error")
		requests := conn.Requests()
		for i := range requests {
			requests[i].ID = ""
		}
		exp := []WebsocketSubscriptionRequest{{Operation: operationUnsubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, subs: equivalent}}
		assert.Equal(t, exp, requests, "the argument should be unsubscribed once its last subscription goes")
		assert.Nil(t, ex.Websocket.GetSubscription(tracked[0]), "the subscription should no longer be tracked")
	})

	t.Run("a request at the frame limit reaches the peer whole", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		var mu sync.Mutex
		var frames []int
		srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
				mu.Lock()
				frames = append(frames, len(msg))
				mu.Unlock()
				var req struct {
					ID        string            `json:"id"`
					Operation string            `json:"op"`
					Arguments []json.RawMessage `json:"args"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				if len(msg) > maxConnByteLen {
					return c.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","event":"error","code":"60012","msg":"Invalid request","connId":"a4d3ae55"}`))
				}
				for _, arg := range req.Arguments {
					if err := c.WriteMessage(gws.TextMessage, []byte(subscriptionReply(req.ID, req.Operation, arg))); err != nil {
						return err
					}
				}
				return nil
			})
		}))
		conn, err := ex.Websocket.CreateTestConnection(privateConnection)
		require.NoError(t, err, "CreateTestConnection must not error")
		conn.SetURL("ws" + strings.TrimPrefix(srv.URL, "http"))
		require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
		read := make(chan struct{})
		go func() {
			defer close(read)
			for {
				resp := conn.ReadMessage()
				if resp.Raw == nil {
					return
				}
				assert.NoError(t, ex.wsHandleData(t.Context(), conn, resp.Raw), "wsHandleData should not error")
			}
		}()
		t.Cleanup(func() {
			assert.NoError(t, conn.Shutdown(), "Shutdown should not error")
			select {
			case <-read:
			case <-time.After(5 * time.Second):
				assert.Fail(t, "the reader should stop once the connection closes")
			}
		})
		greeks, account := SubscriptionArgument{Channel: channelAccountGreeks}, SubscriptionArgument{Channel: channelAccount, ExtraParams: `{"updateInterval":"0"}`}
		encoded, err := json.Marshal(&WebsocketSubscriptionRequest{ID: ex.MessageID(), Operation: operationSubscribe, Arguments: []SubscriptionArgument{greeks, account}})
		require.NoError(t, err, "the request must encode")
		account.ExtraParams = `{"updateInterval":` + strings.Repeat(" ", maxConnByteLen-len(encoded)) + `"0"}`
		subs := make(subscription.List, 0, 2)
		for _, arg := range []SubscriptionArgument{greeks, account} {
			qualifiedChannel, err := json.Marshal(&arg)
			require.NoError(t, err, "the argument must encode")
			subs = append(subs, &subscription.Subscription{Channel: arg.Channel, QualifiedChannel: string(qualifiedChannel)})
		}
		require.NoError(t, ex.handleSubscription(t.Context(), conn, operationSubscribe, subs), "handleSubscription must not error")
		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, frames, "the peer must receive the subscriptions")
		for _, size := range frames {
			assert.LessOrEqual(t, size, maxConnByteLen, "each frame the peer receives should fit the frame limit")
		}
	})

	t.Run("an ended context sends nothing", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		sub := &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`}
		require.ErrorIs(t, ex.handleSubscription(ctx, conn, operationSubscribe, subscription.List{sub}), context.Canceled, "handleSubscription must return the context's error")
		assert.Empty(t, conn.Requests(), "nothing should be sent once the context has ended")
		assert.Nil(t, ex.Websocket.GetSubscription(sub), "nothing should be tracked once the context has ended")
	})
}

func TestSubscriptionRequestsSend(t *testing.T) {
	t.Parallel()
	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	first := &WebsocketSubscriptionRequest{ID: "1", Operation: operationSubscribe}
	second := &WebsocketSubscriptionRequest{ID: "2", Operation: operationUnsubscribe}
	var r subscriptionRequests
	r.send(conn, first)
	r.send(nil, second)
	exp := map[string]*sentSubscriptionRequest{"1": {conn: conn, request: first, awaiting: true}, "2": {request: second, awaiting: true}}
	assert.Equal(t, exp, r.sent, "send should record each request as awaiting its replies on its connection")
}

func TestSubscriptionRequestsRoute(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		room        int
		stopped     bool
		id          string
		exp         bool
		expReceived int
	}{
		{name: "a request awaiting the reply", room: 1, id: "1", exp: true, expReceived: 1},
		{name: "a request holding a reply for each argument", id: "1"},
		{name: "a request no longer awaiting replies", room: 1, stopped: true, id: "1"},
		{name: "no request with the ID", room: 1, id: "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var r subscriptionRequests
			req := &WebsocketSubscriptionRequest{ID: "1", replies: make(chan []byte, tc.room)}
			r.send(nil, req)
			if tc.stopped {
				r.stopAwaiting(req.ID)
			}
			assert.Equal(t, tc.exp, r.route(tc.id, []byte("reply")), "route should report whether a request took the reply")
			assert.Len(t, req.replies, tc.expReceived, "route should pass the reply only to the request awaiting it")
		})
	}
}

func TestSubscriptionRequestsStopAwaiting(t *testing.T) {
	t.Parallel()
	var r subscriptionRequests
	req := &WebsocketSubscriptionRequest{ID: "1", replies: make(chan []byte, 1)}
	r.send(nil, req)
	r.stopAwaiting("2")
	assert.True(t, r.sent["1"].awaiting, "stopping another request should leave the request awaiting")
	r.stopAwaiting("1")
	assert.Equal(t, map[string]*sentSubscriptionRequest{"1": {request: req}}, r.sent, "stopAwaiting should keep the request recorded without awaiting replies")
}

func TestSubscriptionRequestsSettle(t *testing.T) {
	t.Parallel()
	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	var r subscriptionRequests
	r.send(conn, &WebsocketSubscriptionRequest{ID: "1", Operation: operationSubscribe})
	r.send(conn, &WebsocketSubscriptionRequest{ID: "2", Operation: operationSubscribe})
	r.end(conn, channelPositions, errChannelConnectionLimit)
	assert.Equal(t, map[string]error{channelPositions: errChannelConnectionLimit}, r.settle("1"), "settle should return the channels OKX ended since the request was sent")
	assert.NotContains(t, r.sent, "1", "settle should forget the request")
	assert.Contains(t, r.sent, "2", "settle should keep the other requests")
	assert.Nil(t, r.settle("1"), "settling a forgotten request should return nothing")
}

func TestSubscriptionRequestsEnd(t *testing.T) {
	t.Parallel()
	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	other := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	var r subscriptionRequests
	r.send(conn, &WebsocketSubscriptionRequest{ID: "subscribe", Operation: operationSubscribe})
	r.send(conn, &WebsocketSubscriptionRequest{ID: "unsubscribe", Operation: operationUnsubscribe})
	r.send(other, &WebsocketSubscriptionRequest{ID: "other", Operation: operationSubscribe})
	positionsErr, accountErr := fmt.Errorf("%w: positions", errChannelConnectionLimit), fmt.Errorf("%w: account", errChannelConnectionLimit)
	r.end(conn, channelPositions, positionsErr)
	r.end(conn, channelAccount, accountErr)
	ended := map[string]map[string]error{}
	for id, sent := range r.sent {
		ended[id] = sent.ended
	}
	exp := map[string]map[string]error{"subscribe": {channelPositions: positionsErr, channelAccount: accountErr}, "unsubscribe": nil, "other": nil}
	assert.Equal(t, exp, ended, "end should record the ended channels for the subscribe requests sent on the connection alone")
}

func TestSubscriptionRequestsSubscribing(t *testing.T) {
	t.Parallel()
	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	other := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
	tickers := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
	marginBooks := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: books.QualifiedChannel}
	var r subscriptionRequests
	r.send(conn, &WebsocketSubscriptionRequest{ID: "1", Operation: operationSubscribe, subs: subscription.List{books, tickers}})
	r.send(conn, &WebsocketSubscriptionRequest{ID: "2", Operation: operationUnsubscribe, subs: subscription.List{marginBooks}})
	r.send(other, &WebsocketSubscriptionRequest{ID: "3", Operation: operationSubscribe, subs: subscription.List{marginBooks}})
	isBooks := func(s *subscription.Subscription) bool { return s.QualifiedChannel == books.QualifiedChannel }
	assert.Equal(t, subscription.List{books}, r.subscribing(conn, isBooks), "subscribing should return the matching subscriptions being subscribed on the connection")
	assert.Empty(t, r.subscribing(nil, isBooks), "subscribing should return nothing for a connection without requests")
}

func TestSubscriptionRequestsRequest(t *testing.T) {
	t.Parallel()
	var r subscriptionRequests
	r.request([]string{"a", "b"})
	r.request([]string{"a"})
	assert.Equal(t, map[string]int{"a": 2, "b": 1}, r.requestedArguments, "request should count each argument")
	assert.EqualValues(t, 3, r.requested.Load(), "request should count every argument requested")
}

func TestSubscriptionRequestsRelease(t *testing.T) {
	t.Parallel()
	var r subscriptionRequests
	r.request([]string{"a", "b"})
	r.request([]string{"a"})
	r.release([]string{"a", "b"})
	assert.Equal(t, map[string]int{"a": 1}, r.requestedArguments, "release should forget an argument once no request counts it")
	assert.EqualValues(t, 1, r.requested.Load(), "release should uncount the released arguments")
}

func TestSubscriptionRequestsIsRequested(t *testing.T) {
	t.Parallel()
	var r subscriptionRequests
	assert.False(t, r.isRequested("a"), "no argument should be requested before any request")
	r.request([]string{"a"})
	assert.True(t, r.isRequested("a"), "a requested argument should be requested")
	assert.False(t, r.isRequested("b"), "another argument should not be requested")
	r.release([]string{"a"})
	assert.False(t, r.isRequested("a"), "a released argument should not be requested")
}

func TestWebsocketSubscriptionRequestDeliver(t *testing.T) {
	t.Parallel()
	assert.False(t, (&WebsocketSubscriptionRequest{}).deliver([]byte("reply")), "a request not awaiting replies should refuse one")
	req := &WebsocketSubscriptionRequest{replies: make(chan []byte, 1)}
	assert.True(t, req.deliver([]byte("first")), "a request with room should take a reply")
	assert.False(t, req.deliver([]byte("second")), "a request holding a reply for each argument should refuse another")
	require.Len(t, req.replies, 1, "the request must hold one reply")
	assert.Equal(t, []byte("first"), <-req.replies, "the request should hold the reply it took")
}

func TestEndedSubscriptions(t *testing.T) {
	t.Parallel()
	positions := &subscription.Subscription{Channel: channelPositions, QualifiedChannel: `{"channel":"positions","instType":"SWAP"}`}
	account := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account"}`}
	ended, err := endedSubscriptions(subscription.List{positions, account}, nil)
	require.NoError(t, err, "endedSubscriptions must not error when no channel ended")
	assert.Empty(t, ended, "no subscription should end when no channel ended")
	ended, err = endedSubscriptions(subscription.List{positions, account}, map[string]error{channelPositions: errChannelConnectionLimit})
	require.ErrorIs(t, err, errChannelConnectionLimit, "endedSubscriptions must return the error ending the channel")
	assert.ErrorContains(t, err, `ended {"channel":"positions","instType":"SWAP"}`, "the error should name the ended subscription")
	assert.Equal(t, subscription.List{positions}, ended, "the subscriptions to the ended channel alone should end")
}

// lateReplyContext is an ended context which routes late replies as its error
// is read: after a wait for replies has ended but before routing to it stops
type lateReplyContext struct {
	done  chan struct{}
	once  sync.Once
	route func()
}

func newLateReplyContext(route func()) *lateReplyContext {
	done := make(chan struct{})
	close(done)
	return &lateReplyContext{done: done, route: route}
}

func (c *lateReplyContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (c *lateReplyContext) Done() <-chan struct{} { return c.done }

// Err routes the late replies once and returns context.Canceled
func (c *lateReplyContext) Err() error {
	c.once.Do(c.route)
	return context.Canceled
}

func (c *lateReplyContext) Value(any) any { return nil }

func TestAwaitSubscriptionReplies(t *testing.T) {
	t.Parallel()
	ack := func(id string) []byte {
		return []byte(`{"id":"` + id + `","event":"subscribe","arg":{"channel":"tickers","instId":"BTC-USDT"},"connId":"a4d3ae55"}`)
	}
	for _, tc := range []struct {
		name      string
		delivered int
		late      int
		cancel    bool
		sendErr   error
		expErr    error
		expWait   time.Duration
	}{
		{name: "a reply for each argument", delivered: 2},
		{name: "a missing reply", delivered: 1, expErr: websocket.ErrSignatureTimeout, expWait: websocketResponseMaxLimit},
		{name: "an ended context", cancel: true, expErr: context.Canceled},
		{name: "an ended context keeps the replies received", cancel: true, delivered: 1, expErr: context.Canceled},
		{name: "an ended context after a reply for each argument", cancel: true, delivered: 2},
		{name: "replies arriving as the wait ends are kept", late: 2},
		{name: "a request which cannot be sent", sendErr: common.ErrFatal, expErr: common.ErrFatal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				req := &WebsocketSubscriptionRequest{ID: "1512", Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}, {Channel: channelTrades, InstrumentID: mainPair}}}
				waitCtx := ctx
				if tc.late != 0 {
					waitCtx = newLateReplyContext(func() {
						for range tc.late {
							assert.True(t, ex.subscriptionRequests.route(req.ID, ack(req.ID)), "a late reply should be routed to the request")
						}
					})
				}
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore(), reply: func(req *WebsocketSubscriptionRequest) ([][]byte, error) {
					if tc.cancel {
						cancel()
					}
					replies := make([][]byte, tc.delivered)
					for i := range replies {
						replies[i] = ack(req.ID)
					}
					return replies, tc.sendErr
				}}
				start := time.Now()
				replies, err := ex.awaitSubscriptionReplies(waitCtx, conn, req)
				assert.Equal(t, tc.expWait, time.Since(start), "awaitSubscriptionReplies should wait for a missing reply for the response limit alone")
				require.ErrorIs(t, err, tc.expErr, "awaitSubscriptionReplies must return the expected error")
				expReplies := tc.delivered + tc.late
				if tc.sendErr != nil {
					expReplies = 0
				}
				require.Len(t, replies, expReplies, "awaitSubscriptionReplies must return the replies received")
				for _, reply := range replies {
					assert.Equal(t, ack(req.ID), reply, "awaitSubscriptionReplies should return each reply as received")
				}
				assert.False(t, ex.subscriptionRequests.route(req.ID, ack(req.ID)), "the request should no longer await replies")
				assert.Equal(t, map[string]*sentSubscriptionRequest{req.ID: {conn: conn, request: req}}, ex.subscriptionRequests.sent, "the request should stay recorded until it is settled")
			})
		})
	}
}

func TestEndChannelSubscriptions(t *testing.T) {
	t.Parallel()
	positions := func(a asset.Item, instType string) *subscription.Subscription {
		return &subscription.Subscription{Channel: channelPositions, Asset: a, QualifiedChannel: `{"channel":"positions","instType":"` + instType + `"}`}
	}
	t.Run("no connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.Same(t, errChannelConnectionLimit, ex.endChannelSubscriptions(nil, channelPositions, errChannelConnectionLimit), "endChannelSubscriptions should return the error alone without a connection")
	})

	t.Run("the tracked subscriptions to the channel on the connection end", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(privateConnection, conn), "TrackTestConnection must not error")
		swap, untracked := positions(asset.PerpetualSwap, instTypeSwap), positions(asset.Futures, instTypeFutures)
		orders := &subscription.Subscription{Channel: channelOrders, Asset: asset.PerpetualSwap, QualifiedChannel: `{"channel":"orders","instType":"SWAP"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, swap, orders), "AddSuccessfulSubscriptions must not error")
		for _, s := range []*subscription.Subscription{swap, untracked, orders} {
			require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold the subscription")
		}
		err := ex.endChannelSubscriptions(conn, channelPositions, errChannelConnectionLimit)
		require.ErrorIs(t, err, errChannelConnectionLimit, "endChannelSubscriptions must return the error ending the channel")
		assert.ErrorContains(t, err, `ended {"channel":"positions","instType":"SWAP"}`, "the error should name the ended subscription")
		assert.NotContains(t, err.Error(), "FUTURES", "the error should not name a subscription the websocket does not track")
		assert.Nil(t, ex.Websocket.GetSubscription(swap), "the ended subscription should no longer be tracked")
		assert.Equal(t, subscription.UnsubscribedState, swap.State(), "the ended subscription should be unsubscribed")
		assert.ElementsMatch(t, subscription.List{untracked, orders}, conn.Subscriptions().List(), "the connection should release the ended subscription alone")
	})

	t.Run("a subscription to the channel being subscribed is marked ended", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		ex.subscriptionRequests.send(conn, &WebsocketSubscriptionRequest{ID: "1", Operation: operationSubscribe, subs: subscription.List{positions(asset.PerpetualSwap, instTypeSwap)}})
		assert.Same(t, errChannelConnectionLimit, ex.endChannelSubscriptions(conn, channelPositions, errChannelConnectionLimit), "endChannelSubscriptions should return the error alone when nothing tracked ended")
		assert.Equal(t, map[string]error{channelPositions: errChannelConnectionLimit}, ex.subscriptionRequests.settle("1"), "the request being answered should record the ended channel")
	})
}

func TestChunkRequests(t *testing.T) {
	t.Parallel()

	t.Run("carries the instrument of every pair subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		subs, err := ex.generateSubscriptions(t.Context(), publicConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		var checked int
		for _, sub := range subs {
			if len(sub.Pairs) == 0 {
				continue
			}
			reqs, err := ex.chunkRequests(nil, subscription.List{sub}, operationSubscribe)
			require.NoError(t, err, "chunkRequests must not error")
			for _, req := range reqs {
				for _, arg := range req.Arguments {
					require.Falsef(t, arg.InstrumentID.IsEmpty() && arg.SpreadID.IsEmpty(), "%s must keep its instrument through the template round-trip; got %q", sub.Channel, sub.QualifiedChannel)
					checked++
				}
			}
		}
		require.Positive(t, checked, "must exercise at least one pair specific subscription")
	})

	t.Run("splits requests above the byte limit", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		subs := make(subscription.List, 1500)
		for i := range subs {
			pair := currency.NewPairWithDelimiter(fmt.Sprintf("TEST%d", i), "USDT-SWAP", currency.DashDelimiter)
			subs[i] = &subscription.Subscription{Channel: channelTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{pair}, QualifiedChannel: `{"channel":"tickers","instId":"` + pair.String() + `"}`}
		}
		reqs, err := ex.chunkRequests(nil, subs, operationSubscribe)
		require.NoError(t, err, "chunkRequests must not error")
		require.Len(t, reqs, 2, "1500 subscriptions must exceed one request's 64 KiB")
		var total int
		for _, req := range reqs {
			assert.Len(t, req.ID, 32, "a request should carry an ID for its replies to echo")
			b, err := json.Marshal(req)
			require.NoError(t, err, "a request must encode")
			assert.LessOrEqual(t, len(b), maxConnByteLen, "a request should fit the byte limit with its ID")
			assert.Len(t, req.subs, len(req.Arguments), "a request should track each subscription it sends")
			total += len(req.Arguments)
		}
		assert.Equal(t, len(subs), total, "every subscription should be sent once")
		assert.NotEqual(t, reqs[0].ID, reqs[1].ID, "each request should carry its own ID")
	})

	t.Run("sends an argument for each distinct subscription", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			subs subscription.List
			exp  []SubscriptionArgument
		}{
			{
				name: "candle intervals of one instrument",
				subs: subscription.List{
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.FiveMin},
				},
				exp: []SubscriptionArgument{{Channel: "candle1m", InstrumentID: mainPair}, {Channel: "candle5m", InstrumentID: mainPair}},
			},
			{
				name: "spot and margin candles of one interval",
				subs: subscription.List{
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
				},
				exp: []SubscriptionArgument{{Channel: "candle1m", InstrumentID: mainPair}},
			},
			{
				name: "spot and margin candles of different intervals",
				subs: subscription.List{
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
					{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, Interval: kline.FiveMin},
				},
				exp: []SubscriptionArgument{{Channel: "candle1m", InstrumentID: mainPair}, {Channel: "candle5m", InstrumentID: mainPair}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Features.Subscriptions = tc.subs
				subs, err := ex.generateSubscriptions(t.Context(), businessConnection)
				require.NoError(t, err, "generateSubscriptions must not error")
				require.Len(t, subs, len(tc.subs), "each configured subscription must be generated")
				reqs, err := ex.chunkRequests(nil, subs, operationSubscribe)
				require.NoError(t, err, "chunkRequests must not error")
				require.Len(t, reqs, 1, "the subscriptions must fit one request")
				assert.Equal(t, tc.exp, reqs[0].Arguments, "each distinct argument should be sent once")
				assert.Equal(t, subs, reqs[0].subs, "the request should track every subscription")
			})
		}
	})

	t.Run("rejects an unexpanded subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		_, err := ex.chunkRequests(nil, subscription.List{{Channel: channelTickers}}, operationSubscribe)
		assert.Error(t, err, "a subscription without a qualified channel should not encode")
	})

	t.Run("an expanded argument without what its channel requires is not subscribed", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name             string
			qualifiedChannel string
			expErr           error
		}{
			{name: "event contract markets of another instrument type", qualifiedChannel: `{"channel":"event-contract-markets","instType":"SPOT"}`, expErr: errSubscriptionParameter},
			{name: "event contract markets without an instrument type", qualifiedChannel: `{"channel":"event-contract-markets"}`, expErr: errSubscriptionParameter},
			{name: "option summary without an instrument family", qualifiedChannel: `{"channel":"opt-summary"}`, expErr: errSubscriptionParameter},
			{name: "grid positions without an algo ID", qualifiedChannel: `{"channel":"grid-positions"}`, expErr: errSubscriptionParameter},
			{name: "grid sub orders without an algo ID", qualifiedChannel: `{"channel":"grid-sub-orders"}`, expErr: errSubscriptionParameter},
			{name: "an instrument channel without an instrument", qualifiedChannel: `{"channel":"tickers"}`, expErr: currency.ErrCurrencyPairEmpty},
			{name: "an index channel without an index", qualifiedChannel: `{"channel":"index-tickers"}`, expErr: currency.ErrCurrencyPairEmpty},
			{name: "an instrument and type channel without a type", qualifiedChannel: `{"channel":"option-trades","instFamily":"BTC-USD"}`, expErr: errSubscriptionParameter},
			{name: "a spread channel without a spread", qualifiedChannel: `{"channel":"sprd-books5"}`, expErr: currency.ErrCurrencyPairEmpty},
			{name: "an asset channel without a type", qualifiedChannel: `{"channel":"orders"}`, expErr: errInvalidInstrumentType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				valid := &subscription.Subscription{Channel: channelTickers, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
				invalid := &subscription.Subscription{QualifiedChannel: tc.qualifiedChannel}
				_, err := ex.chunkRequests(nil, subscription.List{valid, invalid}, operationSubscribe)
				require.ErrorIs(t, err, tc.expErr, "chunkRequests must refuse to subscribe an argument OKX would refuse")
				reqs, err := ex.chunkRequests(nil, subscription.List{invalid}, operationUnsubscribe)
				require.NoError(t, err, "chunkRequests must not refuse to unsubscribe an argument")
				require.Len(t, reqs, 1, "the unsubscription must be one request")
				assert.Equal(t, subscription.List{invalid}, reqs[0].subs, "the unsubscription should carry the argument as subscribed")
			})
		}
	})

	t.Run("requests are sized as the connection sends them", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name        string
			encoded     int
			expRequests int
		}{
			{name: "an encoding a byte under the limit fits beside the writer's newline", encoded: maxConnByteLen - 1, expRequests: 1},
			{name: "an encoding at the limit leaves no room for the writer's newline", encoded: maxConnByteLen, expRequests: 2},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				greeks, account := SubscriptionArgument{Channel: channelAccountGreeks}, SubscriptionArgument{Channel: channelAccount, ExtraParams: `{"updateInterval":"0"}`}
				encoded, err := json.Marshal(&WebsocketSubscriptionRequest{ID: ex.MessageID(), Operation: operationSubscribe, Arguments: []SubscriptionArgument{greeks, account}})
				require.NoError(t, err, "the request must encode")
				account.ExtraParams = `{"updateInterval":` + strings.Repeat(" ", tc.encoded-len(encoded)) + `"0"}`
				subs := make(subscription.List, 0, 2)
				for _, arg := range []SubscriptionArgument{greeks, account} {
					qualifiedChannel, err := json.Marshal(&arg)
					require.NoError(t, err, "the argument must encode")
					subs = append(subs, &subscription.Subscription{Channel: arg.Channel, QualifiedChannel: string(qualifiedChannel)})
				}
				reqs, err := ex.chunkRequests(nil, subs, operationSubscribe)
				require.NoError(t, err, "chunkRequests must not error")
				require.Len(t, reqs, tc.expRequests, "chunkRequests must split a request the connection would send beyond the limit")
				for i := range reqs {
					written, err := json.Marshal(&reqs[i])
					require.NoError(t, err, "a request must encode")
					assert.LessOrEqual(t, len(written)+1, maxConnByteLen, "a request should fit the limit with the newline the connection's writer ends it with")
				}
			})
		}
	})

	t.Run("an argument too long for a request of its own is refused", func(t *testing.T) {
		t.Parallel()
		for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
			t.Run(operation, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				tickers := &subscription.Subscription{Channel: channelTickers, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
				qualifiedChannel, err := json.Marshal(&SubscriptionArgument{Channel: channelAccount, ExtraParams: `{"updateInterval":` + strings.Repeat(" ", maxConnByteLen) + `"0"}`})
				require.NoError(t, err, "the argument must encode")
				account := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: string(qualifiedChannel)}
				_, err = ex.chunkRequests(nil, subscription.List{tickers, account}, operation)
				require.ErrorIs(t, err, errSubscriptionArgumentTooLong, "chunkRequests must refuse an argument no request can carry")
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.ErrorIs(t, ex.handleSubscription(t.Context(), conn, operation, subscription.List{tickers, account}), errSubscriptionArgumentTooLong, "handleSubscription must refuse an argument no request can carry")
				assert.Empty(t, conn.Requests(), "nothing should be sent")
			})
		}
	})

	t.Run("an argument OKX may echo as another's goes in a later request", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		account := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account","ccy":"USDT"}`}
		configured := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}`}
		user := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account","ccy":"USDT","uid":"77982378738415879"}`}
		tickers := &subscription.Subscription{Channel: channelTickers, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		reqs, err := ex.chunkRequests(nil, subscription.List{account, tickers, configured, user}, operationSubscribe)
		require.NoError(t, err, "chunkRequests must not error")
		require.Len(t, reqs, 3, "each argument echoed alike must start a request of its own")
		exp := []WebsocketSubscriptionRequest{
			{ID: reqs[0].ID, Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelAccount, Currency: currency.USDT}, {Channel: channelTickers, InstrumentID: mainPair}}, subs: subscription.List{account, tickers}},
			{ID: reqs[1].ID, Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelAccount, Currency: currency.USDT, ExtraParams: `{"updateInterval":"0"}`}}, subs: subscription.List{configured}},
			{ID: reqs[2].ID, Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelAccount, Currency: currency.USDT, UserID: "77982378738415879"}}, subs: subscription.List{user}},
		}
		assert.Equal(t, exp, reqs, "no request should carry two arguments OKX may echo alike")
	})

	t.Run("an argument a subscription on the connection keeps is not sent", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, spot), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(spot), "the connection must hold the spot subscription")
		margin := &subscription.Subscription{Channel: channelTickers, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: spot.QualifiedChannel}
		trades := &subscription.Subscription{Channel: channelTrades, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"trades","instId":"BTC-USDT"}`}
		reqs, err := ex.chunkRequests(conn, subscription.List{margin, trades}, operationSubscribe)
		require.NoError(t, err, "chunkRequests must not error")
		require.Len(t, reqs, 1, "the subscriptions must fit one request")
		exp := []WebsocketSubscriptionRequest{{ID: reqs[0].ID, Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelTrades, InstrumentID: mainPair}}, subs: subscription.List{margin, trades}}}
		assert.Equal(t, exp, reqs, "the argument the connection keeps should not be sent again")
	})
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		frame    string
		expErr   error
		errText  string
		relayed  bool
		relayErr bool
		notice   *WebsocketNotice
	}{
		{name: "pong", frame: "pong"},
		{name: "subscription rejected", frame: `{"event":"error","code":"60012","msg":"Invalid request: {\"op\": \"subscribe\", \"args\":[{ \"channel\" : \"block-tickers\", \"instId\" : \"LTC-USD-200327\"}]}","connId":"a4d3ae55"}`, expErr: errWebsocketRequestRejected, errText: "60012"},
		{name: "liquidation subscription rejected", frame: `{"event":"error","code":"60012","msg":"Invalid request","arg":{"channel":"liquidation-orders","instType":"SWAP"}}`, expErr: errWebsocketRequestRejected, errText: "liquidation-orders"},
		{name: "unknown error code", frame: `{"event":"error","code":"future-code","msg":"Request rejected"}`, expErr: errWebsocketRequestRejected},
		{name: "error without a code", frame: `{"event":"error","msg":"Request rejected"}`, expErr: errWebsocketRequestRejected},
		{name: "channel connection count", frame: `{"event":"channel-conn-count","channel":"orders","connCount":"2","connId":"abcd1234"}`},
		{name: "channel connection limit", frame: `{"event":"channel-conn-count-error","channel":"orders","connCount":"30","connId":"a4d3ae55"}`, expErr: errChannelConnectionLimit, errText: "orders"},
		{name: "subscription acknowledged", frame: `{"event":"subscribe","arg":{"channel":"liquidation-orders","instType":"SWAP"},"connId":"accb8e21"}`},
		{name: "unsubscription acknowledged", frame: `{"event":"unsubscribe","arg":{"channel":"liquidation-orders","instType":"SWAP"},"connId":"d0b44253"}`},
		{name: "connection notice", frame: `{"event":"notice","code":"64008","msg":"The connection will soon be closed for a service upgrade. Please reconnect.","connId":"a4d3ae55"}`, notice: &WebsocketNotice{Code: "64008", Message: "The connection will soon be closed for a service upgrade. Please reconnect.", ConnectionID: "a4d3ae55"}},
		{name: "port retirement notice", frame: `{"event":"notice","code":"64010","msg":"WebSocket port 8443 will be retired soon. Please migrate to port 443.","connId":"d7d6f1f5"}`, notice: &WebsocketNotice{Code: "64010", Message: "WebSocket port 8443 will be retired soon. Please migrate to port 443.", ConnectionID: "d7d6f1f5"}},
		{name: "public subscription acknowledged", frame: `{"event":"subscribe","arg":{"channel":"liquidation-orders","instType":"SWAP"}}`},
		{name: "public unsubscription acknowledged", frame: `{"event":"unsubscribe","arg":{"channel":"liquidation-orders","instType":"SWAP"}}`},
		{name: "connection notice for a channel", frame: `{"event":"notice","code":"64008","msg":"The connection will soon be closed","arg":{"channel":"liquidation-orders"}}`, notice: &WebsocketNotice{Code: "64008", Message: "The connection will soon be closed"}},
		{name: "empty envelope", frame: `{}`},
		{name: "channel header without data", frame: `{"arg":{"channel":"funding-rate","instId":"BTC-USDT-SWAP"}}`},
		{name: "malformed frame", frame: `{"event":`, relayErr: true},
		{name: "unhandled channel", frame: `{"arg":{"channel":"unknown-channel"},"data":[{}]}`, relayed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			err := ex.wsHandleData(t.Context(), nil, []byte(tc.frame))
			switch {
			case tc.expErr != nil:
				require.ErrorIs(t, err, tc.expErr, "wsHandleData must report the rejection")
				if tc.errText != "" {
					assert.ErrorContains(t, err, tc.errText, "the error should carry the rejection's detail")
				}
			case tc.relayErr:
				require.Error(t, err, "a malformed frame must error")
			default:
				require.NoError(t, err, "wsHandleData must not error")
			}
			if tc.notice != nil {
				require.Len(t, ex.Websocket.DataHandler.C, 1, "a notice must be relayed")
				assert.Equal(t, tc.notice, (<-ex.Websocket.DataHandler.C).Data, "a notice should be relayed as OKX sent it")
				return
			}
			if !tc.relayed {
				assert.Empty(t, ex.Websocket.DataHandler.C, "control frames should not be relayed")
				return
			}
			require.Len(t, ex.Websocket.DataHandler.C, 1, "an unhandled push must be relayed")
			_, ok := (<-ex.Websocket.DataHandler.C).Data.(websocket.UnhandledMessageWarning)
			assert.True(t, ok, "an unhandled push should be relayed as a warning")
		})
	}

	t.Run("documented push frames", func(t *testing.T) {
		t.Parallel()
		for name, frame := range pushDataMap {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.API.AuthenticatedSupport = false
				ex.API.AuthenticatedWebsocketSupport = false
				var conn websocket.Connection
				switch name {
				case "Balance And Position", "Accounts":
					creds := &accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"}
					ex.API.AuthenticatedSupport = true
					ex.API.AuthenticatedWebsocketSupport = true
					ex.SetCredentials(creds)
					// The account push saves its balances under the login of the connection it arrives on
					conn = new(accountStreamConnection)
					ex.recordWebsocketLogin(conn, creds)
				case "Public Block Trades":
					// Block trades are pushed for the instruments subscribed, which are enabled
					enableWebsocketTestPair(t, ex, asset.Options, currency.NewPairWithDelimiter("BTC", "USD-231020-5000-P", currency.DashDelimiter))
				}
				err := ex.wsHandleData(t.Context(), conn, []byte(frame))
				switch name {
				case "Failure":
					require.ErrorIs(t, err, errWebsocketRequestRejected, "subscription rejection must reach the websocket manager")
					assert.ErrorContains(t, err, "60012", "rejection should preserve the exchange status code")
				default:
					require.NoErrorf(t, err, "%s must not error", name)
				}
			})
		}
	})

	t.Run("swap pushes", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		for name, frame := range map[string]string{
			"Ticker":             `{"arg": {"channel": "tickers","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`,
			"Candlesticks":       `{"arg": {"channel": "candle1D","instId": "BTC-USD-SWAP"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`,
			"Snapshot OrderBook": `{"arg":{"channel":"books","instId":"BTC-USD-SWAP"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
		} {
			require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, []byte(frame)), "%s must not error", name)
		}
	})

	t.Run("typed funding push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(`{"arg":{"channel":"funding-rate","instId":"BTC-USDT-SWAP"},"data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","fundingTime":"1700000000000","ts":"1699999990000"}]}`)), "funding push must decode through its channel type")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "one typed funding response must be relayed")
		response, ok := (<-ex.Websocket.DataHandler.C).Data.(*WebsocketFundingRatePush)
		require.True(t, ok, "funding push must retain its concrete response type")
		require.Len(t, response.Data, 1, "funding push must contain one rate")
		assert.Equal(t, "BTC-USDT-SWAP", response.Argument.InstrumentID.String(), "routing metadata should remain available")
		assert.Equal(t, 0.0001, response.Data[0].FundingRate.Float64(), "funding rate should decode into its numeric type")
	})

	t.Run("routes a reply to its request", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"ordId": "12345", "sCode": "0"}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		var resp []*OrderActionResponse
		require.NoError(t, ex.sendWebsocketRequest(t.Context(), privateConnection, placeOrderEPL, "order", []PlaceOrderRequest{}, &resp), "the reply must match its request by ID")
		require.Len(t, resp, 1, "the reply must decode")
		assert.Equal(t, "12345", resp[0].OrderID, "the reply's row should decode")
	})

	t.Run("a reply to a subscription request is routed to the request awaiting it", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		req := &WebsocketSubscriptionRequest{ID: "1512", Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, replies: make(chan []byte, 1)}
		ex.subscriptionRequests.send(nil, req)
		reply := []byte(`{"id":"1512","event":"subscribe","arg":{"channel":"tickers","instId":"BTC-USDT"},"connId":"a4d3ae55"}`)
		require.NoError(t, ex.wsHandleData(t.Context(), nil, reply), "a reply to a request awaiting it must be routed")
		require.Len(t, req.replies, 1, "the reply must reach the request")
		assert.Equal(t, reply, <-req.replies, "the request should receive the reply as sent")
	})

	t.Run("a connection count error ends the channel's subscriptions on the connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(privateConnection, conn), "TrackTestConnection must not error")
		swap := &subscription.Subscription{Channel: channelPositions, Asset: asset.PerpetualSwap, QualifiedChannel: `{"channel":"positions","instType":"SWAP"}`}
		futures := &subscription.Subscription{Channel: channelPositions, Asset: asset.Futures, QualifiedChannel: `{"channel":"positions","instType":"FUTURES"}`}
		orders := &subscription.Subscription{Channel: channelOrders, Asset: asset.PerpetualSwap, QualifiedChannel: `{"channel":"orders","instType":"SWAP"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, swap, futures, orders), "AddSuccessfulSubscriptions must not error")
		for _, s := range []*subscription.Subscription{swap, futures, orders} {
			require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold the subscription")
		}
		err := ex.wsHandleData(t.Context(), conn, []byte(`{"event":"channel-conn-count-error","channel":"positions","connCount":"30","connId":"a4d3ae55"}`))
		require.ErrorIs(t, err, errChannelConnectionLimit, "wsHandleData must report the ended subscriptions")
		assert.ErrorContains(t, err, `ended {"channel":"positions","instType":"FUTURES"}, {"channel":"positions","instType":"SWAP"}`, "the error should name the ended subscriptions")
		for _, s := range []*subscription.Subscription{swap, futures} {
			assert.Nilf(t, ex.Websocket.GetSubscription(s), "the ended %s subscription should no longer be tracked", s.QualifiedChannel)
		}
		assert.Same(t, orders, ex.Websocket.GetSubscription(orders), "another channel's subscription should stay tracked")
		assert.Equal(t, subscription.List{orders}, conn.Subscriptions().List(), "the connection should hold only the subscription OKX kept")
	})

	t.Run("a connection count error during the subscription leaves it untracked", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		positions := &subscription.Subscription{Channel: channelPositions, Asset: asset.PerpetualSwap, QualifiedChannel: `{"channel":"positions","instType":"SWAP"}`}
		account := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: `{"channel":"account"}`}
		synctest.Test(t, func(t *testing.T) {
			conn := dialWebsocketTestServer(t, ex, privateConnection, func(id, operation string, arguments []json.RawMessage) []string {
				return []string{
					subscriptionReply(id, operation, arguments[0]),
					`{"event":"channel-conn-count-error","channel":"positions","connCount":"30","connId":"a4d3ae55"}`,
					subscriptionReply(id, operation, arguments[1]),
				}
			})
			err := ex.handleSubscription(t.Context(), conn, operationSubscribe, subscription.List{positions, account})
			require.ErrorIs(t, err, errChannelConnectionLimit, "handleSubscription must report the ended subscription")
			assert.ErrorContains(t, err, `ended {"channel":"positions","instType":"SWAP"}`, "the error should name the ended subscription")
			assert.Nil(t, ex.Websocket.GetSubscription(positions), "the ended subscription should not be tracked")
			assert.Same(t, account, ex.Websocket.GetSubscription(account), "the subscription OKX kept should be tracked")
		})
	})

	t.Run("a service upgrade notice is relayed and reconnects the websocket", func(t *testing.T) {
		t.Parallel()
		var noticed atomic.Bool
		srv := &websocketTestServer{push: func(string, json.RawMessage) []string {
			if noticed.CompareAndSwap(false, true) {
				return []string{`{"event":"notice","code":"64008","msg":"The connection will soon be closed for a service upgrade. Please reconnect.","connId":"a4d3ae55"}`}
			}
			return nil
		}}
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}}}, func(ex *Exchange) {
			ex.Config.ConnectionMonitorDelay = 10 * time.Millisecond
		})
		select {
		case resp := <-ex.Websocket.DataHandler.C:
			assert.Equal(t, &WebsocketNotice{Code: "64008", Message: "The connection will soon be closed for a service upgrade. Please reconnect.", ConnectionID: "a4d3ae55"}, resp.Data, "the notice should be relayed")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the notice must be relayed")
		}
		require.Eventually(t, func() bool { return len(srv.requests(operationSubscribe)) == 2 }, 5*time.Second, 10*time.Millisecond, "the websocket must reconnect and subscribe again")
		dials, _ := srv.recorded()
		assert.Equal(t, []string{"/ws/v5/public", "/ws/v5/public"}, dials, "the connection should be dialled again")
	})

	t.Run("another notice is relayed without reconnecting", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{push: func(string, json.RawMessage) []string {
			return []string{`{"event":"notice","code":"64010","msg":"WebSocket port 8443 will be retired soon. Please migrate to port 443.","connId":"d7d6f1f5"}`}
		}}
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}}}, func(ex *Exchange) {
			ex.Config.ConnectionMonitorDelay = 10 * time.Millisecond
		})
		select {
		case resp := <-ex.Websocket.DataHandler.C:
			assert.Equal(t, &WebsocketNotice{Code: "64010", Message: "WebSocket port 8443 will be retired soon. Please migrate to port 443.", ConnectionID: "d7d6f1f5"}, resp.Data, "the notice should be relayed")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the notice must be relayed")
		}
		assert.Never(t, func() bool { dials, _ := srv.recorded(); return len(dials) > 1 }, 200*time.Millisecond, 10*time.Millisecond, "the websocket should not reconnect")
	})
}

func TestReconnectWebsocket(t *testing.T) {
	t.Parallel()
	fastMonitor := func(ex *Exchange) { ex.Config.ConnectionMonitorDelay = 10 * time.Millisecond }

	t.Run("reconnects every connection, and again once that is done", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
			{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneMin},
		}, fastMonitor)
		for cycle := 1; cycle <= 2; cycle++ {
			ex.reconnectWebsocket()
			require.Eventuallyf(t, func() bool {
				return len(srv.requests(operationSubscribe)) == 2*(cycle+1) && ex.Websocket.IsConnected() && !ex.websocketReconnecting.Load()
			}, 5*time.Second, 10*time.Millisecond, "reconnection %d must resubscribe both connections", cycle)
		}
		dials, _ := srv.recorded()
		slices.Sort(dials)
		assert.Equal(t, []string{"/ws/v5/business", "/ws/v5/business", "/ws/v5/business", "/ws/v5/public", "/ws/v5/public", "/ws/v5/public"}, dials, "each reconnection should dial every connection again")
	})

	t.Run("a reconnection under way absorbs another", func(t *testing.T) {
		t.Parallel()
		srv := new(websocketTestServer)
		ex := newWebsocketTestExchange(t, srv, false, subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}}}, fastMonitor)
		ex.websocketReconnecting.Store(true)
		ex.reconnectWebsocket()
		assert.Never(t, func() bool { dials, _ := srv.recorded(); return len(dials) > 1 }, 200*time.Millisecond, 10*time.Millisecond, "a second reconnection should not start")
		assert.True(t, ex.Websocket.IsConnected(), "the websocket should stay connected")
	})

	t.Run("a disconnected websocket is left alone", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.reconnectWebsocket()
		assert.False(t, ex.websocketReconnecting.Load(), "no reconnection should start for a websocket which is not connected")
	})
}

func TestRelayPush(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.Error(t, relayPush[WebsocketMarkPricePush](t.Context(), ex, []byte(`{"data":[{"markPx":true}]}`)), "a push which does not decode must error")
	require.Empty(t, ex.Websocket.DataHandler.C, "a push which does not decode must not be relayed")
	require.NoError(t, relayPush[WebsocketMarkPricePush](t.Context(), ex, []byte(`{"arg":{"channel":"mark-price","instId":"BTC-USDT-SWAP"},"data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","markPx":"61475.7","ts":"1795405194146"}]}`)), "relayPush must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "relayPush must relay the push")
	exp := &WebsocketMarkPricePush{
		Argument: SubscriptionArgument{Channel: channelMarkPrice, InstrumentID: perpetualSwapPair},
		Data:     []WebsocketMarkPrice{{InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair, MarkPrice: 61475.7, Timestamp: types.Time(time.UnixMilli(1795405194146))}},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "relayPush should relay the decoded push")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub    *subscription.Subscription
		asset  asset.Item
		exp    string
		expErr error
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel}, asset.Spot, channelTickers, nil},
		{&subscription.Subscription{Channel: subscription.TickerChannel}, asset.Spread, channelSpreadTickers, nil},
		{&subscription.Subscription{Channel: subscription.AllTradesChannel}, asset.PerpetualSwap, channelTrades, nil},
		{&subscription.Subscription{Channel: subscription.AllTradesChannel}, asset.Spread, channelSpreadPublicTrades, nil},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel}, asset.Margin, channelOrderBooks, nil},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel}, asset.Spread, channelSpreadOrderBooks5, nil},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMin}, asset.Spot, "candle1m", nil},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, asset.Spot, "candle1Dutc", nil},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, asset.Spread, "sprd-candle1Dutc", nil},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThousandMilliseconds}, asset.Spot, "candle1s", nil},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThousandMilliseconds}, asset.Spread, "", errUnsupportedCandleInterval},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, asset.Spot, "", errUnsupportedCandleInterval},
		{&subscription.Subscription{Channel: subscription.MyOrdersChannel}, asset.Futures, channelOrders, nil},
		{&subscription.Subscription{Channel: subscription.MyOrdersChannel}, asset.Spread, channelSpreadOrders, nil},
		{&subscription.Subscription{Channel: subscription.MyTradesChannel}, asset.Spot, channelFills, nil},
		{&subscription.Subscription{Channel: subscription.MyTradesChannel}, asset.Spread, channelSpreadTrades, nil},
		{&subscription.Subscription{Channel: subscription.MyAccountChannel}, asset.Empty, channelAccount, nil},
		{&subscription.Subscription{Channel: channelOptionSummary}, asset.Empty, channelOptionSummary, nil},
	} {
		t.Run(tc.sub.Channel+"/"+tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			got, err := channelName(tc.sub, tc.asset)
			require.ErrorIs(t, err, tc.expErr, "channelName must return the expected error")
			assert.Equal(t, tc.exp, got, "channelName should select the channel OKX serves the asset on")
		})
	}
}

func TestSubscriptionChannel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, channelTickers, subscriptionChannel(&subscription.Subscription{QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}), "the qualified channel's name should be returned")
	assert.Empty(t, subscriptionChannel(&subscription.Subscription{}), "an unexpanded subscription should have no channel")
}

func TestChannelConnection(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		channelTickers:              publicConnection,
		channelOrderBooksTBT:        publicLoginConnection,
		channelOrders:               privateConnection,
		channelAllTrades:            businessConnection,
		"candle1m":                  businessConnection,
		"sprd-candle1Dutc":          businessConnection,
		"mark-price-candle1M":       businessConnection,
		"index-candle1s":            businessConnection,
		channelSpreadOrders:         businessLoginConnection,
		channelEconomicCalendar:     businessLoginConnection,
		channelPublicBlockTrades:    businessConnection,
		"an undescribed channel":    publicConnection,
		channelEventContractMarkets: publicConnection,
	} {
		assert.Equalf(t, exp, channelConnection(channel), "%s should be served by its documented connection", channel)
	}
}

func TestRequiresLogin(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]bool{
		channelTickers:           false,
		channelOrderBooks50TBT:   true,
		channelAccount:           true,
		"candle1m":               false,
		channelDepositInfo:       true,
		"an undescribed channel": false,
	} {
		assert.Equalf(t, exp, requiresLogin(channel), "%s should need a login only on a login connection", channel)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	subs := subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}},
		{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Interval: kline.OneDay},
		{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spot},
		{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spread},
		{Enabled: true, Channel: channelOrderBooksTBT, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
		{Enabled: true, Channel: subscription.MyAccountChannel},
		{Enabled: true, Channel: channelOptionSummary, Params: map[string]any{"instFamily": "BTC-USD"}},
		{Enabled: true, Channel: channelIndexTickers, Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}},
		{Enabled: true, Channel: channelEventContractMarkets, Params: map[string]any{"instType": "EVENTS"}},
		{Enabled: true, Channel: channelGridPositions, Params: map[string]any{"algoId": "449327675342323712"}},
		{Enabled: true, Channel: channelPositions, Asset: asset.All},
	}
	for _, tc := range []struct {
		name          string
		authenticated bool
		exp           map[string][]string
	}{
		{
			name:          "authenticated",
			authenticated: true,
			exp: map[string][]string{
				publicConnection:        {`{"channel":"event-contract-markets","instType":"EVENTS"}`, `{"channel":"index-tickers","instId":"BTC-USDT"}`, `{"channel":"opt-summary","instFamily":"BTC-USD"}`, `{"channel":"tickers","instId":"BTC-USDT"}`},
				publicLoginConnection:   {`{"channel":"books-l2-tbt","instId":"BTC-USDT"}`},
				privateConnection:       {`{"channel":"account"}`, `{"channel":"orders","instType":"SPOT"}`, `{"channel":"positions","instType":"FUTURES"}`, `{"channel":"positions","instType":"MARGIN"}`, `{"channel":"positions","instType":"OPTION"}`, `{"channel":"positions","instType":"SWAP"}`},
				businessConnection:      {`{"channel":"candle1Dutc","instId":"BTC-USDT"}`, `{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
				businessLoginConnection: {`{"channel":"grid-positions","algoId":"449327675342323712"}`, `{"channel":"sprd-orders"}`},
			},
		},
		{
			name: "unauthenticated",
			exp: map[string][]string{
				publicConnection:   {`{"channel":"event-contract-markets","instType":"EVENTS"}`, `{"channel":"index-tickers","instId":"BTC-USDT"}`, `{"channel":"opt-summary","instFamily":"BTC-USD"}`, `{"channel":"tickers","instId":"BTC-USDT"}`},
				businessConnection: {`{"channel":"candle1Dutc","instId":"BTC-USDT"}`, `{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.authenticated)
			ex.Features.Subscriptions = subs
			got := map[string][]string{}
			for _, connection := range []string{publicConnection, publicLoginConnection, privateConnection, businessConnection, businessLoginConnection} {
				generated, err := ex.generateSubscriptions(t.Context(), connection)
				require.NoErrorf(t, err, "generateSubscriptions must not error for %s", connection)
				for _, s := range generated {
					got[connection] = append(got[connection], s.QualifiedChannel)
					assert.Equalf(t, requiresLogin(subscriptionChannel(s)), s.Authenticated, "%s should be authenticated only on a login connection", s.QualifiedChannel)
				}
				slices.Sort(got[connection])
			}
			assert.Equal(t, tc.exp, got, "each connection should generate the subscriptions it serves")
		})
	}

	// generated lists each connection's subscriptions as their asset and
	// qualified channel, sorted, with their authentication checked
	generated := func(t *testing.T, ex *Exchange) map[string][]string {
		t.Helper()
		got := map[string][]string{}
		for _, connection := range []string{publicConnection, publicLoginConnection, privateConnection, businessConnection, businessLoginConnection} {
			subs, err := ex.generateSubscriptions(t.Context(), connection)
			require.NoErrorf(t, err, "generateSubscriptions must not error for %s", connection)
			for _, s := range subs {
				got[connection] = append(got[connection], s.Asset.String()+" "+s.QualifiedChannel)
				assert.Equalf(t, requiresLogin(subscriptionChannel(s)), s.Authenticated, "%s should be authenticated only on a login connection", s.QualifiedChannel)
			}
			slices.Sort(got[connection])
		}
		return got
	}

	t.Run("default subscriptions", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		// Each enabled instrument subscribes its trades, order book and ticker,
		// and each enabled asset its orders; spreads use their own channels on
		// the business connection
		exp := map[string][]string{privateConnection: {` {"channel":"account"}`}}
		for _, a := range ex.GetAssetTypes(true) {
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			if a == asset.Spread {
				for _, p := range pairs {
					for _, channel := range []string{"sprd-books5", "sprd-public-trades", "sprd-tickers"} {
						exp[businessConnection] = append(exp[businessConnection], a.String()+` {"channel":"`+channel+`","sprdId":"`+p.String()+`"}`)
					}
				}
				exp[businessLoginConnection] = append(exp[businessLoginConnection], a.String()+` {"channel":"sprd-orders"}`)
				continue
			}
			for _, p := range pairs {
				for _, channel := range []string{"books", "tickers", "trades"} {
					exp[publicConnection] = append(exp[publicConnection], a.String()+` {"channel":"`+channel+`","instId":"`+p.Format(currency.PairFormat{Uppercase: true, Delimiter: "-"}).String()+`"}`)
				}
			}
			exp[privateConnection] = append(exp[privateConnection], a.String()+` {"channel":"orders","instType":"`+GetInstrumentTypeFromAssetItem(a)+`"}`)
		}
		for _, list := range exp {
			slices.Sort(list)
		}
		require.NotEmpty(t, exp[businessConnection], "the test config must enable a spread")
		assert.Equal(t, exp, generated(t, ex), "the default subscriptions should expand to every enabled instrument on its connection")
	})

	t.Run("channels with a required argument", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			channel    string
			params     map[string]any
			connection string
			exp        string
		}{
			{name: "grid positions with an algo ID", channel: channelGridPositions, params: map[string]any{"algoId": "42"}, connection: businessLoginConnection, exp: ` {"channel":"grid-positions","algoId":"42"}`},
			{name: "grid positions without an algo ID", channel: channelGridPositions, connection: businessLoginConnection},
			{name: "grid sub orders with an algo ID", channel: channelGridSubOrders, params: map[string]any{"algoId": "42"}, connection: businessLoginConnection, exp: ` {"channel":"grid-sub-orders","algoId":"42"}`},
			{name: "grid sub orders without an algo ID", channel: channelGridSubOrders, connection: businessLoginConnection},
			{name: "option summary with an instrument family", channel: channelOptionSummary, params: map[string]any{"instFamily": "BTC-USD"}, connection: publicConnection, exp: ` {"channel":"opt-summary","instFamily":"BTC-USD"}`},
			{name: "option summary without an instrument family", channel: channelOptionSummary, connection: publicConnection},
			{name: "event contract markets", channel: channelEventContractMarkets, connection: publicConnection, exp: ` {"channel":"event-contract-markets","instType":"EVENTS"}`},
			{name: "event contract markets of another instrument type", channel: channelEventContractMarkets, params: map[string]any{"instType": "SPOT"}, connection: publicConnection},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
				ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: tc.channel, Params: tc.params}}
				if tc.exp == "" {
					_, err := ex.generateSubscriptions(t.Context(), tc.connection)
					require.ErrorIs(t, err, errSubscriptionParameter, "generateSubscriptions must refuse a subscription without the argument OKX requires")
					return
				}
				assert.Equal(t, map[string][]string{tc.connection: {tc.exp}}, generated(t, ex), "the subscription should carry the argument OKX requires")
			})
		}
	})

	t.Run("subscriptions differing only in their parameters are tracked apart", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(privateConnection, conn), "TrackTestConnection must not error")
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Channel: channelAccount, Params: map[string]any{"ccy": "USDT"}},
			{Enabled: true, Channel: channelAccount, Params: map[string]any{"ccy": "USDT", "extraParams": `{"updateInterval":"0"}`}},
		}
		subs, err := ex.generateSubscriptions(t.Context(), privateConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		require.Len(t, subs, 2, "generateSubscriptions must generate both account subscriptions")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs[0]), "AddSuccessfulSubscriptions must not error")
		assert.Nil(t, ex.Websocket.GetSubscription(subs[1]), "a subscription differing only in its parameters should not find the tracked one")
		regenerated, err := ex.generateSubscriptions(t.Context(), privateConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		assert.Same(t, subs[0], ex.Websocket.GetSubscription(regenerated[0]), "the subscription generated again should find the tracked one")
		assert.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs[1]), "a subscription differing only in its parameters should be tracked beside the other")
	})

	t.Run("an expanded subscription is generated as a copy of the configuration's", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		configured := &subscription.Subscription{Enabled: true, Channel: channelTickers, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		ex.Features.Subscriptions = subscription.List{configured}
		subs, err := ex.generateSubscriptions(t.Context(), publicConnection)
		require.NoError(t, err, "generateSubscriptions must not error")
		require.Len(t, subs, 1, "generateSubscriptions must generate the expanded subscription")
		require.NotSame(t, configured, subs[0], "the generated subscription must be a copy")
		assert.Equal(t, configured.QualifiedChannel, subs[0].QualifiedChannel, "the copy should keep the argument")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, subs[0]), "AddSuccessfulSubscriptions must not error")
		assert.Equal(t, subscription.InactiveState, configured.State(), "the configuration's subscription should stay inactive")
		assert.IsType(t, (*subscription.ExactKey)(nil), configured.EnsureKeyed(), "the configuration's subscription should keep the default key")
	})

	t.Run("a parameter set OKX rejected is subscribed once it is the only one configured", func(t *testing.T) {
		t.Parallel()
		var rejected atomic.Bool
		srv := &websocketTestServer{reject: func(_, operation string, argument json.RawMessage) string {
			if operation == operationSubscribe && strings.Contains(string(argument), "extraParams") && rejected.CompareAndSwap(false, true) {
				return "60012"
			}
			return ""
		}}
		configured := &subscription.Subscription{Enabled: true, Channel: channelAccount, Params: map[string]any{"ccy": "USDT", "extraParams": `{"updateInterval":"0"}`}}
		ex, err := connectWebsocketTestExchange(t, srv, true, subscription.List{{Enabled: true, Channel: channelAccount, Params: map[string]any{"ccy": "USDT"}}, configured})
		require.ErrorIs(t, err, errWebsocketRequestRejected, "Connect must report the rejected parameter set")
		ex.Features.Subscriptions = subscription.List{configured}
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
		_, frames := srv.recorded()
		var requests []string
		for _, f := range frames {
			if f.operation != operationLogin {
				requests = append(requests, f.operation+" "+string(f.arguments))
			}
		}
		exp := []string{
			`subscribe [{"channel":"account","ccy":"USDT"}]`,
			`subscribe [{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}]`,
			`unsubscribe [{"channel":"account","ccy":"USDT"}]`,
			`subscribe [{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}]`,
		}
		assert.Equal(t, exp, requests, "the rejected parameter set should be requested once it replaces the accepted one")
		tracked := ex.Websocket.GetSubscriptions()
		require.Len(t, tracked, 1, "one subscription must be tracked")
		assert.JSONEq(t, `{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}`, tracked[0].QualifiedChannel, "the configured parameter set should be tracked")
	})
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tmpl, err := e.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	assert.NotNil(t, tmpl, "GetSubscriptionTemplate should return a template")
}

func TestChannelScopeName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		asset   asset.Item
		exp     string
	}{
		{channelTickers, asset.Spot, "pair"},
		{channelTickers, asset.Spread, "none"},
		{channelTickers, asset.Empty, "asset"},
		{channelOrders, asset.Futures, "asset"},
		{channelPositions, asset.Spot, "none"},
		{channelSpreadOrders, asset.Spread, "asset"},
		{channelSpreadTickers, asset.Spread, "pair"},
		{channelAccount, asset.Spot, "asset"},
		{"an undescribed channel", asset.Spot, "asset"},
	} {
		assert.Equalf(t, tc.exp, channelScopeName(tc.channel, tc.asset), "%s for %s should have the expected scope", tc.channel, tc.asset)
	}

	t.Run("symbol channels subscribe per instrument", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			channel string
			want    bool
		}{
			{subscription.CandlesChannel, true},
			{subscription.TickerChannel, true},
			{subscription.OrderbookChannel, true},
			{subscription.AllTradesChannel, true},
			{channelTickers, true},
			{channelTrades, true},
			{channelOrderBooks, true},
			{channelOrderBooks5, true},
			{channelOrderBooks50TBT, true},
			{channelOrderBooksTBT, true},
			{channelBBOTBT, true},
			{channelFundingRate, true},
			{channelMarkPrice, true},
			{channelOpenInterest, true},
			{channelPriceLimit, true},
			{channelInstruments, false},
			{channelLiquidationOrders, false},
			{channelOrders, false},
			{subscription.MyAccountChannel, false},
			{subscription.MyOrdersChannel, false},
			{"unknown", false},
		} {
			t.Run(tc.channel, func(t *testing.T) {
				t.Parallel()
				a, pair := asset.Spot, mainPair
				if tc.channel == channelFundingRate || tc.channel == channelMarkPrice || tc.channel == channelOpenInterest || tc.channel == channelPriceLimit {
					a, pair = asset.PerpetualSwap, perpetualSwapPair
				}
				sub := &subscription.Subscription{Enabled: true, Channel: tc.channel, Asset: a, Pairs: currency.Pairs{pair}, Interval: kline.OneMin}
				name, err := channelName(sub, a)
				require.NoError(t, err, "channelName must not error")
				require.Equal(t, tc.want, channelScopeName(name, a) == "pair", "symbol channel classification must match the API scope")
				if !tc.want {
					return
				}
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "Test exchange setup must succeed")
				tracked.Websocket.SetCanUseAuthenticatedEndpoints(true)
				tracked.Features.Subscriptions = subscription.List{sub}
				got, err := tracked.generateSubscriptions(t.Context(), channelConnection(name))
				require.NoError(t, err, "Symbol subscription expansion must succeed")
				require.Len(t, got, 1, "Explicit single instrument must produce one subscription")
				assert.JSONEq(t, fmt.Sprintf(`{"channel":%q,"instId":%q}`, name, pair), got[0].QualifiedChannel, "symbol channels should name the chosen exchange channel and the subscribed instrument")
			})
		}
	})
}

func TestChannelArgument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		sub     *subscription.Subscription
		channel string
		asset   asset.Item
		pair    any
		exp     string
		expErr  error
	}{
		{name: "instrument", sub: &subscription.Subscription{}, channel: channelTickers, asset: asset.Spot, pair: mainPair, exp: `{"channel":"tickers","instId":"BTC-USDT"}`},
		{name: "asset", sub: &subscription.Subscription{}, channel: channelOrders, asset: asset.PerpetualSwap, exp: `{"channel":"orders","instType":"SWAP"}`},
		{name: "spread asset", sub: &subscription.Subscription{}, channel: channelSpreadOrders, asset: asset.Spread, exp: `{"channel":"sprd-orders"}`},
		{name: "spread", sub: &subscription.Subscription{}, channel: channelSpreadOrderBooks5, asset: asset.Spread, pair: spreadPair, exp: `{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
		{name: "index", sub: &subscription.Subscription{}, channel: channelIndexTickers, asset: asset.Futures, pair: currency.NewPairWithDelimiter("BTC", "USD_UM-261225", currency.DashDelimiter), exp: `{"channel":"index-tickers","instId":"BTC-USD"}`},
		{name: "instrument and type", sub: &subscription.Subscription{}, channel: channelEstimatedPrice, asset: asset.Futures, pair: currency.NewPairWithDelimiter("BTC", "USD-261127", currency.DashDelimiter), exp: `{"channel":"estimated-price","instType":"FUTURES","instId":"BTC-USD-261127"}`},
		{name: "family parameter", sub: &subscription.Subscription{Params: map[string]any{"instType": "OPTION", "instFamily": "BTC-USD"}}, channel: channelOptionTrades, asset: asset.Empty, exp: `{"channel":"option-trades","instType":"OPTION","instFamily":"BTC-USD"}`},
		{name: "every parameter", sub: &subscription.Subscription{Params: map[string]any{"instType": "SWAP", "instFamily": "BTC-USDT", "instId": "BTC-USDT-SWAP", "sprdId": "BTC-USDT_BTC-USDT-SWAP", "algoId": "1", "ccy": "BTC", "extraParams": `{"updateInterval":"0"}`}}, channel: "an undescribed channel", asset: asset.Empty, exp: `{"channel":"an undescribed channel","instType":"SWAP","instFamily":"BTC-USDT","instId":"BTC-USDT-SWAP","sprdId":"BTC-USDT_BTC-USDT-SWAP","algoId":"1","ccy":"BTC","extraParams":"{\"updateInterval\":\"0\"}"}`},
		{name: "parameter of the wrong type", sub: &subscription.Subscription{Params: map[string]any{"algoId": 1}}, channel: channelGridPositions, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "malformed instrument parameter", sub: &subscription.Subscription{Params: map[string]any{"instId": "BT"}}, channel: channelTickers, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "malformed spread parameter", sub: &subscription.Subscription{Params: map[string]any{"sprdId": "BT"}}, channel: channelSpreadTickers, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "instrument channel without an instrument", sub: &subscription.Subscription{}, channel: "candle1D", asset: asset.Empty, expErr: currency.ErrCurrencyPairEmpty},
		{name: "spread channel without a spread", sub: &subscription.Subscription{}, channel: channelSpreadTickers, asset: asset.Empty, expErr: currency.ErrCurrencyPairEmpty},
		{name: "instrument and type channel without an instrument", sub: &subscription.Subscription{Params: map[string]any{"instType": "OPTION"}}, channel: channelOptionTrades, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "asset channel without a type", sub: &subscription.Subscription{}, channel: channelOrders, asset: asset.Empty, expErr: errInvalidInstrumentType},
		{name: "event contract markets", sub: &subscription.Subscription{}, channel: channelEventContractMarkets, asset: asset.Empty, exp: `{"channel":"event-contract-markets","instType":"EVENTS"}`},
		{name: "event contract markets of a given type", sub: &subscription.Subscription{Params: map[string]any{"instType": "EVENTS"}}, channel: channelEventContractMarkets, asset: asset.Spot, exp: `{"channel":"event-contract-markets","instType":"EVENTS"}`},
		{name: "event contract markets of another type", sub: &subscription.Subscription{Params: map[string]any{"instType": "SPOT"}}, channel: channelEventContractMarkets, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "event contract markets of a type spelled in lower case", sub: &subscription.Subscription{Params: map[string]any{"instType": "events"}}, channel: channelEventContractMarkets, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "option summary", sub: &subscription.Subscription{Params: map[string]any{"instFamily": "BTC-USD"}}, channel: channelOptionSummary, asset: asset.Options, exp: `{"channel":"opt-summary","instFamily":"BTC-USD"}`},
		{name: "option summary without a family", sub: &subscription.Subscription{}, channel: channelOptionSummary, asset: asset.Options, expErr: errSubscriptionParameter},
		{name: "grid positions", sub: &subscription.Subscription{Params: map[string]any{"algoId": "449327675342323712"}}, channel: channelGridPositions, asset: asset.Empty, exp: `{"channel":"grid-positions","algoId":"449327675342323712"}`},
		{name: "grid positions without an algo ID", sub: &subscription.Subscription{}, channel: channelGridPositions, asset: asset.Empty, expErr: errSubscriptionParameter},
		{name: "grid sub orders", sub: &subscription.Subscription{Params: map[string]any{"algoId": "449327675342323712"}}, channel: channelGridSubOrders, asset: asset.Empty, exp: `{"channel":"grid-sub-orders","algoId":"449327675342323712"}`},
		{name: "grid sub orders without an algo ID", sub: &subscription.Subscription{}, channel: channelGridSubOrders, asset: asset.Empty, expErr: errSubscriptionParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.channelArgument(tc.sub, tc.channel, tc.asset, tc.pair)
			require.ErrorIs(t, err, tc.expErr, "channelArgument must return the expected error")
			if tc.expErr != nil {
				return
			}
			assert.JSONEq(t, tc.exp, got, "channelArgument should form the documented argument")
		})
	}
}

func TestCheckSubscriptionArgument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		arg    SubscriptionArgument
		expErr error
	}{
		{name: "an instrument channel with an instrument", arg: SubscriptionArgument{Channel: channelTickers, InstrumentID: mainPair}},
		{name: "an instrument channel without an instrument", arg: SubscriptionArgument{Channel: channelTickers}, expErr: currency.ErrCurrencyPairEmpty},
		{name: "fills without an instrument", arg: SubscriptionArgument{Channel: channelFills}},
		{name: "an index channel without an index", arg: SubscriptionArgument{Channel: channelIndexTickers}, expErr: currency.ErrCurrencyPairEmpty},
		{name: "an instrument and type channel with a family", arg: SubscriptionArgument{Channel: channelOptionTrades, InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"}},
		{name: "an instrument and type channel without a type", arg: SubscriptionArgument{Channel: channelOptionTrades, InstrumentID: websocketOptionPair}, expErr: errSubscriptionParameter},
		{name: "an instrument and type channel without an instrument or family", arg: SubscriptionArgument{Channel: channelOptionTrades, InstrumentType: instTypeOption}, expErr: errSubscriptionParameter},
		{name: "a spread channel without a spread", arg: SubscriptionArgument{Channel: channelSpreadTickers}, expErr: currency.ErrCurrencyPairEmpty},
		{name: "an asset channel without a type", arg: SubscriptionArgument{Channel: channelOrders}, expErr: errInvalidInstrumentType},
		{name: "a spread asset channel without a type", arg: SubscriptionArgument{Channel: channelSpreadOrders}},
		{name: "event contract markets", arg: SubscriptionArgument{Channel: channelEventContractMarkets, InstrumentType: instTypeEvents}},
		{name: "event contract markets without a type", arg: SubscriptionArgument{Channel: channelEventContractMarkets}, expErr: errSubscriptionParameter},
		{name: "event contract markets of another type", arg: SubscriptionArgument{Channel: channelEventContractMarkets, InstrumentType: instTypeSpot}, expErr: errSubscriptionParameter},
		{name: "option summary without a family", arg: SubscriptionArgument{Channel: channelOptionSummary}, expErr: errSubscriptionParameter},
		{name: "grid positions without an algo ID", arg: SubscriptionArgument{Channel: channelGridPositions}, expErr: errSubscriptionParameter},
		{name: "grid sub orders without an algo ID", arg: SubscriptionArgument{Channel: channelGridSubOrders}, expErr: errSubscriptionParameter},
		{name: "a channel the package does not describe", arg: SubscriptionArgument{Channel: "an undescribed channel"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkSubscriptionArgument(&tc.arg), tc.expErr, "checkSubscriptionArgument should return the expected error")
		})
	}
}

func TestSubscriptionKey(t *testing.T) {
	t.Parallel()
	const tickers, account, configured = `{"channel":"tickers","instId":"BTC-USDT"}`, `{"channel":"account","ccy":"USDT"}`, `{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}`
	spot := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: tickers}
	key := subscriptionKey{spot}
	for _, tc := range []struct {
		name  string
		other subscription.MatchableKey
		exp   bool
	}{
		{name: "the same subscription", other: subscriptionKey{spot.Clone()}, exp: true},
		{name: "an exact key of the same subscription", other: &subscription.ExactKey{Subscription: spot.Clone()}, exp: true},
		{name: "another asset", other: subscriptionKey{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, QualifiedChannel: tickers}}},
		{name: "another argument", other: subscriptionKey{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"ETH-USDT"}`}}},
		{name: "no key", other: nil},
		{name: "a key without a subscription", other: subscriptionKey{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, key.Match(tc.other), "Match should compare the argument beside the exact key's fields")
		})
	}

	t.Run("parameters set apart subscriptions of one channel", func(t *testing.T) {
		t.Parallel()
		store := subscription.NewStore()
		plain := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: account}
		plain.SetKey(subscriptionKey{plain})
		withInterval := &subscription.Subscription{Channel: channelAccount, QualifiedChannel: configured}
		withInterval.SetKey(subscriptionKey{withInterval})
		require.NoError(t, store.Add(plain), "Add must not error")
		require.NoError(t, store.Add(withInterval), "a subscription of other parameters must be added beside the first")
		assert.Same(t, withInterval, store.Get(subscriptionKey{&subscription.Subscription{Channel: channelAccount, QualifiedChannel: configured}}), "the store should find the subscription of its own parameters")
	})

	assert.Same(t, spot, key.GetSubscription(), "GetSubscription should return the keyed subscription")
	assert.JSONEq(t, tickers, key.GetQualifiedChannel(), "GetQualifiedChannel should return the argument")
	assert.Equal(t, "ticker spot BTC/USDT "+tickers, key.String(), "String should name the exact key's fields and the argument")
}

func TestSubscriptionRequestSize(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	sizes := make(chan int, 1)
	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
			sizes <- len(msg)
			return nil
		})
	}))
	conn, err := ex.Websocket.CreateTestConnection(publicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	conn.SetURL("ws" + strings.TrimPrefix(srv.URL, "http"))
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
	t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
	req := &WebsocketSubscriptionRequest{ID: ex.MessageID(), Operation: operationSubscribe, Arguments: []SubscriptionArgument{{Channel: channelAccount, Currency: currency.USDT, ExtraParams: `{"updateInterval":"<0>"}`}, {Channel: channelTickers, InstrumentID: mainPair}}}
	size, err := subscriptionRequestSize(req)
	require.NoError(t, err, "subscriptionRequestSize must not error")
	require.NoError(t, conn.SendJSONMessage(t.Context(), websocketRequestEPL, req), "SendJSONMessage must not error")
	select {
	case sent := <-sizes:
		assert.Equal(t, sent, size, "subscriptionRequestSize should measure the frame the connection sends")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the request must reach the peer")
	}
}

func TestIndexInstrumentID(t *testing.T) {
	t.Parallel()
	for in, exp := range map[currency.Pair]string{
		mainPair:          "BTC-USDT",
		perpetualSwapPair: "BTC-USDT",
		currency.NewPairWithDelimiter("BTC", "USD_UM-261225", currency.DashDelimiter):      "BTC-USD",
		currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", currency.DashDelimiter): "BTC-USD",
	} {
		assert.Equalf(t, exp, indexInstrumentID(in).String(), "%s should be priced from its index", in)
	}
}

func TestInit(t *testing.T) {
	t.Parallel()
	candlePrefixes := []string{channelCandle, channelIndexCandle, channelMarkPriceCandle, channelSpreadCandle}
	instrumentCandle := websocketChannel{businessConnection, channelScopeInstrument, instrumentAssets}
	indexCandle := websocketChannel{businessConnection, channelScopeIndex, instrumentAssets}
	spreadCandle := websocketChannel{businessConnection, channelScopeSpread, spreadAssets}
	for suffix := range candleIntervals {
		t.Run(suffix, func(t *testing.T) {
			t.Parallel()
			exp := map[string]websocketChannel{channelCandle + suffix: instrumentCandle, channelIndexCandle + suffix: indexCandle}
			if suffix != "1s" {
				exp[channelMarkPriceCandle+suffix] = instrumentCandle
				exp[channelSpreadCandle+suffix] = spreadCandle
			}
			registered := make(map[string]websocketChannel, len(candlePrefixes))
			for _, prefix := range candlePrefixes {
				if c, ok := websocketChannels[prefix+suffix]; ok {
					registered[prefix+suffix] = c
				}
			}
			assert.Equal(t, exp, registered, "init should register the candlestick channels OKX documents for the suffix")
		})
	}
	t.Run("undocumented suffix", func(t *testing.T) {
		t.Parallel()
		for _, prefix := range candlePrefixes {
			assert.NotContainsf(t, websocketChannels, prefix+"7m", "init should not register %s", prefix+"7m")
		}
	})
}

func TestCandleChannelSuffix(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.ThousandMilliseconds: "1s",
		kline.FifteenMin:           "15m",
		kline.FourHour:             "4H",
		kline.SixHour:              "6Hutc",
		kline.OneWeek:              "1Wutc",
		kline.ThreeMonth:           "3Mutc",
	} {
		got, err := candleChannelSuffix(interval)
		require.NoErrorf(t, err, "candleChannelSuffix must not error for %s", interval)
		assert.Equalf(t, exp, got, "%s should use the UTC aligned channel where there is one", interval)
	}
	_, err := candleChannelSuffix(kline.OneYear)
	assert.ErrorIs(t, err, errUnsupportedCandleInterval, "an interval OKX does not stream should error")
}

func TestCandleChannelInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel, prefix string
		exp             kline.Interval
		expErr          error
	}{
		{"candle1m", channelCandle, kline.OneMin, nil},
		{"candle1Dutc", channelCandle, kline.OneDay, nil},
		{"candle1s", channelCandle, kline.ThousandMilliseconds, nil},
		{"sprd-candle3M", channelSpreadCandle, kline.ThreeMonth, nil},
		{"candle1Y", channelCandle, 0, kline.ErrInvalidInterval},
		{"candle", channelCandle, 0, kline.ErrInvalidInterval},
		{"tickers", channelCandle, 0, kline.ErrInvalidInterval},
	} {
		got, err := candleChannelInterval(tc.channel, tc.prefix)
		require.ErrorIsf(t, err, tc.expErr, "candleChannelInterval must return the expected error for %s", tc.channel)
		assert.Equalf(t, tc.exp, got, "%s should map to its interval", tc.channel)
	}
}

func TestIsAssetChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp bool
	}{
		{&subscription.Subscription{Channel: subscription.MyOrdersChannel, Asset: asset.Spot}, true},
		{&subscription.Subscription{Channel: subscription.MyOrdersChannel, Asset: asset.Spread}, true},
		{&subscription.Subscription{Channel: channelOrders}, true},
		{&subscription.Subscription{Channel: channelInstruments, Asset: asset.Margin}, true},
		{&subscription.Subscription{Channel: channelLiquidationOrders, Asset: asset.All}, true},
		{&subscription.Subscription{Channel: subscription.MyAccountChannel}, false},
		{&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}, false},
		{&subscription.Subscription{Channel: channelTickers, Asset: asset.Spot}, false},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.TenMin}, false},
		{&subscription.Subscription{Channel: "unknown"}, false},
	} {
		assert.Equalf(t, tc.exp, isAssetChannel(tc.sub), "%s for %s should be classified by its scope", tc.sub.Channel, tc.sub.Asset)
	}

	t.Run("asset channels subscribe once for every instrument", func(t *testing.T) {
		t.Parallel()
		for _, channel := range []string{subscription.MyOrdersChannel, channelOrders, channelInstruments, channelLiquidationOrders} {
			t.Run(channel, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "test exchange setup must succeed")
				sub := &subscription.Subscription{Enabled: true, Channel: channel, Asset: asset.PerpetualSwap}
				name, err := channelName(sub, asset.PerpetualSwap)
				require.NoError(t, err, "channelName must not error")
				ex.Websocket.SetCanUseAuthenticatedEndpoints(requiresLogin(name))
				ex.Features.Subscriptions = subscription.List{sub}
				subs, err := ex.generateSubscriptions(t.Context(), channelConnection(name))
				require.NoError(t, err, "asset subscription expansion must succeed")
				require.Len(t, subs, 1, "all enabled instruments must share one asset subscription")
				assert.JSONEq(t, fmt.Sprintf(`{"channel":%q,"instType":"SWAP"}`, name), subs[0].QualifiedChannel, "asset channels should omit instrument-specific arguments")
			})
		}
	})
}

func TestPublicLiquidationSubscriptionCycle(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "setup must succeed")
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: channelLiquidationOrders, Asset: asset.PerpetualSwap}}
	public, err := ex.generateSubscriptions(t.Context(), publicConnection)
	require.NoError(t, err, "public subscription expansion must succeed")
	require.Len(t, public, 1, "all enabled swap instruments must share one subscription")
	require.False(t, public[0].Authenticated, "liquidation observations must use public data")
	require.JSONEq(t, `{"channel":"liquidation-orders","instType":"SWAP"}`, public[0].QualifiedChannel, "subscription must match the documented asset scope")
	private, err := ex.generateSubscriptions(t.Context(), privateConnection)
	require.NoError(t, err, "private subscription expansion must succeed")
	require.Empty(t, private, "liquidation observations must not require credentials")

	done := make(chan struct{})
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		assert.Equal(t, "/ws/v5/public", r.URL.Path, "liquidations should use the public socket")
		conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err, "local websocket upgrade should succeed") {
			return
		}
		defer conn.Close()
		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Second)), "mock reads should be bounded")
		for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
			_, payload, err := conn.ReadMessage()
			if !assert.NoError(t, err, "subscription message should arrive") {
				return
			}
			var req WebsocketSubscriptionRequest
			if !assert.NoError(t, json.Unmarshal(payload, &req), "subscription message should decode") {
				return
			}
			assert.NotEmpty(t, req.ID, "the request should carry an ID for its reply to echo")
			assert.JSONEq(t, fmt.Sprintf(`{"id":%q,"op":%q,"args":[{"channel":"liquidation-orders","instType":"SWAP"}]}`, req.ID, operation), string(payload), "wire request should contain only documented arguments")
			assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","event":"`+operation+`","arg":{"channel":"liquidation-orders","instType":"SWAP"},"connId":"a4d3ae55"}`)), "acknowledgement should write")
			if operation == operationSubscribe {
				assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"arg":{"channel":"liquidation-orders","instType":"SWAP"},"data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","details":[{"bkPx":"60000","posSide":"short","side":"buy","sz":"13","ts":"1700000000000"}]}]}`)), "liquidation push should write")
			}
		}
	}))
	conn, err := ex.Websocket.CreateTestConnection(publicConnection)
	require.NoError(t, err, "public connection must be available without authentication")
	require.Contains(t, conn.GetURL(), "/ws/v5/public", "default public connection must use the documented route")
	conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v5/public")
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "local public socket must connect")
	read := make(chan struct{})
	go func() {
		defer close(read)
		for {
			response := conn.ReadMessage()
			if response.Raw == nil {
				return
			}
			assert.NoError(t, ex.wsHandleData(context.Background(), conn, response.Raw), "each frame should be handled")
		}
	}()
	t.Cleanup(func() {
		assert.NoError(t, conn.Shutdown(), "local socket should close")
		select {
		case <-read:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the reader should stop once the socket closes")
		}
	})
	require.NoError(t, ex.Subscribe(t.Context(), conn, public), "asset subscription must be acknowledged")
	exp := &WebsocketLiquidationOrdersPush{
		Argument: SubscriptionArgument{Channel: channelLiquidationOrders, InstrumentType: instTypeSwap},
		Data: []WebsocketLiquidationOrder{{
			InstrumentType: instTypeSwap,
			InstrumentID:   perpetualSwapPair,
			Details:        []WebsocketLiquidationOrderDetail{{Side: "buy", PositionSide: positionSideShort, BankruptcyPrice: 60000, Size: 13, Timestamp: types.Time(time.UnixMilli(1700000000000))}},
		}},
	}
	select {
	case response := <-ex.Websocket.DataHandler.C:
		assert.Equal(t, exp, response.Data, "the liquidation push should decode its contracts unconverted")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the liquidation push must be relayed")
	}
	require.NoError(t, ex.Unsubscribe(t.Context(), conn, public), "asset unsubscription must be acknowledged")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "mock subscription cycle must complete")
	}
}

func TestSentSubscriptions(t *testing.T) {
	t.Parallel()
	const btcUSDTTickers, btcUSDTTrades = `{"channel":"tickers","instId":"BTC-USDT"}`, `{"channel":"trades","instId":"BTC-USDT"}`
	const oneMinute, fiveMinutes = `{"channel":"candle1m","instId":"BTC-USDT"}`, `{"channel":"candle5m","instId":"BTC-USDT"}`
	sub := func(channel string, a asset.Item, qualifiedChannel string) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Asset: a, Pairs: currency.Pairs{mainPair}, QualifiedChannel: qualifiedChannel}
	}
	candles := func(a asset.Item, interval kline.Interval, qualifiedChannel string) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: a, Pairs: currency.Pairs{mainPair}, Interval: interval, QualifiedChannel: qualifiedChannel}
	}
	const account, configuredAccount = `{"channel":"account","ccy":"USDT"}`, `{"channel":"account","ccy":"USDT","extraParams":"{\"updateInterval\":\"0\"}"}`
	for _, tc := range []struct {
		name           string
		held           *subscription.Subscription
		tracked        bool
		heldListed     bool
		heldEquivalent bool
		elsewhere      *subscription.Subscription
		noConn         bool
		subs           subscription.List
		exp            []bool
	}{
		{name: "no subscriptions", exp: []bool{}},
		{name: "distinct arguments", subs: subscription.List{sub(channelTickers, asset.Spot, btcUSDTTickers), sub(channelTrades, asset.Spot, btcUSDTTrades), sub(channelTickers, asset.PerpetualSwap, `{"channel":"tickers","instId":"BTC-USDT-SWAP"}`)}, exp: []bool{true, true, true}},
		{name: "spot and margin of one argument", subs: subscription.List{sub(channelTickers, asset.Spot, btcUSDTTickers), sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true, false}},
		{name: "margin and spot of one argument", subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers), sub(channelTickers, asset.Spot, btcUSDTTickers)}, exp: []bool{true, false}},
		{name: "channels of one asset spelled apart with one argument", subs: subscription.List{sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), sub(channelTickers, asset.Spot, btcUSDTTickers)}, exp: []bool{true, false}},
		{name: "spot and margin of different intervals", subs: subscription.List{candles(asset.Spot, kline.OneMin, oneMinute), candles(asset.Margin, kline.FiveMin, fiveMinutes)}, exp: []bool{true, true}},
		{name: "spot and margin of one interval", subs: subscription.List{candles(asset.Spot, kline.OneMin, oneMinute), candles(asset.Margin, kline.OneMin, oneMinute)}, exp: []bool{true, false}},
		{name: "an argument a tracked subscription on the connection keeps", held: sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), tracked: true, subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers), sub(channelTrades, asset.Margin, btcUSDTTrades)}, exp: []bool{false, true}},
		{name: "a subscription the connection holds untracked keeps nothing", held: sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true}},
		{name: "a subscription tracked on another connection keeps nothing", elsewhere: sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true}},
		{name: "a listed subscription the connection holds keeps nothing", held: sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), tracked: true, heldListed: true, subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true, false}},
		{name: "an equivalent of a subscription the connection holds stands for it", held: sub(subscription.TickerChannel, asset.Spot, btcUSDTTickers), tracked: true, heldEquivalent: true, subs: subscription.List{sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true, false}},
		{
			name: "a subscription with another argument does not stand for one the connection holds", held: &subscription.Subscription{Channel: channelAccount, QualifiedChannel: account}, tracked: true,
			subs: subscription.List{{Channel: channelAccount, QualifiedChannel: configuredAccount}, {Channel: subscription.MyAccountChannel, QualifiedChannel: account}}, exp: []bool{true, false},
		},
		{name: "no connection", noConn: true, subs: subscription.List{sub(channelTickers, asset.Spot, btcUSDTTickers), sub(channelTickers, asset.Margin, btcUSDTTickers)}, exp: []bool{true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
			require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
			subs := tc.subs
			if tc.held != nil {
				require.NoError(t, conn.Subscriptions().Add(tc.held), "the connection must hold the subscription")
				if tc.tracked {
					require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, tc.held), "AddSuccessfulSubscriptions must not error")
				}
				if tc.heldListed {
					subs = append(subscription.List{tc.held}, subs...)
				}
				if tc.heldEquivalent {
					subs = append(subscription.List{tc.held.Clone()}, subs...)
				}
			}
			if tc.elsewhere != nil {
				other := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, other), "TrackTestConnection must not error")
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(other, tc.elsewhere), "AddSuccessfulSubscriptions must not error")
				require.NoError(t, other.Subscriptions().Add(tc.elsewhere), "the other connection must hold the subscription")
			}
			var c websocket.Connection = conn
			if tc.noConn {
				c = nil
			}
			exp := make(map[*subscription.Subscription]bool)
			for i, s := range subs {
				if tc.exp[i] {
					exp[s] = true
				}
			}
			assert.Equal(t, exp, ex.sentSubscriptions(c, subs), "a subscription should send its argument unless an earlier one or one the connection keeps does")
		})
	}
}

func TestInverseSpotMarginSubscription(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name     string
		sub      *subscription.Subscription
		expOK    bool
		expAsset asset.Item
	}{
		{name: "nil", sub: nil, expOK: false},
		{name: "spot to margin", sub: &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: true, expAsset: asset.Margin},
		{name: "margin to spot", sub: &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: true, expAsset: asset.Spot},
		{name: "non spot margin", sub: &subscription.Subscription{Asset: asset.USDTMarginedFutures, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: false},
		{name: "asset channel", sub: &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.MyOrdersChannel}, expOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inverse, ok := inverseSpotMarginSubscription(tc.sub)
			require.Equal(t, tc.expOK, ok, "inverseSpotMarginSubscription must report whether an inverse exists")
			if !tc.expOK {
				require.Nil(t, inverse, "no inverse must be returned")
				return
			}
			require.NotNil(t, inverse, "the inverse must be returned")
			require.NotSame(t, tc.sub, inverse, "the inverse must be a copy")
			require.Equal(t, tc.expAsset, inverse.Asset, "the inverse must hold the other asset")
			require.Equal(t, tc.sub.Channel, inverse.Channel, "the inverse must keep the channel")
			require.Equal(t, tc.sub.Pairs, inverse.Pairs, "the inverse must keep the pairs")
		})
	}
}

func TestTrackEquivalentSubscriptionsOnExistingConnection(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{channelInstruments, channelLiquidationOrders, channelOrders, subscription.MyOrdersChannel} {
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			t.Run(channel+"/"+a.String(), func(t *testing.T) {
				t.Parallel()
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
				other := asset.Margin
				if a == asset.Margin {
					other = asset.Spot
				}
				name, err := channelName(&subscription.Subscription{Channel: channel}, a)
				require.NoError(t, err, "channelName must not error")
				incoming := &subscription.Subscription{
					Asset: a, Channel: channel, Pairs: currency.Pairs{currency.NewBTCUSDT()},
					QualifiedChannel: fmt.Sprintf(`{"channel":%q,"instType":%q}`, name, GetInstrumentTypeFromAssetItem(a)),
				}
				existing := incoming.Clone()
				existing.Asset = other
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, existing), "existing asset must be tracked by the manager")
				require.NoError(t, conn.subscriptions.Add(existing), "existing asset must belong to the connection")
				remaining, attached, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), conn, subscription.List{incoming})
				require.NoError(t, err, "asset-scoped subscription must be evaluated")
				require.Equal(t, subscription.List{incoming}, remaining, "different instrument type must remain for outbound subscription")
				require.Empty(t, attached, "asset-scoped subscriptions must not be attached as equivalents")
				require.Empty(t, conn.Requests(), "equivalence hook must leave outbound handling to the caller")
			})
		}
	}

	newEquivalentSubs := func() (*subscription.Subscription, *subscription.Subscription) {
		pair := currency.NewBTCUSDT()
		marginSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		spotSub := &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		marginSub.QualifiedChannel = `{"channel":"tickers","instId":"BTC-USDT"}`
		spotSub.QualifiedChannel = marginSub.QualifiedChannel
		return marginSub, spotSub
	}

	t.Run("TracksEquivalentOnOwningConnection", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
		marginSub, spotSub := newEquivalentSubs()

		existingConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(existingConn, marginSub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, existingConn.subscriptions.Add(marginSub), "the connection must hold the margin subscription")

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{spotSub})
		require.NoError(t, err, "trackEquivalentSubscriptionsOnExistingConnection must not error")
		require.Empty(t, remaining, "equivalent spot subscription must be tracked on the existing connection")
		require.Len(t, trackedSubs, 1, "the spot subscription must be tracked")
		require.Same(t, spotSub, trackedSubs[0], "the spot subscription itself must be tracked")
		require.Empty(t, existingConn.Requests(), "tracking on an existing connection must not emit a new outbound subscribe request")
		require.Nil(t, tracked.Websocket.GetSubscription(spotSub), "hook must not mutate manager-level tracking state directly")
		require.Len(t, existingConn.subscriptions.List(), 1, "hook must not mutate connection subscription stores directly")
	})

	t.Run("SkipsConnectionWithoutInverse", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
		marginSub, spotSub := newEquivalentSubs()

		// The inverse (margin sub) lives in the manager-level store via a
		// different connection, but NOT on wrongConn's own subscription store.
		otherConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(otherConn, marginSub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, otherConn.subscriptions.Add(marginSub), "the other connection must hold the margin subscription")

		wrongConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), wrongConn, subscription.List{spotSub})
		require.NoError(t, err, "trackEquivalentSubscriptionsOnExistingConnection must not error")
		require.Len(t, remaining, 1, "spot sub must NOT be tracked on a connection that doesn't own the inverse")
		require.Same(t, spotSub, remaining[0], "the spot subscription must remain")
		require.Empty(t, trackedSubs, "nothing must be tracked")
		require.Empty(t, wrongConn.subscriptions.List(), "wrongConn must not gain any subscriptions")
	})

	t.Run("waits for OKX to acknowledge an equivalent it must send", func(t *testing.T) {
		t.Parallel()
		for _, rejected := range []bool{false, true} {
			t.Run(fmt.Sprintf("rejected=%t", rejected), func(t *testing.T) {
				t.Parallel()
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
				marginSub, spotSub := newEquivalentSubs()
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				if rejected {
					conn.reply = rejectArguments(0)
				}
				// Only the connection holds the margin subscription, so the spot argument is sent
				require.NoError(t, conn.subscriptions.Add(marginSub), "the connection must hold the margin subscription")
				remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), conn, subscription.List{spotSub})
				requests := conn.Requests()
				require.Len(t, requests, 1, "the spot argument must be sent")
				assert.Equal(t, []SubscriptionArgument{{Channel: channelTickers, InstrumentID: mainPair}}, requests[0].Arguments, "the spot argument should be sent")
				if rejected {
					require.ErrorIs(t, err, errSubscriptionNotAcknowledged, "a rejected equivalent must be reported")
					require.ErrorIs(t, err, errWebsocketRequestRejected, "OKX's rejection must be reported")
					assert.Empty(t, trackedSubs, "a rejected equivalent should not be tracked")
					return
				}
				require.NoError(t, err, "trackEquivalentSubscriptionsOnExistingConnection must not error")
				assert.Empty(t, remaining, "the acknowledged equivalent should not remain")
				assert.Equal(t, subscription.List{spotSub}, trackedSubs, "the acknowledged equivalent should be tracked")
			})
		}
	})

	t.Run("RefreshesOrderbookSnapshotWhenTrackingEquivalentReenable", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
		tracked.Name = t.Name()

		pair := currency.NewBTCUSDT()
		spotSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`,
		}
		marginSub := &subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: spotSub.QualifiedChannel,
		}

		existingConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(existingConn, spotSub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, existingConn.subscriptions.Add(spotSub), "the connection must hold the spot subscription")

		spotSnapshot := &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Spot,
			LastUpdateID:      123,
			LastUpdated:       time.Unix(123, 0),
			Bids:              orderbook.Levels{{Price: 99, Amount: 2}},
			Asks:              orderbook.Levels{{Price: 100, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), spotSnapshot), "LoadSnapshot must not error")
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Margin,
			LastUpdateID:      7,
			LastUpdated:       time.Unix(7, 0),
			Bids:              orderbook.Levels{{Price: 1, Amount: 1}},
			Asks:              orderbook.Levels{{Price: 2, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}), "LoadSnapshot must not error")

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{marginSub})
		require.NoError(t, err, "trackEquivalentSubscriptionsOnExistingConnection must not error")
		require.Empty(t, remaining, "the margin subscription must be tracked")
		require.Len(t, trackedSubs, 1, "the margin subscription must be tracked")
		require.Same(t, marginSub, trackedSubs[0], "the margin subscription itself must be tracked")
		require.Empty(t, existingConn.Requests(), "equivalent re-enable remains a logical track, not a new outbound subscribe")

		marginBook, err := tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.NoError(t, err, "GetOrderbook must not error")
		require.Equal(t, spotSnapshot.LastUpdateID, marginBook.LastUpdateID, "the margin book must take the spot book's sequence")
		require.Equal(t, spotSnapshot.LastUpdated, marginBook.LastUpdated, "the margin book must take the spot book's time")
		require.Equal(t, spotSnapshot.Bids, marginBook.Bids, "the margin book must take the spot book's bids")
		require.Equal(t, spotSnapshot.Asks, marginBook.Asks, "the margin book must take the spot book's asks")
	})
}

func TestWebsocketHandlePush(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		fixture string
		exp     []any
	}{
		{
			fixture: "account",
			exp: []any{
				&WebsocketAccountPush{
					Argument: SubscriptionArgument{
						Channel: "account",
						UserID:  "44*********584",
					},
					EventType:   "snapshot",
					CurrentPage: 1,
					LastPage:    true,
					Data: []WebsocketAccount{
						{
							UpdateTime:                   types.Time(time.UnixMilli(1705564223311)),
							TotalEquity:                  55868.06403501676,
							IsolatedMarginEquity:         120.5,
							AdjustedEquity:               55444.12216906034,
							AvailableEquity:              55444.12216906034,
							OrderFrozen:                  10.5,
							InitialMarginRequirement:     36.2,
							MaintenanceMarginRequirement: 18.1,
							BorrowFrozen:                 5.5,
							MarginRatio:                  57.36,
							NotionalUSD:                  1500,
							NotionalUSDForBorrow:         200,
							NotionalUSDForSwap:           600,
							NotionalUSDForFutures:        300,
							NotionalUSDForOption:         400,
							UnrealisedProfitAndLoss:      -7.4758,
							Delta:                        1250,
							DeltaLeverage:                0.0224,
							DeltaNeutralStatus:           "1",
							Details: []WebsocketAccountDetail{
								{
									Currency:                           currency.USDT,
									Equity:                             4892.951170691435,
									CashBalance:                        4750.426970691436,
									UpdateTime:                         types.Time(time.UnixMilli(1705564213903)),
									IsolatedMarginEquity:               100,
									AvailableEquity:                    4734.371190691435,
									DiscountEquity:                     4889.379316336831,
									FixedBalance:                       12,
									AvailableBalance:                   4734.371190691436,
									FrozenBalance:                      158.57998,
									OrderFrozen:                        20,
									Liabilities:                        4.6,
									UnrealisedProfitAndLoss:            -7.475800000000003,
									UnrealisedProfitAndLossLiabilities: 0.7,
									CrossLiabilities:                   3.5,
									IsolatedLiabilities:                1.1,
									RewardBalance:                      5,
									MarginRatio:                        32.5,
									InitialMarginRequirement:           2.5,
									MaintenanceMarginRequirement:       1.25,
									Interest:                           0.05,
									ForcedRepaymentIndicator:           1,
									ForcedRepaymentType:                "1",
									MaximumLoan:                        5000,
									EquityUSD:                          4889.379316336831,
									BorrowFrozen:                       1.2,
									NotionalLeverage:                   0.8,
									CoinUSDPrice:                       0.99927,
									StrategyEquity:                     150,
									IsolatedUnrealisedProfitAndLoss:    -0.6,
									SpotInUseAmount:                    0.4,
									ClientSpotInUseAmount:              0.3,
									MaximumSpotInUseAmount:             0.6,
									SpotIsolatedBalance:                2,
									SmartSyncEquity:                    25,
									SpotCopyTradingEquity:              30,
									SpotBalance:                        4750.42,
									OpenAveragePrice:                   0.9991,
									AccumulatedAveragePrice:            0.9989,
									SpotUnrealisedProfitAndLoss:        -1.2,
									SpotUnrealisedProfitAndLossRatio:   -0.0003,
									TotalProfitAndLoss:                 -2.4,
									TotalProfitAndLossRatio:            -0.0005,
									CollateralRestriction:              "1",
									CollateralBorrowAutoConversion:     "2",
									CollateralRestrict:                 true,
									CollateralEnabled:                  true,
									AutoLendStatus:                     "active",
									AutoLendMatchedAmount:              100,
								},
							},
						},
					},
				},
				&WebsocketAccountPush{
					Argument: SubscriptionArgument{
						Channel: "account",
						UserID:  "44*********584",
					},
					EventType: "event_update",
					Data: []WebsocketAccount{
						{
							UpdateTime:                   types.Time(time.UnixMilli(1705564223311)),
							TotalEquity:                  55868.06403501676,
							IsolatedMarginEquity:         120.5,
							AdjustedEquity:               55444.12216906034,
							AvailableEquity:              55444.12216906034,
							OrderFrozen:                  10.5,
							InitialMarginRequirement:     36.2,
							MaintenanceMarginRequirement: 18.1,
							BorrowFrozen:                 5.5,
							MarginRatio:                  57.36,
							NotionalUSD:                  1500,
							NotionalUSDForBorrow:         200,
							NotionalUSDForSwap:           600,
							NotionalUSDForFutures:        300,
							NotionalUSDForOption:         400,
							UnrealisedProfitAndLoss:      -7.4758,
							Delta:                        1250,
							DeltaLeverage:                0.0224,
							DeltaNeutralStatus:           "1",
							Details: []WebsocketAccountDetail{
								{
									Currency:                           currency.USDT,
									Equity:                             4892.951170691435,
									CashBalance:                        4750.426970691436,
									UpdateTime:                         types.Time(time.UnixMilli(1705564213903)),
									IsolatedMarginEquity:               100,
									AvailableEquity:                    4734.371190691435,
									DiscountEquity:                     4889.379316336831,
									FixedBalance:                       12,
									AvailableBalance:                   4734.371190691436,
									FrozenBalance:                      158.57998,
									OrderFrozen:                        20,
									Liabilities:                        4.6,
									UnrealisedProfitAndLoss:            -7.475800000000003,
									UnrealisedProfitAndLossLiabilities: 0.7,
									CrossLiabilities:                   3.5,
									IsolatedLiabilities:                1.1,
									RewardBalance:                      5,
									MarginRatio:                        32.5,
									InitialMarginRequirement:           2.5,
									MaintenanceMarginRequirement:       1.25,
									Interest:                           0.05,
									ForcedRepaymentIndicator:           1,
									ForcedRepaymentType:                "1",
									MaximumLoan:                        5000,
									EquityUSD:                          4889.379316336831,
									BorrowFrozen:                       1.2,
									NotionalLeverage:                   0.8,
									CoinUSDPrice:                       0.99927,
									StrategyEquity:                     150,
									IsolatedUnrealisedProfitAndLoss:    -0.6,
									SpotInUseAmount:                    0.4,
									ClientSpotInUseAmount:              0.3,
									MaximumSpotInUseAmount:             0.6,
									SpotIsolatedBalance:                2,
									SmartSyncEquity:                    25,
									SpotCopyTradingEquity:              30,
									SpotBalance:                        4750.42,
									OpenAveragePrice:                   0.9991,
									AccumulatedAveragePrice:            0.9989,
									SpotUnrealisedProfitAndLoss:        -1.2,
									SpotUnrealisedProfitAndLossRatio:   -0.0003,
									TotalProfitAndLoss:                 -2.4,
									TotalProfitAndLossRatio:            -0.0005,
									CollateralRestriction:              "1",
									CollateralBorrowAutoConversion:     "2",
									CollateralRestrict:                 true,
									CollateralEnabled:                  true,
									AutoLendStatus:                     "active",
									AutoLendMatchedAmount:              100,
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "positions",
			exp: []any{
				&WebsocketPositionsPush{
					Argument: SubscriptionArgument{
						Channel:        "positions",
						InstrumentType: "ANY",
						UserID:         "77982378738415879",
					},
					EventType:   "snapshot",
					CurrentPage: 1,
					LastPage:    true,
					Data: []WebsocketPosition{
						{
							InstrumentType:                        "FUTURES",
							MarginMode:                            "isolated",
							PositionID:                            "307173036051017730",
							PositionSide:                          "long",
							Position:                              1,
							HedgedPosition:                        0.5,
							BaseBalance:                           0.15,
							QuoteBalance:                          9000,
							BaseBorrowed:                          0.1,
							BaseInterest:                          0.0001,
							QuoteBorrowed:                         6150,
							QuoteInterest:                         0.6,
							PositionCurrency:                      currency.BTC,
							AvailablePosition:                     1,
							AveragePrice:                          2566.31,
							UnrealisedProfitAndLoss:               -0.0000009932766034,
							UnrealisedProfitAndLossRatio:          -0.0025490556801078,
							UnrealisedProfitAndLossLastPrice:      -0.0000009932766034,
							UnrealisedProfitAndLossRatioLastPrice: -0.0025490556801078,
							InstrumentID:                          currency.NewPairWithDelimiter("BTC", "USD-261127", "-"),
							Leverage:                              10,
							LiquidationPrice:                      2352.8496681818233,
							MarkPrice:                             2353.849,
							InitialMarginRequirement:              0.0002,
							Margin:                                0.0003896645377994,
							MarginRatio:                           11.731726509588816,
							MaintenanceMarginRequirement:          0.0000311811092368,
							Liabilities:                           0.01,
							LiabilitiesCurrency:                   currency.USDT,
							Interest:                              0.00001,
							TradeID:                               "109844",
							NotionalUSD:                           2276.2546609009605,
							OptionValue:                           0.0012,
							PendingCloseOrderLiabilityValue:       0.1,
							AutoDeleveragingIndicator:             "1",
							BusinessReferenceID:                   "1234567890",
							BusinessReferenceType:                 "coupon",
							Currency:                              currency.BTC,
							LastPrice:                             2566.22,
							IndexPrice:                            2566.13,
							USDPrice:                              61500.2,
							BreakEvenPrice:                        2353.949,
							DeltaBlackScholes:                     0.52,
							DeltaPriceAdjusted:                    0.0081,
							GammaBlackScholes:                     0.0001,
							GammaPriceAdjusted:                    0.0002,
							ThetaBlackScholes:                     -12.5,
							ThetaPriceAdjusted:                    -0.0003,
							VegaBlackScholes:                      35.1,
							VegaPriceAdjusted:                     0.0006,
							SpotInUseAmount:                       0.2,
							SpotInUseCurrency:                     currency.BTC,
							ClientSpotInUseAmount:                 0.1,
							MaximumSpotInUseAmount:                0.3,
							RealisedProfitAndLoss:                 0.001,
							ProfitAndLoss:                         0.0011,
							Fee:                                   -0.0001,
							FundingFee:                            -0.00002,
							LiquidationPenalty:                    -0.00003,
							CloseOrderAlgo: []WebsocketPositionCloseOrderAlgo{
								{
									AlgoID:                     "123",
									StopLossTriggerPrice:       123,
									StopLossTriggerPriceType:   "mark",
									TakeProfitTriggerPrice:     123,
									TakeProfitTriggerPriceType: "mark",
									CloseFraction:              0.6,
								},
								{
									AlgoID:                     "123",
									StopLossTriggerPrice:       123,
									StopLossTriggerPriceType:   "mark",
									TakeProfitTriggerPrice:     123,
									TakeProfitTriggerPriceType: "mark",
									CloseFraction:              0.4,
								},
							},
							CreationTime:              types.Time(time.UnixMilli(1619507758793)),
							UpdateTime:                types.Time(time.UnixMilli(1619507761462)),
							PushTime:                  types.Time(time.UnixMilli(1619507761462)),
							NonSettlementAveragePrice: 61400.5,
							SettledProfitAndLoss:      0.0004,
						},
					},
				},
			},
		},
		{
			fixture: "liquidation-warning",
			exp: []any{
				&WebsocketPositionRiskWarningPush{
					Argument: SubscriptionArgument{
						Channel:        "liquidation-warning",
						InstrumentType: "FUTURES",
						UserID:         "77982378738415879",
					},
					Data: []WebsocketPositionRiskWarning{
						{
							InstrumentType:   "FUTURES",
							MarginMode:       "isolated",
							PositionID:       "307173036051017730",
							PositionSide:     "long",
							Position:         1,
							PositionCurrency: currency.BTC,
							InstrumentID:     currency.NewPairWithDelimiter("BTC", "USD-261127", "-"),
							Leverage:         10,
							MarkPrice:        2353.849,
							MarginRatio:      11.731726509588816,
							Currency:         currency.BTC,
							CreationTime:     types.Time(time.UnixMilli(1619507758793)),
							UpdateTime:       types.Time(time.UnixMilli(1619507761462)),
							PushTime:         types.Time(time.UnixMilli(1619507761462)),
						},
					},
				},
			},
		},
		{
			fixture: "account-greeks",
			exp: []any{
				&WebsocketAccountGreeksPush{
					Argument: SubscriptionArgument{
						Channel: "account-greeks",
						UserID:  "614488474791936",
					},
					Data: []WebsocketAccountGreeks{
						{
							DeltaBlackScholes:  1.1246665403011684,
							DeltaPriceAdjusted: -0.0074021163991037,
							GammaPriceAdjusted: 0.0148042327982075,
							ThetaBlackScholes:  2.1342098201092528,
							ThetaPriceAdjusted: -0.0000000200876441,
							Currency:           currency.BTC,
							Timestamp:          types.Time(time.UnixMilli(1729179001692)),
						},
						{
							DeltaBlackScholes:  0.381067016169857,
							DeltaPriceAdjusted: -0.0688347042402955,
							GammaBlackScholes:  -0.0000000000230396,
							GammaPriceAdjusted: 0.137669348344032,
							ThetaBlackScholes:  0.3314776517141782,
							ThetaPriceAdjusted: 0.0000000001316008,
							VegaBlackScholes:   -0.0000000045069794,
							VegaPriceAdjusted:  -0.0000000000017267,
							Currency:           currency.ETH,
							Timestamp:          types.Time(time.UnixMilli(1729179001692)),
						},
					},
				},
				&WebsocketAccountGreeksPush{
					Argument: SubscriptionArgument{
						Channel:  "account-greeks",
						Currency: currency.BTC,
						UserID:   "614488474791936",
					},
					Data: []WebsocketAccountGreeks{
						{
							DeltaBlackScholes:  1.124666540194431,
							DeltaPriceAdjusted: -0.0074076183688949,
							GammaPriceAdjusted: 0.0148152367377899,
							ThetaBlackScholes:  2.0356991946421226,
							ThetaPriceAdjusted: -0.0000000200174309,
							Currency:           currency.BTC,
							Timestamp:          types.Time(time.UnixMilli(1729179082006)),
						},
					},
				},
			},
		},
		{
			fixture: "orders-algo",
			exp: []any{
				&WebsocketAlgoOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "orders-algo",
						InstrumentType: "SPOT",
						InstrumentID:   currency.NewPairWithDelimiter("BTC", "USDC", "-"),
						UserID:         "77982378738415879",
					},
					Data: []WebsocketAlgoOrder{
						{
							InstrumentType: "SPOT",
							InstrumentID:   currency.NewPairWithDelimiter("BTC", "USDC", "-"),
							Currency:       currency.USDC,
							OrderID:        "581878926302093313",
							OrderIDList: []string{
								"581878926302093313",
							},
							SubAlgoIDList: []string{
								"581878926302093314",
							},
							AlgoID:                 "581878926302093312",
							ClientOrderID:          "b16",
							Size:                   11,
							OrderType:              "conditional",
							Side:                   order.Buy,
							PositionSide:           "net",
							TradeMode:              "cross",
							SizeType:               "quote_ccy",
							Leverage:               3,
							State:                  "live",
							TakeProfitTriggerPrice: 1,
							TakeProfitTriggerType:  "last",
							TakeProfitOrderPrice:   -1,
							StopLossTriggerPrice:   25000,
							StopLossTriggerType:    "last",
							StopLossOrderPrice:     -1,
							TriggerPrice:           26100,
							TriggerPriceType:       "mark",
							OrderPrice:             26170,
							AdvanceOrderType:       "chase",
							ChaseParameters: []WebsocketAlgoChaseParameters{
								{
									ChaseType:         "distance",
									ChaseValue:        0.1,
									MaximumChaseType:  "ratio",
									MaximumChaseValue: 0.05,
								},
							},
							LastPrice:               26174.8,
							ActualSize:              11,
							ActualPrice:             26174.9,
							NotionalUSD:             11,
							Tag:                     "gct",
							ActualSide:              "tp",
							TriggerTime:             types.Time(time.UnixMilli(1708679675240)),
							ReduceOnly:              "true",
							CloseFraction:           1,
							QuickMarginType:         "manual",
							FailCode:                "51008",
							AlgoClientOrderID:       "algo-client-2",
							RequestID:               "r2",
							AmendResult:             "0",
							AmendPriceOnTriggerType: "0",
							AttachedAlgoOrders: []WebsocketAlgoAttachedOrder{
								{
									AttachAlgoClientOrderID: "attach-2",
									TakeProfitTriggerPrice:  27000,
									TakeProfitTriggerRatio:  0.3,
									TakeProfitTriggerType:   "last",
									TakeProfitOrderPrice:    -1,
									StopLossTriggerPrice:    25000,
									StopLossTriggerRatio:    0.1,
									StopLossTriggerType:     "index",
									StopLossOrderPrice:      -1,
									CallbackRatio:           0.05,
									CallbackSpread:          10,
									ActivationPrice:         26500,
								},
							},
							LinkedOrder: WebsocketAlgoLinkedOrder{
								OrderID: "98192973880283",
							},
							CreationTime:       types.Time(time.UnixMilli(1685002746818)),
							UpdateTime:         types.Time(time.UnixMilli(1708679675245)),
							IsTradeBorrowMode:  "true",
							ChaseType:          "distance",
							ChaseValue:         0.1,
							MaximumChaseType:   "ratio",
							MaximumChaseValue:  0.05,
							TradeQuoteCurrency: currency.USDT,
						},
					},
				},
			},
		},
		{
			fixture: "algo-advance",
			exp: []any{
				&WebsocketAdvanceAlgoOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "algo-advance",
						InstrumentType: "SPOT",
						InstrumentID:   mainPair,
						UserID:         "77982378738415879",
					},
					Data: []WebsocketAdvanceAlgoOrder{
						{
							InstrumentType:         "SPOT",
							InstrumentID:           mainPair,
							Currency:               currency.USDT,
							OrderID:                "355056228680335361",
							AlgoID:                 "355056228680335360",
							ClientOrderID:          "b17",
							Size:                   0.1,
							OrderType:              "iceberg",
							Side:                   order.Buy,
							PositionSide:           "net",
							TradeMode:              "cash",
							SizeType:               "base_ccy",
							Leverage:               2,
							State:                  "pause",
							TakeProfitTriggerPrice: 63000,
							TakeProfitOrderPrice:   -1,
							StopLossTriggerPrice:   59000,
							StopLossOrderPrice:     -1,
							TriggerPrice:           60500,
							OrderPrice:             61000,
							ActualSize:             0.05,
							ActualPrice:            61000.1,
							NotionalUSD:            6100,
							Tag:                    "adadadadad",
							ActualSide:             "buy",
							TriggerTime:            types.Time(time.UnixMilli(1630924295200)),
							CreationTime:           types.Time(time.UnixMilli(1630924001545)),
							PriceVariation:         0.01,
							PriceSpread:            1,
							SizeLimit:              0.1,
							PriceLimit:             10,
							TimeInterval:           "60",
							Count:                  1,
							CallbackRatio:          0.05,
							CallbackSpread:         10,
							ActivationPrice:        61500,
							MoveTriggerPrice:       60900,
							FailCode:               "51008",
							AlgoClientOrderID:      "algo-client-3",
							ReduceOnly:             "true",
							PushTime:               types.Time(time.UnixMilli(1630924295204)),
							IsTradeBorrowMode:      true,
							TradeQuoteCurrency:     currency.USDT,
						},
					},
				},
			},
		},
		{
			fixture: "grid-orders-spot",
			exp: []any{
				&WebsocketSpotGridAlgoOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "grid-orders-spot",
						InstrumentType: "ANY",
						UserID:         "44705892343619584",
					},
					Data: []WebsocketSpotGridAlgoOrder{
						{
							AlgoID:            "568028283477164032",
							AlgoClientOrderID: "grid-client-1",
							InstrumentType:    "SPOT",
							InstrumentID:      mainPair,
							CreationTime:      types.Time(time.UnixMilli(1681700496249)),
							UpdateTime:        types.Time(time.UnixMilli(1682406665527)),
							AlgoOrderType:     "grid",
							State:             "running",
							RebateTransfers: []WebsocketGridRebateTransfer{
								{
									Rebate:         0.0001,
									RebateCurrency: currency.BTC,
								},
								{
									RebateCurrency: currency.USDT,
								},
							},
							TriggerParameters: []WebsocketGridTriggerParameters{
								{
									TriggerAction:    "start",
									TriggerStrategy:  "rsi",
									DelaySeconds:     5,
									TriggerTime:      types.Time(time.UnixMilli(1681700496250)),
									TriggerType:      "auto",
									Timeframe:        "15m",
									Threshold:        30,
									TriggerCondition: "cross_up",
									TimePeriod:       14,
								},
								{
									TriggerAction:   "stop",
									TriggerStrategy: "price",
									TriggerType:     "manual",
									TriggerPrice:    5500,
									StopType:        "1",
								},
							},
							MaximumPrice:           5000,
							MinimumPrice:           400,
							GridNumber:             10,
							RunType:                "1",
							TakeProfitTriggerPrice: 6000,
							StopLossTriggerPrice:   300,
							TradeNumber:            4,
							ArbitrageNumber:        2,
							SingleAmount:           0.00101214,
							PerMinimumProfitRate:   0.09912004405286344,
							PerMaximumProfitRate:   1.14570215,
							RunPrice:               30031.7,
							TotalProfitAndLoss:     0.5,
							ProfitAndLossRatio:     0.02,
							Investment:             25,
							GridProfit:             0.2,
							FloatProfit:            0.3,
							TotalAnnualisedRate:    0.15,
							AnnualisedRate:         0.12,
							CancelType:             "0",
							StopType:               "2",
							QuoteSize:              25,
							BaseSize:               0.001,
							CurrentQuoteSize:       25,
							CurrentBaseSize:        0.0005,
							Profit:                 0.5,
							StopResult:             "1",
							ActiveOrderNumber:      10,
							Tag:                    "gct",
							ProfitSharingRatio:     0.1,
							CopyType:               "0",
							PushTime:               types.Time(time.UnixMilli(1682416738467)),
							TradeQuoteCurrency:     currency.USDT,
						},
					},
				},
			},
		},
		{
			fixture: "grid-orders-contract",
			exp: []any{
				&WebsocketContractGridAlgoOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "grid-orders-contract",
						InstrumentType: "ANY",
						UserID:         "4470****9584",
					},
					Data: []WebsocketContractGridAlgoOrder{
						{
							AlgoID:            "571039869070475264",
							AlgoClientOrderID: "grid-client-2",
							InstrumentType:    "SWAP",
							InstrumentID:      perpetualSwapPair,
							CreationTime:      types.Time(time.UnixMilli(1682418514204)),
							UpdateTime:        types.Time(time.UnixMilli(1682492552257)),
							AlgoOrderType:     "contract_grid",
							State:             "running",
							RebateTransfers: []WebsocketGridRebateTransfer{
								{
									Rebate:         0.0002,
									RebateCurrency: currency.USDT,
								},
							},
							TriggerParameters: []WebsocketGridTriggerParameters{
								{
									TriggerAction:   "start",
									TriggerStrategy: "price",
									TriggerTime:     types.Time(time.UnixMilli(1682418561497)),
									TriggerType:     "manual",
									TriggerPrice:    1,
								},
								{
									TriggerAction:   "stop",
									TriggerStrategy: "instant",
									TriggerType:     "manual",
									StopType:        "1",
								},
							},
							MaximumPrice:           36437.3,
							MinimumPrice:           26931.9,
							GridNumber:             10,
							RunType:                "1",
							TakeProfitTriggerPrice: 40000,
							StopLossTriggerPrice:   20000,
							TradeNumber:            9,
							ArbitrageNumber:        3,
							SingleAmount:           1,
							PerMinimumProfitRate:   0.1263869357706788,
							PerMaximumProfitRate:   0.1687494513302446,
							RunPrice:               27306.9,
							TotalProfitAndLoss:     8.794565238708968,
							ProfitAndLossRatio:     0.0879456523870897,
							Investment:             100,
							GridProfit:             1.2,
							FloatProfit:            8.794565238708968,
							TotalAnnualisedRate:    38.52019574554529,
							AnnualisedRate:         0.2,
							CancelType:             "1",
							StopType:               "0",
							Direction:              "long",
							BasePosition:           true,
							Size:                   100,
							Leverage:               5,
							ActualLeverage:         2.348149463527665,
							LiquidationPrice:       16370.482143120824,
							OrderFrozen:            5.38638,
							AvailableEquity:        52.301539288708966,
							Equity:                 108.79456523870897,
							ActiveOrderNumber:      10,
							Tag:                    "gct",
							ProfitSharingRatio:     0.1,
							CopyType:               "1",
							TakeProfitRatio:        0.1,
							StopLossRatio:          0.05,
							Fee:                    -0.12,
							FundingFee:             -0.03,
							PushTime:               types.Time(time.UnixMilli(1682492574068)),
						},
					},
				},
			},
		},
		{
			fixture: "grid-positions",
			exp: []any{
				&WebsocketGridPositionsPush{
					Argument: SubscriptionArgument{
						Channel: "grid-positions",
						AlgoID:  "449327675342323712",
						UserID:  "4470****9584",
					},
					Data: []WebsocketGridPosition{
						{
							AlgoID:                       "449327675342323712",
							AlgoClientOrderID:            "grid-client-2",
							InstrumentType:               "SWAP",
							InstrumentID:                 perpetualSwapPair,
							CreationTime:                 types.Time(time.UnixMilli(1653400065917)),
							UpdateTime:                   types.Time(time.UnixMilli(1653445498682)),
							AveragePrice:                 29181.463888888888,
							Currency:                     currency.USDT,
							Leverage:                     5,
							LiquidationPrice:             604.7617536513744,
							PositionSide:                 "net",
							Position:                     35,
							MarginMode:                   "cross",
							MarginRatio:                  217.71740878394456,
							InitialMarginRequirement:     2089.2690000000002,
							MaintenanceMarginRequirement: 41.78538,
							UnrealisedProfitAndLoss:      232.83263888888962,
							UnrealisedProfitAndLossRatio: 0.1139826489932205,
							LastPrice:                    29852.7,
							NotionalUSD:                  10435.794191550001,
							AutoDeleveragingIndicator:    "1",
							MarkPrice:                    29849.7,
							PushTime:                     types.Time(time.UnixMilli(1653536068723)),
						},
					},
				},
			},
		},
		{
			fixture: "grid-sub-orders",
			exp: []any{
				&WebsocketGridSubOrdersPush{
					Argument: SubscriptionArgument{
						Channel: "grid-sub-orders",
						AlgoID:  "449327675342323712",
						UserID:  "44705892343619584",
					},
					Data: []WebsocketGridSubOrder{
						{
							AlgoID:              "449327675342323712",
							AlgoClientOrderID:   "grid-client-2",
							InstrumentType:      "SWAP",
							InstrumentID:        perpetualSwapPair,
							AlgoOrderType:       "contract_grid",
							GroupID:             "-1",
							OrderID:             "449518234142904321",
							CreationTime:        types.Time(time.UnixMilli(1653445498664)),
							UpdateTime:          types.Time(time.UnixMilli(1653445498674)),
							TradeMode:           "cross",
							Tag:                 "gct",
							OrderType:           "limit",
							Size:                1,
							State:               "live",
							Side:                order.Buy,
							Price:               28007.2,
							Fee:                 -0.01,
							FeeCurrency:         currency.USDT,
							Rebate:              0.001,
							RebateCurrency:      currency.USDT,
							AveragePrice:        28007.1,
							AccumulatedFillSize: 0.5,
							PositionSide:        "net",
							ProfitAndLoss:       0.3,
							ContractValue:       0.01,
							Leverage:            5,
							PushTime:            types.Time(time.UnixMilli(1653486524502)),
						},
					},
				},
			},
		},
		{
			fixture: "algo-recurring-buy",
			exp: []any{
				&WebsocketRecurringBuyOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "algo-recurring-buy",
						InstrumentType: "SPOT",
						UserID:         "447*******584",
					},
					Data: []WebsocketRecurringBuyOrder{
						{
							AlgoID:            "644497312047435776",
							AlgoClientOrderID: "recurring-client-1",
							InstrumentType:    "SPOT",
							CreationTime:      types.Time(time.UnixMilli(1699932133373)),
							UpdateTime:        types.Time(time.UnixMilli(1699932136249)),
							AlgoOrderType:     "recurring",
							State:             "running",
							StrategyName:      "stg1",
							RecurringList: []WebsocketRecurringBuyItem{
								{
									Currency:     currency.BTC,
									Ratio:        0.2,
									MinimumPrice: 30000,
									MaximumPrice: 50000,
									TotalAmount:  0.0011,
									Profit:       1.2,
									AveragePrice: 36000,
									Price:        36482,
								},
								{
									Currency:     currency.ETH,
									Ratio:        0.8,
									TotalAmount:  0.08,
									Profit:       4.3,
									AveragePrice: 2000,
									Price:        2057.54,
								},
							},
							Period:               "hourly",
							RecurringDay:         "1",
							RecurringHour:        "1",
							RecurringTime:        "12",
							TimeZone:             "8",
							Amount:               100,
							InvestmentAmount:     200,
							InvestmentCurrency:   currency.USDC,
							NextInvestTime:       types.Time(time.UnixMilli(1699934415300)),
							TotalProfitAndLoss:   5.5,
							TotalAnnualisedRate:  0.12,
							ProfitAndLossRatio:   0.0275,
							MarketCapitalisation: 205.5,
							Cycles:               2,
							Tag:                  "gct",
							PushTime:             types.Time(time.UnixMilli(1699933314691)),
							TradeQuoteCurrency:   currency.USDT,
							RecurringTimeType:    "hourly",
							RecurringTimeMinutes: "30",
							Source: []string{
								"1",
							},
						},
					},
				},
			},
		},
		{
			fixture: "copytrading-lead-notification",
			exp: []any{
				&WebsocketLeadTradingNotificationPush{
					Argument: SubscriptionArgument{
						Channel:        "copytrading-lead-notification",
						InstrumentType: "SWAP",
						UserID:         "525627088439549953",
					},
					Data: []WebsocketLeadTradingNotification{
						{
							InstrumentType:      "SWAP",
							InfoType:            "2",
							SubPositionID:       "667695035433385984",
							UniqueCode:          "3AF72F63E3EAD701",
							MaxLeadTraderNumber: 3,
						},
						{
							InstrumentType:      "SWAP",
							InfoType:            "1",
							SubPositionID:       "667695035433385985",
							UniqueCode:          "3AF72F63E3EAD701",
							InstrumentID:        perpetualSwapPair,
							Side:                "buy",
							PositionSide:        "long",
							MaxLeadTraderNumber: 3,
							MinimumLeadEquity:   1000,
						},
					},
				},
			},
		},
		{
			fixture: "bbo-rpi-tbt",
			exp: []any{
				&WebsocketOrderBookPush{
					Argument: SubscriptionArgument{
						Channel:      "bbo-rpi-tbt",
						InstrumentID: mainPair,
					},
					Data: []WebsocketOrderBook{
						{
							Asks: [][4]types.Number{
								{
									61503.9,
									5.04705961,
									0,
									1,
								},
							},
							Bids: [][4]types.Number{
								{
									61501.1,
									2.44258483,
									0,
									1,
								},
							},
							Timestamp:  types.Time(time.UnixMilli(1795405196709)),
							SequenceID: 84749422527,
						},
					},
				},
			},
		},
		{
			fixture: "call-auction-details",
			exp: []any{
				&WebsocketCallAuctionDetailsPush{
					Argument: SubscriptionArgument{
						Channel:      "call-auction-details",
						InstrumentID: currency.NewPairWithDelimiter("ONDO", "USDC", "-"),
					},
					Data: []WebsocketCallAuctionDetails{
						{
							InstrumentID:     currency.NewPairWithDelimiter("ONDO", "USDC", "-"),
							EquilibriumPrice: 0.6,
							MatchedSize:      44978,
							UnmatchedSize:    9988764,
							AuctionEndTime:   types.Time(time.Unix(1726542000, 0)),
							State:            "continuous_trading",
							Timestamp:        types.Time(time.UnixMilli(1726542000007)),
						},
					},
				},
			},
		},
		{
			fixture: "rfqs",
			exp: []any{
				&WebsocketRequestForQuotesPush{
					Argument: SubscriptionArgument{
						Channel: "rfqs",
						UserID:  "77982378738415879",
					},
					Data: []WebsocketRequestForQuote{
						{
							CreationTime: types.Time(time.UnixMilli(1611033737572)),
							UpdateTime:   types.Time(time.UnixMilli(1611033737572)),
							State:        "active",
							Counterparties: []string{
								"DSK4",
								"DSK5",
							},
							ValidUntil:              types.Time(time.UnixMilli(1611033857557)),
							ClientRequestForQuoteID: "rfq-client-1",
							Tag:                     "123456",
							FlowType:                "normal",
							TraderCode:              "DSK2",
							RequestForQuoteID:       "22534",
							AllowPartialExecution:   true,
							Legs: []WebsocketBlockTradeLeg{
								{
									InstrumentID:       currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", "-"),
									TradeMode:          "cross",
									Currency:           currency.USDT,
									Size:               25,
									Side:               order.Buy,
									PositionSide:       "long",
									SizeType:           "base_ccy",
									TradeQuoteCurrency: currency.USDT,
								},
								{
									InstrumentID:       currency.NewPairWithDelimiter("BTC", "USD-261127-65000-C", "-"),
									TradeMode:          "cross",
									Currency:           currency.USDT,
									Size:               25,
									Side:               order.Sell,
									PositionSide:       "long",
									SizeType:           "base_ccy",
									TradeQuoteCurrency: currency.USDT,
								},
							},
							GroupID: "22533",
							AccountAllocations: []WebsocketAccountAllocation{
								{
									Account: "sub1",
									Legs: []WebsocketAccountAllocationLeg{
										{
											InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", "-"),
											Size:         25,
											TradeMode:    "cross",
											Currency:     currency.USDT,
											PositionSide: "long",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "quotes",
			exp: []any{
				&WebsocketQuotesPush{
					Argument: SubscriptionArgument{
						Channel: "quotes",
						UserID:  "77982378738415879",
					},
					Data: []WebsocketQuote{
						{
							CreationTime:            types.Time(time.UnixMilli(1608267227834)),
							UpdateTime:              types.Time(time.UnixMilli(1608267227834)),
							State:                   "canceled",
							Reason:                  "mmp_canceled",
							ValidUntil:              types.Time(time.UnixMilli(1608997227854)),
							RequestForQuoteID:       "18753",
							ClientRequestForQuoteID: "rfq-client-1",
							QuoteID:                 "25092",
							ClientQuoteID:           "quote-client-1",
							Tag:                     "123456",
							TraderCode:              "SATS",
							QuoteSide:               order.Sell,
							Legs: []WebsocketBlockTradeLeg{
								{
									InstrumentID:       currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", "-"),
									TradeMode:          "cross",
									Currency:           currency.USDT,
									Size:               25,
									Price:              0.0023,
									Side:               order.Sell,
									PositionSide:       "long",
									SizeType:           "base_ccy",
									TradeQuoteCurrency: currency.USDT,
								},
								{
									InstrumentID:       currency.NewPairWithDelimiter("BTC", "USD-261127-65000-C", "-"),
									TradeMode:          "cross",
									Currency:           currency.USDT,
									Size:               25,
									Price:              0.0045,
									Side:               order.Buy,
									PositionSide:       "long",
									SizeType:           "base_ccy",
									TradeQuoteCurrency: currency.USDT,
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "struc-block-trades",
			exp: []any{
				&WebsocketStructureBlockTradesPush{
					Argument: SubscriptionArgument{
						Channel: "struc-block-trades",
						UserID:  "77982378738415879",
					},
					Data: []WebsocketStructureBlockTrade{
						{
							CreationTime:            types.Time(time.UnixMilli(1608267227834)),
							RequestForQuoteID:       "18753",
							ClientRequestForQuoteID: "rfq-client-1",
							QuoteID:                 "25092",
							ClientQuoteID:           "quote-client-1",
							BlockTradeID:            "180184",
							Tag:                     "123456",
							TakerTraderCode:         "ANAND",
							MakerTraderCode:         "WAGMI",
							IsSuccessful:            true,
							ErrorCode:               "0",
							Legs: []WebsocketStructureBlockTradeLeg{
								{
									InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", "-"),
									Price:        0.0023,
									Size:         25,
									Side:         order.Sell,
									SizeType:     "base_ccy",
									Fee:          0.1001,
									FeeCurrency:  currency.BTC,
									TradeID:      "10211",
								},
								{
									InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261127-65000-C", "-"),
									Price:        0.0033,
									Size:         25,
									Side:         order.Buy,
									SizeType:     "base_ccy",
									Fee:          0.1001,
									FeeCurrency:  currency.BTC,
									TradeID:      "10212",
								},
							},
							AccountAllocations: []WebsocketBlockTradeAllocation{
								{
									BlockTradeID: "180184",
									ErrorCode:    "0",
									Account:      "sub1",
									Legs: []WebsocketBlockTradeAllocationLeg{
										{
											InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261127-60000-C", "-"),
											Size:         25,
											TradeID:      "10211",
											Fee:          -0.1001,
											FeeCurrency:  currency.BTC,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "public-struc-block-trades",
			exp: []any{
				&WebsocketPublicStructureBlockTradesPush{
					Argument: SubscriptionArgument{
						Channel: "public-struc-block-trades",
					},
					Data: []WebsocketPublicStructureBlockTrade{
						{
							CreationTime: types.Time(time.UnixMilli(1795401770927)),
							BlockTradeID: "4123818584240009216",
							Strategy:     "CUSTOM",
							Legs: []WebsocketPublicBlockTradeLeg{
								{
									InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261124-60500-C", "-"),
									Price:        0.0128,
									Size:         4,
									Side:         order.Sell,
									TradeID:      "4123818584248397824",
								},
								{
									InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261124-62000-C", "-"),
									Price:        0.0045,
									Size:         4,
									Side:         order.Buy,
									TradeID:      "4123818584248397825",
								},
								{
									InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261124-62500-C", "-"),
									Price:        0.0031,
									Size:         2,
									Side:         order.Buy,
									TradeID:      "4123818584248397826",
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "block-tickers",
			exp: []any{
				&WebsocketBlockTickersPush{
					Argument: SubscriptionArgument{
						Channel:      "block-tickers",
						InstrumentID: currency.NewPairWithDelimiter("BTC", "USD-261127-50000-P", "-"),
					},
					Data: []WebsocketBlockTicker{
						{
							InstrumentID:         currency.NewPairWithDelimiter("BTC", "USD-261127-50000-P", "-"),
							InstrumentType:       "OPTION",
							VolumeCurrency24Hour: 0.05,
							Volume24Hour:         5,
							Timestamp:            types.Time(time.UnixMilli(1795404987314)),
						},
					},
				},
			},
		},
		{
			fixture: "event-contract-markets",
			exp: []any{
				&WebsocketEventContractMarketsPush{
					Argument: SubscriptionArgument{
						Channel:        "event-contract-markets",
						InstrumentType: "EVENTS",
					},
					Data: []WebsocketEventContractMarket{
						{
							SeriesID:     "BTC-UPDOWN-5MIN",
							EventID:      "BTC-UPDOWN-5MIN-261123-1140-1145",
							InstrumentID: currency.NewPairWithDelimiter("BTC", "UPDOWN-5MIN-261123-1140-1145", "-"),
							ListTime:     types.Time(time.Unix(1795403347, 0)),
							FixTime:      types.Time(time.UnixMilli(1795405201033)),
							ExpiryTime:   types.Time(time.Unix(1795405500, 0)),
							State:        "live",
							Outcome:      "0",
							FloorStrike:  61499.5,
						},
					},
				},
				&WebsocketEventContractMarketsPush{
					Argument: SubscriptionArgument{
						Channel:        "event-contract-markets",
						InstrumentType: "EVENTS",
					},
					Data: []WebsocketEventContractMarket{
						{
							SeriesID:     "BTC-UPDOWN-5MIN",
							EventID:      "BTC-UPDOWN-5MIN-261123-1135-1140",
							InstrumentID: currency.NewPairWithDelimiter("BTC", "UPDOWN-5MIN-261123-1135-1140", "-"),
							ListTime:     types.Time(time.UnixMilli(1795403349838)),
							FixTime:      types.Time(time.UnixMilli(1795404903034)),
							ExpiryTime:   types.Time(time.Unix(1795405200, 0)),
							State:        "expired",
							Outcome:      "2",
							FloorStrike:  61666.5,
							SettleValue:  61499.113,
						},
					},
				},
			},
		},
		{
			fixture: "open-interest",
			exp: []any{
				&WebsocketOpenInterestPush{
					Argument: SubscriptionArgument{
						Channel:      "open-interest",
						InstrumentID: perpetualSwapPair,
					},
					Data: []WebsocketOpenInterest{
						{
							InstrumentType:       "SWAP",
							InstrumentID:         perpetualSwapPair,
							OpenInterest:         3135943.750000013,
							OpenInterestCurrency: 31359.43750000013,
							OpenInterestUSD:      1927743021.718758,
							Timestamp:            types.Time(time.UnixMilli(1795405180951)),
						},
					},
				},
			},
		},
		{
			fixture: "funding-rate",
			exp: []any{
				&WebsocketFundingRatePush{
					Argument: SubscriptionArgument{
						Channel:      "funding-rate",
						InstrumentID: perpetualSwapPair,
					},
					Data: []WebsocketFundingRate{
						{
							InstrumentType:        "SWAP",
							InstrumentID:          perpetualSwapPair,
							Method:                "current_period",
							FormulaType:           "withRate",
							FundingRate:           0.0000228511443102,
							FundingTime:           types.Time(time.Unix(1795420800, 0)),
							NextFundingTime:       types.Time(time.Unix(1795449600, 0)),
							PreviousFundingTime:   types.Time(time.Unix(1795392000, 0)),
							MinimumFundingRate:    -0.00375,
							MaximumFundingRate:    0.00375,
							InterestRate:          0.0001,
							ImpactValue:           20000,
							SettlementState:       "settled",
							SettlementFundingRate: 0.0000151578247325,
							Premium:               -0.000421019093785,
							Timestamp:             types.Time(time.UnixMilli(1795405168472)),
						},
					},
				},
			},
		},
		{
			fixture: "price-limit",
			exp: []any{
				&WebsocketPriceLimitPush{
					Argument: SubscriptionArgument{
						Channel:      "price-limit",
						InstrumentID: perpetualSwapPair,
					},
					Data: []WebsocketPriceLimit{
						{
							InstrumentType: "SWAP",
							InstrumentID:   perpetualSwapPair,
							BuyLimit:       61782.9,
							SellLimit:      61167.8,
							Timestamp:      types.Time(time.UnixMilli(1795405193917)),
							Enabled:        true,
						},
					},
				},
			},
		},
		{
			fixture: "opt-summary",
			exp: []any{
				&WebsocketOptionSummaryPush{
					Argument: SubscriptionArgument{
						Channel:          "opt-summary",
						InstrumentFamily: "XAU-USD_UM",
					},
					Data: []WebsocketOptionSummary{
						{
							InstrumentType:       "OPTION",
							InstrumentID:         currency.NewPairWithDelimiter("XAU", "USD_UM-261124-3475-P", "-"),
							Underlying:           "XAU-USD",
							Delta:                -0.09170925,
							Gamma:                0.00158206,
							Vega:                 0.33803538,
							Theta:                -7.25911279,
							DeltaBlackScholes:    -0.09170925,
							GammaBlackScholes:    0.00158206,
							VegaBlackScholes:     0.33803538,
							ThetaBlackScholes:    -7.25911279,
							Leverage:             0.22248587,
							MarkVolatility:       0.50705825,
							BidVolatility:        0.49195237,
							AskVolatility:        0.5456632,
							AtTheMoneyVolatility: 0.43588062,
							ForwardPrice:         3609.4,
							BuyAnnualisedReturn:  0.34324088,
							SellAnnualisedReturn: 0.5062803,
							Distance:             -0.03723611,
							Timestamp:            types.Time(time.UnixMilli(1795405196827)),
						},
						{
							InstrumentType:       "OPTION",
							InstrumentID:         currency.NewPairWithDelimiter("XAU", "USD_UM-261123-3350-C", "-"),
							Underlying:           "XAU-USD",
							Delta:                0.9997693,
							Gamma:                0.00001114,
							Vega:                 0.0006895,
							Theta:                -0.03255672,
							DeltaBlackScholes:    0.9997693,
							GammaBlackScholes:    0.00001114,
							VegaBlackScholes:     0.0006895,
							ThetaBlackScholes:    -0.03255672,
							Leverage:             0.00385499,
							MarkVolatility:       0.95962919,
							AtTheMoneyVolatility: 0.43963757,
							ForwardPrice:         3609.4,
							Distance:             -0.0718679,
							Timestamp:            types.Time(time.UnixMilli(1795405196827)),
						},
					},
				},
			},
		},
		{
			fixture: "estimated-price",
			exp: []any{
				&WebsocketEstimatedPricePush{
					Argument: SubscriptionArgument{
						Channel:          "estimated-price",
						InstrumentType:   "FUTURES",
						InstrumentFamily: "XRP-USDT",
					},
					Data: []WebsocketEstimatedPrice{
						{
							InstrumentType:  "FUTURES",
							InstrumentID:    currency.NewPairWithDelimiter("XRP", "USDT-250307", currency.DashDelimiter),
							SettlementType:  "settlement",
							SettlementPrice: 2.4230631578947368,
							Timestamp:       types.Time(time.UnixMilli(1741244598708)),
						},
					},
				},
			},
		},
		{
			fixture: "mark-price",
			exp: []any{
				&WebsocketMarkPricePush{
					Argument: SubscriptionArgument{
						Channel:      "mark-price",
						InstrumentID: perpetualSwapPair,
					},
					Data: []WebsocketMarkPrice{
						{
							InstrumentType: "SWAP",
							InstrumentID:   perpetualSwapPair,
							MarkPrice:      61475.7,
							Timestamp:      types.Time(time.UnixMilli(1795405194146)),
						},
					},
				},
			},
		},
		{
			fixture: "index-tickers",
			exp: []any{
				&WebsocketIndexTickersPush{
					Argument: SubscriptionArgument{
						Channel:      "index-tickers",
						InstrumentID: mainPair,
					},
					Data: []WebsocketIndexTicker{
						{
							InstrumentID:       mainPair,
							IndexPrice:         61501.6,
							OpenPrice24Hour:    59217.5,
							HighestPrice24Hour: 61957.9,
							LowestPrice24Hour:  58775.5,
							OpenPriceUTC0:      61269.9,
							OpenPriceUTC8:      60890.8,
							Timestamp:          types.Time(time.UnixMilli(1795405194106)),
						},
					},
				},
			},
		},
		{
			fixture: "mark-price-candle1m",
			exp: []any{
				&WebsocketMarkPriceCandlesticksPush{
					Argument: SubscriptionArgument{
						Channel:      "mark-price-candle1m",
						InstrumentID: perpetualSwapPair,
					},
					Data: []WebsocketPriceCandlestick{
						{
							Timestamp:    types.Time(time.Unix(1795405200, 0)),
							OpenPrice:    61472.3,
							HighestPrice: 61473.7,
							LowestPrice:  61367,
							ClosePrice:   61367,
							Confirm:      true,
						},
					},
				},
			},
		},
		{
			fixture: "index-candle1m",
			exp: []any{
				&WebsocketIndexCandlesticksPush{
					Argument: SubscriptionArgument{
						Channel:      "index-candle1m",
						InstrumentID: mainPair,
					},
					Data: []WebsocketPriceCandlestick{
						{
							Timestamp:    types.Time(time.Unix(1795405200, 0)),
							OpenPrice:    61499.5,
							HighestPrice: 61500.5,
							LowestPrice:  61397.4,
							ClosePrice:   61397.4,
							Confirm:      true,
						},
					},
				},
			},
		},
		{
			fixture: "liquidation-orders",
			exp: []any{
				&WebsocketLiquidationOrdersPush{
					Argument: SubscriptionArgument{
						Channel:        "liquidation-orders",
						InstrumentType: "SWAP",
					},
					Data: []WebsocketLiquidationOrder{
						{
							InstrumentType:   "SWAP",
							InstrumentID:     currency.NewPairWithDelimiter("WLD", "USDT-SWAP", "-"),
							InstrumentFamily: "WLD-USDT",
							Underlying:       "WLD-USDT",
							Details: []WebsocketLiquidationOrderDetail{
								{
									Side:            "buy",
									PositionSide:    "short",
									BankruptcyPrice: 0.2365,
									Size:            2786,
									Timestamp:       types.Time(time.UnixMilli(1795370514833)),
								},
							},
						},
					},
				},
			},
		},
		{
			fixture: "adl-warning",
			exp: []any{
				&WebsocketADLWarningPush{
					Argument: SubscriptionArgument{
						Channel:          "adl-warning",
						InstrumentType:   "FUTURES",
						InstrumentFamily: "BTC-USDT",
					},
					Data: []WebsocketADLWarning{
						{
							InstrumentType:          "FUTURES",
							InstrumentFamily:        "BTC-USDT",
							State:                   "warning",
							Balance:                 280784384.9564228,
							Currency:                currency.USDT,
							MaximumBalance:          290000000,
							MaximumBalanceTimestamp: types.Time(time.Unix(1700210000, 0)),
							ADLBalance:              1000,
							ADLRecoveryBalance:      2000,
							Timestamp:               types.Time(time.UnixMilli(1700210763001)),
							DeclineRate:             0.0318,
							ADLRate:                 0.3,
							ADLRecoveryRate:         0.2,
						},
					},
				},
				&WebsocketADLWarningPush{
					Argument: SubscriptionArgument{
						Channel:          "adl-warning",
						InstrumentType:   "FUTURES",
						InstrumentFamily: "BTC-USDT",
					},
					Data: []WebsocketADLWarning{
						{
							InstrumentType:          "FUTURES",
							InstrumentFamily:        "BTC-USDT",
							State:                   "adl",
							Balance:                 280784384.9564228,
							Currency:                currency.USDT,
							MaximumBalance:          290000000,
							MaximumBalanceTimestamp: types.Time(time.Unix(1700210000, 0)),
							ADLType:                 "rate_adl_start",
							ADLBalance:              1000,
							ADLRecoveryBalance:      2000,
							Timestamp:               types.Time(time.UnixMilli(1700210764001)),
							DeclineRate:             0.0318,
							ADLRate:                 0.3,
							ADLRecoveryRate:         0.2,
						},
					},
				},
			},
		},
		{
			fixture: "economic-calendar",
			exp: []any{
				&WebsocketEconomicCalendarPush{
					Argument: SubscriptionArgument{
						Channel: "economic-calendar",
					},
					Data: []WebsocketEconomicCalendar{
						{
							Event:           "S&P Global Manufacturing PMI Final",
							Region:          "United States",
							Category:        "Manufacturing PMI",
							Actual:          49.2,
							Previous:        47.3,
							Forecast:        49.3,
							PreviousInitial: 47.1,
							Date:            types.Time(time.UnixMilli(1597026383085)),
							ReferenceDate:   types.Time(time.UnixMilli(1597026383085)),
							CalendarID:      "319275",
							Unit:            "%",
							Currency:        currency.USD,
							Importance:      "2",
							Timestamp:       types.Time(time.UnixMilli(1698648096590)),
						},
					},
				},
			},
		},
		{
			fixture: "deposit-info",
			exp: []any{
				&WebsocketDepositInfoPush{
					Argument: SubscriptionArgument{
						Channel: "deposit-info",
						UserID:  "289320****60975104",
					},
					Data: []WebsocketDepositInfo{
						{
							UserID:                     "289320****60975104",
							SubAccount:                 "test",
							PushTime:                   types.Time(time.UnixMilli(1674103661147)),
							Currency:                   currency.USDT,
							Chain:                      "USDT-TRC20",
							Amount:                     1,
							From:                       "135****0000",
							AreaCodeFrom:               "86",
							To:                         "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw",
							TransactionID:              "bc5376817*****************dbb0d729f6b",
							Timestamp:                  types.Time(time.UnixMilli(1674103661123)),
							State:                      "2",
							DepositID:                  "88165462",
							FromWithdrawalID:           "63754561",
							ActualDepositBlockConfirms: 18,
						},
					},
				},
			},
		},
		{
			fixture: "withdrawal-info",
			exp: []any{
				&WebsocketWithdrawalInfoPush{
					Argument: SubscriptionArgument{
						Channel: "withdrawal-info",
						UserID:  "289320*****0975104",
					},
					Data: []WebsocketWithdrawalInfo{
						{
							UserID:           "289333*****1101696",
							SubAccount:       "test",
							PushTime:         types.Time(time.UnixMilli(1674103268578)),
							Currency:         currency.USDT,
							Chain:            "USDT-TRC20",
							NonTradableAsset: true,
							Amount:           2,
							Timestamp:        types.Time(time.UnixMilli(1674103268472)),
							From:             "135****0000",
							AreaCodeFrom:     "86",
							To:               "TN8CKTQMnpWfT******8KipbJ24ErguhF",
							AreaCodeTo:       "61",
							ToAddressType:    "1",
							Tag:              "t1",
							PaymentID:        "p1",
							Memo:             "7890",
							AddressExtension: WebsocketWithdrawalAddressExtra{
								"comment": "123456",
							},
							TransactionID: "0x8e7c*****5d3",
							Fee:           0.8,
							FeeCurrency:   currency.USDT,
							State:         "2",
							WithdrawalID:  "63754560",
							ClientID:      "w-client-1",
							Note:          "w1",
						},
					},
				},
			},
		},
		{
			fixture: "status",
			exp: []any{
				&WebsocketStatusPush{
					Argument: SubscriptionArgument{
						Channel: "status",
					},
					Data: []WebsocketStatus{
						{
							Title:           "Copy trading system scheduled maintenance",
							State:           "ongoing",
							Begin:           types.Time(time.Unix(1795071600, 0)),
							End:             types.Time(time.Unix(1795073400, 0)),
							ServiceType:     "11",
							System:          "unified",
							MaintenanceType: "1",
							Environment:     "1",
							Timestamp:       types.Time(time.UnixMilli(1795071600196)),
						},
					},
				},
			},
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			// The account push saves its balances under the login of the connection it arrives on
			creds := &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"}
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(creds)
			conn := new(accountStreamConnection)
			ex.recordWebsocketLogin(conn, creds)
			testexch.FixtureToDataHandler(t, "testdata/websocket/"+tc.fixture+".jsonl", func(ctx context.Context, b []byte) error {
				return ex.wsHandleData(ctx, conn, b)
			})
			got := make([]any, 0, len(tc.exp))
			for len(ex.Websocket.DataHandler.C) > 0 {
				got = append(got, (<-ex.Websocket.DataHandler.C).Data)
			}
			assert.Equal(t, tc.exp, got, "each push should decode every field")
		})
	}

	t.Run("empty grid pushes are ignored", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			channel string
			row     string
			exp     any
		}{
			{
				channel: channelGridPositions,
				row:     `{"algoId":"42","instId":"BTC-USDT-SWAP","pos":"1"}`,
				exp: &WebsocketGridPositionsPush{
					Argument: SubscriptionArgument{Channel: channelGridPositions, AlgoID: "42", UserID: "7"},
					Data:     []WebsocketGridPosition{{AlgoID: "42", InstrumentID: perpetualSwapPair, Position: 1}},
				},
			},
			{
				channel: channelGridSubOrders,
				row:     `{"algoId":"42","instId":"BTC-USDT-SWAP","ordId":"77","sz":"1"}`,
				exp: &WebsocketGridSubOrdersPush{
					Argument: SubscriptionArgument{Channel: channelGridSubOrders, AlgoID: "42", UserID: "7"},
					Data:     []WebsocketGridSubOrder{{AlgoID: "42", InstrumentID: perpetualSwapPair, OrderID: "77", Size: 1}},
				},
			},
		} {
			t.Run(tc.channel, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Websocket.DataHandler = stream.NewRelay(1)
				frame := func(data string) []byte {
					return []byte(`{"arg":{"channel":"` + tc.channel + `","uid":"7","algoId":"42"},"data":` + data + `}`)
				}
				for _, data := range []string{`[]`, `null`} {
					require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, frame(data)), "a push with %s data must not error", data)
				}
				require.NoError(t, ex.wsHandleData(t.Context(), nil, frame(`[`+tc.row+`]`)), "a push with a row must be relayed")
				assert.Equal(t, []any{tc.exp}, relayed(ex), "only the push with a row should be relayed")
			})
		}
	})

	t.Run("index candlestick without its confirm field", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(`{"arg":{"channel":"index-candle30m","instType":"SPOT","instId":"BTC-USDT"},"data":[["1597026383085","3811.31","3811.31","3811.31","3811.31"]]}`)), "wsHandleData must not error")
		exp := &WebsocketIndexCandlesticksPush{
			Argument: SubscriptionArgument{Channel: "index-candle30m", InstrumentType: instTypeSpot, InstrumentID: mainPair},
			Data: []WebsocketPriceCandlestick{{
				Timestamp:    types.Time(time.UnixMilli(1597026383085)),
				OpenPrice:    3811.31,
				HighestPrice: 3811.31,
				LowestPrice:  3811.31,
				ClosePrice:   3811.31,
			}},
		}
		require.Len(t, ex.Websocket.DataHandler.C, 1, "the push must be relayed")
		assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "the index candlestick should decode with its channel's interval in the argument")
	})
	t.Run("documented push schemas", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		testexch.FixtureToDataHandler(t, "testdata/wsPushSchemas.jsonl", func(ctx context.Context, b []byte) error {
			return ex.wsHandleData(ctx, nil, b)
		})
		var (
			priceLimits             []*WebsocketPriceLimitPush
			withdrawals             []*WebsocketWithdrawalInfoPush
			deposits                []*WebsocketDepositInfoPush
			gridPositions           []*WebsocketGridPositionsPush
			accountPushes           []*WebsocketAccountPush
			requestForQuotePushes   []*WebsocketRequestForQuotesPush
			liquidations            []*WebsocketLiquidationOrdersPush
			algoOrderPushes         []*WebsocketAlgoOrdersPush
			advancedAlgoOrderPushes []*WebsocketAdvanceAlgoOrdersPush
			recurringBuyPushes      []*WebsocketRecurringBuyOrdersPush
			spreadTickerBatches     [][]ticker.Price
		)
		for _, v := range relayed(ex) {
			switch v := v.(type) {
			case *WebsocketPriceLimitPush:
				priceLimits = append(priceLimits, v)
			case *WebsocketWithdrawalInfoPush:
				withdrawals = append(withdrawals, v)
			case *WebsocketDepositInfoPush:
				deposits = append(deposits, v)
			case *WebsocketGridPositionsPush:
				gridPositions = append(gridPositions, v)
			case *WebsocketAccountPush:
				accountPushes = append(accountPushes, v)
			case *WebsocketRequestForQuotesPush:
				requestForQuotePushes = append(requestForQuotePushes, v)
			case *WebsocketLiquidationOrdersPush:
				liquidations = append(liquidations, v)
			case *WebsocketAlgoOrdersPush:
				algoOrderPushes = append(algoOrderPushes, v)
			case *WebsocketAdvanceAlgoOrdersPush:
				advancedAlgoOrderPushes = append(advancedAlgoOrderPushes, v)
			case *WebsocketRecurringBuyOrdersPush:
				recurringBuyPushes = append(recurringBuyPushes, v)
			case []ticker.Price:
				spreadTickerBatches = append(spreadTickerBatches, v)
			}
		}

		t.Run("price limit decodes its own fields", func(t *testing.T) {
			t.Parallel()
			require.Len(t, priceLimits, 1, "the price-limit push must decode into the limit price struct")
			require.Len(t, priceLimits[0].Data, 1, "the price limit row must decode")
			assert.Equal(t, 52000.0, priceLimits[0].Data[0].BuyLimit.Float64(), "the documented buyLmt should decode")
			assert.Equal(t, 38000.0, priceLimits[0].Data[0].SellLimit.Float64(), "the documented sellLmt should decode")
			assert.True(t, priceLimits[0].Data[0].Enabled, "the documented enabled flag should decode")
		})

		t.Run("withdrawal info decodes into the withdrawal struct", func(t *testing.T) {
			t.Parallel()
			require.Len(t, withdrawals, 1, "the withdrawal-info push must decode into the withdrawal struct")
			require.Len(t, withdrawals[0].Data, 1, "the withdrawal row must decode")
			assert.Equal(t, "2", withdrawals[0].Data[0].ToAddressType, "the documented toAddrType should decode")
			assert.Equal(t, "w1", withdrawals[0].Data[0].Note, "the documented note should decode")
			assert.Equal(t, "W12345", withdrawals[0].Data[0].WithdrawalID, "the withdrawal id should decode")
		})

		t.Run("deposit info decodes into the deposit struct", func(t *testing.T) {
			t.Parallel()
			require.Len(t, deposits, 1, "the deposit-info push must decode into the deposit struct")
			require.Len(t, deposits[0].Data, 1, "the deposit row must decode")
			assert.Equal(t, "D12345", deposits[0].Data[0].DepositID, "the deposit id should decode")
		})

		t.Run("grid positions decode the position fields", func(t *testing.T) {
			t.Parallel()
			require.Len(t, gridPositions, 1, "the grid-positions push must decode into the grid position struct")
			require.Len(t, gridPositions[0].Data, 1, "the grid position row must decode")
			row := gridPositions[0].Data[0]
			assert.Equal(t, "grid-client-1", row.AlgoClientOrderID, "the documented algoClOrdId should decode")
			assert.Equal(t, 29181.46, row.AveragePrice.Float64(), "the entry price should decode")
			assert.Equal(t, "USDT", row.Currency.String(), "the margin currency should decode")
			assert.Equal(t, 12.5, row.UnrealisedProfitAndLoss.Float64(), "the unrealised PnL should decode")
			assert.Equal(t, 0.012, row.UnrealisedProfitAndLossRatio.Float64(), "the unrealised PnL ratio should decode")
			assert.Equal(t, 35.0, row.Position.Float64(), "the position size should decode")
		})

		t.Run("account snapshot decodes the pagination fields", func(t *testing.T) {
			t.Parallel()
			require.Len(t, accountPushes, 1, "the account push must decode into the account push struct")
			require.Len(t, accountPushes[0].Data, 1, "the account row must decode")
			assert.Equal(t, "snapshot", accountPushes[0].EventType, "the documented eventType should decode")
			assert.Equal(t, uint64(1), accountPushes[0].CurrentPage, "the documented curPage integer should decode")
			assert.True(t, accountPushes[0].LastPage, "the documented lastPage boolean should decode")
		})

		t.Run("rfqs push decodes the millisecond timestamps", func(t *testing.T) {
			t.Parallel()
			require.Len(t, requestForQuotePushes, 1, "the rfqs push must decode into the RFQ push struct")
			require.Len(t, requestForQuotePushes[0].Data, 1, "the rfq row must decode")
			row := requestForQuotePushes[0].Data[0]
			assert.Equal(t, int64(1611033737572), row.CreationTime.Time().UnixMilli(), "the documented cTime millisecond string should decode")
			assert.Equal(t, int64(1611033737572), row.UpdateTime.Time().UnixMilli(), "the documented uTime millisecond string should decode")
			assert.Equal(t, int64(1611033857557), row.ValidUntil.Time().UnixMilli(), "the documented validUntil millisecond string should decode")
			assert.Equal(t, "active", row.State, "the documented state should decode")
			assert.Equal(t, "22534", row.RequestForQuoteID, "the documented rfqId should decode")
		})

		t.Run("liquidation orders decode the envelope data", func(t *testing.T) {
			t.Parallel()
			require.Len(t, liquidations, 1, "the liquidation-orders push must decode into the envelope struct")
			require.Len(t, liquidations[0].Data, 1, "the liquidation order row must decode")
			row := liquidations[0].Data[0]
			assert.Equal(t, "IOST-USDT-SWAP", row.InstrumentID.String(), "the instrument id should decode")
			require.Len(t, row.Details, 1, "the liquidation detail must decode")
			assert.Equal(t, 0.007831, row.Details[0].BankruptcyPrice.Float64(), "the bankruptcy price should decode")
		})

		t.Run("orders-algo push decodes the reduceOnly string", func(t *testing.T) {
			t.Parallel()
			require.Len(t, algoOrderPushes, 1, "the orders-algo push must decode into the algo order struct")
			require.Len(t, algoOrderPushes[0].Data, 1, "the algo order row must decode")
			assert.Equal(t, "true", algoOrderPushes[0].Data[0].ReduceOnly, "the documented reduceOnly string should decode")
			assert.Equal(t, 62916.5, algoOrderPushes[0].Data[0].LastPrice.Float64(), "the last filled price should decode")
		})

		t.Run("algo-advance push decodes the empty reduceOnly form", func(t *testing.T) {
			t.Parallel()
			require.Len(t, advancedAlgoOrderPushes, 1, "the algo-advance push must decode into the advanced algo order struct")
			require.Len(t, advancedAlgoOrderPushes[0].Data, 1, "the advanced algo order row must decode")
			assert.Empty(t, advancedAlgoOrderPushes[0].Data[0].ReduceOnly, "the documented empty reduceOnly form should decode")
		})

		t.Run("algo-recurring-buy push decodes the funding source array", func(t *testing.T) {
			t.Parallel()
			require.Len(t, recurringBuyPushes, 1, "the algo-recurring-buy push must decode into the recurring buy struct")
			require.Len(t, recurringBuyPushes[0].Data, 1, "the recurring buy row must decode")
			row := recurringBuyPushes[0].Data[0]
			assert.Equal(t, []string{"1"}, row.Source, "the documented funding source array should decode")
			assert.Equal(t, "USDT", row.TradeQuoteCurrency.String(), "the documented tradeQuoteCcy should decode")
			require.Len(t, row.RecurringList, 1, "the recurring list must decode")
			assert.Equal(t, 30000.0, row.RecurringList[0].MinimumPrice.Float64(), "the documented minPx should decode")
			assert.Equal(t, 50000.0, row.RecurringList[0].MaximumPrice.Float64(), "the documented maxPx should decode")
		})

		t.Run("sprd-tickers push maps the 24 hour figures into the relayed ticker", func(t *testing.T) {
			t.Parallel()
			require.Len(t, spreadTickerBatches, 1, "the sprd-tickers push must relay a processed ticker batch")
			require.Len(t, spreadTickerBatches[0], 1, "the spread ticker row must relay")
			row := spreadTickerBatches[0][0]
			assert.Equal(t, 4.0, row.Open, "the documented open24h should map into the relayed ticker")
			assert.Equal(t, 14.5, row.High, "the documented high24h should map into the relayed ticker")
			assert.Equal(t, -2.2, row.Low, "the documented low24h should map into the relayed ticker")
			assert.Equal(t, 14.5, row.Last, "the last price should relay")
		})

		t.Run("orders schema decodes every documented field", func(t *testing.T) {
			t.Parallel()
			frame := []byte(`{"arg":{"channel":"orders","instType":"SWAP"},"data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","ordId":"680800019749904384","clOrdId":"ord-client-1","algoId":"algo-1","algoClOrdId":"algo-client-1","attachAlgoClOrdId":"attach-client-1","state":"partially_filled","fillPnl":"12.5","fillPxVol":"0.4","fillPxUsd":"42000","fillMarkVol":"0.39","fillFwdPx":"42001","fillMarkPx":"42000.5","fillIdxPx":"41999","pxUsd":"42000","pxVol":"0.4","pxType":"last","cancelSource":"1","amendSource":"2","stpMode":"cancel_maker","lastPx":"42000.5","reduceOnly":"true","attachAlgoOrds":[{"attachAlgoClOrdId":"tp-1","tpTriggerPx":"50000","tpOrdPx":"-1"}]}]}`)
			var resp WebsocketOrdersPush
			require.NoError(t, json.Unmarshal(frame, &resp), "the orders push schema must decode")
			require.Len(t, resp.Data, 1, "the order row must decode")
			row := resp.Data[0]
			assert.Equal(t, "1", row.CancelSource, "the documented cancelSource should decode")
			assert.Equal(t, "2", row.AmendSource, "the documented amendSource should decode")
			assert.Equal(t, 41999.0, row.FillIndexPrice.Float64(), "the documented fillIdxPx should decode")
			assert.Equal(t, "last", row.PriceType, "the documented pxType should decode")
			assert.True(t, row.ReduceOnly, "the reduceOnly flag should decode from its quoted wire form")
		})
	})

	t.Run("grid order schemas", func(t *testing.T) {
		t.Parallel()
		// The quoted trade count decodes into an unsigned integer, the quoted
		// decimals decode into numbers and the unset trigger prices decode from
		// the empty string
		var spot WebsocketSpotGridAlgoOrdersPush
		require.NoError(t, json.Unmarshal([]byte(`{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`), &spot), "the spot grid push must decode")
		require.Len(t, spot.Data, 1, "the spot grid row must decode")
		spotRow := spot.Data[0]
		assert.Equal(t, uint64(3), spotRow.TradeNumber, "the documented tradeNum quoted count should decode")
		assert.Equal(t, 100.0, spotRow.Investment.Float64(), "the documented investment should decode")
		assert.Equal(t, 30478.1, spotRow.RunPrice.Float64(), "the documented runPx should decode")
		assert.Equal(t, -0.9643551057262827, spotRow.TotalAnnualisedRate.Float64(), "the documented totalAnnualizedRate should decode")
		assert.Zero(t, spotRow.StopLossTriggerPrice.Float64(), "the empty slTriggerPx should decode as zero")

		var contract WebsocketContractGridAlgoOrdersPush
		require.NoError(t, json.Unmarshal([]byte(`{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`), &contract), "the contract grid push must decode")
		require.Len(t, contract.Data, 1, "the contract grid row must decode")
		contractRow := contract.Data[0]
		assert.Equal(t, uint64(37), contractRow.TradeNumber, "the documented tradeNum quoted count should decode")
		assert.Equal(t, 1.02, contractRow.ActualLeverage.Float64(), "the documented actualLever should decode")
		assert.Equal(t, 5.0, contractRow.Leverage.Float64(), "the documented lever should decode")
		assert.Equal(t, 10000.0, contractRow.Investment.Float64(), "the documented investment should decode")
		assert.Zero(t, contractRow.StopLossTriggerPrice.Float64(), "the empty slTriggerPx should decode as zero")
	})
}

var pushDataMap = map[string]string{
	"Algo Orders":                           `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`,
	"Advanced Algo Order":                   `{"arg": {"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`,
	"Position Risk":                         `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Account Greeks":                        `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"RequestForQuotes":                      `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"Accounts":                              `{"arg": {"channel": "account","ccy": "BTC","uid": "77982378738415879"},	"data": [{"uTime": "1597026383085","totalEq": "41624.32","isoEq": "3624.32","adjEq": "41624.32","ordFroz": "0","imr": "4162.33","mmr": "4","notionalUsd": "","mgnRatio": "41624.32","details": [{"availBal": "","availEq": "1","ccy": "BTC","cashBal": "1","uTime": "1617279471503","disEq": "50559.01","eq": "1","eqUsd": "45078.3790756226851775","frozenBal": "0","interest": "0","isoEq": "0","liab": "0","maxLoan": "","mgnRatio": "","notionalLever": "0.0022195262185864","ordFrozen": "0","upl": "0","uplLiab": "0","crossLiab": "0","isoLiab": "0","coinUsdPrice": "60000","stgyEq":"0","spotInUseAmt":"","isoUpl":""}]}]}`,
	"Quotes":                                `{"arg": {"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`,
	"Structure Block Trades":                `{"arg": {"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`,
	"Spot Grid Algo Orders":                 `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`,
	"Contract Grid Algo Orders":             `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589"}]}`,
	"Grid Positions":                        `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`,
	"Grid Sub Orders":                       `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`,
	"Instrument":                            `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","instFamily": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`,
	"Open Interest":                         `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`,
	"Trade":                                 `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`,
	"Estimated Delivery And Exercise Price": `{"arg": {"channel": "estimated-price","instType": "FUTURES","instFamily": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`,
	"Mark Price":                            `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`,
	"Mark Price Candlestick":                `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`,
	"Price Limit":                           `{"arg": {"channel": "price-limit","instId": "LTC-USD-190628"},"data": [{"instId": "LTC-USD-190628","buyLmt": "200","sellLmt": "300","ts": "1597026383085"}]}`,
	"Test Snapshot Orderbook":               `{"arg": {"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
	"Options Trades":                        `{"arg": {"channel": "option-trades", "instType": "OPTION", "instFamily": "BTC-USD" }, "data": [ { "fillVol": "0.5066007836914062", "fwdPx": "16469.69928595038", "idxPx": "16537.2", "instFamily": "BTC-USD", "instId": "BTC-USD-230224-18000-C", "markPx": "0.04690107010619562", "optType": "C", "px": "0.045", "side": "sell", "sz": "2", "tradeId": "38", "ts": "1672286551080" } ] }`,
	"Public Block Trades":                   `{"arg": {"channel":"public-block-trades", "instId":"BTC-USD-231020-5000-P" }, "data":[ { "fillVol":"5", "fwdPx":"26808.16", "idxPx":"27222.5", "instId":"BTC-USD-231020-5000-P", "markPx":"0.0022406326071111", "px":"0.0048", "side":"buy", "sz":"1", "tradeId":"633971452580106242", "ts":"1697422572972"}]}`,
	"Option Summary":                        `{"arg": {"channel": "opt-summary","instFamily": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`,
	"Funding Rate":                          `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`,
	"Index Candlestick":                     `{"arg": {"channel": "index-candle30m","instId": "BTC-USDT"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`,
	"Index Ticker":                          `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`,
	"Status":                                `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`,
	"Public Struct Block Trades":            `{"arg": {"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`,
	"Block Ticker":                          `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Account":                               `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Position":                              `{"arg": {"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Position Data With Underlying":         `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Balance And Position":                  `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`,
	"Deposit Info Details":                  `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Withdrawal Info Details":               `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Recurring Buy Order":                   `{"arg": {"channel": "algo-recurring-buy", "instType": "SPOT", "uid": "447*******584" }, "data": [{ "algoClOrdId": "", "algoId": "644497312047435776", "algoOrdType": "recurring", "amt": "100", "cTime": "1699932133373", "cycles": "0", "instType": "SPOT", "investmentAmt": "0", "investmentCcy": "USDC", "mktCap": "0", "nextInvestTime": "1699934415300", "pTime": "1699933314691", "period": "hourly", "pnlRatio": "0", "recurringDay": "", "recurringHour": "1", "recurringList": [{ "avgPx": "0", "ccy": "BTC", "profit": "0", "px": "36482", "ratio": "0.2", "totalAmt": "0" }, { "avgPx": "0", "ccy": "ETH", "profit": "0", "px": "2057.54", "ratio": "0.8", "totalAmt": "0" }], "recurringTime": "12", "state": "running", "stgyName": "stg1", "tag": "", "timeZone": "8", "totalAnnRate": "0", "totalPnl": "0", "uTime": "1699932136249" }] }`,
	"Liquidation Orders":                    `{"arg": {"channel": "liquidation-orders", "instType": "SWAP" }, "data": [ { "details": [ { "bkLoss": "0", "bkPx": "0.007831", "ccy": "", "posSide": "short", "side": "buy", "sz": "13", "ts": "1692266434010" } ], "instFamily": "IOST-USDT", "instId": "IOST-USDT-SWAP", "instType": "SWAP", "uly": "IOST-USDT"}]}`,
	"Economic Calendar":                     `{"arg": {"channel": "economic-calendar" }, "data": [ { "calendarId": "319275", "date": "1597026383085", "region": "United States", "category": "Manufacturing PMI", "event": "S&P Global Manufacturing PMI Final", "refDate": "1597026383085", "actual": "49.2", "previous": "47.3", "forecast": "49.3", "importance": "2", "prevInitial": "", "ccy": "", "unit": "", "ts": "1698648096590" } ] }`,
	"Failure":                               `{ "event": "error", "code": "60012", "msg": "Invalid request: {\"op\": \"subscribe\", \"args\":[{ \"channel\" : \"block-tickers\", \"instId\" : \"LTC-USD-200327\"}]}", "connId": "a4d3ae55" }`,
	"Balance Without Credentials":           `{"arg": {"channel": "balance_and_position","uid": "77982378738415880"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`,
}

package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errTestNoSnapshot = errors.New("no snapshot in this test")
	errTestDial       = errors.New("test dial failure")
)

// fixtureTime converts a millisecond fixture timestamp
func fixtureTime(ms int64) types.Time {
	return types.Time(time.UnixMilli(ms))
}

// streamRequestRecorder records the requests a mock stream server receives and answers them. reply returns the frame
// sent back for a request, or nil for the success reply
type streamRequestRecorder struct {
	mu       sync.Mutex
	requests []DerivativesStreamRequest
	reply    func(*DerivativesStreamRequest) []byte
}

func (r *streamRequestRecorder) handle(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req DerivativesStreamRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	r.mu.Lock()
	r.requests = append(r.requests, req)
	r.mu.Unlock()
	resp := fmt.Appendf(nil, `{"result":null,"id":%d}`, req.ID)
	if r.reply != nil {
		if custom := r.reply(&req); custom != nil {
			resp = custom
		}
	}
	return w.WriteMessage(gws.TextMessage, resp)
}

func (r *streamRequestRecorder) recorded() []DerivativesStreamRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]DerivativesStreamRequest(nil), r.requests...)
}

// newDerivativesWsTestExchange returns a test exchange whose REST requests are served from testdata/http.json and
// whose connections are dialled, without subscriptions, to a mock stream server answering with recorder
func newDerivativesWsTestExchange(t *testing.T, recorder *streamRequestRecorder) *Exchange {
	t.Helper()
	ex := newDerivativesTestExchange(t)
	require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, recorder.handle))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() { assert.NoError(t, ex.Websocket.Shutdown(), "Shutdown should not error") })
	return ex
}

// failingDialConnection is a connection whose dial fails
type failingDialConnection struct{ websocket.Connection }

func (failingDialConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return errTestDial
}

// matchRecordingConnection records the signatures of the request replies routed to it
type matchRecordingConnection struct {
	websocket.Connection
	signatures []any
}

func (c *matchRecordingConnection) RequireMatchWithData(signature any, _ []byte) error {
	c.signatures = append(c.signatures, signature)
	return nil
}

func TestWsDerivativesConnect(t *testing.T) {
	t.Parallel()
	err := e.WsDerivativesConnect(t.Context(), failingDialConnection{})
	assert.ErrorIs(t, err, errTestDial, "WsDerivativesConnect should return the dial error")

	ex := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mockws.EchoHandler))
	for _, filter := range []string{usdtmPublicFilter, usdtmMarketFilter, usdtmPrivateFilter, coinmFilter, coinmPrivateFilter, optionsPublicFilter, optionsMarketFilter, optionsPrivateFilter} {
		_, err := ex.Websocket.GetConnection(filter)
		assert.NoErrorf(t, err, "WsDerivativesConnect should dial the %s connection at its configured URL", filter)
	}
}

func TestGenerateUFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	subs, err := ex.generateUFuturesPublicSubscriptions()
	require.NoError(t, err, "generateUFuturesPublicSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUFuturesPair}, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusdt@depth@100ms"},
	}, subs)

	subs, err = ex.generateUFuturesMarketSubscriptions()
	require.NoError(t, err, "generateUFuturesMarketSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUFuturesPair}, QualifiedChannel: "btcusdt@ticker"},
		{Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUFuturesPair}, QualifiedChannel: "btcusdt@aggTrade"},
		{Channel: subscription.CandlesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUFuturesPair}, Interval: kline.OneMin, QualifiedChannel: "btcusdt@kline_1m"},
	}, subs)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	for _, generate := range []func() (subscription.List, error){ex.generateUFuturesPublicSubscriptions, ex.generateUFuturesMarketSubscriptions} {
		subs, err = generate()
		require.NoError(t, err, "generating subscriptions for a disabled asset must not error")
		assert.Empty(t, subs, "a disabled asset should generate no subscriptions, so its connection is skipped")
	}
}

func TestGenerateDerivativesSubscriptionsPartialFailure(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// An enabled asset whose pairs cannot be formatted must not fail every other connection
	require.NoError(t, ex.CurrencyPairs.Store(asset.USDTMarginedFutures, &currency.PairStore{
		AssetEnabled: true,
		Available:    currency.Pairs{testUFuturesPair},
		Enabled:      currency.Pairs{testUFuturesPair},
	}), "Store must not error")
	_, err := ex.generateDerivativesSubscriptions(asset.USDTMarginedFutures, defaultUFuturesMarketSubscriptions)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateDerivativesSubscriptions should return a partial subscription error")
}

func TestDerivativesStreamName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a       asset.Item
		sub     *subscription.Subscription
		exp     string
		wantErr error
	}{
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "ticker"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "optionTicker"},
		{a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "aggTrade"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "optionTrade"},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, exp: "kline_1d"},
		{a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth}, exp: "kline_1M"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneWeek}, exp: "kline_1w"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.EightHour}, wantErr: kline.ErrUnsupportedInterval},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThousandMilliseconds}, wantErr: kline.ErrUnsupportedInterval},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth"},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, exp: "depth@100ms"},
		{a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.FiveHundredMilliseconds, Levels: 20}, exp: "depth20@500ms"},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5}, exp: "depth5"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.FiveHundredMilliseconds, Levels: 10}, exp: "depth10@500ms"},
		{a: asset.Options, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, wantErr: errUnsupportedStreamOption},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}, wantErr: errUnsupportedStreamOption},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 50}, wantErr: errUnsupportedStreamOption},
		{a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, wantErr: subscription.ErrNotSupported},
	} {
		got, err := e.derivativesStreamName(tc.a, tc.sub)
		if tc.wantErr != nil {
			assert.ErrorIsf(t, err, tc.wantErr, "derivativesStreamName should reject %s %s", tc.a, tc.sub.Channel)
			continue
		}
		require.NoErrorf(t, err, "derivativesStreamName must not error for %s %s", tc.a, tc.sub.Channel)
		assert.Equalf(t, tc.exp, got, "derivativesStreamName should name %s %s", tc.a, tc.sub.Channel)
	}
}

func TestSubscribeDerivatives(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		switch req.Params[0] {
		case "btcusdt@rejected":
			return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
		case "btcusdt@unexpected":
			return fmt.Appendf(nil, `{"result":true,"id":%d}`, req.ID)
		}
		return nil
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
	require.NoError(t, err, "GetConnection must not error")
	subs, err := ex.generateUFuturesMarketSubscriptions()
	require.NoError(t, err, "generateUFuturesMarketSubscriptions must not error")

	require.NoError(t, ex.SubscribeDerivatives(t.Context(), conn, subs), "SubscribeDerivatives must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
	}
	require.NoError(t, ex.UnsubscribeDerivatives(t.Context(), conn, subs), "UnsubscribeDerivatives must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "%s should be unsubscribed", s.QualifiedChannel)
	}
	requests := recorder.recorded()
	require.Len(t, requests, 2, "SubscribeDerivatives and UnsubscribeDerivatives must send one request each")
	for i := range requests {
		slices.SortFunc(requests[i].Params, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
	}
	exp := []DerivativesStreamRequest{
		{Method: "SUBSCRIBE", Params: []any{"btcusdt@aggTrade", "btcusdt@kline_1m", "btcusdt@ticker"}, ID: requests[0].ID},
		{Method: "UNSUBSCRIBE", Params: []any{"btcusdt@aggTrade", "btcusdt@kline_1m", "btcusdt@ticker"}, ID: requests[1].ID},
	}
	assert.Equal(t, exp, requests, "the requests should carry the stream names")
	assert.NotEqual(t, requests[0].ID, requests[1].ID, "each request should have its own ID")

	rejected := subscription.List{{Channel: "rejected", Asset: asset.USDTMarginedFutures, QualifiedChannel: "btcusdt@rejected"}}
	err = ex.SubscribeDerivatives(t.Context(), conn, rejected)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "SubscribeDerivatives should report a subscription failure")
	assert.ErrorIs(t, err, errStreamRequestRejected, "SubscribeDerivatives should report the rejection")
	assert.Nil(t, ex.Websocket.GetSubscription(rejected[0]), "a rejected subscription should not be stored")

	unexpected := subscription.List{{Channel: "unexpected", Asset: asset.USDTMarginedFutures, QualifiedChannel: "btcusdt@unexpected"}}
	assert.ErrorIs(t, ex.SubscribeDerivatives(t.Context(), conn, unexpected), errUnexpectedStreamResult, "SubscribeDerivatives should reject a result other than null")
}

func TestListSubscriptions(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		return fmt.Appendf(nil, `{"result":["btcusdt@aggTrade","btcusdt@kline_1m"],"id":%d}`, req.ID)
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
	require.NoError(t, err, "GetConnection must not error")
	result, err := ex.ListSubscriptions(t.Context(), conn)
	require.NoError(t, err, "ListSubscriptions must not error")
	assert.Equal(t, []string{"btcusdt@aggTrade", "btcusdt@kline_1m"}, result, "ListSubscriptions should decode the stream names")
	requests := recorder.recorded()
	require.Len(t, requests, 1, "ListSubscriptions must send one request")
	assert.Equal(t, DerivativesStreamRequest{Method: "LIST_SUBSCRIPTIONS", ID: requests[0].ID}, requests[0], "ListSubscriptions should send a request without params")
}

func TestSetProperty(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		if req.Params[0] == "unknown" {
			return fmt.Appendf(nil, `{"error":{"code":0,"msg":"Unknown property"},"id":%d}`, req.ID)
		}
		return nil
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(coinmFilter)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.SetProperty(t.Context(), conn, "combined", true), "SetProperty must not error")
	requests := recorder.recorded()
	require.Len(t, requests, 1, "SetProperty must send one request")
	assert.Equal(t, DerivativesStreamRequest{Method: "SET_PROPERTY", Params: []any{"combined", true}, ID: requests[0].ID}, requests[0], "SetProperty should send the property and value")
	assert.ErrorIs(t, ex.SetProperty(t.Context(), conn, "unknown", true), errStreamRequestRejected, "SetProperty should report a rejected property")
}

func TestGenerateUserDataSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	generators := map[asset.Item]func() (subscription.List, error){
		asset.USDTMarginedFutures: ex.generateUFuturesUserDataSubscriptions,
		asset.CoinMarginedFutures: ex.generateCFuturesUserDataSubscriptions,
		asset.Options:             ex.generateOptionsUserDataSubscriptions,
	}
	for a, generate := range generators {
		subs, err := generate()
		require.NoErrorf(t, err, "%s user data generation must not error", a)
		assert.Emptyf(t, subs, "%s user data should not be subscribed without authenticated websocket use", a)
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	for a, generate := range generators {
		subs, err := generate()
		require.NoErrorf(t, err, "%s user data generation must not error", a)
		require.Lenf(t, subs, 1, "%s user data generation must return one subscription", a)
		testsubs.EqualLists(t, subscription.List{{Channel: userDataStreamChannel, Asset: a}}, subs)
		assert.Truef(t, subs[0].Authenticated, "%s user data subscription should be authenticated", a)
	}
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	subs, err := ex.generateOptionsUserDataSubscriptions()
	require.NoError(t, err, "options user data generation must not error")
	assert.Empty(t, subs, "a disabled asset should not subscribe its user data")
	ex.SetCredentials(&accounts.Credentials{})
	subs, err = ex.generateUFuturesUserDataSubscriptions()
	require.NoError(t, err, "USDⓈ-M user data generation must not error")
	assert.Empty(t, subs, "user data should not be subscribed without usable credentials")
}

func TestSubscribeUserData(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	for _, tc := range []struct {
		filter string
		a      asset.Item
	}{
		{usdtmPrivateFilter, asset.USDTMarginedFutures},
		{coinmPrivateFilter, asset.CoinMarginedFutures},
		{optionsPrivateFilter, asset.Options},
	} {
		conn, err := ex.Websocket.GetConnection(tc.filter)
		require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
		subs := subscription.List{{Channel: userDataStreamChannel, Asset: tc.a, Authenticated: true}}
		require.NoErrorf(t, ex.SubscribeUserData(t.Context(), conn, subs), "SubscribeUserData must not error for %s", tc.a)
		assert.Equalf(t, subscription.SubscribedState, subs[0].State(), "%s user data should be subscribed", tc.a)
		require.NoErrorf(t, ex.UnsubscribeUserData(t.Context(), conn, subs), "UnsubscribeUserData must not error for %s", tc.a)
		assert.Equalf(t, subscription.UnsubscribedState, subs[0].State(), "%s user data should be unsubscribed", tc.a)
	}
	requests := recorder.recorded()
	require.Len(t, requests, 6, "each subscribe and unsubscribe must send one request")
	for i := range requests {
		exp := DerivativesStreamRequest{Method: "SUBSCRIBE", Params: []any{testListenKey}, ID: requests[i].ID}
		if i%2 == 1 {
			exp.Method = "UNSUBSCRIBE"
		}
		assert.Equal(t, exp, requests[i], "user data requests should name the listen key as the stream")
	}

	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	spot := subscription.List{{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}}
	err = ex.SubscribeUserData(t.Context(), conn, spot)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "SubscribeUserData should report a subscription failure")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "SubscribeUserData should reject an asset without user data streams")
	assert.Nil(t, ex.Websocket.GetSubscription(spot[0]), "a failed user data subscription should not be stored")
	err = ex.UnsubscribeUserData(t.Context(), conn, spot)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "UnsubscribeUserData should report a subscription failure")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UnsubscribeUserData should reject an asset without user data streams")
}

func TestStartUserDataStream(t *testing.T) {
	t.Parallel()
	serve := func(listenKey string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/fapi/v1/listenKey", "/dapi/v1/listenKey", "/eapi/v1/listenKey":
				if r.Method == http.MethodPost {
					_, _ = fmt.Fprintf(w, `{"listenKey":%q}`, listenKey)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}
	active := newTradingTestExchange(t, serve(testListenKey))
	empty := newTradingTestExchange(t, serve(""))
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		listenKey, err := active.startUserDataStream(t.Context(), a)
		require.NoErrorf(t, err, "startUserDataStream must not error for %s", a)
		assert.Equalf(t, testListenKey, listenKey, "startUserDataStream should return the %s listen key", a)
		_, err = empty.startUserDataStream(t.Context(), a)
		assert.ErrorIsf(t, err, errListenKeyEmpty, "startUserDataStream should reject an empty %s listen key", a)
	}
	_, err := active.startUserDataStream(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "startUserDataStream should reject an asset without user data streams")
}

func TestKeepaliveUserDataSubscription(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
		require.NoError(t, s.SetState(subscription.SubscribedState), "SetState must not error")
		assert.Truef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should keep the %s stream alive", a)
		require.NoError(t, s.SetState(subscription.UnsubscribedState), "SetState must not error")
		assert.Falsef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should stop for the unsubscribed %s stream", a)
	}
	assert.Empty(t, recorder.recorded(), "a successful keepalive should not resubscribe")
	assert.Empty(t, relayedPayloads(ex), "a successful keepalive should relay nothing")

	// A failed keepalive renews the stream; an asset without user data streams makes the renewal fail too
	unsupported := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}
	require.NoError(t, unsupported.SetState(subscription.SubscribedState), "SetState must not error")
	assert.True(t, ex.keepaliveUserDataSubscription(t.Context(), conn, unsupported), "keepaliveUserDataSubscription should keep trying while subscribed")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 1, "a failed renewal must relay one error")
	relayedErr, ok := payloads[0].(error)
	require.True(t, ok, "the relayed payload must be an error")
	assert.ErrorIs(t, relayedErr, asset.ErrNotSupported, "the renewal error should carry its cause")
}

func TestProcessListenKeyExpired(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	for _, tc := range []struct {
		filter  string
		fixture string
		handle  func(context.Context, websocket.Connection, []byte) error
		exp     *ListenKeyExpired
	}{
		{usdtmPrivateFilter, "testdata/wsUFuturesUserData.json", ex.wsHandleUFuturesUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1736996475556), ListenKey: testListenKey}},
		{coinmPrivateFilter, "testdata/wsCFuturesUserData.json", ex.wsHandleCFuturesUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1576653824250), ListenKey: testListenKey}},
		{optionsPrivateFilter, "testdata/wsOptionsUserData.json", ex.wsHandleOptionsUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1736996475556), ListenKey: testListenKey}},
	} {
		conn, err := ex.Websocket.GetConnection(tc.filter)
		require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
		require.NoErrorf(t, tc.handle(t.Context(), conn, fixtureFrame(t, tc.fixture, "listenKeyExpired")), "handling listenKeyExpired must not error for %s", tc.filter)
		assert.Equalf(t, []any{tc.exp}, relayedPayloads(ex), "listenKeyExpired should be relayed for %s", tc.filter)
	}
	require.Eventually(t, func() bool { return len(recorder.recorded()) == 3 }, time.Second*5, time.Millisecond*10, "each expiry must renew its stream")
	for _, req := range recorder.recorded() {
		assert.Equal(t, DerivativesStreamRequest{Method: "SUBSCRIBE", Params: []any{testListenKey}, ID: req.ID}, req, "the renewal should subscribe the new listen key")
	}
	assert.Empty(t, relayedPayloads(ex), "a successful renewal should relay nothing")
}

func TestFetchDerivativesOrderbook(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/fapi/v1/depth", "/dapi/v1/depth":
			assert.Equal(t, "1000", r.URL.Query().Get("limit"), "the futures order book should request 1000 levels")
			body = `{"lastUpdateId":1027024,"E":1589436922972,"T":1589436922959,"bids":[["4.00000000","431.00000000"]],"asks":[["4.00000200","12.00000000"]]}`
		case "/eapi/v1/depth":
			assert.Equal(t, "50", r.URL.Query().Get("limit"), "the options order book should request 50 levels")
			body = `{"T":1762866729358,"lastUpdateId":42788078000,"bids":[["3645.000","1.00"]],"asks":[["3675.000","5.00"]]}`
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	}))
	srv.Start()
	for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined, exchange.RestOptions} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), srv.URL), "SetRunningURL must not error")
	}
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  *orderbook.Book
	}{
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.CoinMarginedFutures, testCFuturesPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.Options, testOptionsPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 3645, Amount: 1}}, Asks: orderbook.Levels{{Price: 3675, Amount: 5}}, LastUpdateID: 42788078000, LastUpdated: time.UnixMilli(1762866729358)}},
	} {
		book, err := ex.fetchDerivativesOrderbook(t.Context(), tc.pair, tc.a)
		require.NoErrorf(t, err, "fetchDerivativesOrderbook must not error for %s", tc.a)
		tc.exp.Exchange, tc.exp.Pair, tc.exp.Asset, tc.exp.ValidateOrderbook = ex.Name, tc.pair, tc.a, ex.ValidateOrderbook
		assert.Equalf(t, tc.exp, book, "fetchDerivativesOrderbook should convert the %s snapshot", tc.a)
	}
	_, err := ex.fetchDerivativesOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "fetchDerivativesOrderbook should reject an asset without derivatives depth")

	overflow := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"lastUpdateId":18446744073709551615,"T":1589436922959,"bids":[],"asks":[]}`))
		assert.NoError(t, err, "Write should not error")
	}))
	overflow.Start()
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), overflow.URL), "SetRunningURL must not error")
	_, err = ex.fetchDerivativesOrderbook(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, errUpdateIDOutOfRange, "fetchDerivativesOrderbook should reject an update ID beyond int64")
}

func TestCheckDerivativesPendingUpdate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                  string
		lastUpdateID          int64
		previousFinalUpdateID int64
		finalUpdateID         int64
		skip                  bool
		wantErr               error
	}{
		{name: "ends before the snapshot", lastUpdateID: 100, previousFinalUpdateID: 80, finalUpdateID: 99, skip: true},
		{name: "spans the snapshot", lastUpdateID: 100, previousFinalUpdateID: 90, finalUpdateID: 110},
		{name: "ends at the snapshot", lastUpdateID: 100, previousFinalUpdateID: 90, finalUpdateID: 100},
		{name: "follows the snapshot", lastUpdateID: 100, previousFinalUpdateID: 100, finalUpdateID: 120},
		{name: "starts after the snapshot", lastUpdateID: 100, previousFinalUpdateID: 101, finalUpdateID: 120, wantErr: orderbookmanager.ErrOrderbookSnapshotOutdated},
	} {
		skip, err := checkDerivativesPendingUpdate(tc.lastUpdateID, tc.previousFinalUpdateID+1, &orderbook.Update{UpdateID: tc.finalUpdateID})
		assert.ErrorIsf(t, err, tc.wantErr, "checkDerivativesPendingUpdate should return the expected error for an event that %s", tc.name)
		assert.Equalf(t, tc.skip, skip, "checkDerivativesPendingUpdate should decide whether to skip an event that %s", tc.name)
	}
}

func TestProcessDerivativesDepthUpdate(t *testing.T) {
	t.Parallel()
	err := new(Exchange).processDerivativesDepthUpdate(t.Context(), 1, &orderbook.Update{})
	assert.ErrorIs(t, err, errOrderbookSyncNotSetUp, "processDerivativesDepthUpdate should require the order book synchronisation")
}

// orderbookState is the part of a stored order book the synchronisation tests compare
type orderbookState struct {
	Bids, Asks   orderbook.Levels
	LastUpdateID int64
	LastUpdated  time.Time
	LastPushed   time.Time
}

func storedOrderbookState(ex *Exchange, pair currency.Pair, a asset.Item) (*orderbookState, error) {
	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
	if err != nil {
		return nil, err
	}
	return &orderbookState{Bids: book.Bids, Asks: book.Asks, LastUpdateID: book.LastUpdateID, LastUpdated: book.LastUpdated, LastPushed: book.LastPushed}, nil
}

func TestDerivativesOrderbookSync(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		fixture  string
		stream   string
		handle   func(*Exchange) func(context.Context, websocket.Connection, []byte) error
		a        asset.Item
		pair     currency.Pair
		snapshot *orderbook.Book
		exp      *orderbookState
	}{
		{
			name: "USDⓈ-M", fixture: "testdata/wsUFuturesPublic.json", stream: "btcusdt@depth@100ms", a: asset.USDTMarginedFutures, pair: testUFuturesPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleUFuturesData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 85999.9, Amount: 15}, {Price: 85999.8, Amount: 0.2}, {Price: 85999.7, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 86000, Amount: 4}, {Price: 86000.1, Amount: 0.5}},
				LastUpdateID: 11747004259000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 85999.9, Amount: 16.201}, {Price: 85999.8, Amount: 0.381}},
				Asks:         orderbook.Levels{{Price: 86000.1, Amount: 0.702}, {Price: 86000.2, Amount: 1.157}},
				LastUpdateID: 11747004265102,
				LastUpdated:  time.UnixMilli(1791283218271),
				LastPushed:   time.UnixMilli(1791283218273),
			},
		},
		{
			name: "COIN-M", fixture: "testdata/wsCFutures.json", stream: "btcusd_perp@depth@100ms", a: asset.CoinMarginedFutures, pair: testCFuturesPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleCFuturesData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 86135.4, Amount: 2649}, {Price: 86135.3, Amount: 21}, {Price: 86133.6, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 86135.5, Amount: 3502}, {Price: 86135.6, Amount: 32}},
				LastUpdateID: 11747023540000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 86135.4, Amount: 2700}, {Price: 86135.3, Amount: 815}},
				Asks:         orderbook.Levels{{Price: 86135.6, Amount: 858}, {Price: 86135.7, Amount: 12}},
				LastUpdateID: 11747023544100,
				LastUpdated:  time.UnixMilli(1791283450364),
				LastPushed:   time.UnixMilli(1791283450370),
			},
		},
		{
			name: "options", fixture: "testdata/wsOptions.json", stream: "btc-261030-85000-c@depth@100ms", a: asset.Options, pair: testOptionsPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleOptionsData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 3645, Amount: 1}, {Price: 3640, Amount: 5.12}},
				Asks:         orderbook.Levels{{Price: 3675, Amount: 5}, {Price: 3680, Amount: 4.01}, {Price: 3690, Amount: 4.01}},
				LastUpdateID: 42788078000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 3645, Amount: 2.17}, {Price: 3640, Amount: 5.12}},
				Asks:         orderbook.Levels{{Price: 3670, Amount: 0.56}, {Price: 3680, Amount: 4.01}, {Price: 3690, Amount: 4.01}},
				LastUpdateID: 42788078248,
				LastUpdated:  time.UnixMilli(1791283337855),
				LastPushed:   time.UnixMilli(1791283337936),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newDerivativesTestExchange(t)
			useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				book := *tc.snapshot
				book.Exchange, book.Pair, book.Asset, book.LastUpdated = ex.Name, p, a, time.UnixMilli(1)
				book.Bids, book.Asks = append(orderbook.Levels(nil), tc.snapshot.Bids...), append(orderbook.Levels(nil), tc.snapshot.Asks...)
				return &book, nil
			})
			handle := tc.handle(ex)
			for _, frame := range fixtureFrames(t, tc.fixture) {
				if stream, _, _, _ := decodeStreamFrame(frame); stream == tc.stream {
					require.NoError(t, handle(t.Context(), nil, frame), "handling a diff depth frame must not error")
				}
			}
			var got *orderbookState
			require.Eventually(t, func() bool {
				var err error
				got, err = storedOrderbookState(ex, tc.pair, tc.a)
				return err == nil && got.LastUpdateID == tc.exp.LastUpdateID
			}, time.Second*5, time.Millisecond*10, "the order book must synchronise to the last event")
			assert.Equal(t, tc.exp, got, "the synchronised order book should apply every event to the snapshot")
		})
	}
}

func TestFuturesPartialDepth(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	require.NoError(t, ex.wsHandleUFuturesData(t.Context(), nil, fixtureFrame(t, "testdata/wsUFuturesPublic.json", "btcusdt@depth5@100ms")), "wsHandleUFuturesData must not error")
	got, err := storedOrderbookState(ex, testUFuturesPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "the partial depth snapshot must be stored")
	exp := &orderbookState{
		Bids:         orderbook.Levels{{Price: 85999.9, Amount: 15.739}, {Price: 85999.8, Amount: 0.195}},
		Asks:         orderbook.Levels{{Price: 86000, Amount: 4.613}, {Price: 86000.1, Amount: 0.005}},
		LastUpdateID: 11747004261842,
		LastUpdated:  time.UnixMilli(1791283218111),
		LastPushed:   time.UnixMilli(1791283218113),
	}
	assert.Equal(t, exp, got, "partial depth should load the top levels as a snapshot")
}

func TestIsPartialDepthStream(t *testing.T) {
	t.Parallel()
	for stream, exp := range map[string]bool{
		"btcusdt@depth":                    false,
		"btcusdt@depth@100ms":              false,
		"btcusd_perp@depth@500ms":          false,
		"btcusdt@depth5":                   true,
		"btcusdt@depth10@100ms":            true,
		"btc-261030-85000-c@depth20@500ms": true,
		"btcusdt@rpiDepth@500ms":           false,
		"btcusdt@bookTicker":               false,
	} {
		assert.Equalf(t, exp, isPartialDepthStream(stream), "isPartialDepthStream should classify %q", stream)
	}
}

func TestFuturesSymbolTypeAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbolType uint64
		connAsset  asset.Item
		exp        asset.Item
		wantErr    error
	}{
		{0, asset.USDTMarginedFutures, asset.USDTMarginedFutures, nil},
		{0, asset.CoinMarginedFutures, asset.CoinMarginedFutures, nil},
		{1, asset.CoinMarginedFutures, asset.USDTMarginedFutures, nil},
		{2, asset.USDTMarginedFutures, asset.CoinMarginedFutures, nil},
		{3, asset.USDTMarginedFutures, asset.Empty, errUnknownSymbolType},
	} {
		got, err := futuresSymbolTypeAsset(tc.symbolType, tc.connAsset)
		assert.ErrorIsf(t, err, tc.wantErr, "futuresSymbolTypeAsset should return the expected error for symbol type %d", tc.symbolType)
		assert.Equalf(t, tc.exp, got, "futuresSymbolTypeAsset should map symbol type %d", tc.symbolType)
	}
}

func TestOwnProductEntries(t *testing.T) {
	t.Parallel()
	entries := []FuturesMarkPrice{{Symbol: "BTCUSDT", SymbolType: 1}, {Symbol: "BTCUSD_PERP", SymbolType: 2}, {Symbol: "ETHUSDT"}, {Symbol: "UNKNOWN", SymbolType: 9}}
	got := ownProductEntries(entries, func(m *FuturesMarkPrice) uint64 { return m.SymbolType }, asset.USDTMarginedFutures)
	assert.Equal(t, []FuturesMarkPrice{{Symbol: "BTCUSDT", SymbolType: 1}, {Symbol: "ETHUSDT"}}, got, "ownProductEntries should keep the connection's product")
}

func TestMatchDerivativesPair(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, tc := range []struct {
		symbol  string
		a       asset.Item
		exp     currency.Pair
		enabled bool
		wantErr error
	}{
		{"BTCUSDT", asset.USDTMarginedFutures, testUFuturesPair, true, nil},
		{"ETHUSDT", asset.USDTMarginedFutures, testUFuturesDisabledPair, false, nil},
		{"BTCUSD_PERP", asset.CoinMarginedFutures, testCFuturesPair, true, nil},
		{"ETHUSD_PERP", asset.CoinMarginedFutures, testCFuturesDisabledPair, false, nil},
		{"BTC-261030-85000-C", asset.Options, testOptionsPair, true, nil},
		{"ETH-261009-2700-C", asset.Options, testOptionsDisabledPair, false, nil},
		{"BTCUSD_PERP", asset.USDTMarginedFutures, currency.EMPTYPAIR, false, currency.ErrPairNotFound},
	} {
		pair, enabled, err := ex.matchDerivativesPair(tc.symbol, tc.a)
		assert.ErrorIsf(t, err, tc.wantErr, "matchDerivativesPair should return the expected error for %s %s", tc.a, tc.symbol)
		assert.Equalf(t, tc.exp, pair, "matchDerivativesPair should match %s %s", tc.a, tc.symbol)
		assert.Equalf(t, tc.enabled, enabled, "matchDerivativesPair should report whether %s %s is enabled", tc.a, tc.symbol)
	}
}

func TestDecodeStreamFrame(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		frame   string
		stream  string
		data    string
		event   string
		wantErr error
		errText string
	}{
		{frame: `{"stream":"btcusdt@ticker","data":{"e":"24hrTicker"}}`, stream: "btcusdt@ticker", data: `{"e":"24hrTicker"}`, event: "24hrTicker"},
		{frame: `{"stream":"!ticker@arr","data":[{"e":"24hrTicker"},{"e":"24hrTicker"}]}`, stream: "!ticker@arr", data: `[{"e":"24hrTicker"},{"e":"24hrTicker"}]`, event: "24hrTicker"},
		{frame: `{"stream":"!ticker@arr","data":[]}`, stream: "!ticker@arr", data: `[]`},
		{frame: `{"stream":"!ticker@arr","data":[ ]}`, stream: "!ticker@arr", data: `[ ]`},
		{frame: `{"data":{"e":"24hrTicker"}}`, wantErr: errUnhandledStreamEvent, errText: "frame without a stream"},
		{frame: `{"stream":"btcusdt@ticker"}`, wantErr: errUnhandledStreamEvent, errText: "frame without data"},
		{frame: `{"stream":"btcusdt@ticker","data":{"s":"BTCUSDT"}}`, wantErr: errUnhandledStreamEvent, errText: "frame without an event type"},
		{frame: `{"stream":"!ticker@arr","data":["24hrTicker"]}`, wantErr: errUnhandledStreamEvent, errText: "frame without an event type"},
		// jsonparser reports an unreadable first element as missing, as it does for an empty array
		{frame: `{"stream":"!ticker@arr","data":[x]}`, wantErr: errUnhandledStreamEvent, errText: "frame with an invalid array"},
	} {
		stream, data, event, err := decodeStreamFrame([]byte(tc.frame))
		require.ErrorIsf(t, err, tc.wantErr, "decodeStreamFrame must return the expected error for %s", tc.frame)
		if tc.wantErr != nil {
			assert.ErrorContainsf(t, err, tc.errText, "decodeStreamFrame should give the reason it rejects %s", tc.frame)
			continue
		}
		assert.Equalf(t, tc.stream, stream, "decodeStreamFrame should return the stream of %s", tc.frame)
		assert.Equalf(t, tc.data, string(data), "decodeStreamFrame should return the data of %s", tc.frame)
		assert.Equalf(t, tc.event, event, "decodeStreamFrame should return the event of %s", tc.frame)
	}
}

func TestDerivativesOrderConversions(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Type{
		"LIMIT": order.Limit, "MARKET": order.Market, "STOP": order.StopLimit, "STOP_MARKET": order.StopMarket, "TAKE_PROFIT": order.TakeProfitLimit,
		"TAKE_PROFIT_MARKET": order.TakeProfitMarket, "TRAILING_STOP_MARKET": order.TrailingStop, "LIQUIDATION": order.Liquidation,
	} {
		got, err := derivativesOrderType(in)
		require.NoErrorf(t, err, "derivativesOrderType must not error for %s", in)
		assert.Equalf(t, exp, got, "derivativesOrderType should convert %s", in)
	}
	_, err := derivativesOrderType("ICEBERG")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "derivativesOrderType should reject an unknown type")

	for in, exp := range map[string]order.Status{
		"NEW": order.New, "PARTIALLY_FILLED": order.PartiallyFilled, "FILLED": order.Filled, "CANCELED": order.Cancelled, "EXPIRED": order.Expired,
		"EXPIRED_IN_MATCH": order.Expired, "NEW_INSURANCE": order.Liquidated, "NEW_ADL": order.AutoDeleverage,
	} {
		got, err := derivativesOrderStatus(in)
		require.NoErrorf(t, err, "derivativesOrderStatus must not error for %s", in)
		assert.Equalf(t, exp, got, "derivativesOrderStatus should convert %s", in)
	}
	_, err = derivativesOrderStatus("SUSPENDED")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "derivativesOrderStatus should reject an unknown status")

	for in, exp := range map[string]order.TimeInForce{
		"GTC": order.GoodTillCancel, "IOC": order.ImmediateOrCancel, "FOK": order.FillOrKill, "GTX": order.GoodTillCrossing, "GTD": order.GoodTillDay, "RPI": order.PostOnly,
		"GTE_GTC": order.GoodTillCancel,
	} {
		got, err := derivativesTimeInForce(in)
		require.NoErrorf(t, err, "derivativesTimeInForce must not error for %s", in)
		assert.Equalf(t, exp, got, "derivativesTimeInForce should convert %s", in)
	}
	_, err = derivativesTimeInForce("DAY")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "derivativesTimeInForce should reject an unknown time in force")
}

func TestFuturesPosition(t *testing.T) {
	t.Parallel()
	updated := time.UnixMilli(1564745798938)
	for _, tc := range []struct {
		positionSide string
		amount       float64
		direction    order.Side
		closed       bool
	}{
		{"BOTH", 1, order.Long, false},
		{"BOTH", -1, order.Short, false},
		{"BOTH", 0, order.UnknownSide, true},
		{"LONG", 0, order.Long, true},
		{"SHORT", 2, order.Short, false},
		{"", -0.1, order.Short, false},
	} {
		amount := decimal.MustFromFloat(tc.amount)
		exp := futures.Position{
			Exchange: "test", Asset: asset.USDTMarginedFutures, Pair: testUFuturesPair, LatestSize: amount.Abs(), Status: order.Open,
			OpeningDirection: tc.direction, LatestDirection: tc.direction, LastUpdated: updated,
		}
		if tc.closed {
			exp.Status, exp.CloseDate = order.Closed, updated
		}
		got := futuresPosition("test", asset.USDTMarginedFutures, testUFuturesPair, tc.positionSide, amount, updated)
		assert.Equalf(t, exp, got, "futuresPosition should build the %s %v position", tc.positionSide, tc.amount)
	}
}

func TestProcessAndSendTickers(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	valid := ticker.Price{Last: 1, Bid: 1, Ask: 2, Pair: testUFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures}
	crossed := ticker.Price{Last: 1, Bid: 3, Ask: 3, Pair: testCFuturesPair, ExchangeName: ex.Name, AssetType: asset.CoinMarginedFutures}

	err := ex.processAndSendTicker(t.Context(), &crossed)
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTicker should return the processing error")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTicker should not relay a ticker that fails processing")

	require.NoError(t, ex.processAndSendTicker(t.Context(), &valid), "processAndSendTicker must not error")
	assert.Equal(t, []any{&valid}, relayedPayloads(ex), "processAndSendTicker should relay the processed ticker")
	stored, err := ticker.GetTicker(ex.Name, testUFuturesPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, valid.Last, stored.Last, "processAndSendTicker should store the ticker")

	err = ex.processAndSendTickers(t.Context(), []ticker.Price{valid, crossed})
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTickers should report the entry that fails processing")
	assert.Equal(t, []any{[]ticker.Price{valid}}, relayedPayloads(ex), "processAndSendTickers should relay only the processed entries")

	err = ex.processAndSendTickers(t.Context(), []ticker.Price{crossed})
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTickers should report a batch that fails processing")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTickers should not relay an empty batch")

	assert.NoError(t, ex.processAndSendTickers(t.Context(), nil), "processAndSendTickers should not error without tickers")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTickers should not relay without tickers")
}

func TestHandleFuturesStreamErrors(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	conn := &matchRecordingConnection{}
	require.NoError(t, ex.wsHandleUFuturesData(t.Context(), conn, []byte(`{"result":null,"id":7}`)), "a request reply must be routed")
	assert.Equal(t, []any{int64(7)}, conn.signatures, "a request reply should be routed by its id")

	for frame, wantErr := range map[string]error{
		`{"stream":"btcusdt@unknown","data":{"e":"unknownEvent"}}`:                                 errUnhandledStreamEvent,
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","s":"BTCUSDT","st":3}}`:               errUnknownSymbolType,
		`{"stream":"btcusdt@depth@100ms","data":{"e":"depthUpdate","s":"BTCUSDT","st":9}}`:         errUnknownSymbolType,
		`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","s":"BTCUSDT","st":7}}`:               errUnknownSymbolType,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","s":"BTCUSDT","st":5}}`:       errUnknownSymbolType,
		`{"stream":"notlisted@kline_1m","data":{"e":"kline","s":"NOTLISTED","k":{"i":"1m"}}}`:      currency.ErrPairNotFound,
		`{"stream":"btcusdt@kline_1m","data":{"e":"kline","s":"BTCUSDT","k":{"i":"1s0"}}}`:         kline.ErrInvalidInterval,
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","s":"NOTLISTEDUSDT","st":1}}`:         currency.ErrPairNotFound,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","s":"NOTLISTEDUSDT","st":1}}`: currency.ErrPairNotFound,
	} {
		assert.ErrorIsf(t, ex.wsHandleUFuturesData(t.Context(), conn, []byte(frame)), wantErr, "wsHandleUFuturesData should reject %s", frame)
	}
	for _, frame := range []string{
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":"invalid"}}`,
		`{"stream":"!ticker@arr","data":[{"e":"24hrTicker","E":"invalid"}]}`,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","E":"invalid"}}`,
		`{"stream":"!miniTicker@arr","data":[{"e":"24hrMiniTicker","E":"invalid"}]}`,
		`{"stream":"btcusdt@bookTicker","data":{"e":"bookTicker","u":"invalid"}}`,
		`{"stream":"btcusdt@depth@100ms","data":{"e":"depthUpdate","U":"invalid"}}`,
		`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","a":"invalid"}}`,
		`{"stream":"btcusdt@forceOrder","data":{"e":"forceOrder","E":"invalid"}}`,
		`{"stream":"!contractInfo","data":{"e":"contractInfo","E":"invalid"}}`,
		`{"stream":"btcusdt@kline_1m","data":{"e":"kline","E":"invalid"}}`,
		`{"stream":"btcusdt@markPrice","data":{"e":"markPriceUpdate","E":"invalid"}}`,
		`{"stream":"!markPrice@arr","data":[{"e":"markPriceUpdate","E":"invalid"}]}`,
		`{"stream":"!assetIndex@arr","data":[{"e":"assetIndexUpdate","E":"invalid"}]}`,
	} {
		assert.Errorf(t, ex.wsHandleUFuturesData(t.Context(), conn, []byte(frame)), "wsHandleUFuturesData should report the decoding error of %s", frame)
	}
	require.NoError(t, ex.wsHandleUFuturesData(t.Context(), conn, []byte(`{"stream":"ethusdt@kline_1m","data":{"e":"kline","s":"ETHUSDT","k":{"i":"1m"}}}`)), "a disabled pair must be skipped")
	assert.Empty(t, relayedPayloads(ex), "nothing should be relayed for rejected or disabled frames")
}

func TestHandleFuturesUserDataErrors(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	conn := &matchRecordingConnection{}
	require.NoError(t, ex.wsHandleUFuturesUserData(t.Context(), conn, []byte(`{"result":null,"id":3}`)), "a request reply must be routed")
	assert.Equal(t, []any{int64(3)}, conn.signatures, "a request reply should be routed by its id")
	for data, wantErr := range map[string]error{
		``:                      errUnhandledStreamEvent,
		`{"E":1}`:               errUnhandledStreamEvent,
		`{"e":"UNKNOWN_EVENT"}`: errUnhandledStreamEvent,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"UP"}}`:                                  order.ErrSideIsInvalid,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"ICEBERG"}}`:                   order.ErrUnrecognisedOrderType,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"SUSPENDED"}}`:     errUnknownOrderStatus,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"NEW","f":"GTE"}}`: order.ErrInvalidTimeInForce,
	} {
		frame := `{"stream":"` + testListenKey + `"}`
		if data != "" {
			frame = `{"stream":"` + testListenKey + `","data":` + data + `}`
		}
		err := ex.wsHandleUFuturesUserData(t.Context(), conn, []byte(frame))
		require.ErrorIsf(t, err, wantErr, "wsHandleUFuturesUserData must reject %s", data)
		assert.NotContains(t, err.Error(), testListenKey, "user data errors should never quote the listen key")
	}
	for _, data := range []string{
		`{"e":"ORDER_TRADE_UPDATE","E":"invalid"}`,
		`{"e":"ALGO_UPDATE","E":"invalid"}`,
		`{"e":"ACCOUNT_UPDATE","E":"invalid"}`,
		`{"e":"MARGIN_CALL","E":"invalid"}`,
		`{"e":"listenKeyExpired","E":"invalid"}`,
	} {
		assert.Errorf(t, ex.wsHandleUFuturesUserData(t.Context(), conn, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), "wsHandleUFuturesUserData should reject %s", data)
	}
	assert.Empty(t, relayedPayloads(ex), "nothing should be relayed for rejected frames")
}

func TestWsHandleUFuturesData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for _, path := range []string{"testdata/wsUFuturesPublic.json", "testdata/wsUFuturesMarket.json"} {
		for _, frame := range fixtureFrames(t, path) {
			require.NoErrorf(t, ex.wsHandleUFuturesData(t.Context(), nil, frame), "wsHandleUFuturesData must not error for %s", frame)
		}
	}
	miniTicker := ticker.Price{
		Last: 86026.4, High: 86683.9, Low: 84910, Open: 86016.5, BaseVolume: 119489.809, QuoteVolume: 10238568766.13,
		Pair: testUFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures, LastUpdated: time.UnixMilli(1791283239234),
	}
	tick := ticker.Price{
		Last: 86026.4, LastSize: 0.05, VolumeWeightedAveragePrice: 85685.71, High: 86683.9, Low: 84910, Open: 86016.5, PercentChange24Hour: 0.012,
		BaseVolume: 119489.809, QuoteVolume: 10238568766.13, Pair: testUFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures,
		LastUpdated: time.UnixMilli(1791283239234),
	}
	markPrice := FuturesMarkPrice{
		EventType: "markPriceUpdate", EventTime: fixtureTime(1791283239000), Symbol: "BTCUSDT", MarkPrice: 86026.4, IndexPrice: 86059.26913043,
		EstimatedSettlePrice: 86090.55903865, FundingRate: -0.00003629, MarkPriceMovingAverage: 86026.3, NextFundingTime: fixtureTime(1791302400000), SymbolType: 1,
	}
	assetIndex := FuturesAssetIndex{
		EventType: "assetIndexUpdate", EventTime: fixtureTime(1791285204000), Symbol: "BTCUSD", IndexPrice: 86156.1448564, BidBuffer: 0.05, AskBuffer: 0.06,
		BidRate: 81848.33761358, AskRate: 90463.95209922, AutoExchangeBidBuffer: 0.025, AutoExchangeAskBuffer: 0.035, AutoExchangeBidRate: 84002.24123499,
		AutoExchangeAskRate: 88310.04847781,
	}
	exp := []any{
		&FuturesBookTicker{
			EventType: "bookTicker", UpdateID: 11747004261523, EventTime: fixtureTime(1791283218110), TransactionTime: fixtureTime(1791283218109),
			Symbol: "BTCUSDT", Pair: "BTCUSDT", BestBidPrice: 85999.9, BestBidQuantity: 15.739, BestAskPrice: 86000, BestAskQuantity: 4.613, SymbolType: 1,
		},
		&FuturesDepthUpdate{
			EventType: "depthUpdate", EventTime: fixtureTime(1791285277442), TransactionTime: fixtureTime(1791285277441), Symbol: "BTCUSDT", Pair: "BTCUSDT",
			FirstUpdateID: 11747185666708, FinalUpdateID: 11747185738417, PreviousFinalUpdateID: 11747185666537,
			Bids:       orderbook.LevelsArrayPriceAmount{{Price: 84441.6, Amount: 0.019}, {Price: 84444.9, Amount: 0.01}},
			Asks:       orderbook.LevelsArrayPriceAmount{{Price: 86160.1, Amount: 0.32}, {Price: 86160.2, Amount: 0.004}},
			SymbolType: 1,
		},
		[]trade.Data{{
			TID: "3476465318", Exchange: ex.Name, CurrencyPair: testUFuturesPair, AssetType: asset.USDTMarginedFutures, Side: order.Sell,
			Price: 86000, Amount: 4.577, Timestamp: time.UnixMilli(1791283227688),
		}},
		&FuturesLiquidationOrder{
			EventType: "forceOrder", EventTime: fixtureTime(1791285206184),
			Order: FuturesLiquidationOrderDetail{
				Symbol: "BTCUSDT", Pair: "BTCUSDT", Side: "SELL", OrderType: "LIMIT", TimeInForce: "IOC", OriginalQuantity: 0.042, Price: 85910.1,
				AveragePrice: 86012.3, OrderStatus: "FILLED", OrderLastFilledQuantity: 0.012, OrderFilledAccumulatedQuantity: 0.042,
				OrderTradeTime: fixtureTime(1791285205175), SymbolType: 1,
			},
		},
		[]ticker.Price{miniTicker},
		&miniTicker,
		[]ticker.Price{tick},
		&tick,
		&FuturesCompositeIndex{
			EventType: "compositeIndex", EventTime: fixtureTime(1791285204001), Symbol: "BTCDOMUSDT", Price: 5202.64806682, BaseAssetCategory: "quoteAsset",
			Composition: []FuturesCompositeIndexComposition{
				{BaseAsset: currency.BTC, QuoteAsset: currency.ETH, WeightInQuantity: 73.39600873, WeightInPercentage: 0.447459, IndexPrice: 31.76492999},
				{BaseAsset: currency.BTC, QuoteAsset: currency.SOL, WeightInQuantity: 0.68201057, WeightInPercentage: 0.094744, IndexPrice: 716.7927661},
			},
		},
		&FuturesContinuousKline{
			EventType: "continuous_kline", EventTime: fixtureTime(1791283239106), Pair: "BTCUSDT", ContractType: "PERPETUAL",
			Kline: FuturesContinuousKlineData{
				StartTime: fixtureTime(1791283200000), CloseTime: fixtureTime(1791283259999), Interval: "1m", FirstUpdateID: 11747002966903,
				LastUpdateID: 11747005833109, OpenPrice: 85999.9, ClosePrice: 86026.5, HighPrice: 86026.5, LowPrice: 85999.8, Volume: 10.516,
				NumberOfTrades: 447, IsClosed: true, QuoteAssetVolume: 904406.8029, TakerBuyVolume: 7.507, TakerBuyQuoteAssetVolume: 645620.8782, Ignore: 1,
			},
		},
		&FuturesContractInfo{
			EventType: "contractInfo", EventTime: fixtureTime(1669356423908), Symbol: "BTCUSDT", ContractType: "PERPETUAL",
			DeliveryDateTime: fixtureTime(4133404800000), OnboardDateTime: fixtureTime(1569398400000), ContractStatus: "TRADING",
			NotionalBrackets: []FuturesContractInfoBracket{
				{NotionalBracket: 1, BracketFloorNotional: 0.5, BracketNotionalCap: 5000, MaintenanceRatio: 0.01, AuxiliaryNumber: 0.25, MinLeverage: 21, MaxLeverage: 50},
				{NotionalBracket: 2, BracketFloorNotional: 5000, BracketNotionalCap: 25000, MaintenanceRatio: 0.025, AuxiliaryNumber: 75, MinLeverage: 11, MaxLeverage: 20},
			},
			SymbolType: 1,
		},
		kline.Item{
			Exchange: ex.Name, Pair: testUFuturesPair, Asset: asset.USDTMarginedFutures, Interval: kline.OneMin,
			Candles: []kline.Candle{{Time: time.UnixMilli(1791283200000), Open: 85999.9, High: 86000.1, Low: 85999.8, Close: 86000, Volume: 5.427}},
		},
		&markPrice,
		[]FuturesMarkPrice{markPrice},
		[]FuturesAssetIndex{assetIndex, {
			EventType: "assetIndexUpdate", EventTime: fixtureTime(1791285204000), Symbol: "USDTUSD", IndexPrice: 0.99968617, BidBuffer: 0.0001, AskBuffer: 0.0002,
			BidRate: 0.9995862, AskRate: 0.99978614, AutoExchangeBidBuffer: 0.0001, AutoExchangeAskBuffer: 0.0003, AutoExchangeBidRate: 0.9995862,
			AutoExchangeAskRate: 0.99978614,
		}},
		&assetIndex,
		&FuturesTradingSession{EventType: "EquityUpdate", EventTime: fixtureTime(1791285203779), SessionStartTime: fixtureTime(1791273600000), SessionEndTime: fixtureTime(1791293400000), SessionType: "PRE_MARKET"},
	}
	assert.Equal(t, exp, relayedPayloads(ex), "wsHandleUFuturesData should relay every stream of the connection's product and enabled pairs")
}

// normaliseBalanceTimes clears the local arrival times UpdateBalance stamps, after checking they are set
func normaliseBalanceTimes(tb testing.TB, payloads []any) {
	tb.Helper()
	for _, p := range payloads {
		subAccounts, ok := p.(accounts.SubAccounts)
		if !ok {
			continue
		}
		for _, s := range subAccounts {
			for c, b := range s.Balances {
				assert.WithinDurationf(tb, time.Now(), b.UpdatedAt, time.Minute, "%s balance should be stamped with its arrival time", c)
				b.UpdatedAt = time.Time{}
				s.Balances[c] = b
			}
		}
	}
}

// isEventFrame reports whether a user data frame carries an event, which some tests exercise separately
func isEventFrame(frame []byte, event string) bool {
	_, _, e, err := decodeStreamFrame(frame)
	return err == nil && e == event
}

func TestWsHandleUFuturesUserData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, frame := range fixtureFrames(t, "testdata/wsUFuturesUserData.json") {
		if isEventFrame(frame, "listenKeyExpired") {
			continue // TestProcessListenKeyExpired renews the stream over a connection
		}
		require.NoErrorf(t, ex.wsHandleUFuturesUserData(t.Context(), nil, frame), "wsHandleUFuturesUserData must not error for %s", frame)
	}
	payloads := relayedPayloads(ex)
	normaliseBalanceTimes(t, payloads)
	updated := time.UnixMilli(1564745798938)
	exp := []any{
		&FuturesMarginCall{
			EventType: "MARGIN_CALL", EventTime: fixtureTime(1587727187525), CrossWalletBalance: 3.16812045,
			Positions: []FuturesMarginCallPosition{{
				Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmount: 1.327, MarginType: "CROSSED", IsolatedWallet: 0.5, MarkPrice: 86187.17127,
				UnrealizedPNL: -1.166074, MaintenanceMarginRequired: 1.614445,
			}},
		},
		accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
			currency.USDT: {Currency: currency.USDT, Total: 122624.12345678, Free: 122624.12345678},
			currency.BNB:  {Currency: currency.BNB, Total: 1, Free: 1},
		}}},
		[]futures.Position{
			{
				Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUFuturesPair, PositionMargin: decimal.MustFromFloat(86.1),
				RealisedPNL: decimal.MustFromFloat(200), UnrealisedPNL: decimal.MustFromFloat(1.8245), Status: order.Open,
				OpeningPrice: decimal.MustFromFloat(86100), OpeningDirection: order.Short, LatestSize: decimal.MustFromFloat(-0.02).Abs(),
				LatestDirection: order.Short, LastUpdated: updated,
			},
			{
				Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUFuturesPair, PositionMargin: decimal.MustFromFloat(0),
				RealisedPNL: decimal.MustFromFloat(12.5), UnrealisedPNL: decimal.MustFromFloat(0), Status: order.Closed,
				OpeningPrice: decimal.MustFromFloat(0), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(0).Abs(),
				LatestDirection: order.Long, LastUpdated: updated, CloseDate: updated,
			},
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 86100.1, Amount: 0.5, TriggerPrice: 86476.89, AverageExecutedPrice: 86104.2,
			ExecutedAmount: 0.25, Cost: 21526.05, RemainingAmount: 0.25, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "8886774", ClientOrderID: "TEST",
			Type: order.TrailingStop, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.USDTMarginedFutures,
			Date: time.UnixMilli(1568879465650), LastUpdated: time.UnixMilli(1568879465650), Pair: testUFuturesPair,
			Trades: []order.TradeHistory{{
				Price: 86104.2, Amount: 0.125, Fee: 0.03444168, Exchange: ex.Name, TID: "10125", Type: order.TrailingStop, Side: order.Sell,
				Timestamp: time.UnixMilli(1568879465650), IsMaker: true, FeeAsset: "USDT",
			}},
		},
		&FuturesAccountConfigUpdate{
			EventType: "ACCOUNT_CONFIG_UPDATE", EventTime: fixtureTime(1611646737479), TransactionTime: fixtureTime(1611646737476),
			AccountConfiguration: &FuturesAccountConfiguration{Symbol: "BTCUSDT", Leverage: 25},
		},
		&FuturesAccountConfigUpdate{
			EventType: "ACCOUNT_CONFIG_UPDATE", EventTime: fixtureTime(1611646737480), TransactionTime: fixtureTime(1611646737477),
			UserAccountConfiguration: &FuturesUserAccountConfiguration{MultiAssetsMode: true},
		},
		&FuturesTradeLite{
			EventType: "TRADE_LITE", EventTime: fixtureTime(1721895408092), TransactionTime: fixtureTime(1721895408214), Symbol: "BTCUSDT",
			OriginalQuantity: 0.001, OriginalPrice: 86000.1, IsMaker: true, ClientOrderID: "z8hcUoOsqEdKMeKPSABslD", Side: "BUY",
			LastFilledPrice: 86000.1, OrderLastFilledQuantity: 0.001, TradeID: 109100866, OrderID: 8886775,
		},
		&FuturesConditionalOrderTriggerReject{
			EventType: "CONDITIONAL_ORDER_TRIGGER_REJECT", EventTime: fixtureTime(1685517224945), MessageSendTime: fixtureTime(1685517224955),
			OrderReject: FuturesConditionalOrderReject{Symbol: "BTCUSDT", OrderID: 155618472834, RejectReason: "Due to the order could not be filled immediately, the FOK order has been rejected."},
		},
		&FuturesStrategyUpdate{
			EventType: "STRATEGY_UPDATE", TransactionTime: fixtureTime(1669261797627), EventTime: fixtureTime(1669261797628),
			StrategyUpdate: FuturesStrategyUpdateData{StrategyID: 176054594, StrategyType: "GRID", StrategyStatus: "NEW", Symbol: "BTCUSDT", UpdateTime: fixtureTime(1669261797627), OpCode: 8007},
		},
		&FuturesGridUpdate{
			EventType: "GRID_UPDATE", TransactionTime: fixtureTime(1669262908216), EventTime: fixtureTime(1669262908218),
			GridUpdate: FuturesGridUpdateData{
				StrategyID: 176057039, StrategyType: "GRID", StrategyStatus: "WORKING", Symbol: "BTCUSDT", RealizedPNL: -0.00300716,
				UnmatchedAveragePrice: 86720, UnmatchedQuantity: -0.001, UnmatchedFee: -0.00300716, MatchedPNL: 0.5, UpdateTime: fixtureTime(1669262908197),
			},
		},
		&FuturesAlgoUpdate{
			EventType: "ALGO_UPDATE", TransactionTime: fixtureTime(1750515742297), EventTime: fixtureTime(1750515742303),
			Order: FuturesAlgoUpdateOrder{
				ClientAlgoID: "Q5xaq5EGKgXXa0fD7fs0Ip", AlgoID: 2148719, AlgoType: "CONDITIONAL", OrderType: "TAKE_PROFIT", Symbol: "BTCUSDT",
				Side: "SELL", PositionSide: "BOTH", TimeInForce: "GTC", Quantity: 0.01, AlgoStatus: "TRIGGERED", OrderID: "8886776",
				AverageFillPrice: 86750, ExecutedQuantity: 0.01, ActualOrderType: "LIMIT", TriggerPrice: 86750, OrderPrice: 86750,
				SelfTradePreventionMode: "EXPIRE_MAKER", WorkingType: "CONTRACT_PRICE", PriceMatchMode: "NONE", IsCloseAll: true,
				IsPriceProtected: true, IsReduceOnly: true, TriggerTime: fixtureTime(1750515742290), GoodTillDate: fixtureTime(1750602142297),
				FailedReason: "Reduce Only reject", IsActivated: true,
			},
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 86750, Amount: 0.01, TriggerPrice: 86750, AverageExecutedPrice: 86750,
			ExecutedAmount: 0.01, Exchange: ex.Name, OrderID: "2148719", ClientOrderID: "Q5xaq5EGKgXXa0fD7fs0Ip", Type: order.TakeProfitLimit,
			Side: order.Sell, Status: order.Active, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1750515742297),
			LastUpdated: time.UnixMilli(1750515742297), Pair: testUFuturesPair,
		},
	}
	assert.Equal(t, exp, payloads, "wsHandleUFuturesUserData should relay every user data event")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
	require.NoError(t, err, "the account update must store the balance")
	assert.Equal(t, 122624.12345678, stored.Total, "the account update should store the wallet balance")
}

// Pairs of symbols listed since the available pairs were stored, as user data derives them. Formatting a pair upper
// case, as the stored format is, marks its codes upper case, so a delivery date has to be marked too
var (
	testCFuturesUnlistedPair = currency.NewPairWithDelimiter("BTCUSD", "261225", currency.UnderscoreDelimiter).Upper()
	testOptionsUnlistedPair  = currency.NewPairWithDelimiter("BTC", "261031-90000-C", currency.DashDelimiter)
)

func TestUserDataPair(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, tc := range []struct {
		symbol  string
		a       asset.Item
		exp     currency.Pair
		wantErr error
	}{
		{"BTCUSDT", asset.USDTMarginedFutures, testUFuturesPair, nil},
		{"ETHUSD_PERP", asset.CoinMarginedFutures, testCFuturesDisabledPair, nil},
		{"BTC-261030-85000-C", asset.Options, testOptionsPair, nil},
		{"BTCUSD_261225", asset.CoinMarginedFutures, testCFuturesUnlistedPair, nil},
		{"BTC-261031-90000-C", asset.Options, testOptionsUnlistedPair, nil},
		{"NEWCOINUSDT", asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"BTCUSDT_261225", asset.USDTMarginedFutures, currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter).Upper(), nil},
		{"BTCUSD261225", asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"BTC261031C", asset.Options, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"", asset.Options, currency.EMPTYPAIR, currency.ErrSymbolStringEmpty},
	} {
		pair, err := ex.userDataPair(tc.symbol, tc.a)
		require.ErrorIsf(t, err, tc.wantErr, "userDataPair must return the expected error for %s %q", tc.a, tc.symbol)
		assert.Equalf(t, tc.exp, pair, "userDataPair should return the pair of %s %q", tc.a, tc.symbol)
	}
	ex.CurrencyPairs.Delete(asset.Options)
	_, err := ex.userDataPair("BTC-261031-90000-C", asset.Options)
	assert.ErrorIs(t, err, currency.ErrAssetNotFound, "userDataPair should return the error of a missing pair format")
}

func TestUserDataUnknownSymbols(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	updated := time.UnixMilli(1791300000000)
	for _, tc := range []struct {
		name   string
		handle func(context.Context, websocket.Connection, []byte) error
		data   string
		exp    []any
	}{
		{
			name:   "USDⓈ-M order of a symbol without a pair",
			handle: ex.wsHandleUFuturesUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"NEWCOINUSDT","c":"new-coin","S":"BUY","o":"LIMIT","f":"GTC","q":"5","p":"1.5","ap":"0","sp":"0","x":"NEW","X":"NEW","i":31,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"7.5","a":"0","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}`,
		},
		{
			name:   "USDⓈ-M account update with a position of a symbol without a pair",
			handle: ex.wsHandleUFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"a":{"m":"ORDER","B":[{"a":"USDT","wb":"1000","cw":"1000","bc":"0"}],"P":[{"s":"NEWCOINUSDT","pa":"5","ep":"1.5","bep":"1.5","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"},{"s":"BTCUSDT","pa":"0.1","ep":"86000","bep":"86034.4","cr":"0","up":"1.5","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
					currency.USDT: {Currency: currency.USDT, Total: 1000, Free: 1000},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUFuturesPair, PositionMargin: decimal.MustFromFloat(0),
					RealisedPNL: decimal.MustFromFloat(0), UnrealisedPNL: decimal.MustFromFloat(1.5), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(86000), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(0.1).Abs(),
					LatestDirection: order.Long, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "USDⓈ-M account update whose only position has no pair",
			handle: ex.wsHandleUFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000003,"T":1791300000002,"a":{"m":"ORDER","B":[{"a":"USDT","wb":"990","cw":"990","bc":"0"}],"P":[{"s":"NEWCOINUSDT","pa":"0","ep":"0","bep":"0","cr":"-10","up":"0","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 990, Free: 990},
			}}}},
		},
		{
			name:   "COIN-M order of a contract listed since the pairs were stored",
			handle: ex.wsHandleCFuturesUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","o":{"s":"BTCUSD_261225","c":"quarterly","S":"SELL","o":"LIMIT","f":"GTC","q":"10","p":"88000","ap":"0","sp":"0","x":"NEW","X":"NEW","i":32,"l":"0","z":"0","L":"0","ma":"BTC","N":"BTC","n":"0","T":1791300000000,"t":0,"b":"0","a":"0.01136364","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}`,
			exp: []any{&order.Detail{
				TimeInForce: order.GoodTillCancel, Price: 88000, Amount: 10, RemainingAmount: 10, FeeAsset: currency.BTC, Exchange: ex.Name,
				OrderID: "32", ClientOrderID: "quarterly", Type: order.Limit, Side: order.Sell, Status: order.New, AssetType: asset.CoinMarginedFutures,
				Date: updated, LastUpdated: updated, Pair: testCFuturesUnlistedPair,
			}},
		},
		{
			name:   "COIN-M account update with positions of unlisted symbols",
			handle: ex.wsHandleCFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","a":{"m":"ORDER","B":[{"a":"BTC","wb":"1.5","cw":"1.5","bc":"0"}],"P":[{"s":"BTCUSD_261225","pa":"-10","ep":"88000","bep":"88010","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"},{"s":"BTCUSD261225","pa":"1","ep":"88000","bep":"88000","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.CoinMarginedFutures, Balances: accounts.CurrencyBalances{
					currency.BTC: {Currency: currency.BTC, Total: 1.5, Free: 1.5},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.CoinMarginedFutures, Pair: testCFuturesUnlistedPair, PositionMargin: decimal.MustFromFloat(0),
					RealisedPNL: decimal.MustFromFloat(0), UnrealisedPNL: decimal.MustFromFloat(0), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(88000), OpeningDirection: order.Short, LatestSize: decimal.MustFromFloat(-10).Abs(),
					LatestDirection: order.Short, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "options order of an option listed since the pairs were stored",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTC-261031-90000-C","c":"new-option","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"500","ap":"0","x":"NEW","X":"NEW","i":33,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"1","a":"0","m":false,"R":false,"ot":"LIMIT","rp":"0","V":"NONE"}}`,
			exp: []any{&order.Detail{
				TimeInForce: order.GoodTillCancel, Price: 500, Amount: 1, RemainingAmount: 1, FeeAsset: currency.USDT, Exchange: ex.Name,
				OrderID: "33", ClientOrderID: "new-option", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: asset.Options,
				Date: updated, LastUpdated: updated, Pair: testOptionsUnlistedPair,
			}},
		},
		{
			name:   "options balance and position update with positions of unlisted symbols",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"BALANCE_POSITION_UPDATE","E":1791300000001,"T":1791300000000,"m":"ORDER","B":[{"a":"USDT","b":"9500","bc":"0"}],"P":[{"s":"BTC261031C","c":"2","p":"1000","a":"500"},{"s":"BTC-261031-90000-C","c":"1","p":"500","a":"500"}]}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.Options, Balances: accounts.CurrencyBalances{
					currency.USDT: {Currency: currency.USDT, Total: 9500, Free: 9500},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.Options, Pair: testOptionsUnlistedPair, NotionalSize: decimal.MustFromFloat(500).Abs(), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(500), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(1).Abs(),
					LatestDirection: order.Long, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "options balance and position update whose only position has no pair",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"BALANCE_POSITION_UPDATE","E":1791300000003,"T":1791300000002,"m":"ORDER","B":[{"a":"USDT","b":"9400","bc":"0"}],"P":[{"s":"BTC261031C","c":"0","p":"0","a":"0"}]}`,
			exp: []any{accounts.SubAccounts{{AssetType: asset.Options, Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 9400, Free: 9400},
			}}}},
		},
		{
			name:   "options order of a symbol without a pair",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTC261031C","c":"odd-option","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"500","ap":"0","x":"NEW","X":"NEW","i":34,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"1","a":"0","m":false,"R":false,"ot":"LIMIT","rp":"0","V":"NONE"}}`,
		},
	} {
		require.NoErrorf(t, tc.handle(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+tc.data+`}`)), "handling the %s must not error", tc.name)
		payloads := relayedPayloads(ex)
		normaliseBalanceTimes(t, payloads)
		assert.Equalf(t, tc.exp, payloads, "the %s should be relayed without the entries that have no pair", tc.name)
	}
}

func TestFuturesOrderTradeUpdateFills(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a        asset.Item
		symbol   string
		pair     currency.Pair
		feeAsset currency.Code
		fees     [2]string
		expFees  [2]float64
		expCosts [2]float64
	}{
		{asset.USDTMarginedFutures, "BTCUSDT", testUFuturesPair, currency.USDT, [2]string{"34.4", "68.8"}, [2]float64{34.4, 68.8}, [2]float64{86000, 258000}},
		{asset.CoinMarginedFutures, "BTCUSD_PERP", testCFuturesPair, currency.BTC, [2]string{"0.00000047", "0.00000093"}, [2]float64{0.00000047, 0.00000093}, [2]float64{}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newDerivativesTestExchange(t)
			// A limit buy of 3 is placed, then filled by two trades
			for _, u := range []struct {
				execution, status, last, cumulative, averagePrice, commission string
				tradeID                                                       uint64
				tradeTime                                                     int64
			}{
				{"NEW", "NEW", "0", "0", "0", "0", 0, 1791300000000},
				{"TRADE", "PARTIALLY_FILLED", "1", "1", "86000", tc.fees[0], 5001, 1791300000100},
				{"TRADE", "FILLED", "2", "3", "86000", tc.fees[1], 5002, 1791300000200},
			} {
				frame := fmt.Appendf(nil, `{"stream":%q,"data":{"e":"ORDER_TRADE_UPDATE","E":%d,"T":%d,"o":{"s":%q,"c":"two-fills","S":"BUY","o":"LIMIT","f":"GTC","q":"3","p":"86000","ap":%q,"sp":"0","x":%q,"X":%q,"i":4242,"l":%q,"z":%q,"L":"86000","N":%q,"n":%q,"T":%d,"t":%d,"b":"0","a":"0","m":true,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}}`,
					testListenKey, u.tradeTime+1, u.tradeTime, tc.symbol, u.averagePrice, u.execution, u.status, u.last, u.cumulative, tc.feeAsset, u.commission, u.tradeTime, u.tradeID)
				require.NoErrorf(t, ex.handleFuturesUserData(t.Context(), nil, frame, tc.a), "handleFuturesUserData must not error for the %s update", u.status)
			}
			trades := []order.TradeHistory{
				{
					Price: 86000, Amount: 1, Fee: tc.expFees[0], Exchange: ex.Name, TID: "5001", Type: order.Limit, Side: order.Buy,
					Timestamp: time.UnixMilli(1791300000100), IsMaker: true, FeeAsset: tc.feeAsset.String(),
				},
				{
					Price: 86000, Amount: 2, Fee: tc.expFees[1], Exchange: ex.Name, TID: "5002", Type: order.Limit, Side: order.Buy,
					Timestamp: time.UnixMilli(1791300000200), IsMaker: true, FeeAsset: tc.feeAsset.String(),
				},
			}
			exp := []any{
				&order.Detail{
					TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, RemainingAmount: 3, FeeAsset: tc.feeAsset, Exchange: ex.Name,
					OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: tc.a,
					Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: tc.pair,
				},
				&order.Detail{
					TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, AverageExecutedPrice: 86000, ExecutedAmount: 1, Cost: tc.expCosts[0],
					RemainingAmount: 2, FeeAsset: tc.feeAsset, Exchange: ex.Name, OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy,
					Status: order.PartiallyFilled, AssetType: tc.a, Date: time.UnixMilli(1791300000100), LastUpdated: time.UnixMilli(1791300000100),
					Pair: tc.pair, Trades: trades[:1],
				},
				&order.Detail{
					TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, AverageExecutedPrice: 86000, ExecutedAmount: 3, Cost: tc.expCosts[1],
					FeeAsset: tc.feeAsset, Exchange: ex.Name, OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy, Status: order.Filled,
					AssetType: tc.a, Date: time.UnixMilli(1791300000200), LastUpdated: time.UnixMilli(1791300000200), Pair: tc.pair,
					Trades: trades[1:],
				},
			}
			payloads := relayedPayloads(ex)
			require.Equal(t, exp, payloads, "handleFuturesUserData must relay each update with its own fill")

			// The engine stores the first detail of an order, merges later ones into it and tracks the merged order's
			// position, which needs the order's date
			details := make([]*order.Detail, len(payloads))
			for i := range payloads {
				var ok bool
				details[i], ok = payloads[i].(*order.Detail)
				require.True(t, ok, "every payload must be an order detail")
			}
			stored := details[0].Copy()
			controller := futures.SetupPositionController()
			require.NoError(t, controller.TrackNewOrder(stored.CopyToPointer()), "TrackNewOrder must accept an order first seen on the stream")
			for _, d := range details[1:] {
				require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
				require.NoError(t, controller.TrackNewOrder(stored.CopyToPointer()), "TrackNewOrder must accept the order's updates")
			}
			assert.Equal(t, trades, stored.Trades, "the stored order should keep every fill with its own commission")
			assert.Zero(t, stored.Fee, "the stored order should leave the commissions to its trades")
			assert.Equal(t, tc.expCosts[1], stored.Cost, "the stored order should keep the cost of its latest update")
			positions, err := controller.GetPositionsForExchange(ex.Name, tc.a, tc.pair)
			require.NoError(t, err, "GetPositionsForExchange must not error")
			require.Len(t, positions, 1, "the order must open one position")
			assert.Equal(t, time.UnixMilli(1791300000000), positions[0].OpeningDate, "the position should open when the order was placed")
		})
	}
}

func TestFuturesOrderTradeUpdateTrailingStop(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// The REST conversions of the same trailing stops, which the stream must agree with
	var uOrder UOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":777,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"trail","price":"0","avgPrice":"0","origQty":"0.3","executedQty":"0","cumQuote":"0","timeInForce":"GTC","type":"TRAILING_STOP_MARKET","origType":"TRAILING_STOP_MARKET","reduceOnly":true,"side":"SELL","positionSide":"BOTH","stopPrice":"86100.1","activatePrice":"86476.8","priceRate":"5.0","time":1791300000000,"updateTime":1791300000000}`), &uOrder), "Unmarshal must not error")
	uREST, err := ex.uOrderDetail(&uOrder, testUFuturesPair)
	require.NoError(t, err, "uOrderDetail must not error")
	require.Equal(t, 86476.8, uREST.TriggerPrice, "the REST USDⓈ-M conversion must trigger a trailing stop at its activation price")
	var cOrder FuturesOrderDetailResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":778,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"NEW","clientOrderId":"trail","price":"0","avgPrice":"0","origQty":"2","executedQty":"0","cumBase":"0","timeInForce":"GTC","type":"TRAILING_STOP_MARKET","origType":"TRAILING_STOP_MARKET","reduceOnly":true,"side":"SELL","positionSide":"BOTH","stopPrice":"86100.1","activatePrice":"86476.8","priceRate":"5.0","time":1791300000000,"updateTime":1791300000000}`), &cOrder), "Unmarshal must not error")
	cREST, err := ex.cOrderDetail(&cOrder, testCFuturesPair)
	require.NoError(t, err, "cOrderDetail must not error")
	require.Equal(t, 86476.8, cREST.TriggerPrice, "the REST COIN-M conversion must trigger a trailing stop at its activation price")
	// A triggered trailing stop reports MARKET as its type and keeps its original type
	uOrder.Type, cOrder.OrderType = "MARKET", "MARKET"
	uREST, err = ex.uOrderDetail(&uOrder, testUFuturesPair)
	require.NoError(t, err, "uOrderDetail must not error for a triggered trailing stop")
	require.Equal(t, 86476.8, uREST.TriggerPrice, "the REST USDⓈ-M conversion must keep a triggered trailing stop's activation price")
	cREST, err = ex.cOrderDetail(&cOrder, testCFuturesPair)
	require.NoError(t, err, "cOrderDetail must not error for a triggered trailing stop")
	require.Equal(t, 86476.8, cREST.TriggerPrice, "the REST COIN-M conversion must keep a triggered trailing stop's activation price")

	for _, tc := range []struct {
		name string
		a    asset.Item
		data string
		exp  *order.Detail
	}{
		{
			name: "placed COIN-M trailing stop",
			a:    asset.CoinMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","o":{"s":"BTCUSD_PERP","c":"trail","S":"SELL","o":"TRAILING_STOP_MARKET","f":"GTC","q":"2","p":"0","ap":"0","sp":"86100.1","x":"NEW","X":"NEW","i":778,"l":"0","z":"0","L":"0","ma":"BTC","N":"BTC","n":"0","T":1791300000000,"t":0,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"rp":"0","V":"NONE","pm":"NONE"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 2, TriggerPrice: 86476.8, RemainingAmount: 2, FeeAsset: currency.BTC,
				Exchange: ex.Name, OrderID: "778", ClientOrderID: "trail", Type: order.TrailingStop, Side: order.Sell, Status: order.New,
				AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: testCFuturesPair,
			},
		},
		{
			name: "triggered COIN-M trailing stop",
			a:    asset.CoinMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000501,"T":1791300000500,"i":"SfsR","o":{"s":"BTCUSD_PERP","c":"trail","S":"SELL","o":"MARKET","f":"GTC","q":"2","p":"0","ap":"86400.5","sp":"86100.1","x":"TRADE","X":"PARTIALLY_FILLED","i":778,"l":"1","z":"1","L":"86400.5","ma":"BTC","N":"BTC","n":"0.00000046","T":1791300000500,"t":9001,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"rp":"0","V":"NONE","pm":"NONE"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 2, TriggerPrice: 86476.8, AverageExecutedPrice: 86400.5, ExecutedAmount: 1,
				RemainingAmount: 1, FeeAsset: currency.BTC, Exchange: ex.Name, OrderID: "778", ClientOrderID: "trail", Type: order.Market,
				Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791300000500),
				LastUpdated: time.UnixMilli(1791300000500), Pair: testCFuturesPair,
				Trades: []order.TradeHistory{{
					Price: 86400.5, Amount: 1, Fee: 0.00000046, Exchange: ex.Name, TID: "9001", Type: order.Market, Side: order.Sell,
					Timestamp: time.UnixMilli(1791300000500), FeeAsset: "BTC",
				}},
			},
		},
		{
			name: "triggered USDⓈ-M trailing stop",
			a:    asset.USDTMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000501,"T":1791300000500,"o":{"s":"BTCUSDT","c":"trail","S":"SELL","o":"MARKET","f":"GTC","q":"0.3","p":"0","ap":"86400.5","sp":"86100.1","x":"TRADE","X":"PARTIALLY_FILLED","i":777,"l":"0.1","z":"0.1","L":"86400.5","N":"USDT","n":"3.45602","T":1791300000500,"t":9002,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"si":0,"ss":0,"rp":"0","V":"NONE","pm":"NONE","gtd":0,"er":"0"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 0.3, TriggerPrice: 86476.8, AverageExecutedPrice: 86400.5,
				ExecutedAmount: 0.1, Cost: 8640.05, RemainingAmount: 0.2, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "777", ClientOrderID: "trail",
				Type: order.Market, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.USDTMarginedFutures,
				Date: time.UnixMilli(1791300000500), LastUpdated: time.UnixMilli(1791300000500), Pair: testUFuturesPair,
				Trades: []order.TradeHistory{{
					Price: 86400.5, Amount: 0.1, Fee: 3.45602, Exchange: ex.Name, TID: "9002", Type: order.Market, Side: order.Sell,
					Timestamp: time.UnixMilli(1791300000500), FeeAsset: "USDT",
				}},
			},
		},
	} {
		require.NoErrorf(t, ex.handleFuturesUserData(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+tc.data+`}`), tc.a), "handleFuturesUserData must not error for the %s", tc.name)
		assert.Equalf(t, []any{tc.exp}, relayedPayloads(ex), "the %s should trigger at its activation price", tc.name)
	}
}

func TestFuturesOrderTradeUpdateLiquidation(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// A liquidation closes the position with an order of its own, whose execution is CALCULATED rather than TRADE
	data := `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTCUSDT","c":"autoclose-1791300000000","S":"SELL","o":"LIQUIDATION","f":"IOC","q":"0.1","p":"84000","ap":"84100","sp":"0","x":"CALCULATED","X":"FILLED","i":8886780,"l":"0.1","z":"0.1","L":"84100","N":"USDT","n":"42.05","T":1791300000000,"t":10200,"b":"0","a":"0","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIQUIDATION","ps":"BOTH","cp":false,"rp":"-190","pP":false,"si":0,"ss":0,"V":"NONE","pm":"NONE","gtd":0,"er":"0"}}`
	require.NoError(t, ex.wsHandleUFuturesUserData(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), "wsHandleUFuturesUserData must not error")
	exp := []any{&order.Detail{
		TimeInForce: order.ImmediateOrCancel, Price: 84000, Amount: 0.1, AverageExecutedPrice: 84100, ExecutedAmount: 0.1, Cost: 8410,
		FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "8886780", ClientOrderID: "autoclose-1791300000000", Type: order.Liquidation, Side: order.Sell,
		Status: order.Filled, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000),
		Pair: testUFuturesPair,
		Trades: []order.TradeHistory{{
			Price: 84100, Amount: 0.1, Fee: 42.05, Exchange: ex.Name, TID: "10200", Type: order.Liquidation, Side: order.Sell,
			Timestamp: time.UnixMilli(1791300000000), FeeAsset: "USDT",
		}},
	}}
	assert.Equal(t, exp, relayedPayloads(ex), "wsHandleUFuturesUserData should relay a liquidation's execution as the order's trade")
}

func TestProcessFuturesAlgoUpdate(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	algoUpdate := func(symbol, side, orderType, timeInForce, status string) []byte {
		return fmt.Appendf(nil, `{"e":"ALGO_UPDATE","T":1791300000000,"E":1791300000006,"o":{"caid":"stop-loss","aid":2148720,"at":"CONDITIONAL","o":%q,"s":%q,"S":%q,"ps":"BOTH","f":%q,"q":"0.01","X":%q,"ai":"","tp":"85000","p":"0","V":"EXPIRE_MAKER","wt":"MARK_PRICE","pm":"NONE","cp":false,"pP":true,"R":true,"tt":0,"gtd":0,"rm":"","ia":false}}`,
			orderType, symbol, side, timeInForce, status)
	}
	require.NoError(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTE_GTC", "CANCELED"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 2, "processFuturesAlgoUpdate must relay the event and its algo order")
	assert.IsType(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event first")
	exp := &order.Detail{
		TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 0.01, TriggerPrice: 85000, Exchange: ex.Name, OrderID: "2148720",
		ClientOrderID: "stop-loss", Type: order.StopMarket, Side: order.Sell, Status: order.Cancelled, AssetType: asset.USDTMarginedFutures,
		Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: testUFuturesPair,
	}
	assert.Equal(t, exp, payloads[1], "processFuturesAlgoUpdate should relay the algo order identified by its algo ID")
	rest, err := ex.uAlgoOrderDetail(&UAlgoOrderBase{
		AlgoID: 2148720, ClientAlgoID: "stop-loss", AlgoType: "CONDITIONAL", OrderType: "STOP_MARKET", Symbol: "BTCUSDT", Side: "SELL",
		PositionSide: "BOTH", TimeInForce: "GTE_GTC", Quantity: 0.01, AlgoStatus: "CANCELED", TriggerPrice: 85000, ReduceOnly: true,
		CreateTime: types.Time(time.UnixMilli(1791300000000)), UpdateTime: types.Time(time.UnixMilli(1791300000000)),
	}, 0, testUFuturesPair)
	require.NoError(t, err, "uAlgoOrderDetail must not error")
	assert.Equal(t, &rest, payloads[1], "processFuturesAlgoUpdate should convert the algo order as the REST algo order queries do")

	// The order manager holds a conditional order SubmitOrder placed under its algo ID, so the update reaches it
	update, ok := payloads[1].(*order.Detail)
	require.True(t, ok, "the second payload must be an order detail")
	submitted := order.Detail{
		Amount: 0.01, TriggerPrice: 85000, Exchange: ex.Name, OrderID: "2148720", ClientOrderID: "stop-loss", Type: order.StopMarket,
		Side: order.Sell, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791300000000), Pair: testUFuturesPair,
	}
	require.NoError(t, submitted.UpdateOrderFromDetail(update), "UpdateOrderFromDetail must not error")
	assert.Equal(t, order.Cancelled, submitted.Status, "the submitted conditional order should take the update's status")

	for _, tc := range []struct {
		data    []byte
		wantErr error
	}{
		{algoUpdate("BTCUSDT", "UP", "STOP_MARKET", "GTC", "NEW"), order.ErrSideIsInvalid},
		{algoUpdate("BTCUSDT", "SELL", "ICEBERG", "GTC", "NEW"), order.ErrUnrecognisedOrderType},
		{algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "DAY", "NEW"), order.ErrInvalidTimeInForce},
		{algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTC", "SUSPENDED"), errUnknownOrderStatus},
	} {
		assert.ErrorIsf(t, ex.processFuturesAlgoUpdate(t.Context(), tc.data, asset.USDTMarginedFutures), tc.wantErr, "processFuturesAlgoUpdate should reject %s", tc.data)
		payloads = relayedPayloads(ex)
		require.Lenf(t, payloads, 1, "processFuturesAlgoUpdate must relay only the event of %s", tc.data)
		assert.IsTypef(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event of %s", tc.data)
	}

	require.NoError(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("NEWCOINUSDT", "SELL", "STOP_MARKET", "GTC", "NEW"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error for a symbol without a pair")
	payloads = relayedPayloads(ex)
	require.Len(t, payloads, 1, "processFuturesAlgoUpdate must relay only the event of a symbol without a pair")
	assert.IsType(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event of a symbol without a pair")

	assert.Error(t, ex.processFuturesAlgoUpdate(t.Context(), []byte(`{"e":"ALGO_UPDATE","E":"invalid"}`), asset.USDTMarginedFutures), "processFuturesAlgoUpdate should report a decoding error")
	assert.Empty(t, relayedPayloads(ex), "processFuturesAlgoUpdate should relay nothing it cannot decode")

	for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
		require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "backlog"), "DataHandler must accept the backlog")
	}
	assert.Error(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTC", "NEW"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate should return the error of an event it cannot relay")
}

func TestFuturesPayloadDecoding(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		fixture string
		key     string
		target  any
		exp     any
	}{
		{
			name: "ticker", fixture: "testdata/wsUFuturesMarket.json", key: "btcusdt@ticker", target: new(FuturesTicker),
			exp: &FuturesTicker{
				EventType: "24hrTicker", EventTime: fixtureTime(1791283239234), Symbol: "BTCUSDT", Pair: "BTCUSDT", PriceChange: 9.9, PriceChangePercent: 0.012,
				WeightedAveragePrice: 85685.71, LastPrice: 86026.4, LastQuantity: 0.05, OpenPrice: 86016.5, HighPrice: 86683.9, LowPrice: 84910,
				TotalTradedBaseAssetVolume: 119489.809, TotalTradedQuoteAssetVolume: 10238568766.13, StatisticsOpenTime: fixtureTime(1791196800000),
				StatisticsCloseTime: fixtureTime(1791283239234), FirstTradeID: 8146765064, LastTradeID: 8149534072, TotalNumberOfTrades: 2756829, SymbolType: 1,
			},
		},
		{
			name: "mini ticker", fixture: "testdata/wsCFutures.json", key: "btcusd_perp@miniTicker", target: new(FuturesMiniTicker),
			exp: &FuturesMiniTicker{
				EventType: "24hrMiniTicker", EventTime: fixtureTime(1791283463856), Symbol: "BTCUSD_PERP", Pair: "BTCUSD", ClosePrice: 86137.8,
				OpenPrice: 85973.7, HighPrice: 86664.7, LowPrice: 84911.4, TotalTradedBaseAssetVolume: 7354554, TotalTradedQuoteAssetVolume: 8584, SymbolType: 2,
			},
		},
		{
			name: "aggregate trade", fixture: "testdata/wsUFuturesMarket.json", key: "btcusdt@aggTrade", target: new(FuturesAggregateTrade),
			exp: &FuturesAggregateTrade{
				EventType: "aggTrade", EventTime: fixtureTime(1791283227690), Symbol: "BTCUSDT", AggregateTradeID: 3476465318, Price: 86000, Quantity: 4.577,
				NormalQuantity: 4.571, FirstTradeID: 8149533714, LastTradeID: 8149533773, TradeTime: fixtureTime(1791283227688), IsBuyerMaker: true, SymbolType: 1,
			},
		},
		{
			name: "diff depth", fixture: "testdata/wsCFutures.json", key: "btcusd_perp@depth@100ms", target: new(FuturesDepthUpdate),
			exp: &FuturesDepthUpdate{
				EventType: "depthUpdate", EventTime: fixtureTime(1791283450270), TransactionTime: fixtureTime(1791283450264), Symbol: "BTCUSD_PERP", Pair: "BTCUSD",
				FirstUpdateID: 11747023537518, FinalUpdateID: 11747023543725, PreviousFinalUpdateID: 11747023536504,
				Bids:       orderbook.LevelsArrayPriceAmount{{Price: 86135.3, Amount: 815}, {Price: 86133.6, Amount: 0}},
				Asks:       orderbook.LevelsArrayPriceAmount{{Price: 86135.6, Amount: 858}},
				SymbolType: 2,
			},
		},
		{
			name: "kline", fixture: "testdata/wsUFuturesMarket.json", key: "btcusdt@kline_1m", target: new(FuturesKline),
			exp: &FuturesKline{
				EventType: "kline", EventTime: fixtureTime(1791283227688), Symbol: "BTCUSDT",
				Kline: FuturesKlineData{
					StartTime: fixtureTime(1791283200000), CloseTime: fixtureTime(1791283259999), Symbol: "BTCUSDT", Interval: "1m", FirstTradeID: 8149533620,
					LastTradeID: 8149533714, OpenPrice: 85999.9, ClosePrice: 86000, HighPrice: 86000.1, LowPrice: 85999.8, Volume: 5.427, NumberOfTrades: 92,
					IsClosed: true, QuoteAssetVolume: 466721.747, TakerBuyBaseAssetVolume: 2.897, TakerBuyQuoteAssetVolume: 249142, Ignore: 1,
				},
			},
		},
		{
			name: "account update", fixture: "testdata/wsUFuturesUserData.json", key: "ACCOUNT_UPDATE", target: new(FuturesAccountUpdate),
			exp: &FuturesAccountUpdate{
				EventType: "ACCOUNT_UPDATE", EventTime: fixtureTime(1564745798939), TransactionTime: fixtureTime(1564745798938),
				UpdateData: FuturesAccountUpdateData{
					EventReasonType: "FUNDING_FEE",
					Balances: []FuturesAccountUpdateBalance{
						{Asset: currency.USDT, WalletBalance: 122624.12345678, CrossWalletBalance: 100.12345678, BalanceChange: 50.12345678},
						{Asset: currency.BNB, WalletBalance: 1, CrossWalletBalance: 1, BalanceChange: 0.5},
					},
					Positions: []FuturesAccountUpdatePosition{
						{Symbol: "BTCUSDT", PositionAmount: -0.02, EntryPrice: 86100, BreakevenPrice: 86134.44, AccumulatedRealized: 200, UnrealizedPNL: 1.8245, MarginType: "isolated", IsolatedWallet: 86.1, PositionSide: "BOTH"},
						{Symbol: "BTCUSDT", AccumulatedRealized: 12.5, MarginType: "cross", PositionSide: "LONG"},
					},
					Symbol: "BTCUSDT",
				},
			},
		},
		{
			name: "COIN-M account update", fixture: "testdata/wsCFuturesUserData.json", key: "ACCOUNT_UPDATE", target: new(FuturesAccountUpdate),
			exp: &FuturesAccountUpdate{
				EventType: "ACCOUNT_UPDATE", EventTime: fixtureTime(1564745798939), TransactionTime: fixtureTime(1564745798938), AccountAlias: "SfsR",
				UpdateData: FuturesAccountUpdateData{
					EventReasonType: "ORDER",
					Balances:        []FuturesAccountUpdateBalance{{Asset: currency.BTC, WalletBalance: 1.12345678, CrossWalletBalance: 1.00345678, BalanceChange: 0.02345678}},
					Positions: []FuturesAccountUpdatePosition{{
						Symbol: "BTCUSD_PERP", PositionAmount: 20, EntryPrice: 86000, BreakevenPrice: 86012.5, AccumulatedRealized: 0.0002, UnrealizedPNL: 0.00001,
						MarginType: "isolated", IsolatedWallet: 0.00002325, PositionSide: "LONG",
					}},
					Symbol: "BTCUSD_PERP",
				},
			},
		},
		{
			name: "order trade update", fixture: "testdata/wsUFuturesUserData.json", key: "ORDER_TRADE_UPDATE", target: new(FuturesOrderTradeUpdate),
			exp: &FuturesOrderTradeUpdate{
				EventType: "ORDER_TRADE_UPDATE", EventTime: fixtureTime(1568879465651), TransactionTime: fixtureTime(1568879465650),
				Order: FuturesOrderTradeUpdateOrder{
					Symbol: "BTCUSDT", ClientOrderID: "TEST", Side: "SELL", OrderType: "TRAILING_STOP_MARKET", TimeInForce: "GTC", OriginalQuantity: 0.5,
					OriginalPrice: 86100.1, AveragePrice: 86104.2, StopPrice: 86103.04, ExecutionType: "TRADE", OrderStatus: "PARTIALLY_FILLED", OrderID: 8886774,
					ModifyID: "44444", OrderLastFilledQuantity: 0.125, OrderFilledAccumulatedQuantity: 0.25, LastFilledPrice: 86104.2, CommissionAsset: currency.USDT,
					Commission: 0.03444168, OrderTradeTime: fixtureTime(1568879465650), TradeID: 10125, BidsNotional: 172.2, AskNotional: 9.91, IsMaker: true,
					IsReduceOnly: true, StopPriceWorkingType: "CONTRACT_PRICE", OriginalOrderType: "TRAILING_STOP_MARKET", PositionSide: "LONG", IsCloseAll: true,
					ActivationPrice: 86476.89, CallbackRate: 5, IsPriceProtected: true, IgnoreSI: 3, IgnoreSS: 4, RealizedProfit: 0.24,
					SelfTradePreventionMode: "EXPIRE_TAKER", PriceMatchMode: "OPPONENT", GoodTillDate: fixtureTime(1568879965650), ExpiryReason: "1",
				},
			},
		},
		{
			name: "COIN-M order trade update", fixture: "testdata/wsCFuturesUserData.json", key: "ORDER_TRADE_UPDATE", target: new(FuturesOrderTradeUpdate),
			exp: &FuturesOrderTradeUpdate{
				EventType: "ORDER_TRADE_UPDATE", EventTime: fixtureTime(1591274595442), TransactionTime: fixtureTime(1591274595441), AccountAlias: "SfsR",
				Order: FuturesOrderTradeUpdateOrder{
					Symbol: "BTCUSD_PERP", ClientOrderID: "TEST", Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTX", OriginalQuantity: 2, OriginalPrice: 86000.1,
					AveragePrice: 86000.1, StopPrice: 86100.1, ExecutionType: "TRADE", OrderStatus: "FILLED", OrderID: 8888888, ModifyID: "44445",
					OrderLastFilledQuantity: 1, OrderFilledAccumulatedQuantity: 2, LastFilledPrice: 86000.1, MarginAsset: currency.BTC, CommissionAsset: currency.BTC,
					Commission: 0.00000116, OrderTradeTime: fixtureTime(1591274595442), TradeID: 1157454910, BidsNotional: 0.00002325, AskNotional: 0.00001163,
					IsMaker: true, IsReduceOnly: true, StopPriceWorkingType: "MARK_PRICE", OriginalOrderType: "LIMIT", PositionSide: "LONG", IsCloseAll: true,
					ActivationPrice: 86476.8, CallbackRate: 5, IsPriceProtected: true, RealizedProfit: 0.00000005, SelfTradePreventionMode: "EXPIRE_TAKER",
					PriceMatchMode: "OPPONENT", ExpiryReason: "2",
				},
			},
		},
	} {
		require.NoErrorf(t, json.Unmarshal(fixtureFrameData(t, tc.fixture, tc.key), tc.target), "Unmarshal must not error for the %s", tc.name)
		assert.Equalf(t, tc.exp, tc.target, "the %s should decode every field", tc.name)
	}
}

func TestDerivativesStreamsLive(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("live websocket streams are only tested with -tags mock_test_off")
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ex.Features.Subscriptions = subscription.List{} // no spot streams
	for a, p := range map[asset.Item]currency.Pair{
		asset.USDTMarginedFutures: currency.NewBTCUSDT(),
		asset.CoinMarginedFutures: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
	} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for %s", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for %s", a)
	}
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() { assert.NoError(t, ex.Websocket.Shutdown(), "Shutdown should not error") })

	type key struct {
		a    asset.Item
		kind string
	}
	seen := map[key]bool{}
	want := []key{
		{asset.USDTMarginedFutures, "ticker"},
		{asset.USDTMarginedFutures, "kline"},
		{asset.USDTMarginedFutures, "trade"},
		{asset.USDTMarginedFutures, "orderbook"},
		{asset.CoinMarginedFutures, "ticker"},
		{asset.CoinMarginedFutures, "kline"},
		{asset.CoinMarginedFutures, "orderbook"},
	}
	deadline := time.After(time.Minute)
	for {
		complete := true
		for _, k := range want {
			complete = complete && seen[k]
		}
		if complete {
			return
		}
		select {
		case p := <-ex.Websocket.DataHandler.C:
			switch d := p.Data.(type) {
			case *ticker.Price:
				seen[key{d.AssetType, "ticker"}] = true
			case kline.Item:
				seen[key{d.Asset, "kline"}] = true
			case []trade.Data:
				seen[key{d[0].AssetType, "trade"}] = true
			case *orderbook.Depth:
				// A REST synchronised book holds far more levels than one event carries
				if book, err := d.Retrieve(); err == nil && len(book.Bids) > 100 {
					seen[key{book.Asset, "orderbook"}] = true
				}
			case error:
				assert.NoError(t, d, "the live streams should not relay errors")
			}
		case <-deadline:
			require.Failf(t, "live streams must deliver every default stream", "seen %v of %v", seen, want)
		}
	}
}

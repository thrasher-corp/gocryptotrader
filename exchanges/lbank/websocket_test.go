package lbank

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

var errSendFailed = errors.New("send failed")

// recordingConnection is a websocket connection that records what is sent on it, dialled or not
type recordingConnection struct {
	websocket.Connection
	mu          sync.Mutex
	sent        []any
	sendErr     error
	dialErr     error
	dialled     int
	pingHandler *websocket.PingHandler
	connected   bool
}

func (c *recordingConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return c.sendErr
	}
	c.sent = append(c.sent, payload)
	return nil
}

func (c *recordingConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dialled++
	return c.dialErr
}

func (c *recordingConnection) SetupPingHandler(_ request.EndpointLimit, h websocket.PingHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pingHandler = &h
}

func (c *recordingConnection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *recordingConnection) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = connected
}

func (c *recordingConnection) messages() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]any(nil), c.sent...)
}

// newChannelTestExchange returns an exchange named after the test, with its trade and fills feeds on and its
// authenticated channels usable, for handling recorded frames
func newChannelTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	ex.Name = t.Name()
	ex.SkipAuthCheck = true
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	return ex
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	pairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.Len(t, pairs, 1, "the test exchange must enable one spot pair")
	pair := pairs[0]

	public, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	exp := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{pair}, QualifiedChannel: "tick:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: wsDefaultDepth, Pairs: currency.Pairs{pair}, QualifiedChannel: "depth:100:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{pair}, QualifiedChannel: "trade:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin, Pairs: currency.Pairs{pair}, QualifiedChannel: "kbar:1min:btc_usdt"},
	}
	testsubs.EqualLists(t, exp, public)

	account, err := ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error")
	exp = subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsOrderUpdateChannel},
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true, QualifiedChannel: wsAssetUpdateChannel},
	}
	testsubs.EqualLists(t, exp, account)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	account, err = ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error without authentication")
	assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without authentication")
	public, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error without authentication")
	assert.Len(t, public, 4, "generatePublicSubscriptions should not depend on authentication")

	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel}}
	_, err = ex.generatePublicSubscriptions()
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "generatePublicSubscriptions should reject a channel LBank doesn't stream")

	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.TwoHour}}
	_, err = ex.generatePublicSubscriptions()
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generatePublicSubscriptions should reject a candle interval LBank doesn't stream")
}

func TestNormaliseSubscriptions(t *testing.T) {
	t.Parallel()
	tickerSub := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}
	orders := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Pairs: currency.Pairs{spotTestPair}}
	expanded := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, QualifiedChannel: wsAssetUpdateChannel}
	got := normaliseSubscriptions(subscription.List{tickerSub, orders, expanded})
	require.Len(t, got, 3, "normaliseSubscriptions must keep every subscription")
	assert.Same(t, tickerSub, got[0], "normaliseSubscriptions should leave a market data subscription as it is")
	assert.Same(t, expanded, got[2], "normaliseSubscriptions should leave an expanded subscription as it is")
	assert.NotSame(t, orders, got[1], "normaliseSubscriptions should change a copy of an account subscription")
	exp := subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}}
	testsubs.EqualLists(t, exp, got[1:2])
	assert.Equal(t, asset.Spot, orders.Asset, "normaliseSubscriptions should not change the configured subscription")
}

func TestQualifiedChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
		err error
	}{
		{sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "tick:btc_usdt"},
		{sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "trade:btc_usdt"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth:100:btc_usdt"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10}, exp: "depth:10:btc_usdt"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FourHour}, exp: "kbar:4hr:btc_usdt"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeMin}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Channel: subscription.MyTradesChannel}, err: subscription.ErrNotSupported},
	} {
		got, err := e.qualifiedChannel(tc.sub, asset.Spot, currency.NewBTCUSDT())
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "qualifiedChannel should reject %s", tc.sub)
			continue
		}
		require.NoErrorf(t, err, "qualifiedChannel must not error for %s", tc.sub)
		assert.Equalf(t, tc.exp, got, "qualifiedChannel should qualify %s", tc.sub)
	}
}

func TestSubscriptionRequest(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	keyed, unkeyed := &recordingConnection{}, &recordingConnection{}
	ex.subscribeKeys.set(keyed, "4e99")
	pairs := currency.Pairs{spotTestPair}
	for _, tc := range []struct {
		name string
		conn websocket.Connection
		sub  *subscription.Subscription
		exp  *websocketSubscribeRequest
		err  error
	}{
		{name: "ticker", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: pairs}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"}},
		{name: "trades", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: pairs}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "trade", Pair: "btc_usdt"}},
		{name: "order book", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50, Pairs: pairs}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "depth", Pair: "btc_usdt", Depth: "50"}},
		{name: "candles", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneDay, Pairs: pairs}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "kbar", Pair: "btc_usdt", Kline: "day"}},
		{name: "orders", conn: keyed, sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "orderUpdate", Pair: "all", SubscribeKey: "4e99"}},
		{name: "balances", conn: keyed, sub: &subscription.Subscription{Channel: subscription.MyAccountChannel}, exp: &websocketSubscribeRequest{Action: "subscribe", Subscribe: "assetUpdate", SubscribeKey: "4e99"}},
		{name: "account without a key", conn: unkeyed, sub: &subscription.Subscription{Channel: subscription.MyAccountChannel}, err: errSubscribeKeyRequired},
		{name: "unsupported channel", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.MyTradesChannel, Pairs: pairs}, err: errUnsupportedChannel},
		{name: "two pairs", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair, currency.NewPair(currency.ETH, currency.USDT)}}, err: errSubscriptionPairs},
		{name: "unsupported interval", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.TwoHour, Pairs: pairs}, err: kline.ErrUnsupportedInterval},
	} {
		conn := tc.conn
		if conn == nil {
			conn = unkeyed
		}
		got, err := ex.subscriptionRequest(conn, "subscribe", tc.sub)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "subscriptionRequest should reject the %s subscription", tc.name)
			continue
		}
		require.NoErrorf(t, err, "subscriptionRequest must not error for the %s subscription", tc.name)
		assert.Equalf(t, tc.exp, got, "subscriptionRequest should build the %s request", tc.name)
	}
}

func TestManageSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	pairs := currency.Pairs{spotTestPair}
	tick := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: pairs, QualifiedChannel: "tick:btc_usdt"}
	depth := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Pairs: pairs, QualifiedChannel: "depth:10:btc_usdt"}
	require.NoError(t, ex.wsSubscribe(t.Context(), conn, subscription.List{tick, depth}), "wsSubscribe must not error")
	exp := []any{
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"},
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: "depth", Pair: "btc_usdt", Depth: "10"},
	}
	assert.Equal(t, exp, conn.messages(), "wsSubscribe should send a request for each subscription")
	assert.Equal(t, subscription.SubscribedState, tick.State(), "wsSubscribe should record the subscriptions it sent")
	assert.Equal(t, subscription.SubscribedState, depth.State(), "wsSubscribe should record the subscriptions it sent")

	require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subscription.List{depth}), "wsUnsubscribe must not error")
	exp = []any{
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"},
		&websocketSubscribeRequest{Action: "subscribe", Subscribe: "depth", Pair: "btc_usdt", Depth: "10"},
		&websocketSubscribeRequest{Action: "unsubscribe", Subscribe: "depth", Pair: "btc_usdt", Depth: "10"},
	}
	assert.Equal(t, exp, conn.messages(), "wsUnsubscribe should send a request for the subscription")
	assert.Equal(t, subscription.UnsubscribedState, depth.State(), "wsUnsubscribe should forget the subscription")
	assert.Equal(t, subscription.SubscribedState, tick.State(), "wsUnsubscribe should leave other subscriptions alone")

	failing := &recordingConnection{sendErr: errSendFailed}
	tradeSub := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: pairs, QualifiedChannel: "trade:btc_usdt"}
	err := ex.wsSubscribe(t.Context(), failing, subscription.List{tradeSub})
	assert.ErrorIs(t, err, errSendFailed, "wsSubscribe should report a request it couldn't send")
	assert.Equal(t, subscription.InactiveState, tradeSub.State(), "wsSubscribe should not record a subscription it couldn't send")

	account := &subscription.Subscription{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsOrderUpdateChannel}
	err = ex.wsSubscribe(t.Context(), conn, subscription.List{account, tradeSub})
	assert.ErrorIs(t, err, errSubscribeKeyRequired, "wsSubscribe should refuse an account channel on a connection without a key")
	assert.Equal(t, subscription.SubscribedState, tradeSub.State(), "wsSubscribe should still record the subscriptions it sent")
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	conn := &recordingConnection{}
	require.NoError(t, e.wsConnect(t.Context(), conn), "wsConnect must not error")
	assert.Equal(t, 1, conn.dialled, "wsConnect should dial the connection")
	exp := &websocket.PingHandler{MessageType: gws.TextMessage, Message: []byte(`{"action":"ping","ping":"gct"}`), Delay: wsPingDelay}
	assert.Equal(t, exp, conn.pingHandler, "wsConnect should ping the connection")

	failing := &recordingConnection{dialErr: errSendFailed}
	assert.ErrorIs(t, e.wsConnect(t.Context(), failing), errSendFailed, "wsConnect should report a failed dial")
	assert.Nil(t, failing.pingHandler, "wsConnect should not ping a connection it couldn't dial")
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"ping":"1792889772462","action":"ping"}`)), "wsHandleData must answer a ping")
	assert.Equal(t, []any{&websocketHeartbeatRequest{Action: "pong", Pong: "1792889772462"}}, conn.messages(), "wsHandleData should answer a ping with its ID")

	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"action":"pong","pong":"gct"}`)), "wsHandleData must accept a pong")

	err := ex.wsHandleData(t.Context(), conn, []byte(liveInvalidPairFrame))
	require.ErrorIs(t, err, errWebsocketError, "wsHandleData must report an error frame")
	assert.EqualError(t, err, "websocket error: Invalid order pairs:[notapair_usdt]", "wsHandleData should leave the caller's IP out of the error")

	assert.Error(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":`)), "wsHandleData should report a malformed frame")
	for _, frameType := range []string{wsTickerChannel, wsDepthChannel, wsTradeChannel, wsKlineChannel, wsOrderUpdateChannel, wsAssetUpdateChannel} {
		err := ex.wsHandleData(t.Context(), conn, []byte(`{"type":"`+frameType+`","pair":"btc_usdt"}`))
		assert.ErrorIsf(t, err, common.ErrNoResponse, "wsHandleData should report a %s frame without its update", frameType)
	}

	require.Empty(t, relayedPayloads(ex), "wsHandleData must relay nothing for the frames so far")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":"incrDepth","pair":"btc_usdt"}`)), "wsHandleData must not error for an unknown frame")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 1, "wsHandleData must relay an unknown frame")
	assert.IsType(t, websocket.UnhandledMessageWarning{}, payloads[0], "wsHandleData should relay an unknown frame as unhandled")
}

func TestWebsocketTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]time.Time{
		`"2019-06-28T17:49:22.722"`: time.Date(2019, 6, 28, 17, 49, 22, 722000000, beijingTime),
		`"2019-06-28T17:45:00"`:     time.Date(2019, 6, 28, 17, 45, 0, 0, beijingTime),
		`""`:                        {},
	} {
		var got WebsocketTime
		require.NoErrorf(t, got.UnmarshalJSON([]byte(in)), "UnmarshalJSON must not error for %s", in)
		assert.Truef(t, exp.Equal(got.Time()), "UnmarshalJSON should decode %s at UTC+8", in)
	}
	var got WebsocketTime
	assert.ErrorIs(t, got.UnmarshalJSON([]byte(`"28/06/2019"`)), errMalformedWebsocketTime, "UnmarshalJSON should reject a time in another layout")
	assert.Error(t, got.UnmarshalJSON([]byte(`1561704577786`)), "UnmarshalJSON should reject a time that isn't a string")
}

// liveInvalidPairFrame is LBank's reply to a subscription to a pair it doesn't list, recorded live with the caller's IP
// address replaced by a documentation address
const liveInvalidPairFrame = `{"SERVER":"V2","message":"Invalid order pairs:[notapair_usdt],remoteIp is: [192.0.2.1]","status":"error","TS":"2026-10-25T08:41:14.813"}`

// relayRecorder keeps the errors relayed by a connected exchange, draining its data handler from before it connects, as
// a busy feed fills the handler's buffer in seconds
type relayRecorder struct {
	mu   sync.Mutex
	errs []error
}

// relayRecorders holds each connected test exchange's recorder
var relayRecorders sync.Map

// startRelayRecorder drains an exchange's data handler until the test ends
func startRelayRecorder(t *testing.T, ex *Exchange) {
	t.Helper()
	r := new(relayRecorder)
	relayRecorders.Store(ex, r)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case p := <-ex.Websocket.DataHandler.C:
				if err, ok := p.Data.(error); ok {
					r.mu.Lock()
					r.errs = append(r.errs, err)
					r.mu.Unlock()
				}
			case <-done:
				return
			}
		}
	}()
	t.Cleanup(func() {
		close(done)
		relayRecorders.Delete(ex)
	})
}

// recordedErrors returns the errors a connected exchange has relayed
func recordedErrors(t *testing.T, ex *Exchange) []error {
	t.Helper()
	stored, ok := relayRecorders.Load(ex)
	require.True(t, ok, "the exchange must be connected with a relay recorder")
	r, ok := stored.(*relayRecorder)
	require.True(t, ok, "the relay recorder must be stored")
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.errs)
}

// relayedPayloads drains what has been relayed so far, leaving out order book depth pointers, which the order book
// tests inspect through the stored books instead
func relayedPayloads(ex *Exchange) []any {
	var got []any
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			if _, ok := p.Data.(*orderbook.Depth); !ok {
				got = append(got, p.Data)
			}
		default:
			return got
		}
	}
}

func TestWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	// The order book takes 7 levels, the depth the recorded frame was streamed at: LBank documents 10, 50 and 100 only,
	// but streams other depths too
	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 7},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	}
	ex, venue := newWebsocketTestExchange(t, subs)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := ticker.GetTicker(ex.Name, spotTestPair, asset.Spot)
		assert.NoError(c, err, "GetTicker should find the subscribed ticker")
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, asset.Spot)
		if assert.NoError(c, err, "GetOrderbook should find the subscribed book") {
			assert.NotEmpty(c, book.Bids, "the subscribed book should hold bids")
			assert.NotEmpty(c, book.Asks, "the subscribed book should hold asks")
		}
	}, 30*time.Second, 50*time.Millisecond, "the subscribed channels must deliver data")
	assert.Empty(t, recordedErrors(t, ex), "the subscribed channels should be handled without error")
	if mockTests {
		exp := []websocketSubscribeRequest{
			{Action: "subscribe", Subscribe: "tick", Pair: "btc_usdt"},
			{Action: "subscribe", Subscribe: "depth", Pair: "btc_usdt", Depth: "7"},
			{Action: "subscribe", Subscribe: "trade", Pair: "btc_usdt"},
			{Action: "subscribe", Subscribe: "kbar", Pair: "btc_usdt", Kline: "1min"},
		}
		// LBank confirms no subscription, so the mock may still be reading the last requests
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.ElementsMatch(c, exp, venue.received("subscribe"), "each subscription should be sent")
		}, 30*time.Second, 50*time.Millisecond, "the mock should receive each subscription")
	} else {
		// LBank stamps its pushes at UTC+8 without a zone, so a time read in another zone would be hours out
		tick, err := ticker.GetTicker(ex.Name, spotTestPair, asset.Spot)
		require.NoError(t, err, "GetTicker must find the subscribed ticker")
		assert.WithinDuration(t, time.Now(), tick.LastUpdated, time.Minute, "the ticker should be dated now")
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, asset.Spot)
		require.NoError(t, err, "GetOrderbook must find the subscribed book")
		assert.WithinDuration(t, time.Now(), book.LastPushed, time.Minute, "the book should be pushed now")
	}

	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 4, "every subscription must be stored")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, stored), "UnsubscribeChannels must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribing should forget the subscriptions")
	if mockTests {
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.Len(c, venue.received("unsubscribe"), 4, "each channel should be unsubscribed")
		}, 30*time.Second, 50*time.Millisecond, "the mock should receive each unsubscription")
	}
}

func TestWebsocketSubscribeUnlistedPair(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}})
	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	unlisted := &subscription.Subscription{
		Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: "tick:notapair_usdt",
		Pairs: currency.Pairs{currency.NewPairWithDelimiter("notapair", "usdt", currency.UnderscoreDelimiter)},
	}
	require.NoError(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{unlisted}), "SubscribeToChannels must not error, as LBank confirms no subscription")
	assert.Equal(t, subscription.SubscribedState, unlisted.State(), "a subscription LBank refuses should still be held, as LBank confirms none")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NotEmpty(c, recordedErrors(t, ex), "the refusal should be relayed")
	}, 30*time.Second, 50*time.Millisecond, "LBank's refusal of a pair it doesn't list must be relayed")
	refusal := recordedErrors(t, ex)[0]
	assert.ErrorIs(t, refusal, errWebsocketError, "the refusal should be relayed as a websocket error")
	assert.EqualError(t, errors.Unwrap(refusal), "websocket error: Invalid order pairs:[notapair_usdt]", "the refusal should name the pair and leave out the caller's IP address")
}

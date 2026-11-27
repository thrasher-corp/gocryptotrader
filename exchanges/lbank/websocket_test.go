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
	"github.com/thrasher-corp/gocryptotrader/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/asset"
	"github.com/thrasher-corp/gocryptotrader/exchange/kline"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
	"github.com/thrasher-corp/gocryptotrader/exchange/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchange/request"
	"github.com/thrasher-corp/gocryptotrader/exchange/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchange/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchange/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

var errSendFailed = errors.New("send failed")

// recordingConnection is a websocket connection that records what is sent on it, dialled or not. It refuses every send
// with sendErr, or only subscriptions to failChannel
type recordingConnection struct {
	websocket.Connection
	mu            sync.Mutex
	sent          []any
	sendErr       error
	failChannel   string
	dialErr       error
	dialled       int
	pingHandler   *websocket.PingHandler
	connected     bool
	subscriptions *subscription.Store
}

func (c *recordingConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return c.sendErr
	}
	if req, ok := payload.(*websocketSubscribeRequest); ok && c.failChannel != "" && req.Subscribe == c.failChannel {
		return errSendFailed
	}
	c.sent = append(c.sent, payload)
	return nil
}

func (c *recordingConnection) clearMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = nil
}

func (c *recordingConnection) setSendFailures(sendErr error, failChannel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendErr, c.failChannel = sendErr, failChannel
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

func (c *recordingConnection) Subscriptions() *subscription.Store {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscriptions == nil {
		c.subscriptions = subscription.NewStore()
	}
	return c.subscriptions
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

func TestGeneratePublicSubscriptions(t *testing.T) { //nolint:tparallel // Cases share one exchange, whose authentication and subscriptions later cases change, and run in order
	t.Parallel()
	ex := newChannelTestExchange(t)
	pairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.Len(t, pairs, 1, "the test exchange must enable one spot pair")
	pair := pairs[0]

	t.Run("with authentication", func(t *testing.T) {
		public, err := ex.generatePublicSubscriptions(t.Context())
		require.NoError(t, err, "generatePublicSubscriptions must not error")
		testsubs.EqualLists(t, publicTestSubscriptions(pair), public)
	})
	t.Run("without authentication", func(t *testing.T) {
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		public, err := ex.generatePublicSubscriptions(t.Context())
		require.NoError(t, err, "generatePublicSubscriptions must not error without authentication")
		assert.Len(t, public, 4, "generatePublicSubscriptions should not depend on authentication")
	})
	t.Run("unsupported channel", func(t *testing.T) {
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel}}
		_, err := ex.generatePublicSubscriptions(t.Context())
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "generatePublicSubscriptions should reject a channel LBank doesn't stream")
	})
	t.Run("unsupported candle interval", func(t *testing.T) {
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.TwoHour}}
		_, err := ex.generatePublicSubscriptions(t.Context())
		assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generatePublicSubscriptions should reject a candle interval LBank doesn't stream")
	})
}

func TestGenerateAccountSubscriptions(t *testing.T) { //nolint:tparallel // Cases share one exchange, whose authentication the second case turns off, and run in order, as do the sequence's steps on another
	t.Parallel()
	ex := newChannelTestExchange(t)
	pairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.Len(t, pairs, 1, "the test exchange must enable one spot pair")
	t.Run("with authentication", func(t *testing.T) {
		account, err := ex.generateAccountSubscriptions(t.Context())
		require.NoError(t, err, "generateAccountSubscriptions must not error")
		exp := subscription.List{
			{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsOrderUpdateChannel},
			{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true, QualifiedChannel: wsAssetUpdateChannel},
		}
		testsubs.EqualLists(t, exp, account)
	})
	t.Run("without authentication", func(t *testing.T) {
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		account, err := ex.generateAccountSubscriptions(t.Context())
		require.NoError(t, err, "generateAccountSubscriptions must not error without authentication")
		assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without authentication")
	})
	// The public and account connections generate their subscriptions on one exchange, so each must leave the other's
	// alone
	t.Run("public subscriptions after", func(t *testing.T) {
		public, err := ex.generatePublicSubscriptions(t.Context())
		require.NoError(t, err, "generatePublicSubscriptions must not error after the account subscriptions")
		testsubs.EqualLists(t, publicTestSubscriptions(pairs[0]), public)
	})
	t.Run("between public generations", func(t *testing.T) {
		seq := newChannelTestExchange(t)
		var first subscription.List
		t.Run("public", func(t *testing.T) {
			var err error
			first, err = seq.generatePublicSubscriptions(t.Context())
			require.NoError(t, err, "generatePublicSubscriptions must not error")
			testsubs.EqualLists(t, publicTestSubscriptions(pairs[0]), first)
		})
		t.Run("account", func(t *testing.T) {
			account, err := seq.generateAccountSubscriptions(t.Context())
			require.NoError(t, err, "generateAccountSubscriptions must not error after the public subscriptions")
			testsubs.EqualLists(t, accountSubscriptions(), account)
		})
		t.Run("account without authentication", func(t *testing.T) {
			seq.Websocket.SetCanUseAuthenticatedEndpoints(false)
			account, err := seq.generateAccountSubscriptions(t.Context())
			require.NoError(t, err, "generateAccountSubscriptions must not error without authentication")
			assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without authentication")
		})
		t.Run("public again", func(t *testing.T) {
			public, err := seq.generatePublicSubscriptions(t.Context())
			require.NoError(t, err, "generatePublicSubscriptions must not error after the account subscriptions")
			testsubs.EqualLists(t, publicTestSubscriptions(pairs[0]), public)
			testsubs.EqualLists(t, publicTestSubscriptions(pairs[0]), first)
		})
		t.Run("unsupported channel", func(t *testing.T) {
			seq.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel}}
			_, err := seq.generatePublicSubscriptions(t.Context())
			assert.ErrorIs(t, err, subscription.ErrNotSupported, "generatePublicSubscriptions should reject a channel LBank doesn't stream after the account subscriptions")
		})
		t.Run("unsupported candle interval", func(t *testing.T) {
			seq.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.TwoHour}}
			_, err := seq.generatePublicSubscriptions(t.Context())
			assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generatePublicSubscriptions should reject a candle interval LBank doesn't stream after the account subscriptions")
		})
	})
}

// publicTestSubscriptions returns the public subscriptions newChannelTestExchange's configuration generates for its pair
func publicTestSubscriptions(pair currency.Pair) subscription.List {
	return subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{pair}, QualifiedChannel: "tick:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: wsDefaultDepth, Pairs: currency.Pairs{pair}, QualifiedChannel: "depth:100:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{pair}, QualifiedChannel: "trade:btc_usdt"},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin, Pairs: currency.Pairs{pair}, QualifiedChannel: "kbar:1min:btc_usdt"},
	}
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
		name string
		sub  *subscription.Subscription
		exp  string
		err  error
	}{
		{name: "ticker", sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "tick:btc_usdt"},
		{name: "trades", sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "trade:btc_usdt"},
		{name: "order book at the default depth", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth:100:btc_usdt"},
		{name: "order book at 10 levels", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10}, exp: "depth:10:btc_usdt"},
		{name: "four hour candles", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FourHour}, exp: "kbar:4hr:btc_usdt"},
		{name: "unsupported candle interval", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeMin}, err: kline.ErrUnsupportedInterval},
		{name: "unsupported channel", sub: &subscription.Subscription{Channel: subscription.MyTradesChannel}, err: subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.qualifiedChannel(tc.sub, asset.Spot, currency.NewBTCUSDT())
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "qualifiedChannel should reject %s", tc.sub)
				return
			}
			require.NoErrorf(t, err, "qualifiedChannel must not error for %s", tc.sub)
			assert.Equalf(t, tc.exp, got, "qualifiedChannel should qualify %s", tc.sub)
		})
	}
}

func TestSubscriptionRequest(t *testing.T) { //nolint:tparallel // Cases share one exchange, its connections and pairs, and run in order
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
		t.Run(tc.name, func(t *testing.T) {
			conn := tc.conn
			if conn == nil {
				conn = unkeyed
			}
			got, err := ex.subscriptionRequest(conn, "subscribe", tc.sub)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "subscriptionRequest should reject the %s subscription", tc.name)
				return
			}
			require.NoErrorf(t, err, "subscriptionRequest must not error for the %s subscription", tc.name)
			assert.Equalf(t, tc.exp, got, "subscriptionRequest should build the %s request", tc.name)
		})
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
	err = ex.manageSubscriptions(t.Context(), conn, "subscribe", subscription.List{account, tradeSub})
	assert.ErrorIs(t, err, errSubscribeKeyRequired, "manageSubscriptions should refuse an account channel on a connection without a key")
	assert.Equal(t, subscription.SubscribedState, tradeSub.State(), "manageSubscriptions should still record the subscriptions it sent")
}

func TestSetupWebsocket(t *testing.T) { //nolint:tparallel // Cases share one exchange configuration and run in order, the second copying it as the first left it
	t.Parallel()
	exch := newTestExchangeConfig(t)
	t.Run("valid configuration", func(t *testing.T) {
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket = sharedtestvalues.NewTestWebsocket()
		ex.Accounts = accounts.MustNewAccounts(ex.GetBase())
		require.NoError(t, ex.SetupDefaults(exch), "SetupDefaults must not error")
		require.NoError(t, ex.setupWebsocket(exch), "setupWebsocket must not error")
		urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
		require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
		assert.Equal(t, []string{websocketURL}, urls, "setupWebsocket should set its connections up on LBank's websocket")
	})
	t.Run("configuration without features", func(t *testing.T) {
		featureless := *exch
		featureless.Features = nil
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket = sharedtestvalues.NewTestWebsocket()
		assert.ErrorIs(t, ex.setupWebsocket(&featureless), common.ErrNilPointer, "setupWebsocket should report a websocket it can't set up")
	})
	t.Run("missing websocket endpoint", func(t *testing.T) {
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket = sharedtestvalues.NewTestWebsocket()
		ex.Accounts = accounts.MustNewAccounts(ex.GetBase())
		require.NoError(t, ex.SetupDefaults(exch), "SetupDefaults must not error")
		ex.API.Endpoints = ex.NewEndpoints()
		assert.ErrorIs(t, ex.setupWebsocket(exch), exchange.ErrEndpointPathNotFound, "setupWebsocket should report a missing websocket endpoint")
	})
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	t.Run("successful dial", func(t *testing.T) {
		t.Parallel()
		conn := &recordingConnection{}
		require.NoError(t, e.wsConnect(t.Context(), conn), "wsConnect must not error")
		assert.Equal(t, 1, conn.dialled, "wsConnect should dial the connection")
		exp := &websocket.PingHandler{MessageType: gws.TextMessage, Message: []byte(`{"action":"ping","ping":"gct"}`), Delay: wsPingDelay}
		assert.Equal(t, exp, conn.pingHandler, "wsConnect should ping the connection")
	})
	t.Run("failed dial", func(t *testing.T) {
		t.Parallel()
		failing := &recordingConnection{dialErr: errSendFailed}
		assert.ErrorIs(t, e.wsConnect(t.Context(), failing), errSendFailed, "wsConnect should report a failed dial")
		assert.Nil(t, failing.pingHandler, "wsConnect should not ping a connection it couldn't dial")
	})
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	// The frame cases share one exchange and connection, so they run serially and in order: the ping case expects its
	// pong alone sent, and the unknown frame case nothing relayed by the frames before it
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	t.Run("ping", func(t *testing.T) {
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"ping":"1792889772462","action":"ping"}`)), "wsHandleData must answer a ping")
		assert.Equal(t, []any{&websocketHeartbeatRequest{Action: "pong", Pong: "1792889772462"}}, conn.messages(), "wsHandleData should answer a ping with its ID")
	})
	t.Run("pong", func(t *testing.T) {
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"action":"pong","pong":"gct"}`)), "wsHandleData must accept a pong")
	})
	t.Run("error frame", func(t *testing.T) {
		err := ex.wsHandleData(t.Context(), conn, []byte(liveInvalidPairFrame))
		require.ErrorIs(t, err, errWebsocketError, "wsHandleData must report an error frame")
		assert.EqualError(t, err, "websocket error: Invalid order pairs:[notapair_usdt]", "wsHandleData should leave the caller's IP out of the error")
	})
	t.Run("malformed frame", func(t *testing.T) {
		assert.Error(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":`)), "wsHandleData should report a malformed frame")
	})
	for _, frameType := range []string{wsTickerChannel, wsDepthChannel, wsTradeChannel, wsKlineChannel, wsOrderUpdateChannel, wsAssetUpdateChannel} {
		t.Run(frameType+" frame without its update", func(t *testing.T) {
			err := ex.wsHandleData(t.Context(), conn, []byte(`{"type":"`+frameType+`","pair":"btc_usdt"}`))
			assert.ErrorIsf(t, err, common.ErrNoResponse, "wsHandleData should report a %s frame without its update", frameType)
		})
	}
	t.Run("unknown frame", func(t *testing.T) {
		require.Empty(t, relayedPayloads(ex), "wsHandleData must relay nothing for the frames so far")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"type":"incrDepth","pair":"btc_usdt"}`)), "wsHandleData must not error for an unknown frame")
		payloads := relayedPayloads(ex)
		require.Len(t, payloads, 1, "wsHandleData must relay an unknown frame")
		assert.IsType(t, websocket.UnhandledMessageWarning{}, payloads[0], "wsHandleData should relay an unknown frame as unhandled")
	})

	t.Run("public handler errors", func(t *testing.T) {
		t.Parallel()
		// The cases share one exchange and run serially, so the check after them sees anything any of them relayed
		ex := newChannelTestExchange(t)
		conn := &recordingConnection{}
		for _, tc := range []struct {
			name  string
			frame string
			err   error
		}{
			{name: "unknown pair ticker", frame: `{"tick":{"latest":1},"type":"tick","pair":"nope_usdt","TS":"2026-10-25T08:23:20.879"}`, err: currency.ErrPairNotFound},
			{name: "unknown pair depth", frame: `{"depth":{"asks":[],"bids":[]},"type":"depth","pair":"nope_usdt","TS":"2026-10-25T08:23:20.879"}`, err: currency.ErrPairNotFound},
			{name: "unknown pair trade", frame: `{"trade":{"price":1,"volume":1,"direction":"buy"},"type":"trade","pair":"nope_usdt"}`, err: currency.ErrPairNotFound},
			{name: "unknown pair candle", frame: `{"kbar":{"slot":"1min"},"type":"kbar","pair":"nope_usdt"}`, err: currency.ErrPairNotFound},
			{name: "unknown trade direction", frame: `{"trade":{"price":1,"volume":1,"direction":"hold"},"type":"trade","pair":"btc_usdt"}`, err: order.ErrSideIsInvalid},
			{name: "unknown candle slot", frame: `{"kbar":{"slot":"2min"},"type":"kbar","pair":"btc_usdt"}`, err: kline.ErrUnsupportedInterval},
		} {
			t.Run(tc.name, func(t *testing.T) {
				err := ex.wsHandleData(t.Context(), conn, []byte(tc.frame))
				assert.ErrorIsf(t, err, tc.err, "wsHandleData should reject the %s", tc.name)
			})
		}
		assert.Empty(t, relayedPayloads(ex), "wsHandleData should relay nothing it rejected")
	})

	t.Run("account handler errors", func(t *testing.T) {
		t.Parallel()
		// The cases share one exchange and run serially, so the check after the rejected updates sees anything any of
		// them relayed
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
			t.Run(tc.name, func(t *testing.T) {
				err := ex.wsHandleData(t.Context(), conn, []byte(tc.frame))
				assert.ErrorIsf(t, err, tc.err, "wsHandleData should reject an update with an %s", tc.name)
			})
		}
		assert.Empty(t, relayedPayloads(ex), "wsHandleData should relay nothing it rejected")
		t.Run("trade without the fills feed", func(t *testing.T) {
			ex.Websocket.Fills.Setup(false, ex.Websocket.DataHandler)
			frame := `{"orderUpdate":{"accAmt":"1","amount":"1","symbol":"btc_usdt","type":"buy","orderStatus":2,"price":"1","txUuid":"t1","uuid":"o1","updateTime":1792888874775},"type":"orderUpdate"}`
			require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must relay an order's trade without the fills feed")
			payloads := relayedPayloads(ex)
			require.Len(t, payloads, 1, "wsHandleData must relay only the order without the fills feed")
			assert.IsType(t, &order.Detail{}, payloads[0], "wsHandleData should relay the order")
		})
	})
}

func TestWebsocketTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		in   string
		exp  time.Time
	}{
		{name: "milliseconds", in: `"2019-06-28T17:49:22.722"`, exp: time.Date(2019, 6, 28, 17, 49, 22, 722000000, beijingTime)},
		{name: "whole seconds", in: `"2019-06-28T17:45:00"`, exp: time.Date(2019, 6, 28, 17, 45, 0, 0, beijingTime)},
		{name: "empty string", in: `""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got WebsocketTime
			require.NoErrorf(t, got.UnmarshalJSON([]byte(tc.in)), "UnmarshalJSON must not error for %s", tc.in)
			assert.Truef(t, tc.exp.Equal(got.Time()), "UnmarshalJSON should decode %s at UTC+8", tc.in)
		})
	}
	t.Run("another layout", func(t *testing.T) {
		t.Parallel()
		var got WebsocketTime
		assert.ErrorIs(t, got.UnmarshalJSON([]byte(`"28/06/2019"`)), errMalformedWebsocketTime, "UnmarshalJSON should reject a time in another layout")
	})
	t.Run("not a string", func(t *testing.T) {
		t.Parallel()
		var got WebsocketTime
		assert.Error(t, got.UnmarshalJSON([]byte(`1561704577786`)), "UnmarshalJSON should reject a time that isn't a string")
	})
	t.Run("both rejected by one receiver in turn", func(t *testing.T) {
		t.Parallel()
		var got WebsocketTime
		assert.ErrorIs(t, got.UnmarshalJSON([]byte(`"28/06/2019"`)), errMalformedWebsocketTime, "UnmarshalJSON should reject a time in another layout")
		assert.Error(t, got.UnmarshalJSON([]byte(`1561704577786`)), "UnmarshalJSON should reject a time that isn't a string after rejecting another")
	})
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
	t.Run("market data channels", func(t *testing.T) {
		t.Parallel()
		// The order book takes 7 levels, the depth the recorded frame was streamed at: LBank documents 10, 50 and 100
		// only, but streams other depths too
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
	})
	t.Run("unlisted pair", func(t *testing.T) {
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
	})
}

// stopWebsocketTestExchange disables an exchange's websocket, so its connection monitor stops, then shuts it down.
// Shutdown waits for every routine the websocket started, so one that ignores it fails the test rather than hanging it
func stopWebsocketTestExchange(t *testing.T, ex *Exchange) {
	t.Helper()
	if ex.Websocket.IsEnabled() {
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- ex.Websocket.Shutdown() }()
	select {
	case err := <-stopped:
		if err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	case <-time.After(30 * time.Second):
		assert.Fail(t, "Shutdown should stop every routine the websocket started")
	}
}

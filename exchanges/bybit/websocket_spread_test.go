package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errFixtureSend = errors.New("fixture send failure")

// topicFixtureConnection answers subscription requests as a Bybit topic connection does: reply builds the frame the
// server would send, and handler routes it through the connection's match to the request waiting for it, so a request
// whose signature the handler cannot derive fails. Without a handler the reply is returned as is
type topicFixtureConnection struct {
	websocket.Connection
	match   *websocket.Match
	reply   func(*SubscriptionRequest) []byte
	handler func(context.Context, websocket.Connection, []byte) error
	sent    []*SubscriptionRequest
	sendErr error
}

func (c *topicFixtureConnection) SendMessageReturnResponse(ctx context.Context, _ request.EndpointLimit, signature, payload any) ([]byte, error) {
	req, ok := payload.(*SubscriptionRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected payload %T", payload)
	}
	c.sent = append(c.sent, req)
	if c.sendErr != nil {
		return nil, c.sendErr
	}
	if c.handler == nil {
		return c.reply(req), nil
	}
	ch, err := c.match.Set(signature, 1)
	if err != nil {
		return nil, err
	}
	if err := c.handler(ctx, c, c.reply(req)); err != nil {
		c.match.RemoveSignature(signature)
		return nil, err
	}
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(time.Second):
		// A reply the handler never routes would otherwise keep the test waiting until go test times it out. A stall
		// can leave the timer and a routed reply both ready, and select picks either, so the reply is looked for again
		select {
		case resp := <-ch:
			return resp, nil
		default:
		}
		c.match.RemoveSignature(signature)
		return nil, fmt.Errorf("%w: %v", websocket.ErrSignatureTimeout, signature)
	}
}

func (c *topicFixtureConnection) RequireMatchWithData(signature any, data []byte) error {
	return c.match.RequireMatchWithData(signature, data)
}

func TestTopicFixtureConnectionUnroutedReply(t *testing.T) {
	t.Parallel()
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`{}`) }, handler: func(context.Context, websocket.Connection, []byte) error { return nil }}
	// The fixture's own timeout is under test, so the wait for it is bounded here
	errs := make(chan error, 1)
	go func() {
		_, err := conn.SendMessageReturnResponse(t.Context(), wsSubscriptionEPL, "unrouted", &SubscriptionRequest{})
		errs <- err
	}()
	select {
	case err := <-errs:
		require.ErrorIs(t, err, websocket.ErrSignatureTimeout, "SendMessageReturnResponse must time out waiting for a reply the handler did not route")
	case <-time.After(10 * time.Second):
		require.FailNow(t, "SendMessageReturnResponse must give up waiting for a reply the handler did not route")
	}
	_, err := conn.match.Set("unrouted", 1)
	assert.NoError(t, err, "SendMessageReturnResponse should give up the signature it timed out on")
}

// commandReply returns a function building the COMMAND_RESP reply to a request; it lists the request's topics in
// reverse, as Bybit does not keep their order, and fails topics containing FAIL
func commandReply(tb testing.TB) func(*SubscriptionRequest) []byte {
	tb.Helper()
	return func(req *SubscriptionRequest) []byte {
		reply := CommandResponse{Success: true, ConnectionID: "dat42tru7geqm1secqg0-1k55h", Type: "COMMAND_RESP", Data: CommandResponseTopics{FailTopics: []string{}, SuccessTopics: []string{}}}
		for _, topic := range slices.Backward(req.Arguments) {
			if strings.Contains(topic, "FAIL") {
				reply.Data.FailTopics = append(reply.Data.FailTopics, topic)
			} else {
				reply.Data.SuccessTopics = append(reply.Data.SuccessTopics, topic)
			}
		}
		b, err := json.Marshal(reply)
		require.NoError(tb, err, "Marshal must not error")
		return b
	}
}

// liveWindowFrames subscribes to topics on a Bybit public stream and returns the frames received within window. Reads
// run in a goroutine, since a read deadline would break the connection
func liveWindowFrames(t *testing.T, url string, topics []string, window time.Duration) [][]byte {
	t.Helper()
	conn, resp, err := gws.DefaultDialer.DialContext(t.Context(), url, nil)
	require.NoError(t, err, "Dial must not error")
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close(), "Closing the handshake body must not error")
	}
	defer conn.Close()
	require.NoError(t, conn.WriteJSON(&SubscriptionRequest{RequestID: "gct-live", Operation: "subscribe", Arguments: topics}), "WriteJSON must not error")
	frames := make(chan []byte)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case frames <- msg:
			case <-done:
				return
			}
		}
	}()
	var got [][]byte
	timeout := time.After(window)
	for {
		select {
		case msg := <-frames:
			got = append(got, msg)
		case <-timeout:
			return got
		}
	}
}

// livePrivateSubscribe authenticates on the private stream and subscribes to topics, returning Bybit's reply to the
// subscription. Reads run in a goroutine, since a read deadline would break the connection
func livePrivateSubscribe(t *testing.T, topics []string) *SubscriptionResponse {
	t.Helper()
	conn, resp, err := gws.DefaultDialer.DialContext(t.Context(), websocketPrivate, nil)
	require.NoError(t, err, "Dial must not error")
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close(), "Closing the handshake body must not error")
	}
	defer conn.Close()
	auth, err := e.GetAuthenticationPayload(t.Context(), "gct-live-auth")
	require.NoError(t, err, "GetAuthenticationPayload must not error")
	require.NoError(t, conn.WriteJSON(auth), "WriteJSON must not error for the authentication")
	require.NoError(t, conn.WriteJSON(&SubscriptionRequest{RequestID: "gct-live-subscribe", Operation: "subscribe", Arguments: topics}), "WriteJSON must not error for the subscription")
	replies := make(chan *SubscriptionResponse)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var reply SubscriptionResponse
			if json.Unmarshal(msg, &reply) != nil || reply.RequestID != "gct-live-subscribe" {
				continue
			}
			select {
			case replies <- &reply:
			case <-done:
			}
			return
		}
	}()
	select {
	case reply := <-replies:
		return reply
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the private stream must answer the subscription")
	}
	return nil
}

// relayedFrames returns the payloads ex relayed, in order
func relayedFrames(ex *Exchange) []any {
	var got []any
	for len(ex.Websocket.DataHandler.C) > 0 {
		got = append(got, (<-ex.Websocket.DataHandler.C).Data)
	}
	return got
}

// TestTopicConnectionSetups covers the connections whose subscriptions name a Bybit topic rather than an asset
func TestTopicConnectionSetups(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	cfg := &config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: 2 * time.Second}
	spreadURL, eventURL, rfqURL := testnetStreamOrigin+spreadPublicPath, testnetStreamOrigin+eventPublicPath, testnetStreamOrigin+rfqPublicPath
	for name, tc := range map[string]struct {
		setup     *websocket.ConnectionSetup
		streamURL string
	}{
		"SpreadPublicConnectionSetup": {setup: ex.SpreadPublicConnectionSetup(cfg, spreadURL), streamURL: spreadURL},
		"EventPublicConnectionSetup":  {setup: ex.EventPublicConnectionSetup(cfg, eventURL), streamURL: eventURL},
		"RFQPublicConnectionSetup":    {setup: ex.RFQPublicConnectionSetup(cfg, rfqURL), streamURL: rfqURL},
	} {
		setup := tc.setup
		assert.Equalf(t, tc.streamURL, setup.URL, "%s should connect to the URL given", name)
		assert.Equalf(t, time.Second, setup.ResponseCheckTimeout, "%s should use the configured response check timeout", name)
		assert.Equalf(t, 2*time.Second, setup.ResponseMaxLimit, "%s should use the configured response limit", name)
		for field, fn := range map[string]any{"Connector": setup.Connector, "Subscriber": setup.Subscriber, "Unsubscriber": setup.Unsubscriber, "Handler": setup.Handler} {
			assert.NotNilf(t, fn, "%s should set %s", name, field)
		}
		require.NotNilf(t, setup.GenerateSubscriptions, "%s must set GenerateSubscriptions", name)
		subs, err := setup.GenerateSubscriptions()
		require.NoErrorf(t, err, "%s GenerateSubscriptions must not error", name)
		assert.Emptyf(t, subs, "%s GenerateSubscriptions should generate no subscriptions by default", name)
	}
}

func TestGenerateSpreadPublicSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	symbol := map[string]any{"symbol": spreadTestSymbol}
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel},
		{Enabled: true, Channel: ChannelSpreadOrderbook, Params: symbol},
		{Enabled: true, Channel: ChannelSpreadOrderbook, Params: map[string]any{"symbol": "ETHUSDT_ETH/USDT"}, Levels: 1},
		{Enabled: true, Channel: ChannelSpreadPublicTrade, Params: symbol},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol},
		{Enabled: false, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": "SOLUSDT_SOL/USDT"}},
		{Enabled: true, Channel: ChannelEventOrderbook, Params: symbol},
	}
	subs, err := ex.generateSpreadPublicSubscriptions()
	require.NoError(t, err, "generateSpreadPublicSubscriptions must not error")
	assert.Equal(t, []string{
		"orderbook.1.ETHUSDT_ETH/USDT",
		"orderbook.25." + spreadTestSymbol,
		"publicTrade." + spreadTestSymbol,
		"tickers." + spreadTestSymbol,
	}, subs.QualifiedChannels(), "generateSpreadPublicSubscriptions should qualify each enabled spread subscription by its topic")
	for _, s := range subs {
		assert.IsTypef(t, topicKey{}, s.Key, "subscription %s should be keyed by its topic", s.QualifiedChannel)
	}

	unusable := subscription.List{
		{Enabled: true, Channel: ChannelSpreadTicker},
		{Enabled: true, Channel: ChannelSpreadOrderbook, Params: map[string]any{"symbol": ""}},
		{Enabled: true, Channel: ChannelSpreadPublicTrade, Params: map[string]any{"symbol": 1}},
	}
	ex.Features.Subscriptions = append(subscription.List{{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol}}, unusable...)
	subs, err = ex.generateSpreadPublicSubscriptions()
	assert.NoError(t, err, "generateSpreadPublicSubscriptions should not error for subscriptions without a usable symbol")
	assert.Equal(t, []string{"tickers." + spreadTestSymbol}, subs.QualifiedChannels(), "generateSpreadPublicSubscriptions should return only the subscriptions with a symbol")
	for _, s := range unusable {
		assert.Truef(t, s.Enabled, "generateSpreadPublicSubscriptions should leave %s with params %v as configured", s.Channel, s.Params)
	}
	// Generation only reads the configuration, so it can run while the configuration is saved
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 50 {
			_, err := json.Marshal(ex.Features.Subscriptions)
			assert.NoError(t, err, "Marshal should not error")
		}
	})
	for range 50 {
		_, err := ex.generateSpreadPublicSubscriptions()
		assert.NoError(t, err, "generateSpreadPublicSubscriptions should not error while the configuration is saved")
	}
	wg.Wait()

	// Setup disables a configured spread or event subscription without a symbol, and the websocket connects every
	// other subscription
	t.Run("connect", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		cfg, err := ex.GetStandardConfig()
		require.NoError(t, err, "GetStandardConfig must not error")
		noSymbol := &subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker}
		emptySymbol := &subscription.Subscription{Enabled: true, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": ""}}
		cfg.Features = &config.FeaturesConfig{
			Enabled: config.FeaturesEnabledConfig{Websocket: true},
			Subscriptions: subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
				noSymbol,
				{Enabled: true, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": spreadTestSymbol}},
				emptySymbol,
				{Enabled: true, Channel: ChannelEventOrderbook, Params: map[string]any{"symbol": eventTestTargetSymbol}},
			},
		}
		require.NoError(t, ex.Setup(cfg), "Setup must not error")
		assert.False(t, noSymbol.Enabled, "Setup should disable the spread subscription without a symbol in the configuration")
		assert.False(t, emptySymbol.Enabled, "Setup should disable the event subscription with an empty symbol in the configuration")
		ex.Name = t.Name()
		spotPair := currency.NewPair(currency.ETH, currency.USDT)
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotPair}, false), "StorePairs must not error for the available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotPair}, true), "StorePairs must not error for the enabled pairs")
		// Spot answers with the request ID, and the spread and event streams with the topics they applied
		commandTopics := []string{"tickers." + spreadTestSymbol, "orderbook.25." + eventTestTargetSymbol}
		server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			var req SubscriptionRequest
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			if req.Operation != subscribeOperation && req.Operation != unsubscribeOperation {
				return nil
			}
			if slices.ContainsFunc(req.Arguments, func(topic string) bool { return slices.Contains(commandTopics, topic) }) {
				return c.WriteJSON(&CommandResponse{Success: true, ConnectionID: "dat42tru7geqm1secqg0-1k55h", Type: commandResponseType, Data: CommandResponseTopics{FailTopics: []string{}, SuccessTopics: req.Arguments}})
			}
			return c.WriteJSON(&SubscriptionResponse{Success: true, ConnectionID: "d9avrat34tmf5pih98i0-gsmne", RequestID: req.RequestID, Operation: req.Operation})
		}))
		server.Start()
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
		require.NoError(t, ex.Websocket.Connect(context.WithoutCancel(t.Context())), "Connect must not error")
		t.Cleanup(func() { stopWebsocket(t, ex) })
		assert.True(t, ex.Websocket.IsConnected(), "the websocket should be connected")
		exp := []string{"orderbook.25." + eventTestTargetSymbol, "tickers." + spreadTestSymbol, "tickers.ETHUSDT"}
		assert.Equal(t, exp, ex.Websocket.GetSubscriptions().QualifiedChannels(), "Connect should subscribe every subscription with a symbol")
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
		assert.Equal(t, exp, ex.Websocket.GetSubscriptions().QualifiedChannels(), "FlushChannels should keep the subscriptions")
	})
}

func TestTopicKey(t *testing.T) {
	t.Parallel()
	a := topicSubscription(&subscription.Subscription{Channel: ChannelSpreadTicker}, "tickers.BTCUSDT_BTC/USDT")
	b := topicSubscription(&subscription.Subscription{Channel: ChannelSpreadTicker}, "tickers.ETHUSDT_ETH/USDT")
	c := topicSubscription(&subscription.Subscription{Channel: ChannelSpreadTicker}, "tickers.BTCUSDT_BTC/USDT")
	key, ok := a.Key.(topicKey)
	require.True(t, ok, "topicSubscription must key the subscription by its topic")
	sameTopic, ok := c.Key.(topicKey)
	require.True(t, ok, "topicSubscription must key the subscription by its topic")
	otherTopic, ok := b.Key.(topicKey)
	require.True(t, ok, "topicSubscription must key the subscription by its topic")
	assert.True(t, key.Match(sameTopic), "Match should match a subscription to the same channel and topic")
	assert.False(t, key.Match(otherTopic), "Match should not match a subscription to another topic")
	assert.False(t, key.Match(nil), "Match should not match a nil key")
	assert.False(t, topicKey{}.Match(key), "Match should not match from an uninitialised key")
	assert.Equal(t, a, key.GetSubscription(), "GetSubscription should return the keyed subscription")
	assert.Equal(t, "spread.tickers tickers.BTCUSDT_BTC/USDT", key.String(), "String should name the channel and topic")
	assert.Equal(t, "Uninitialised topicKey", topicKey{}.String(), "String should describe an uninitialised key")

	store := subscription.NewStore()
	require.NoError(t, store.Add(a), "Add must not error")
	require.NoError(t, store.Add(b), "Add must store a second topic of the same channel")
	assert.ErrorIs(t, store.Add(c), subscription.ErrDuplicate, "Add should reject a second subscription to the same topic")
}

func TestSubscriptionSymbol(t *testing.T) {
	t.Parallel()
	symbol, err := subscriptionSymbol(&subscription.Subscription{Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": spreadTestSymbol}})
	require.NoError(t, err, "subscriptionSymbol must not error")
	assert.Equal(t, spreadTestSymbol, symbol, "subscriptionSymbol should return the symbol param")
	for _, params := range []map[string]any{nil, {"symbol": ""}, {"symbol": 1}, {"pair": spreadTestSymbol}} {
		_, err = subscriptionSymbol(&subscription.Subscription{Channel: ChannelSpreadTicker, Params: params})
		require.ErrorIsf(t, err, errSubscriptionSymbolMissing, "subscriptionSymbol must refuse params %v", params)
		assert.ErrorContainsf(t, err, ChannelSpreadTicker, "subscriptionSymbol should name the channel missing a symbol for params %v", params)
	}
}

func TestDisableUnusableSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	symbol := map[string]any{"symbol": spreadTestSymbol}
	usable := subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol},
		{Enabled: true, Channel: ChannelEventPublicTrade, Params: map[string]any{"symbol": eventTestTargetSymbol}},
	}
	unusable := subscription.List{
		{Enabled: true, Channel: ChannelSpreadTicker},
		{Enabled: true, Channel: ChannelSpreadOrderbook, Params: map[string]any{"symbol": ""}},
		{Enabled: true, Channel: ChannelSpreadPublicTrade, Params: map[string]any{"symbol": 1}},
		{Enabled: true, Channel: ChannelEventOrderbook},
		{Enabled: true, Channel: ChannelEventPublicTrade, Params: map[string]any{"symbol": ""}},
		// The private connection generates its own subscriptions, so a configured authenticated one is never used
		{Enabled: true, Channel: subscription.MyOrdersChannel, Asset: asset.Spot, Authenticated: true},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol, Authenticated: true},
	}
	ex.Config.Features.Subscriptions = slices.Concat(usable, unusable)
	ex.SetSubscriptionsFromConfig()
	ex.disableUnusableSubscriptions()
	for _, s := range unusable {
		assert.Falsef(t, s.Enabled, "disableUnusableSubscriptions should disable %s with params %v", s.Channel, s.Params)
	}
	for _, s := range usable {
		assert.Truef(t, s.Enabled, "disableUnusableSubscriptions should leave %s with params %v enabled", s.Channel, s.Params)
	}
	assert.Equal(t, usable, ex.Features.Subscriptions, "disableUnusableSubscriptions should leave only the usable subscriptions enabled")

	// With nothing to disable, the enabled subscriptions are left as they are
	ex.Features.Subscriptions = slices.Clone(usable[:1])
	ex.disableUnusableSubscriptions()
	assert.Equal(t, usable[:1], ex.Features.Subscriptions, "disableUnusableSubscriptions should not take the subscriptions from the configuration again when it disables none")

	// An authenticated subscription is disabled whether or not the websocket can authenticate, and alone is reason to
	// take the enabled subscriptions from the configuration again
	for _, wsAuth := range []bool{false, true} {
		authEx := testInstance(t)
		authEx.API.AuthenticatedWebsocketSupport = wsAuth
		public := &subscription.Subscription{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot}
		authenticated := &subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker, Params: symbol, Authenticated: true}
		authEx.Config.Features.Subscriptions = subscription.List{public, authenticated}
		authEx.SetSubscriptionsFromConfig()
		authEx.disableUnusableSubscriptions()
		assert.Falsef(t, authenticated.Enabled, "disableUnusableSubscriptions should disable an authenticated subscription with websocket authentication %t", wsAuth)
		assert.Equalf(t, subscription.List{public}, authEx.Features.Subscriptions, "disableUnusableSubscriptions should leave only the unauthenticated subscription enabled with websocket authentication %t", wsAuth)
	}
}

func TestNeedsSymbol(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{ChannelSpreadOrderbook, ChannelSpreadPublicTrade, ChannelSpreadTicker, ChannelEventOrderbook, ChannelEventPublicTrade} {
		assert.Truef(t, needsSymbol(channel), "needsSymbol should report that %s needs a symbol", channel)
	}
	for _, channel := range []string{subscription.TickerChannel, ChannelSpreadOrder, ChannelSpreadExecution} {
		assert.Falsef(t, needsSymbol(channel), "needsSymbol should report that %s needs no symbol", channel)
	}
}

func TestTopicOrderbookDepth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		levels int
		exp    int
	}{
		{levels: 0, exp: 25},
		{levels: 1, exp: 1},
		{levels: 25, exp: 25},
	} {
		assert.Equalf(t, tc.exp, topicOrderbookDepth(&subscription.Subscription{Channel: ChannelSpreadOrderbook, Levels: tc.levels}), "topicOrderbookDepth should return the depth of a subscription with %d levels", tc.levels)
	}
}

func TestSpreadPublicSubscribe(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: ChannelSpreadOrderbook, Params: map[string]any{"symbol": spreadTestSymbol}},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": spreadTestSymbol}},
		{Enabled: true, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": "FAILUSDT_FAIL/USDT"}},
	}
	subs, err := ex.generateSpreadPublicSubscriptions()
	require.NoError(t, err, "generateSpreadPublicSubscriptions must not error")
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: commandReply(t), handler: ex.wsHandleSpreadPublicData}

	err = ex.spreadPublicSubscribe(t.Context(), conn, subs)
	require.ErrorIs(t, err, errTopicsRejected, "spreadPublicSubscribe must report the topics Bybit failed")
	assert.ErrorContains(t, err, "tickers.FAILUSDT_FAIL/USDT", "spreadPublicSubscribe should name the failed topic")
	require.Len(t, conn.sent, 1, "spreadPublicSubscribe must send one request for the batch")
	assert.Equal(t, "subscribe", conn.sent[0].Operation, "spreadPublicSubscribe should send a subscribe request")
	assert.Equal(t, subs.QualifiedChannels(), conn.sent[0].Arguments, "spreadPublicSubscribe should request every topic")
	for _, s := range subs[:2] {
		stored := ex.Websocket.GetSubscription(s)
		require.NotNilf(t, stored, "spreadPublicSubscribe must store subscription %s", s.QualifiedChannel)
		assert.Equalf(t, subscription.SubscribedState, stored.State(), "subscription %s should be subscribed", s.QualifiedChannel)
	}
	assert.Nil(t, ex.Websocket.GetSubscription(subs[2]), "spreadPublicSubscribe should not store a failed subscription")

	err = ex.spreadPublicUnsubscribe(t.Context(), conn, subs[:2])
	require.NoError(t, err, "spreadPublicUnsubscribe must not error")
	assert.Equal(t, "unsubscribe", conn.sent[1].Operation, "spreadPublicUnsubscribe should send an unsubscribe request")
	for _, s := range subs[:2] {
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "spreadPublicUnsubscribe should remove subscription %s", s.QualifiedChannel)
	}

	conn.sendErr = errFixtureSend
	err = ex.spreadPublicSubscribe(t.Context(), conn, subs[:1])
	assert.ErrorIs(t, err, errFixtureSend, "spreadPublicSubscribe should return a send failure")

	conn.sendErr = nil
	conn.reply = func(*SubscriptionRequest) []byte {
		return []byte(`{"success":false,"conn_id":"dat42tru7geqm1secqg0-1k55h","data":{"failTopics":[],"successTopics":["orderbook.25.BTCUSDT_BTC/USDT"]},"type":"COMMAND_RESP"}`)
	}
	err = ex.spreadPublicSubscribe(t.Context(), conn, subs[:1])
	assert.ErrorIs(t, err, errTopicsRejected, "spreadPublicSubscribe should fail every topic of an unsuccessful reply")

	unrouted := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`{"success":`) }}
	assert.Error(t, ex.spreadPublicSubscribe(t.Context(), unrouted, subs[:1]), "spreadPublicSubscribe should return a malformed reply's error")
}

func TestManageCommandSubscriptions(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	bases := []string{"BTC", "ETH", "FAIL", "XRP", "DOGE", "ADA", "LTC", "DOT", "LINK", "TRX", "AVAX"}
	subs := make(subscription.List, len(bases))
	for i, base := range bases {
		symbol := base + "USDT_" + base + "/USDT"
		subs[i] = topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker, Params: map[string]any{"symbol": symbol}}, "tickers."+symbol)
	}
	// The handler passes each reply to the request awaiting the topics it lists, in any order, so a request waiting on
	// anything else would not be answered
	conn := &topicFixtureConnection{match: websocket.NewMatch(), reply: commandReply(t), handler: ex.wsHandleSpreadPublicData}
	err := ex.manageCommandSubscriptions(t.Context(), conn, "subscribe", subs)
	require.ErrorIs(t, err, errTopicsRejected, "manageCommandSubscriptions must report the topics Bybit failed")
	assert.ErrorContains(t, err, "subscribe tickers.FAILUSDT_FAIL/USDT", "manageCommandSubscriptions should name the failed topic")
	require.Len(t, conn.sent, 2, "manageCommandSubscriptions must send ten topics a request")
	assert.Equal(t, subs[:10].QualifiedChannels(), conn.sent[0].Arguments, "manageCommandSubscriptions should send the first ten topics together")
	assert.Equal(t, []string{"tickers.AVAXUSDT_AVAX/USDT"}, conn.sent[1].Arguments, "manageCommandSubscriptions should send the rest in another request")
	for i, s := range subs {
		if i == 2 {
			assert.Nil(t, ex.Websocket.GetSubscription(s), "manageCommandSubscriptions should not store a failed topic")
			continue
		}
		stored := ex.Websocket.GetSubscription(s)
		require.NotNilf(t, stored, "manageCommandSubscriptions must store topic %s", s.QualifiedChannel)
		assert.Equalf(t, subscription.SubscribedState, stored.State(), "manageCommandSubscriptions should mark topic %s subscribed", s.QualifiedChannel)
	}

	conn.sent = nil
	require.NoError(t, ex.manageCommandSubscriptions(t.Context(), conn, "unsubscribe", subs[:2]), "manageCommandSubscriptions must not error when unsubscribing")
	require.Len(t, conn.sent, 1, "manageCommandSubscriptions must send one unsubscribe request")
	assert.Equal(t, "unsubscribe", conn.sent[0].Operation, "manageCommandSubscriptions should send the operation")
	for _, s := range subs[:2] {
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "manageCommandSubscriptions should remove topic %s", s.QualifiedChannel)
	}

	conn.reply = func(req *SubscriptionRequest) []byte {
		b, err := json.Marshal(&CommandResponse{ConnectionID: "dat42tru7geqm1secqg0-1k55h", Type: "COMMAND_RESP", Data: CommandResponseTopics{FailTopics: []string{}, SuccessTopics: req.Arguments}})
		require.NoError(t, err, "Marshal must not error")
		return b
	}
	err = ex.manageCommandSubscriptions(t.Context(), conn, "subscribe", subs[:2])
	require.ErrorIs(t, err, errTopicsRejected, "manageCommandSubscriptions must fail every topic of an unsuccessful reply")
	for _, s := range subs[:2] {
		assert.ErrorContainsf(t, err, s.QualifiedChannel, "manageCommandSubscriptions should name failed topic %s", s.QualifiedChannel)
		assert.Nilf(t, ex.Websocket.GetSubscription(s), "manageCommandSubscriptions should not store failed topic %s", s.QualifiedChannel)
	}

	conn.sendErr = errFixtureSend
	conn.sent = nil
	err = ex.manageCommandSubscriptions(t.Context(), conn, "subscribe", subs[:2])
	require.ErrorIs(t, err, errFixtureSend, "manageCommandSubscriptions must return a failed send")
	assert.ErrorContains(t, err, "subscribe tickers.BTCUSDT_BTC/USDT, tickers.ETHUSDT_ETH/USDT", "manageCommandSubscriptions should name the topics it could not send")
	assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "manageCommandSubscriptions should not store a topic it could not send")

	nullReply := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`null`) }}
	require.ErrorIs(t, ex.manageCommandSubscriptions(t.Context(), nullReply, "subscribe", subs[:2]), errTopicsRejected, "manageCommandSubscriptions must fail every topic of a null reply")
	unrouted := &topicFixtureConnection{match: websocket.NewMatch(), reply: func(*SubscriptionRequest) []byte { return []byte(`{"success":`) }}
	require.Error(t, ex.manageCommandSubscriptions(t.Context(), unrouted, "subscribe", subs[:2]), "manageCommandSubscriptions must error for a reply that does not decode")
	assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "manageCommandSubscriptions should not store a topic whose reply does not decode")
	// Bybit lists a topic once in its reply however often a request names it, as two depths of one spread orderbook do
	ethBook := func(levels int) *subscription.Subscription {
		return topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelSpreadOrderbook, Params: map[string]any{"symbol": "ETHUSDT_ETH/USDT"}, Levels: levels}, "orderbook.25.ETHUSDT_ETH/USDT")
	}
	once := &topicFixtureConnection{match: websocket.NewMatch(), handler: ex.wsHandleSpreadPublicData, reply: func(req *SubscriptionRequest) []byte {
		b, err := json.Marshal(&CommandResponse{Success: true, ConnectionID: "dat42tru7geqm1secqg0-1k55h", Type: "COMMAND_RESP", Data: CommandResponseTopics{FailTopics: []string{}, SuccessTopics: slices.Compact(slices.Sorted(slices.Values(req.Arguments)))}})
		require.NoError(t, err, "Marshal must not error")
		return b
	}}
	repeated := subscription.List{ethBook(0), ethBook(25)}
	err = ex.manageCommandSubscriptions(t.Context(), once, "subscribe", repeated)
	require.ErrorIs(t, err, subscription.ErrDuplicate, "manageCommandSubscriptions must report the repeated subscription as a duplicate")
	require.Len(t, once.sent, 1, "manageCommandSubscriptions must send one request")
	assert.Equal(t, []string{"orderbook.25.ETHUSDT_ETH/USDT"}, once.sent[0].Arguments, "manageCommandSubscriptions should name a repeated topic once")
	assert.NotNil(t, ex.Websocket.GetSubscription(repeated[0]), "manageCommandSubscriptions should store a topic its batch names twice")
}

func TestApplyTopicResults(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	conn := &FixtureConnection{}
	btc := topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker}, "tickers.BTCUSDT_BTC/USDT")
	eth := topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker}, "tickers.ETHUSDT_ETH/USDT")
	sol := topicSubscription(&subscription.Subscription{Enabled: true, Channel: ChannelSpreadTicker}, "tickers.SOLUSDT_SOL/USDT")

	err := ex.applyTopicResults(conn, "subscribe", subscription.List{btc, eth, sol}, []string{"tickers.ETHUSDT_ETH/USDT"})
	require.ErrorIs(t, err, errTopicsRejected, "applyTopicResults must report a failed topic")
	assert.ErrorContains(t, err, "subscribe tickers.ETHUSDT_ETH/USDT", "applyTopicResults should name the failed topic")
	assert.ElementsMatch(t, []string{"tickers.BTCUSDT_BTC/USDT", "tickers.SOLUSDT_SOL/USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "applyTopicResults should store the topics that did not fail")
	assert.Equal(t, subscription.SubscribedState, btc.State(), "applyTopicResults should mark a stored topic subscribed")

	require.NoError(t, ex.applyTopicResults(conn, "unsubscribe", subscription.List{btc}, nil), "applyTopicResults must not error when unsubscribing")
	assert.Equal(t, []string{"tickers.SOLUSDT_SOL/USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "applyTopicResults should remove an unsubscribed topic")
	assert.Equal(t, subscription.UnsubscribedState, btc.State(), "applyTopicResults should mark a removed topic unsubscribed")

	err = ex.applyTopicResults(conn, "unsubscribe", subscription.List{sol}, []string{"tickers.SOLUSDT_SOL/USDT"})
	require.ErrorIs(t, err, errTopicsRejected, "applyTopicResults must report a topic that failed to unsubscribe")
	assert.ErrorContains(t, err, "unsubscribe tickers.SOLUSDT_SOL/USDT", "applyTopicResults should name the topic that failed to unsubscribe")
	assert.Equal(t, []string{"tickers.SOLUSDT_SOL/USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "applyTopicResults should keep a topic that failed to unsubscribe")

	err = ex.applyTopicResults(conn, "subscribe", subscription.List{sol}, nil)
	require.ErrorIs(t, err, subscription.ErrDuplicate, "applyTopicResults must return a store error")
}

func TestHandleCommandFrame(t *testing.T) {
	t.Parallel()
	conn := &FixtureConnection{match: websocket.NewMatch()}
	replies, err := conn.match.Set(commandSignature([]string{"orderbook.25.BTCUSDT_BTC/USDT", "orderbook.25.FAILUSDT_FAIL/USDT", "tickers.BTCUSDT_BTC/USDT"}), 1)
	require.NoError(t, err, "match.Set must not error")
	// The reply lists its topics in neither the order they were requested nor sorted order
	reply := []byte(`{"success":true,"conn_id":"dat42tru7geqm1secqg0-1k55h","data":{"failTopics":["orderbook.25.FAILUSDT_FAIL/USDT"],"successTopics":["tickers.BTCUSDT_BTC/USDT","orderbook.25.BTCUSDT_BTC/USDT"]},"type":"COMMAND_RESP"}`)
	require.NoError(t, handleCommandFrame(conn, reply), "handleCommandFrame must pass a reply to the request awaiting its topics")
	require.Len(t, replies, 1, "handleCommandFrame must deliver the reply")
	assert.Equal(t, reply, <-replies, "handleCommandFrame should deliver the reply unchanged, whatever the order of its topics")

	for _, tc := range []struct {
		name  string
		frame string
		err   error
	}{
		{name: "a pong", frame: `{"args":["1791534256294"],"op":"pong"}`},
		{name: "a reply no request awaits", frame: `{"success":true,"conn_id":"dat42tru7geqm1secqg0-1k55h","data":{"failTopics":[],"successTopics":["tickers.ETHUSDT_ETH/USDT"]},"type":"COMMAND_RESP"}`, err: websocket.ErrSignatureNotMatched},
		{name: "a frame that is not a command reply", frame: `{"success":true,"ret_msg":"","op":"auth","conn_id":"dat42tru7geqm1secqg0-1k55h"}`, err: errUnhandledStreamData},
		{name: "a null frame", frame: `null`, err: errUnhandledStreamData},
	} {
		assert.ErrorIsf(t, handleCommandFrame(conn, []byte(tc.frame)), tc.err, "handleCommandFrame should handle %s", tc.name)
	}
	require.Error(t, handleCommandFrame(conn, []byte(`{"success":`)), "handleCommandFrame must error for a frame that does not decode")
}

func TestWebsocketHandleSpreadPublicData(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Name = t.Name()
	conn := &topicFixtureConnection{match: websocket.NewMatch()}
	handle := func(ctx context.Context, b []byte) error { return ex.wsHandleSpreadPublicData(ctx, conn, b) }

	if !mockTests {
		topics := []string{"orderbook.25." + spreadTestSymbol, "tickers." + spreadTestSymbol, "publicTrade." + spreadTestSymbol}
		replies, err := conn.match.Set(commandSignature(topics), 1)
		require.NoError(t, err, "match.Set must not error")
		for _, frame := range liveWindowFrames(t, mainnetStreamOrigin+spreadPublicPath, topics, 5*time.Second) {
			require.NoErrorf(t, handle(t.Context(), frame), "wsHandleSpreadPublicData must handle %s", frame)
		}
		require.Len(t, replies, 1, "wsHandleSpreadPublicData must route the subscription reply")
		var reply CommandResponse
		require.NoError(t, json.Unmarshal(<-replies, &reply), "Unmarshal must not error")
		assert.ElementsMatch(t, topics, reply.Data.SuccessTopics, "the spread stream should accept every topic")
		got := relayedFrames(ex)
		require.NotEmpty(t, got, "wsHandleSpreadPublicData must relay the stream's frames")
		for _, frame := range got {
			switch f := frame.(type) {
			case *WebsocketSpreadOrderbookUpdate:
				assert.Equal(t, spreadTestSymbol, f.Data.Symbol, "orderbook frames should be for the subscribed symbol")
			case *WebsocketSpreadTickerUpdate:
				assert.Equal(t, spreadTestSymbol, f.Data.Symbol, "ticker frames should be for the subscribed symbol")
				pair, err := currency.NewPairFromString(f.Data.Symbol)
				require.NoError(t, err, "NewPairFromString must not error")
				_, err = ex.GetCachedTicker(pair, asset.Spread)
				assert.NoError(t, err, "a relayed ticker frame's ticker should be stored")
			case *WebsocketSpreadPublicTradeUpdate:
				assert.NotEmpty(t, f.Data, "trade frames should hold trades")
			default:
				assert.Failf(t, "unexpected relayed frame", "the spread stream should not relay %T", frame)
			}
		}
		return
	}

	testexch.FixtureToDataHandler(t, "testdata/wsSpreadPublic.json", handle)
	exp := []any{
		&WebsocketSpreadOrderbookUpdate{
			Topic:     "orderbook.25.BTCUSDT_BTC/USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534204555)),
			Data: SpreadOrderbookData{
				Symbol: "BTCUSDT_BTC/USDT",
				Bids: []orderbook.Level{
					{Amount: 7.268, Price: -44.9},
					{Amount: 1.5, Price: -45.2},
				},
				Asks: []orderbook.Level{
					{Amount: 7.266, Price: -37.8},
				},
				UpdateID: 642456,
				Sequence: 3303636553,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791528922919)),
		},
		&WebsocketSpreadOrderbookUpdate{
			Topic:     "orderbook.25.BTCUSDT_BTC/USDT",
			Type:      "delta",
			Timestamp: types.Time(time.UnixMilli(1791534289889)),
			Data: SpreadOrderbookData{
				Symbol: "BTCUSDT_BTC/USDT",
				Bids: []orderbook.Level{
					{Price: -45.2},
				},
				Asks: []orderbook.Level{
					{Amount: 1.2, Price: -38.1},
				},
				UpdateID: 642457,
				Sequence: 3303636601,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791534289875)),
		},
		&WebsocketSpreadPublicTradeUpdate{
			Topic:     "publicTrade.BTCUSDT_BTC/USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791362633860)),
			Data: []SpreadWebsocketTrade{
				{
					OrderFillTimestamp: types.Time(time.UnixMilli(1791362633856)),
					Symbol:             "BTCUSDT_BTC/USDT",
					Side:               "Sell",
					Size:               0.119,
					Price:              -43.5,
					TickDirection:      "MinusTick",
					TradeID:            "b311eff0-b8a1-546c-992d-981b5c81f857",
					Sequence:           3302117152,
				},
			},
		},
		&WebsocketSpreadTickerUpdate{
			Topic:     "tickers.BTCUSDT_BTC/USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534255189)),
			Data: SpreadTicker{
				Symbol:              "BTCUSDT_BTC/USDT",
				BidPrice:            -44.9,
				BidSize:             7.268,
				AskPrice:            -37.8,
				AskSize:             7.266,
				LastPrice:           -43.5,
				HighPrice24Hour:     -43.3,
				LowPrice24Hour:      -43.5,
				PreviousPrice24Hour: -43.4,
				Volume24Hour:        0.238,
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleSpreadPublicData should relay every frame")
	pair, err := currency.NewPairFromString(spreadTestSymbol)
	require.NoError(t, err, "NewPairFromString must not error")
	_, err = ex.GetCachedTicker(pair, asset.Spread)
	assert.NoError(t, err, "wsHandleSpreadPublicData should store the ticker before relaying it")

	assert.NoError(t, handle(t.Context(), []byte(`{"args":["1791534256294"],"op":"pong"}`)), "wsHandleSpreadPublicData should ignore a pong")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"success":true,"conn_id":"x","data":{"failTopics":[],"successTopics":["tickers.X"]},"type":"COMMAND_RESP"}`)), websocket.ErrSignatureNotMatched, "wsHandleSpreadPublicData should reject a reply no request awaits")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"success":true,"type":"AUTH_RESP"}`)), errUnhandledStreamData, "wsHandleSpreadPublicData should reject an unknown frame")
	assert.ErrorIs(t, handle(t.Context(), []byte(`{"topic":"kline.1.BTCUSDT_BTC/USDT"}`)), errUnhandledStreamData, "wsHandleSpreadPublicData should reject an unknown topic")
	assert.Error(t, handle(t.Context(), []byte(`{"topic":`)), "wsHandleSpreadPublicData should reject malformed JSON")
	assert.Error(t, handle(t.Context(), []byte(`{"data":[]}`)), "wsHandleSpreadPublicData should reject a frame that is neither data nor a reply")
}

func TestWebsocketProcessSpreadTicker(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Name = t.Name()
	frames := fixtureFrames(t, "testdata/wsSpreadPublic.json")
	require.Len(t, frames, 4, "the spread fixture must hold two orderbook pushes, a trade push and a ticker push")
	require.NoError(t, ex.wsProcessSpreadTicker(t.Context(), frames[3]), "wsProcessSpreadTicker must not error")
	pair, err := currency.NewPairFromString(spreadTestSymbol)
	require.NoError(t, err, "NewPairFromString must not error")
	stored, err := ex.GetCachedTicker(pair, asset.Spread)
	require.NoError(t, err, "GetCachedTicker must find the stored spread ticker")
	expTicker := &ticker.Price{
		Last:         -43.5,
		High:         -43.3,
		Low:          -43.5,
		Bid:          -44.9,
		BidSize:      7.268,
		Ask:          -37.8,
		AskSize:      7.266,
		BaseVolume:   0.238,
		Open24Hour:   -43.4,
		Pair:         currency.NewPairWithDelimiter("BTCUSDT", "BTC/USDT", "_"),
		ExchangeName: t.Name(),
		AssetType:    asset.Spread,
		LastUpdated:  time.UnixMilli(1791534255189),
	}
	assert.Equal(t, expTicker, stored, "wsProcessSpreadTicker should store the ticker under the spread asset")
	assert.Equal(t, spreadTestSymbol, stored.Pair.String(), "the stored pair should round-trip to the spread symbol")
	expFrame := []any{
		&WebsocketSpreadTickerUpdate{
			Topic:     "tickers.BTCUSDT_BTC/USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534255189)),
			Data: SpreadTicker{
				Symbol:              "BTCUSDT_BTC/USDT",
				BidPrice:            -44.9,
				BidSize:             7.268,
				AskPrice:            -37.8,
				AskSize:             7.266,
				LastPrice:           -43.5,
				HighPrice24Hour:     -43.3,
				LowPrice24Hour:      -43.5,
				PreviousPrice24Hour: -43.4,
				Volume24Hour:        0.238,
			},
		},
	}
	assert.Equal(t, expFrame, relayedFrames(ex), "wsProcessSpreadTicker should relay the stored ticker's frame")

	// Live spread symbols join two legs with an underscore, and a leg can hold a dash or a slash
	shapes := []string{"SOLUSDT_SOL/USDT", "BTCUSDT-25DEC26_BTC/USDT", "BTCUSDT-25DEC26_BTCUSDT", "BTCUSDT-26MAR27_BTCUSDT-25DEC26"}
	for _, symbol := range shapes {
		frame := `{"topic":"tickers.` + symbol + `","ts":1791534255189,"type":"snapshot","data":{"symbol":"` + symbol + `","lastPrice":"-1.5"}}`
		require.NoErrorf(t, ex.wsProcessSpreadTicker(t.Context(), []byte(frame)), "wsProcessSpreadTicker must not error for %s", symbol)
		p, err := currency.NewPairFromString(symbol)
		require.NoErrorf(t, err, "NewPairFromString must not error for %s", symbol)
		stored, err := ex.GetCachedTicker(p, asset.Spread)
		require.NoErrorf(t, err, "GetCachedTicker must find the ticker of %s", symbol)
		assert.Equalf(t, symbol, stored.Pair.String(), "the stored pair should round-trip to %s", symbol)
	}
	assert.Len(t, relayedFrames(ex), len(shapes), "wsProcessSpreadTicker should relay each stored ticker's frame")

	for _, tc := range []struct {
		name   string
		frame  string
		symbol string
		err    error
	}{
		{name: "a frame that does not decode", frame: `{"topic":"tickers.ETHUSDT_ETH/USDT","data":"bad"}`, symbol: "ETHUSDT_ETH/USDT"},
		{name: "a ticker without a symbol", frame: `{"topic":"tickers.","ts":1791534255189,"type":"snapshot","data":{"bidPrice":"-44.9"}}`, err: currency.ErrCreatingPair},
		{name: "a symbol too short to pair", frame: `{"topic":"tickers.BT","ts":1791534255189,"type":"snapshot","data":{"symbol":"BT"}}`, err: currency.ErrCreatingPair},
		{name: "a bid equal to the ask", frame: `{"topic":"tickers.ETHUSDT_ETH/USDT","ts":1791534255189,"type":"snapshot","data":{"symbol":"ETHUSDT_ETH/USDT","bidPrice":"-1.2","askPrice":"-1.2"}}`, symbol: "ETHUSDT_ETH/USDT", err: ticker.ErrBidEqualsAsk},
	} {
		err := ex.wsProcessSpreadTicker(t.Context(), []byte(tc.frame))
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "wsProcessSpreadTicker should reject %s", tc.name)
		} else {
			assert.Errorf(t, err, "wsProcessSpreadTicker should reject %s", tc.name)
		}
		assert.Emptyf(t, relayedFrames(ex), "wsProcessSpreadTicker should not relay %s", tc.name)
		if tc.symbol != "" {
			p, err := currency.NewPairFromString(tc.symbol)
			require.NoError(t, err, "NewPairFromString must not error")
			_, err = ex.GetCachedTicker(p, asset.Spread)
			assert.ErrorIsf(t, err, ticker.ErrTickerNotFound, "wsProcessSpreadTicker should not store %s", tc.name)
		}
	}

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessSpreadTicker(t.Context(), frames[3]), errBufferFull, "wsProcessSpreadTicker must return the data handler's error")
}

func TestRelayFrame(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	frames := fixtureFrames(t, "testdata/wsSpreadPublic.json")
	require.Len(t, frames, 4, "the spread fixture must hold two orderbook pushes, a trade push and a ticker push")
	require.NoError(t, ex.relayFrame(t.Context(), frames[3], new(WebsocketSpreadTickerUpdate)), "relayFrame must not error")
	exp := []any{
		&WebsocketSpreadTickerUpdate{
			Topic:     "tickers.BTCUSDT_BTC/USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534255189)),
			Data: SpreadTicker{
				Symbol:              "BTCUSDT_BTC/USDT",
				BidPrice:            -44.9,
				BidSize:             7.268,
				AskPrice:            -37.8,
				AskSize:             7.266,
				LastPrice:           -43.5,
				HighPrice24Hour:     -43.3,
				LowPrice24Hour:      -43.5,
				PreviousPrice24Hour: -43.4,
				Volume24Hour:        0.238,
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "relayFrame should relay the frame decoded into the type given")

	require.Error(t, ex.relayFrame(t.Context(), []byte(`{"topic":"tickers.BTCUSDT_BTC/USDT","data":"bad"}`), new(WebsocketSpreadTickerUpdate)), "relayFrame must error for a frame that does not decode")
	assert.Empty(t, relayedFrames(ex), "relayFrame should not relay a frame that does not decode")

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.relayFrame(t.Context(), frames[3], new(WebsocketSpreadTickerUpdate)), errBufferFull, "relayFrame must return the data handler's error")
}

func TestWebsocketHandleSpreadPrivateData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		reply := livePrivateSubscribe(t, []string{ChannelSpreadOrder, ChannelSpreadExecution})
		assert.Truef(t, reply.Success, "the private stream should accept the spread topics: %s", reply.ReturnMessage)
		return
	}
	ex := testInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsSpreadPrivate.json", ex.wsHandleSpreadPrivateData)
	exp := []any{
		&WebsocketSpreadOrderUpdate{
			ID:           "1448939_BTCUSDT_28732003549",
			Topic:        "spread.order",
			CreationTime: types.Time(time.UnixMilli(1791008160130)),
			Data: []SpreadOrderUpdate{
				{
					Category:                   "combination",
					Symbol:                     "BTCUSDT_BTC/USDT",
					OrderID:                    "1b00b997-d825-465e-ad1d-80b0eb1955af",
					OrderLinkID:                "spread-gct-001",
					Side:                       "Buy",
					OrderStatus:                "PartiallyFilled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					TimeInForce:                "GTC",
					Price:                      -44.9,
					Quantity:                   0.119,
					LeavesQuantity:             0.1,
					CumulativeExecutedQuantity: 0.019,
					OrderType:                  "Limit",
					CreatedTime:                types.Time(time.UnixMilli(1791008100508)),
					UpdatedTime:                types.Time(time.UnixMilli(1791008160123)),
					CreateType:                 "CreateByUser",
					CumulativeFeeDetail: map[currency.Code]types.Number{
						currency.USDT: 3.8242308,
					},
				},
				{
					Category:                   "future_leg",
					Symbol:                     "BTCUSDT",
					ParentOrderID:              "1b00b997-d825-465e-ad1d-80b0eb1955af",
					OrderID:                    "82335b0a-b7d9-4ea5-9230-e71271a65100",
					Side:                       "Buy",
					OrderStatus:                "PartiallyFilled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					TimeInForce:                "GTC",
					Price:                      121955.1,
					Quantity:                   0.119,
					AveragePrice:               121955.1,
					LeavesQuantity:             0.1,
					LeavesValue:                12195.51,
					CumulativeExecutedQuantity: 0.019,
					CumulativeExecutedValue:    2317.1469,
					CumulativeExecutedFee:      1.2744308,
					OrderType:                  "Limit",
					IsLeverage:                 "0",
					CreatedTime:                types.Time(time.UnixMilli(1791008160120)),
					UpdatedTime:                types.Time(time.UnixMilli(1791008160124)),
					CreateType:                 "CreateByFutureSpread",
					ClosedPNL:                  -0.3402,
					CumulativeFeeDetail: map[currency.Code]types.Number{
						currency.USDT: 1.2744308,
					},
				},
				{
					Category:                   "spot_leg",
					Symbol:                     "BTCUSDT",
					ParentOrderID:              "1b00b997-d825-465e-ad1d-80b0eb1955af",
					OrderID:                    "1924011967786517249",
					Side:                       "Sell",
					OrderStatus:                "PartiallyFilled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					TimeInForce:                "GTC",
					Price:                      122000,
					Quantity:                   0.119,
					AveragePrice:               122000,
					LeavesQuantity:             0.1,
					LeavesValue:                12200,
					CumulativeExecutedQuantity: 0.019,
					CumulativeExecutedValue:    2318,
					CumulativeExecutedFee:      2.5498,
					OrderType:                  "Limit",
					IsLeverage:                 "1",
					CreatedTime:                types.Time(time.UnixMilli(1791008160120)),
					UpdatedTime:                types.Time(time.UnixMilli(1791008160124)),
					FeeCurrency:                currency.USDT,
					CreateType:                 "CreateByFutureSpread",
					CumulativeFeeDetail: map[currency.Code]types.Number{
						currency.USDT: 2.5498,
					},
				},
			},
		},
		&WebsocketSpreadExecutionUpdate{
			ID:           "cvqes8141ilt347i9l20",
			Topic:        "spread.execution",
			CreationTime: types.Time(time.UnixMilli(1791008160126)),
			Data: []SpreadExecutionUpdate{
				{
					Category:          "combination",
					Symbol:            "BTCUSDT_BTC/USDT",
					OrderID:           "1b00b997-d825-465e-ad1d-80b0eb1955af",
					OrderLinkID:       "spread-gct-001",
					Side:              "Buy",
					OrderPrice:        -44.9,
					OrderQuantity:     0.119,
					LeavesQuantity:    0.1,
					CreateType:        "CreateByUser",
					OrderType:         "Limit",
					ExecutionID:       "82c82077-0caa-5304-894d-58a50a342bd7",
					ExecutionPrice:    -44.9,
					ExecutionQuantity: 0.019,
					ExecutionType:     "Trade",
					ExecutionTime:     types.Time(time.UnixMilli(1791008160120)),
					IsMaker:           true,
					Sequence:          241321,
				},
				{
					Category:          "future_leg",
					Symbol:            "BTCUSDT",
					IsLeverage:        "0",
					Side:              "Buy",
					OrderPrice:        121955.1,
					OrderQuantity:     0.019,
					CreateType:        "CreateByFutureSpread",
					OrderType:         "Limit",
					ExecutionFee:      1.2744308,
					FeeCoin:           currency.USDT,
					ParentExecutionID: "82c82077-0caa-5304-894d-58a50a342bd7",
					ExecutionID:       "99a18f80-d3b5-4c6f-a1f1-8c5870e3f3bc",
					ExecutionPrice:    121955.1,
					ExecutionQuantity: 0.019,
					ExecutionPNL:      -0.3402,
					ExecutionType:     "FutureSpread",
					ExecutionValue:    2317.1469,
					ExecutionTime:     types.Time(time.UnixMilli(1791008160124)),
					FeeRate:           0.00055,
					MarkPrice:         121948.3,
					ClosedSize:        0.019,
					Sequence:          28731107101,
				},
				{
					Category:          "spot_leg",
					Symbol:            "BTCUSDT",
					IsLeverage:        "1",
					Side:              "Sell",
					OrderPrice:        122000,
					OrderQuantity:     0.019,
					CreateType:        "CreateByFutureSpread",
					OrderType:         "Limit",
					ExecutionFeeV2:    2.5498,
					FeeCoin:           currency.USDT,
					ParentExecutionID: "82c82077-0caa-5304-894d-58a50a342bd7",
					ExecutionID:       "2110000000061481958",
					ExecutionPrice:    122000,
					ExecutionQuantity: 0.019,
					ExecutionType:     "FutureSpread",
					ExecutionValue:    2318,
					ExecutionTime:     types.Time(time.UnixMilli(1791008160124)),
					FeeRate:           0.0011,
					MarkPrice:         121990.5,
					Sequence:          28731107102,
				},
			},
		},
	}
	assert.Equal(t, exp, relayedFrames(ex), "wsHandleSpreadPrivateData should relay every frame")

	assert.ErrorIs(t, ex.wsHandleSpreadPrivateData(t.Context(), []byte(`{"topic":"spread.position"}`)), errUnhandledStreamData, "wsHandleSpreadPrivateData should reject an unknown topic")
	assert.Error(t, ex.wsHandleSpreadPrivateData(t.Context(), []byte(`{"op":"pong"}`)), "wsHandleSpreadPrivateData should reject a frame without a topic")
	assert.Error(t, ex.wsHandleSpreadPrivateData(t.Context(), []byte(`{"topic":"spread.order","data":{}}`)), "wsHandleSpreadPrivateData should reject a malformed frame")
}

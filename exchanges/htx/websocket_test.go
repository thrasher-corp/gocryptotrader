package htx

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestStringToOrderSide(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Side
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Buy},
		{Case: "sell-limit", Result: order.Sell},
		{Case: "woah-nelly", Result: order.UnknownSide},
	}
	for i := range testCases {
		result, _ := stringToOrderSide(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input    string
		expected order.Type
	}{
		{"buy-limit", order.Limit}, {"sell-market", order.Market}, {"buy-stop-limit", order.StopLimit}, {"sell-stop-limit-fok", order.StopLimit}, {"buy-ioc", order.Limit}, {"sell-ioc", order.Limit}, {"woah-nelly", order.UnknownType},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			result, err := stringToOrderType(tc.input)
			if tc.expected == order.UnknownType {
				require.Error(t, err, "unknown type must fail")
			} else {
				require.NoError(t, err, "documented type must decode")
			}
			assert.Equal(t, tc.expected, result, "order type should match")
		})
	}
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	tmpl, err := h.GetSubscriptionTemplate(&subscription.Subscription{})
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	assert.NotNil(t, tmpl, "subscription template should be returned")
}

func wsFixture(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	operation, _ := jsonparser.GetString(msg, "op")
	topic, _ := jsonparser.GetString(msg, "topic")
	if operation == wsAuthChannel {
		return w.WriteMessage(gws.TextMessage, []byte(`{"op":"auth","type":"api","ts":1489474081631,"err-code":0,"data":{"user-id":"12345678"}}`))
	}
	if operation == wsSubOp || operation == wsUnsubOp {
		return w.WriteMessage(gws.TextMessage, []byte(`{"op":"`+operation+`","topic":"`+topic+`","ts":1489474081631,"err-code":0}`))
	}
	if operation == "pong" {
		return nil
	}
	cid, _ := jsonparser.GetString(msg, "cid")
	switch operation {
	case "place_order", "cancel_order":
		return w.WriteMessage(gws.TextMessage, []byte(`{"code":200,"message":"Success","cid":"`+cid+`","data":{"order_id":"1","client_order_id":"2"},"rate_limit":{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}}`))
	case "place_batch_orders", "cancel_batch_orders", "cancel_all_orders":
		return w.WriteMessage(gws.TextMessage, []byte(`{"code":200,"message":"Success","cid":"`+cid+`","data":[{"order_id":"1","client_order_id":"2"}],"rate_limit":{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}}`))
	}
	action, _ := jsonparser.GetString(msg, "action")
	ch, _ := jsonparser.GetString(msg, "ch")
	if action == "req" && ch == "auth" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"req","code":200,"ch":"auth","data":{}}`))
	}
	if action == "sub" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"sub","code":200,"ch":"`+ch+`"}`))
	}
	id, _ := jsonparser.GetString(msg, "id")
	sub, _ := jsonparser.GetString(msg, "sub")
	if id != "" && sub != "" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"ok","subbed":"`+sub+`"}`))
	}
	unsub, _ := jsonparser.GetString(msg, "unsub")
	if id != "" && unsub != "" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"ok","unsubbed":"`+unsub+`"}`))
	}
	return fmt.Errorf("%w: %s", errUnhandledMockWebsocketMessage, msg)
}

func TestWSHandleData(t *testing.T) {
	t.Parallel()
	t.Run("futures private notification", func(t *testing.T) {
		t.Parallel()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		conn, err := h.Websocket.GetConnection(exchange.WebsocketCoinMarginedPrivate)
		require.NoError(t, err, "coin-margined private websocket connection must be available")
		sub := &subscription.Subscription{
			Key:              "positions.*",
			Asset:            asset.CoinMarginedFutures,
			Channel:          wsPositionsChannel,
			Authenticated:    true,
			QualifiedChannel: "positions.*",
		}
		sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
		require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "wildcard subscription must be registered")
		raw := []byte(`{"op":"notify","topic":"positions.BTC-USD","ts":1603878749908,"event":"order.match","data":[]}`)
		require.NoError(t, h.wsHandleData(t.Context(), conn, raw), "wsHandleData must route a concrete private derivative topic")
		require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
		message := <-h.Websocket.DataHandler.C
		assert.IsType(t, &SwapWsSubPositionUpdates{}, message.Data, "notification should be decoded using its subscription")
	})
	t.Run("futures subscription response", func(t *testing.T) {
		t.Parallel()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
		require.NoError(t, err, "delivery private websocket connection must be available")
		err = h.wsHandleData(t.Context(), conn, []byte(`{"op":"sub"}`))
		require.ErrorIs(t, err, common.ErrParsingWSField, "derivative acknowledgement must require a topic")
	})
	t.Run("spot subscription response", func(t *testing.T) {
		t.Parallel()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
		require.NoError(t, err, "private spot websocket connection must be available")
		require.ErrorIs(t, h.wsHandleData(t.Context(), conn, []byte(`{"action":"sub"}`)), common.ErrParsingWSField, "subscription response must require a channel")
	})
	t.Run("early setup lookup", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		h.Name = t.Name()
		conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
		require.NoError(t, err, "spot test connection must be created")
		require.NoError(t, h.Websocket.TrackTestConnection(exchange.WebsocketSpot, conn), "connection must belong to setup")
		sub := &subscription.Subscription{Key: "market.btcusdt.detail", QualifiedChannel: "market.btcusdt.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.TickerChannel}
		sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
		require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "pending subscription must enter setup store")
		assert.Empty(t, conn.Subscriptions().List(), "local store should still await batch completion")
		require.NoError(t, h.wsHandleData(t.Context(), conn, []byte(`{"ch":"market.btcusdt.detail","tick":{"close":100}}`)), "early ticker must resolve acknowledged subscription")
		stored, err := ticker.GetTicker(h.Name, btcusdtPair, asset.Spot)
		require.NoError(t, err, "early ticker must reach store")
		assert.Equal(t, 100.0, stored.Last)
	})

	t.Run("connection matcher isolation", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
		require.NoError(t, err, "test connection must be created")
		local, err := conn.MatchReturnResponses(t.Context(), "same-id", 1)
		require.NoError(t, err, "connection matcher must register")
		global, err := h.Websocket.Match.Set("same-id", 1)
		require.NoError(t, err, "manager matcher must register")
		defer h.Websocket.Match.RemoveSignature("same-id")
		raw := []byte(`{"id":"same-id","status":"ok"}`)
		require.NoError(t, h.wsHandleData(t.Context(), conn, raw), "response must route to its connection")
		select {
		case response := <-local:
			require.NoError(t, response.Err, "connection matcher must complete")
			require.Len(t, response.Responses, 1, "one connection response must arrive")
			assert.Equal(t, raw, response.Responses[0])
		case <-time.After(5 * time.Second):
			require.FailNow(t, "connection response missing")
		}
		assert.Empty(t, global, "connection response should leave manager matcher untouched")
		require.NoError(t, h.wsHandleData(t.Context(), nil, raw), "legacy response must route to manager")
		select {
		case data := <-global:
			assert.Equal(t, raw, data)
		default:
			require.FailNow(t, "manager response missing")
		}
	})

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.wsHandleData(t.Context(), nil, []byte(`{"unexpected":true}`)), "unmatched websocket data must be forwarded")
	require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
	message := <-h.Websocket.DataHandler.C
	assert.IsType(t, websocket.UnhandledMessageWarning{}, message.Data, "unmatched websocket data should produce a warning")
	require.ErrorIs(t, h.wsHandleData(t.Context(), nil, []byte(`{"ping":123}`)), common.ErrNilPointer, "websocket ping data must require a connection")
}

func TestWSHandleChannelMsgs(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.CandlesChannel,
		Pairs:            currency.Pairs{currency.NewBTCUSDT()},
		Interval:         kline.OneMin,
		QualifiedChannel: "market.btcusdt.kline.1min",
	}
	testexch.FixtureToDataHandler(t, "testdata/wsCandles.json", func(ctx context.Context, data []byte) error {
		return h.wsHandleChannelMsgs(ctx, sub, data)
	})
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	t.Run("expired delivery does not block later flush", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
		server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
			tb.Helper()
			unsub, _ := jsonparser.GetString(message, "unsub")
			assert.NotEqual(tb, "market.BTC200101.detail", unsub, "expired topics should be removed locally")
			return wsFixture(tb, message, conn)
		}))
		t.Cleanup(server.Close)
		t.Cleanup(func() {
			if err := h.Websocket.Disable(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "manager should disable cleanly")
			}
			if err := h.Websocket.Shutdown(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected, "test sockets should shut down cleanly")
			}
		})
		futurePair := currency.NewPair(currency.BTC, currency.NewCode(time.Now().UTC().AddDate(0, 0, 7).Format("060102")))
		h.API.AuthenticatedWebsocketSupport = false
		h.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{futurePair}},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{btcusdtPair}},
		}
		require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "mock websocket URLs must be set")
		require.NoError(t, h.Websocket.Connect(t.Context()), "test connections must open")
		conn, err := h.Websocket.GetConnection(exchange.WebsocketFutures)
		require.NoError(t, err, "delivery connection must exist")
		expired := &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel, QualifiedChannel: "market.BTC200101.detail", Pairs: currency.Pairs{currency.NewPair(currency.BTC, currency.NewCode("200101"))}}
		require.NoError(t, h.manageSubs(t.Context(), conn, wsSubOp, subscription.List{expired}), "subscription from before delivery must be seeded")
		require.NoError(t, conn.Subscriptions().Add(expired), "connection ownership from before delivery must be seeded")
		ethusdt := currency.NewPair(currency.ETH, currency.USDT)
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair, ethusdt}, asset.USDTMarginedFutures, false), "USDT pairs must be available")
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair, ethusdt}, asset.USDTMarginedFutures, true), "USDT pairs must be enabled")
		h.Features.Subscriptions = subscription.List{h.Features.Subscriptions[0], expired, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{ethusdt}}}
		require.NoError(t, h.Websocket.FlushChannels(t.Context()), "expired delivery must not block later connection changes")
		assert.Nil(t, h.Websocket.GetSubscription(expired.EnsureKeyed()), "expired subscription should no longer be registered")
		usdtConn, err := h.Websocket.GetConnection(exchange.WebsocketUSDTMargined)
		require.NoError(t, err, "USDT connection must remain usable")
		assert.NotNil(t, usdtConn, "USDT connection should remain connected")
		var later *subscription.Subscription
		for _, sub := range h.Websocket.GetSubscriptions() {
			if sub.Asset == asset.USDTMarginedFutures && len(sub.Pairs) == 1 && sub.Pairs[0].Equal(ethusdt) {
				later = sub
			}
		}
		require.NotNil(t, later, "later USDT subscription must be registered")
		assert.Equal(t, subscription.SubscribedState, later.State(), "later USDT subscription should be active")
	})
	t.Run("expired delivery unregisters without sending", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
		conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketFutures)
		require.NoError(t, err, "delivery test connection must be created")
		require.NoError(t, h.Websocket.TrackTestConnection(exchange.WebsocketFutures, conn), "delivery connection must be tracked")
		sub := &subscription.Subscription{Asset: asset.Futures, Channel: subscription.TickerChannel, QualifiedChannel: "market.BTC200101.detail", Pairs: currency.Pairs{currency.NewPair(currency.BTC, currency.NewCode("200101"))}}
		require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "expired registration must be seeded")
		require.NoError(t, sub.SetState(subscription.SubscribedState), "seeded subscription must be active")
		require.NoError(t, h.manageSubs(t.Context(), conn, wsUnsubOp, subscription.List{sub}), "expired topic must be removed without using the disconnected socket")
		assert.Empty(t, h.Websocket.GetSubscriptions(), "expired registration should be removed")
	})
	t.Run("failed subscription removes registration", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
		require.NoError(t, err, "spot connection must be created")
		require.NoError(t, h.Websocket.TrackTestConnection(exchange.WebsocketSpot, conn), "spot connection must be tracked")
		sub := &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			QualifiedChannel: "market.btcusdt.detail",
			Pairs:            currency.Pairs{btcusdtPair},
		}
		require.ErrorContains(t, h.manageSubs(t.Context(), conn, wsSubOp, subscription.List{sub}), "websocket connection is disconnected", "subscription must fail on a disconnected socket")
		assert.Empty(t, h.Websocket.GetSubscriptions(), "failed subscription should leave no registration")
	})
	for _, tc := range []struct {
		name         string
		channel      string
		topic        string
		contractCode string
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, topic: "orders", contractCode: "*"},
		{name: "account", channel: subscription.MyAccountChannel, topic: "account"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				operation, err := jsonparser.GetString(msg, "op")
				require.NoError(t, err, "subscription operation must decode")
				topic, err := jsonparser.GetString(msg, "topic")
				require.NoError(t, err, "subscription topic must decode")
				contractCode, _ := jsonparser.GetString(msg, "contract_code")
				assert.Equal(t, tc.topic, topic, "V5 subscription topic should match")
				assert.Equal(t, tc.contractCode, contractCode, "V5 subscription contract code should match")
				return w.WriteMessage(gws.TextMessage, []byte(`{"op":"`+operation+`","topic":"`+topic+`","ts":1489474081631,"err-code":0}`))
			}))
			conn, err := h.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
			require.NoError(t, err, "private USDT-margined websocket connection must be available")
			sub := &subscription.Subscription{
				Asset:            asset.USDTMarginedFutures,
				Channel:          tc.channel,
				Authenticated:    true,
				QualifiedChannel: tc.topic,
			}
			require.NoError(t, h.manageSubs(t.Context(), conn, wsSubOp, subscription.List{sub}), "V5 subscription must not error")
			assert.Equal(t, subscription.SubscribedState, sub.State(), "V5 subscription should be active")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.ErrorIs(t, h.manageSubs(t.Context(), nil, wsSubOp, nil), subscription.ErrBatchingNotSupported, "empty subscription batches must be rejected")
}

// TestSubscribe exercises public subscriptions against the deterministic websocket fixture.
func TestSubscribe(t *testing.T) {
	t.Parallel()
	t.Run("authenticated derivatives", func(t *testing.T) {
		t.Parallel()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
		for _, sub := range h.Features.Subscriptions {
			if sub.Authenticated {
				sub.Enabled = true
			}
		}
		for _, tt := range []struct {
			a     asset.Item
			pairs currency.Pairs
		}{
			{a: asset.Futures, pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "CW", "_")}},
			{a: asset.CoinMarginedFutures, pairs: currency.Pairs{btcusdPair}},
			{a: asset.USDTMarginedFutures, pairs: currency.Pairs{btcusdtPair}},
		} {
			require.NoErrorf(t, h.SetPairs(tt.pairs, tt.a, false), "%s available pairs must be set", tt.a)
			require.NoErrorf(t, h.SetPairs(tt.pairs, tt.a, true), "%s enabled pairs must be set", tt.a)
		}
		var all subscription.List
		for _, tt := range []struct {
			a        asset.Item
			endpoint exchange.URL
			count    int
		}{
			{a: asset.Futures, endpoint: exchange.WebsocketFuturesPrivate, count: 5},
			{a: asset.CoinMarginedFutures, endpoint: exchange.WebsocketCoinMarginedPrivate, count: 5},
			{a: asset.USDTMarginedFutures, endpoint: exchange.WebsocketUSDTMarginedPrivate, count: 7},
		} {
			subs, err := h.generateSubscriptionsForAsset(tt.a, true)
			require.NoErrorf(t, err, "%s private subscriptions must be generated", tt.a)
			require.Lenf(t, subs, tt.count, "%s must include every documented private topic", tt.a)
			all = append(all, subs...)
		}
		require.NoError(t, h.Subscribe(all), "all private derivative subscriptions must succeed")
		for _, tt := range []struct {
			endpoint exchange.URL
			asset    asset.Item
			count    int
		}{
			{endpoint: exchange.WebsocketFuturesPrivate, asset: asset.Futures, count: 5},
			{endpoint: exchange.WebsocketCoinMarginedPrivate, asset: asset.CoinMarginedFutures, count: 5},
			{endpoint: exchange.WebsocketUSDTMarginedPrivate, asset: asset.USDTMarginedFutures, count: 7},
		} {
			_, err := h.Websocket.GetConnection(tt.endpoint)
			require.NoError(t, err, "private derivative websocket connection must be available")
			count := 0
			for _, sub := range h.Websocket.GetSubscriptions() {
				if sub.Asset == tt.asset && sub.Authenticated {
					count++
				}
			}
			assert.Equal(t, tt.count, count, "manager should contain every authenticated subscription")
		}
	})
	t.Run("authenticated spot", func(t *testing.T) {
		t.Parallel()
		subCfg := defaultSubscriptions.Clone()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		subs, err := subCfg.ExpandTemplates(h)
		require.NoError(t, err, "ExpandTemplates must not error")
		err = h.Subscribe(subs)
		require.NoError(t, err, "Subscribe must not error")
		_, err = h.Websocket.GetConnection(exchange.WebsocketSpot)
		require.NoError(t, err, "spot public websocket connection must be available")
		_, err = h.Websocket.GetConnection(exchange.WebsocketPrivate)
		require.NoError(t, err, "spot private websocket connection must be available")
		got := h.Websocket.GetSubscriptions()
		require.Equal(t, 11, len(got), "Must get correct number of subscriptions")
		for _, s := range got {
			assert.Equal(t, subscription.SubscribedState, s.State())
		}
	})
	e := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	subs, err := defaultSubscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	err = e.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	_, err = e.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	got := e.Websocket.GetSubscriptions()
	require.Equal(t, 8, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	subs, err := defaultSubscriptions.ExpandTemplates(h)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, h.Subscribe(subs), "Subscribe must not error")
	require.NoError(t, h.Unsubscribe(subs), "Unsubscribe must not error")
	for _, endpoint := range []exchange.URL{exchange.WebsocketSpot, exchange.WebsocketPrivate} {
		conn, err := h.Websocket.GetConnection(endpoint)
		require.NoErrorf(t, err, "%s connection must be available", endpoint)
		assert.Emptyf(t, conn.Subscriptions().List(), "%s subscriptions should be empty", endpoint)
	}
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "market.BTC-USD.kline", channelName(&subscription.Subscription{Channel: subscription.CandlesChannel}, btcusdPair))
	assert.Equal(t, "public.BTC-USD.funding_rate", channelName(&subscription.Subscription{Channel: wsFundingRateChannel}, btcusdPair), "funding-rate channel should expand to the HTX topic")
	assert.Equal(t, "public.BTC-USD.funding_rate", channelName(&subscription.Subscription{Channel: wsFundingRateTopic}, btcusdPair), "saved funding-rate subscriptions should still expand")
	assert.Equal(t, "trade.clearing#*#1", channelName(&subscription.Subscription{Channel: subscription.MyTradesChannel}, btcusdPair))
	for _, tt := range []struct {
		channel string
		want    string
	}{
		{channel: subscription.MyOrdersChannel, want: "orders"},
		{channel: wsTradeUpdatesChannel, want: "trade"},
		{channel: wsExecutionDetailsChannel, want: "trade_detail"},
		{channel: wsPositionsChannel, want: "positions"},
		{channel: subscription.MyAccountChannel, want: "account"},
		{channel: subscription.MyTradesChannel, want: "match_orders"},
		{channel: wsTriggerOrdersChannel, want: "algo_orders"},
	} {
		t.Run(tt.channel, func(t *testing.T) {
			t.Parallel()
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			assert.Equal(t, tt.want, channelName(sub), "private USDT-margined channel should use the documented V5 topic")
		})
	}
	assert.Equal(t, "orders.*", channelName(&subscription.Subscription{
		Asset:         asset.CoinMarginedFutures,
		Channel:       subscription.MyOrdersChannel,
		Authenticated: true,
	}), "coin-margined private channels should retain the documented wildcard topic")
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}, btcusdPair) })
}

func TestIsWildcardChannel(t *testing.T) {
	t.Parallel()
	assert.False(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.CandlesChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.MyAccountChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: wsPositionsChannel, Authenticated: true}))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}) })
}

func TestGetErrResp(t *testing.T) {
	t.Parallel()
	err := getErrResp([]byte(`{"status":"error","err-code":"bad-request","err-msg":"invalid topic promiscuous.drop🐻s.nearby"}`))
	assert.ErrorIs(t, err, errAPIResponse, "V1 API errors should wrap the API sentinel")
	assert.ErrorContains(t, err, "invalid topic promiscuous.drop🐻s.nearby (bad-request)", "V1 errors should return correctly")
	err = getErrResp([]byte(`{"status":"ok","subbed":"market.btcusdt.trade.detail"}`))
	assert.NoError(t, err, "V1 success should not error")

	err = getErrResp([]byte(`{"action":"sub","code":2001,"ch":"naughty.drop🐻s.locally","message":"invalid.ch"}`))
	assert.ErrorIs(t, err, errAPIResponse, "V2 API errors should wrap the API sentinel")
	assert.ErrorContains(t, err, "invalid.ch (2001)", "V2 errors should return correctly")

	err = getErrResp([]byte(`{"action":"sub","code":200,"ch":"orders#btcusdt","data":{}}`))
	assert.NoError(t, err, "V2 success should not error")

	err = getErrResp([]byte(`{"op":"auth","err-code":0}`))
	assert.NoError(t, err, "derivative websocket success should not error")
	err = getErrResp([]byte(`{"op":"auth","err-code":2001,"err-msg":"invalid authentication"}`))
	assert.ErrorIs(t, err, errAPIResponse, "derivative API errors should wrap the API sentinel")
	assert.ErrorContains(t, err, "invalid authentication (2001)", "derivative websocket errors should include their numeric code")
}

func TestWSSubscriptionKeyMatch(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
	conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "first connection must be created")
	other, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "second connection must be created")
	sub := &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: "market.btcusdt.detail", Pairs: currency.Pairs{btcusdtPair}}
	key := wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn}
	for _, tc := range []struct {
		name  string
		query wsSubscriptionKey
		other subscription.MatchableKey
		want  bool
	}{
		{name: "same connection and topic", query: key, other: key, want: true},
		{name: "different connection", query: key, other: wsSubscriptionKey{ExactKey: key.ExactKey, connection: other}},
		{name: "nil connection cannot select managed topic", query: wsSubscriptionKey{ExactKey: key.ExactKey}, other: key},
		{name: "different topic", query: key, other: wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "orders.*"}}, connection: conn}},
		{name: "nil query key", query: wsSubscriptionKey{}, other: key},
		{name: "nil query subscription", query: wsSubscriptionKey{ExactKey: &subscription.ExactKey{}}, other: key},
		{name: "nil candidate key", query: key, other: wsSubscriptionKey{}},
		{name: "nil candidate subscription", query: key, other: wsSubscriptionKey{ExactKey: &subscription.ExactKey{}}},
		{name: "ordinary key", query: key, other: key.ExactKey},
		{name: "nil candidate", query: key},
		{name: "typed nil candidate", query: key, other: (*wsSubscriptionKey)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.query.Match(tc.other), "topic lookup should remain scoped to its connection")
		})
	}
	t.Run("regenerated settings retain registration", func(t *testing.T) {
		t.Parallel()
		store := subscription.NewStore()
		registered := sub.Clone()
		registered.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: registered}, connection: conn})
		require.NoError(t, store.Add(registered), "scoped subscription must register")
		fresh := registered.Clone()
		fresh.SetKey(nil)
		adds, removes := store.Diff(subscription.List{fresh})
		assert.Empty(t, adds, "equivalent regenerated settings should not subscribe again")
		assert.Empty(t, removes, "equivalent regenerated settings should not unsubscribe")
		fresh.Levels++
		adds, removes = store.Diff(subscription.List{fresh})
		assert.Len(t, adds, 1, "changed settings should require a new subscription")
		assert.Len(t, removes, 1, "changed settings should retire the previous subscription")
	})
}

func TestWSHandleSubscriptionResponse(t *testing.T) {
	t.Parallel()
	t.Run("spot nil connection", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, new(Exchange).wsHandleSubscriptionResponse(nil, wsSubOp, "orders", []byte(`{}`)), common.ErrNilPointer, "subscription response must require a connection")
	})
	t.Run("futures nil connection", func(t *testing.T) {
		t.Parallel()
		err := new(Exchange).wsHandleSubscriptionResponse(nil, wsSubOp, "orders.*", []byte(`{"op":"sub","topic":"orders.*"}`))
		require.ErrorIs(t, err, common.ErrNilPointer, "derivative acknowledgement must reject nil connections")
	})
	for _, protocol := range []string{"public", "spot private", "derivative"} {
		for _, operation := range []string{wsSubOp, wsUnsubOp} {
			for _, rejected := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/rejected=%t", protocol, operation, rejected), func(t *testing.T) {
					t.Parallel()
					h := new(Exchange)
					require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
					h.Name = t.Name()
					conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
					require.NoError(t, err, "connection must be created")
					require.NoError(t, h.Websocket.TrackTestConnection(exchange.WebsocketSpot, conn), "connection must be tracked")
					channel := "market.btcusdt.detail"
					sub := &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: channel, Pairs: currency.Pairs{btcusdtPair}}
					sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
					require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "pending subscription must register")
					if operation == wsUnsubOp {
						require.NoError(t, sub.SetState(subscription.SubscribedState), "unsubscribe must start with an accepted subscription")
					}
					responses, err := conn.MatchReturnResponses(t.Context(), operation+":"+channel, 1)
					require.NoError(t, err, "acknowledgement waiter must register")
					status, code := "ok", 200
					if rejected {
						status, code = "error", 1000
					}
					var raw string
					switch protocol {
					case "public":
						raw = fmt.Sprintf(`{"id":%q,"status":%q,"code":%d}`, operation+":"+channel, status, code)
					case "spot private":
						raw = fmt.Sprintf(`{"action":%q,"ch":%q,"code":%d}`, operation, channel, code)
					case "derivative":
						raw = fmt.Sprintf(`{"op":%q,"topic":%q,"code":%d}`, operation, channel, code)
					}
					// Process the next push without waiting for the request goroutine or batch to finish.
					require.NoError(t, h.wsHandleData(t.Context(), conn, []byte(raw)), "acknowledgement must update registration before returning")
					retained := operation == wsSubOp && !rejected || operation == wsUnsubOp && rejected
					if retained {
						assert.Equal(t, subscription.SubscribedState, sub.State(), "accepted subscription should already be active")
					} else {
						assert.Equal(t, subscription.UnsubscribedState, sub.State(), "removed subscription should already be inactive")
					}
					assert.Empty(t, conn.Subscriptions().List(), "routing should not depend on batch connection bookkeeping")
					err = h.wsHandleData(t.Context(), conn, []byte(`{"ch":"market.btcusdt.detail","tick":{"close":100}}`))
					if retained {
						require.NoError(t, err, "immediate push must resolve its accepted subscription")
						stored, err := ticker.GetTicker(h.Name, btcusdtPair, asset.Spot)
						require.NoError(t, err, "immediate ticker must reach the shared store")
						assert.Equal(t, 100.0, stored.Last, "immediate push should retain its price")
					} else {
						require.ErrorIs(t, err, subscription.ErrNotFound, "push for removed subscription must not route")
					}
					select {
					case response := <-responses:
						require.NoError(t, response.Err, "acknowledgement must reach the request waiter")
						assert.Equal(t, []byte(raw), response.Responses[0], "request waiter should receive the original reply")
					case <-time.After(time.Second):
						require.FailNow(t, "request waiter must receive its acknowledgement")
					}
				})
			}
		}
	}
	require.ErrorIs(t, new(Exchange).wsHandleSubscriptionResponse(nil, wsSubOp, "orders.*", nil), common.ErrNilPointer, "acknowledgement must require a connection")
	var typedNil *struct{ websocket.Connection }
	require.ErrorIs(t, new(Exchange).wsHandleSubscriptionResponse(typedNil, wsSubOp, "orders.*", nil), common.ErrNilPointer, "acknowledgement must reject a typed nil connection")
}

func TestWSConnect(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.SetEnabled(false)
	err := h.wsConnect(t.Context(), nil)
	require.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled, "wsConnect must reject a disabled exchange")
}

func TestGenerateSubscriptionsForAsset(t *testing.T) {
	t.Parallel()
	for _, scoped := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired deliveries scoped=%t", scoped), func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			futurePair := currency.NewPair(currency.BTC, currency.NewCode(time.Now().UTC().AddDate(0, 0, 7).Format("060102")))
			pairs := currency.Pairs{currency.NewPair(currency.BTC, currency.NewCode("200101")), futurePair}
			require.NoError(t, h.SetPairs(pairs, asset.Futures, false), "delivery pairs must be available")
			require.NoError(t, h.SetPairs(pairs, asset.Futures, true), "delivery pairs must be enabled")
			sub := &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel}
			if scoped {
				sub.Pairs = pairs
			}
			h.Features.Subscriptions = subscription.List{sub}
			subs, err := h.generateSubscriptionsForAsset(asset.Futures, false)
			require.NoError(t, err, "delivery subscriptions must generate")
			require.Len(t, subs, 1, "only the undelivered contract must be subscribed")
			require.Len(t, subs[0].Pairs, 1, "one future pair must remain")
			assert.True(t, subs[0].Pairs[0].Equal(futurePair), "future contract identity should remain intact")
		})
	}
	t.Run("spot", func(t *testing.T) {
		t.Parallel()

		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
		e.Features.Subscriptions = defaultSubscriptions.Clone()

		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
		subs, err := e.generateSubscriptionsForAsset(asset.Spot, false)
		require.NoError(t, err, "public generateSubscriptionsForAsset must not error")
		privateSubs, err := e.generateSubscriptionsForAsset(asset.Spot, true)
		require.NoError(t, err, "private generateSubscriptionsForAsset must not error")
		subs = append(subs, privateSubs...)
		exp := subscription.List{}
		for _, s := range e.Features.Subscriptions {
			if s.Asset == asset.Empty {
				s := s.Clone() //nolint:govet // Intentional lexical scope shadow
				s.QualifiedChannel = channelName(s)
				exp = append(exp, s)
				continue
			}
			for _, a := range e.GetAssetTypes(true) {
				if s.Asset != asset.All && s.Asset != a {
					continue
				}
				pairs, err := e.GetEnabledPairs(a)
				require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
				pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: false, Delimiter: ""})
				s := s.Clone() //nolint:govet // Intentional lexical scope shadow
				s.Asset = a
				if isWildcardChannel(s) {
					s.Pairs = pairs
					s.QualifiedChannel = channelName(s)
					exp = append(exp, s)
					continue
				}
				for i, p := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					s.QualifiedChannel = channelName(s, p)
					switch s.Channel {
					case subscription.OrderbookChannel:
						s.QualifiedChannel += ".step0"
					case subscription.CandlesChannel:
						s.QualifiedChannel += ".1min"
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			}
		}
		testsubs.EqualLists(t, exp, subs)
	})
	for _, monitorShutdown := range []bool{false, true} {
		t.Run(fmt.Sprintf("valid spot survives unsupported funding connection/monitorShutdown=%t", monitorShutdown), func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h))
			h.API.AuthenticatedWebsocketSupport = false
			h.Features.Subscriptions = subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{btcusdtPair}},
				{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: wsFundingRateChannel, Pairs: currency.Pairs{btcusdPair}},
			}
			server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsFixture))
			t.Cleanup(server.Close)
			t.Cleanup(func() {
				if err := h.Websocket.Disable(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "Disable should not error")
				}
				if err := h.Websocket.Shutdown(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should not error")
				}
			})
			require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")))
			err := h.Websocket.Connect(t.Context())
			require.ErrorIs(t, err, websocket.ErrSubscriptionPartial)
			require.ErrorIs(t, err, common.ErrNotYetImplemented)
			conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
			require.NoError(t, err, "valid spot connection must survive")
			assert.NotEmpty(t, conn.Subscriptions().List())
			if monitorShutdown {
				require.NoError(t, h.Websocket.Disable(), "manager must stop reconnecting")
				require.Eventually(t, func() bool { return !h.Websocket.IsConnected() }, 5*time.Second, 10*time.Millisecond, "monitor must shut down the disabled manager before cleanup")
			}
		})
	}

	for _, channel := range []string{wsFundingRateChannel, wsFundingRateTopic} {
		t.Run("unsupported "+channel, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
			h.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: channel}}
			_, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, false)
			require.ErrorIs(t, err, common.ErrNotYetImplemented, "unsupported funding must report its contract error")
			require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "unsupported funding must leave other subscriptions usable")
			h.Features.Subscriptions[0].Enabled = false
			subs, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, false)
			require.NoError(t, err, "disabled legacy funding subscription must be preserved safely")
			assert.Empty(t, subs)
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdPair}, asset.CoinMarginedFutures, true), "enabled pairs must be set")

	subs, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, false)
	require.NoError(t, err, "generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 4, "coin-margined subscriptions must include each public market channel")
	for _, sub := range subs {
		assert.Equal(t, asset.CoinMarginedFutures, sub.Asset, "subscription asset should match")
		assert.False(t, sub.Authenticated, "public futures subscriptions should not require authentication")
		assert.NotEmpty(t, sub.QualifiedChannel, "subscription channel should be expanded")
	}

	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	for _, sub := range h.Features.Subscriptions {
		if sub.Authenticated {
			sub.Enabled = true
		}
	}
	privateSubs, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, true)
	require.NoError(t, err, "private generateSubscriptionsForAsset must not error")
	require.Len(t, privateSubs, 5, "coin-margined subscriptions must include each private notification channel")
	for _, sub := range privateSubs {
		assert.True(t, sub.Authenticated, "private futures subscriptions should require authentication")
		assert.Contains(t, sub.QualifiedChannel, "*", "private notification channels should use one wildcard subscription")
	}

	h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
	for _, sub := range h.Features.Subscriptions {
		if sub.Authenticated {
			sub.Enabled = true
		}
	}
	deliveryDate := time.Now().UTC().AddDate(0, 0, 7).Format("060102")
	h.futureContractCodes = map[string]currency.Code{"CW": currency.NewCode(deliveryDate)}
	deliveryPairs := currency.Pairs{currency.NewPairWithDelimiter("BTC", deliveryDate, "-")}
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, false), "available delivery pairs must be set")
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, true), "enabled delivery pairs must be set")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "delivery generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 4, "delivery subscriptions must include each public market channel")
	for _, sub := range subs {
		assert.Contains(t, sub.QualifiedChannel, "BTC"+deliveryDate, "dated delivery channel should use its immutable contract code")
		require.Len(t, sub.Pairs, 1, "delivery subscription must retain one canonical pair")
		assert.True(t, sub.Pairs[0].Equal(deliveryPairs[0]), "delivery subscription should retain the configured canonical pair")
	}
	h.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel, Pairs: deliveryPairs}}
	multipleDeliveryPairs := currency.Pairs{deliveryPairs[0], currency.NewPairWithDelimiter("ETH", deliveryDate, "-")}
	require.NoError(t, h.SetPairs(multipleDeliveryPairs, asset.Futures, false), "multiple delivery pairs must be available")
	require.NoError(t, h.SetPairs(multipleDeliveryPairs, asset.Futures, true), "multiple delivery pairs must be enabled")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "scoped delivery subscription generation must not error")
	require.Len(t, subs, 1, "scoped delivery subscription must not expand to every enabled contract")
	assert.Equal(t, deliveryPairs, subs[0].Pairs, "scoped delivery subscription should preserve its configured pair")
	beforeRollover := subs[0].QualifiedChannel
	h.futureContractCodes = map[string]currency.Code{"CW": currency.NewCode("250905")}
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "generation after rollover must succeed")
	require.Len(t, subs, 1)
	assert.Equal(t, beforeRollover, subs[0].QualifiedChannel, "dated topic should remain stable across rollover and reconnect generation")
	aliasPair := currency.NewPairWithDelimiter("BTC", "CW", "_")
	require.NoError(t, h.SetPairs(currency.Pairs{multipleDeliveryPairs[0], multipleDeliveryPairs[1], aliasPair}, asset.Futures, false))
	require.NoError(t, h.SetPairs(currency.Pairs{aliasPair}, asset.Futures, true))
	h.Features.Subscriptions[0].Pairs = currency.Pairs{aliasPair}
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Contains(t, subs[0].QualifiedChannel, "BTC_CW", "alias pairs should retain their moving venue aliases")
	assert.Equal(t, currency.Pairs{aliasPair}, subs[0].Pairs)

	require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "delivery futures must be disabled")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "disabled delivery subscription generation must not error")
	assert.Empty(t, subs, "disabled delivery futures should not generate subscriptions")
}

func TestIsExpiredDeliveryContract(t *testing.T) {
	t.Parallel()
	delivery := time.Date(2026, time.October, 16, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, quote string
		now         time.Time
		expired     bool
	}{
		{name: "before delivery", quote: "261016", now: delivery.Add(-time.Nanosecond)},
		{name: "at delivery", quote: "261016", now: delivery, expired: true},
		{name: "after delivery", quote: "261016", now: delivery.Add(time.Nanosecond), expired: true},
		{name: "local timezone", quote: "261016", now: delivery.In(time.FixedZone("AEDT", 11*60*60)), expired: true},
		{name: "alias", quote: "CW", now: delivery},
		{name: "invalid date", quote: "261332", now: delivery},
		{name: "non date", quote: "ABCDEF", now: delivery},
		{name: "perpetual", quote: "USD", now: delivery},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expired, isExpiredDeliveryContract(currency.NewPair(currency.BTC, currency.NewCode(tc.quote)), tc.now), "expiry should follow the delivery instant")
		})
	}
}

func TestGetWebsocketConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	for _, tt := range []struct {
		name          string
		asset         asset.Item
		authenticated bool
		endpoint      exchange.URL
	}{
		{name: "spot public", asset: asset.Spot, endpoint: exchange.WebsocketSpot},
		{name: "spot private", asset: asset.Spot, authenticated: true, endpoint: exchange.WebsocketPrivate},
		{name: "delivery public", asset: asset.Futures, endpoint: exchange.WebsocketFutures},
		{name: "delivery private", asset: asset.Futures, authenticated: true, endpoint: exchange.WebsocketFuturesPrivate},
		{name: "coin public", asset: asset.CoinMarginedFutures, endpoint: exchange.WebsocketCoinMargined},
		{name: "coin private", asset: asset.CoinMarginedFutures, authenticated: true, endpoint: exchange.WebsocketCoinMarginedPrivate},
		{name: "USDT public", asset: asset.USDTMarginedFutures, endpoint: exchange.WebsocketUSDTMargined},
		{name: "USDT private", asset: asset.USDTMarginedFutures, authenticated: true, endpoint: exchange.WebsocketUSDTMarginedPrivate},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conn, err := h.getWebsocketConnection(&subscription.Subscription{Asset: tt.asset, Authenticated: tt.authenticated})
			require.NoError(t, err, "getWebsocketConnection must find the asset connection")
			expected, err := h.Websocket.GetConnection(tt.endpoint)
			require.NoError(t, err, "expected websocket connection must be available")
			assert.Equal(t, expected, conn, "subscription should route to its dedicated connection")
		})
	}

	_, err := h.getWebsocketConnection(&subscription.Subscription{Asset: asset.Binary})
	require.ErrorIs(t, err, asset.ErrNotSupported, "getWebsocketConnection must reject unsupported assets")
}

func TestSubscribeConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketCoinMarginedPrivate)
	require.NoError(t, err, "coin-margined private websocket connection must be available")
	sub := &subscription.Subscription{
		Asset:            asset.CoinMarginedFutures,
		Channel:          subscription.MyOrdersChannel,
		Authenticated:    true,
		Pairs:            currency.Pairs{btcusdPair},
		QualifiedChannel: "orders.*",
	}
	err = h.subscribeConnection(t.Context(), conn, subscription.List{sub})
	require.NoError(t, err, "subscribeConnection must not error")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "subscription should be active")
}

func TestUnsubscribeConnection(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	sub := &subscription.Subscription{
		Asset:            asset.CoinMarginedFutures,
		Channel:          subscription.TickerChannel,
		Pairs:            currency.Pairs{btcusdPair},
		QualifiedChannel: "market.BTC-USD.detail",
	}
	require.NoError(t, h.subscribeConnection(t.Context(), conn, subscription.List{sub}), "subscribeConnection must not error")
	require.NoError(t, h.unsubscribeConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeConnection must not error")
	assert.Nil(t, conn.Subscriptions().Get(sub), "subscription should be removed")
}

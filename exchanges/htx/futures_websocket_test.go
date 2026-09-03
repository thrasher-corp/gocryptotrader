package htx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

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
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.Features.Subscriptions = defaultFuturesSubscriptions.Clone()
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdPair}, asset.CoinMarginedFutures, true), "enabled pairs must be set")

	subs, err := h.generateSubscriptionsForAsset(asset.CoinMarginedFutures, false)
	require.NoError(t, err, "generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 5, "coin-margined subscriptions must include each public market channel")
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
	h.futureContractCodes = map[string]currency.Code{"CW": currency.NewCode("250829")}
	deliveryPairs := currency.Pairs{currency.NewPairWithDelimiter("BTC", "250829", "-")}
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, false), "available delivery pairs must be set")
	require.NoError(t, h.SetPairs(deliveryPairs, asset.Futures, true), "enabled delivery pairs must be set")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "delivery generateSubscriptionsForAsset must not error")
	require.Len(t, subs, 4, "delivery subscriptions must include each public market channel")
	for _, sub := range subs {
		assert.Contains(t, sub.QualifiedChannel, "BTC_CW", "delivery channel should use HTX's expiry shorthand")
		require.Len(t, sub.Pairs, 1, "delivery subscription must retain one canonical pair")
		assert.True(t, sub.Pairs[0].Equal(deliveryPairs[0]), "delivery subscription should retain the configured canonical pair")
	}
	h.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel, Pairs: deliveryPairs}}
	multipleDeliveryPairs := currency.Pairs{deliveryPairs[0], currency.NewPairWithDelimiter("ETH", "250829", "-")}
	require.NoError(t, h.SetPairs(multipleDeliveryPairs, asset.Futures, false), "multiple delivery pairs must be available")
	require.NoError(t, h.SetPairs(multipleDeliveryPairs, asset.Futures, true), "multiple delivery pairs must be enabled")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "scoped delivery subscription generation must not error")
	require.Len(t, subs, 1, "scoped delivery subscription must not expand to every enabled contract")
	assert.Equal(t, deliveryPairs, subs[0].Pairs, "scoped delivery subscription should preserve its configured pair")
	require.NoError(t, h.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "delivery futures must be disabled")
	subs, err = h.generateSubscriptionsForAsset(asset.Futures, false)
	require.NoError(t, err, "disabled delivery subscription generation must not error")
	assert.Empty(t, subs, "disabled delivery futures should not generate subscriptions")
}

func TestFormatLegacyFuturesWSOrder(t *testing.T) {
	t.Parallel()
	base := legacyFuturesWSOrder{
		asset:          asset.Futures,
		contractCode:   "BTC-USD",
		direction:      "buy",
		status:         6,
		orderID:        123,
		clientOrderID:  456,
		volume:         2,
		price:          100,
		tradeVolume:    1,
		tradeTurnover:  100,
		fee:            0.1,
		feeAsset:       "BTC",
		leverage:       5,
		createdAt:      1000,
		cancelledAt:    2000,
		reduceOnly:     true,
		orderIDString:  "order-id",
		orderPriceType: "limit",
	}
	for _, tc := range []struct {
		name             string
		update           func(*legacyFuturesWSOrder)
		expectedType     order.Type
		expectedTIF      order.TimeInForce
		expectedOrderID  string
		expectedErr      error
		wantErr          bool
		checkFullDetails bool
	}{
		{name: "explicit price type", expectedType: order.Limit, expectedOrderID: "order-id", checkFullDetails: true},
		{name: "numeric limit", update: func(data *legacyFuturesWSOrder) {
			data.orderPriceType = ""
			data.orderType = 1
			data.orderIDString = ""
		}, expectedType: order.Limit, expectedOrderID: "123"},
		{name: "numeric market", update: func(data *legacyFuturesWSOrder) { data.orderPriceType = ""; data.orderType = 3 }, expectedType: order.Market, expectedOrderID: "order-id"},
		{name: "numeric post only", update: func(data *legacyFuturesWSOrder) { data.orderPriceType = ""; data.orderType = 6 }, expectedType: order.Limit, expectedTIF: order.PostOnly, expectedOrderID: "order-id"},
		{name: "invalid numeric type", update: func(data *legacyFuturesWSOrder) { data.orderPriceType = "" }, expectedErr: errInvalidOrderPriceType},
		{name: "invalid direction", update: func(data *legacyFuturesWSOrder) { data.direction = "invalid" }, expectedErr: errUnrecognisedOrderSide},
		{name: "invalid price type", update: func(data *legacyFuturesWSOrder) { data.orderPriceType = "invalid" }, expectedErr: errInvalidOrderPriceType},
		{name: "invalid status", update: func(data *legacyFuturesWSOrder) { data.status = 99 }, expectedErr: errInvalidOrderStatus},
		{name: "invalid pair", update: func(data *legacyFuturesWSOrder) { data.contractCode = "" }, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := base
			if tc.update != nil {
				tc.update(&data)
			}
			h := new(Exchange)
			h.Name = "HTX"
			detail, err := h.formatLegacyFuturesWSOrder(&data)
			if tc.wantErr || tc.expectedErr != nil {
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr, "formatLegacyFuturesWSOrder must return the expected error")
				} else {
					require.Error(t, err, "formatLegacyFuturesWSOrder must reject invalid data")
				}
				return
			}
			require.NoError(t, err, "formatLegacyFuturesWSOrder must convert the notification")
			assert.Equal(t, tc.expectedType, detail.Type, "order type should match")
			assert.Equal(t, tc.expectedTIF, detail.TimeInForce, "time in force should match")
			assert.Equal(t, tc.expectedOrderID, detail.OrderID, "order ID should match")
			if tc.checkFullDetails {
				assert.Equal(t, "HTX", detail.Exchange, "exchange should match")
				assert.Equal(t, "456", detail.ClientOrderID, "client order ID should match")
				assert.Equal(t, btcusdPair, detail.Pair, "pair should match")
				assert.Equal(t, order.Buy, detail.Side, "side should match")
				assert.Equal(t, order.Filled, detail.Status, "status should match")
				assert.Equal(t, time.UnixMilli(1000), detail.Date, "creation time should match")
				assert.Equal(t, time.UnixMilli(2000), detail.CloseTime, "cancellation time should match")
				assert.Equal(t, 100.0, detail.Price, "price should match")
				assert.Equal(t, 2.0, detail.Amount, "amount should match")
				assert.Equal(t, 1.0, detail.ExecutedAmount, "executed amount should match")
				assert.Equal(t, 1.0, detail.RemainingAmount, "remaining amount should match")
				assert.Equal(t, 100.0, detail.Cost, "cost should match")
				assert.Equal(t, 0.1, detail.Fee, "fee should match")
				assert.Equal(t, currency.BTC, detail.FeeAsset, "fee asset should match")
				assert.Equal(t, 5.0, detail.Leverage, "leverage should match")
				assert.True(t, detail.ReduceOnly, "reduce-only should match")
				assert.Equal(t, asset.Futures, detail.AssetType, "asset type should match")
			}
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

func TestWSAuthenticateConnection(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsAuthenticateConnection(t.Context(), h.Websocket.AuthConn)
	require.ErrorIs(t, err, common.ErrNilPointer, "wsAuthenticateConnection must reject a nil connection")

	mock := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	mock.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	conn, err := mock.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
	require.NoError(t, err, "delivery private websocket connection must be available")
	conn.SetURL(conn.GetURL() + "/notification")
	require.NoError(t, mock.wsAuthenticateConnection(t.Context(), conn), "wsAuthenticateConnection must authenticate derivative connections")
	assert.True(t, mock.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints should be enabled after login")

	spotConn, err := mock.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "spot private websocket connection must be available")
	spotConn.SetURL(spotConn.GetURL() + wsPrivatePath)
	require.NoError(t, mock.wsAuthenticateConnection(t.Context(), spotConn), "wsAuthenticateConnection must retain spot authentication")
	spotConn.SetURL("://invalid")
	err = mock.wsAuthenticateConnection(t.Context(), spotConn)
	require.ErrorIs(t, err, errInvalidEndpoint, "wsAuthenticateConnection must reject invalid endpoints")
}

func TestWSGenerateFuturesSignature(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketCoinMarginedPrivate)
	require.NoError(t, err, "coin-margined private websocket connection must be available")
	conn.SetURL("wss://api.hbdm.com/swap-notification")
	creds := &accounts.Credentials{Key: "access", Secret: "secret"}
	signature, err := h.wsGenerateFuturesSignature(conn, creds, "2017-05-11T15:19:30")
	require.NoError(t, err, "wsGenerateFuturesSignature must not error")
	payload := "GET\napi.hbdm.com\n/swap-notification\nAccessKeyId=access&SignatureMethod=HmacSHA256&SignatureVersion=2&Timestamp=2017-05-11T15%3A19%3A30"
	mac := hmac.New(sha256.New, []byte(creds.Secret))
	_, err = mac.Write([]byte(payload))
	require.NoError(t, err, "hash write must not error")
	assert.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), base64.StdEncoding.EncodeToString(signature), "signature should match HTX's derivative signing format")

	conn.SetURL("://invalid")
	_, err = h.wsGenerateFuturesSignature(conn, creds, "timestamp")
	require.ErrorIs(t, err, errInvalidEndpoint, "wsGenerateFuturesSignature must reject invalid endpoints")
	_, err = h.wsGenerateFuturesSignature(nil, creds, "timestamp")
	require.ErrorIs(t, err, common.ErrNilPointer, "wsGenerateFuturesSignature must reject nil connections")
	_, err = h.wsGenerateFuturesSignature(conn, nil, "timestamp")
	require.ErrorIs(t, err, common.ErrNilPointer, "wsGenerateFuturesSignature must reject nil credentials")
}

func TestWSFuturesLogin(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	conn, err := h.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
	require.NoError(t, err, "USDT private websocket connection must be available")
	conn.SetURL(wsUSDTMarginedPrivateURL)
	require.NoError(t, h.wsFuturesLogin(t.Context(), conn), "wsFuturesLogin must accept a successful authentication response")

	plain := new(Exchange)
	require.NoError(t, testexch.Setup(plain), "HTX setup must not error")
	err = plain.wsFuturesLogin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "wsFuturesLogin must reject nil connections")
}

func TestWSHandleFuturesPing(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
	require.NoError(t, err, "delivery private websocket connection must be available")
	require.NoError(t, h.wsHandleFuturesPing(t.Context(), conn, []byte(`{"op":"ping","ts":"1492420473058"}`)), "wsHandleFuturesPing must respond to string timestamps")
	err = h.wsHandleFuturesPing(t.Context(), conn, []byte(`{"op":"ping"}`))
	require.ErrorIs(t, err, common.ErrParsingWSField, "wsHandleFuturesPing must require a timestamp")
	err = h.wsHandleFuturesPing(t.Context(), conn, []byte(`{`))
	require.Error(t, err, "wsHandleFuturesPing must reject malformed messages")
	err = h.wsHandleFuturesPing(t.Context(), nil, []byte(`{"op":"ping","ts":1}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "wsHandleFuturesPing must reject nil connections")
}

func TestWSHandleFuturesOperationResponse(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
	require.NoError(t, err, "delivery private websocket connection must be available")
	err = h.wsHandleFuturesOperationResponse(conn, wsSubOp, []byte(`{"op":"sub"}`))
	require.ErrorIs(t, err, common.ErrParsingWSField, "wsHandleFuturesOperationResponse must require a topic")
	err = h.wsHandleFuturesOperationResponse(nil, wsSubOp, []byte(`{"op":"sub","topic":"orders.*"}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "wsHandleFuturesOperationResponse must reject nil connections")
}

func TestGetFuturesPrivateSubscription(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
	require.NoError(t, err, "delivery private websocket connection must be available")
	sub := &subscription.Subscription{
		Key:              "orders.*",
		Asset:            asset.Futures,
		Channel:          subscription.MyOrdersChannel,
		Authenticated:    true,
		QualifiedChannel: "orders.*",
	}
	require.NoError(t, conn.Subscriptions().Add(sub), "wildcard subscription must be stored")
	assert.Equal(t, sub, h.getFuturesPrivateSubscription(conn, "orders.BTC"), "concrete notifications should resolve to their wildcard subscription")
	assert.Equal(t, sub, h.getFuturesPrivateSubscription(conn, "orders"), "bare notification topics should resolve to their wildcard subscription")
	assert.Nil(t, h.getFuturesPrivateSubscription(conn, "invalid"), "unrecognised topics should not resolve")
}

func TestWSHandleDataFuturesPrivateNotification(t *testing.T) {
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
	require.NoError(t, conn.Subscriptions().Add(sub), "wildcard subscription must be stored")
	raw := []byte(`{"op":"notify","topic":"positions.BTC-USD","ts":1603878749908,"event":"order.match","data":[]}`)
	require.NoError(t, h.wsHandleData(t.Context(), conn, raw), "wsHandleData must route a concrete private derivative topic")
	message := <-h.Websocket.DataHandler.C
	assert.IsType(t, &SwapWsSubPositionUpdates{}, message.Data, "notification should be decoded using its subscription")
}

func TestWSHandleFuturesPrivateMessage(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleFuturesPrivateMessage(t.Context(), &subscription.Subscription{Asset: asset.Binary}, []byte(`{}`))
	require.ErrorIs(t, err, asset.ErrNotSupported, "wsHandleFuturesPrivateMessage must reject unsupported assets")
	err = h.wsHandleFuturesPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "wsHandleFuturesPrivateMessage must reject nil subscriptions")

	for _, tt := range []struct {
		a        asset.Item
		expected any
	}{
		{a: asset.Futures, expected: &FWsSubPositionUpdates{}},
		{a: asset.CoinMarginedFutures, expected: &SwapWsSubPositionUpdates{}},
		{a: asset.USDTMarginedFutures, expected: &V5WsPositionUpdate{}},
	} {
		sub := &subscription.Subscription{Asset: tt.a, Channel: wsPositionsChannel, Authenticated: true}
		require.NoErrorf(t, h.wsHandleFuturesPrivateMessage(t.Context(), sub, []byte(`{"op":"notify","topic":"orders.*","ts":1603878749908}`)), "%s private message must be dispatched", tt.a)
		message := <-h.Websocket.DataHandler.C
		assert.IsType(t, tt.expected, message.Data, "notification should use the asset schema")
	}
}

func TestFuturesAuthSubscribe(t *testing.T) {
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
}

func TestWSHandleDeliveryFuturesPrivateMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		channel  string
		expected any
		raw      string
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6,"order_id":1,"volume":2,"trade_volume":1}`},
		{name: "matches", channel: subscription.MyTradesChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USD","direction":"buy","order_type":1,"status":6,"order_id":1,"volume":2,"trade_volume":1}`},
		{name: "accounts", channel: subscription.MyAccountChannel, expected: []accounts.Change{}, raw: `{"ts":1603878749908,"data":[{"symbol":"BTC","margin_balance":2,"margin_frozen":1,"margin_available":1}]}`},
		{name: "positions", channel: wsPositionsChannel, expected: &FWsSubPositionUpdates{}},
		{name: "trigger orders", channel: wsTriggerOrdersChannel, expected: &FWsSubTriggerOrderUpdates{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			sub := &subscription.Subscription{Asset: asset.Futures, Channel: tt.channel, Authenticated: true}
			raw := []byte(tt.raw)
			if len(raw) == 0 {
				raw = []byte(`{"op":"notify","topic":"private.*","ts":1603878749908,"symbol":"BTC","contract_code":"BTC250829","data":[]}`)
			}
			require.NoError(t, h.wsHandleDeliveryFuturesPrivateMessage(t.Context(), sub, raw), "private delivery notification must be decoded")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleDeliveryFuturesPrivateMessage(t.Context(), &subscription.Subscription{Channel: "unsupported"}, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "unknown private delivery channels must be rejected")
	err = h.wsHandleDeliveryFuturesPrivateMessage(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel}, []byte(`{`))
	require.Error(t, err, "malformed private delivery notifications must be rejected")
	err = h.wsHandleDeliveryFuturesPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "nil private delivery subscriptions must be rejected")
}

func TestWSHandleFundingRateMsg(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	sub := &subscription.Subscription{
		Asset: asset.CoinMarginedFutures,
		Pairs: currency.Pairs{btcusdPair},
	}
	err := h.wsHandleFundingRateMsg(t.Context(), sub, []byte(`{
		"ch":"public.BTC-USD.funding_rate",
		"ts":1604312615051,
		"tick":{
			"estimated_rate":"0.0002",
			"funding_rate":"0.0001",
			"contract_code":"BTC-USD",
			"funding_time":"1604312615051",
			"next_funding_time":"1604341415051"
		}
	}`))
	require.NoError(t, err, "wsHandleFundingRateMsg must not error")
	message := <-h.Websocket.DataHandler.C
	rate, ok := message.Data.(*WsFundingRate)
	require.True(t, ok, "websocket funding update must have the expected type")
	assert.Equal(t, asset.CoinMarginedFutures, rate.Asset, "asset should match")
	assert.Equal(t, btcusdPair, rate.Pair, "pair should match")
	assert.Equal(t, 0.0001, rate.Tick.FundingRate.Float64(), "funding rate should match")
	assert.Equal(t, time.UnixMilli(1604312615051), rate.Tick.FundingTime.Time(), "funding time should match")
}

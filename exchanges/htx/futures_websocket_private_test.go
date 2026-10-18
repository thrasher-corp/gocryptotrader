package htx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestFormatLegacyFuturesWSOrder(t *testing.T) {
	t.Parallel()
	for _, item := range []asset.Item{asset.Futures, asset.CoinMarginedFutures} {
		t.Run(item.String()+" cost asset", func(t *testing.T) {
			t.Parallel()
			detail, err := e.formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{asset: item, contractCode: "ADA-USD", direction: "sell", orderPriceType: "post_only", status: 6, volume: 1, tradeVolume: 1, tradeTurnover: 10})
			require.NoError(t, err, "order notification must convert")
			assert.Equal(t, currency.USD, detail.CostAsset, "inverse contract turnover should be costed in USD")
		})
	}
	t.Run("absent optional values", func(t *testing.T) {
		t.Parallel()
		detail, err := new(Exchange).formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{contractCode: "BTC-USD", direction: "buy", orderPriceType: "limit", status: 3})
		require.NoError(t, err, "conversion must succeed")
		assert.Empty(t, detail.ClientOrderID, "absent client ID should stay empty")
		assert.True(t, detail.Date.IsZero(), "absent creation time should stay zero")
		assert.True(t, detail.CloseTime.IsZero(), "absent cancellation time should stay zero")
	})
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
		fee:            -0.1,
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
	for _, tc := range []struct {
		name         string
		authResponse string
		connectErr   error
		tradeErr     error
	}{
		{name: "successful trade login", authResponse: `{"op":"auth","err-code":0}`},
		{name: "rejected trade login", authResponse: `{"op":"auth","err-code":2001,"err-msg":"invalid authentication"}`, connectErr: errAPIResponse, tradeErr: websocket.ErrNotConnected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange defaults must load")
			cfg := h.Config
			cfg.API.AuthenticatedWebsocketSupport = true
			cfg.API.AuthenticatedSupport = true
			h.SetDefaults()
			require.NoError(t, h.Setup(cfg), "authenticated trade setup must succeed")
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			h.Features.Subscriptions = nil

			server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
				tb.Helper()
				var req wsFuturesAuthRequest
				if err := json.Unmarshal(message, &req); err != nil {
					return err
				}
				if req.Operation == wsAuthChannel {
					assert.False(tb, h.Websocket.IsConnected(), "connection should await authentication")
					return conn.WriteMessage(gws.TextMessage, []byte(tc.authResponse))
				}
				return wsFixture(tb, message, conn)
			}))
			server.Start()
			require.NoError(t, h.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "trade connection must use the mock server")
			t.Cleanup(func() {
				if err := h.Websocket.Disable(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled, "cleanup should disable reconnection")
				}
				if err := h.Websocket.Shutdown(); err != nil {
					assert.ErrorIs(t, err, websocket.ErrNotConnected, "cleanup should close owned connections")
				}
			})

			err := h.Websocket.Connect(t.Context())
			require.ErrorIs(t, err, tc.connectErr, "connection setup must match the login result")
			_, err = h.WSPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT"})
			assert.ErrorIs(t, err, tc.tradeErr, "order submission should require successful login")
			_, err = h.WSCancelV5Order(t.Context(), &V5CancelOrderRequest{ContractCode: "BTC-USDT", OrderID: "1"})
			assert.ErrorIs(t, err, tc.tradeErr, "order cancellation should require successful login")
		})
	}
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	conn, err := h.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
	require.NoError(t, err, "USDT private websocket connection must be available")
	conn.SetURL(wsUSDTMarginedPrivateURL)
	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, h.wsFuturesLogin(t.Context(), conn), "wsFuturesLogin must accept a successful authentication response")
	assert.True(t, h.Websocket.CanUseAuthenticatedEndpoints(), "successful login should enable authenticated endpoints")

	plain := new(Exchange)
	require.NoError(t, testexch.Setup(plain), "HTX setup must not error")
	err = plain.wsFuturesLogin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "wsFuturesLogin must reject nil connections")
}

func TestWSHandleFuturesPing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expected string
	}{
		{name: "string", input: `{"op":"ping","ts":"1492420473058"}`, expected: `{"op":"pong","ts":"1492420473058"}`},
		{name: "number", input: `{"op":"ping","ts":1492420473027}`, expected: `{"op":"pong","ts":1492420473027}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			messages := make(chan []byte, 1)
			h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(tb testing.TB, msg []byte, conn *gws.Conn) error {
				tb.Helper()
				operation, _ := jsonparser.GetString(msg, "op")
				if operation == "pong" {
					messages <- msg
					return nil
				}
				return wsFixture(tb, msg, conn)
			}))
			conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
			require.NoError(t, err, "delivery private websocket connection must be available")
			require.NoError(t, h.wsHandleFuturesPing(t.Context(), conn, []byte(tc.input)), "wsHandleFuturesPing must respond to valid timestamps")
			select {
			case msg := <-messages:
				assert.JSONEq(t, tc.expected, string(msg), "futures pong should echo the timestamp representation")
			case <-time.After(time.Second):
				require.Fail(t, "futures pong must be sent")
			}
		})
	}
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
	require.NoError(t, err, "delivery private websocket connection must be available")
	err = h.wsHandleFuturesPing(t.Context(), conn, []byte(`{"op":"ping"}`))
	require.ErrorIs(t, err, common.ErrParsingWSField, "wsHandleFuturesPing must require a timestamp")
	err = h.wsHandleFuturesPing(t.Context(), conn, []byte(`{`))
	require.Error(t, err, "wsHandleFuturesPing must reject malformed messages")
	err = h.wsHandleFuturesPing(t.Context(), conn, []byte(`{"op":"ping","ts":1}`))
	require.ErrorContains(t, err, "invalid timestamp format", "wsHandleFuturesPing must reject a malformed timestamp")
	err = h.wsHandleFuturesPing(t.Context(), nil, []byte(`{"op":"ping","ts":1}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "wsHandleFuturesPing must reject nil connections")
}

func TestGetFuturesPrivateSubscription(t *testing.T) {
	t.Parallel()
	t.Run("early scoped setup lookup", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		for _, tc := range []struct {
			filter any
			asset  asset.Item
		}{
			{filter: exchange.WebsocketFuturesPrivate, asset: asset.Futures},
			{filter: exchange.WebsocketCoinMarginedPrivate, asset: asset.CoinMarginedFutures},
		} {
			conn, err := h.Websocket.CreateTestConnection(tc.filter)
			require.NoError(t, err, "private test connection must be created")
			require.NoError(t, h.Websocket.TrackTestConnection(tc.filter, conn), "connection must belong to its setup")
			sub := &subscription.Subscription{Key: "orders.*", QualifiedChannel: "orders.*", Asset: tc.asset, Channel: subscription.MyOrdersChannel, Authenticated: true}
			sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
			require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "pending subscription must enter setup store")
			assert.Empty(t, conn.Subscriptions().List(), "local store should still await full batch completion")
			assert.Same(t, sub, h.getFuturesPrivateSubscription(conn, "orders.BTC-USD"), "early notification should resolve its own setup wildcard")
			assert.Same(t, sub, h.getFuturesPrivateSubscription(conn, "orders.*"), "exact topic should resolve its own setup")
			require.NoError(t, h.wsHandleData(t.Context(), conn, []byte(`{"op":"notify","topic":"orders.BTC-USD","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6,"order_id":1,"volume":2,"trade_volume":1}`)), "notification must dispatch before the batch finishes")
			require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
			message := <-h.Websocket.DataHandler.C
			detail, ok := message.Data.(*order.Detail)
			require.True(t, ok, "notification must produce canonical order")
			assert.Equal(t, tc.asset, detail.AssetType, "same wildcard across setups should retain correct asset")
		}
	})

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
	sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
	require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "wildcard subscription must be registered")
	assert.Equal(t, sub, h.getFuturesPrivateSubscription(conn, "orders.BTC"), "concrete notifications should resolve to their wildcard subscription")
	assert.Equal(t, sub, h.getFuturesPrivateSubscription(conn, "orders"), "bare notification topics should resolve to their wildcard subscription")
	assert.Nil(t, h.getFuturesPrivateSubscription(conn, "invalid"), "unrecognised topics should not resolve")
}

func TestWSHandleFuturesPrivateMessage(t *testing.T) {
	t.Parallel()
	t.Run("fill lifecycle", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures} {
			for _, feeAsset := range []string{"", "BTC"} {
				t.Run(fmt.Sprintf("%s/fillFeeAsset=%s", a, feeAsset), func(t *testing.T) {
					t.Parallel()
					h := new(Exchange)
					require.NoError(t, testexch.Setup(h), "isolated exchange setup must succeed")
					sub := &subscription.Subscription{Asset: a, Channel: subscription.MyOrdersChannel, Authenticated: true}
					// Distinct global IDs can share the venue's non-unique match trade_id.
					raw := []byte(fmt.Sprintf(`{"op":"notify","topic":"orders.BTC-USD","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6,"order_id":1,"order_id_str":"order-1","volume":2,"price":110,"trade_volume":2,"trade_turnover":200,"trade_avg_price":100,"fee":-0.003,"fee_asset":"BTC","trade":[{"id":"9007199254740993","trade_id":42,"trade_volume":1,"trade_price":90,"trade_fee":-0.001,"trade_turnover":90,"created_at":1603878749908,"role":"maker","fee_asset":%q},{"id":"9007199254740994","trade_id":42,"trade_volume":1,"trade_price":110,"trade_fee":-0.002,"trade_turnover":110,"created_at":1603878749909,"role":"taker","fee_asset":%q}]}`, feeAsset, feeAsset))
					require.NoError(t, h.wsHandleFuturesPrivateMessage(t.Context(), sub, raw), "order notification must pass through its asset handler")
					read := func() *order.Detail {
						t.Helper()
						select {
						case message := <-h.Websocket.DataHandler.C:
							detail, ok := message.Data.(*order.Detail)
							require.True(t, ok, "handler must relay canonical order details")
							return detail
						case <-time.After(time.Second):
							require.FailNow(t, "handler must relay its notification before the deadline")
							return nil
						}
					}
					detail := read()
					assert.Equal(t, a, detail.AssetType, "handler should retain the derivative asset")
					assert.Equal(t, 100.0, detail.AverageExecutedPrice, "wire average should remain independent of the limit price")
					assert.Equal(t, 0.003, detail.Fee, "aggregate charged fee should be positive in GCT")
					assert.Equal(t, currency.BTC, detail.FeeAsset, "aggregate fee currency should be retained")
					require.Len(t, detail.Trades, 2, "both fills must be retained despite their shared match ID")
					for i, expected := range []struct {
						id         string
						price, fee float64
						maker      bool
					}{
						{id: "9007199254740993", price: 90, fee: 0.001, maker: true},
						{id: "9007199254740994", price: 110, fee: 0.002},
					} {
						fill := detail.Trades[i]
						assert.Equal(t, expected.id, fill.TID, "fill identity should preserve its global integer string")
						assert.Equal(t, expected.price, fill.Price, "fill execution price should be retained")
						assert.Equal(t, 1.0, fill.Amount, "fill contract volume should be retained")
						assert.Equal(t, expected.fee, fill.Fee, "fill charged fee should be positive in GCT")
						assert.Equal(t, "BTC", fill.FeeAsset, "fill fee currency should use the order currency when omitted")
						assert.Equal(t, expected.maker, fill.IsMaker, "fill role should distinguish maker and taker")
						assert.Equal(t, time.UnixMilli(1603878749908+int64(i)), fill.Timestamp, "fill timestamp should retain millisecond precision")
					}
					stored := &order.Detail{OrderID: detail.OrderID, Exchange: detail.Exchange, Pair: detail.Pair, AssetType: detail.AssetType}
					require.NoError(t, stored.UpdateOrderFromDetail(detail), "order store must accept the charged fee")
					assert.Equal(t, 0.003, stored.Fee, "order store should record the derivative fee")
					sub.Channel = subscription.MyTradesChannel
					match := []byte(`{"op":"notify","topic":"matchOrders.BTC-USD","contract_code":"BTC-USD","direction":"buy","order_type":1,"status":6,"order_id":1,"order_id_str":"order-1","volume":2,"price":110,"trade_volume":2,"trade":[{"id":"9007199254740993","trade_id":42,"trade_volume":1,"trade_price":90,"created_at":1603878749908,"role":"maker"},{"id":"9007199254740994","trade_id":42,"trade_volume":1,"trade_price":110,"created_at":1603878749909,"role":"taker"}]}`)
					require.NoError(t, h.wsHandleFuturesPrivateMessage(t.Context(), sub, match), "fee-less match notification must pass through its asset handler")
					matched := read()
					assert.Empty(t, matched.Trades, "fee-less matches should not overwrite fee-bearing fills")
					require.NoError(t, stored.UpdateOrderFromDetail(matched), "order management merge must accept the match update")
					assert.Equal(t, detail.Trades, stored.Trades, "subsequent fee-less match should preserve every recorded fill field")
					assert.Equal(t, detail.AverageExecutedPrice, stored.AverageExecutedPrice, "subsequent match should preserve the average execution price")
					assert.Equal(t, detail.Fee, stored.Fee, "subsequent match should preserve the aggregate fee")
					assert.Empty(t, h.Websocket.DataHandler.C, "each handler invocation should relay exactly one order")
				})
			}
		}
	})
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
		require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
		message := <-h.Websocket.DataHandler.C
		assert.IsType(t, tt.expected, message.Data, "notification should use the asset schema")
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
		{name: "cancellation pending", channel: subscription.MyTradesChannel, raw: `{"status":9}`},
		{name: "cancelling", channel: subscription.MyTradesChannel, raw: `{"status":10}`},
		{name: "accounts", channel: subscription.MyAccountChannel, expected: []accounts.Change{}, raw: `{"ts":1603878749908,"data":[{"symbol":"BTC","margin_balance":2,"margin_frozen":1,"margin_available":1}]}`},
		{name: "positions", channel: wsPositionsChannel, expected: &FWsSubPositionUpdates{}},
		{name: "trigger orders", channel: wsTriggerOrdersChannel, expected: &FWsSubTriggerOrderUpdates{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			sub := &subscription.Subscription{Asset: asset.Futures, Channel: tt.channel, Authenticated: true}
			raw := []byte(tt.raw)
			if len(raw) == 0 {
				raw = []byte(`{"op":"notify","topic":"private.*","ts":1603878749908,"symbol":"BTC","contract_code":"BTC250829","data":[]}`)
			}
			require.NoError(t, h.wsHandleDeliveryFuturesPrivateMessage(t.Context(), sub, raw), "private delivery notification must be decoded")
			if tt.expected == nil {
				assert.Empty(t, h.Websocket.DataHandler.C, "transient cancellation states should not dispatch an order")
				return
			}
			require.Len(t, h.Websocket.DataHandler.C, 1, "handler must dispatch one message")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
			if changes, ok := message.Data.([]accounts.Change); ok {
				for _, change := range changes {
					stored, err := h.Accounts.GetBalance("", &accounts.Credentials{Key: "key", Secret: "secret"}, sub.Asset, change.Balance.Currency)
					require.NoError(t, err, "account notification must reach the balance store")
					assert.Equal(t, change.Balance.Total, stored.Total, "stored total should match the notification")
					assert.Equal(t, change.Balance.Free, stored.Free, "stored free amount should match the notification")
				}
			}
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

func TestStoreWebsocketBalances(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h))
			h.API.AuthenticatedSupport = true
			creds := &accounts.Credentials{Key: "key", Secret: "secret"}
			h.SetCredentials(creds)
			for i, skew := range []time.Duration{-time.Second, -200 * time.Millisecond, 2 * time.Second} {
				before := time.Now()
				change := accounts.Change{AssetType: a, Balance: accounts.Balance{Currency: currency.USDT, Total: float64(i + 2), Free: float64(i + 1), Hold: 1, UpdatedAt: before.Add(skew)}}
				require.NoError(t, h.storeWebsocketBalances(t.Context(), a, []accounts.Change{change}))
				stored, err := h.Accounts.GetBalance("", creds, a, currency.USDT)
				require.NoError(t, err)
				assert.Equal(t, change.Balance.Total, stored.Total)
				assert.Equal(t, change.Balance.Free, stored.Free)
				assert.Equal(t, change.Balance.Hold, stored.Hold)
				assert.WithinRange(t, stored.UpdatedAt, before, time.Now())
				// A REST-style snapshot must remain admissible after a future-dated push.
				snapshot := accounts.SubAccounts{accounts.NewSubAccount(a, "")}
				snapshot[0].Balances.Set(currency.USDT, accounts.Balance{Currency: currency.USDT, Total: 100, Free: 99, Hold: 1, UpdatedAt: time.Now()})
				require.NoError(t, h.Accounts.Save(t.Context(), snapshot, false))
			}
			require.NoError(t, h.storeWebsocketBalances(t.Context(), a, nil))
			err := h.storeWebsocketBalances(t.Context(), a, []accounts.Change{{Balance: accounts.Balance{Currency: currency.EMPTYCODE}}})
			require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
		})
	}
}

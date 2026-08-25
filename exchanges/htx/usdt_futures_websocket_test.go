package htx

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newV5TradeWebsocketTestExchange configures the dedicated trade connection used by V5 websocket tests.
func newV5TradeWebsocketTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	h := testexch.MockWsInstance[Exchange](t, handler)
	spotConn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot connection must be available")
	require.NoError(t, h.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      spotConn.GetURL(),
		RateLimit:                request.NewWeightedRateLimitByDuration(3 * time.Second / 24),
		ResponseCheckTimeout:     time.Second,
		ResponseMaxLimit:         time.Second,
		Connector:                h.wsConnect,
		Handler:                  h.wsHandleData,
		MessageFilter:            exchange.WebsocketTrade,
		SubscriptionsNotRequired: true,
	}), "trade connection must be configured")
	require.NoError(t, h.Websocket.Shutdown(), "existing websocket connections must shut down")
	require.NoError(t, h.Websocket.Connect(t.Context()), "websocket connections must reconnect with the trade endpoint")
	_, err = h.Websocket.GetConnection(exchange.WebsocketTrade)
	require.NoError(t, err, "trade connection must be available")
	return h
}

func TestSendV5TradeRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		setup       func(*testing.T) *Exchange
		expectedErr bool
		cancel      bool
	}{
		{
			name: "success",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
			},
		},
		{
			name: "missing connection",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				h := new(Exchange)
				require.NoError(t, testexch.Setup(h), "HTX setup must not error")
				return h
			},
			expectedErr: true,
		},
		{
			name: "cancelled context",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
			},
			expectedErr: true,
			cancel:      true,
		},
		{
			name: "exchange error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				fixture := func(tb testing.TB, message []byte, conn *gws.Conn) error {
					tb.Helper()
					cid, err := jsonparser.GetString(message, "cid")
					if err != nil {
						return err
					}
					return conn.WriteMessage(gws.TextMessage, []byte(`{"code":400,"message":"invalid request","cid":"`+cid+`"}`))
				}
				return newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, fixture))
			},
			expectedErr: true,
		},
		{
			name: "send error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
				conn, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
				require.NoError(t, err, "trade connection must be available")
				require.NoError(t, conn.Shutdown(), "trade connection must shut down")
				return h
			},
			expectedErr: true,
		},
		{
			name: "decode error",
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				fixture := func(tb testing.TB, message []byte, conn *gws.Conn) error {
					tb.Helper()
					cid, err := jsonparser.GetString(message, "cid")
					if err != nil {
						return err
					}
					return conn.WriteMessage(gws.TextMessage, []byte(`{"code":200,"cid":"`+cid+`","data":[]}`))
				}
				return newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, fixture))
			},
			expectedErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := tc.setup(t)
			var response *V5WsOrderResponse
			ctx := t.Context()
			if tc.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := h.sendV5TradeRequest(ctx, "place_order", &V5OrderRequest{ContractCode: "BTC-USDT"}, &response)
			if tc.expectedErr {
				require.Error(t, err, "sendV5TradeRequest must return the expected error")
				return
			}
			require.NoError(t, err, "sendV5TradeRequest must not error")
			require.NotNil(t, response, "sendV5TradeRequest response must not be nil")
			assert.Equal(t, "1", response.Data.OrderID, "order ID should decode")
		})
	}
}

func TestWSPlaceV5Order(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSPlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSPlaceV5Order must reject nil request")
	resp, err := h.WSPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1})
	require.NoError(t, err, "WSPlaceV5Order must not error")
	assert.Equal(t, "1", resp.Data.OrderID, "order ID should decode")
	assert.Equal(t, types.Number(19), resp.RateLimit.Remaining, "remaining rate limit should decode")
}

func TestWSPlaceV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSPlaceV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WSPlaceV5BatchOrders must reject empty request")
	resp, err := h.WSPlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1}})
	require.NoError(t, err, "WSPlaceV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one order acknowledgement must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestWSCancelV5Order(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelV5Order must reject nil request")
	resp, err := h.WSCancelV5Order(t.Context(), &V5CancelOrderRequest{ContractCode: "BTC-USDT", OrderID: "1"})
	require.NoError(t, err, "WSCancelV5Order must not error")
	assert.Equal(t, "1", resp.Data.OrderID, "order ID should decode")
}

func TestWSCancelV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelV5BatchOrders must reject nil request")
	resp, err := h.WSCancelV5BatchOrders(t.Context(), &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1"}})
	require.NoError(t, err, "WSCancelV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestWSCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	h := newV5TradeWebsocketTestExchange(t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelAllV5Orders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelAllV5Orders must reject nil request")
	resp, err := h.WSCancelAllV5Orders(t.Context(), &V5CancelAllOrdersRequest{ContractCode: "BTC-USDT"})
	require.NoError(t, err, "WSCancelAllV5Orders must not error")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestSendV5WSOrderUpdate(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		data := &V5WsOrderData{
			Side:              "buy",
			PositionSide:      "long",
			Type:              orderPriceTypePostOnly,
			OrderID:           "1",
			ClientOrderID:     "2",
			MarginMode:        "cross",
			Price:             100,
			Volume:            2,
			LeverageRate:      5,
			State:             "filled",
			ReduceOnly:        true,
			TimeInForce:       "gtc",
			TradeAveragePrice: 100,
			TradeVolume:       1,
			TradeTurnover:     100,
			FeeCurrency:       "USDT",
			Fee:               0.1,
			CreatedTime:       1000,
			UpdatedTime:       2000,
		}
		sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures}
		require.NoError(t, h.sendV5WSOrderUpdate(t.Context(), sub, "BTC-USDT", data), "sendV5WSOrderUpdate must convert and send the update")
		message := <-h.Websocket.DataHandler.C
		detail, ok := message.Data.(*order.Detail)
		require.True(t, ok, "update must use the canonical order type")
		assert.Equal(t, "1", detail.OrderID, "order ID should match")
		assert.Equal(t, "2", detail.ClientOrderID, "client order ID should match")
		assert.Equal(t, order.Limit, detail.Type, "order type should match")
		assert.Equal(t, order.PostOnly, detail.TimeInForce, "time in force should match")
		assert.Equal(t, 5.0, detail.Leverage, "leverage should match")
		assert.True(t, detail.ReduceOnly, "reduce-only should match")
		assert.Equal(t, time.UnixMilli(1000), detail.Date, "creation time should match")
		assert.Equal(t, time.UnixMilli(2000), detail.LastUpdated, "update time should match")
	})

	t.Run("invalid order", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		err := h.sendV5WSOrderUpdate(t.Context(), &subscription.Subscription{Asset: asset.USDTMarginedFutures}, "BTC-USDT", &V5WsOrderData{Side: "invalid"})
		require.Error(t, err, "sendV5WSOrderUpdate must reject an invalid order")
	})
}

func TestWSHandleUSDTMarginedPrivateMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		channel  string
		expected any
		raw      string
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":{"side":"buy","type":"limit","order_id":"1","margin_mode":"cross","state":"filled","time_in_force":"gtc","volume":"2","trade_volume":"1"}}`},
		{name: "trades", channel: wsTradeUpdatesChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":{"side":"buy","type":"limit","order_id":"1","margin_mode":"cross","state":"filled","time_in_force":"gtc","volume":"2","trade_volume":"1"}}`},
		{name: "trade details", channel: wsExecutionDetailsChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":{"side":"buy","type":"limit","order_id":"1","margin_mode":"cross","state":"filled","time_in_force":"gtc","volume":"2","trade_volume":"1"}}`},
		{name: "positions", channel: wsPositionsChannel, expected: &V5WsPositionUpdate{}},
		{name: "account", channel: subscription.MyAccountChannel, expected: []accounts.Change{}, raw: `{"ts":1603878749908,"data":{"details":[{"currency":"USDT","equity":"2","available":"1","isolated_available":"0.5"}]}}`},
		{name: "matches", channel: subscription.MyTradesChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USDT","data":{"side":"buy","type":"limit","order_id":"1","margin_mode":"cross","state":"filled","time_in_force":"gtc","volume":"2","trade_volume":"1"}}`},
		{name: "algo orders", channel: wsTriggerOrdersChannel, expected: &V5WsAlgoOrderUpdate{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			raw := []byte(tt.raw)
			if len(raw) == 0 {
				raw = []byte(`{"op":"notify","topic":"private","ts":1603878749908,"uid":"123","contract_code":"BTC-USDT","data":{}}`)
			}
			require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "private USDT-margined notification must be decoded")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: "unsupported"}, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "unknown private USDT-margined channels must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel}, []byte(`{`))
	require.Error(t, err, "malformed private USDT-margined notifications must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "nil private USDT-margined subscriptions must be rejected")

	raw := []byte(`{
		"op":"notify",
		"topic":"orders",
		"contract_code":"SHIB-USDT",
		"ts":1640756528985,
		"uid":"502061937",
		"data":{
			"side":"buy",
			"position_side":"short",
			"type":"limit",
			"price":"0.0000124",
			"volume":"2",
			"lever_rate":30,
			"state":"new",
			"order_id":"1381668675223068672",
			"client_order_id":"1381668675223068672",
			"margin_mode":"cross",
			"reduce_only":true,
			"time_in_force":"gtc"
		}
	}`)
	sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}
	require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "V5 order notification must be decoded")
	message := <-h.Websocket.DataHandler.C
	orderUpdate, ok := message.Data.(*order.Detail)
	require.True(t, ok, "V5 order notification must use the canonical order type")
	assert.Equal(t, "1381668675223068672", orderUpdate.OrderID, "order ID should be retained")
	assert.Equal(t, 30.0, orderUpdate.Leverage, "leverage should be retained")
	assert.True(t, orderUpdate.ReduceOnly, "reduce-only should be retained")
}

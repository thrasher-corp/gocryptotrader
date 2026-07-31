package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWSPlaceV5Order(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSPlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSPlaceV5Order must reject nil request")
	resp, err := h.WSPlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1})
	require.NoError(t, err, "WSPlaceV5Order must not error")
	assert.Equal(t, "1", resp.Data.OrderID, "order ID should decode")
	assert.Equal(t, types.Number(19), resp.RateLimit.Remaining, "remaining rate limit should decode")
}

func TestWSPlaceV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSPlaceV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WSPlaceV5BatchOrders must reject empty request")
	resp, err := h.WSPlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1}})
	require.NoError(t, err, "WSPlaceV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one order acknowledgement must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestWSCancelV5Order(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelV5Order must reject nil request")
	resp, err := h.WSCancelV5Order(t.Context(), &V5CancelOrderRequest{ContractCode: "BTC-USDT", OrderID: "1"})
	require.NoError(t, err, "WSCancelV5Order must not error")
	assert.Equal(t, "1", resp.Data.OrderID, "order ID should decode")
}

func TestWSCancelV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelV5BatchOrders must reject nil request")
	resp, err := h.WSCancelV5BatchOrders(t.Context(), &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1"}})
	require.NoError(t, err, "WSCancelV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestWSCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	_, err := h.WSCancelAllV5Orders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WSCancelAllV5Orders must reject nil request")
	resp, err := h.WSCancelAllV5Orders(t.Context(), &V5CancelAllOrdersRequest{ContractCode: "BTC-USDT"})
	require.NoError(t, err, "WSCancelAllV5Orders must not error")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestWSHandleUSDTMarginedPrivateMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		channel  string
		expected any
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, expected: &V5WsOrderUpdate{}},
		{name: "trades", channel: wsTradeUpdatesChannel, expected: &V5WsTradeUpdate{}},
		{name: "trade details", channel: wsExecutionDetailsChannel, expected: &V5WsTradeDetailUpdate{}},
		{name: "positions", channel: wsPositionsChannel, expected: &V5WsPositionUpdate{}},
		{name: "account", channel: subscription.MyAccountChannel, expected: &V5WsAccountUpdate{}},
		{name: "matches", channel: subscription.MyTradesChannel, expected: &V5WsMatchOrderUpdate{}},
		{name: "algo orders", channel: wsTriggerOrdersChannel, expected: &V5WsAlgoOrderUpdate{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			raw := []byte(`{"op":"notify","topic":"private","ts":1603878749908,"uid":"123","contract_code":"BTC-USDT","data":{}}`)
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
			"reduce_only":true
		}
	}`)
	sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}
	require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "V5 order notification must be decoded")
	message := <-h.Websocket.DataHandler.C
	orderUpdate, ok := message.Data.(*V5WsOrderUpdate)
	require.True(t, ok, "V5 order notification must use the order response type")
	assert.Equal(t, "1381668675223068672", orderUpdate.Data.OrderID, "order ID should be retained")
	assert.Equal(t, uint64(30), orderUpdate.Data.LeverageRate, "leverage should be retained")
	assert.True(t, orderUpdate.Data.ReduceOnly, "reduce-only should be retained")
}

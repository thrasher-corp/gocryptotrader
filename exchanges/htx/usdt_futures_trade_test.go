package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/trade/batch_orders", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	_, err := h.PlaceV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceV5BatchOrders must reject empty request")
	resp, err := h.PlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1}})
	require.NoError(t, err, "PlaceV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one order acknowledgement must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestCancelV5BatchOrders(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/trade/cancel_batch_orders", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	_, err := h.CancelV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelV5BatchOrders must reject nil request")
	resp, err := h.CancelV5BatchOrders(t.Context(), &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1"}})
	require.NoError(t, err, "CancelV5BatchOrders must not error")
	require.Len(t, resp.Data, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestSetV5CancelAfter(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/trade/cancel-after", `{"code":200,"data":{"current_time":"1767145577000","trigger_time":"1767145637000"}}`, nil)
	_, err := h.SetV5CancelAfter(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetV5CancelAfter must reject nil request")
	resp, err := h.SetV5CancelAfter(t.Context(), &V5CancelAfterRequest{Enabled: "on", Timeout: 60})
	require.NoError(t, err, "SetV5CancelAfter must not error")
	assert.False(t, resp.Data.TriggerTime.Time().IsZero(), "trigger time should decode")
}

func TestCloseV5Position(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/trade/position", `{"code":200,"data":{"order_id":"1"}}`, nil)
	_, err := h.CloseV5Position(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CloseV5Position must reject nil request")
	resp, err := h.CloseV5Position(t.Context(), &V5ClosePositionRequest{ContractCode: "BTC-USDT", MarginMode: "cross", PositionSide: "long"})
	require.NoError(t, err, "CloseV5Position must not error")
	assert.Equal(t, "1", resp.Data.OrderID, "order ID should decode")
}

func TestCloseAllV5Positions(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/trade/position_all", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	resp, err := h.CloseAllV5Positions(t.Context())
	require.NoError(t, err, "CloseAllV5Positions must not error")
	require.Len(t, resp.Data, 1, "one close acknowledgement must decode")
	assert.Equal(t, "1", resp.Data[0].OrderID, "order ID should decode")
}

func TestGetV5OrderHistory(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/trade/order/history", `{"code":200,"data":[{"order_id":"1","volume":"2"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
		assert.Equal(t, "cross", r.URL.Query().Get("margin_mode"), "margin mode should be sent")
	})
	_, err := h.GetV5OrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5OrderHistory must reject nil request")
	resp, err := h.GetV5OrderHistory(t.Context(), &V5OrderHistoryRequest{
		ContractCode: "BTC-USDT",
		MarginMode:   "cross",
		States:       "filled",
		Type:         "limit",
		PriceMatch:   "opponent",
		TimeInForce:  "gtc",
		StartTime:    time.UnixMilli(1),
		EndTime:      time.UnixMilli(2),
		From:         1,
		Limit:        10,
		Direction:    "next",
	})
	require.NoError(t, err, "GetV5OrderHistory must not error")
	require.Len(t, resp.Data, 1, "one historical order must decode")
	assert.Equal(t, types.Number(2), resp.Data[0].Volume, "volume should decode")
}

func TestGetV5OrderDetails(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/trade/order/details", `{"code":200,"data":[{"id":"1124147771","contract_code":"BTC-USDT","order_id":"1343541341268738048","trade_id":"100000032538647","side":"sell","position_side":"short","order_type":"1","margin_mode":"cross","type":"limit","role":"TAKER","trade_price":"31400","trade_volume":"1","trade_turnover":"31.4","created_time":1740366817564,"updated_time":1740366817564,"order_source":"api","fee_currency":"USDT","trade_fee":"0.01884","deduction_price":"","profit":"0","contract_type":"swap"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
		assert.Equal(t, "1343541341268738048", r.URL.Query().Get("order_id"), "order ID should be sent")
		assert.Equal(t, "100", r.URL.Query().Get("limit"), "limit should be sent")
	})
	_, err := h.GetV5OrderDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5OrderDetails must reject a nil request")
	resp, err := h.GetV5OrderDetails(t.Context(), &V5OrderDetailsRequest{
		ContractCode: "BTC-USDT",
		OrderID:      "1343541341268738048",
		StartTime:    time.UnixMilli(1),
		EndTime:      time.UnixMilli(2),
		From:         "1",
		Limit:        100,
		Direction:    "next",
	})
	require.NoError(t, err, "GetV5OrderDetails must not error")
	require.Len(t, resp.Data, 1, "one execution detail must decode")
	assert.Equal(t, "100000032538647", resp.Data[0].TradeID, "trade ID should decode")
	assert.Equal(t, types.Number(31400), resp.Data[0].TradePrice, "trade price should decode")
	assert.Equal(t, types.Number(0.01884), resp.Data[0].TradeFee, "trade fee should decode")
	assert.False(t, resp.Data[0].CreatedTime.Time().IsZero(), "creation time should decode")
}

func TestGetV5OpenPositions(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/trade/position/opens", `{"code":200,"data":[{"contract_code":"BTC-USDT","volume":"2"}]}`, nil)
	resp, err := h.GetV5OpenPositions(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetV5OpenPositions must not error")
	require.Len(t, resp.Data, 1, "one position must decode")
	assert.Equal(t, types.Number(2), resp.Data[0].Volume, "volume should decode")
}

package htx

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceV5BatchOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.PlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", PositionSide: "long", Side: "invalid", Type: "post_only", Price: 1, Volume: 1}})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live batch rejection response must decode")
			require.Len(t, response.Acknowledgements, 1, "test order must have a rejection acknowledgement")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgements[0].Code, "invalid side should be rejected")
			assert.Empty(t, response.Acknowledgements[0].OrderID, "invalid side should never create an order")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/batch_orders", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	_, err := h.PlaceV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceV5BatchOrders must reject empty request")
	resp, err := h.PlaceV5BatchOrders(t.Context(), []*V5OrderRequest{{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "market", Volume: 1}})
	require.NoError(t, err, "PlaceV5BatchOrders must not error")
	require.Len(t, resp.Acknowledgements, 1, "one order acknowledgement must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")
}

func TestCancelV5BatchOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		const fabricatedOrderID = "9223372036854775807"
		response, err := h.CancelV5BatchOrders(t.Context(), &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{fabricatedOrderID}})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live batch rejection response must decode")
			require.Len(t, response.Acknowledgements, 1, "fabricated cancellation must have a rejection acknowledgement")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgements[0].Code, "fabricated order ID should be rejected")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_batch_orders", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	_, err := h.CancelV5BatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelV5BatchOrders must reject nil request")
	resp, err := h.CancelV5BatchOrders(t.Context(), &V5CancelBatchOrdersRequest{ContractCode: "BTC-USDT", OrderIDs: []string{"1"}})
	require.NoError(t, err, "CancelV5BatchOrders must not error")
	require.Len(t, resp.Acknowledgements, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")
}

func TestSetV5CancelAfter(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.SetV5CancelAfter(t.Context(), &V5CancelAfterRequest{Enabled: "invalid", Timeout: 60})
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "fabricated cancellation setting should be rejected")
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel-after", `{"code":200,"data":{"current_time":"1767145577000","trigger_time":"1767145637000"}}`, nil)
	_, err := h.SetV5CancelAfter(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetV5CancelAfter must reject nil request")
	resp, err := h.SetV5CancelAfter(t.Context(), &V5CancelAfterRequest{Enabled: "on", Timeout: 60})
	require.NoError(t, err, "SetV5CancelAfter must not error")
	assert.False(t, resp.Schedule.TriggerTime.Time().IsZero(), "trigger time should decode")
}

func TestCloseV5Position(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.CloseV5Position(t.Context(), &V5ClosePositionRequest{ContractCode: "INVALID-USDT", MarginMode: "cross", PositionSide: "long"})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "position close rejection must decode")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgement.Code, "fabricated contract should be rejected")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/position", `{"code":200,"data":{"order_id":"1"}}`, nil)
	_, err := h.CloseV5Position(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CloseV5Position must reject nil request")
	resp, err := h.CloseV5Position(t.Context(), &V5ClosePositionRequest{ContractCode: "BTC-USDT", MarginMode: "cross", PositionSide: "long"})
	require.NoError(t, err, "CloseV5Position must not error")
	assert.Equal(t, "1", resp.Acknowledgement.OrderID, "order ID should decode")
}

func TestCloseAllV5Positions(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.CloseAllV5Positions(t.Context())
		require.NoError(t, err, "live close-all must be accepted")
		require.NotNil(t, response, "close-all acknowledgement must decode")
		for _, ack := range response.Acknowledgements {
			assert.Contains(t, []uint64{0, 200}, ack.Code, "position close should report success")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/position_all", `{"code":200,"data":[{"order_id":"1"}]}`, nil)
	resp, err := h.CloseAllV5Positions(t.Context())
	require.NoError(t, err, "CloseAllV5Positions must not error")
	require.Len(t, resp.Acknowledgements, 1, "one close acknowledgement must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].OrderID, "order ID should decode")
}

func TestGetV5OrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		for _, mode := range []string{"cross", "isolated"} {
			t.Run(mode, func(t *testing.T) {
				t.Parallel()
				end := time.Now().UTC()
				query := V5OrderHistoryRequest{ContractCode: "BTC-USDT", MarginMode: mode, StartTime: end.Add(-24 * time.Hour), EndTime: end, Limit: 2}
				first, err := h.GetV5OrderHistory(t.Context(), &query)
				require.NoError(t, err, "live order history must be readable")
				require.NotNil(t, first, "live order history response must decode")
				for _, row := range first.Orders {
					assert.Equal(t, "BTC-USDT", row.ContractCode, "order should belong to the requested contract")
					assert.NotEmpty(t, row.OrderID, "historical order ID should be present")
				}
				if len(first.Orders) < 2 {
					t.Skip("The account needs at least two recent orders in this margin mode to verify pagination")
				}
				query.From, err = strconv.ParseUint(first.Orders[1].ID, 10, 64)
				require.NoError(t, err, "live order history cursor must be numeric")
				query.Direction = "next"
				second, err := h.GetV5OrderHistory(t.Context(), &query)
				require.NoError(t, err, "live order history must accept the returned cursor")
				require.NotNil(t, second, "second page must decode")
				for _, row := range second.Orders {
					assert.NotEqual(t, first.Orders[0].OrderID, row.OrderID, "subsequent page should advance beyond the first order")
				}
			})
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/history", `{"code":200,"data":[{"order_id":"1","volume":"2"}]}`, func(r *http.Request) {
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
	require.Len(t, resp.Orders, 1, "one historical order must decode")
	assert.Equal(t, types.Number(2), resp.Orders[0].Volume, "volume should decode")
}

func TestGetV5OrderDetails(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		end := time.Now().UTC()
		start := end.Add(-24 * time.Hour)
		history, err := h.GetV5OrderHistory(t.Context(), &V5OrderHistoryRequest{ContractCode: "BTC-USDT", MarginMode: "cross", States: "filled", StartTime: start, EndTime: end, Limit: 1})
		require.NoError(t, err, "live filled order must be discoverable")
		require.NotNil(t, history, "live order history response must decode")
		if len(history.Orders) == 0 {
			t.Skip("The account has no recent filled BTC-USDT cross-margin order")
		}
		id := history.Orders[0].OrderID
		response, err := h.GetV5OrderDetails(t.Context(), &V5OrderDetailsRequest{ContractCode: "BTC-USDT", OrderID: id, StartTime: start, EndTime: end, Limit: 10})
		require.NoError(t, err, "live execution details must be readable")
		require.NotNil(t, response, "live execution details response must decode")
		require.NotEmpty(t, response.Orders, "a filled order must have execution details")
		for _, row := range response.Orders {
			assert.Equal(t, id, row.OrderID, "execution should belong to the requested order")
			assert.NotEmpty(t, row.TradeID, "execution trade ID should be present")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/details", `{"code":200,"data":[{"id":"1124147771","contract_code":"BTC-USDT","order_id":"1343541341268738048","trade_id":"100000032538647","side":"sell","position_side":"short","order_type":"1","margin_mode":"cross","type":"limit","role":"TAKER","trade_price":"31400","trade_volume":"1","trade_turnover":"31.4","created_time":1740366817564,"updated_time":1740366817564,"order_source":"api","fee_currency":"USDT","trade_fee":"0.01884","deduction_price":"","profit":"0","contract_type":"swap"}]}`, func(r *http.Request) {
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
	require.Len(t, resp.Orders, 1, "one execution detail must decode")
	assert.Equal(t, "100000032538647", resp.Orders[0].TradeID, "trade ID should decode")
	assert.Equal(t, types.Number(31400), resp.Orders[0].TradePrice, "trade price should decode")
	assert.Equal(t, types.Number(0.01884), resp.Orders[0].TradeFee, "trade fee should decode")
	assert.True(t, currency.USDT.Equal(resp.Orders[0].FeeCurrency), "fee currency should decode as a single currency")
	assert.False(t, resp.Orders[0].CreatedTime.Time().IsZero(), "creation time should decode")
}

func TestGetV5OpenPositions(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.GetV5OpenPositions(t.Context(), btcusdtPair)
		require.NoError(t, err, "live open positions must be readable")
		require.NotNil(t, response, "live open positions response must decode")
		for _, position := range response.Positions {
			assert.Equal(t, "BTC-USDT", position.ContractCode, "positions should belong to the requested contract")
			assert.Contains(t, []string{"cross", "isolated"}, position.MarginMode, "position margin mode should be recognised")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/position/opens", `{"code":200,"data":[{"contract_code":"BTC-USDT","volume":"2"}]}`, nil)
	resp, err := h.GetV5OpenPositions(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetV5OpenPositions must not error")
	require.Len(t, resp.Positions, 1, "one position must decode")
	assert.Equal(t, types.Number(2), resp.Positions[0].Volume, "volume should decode")
}

func TestPlaceV5Order(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.PlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", PositionSide: "long", Side: "invalid", Type: "post_only", Price: 1, Volume: 1})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection acknowledgement must decode")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgement.Code, "invalid side should be rejected")
			assert.Empty(t, response.Acknowledgement.OrderID, "invalid side should never create an order")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/order", `{"code":200,"data":{"order_id":"123"}}`, nil)
	_, err := h.PlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceV5Order must reject nil request")
	resp, err := h.PlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "limit", Volume: types.Number(1)})
	require.NoError(t, err, "PlaceV5Order must not error")
	require.NotNil(t, resp, "decoded order response must be returned")
	assert.Equal(t, "123", resp.Acknowledgement.OrderID, "order ID should decode")
}

func TestCancelV5Order(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		const fabricatedOrderID = "9223372036854775807"
		response, err := h.CancelV5Order(t.Context(), btcusdtPair, fabricatedOrderID, "")
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live cancellation rejection must decode")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgement.Code, "fabricated order ID should be rejected")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_order", `{"code":200,"data":{"order_id":"123"}}`, nil)
	resp, err := h.CancelV5Order(t.Context(), btcusdtPair, "1", "")
	require.NoError(t, err, "CancelV5Order must not error")
	require.NotNil(t, resp, "decoded cancellation must be returned")
	assert.Equal(t, "123", resp.Acknowledgement.OrderID, "order ID should decode")
}

func TestCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.CancelAllV5Orders(t.Context(), currency.NewPair(currency.NewCode("INVALID"), currency.USDT), "buy", "long")
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live cancel-all rejection must decode")
			require.NotEmpty(t, response.Acknowledgements, "invalid contract must have a rejection acknowledgement")
			for _, ack := range response.Acknowledgements {
				assert.NotContains(t, []uint64{0, 200}, ack.Code, "fabricated contract should be rejected")
			}
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_all_orders", `{"code":200,"data":[{"order_id":"123"}]}`, nil)
	resp, err := h.CancelAllV5Orders(t.Context(), btcusdtPair, "buy", "long")
	require.NoError(t, err, "CancelAllV5Orders must not error")
	require.NotNil(t, resp, "decoded cancellations must be returned")
	require.Len(t, resp.Acknowledgements, 1, "one cancellation must decode")
	assert.Equal(t, "123", resp.Acknowledgements[0].OrderID, "order ID should decode")
}

func TestGetV5Order(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		for _, mode := range []string{"cross", "isolated"} {
			t.Run(mode, func(t *testing.T) {
				t.Parallel()
				open, err := h.GetV5OpenOrders(t.Context(), btcusdtPair, mode, "", "", 0, 1, "")
				require.NoError(t, err, "live open order must be discoverable")
				require.NotNil(t, open, "live open orders response must decode")
				if len(open.Orders) == 0 {
					t.Skip("The account has no open BTC-USDT order in this margin mode")
				}
				response, err := h.GetV5Order(t.Context(), btcusdtPair, mode, open.Orders[0].OrderID, "")
				require.NoError(t, err, "live order must be readable by ID")
				require.NotNil(t, response, "live order response must decode")
				assert.Equal(t, open.Orders[0].OrderID, response.Order.OrderID, "order lookup should retain the exchange order ID")
				assert.Equal(t, open.Orders[0].ClientOrderID, response.Order.ClientOrderID, "order lookup should retain the client order ID")
			})
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order", `{"code":200,"data":{"order_id":"123"}}`, nil)
	resp, err := h.GetV5Order(t.Context(), btcusdtPair, "cross", "1", "")
	require.NoError(t, err, "GetV5Order must not error")
	require.NotNil(t, resp, "decoded order must be returned")
	assert.Equal(t, "123", resp.Order.OrderID, "order ID should decode")
}

func TestGetV5OpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		for _, mode := range []string{"cross", "isolated"} {
			t.Run(mode, func(t *testing.T) {
				t.Parallel()
				response, err := h.GetV5OpenOrders(t.Context(), btcusdtPair, mode, "", "", 0, 10, "")
				require.NoError(t, err, "live open orders must be readable")
				require.NotNil(t, response, "live open orders response must decode")
				for _, row := range response.Orders {
					assert.Equal(t, "BTC-USDT", row.ContractCode, "open order should belong to the requested contract")
					assert.Equal(t, mode, row.MarginMode, "open order should belong to the requested margin mode")
					assert.NotEmpty(t, row.OrderID, "open order ID should be present")
				}
			})
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/opens", `{"code":200,"data":[{"order_id":"123"}]}`, nil)
	resp, err := h.GetV5OpenOrders(t.Context(), btcusdtPair, "cross", "", "", 1, 10, "next")
	require.NoError(t, err, "GetV5OpenOrders must not error")
	require.NotNil(t, resp, "decoded open orders must be returned")
	require.Len(t, resp.Orders, 1, "one open order must decode")
	assert.Equal(t, "123", resp.Orders[0].OrderID, "order ID should decode")
}

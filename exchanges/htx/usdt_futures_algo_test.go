package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceV5AlgoOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.PlaceV5AlgoOrder(t.Context(), &V5AlgoOrderRequest{ContractCode: "INVALID-USDT", MarginMode: "cross", PositionSide: "long", Side: "buy", Type: "trigger", Volume: 1, Price: 1, TriggerPrice: 2, TriggerPriceType: "last", OrderPriceType: "limit"})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "strategy rejection must decode")
			require.Len(t, response.Acknowledgements, 1, "strategy order must have one rejection acknowledgement")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgements[0].Code, "fabricated contract should be rejected")
			assert.Empty(t, response.Acknowledgements[0].AlgoID, "fabricated contract should never create an order")
		}
	})
	for _, priceMatch := range []string{"", "optimal_5"} {
		t.Run("price match "+priceMatch, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/algo/order", `{"code":200,"data":[{"code":200,"algo_id":"1"}]}`, func(r *http.Request) {
				var payload map[string]any
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "algo order request should decode") {
					return
				}
				if priceMatch == "" {
					assert.NotContains(t, payload, "price_match", "unset optional field should be omitted")
				} else {
					assert.Equal(t, priceMatch, payload["price_match"], "explicit price matching should be preserved")
				}
				assert.NotContains(t, payload, "price_type", "trigger order should omit the unsupported price_type field")
			})
			_, err := h.PlaceV5AlgoOrder(t.Context(), nil)
			require.ErrorIs(t, err, common.ErrNilPointer, "PlaceV5AlgoOrder must reject nil request")
			resp, err := h.PlaceV5AlgoOrder(t.Context(), &V5AlgoOrderRequest{
				ContractCode: "BTC-USDT",
				Type:         "trigger",
				PositionSide: "long",
				Side:         "buy",
				MarginMode:   "cross",
				Volume:       1,
				PriceMatch:   priceMatch,
			})
			require.NoError(t, err, "PlaceV5AlgoOrder must not error")
			require.Len(t, resp.Acknowledgements, 1, "one algo acknowledgement must decode")
			assert.Equal(t, "1", resp.Acknowledgements[0].AlgoID, "algo ID should decode")
			assert.Equal(t, uint64(200), resp.Acknowledgements[0].Code, "algo acknowledgement code should decode as unsigned")
		})
	}
}

func TestCancelV5AlgoOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		const fabricatedOrderID = "9223372036854775807"
		response, err := h.CancelV5AlgoOrders(t.Context(), []*V5CancelAlgoOrderRequest{{ContractCode: "BTC-USDT", AlgoID: fabricatedOrderID}})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "live venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "strategy cancellation rejection must decode")
			require.Len(t, response.Acknowledgements, 1, "strategy cancellation must have one rejection acknowledgement")
			assert.NotContains(t, []uint64{0, 200}, response.Acknowledgements[0].Code, "fabricated strategy ID should be rejected")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/algo/cancel_orders", `{"code":200,"data":[{"algo_id":"1"}]}`, nil)
	_, err := h.CancelV5AlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CancelV5AlgoOrders must reject empty request")
	resp, err := h.CancelV5AlgoOrders(t.Context(), []*V5CancelAlgoOrderRequest{{ContractCode: "BTC-USDT", AlgoID: "1"}})
	require.NoError(t, err, "CancelV5AlgoOrders must not error")
	require.Len(t, resp.Acknowledgements, 1, "one cancellation must decode")
	assert.Equal(t, "1", resp.Acknowledgements[0].AlgoID, "algo ID should decode")
}

func TestGetV5AlgoOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		open, err := h.GetV5OpenAlgoOrders(t.Context(), &V5OpenAlgoOrdersRequest{ContractCode: "BTC-USDT", Type: "trigger", Limit: 1})
		require.NoError(t, err, "live strategy order must be discoverable")
		require.NotNil(t, open, "live open strategy response must decode")
		if len(open.Orders) == 0 {
			t.Skip("The account has no open BTC-USDT trigger order")
		}
		id := open.Orders[0].AlgoID
		response, err := h.GetV5AlgoOrder(t.Context(), id, "", "trigger")
		require.NoError(t, err, "live strategy order must be readable by ID")
		require.NotNil(t, response, "live strategy order response must decode")
		require.Len(t, response.Orders, 1, "lookup by strategy ID must return one order")
		assert.Equal(t, id, response.Orders[0].AlgoID, "strategy lookup should retain the order ID")
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order", `{"code":200,"data":[{"algo_id":"1","volume":"2"}]}`, func(r *http.Request) {
		assert.Equal(t, "1", r.URL.Query().Get("algo_id"), "algo ID should be sent")
		assert.Equal(t, "trigger", r.URL.Query().Get("type"), "order type should be sent")
	})
	resp, err := h.GetV5AlgoOrder(t.Context(), "1", "", "trigger")
	require.NoError(t, err, "GetV5AlgoOrder must not error")
	require.Len(t, resp.Orders, 1, "one algo order must decode")
	assert.Equal(t, types.Number(2), resp.Orders[0].Volume, "volume should decode")
}

func TestGetV5OpenAlgoOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.GetV5OpenAlgoOrders(t.Context(), &V5OpenAlgoOrdersRequest{ContractCode: "BTC-USDT", Type: "trigger", Limit: 10})
		require.NoError(t, err, "live open strategy orders must be readable")
		require.NotNil(t, response, "live open strategy response must decode")
		for _, row := range response.Orders {
			assert.Equal(t, "BTC-USDT", row.ContractCode, "open strategy should belong to the requested contract")
			assert.NotEmpty(t, row.AlgoID, "strategy order ID should be present")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order/opens", `{"code":200,"data":[{"algo_id":"1"}]}`, nil)
	_, err := h.GetV5OpenAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5OpenAlgoOrders must reject nil request")
	resp, err := h.GetV5OpenAlgoOrders(t.Context(), &V5OpenAlgoOrdersRequest{
		ContractCode:  "BTC-USDT",
		AlgoID:        "1",
		ClientOrderID: "2",
		Type:          "trigger",
		From:          1,
		Limit:         10,
		Direction:     "next",
	})
	require.NoError(t, err, "GetV5OpenAlgoOrders must not error")
	require.Len(t, resp.Orders, 1, "one open algo order must decode")
	assert.Equal(t, "1", resp.Orders[0].AlgoID, "algo ID should decode")
}

func TestGetV5AlgoOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		end := time.Now().UTC()
		response, err := h.GetV5AlgoOrderHistory(t.Context(), &V5AlgoOrderHistoryRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Type: "trigger", StartTime: end.Add(-24 * time.Hour), EndTime: end, Limit: 10})
		require.NoError(t, err, "live strategy history must be readable")
		require.NotNil(t, response, "live strategy history response must decode")
		for _, row := range response.Orders {
			assert.Equal(t, "BTC-USDT", row.ContractCode, "historical strategy should belong to the requested contract")
			assert.NotEmpty(t, row.AlgoID, "strategy order ID should be present")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order/history", `{"code":200,"data":[{"algo_id":"1","actual_volume":"2"}]}`, func(r *http.Request) {
		assert.Equal(t, "trigger", r.URL.Query().Get("type"), "order type should be sent")
	})
	_, err := h.GetV5AlgoOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5AlgoOrderHistory must reject nil request")
	resp, err := h.GetV5AlgoOrderHistory(t.Context(), &V5AlgoOrderHistoryRequest{
		ContractCode: "BTC-USDT",
		MarginMode:   "cross",
		States:       "canceled",
		Type:         "trigger",
		StartTime:    time.UnixMilli(1),
		EndTime:      time.UnixMilli(2),
		From:         1,
		Limit:        10,
		Direction:    "next",
	})
	require.NoError(t, err, "GetV5AlgoOrderHistory must not error")
	require.Len(t, resp.Orders, 1, "one historical algo order must decode")
	assert.Equal(t, types.Number(2), resp.Orders[0].ActualVolume, "actual volume should decode")
}

package kucoin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPostFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PostFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostFuturesOrder must reject a nil request")
	_, err = e.PostFuturesOrder(t.Context(), &FuturesOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PostFuturesOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PostFuturesOrder(t.Context(), &FuturesOrderRequest{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", StopPrice: 123456, Size: 1, Price: 1000, Leverage: 1,
	})
	require.NoError(t, err, "PostFuturesOrder must not error")
	if mockTests {
		exp := &FuturesOrderResponse{
			OrderID:       "234125150956625920",
			ClientOrderID: "5bd6e9286d99522a52e458de",
		}
		assert.Equal(t, exp, result, "PostFuturesOrder should decode every field")
		return
	}
	require.NotNil(t, result, "PostFuturesOrder must return a response")
	assert.NotEmpty(t, result.OrderID, "PostFuturesOrder should return the order ID")
}

func TestFuturesOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FuturesOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &FuturesOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: "XBTUSDTM"}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "stop without stop price type", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Stop: "up", StopPrice: 123456}, err: errInvalidStopPriceType},
		{name: "stop without stop price", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Stop: "up", StopPriceType: "TP"}, err: limits.ErrPriceBelowMin},
		{name: "limit without price", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Size: 1}, err: limits.ErrPriceBelowMin},
		{name: "default type without price", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Size: 1}, err: limits.ErrPriceBelowMin},
		{name: "size and quantity", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Size: 1, Quantity: 0.001}, err: errSizeQuantityValueQuantityExclusive},
		{name: "quantity and value quantity", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Quantity: 0.001, ValueQuantity: 100}, err: errSizeQuantityValueQuantityExclusive},
		{name: "close order without side", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true}},
		{name: "stop limit", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Stop: "up", StopPriceType: "TP", StopPrice: 123456, Price: 1000, Size: 1}},
		{name: "market by value", req: &FuturesOrderRequest{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "sell", Symbol: "XBTUSDTM", Type: "market", ValueQuantity: 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestFuturesOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*FuturesOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&FuturesOrderRequest{}).Payload()
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "Payload must validate the request")

	for _, tc := range []struct {
		name string
		req  *FuturesOrderRequest
		exp  map[string]any
	}{
		{
			name: "size",
			req: &FuturesOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit",
				Remark: "order_remarks", Stop: "down", StopPriceType: "MP", StopPrice: 95000, ReduceOnly: true,
				CloseOrder: true, ForceHold: true, SelfTradePrevention: "CN", MarginMode: "ISOLATED", Price: 94000.5,
				Size: 1, TimeInForce: "GTC", PostOnly: true, PositionSide: "LONG",
			},
			exp: map[string]any{
				"clientOid": "5c52e11203aa677f33e493fb", "side": "buy", "symbol": "XBTUSDTM", "leverage": uint64(3),
				"type": "limit", "remark": "order_remarks", "stop": "down", "stopPriceType": "MP", "stopPrice": "95000",
				"reduceOnly": true, "closeOrder": true, "forceHold": true, "stp": "CN", "marginMode": "ISOLATED",
				"price": "94000.5", "size": uint64(1), "timeInForce": "GTC", "postOnly": true, "positionSide": "LONG",
			},
		},
		{
			name: "quantity",
			req:  &FuturesOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "sell", Symbol: "XBTUSDTM", Type: "market", Quantity: 0.002},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fc", "side": "sell", "symbol": "XBTUSDTM", "type": "market", "qty": "0.002"},
		},
		{
			name: "value quantity",
			req:  &FuturesOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fd", Side: "buy", Symbol: "XBTUSDTM", Type: "market", ValueQuantity: 150.5},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fd", "side": "buy", "symbol": "XBTUSDTM", "type": "market", "valueQty": "150.5"},
		},
		{
			name: "close order",
			req:  &FuturesOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fe", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fe", "symbol": "XBTUSDTM", "type": "market", "closeOrder": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := tc.req.Payload()
			require.NoError(t, err, "Payload must not error")
			assert.Equal(t, tc.exp, body, "Payload should send every documented parameter")
		})
	}
}

func TestCountSet(t *testing.T) {
	t.Parallel()
	assert.Zero(t, countSet(), "countSet should count nothing without conditions")
	assert.Equal(t, 2, countSet(true, false, true), "countSet should count the conditions that hold")
}

func TestPostFuturesOrderTest(t *testing.T) {
	t.Parallel()
	_, err := e.PostFuturesOrderTest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostFuturesOrderTest must reject a nil request")
	_, err = e.PostFuturesOrderTest(t.Context(), &FuturesOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PostFuturesOrderTest must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PostFuturesOrderTest(t.Context(), &FuturesOrderRequest{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Remark: "10",
		Size: 1, Leverage: 1,
	})
	require.NoError(t, err, "PostFuturesOrderTest must not error")
	if mockTests {
		exp := &FuturesOrderResponse{
			OrderID:       "234125150956625921",
			ClientOrderID: "5bd6e9286d99522a52e458de",
		}
		assert.Equal(t, exp, result, "PostFuturesOrderTest should decode every field")
		return
	}
	require.NotNil(t, result, "PostFuturesOrderTest must return a response")
	assert.NotEmpty(t, result.OrderID, "PostFuturesOrderTest should return an order ID")
}

func TestFuturesOrderListBody(t *testing.T) {
	t.Parallel()
	_, err := futuresOrderListBody(nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "futuresOrderListBody must reject no orders")
	_, err = futuresOrderListBody([]FuturesOrderRequest{{ClientOrderID: "5c52e11203aa677f33e491", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Size: 1}, {}})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "futuresOrderListBody must validate every order")
	assert.ErrorContains(t, err, "order 1", "futuresOrderListBody should name the order that failed validation")
	got, err := futuresOrderListBody([]FuturesOrderRequest{
		{ClientOrderID: "5c52e11203aa677f33e491", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Size: 1},
		{ClientOrderID: "5c52e11203aa677f33e492", Side: "sell", Symbol: "XBTUSDTM", Type: "limit", Price: 32150, Size: 2, Leverage: 3},
	})
	require.NoError(t, err, "futuresOrderListBody must not error")
	exp := []map[string]any{
		{"clientOid": "5c52e11203aa677f33e491", "side": "buy", "symbol": "XBTUSDTM", "type": "market", "size": uint64(1)},
		{"clientOid": "5c52e11203aa677f33e492", "side": "sell", "symbol": "XBTUSDTM", "type": "limit", "price": "32150", "size": uint64(2), "leverage": uint64(3)},
	}
	assert.Equal(t, exp, got, "futuresOrderListBody should return each order's Add Order body")
}

func TestPlaceMultipleFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMultipleFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceMultipleFuturesOrders must reject an empty batch")
	_, err = e.PlaceMultipleFuturesOrders(t.Context(), []FuturesOrderRequest{{ClientOrderID: "5c52e11203aa677f33e491", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Size: 1}, {}})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PlaceMultipleFuturesOrders must validate every order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceMultipleFuturesOrders(t.Context(), []FuturesOrderRequest{
		{ClientOrderID: "5c52e11203aa677f33e491", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Price: 2150, Size: 2, Leverage: 1},
		{ClientOrderID: "5c52e11203aa677f33e492", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Price: 32150, Size: 2, Leverage: 1},
	})
	require.NoError(t, err, "PlaceMultipleFuturesOrders must not error")
	if mockTests {
		exp := []FuturesBatchOrderResponse{
			{
				ClientOrderID: "5c52e11203aa677f33e491",
				Symbol:        "XBTUSDTM",
				Code:          "300012",
				Message:       "Order price cannot be lower than 30000",
			},
			{
				OrderID:       "235919387855482880",
				ClientOrderID: "5c52e11203aa677f33e492",
				Symbol:        "XBTUSDTM",
				Code:          "200000",
				Message:       "success",
			},
		}
		assert.Equal(t, exp, result, "PlaceMultipleFuturesOrders should decode every order's outcome")
		return
	}
	assert.Len(t, result, 2, "PlaceMultipleFuturesOrders should return an outcome per order")
}

func TestPostFuturesTakeProfitStopLossOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PostFuturesTakeProfitStopLossOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostFuturesTakeProfitStopLossOrder must reject a nil request")
	_, err = e.PostFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "PostFuturesTakeProfitStopLossOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PostFuturesTakeProfitStopLossOrder(t.Context(), &FuturesTakeProfitStopLossOrderRequest{
		ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit",
		Remark: "order remarks", MarginMode: "ISOLATED", PositionSide: "BOTH", Price: 100000, Size: 1,
		TimeInForce: "GTC", TriggerStopUpPrice: 110000, TriggerStopDownPrice: 90000, StopPriceType: "TP",
	})
	require.NoError(t, err, "PostFuturesTakeProfitStopLossOrder must not error")
	if mockTests {
		exp := &FuturesOrderResponse{
			OrderID:       "234125150956625922",
			ClientOrderID: "5c52e11203aa677f33e493fb",
		}
		assert.Equal(t, exp, result, "PostFuturesTakeProfitStopLossOrder should decode every field")
		return
	}
	require.NotNil(t, result, "PostFuturesTakeProfitStopLossOrder must return a response")
	assert.NotEmpty(t, result.OrderID, "PostFuturesTakeProfitStopLossOrder should return the order ID")
}

func TestFuturesTakeProfitStopLossOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FuturesTakeProfitStopLossOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &FuturesTakeProfitStopLossOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no side", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Symbol: "XBTUSDTM"}, err: order.ErrSideIsInvalid},
		{name: "no symbol", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy"}, err: currency.ErrSymbolStringEmpty},
		{name: "limit without price", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Size: 1}, err: limits.ErrPriceBelowMin},
		{name: "default type without price", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Size: 1}, err: limits.ErrPriceBelowMin},
		{name: "size and value quantity", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Size: 1, ValueQuantity: 100}, err: errSizeQuantityValueQuantityExclusive},
		{name: "close order without side", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true, TriggerStopDownPrice: 90000}},
		{name: "limit", req: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Price: 100000, Size: 1, TriggerStopUpPrice: 110000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestFuturesTakeProfitStopLossOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*FuturesTakeProfitStopLossOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&FuturesTakeProfitStopLossOrderRequest{}).Payload()
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "Payload must validate the request")

	for _, tc := range []struct {
		name string
		req  *FuturesTakeProfitStopLossOrderRequest
		exp  map[string]any
	}{
		{
			name: "size",
			req: &FuturesTakeProfitStopLossOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit",
				Remark: "order remarks", StopPriceType: "TP", ReduceOnly: true, CloseOrder: true, ForceHold: true,
				SelfTradePrevention: "CO", MarginMode: "ISOLATED", Price: 100000, Size: 1, TimeInForce: "GTC",
				PostOnly: true, TriggerStopUpPrice: 110000, TriggerStopDownPrice: 90000.5, PositionSide: "BOTH",
			},
			exp: map[string]any{
				"clientOid": "5c52e11203aa677f33e493fb", "side": "buy", "symbol": "XBTUSDTM", "leverage": uint64(3),
				"type": "limit", "remark": "order remarks", "stopPriceType": "TP", "reduceOnly": true, "closeOrder": true,
				"forceHold": true, "stp": "CO", "marginMode": "ISOLATED", "price": "100000", "size": uint64(1),
				"timeInForce": "GTC", "postOnly": true, "triggerStopUpPrice": "110000", "triggerStopDownPrice": "90000.5",
				"positionSide": "BOTH",
			},
		},
		{
			name: "quantity",
			req:  &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fc", Side: "sell", Symbol: "XBTUSDTM", Type: "market", Quantity: 0.002, TriggerStopDownPrice: 90000},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fc", "side": "sell", "symbol": "XBTUSDTM", "type": "market", "qty": "0.002", "triggerStopDownPrice": "90000"},
		},
		{
			name: "value quantity",
			req:  &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fd", Side: "buy", Symbol: "XBTUSDTM", Type: "market", ValueQuantity: 150.5, TriggerStopUpPrice: 110000},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fd", "side": "buy", "symbol": "XBTUSDTM", "type": "market", "valueQty": "150.5", "triggerStopUpPrice": "110000"},
		},
		{
			name: "close order",
			req:  &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "5c52e11203aa677f33e493fe", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true, TriggerStopDownPrice: 90000},
			exp:  map[string]any{"clientOid": "5c52e11203aa677f33e493fe", "symbol": "XBTUSDTM", "type": "market", "closeOrder": true, "triggerStopDownPrice": "90000"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := tc.req.Payload()
			require.NoError(t, err, "Payload must not error")
			assert.Equal(t, tc.exp, body, "Payload should send every documented parameter")
		})
	}
}

func TestCancelFuturesOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFuturesOrderByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesOrderByOrderID(t.Context(), "235303670076489728")
	require.NoError(t, err, "CancelFuturesOrderByOrderID must not error")
	if mockTests {
		exp := &FuturesCancelledOrdersResponse{
			CancelledOrderIDs: []string{
				"235303670076489728",
			},
		}
		assert.Equal(t, exp, result, "CancelFuturesOrderByOrderID should decode every field")
		return
	}
	require.NotNil(t, result, "CancelFuturesOrderByOrderID must return a response")
	assert.NotEmpty(t, result.CancelledOrderIDs, "CancelFuturesOrderByOrderID should return the cancelled order ID")
}

func TestCancelFuturesOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrderByClientOrderID(t.Context(), "", "5bd6e9286d99522a52e458de")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelFuturesOrderByClientOrderID must reject an empty symbol")
	_, err = e.CancelFuturesOrderByClientOrderID(t.Context(), "XBTUSDTM", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "CancelFuturesOrderByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesOrderByClientOrderID(t.Context(), "XBTUSDTM", "5bd6e9286d99522a52e458de")
	require.NoError(t, err, "CancelFuturesOrderByClientOrderID must not error")
	if mockTests {
		exp := &FuturesCancelledClientOrderResponse{
			ClientOrderID: "5bd6e9286d99522a52e458de",
		}
		assert.Equal(t, exp, result, "CancelFuturesOrderByClientOrderID should decode every field")
		return
	}
	require.NotNil(t, result, "CancelFuturesOrderByClientOrderID must return a response")
	assert.Equal(t, "5bd6e9286d99522a52e458de", result.ClientOrderID, "CancelFuturesOrderByClientOrderID should return the client order ID")
}

func TestBatchCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchCancelFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BatchCancelFuturesOrders must reject a nil request")
	_, err = e.BatchCancelFuturesOrders(t.Context(), &BatchCancelFuturesOrdersRequest{})
	require.ErrorIs(t, err, errBatchCancelOrderIDsRequired, "BatchCancelFuturesOrders must reject a request without orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchCancelFuturesOrders(t.Context(), &BatchCancelFuturesOrdersRequest{OrderIDs: []string{"250445104152670209", "250445181751463936"}})
	require.NoError(t, err, "BatchCancelFuturesOrders must not error")
	if mockTests {
		exp := []FuturesBatchCancelResponse{
			{
				OrderID: "250445104152670209",
				Code:    "200",
				Message: "success",
			},
			{
				OrderID: "250445181751463936",
				Code:    "100004",
				Message: "Order is in not cancelable state",
			},
		}
		assert.Equal(t, exp, result, "BatchCancelFuturesOrders should decode every order's outcome")
		return
	}
	assert.Len(t, result, 2, "BatchCancelFuturesOrders should return an outcome per order")
}

// TestBatchCancelFuturesOrdersByClientOrderID cancels by client order ID against a local server: the recorded mock
// matches DELETE requests by their query alone, so it cannot tell this body from one that cancels by order ID
func TestBatchCancelFuturesOrdersByClientOrderID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "the request method should be DELETE")
		assert.Equal(t, "/v1/orders/multi-cancel", r.URL.Path, "the request path should be Batch Cancel Orders")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		assert.JSONEq(t, `{"clientOidsList":[{"clientOid":"5c52e11203aa1677f133e493fb","symbol":"XRPUSDTM"},{"clientOid":"5c52214e112","symbol":"ETHUSDTM"}]}`, string(body), "the request body should list the client orders")
		_, err = io.WriteString(w, `{"code":"200000","data":[{"orderId":null,"clientOid":"5c52e11203aa1677f133e493fb","code":"200","msg":"success"},{"orderId":null,"clientOid":"5c52214e112","code":"200","msg":"success"}]}`)
		assert.NoError(t, err, "writing the response should not error")
	}))
	t.Cleanup(server.Close)

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "SetRunningURL must not error")

	result, err := ex.BatchCancelFuturesOrders(t.Context(), &BatchCancelFuturesOrdersRequest{ClientOrders: []BatchCancelFuturesClientOrder{
		{Symbol: "XRPUSDTM", ClientOrderID: "5c52e11203aa1677f133e493fb"},
		{Symbol: "ETHUSDTM", ClientOrderID: "5c52214e112"},
	}})
	require.NoError(t, err, "BatchCancelFuturesOrders must not error")
	exp := []FuturesBatchCancelResponse{
		{ClientOrderID: "5c52e11203aa1677f133e493fb", Code: "200", Message: "success"},
		{ClientOrderID: "5c52214e112", Code: "200", Message: "success"},
	}
	assert.Equal(t, exp, result, "BatchCancelFuturesOrders should decode every order's outcome")
}

func TestBatchCancelFuturesOrdersRequestPayload(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BatchCancelFuturesOrdersRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no orders", req: &BatchCancelFuturesOrdersRequest{}, err: errBatchCancelOrderIDsRequired},
		{name: "empty order ID", req: &BatchCancelFuturesOrdersRequest{OrderIDs: []string{"250445104152670209", ""}}, err: order.ErrOrderIDNotSet},
		{name: "client order without symbol", req: &BatchCancelFuturesOrdersRequest{ClientOrders: []BatchCancelFuturesClientOrder{{ClientOrderID: "5c52214e112"}}}, err: currency.ErrSymbolStringEmpty},
		{name: "client order without client order ID", req: &BatchCancelFuturesOrdersRequest{ClientOrders: []BatchCancelFuturesClientOrder{{Symbol: "ETHUSDTM"}}}, err: order.ErrClientOrderIDMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.req.Payload()
			assert.ErrorIs(t, err, tc.err, "Payload should return the expected error")
		})
	}

	r := &BatchCancelFuturesOrdersRequest{
		OrderIDs:     []string{"250445104152670209", "250445181751463936"},
		ClientOrders: []BatchCancelFuturesClientOrder{{Symbol: "XRPUSDTM", ClientOrderID: "5c52e11203aa1677f133e493fb"}},
	}
	exp := map[string]any{
		"orderIdsList":   []string{"250445104152670209", "250445181751463936"},
		"clientOidsList": []map[string]string{{"symbol": "XRPUSDTM", "clientOid": "5c52e11203aa1677f133e493fb"}},
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")
}

func TestCancelAllFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "CancelAllFuturesOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllFuturesOrders(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "CancelAllFuturesOrders must not error")
	if mockTests {
		exp := &FuturesCancelledOrdersResponse{
			CancelledOrderIDs: []string{
				"235919172150824960",
				"235919172150824961",
			},
		}
		assert.Equal(t, exp, result, "CancelAllFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllFuturesOrders should return a response")
}

func TestCancelAllFuturesStopOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err := e.CancelAllFuturesStopOrders(cancelledContext(t), "")
	require.ErrorIs(t, err, context.Canceled, "CancelAllFuturesStopOrders must return the request error")
	result, err := e.CancelAllFuturesStopOrders(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "CancelAllFuturesStopOrders must not error")
	if mockTests {
		exp := &FuturesCancelledOrdersResponse{
			CancelledOrderIDs: []string{
				"235919172150824962",
			},
		}
		assert.Equal(t, exp, result, "CancelAllFuturesStopOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllFuturesStopOrders should return a response")
}

func TestGetFuturesOrderDetailsByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderDetailsByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFuturesOrderDetailsByOrderID must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrderDetailsByOrderID(t.Context(), "236655147005071361")
	require.NoError(t, err, "GetFuturesOrderDetailsByOrderID must not error")
	if mockTests {
		exp := &FuturesOrderDetailResponse{
			ID:                  "236655147005071361",
			Symbol:              "XBTUSDTM",
			Type:                "limit",
			Side:                "buy",
			Price:               65000.1,
			Size:                3,
			Value:               195.0003,
			DealValue:           65.0001,
			DealSize:            1,
			SelfTradePrevention: "CN",
			Stop:                "down",
			StopPriceType:       "TP",
			StopTriggered:       true,
			StopPrice:           65500,
			TimeInForce:         "GTC",
			Leverage:            3,
			ForceHold:           true,
			ClientOrderID:       "5c52e11203aa677f33e493fb",
			Remark:              "order_remarks",
			Tags:                "api",
			IsActive:            true,
			CreatedAt:           types.Time(time.UnixMilli(1729236185949)),
			UpdatedAt:           types.Time(time.UnixMilli(1729236190949)),
			OrderTime:           types.Time(time.UnixMicro(1729236185885648)),
			SettleCurrency:      currency.USDT,
			MarginMode:          "ISOLATED",
			AverageDealPrice:    65000.1,
			FilledSize:          1,
			FilledValue:         65.0001,
			Status:              "open",
			PositionSide:        "BOTH",
		}
		assert.Equal(t, exp, result, "GetFuturesOrderDetailsByOrderID should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesOrderDetailsByOrderID must return the order")
	assert.Equal(t, "236655147005071361", result.ID, "GetFuturesOrderDetailsByOrderID should return the requested order")
}

func TestGetFuturesOrderDetailsByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderDetailsByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "GetFuturesOrderDetailsByClientOrderID must reject an empty client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrderDetailsByClientOrderID(t.Context(), "5c52e11203aa677f33e493fc")
	require.NoError(t, err, "GetFuturesOrderDetailsByClientOrderID must not error")
	if mockTests {
		exp := &FuturesOrderDetailResponse{
			ID:                  "250444645610336256",
			Symbol:              "XRPUSDTM",
			Type:                "limit",
			Side:                "sell",
			Price:               0.6,
			Size:                10,
			Value:               60,
			DealValue:           30,
			DealSize:            5,
			SelfTradePrevention: "CO",
			TimeInForce:         "GTC",
			PostOnly:            true,
			Leverage:            5,
			ClientOrderID:       "5c52e11203aa677f33e493fc",
			Remark:              "order_remarks",
			CancelExist:         true,
			CreatedAt:           types.Time(time.UnixMilli(1732523858568)),
			UpdatedAt:           types.Time(time.UnixMilli(1732524858568)),
			EndAt:               types.Time(time.UnixMilli(1732524858568)),
			OrderTime:           types.Time(time.Unix(0, 1732523858550892300)),
			SettleCurrency:      currency.USDT,
			MarginMode:          "CROSS",
			AverageDealPrice:    0.6,
			FilledSize:          5,
			FilledValue:         30,
			Status:              "done",
			ReduceOnly:          true,
			PositionSide:        "LONG",
		}
		assert.Equal(t, exp, result, "GetFuturesOrderDetailsByClientOrderID should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesOrderDetailsByClientOrderID must return the order")
	assert.Equal(t, "5c52e11203aa677f33e493fc", result.ClientOrderID, "GetFuturesOrderDetailsByClientOrderID should return the requested order")
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesOrders must reject a nil request")
	_, err = e.GetFuturesOrders(t.Context(), &GetFuturesOrdersRequest{StartAt: time.UnixMilli(1727740800000), EndAt: time.UnixMilli(1727654400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetFuturesOrders(cancelledContext(t), &GetFuturesOrdersRequest{})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesOrders must return the request error")
	result, err := e.GetFuturesOrders(t.Context(), &GetFuturesOrdersRequest{
		Status: "done", Symbol: "XBTUSDTM", Side: "buy", Type: "limit", StartAt: time.UnixMilli(1727654400000),
		EndAt: time.UnixMilli(1727740800000), CurrentPage: 1, PageSize: 50,
	})
	require.NoError(t, err, "GetFuturesOrders must not error")
	if mockTests {
		exp := &FuturesOrdersResponse{
			CurrentPage: 1,
			PageSize:    50,
			TotalNumber: 2,
			TotalPage:   1,
			Items: []FuturesOrderDetailResponse{
				{
					ID:               "230181737576050688",
					Symbol:           "XBTUSDTM",
					Type:             "limit",
					Side:             "buy",
					Price:            62000,
					Size:             10,
					Value:            620,
					DealValue:        620,
					DealSize:         10,
					TimeInForce:      "GTC",
					Iceberg:          true,
					Leverage:         2,
					VisibleSize:      2,
					ClientOrderID:    "5a80bd847f1811ef8a7faa665a37b3d7",
					Remark:           "order_remarks",
					CreatedAt:        types.Time(time.UnixMilli(1727692804813)),
					UpdatedAt:        types.Time(time.UnixMilli(1727693104813)),
					EndAt:            types.Time(time.UnixMilli(1727693104813)),
					OrderTime:        types.Time(time.UnixMicro(1727692804808418)),
					SettleCurrency:   currency.USDT,
					MarginMode:       "ISOLATED",
					AverageDealPrice: 62000,
					FilledSize:       10,
					FilledValue:      620,
					Status:           "done",
					PositionSide:     "BOTH",
				},
				{
					ID:                  "230181737576050689",
					Symbol:              "XBTUSDTM",
					Type:                "limit",
					Side:                "buy",
					Price:               61500,
					Size:                4,
					Value:               246,
					DealValue:           61.5,
					DealSize:            1,
					SelfTradePrevention: "CB",
					TimeInForce:         "GTC",
					Hidden:              true,
					Leverage:            2,
					ClientOrderID:       "5a80bd847f1811ef8a7faa665a37b3d8",
					Remark:              "order_remarks",
					CancelExist:         true,
					CreatedAt:           types.Time(time.Unix(1727700000, 0)),
					UpdatedAt:           types.Time(time.Unix(1727710000, 0)),
					EndAt:               types.Time(time.Unix(1727710000, 0)),
					OrderTime:           types.Time(time.UnixMicro(1727699999998123)),
					SettleCurrency:      currency.USDT,
					MarginMode:          "ISOLATED",
					AverageDealPrice:    61500,
					FilledSize:          1,
					FilledValue:         61.5,
					Status:              "done",
					PositionSide:        "BOTH",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesOrders should return a page")
}

func TestGetFuturesRecentCompletedOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetFuturesRecentCompletedOrders(cancelledContext(t), "")
	require.ErrorIs(t, err, context.Canceled, "GetFuturesRecentCompletedOrders must return the request error")
	result, err := e.GetFuturesRecentCompletedOrders(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesRecentCompletedOrders must not error")
	if mockTests {
		exp := []FuturesOrderDetailResponse{
			{
				ID:               "236387137732231168",
				Symbol:           "XBTUSDTM",
				Type:             "market",
				Side:             "sell",
				Size:             1,
				Value:            67.4309,
				DealValue:        67.4309,
				DealSize:         1,
				TimeInForce:      "GTC",
				Leverage:         3,
				CloseOrder:       true,
				ClientOrderID:    "16698fe6-2746-4aeb-a7fa-61f633ab6090",
				Remark:           "close_position",
				CreatedAt:        types.Time(time.UnixMilli(1729172287496)),
				UpdatedAt:        types.Time(time.UnixMilli(1729172287568)),
				EndAt:            types.Time(time.UnixMilli(1729172287568)),
				OrderTime:        types.Time(time.Unix(0, 1729172287496950800)),
				SettleCurrency:   currency.USDT,
				MarginMode:       "ISOLATED",
				AverageDealPrice: 67430.9,
				FilledSize:       1,
				FilledValue:      67.4309,
				Status:           "done",
				ReduceOnly:       true,
				PositionSide:     "BOTH",
			},
			{
				ID:               "236317213710184449",
				Symbol:           "XBTUSDTM",
				Type:             "market",
				Side:             "buy",
				Size:             1,
				Value:            67.445,
				DealValue:        67.445,
				DealSize:         1,
				Stop:             "up",
				StopPriceType:    "MP",
				StopTriggered:    true,
				StopPrice:        67400,
				TimeInForce:      "GTC",
				Leverage:         3,
				ForceHold:        true,
				ClientOrderID:    "5c52e11203aa677f33e493fe",
				Remark:           "breakout",
				CreatedAt:        types.Time(time.UnixMilli(1729155587873)),
				UpdatedAt:        types.Time(time.UnixMilli(1729155587946)),
				EndAt:            types.Time(time.UnixMilli(1729155587946)),
				OrderTime:        types.Time(time.UnixMicro(1729155587873332)),
				SettleCurrency:   currency.USDT,
				MarginMode:       "ISOLATED",
				AverageDealPrice: 67445,
				FilledSize:       1,
				FilledValue:      67.445,
				Status:           "done",
				PositionSide:     "BOTH",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesRecentCompletedOrders should decode every field")
		return
	}
	for i := range result {
		assert.Equalf(t, "XBTUSDTM", result[i].Symbol, "GetFuturesRecentCompletedOrders should return orders of the requested contract, order %d", i)
	}
}

func TestGetUntriggeredFuturesStopOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUntriggeredFuturesStopOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUntriggeredFuturesStopOrders must reject a nil request")
	_, err = e.GetUntriggeredFuturesStopOrders(t.Context(), &GetUntriggeredFuturesStopOrdersRequest{StartAt: time.UnixMilli(1727740800000), EndAt: time.UnixMilli(1727654400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUntriggeredFuturesStopOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUntriggeredFuturesStopOrders(t.Context(), &GetUntriggeredFuturesStopOrdersRequest{
		Symbol: "XBTUSDTM", Side: "sell", Type: "limit", StartAt: time.UnixMilli(1727654400000),
		EndAt: time.UnixMilli(1727740800000), CurrentPage: 1, PageSize: 50,
	})
	require.NoError(t, err, "GetUntriggeredFuturesStopOrders must not error")
	if mockTests {
		exp := &FuturesOrdersResponse{
			CurrentPage: 1,
			PageSize:    50,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []FuturesOrderDetailResponse{
				{
					ID:                  "230181737576050690",
					Symbol:              "XBTUSDTM",
					Type:                "limit",
					Side:                "sell",
					Price:               70000,
					Size:                2,
					Value:               140,
					SelfTradePrevention: "CN",
					Stop:                "up",
					StopPriceType:       "MP",
					StopPrice:           70500,
					TimeInForce:         "GTC",
					Leverage:            3,
					ClientOrderID:       "5c52e11203aa677f33e493fd",
					Remark:              "take_profit",
					IsActive:            true,
					CreatedAt:           types.Time(time.UnixMilli(1727692804813)),
					UpdatedAt:           types.Time(time.UnixMilli(1727692804813)),
					OrderTime:           types.Time(time.UnixMicro(1727692804808418)),
					SettleCurrency:      currency.USDT,
					MarginMode:          "CROSS",
					Status:              "open",
					ReduceOnly:          true,
					PositionSide:        "BOTH",
				},
			},
		}
		assert.Equal(t, exp, result, "GetUntriggeredFuturesStopOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUntriggeredFuturesStopOrders should return a page")
}

func TestGetFuturesOpenOrderStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOpenOrderStats(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesOpenOrderStats must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOpenOrderStats(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesOpenOrderStats must not error")
	if mockTests {
		exp := &FuturesOpenOrderStatsResponse{
			OpenOrderBuySize:  2,
			OpenOrderSellSize: 1,
			OpenOrderBuyCost:  130.0002,
			OpenOrderSellCost: 70,
			SettleCurrency:    currency.USDT,
		}
		assert.Equal(t, exp, result, "GetFuturesOpenOrderStats should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesOpenOrderStats must return a response")
	assert.False(t, result.SettleCurrency.IsEmpty(), "GetFuturesOpenOrderStats should return the settlement currency")
}

func TestGetFuturesRecentFills(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetFuturesRecentFills(cancelledContext(t), "")
	require.ErrorIs(t, err, context.Canceled, "GetFuturesRecentFills must return the request error")
	result, err := e.GetFuturesRecentFills(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesRecentFills must not error")
	if mockTests {
		exp := []FuturesRecentFillResponse{
			{
				Symbol:         "XBTUSDTM",
				TradeID:        "1784277229880",
				OrderID:        "236317213710184449",
				Side:           "buy",
				Liquidity:      "taker",
				ForceTaker:     true,
				Price:          67430.9,
				Size:           1,
				Value:          67.4309,
				OpenFeePay:     0.04045854,
				Stop:           "up",
				FeeRate:        0.0006,
				FeeCurrency:    currency.USDT,
				TradeTime:      types.Time(time.UnixMilli(1729155616320)),
				MarginMode:     "ISOLATED",
				DisplayType:    "market",
				Fee:            0.04045854,
				SettleCurrency: currency.USDT,
				OrderType:      "market",
				TradeType:      "trade",
				CreatedAt:      types.Time(time.UnixMilli(1729155616493)),
				PositionSide:   "BOTH",
			},
			{
				Symbol:         "XBTUSDTM",
				TradeID:        "1784277132002",
				OrderID:        "236387137732231168",
				Side:           "sell",
				Liquidity:      "maker",
				Price:          67445,
				Size:           1,
				Value:          67.445,
				CloseFeePay:    0.013489,
				FeeRate:        0.0002,
				FixFee:         0.0001,
				FeeCurrency:    currency.USDT,
				TradeTime:      types.Time(time.UnixMilli(1729155587944)),
				SubTradeType:   "trade",
				MarginMode:     "CROSS",
				DisplayType:    "limit",
				Fee:            0.013489,
				SettleCurrency: currency.USDT,
				OrderType:      "limit",
				TradeType:      "trade",
				CreatedAt:      types.Time(time.UnixMilli(1729155588104)),
				PositionSide:   "LONG",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesRecentFills should decode every field")
		return
	}
	for i := range result {
		assert.Equalf(t, "XBTUSDTM", result[i].Symbol, "GetFuturesRecentFills should return fills of the requested contract, fill %d", i)
	}
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFills(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesFills must reject a nil request")
	_, err = e.GetFuturesFills(t.Context(), &GetFuturesFillsRequest{StartAt: time.UnixMilli(1740700800000), EndAt: time.UnixMilli(1740614400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesFills must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetFuturesFills(cancelledContext(t), &GetFuturesFillsRequest{})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesFills must return the request error")
	result, err := e.GetFuturesFills(t.Context(), &GetFuturesFillsRequest{
		OrderID: "284486580251463680", Symbol: "XBTUSDTM", Side: "buy", Type: "market", TradeTypes: []string{"trade", "liquid"},
		StartAt: time.UnixMilli(1740614400000), EndAt: time.UnixMilli(1740700800000), CurrentPage: 1, PageSize: 50,
	})
	require.NoError(t, err, "GetFuturesFills must not error")
	if mockTests {
		exp := &FuturesFillsResponse{
			CurrentPage: 1,
			PageSize:    50,
			TotalNumber: 2,
			TotalPage:   1,
			Items: []FuturesFill{
				{
					Symbol:         "XBTUSDTM",
					TradeID:        "1828954878212",
					OrderID:        "284486580251463680",
					Side:           "buy",
					Liquidity:      "taker",
					Price:          86275.1,
					Size:           1,
					Value:          86.2751,
					OpenFeePay:     0.05176506,
					Stop:           "up",
					FeeRate:        0.0006,
					FeeCurrency:    currency.USDT,
					TradeTime:      types.Time(time.UnixMilli(1740640088244)),
					MarginMode:     "CROSS",
					SettleCurrency: currency.USDT,
					DisplayType:    "market",
					Fee:            0.05176506,
					OrderType:      "market",
					TradeType:      "trade",
					CreatedAt:      types.Time(time.UnixMilli(1740640088427)),
					OpenFeeTaxPay:  0.00517651,
					PositionSide:   "BOTH",
				},
				{
					Symbol:         "XBTUSDTM",
					TradeID:        "1828954878213",
					OrderID:        "284486580251463680",
					Side:           "buy",
					Liquidity:      "taker",
					ForceTaker:     true,
					Price:          86276,
					Size:           2,
					Value:          172.552,
					CloseFeePay:    0.1035312,
					Stop:           "up",
					FeeRate:        0.0006,
					FixFee:         0.0001,
					FeeCurrency:    currency.USDT,
					TradeTime:      types.Time(time.UnixMilli(1740640088245)),
					SubTradeType:   "trade",
					MarginMode:     "CROSS",
					SettleCurrency: currency.USDT,
					DisplayType:    "market",
					Fee:            0.1035312,
					OrderType:      "market",
					TradeType:      "trade",
					CreatedAt:      types.Time(time.UnixMilli(1740640088428)),
					CloseFeeTaxPay: 0.01035312,
					PositionSide:   "BOTH",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesFills should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesFills should return a page")
}

func TestFuturesOrderListParams(t *testing.T) {
	t.Parallel()
	_, err := futuresOrderListParams("", "", "", time.UnixMilli(1727740800000), time.UnixMilli(1727654400000), 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresOrderListParams must reject a start after the end")

	params, err := futuresOrderListParams("", "", "", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err, "futuresOrderListParams must not error without parameters")
	assert.Empty(t, params, "futuresOrderListParams should omit unset parameters")

	params, err = futuresOrderListParams("", "", "", time.UnixMilli(1727654400000), time.Time{}, 0, 0)
	require.NoError(t, err, "futuresOrderListParams must not error with only a start")
	assert.Equal(t, url.Values{"startAt": {"1727654400000"}}, params, "futuresOrderListParams should send a start without an end")

	params, err = futuresOrderListParams("XBTUSDTM", "buy", "limit", time.UnixMilli(1727654400000), time.UnixMilli(1727740800000), 2, 100)
	require.NoError(t, err, "futuresOrderListParams must not error")
	exp := url.Values{
		"symbol": {"XBTUSDTM"}, "side": {"buy"}, "type": {"limit"}, "startAt": {"1727654400000"},
		"endAt": {"1727740800000"}, "currentPage": {"2"}, "pageSize": {"100"},
	}
	assert.Equal(t, exp, params, "futuresOrderListParams should send every parameter")
}

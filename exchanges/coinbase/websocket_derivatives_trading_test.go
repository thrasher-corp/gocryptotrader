package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPIBuy(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesBuyRequest()
	if !mockTests {
		req = newDerivativesTradingLiveOrderRequest(t, DerivativesDirectionBuy)
	}
	resp, err := ex.WebsocketAPI().Buy(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.Buy must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "WebsocketAPI.Buy should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateOpen, resp.Order.OrderState, "WebsocketAPI.Buy should place a resting order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().Buy(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.Buy should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelOrderByID(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesCancelOrderByIDRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesCancelOrderByIDRequest{OrderID: order.OrderID}
	}
	resp, err := ex.WebsocketAPI().CancelOrderByID(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.CancelOrderByID must not error")
	if mockTests {
		assert.Equal(t, testDerivativesCancelledOrder(), resp, "WebsocketAPI.CancelOrderByID should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateCancelled, resp.OrderState, "WebsocketAPI.CancelOrderByID should cancel the order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelOrderByID(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelOrderByID should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelAll(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAll is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CancelAll(t.Context(), mockDerivativesCancelAllRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAll must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelAll should decode the count")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelAll(t.Context(), mockDerivativesCancelAllRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelAll should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelAllByCurrency(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByCurrency is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CancelAllByCurrency(t.Context(), mockDerivativesCancelAllByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByCurrency must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelAllByCurrency should decode the count")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelAllByCurrency(t.Context(), mockDerivativesCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByCurrency should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelAllByCurrencyPair(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByCurrencyPair is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CancelAllByCurrencyPair(t.Context(), mockDerivativesCancelAllByCurrencyPairRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByCurrencyPair must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelAllByCurrencyPair should decode the count")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelAllByCurrencyPair(t.Context(), mockDerivativesCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByCurrencyPair should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelAllByInstrument(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByInstrument is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CancelAllByInstrument(t.Context(), mockDerivativesCancelAllByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByInstrument must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelAllByInstrument should decode the count")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelAllByInstrument(t.Context(), mockDerivativesCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByInstrument should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelAllByKindOrType(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByKindOrType is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CancelAllByKindOrType(t.Context(), mockDerivativesCancelAllByKindOrTypeRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByKindOrType must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelAllByKindOrType should decode the count")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelAllByKindOrType(t.Context(), mockDerivativesCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByKindOrType should refuse an unauthenticated connection")
}

func TestWebsocketAPICancelByLabel(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesCancelByLabelRequest()
	if !mockTests {
		placed, _ := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesCancelByLabelRequest{Label: placed.Label, Currency: derivativesTradingLiveCurrency}
	}
	resp, err := ex.WebsocketAPI().CancelByLabel(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.CancelByLabel must not error")
	if mockTests {
		assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "WebsocketAPI.CancelByLabel should decode the count")
	} else {
		assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 1}, resp, "WebsocketAPI.CancelByLabel should cancel the order with the label")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CancelByLabel(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CancelByLabel should refuse an unauthenticated connection")
}

func TestWebsocketAPIClosePosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ClosePosition is not tested live, as " + derivativesTradingSkipClosePosition)
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().ClosePosition(t.Context(), mockDerivativesClosePositionRequest())
	require.NoError(t, err, "WebsocketAPI.ClosePosition must not error")
	assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "WebsocketAPI.ClosePosition should decode every field")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().ClosePosition(t.Context(), mockDerivativesClosePositionRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.ClosePosition should refuse an unauthenticated connection")
}

func TestWebsocketAPIEditOrder(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesEditOrderRequest()
	if !mockTests {
		placed, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesEditOrderRequest{OrderID: order.OrderID, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	resp, err := ex.WebsocketAPI().EditOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.EditOrder must not error")
	if mockTests {
		assert.Equal(t, testDerivativesEditedOrderResponse(), resp, "WebsocketAPI.EditOrder should decode every field")
	} else {
		assert.Equal(t, req.Price, resp.Order.Price.Value, "WebsocketAPI.EditOrder should reprice the order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().EditOrder(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.EditOrder should refuse an unauthenticated connection")
}

func TestWebsocketAPIEditOrderByLabel(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesEditOrderByLabelRequest()
	if !mockTests {
		placed, _ := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesEditOrderByLabelRequest{InstrumentName: placed.InstrumentName, Label: placed.Label, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	resp, err := ex.WebsocketAPI().EditOrderByLabel(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.EditOrderByLabel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesEditedOrderResponse(), resp, "WebsocketAPI.EditOrderByLabel should decode every field")
	} else {
		assert.Equal(t, req.Price, resp.Order.Price.Value, "WebsocketAPI.EditOrderByLabel should reprice the order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().EditOrderByLabel(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.EditOrderByLabel should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetMargins(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesMarginsRequest()
	if !mockTests {
		req = &DerivativesMarginsRequest{InstrumentName: derivativesTradingLiveInstrument, Amount: derivativesTradingLiveAmount, Price: getDerivativesTradingLiveTicker(t).MarkPrice}
	}
	resp, err := ex.WebsocketAPI().GetMargins(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetMargins must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMarginsResponse(), resp, "WebsocketAPI.GetMargins should decode every field")
	} else {
		assert.Positive(t, resp.MaximumPrice, "WebsocketAPI.GetMargins should return the highest buy price")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetMargins(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetMargins should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetMaxOrderSize(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesMaxOrderSizeRequest()
	if !mockTests {
		req = &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument}
	}
	resp, err := ex.WebsocketAPI().GetMaxOrderSize(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetMaxOrderSize must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMaxOrderSizeResponse(), resp, "WebsocketAPI.GetMaxOrderSize should decode every field")
	} else {
		assert.Positive(t, resp.BuyPrice, "WebsocketAPI.GetMaxOrderSize should return the reference buy price")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetMaxOrderSize(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetMaxOrderSize should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOpenOrders(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOpenOrdersRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersRequest{}
	}
	resp, err := ex.WebsocketAPI().GetOpenOrders(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOpenOrders must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "WebsocketAPI.GetOpenOrders should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrders should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOpenOrdersByCurrencyRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByCurrencyRequest{Currency: derivativesTradingLiveCurrency}
	}
	resp, err := ex.WebsocketAPI().GetOpenOrdersByCurrency(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "WebsocketAPI.GetOpenOrdersByCurrency should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOpenOrdersByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByCurrency should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOpenOrdersByInstrumentRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument}
	}
	resp, err := ex.WebsocketAPI().GetOpenOrdersByInstrument(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "WebsocketAPI.GetOpenOrdersByInstrument should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOpenOrdersByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByInstrument should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOpenOrdersByLabelRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByLabelRequest{Currency: derivativesTradingLiveCurrency, Label: "gct-no-orders"}
	}
	resp, err := ex.WebsocketAPI().GetOpenOrdersByLabel(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByLabel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "WebsocketAPI.GetOpenOrdersByLabel should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOpenOrdersByLabel(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByLabel should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOrderHistoryByCurrencyRequest()
	if !mockTests {
		req = &DerivativesOrderHistoryByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetOrderHistoryByCurrency(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOrderHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderHistoryResponse(), resp, "WebsocketAPI.GetOrderHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Orders), 1, "WebsocketAPI.GetOrderHistoryByCurrency should return at most the count of orders")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOrderHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOrderHistoryByCurrency should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOrderHistoryByInstrumentRequest()
	if !mockTests {
		req = &DerivativesOrderHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetOrderHistoryByInstrument(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOrderHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderHistoryResponse(), resp, "WebsocketAPI.GetOrderHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Orders), 1, "WebsocketAPI.GetOrderHistoryByInstrument should return at most the count of orders")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOrderHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOrderHistoryByInstrument should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOrderMarginByIDs(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOrderMarginByIDsRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesOrderMarginByIDsRequest{OrderIDs: []string{order.OrderID}}
	}
	resp, err := ex.WebsocketAPI().GetOrderMarginByIDs(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOrderMarginByIDs must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderMargins(), resp, "WebsocketAPI.GetOrderMarginByIDs should decode every field")
	} else if assert.Len(t, resp, 1, "WebsocketAPI.GetOrderMarginByIDs should return the order's margin") {
		assert.Equal(t, req.OrderIDs[0], resp[0].OrderID, "WebsocketAPI.GetOrderMarginByIDs should return the order's margin")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOrderMarginByIDs(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOrderMarginByIDs should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOrderState(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOrderStateRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesOrderStateRequest{OrderID: order.OrderID}
	}
	resp, err := ex.WebsocketAPI().GetOrderState(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOrderState must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderState(""), resp, "WebsocketAPI.GetOrderState should decode every field")
	} else {
		assert.Equal(t, req.OrderID, resp.OrderID, "WebsocketAPI.GetOrderState should return the order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOrderState(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOrderState should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesOrderStateByLabelRequest()
	if !mockTests {
		req = &DerivativesOrderStateByLabelRequest{Currency: derivativesTradingLiveCurrency, Label: "gct-no-orders"}
	}
	resp, err := ex.WebsocketAPI().GetOrderStateByLabel(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetOrderStateByLabel must not error")
	if mockTests {
		assert.Equal(t, []DerivativesOrder{*testDerivativesOrderState("fooBar")}, resp, "WebsocketAPI.GetOrderStateByLabel should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetOrderStateByLabel(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetOrderStateByLabel should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetSettlementHistoryByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesSettlementHistoryByCurrencyRequest()
	if !mockTests {
		req = &DerivativesSettlementHistoryByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetSettlementHistoryByCurrency(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetSettlementHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSettlementHistoryResponse(), resp, "WebsocketAPI.GetSettlementHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Settlements), 1, "WebsocketAPI.GetSettlementHistoryByCurrency should return at most the count of events")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetSettlementHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetSettlementHistoryByCurrency should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesSettlementHistoryByInstrumentRequest()
	if !mockTests {
		req = &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetSettlementHistoryByInstrument(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetSettlementHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSettlementHistoryResponse(), resp, "WebsocketAPI.GetSettlementHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Settlements), 1, "WebsocketAPI.GetSettlementHistoryByInstrument should return at most the count of events")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetSettlementHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetSettlementHistoryByInstrument should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesTriggerOrderHistoryRequest()
	if !mockTests {
		req = &DerivativesTriggerOrderHistoryRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetTriggerOrderHistory(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetTriggerOrderHistory must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTriggerOrderHistoryResponse(), resp, "WebsocketAPI.GetTriggerOrderHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Entries), 1, "WebsocketAPI.GetTriggerOrderHistory should return at most the count of entries")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetTriggerOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetTriggerOrderHistory should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesUserTradesByCurrencyRequest()
	if !mockTests {
		req = &DerivativesUserTradesByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetUserTradesByCurrency(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "WebsocketAPI.GetUserTradesByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "WebsocketAPI.GetUserTradesByCurrency should return at most the count of trades")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserTradesByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByCurrency should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesUserTradesByCurrencyAndTimeRequest()
	if !mockTests {
		req = &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: derivativesTradingLiveCurrency, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetUserTradesByCurrencyAndTime(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "WebsocketAPI.GetUserTradesByCurrencyAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "WebsocketAPI.GetUserTradesByCurrencyAndTime should return at most the count of trades")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserTradesByCurrencyAndTime(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByCurrencyAndTime should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesUserTradesByInstrumentRequest()
	if !mockTests {
		req = &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetUserTradesByInstrument(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "WebsocketAPI.GetUserTradesByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "WebsocketAPI.GetUserTradesByInstrument should return at most the count of trades")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserTradesByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByInstrument should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesUserTradesByInstrumentAndTimeRequest()
	if !mockTests {
		req = &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	resp, err := ex.WebsocketAPI().GetUserTradesByInstrumentAndTime(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "WebsocketAPI.GetUserTradesByInstrumentAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "WebsocketAPI.GetUserTradesByInstrumentAndTime should return at most the count of trades")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserTradesByInstrumentAndTime(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByInstrumentAndTime should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesUserTradesByOrderRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesUserTradesByOrderRequest{OrderID: order.OrderID}
	}
	resp, err := ex.WebsocketAPI().GetUserTradesByOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByOrder must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesByOrder(), resp, "WebsocketAPI.GetUserTradesByOrder should decode every field")
	} else {
		assert.Empty(t, resp, "WebsocketAPI.GetUserTradesByOrder should return no trades for an unfilled order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserTradesByOrder(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByOrder should refuse an unauthenticated connection")
}

func TestWebsocketAPISell(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	req := mockDerivativesSellRequest()
	if !mockTests {
		req = newDerivativesTradingLiveOrderRequest(t, DerivativesDirectionSell)
	}
	resp, err := ex.WebsocketAPI().Sell(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.Sell must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "WebsocketAPI.Sell should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateOpen, resp.Order.OrderState, "WebsocketAPI.Sell should place a resting order")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().Sell(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.Sell should refuse an unauthenticated connection")
}

func TestWebsocketAPISetLeverage(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	if !mockTests {
		assert.Equal(t, 2.0, setDerivativesTradingLiveLeverage(t, ex.WebsocketAPI().SetLeverage), "WebsocketAPI.SetLeverage should return the leverage set")
		return
	}
	leverage, err := ex.WebsocketAPI().SetLeverage(t.Context(), mockDerivativesSetLeverageRequest())
	require.NoError(t, err, "WebsocketAPI.SetLeverage must not error")
	assert.Equal(t, 50.0, leverage, "WebsocketAPI.SetLeverage should return the leverage stored")
	leverage, err = ex.WebsocketAPI().SetLeverage(t.Context(), mockDerivativesRemoveLeverageRequest())
	require.NoError(t, err, "WebsocketAPI.SetLeverage must not error removing the custom leverage")
	assert.Zero(t, leverage, "WebsocketAPI.SetLeverage should return zero once the custom leverage is removed")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().SetLeverage(t.Context(), mockDerivativesSetLeverageRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.SetLeverage should refuse an unauthenticated connection")
}

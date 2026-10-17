package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestWebsocketAPIBuy(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockBuyRequest()
	if !mockTests {
		arg = newLiveTradingOrderRequest(t, DirectionBuy)
	}
	result, err := ex.WebsocketAPI().Buy(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.Buy must not error")
	if mockTests {
		assert.Equal(t, testPlacedOrderResponse(), result, "WebsocketAPI.Buy should decode every field")
	} else {
		assert.Equal(t, "open", result.Order.OrderState, "WebsocketAPI.Buy should place a resting order")
	}

	_, err = newTestExchange(t).WebsocketAPI().Buy(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Buy should error without an authenticated connection")
}

func TestWebsocketAPICancelOrderByID(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	orderID := mockCancelOrderID
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderID = order.OrderID
	}
	result, err := ex.WebsocketAPI().CancelOrderByID(t.Context(), orderID)
	require.NoError(t, err, "WebsocketAPI.CancelOrderByID must not error")
	if mockTests {
		assert.Equal(t, testCancelledOrder(), result, "WebsocketAPI.CancelOrderByID should decode every field")
	} else {
		assert.Equal(t, "cancelled", result.OrderState, "WebsocketAPI.CancelOrderByID should cancel the order")
	}

	_, err = newTestExchange(t).WebsocketAPI().CancelOrderByID(t.Context(), orderID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelOrderByID should error without an authenticated connection")
}

func TestWebsocketAPICancelAll(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAll is not tested live, as " + liveTradingSkipCancelAll)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAll(t.Context(), &CancelAllRequest{})
	require.NoError(t, err, "WebsocketAPI.CancelAll must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelAll should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelAll(t.Context(), &CancelAllRequest{})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAll should error without an authenticated connection")
}

func TestWebsocketAPICancelAllByCurrency(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByCurrency is not tested live, as " + liveTradingSkipCancelAll)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAllByCurrency(t.Context(), mockCancelAllByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByCurrency must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelAllByCurrency should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelAllByCurrency(t.Context(), mockCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByCurrency should error without an authenticated connection")
}

func TestWebsocketAPICancelAllByCurrencyPair(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByCurrencyPair is not tested live, as " + liveTradingSkipCancelAll)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAllByCurrencyPair(t.Context(), mockCancelAllByCurrencyPairRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByCurrencyPair must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelAllByCurrencyPair should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelAllByCurrencyPair(t.Context(), mockCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByCurrencyPair should error without an authenticated connection")
}

func TestWebsocketAPICancelAllByInstrument(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByInstrument is not tested live, as " + liveTradingSkipCancelAll)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAllByInstrument(t.Context(), mockCancelAllByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByInstrument must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelAllByInstrument should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelAllByInstrument(t.Context(), mockCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByInstrument should error without an authenticated connection")
}

func TestWebsocketAPICancelAllByKindOrType(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelAllByKindOrType is not tested live, as " + liveTradingSkipCancelAll)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAllByKindOrType(t.Context(), mockCancelAllByKindOrTypeRequest())
	require.NoError(t, err, "WebsocketAPI.CancelAllByKindOrType must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelAllByKindOrType should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelAllByKindOrType(t.Context(), mockCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAllByKindOrType should error without an authenticated connection")
}

func TestWebsocketAPICancelByLabel(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockCancelByLabelRequest()
	if !mockTests {
		placed, _ := placeLiveTradingOrder(t)
		arg = &CancelByLabelRequest{Label: placed.Label, Currency: currency.BTC}
	}
	result, err := ex.WebsocketAPI().CancelByLabel(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CancelByLabel must not error")
	if mockTests {
		assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelByLabel should decode the count")
	} else {
		assert.Equal(t, &CancelAllResponse{CancelledCount: 1}, result, "WebsocketAPI.CancelByLabel should cancel the order with the label")
	}

	_, err = newTestExchange(t).WebsocketAPI().CancelByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelByLabel should error without an authenticated connection")
}

func TestWebsocketAPICancelQuotes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelQuotes is not tested live, as " + liveTradingSkipMassQuotes)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelQuotes(t.Context(), mockCancelQuotesRequest())
	require.NoError(t, err, "WebsocketAPI.CancelQuotes must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "WebsocketAPI.CancelQuotes should decode the count")

	_, err = newTestExchange(t).WebsocketAPI().CancelQuotes(t.Context(), mockCancelQuotesRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelQuotes should error without an authenticated connection")
}

func TestWebsocketAPIClosePosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ClosePosition needs an open position, which a live test would have to open at market")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().ClosePosition(t.Context(), mockClosePositionRequest())
	require.NoError(t, err, "WebsocketAPI.ClosePosition must not error")
	assert.Equal(t, testPlacedOrderResponse(), result, "WebsocketAPI.ClosePosition should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().ClosePosition(t.Context(), mockClosePositionRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ClosePosition should error without an authenticated connection")
}

func TestWebsocketAPIEditOrder(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockEditOrderRequest()
	if !mockTests {
		placed, order := placeLiveTradingOrder(t)
		arg = &EditOrderRequest{OrderID: order.OrderID, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	result, err := ex.WebsocketAPI().EditOrder(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.EditOrder must not error")
	if mockTests {
		assert.Equal(t, testEditedOrderResponse(), result, "WebsocketAPI.EditOrder should decode every field")
	} else {
		assert.Equal(t, arg.Price, result.Order.Price.Value, "WebsocketAPI.EditOrder should reprice the order")
	}

	_, err = newTestExchange(t).WebsocketAPI().EditOrder(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EditOrder should error without an authenticated connection")
}

func TestWebsocketAPIEditOrderByLabel(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockEditOrderByLabelRequest()
	if !mockTests {
		placed, _ := placeLiveTradingOrder(t)
		arg = &EditOrderByLabelRequest{InstrumentName: placed.InstrumentName, Label: placed.Label, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	result, err := ex.WebsocketAPI().EditOrderByLabel(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.EditOrderByLabel must not error")
	if mockTests {
		assert.Equal(t, testEditedOrderResponse(), result, "WebsocketAPI.EditOrderByLabel should decode every field")
	} else {
		assert.Equal(t, arg.Price, result.Order.Price.Value, "WebsocketAPI.EditOrderByLabel should reprice the order")
	}

	_, err = newTestExchange(t).WebsocketAPI().EditOrderByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EditOrderByLabel should error without an authenticated connection")
}

func TestWebsocketAPIGetMargins(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockMarginsRequest()
	if !mockTests {
		arg = &MarginsRequest{InstrumentName: liveTradingInstrument, Amount: liveTradingOrderAmount, Price: getLiveTradingTicker(t).MarkPrice}
	}
	result, err := ex.WebsocketAPI().GetMargins(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetMargins must not error")
	if mockTests {
		assert.Equal(t, testMarginsResponse(), result, "WebsocketAPI.GetMargins should decode every field")
	} else {
		assert.Positive(t, result.MaxPrice, "WebsocketAPI.GetMargins should return the highest buy price")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetMargins(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetMargins should error without an authenticated connection")
}

func TestWebsocketAPIGetMMPConfig(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		arg = &MMPGroupRequest{}
	}
	result, err := ex.WebsocketAPI().GetMMPConfig(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetMMPConfig must not error")
	if mockTests {
		assert.Equal(t, testMMPConfigs(), result, "WebsocketAPI.GetMMPConfig should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetMMPConfig(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetMMPConfig should error without an authenticated connection")
}

func TestWebsocketAPIGetMMPStatus(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		arg = &MMPGroupRequest{}
	}
	result, err := ex.WebsocketAPI().GetMMPStatus(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetMMPStatus must not error")
	if mockTests {
		assert.Equal(t, testMMPStatuses(), result, "WebsocketAPI.GetMMPStatus should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetMMPStatus(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetMMPStatus should error without an authenticated connection")
}

func TestWebsocketAPIGetOpenOrders(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	require.NoError(t, err, "WebsocketAPI.GetOpenOrders must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "WebsocketAPI.GetOpenOrders should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrders should error without an authenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOpenOrdersByCurrency(t.Context(), mockOpenOrdersByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "WebsocketAPI.GetOpenOrdersByCurrency should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOpenOrdersByCurrency(t.Context(), mockOpenOrdersByCurrencyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByCurrency should error without an authenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockOpenOrdersByInstrumentRequest()
	if !mockTests {
		arg = &OpenOrdersByInstrumentRequest{InstrumentName: liveTradingInstrument}
	}
	result, err := ex.WebsocketAPI().GetOpenOrdersByInstrument(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByInstrument must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "WebsocketAPI.GetOpenOrdersByInstrument should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOpenOrdersByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByInstrument should error without an authenticated connection")
}

func TestWebsocketAPIGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOpenOrdersByLabel(t.Context(), mockOpenOrdersByLabelRequest())
	require.NoError(t, err, "WebsocketAPI.GetOpenOrdersByLabel must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "WebsocketAPI.GetOpenOrdersByLabel should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOpenOrdersByLabel(t.Context(), mockOpenOrdersByLabelRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOpenOrdersByLabel should error without an authenticated connection")
}

func TestWebsocketAPIGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOrderHistoryByCurrency(t.Context(), mockOrderHistoryByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetOrderHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOrderHistoryResponse(), result, "WebsocketAPI.GetOrderHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Orders), 1, "WebsocketAPI.GetOrderHistoryByCurrency should return at most the count of orders")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderHistoryByCurrency(t.Context(), mockOrderHistoryByCurrencyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOrderHistoryByCurrency should error without an authenticated connection")
}

func TestWebsocketAPIGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOrderHistoryByInstrument(t.Context(), mockOrderHistoryByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.GetOrderHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testOrderHistoryResponse(), result, "WebsocketAPI.GetOrderHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Orders), 1, "WebsocketAPI.GetOrderHistoryByInstrument should return at most the count of orders")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderHistoryByInstrument(t.Context(), mockOrderHistoryByInstrumentRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOrderHistoryByInstrument should error without an authenticated connection")
}

func TestWebsocketAPIGetOrderMarginByIDs(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	orderIDs := mockOrderMarginIDs
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderIDs = []string{order.OrderID}
	}
	result, err := ex.WebsocketAPI().GetOrderMarginByIDs(t.Context(), orderIDs)
	require.NoError(t, err, "WebsocketAPI.GetOrderMarginByIDs must not error")
	if mockTests {
		assert.Equal(t, testOrderMargins(), result, "WebsocketAPI.GetOrderMarginByIDs should decode every field")
	} else {
		assert.Len(t, result, 1, "WebsocketAPI.GetOrderMarginByIDs should return the order's margin")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderMarginByIDs(t.Context(), orderIDs)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOrderMarginByIDs should error without an authenticated connection")
}

func TestWebsocketAPIGetOrderState(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	orderID := mockOrderStateID
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderID = order.OrderID
	}
	result, err := ex.WebsocketAPI().GetOrderState(t.Context(), orderID)
	require.NoError(t, err, "WebsocketAPI.GetOrderState must not error")
	if mockTests {
		assert.Equal(t, testOrderState(""), result, "WebsocketAPI.GetOrderState should decode every field")
	} else {
		assert.Equal(t, orderID, result.OrderID, "WebsocketAPI.GetOrderState should return the order")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderState(t.Context(), orderID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOrderState should error without an authenticated connection")
}

func TestWebsocketAPIGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetOrderStateByLabel(t.Context(), mockOrderStateByLabelRequest())
	require.NoError(t, err, "WebsocketAPI.GetOrderStateByLabel must not error")
	if mockTests {
		assert.Equal(t, []Order{*testOrderState("fooBar")}, result, "WebsocketAPI.GetOrderStateByLabel should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderStateByLabel(t.Context(), mockOrderStateByLabelRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetOrderStateByLabel should error without an authenticated connection")
}

func TestWebsocketAPIGetSettlementHistoryByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockSettlementHistoryByCurrencyRequest()
	if !mockTests {
		arg = &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := ex.WebsocketAPI().GetSettlementHistoryByCurrency(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetSettlementHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testSettlementHistoryResponse(), result, "WebsocketAPI.GetSettlementHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Settlements), 1, "WebsocketAPI.GetSettlementHistoryByCurrency should return at most the count of events")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetSettlementHistoryByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetSettlementHistoryByCurrency should error without an authenticated connection")
}

func TestWebsocketAPIGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockSettlementHistoryByInstrumentRequest()
	if !mockTests {
		arg = &SettlementHistoryByInstrumentRequest{InstrumentName: liveTradingInstrument, Count: 1}
	}
	result, err := ex.WebsocketAPI().GetSettlementHistoryByInstrument(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetSettlementHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testSettlementHistoryResponse(), result, "WebsocketAPI.GetSettlementHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Settlements), 1, "WebsocketAPI.GetSettlementHistoryByInstrument should return at most the count of events")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetSettlementHistoryByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetSettlementHistoryByInstrument should error without an authenticated connection")
}

func TestWebsocketAPIGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockTriggerOrderHistoryRequest()
	if !mockTests {
		arg = &TriggerOrderHistoryRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := ex.WebsocketAPI().GetTriggerOrderHistory(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetTriggerOrderHistory must not error")
	if mockTests {
		assert.Equal(t, testTriggerOrderHistoryResponse(), result, "WebsocketAPI.GetTriggerOrderHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Entries), 1, "WebsocketAPI.GetTriggerOrderHistory should return at most the count of entries")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTriggerOrderHistory(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetTriggerOrderHistory should error without an authenticated connection")
}

func TestWebsocketAPIGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockUserTradesByCurrencyRequest()
	if !mockTests {
		arg = &UserTradesByCurrencyRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := ex.WebsocketAPI().GetUserTradesByCurrency(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "WebsocketAPI.GetUserTradesByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "WebsocketAPI.GetUserTradesByCurrency should return at most the count of trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetUserTradesByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByCurrency should error without an authenticated connection")
}

func TestWebsocketAPIGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockUserTradesByCurrencyAndTimeRequest()
	if !mockTests {
		arg = &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	result, err := ex.WebsocketAPI().GetUserTradesByCurrencyAndTime(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "WebsocketAPI.GetUserTradesByCurrencyAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "WebsocketAPI.GetUserTradesByCurrencyAndTime should return at most the count of trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetUserTradesByCurrencyAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByCurrencyAndTime should error without an authenticated connection")
}

func TestWebsocketAPIGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockUserTradesByInstrumentRequest()
	if !mockTests {
		arg = &UserTradesByInstrumentRequest{InstrumentName: liveTradingInstrument, Count: 1}
	}
	result, err := ex.WebsocketAPI().GetUserTradesByInstrument(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "WebsocketAPI.GetUserTradesByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "WebsocketAPI.GetUserTradesByInstrument should return at most the count of trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetUserTradesByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByInstrument should error without an authenticated connection")
}

func TestWebsocketAPIGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockUserTradesByInstrumentAndTimeRequest()
	if !mockTests {
		arg = &UserTradesByInstrumentAndTimeRequest{InstrumentName: liveTradingInstrument, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	result, err := ex.WebsocketAPI().GetUserTradesByInstrumentAndTime(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "WebsocketAPI.GetUserTradesByInstrumentAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "WebsocketAPI.GetUserTradesByInstrumentAndTime should return at most the count of trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetUserTradesByInstrumentAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByInstrumentAndTime should error without an authenticated connection")
}

func TestWebsocketAPIGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := &UserTradesByOrderRequest{OrderID: mockUserTradesOrderID}
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		arg = &UserTradesByOrderRequest{OrderID: order.OrderID}
	}
	result, err := ex.WebsocketAPI().GetUserTradesByOrder(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetUserTradesByOrder must not error")
	if mockTests {
		assert.Equal(t, testUserTradesByOrder(), result, "WebsocketAPI.GetUserTradesByOrder should decode every field")
	} else {
		assert.Empty(t, result, "WebsocketAPI.GetUserTradesByOrder should return no trades for an unfilled order")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetUserTradesByOrder(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserTradesByOrder should error without an authenticated connection")
}

func TestWebsocketAPIMassQuote(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.MassQuote is not tested live, as " + liveTradingSkipMassQuotes)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().MassQuote(t.Context(), mockMassQuoteRequest())
	require.NoError(t, err, "WebsocketAPI.MassQuote must not error")
	assert.Equal(t, testMassQuoteResponse(), result, "WebsocketAPI.MassQuote should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().MassQuote(t.Context(), mockMassQuoteRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.MassQuote should error without an authenticated connection")
}

func TestWebsocketAPIMovePositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.MovePositions needs two subaccounts and a position to move, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().MovePositions(t.Context(), mockMovePositionsRequest())
	require.NoError(t, err, "WebsocketAPI.MovePositions must not error")
	assert.Equal(t, testPositionMoveTrades(), result, "WebsocketAPI.MovePositions should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().MovePositions(t.Context(), mockMovePositionsRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.MovePositions should error without an authenticated connection")
}

func TestWebsocketAPIResetMMP(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		setLiveTradingMMPConfig(t, e.SetMMPConfig)
		arg = &MMPGroupRequest{IndexName: liveTradingIndexName}
	}
	require.NoError(t, ex.WebsocketAPI().ResetMMP(t.Context(), arg), "WebsocketAPI.ResetMMP must not error")

	err := newTestExchange(t).WebsocketAPI().ResetMMP(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ResetMMP should error without an authenticated connection")
}

func TestWebsocketAPISell(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	arg := mockSellRequest()
	if !mockTests {
		arg = newLiveTradingOrderRequest(t, DirectionSell)
	}
	result, err := ex.WebsocketAPI().Sell(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.Sell must not error")
	if mockTests {
		assert.Equal(t, testPlacedOrderResponse(), result, "WebsocketAPI.Sell should decode every field")
	} else {
		assert.Equal(t, "open", result.Order.OrderState, "WebsocketAPI.Sell should place a resting order")
	}

	_, err = newTestExchange(t).WebsocketAPI().Sell(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.Sell should error without an authenticated connection")
}

func TestWebsocketAPISetMMPConfig(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	if !mockTests {
		result := setLiveTradingMMPConfig(t, ex.WebsocketAPI().SetMMPConfig)
		assert.NotEmpty(t, result, "WebsocketAPI.SetMMPConfig should return the configuration")
		return
	}
	result, err := ex.WebsocketAPI().SetMMPConfig(t.Context(), mockSetMMPConfigRequest())
	require.NoError(t, err, "WebsocketAPI.SetMMPConfig must not error")
	assert.Equal(t, testSetMMPConfigs(), result, "WebsocketAPI.SetMMPConfig should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().SetMMPConfig(t.Context(), mockSetMMPConfigRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetMMPConfig should error without an authenticated connection")
}

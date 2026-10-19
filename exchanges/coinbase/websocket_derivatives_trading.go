package coinbase

import (
	"context"
)

// Buy places a buy order as GlobalDerivativesAPI.Buy does, over the websocket
func (w WebsocketAPI) Buy(ctx context.Context, req *DerivativesOrderRequest) (*DerivativesOrderResponse, error) {
	return w.e.GlobalDerivatives().placeOrder(ctx, true, derivativesBuyMethod, req)
}

// CancelOrderByID cancels an order as GlobalDerivativesAPI.CancelOrderByID does, over the websocket
func (w WebsocketAPI) CancelOrderByID(ctx context.Context, req *DerivativesCancelOrderByIDRequest) (*DerivativesOrder, error) {
	return w.e.GlobalDerivatives().cancelOrderByID(ctx, true, req)
}

// CancelAll cancels every order as GlobalDerivativesAPI.CancelAll does, over the websocket
func (w WebsocketAPI) CancelAll(ctx context.Context, req *DerivativesCancelAllRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelAll(ctx, true, req)
}

// CancelAllByCurrency cancels a currency's orders as GlobalDerivativesAPI.CancelAllByCurrency does, over the websocket
func (w WebsocketAPI) CancelAllByCurrency(ctx context.Context, req *DerivativesCancelAllByCurrencyRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelAllByCurrency(ctx, true, req)
}

// CancelAllByCurrencyPair cancels a currency pair's orders as GlobalDerivativesAPI.CancelAllByCurrencyPair does, over
// the websocket
func (w WebsocketAPI) CancelAllByCurrencyPair(ctx context.Context, req *DerivativesCancelAllByCurrencyPairRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelAllByCurrencyPair(ctx, true, req)
}

// CancelAllByInstrument cancels an instrument's orders as GlobalDerivativesAPI.CancelAllByInstrument does, over the
// websocket
func (w WebsocketAPI) CancelAllByInstrument(ctx context.Context, req *DerivativesCancelAllByInstrumentRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelAllByInstrument(ctx, true, req)
}

// CancelAllByKindOrType cancels orders as GlobalDerivativesAPI.CancelAllByKindOrType does, over the websocket
func (w WebsocketAPI) CancelAllByKindOrType(ctx context.Context, req *DerivativesCancelAllByKindOrTypeRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelAllByKindOrType(ctx, true, req)
}

// CancelByLabel cancels the orders with a label as GlobalDerivativesAPI.CancelByLabel does, over the websocket
func (w WebsocketAPI) CancelByLabel(ctx context.Context, req *DerivativesCancelByLabelRequest) (*DerivativesCancelAllResponse, error) {
	return w.e.GlobalDerivatives().cancelByLabel(ctx, true, req)
}

// ClosePosition closes a position as GlobalDerivativesAPI.ClosePosition does, over the websocket
func (w WebsocketAPI) ClosePosition(ctx context.Context, req *DerivativesClosePositionRequest) (*DerivativesOrderResponse, error) {
	return w.e.GlobalDerivatives().closePosition(ctx, true, req)
}

// EditOrder edits an order as GlobalDerivativesAPI.EditOrder does, over the websocket
func (w WebsocketAPI) EditOrder(ctx context.Context, req *DerivativesEditOrderRequest) (*DerivativesOrderResponse, error) {
	return w.e.GlobalDerivatives().editOrder(ctx, true, req)
}

// EditOrderByLabel edits an order by its label as GlobalDerivativesAPI.EditOrderByLabel does, over the websocket
func (w WebsocketAPI) EditOrderByLabel(ctx context.Context, req *DerivativesEditOrderByLabelRequest) (*DerivativesOrderResponse, error) {
	return w.e.GlobalDerivatives().editOrderByLabel(ctx, true, req)
}

// GetMargins returns a hypothetical order's margins as GlobalDerivativesAPI.GetMargins does, over the websocket
func (w WebsocketAPI) GetMargins(ctx context.Context, req *DerivativesMarginsRequest) (*DerivativesMarginsResponse, error) {
	return w.e.GlobalDerivatives().getMargins(ctx, true, req)
}

// GetMaxOrderSize returns a future's largest orders as GlobalDerivativesAPI.GetMaxOrderSize does, over the websocket
func (w WebsocketAPI) GetMaxOrderSize(ctx context.Context, req *DerivativesMaxOrderSizeRequest) (*DerivativesMaxOrderSizeResponse, error) {
	return w.e.GlobalDerivatives().getMaxOrderSize(ctx, true, req)
}

// GetOpenOrders returns the account's open orders as GlobalDerivativesAPI.GetOpenOrders does, over the websocket
func (w WebsocketAPI) GetOpenOrders(ctx context.Context, req *DerivativesOpenOrdersRequest) ([]DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOpenOrders(ctx, true, req)
}

// GetOpenOrdersByCurrency returns a currency's open orders as GlobalDerivativesAPI.GetOpenOrdersByCurrency does, over
// the websocket
func (w WebsocketAPI) GetOpenOrdersByCurrency(ctx context.Context, req *DerivativesOpenOrdersByCurrencyRequest) ([]DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOpenOrdersByCurrency(ctx, true, req)
}

// GetOpenOrdersByInstrument returns an instrument's open orders as GlobalDerivativesAPI.GetOpenOrdersByInstrument
// does, over the websocket
func (w WebsocketAPI) GetOpenOrdersByInstrument(ctx context.Context, req *DerivativesOpenOrdersByInstrumentRequest) ([]DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOpenOrdersByInstrument(ctx, true, req)
}

// GetOpenOrdersByLabel returns the open orders with a label as GlobalDerivativesAPI.GetOpenOrdersByLabel does, over the
// websocket
func (w WebsocketAPI) GetOpenOrdersByLabel(ctx context.Context, req *DerivativesOpenOrdersByLabelRequest) ([]DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOpenOrdersByLabel(ctx, true, req)
}

// GetOrderHistoryByCurrency returns a currency's order history as GlobalDerivativesAPI.GetOrderHistoryByCurrency does,
// over the websocket
func (w WebsocketAPI) GetOrderHistoryByCurrency(ctx context.Context, req *DerivativesOrderHistoryByCurrencyRequest) (*DerivativesOrderHistoryResponse, error) {
	return w.e.GlobalDerivatives().getOrderHistoryByCurrency(ctx, true, req)
}

// GetOrderHistoryByInstrument returns an instrument's order history as GlobalDerivativesAPI.GetOrderHistoryByInstrument
// does, over the websocket
func (w WebsocketAPI) GetOrderHistoryByInstrument(ctx context.Context, req *DerivativesOrderHistoryByInstrumentRequest) (*DerivativesOrderHistoryResponse, error) {
	return w.e.GlobalDerivatives().getOrderHistoryByInstrument(ctx, true, req)
}

// GetOrderMarginByIDs returns the orders' initial margins as GlobalDerivativesAPI.GetOrderMarginByIDs does, over the
// websocket
func (w WebsocketAPI) GetOrderMarginByIDs(ctx context.Context, req *DerivativesOrderMarginByIDsRequest) ([]DerivativesOrderMargin, error) {
	return w.e.GlobalDerivatives().getOrderMarginByIDs(ctx, true, req)
}

// GetOrderState returns an order as GlobalDerivativesAPI.GetOrderState does, over the websocket
func (w WebsocketAPI) GetOrderState(ctx context.Context, req *DerivativesOrderStateRequest) (*DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOrderState(ctx, true, req)
}

// GetOrderStateByLabel returns the orders with a label as GlobalDerivativesAPI.GetOrderStateByLabel does, over the
// websocket
func (w WebsocketAPI) GetOrderStateByLabel(ctx context.Context, req *DerivativesOrderStateByLabelRequest) ([]DerivativesOrder, error) {
	return w.e.GlobalDerivatives().getOrderStateByLabel(ctx, true, req)
}

// GetSettlementHistoryByCurrency returns a currency's settlement events as
// GlobalDerivativesAPI.GetSettlementHistoryByCurrency does, over the websocket
func (w WebsocketAPI) GetSettlementHistoryByCurrency(ctx context.Context, req *DerivativesSettlementHistoryByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	return w.e.GlobalDerivatives().getSettlementHistoryByCurrency(ctx, true, req)
}

// GetSettlementHistoryByInstrument returns an instrument's settlement events as
// GlobalDerivativesAPI.GetSettlementHistoryByInstrument does, over the websocket
func (w WebsocketAPI) GetSettlementHistoryByInstrument(ctx context.Context, req *DerivativesSettlementHistoryByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	return w.e.GlobalDerivatives().getSettlementHistoryByInstrument(ctx, true, req)
}

// GetTriggerOrderHistory returns the trigger order log as GlobalDerivativesAPI.GetTriggerOrderHistory does, over the
// websocket
func (w WebsocketAPI) GetTriggerOrderHistory(ctx context.Context, req *DerivativesTriggerOrderHistoryRequest) (*DerivativesTriggerOrderHistoryResponse, error) {
	return w.e.GlobalDerivatives().getTriggerOrderHistory(ctx, true, req)
}

// GetUserTradesByCurrency returns a currency's trades as GlobalDerivativesAPI.GetUserTradesByCurrency does, over the
// websocket
func (w WebsocketAPI) GetUserTradesByCurrency(ctx context.Context, req *DerivativesUserTradesByCurrencyRequest) (*DerivativesUserTradesResponse, error) {
	return w.e.GlobalDerivatives().getUserTradesByCurrency(ctx, true, req)
}

// GetUserTradesByCurrencyAndTime returns a currency's trades within a time range as
// GlobalDerivativesAPI.GetUserTradesByCurrencyAndTime does, over the websocket
func (w WebsocketAPI) GetUserTradesByCurrencyAndTime(ctx context.Context, req *DerivativesUserTradesByCurrencyAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	return w.e.GlobalDerivatives().getUserTradesByCurrencyAndTime(ctx, true, req)
}

// GetUserTradesByInstrument returns an instrument's trades as GlobalDerivativesAPI.GetUserTradesByInstrument does, over
// the websocket
func (w WebsocketAPI) GetUserTradesByInstrument(ctx context.Context, req *DerivativesUserTradesByInstrumentRequest) (*DerivativesUserTradesResponse, error) {
	return w.e.GlobalDerivatives().getUserTradesByInstrument(ctx, true, req)
}

// GetUserTradesByInstrumentAndTime returns an instrument's trades within a time range as
// GlobalDerivativesAPI.GetUserTradesByInstrumentAndTime does, over the websocket
func (w WebsocketAPI) GetUserTradesByInstrumentAndTime(ctx context.Context, req *DerivativesUserTradesByInstrumentAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	return w.e.GlobalDerivatives().getUserTradesByInstrumentAndTime(ctx, true, req)
}

// GetUserTradesByOrder returns an order's trades as GlobalDerivativesAPI.GetUserTradesByOrder does, over the websocket
func (w WebsocketAPI) GetUserTradesByOrder(ctx context.Context, req *DerivativesUserTradesByOrderRequest) ([]DerivativesUserTrade, error) {
	return w.e.GlobalDerivatives().getUserTradesByOrder(ctx, true, req)
}

// Sell places a sell order as GlobalDerivativesAPI.Sell does, over the websocket
func (w WebsocketAPI) Sell(ctx context.Context, req *DerivativesOrderRequest) (*DerivativesOrderResponse, error) {
	return w.e.GlobalDerivatives().placeOrder(ctx, true, derivativesSellMethod, req)
}

// SetLeverage sets or removes a future's custom leverage as GlobalDerivativesAPI.SetLeverage does, over the websocket
func (w WebsocketAPI) SetLeverage(ctx context.Context, req *DerivativesSetLeverageRequest) (float64, error) {
	return w.e.GlobalDerivatives().setLeverage(ctx, true, req)
}

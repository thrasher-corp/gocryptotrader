package deribit

import (
	"context"
)

// Buy places a buy order as Exchange.Buy does, over the websocket
func (w WebsocketAPI) Buy(ctx context.Context, arg *OrderRequest) (*OrderWithTradesResponse, error) {
	return w.e.placeOrder(ctx, "private/buy", arg, true)
}

// CancelOrderByID cancels an order as Exchange.CancelOrderByID does, over the websocket
func (w WebsocketAPI) CancelOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	return w.e.cancelOrderByID(ctx, orderID, true)
}

// CancelAll cancels every order as Exchange.CancelAll does, over the websocket
func (w WebsocketAPI) CancelAll(ctx context.Context, arg *CancelAllRequest) (*CancelAllResponse, error) {
	return w.e.cancelAll(ctx, arg, true)
}

// CancelAllByCurrency cancels a currency's orders as Exchange.CancelAllByCurrency does, over the websocket
func (w WebsocketAPI) CancelAllByCurrency(ctx context.Context, arg *CancelAllByCurrencyRequest) (*CancelAllResponse, error) {
	return w.e.cancelAllByCurrency(ctx, arg, true)
}

// CancelAllByCurrencyPair cancels a currency pair's orders as Exchange.CancelAllByCurrencyPair does, over the websocket
func (w WebsocketAPI) CancelAllByCurrencyPair(ctx context.Context, arg *CancelAllByCurrencyPairRequest) (*CancelAllResponse, error) {
	return w.e.cancelAllByCurrencyPair(ctx, arg, true)
}

// CancelAllByInstrument cancels an instrument's orders as Exchange.CancelAllByInstrument does, over the websocket
func (w WebsocketAPI) CancelAllByInstrument(ctx context.Context, arg *CancelAllByInstrumentRequest) (*CancelAllResponse, error) {
	return w.e.cancelAllByInstrument(ctx, arg, true)
}

// CancelAllByKindOrType cancels orders as Exchange.CancelAllByKindOrType does, over the websocket
func (w WebsocketAPI) CancelAllByKindOrType(ctx context.Context, arg *CancelAllByKindOrTypeRequest) (*CancelAllResponse, error) {
	return w.e.cancelAllByKindOrType(ctx, arg, true)
}

// CancelByLabel cancels orders with a label as Exchange.CancelByLabel does, over the websocket
func (w WebsocketAPI) CancelByLabel(ctx context.Context, arg *CancelByLabelRequest) (*CancelAllResponse, error) {
	return w.e.cancelByLabel(ctx, arg, true)
}

// CancelQuotes cancels mass quotes as Exchange.CancelQuotes does, over the websocket
func (w WebsocketAPI) CancelQuotes(ctx context.Context, arg *CancelQuotesRequest) (*CancelAllResponse, error) {
	return w.e.cancelQuotes(ctx, arg, true)
}

// ClosePosition closes a position as Exchange.ClosePosition does, over the websocket
func (w WebsocketAPI) ClosePosition(ctx context.Context, arg *ClosePositionRequest) (*OrderWithTradesResponse, error) {
	return w.e.closePosition(ctx, arg, true)
}

// EditOrder edits an order as Exchange.EditOrder does, over the websocket
func (w WebsocketAPI) EditOrder(ctx context.Context, arg *EditOrderRequest) (*OrderWithTradesResponse, error) {
	return w.e.editOrder(ctx, arg, true)
}

// EditOrderByLabel edits an order by its label as Exchange.EditOrderByLabel does, over the websocket
func (w WebsocketAPI) EditOrderByLabel(ctx context.Context, arg *EditOrderByLabelRequest) (*OrderWithTradesResponse, error) {
	return w.e.editOrderByLabel(ctx, arg, true)
}

// GetMargins returns a hypothetical order's margins as Exchange.GetMargins does, over the websocket
func (w WebsocketAPI) GetMargins(ctx context.Context, arg *MarginsRequest) (*MarginsResponse, error) {
	return w.e.getMargins(ctx, arg, true)
}

// GetMMPConfig returns MMP configurations as Exchange.GetMMPConfig does, over the websocket
func (w WebsocketAPI) GetMMPConfig(ctx context.Context, arg *MMPGroupRequest) ([]MMPConfigResponse, error) {
	return w.e.getMMPConfig(ctx, arg, true)
}

// GetMMPStatus returns triggered MMP groups as Exchange.GetMMPStatus does, over the websocket
func (w WebsocketAPI) GetMMPStatus(ctx context.Context, arg *MMPGroupRequest) ([]MMPStatusResponse, error) {
	return w.e.getMMPStatus(ctx, arg, true)
}

// GetOpenOrders returns the account's open orders as Exchange.GetOpenOrders does, over the websocket
func (w WebsocketAPI) GetOpenOrders(ctx context.Context, arg *OpenOrdersRequest) ([]OrderResponse, error) {
	return w.e.getOpenOrders(ctx, arg, true)
}

// GetOpenOrdersByCurrency returns a currency's open orders as Exchange.GetOpenOrdersByCurrency does, over the websocket
func (w WebsocketAPI) GetOpenOrdersByCurrency(ctx context.Context, arg *OpenOrdersByCurrencyRequest) ([]OrderResponse, error) {
	return w.e.getOpenOrdersByCurrency(ctx, arg, true)
}

// GetOpenOrdersByInstrument returns an instrument's open orders as Exchange.GetOpenOrdersByInstrument does, over the
// websocket
func (w WebsocketAPI) GetOpenOrdersByInstrument(ctx context.Context, arg *OpenOrdersByInstrumentRequest) ([]OrderResponse, error) {
	return w.e.getOpenOrdersByInstrument(ctx, arg, true)
}

// GetOpenOrdersByLabel returns the open orders with a label as Exchange.GetOpenOrdersByLabel does, over the websocket
func (w WebsocketAPI) GetOpenOrdersByLabel(ctx context.Context, arg *OrdersByLabelRequest) ([]OrderResponse, error) {
	return w.e.getOrdersByLabel(ctx, "private/get_open_orders_by_label", arg, true)
}

// GetOrderHistoryByCurrency returns a currency's order history as Exchange.GetOrderHistoryByCurrency does, over the
// websocket
func (w WebsocketAPI) GetOrderHistoryByCurrency(ctx context.Context, arg *OrderHistoryByCurrencyRequest) (*OrderHistoryResponse, error) {
	return w.e.getOrderHistoryByCurrency(ctx, arg, true)
}

// GetOrderHistoryByInstrument returns an instrument's order history as Exchange.GetOrderHistoryByInstrument does, over
// the websocket
func (w WebsocketAPI) GetOrderHistoryByInstrument(ctx context.Context, arg *OrderHistoryByInstrumentRequest) (*OrderHistoryResponse, error) {
	return w.e.getOrderHistoryByInstrument(ctx, arg, true)
}

// GetOrderMarginByIDs returns the orders' initial margins as Exchange.GetOrderMarginByIDs does, over the websocket
func (w WebsocketAPI) GetOrderMarginByIDs(ctx context.Context, orderIDs []string) ([]OrderMarginResponse, error) {
	return w.e.getOrderMarginByIDs(ctx, orderIDs, true)
}

// GetOrderState returns an order as Exchange.GetOrderState does, over the websocket
func (w WebsocketAPI) GetOrderState(ctx context.Context, orderID string) (*OrderResponse, error) {
	return w.e.getOrderState(ctx, orderID, true)
}

// GetOrderStateByLabel returns the orders with a label as Exchange.GetOrderStateByLabel does, over the websocket
func (w WebsocketAPI) GetOrderStateByLabel(ctx context.Context, arg *OrdersByLabelRequest) ([]OrderResponse, error) {
	return w.e.getOrdersByLabel(ctx, "private/get_order_state_by_label", arg, true)
}

// GetSettlementHistoryByCurrency returns a currency's settlement events as Exchange.GetSettlementHistoryByCurrency
// does, over the websocket
func (w WebsocketAPI) GetSettlementHistoryByCurrency(ctx context.Context, arg *SettlementHistoryByCurrencyRequest) (*SettlementsResponse, error) {
	return w.e.getSettlementHistoryByCurrency(ctx, arg, true)
}

// GetSettlementHistoryByInstrument returns an instrument's settlement events as
// Exchange.GetSettlementHistoryByInstrument does, over the websocket
func (w WebsocketAPI) GetSettlementHistoryByInstrument(ctx context.Context, arg *SettlementHistoryByInstrumentRequest) (*SettlementsResponse, error) {
	return w.e.getSettlementHistoryByInstrument(ctx, arg, true)
}

// GetTriggerOrderHistory returns the trigger order log as Exchange.GetTriggerOrderHistory does, over the websocket
func (w WebsocketAPI) GetTriggerOrderHistory(ctx context.Context, arg *TriggerOrderHistoryRequest) (*TriggerOrderHistoryResponse, error) {
	return w.e.getTriggerOrderHistory(ctx, arg, true)
}

// GetUserTradesByCurrency returns a currency's trades as Exchange.GetUserTradesByCurrency does, over the websocket
func (w WebsocketAPI) GetUserTradesByCurrency(ctx context.Context, arg *UserTradesByCurrencyRequest) (*UserTradesResponse, error) {
	return w.e.getUserTradesByCurrency(ctx, arg, true)
}

// GetUserTradesByCurrencyAndTime returns a currency's trades within a time range as
// Exchange.GetUserTradesByCurrencyAndTime does, over the websocket
func (w WebsocketAPI) GetUserTradesByCurrencyAndTime(ctx context.Context, arg *UserTradesByCurrencyAndTimeRequest) (*UserTradesResponse, error) {
	return w.e.getUserTradesByCurrencyAndTime(ctx, arg, true)
}

// GetUserTradesByInstrument returns an instrument's trades as Exchange.GetUserTradesByInstrument does, over the
// websocket
func (w WebsocketAPI) GetUserTradesByInstrument(ctx context.Context, arg *UserTradesByInstrumentRequest) (*UserTradesResponse, error) {
	return w.e.getUserTradesByInstrument(ctx, arg, true)
}

// GetUserTradesByInstrumentAndTime returns an instrument's trades within a time range as
// Exchange.GetUserTradesByInstrumentAndTime does, over the websocket
func (w WebsocketAPI) GetUserTradesByInstrumentAndTime(ctx context.Context, arg *UserTradesByInstrumentAndTimeRequest) (*UserTradesResponse, error) {
	return w.e.getUserTradesByInstrumentAndTime(ctx, arg, true)
}

// GetUserTradesByOrder returns an order's trades as Exchange.GetUserTradesByOrder does, over the websocket
func (w WebsocketAPI) GetUserTradesByOrder(ctx context.Context, arg *UserTradesByOrderRequest) ([]UserTradeResponse, error) {
	return w.e.getUserTradesByOrder(ctx, arg, true)
}

// MassQuote places or amends quotes as Exchange.MassQuote does, over the websocket, whose connection can enable cancel
// on disconnect
func (w WebsocketAPI) MassQuote(ctx context.Context, arg *MassQuoteRequest) (*MassQuoteResponse, error) {
	return w.e.massQuote(ctx, arg, true)
}

// MovePositions moves positions as Exchange.MovePositions does, over the websocket
func (w WebsocketAPI) MovePositions(ctx context.Context, arg *MovePositionsRequest) ([]PositionMoveTradeResponse, error) {
	return w.e.movePositions(ctx, arg, true)
}

// ResetMMP resets triggered MMP as Exchange.ResetMMP does, over the websocket
func (w WebsocketAPI) ResetMMP(ctx context.Context, arg *MMPGroupRequest) error {
	return w.e.resetMMP(ctx, arg, true)
}

// Sell places a sell order as Exchange.Sell does, over the websocket
func (w WebsocketAPI) Sell(ctx context.Context, arg *OrderRequest) (*OrderWithTradesResponse, error) {
	return w.e.placeOrder(ctx, "private/sell", arg, true)
}

// SetMMPConfig sets an MMP configuration as Exchange.SetMMPConfig does, over the websocket
func (w WebsocketAPI) SetMMPConfig(ctx context.Context, arg *SetMMPConfigRequest) ([]MMPConfigResponse, error) {
	return w.e.setMMPConfig(ctx, arg, true)
}

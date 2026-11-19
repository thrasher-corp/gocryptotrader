package binanceus

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetCurrentOrderCountUsage returns the account's unfilled order count for every interval as
// Exchange.GetCurrentOrderCountUsage does, over the WebSocket API
func (w WebsocketAPI) GetCurrentOrderCountUsage(ctx context.Context) ([]RateLimitItemResponse, error) {
	return w.e.getCurrentOrderCountUsage(ctx, true)
}

// NewOrder places an order as Exchange.NewOrder does, over the WebSocket API
func (w WebsocketAPI) NewOrder(ctx context.Context, req *NewOrderRequest) (*NewOrderResponse, error) {
	return w.e.newOrder(ctx, req, true)
}

// NewOrderTest tests an order without sending it to the matching engine as Exchange.NewOrderTest does, over the
// WebSocket API
func (w WebsocketAPI) NewOrderTest(ctx context.Context, req *NewOrderRequest) error {
	return w.e.newOrderTest(ctx, req, true)
}

// QueryOrder returns an order as Exchange.QueryOrder does, over the WebSocket API. Unlike REST, the WebSocket API
// ignores origClientOrderID when orderID is set
func (w WebsocketAPI) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID uint64) (*TradeOrderResponse, error) {
	return w.e.queryOrder(ctx, symbol, origClientOrderID, orderID, queryOrderRate, true)
}

// OpenOrders returns open orders as Exchange.OpenOrders does, over the WebSocket API
func (w WebsocketAPI) OpenOrders(ctx context.Context, pair currency.Pair) ([]TradeOrderResponse, error) {
	return w.e.openOrders(ctx, pair, openOrdersAllRate, true)
}

// CancelExistingOrder cancels an active order as Exchange.CancelExistingOrder does, over the WebSocket API. Unlike
// REST, the WebSocket API ignores OriginalClientOrderID when OrderID is set
func (w WebsocketAPI) CancelExistingOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	return w.e.cancelExistingOrder(ctx, req, true)
}

// CancelAllOpenOrderOnSymbol cancels every open order on a symbol as Exchange.CancelAllOpenOrderOnSymbol does, over the
// WebSocket API
func (w WebsocketAPI) CancelAllOpenOrderOnSymbol(ctx context.Context, symbol currency.Pair) ([]CancelledOpenOrderResponse, error) {
	return w.e.cancelAllOpenOrderOnSymbol(ctx, symbol, true)
}

// GetAccountTradeList returns the account's trades on a symbol as Exchange.GetAccountTradeList does, over the WebSocket
// API
func (w WebsocketAPI) GetAccountTradeList(ctx context.Context, req *AccountTradeListRequest) ([]AccountTradeResponse, error) {
	return w.e.getAccountTradeList(ctx, req, true)
}

// CancelExistingOrderAndSendNewOrder cancels an order and places a new one as
// Exchange.CancelExistingOrderAndSendNewOrder does, over the WebSocket API
func (w WebsocketAPI) CancelExistingOrderAndSendNewOrder(ctx context.Context, req *CancelReplaceOrderRequest) (*CancelAndReplaceResponse, error) {
	return w.e.cancelExistingOrderAndSendNewOrder(ctx, req, true)
}

// GetPreventedMatches returns the orders that expired because of self-trade prevention as Exchange.GetPreventedMatches
// does, over the WebSocket API
func (w WebsocketAPI) GetPreventedMatches(ctx context.Context, req *PreventedMatchesRequest) ([]PreventedMatchResponse, error) {
	return w.e.getPreventedMatches(ctx, req, true)
}

// AllOrders returns the account's orders on a symbol as Exchange.AllOrders does, over the WebSocket API
func (w WebsocketAPI) AllOrders(ctx context.Context, req *AllOrdersRequest) ([]TradeOrderResponse, error) {
	return w.e.allOrders(ctx, req, true)
}

// NewOneCancelsTheOtherOrder places an OCO order list as Exchange.NewOneCancelsTheOtherOrder does, over the WebSocket
// API. Unlike REST, the WebSocket API takes TrailingDeltaInBasisPoints in place of StopPrice
func (w WebsocketAPI) NewOneCancelsTheOtherOrder(ctx context.Context, req *OneCancelsTheOtherOrderRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return w.e.newOneCancelsTheOtherOrder(ctx, req, true)
}

// GetOneCancelsTheOtherOrders returns an order list as Exchange.GetOneCancelsTheOtherOrders does, over the WebSocket
// API
func (w WebsocketAPI) GetOneCancelsTheOtherOrders(ctx context.Context, req *QueryOrderListRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return w.e.getOneCancelsTheOtherOrders(ctx, req, queryOrderListRate, true)
}

// GetAllOneCancelsTheOtherOrders returns the account's order lists as Exchange.GetAllOneCancelsTheOtherOrders does,
// over the WebSocket API
func (w WebsocketAPI) GetAllOneCancelsTheOtherOrders(ctx context.Context, req *AllOrderListsRequest) ([]OneCancelsTheOtherOrderResponse, error) {
	return w.e.getAllOneCancelsTheOtherOrders(ctx, req, true)
}

// GetOpenOneCancelsTheOtherList returns the account's open order lists as Exchange.GetOpenOneCancelsTheOtherList does,
// over the WebSocket API
func (w WebsocketAPI) GetOpenOneCancelsTheOtherList(ctx context.Context) ([]OneCancelsTheOtherOrderResponse, error) {
	return w.e.getOpenOneCancelsTheOtherList(ctx, true)
}

// CancelOneCancelsTheOtherOrder cancels an order list as Exchange.CancelOneCancelsTheOtherOrder does, over the
// WebSocket API
func (w WebsocketAPI) CancelOneCancelsTheOtherOrder(ctx context.Context, req *CancelOrderListRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return w.e.cancelOneCancelsTheOtherOrder(ctx, req, true)
}

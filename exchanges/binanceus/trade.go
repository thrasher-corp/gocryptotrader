package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errCancelReplaceModeRequired        = errors.New("cancel replace mode required")
	errStopLimitTimeInForceRequired     = errors.New("stop limit time in force required with a stop limit price")
	errStopPriceRequired                = errors.New("stop price required")
	errStopPriceOrTrailingDeltaRequired = errors.New("stop price or trailing delta required")
)

// GetCurrentOrderCountUsage returns the account's unfilled order count for every interval
func (e *Exchange) GetCurrentOrderCountUsage(ctx context.Context) ([]RateLimitItemResponse, error) {
	return e.getCurrentOrderCountUsage(ctx, false)
}

func (e *Exchange) getCurrentOrderCountUsage(ctx context.Context, useWebsocket bool) ([]RateLimitItemResponse, error) {
	var resp []RateLimitItemResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/rateLimit/order", "account.rateLimits.orders", nil, orderRateLimitRate, &resp)
}

// NewOrder sends in a new order
func (e *Exchange) NewOrder(ctx context.Context, req *NewOrderRequest) (*NewOrderResponse, error) {
	return e.newOrder(ctx, req, false)
}

func (e *Exchange) newOrder(ctx context.Context, req *NewOrderRequest, useWebsocket bool) (*NewOrderResponse, error) {
	params, err := e.newOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *NewOrderResponse
	return resp, e.sendSignedRequest(request.WithRetryNotAllowed(ctx), useWebsocket, http.MethodPost, "/api/v3/order", "order.place", params, orderRate, &resp)
}

// newOrderParams validates a new order and builds its parameters, which a test order takes too. Every parameter a
// request sets is sent, whatever its type, so the venue rejects one the type does not take rather than the order being
// placed without it
func (e *Exchange) newOrderParams(req *NewOrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	if req.Quantity > 0 {
		params.Set("quantity", formatFloat(req.Quantity))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", formatFloat(req.QuoteOrderQuantity))
	}
	if req.Price > 0 {
		params.Set("price", formatFloat(req.Price))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", formatFloat(req.StopPrice))
	}
	if req.TrailingDeltaInBasisPoints != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDeltaInBasisPoints, 10))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", formatFloat(req.IcebergQuantity))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(req.NewOrderResponseType))
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(req.SelfTradePreventionMode))
	}
	return params, nil
}

// NewOrderTest tests new order creation and the request's signature and recvWindow without sending the order to the
// matching engine
func (e *Exchange) NewOrderTest(ctx context.Context, req *NewOrderRequest) error {
	return e.newOrderTest(ctx, req, false)
}

func (e *Exchange) newOrderTest(ctx context.Context, req *NewOrderRequest, useWebsocket bool) error {
	params, err := e.newOrderParams(req)
	if err != nil {
		return err
	}
	return e.sendSignedRequest(ctx, useWebsocket, http.MethodPost, "/api/v3/order/test", "order.test", params, defaultRate, nil)
}

// QueryOrder returns an order by orderID or origClientOrderID; when both are sent they must name the same order
func (e *Exchange) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID uint64) (*TradeOrderResponse, error) {
	return e.queryOrder(ctx, symbol, origClientOrderID, orderID, queryOrderRate, false)
}

// queryOrder returns an order drawing on f, so the lookup a cancellation makes can draw on the cancel pool
func (e *Exchange) queryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID uint64, f request.EndpointLimit, useWebsocket bool) (*TradeOrderResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *TradeOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/order", "order.status", params, f, &resp)
}

// OpenOrders returns the current open orders of a symbol, or of every symbol for an empty pair, which weighs far more
func (e *Exchange) OpenOrders(ctx context.Context, pair currency.Pair) ([]TradeOrderResponse, error) {
	return e.openOrders(ctx, pair, openOrdersAllRate, false)
}

// openOrders lists open orders, drawing on allSymbols for a listing of every symbol's, so the listing a cancellation of
// every symbol's orders makes can draw on the cancel pool
func (e *Exchange) openOrders(ctx context.Context, pair currency.Pair, allSymbols request.EndpointLimit, useWebsocket bool) ([]TradeOrderResponse, error) {
	params := url.Values{}
	rateLimit := allSymbols
	if !pair.IsEmpty() {
		symbol, err := e.formatSymbol(pair)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
		rateLimit = openOrdersRate
	}
	var resp []TradeOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/openOrders", "openOrders.status", params, rateLimit, &resp)
}

// CancelExistingOrder cancels an active order by OrderID or OriginalClientOrderID. When both are set both are sent, and
// Binance.US cancels the order only when they name the same one
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	return e.cancelExistingOrder(ctx, req, false)
}

func (e *Exchange) cancelExistingOrder(ctx context.Context, req *CancelOrderRequest, useWebsocket bool) (*CancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("origClientOrderId", req.OriginalClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", string(req.CancelRestrictions))
	}
	var resp *CancelOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodDelete, "/api/v3/order", "order.cancel", params, cancelRate, &resp)
}

// CancelAllOpenOrderOnSymbol cancels all open orders on a symbol, including orders that are part of an order list
func (e *Exchange) CancelAllOpenOrderOnSymbol(ctx context.Context, symbol currency.Pair) ([]CancelledOpenOrderResponse, error) {
	return e.cancelAllOpenOrderOnSymbol(ctx, symbol, false)
}

func (e *Exchange) cancelAllOpenOrderOnSymbol(ctx context.Context, symbol currency.Pair, useWebsocket bool) ([]CancelledOpenOrderResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp []CancelledOpenOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodDelete, "/api/v3/openOrders", "openOrders.cancelAll", params, cancelRate, &resp)
}

// GetAccountTradeList returns the account's trades on a symbol
func (e *Exchange) GetAccountTradeList(ctx context.Context, req *AccountTradeListRequest) ([]AccountTradeResponse, error) {
	return e.getAccountTradeList(ctx, req, false)
}

func (e *Exchange) getAccountTradeList(ctx context.Context, req *AccountTradeListRequest, useWebsocket bool) ([]AccountTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if (req.OrderID != 0 || req.FromID != 0) && (!req.StartTime.IsZero() || !req.EndTime.IsZero()) {
		return nil, fmt.Errorf("%w: orderId and fromId cannot be sent with startTime or endTime", errInvalidParameterCombination)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []AccountTradeResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/myTrades", "myTrades", params, accountTradesLimit(req.OrderID), &resp)
}

// CancelExistingOrderAndSendNewOrder cancels an existing order and places a new order on the same symbol. Filters and
// the unfilled order count are evaluated before either happens. When a half fails the returned error carries the
// outcome of both halves, which the returned response then holds too, so a caller can learn whether a new order was
// still placed
func (e *Exchange) CancelExistingOrderAndSendNewOrder(ctx context.Context, req *CancelReplaceOrderRequest) (*CancelAndReplaceResponse, error) {
	return e.cancelExistingOrderAndSendNewOrder(ctx, req, false)
}

func (e *Exchange) cancelExistingOrderAndSendNewOrder(ctx context.Context, req *CancelReplaceOrderRequest, useWebsocket bool) (*CancelAndReplaceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.CancelReplaceMode == "" {
		return nil, errCancelReplaceModeRequired
	}
	if req.CancelOrderID == 0 && req.CancelOriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	params.Set("cancelReplaceMode", string(req.CancelReplaceMode))
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	if req.Quantity > 0 {
		params.Set("quantity", formatFloat(req.Quantity))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", formatFloat(req.QuoteOrderQuantity))
	}
	if req.Price > 0 {
		params.Set("price", formatFloat(req.Price))
	}
	if req.CancelNewClientOrderID != "" {
		params.Set("cancelNewClientOrderId", req.CancelNewClientOrderID)
	}
	if req.CancelOriginalClientOrderID != "" {
		params.Set("cancelOrigClientOrderId", req.CancelOriginalClientOrderID)
	}
	if req.CancelOrderID != 0 {
		params.Set("cancelOrderId", strconv.FormatUint(req.CancelOrderID, 10))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StrategyID != 0 {
		params.Set("strategyId", strconv.FormatUint(req.StrategyID, 10))
	}
	if req.StrategyType != 0 {
		params.Set("strategyType", strconv.FormatUint(req.StrategyType, 10))
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", formatFloat(req.StopPrice))
	}
	if req.TrailingDeltaInBasisPoints != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDeltaInBasisPoints, 10))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", formatFloat(req.IcebergQuantity))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(req.NewOrderResponseType))
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(req.SelfTradePreventionMode))
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", string(req.CancelRestrictions))
	}
	var resp *CancelAndReplaceResponse
	err = e.sendSignedRequest(request.WithRetryNotAllowed(ctx), useWebsocket, http.MethodPost, "/api/v3/order/cancelReplace", "order.cancelReplace", params, orderRate, &resp)
	return cancelReplaceOutcome(resp, err)
}

// cancelReplaceOutcome returns a cancel-replace's outcome with its error. When a half fails, Binance.US puts the outcome
// of both halves in the error's data, which the caller needs to learn whether a new order was still placed
func cancelReplaceOutcome[T any](resp *T, err error) (*T, error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Data) == 0 {
		return resp, err
	}
	if decodeErr := json.Unmarshal(apiErr.Data, &resp); decodeErr != nil {
		return nil, common.AppendError(err, fmt.Errorf("error decoding cancel-replace outcome: %w", decodeErr))
	}
	return resp, err
}

// GetPreventedMatches returns the orders that expired because of self-trade prevention (STP), by PreventedMatchID or by
// OrderID
func (e *Exchange) GetPreventedMatches(ctx context.Context, req *PreventedMatchesRequest) ([]PreventedMatchResponse, error) {
	return e.getPreventedMatches(ctx, req, false)
}

func (e *Exchange) getPreventedMatches(ctx context.Context, req *PreventedMatchesRequest, useWebsocket bool) ([]PreventedMatchResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.PreventedMatchID == 0 && req.OrderID == 0 {
		return nil, fmt.Errorf("%w: preventedMatchId or orderId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.PreventedMatchID != 0 {
		params.Set("preventedMatchId", strconv.FormatUint(req.PreventedMatchID, 10))
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.FromPreventedMatchID != 0 {
		params.Set("fromPreventedMatchId", strconv.FormatUint(req.FromPreventedMatchID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []PreventedMatchResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/myPreventedMatches", "myPreventedMatches", params, preventedMatchesLimit(req.OrderID), &resp)
}

// AllOrders returns the account's orders on a symbol: active, cancelled or filled
func (e *Exchange) AllOrders(ctx context.Context, req *AllOrdersRequest) ([]TradeOrderResponse, error) {
	return e.allOrders(ctx, req, false)
}

func (e *Exchange) allOrders(ctx context.Context, req *AllOrdersRequest, useWebsocket bool) ([]TradeOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []TradeOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/allOrders", "allOrders", params, allOrdersRate, &resp)
}

// NewOneCancelsTheOtherOrder places a one-cancels-the-other (OCO) order list of a LIMIT_MAKER order and a STOP_LOSS or
// STOP_LOSS_LIMIT order, where the activation of one immediately cancels the other. An OCO counts as two orders against
// the order rate limit
func (e *Exchange) NewOneCancelsTheOtherOrder(ctx context.Context, req *OneCancelsTheOtherOrderRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return e.newOneCancelsTheOtherOrder(ctx, req, false)
}

func (e *Exchange) newOneCancelsTheOtherOrder(ctx context.Context, req *OneCancelsTheOtherOrderRequest, useWebsocket bool) (*OneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	// REST requires a stop price, which the WebSocket API takes a trailing delta in place of
	switch {
	case useWebsocket && req.StopPrice <= 0 && req.TrailingDeltaInBasisPoints == 0:
		return nil, errStopPriceOrTrailingDeltaRequired
	case !useWebsocket && req.StopPrice <= 0:
		return nil, errStopPriceRequired
	}
	if req.StopLimitPrice > 0 && req.StopLimitTimeInForce == "" {
		return nil, errStopLimitTimeInForceRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", string(req.Side))
	params.Set("quantity", formatFloat(req.Quantity))
	params.Set("price", formatFloat(req.Price))
	if req.StopPrice > 0 {
		params.Set("stopPrice", formatFloat(req.StopPrice))
	}
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", req.LimitClientOrderID)
	}
	if req.LimitIcebergQuantity > 0 {
		params.Set("limitIcebergQty", formatFloat(req.LimitIcebergQuantity))
	}
	if req.TrailingDeltaInBasisPoints != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDeltaInBasisPoints, 10))
	}
	if req.StopClientOrderID != "" {
		params.Set("stopClientOrderId", req.StopClientOrderID)
	}
	if req.StopLimitPrice > 0 {
		params.Set("stopLimitPrice", formatFloat(req.StopLimitPrice))
	}
	if req.StopIcebergQuantity > 0 {
		params.Set("stopIcebergQty", formatFloat(req.StopIcebergQuantity))
	}
	if req.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", string(req.StopLimitTimeInForce))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(req.NewOrderResponseType))
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(req.SelfTradePreventionMode))
	}
	var resp *OneCancelsTheOtherOrderResponse
	return resp, e.sendSignedRequest(request.WithRetryNotAllowed(ctx), useWebsocket, http.MethodPost, "/api/v3/order/oco", "orderList.place", params, oneCancelsTheOtherOrderRate, &resp)
}

// GetOneCancelsTheOtherOrders returns an order list by its order list ID or by its list client order ID
func (e *Exchange) GetOneCancelsTheOtherOrders(ctx context.Context, req *QueryOrderListRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return e.getOneCancelsTheOtherOrders(ctx, req, queryOrderListRate, false)
}

// getOneCancelsTheOtherOrders returns an order list drawing on f, so the lookup a cancellation makes can draw on the
// cancel pool
func (e *Exchange) getOneCancelsTheOtherOrders(ctx context.Context, req *QueryOrderListRequest, f request.EndpointLimit, useWebsocket bool) (*OneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderListID == nil && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if req.OrderListID != nil {
		params.Set("orderListId", strconv.FormatUint(*req.OrderListID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("origClientOrderId", req.OriginalClientOrderID)
	}
	var resp *OneCancelsTheOtherOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/orderList", "orderList.status", params, f, &resp)
}

// GetAllOneCancelsTheOtherOrders returns the account's order lists. A nil request returns the most recent order lists
func (e *Exchange) GetAllOneCancelsTheOtherOrders(ctx context.Context, req *AllOrderListsRequest) ([]OneCancelsTheOtherOrderResponse, error) {
	return e.getAllOneCancelsTheOtherOrders(ctx, req, false)
}

func (e *Exchange) getAllOneCancelsTheOtherOrders(ctx context.Context, req *AllOrderListsRequest, useWebsocket bool) ([]OneCancelsTheOtherOrderResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.FromID != 0 && (!req.StartTime.IsZero() || !req.EndTime.IsZero()) {
			return nil, fmt.Errorf("%w: fromId cannot be sent with startTime or endTime", errInvalidParameterCombination)
		}
		if req.FromID != 0 {
			params.Set("fromId", strconv.FormatUint(req.FromID, 10))
		}
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp []OneCancelsTheOtherOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/allOrderList", "allOrderLists", params, allOrderListsRate, &resp)
}

// GetOpenOneCancelsTheOtherList returns the account's open order lists
func (e *Exchange) GetOpenOneCancelsTheOtherList(ctx context.Context) ([]OneCancelsTheOtherOrderResponse, error) {
	return e.getOpenOneCancelsTheOtherList(ctx, false)
}

func (e *Exchange) getOpenOneCancelsTheOtherList(ctx context.Context, useWebsocket bool) ([]OneCancelsTheOtherOrderResponse, error) {
	var resp []OneCancelsTheOtherOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/openOrderList", "openOrderLists.status", nil, openOrderListsRate, &resp)
}

// CancelOneCancelsTheOtherOrder cancels an entire order list. Cancelling any order of a list cancels the whole list
func (e *Exchange) CancelOneCancelsTheOtherOrder(ctx context.Context, req *CancelOrderListRequest) (*OneCancelsTheOtherOrderResponse, error) {
	return e.cancelOneCancelsTheOtherOrder(ctx, req, false)
}

func (e *Exchange) cancelOneCancelsTheOtherOrder(ctx context.Context, req *CancelOrderListRequest, useWebsocket bool) (*OneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderListID == nil && req.ListClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderListID != nil {
		params.Set("orderListId", strconv.FormatUint(*req.OrderListID, 10))
	}
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *OneCancelsTheOtherOrderResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodDelete, "/api/v3/orderList", "orderList.cancel", params, cancelRate, &resp)
}

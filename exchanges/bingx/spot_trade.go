package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSpotOrderIDMissing          = errors.New("response holds no order ID")
	errSpotTooManyOrders           = errors.New("too many orders")
	errSpotCancelReplaceModeEmpty  = errors.New("cancel replace mode empty")
	errSpotPageLimitExceeded       = errors.New("page limit exceeded")
	errSpotPageIndexRequired       = errors.New("page index required")
	errSpotPageSizeRequired        = errors.New("page size required")
	errSpotCancelAllAfterTypeEmpty = errors.New("cancel all after type empty")
	errSpotInvalidTimeout          = errors.New("timeout must be a positive whole number of seconds")
	errSpotPriceRequired           = errors.New("price required")
)

// Spot order history paging limits: the largest page, and the furthest record a page can reach
const (
	spotOrderHistoryMaxPageSize   = 100
	spotOrderHistoryMaxPageWindow = 10000
)

// PlaceSpotOrder places a spot order. A limit order needs a price, and a quantity or quote order quantity; a market buy
// takes a quote order quantity and a market sell a quantity. Spot copy traders sell through the copy trading endpoint
func (e *Exchange) PlaceSpotOrder(ctx context.Context, req *PlaceSpotOrderRequest) (*SpotOrderResponse, error) {
	if err := spotValidateOrder(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	if req.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(req.QuoteOrderQuantity, 'f', -1, 64))
	}
	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	var resp *SpotOrderResponse
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/order", params, spotPlaceOrderRate, &resp); err != nil {
		return nil, err
	}
	if resp.OrderID == 0 {
		return nil, errSpotOrderIDMissing
	}
	return resp, nil
}

// PlaceSpotBatchOrders places up to five spot orders. Without Sync they are placed in parallel and must share a symbol,
// side and type
func (e *Exchange) PlaceSpotBatchOrders(ctx context.Context, req *PlaceSpotBatchOrdersRequest) ([]SpotOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(req.Orders) > spotBatchOrdersLimit {
		return nil, fmt.Errorf("%w: %d orders, at most %d", errSpotTooManyOrders, len(req.Orders), spotBatchOrdersLimit)
	}
	items := make([]SpotBatchOrderItemRequest, len(req.Orders))
	for i := range req.Orders {
		item, err := e.spotBatchOrderItem(&req.Orders[i])
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		items[i] = item
	}
	data, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("data", string(data))
	if req.Sync {
		params.Set("sync", "true")
	}
	var resp *SpotOrdersResponse
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/batchOrders", params, spotPlaceBatchOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// CancelSpotOrder cancels a spot order by its order ID or client order ID. An order that does not exist returns
// order.ErrOrderNotFound
func (e *Exchange) CancelSpotOrder(ctx context.Context, req *CancelSpotOrderRequest) (*SpotOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderID", req.ClientOrderID)
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", string(req.CancelRestrictions))
	}
	var resp *SpotOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/cancel", params, spotCancelOrderRate, &resp); err != nil {
		return nil, spotOrderNotFoundError(err)
	}
	if resp.OrderID == 0 {
		return nil, errSpotOrderIDMissing
	}
	return resp, nil
}

// CancelSpotBatchOrders cancels several of a spot symbol's orders by order ID or client order ID. An order that does
// not exist returns order.ErrOrderNotFound unless Process reports it as a failure
func (e *Exchange) CancelSpotBatchOrders(ctx context.Context, req *CancelSpotBatchOrdersRequest) (*CancelSpotBatchOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(req.OrderIDs) == 0 && len(req.ClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.Process {
		params.Set("process", "1")
	}
	if len(req.OrderIDs) > 0 {
		ids := make([]string, len(req.OrderIDs))
		for i, id := range req.OrderIDs {
			ids[i] = strconv.FormatUint(id, 10)
		}
		params.Set("orderIds", strings.Join(ids, ","))
	}
	if len(req.ClientOrderIDs) > 0 {
		params.Set("clientOrderIDs", strings.Join(req.ClientOrderIDs, ","))
	}
	var resp *CancelSpotBatchOrdersResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/cancelOrders", params, spotCancelBatchOrdersRate, &resp); err != nil {
		return nil, spotOrderNotFoundError(err)
	}
	return resp, nil
}

// CancelSpotOpenOrders cancels a spot symbol's open orders, or every open spot order when symbol is empty
func (e *Exchange) CancelSpotOpenOrders(ctx context.Context, symbol currency.Pair) ([]SpotOrderResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp *SpotOrdersResponse
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/cancelOpenOrders", params, spotCancelOpenOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// CancelReplaceSpotOrder cancels a spot order and places a new one on the same symbol, returning the outcome of both
// halves: a half that fails is reported in its result while the request succeeds. When BingX refuses the request with
// an outcome in its error data, both are returned. An order to cancel that does not exist returns
// order.ErrOrderNotFound
func (e *Exchange) CancelReplaceSpotOrder(ctx context.Context, req *CancelReplaceSpotOrderRequest) (*CancelReplaceSpotOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.CancelOrderID == 0 && req.CancelClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.CancelReplaceMode == "" {
		return nil, errSpotCancelReplaceModeEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Quantity <= 0 && req.QuoteOrderQuantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.CancelOrderID != 0 {
		params.Set("cancelOrderId", strconv.FormatUint(req.CancelOrderID, 10))
	}
	if req.CancelClientOrderID != "" {
		params.Set("cancelClientOrderID", req.CancelClientOrderID)
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", string(req.CancelRestrictions))
	}
	params.Set("cancelReplaceMode", string(req.CancelReplaceMode))
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	if req.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(req.QuoteOrderQuantity, 'f', -1, 64))
	}
	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *CancelReplaceSpotOrderResponse
	err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/order/cancelReplace", params, spotCancelReplaceOrderRate, &resp)
	return spotCancelReplaceOutcome(resp, spotOrderNotFoundError(err))
}

// GetSpotOrderDetails returns a spot order by its order ID or client order ID. An order that does not exist returns
// order.ErrOrderNotFound
func (e *Exchange) GetSpotOrderDetails(ctx context.Context, req *SpotOrderDetailsRequest) (*SpotOrderDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderID", req.ClientOrderID)
	}
	var resp *SpotOrderDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/trade/query", params, spotOrderDetailsRate, &resp); err != nil {
		return nil, spotOrderNotFoundError(err)
	}
	if resp.OrderID == 0 {
		return nil, errSpotOrderIDMissing
	}
	return resp, nil
}

// GetSpotOpenOrders returns a spot symbol's open orders, or every open spot order when symbol is empty, up to 2000
func (e *Exchange) GetSpotOpenOrders(ctx context.Context, symbol currency.Pair) ([]SpotOrderDetailResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp *SpotOrderDetailsResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/trade/openOrders", params, spotOpenOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetSpotOrderHistory returns the account's spot orders, a page at a time
func (e *Exchange) GetSpotOrderHistory(ctx context.Context, req *SpotOrderHistoryRequest) ([]SpotOrderDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PageSize > spotOrderHistoryMaxPageSize {
		return nil, fmt.Errorf("%w: page size %d, at most %d", errSpotPageLimitExceeded, req.PageSize, spotOrderHistoryMaxPageSize)
	}
	// BingX defaults an unset page index to 1 and an unset page size to 100, which the window limit applies to
	pageIndex, pageSize := max(req.PageIndex, 1), req.PageSize
	if pageSize == 0 {
		pageSize = spotOrderHistoryMaxPageSize
	}
	if pageIndex*pageSize > spotOrderHistoryMaxPageWindow {
		return nil, fmt.Errorf("%w: page index %d times page size %d exceeds %d", errSpotPageLimitExceeded, pageIndex, pageSize, spotOrderHistoryMaxPageWindow)
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, req.Symbol); err != nil {
			return nil, err
		}
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.Status != "" {
		params.Set("status", string(req.Status))
	}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	var resp *SpotOrderDetailsResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/trade/historyOrders", params, spotOrderHistoryRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetSpotTransactionDetails returns the account's spot fills from the last seven days, oldest first. Without a time
// range it returns the last 24 hours; Limit defaults to 500
func (e *Exchange) GetSpotTransactionDetails(ctx context.Context, req *SpotTransactionDetailsRequest) ([]SpotFillResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, req.Symbol); err != nil {
			return nil, err
		}
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *SpotFillsResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/trade/myTrades", params, spotTradeDetailsRate, &resp); err != nil {
		return nil, err
	}
	return resp.Fills, nil
}

// GetSpotCommissionRate returns the account's spot commission rates for a symbol
func (e *Exchange) GetSpotCommissionRate(ctx context.Context, symbol currency.Pair) (*SpotCommissionRateResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp *SpotCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/user/commissionRate", params, spotCommissionRateRate, &resp)
}

// CancelAllSpotOrdersAfter starts, extends or stops a countdown that cancels every open spot order when it ends.
// ACTIVATE sets the countdown to timeout, from 10 to 120 seconds, and is repeated to extend it; CLOSE stops it
func (e *Exchange) CancelAllSpotOrdersAfter(ctx context.Context, requestType CancelAllAfterType, timeout time.Duration) (*CancelAllSpotOrdersAfterResponse, error) {
	if requestType == "" {
		return nil, errSpotCancelAllAfterTypeEmpty
	}
	if timeout <= 0 || timeout%time.Second != 0 {
		return nil, fmt.Errorf("%w: %s", errSpotInvalidTimeout, timeout)
	}
	params := url.Values{}
	params.Set("type", string(requestType))
	params.Set("timeOut", strconv.FormatInt(int64(timeout/time.Second), 10))
	var resp *CancelAllSpotOrdersAfterResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/trade/cancelAllAfter", params, spotCancelAllAfterRate, &resp)
}

// PlaceSpotOCOOrder places a spot one-cancels-the-other (OCO) order list, returning its limit and stop-limit orders
func (e *Exchange) PlaceSpotOCOOrder(ctx context.Context, req *PlaceSpotOCOOrderRequest) ([]SpotOCOOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.LimitPrice <= 0 {
		return nil, fmt.Errorf("%w: limitPrice", errSpotPriceRequired)
	}
	if req.OrderPrice <= 0 {
		return nil, fmt.Errorf("%w: orderPrice", errSpotPriceRequired)
	}
	if req.TriggerPrice <= 0 {
		return nil, fmt.Errorf("%w: triggerPrice", errSpotPriceRequired)
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	params.Set("side", string(req.Side))
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	params.Set("limitPrice", strconv.FormatFloat(req.LimitPrice, 'f', -1, 64))
	params.Set("orderPrice", strconv.FormatFloat(req.OrderPrice, 'f', -1, 64))
	params.Set("triggerPrice", strconv.FormatFloat(req.TriggerPrice, 'f', -1, 64))
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.AboveClientOrderID != "" {
		params.Set("aboveClientOrderId", req.AboveClientOrderID)
	}
	if req.BelowClientOrderID != "" {
		params.Set("belowClientOrderId", req.BelowClientOrderID)
	}
	var resp []SpotOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/oco/order", params, spotPlaceOCOOrderRate, &resp)
}

// CancelSpotOCOOrderList cancels a spot one-cancels-the-other (OCO) order list, found by the order ID or client order
// ID of its limit or stop-limit order. An order that does not exist returns order.ErrOrderNotFound
func (e *Exchange) CancelSpotOCOOrderList(ctx context.Context, orderID uint64, clientOrderID string) (*CancelSpotOCOOrderListResponse, error) {
	params, err := spotOCOOrderParams(orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CancelSpotOCOOrderListResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/spot/v1/oco/cancel", params, spotCancelOCOOrderRate, &resp); err != nil {
		return nil, spotOrderNotFoundError(err)
	}
	return resp, nil
}

// GetSpotOCOOrderList returns a spot one-cancels-the-other (OCO) order list's orders, found by the order ID or client
// order ID of its limit or stop-limit order. An order that does not exist returns order.ErrOrderNotFound
func (e *Exchange) GetSpotOCOOrderList(ctx context.Context, orderID uint64, clientOrderID string) ([]SpotOCOOrderResponse, error) {
	params, err := spotOCOOrderParams(orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	var resp []SpotOCOOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/oco/orderList", params, spotOCOOrderListRate, &resp); err != nil {
		return nil, spotOrderNotFoundError(err)
	}
	return resp, nil
}

// GetSpotOpenOCOOrders returns the orders of the account's open spot one-cancels-the-other (OCO) order lists, a page
// at a time
func (e *Exchange) GetSpotOpenOCOOrders(ctx context.Context, pageIndex, pageSize uint64) ([]SpotOCOOrderResponse, error) {
	params, err := spotPageParams(pageIndex, pageSize)
	if err != nil {
		return nil, err
	}
	var resp []SpotOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/oco/openOrderList", params, spotOpenOCOOrdersRate, &resp)
}

// GetSpotOCOOrderHistory returns the orders of the account's past spot one-cancels-the-other (OCO) order lists, a page
// at a time
func (e *Exchange) GetSpotOCOOrderHistory(ctx context.Context, req *SpotOCOOrderHistoryRequest) ([]SpotOCOOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := spotPageParams(req.PageIndex, req.PageSize)
	if err != nil {
		return nil, err
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp []SpotOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/oco/historyOrderList", params, spotOCOOrderHistoryRate, &resp)
}

// spotValidateOrder checks the parameters every spot order needs: a symbol, side and type, and a quantity or quote
// order quantity
func spotValidateOrder(req *PlaceSpotOrderRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return order.ErrTypeIsInvalid
	}
	if req.Quantity <= 0 && req.QuoteOrderQuantity <= 0 {
		return order.ErrAmountIsInvalid
	}
	return nil
}

// spotBatchOrderItem validates an order of a spot batch order request and converts it to its data array element
func (e *Exchange) spotBatchOrderItem(req *PlaceSpotOrderRequest) (SpotBatchOrderItemRequest, error) {
	if err := spotValidateOrder(req); err != nil {
		return SpotBatchOrderItemRequest{}, err
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return SpotBatchOrderItemRequest{}, err
	}
	return SpotBatchOrderItemRequest{
		Symbol:             symbol,
		Side:               req.Side,
		Type:               req.Type,
		StopPrice:          req.StopPrice,
		Quantity:           req.Quantity,
		QuoteOrderQuantity: req.QuoteOrderQuantity,
		Price:              req.Price,
		NewClientOrderID:   req.NewClientOrderID,
		TimeInForce:        req.TimeInForce,
	}, nil
}

// spotOCOOrderParams returns the parameters identifying a spot OCO order list by one of its orders, which needs an
// order ID or a client order ID
func spotOCOOrderParams(orderID uint64, clientOrderID string) (url.Values, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	return params, nil
}

// spotPageParams returns the page parameters the spot OCO order list endpoints require
func spotPageParams(pageIndex, pageSize uint64) (url.Values, error) {
	if pageIndex == 0 {
		return nil, errSpotPageIndexRequired
	}
	if pageSize == 0 {
		return nil, errSpotPageSizeRequired
	}
	params := url.Values{}
	params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	return params, nil
}

// spotOrderNotFoundError wraps order.ErrOrderNotFound around a spot order request BingX refused with code 100404,
// which spot uses for an order that does not exist and other products for an unknown path
func spotOrderNotFoundError(err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == 100404 {
		return fmt.Errorf("%w: %w", order.ErrOrderNotFound, err)
	}
	return err
}

// spotCancelReplaceOutcome returns a spot cancel-replace's outcome with its error. A refused request can carry the
// outcome of both halves in its data, which tells the caller whether a new order was still placed
func spotCancelReplaceOutcome(resp *CancelReplaceSpotOrderResponse, err error) (*CancelReplaceSpotOrderResponse, error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Data) == 0 {
		return resp, err
	}
	if decodeErr := json.Unmarshal(apiErr.Data, &resp); decodeErr != nil {
		return nil, common.AppendError(err, fmt.Errorf("error decoding cancel-replace outcome: %w", decodeErr))
	}
	return resp, err
}

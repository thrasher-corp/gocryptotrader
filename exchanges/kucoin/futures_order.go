package kucoin

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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com/api"
	kucoinWebsocketURL  = "wss://ws-api.kucoin.com/endpoint"
)

// Sentinel errors for futures order requests
var (
	errSizeQuantityValueQuantityExclusive = errors.New("only one of size, qty and valueQty can be set")
	errBatchCancelOrderIDsRequired        = errors.New("order IDs or client order IDs are required")
)

// PostFuturesOrder places a limit or market futures order, or a stop order when the request sets Stop (Add Order)
func (e *Exchange) PostFuturesOrder(ctx context.Context, r *FuturesOrderRequest) (*FuturesOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, postFuturesOrderEPL, http.MethodPost, "/v1/orders", body, &resp)
}

// PostFuturesOrderTest checks a futures order and its signature without placing it, returning the IDs Add Order would
// (Add Order Test)
func (e *Exchange) PostFuturesOrderTest(ctx context.Context, r *FuturesOrderRequest) (*FuturesOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, postFuturesOrderTestEPL, http.MethodPost, "/v1/orders/test", body, &resp)
}

// PlaceMultipleFuturesOrders places up to 20 futures orders, returning KuCoin's verdict on each in request order
// (Batch Add Orders)
func (e *Exchange) PlaceMultipleFuturesOrders(ctx context.Context, orders []FuturesOrderRequest) ([]FuturesBatchOrderResponse, error) {
	body, err := futuresOrderListBody(orders)
	if err != nil {
		return nil, err
	}
	var resp []FuturesBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, placeMultipleFuturesOrdersEPL, http.MethodPost, "/v1/orders/multi", body, &resp)
}

// futuresOrderListBody returns the body Batch Add Orders and the futures.multi_order websocket operation take, naming
// the position of an order that fails validation
func futuresOrderListBody(orders []FuturesOrderRequest) ([]map[string]any, error) {
	if len(orders) == 0 {
		return nil, fmt.Errorf("%w: no orders to place", common.ErrEmptyParams)
	}
	body := make([]map[string]any, len(orders))
	for i := range orders {
		var err error
		if body[i], err = orders[i].Payload(); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
	}
	return body, nil
}

// PostFuturesTakeProfitStopLossOrder places a futures order with take profit and stop loss trigger prices (Add Take
// Profit And Stop Loss Order)
func (e *Exchange) PostFuturesTakeProfitStopLossOrder(ctx context.Context, r *FuturesTakeProfitStopLossOrderRequest) (*FuturesOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, postFuturesTakeProfitStopLossOrderEPL, http.MethodPost, "/v1/st-orders", body, &resp)
}

// CancelFuturesOrderByOrderID cancels a futures order, including a stop order, by its order ID (Cancel Order By
// OrderId). Cancelling is asynchronous: the order's status shows when KuCoin has processed the request
func (e *Exchange) CancelFuturesOrderByOrderID(ctx context.Context, orderID string) (*FuturesCancelledOrdersResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesCancelledOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelFuturesOrderByOrderIDEPL, http.MethodDelete, "/v1/orders/"+orderID, nil, &resp)
}

// CancelFuturesOrderByClientOrderID cancels a futures order, including a stop order, by its symbol and client order
// ID (Cancel Order By ClientOid). Cancelling is asynchronous: the order's status shows when KuCoin has processed the
// request
func (e *Exchange) CancelFuturesOrderByClientOrderID(ctx context.Context, symbol, clientOrderID string) (*FuturesCancelledClientOrderResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesCancelledClientOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelFuturesOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/orders/client-order/"+clientOrderID, params), nil, &resp)
}

// BatchCancelFuturesOrders cancels up to 10 futures orders by order ID or by symbol and client order ID, returning
// KuCoin's verdict on each (Batch Cancel Orders)
func (e *Exchange) BatchCancelFuturesOrders(ctx context.Context, r *BatchCancelFuturesOrdersRequest) ([]FuturesBatchCancelResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp []FuturesBatchCancelResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, batchCancelFuturesOrdersEPL, http.MethodDelete, "/v1/orders/multi-cancel", body, &resp)
}

// CancelAllFuturesOrders cancels a contract's open orders, except stop orders (Cancel All Orders)
func (e *Exchange) CancelAllFuturesOrders(ctx context.Context, symbol string) (*FuturesCancelledOrdersResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesCancelledOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelAllFuturesOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v3/orders", params), nil, &resp)
}

// CancelAllFuturesStopOrders cancels the untriggered stop orders of a contract, or of every contract when symbol is
// empty (Cancel All Stop orders)
func (e *Exchange) CancelAllFuturesStopOrders(ctx context.Context, symbol string) (*FuturesCancelledOrdersResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesCancelledOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelAllFuturesStopOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v1/stopOrders", params), nil, &resp)
}

// GetFuturesOrderDetailsByOrderID returns a futures order, including a stop order, by its order ID (Get Order By
// OrderId). Cancelled and filled orders are kept for three months
func (e *Exchange) GetFuturesOrderDetailsByOrderID(ctx context.Context, orderID string) (*FuturesOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesOrderDetailsByOrderIDEPL, http.MethodGet, "/v1/orders/"+orderID, nil, &resp)
}

// GetFuturesOrderDetailsByClientOrderID returns a futures order, including a stop order, by its client order ID (Get
// Order By ClientOid). Cancelled and filled orders are kept for seven days
func (e *Exchange) GetFuturesOrderDetailsByClientOrderID(ctx context.Context, clientOrderID string) (*FuturesOrderDetailResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	var resp *FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesOrderDetailsByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v1/orders/byClientOid", params), nil, &resp)
}

// GetFuturesOrders returns a page of futures orders, done ones unless the request asks for active ones (Get Order
// List)
func (e *Exchange) GetFuturesOrders(ctx context.Context, r *GetFuturesOrdersRequest) (*FuturesOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params, err := futuresOrderListParams(r.Symbol, r.Side, r.Type, r.StartAt, r.EndAt, r.CurrentPage, r.PageSize)
	if err != nil {
		return nil, err
	}
	if r.Status != "" {
		params.Set("status", r.Status)
	}
	var resp *FuturesOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/orders", params), nil, &resp)
}

// GetFuturesRecentCompletedOrders returns up to 1000 futures orders closed in the last 24 hours, for one contract or
// all of them when symbol is empty (Get Recent Closed Orders)
func (e *Exchange) GetFuturesRecentCompletedOrders(ctx context.Context, symbol string) ([]FuturesOrderDetailResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesRecentCompletedOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/recentDoneOrders", params), nil, &resp)
}

// GetUntriggeredFuturesStopOrders returns a page of untriggered futures stop orders; triggered ones are listed by Get
// Order List (Get Stop Order List)
func (e *Exchange) GetUntriggeredFuturesStopOrders(ctx context.Context, r *GetUntriggeredFuturesStopOrdersRequest) (*FuturesOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params, err := futuresOrderListParams(r.Symbol, r.Side, r.Type, r.StartAt, r.EndAt, r.CurrentPage, r.PageSize)
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getUntriggeredFuturesStopOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/stopOrders", params), nil, &resp)
}

// GetFuturesOpenOrderStats returns the number and value of a contract's active orders (Get Open Order Value)
func (e *Exchange) GetFuturesOpenOrderStats(ctx context.Context, symbol string) (*FuturesOpenOrderStatsResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesOpenOrderStatsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesOpenOrderStatsEPL, http.MethodGet, common.EncodeURLValues("/v1/openOrderStatistics", params), nil, &resp)
}

// GetFuturesRecentFills returns up to 1000 futures fills of the last 24 hours, for one contract or all of them when
// symbol is empty (Get Recent Trade History)
func (e *Exchange) GetFuturesRecentFills(ctx context.Context, symbol string) ([]FuturesRecentFillResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []FuturesRecentFillResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesRecentFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/recentFills", params), nil, &resp)
}

// GetFuturesFills returns a page of futures fills (Get Trade History)
func (e *Exchange) GetFuturesFills(ctx context.Context, r *GetFuturesFillsRequest) (*FuturesFillsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params, err := futuresOrderListParams(r.Symbol, r.Side, r.Type, r.StartAt, r.EndAt, r.CurrentPage, r.PageSize)
	if err != nil {
		return nil, err
	}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if len(r.TradeTypes) != 0 {
		params.Set("tradeTypes", strings.Join(r.TradeTypes, ","))
	}
	var resp *FuturesFillsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/fills", params), nil, &resp)
}

// futuresOrderListParams builds the query parameters Get Order List, Get Stop Order List and Get Trade History share
func futuresOrderListParams(symbol, side, orderType string, startAt, endAt time.Time, currentPage, pageSize uint64) (url.Values, error) {
	if !startAt.IsZero() && !endAt.IsZero() {
		if err := common.StartEndTimeCheck(startAt, endAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	return params, nil
}

// countSet returns how many of the conditions hold, for parameters the API takes only one of
func countSet(conditions ...bool) int {
	var n int
	for _, c := range conditions {
		if c {
			n++
		}
	}
	return n
}

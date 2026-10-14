package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Margin order validation errors
var (
	errSizeAndFundsExclusive = errors.New("size and funds cannot both be set")
)

// PlaceMarginHFOrder places a cross or isolated margin order (Add Order)
func (e *Exchange) PlaceMarginHFOrder(ctx context.Context, r *PlaceMarginHFOrderRequest) (*PlaceMarginHFOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PlaceMarginHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginHFOrderEPL, http.MethodPost, "/v3/hf/margin/order", body, &resp)
}

// PlaceMarginHFOrderTest checks a margin order and the request signature without placing the order (Add Order Test)
func (e *Exchange) PlaceMarginHFOrderTest(ctx context.Context, r *PlaceMarginHFOrderRequest) (*PlaceMarginHFOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PlaceMarginHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginHFOrderTestEPL, http.MethodPost, "/v3/hf/margin/order/test", body, &resp)
}

// CancelMarginHFOrderByOrderID requests the cancellation of a margin order; the order's status or the websocket order
// channel shows whether it was cancelled (Cancel Order By OrderId)
func (e *Exchange) CancelMarginHFOrderByOrderID(ctx context.Context, orderID, symbol string) (*CancelMarginHFOrderByOrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *CancelMarginHFOrderByOrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginHFOrderByOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/orders/"+orderID, params), nil, &resp)
}

// CancelMarginHFOrderByClientOrderID requests the cancellation of a margin order by its client order ID; the order's
// status or the websocket order channel shows whether it was cancelled (Cancel Order By ClientOid)
func (e *Exchange) CancelMarginHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*CancelMarginHFOrderByClientOrderIDResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *CancelMarginHFOrderByClientOrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginHFOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/orders/client-order/"+clientOrderID, params), nil, &resp)
}

// CancelAllMarginHFOrdersBySymbol requests the cancellation of every open margin order of a symbol and trade type,
// MARGIN_TRADE for cross margin or MARGIN_ISOLATED_TRADE for isolated margin (Cancel All Orders By Symbol)
func (e *Exchange) CancelAllMarginHFOrdersBySymbol(ctx context.Context, symbol, tradeType string) (string, error) {
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	if tradeType == "" {
		return "", errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("tradeType", tradeType)
	var resp string
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllMarginHFOrdersBySymbolEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/orders", params), nil, &resp)
}

// GetActiveMarginHFOrderSymbols returns the symbols that have open margin orders of a trade type, MARGIN_TRADE for
// cross margin or MARGIN_ISOLATED_TRADE for isolated margin (Get Symbols With Open Order)
func (e *Exchange) GetActiveMarginHFOrderSymbols(ctx context.Context, tradeType string) (*ActiveMarginHFOrderSymbolsResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	var resp *ActiveMarginHFOrderSymbolsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getActiveMarginHFOrderSymbolsEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/order/active/symbols", params), nil, &resp)
}

// GetActiveMarginHFOrders returns the open margin orders of a symbol and trade type, MARGIN_TRADE for cross margin or
// MARGIN_ISOLATED_TRADE for isolated margin, latest update first (Get Open Orders)
func (e *Exchange) GetActiveMarginHFOrders(ctx context.Context, symbol, tradeType string) ([]MarginHFOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("tradeType", tradeType)
	var resp []MarginHFOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getActiveMarginHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/active", params), nil, &resp)
}

// GetClosedMarginHFOrders returns a page of the filled and cancelled margin orders of the last seven days, latest
// update first (Get Closed Orders)
func (e *Exchange) GetClosedMarginHFOrders(ctx context.Context, r *GetClosedMarginHFOrdersRequest) (*ClosedMarginHFOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("tradeType", r.TradeType)
	if r.Side != "" {
		params.Set("side", r.Side)
	}
	if r.Type != "" {
		params.Set("type", r.Type)
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	var resp *ClosedMarginHFOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getClosedMarginHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/done", params), nil, &resp)
}

// GetMarginHFTradeFills returns a page of margin trade fills, latest first (Get Trade History)
func (e *Exchange) GetMarginHFTradeFills(ctx context.Context, r *GetMarginHFTradeFillsRequest) (*MarginHFTradeFillsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("tradeType", r.TradeType)
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.Side != "" {
		params.Set("side", r.Side)
	}
	if r.Type != "" {
		params.Set("type", r.Type)
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	var resp *MarginHFTradeFillsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginHFTradeFillsEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/fills", params), nil, &resp)
}

// GetMarginHFOrderDetailByOrderID returns a margin order; KuCoin keeps closed orders for seven days (Get Order By
// OrderId)
func (e *Exchange) GetMarginHFOrderDetailByOrderID(ctx context.Context, orderID, symbol string) (*MarginHFOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *MarginHFOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginHFOrderDetailByOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/"+orderID, params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return resp, nil
}

// GetMarginHFOrderDetailByClientOrderID returns a margin order by its client order ID; KuCoin keeps closed orders for
// seven days (Get Order By ClientOid)
func (e *Exchange) GetMarginHFOrderDetailByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*MarginHFOrderResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *MarginHFOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginHFOrderDetailByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/client-order/"+clientOrderID, params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, clientOrderID)
	}
	return resp, nil
}

// PlaceMarginStopOrder places a margin stop order; an account may hold 20 untriggered stop orders per symbol (Add
// Stop Order)
func (e *Exchange) PlaceMarginStopOrder(ctx context.Context, r *PlaceMarginStopOrderRequest) (*PlaceMarginStopOrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if r.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.StopPrice <= 0 {
		return nil, errStopPriceRequired
	}
	if err := validateMarginOrderAmounts(r.Type, r.Price, r.Size, r.Funds); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["side"] = r.Side
	body["symbol"] = r.Symbol
	body["isIsolated"] = r.IsIsolated
	body["autoBorrow"] = r.AutoBorrow
	body["autoRepay"] = r.AutoRepay
	body["stopPrice"] = strconv.FormatFloat(r.StopPrice, 'f', -1, 64)
	if r.Type != "" {
		body["type"] = r.Type
	}
	if r.SelfTradePrevention != "" {
		body["stp"] = r.SelfTradePrevention
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = strconv.FormatFloat(r.Size, 'f', -1, 64)
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.CancelAfter != 0 {
		body["cancelAfter"] = r.CancelAfter
	}
	if r.Funds != 0 {
		body["funds"] = strconv.FormatFloat(r.Funds, 'f', -1, 64)
	}
	if r.Remark != "" {
		body["remark"] = r.Remark
	}
	if r.Stop != "" {
		body["stop"] = r.Stop
	}
	var resp *PlaceMarginStopOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginStopOrderEPL, http.MethodPost, "/v3/hf/margin/stop-order", body, &resp)
}

// CancelMarginStopOrderByOrderID requests the cancellation of a margin stop order (Cancel Stop Order By OrderId)
func (e *Exchange) CancelMarginStopOrderByOrderID(ctx context.Context, orderID string) (*CancelledMarginOrdersResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	// The docs list no parameters for this endpoint; orderId is sent as Cancel OCO Order By OrderId documents it
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginStopOrderByOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/stop-order/cancel-by-id", params), nil, &resp)
}

// CancelMarginStopOrderByClientOrderID requests the cancellation of a margin stop order by its client order ID (Cancel
// Stop Order By ClientOid)
func (e *Exchange) CancelMarginStopOrderByClientOrderID(ctx context.Context, clientOrderID string) (*CancelledMarginOrdersResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginStopOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/stop-order/cancel-by-clientOid", params), nil, &resp)
}

// CancelMarginStopOrders requests the cancellation of the margin stop orders of a trade type, narrowed by symbol and
// order IDs when they are set (Batch Cancel Stop Orders)
func (e *Exchange) CancelMarginStopOrders(ctx context.Context, r *CancelMarginStopOrdersRequest) (*CancelledMarginOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", r.TradeType)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginStopOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/stop-order/cancel", params), nil, &resp)
}

// GetMarginStopOrders returns a page of untriggered margin stop orders, latest first (Get Stop Orders List)
func (e *Exchange) GetMarginStopOrders(ctx context.Context, r *GetMarginStopOrdersRequest) (*MarginStopOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if r.Side != "" {
		params.Set("side", r.Side)
	}
	if r.Type != "" {
		params.Set("type", r.Type)
	}
	if r.TradeType != "" {
		params.Set("tradeType", r.TradeType)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(r.CurrentPage, 10))
	}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	if r.Stop != "" {
		params.Set("stop", r.Stop)
	}
	var resp *MarginStopOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginStopOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/stop-orders", params), nil, &resp)
}

// GetMarginStopOrderByOrderID returns a margin stop order (Get Stop Order By OrderId)
func (e *Exchange) GetMarginStopOrderByOrderID(ctx context.Context, orderID string) (*MarginStopOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *MarginStopOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginStopOrderByOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/stop-order/orderId", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return resp, nil
}

// GetMarginStopOrderByClientOrderID returns the margin stop orders with a client order ID (Get Stop Order By
// ClientOid)
func (e *Exchange) GetMarginStopOrderByClientOrderID(ctx context.Context, clientOrderID string) ([]MarginStopOrderResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	var resp MarginStopOrdersByClientOrderID
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginStopOrderByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/stop-order/clientOid", params), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PlaceMarginOCOOrder places a margin OCO order: a limit order at Price and a stop order that triggers at StopPrice
// and places a limit order at LimitPrice, the first to fill cancelling the other (Add OCO Order)
func (e *Exchange) PlaceMarginOCOOrder(ctx context.Context, r *PlaceMarginOCOOrderRequest) (*PlaceMarginOCOOrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if r.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if r.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if r.StopPrice <= 0 {
		return nil, errStopPriceRequired
	}
	if r.LimitPrice <= 0 {
		return nil, errLimitPriceRequired
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["side"] = r.Side
	body["symbol"] = r.Symbol
	body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	body["size"] = strconv.FormatFloat(r.Size, 'f', -1, 64)
	body["stopPrice"] = strconv.FormatFloat(r.StopPrice, 'f', -1, 64)
	body["limitPrice"] = strconv.FormatFloat(r.LimitPrice, 'f', -1, 64)
	body["isIsolated"] = r.IsIsolated
	if r.AutoRepay {
		body["autoRepay"] = true
	}
	if r.AutoBorrow {
		body["autoBorrow"] = true
	}
	var resp *PlaceMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginOCOOrderEPL, http.MethodPost, "/v3/hf/margin/oco-order", body, &resp)
}

// CancelMarginOCOOrderByOrderID requests the cancellation of a margin OCO order, returning the IDs of its two orders
// (Cancel OCO Order By OrderId)
func (e *Exchange) CancelMarginOCOOrderByOrderID(ctx context.Context, orderID string) (*CancelledMarginOrdersResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginOCOOrderByOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/oco-order/cancel-by-id", params), nil, &resp)
}

// CancelMarginOCOOrderByClientOrderID requests the cancellation of a margin OCO order by its client order ID,
// returning the IDs of its two orders (Cancel OCO Order By ClientOid)
func (e *Exchange) CancelMarginOCOOrderByClientOrderID(ctx context.Context, clientOrderID string) (*CancelledMarginOrdersResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginOCOOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/oco-order/cancel-by-clientOid", params), nil, &resp)
}

// CancelMarginOCOOrders requests the cancellation of the margin OCO orders the request's filters match, every OCO
// order when it sets none (Batch Cancel OCO Order)
func (e *Exchange) CancelMarginOCOOrders(ctx context.Context, r *CancelMarginOCOOrdersRequest) (*CancelledMarginOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if r.TradeType != "" {
		params.Set("tradeType", r.TradeType)
	}
	var resp *CancelledMarginOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginOCOOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/oco-order/cancel", params), nil, &resp)
}

// GetMarginOCOOrderByOrderID returns a margin OCO order (Get OCO Order By OrderId)
func (e *Exchange) GetMarginOCOOrderByOrderID(ctx context.Context, orderID string) (*MarginOCOOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *MarginOCOOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginOCOOrderByOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/oco-order/orderId", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return resp, nil
}

// GetMarginOCOOrderByClientOrderID returns a margin OCO order by its client order ID (Get OCO Order By ClientOid)
func (e *Exchange) GetMarginOCOOrderByClientOrderID(ctx context.Context, clientOrderID string) (*MarginOCOOrderResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	var resp *MarginOCOOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginOCOOrderByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/oco-order/clientOid", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, clientOrderID)
	}
	return resp, nil
}

// GetMarginOCOOrderDetailByOrderID returns a margin OCO order with its two orders (Get OCO Order Detail By OrderId)
func (e *Exchange) GetMarginOCOOrderDetailByOrderID(ctx context.Context, orderID string) (*MarginOCOOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *MarginOCOOrderDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginOCOOrderDetailByOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/oco-order/detail/orderId", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return resp, nil
}

// GetMarginOCOOrders returns a page of margin OCO orders, latest first (Get OCO Order List)
func (e *Exchange) GetMarginOCOOrders(ctx context.Context, r *GetMarginOCOOrdersRequest) (*MarginOCOOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.PageSize == 0 {
		return nil, errPageSizeRequired
	}
	if r.CurrentPage == 0 {
		return nil, errCurrentPageRequired
	}
	params := url.Values{}
	params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	params.Set("currentPage", strconv.FormatUint(r.CurrentPage, 10))
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	if r.TradeType != "" {
		params.Set("tradeType", r.TradeType)
	}
	var resp *MarginOCOOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginOCOOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/oco-orders", params), nil, &resp)
}

// validateMarginOrderAmounts checks the amounts KuCoin requires of a margin order type: a price and a size for limit
// orders, which are the default, and either a size or funds for market orders
func validateMarginOrderAmounts(orderType string, price, size, funds float64) error {
	switch orderType {
	case "", "limit":
		if price <= 0 {
			return limits.ErrPriceBelowMin
		}
		if size <= 0 {
			return limits.ErrAmountBelowMin
		}
	case "market":
		if size == 0 && funds == 0 {
			return errSizeOrFundIsRequired
		}
		if size != 0 && funds != 0 {
			return errSizeAndFundsExclusive
		}
	}
	return nil
}

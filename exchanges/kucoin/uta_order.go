package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Unified trading account order validation errors
var (
	errSizeUnitRequired                 = errors.New("size unit is required")
	errTriggerDirectionAndPriceRequired = errors.New("trigger direction and trigger price must be set together")
	errConditionalOrderWithTPSL         = errors.New("a conditional order cannot carry take profit or stop loss trigger prices")
	errCancelOrderListEmpty             = errors.New("cancel order list is empty")
	errOrderFilterRequired              = errors.New("order filter is required")
)

// PlaceUTAOrder places a spot, margin or futures order from the unified trading account (Place Order)
func (e *Exchange) PlaceUTAOrder(ctx context.Context, r *PlaceUTAOrderRequest) (*UTAOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *UTAOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeUTAOrderEPL, http.MethodPost, "/ua/v2/unified/order/place", body, &resp)
}

// AmendUTAOrder changes the price, size or take profit and stop loss trigger prices of a unified trading account
// futures order (Amend Order)
func (e *Exchange) AmendUTAOrder(ctx context.Context, r *AmendUTAOrderRequest) (*AmendUTAOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *AmendUTAOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, amendUTAOrderEPL, http.MethodPost, "/ua/v2/unified/order/amend", body, &resp)
}

// CancelUTAOrder cancels a unified trading account order by order ID or client order ID (Cancel Order)
func (e *Exchange) CancelUTAOrder(ctx context.Context, r *CancelUTAOrderRequest) (*UTAOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *UTAOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelUTAOrderEPL, http.MethodPost, "/ua/v2/unified/order/cancel", body, &resp)
}

// BatchCancelUTAOrdersByID cancels unified trading account orders by order ID or client order ID. Each order of the
// result carries KuCoin's verdict on it, since one failed cancellation does not fail the request (Batch Cancel Orders
// By ID)
func (e *Exchange) BatchCancelUTAOrdersByID(ctx context.Context, r *BatchCancelUTAOrdersByIDRequest) (*BatchCancelUTAOrdersByIDResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.TradeType == "":
		return nil, errTradeTypeMissing
	case len(r.CancelOrderList) == 0:
		return nil, errCancelOrderListEmpty
	}
	orders := make([]map[string]any, len(r.CancelOrderList))
	for i := range r.CancelOrderList {
		item := &r.CancelOrderList[i]
		switch {
		case item.Symbol == "":
			return nil, fmt.Errorf("cancel order list entry %d: %w", i, currency.ErrSymbolStringEmpty)
		case item.OrderID == "" && item.ClientOrderID == "":
			return nil, fmt.Errorf("cancel order list entry %d: %w", i, order.ErrOrderIDNotSet)
		}
		orders[i] = make(map[string]any)
		orders[i]["symbol"] = item.Symbol
		if item.OrderID != "" {
			orders[i]["orderId"] = item.OrderID
		}
		if item.ClientOrderID != "" {
			orders[i]["clientOid"] = item.ClientOrderID
		}
	}
	body := make(map[string]any)
	body["tradeType"] = r.TradeType
	body["cancelOrderList"] = orders
	var resp *BatchCancelUTAOrdersByIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, batchCancelUTAOrdersByIDEPL, http.MethodPost, "/ua/v2/unified/order/cancel-batch", body, &resp)
}

// BatchCancelUTAOrdersBySymbol cancels up to 100 unified trading account orders of a symbol (Batch Cancel Orders By
// Symbol)
func (e *Exchange) BatchCancelUTAOrdersBySymbol(ctx context.Context, r *BatchCancelUTAOrdersBySymbolRequest) (*BatchCancelUTAOrdersBySymbolResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.TradeType == "":
		return nil, errTradeTypeMissing
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.OrderFilter == "":
		return nil, errOrderFilterRequired
	}
	body := make(map[string]any)
	body["tradeType"] = r.TradeType
	body["symbol"] = r.Symbol
	body["orderFilter"] = r.OrderFilter
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	var resp *BatchCancelUTAOrdersBySymbolResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, batchCancelUTAOrdersBySymbolEPL, http.MethodPost, "/ua/v2/unified/order/cancel-all", body, &resp)
}

// GetUTAOrderDetails returns a unified trading account order by order ID or client order ID (Get Order Details)
func (e *Exchange) GetUTAOrderDetails(ctx context.Context, r *UTAOrderDetailsRequest) (*UTAOrderDetailsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.TradeType == "":
		return nil, errTradeTypeMissing
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.OrderID == "" && r.ClientOrderID == "":
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("tradeType", r.TradeType)
	params.Set("symbol", r.Symbol)
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.ClientOrderID != "" {
		params.Set("clientOid", r.ClientOrderID)
	}
	var resp *UTAOrderDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/order/detail", params), nil, &resp)
}

// GetUTAOpenOrderList returns a page of the unified trading account's open orders, newest first (Get Open Order List)
func (e *Exchange) GetUTAOpenOrderList(ctx context.Context, r *UTAOpenOrderListRequest) (*UTAOpenOrderListResponse, error) {
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
	if r.OrderFilter != "" {
		params.Set("orderFilter", r.OrderFilter)
	}
	if r.PageNumber != 0 {
		params.Set("pageNumber", strconv.FormatUint(r.PageNumber, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	var resp *UTAOpenOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAOpenOrderListEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/order/open-list", params), nil, &resp)
}

// GetUTAOrderHistory returns a page of the unified trading account's order history (Get Order History)
func (e *Exchange) GetUTAOrderHistory(ctx context.Context, r *UTAOrderHistoryRequest) (*UTAOrderHistoryResponse, error) {
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
	if r.Side != "" {
		params.Set("side", r.Side)
	}
	if r.OrderFilter != "" {
		params.Set("orderFilter", r.OrderFilter)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	var resp *UTAOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/order/history", params), nil, &resp)
}

// GetUTATradeHistory returns a page of the unified trading account's fills (Get Trade History)
func (e *Exchange) GetUTATradeHistory(ctx context.Context, r *UTATradeHistoryRequest) (*UTATradeHistoryResponse, error) {
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
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.Side != "" {
		params.Set("side", r.Side)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	if r.FillType != "" {
		params.Set("fillType", r.FillType)
	}
	var resp *UTATradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTATradeHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/order/execution", params), nil, &resp)
}

// GetUTAMaxOrderQuantity returns the most the unified trading account can buy and sell of a symbol (Get Max Order
// Quantity)
func (e *Exchange) GetUTAMaxOrderQuantity(ctx context.Context, r *UTAMaxOrderQuantityRequest) (*UTAMaxOrderQuantityResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.TradeType == "":
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("tradeType", r.TradeType)
	if r.Price != 0 {
		params.Set("price", strconv.FormatFloat(r.Price, 'f', -1, 64))
	}
	var resp *UTAMaxOrderQuantityResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAMaxOrderQuantityEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/order/max-order-quantity", params), nil, &resp)
}

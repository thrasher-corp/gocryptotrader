package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// errQuantityOrPriceRequired is returned when an amendment changes neither the quantity nor the price
var errQuantityOrPriceRequired = errors.New("either quantity or price is required")

// GetSpreadInstrumentsInfo returns the specifications of spread combinations (Spread Trading: Get Instruments Info)
func (e *Exchange) GetSpreadInstrumentsInfo(ctx context.Context, req *SpreadInstrumentsInfoRequest) (*SpreadInstrumentsInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *SpreadInstrumentsInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spread/instrument", params), defaultEPL, &resp)
}

// GetSpreadOrderbook returns a spread combination's orderbook depth; limit is the number of levels each side
// (Spread Trading: Get Orderbook)
func (e *Exchange) GetSpreadOrderbook(ctx context.Context, symbol string, limit uint64) (*SpreadOrderbookResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *SpreadOrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spread/orderbook", params), defaultEPL, &resp)
}

// GetSpreadTickers returns a spread combination's best bid and ask, last price and 24 hour statistics
// (Spread Trading: Get Tickers)
func (e *Exchange) GetSpreadTickers(ctx context.Context, symbol string) (*SpreadTickersResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SpreadTickersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spread/tickers", params), defaultEPL, &resp)
}

// GetSpreadRecentPublicTrades returns a spread combination's recent public trades (Spread Trading: Get Recent Public
// Trades)
func (e *Exchange) GetSpreadRecentPublicTrades(ctx context.Context, symbol string, limit uint64) (*SpreadRecentPublicTradesResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *SpreadRecentPublicTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spread/recent-trade", params), defaultEPL, &resp)
}

// CreateSpreadOrder places a spread combination order. Bybit processes it asynchronously, so the order stream
// confirms its state (Spread Trading: Create Order)
func (e *Exchange) CreateSpreadOrder(ctx context.Context, req *CreateSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	body := map[string]any{
		"symbol":    req.Symbol,
		"side":      req.Side,
		"orderType": req.OrderType,
		"qty":       strconv.FormatFloat(req.Quantity, 'f', -1, 64),
	}
	if req.Price != nil {
		body["price"] = strconv.FormatFloat(*req.Price, 'f', -1, 64)
	}
	if req.OrderLinkID != "" {
		body["orderLinkId"] = req.OrderLinkID
	}
	if req.TimeInForce != "" {
		body["timeInForce"] = req.TimeInForce
	}
	var resp *SpreadOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spread/order/create", nil, body, &resp, createSpreadOrderEPL)
}

// AmendSpreadOrder changes the quantity or price of an unfilled or partially filled spread order. Bybit processes it
// asynchronously, so the order stream confirms its state (Spread Trading: Amend Order)
func (e *Exchange) AmendSpreadOrder(ctx context.Context, req *AmendSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.OrderID == "" && req.OrderLinkID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Quantity == 0 && req.Price == nil {
		return nil, errQuantityOrPriceRequired
	}
	body := map[string]any{"symbol": req.Symbol}
	if req.OrderID != "" {
		body["orderId"] = req.OrderID
	}
	if req.OrderLinkID != "" {
		body["orderLinkId"] = req.OrderLinkID
	}
	if req.Quantity != 0 {
		body["qty"] = strconv.FormatFloat(req.Quantity, 'f', -1, 64)
	}
	if req.Price != nil {
		body["price"] = strconv.FormatFloat(*req.Price, 'f', -1, 64)
	}
	var resp *SpreadOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spread/order/amend", nil, body, &resp, amendSpreadOrderEPL)
}

// CancelSpreadOrder cancels a spread order by its order ID or order link ID. Bybit processes it asynchronously, so the
// order stream confirms its state (Spread Trading: Cancel Order)
func (e *Exchange) CancelSpreadOrder(ctx context.Context, orderID, orderLinkID string) (*SpreadOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	body := map[string]any{}
	if orderID != "" {
		body["orderId"] = orderID
	}
	if orderLinkID != "" {
		body["orderLinkId"] = orderLinkID
	}
	var resp *SpreadOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spread/order/cancel", nil, body, &resp, cancelSpreadOrderEPL)
}

// CancelAllSpreadOrders cancels the open spread orders of symbol, or of every symbol when symbol is empty and
// cancelAll is true. Bybit processes it asynchronously, so the order stream confirms each order's state
// (Spread Trading: Cancel All Orders)
func (e *Exchange) CancelAllSpreadOrders(ctx context.Context, symbol string, cancelAll bool) (*CancelAllSpreadOrdersResponse, error) {
	body := map[string]any{}
	if symbol != "" {
		body["symbol"] = symbol
	}
	if cancelAll {
		body["cancelAll"] = true
	}
	var resp *CancelAllSpreadOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spread/order/cancel-all", nil, body, &resp, cancelAllSpreadOrdersEPL)
}

// GetSpreadOpenOrders returns unfilled and partially filled spread orders (Spread Trading: Get Open Orders)
func (e *Exchange) GetSpreadOpenOrders(ctx context.Context, req *SpreadOpenOrdersRequest) (*SpreadOpenOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *SpreadOpenOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spread/order/realtime", params, nil, &resp, getSpreadOpenOrdersEPL)
}

// GetSpreadOrderHistory returns spread orders in a final state; a window longer than 7 days is rejected
// (Spread Trading: Get Order History)
func (e *Exchange) GetSpreadOrderHistory(ctx context.Context, req *SpreadOrderHistoryRequest) (*SpreadOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *SpreadOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spread/order/history", params, nil, &resp, getSpreadOrderHistoryEPL)
}

// GetSpreadTradeHistory returns spread combination executions with the executions of their legs; a window longer
// than 7 days is rejected (Spread Trading: Get Trade History)
func (e *Exchange) GetSpreadTradeHistory(ctx context.Context, req *SpreadTradeHistoryRequest) (*SpreadTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *SpreadTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spread/execution/list", params, nil, &resp, getSpreadTradeHistoryEPL)
}

// GetSpreadMaxQuantity returns the maximum quantity a spread order can be placed for at a price (Spread Trading: Get
// Max Qty)
func (e *Exchange) GetSpreadMaxQuantity(ctx context.Context, req *SpreadMaxQuantityRequest) (*SpreadMaxQuantityResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", req.Side)
	params.Set("orderPrice", strconv.FormatFloat(req.OrderPrice, 'f', -1, 64))
	var resp *SpreadMaxQuantityResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spread/max-qty", params, nil, &resp, getSpreadMaxQuantityEPL)
}

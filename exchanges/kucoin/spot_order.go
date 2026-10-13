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

// Spot order validation errors
var (
	errSizeAndFundsBothSet       = errors.New("size and funds cannot both be set")
	errClientTimestampRequired   = errors.New("client timestamp is required with a max time window")
	errNewPriceOrNewSizeRequired = errors.New("new price or new size is required")
)

// PlaceHFOrder places a spot order (Add Order). KuCoin answers once the matching engine has received the order, so
// its fills follow on the order websocket channel or from GetHFOrderByOrderID
func (e *Exchange) PlaceHFOrder(ctx context.Context, r *PlaceHFOrderRequest) (*PlaceHFOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PlaceHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeHFOrderEPL, http.MethodPost, "/v1/hf/orders", body, &resp)
}

// PlaceHFOrderSync places a spot order and returns its state once matching has finished (Add Order Sync)
func (e *Exchange) PlaceHFOrderSync(ctx context.Context, r *PlaceHFOrderRequest) (*PlaceHFOrderSyncResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PlaceHFOrderSyncResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeHFOrderSyncEPL, http.MethodPost, "/v1/hf/orders/sync", body, &resp)
}

// PlaceHFOrderTest checks a spot order, including its signature, without sending it to the matching engine
// (Add Order Test)
func (e *Exchange) PlaceHFOrderTest(ctx context.Context, r *PlaceHFOrderRequest) (*PlaceHFOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PlaceHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeHFOrderTestEPL, http.MethodPost, "/v1/hf/orders/test", body, &resp)
}

// BatchPlaceHFOrders places up to 20 spot orders (Batch Add Orders). KuCoin accepts or rejects each order on its own,
// so every result carries its order's outcome: Success is false and FailMessage says why for a rejected order
func (e *Exchange) BatchPlaceHFOrders(ctx context.Context, orders []PlaceHFOrderRequest) ([]BatchPlacedHFOrder, error) {
	body, err := hfOrderListBody(orders)
	if err != nil {
		return nil, err
	}
	var resp []BatchPlacedHFOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, batchPlaceHFOrdersEPL, http.MethodPost, "/v1/hf/orders/multi", body, &resp)
}

// BatchPlaceHFOrdersSync places up to 20 spot orders and returns their states once matching has finished
// (Batch Add Orders Sync). Every result carries its order's outcome: Success is false and FailMessage says why for a
// rejected order
func (e *Exchange) BatchPlaceHFOrdersSync(ctx context.Context, orders []PlaceHFOrderRequest) ([]BatchPlacedHFOrderSync, error) {
	body, err := hfOrderListBody(orders)
	if err != nil {
		return nil, err
	}
	var resp []BatchPlacedHFOrderSync
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, batchPlaceHFOrdersSyncEPL, http.MethodPost, "/v1/hf/orders/multi/sync", body, &resp)
}

// hfOrderListBody returns the body the batch order endpoints take, naming the position of an order that fails
// validation
func hfOrderListBody(orders []PlaceHFOrderRequest) (map[string]any, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	orderList := make([]map[string]any, len(orders))
	for i := range orders {
		item, err := orders[i].Payload()
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		orderList[i] = item
	}
	return map[string]any{"orderList": orderList}, nil
}

// CancelHFOrderByOrderID requests the cancellation of a spot order (Cancel Order By OrderId). KuCoin answers before
// the matching engine processes the request, so the outcome follows on the order websocket channel
func (e *Exchange) CancelHFOrderByOrderID(ctx context.Context, orderID, symbol string) (*OrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelHFOrderByOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/"+orderID, params), nil, &resp)
}

// CancelHFOrderByOrderIDSync cancels a spot order and returns its state once the cancellation has finished
// (Cancel Order By OrderId Sync)
func (e *Exchange) CancelHFOrderByOrderIDSync(ctx context.Context, orderID, symbol string) (*CancelHFOrderByOrderIDSyncResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *CancelHFOrderByOrderIDSyncResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelHFOrderByOrderIDSyncEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/sync/"+orderID, params), nil, &resp)
}

// CancelHFOrderByClientOrderID requests the cancellation of a spot order by its client order ID
// (Cancel Order By ClientOid). KuCoin answers before the matching engine processes the request
func (e *Exchange) CancelHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*ClientOrderIDResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ClientOrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelHFOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/client-order/"+clientOrderID, params), nil, &resp)
}

// CancelHFOrderByClientOrderIDSync cancels a spot order by its client order ID and returns its state once the
// cancellation has finished (Cancel Order By ClientOid Sync)
func (e *Exchange) CancelHFOrderByClientOrderIDSync(ctx context.Context, clientOrderID, symbol string) (*CancelHFOrderByClientOrderIDSyncResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *CancelHFOrderByClientOrderIDSyncResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelHFOrderByClientOrderIDSyncEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/sync/client-order/"+clientOrderID, params), nil, &resp)
}

// CancelPartialHFOrder cancels part of a spot order's size without changing its place in the queue
// (Cancel Partial Order)
func (e *Exchange) CancelPartialHFOrder(ctx context.Context, orderID, symbol string, cancelSize float64) (*CancelPartialHFOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if cancelSize <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("cancelSize", strconv.FormatFloat(cancelSize, 'f', -1, 64))
	var resp *CancelPartialHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelPartialHFOrderEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/cancel/"+orderID, params), nil, &resp)
}

// CancelAllHFOrdersBySymbol requests the cancellation of every open spot order of a symbol and returns KuCoin's
// acknowledgement, "success" (Cancel All Orders By Symbol)
func (e *Exchange) CancelAllHFOrdersBySymbol(ctx context.Context, symbol string) (string, error) {
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp string
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllHFOrdersBySymbolEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders", params), nil, &resp)
}

// CancelAllHFOrders requests the cancellation of every open spot order and reports the outcome per symbol
// (Cancel All Orders)
func (e *Exchange) CancelAllHFOrders(ctx context.Context) (*CancelAllHFOrdersResponse, error) {
	var resp *CancelAllHFOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllHFOrdersEPL, http.MethodDelete, "/v1/hf/orders/cancelAll", nil, &resp)
}

// ModifyHFOrder changes the price or size of a spot order (Modify Order). KuCoin cancels the order and places a new
// one, whose ID it returns; an order whose new size is below its filled size is cancelled without a replacement
func (e *Exchange) ModifyHFOrder(ctx context.Context, r *ModifyHFOrderRequest) (*ModifyHFOrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if r.NewPrice == 0 && r.NewSize == 0 {
		return nil, errNewPriceOrNewSizeRequired
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	if r.OrderID != "" {
		body["orderId"] = r.OrderID
	}
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
	}
	if r.NewPrice != 0 {
		body["newPrice"] = strconv.FormatFloat(r.NewPrice, 'f', -1, 64)
	}
	if r.NewSize != 0 {
		body["newSize"] = strconv.FormatFloat(r.NewSize, 'f', -1, 64)
	}
	var resp *ModifyHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyHFOrderEPL, http.MethodPost, "/v1/hf/orders/alter", body, &resp)
}

// GetHFOrderByOrderID returns a spot order (Get Order By OrderId). Cancelled orders can be queried for 2 days and
// filled orders for 7; KuCoin returns no order outside those windows, which is reported as order.ErrOrderNotFound
func (e *Exchange) GetHFOrderByOrderID(ctx context.Context, orderID, symbol string) (*HFOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *HFOrderDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getHFOrderByOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/"+orderID, params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, order.ErrOrderNotFound
	}
	return resp, nil
}

// GetHFOrderByClientOrderID returns a spot order by its client order ID (Get Order By ClientOid). Cancelled orders
// can be queried for 2 days and filled orders for 7; KuCoin returns no order outside those windows, which is
// reported as order.ErrOrderNotFound
func (e *Exchange) GetHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*HFOrderDetailResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *HFOrderDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getHFOrderByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/client-order/"+clientOrderID, params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, order.ErrOrderNotFound
	}
	return resp, nil
}

// GetSymbolsWithOpenHFOrder returns the symbols that have open spot orders (Get Symbols With Open Order)
func (e *Exchange) GetSymbolsWithOpenHFOrder(ctx context.Context) (*SymbolsWithOpenHFOrderResponse, error) {
	var resp *SymbolsWithOpenHFOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSymbolsWithOpenHFOrderEPL, http.MethodGet, "/v1/hf/orders/active/symbols", nil, &resp)
}

// GetOpenHFOrders returns a symbol's open spot orders, latest update first (Get Open Orders)
func (e *Exchange) GetOpenHFOrders(ctx context.Context, symbol string) ([]HFOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []HFOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOpenHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/active", params), nil, &resp)
}

// GetOpenHFOrdersByPage returns a page of a symbol's open spot orders, latest first (Get Open Orders By Page). Zero
// page values leave KuCoin's defaults
func (e *Exchange) GetOpenHFOrdersByPage(ctx context.Context, symbol string, pageNumber, pageSize uint64) (*OpenHFOrdersPageResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if pageNumber != 0 {
		params.Set("pageNum", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *OpenHFOrdersPageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOpenHFOrdersByPageEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/active/page", params), nil, &resp)
}

// GetClosedHFOrders returns a symbol's closed spot orders, latest update first (Get Closed Orders). Cancelled orders
// can be queried for 2 days and filled orders for 7
func (e *Exchange) GetClosedHFOrders(ctx context.Context, r *ClosedHFOrdersRequest) (*ClosedHFOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !r.StartAt.IsZero() && !r.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(r.StartAt, r.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
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
	if !r.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(r.StartAt.UnixMilli(), 10))
	}
	if !r.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(r.EndAt.UnixMilli(), 10))
	}
	var resp *ClosedHFOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getClosedHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/done", params), nil, &resp)
}

// GetHFTradeHistory returns a symbol's spot fills, latest first (Get Trade History). Fills of orders that are no
// longer open can be queried for 7 days; KuCoin ignores the other filters when an order ID is given
func (e *Exchange) GetHFTradeHistory(ctx context.Context, r *HFTradeHistoryRequest) (*HFTradeHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !r.StartAt.IsZero() && !r.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(r.StartAt, r.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
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
	if !r.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(r.StartAt.UnixMilli(), 10))
	}
	if !r.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(r.EndAt.UnixMilli(), 10))
	}
	var resp *HFTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getHFTradeHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/fills", params), nil, &resp)
}

// GetDCP returns the disconnection protection (deadman switch) setting (Get DCP). KuCoin returns no setting when
// none is set, which leaves the response at its zero value
func (e *Exchange) GetDCP(ctx context.Context) (*DCPResponse, error) {
	resp := new(DCPResponse)
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getDCPEPL, http.MethodGet, "/v1/hf/orders/dead-cancel-all/query", nil, resp)
}

// SetDCP sets the disconnection protection (deadman switch), which cancels the open spot orders of the given
// symbols, or of every symbol when none are given, unless it is renewed within timeout seconds (Set DCP). A timeout
// of -1 removes the setting; otherwise it is 5 to 86400
func (e *Exchange) SetDCP(ctx context.Context, timeout int64, symbols []string) (*SetDCPResponse, error) {
	if timeout == 0 {
		return nil, errTimeoutRequired
	}
	body := make(map[string]any)
	body["timeout"] = timeout
	if len(symbols) != 0 {
		body["symbols"] = strings.Join(symbols, ",")
	}
	var resp *SetDCPResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, setDCPEPL, http.MethodPost, "/v1/hf/orders/dead-cancel-all", body, &resp)
}

// PlaceStopOrder places a spot or margin stop order (Add Stop Order). KuCoin has abandoned this endpoint without
// documenting a Classic replacement; the wrapper still uses it
func (e *Exchange) PlaceStopOrder(ctx context.Context, r *PlaceStopOrderRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Side == "":
		return nil, order.ErrSideIsInvalid
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.Type == "":
		return nil, order.ErrTypeIsInvalid
	case r.StopPrice <= 0:
		return nil, errStopPriceRequired
	}
	if err := validateOrderAmounts(r.Type, r.Price, r.Size, r.Funds); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	body["side"] = r.Side
	body["symbol"] = r.Symbol
	body["type"] = r.Type
	body["stopPrice"] = strconv.FormatFloat(r.StopPrice, 'f', -1, 64)
	if r.ClientOrderID != "" {
		body["clientOid"] = r.ClientOrderID
	}
	if r.Remark != "" {
		body["remark"] = r.Remark
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
	if r.TradeType != "" {
		body["tradeType"] = r.TradeType
	}
	if r.Stop != "" {
		body["stop"] = r.Stop
	}
	var resp *OrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeStopOrderEPL, http.MethodPost, "/v1/stop-order", body, &resp)
}

// BatchCancelStopOrders requests the cancellation of stop orders, filtered by symbol, trade type and order IDs
// (Batch Cancel Stop Orders)
func (e *Exchange) BatchCancelStopOrders(ctx context.Context, r *BatchCancelStopOrdersRequest) (*CancelledOrderIDsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if r.TradeType != "" {
		params.Set("tradeType", r.TradeType)
	}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	var resp *CancelledOrderIDsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, batchCancelStopOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancel", params), nil, &resp)
}

// GetStopOrderList returns a page of untriggered stop orders, latest first (Get Stop Orders List)
func (e *Exchange) GetStopOrderList(ctx context.Context, r *StopOrderListRequest) (*StopOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !r.StartAt.IsZero() && !r.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(r.StartAt, r.EndAt); err != nil {
			return nil, err
		}
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
	if !r.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(r.StartAt.UnixMilli(), 10))
	}
	if !r.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(r.EndAt.UnixMilli(), 10))
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
	var resp *StopOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStopOrderListEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order", params), nil, &resp)
}

// GetStopOrderByOrderID returns a stop order (Get Stop Order By OrderId)
func (e *Exchange) GetStopOrderByOrderID(ctx context.Context, orderID string) (*StopOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *StopOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStopOrderByOrderIDEPL, http.MethodGet, "/v1/stop-order/"+orderID, nil, &resp)
}

// GetStopOrderByClientOrderID returns the stop orders with a client order ID (Get Stop Order By ClientOid)
func (e *Exchange) GetStopOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) ([]StopOrder, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []StopOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStopOrderByClientOrderIDEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order/queryOrderByClientOid", params), nil, &resp)
}

// PlaceOCOOrder places a one-cancels-the-other spot order (Add OCO Order). KuCoin has abandoned this endpoint without
// documenting a Classic replacement; the wrapper still uses it
func (e *Exchange) PlaceOCOOrder(ctx context.Context, r *PlaceOCOOrderRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.ClientOrderID == "":
		return nil, order.ErrClientOrderIDMustBeSet
	case r.Side == "":
		return nil, order.ErrSideIsInvalid
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.Price <= 0:
		return nil, limits.ErrPriceBelowMin
	case r.Size <= 0:
		return nil, limits.ErrAmountBelowMin
	case r.StopPrice <= 0:
		return nil, errStopPriceRequired
	case r.LimitPrice <= 0:
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
	if r.Remark != "" {
		body["remark"] = r.Remark
	}
	if r.TradeType != "" {
		body["tradeType"] = r.TradeType
	}
	var resp *OrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOCOOrderEPL, http.MethodPost, "/v3/oco/order", body, &resp)
}

// CancelOCOOrderByOrderID requests the cancellation of an OCO order and returns the IDs of its two orders
// (Cancel OCO Order By OrderId). KuCoin has abandoned this endpoint; the wrapper still uses it
func (e *Exchange) CancelOCOOrderByOrderID(ctx context.Context, orderID string) (*CancelledOrderIDsResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelledOrderIDsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOCOOrderByOrderIDEPL, http.MethodDelete, "/v3/oco/order/"+orderID, nil, &resp)
}

// CancelOCOOrderByClientOrderID requests the cancellation of an OCO order by its client order ID and returns the IDs
// of its two orders (Cancel OCO Order By ClientOid). KuCoin has abandoned this endpoint; the wrapper still uses it
func (e *Exchange) CancelOCOOrderByClientOrderID(ctx context.Context, clientOrderID string) (*CancelledOrderIDsResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	var resp *CancelledOrderIDsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOCOOrderByClientOrderIDEPL, http.MethodDelete, "/v3/oco/client-order/"+clientOrderID, nil, &resp)
}

// CancelOCOMultipleOrders requests the cancellation of OCO orders, every one unless order IDs or a symbol narrow it
// (Batch Cancel OCO Order). KuCoin has abandoned this endpoint; the wrapper still uses it
func (e *Exchange) CancelOCOMultipleOrders(ctx context.Context, orderIDs []string, symbol string) (*CancelledOrderIDsResponse, error) {
	params := url.Values{}
	if len(orderIDs) != 0 {
		params.Set("orderIds", strings.Join(orderIDs, ","))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *CancelledOrderIDsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOCOMultipleOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v3/oco/orders", params), nil, &resp)
}

// GetOCOOrderList returns a page of OCO orders, latest first (Get OCO Order List). KuCoin has abandoned this
// endpoint; the wrapper still uses it, and GetStopOrderList with Stop set to oco lists OCO orders instead
func (e *Exchange) GetOCOOrderList(ctx context.Context, r *OCOOrderListRequest) (*OCOOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !r.StartAt.IsZero() && !r.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(r.StartAt, r.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(r.StartAt.UnixMilli(), 10))
	}
	if !r.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(r.EndAt.UnixMilli(), 10))
	}
	if len(r.OrderIDs) != 0 {
		params.Set("orderIds", strings.Join(r.OrderIDs, ","))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	if r.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(r.CurrentPage, 10))
	}
	var resp *OCOOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOCOOrderListEPL, http.MethodGet, common.EncodeURLValues("/v3/oco/orders", params), nil, &resp)
}

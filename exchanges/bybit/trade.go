package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Trade errors
var errBatchOutcomeMismatch = errors.New("batch order outcomes do not match the orders")

// PlaceOrder places a spot, margin, perpetual, futures or options order (Place Order)
func (e *Exchange) PlaceOrder(ctx context.Context, r *PlaceOrderRequest) (*OrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/create", nil, body, &resp, createOrderBudgets.forCategory(r.Category).rest)
}

// AmendOrder amends an unfilled or partially filled order (Amend Order)
func (e *Exchange) AmendOrder(ctx context.Context, r *AmendOrderRequest) (*OrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/amend", nil, body, &resp, amendOrderBudgets.forCategory(r.Category).rest)
}

// CancelTradeOrder cancels an unfilled or partially filled order (Cancel Order)
func (e *Exchange) CancelTradeOrder(ctx context.Context, r *CancelOrderRequest) (*OrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/cancel", nil, body, &resp, cancelOrderBudgets.forCategory(r.Category).rest)
}

// GetOpenOrders returns open orders in real time, or the most recent 500 closed orders when OpenOnly is 1
// (Get Open & Closed Orders)
func (e *Exchange) GetOpenOrders(ctx context.Context, r *OpenOrdersRequest) (*OpenOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.BaseCoin.IsEmpty() {
		params.Set("baseCoin", r.BaseCoin.Upper().String())
	}
	if !r.SettleCoin.IsEmpty() {
		params.Set("settleCoin", r.SettleCoin.Upper().String())
	}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.OrderLinkID != "" {
		params.Set("orderLinkId", r.OrderLinkID)
	}
	if r.OpenOnly != 0 {
		params.Set("openOnly", strconv.FormatUint(r.OpenOnly, 10))
	}
	if r.OrderFilter != "" {
		params.Set("orderFilter", r.OrderFilter)
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *OpenOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/realtime", params, nil, &resp, getOpenOrdersEPL)
}

// CancelAllTradeOrders cancels all open orders of a category, or of a symbol, base coin or settle coin within it
// (Cancel All Orders)
func (e *Exchange) CancelAllTradeOrders(ctx context.Context, r *CancelAllOrdersRequest) (*CancelAllOrdersResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	body := make(map[string]any)
	body["category"] = r.Category
	if r.Symbol != "" {
		body["symbol"] = r.Symbol
	}
	if !r.BaseCoin.IsEmpty() {
		body["baseCoin"] = r.BaseCoin.Upper().String()
	}
	if !r.SettleCoin.IsEmpty() {
		body["settleCoin"] = r.SettleCoin.Upper().String()
	}
	if r.OrderFilter != "" {
		body["orderFilter"] = r.OrderFilter
	}
	if r.StopOrderType != "" {
		body["stopOrderType"] = r.StopOrderType
	}
	var resp *CancelAllOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/cancel-all", nil, body, &resp, cancelAllOrdersBudgets.forCategory(r.Category).rest)
}

// GetTradeOrderHistory returns orders of the last two years. Orders are created and cancelled asynchronously, so the
// history can lag; the order websocket stream is the real-time source (Get Order History)
func (e *Exchange) GetTradeOrderHistory(ctx context.Context, r *OrderHistoryRequest) (*OrderHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.BaseCoin.IsEmpty() {
		params.Set("baseCoin", r.BaseCoin.Upper().String())
	}
	if !r.SettleCoin.IsEmpty() {
		params.Set("settleCoin", r.SettleCoin.Upper().String())
	}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.OrderLinkID != "" {
		params.Set("orderLinkId", r.OrderLinkID)
	}
	if r.OrderFilter != "" {
		params.Set("orderFilter", r.OrderFilter)
	}
	if r.OrderStatus != "" {
		params.Set("orderStatus", r.OrderStatus)
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
	}
	if !r.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *OrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/history", params, nil, &resp, getTradeOrderHistoryEPL)
}

// GetExecution returns fills of the last two years, newest first (Get Trade History)
func (e *Exchange) GetExecution(ctx context.Context, r *ExecutionRequest) (*ExecutionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if r.OrderLinkID != "" {
		params.Set("orderLinkId", r.OrderLinkID)
	}
	if !r.BaseCoin.IsEmpty() {
		params.Set("baseCoin", r.BaseCoin.Upper().String())
	}
	if !r.SettleCoin.IsEmpty() {
		params.Set("settleCoin", r.SettleCoin.Upper().String())
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
	}
	if !r.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	if r.ExecutionType != "" {
		params.Set("execType", r.ExecutionType)
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *ExecutionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/execution/list", params, nil, &resp, getExecutionEPL)
}

// PlaceBatchOrder places several orders of one category in a single request. Each order of the result carries Bybit's
// verdict on it, since one rejected order does not fail the request (Batch Place Order)
func (e *Exchange) PlaceBatchOrder(ctx context.Context, r *BatchPlaceOrderRequest) (*BatchPlaceOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	ctx, epl := batchOrderLimit(ctx, createBatchOrderBudgets, r.Category, len(r.Request))
	var resp *BatchPlaceOrderResponse
	outcomes, err := e.sendBatchOrderRequest(ctx, "/v5/order/create-batch", body, epl, &resp)
	if err != nil {
		return nil, err
	}
	if err := checkBatchCounts(len(r.Request), len(resp.List), len(outcomes)); err != nil {
		return nil, err
	}
	for i := range resp.List {
		resp.List[i].Code = outcomes[i].Code
		resp.List[i].Message = outcomes[i].Message
	}
	return resp, nil
}

// BatchAmendOrder amends several orders of one category in a single request. Each order of the result carries Bybit's
// verdict on it, since one rejected amendment does not fail the request (Batch Amend Order)
func (e *Exchange) BatchAmendOrder(ctx context.Context, r *BatchAmendOrderRequest) (*BatchOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	ctx, epl := batchOrderLimit(ctx, amendBatchOrderBudgets, r.Category, len(r.Request))
	var resp *BatchOrderResponse
	outcomes, err := e.sendBatchOrderRequest(ctx, "/v5/order/amend-batch", body, epl, &resp)
	if err != nil {
		return nil, err
	}
	return applyBatchOrderOutcomes(resp, outcomes, len(r.Request))
}

// CancelBatchOrder cancels several orders of one category in a single request. Each order of the result carries
// Bybit's verdict on it, since one failed cancellation does not fail the request (Batch Cancel Order)
func (e *Exchange) CancelBatchOrder(ctx context.Context, r *BatchCancelOrderRequest) (*BatchOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	ctx, epl := batchOrderLimit(ctx, cancelBatchOrderBudgets, r.Category, len(r.Request))
	var resp *BatchOrderResponse
	outcomes, err := e.sendBatchOrderRequest(ctx, "/v5/order/cancel-batch", body, epl, &resp)
	if err != nil {
		return nil, err
	}
	return applyBatchOrderOutcomes(resp, outcomes, len(r.Request))
}

// sendBatchOrderRequest sends a batch order request, decodes its result into result and returns Bybit's verdict on
// each order, which it reports in retExtInfo in request order
func (e *Exchange) sendBatchOrderRequest(ctx context.Context, path string, body map[string]any, f request.EndpointLimit, result any) ([]BatchOrderOutcome, error) {
	envelope := new(RestResponse)
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, body, envelope, f); err != nil {
		return nil, err
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil, common.ErrNoResponse
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return nil, fmt.Errorf("decoding batch order result: %w", err)
	}
	var extInfo BatchOrderExtInfo
	if err := json.Unmarshal(envelope.ReturnExtendedInfo, &extInfo); err != nil {
		return nil, fmt.Errorf("decoding batch order outcomes: %w", err)
	}
	return extInfo.List, nil
}

// applyBatchOrderOutcomes sets each order's verdict on the result of a batch amend or cancel request of the given number
// of orders
func applyBatchOrderOutcomes(resp *BatchOrderResponse, outcomes []BatchOrderOutcome, orders int) (*BatchOrderResponse, error) {
	if err := checkBatchCounts(orders, len(resp.List), len(outcomes)); err != nil {
		return nil, err
	}
	for i := range resp.List {
		resp.List[i].Code = outcomes[i].Code
		resp.List[i].Message = outcomes[i].Message
	}
	return resp, nil
}

// checkBatchCounts returns an error unless a batch order reply holds one result and one outcome for each order sent, as
// Bybit answers, in request order, since otherwise neither can be matched to its order
func checkBatchCounts(orders, results, outcomes int) error {
	if results != orders || outcomes != orders {
		return fmt.Errorf("%w: %d orders, %d results, %d outcomes", errBatchOutcomeMismatch, orders, results, outcomes)
	}
	return nil
}

// GetBorrowQuota returns the most a spot order can trade, with and without spot margin borrowing
// (Get Borrow Quota (Spot))
func (e *Exchange) GetBorrowQuota(ctx context.Context, r *BorrowQuotaRequest) (*BorrowQuotaResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Category == "":
		return nil, errCategoryNotSet
	case r.Symbol == "":
		return nil, errSymbolMissing
	case r.Side == "":
		return nil, order.ErrSideIsInvalid
	}
	params := url.Values{}
	params.Set("category", r.Category)
	params.Set("symbol", r.Symbol)
	params.Set("side", r.Side)
	var resp *BorrowQuotaResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/spot-borrow-check", params, nil, &resp, getBorrowQuotaEPL)
}

// SetDisconnectCancelAll sets the window after which Bybit cancels the account's orders when its private websocket
// connection drops. The connection must subscribe to the dcp topic for it to take effect (Set Disconnect Cancel All)
func (e *Exchange) SetDisconnectCancelAll(ctx context.Context, r *SetDisconnectCancelAllRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.TimeWindow < time.Second {
		return errDisconnectTimeWindowNotSet
	}
	body := make(map[string]any)
	body["timeWindow"] = int64(r.TimeWindow / time.Second)
	if r.Product != "" {
		body["product"] = r.Product
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/disconnected-cancel-all", nil, body, nil, setDisconnectCancelAllEPL)
}

// PreCheckOrder returns the account's initial and maintenance margin rates before and after placing an order, without
// placing it (Pre Check Order)
func (e *Exchange) PreCheckOrder(ctx context.Context, r *PlaceOrderRequest) (*PreCheckOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *PreCheckOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/pre-check", nil, body, &resp, preCheckOrderBudgets.forCategory(r.Category).rest)
}

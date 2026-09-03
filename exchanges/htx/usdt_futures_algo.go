package htx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// PlaceV5AlgoOrder places a strategy order.
func (e *Exchange) PlaceV5AlgoOrder(ctx context.Context, req *V5AlgoOrderRequest) (*V5AlgoAcknowledgementsResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5AlgoAcknowledgementsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/algo/order", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelV5AlgoOrders cancels strategy orders.
func (e *Exchange) CancelV5AlgoOrders(ctx context.Context, req []*V5CancelAlgoOrderRequest) (*V5AlgoAcknowledgementsResponse, error) {
	if len(req) == 0 {
		return nil, common.ErrEmptyParams
	}
	var resp *V5AlgoAcknowledgementsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/algo/cancel_orders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5AlgoOrder gets a strategy order.
func (e *Exchange) GetV5AlgoOrder(ctx context.Context, algoID, clientOrderID, orderType string) (*V5AlgoOrdersResponse, error) {
	params := url.Values{}
	if algoID != "" {
		params.Set("algo_id", algoID)
	}
	if clientOrderID != "" {
		params.Set("algo_client_order_id", clientOrderID)
	}
	params.Set("type", orderType)
	var resp *V5AlgoOrdersResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5OpenAlgoOrders gets open strategy orders.
func (e *Exchange) GetV5OpenAlgoOrders(ctx context.Context, req *V5OpenAlgoOrdersRequest) (*V5AlgoOrdersResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.ContractCode != "" {
		params.Set("contract_code", req.ContractCode)
	}
	if req.AlgoID != "" {
		params.Set("algo_id", req.AlgoID)
	}
	if req.ClientOrderID != "" {
		params.Set("algo_client_order_id", req.ClientOrderID)
	}
	params.Set("type", req.Type)
	if req.From != 0 {
		params.Set("from", strconv.FormatUint(req.From, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5AlgoOrdersResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order/opens", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5AlgoOrderHistory gets historical strategy orders.
func (e *Exchange) GetV5AlgoOrderHistory(ctx context.Context, req *V5AlgoOrderHistoryRequest) (*V5AlgoOrdersResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.ContractCode != "" {
		params.Set("contract_code", req.ContractCode)
	}
	if req.MarginMode != "" {
		params.Set("margin_mode", req.MarginMode)
	}
	if req.States != "" {
		params.Set("states", req.States)
	}
	params.Set("type", req.Type)
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.From != 0 {
		params.Set("from", strconv.FormatUint(req.From, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5AlgoOrdersResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/algo/order/history", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

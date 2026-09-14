package htx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// PlaceV5Order places a USDT-margined order.
func (e *Exchange) PlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5OrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5OrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/order", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceV5BatchOrders places multiple USDT-margined orders.
func (e *Exchange) PlaceV5BatchOrders(ctx context.Context, req []*V5OrderRequest) (*V5BatchOrderResponse, error) {
	if len(req) == 0 {
		return nil, common.ErrEmptyParams
	}
	var resp *V5BatchOrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/batch_orders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelV5Order cancels a USDT-margined order.
func (e *Exchange) CancelV5Order(ctx context.Context, code currency.Pair, orderID, clientOrderID string) (*V5OrderResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	req := &V5CancelOrderRequest{
		ContractCode:  codeValue,
		OrderID:       orderID,
		ClientOrderID: clientOrderID,
	}
	var resp *V5OrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_order", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelV5BatchOrders cancels multiple USDT-margined orders.
func (e *Exchange) CancelV5BatchOrders(ctx context.Context, req *V5CancelBatchOrdersRequest) (*V5BatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5BatchOrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_batch_orders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelAllV5Orders cancels all matching USDT-margined orders.
func (e *Exchange) CancelAllV5Orders(ctx context.Context, code currency.Pair, side, positionSide string) (*V5CancelAllOrdersResponse, error) {
	req := &V5CancelAllOrdersRequest{
		Side:         side,
		PositionSide: positionSide,
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		req.ContractCode = codeValue
	}
	var resp *V5CancelAllOrdersResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_all_orders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SetV5CancelAfter enables or disables automatic order cancellation.
func (e *Exchange) SetV5CancelAfter(ctx context.Context, req *V5CancelAfterRequest) (*V5CancelAfterResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5CancelAfterResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel-after", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CloseV5Position closes a position at market.
func (e *Exchange) CloseV5Position(ctx context.Context, req *V5ClosePositionRequest) (*V5OrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5OrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/position", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CloseAllV5Positions closes all positions at market.
func (e *Exchange) CloseAllV5Positions(ctx context.Context) (*V5BatchOrderResponse, error) {
	var resp *V5BatchOrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/position_all", nil, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5Order gets a USDT-margined order.
func (e *Exchange) GetV5Order(ctx context.Context, code currency.Pair, marginMode, orderID, clientOrderID string) (*V5OrderQueryResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	var resp *V5OrderQueryResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5OpenOrders gets open USDT-margined orders.
func (e *Exchange) GetV5OpenOrders(ctx context.Context, code currency.Pair, marginMode, orderID, clientOrderID string, from, limit uint64, direct string) (*V5OrdersQueryResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if from != 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if direct != "" {
		params.Set("direct", direct)
	}
	var resp *V5OrdersQueryResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/opens", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5OrderHistory gets historical USDT-margined orders.
func (e *Exchange) GetV5OrderHistory(ctx context.Context, req *V5OrderHistoryRequest) (*V5OrdersQueryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	params.Set("contract_code", req.ContractCode)
	params.Set("margin_mode", req.MarginMode)
	if req.States != "" {
		params.Set("states", req.States)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.PriceMatch != "" {
		params.Set("price_match", req.PriceMatch)
	}
	if req.TimeInForce != "" {
		params.Set("time_in_force", req.TimeInForce)
	}
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
	var resp *V5OrdersQueryResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/history", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5OrderDetails gets recent execution details for a USDT-margined contract.
func (e *Exchange) GetV5OrderDetails(ctx context.Context, req *V5OrderDetailsRequest) (*V5OrderDetailsResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	params.Set("contract_code", req.ContractCode)
	if req.OrderID != "" {
		params.Set("order_id", req.OrderID)
	}
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.From != "" {
		params.Set("from", req.From)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5OrderDetailsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/details", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5OpenPositions gets current USDT-margined positions.
func (e *Exchange) GetV5OpenPositions(ctx context.Context, code currency.Pair) (*V5OpenPositionsResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	var resp *V5OpenPositionsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/position/opens", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetRWAProductList returns the real world asset products, optionally for one settlement coin (Get Product List). The
// user quota is only filled for signed requests, so the request is signed whenever credentials are set
func (e *Exchange) GetRWAProductList(ctx context.Context, coin currency.Code) (*RWAProductListResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *RWAProductListResponse
	if e.AreCredentialsValid(ctx) {
		return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/rwa/product", params, nil, &resp, getRWAProductListEPL)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/rwa/product", params), getRWAProductListEPL, &resp)
}

// PlaceRWAOrder stakes into or redeems shares of a real world asset product (Place Order)
func (e *Exchange) PlaceRWAOrder(ctx context.Context, req *PlaceRWAOrderRequest) (*PlaceRWAOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == 0 {
		return nil, errProductIDRequired
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	if req.StakeAmount <= 0 && req.RedeemShares <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["orderType"] = req.OrderType
	body["coin"] = req.Coin.Upper().String()
	body["orderLinkId"] = req.OrderLinkID
	if req.StakeAmount > 0 {
		body["stakeAmount"] = strconv.FormatFloat(req.StakeAmount, 'f', -1, 64)
	}
	if req.RedeemShares > 0 {
		body["redeemShares"] = strconv.FormatFloat(req.RedeemShares, 'f', -1, 64)
	}
	if req.AccountType != "" {
		body["accountType"] = req.AccountType
	}
	var resp *PlaceRWAOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/rwa/place-order", nil, body, &resp, placeRWAOrderEPL)
}

// GetRWAPositionList returns the real world asset positions (Get Position List)
func (e *Exchange) GetRWAPositionList(ctx context.Context) (*RWAPositionListResponse, error) {
	var resp *RWAPositionListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/rwa/position", nil, nil, &resp, getRWAPositionListEPL)
}

// GetRWAOrderList returns real world asset stake and redeem orders (Get Order List)
func (e *Exchange) GetRWAOrderList(ctx context.Context, req *RWAOrderListRequest) (*RWAOrderListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.OrderType != "" {
		params.Set("orderType", req.OrderType)
	}
	if req.ProductID != 0 {
		params.Set("productId", strconv.FormatUint(req.ProductID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *RWAOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/rwa/order", params, nil, &resp, getRWAOrderListEPL)
}

// GetRWANAVChart returns the daily net asset value of a real world asset product (Get NAV Chart)
func (e *Exchange) GetRWANAVChart(ctx context.Context, req *RWANAVChartRequest) (*RWANAVChartResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == 0 {
		return nil, errProductIDRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("productId", strconv.FormatUint(req.ProductID, 10))
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	var resp *RWANAVChartResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/rwa/nav-chart", params), getRWANAVChartEPL, &resp)
}

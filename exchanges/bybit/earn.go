package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Earn request validation errors
var (
	errProductIDRequired        = errors.New("product ID is required")
	errPositionIDRequired       = errors.New("position ID is required")
	errAwardIDRequired          = errors.New("award ID is required")
	errSpecCodeRequired         = errors.New("spec code is required")
	errAutoInvestStatusRequired = errors.New("auto-invest status is required")
	errAPRRangeRequired         = errors.New("APR time range is required")
	errOnChainProductIDRejected = errors.New("product ID is not supported for OnChain")
)

// GetEarnProductInfo returns Flexible Savings and On-Chain Earn products (Get Product Info)
func (e *Exchange) GetEarnProductInfo(ctx context.Context, category string, coin currency.Code) (*EarnProductInfoResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *EarnProductInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/product", params), defaultEPL, &resp)
}

// PlaceEarnOrder stakes into or redeems from a Flexible Savings or On-Chain Earn product (Stake / Redeem)
func (e *Exchange) PlaceEarnOrder(ctx context.Context, req *PlaceEarnOrderRequest) (*PlaceEarnOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.AccountType == "" {
		return nil, errMissingAccountType
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	if req.InterestCard != nil {
		if req.InterestCard.AwardID == 0 {
			return nil, errAwardIDRequired
		}
		if req.InterestCard.SpecCode == "" {
			return nil, errSpecCodeRequired
		}
	}
	body := make(map[string]any)
	body["category"] = req.Category
	body["orderType"] = req.OrderType
	body["accountType"] = req.AccountType
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["coin"] = req.Coin.Upper().String()
	body["productId"] = req.ProductID
	body["orderLinkId"] = req.OrderLinkID
	if req.RedeemPositionID != "" {
		body["redeemPositionId"] = req.RedeemPositionID
	}
	if req.ToAccountType != "" {
		body["toAccountType"] = req.ToAccountType
	}
	if req.InterestCard != nil {
		body["interestCard"] = map[string]any{
			"awardId":  req.InterestCard.AwardID,
			"specCode": req.InterestCard.SpecCode,
		}
	}
	var resp *PlaceEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/place-order", nil, body, &resp, defaultEPL)
}

// GetEarnOrderHistory returns Flexible Savings and On-Chain Earn stake and redeem orders (Get Stake/Redeem Order History)
func (e *Exchange) GetEarnOrderHistory(ctx context.Context, req *EarnOrderHistoryRequest) (*EarnOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Category == "OnChain" && req.OrderID == "" && req.OrderLinkID == "" {
		return nil, errEitherOrderIDOROrderLinkIDRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
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
	var resp *EarnOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/order", params, nil, &resp, defaultEPL)
}

// GetEarnStakedPosition returns Flexible Savings and On-Chain Earn positions (Get Staked Position)
func (e *Exchange) GetEarnStakedPosition(ctx context.Context, req *EarnStakedPositionRequest) (*EarnStakedPositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	var resp *EarnStakedPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/position", params, nil, &resp, defaultEPL)
}

// ModifyEarnPosition switches auto-reinvestment of an On-Chain Earn fixed term position on or off (Modify Position)
func (e *Exchange) ModifyEarnPosition(ctx context.Context, req *ModifyEarnPositionRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Category == "" {
		return errCategoryNotSet
	}
	if req.ProductID == 0 {
		return errProductIDRequired
	}
	if req.PositionID == 0 {
		return errPositionIDRequired
	}
	var autoReinvest uint8
	if req.AutoReinvest {
		autoReinvest = 1
	}
	body := make(map[string]any)
	body["category"] = req.Category
	body["productId"] = req.ProductID
	body["positionId"] = req.PositionID
	body["autoReinvest"] = autoReinvest
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/position/modify", nil, body, nil, defaultEPL)
}

// GetEarnYieldHistory returns Flexible Savings and On-Chain Earn yield distributions of the past three months (Get Yield History)
func (e *Exchange) GetEarnYieldHistory(ctx context.Context, req *EarnYieldHistoryRequest) (*EarnYieldHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Category == "OnChain" && req.ProductID != "" {
		return nil, errOnChainProductIDRejected
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
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
	var resp *EarnYieldHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/yield", params, nil, &resp, defaultEPL)
}

// GetEarnHourlyYieldHistory returns Flexible Savings hourly yields (Get Hourly Yield History)
func (e *Exchange) GetEarnHourlyYieldHistory(ctx context.Context, req *EarnYieldHistoryRequest) (*EarnHourlyYieldHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
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
	var resp *EarnHourlyYieldHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/hourly-yield", params, nil, &resp, defaultEPL)
}

// GetEarnAPRHistory returns up to six months of a Flexible Savings or On-Chain Earn product's APR (Get APR History)
func (e *Exchange) GetEarnAPRHistory(ctx context.Context, req *EarnAPRHistoryRequest) (*EarnAPRHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	params.Set("productId", req.ProductID)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *EarnAPRHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/apr-history", params), defaultEPL, &resp)
}

// GetEarnCouponList returns the interest-rate coupons and Dual Assets reward cards of a product category (Get Coupon List)
func (e *Exchange) GetEarnCouponList(ctx context.Context, category string) (*EarnCouponListResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	var resp *EarnCouponListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/coupons", params, nil, &resp, getEarnCouponListEPL)
}

// GetBYUSDTProductInfo returns the BYUSDT product (Get Product Info). The holding, quota, mint and savings balance
// fields are only filled for signed requests, so the request is signed whenever credentials are set
func (e *Exchange) GetBYUSDTProductInfo(ctx context.Context, coin currency.Code) (*BYUSDTProductInfoResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	var resp *BYUSDTProductInfoResponse
	if e.AreCredentialsValid(ctx) {
		return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/token/product", params, nil, &resp, defaultEPL)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/token/product", params), defaultEPL, &resp)
}

// PlaceBYUSDTOrder mints BYUSDT from Flexible Savings USDT or redeems it back to USDT (Place Order)
func (e *Exchange) PlaceBYUSDTOrder(ctx context.Context, req *PlaceBYUSDTOrderRequest) (*PlaceBYUSDTOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.AccountType == "" {
		return nil, errMissingAccountType
	}
	body := make(map[string]any)
	body["coin"] = req.Coin.Upper().String()
	body["orderLinkId"] = req.OrderLinkID
	body["orderType"] = req.OrderType
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["accountType"] = req.AccountType
	var resp *PlaceBYUSDTOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/token/place-order", nil, body, &resp, defaultEPL)
}

// GetBYUSDTOrderList returns BYUSDT mint and redeem orders (Get Order List)
func (e *Exchange) GetBYUSDTOrderList(ctx context.Context, req *BYUSDTOrderListRequest) (*BYUSDTOrderListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderType != "" {
		params.Set("orderType", req.OrderType)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *BYUSDTOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/token/order", params, nil, &resp, defaultEPL)
}

// GetBYUSDTPositionInfo returns the BYUSDT holding and its yield (Get Position Info)
func (e *Exchange) GetBYUSDTPositionInfo(ctx context.Context, coin currency.Code) (*BYUSDTPositionInfoResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	var resp *BYUSDTPositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/token/position", params, nil, &resp, defaultEPL)
}

// GetBYUSDTDailyYieldRecords returns BYUSDT daily yields (Get Daily Yield Records)
func (e *Exchange) GetBYUSDTDailyYieldRecords(ctx context.Context, req *BYUSDTYieldRecordsRequest) (*BYUSDTDailyYieldRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *BYUSDTDailyYieldRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/token/yield", params, nil, &resp, defaultEPL)
}

// GetBYUSDTHourlyYieldRecords returns BYUSDT hourly yields (Get Hourly Yield Records)
func (e *Exchange) GetBYUSDTHourlyYieldRecords(ctx context.Context, req *BYUSDTYieldRecordsRequest) (*BYUSDTHourlyYieldRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.Unix(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.Unix(), 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *BYUSDTHourlyYieldRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/token/hourly-yield", params, nil, &resp, defaultEPL)
}

// GetBYUSDTHistoryAPR returns the BYUSDT daily APR over a time range: 1 for 7 days, 2 for 30 days or 3 for 180 days
// (Get History APR)
func (e *Exchange) GetBYUSDTHistoryAPR(ctx context.Context, coin currency.Code, timeRange uint64) (*BYUSDTHistoryAPRResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if timeRange == 0 {
		return nil, errAPRRangeRequired
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	params.Set("range", strconv.FormatUint(timeRange, 10))
	var resp *BYUSDTHistoryAPRResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/token/history-apr", params), defaultEPL, &resp)
}

// GetFixedSavingProductInfo returns Fixed Saving products, optionally for one coin (Get Product Info)
func (e *Exchange) GetFixedSavingProductInfo(ctx context.Context, coin currency.Code) (*FixedSavingProductInfoResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *FixedSavingProductInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/fixed-term/product", params), getFixedSavingProductInfoEPL, &resp)
}

// PlaceFixedSavingOrder subscribes to a Fixed Saving product (Place Order)
func (e *Exchange) PlaceFixedSavingOrder(ctx context.Context, req *PlaceFixedSavingOrderRequest) (*PlaceFixedSavingOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.AccountType == "" {
		return nil, errMissingAccountType
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["category"] = req.Category
	body["coin"] = req.Coin.Upper().String()
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["accountType"] = req.AccountType
	body["orderLinkId"] = req.OrderLinkID
	if req.AutoInvest {
		body["autoInvest"] = true
	}
	var resp *PlaceFixedSavingOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/fixed-term/place-order", nil, body, &resp, placeFixedSavingOrderEPL)
}

// RedeemFixedSaving redeems a FundPool Fixed Saving position early (Redeem)
func (e *Exchange) RedeemFixedSaving(ctx context.Context, req *RedeemFixedSavingRequest) (*RedeemFixedSavingResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.PositionID == "" {
		return nil, errPositionIDRequired
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["category"] = req.Category
	body["positionId"] = req.PositionID
	var resp *RedeemFixedSavingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/fixed-term/redeem", nil, body, &resp, redeemFixedSavingEPL)
}

// GetFixedSavingPositionInfo returns active Fixed Saving positions (Get Position Info)
func (e *Exchange) GetFixedSavingPositionInfo(ctx context.Context, req *FixedSavingPositionInfoRequest) (*FixedSavingPositionInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	var resp *FixedSavingPositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/fixed-term/position", params, nil, &resp, getFixedSavingPositionInfoEPL)
}

// GetFixedSavingOrderList returns Fixed Saving stake, redeem and reinvest orders (Get Order List)
func (e *Exchange) GetFixedSavingOrderList(ctx context.Context, req *FixedSavingOrderListRequest) (*FixedSavingOrderListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID != "" && req.Category == "" {
		return nil, errCategoryNotSet
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.OrderType != "" {
		params.Set("orderType", req.OrderType)
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
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
	var resp *FixedSavingOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/fixed-term/order", params, nil, &resp, getFixedSavingOrderListEPL)
}

// SetFixedSavingAutoInvest enables or disables auto-reinvestment of a Fixed Saving position (Set Auto-Invest)
func (e *Exchange) SetFixedSavingAutoInvest(ctx context.Context, req *SetFixedSavingAutoInvestRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.ProductID == "" {
		return errProductIDRequired
	}
	if req.Category == "" {
		return errCategoryNotSet
	}
	if req.PositionID == "" {
		return errPositionIDRequired
	}
	if req.Status == "" {
		return errAutoInvestStatusRequired
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["category"] = req.Category
	body["positionId"] = req.PositionID
	body["status"] = req.Status
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/fixed-term/position/auto-invest", nil, body, nil, setFixedSavingAutoInvestEPL)
}

// GetFlexibleSavingAutoSavings returns the Flexible Savings auto savings settings and APR of up to 50 coins, or of every
// supported coin when coins is empty (Get Auto Savings Settings)
func (e *Exchange) GetFlexibleSavingAutoSavings(ctx context.Context, coins []currency.Code) (*FlexibleSavingAutoSavingsResponse, error) {
	params := url.Values{}
	for _, coin := range coins {
		if coin.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		params.Add("coins", coin.Upper().String())
	}
	var resp *FlexibleSavingAutoSavingsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/flexible-saving/auto-savings", params, nil, &resp, getFlexibleSavingAutoSavingsEPL)
}

// SetFlexibleSavingAutoSavings enables or disables Flexible Savings auto savings for one coin, or for every coin when
// the coin is empty (Set Auto Savings Settings)
func (e *Exchange) SetFlexibleSavingAutoSavings(ctx context.Context, req *SetFlexibleSavingAutoSavingsRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	body := make(map[string]any)
	body["isSelected"] = req.IsSelected
	if !req.Coin.IsEmpty() {
		body["coin"] = req.Coin.Upper().String()
	}
	if req.PurchaseImmediately {
		body["purchaseImmediately"] = true
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/flexible-saving/auto-savings", nil, body, nil, setFlexibleSavingAutoSavingsEPL)
}

// GetAirdropProducts returns the Hold To Earn airdrop products (Get Airdrop Products). Signed requests return only the
// products the account is eligible for, with its personal APR, so the request is signed whenever credentials are set
func (e *Exchange) GetAirdropProducts(ctx context.Context) (*AirdropProductsResponse, error) {
	var resp *AirdropProductsResponse
	if e.AreCredentialsValid(ctx) {
		return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/hold-to-earn/product", nil, nil, &resp, defaultEPL)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v5/earn/hold-to-earn/product", defaultEPL, &resp)
}

// GetAirdropDailyPNLRecords returns up to three months of distributed Hold To Earn airdrop yields, newest first (Get
// Airdrop Daily PnL Records)
func (e *Exchange) GetAirdropDailyPNLRecords(ctx context.Context, req *AirdropDailyPNLRecordsRequest) (*AirdropDailyPNLRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Limit == 0 {
		return nil, errLimitRequired
	}
	if !req.TimeStart.IsZero() && !req.TimeEnd.IsZero() {
		if err := common.StartEndTimeCheck(req.TimeStart, req.TimeEnd); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.TimeStart.IsZero() {
		params.Set("timeStart", strconv.FormatInt(req.TimeStart.Unix(), 10))
	}
	if !req.TimeEnd.IsZero() {
		params.Set("timeEnd", strconv.FormatInt(req.TimeEnd.Unix(), 10))
	}
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *AirdropDailyPNLRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/hold-to-earn/yield-history", params, nil, &resp, defaultEPL)
}

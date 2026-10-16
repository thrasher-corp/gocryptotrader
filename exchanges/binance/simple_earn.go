package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetSimpleEarnCollateralRecord returns Simple Earn flexible products used as loan collateral
func (e *Exchange) GetSimpleEarnCollateralRecord(ctx context.Context, req *SimpleEarnCollateralRecordRequest) (*SimpleEarnCollateralRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	var resp *SimpleEarnCollateralRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/collateralRecord", params, sapiDefaultRate, &resp)
}

// GetFlexiblePersonalLeftQuota returns the account's remaining subscription quota for a flexible product
func (e *Exchange) GetFlexiblePersonalLeftQuota(ctx context.Context, productID string) (*SimpleEarnPersonalLeftQuotaResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	params := url.Values{}
	params.Set("productId", productID)
	var resp *SimpleEarnPersonalLeftQuotaResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/personalLeftQuota", params, personalLeftQuotaRate, &resp)
}

// GetFlexibleProductPosition returns the account's flexible product positions
func (e *Exchange) GetFlexibleProductPosition(ctx context.Context, req *SimpleEarnFlexiblePositionRequest) (*SimpleEarnFlexiblePositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	setCurrentSizeParams(params, req.Current, req.Size)
	var resp *SimpleEarnFlexiblePositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/position", params, getFlexibleSimpleEarnProductPositionRate, &resp)
}

// GetFlexibleRedemptionRecord returns flexible product redemptions
func (e *Exchange) GetFlexibleRedemptionRecord(ctx context.Context, req *SimpleEarnFlexibleRedemptionRecordRequest) (*SimpleEarnFlexibleRedemptionRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.RedeemID > 0 {
		params.Set("redeemId", strconv.FormatUint(req.RedeemID, 10))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp *SimpleEarnFlexibleRedemptionRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/redemptionRecord", params, getRedemptionRecordRate, &resp)
}

// GetFlexibleRewardHistory returns flexible product rewards
func (e *Exchange) GetFlexibleRewardHistory(ctx context.Context, req *SimpleEarnFlexibleRewardHistoryRequest) (*SimpleEarnFlexibleRewardHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	var resp *SimpleEarnFlexibleRewardHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/rewardsRecord", params, getRewardHistoryRate, &resp)
}

// GetFlexibleSubscriptionPreview returns the estimated rewards of a flexible product subscription
func (e *Exchange) GetFlexibleSubscriptionPreview(ctx context.Context, productID string, amount float64) (*SimpleEarnFlexibleSubscriptionPreviewResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("productId", productID)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SimpleEarnFlexibleSubscriptionPreviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/subscriptionPreview", params, subscriptionPreviewRate, &resp)
}

// GetFlexibleSubscriptionRecord returns flexible product subscriptions
func (e *Exchange) GetFlexibleSubscriptionRecord(ctx context.Context, req *SimpleEarnFlexibleSubscriptionRecordRequest) (*SimpleEarnFlexibleSubscriptionRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.PurchaseID > 0 {
		params.Set("purchaseId", strconv.FormatUint(req.PurchaseID, 10))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp *SimpleEarnFlexibleSubscriptionRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/subscriptionRecord", params, getFlexibleSubscriptionRecordRate, &resp)
}

// GetLockedPersonalLeftQuota returns the account's remaining subscription quota for a locked product
func (e *Exchange) GetLockedPersonalLeftQuota(ctx context.Context, projectID string) (*SimpleEarnPersonalLeftQuotaResponse, error) {
	if projectID == "" {
		return nil, errProjectIDRequired
	}
	params := url.Values{}
	params.Set("projectId", projectID)
	var resp *SimpleEarnPersonalLeftQuotaResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/personalLeftQuota", params, personalLeftQuotaRate, &resp)
}

// GetLockedProductPosition returns the account's locked product positions
func (e *Exchange) GetLockedProductPosition(ctx context.Context, req *SimpleEarnLockedPositionRequest) (*SimpleEarnLockedPositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.PositionID > 0 {
		params.Set("positionId", strconv.FormatUint(req.PositionID, 10))
	}
	if req.ProjectID != "" {
		params.Set("projectId", req.ProjectID)
	}
	setCurrentSizeParams(params, req.Current, req.Size)
	var resp *SimpleEarnLockedPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/position", params, getSimpleEarnProductPositionRate, &resp)
}

// GetLockedRedemptionRecord returns locked product redemptions
func (e *Exchange) GetLockedRedemptionRecord(ctx context.Context, req *SimpleEarnLockedRedemptionRecordRequest) (*SimpleEarnLockedRedemptionRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.PositionID > 0 {
		params.Set("positionId", strconv.FormatUint(req.PositionID, 10))
	}
	if req.RedeemID > 0 {
		params.Set("redeemId", strconv.FormatUint(req.RedeemID, 10))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp *SimpleEarnLockedRedemptionRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/history/redemptionRecord", params, getRedemptionRecordRate, &resp)
}

// GetLockedRewardHistory returns locked product rewards
func (e *Exchange) GetLockedRewardHistory(ctx context.Context, req *SimpleEarnLockedRewardHistoryRequest) (*SimpleEarnLockedRewardHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.PositionID > 0 {
		params.Set("positionId", strconv.FormatUint(req.PositionID, 10))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp *SimpleEarnLockedRewardHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/history/rewardsRecord", params, getRewardHistoryRate, &resp)
}

// GetLockedSubscriptionPreview returns the estimated rewards and schedule of a locked product subscription
func (e *Exchange) GetLockedSubscriptionPreview(ctx context.Context, projectID string, amount float64, autoSubscribe bool) ([]SimpleEarnLockedSubscriptionPreview, error) {
	if projectID == "" {
		return nil, errProjectIDRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("projectId", projectID)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	// Always sent, as the endpoint previews auto-subscription when it is left out
	params.Set("autoSubscribe", strconv.FormatBool(autoSubscribe))
	var resp []SimpleEarnLockedSubscriptionPreview
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/subscriptionPreview", params, subscriptionPreviewRate, &resp)
}

// GetLockedSubscriptionRecord returns locked product subscriptions
func (e *Exchange) GetLockedSubscriptionRecord(ctx context.Context, req *SimpleEarnLockedSubscriptionRecordRequest) (*SimpleEarnLockedSubscriptionRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.PurchaseID > 0 {
		params.Set("purchaseId", strconv.FormatUint(req.PurchaseID, 10))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp *SimpleEarnLockedSubscriptionRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/history/subscriptionRecord", params, getLockedSubscriptionRecordsRate, &resp)
}

// GetSimpleEarnRateHistory returns the annual percentage rate history of a flexible product
func (e *Exchange) GetSimpleEarnRateHistory(ctx context.Context, req *SimpleEarnRateHistoryRequest) (*SimpleEarnRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	params.Set("productId", req.ProductID)
	if req.APRPeriod != "" {
		params.Set("aprPeriod", req.APRPeriod)
	}
	var resp *SimpleEarnRateHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/rateHistory", params, simpleEarnRateHistoryRate, &resp)
}

// GetSimpleEarnFlexibleProductList returns the available Simple Earn flexible products
func (e *Exchange) GetSimpleEarnFlexibleProductList(ctx context.Context, asset currency.Code, current, size uint64) (*SimpleEarnFlexibleProductListResponse, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.Upper().String())
	}
	setCurrentSizeParams(params, current, size)
	var resp *SimpleEarnFlexibleProductListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/flexible/list", params, simpleEarnProductsRate, &resp)
}

// GetSimpleEarnLockedProducts returns the available Simple Earn locked products
func (e *Exchange) GetSimpleEarnLockedProducts(ctx context.Context, asset currency.Code, current, size uint64) (*SimpleEarnLockedProductListResponse, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.Upper().String())
	}
	setCurrentSizeParams(params, current, size)
	var resp *SimpleEarnLockedProductListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/locked/list", params, simpleEarnProductsRate, &resp)
}

// RedeemFlexibleProduct redeems some or all of a flexible product
func (e *Exchange) RedeemFlexibleProduct(ctx context.Context, req *RedeemFlexibleProductRequest) (*SimpleEarnRedemptionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if !req.RedeemAll && req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("productId", req.ProductID)
	if req.RedeemAll {
		params.Set("redeemAll", "true")
	}
	if req.Amount > 0 {
		params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	}
	if req.DestinationAccount != "" {
		params.Set("destAccount", req.DestinationAccount)
	}
	var resp *SimpleEarnRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/flexible/redeem", params, sapiDefaultRate, &resp)
}

// RedeemLockedProduct redeems a locked product position
func (e *Exchange) RedeemLockedProduct(ctx context.Context, positionID uint64) (*SimpleEarnRedemptionResponse, error) {
	if positionID == 0 {
		return nil, errPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatUint(positionID, 10))
	var resp *SimpleEarnRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/locked/redeem", params, sapiDefaultRate, &resp)
}

// SetFlexibleAutoSubscribe turns auto-subscription of a flexible product on or off
func (e *Exchange) SetFlexibleAutoSubscribe(ctx context.Context, productID string, autoSubscribe bool) (*EarnSuccessResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	params := url.Values{}
	params.Set("productId", productID)
	params.Set("autoSubscribe", strconv.FormatBool(autoSubscribe))
	var resp *EarnSuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/flexible/setAutoSubscribe", params, setAutoSubscribeRate, &resp)
}

// SetLockedAutoSubscribe turns auto-subscription of a locked product position on or off
func (e *Exchange) SetLockedAutoSubscribe(ctx context.Context, positionID uint64, autoSubscribe bool) (*EarnSuccessResponse, error) {
	if positionID == 0 {
		return nil, errPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatUint(positionID, 10))
	params.Set("autoSubscribe", strconv.FormatBool(autoSubscribe))
	var resp *EarnSuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/locked/setAutoSubscribe", params, setAutoSubscribeRate, &resp)
}

// SetLockedProductRedeemOption sets where a locked product position is paid out at maturity, SPOT or FLEXIBLE
func (e *Exchange) SetLockedProductRedeemOption(ctx context.Context, positionID uint64, redeemTo string) (*EarnSuccessResponse, error) {
	if positionID == 0 {
		return nil, errPositionIDRequired
	}
	if redeemTo == "" {
		return nil, errRedemptionAccountRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatUint(positionID, 10))
	params.Set("redeemTo", redeemTo)
	var resp *EarnSuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/locked/setRedeemOption", params, setRedeemOptionRate, &resp)
}

// SimpleAccount returns the value held in Simple Earn flexible and locked products
func (e *Exchange) SimpleAccount(ctx context.Context) (*SimpleAccountResponse, error) {
	var resp *SimpleAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/simple-earn/account", nil, simpleAccountRate, &resp)
}

// SubscribeToFlexibleProducts subscribes to a flexible product
func (e *Exchange) SubscribeToFlexibleProducts(ctx context.Context, req *SubscribeFlexibleProductRequest) (*SimpleEarnFlexibleSubscriptionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("productId", req.ProductID)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.AutoSubscribe != nil {
		params.Set("autoSubscribe", strconv.FormatBool(*req.AutoSubscribe))
	}
	if req.SourceAccount != "" {
		params.Set("sourceAccount", req.SourceAccount)
	}
	var resp *SimpleEarnFlexibleSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/flexible/subscribe", params, sapiDefaultRate, &resp)
}

// SubscribeToLockedProducts subscribes to a locked product
func (e *Exchange) SubscribeToLockedProducts(ctx context.Context, req *SubscribeLockedProductRequest) (*SimpleEarnLockedSubscriptionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProjectID == "" {
		return nil, errProjectIDRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("projectId", req.ProjectID)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.AutoSubscribe != nil {
		params.Set("autoSubscribe", strconv.FormatBool(*req.AutoSubscribe))
	}
	if req.SourceAccount != "" {
		params.Set("sourceAccount", req.SourceAccount)
	}
	if req.RedeemTo != "" {
		params.Set("redeemTo", req.RedeemTo)
	}
	var resp *SimpleEarnLockedSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/simple-earn/locked/subscribe", params, sapiDefaultRate, &resp)
}

// setCurrentSizeParams sets the current page and page size, leaving out unset values so the endpoint defaults apply
func setCurrentSizeParams(params url.Values, current, size uint64) {
	if current > 0 {
		params.Set("current", strconv.FormatUint(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
}

// historyPageParams builds the time range, current page and page size query of the Earn history endpoints
func historyPageParams(startTime, endTime time.Time, current, size uint64) (url.Values, error) {
	params := url.Values{}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	setCurrentSizeParams(params, current, size)
	return params, nil
}

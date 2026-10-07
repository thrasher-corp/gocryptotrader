package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Binance dropped Auto-Invest from its API reference without a retirement notice and its routes still answer, so
// these endpoints follow the last published reference (developers.binance.com/docs/auto_invest, 2025-02)

var errHistoricalROITypeRequired = errors.New("historical ROI type is required")

// GetTargetAssetList returns the Auto-Invest target assets and their simulated returns
func (e *Exchange) GetTargetAssetList(ctx context.Context, targetAsset currency.Code, size, current uint64) (*AutoInvestTargetAssetListResponse, error) {
	params := url.Values{}
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.Upper().String())
	}
	setCurrentSizeParams(params, current, size)
	var resp *AutoInvestTargetAssetListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/target-asset/list", params, sapiDefaultRate, &resp)
}

// GetTargetAssetROIData returns the simulated return history of a target asset over a period such as ONE_YEAR
func (e *Exchange) GetTargetAssetROIData(ctx context.Context, targetAsset currency.Code, hisRoiType string) ([]AutoInvestTargetAssetROI, error) {
	if targetAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: target asset", currency.ErrCurrencyCodeEmpty)
	}
	if hisRoiType == "" {
		return nil, errHistoricalROITypeRequired
	}
	params := url.Values{}
	params.Set("targetAsset", targetAsset.Upper().String())
	params.Set("hisRoiType", hisRoiType)
	var resp []AutoInvestTargetAssetROI
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/target-asset/roi/list", params, sapiDefaultRate, &resp)
}

// GetAllSourceAssetAndTargetAsset returns every Auto-Invest source asset and target asset
func (e *Exchange) GetAllSourceAssetAndTargetAsset(ctx context.Context) (*AutoInvestAllAssetsResponse, error) {
	var resp *AutoInvestAllAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/all/asset", nil, sapiDefaultRate, &resp)
}

// GetSourceAssetList returns the source assets that can fund recurring or one-time Auto-Invest subscriptions
func (e *Exchange) GetSourceAssetList(ctx context.Context, req *AutoInvestSourceAssetListRequest) (*AutoInvestSourceAssetListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UsageType == "" {
		return nil, errUsageTypeRequired
	}
	params := url.Values{}
	if !req.TargetAsset.IsEmpty() {
		params.Set("targetAsset", req.TargetAsset.Upper().String())
	}
	if req.IndexID > 0 {
		params.Set("indexId", strconv.FormatUint(req.IndexID, 10))
	}
	params.Set("usageType", req.UsageType)
	if req.FlexibleAllowedToUse {
		params.Set("flexibleAllowedToUse", "true")
	}
	if req.SourceType != "" {
		params.Set("sourceType", req.SourceType)
	}
	var resp *AutoInvestSourceAssetListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/source-asset/list", params, sapiDefaultRate, &resp)
}

// InvestmentPlanCreation creates an Auto-Invest plan
func (e *Exchange) InvestmentPlanCreation(ctx context.Context, req *InvestmentPlanCreationRequest) (*AutoInvestPlanResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SourceType == "" {
		return nil, errSourceTypeRequired
	}
	if req.PlanType == "" {
		return nil, errPlanTypeRequired
	}
	params := url.Values{}
	params.Set("sourceType", req.SourceType)
	if req.RequestID != "" {
		params.Set("requestId", req.RequestID)
	}
	params.Set("planType", req.PlanType)
	if req.IndexID > 0 {
		params.Set("indexId", strconv.FormatUint(req.IndexID, 10))
	}
	if err := setInvestmentPlanParams(params, &req.InvestmentPlanSchedule); err != nil {
		return nil, err
	}
	var resp *AutoInvestPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/add", params, autoInvestPlanCreationRate, &resp)
}

// InvestmentPlanAdjustment changes the subscription and portfolio of an Auto-Invest plan
func (e *Exchange) InvestmentPlanAdjustment(ctx context.Context, req *InvestmentPlanAdjustmentRequest) (*AutoInvestPlanResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == 0 {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("planId", strconv.FormatUint(req.PlanID, 10))
	if err := setInvestmentPlanParams(params, &req.InvestmentPlanSchedule); err != nil {
		return nil, err
	}
	var resp *AutoInvestPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/edit", params, autoInvestPlanAdjustmentRate, &resp)
}

// ChangePlanStatus sets an Auto-Invest plan's status to ONGOING, PAUSED or REMOVED
func (e *Exchange) ChangePlanStatus(ctx context.Context, planID uint64, status string) (*AutoInvestPlanStatusResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	if status == "" {
		return nil, errPlanStatusRequired
	}
	params := url.Values{}
	params.Set("planId", strconv.FormatUint(planID, 10))
	params.Set("status", status)
	var resp *AutoInvestPlanStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/edit-status", params, autoInvestPlanStatusRate, &resp)
}

// GetListOfPlans returns the account's Auto-Invest plans of a plan type
func (e *Exchange) GetListOfPlans(ctx context.Context, planType string) (*AutoInvestPlanListResponse, error) {
	if planType == "" {
		return nil, errPlanTypeRequired
	}
	params := url.Values{}
	params.Set("planType", planType)
	var resp *AutoInvestPlanListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/plan/list", params, autoInvestPlanListRate, &resp)
}

// GetHoldingDetailsOfPlan returns the holdings of an Auto-Invest plan, found by plan ID or creation request ID
func (e *Exchange) GetHoldingDetailsOfPlan(ctx context.Context, planID uint64, requestID string) (*AutoInvestPlanHoldingResponse, error) {
	params := url.Values{}
	if planID > 0 {
		params.Set("planId", strconv.FormatUint(planID, 10))
	}
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	var resp *AutoInvestPlanHoldingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/plan/id", params, sapiDefaultRate, &resp)
}

// GetSubscriptionsTransactionHistory returns Auto-Invest subscription transactions
func (e *Exchange) GetSubscriptionsTransactionHistory(ctx context.Context, req *AutoInvestSubscriptionHistoryRequest) ([]AutoInvestSubscriptionTransaction, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.PlanID > 0 {
		params.Set("planId", strconv.FormatUint(req.PlanID, 10))
	}
	if !req.TargetAsset.IsEmpty() {
		params.Set("targetAsset", req.TargetAsset.Upper().String())
	}
	if req.PlanType != "" {
		params.Set("planType", req.PlanType)
	}
	var resp []AutoInvestSubscriptionTransaction
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/history/list", params, sapiDefaultRate, &resp)
}

// GetIndexDetail returns an Auto-Invest index and its asset allocation
func (e *Exchange) GetIndexDetail(ctx context.Context, indexID uint64) (*AutoInvestIndexDetailResponse, error) {
	if indexID == 0 {
		return nil, errIndexIDIsRequired
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatUint(indexID, 10))
	var resp *AutoInvestIndexDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/index/info", params, sapiDefaultRate, &resp)
}

// GetIndexLinkedPlanPositionDetails returns the account's position in an index-linked Auto-Invest plan
func (e *Exchange) GetIndexLinkedPlanPositionDetails(ctx context.Context, indexID uint64) (*IndexLinkedPlanPositionResponse, error) {
	if indexID == 0 {
		return nil, errIndexIDIsRequired
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatUint(indexID, 10))
	var resp *IndexLinkedPlanPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/index/user-summary", params, sapiDefaultRate, &resp)
}

// OneTimeTransaction makes a one-time Auto-Invest purchase for a portfolio plan, an index or a set of target assets
func (e *Exchange) OneTimeTransaction(ctx context.Context, req *OneTimeTransactionRequest) (*AutoInvestOneTimeTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SourceType == "" {
		return nil, errSourceTypeRequired
	}
	if req.SubscriptionAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.SourceAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: source asset", currency.ErrCurrencyCodeEmpty)
	}
	if req.PlanID == 0 && req.IndexID == 0 && len(req.Details) == 0 {
		return nil, fmt.Errorf("%w: give a plan ID, an index ID or portfolio details", errPortfolioDetailRequired)
	}
	params := url.Values{}
	params.Set("sourceType", req.SourceType)
	if req.RequestID != "" {
		params.Set("requestId", req.RequestID)
	}
	params.Set("subscriptionAmount", strconv.FormatFloat(req.SubscriptionAmount, 'f', -1, 64))
	params.Set("sourceAsset", req.SourceAsset.Upper().String())
	if req.FlexibleAllowedToUse {
		params.Set("flexibleAllowedToUse", "true")
	}
	if req.PlanID > 0 {
		params.Set("planId", strconv.FormatUint(req.PlanID, 10))
	}
	if req.IndexID > 0 {
		params.Set("indexId", strconv.FormatUint(req.IndexID, 10))
	}
	if err := setPortfolioDetailParams(params, req.Details); err != nil {
		return nil, err
	}
	var resp *AutoInvestOneTimeTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/lending/auto-invest/one-off", params, autoInvestOneTimeTransactionRate, &resp)
}

// GetOneTimeTransactionStatus returns the status of a one-time Auto-Invest transaction, found by transaction ID or
// request ID
func (e *Exchange) GetOneTimeTransactionStatus(ctx context.Context, transactionID uint64, requestID string) (*AutoInvestOneTimeTransactionStatusResponse, error) {
	if transactionID == 0 && requestID == "" {
		return nil, errTransactionIDRequired
	}
	params := url.Values{}
	if transactionID > 0 {
		params.Set("transactionId", strconv.FormatUint(transactionID, 10))
	}
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	var resp *AutoInvestOneTimeTransactionStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/one-off/status", params, sapiDefaultRate, &resp)
}

// IndexLinkedPlanRedemption redeems a percentage, such as 10, 20 or 100, of an index-linked plan's holdings
func (e *Exchange) IndexLinkedPlanRedemption(ctx context.Context, indexID, redemptionPercentage uint64, requestID string) (*AutoInvestRedemptionResponse, error) {
	if indexID == 0 {
		return nil, errIndexIDIsRequired
	}
	if redemptionPercentage == 0 {
		return nil, errInvalidPercentageAmount
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatUint(indexID, 10))
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	params.Set("redemptionPercentage", strconv.FormatUint(redemptionPercentage, 10))
	var resp *AutoInvestRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/lending/auto-invest/redeem", params, autoInvestRedemptionRate, &resp)
}

// GetIndexLinkedPlanRedemption returns the redemptions of an index-linked plan redemption request
func (e *Exchange) GetIndexLinkedPlanRedemption(ctx context.Context, req *IndexLinkedPlanRedemptionHistoryRequest) ([]IndexLinkedPlanRedemption, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.RequestID == "" {
		return nil, errRequestIDRequired
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	params.Set("requestId", req.RequestID)
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	var resp []IndexLinkedPlanRedemption
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/redeem/history", params, sapiDefaultRate, &resp)
}

// GetIndexLinkedPlanRebalanceDetails returns index-linked plan rebalances
func (e *Exchange) GetIndexLinkedPlanRebalanceDetails(ctx context.Context, req *EarnHistoryRequest) ([]IndexLinkedPlanRebalance, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp []IndexLinkedPlanRebalance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/lending/auto-invest/rebalance/history", params, sapiDefaultRate, &resp)
}

// setInvestmentPlanParams validates and sets the subscription schedule and portfolio shared by plan creation and
// adjustment
func setInvestmentPlanParams(params url.Values, s *InvestmentPlanSchedule) error {
	if s.SubscriptionAmount <= 0 {
		return fmt.Errorf("%w: subscription amount", limits.ErrAmountBelowMin)
	}
	if s.SubscriptionCycle == "" {
		return fmt.Errorf("%w: empty", errInvalidSubscriptionCycle)
	}
	if s.SourceAsset.IsEmpty() {
		return fmt.Errorf("%w: source asset", currency.ErrCurrencyCodeEmpty)
	}
	if len(s.Details) == 0 {
		return errPortfolioDetailRequired
	}
	params.Set("subscriptionAmount", strconv.FormatFloat(s.SubscriptionAmount, 'f', -1, 64))
	params.Set("subscriptionCycle", s.SubscriptionCycle)
	if s.SubscriptionStartDay > 0 {
		params.Set("subscriptionStartDay", strconv.FormatUint(s.SubscriptionStartDay, 10))
	}
	if s.SubscriptionStartWeekday != "" {
		params.Set("subscriptionStartWeekday", s.SubscriptionStartWeekday)
	}
	// Always sent, as the hour is required and hour 0 is valid
	params.Set("subscriptionStartTime", strconv.FormatUint(s.SubscriptionStartTime, 10))
	params.Set("sourceAsset", s.SourceAsset.Upper().String())
	if s.FlexibleAllowedToUse {
		params.Set("flexibleAllowedToUse", "true")
	}
	return setPortfolioDetailParams(params, s.Details)
}

// setPortfolioDetailParams validates portfolio details and sets them in the indexed form details[0].targetAsset and
// details[0].percentage
func setPortfolioDetailParams(params url.Values, details []PortfolioDetail) error {
	for i := range details {
		if details[i].TargetAsset.IsEmpty() {
			return fmt.Errorf("%w: portfolio detail target asset", currency.ErrCurrencyCodeEmpty)
		}
		if details[i].Percentage == 0 {
			return errInvalidPercentageAmount
		}
		prefix := "details[" + strconv.Itoa(i) + "]."
		params.Set(prefix+"targetAsset", details[i].TargetAsset.Upper().String())
		params.Set(prefix+"percentage", strconv.FormatUint(details[i].Percentage, 10))
	}
	return nil
}

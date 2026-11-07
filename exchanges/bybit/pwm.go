package bybit

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

// PWM request validation errors
var (
	errPlanIDRequired        = errors.New("investment plan ID is required")
	errFundIDRequired        = errors.New("fund ID is required")
	errRequestLinkIDRequired = errors.New("request link ID is required")
	errFundNameRequired      = errors.New("fund name is required")
	errPlanNameRequired      = errors.New("investment plan name is required")
	errPlanTypeRequired      = errors.New("investment plan type is required")
	errAccountUIDRequired    = errors.New("account UID is required")
	errActionRequired        = errors.New("action is required")
	errNoProductsProvided    = errors.New("at least one product is required")
	errNoFundsProvided       = errors.New("at least one fund is required")
)

// TransferPWMFunds transfers funds between two custodian sub-accounts of a fund; it must be called with the fund
// custodian sub-account's API key (Fund Transfer Between Sub-Accounts)
func (e *Exchange) TransferPWMFunds(ctx context.Context, req *PWMFundTransferRequest) (*PWMFundTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TransferID == "" {
		return nil, errMissingTransferID
	}
	if req.FromUserID == 0 {
		return nil, fmt.Errorf("%w: from user ID", errMissingUserID)
	}
	if req.ToUserID == 0 {
		return nil, fmt.Errorf("%w: to user ID", errMissingUserID)
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := make(map[string]any)
	body["transferId"] = req.TransferID
	body["fromUserId"] = req.FromUserID
	body["toUserId"] = req.ToUserID
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["coin"] = req.Coin.Upper().String()
	var resp *PWMFundTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/fund-transfer", nil, body, &resp, defaultEPL)
}

// GetPWMFundTransferRecords returns the fund transfer between custodian sub-accounts with the given ID; it must be
// called with the fund custodian sub-account's API key (Get Fund Transfer Records). The ID is required: without one
// Bybit lists up to 20 transfers, in a reply the docs do not describe
func (e *Exchange) GetPWMFundTransferRecords(ctx context.Context, transferID string, fromUserID uint64) (*PWMFundTransferRecordsResponse, error) {
	if transferID == "" {
		return nil, errMissingTransferID
	}
	params := url.Values{}
	params.Set("transferId", transferID)
	if fromUserID != 0 {
		params.Set("fromUserId", strconv.FormatUint(fromUserID, 10))
	}
	var resp *PWMFundTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/query-fund-transfer-result", params, nil, &resp, defaultEPL)
}

// GetAllPWMInvestmentPlans returns the account's investment plans (Get All Investment Plans)
func (e *Exchange) GetAllPWMInvestmentPlans(ctx context.Context, req *PWMInvestmentPlansRequest) (*PWMInvestmentPlansResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.PlanID != "" {
		params.Set("planId", req.PlanID)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PWMInvestmentPlansResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/all", params, nil, &resp, defaultEPL)
}

// GetPWMInvestmentPlanDetail returns the assets and positions of an Active or Closed investment plan (Get Investment
// Plan Detail)
func (e *Exchange) GetPWMInvestmentPlanDetail(ctx context.Context, planID string) (*PWMInvestmentPlanDetailResponse, error) {
	if planID == "" {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("planId", planID)
	var resp *PWMInvestmentPlanDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/detail", params, nil, &resp, defaultEPL)
}

// GetPendingPWMInvestmentPlanDetail returns the configured products of a PendingSubscription investment plan (Get
// Pending Investment Plan Detail)
func (e *Exchange) GetPendingPWMInvestmentPlanDetail(ctx context.Context, planID string) (*PWMPendingInvestmentPlanDetailResponse, error) {
	if planID == "" {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("planId", planID)
	var resp *PWMPendingInvestmentPlanDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/new-plan", params, nil, &resp, defaultEPL)
}

// ClaimPWMWithdrawableFunds transfers the withdrawable funds of an Active investment plan out of it (Claim Withdrawable
// Funds)
func (e *Exchange) ClaimPWMWithdrawableFunds(ctx context.Context, req *PWMClaimRequest) (*PWMClaimResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	body := make(map[string]any)
	body["planId"] = req.PlanID
	body["orderLinkId"] = req.OrderLinkID
	if req.ToAccountType != "" {
		body["toAccountType"] = req.ToAccountType
	}
	var resp *PWMClaimResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/investment-plan/claim", nil, body, &resp, defaultEPL)
}

// GetPWMAssetTrend returns an investment plan's daily asset value (Get Asset Trend)
func (e *Exchange) GetPWMAssetTrend(ctx context.Context, req *PWMAssetTrendRequest) (*PWMAssetTrendResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("planId", req.PlanID)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *PWMAssetTrendResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/asset-trend", params, nil, &resp, defaultEPL)
}

// GetPWMFundHistoricalNAV returns a fund's daily net asset value (Get Fund Historical NAV)
func (e *Exchange) GetPWMFundHistoricalNAV(ctx context.Context, req *PWMFundHistoricalNAVRequest) (*PWMFundHistoricalNAVResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FundID == "" {
		return nil, errFundIDRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("fundId", req.FundID)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *PWMFundHistoricalNAVResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/fund-nav", params, nil, &resp, defaultEPL)
}

// SubscribePWMInvestmentPlan subscribes to a PendingSubscription investment plan (Subscribe Investment Plan)
func (e *Exchange) SubscribePWMInvestmentPlan(ctx context.Context, req *PWMSubscribeRequest) (*PWMSubscribeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	body := make(map[string]any)
	body["planId"] = req.PlanID
	body["orderLinkId"] = req.OrderLinkID
	if req.AccountType != "" {
		body["accountType"] = req.AccountType
	}
	var resp *PWMSubscribeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/investment-plan/subscribe", nil, body, &resp, defaultEPL)
}

// InvestMoreInPWMInvestmentPlan adds to a product of an Active investment plan (Invest More)
func (e *Exchange) InvestMoreInPWMInvestmentPlan(ctx context.Context, req *PWMInvestMoreRequest) (*PWMInvestMoreResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	body := make(map[string]any)
	body["planId"] = req.PlanID
	body["category"] = req.Category
	body["productId"] = req.ProductID
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["orderLinkId"] = req.OrderLinkID
	if req.AccountType != "" {
		body["accountType"] = req.AccountType
	}
	var resp *PWMInvestMoreResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/investment-plan/invest-more", nil, body, &resp, defaultEPL)
}

// RedeemPWMInvestmentPlan redeems from a product of an investment plan; fund products redeem shares, other products an
// amount (Redeem)
func (e *Exchange) RedeemPWMInvestmentPlan(ctx context.Context, req *PWMRedeemRequest) (*PWMRedeemResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	// Equity funds redeem shares, and FundPool and On-chain Earn redemptions name their position. The docs do not say
	// which other categories count as fund products, so any other redemption needs only shares or an amount
	switch {
	case req.Category == "equityFund" && req.Shares <= 0:
		return nil, fmt.Errorf("%w: an equity fund redeems shares", order.ErrAmountIsInvalid)
	case req.Shares <= 0 && req.Amount <= 0:
		return nil, order.ErrAmountIsInvalid
	case (req.Category == "fundPool" || req.Category == "onchainEarn") && req.PositionID == "":
		return nil, errPositionIDRequired
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	body := make(map[string]any)
	body["planId"] = req.PlanID
	body["category"] = req.Category
	body["productId"] = req.ProductID
	body["orderLinkId"] = req.OrderLinkID
	if req.Shares > 0 {
		body["shares"] = strconv.FormatFloat(req.Shares, 'f', -1, 64)
	}
	if req.Amount > 0 {
		body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	if req.PositionID != "" {
		body["positionId"] = req.PositionID
	}
	var resp *PWMRedeemResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/investment-plan/redeem", nil, body, &resp, defaultEPL)
}

// GetPWMInvestmentPlanOrders returns the subscription, redemption and auto-reinvest orders of the account's investment
// plans, newest first (Get Investment Plan Orders)
func (e *Exchange) GetPWMInvestmentPlanOrders(ctx context.Context, req *PWMInvestmentPlanOrdersRequest) (*PWMInvestmentPlanOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.PlanID != "" {
		params.Set("planId", req.PlanID)
	}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
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
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	var resp *PWMInvestmentPlanOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/investment-plan/order", params, nil, &resp, defaultEPL)
}

// GetPWMSubscribableProductInfo returns the products a customised investment plan can hold, by category (Get
// Subscribable Product Info)
func (e *Exchange) GetPWMSubscribableProductInfo(ctx context.Context) (*PWMSubscribableProductInfoResponse, error) {
	var resp *PWMSubscribableProductInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v5/earn/pwm/customize-plan/product", defaultEPL, &resp)
}

// CreatePWMCustomisedInvestmentPlan creates an investment plan from subscribable products and subscribes to it in one
// step; an account may hold at most 20 Active and Pending plans (Create Customise Investment Plan)
func (e *Exchange) CreatePWMCustomisedInvestmentPlan(ctx context.Context, req *PWMCustomisedPlanRequest) (*PWMCustomisedPlanResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Products) == 0 {
		return nil, errNoProductsProvided
	}
	products := make([]map[string]any, len(req.Products))
	for i := range req.Products {
		if req.Products[i].Category == "" {
			return nil, fmt.Errorf("%w: product %d", errCategoryNotSet, i)
		}
		if req.Products[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: product %d", order.ErrAmountIsInvalid, i)
		}
		// The product ID and fund name are passed through from GetPWMSubscribableProductInfo, where a fund has no
		// product ID and other products no fund name, so both are always sent
		products[i] = make(map[string]any)
		products[i]["category"] = req.Products[i].Category
		products[i]["productId"] = req.Products[i].ProductID
		products[i]["fundName"] = req.Products[i].FundName
		products[i]["amount"] = strconv.FormatFloat(req.Products[i].Amount, 'f', -1, 64)
	}
	body := make(map[string]any)
	body["products"] = products
	if req.AccountType != "" {
		body["accountType"] = req.AccountType
	}
	if req.OrderLinkID != "" {
		body["orderLinkId"] = req.OrderLinkID
	}
	var resp *PWMCustomisedPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/customize-plan/create", nil, body, &resp, defaultEPL)
}

// GetAllPWMFunds returns the funds the institution created through the API, by descending fund ID (Get All Funds)
func (e *Exchange) GetAllPWMFunds(ctx context.Context, req *PWMFundsRequest) (*PWMFundsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.FundID != "" {
		params.Set("fundId", req.FundID)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PWMFundsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/asset-manager/all-funds", params, nil, &resp, defaultEPL)
}

// SettlePWMFundProfit settles the profit share of an Active fund; repeating the request link ID returns the settlement's
// current status (Settle Fund Profit)
func (e *Exchange) SettlePWMFundProfit(ctx context.Context, fundID, requestLinkID string) (*PWMSettleFundProfitResponse, error) {
	if fundID == "" {
		return nil, errFundIDRequired
	}
	if requestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	body := make(map[string]any)
	body["fundId"] = fundID
	body["reqLinkId"] = requestLinkID
	var resp *PWMSettleFundProfitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/settle-profit", nil, body, &resp, defaultEPL)
}

// CreatePWMFund creates a fund pending subscription; an institution may create at most 10 funds (Create Fund (Pending
// Subscription))
func (e *Exchange) CreatePWMFund(ctx context.Context, req *PWMCreateFundRequest) (*PWMCreateFundResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FundName == "" {
		return nil, errFundNameRequired
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.RequestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	body := make(map[string]any)
	body["fundName"] = req.FundName
	body["coin"] = req.Coin.Upper().String()
	body["profitShareRate"] = strconv.FormatFloat(req.ProfitShareRate, 'f', -1, 64)
	body["managementFeeRate"] = strconv.FormatFloat(req.ManagementFeeRate, 'f', -1, 64)
	body["reqLinkId"] = req.RequestLinkID
	if req.FundIntroduction != "" {
		body["fundIntroduction"] = req.FundIntroduction
	}
	var resp *PWMCreateFundResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/create-fund", nil, body, &resp, defaultEPL)
}

// CreatePWMInvestmentPlan creates an investment plan of PendingSubscribe funds for a user (Create Investment Plan)
func (e *Exchange) CreatePWMInvestmentPlan(ctx context.Context, req *PWMCreateInvestmentPlanRequest) (*PWMCreateInvestmentPlanResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountUID == "" {
		return nil, errAccountUIDRequired
	}
	if req.PlanName == "" {
		return nil, errPlanNameRequired
	}
	if req.PlanType == "" {
		return nil, errPlanTypeRequired
	}
	if len(req.InvestmentDistribution) == 0 {
		return nil, errNoFundsProvided
	}
	if req.RequestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	distribution := make([]map[string]any, len(req.InvestmentDistribution))
	for i := range req.InvestmentDistribution {
		if req.InvestmentDistribution[i].ProductID == "" {
			return nil, fmt.Errorf("%w: fund %d", errProductIDRequired, i)
		}
		if req.InvestmentDistribution[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: fund %d", order.ErrAmountIsInvalid, i)
		}
		distribution[i] = make(map[string]any)
		distribution[i]["productId"] = req.InvestmentDistribution[i].ProductID
		distribution[i]["amount"] = strconv.FormatFloat(req.InvestmentDistribution[i].Amount, 'f', -1, 64)
	}
	body := make(map[string]any)
	body["accountUid"] = req.AccountUID
	body["planName"] = req.PlanName
	body["planType"] = req.PlanType
	body["investmentDistribution"] = distribution
	body["reqLinkId"] = req.RequestLinkID
	var resp *PWMCreateInvestmentPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/create-investment-plan", nil, body, &resp, defaultEPL)
}

// GetPWMAssetManagerInvestmentPlans returns the investment plans the institution created through the API (Get
// Investment Plans)
func (e *Exchange) GetPWMAssetManagerInvestmentPlans(ctx context.Context, req *PWMAssetManagerInvestmentPlansRequest) (*PWMAssetManagerInvestmentPlansResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.PlanID != "" {
		params.Set("planId", req.PlanID)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.SubscriptionUID != "" {
		params.Set("subscriptionUid", req.SubscriptionUID)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PWMAssetManagerInvestmentPlansResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/asset-manager/get-investment-plan", params, nil, &resp, defaultEPL)
}

// ManagePWMInvestmentPlan closes or deletes an investment plan, or adds funds to it and updates their amounts (Manage
// Investment Plan)
func (e *Exchange) ManagePWMInvestmentPlan(ctx context.Context, req *PWMManageInvestmentPlanRequest) (*PWMManageInvestmentPlanResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if req.RequestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	body := make(map[string]any)
	body["planId"] = req.PlanID
	body["reqLinkId"] = req.RequestLinkID
	if req.UpdateStatus != "" {
		body["updateStatus"] = req.UpdateStatus
	}
	if len(req.UpdateFunds) != 0 {
		funds := make([]map[string]any, len(req.UpdateFunds))
		for i := range req.UpdateFunds {
			if req.UpdateFunds[i].FundID == "" {
				return nil, fmt.Errorf("%w: fund %d", errFundIDRequired, i)
			}
			if req.UpdateFunds[i].Amount <= 0 {
				return nil, fmt.Errorf("%w: fund %d", order.ErrAmountIsInvalid, i)
			}
			funds[i] = make(map[string]any)
			funds[i]["fundId"] = req.UpdateFunds[i].FundID
			funds[i]["amount"] = strconv.FormatFloat(req.UpdateFunds[i].Amount, 'f', -1, 64)
		}
		body["updateFunds"] = funds
	}
	var resp *PWMManageInvestmentPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/manage-investment-plan", nil, body, &resp, defaultEPL)
}

// GetAllPWMFundOrders returns the subscription and redemption orders of the institution's funds (Get All Fund Orders)
func (e *Exchange) GetAllPWMFundOrders(ctx context.Context, req *PWMFundOrdersRequest) (*PWMFundOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.FundID != "" {
		params.Set("fundId", req.FundID)
	}
	if req.OrderType != "" {
		params.Set("orderType", req.OrderType)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
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
	var resp *PWMFundOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/pwm/asset-manager/all-order", params, nil, &resp, defaultEPL)
}

// ManagePWMFundOrder approves or rejects a fund order pending review; repeating the request link ID returns the
// order's current status (Manage Order)
func (e *Exchange) ManagePWMFundOrder(ctx context.Context, req *PWMManageOrderRequest) (*PWMManageOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Action == "" {
		return nil, errActionRequired
	}
	if req.RequestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	body := make(map[string]any)
	body["orderId"] = req.OrderID
	body["action"] = req.Action
	body["reqLinkId"] = req.RequestLinkID
	var resp *PWMManageOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/manage-order", nil, body, &resp, defaultEPL)
}

// CreatePWMFundSubAccount creates a sub-account for an Active fund. Creation is asynchronous: repeat the request with
// the same request link ID to get the sub-account's UID once it is created (Create Fund Sub-Account)
func (e *Exchange) CreatePWMFundSubAccount(ctx context.Context, fundID, requestLinkID string) (*PWMCreateFundSubAccountResponse, error) {
	if fundID == "" {
		return nil, errFundIDRequired
	}
	if requestLinkID == "" {
		return nil, errRequestLinkIDRequired
	}
	body := make(map[string]any)
	body["fundId"] = fundID
	body["reqLinkId"] = requestLinkID
	var resp *PWMCreateFundSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/pwm/asset-manager/create-sub-account", nil, body, &resp, defaultEPL)
}

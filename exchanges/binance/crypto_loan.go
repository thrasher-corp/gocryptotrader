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

var errLTVAdjustmentDirectionRequired = errors.New("LTV adjustment direction is required")

// CheckCollateralRepayRate returns the latest collateral coin to loan coin rate used to repay a flexible loan with
// its collateral
func (e *Exchange) CheckCollateralRepayRate(ctx context.Context, loanCoin, collateralCoin currency.Code) (*FlexibleLoanCollateralRepayRateResponse, error) {
	if loanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	if collateralCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.Upper().String())
	params.Set("collateralCoin", collateralCoin.Upper().String())
	var resp *FlexibleLoanCollateralRepayRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/repay/rate", params, checkCollateralRepayRate, &resp)
}

// FlexibleLoanAdjustLTV adds collateral to or removes collateral from a flexible loan
func (e *Exchange) FlexibleLoanAdjustLTV(ctx context.Context, req *FlexibleLoanAdjustLTVRequest) (*FlexibleLoanAdjustLTVResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.CollateralCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.AdjustmentAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Direction == "" {
		return nil, errLTVAdjustmentDirectionRequired
	}
	params := url.Values{}
	params.Set("loanCoin", req.LoanCoin.Upper().String())
	params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	params.Set("adjustmentAmount", strconv.FormatFloat(req.AdjustmentAmount, 'f', -1, 64))
	params.Set("direction", req.Direction)
	var resp *FlexibleLoanAdjustLTVResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/loan/flexible/adjust/ltv", params, adjustFlexibleLoanRate, &resp)
}

// FlexibleLoanBorrow borrows a flexible rate loan, sized by either the loan amount or the collateral amount
func (e *Exchange) FlexibleLoanBorrow(ctx context.Context, req *FlexibleLoanBorrowRequest) (*FlexibleLoanBorrowResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.CollateralCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.LoanAmount <= 0 && req.CollateralAmount <= 0 {
		return nil, fmt.Errorf("%w: loan or collateral amount", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("loanCoin", req.LoanCoin.Upper().String())
	if req.LoanAmount > 0 {
		params.Set("loanAmount", strconv.FormatFloat(req.LoanAmount, 'f', -1, 64))
	}
	params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	if req.CollateralAmount > 0 {
		params.Set("collateralAmount", strconv.FormatFloat(req.CollateralAmount, 'f', -1, 64))
	}
	var resp *FlexibleLoanBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/loan/flexible/borrow", params, borrowFlexibleRate, &resp)
}

// FlexibleLoanRepay repays a flexible loan with the loan asset or its collateral
func (e *Exchange) FlexibleLoanRepay(ctx context.Context, req *FlexibleLoanRepayRequest) (*FlexibleLoanRepayResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.CollateralCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	// The repay amount is required even for a full repayment
	if req.RepayAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("loanCoin", req.LoanCoin.Upper().String())
	params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	params.Set("repayAmount", strconv.FormatFloat(req.RepayAmount, 'f', -1, 64))
	if req.CollateralReturn != nil {
		params.Set("collateralReturn", strconv.FormatBool(*req.CollateralReturn))
	}
	if req.FullRepayment {
		params.Set("fullRepayment", "true")
	}
	if req.RepaymentType > 0 {
		params.Set("repaymentType", strconv.FormatUint(req.RepaymentType, 10))
	}
	var resp *FlexibleLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/loan/flexible/repay", params, repayFlexibleLoanRate, &resp)
}

// FlexibleLoanAssetsData returns the interest rates and USD borrow limits of flexible loanable assets
func (e *Exchange) FlexibleLoanAssetsData(ctx context.Context, loanCoin currency.Code) (*FlexibleLoanAssetsDataResponse, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.Upper().String())
	}
	var resp *FlexibleLoanAssetsDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/loanable/data", params, flexibleLoanAssetDataRate, &resp)
}

// FlexibleLoanBorrowHistory returns flexible loan borrows
func (e *Exchange) FlexibleLoanBorrowHistory(ctx context.Context, req *FlexibleLoanHistoryRequest) (*FlexibleLoanBorrowHistoryResponse, error) {
	params, err := flexibleLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleLoanBorrowHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/borrow/history", params, flexibleBorrowHistoryRate, &resp)
}

// FlexibleCollateralAssetsData returns the LTV thresholds and USD collateral limits of flexible loan collateral assets
func (e *Exchange) FlexibleCollateralAssetsData(ctx context.Context, collateralCoin currency.Code) (*FlexibleCollateralAssetsDataResponse, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.Upper().String())
	}
	var resp *FlexibleCollateralAssetsDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/collateral/data", params, flexibleLoanCollateralAssetRate, &resp)
}

// GetFlexibleLoanLiquidationHistory returns flexible loan liquidations
func (e *Exchange) GetFlexibleLoanLiquidationHistory(ctx context.Context, req *FlexibleLoanHistoryRequest) (*FlexibleLoanLiquidationHistoryResponse, error) {
	params, err := flexibleLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleLoanLiquidationHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/liquidation/history", params, flexibleLoanLiquidationHistoryRate, &resp)
}

// FlexibleLoanLTVAdjustmentHistory returns flexible loan LTV adjustments
func (e *Exchange) FlexibleLoanLTVAdjustmentHistory(ctx context.Context, req *FlexibleLoanHistoryRequest) (*FlexibleLoanLTVAdjustmentHistoryResponse, error) {
	params, err := flexibleLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleLoanLTVAdjustmentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/ltv/adjustment/history", params, flexibleLoanLTVAdjustmentHistoryRate, &resp)
}

// FlexibleLoanOngoingOrders returns ongoing flexible loans
func (e *Exchange) FlexibleLoanOngoingOrders(ctx context.Context, req *FlexibleLoanOngoingOrdersRequest) (*FlexibleLoanOngoingOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.LoanCoin.IsEmpty() {
		params.Set("loanCoin", req.LoanCoin.Upper().String())
	}
	if !req.CollateralCoin.IsEmpty() {
		params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	}
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *FlexibleLoanOngoingOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/ongoing/orders", params, getFlexibleLoanOngoingOrdersRate, &resp)
}

// FlexibleLoanRepayHistory returns flexible loan repayments
func (e *Exchange) FlexibleLoanRepayHistory(ctx context.Context, req *FlexibleLoanHistoryRequest) (*FlexibleLoanRepayHistoryResponse, error) {
	params, err := flexibleLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleLoanRepayHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/loan/flexible/repay/history", params, flexibleLoanRepaymentHistoryRate, &resp)
}

// CryptoLoanIncomeHistory returns the asset movements of stable rate crypto loans
func (e *Exchange) CryptoLoanIncomeHistory(ctx context.Context, req *CryptoLoanIncomeHistoryRequest) ([]CryptoLoanIncome, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CryptoLoanIncome
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/income", params, cryptoLoansIncomeHistoryRate, &resp)
}

// CryptoLoanBorrowHistory returns stable rate crypto loan borrows
func (e *Exchange) CryptoLoanBorrowHistory(ctx context.Context, req *CryptoLoanHistoryRequest) (*CryptoLoanBorrowHistoryResponse, error) {
	params, err := cryptoLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CryptoLoanBorrowHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/borrow/history", params, getLoanBorrowHistoryRate, &resp)
}

// CryptoLoanLTVAdjustmentHistory returns stable rate crypto loan LTV adjustments
func (e *Exchange) CryptoLoanLTVAdjustmentHistory(ctx context.Context, req *CryptoLoanHistoryRequest) (*CryptoLoanLTVAdjustmentHistoryResponse, error) {
	params, err := cryptoLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CryptoLoanLTVAdjustmentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/ltv/adjustment/history", params, getLoanLTVAdjustmentHistoryRate, &resp)
}

// CryptoLoanRepaymentHistory returns stable rate crypto loan repayments
func (e *Exchange) CryptoLoanRepaymentHistory(ctx context.Context, req *CryptoLoanHistoryRequest) (*CryptoLoanRepaymentHistoryResponse, error) {
	params, err := cryptoLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CryptoLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/repay/history", params, repaymentHistoryRate, &resp)
}

// flexibleLoanHistoryParams builds the query shared by the flexible loan history endpoints
func flexibleLoanHistoryParams(req *FlexibleLoanHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if !req.LoanCoin.IsEmpty() {
		params.Set("loanCoin", req.LoanCoin.Upper().String())
	}
	if !req.CollateralCoin.IsEmpty() {
		params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	}
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

// cryptoLoanHistoryParams builds the query shared by the stable rate crypto loan history endpoints
func cryptoLoanHistoryParams(req *CryptoLoanHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.LoanCoin.IsEmpty() {
		params.Set("loanCoin", req.LoanCoin.Upper().String())
	}
	if !req.CollateralCoin.IsEmpty() {
		params.Set("collateralCoin", req.CollateralCoin.Upper().String())
	}
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

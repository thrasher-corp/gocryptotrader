package binance

import (
	"context"
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

// GetVIPBorrowInterestRate returns the flexible borrow interest rates of up to ten loan coins
func (e *Exchange) GetVIPBorrowInterestRate(ctx context.Context, loanCoins []currency.Code) ([]VIPLoanBorrowInterestRate, error) {
	if len(loanCoins) == 0 {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	coins := make([]string, len(loanCoins))
	for i := range loanCoins {
		if loanCoins[i].IsEmpty() {
			return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
		}
		coins[i] = loanCoins[i].Upper().String()
	}
	params := url.Values{}
	params.Set("loanCoin", strings.Join(coins, ","))
	var resp []VIPLoanBorrowInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/request/interestRate", params, getVIPBorrowInterestRate, &resp)
}

// GetVIPCollateralAssetData returns the collateral ratio tiers of VIP loan collateral assets
func (e *Exchange) GetVIPCollateralAssetData(ctx context.Context, collateralCoin currency.Code) (*VIPCollateralAssetDataResponse, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.Upper().String())
	}
	var resp *VIPCollateralAssetDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/collateral/data", params, getCollateralAssetDataRate, &resp)
}

// GetVIPLoanableAssetsData returns the interest rates and USD borrow limits of VIP loanable assets; vipLevel defaults
// to the account's VIP level when zero
func (e *Exchange) GetVIPLoanableAssetsData(ctx context.Context, loanCoin currency.Code, vipLevel uint64) (*VIPLoanableAssetsDataResponse, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.Upper().String())
	}
	if vipLevel > 0 {
		params.Set("vipLevel", strconv.FormatUint(vipLevel, 10))
	}
	var resp *VIPLoanableAssetsDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/loanable/data", params, getVIPLoanableAssetsRate, &resp)
}

// GetVIPLoanInterestRateHistory returns the flexible interest rate history of a VIP loan coin
func (e *Exchange) GetVIPLoanInterestRateHistory(ctx context.Context, req *VIPLoanInterestRateHistoryRequest) (*VIPLoanInterestRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, fmt.Errorf("%w: coin", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params.Set("coin", req.Coin.Upper().String())
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *VIPLoanInterestRateHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/interestRateHistory", params, vipLoanInterestRateHistoryRate, &resp)
}

// VIPLoanBorrow borrows a VIP loan against the spot balances of one or more collateral accounts
func (e *Exchange) VIPLoanBorrow(ctx context.Context, req *VIPLoanBorrowRequest) (*VIPLoanBorrowResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanAccountID == 0 {
		return nil, fmt.Errorf("%w: loan account ID", errAccountIDRequired)
	}
	if req.LoanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loan coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.LoanAmount <= 0 {
		return nil, fmt.Errorf("%w: loan amount", limits.ErrAmountBelowMin)
	}
	if len(req.CollateralAccountIDs) == 0 {
		return nil, fmt.Errorf("%w: collateral account ID", errAccountIDRequired)
	}
	if len(req.CollateralCoins) == 0 {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	// A fixed rate loan runs for a chosen term, so the term is only optional at a flexible rate
	if !req.IsFlexibleRate && req.LoanTerm == 0 {
		return nil, errLoanTermMustBeSet
	}
	accountIDs := make([]string, len(req.CollateralAccountIDs))
	for i := range req.CollateralAccountIDs {
		if req.CollateralAccountIDs[i] == 0 {
			return nil, fmt.Errorf("%w: collateral account ID", errAccountIDRequired)
		}
		accountIDs[i] = strconv.FormatUint(req.CollateralAccountIDs[i], 10)
	}
	collateralCoins := make([]string, len(req.CollateralCoins))
	for i := range req.CollateralCoins {
		if req.CollateralCoins[i].IsEmpty() {
			return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
		}
		collateralCoins[i] = req.CollateralCoins[i].Upper().String()
	}
	params := url.Values{}
	params.Set("loanAccountId", strconv.FormatUint(req.LoanAccountID, 10))
	params.Set("loanCoin", req.LoanCoin.Upper().String())
	params.Set("loanAmount", strconv.FormatFloat(req.LoanAmount, 'f', -1, 64))
	params.Set("collateralAccountId", strings.Join(accountIDs, ","))
	params.Set("collateralCoin", strings.Join(collateralCoins, ","))
	params.Set("isFlexibleRate", strconv.FormatBool(req.IsFlexibleRate))
	if req.LoanTerm > 0 {
		params.Set("loanTerm", strconv.FormatUint(req.LoanTerm, 10))
	}
	var resp *VIPLoanBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/vip/borrow", params, vipLoanBorrowRate, &resp)
}

// VIPLoanRenew renews a fixed rate VIP loan for another term of loanTerm days
func (e *Exchange) VIPLoanRenew(ctx context.Context, orderID, loanTerm uint64) (*VIPLoanRenewResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if loanTerm == 0 {
		return nil, errLoanTermMustBeSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatUint(orderID, 10))
	params.Set("loanTerm", strconv.FormatUint(loanTerm, 10))
	var resp *VIPLoanRenewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/vip/renew", params, vipLoanRenewRate, &resp)
}

// VIPLoanRepay repays a VIP loan
func (e *Exchange) VIPLoanRepay(ctx context.Context, orderID uint64, amount float64) (*VIPLoanRepayResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatUint(orderID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *VIPLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/vip/repay", params, vipLoanRepayRate, &resp)
}

// CheckVIPLoanCollateralAccount returns the collateral accounts bound to a VIP loan account, or the collateral
// assets of a collateral account; zero IDs are not sent
func (e *Exchange) CheckVIPLoanCollateralAccount(ctx context.Context, orderID, collateralAccountID uint64) (*VIPLoanCollateralAccountResponse, error) {
	params := url.Values{}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if collateralAccountID > 0 {
		params.Set("collateralAccountId", strconv.FormatUint(collateralAccountID, 10))
	}
	var resp *VIPLoanCollateralAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/collateral/account", params, checkLockedValueVIPCollateralAccountRate, &resp)
}

// GetVIPLoanAccruedInterest returns VIP loan interest records
func (e *Exchange) GetVIPLoanAccruedInterest(ctx context.Context, req *VIPLoanHistoryRequest) (*VIPLoanAccruedInterestResponse, error) {
	params, err := vipLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *VIPLoanAccruedInterestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/accruedInterest", params, getVIPLoanAccruedInterestRate, &resp)
}

// GetVIPLoanOngoingOrders returns ongoing VIP loan orders
func (e *Exchange) GetVIPLoanOngoingOrders(ctx context.Context, req *VIPLoanOngoingOrdersRequest) (*VIPLoanOngoingOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.CollateralAccountID > 0 {
		params.Set("collateralAccountId", strconv.FormatUint(req.CollateralAccountID, 10))
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
	var resp *VIPLoanOngoingOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/ongoing/orders", params, getVIPLoanOngoingOrdersRate, &resp)
}

// GetVIPLoanRepaymentHistory returns VIP loan repayments
func (e *Exchange) GetVIPLoanRepaymentHistory(ctx context.Context, req *VIPLoanHistoryRequest) (*VIPLoanRepaymentHistoryResponse, error) {
	params, err := vipLoanHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *VIPLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/repay/history", params, getVIPLoanRepaymentHistoryRate, &resp)
}

// GetVIPApplicationStatus returns the status of VIP loan applications
func (e *Exchange) GetVIPApplicationStatus(ctx context.Context, current, limit uint64) (*VIPLoanApplicationStatusResponse, error) {
	params := url.Values{}
	if current > 0 {
		params.Set("current", strconv.FormatUint(current, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *VIPLoanApplicationStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/vip/request/data", params, getApplicationStatusRate, &resp)
}

// vipLoanHistoryParams builds the query shared by the VIP loan accrued interest and repayment history endpoints
func vipLoanHistoryParams(req *VIPLoanHistoryRequest) (url.Values, error) {
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
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

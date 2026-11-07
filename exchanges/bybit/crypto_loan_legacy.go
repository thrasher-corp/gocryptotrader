package bybit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetLegacyCryptoLoanCollateralCoins returns the legacy crypto loan collateral coins of each VIP level, optionally
// filtered by VIP level and coin (Get Collateral Coins)
func (e *Exchange) GetLegacyCryptoLoanCollateralCoins(ctx context.Context, vipLevel string, coin currency.Code) (*LegacyCryptoLoanCollateralCoinsResponse, error) {
	params := url.Values{}
	if vipLevel != "" {
		params.Set("vipLevel", vipLevel)
	}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *LegacyCryptoLoanCollateralCoinsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/crypto-loan/collateral-data", params), defaultEPL, &resp)
}

// GetLegacyCryptoLoanBorrowableCoins returns the legacy crypto loan borrowable coins of each VIP level, optionally
// filtered by VIP level and coin (Get Borrowable Coins)
func (e *Exchange) GetLegacyCryptoLoanBorrowableCoins(ctx context.Context, vipLevel string, coin currency.Code) (*LegacyCryptoLoanBorrowableCoinsResponse, error) {
	params := url.Values{}
	if vipLevel != "" {
		params.Set("vipLevel", vipLevel)
	}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *LegacyCryptoLoanBorrowableCoinsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/crypto-loan/loanable-data", params), defaultEPL, &resp)
}

// GetLegacyCryptoLoanBorrowableCollateralisableLimit returns the minimum and maximum amounts the account can borrow
// and put up as collateral for a legacy crypto loan (Get Account Borrowable/Collateralizable Limit)
func (e *Exchange) GetLegacyCryptoLoanBorrowableCollateralisableLimit(ctx context.Context, loanCurrency, collateralCurrency currency.Code) (*LegacyCryptoLoanBorrowableCollateralisableLimitResponse, error) {
	if loanCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: loan currency", currency.ErrCurrencyCodeEmpty)
	}
	if collateralCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral currency", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("loanCurrency", loanCurrency.Upper().String())
	params.Set("collateralCurrency", collateralCurrency.Upper().String())
	var resp *LegacyCryptoLoanBorrowableCollateralisableLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/borrowable-collateralisable-number", params, nil, &resp, defaultEPL)
}

// RepayLegacyCryptoLoan repays a legacy crypto loan in full or in part from the funding wallet, paying due interest
// first (Repay)
func (e *Exchange) RepayLegacyCryptoLoan(ctx context.Context, orderID string, amount float64) (*CryptoLoanRepayResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["orderId"] = orderID
	body["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	var resp *CryptoLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan/repay", nil, body, &resp, defaultEPL)
}

// GetLegacyCryptoLoanUnpaidLoans returns the account's ongoing legacy crypto loans (Get Unpaid Loans)
func (e *Exchange) GetLegacyCryptoLoanUnpaidLoans(ctx context.Context, req *LegacyCryptoLoanUnpaidLoansRequest) (*LegacyCryptoLoanUnpaidLoansResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanTermType == "1" && req.LoanTerm == 0 {
		return nil, fmt.Errorf("%w for fixed term loans", errLoanTermRequired)
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if !req.CollateralCurrency.IsEmpty() {
		params.Set("collateralCurrency", req.CollateralCurrency.Upper().String())
	}
	if req.LoanTermType != "" {
		params.Set("loanTermType", req.LoanTermType)
	}
	if req.LoanTerm > 0 {
		params.Set("loanTerm", strconv.FormatUint(req.LoanTerm, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LegacyCryptoLoanUnpaidLoansResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/ongoing-orders", params, nil, &resp, defaultEPL)
}

// GetLegacyCryptoLoanRepaymentHistory returns up to six months of successful legacy crypto loan repayments (Get Loan
// Repayment History)
func (e *Exchange) GetLegacyCryptoLoanRepaymentHistory(ctx context.Context, req *LegacyCryptoLoanRepaymentHistoryRequest) (*LegacyCryptoLoanRepaymentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.RepayID != "" {
		params.Set("repayId", req.RepayID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LegacyCryptoLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/repayment-history", params, nil, &resp, defaultEPL)
}

// GetLegacyCryptoLoanCompletedLoanHistory returns up to six months of fully repaid legacy crypto loans (Get Completed
// Loan History)
func (e *Exchange) GetLegacyCryptoLoanCompletedLoanHistory(ctx context.Context, req *LegacyCryptoLoanCompletedLoanHistoryRequest) (*LegacyCryptoLoanCompletedLoanHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if !req.CollateralCurrency.IsEmpty() {
		params.Set("collateralCurrency", req.CollateralCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LegacyCryptoLoanCompletedLoanHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/borrow-history", params, nil, &resp, defaultEPL)
}

// GetLegacyCryptoLoanMaxCollateralReductionAmount returns the maximum amount by which a legacy crypto loan's
// collateral can be reduced (Get Max. Allowed Collateral Reduction Amount)
func (e *Exchange) GetLegacyCryptoLoanMaxCollateralReductionAmount(ctx context.Context, orderID string) (*CryptoLoanMaxCollateralReductionAmountResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *CryptoLoanMaxCollateralReductionAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/max-collateral-amount", params, nil, &resp, defaultEPL)
}

// AdjustLegacyCryptoLoanCollateralAmount adds collateral to or removes collateral from a legacy crypto loan through
// the funding wallet (Adjust Collateral Amount)
func (e *Exchange) AdjustLegacyCryptoLoanCollateralAmount(ctx context.Context, req *AdjustLegacyCryptoLoanCollateralAmountRequest) (*AdjustLegacyCryptoLoanCollateralAmountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Direction == "" {
		return nil, errCollateralAdjustmentDirectionRequired
	}
	var resp *AdjustLegacyCryptoLoanCollateralAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan/adjust-ltv", nil, map[string]any{
		"orderId":   req.OrderID,
		"amount":    strconv.FormatFloat(req.Amount, 'f', -1, 64),
		"direction": req.Direction,
	}, &resp, defaultEPL)
}

// GetLegacyCryptoLoanLTVAdjustmentHistory returns up to six months of the account's legacy crypto loan collateral
// adjustments (Get Loan LTV Adjustment History)
func (e *Exchange) GetLegacyCryptoLoanLTVAdjustmentHistory(ctx context.Context, req *LegacyCryptoLoanLTVAdjustmentHistoryRequest) (*LegacyCryptoLoanLTVAdjustmentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.AdjustID != "" {
		params.Set("adjustId", req.AdjustID)
	}
	if !req.CollateralCurrency.IsEmpty() {
		params.Set("collateralCurrency", req.CollateralCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LegacyCryptoLoanLTVAdjustmentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan/adjustment-history", params, nil, &resp, defaultEPL)
}

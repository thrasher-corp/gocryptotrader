package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Gate no longer documents the cross margin endpoints below, having folded cross margin into the unified account. They
// still answer, and the wrapper relies on them for the CrossMargin asset.

// CurrencySupportedByCrossMargin currencies supported by cross margin.
func (e *Exchange) CurrencySupportedByCrossMargin(ctx context.Context) ([]*CrossMarginCurrenciesResponse, error) {
	var response []*CrossMarginCurrenciesResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossListEPL, "margin/cross/currencies", &response)
}

// GetCrossMarginSupportedCurrencyDetail retrieve detail of one single currency supported by cross margin
func (e *Exchange) GetCrossMarginSupportedCurrencyDetail(ctx context.Context, ccy currency.Code) (*CrossMarginCurrenciesResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var response *CrossMarginCurrenciesResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossEPL, "margin/cross/currencies/"+ccy.String(), &response)
}

// GetCrossMarginAccounts retrieve cross margin account
func (e *Exchange) GetCrossMarginAccounts(ctx context.Context) (*CrossMarginAccountResponse, error) {
	var response *CrossMarginAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountsEPL, http.MethodGet, "margin/cross/accounts", nil, nil, &response)
}

// GetCrossMarginAccountChangeHistory retrieve cross margin account change history
// Record time range cannot exceed 30 days
// possible values of account change types are "in", "out", "repay", "new_order", "order_fill", "referral_fee", "order_fee", and "unknown"
func (e *Exchange) GetCrossMarginAccountChangeHistory(ctx context.Context, ccy currency.Code, from, to time.Time, page, limit uint64, accountChangeType string) ([]*CrossMarginAccountHistoryItemResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if accountChangeType != "" {
		params.Set("type", accountChangeType)
	}
	var response []*CrossMarginAccountHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountHistoryEPL, http.MethodGet, "margin/cross/account_book", params, nil, &response)
}

// CreateCrossMarginBorrowLoan create a cross margin borrow loan
// Borrow amount cannot be less than currency minimum borrow amount
func (e *Exchange) CreateCrossMarginBorrowLoan(ctx context.Context, arg *CrossMarginBorrowLoanRequest) (*CrossMarginLoanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, borrow amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var resp *CrossMarginLoanResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateCrossBorrowLoanEPL, http.MethodPost, "margin/cross/loans", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ExecuteRepayment when the liquidity of the currency is insufficient and the transaction risk is high, the currency will be disabled,
// and funds cannot be transferred.When the available balance of cross-margin is insufficient, the balance of the spot account can be used for repayment.
// Please ensure that the balance of the spot account is sufficient, and system uses cross-margin account for repayment first
func (e *Exchange) ExecuteRepayment(ctx context.Context, arg CurrencyAndAmountResponse) ([]*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, repay amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginExecuteRepaymentsEPL, http.MethodPost, "margin/cross/repayments", nil, &arg, &response)
}

// GetCrossMarginRepayments retrieves list of cross margin repayments
func (e *Exchange) GetCrossMarginRepayments(ctx context.Context, ccy currency.Code, loanID string, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if loanID != "" {
		params.Set("loan_id", loanID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", "true")
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossMarginRepaymentsEPL, http.MethodGet, "margin/cross/repayments", params, nil, &response)
}

// GetMaxTransferableAmountForSpecificCrossMarginCurrency get the max transferable amount for a specific cross margin currency
func (e *Exchange) GetMaxTransferableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	var response *CurrencyAndAmountResponse
	params.Set("currency", ccy.String())
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferCrossEPL, http.MethodGet, "margin/cross/transferable", params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificCrossMarginCurrency returns the max borrowable amount for a specific cross margin currency
func (e *Exchange) GetMaxBorrowableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyAndAmountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowCrossEPL, http.MethodGet, "margin/cross/borrowable", params, nil, &response)
}

// GetCrossMarginBorrowHistory retrieves cross margin borrow history sorted by creation time in descending order by default.
// Set reverse=false to return ascending results.
func (e *Exchange) GetCrossMarginBorrowHistory(ctx context.Context, status uint64, ccy currency.Code, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	if status < 1 || status > 3 {
		return nil, fmt.Errorf("%s %w, only allowed status values are 1:failed, 2:borrowed, and 3:repayment", e.Name, errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(status, 10))
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossBorrowHistoryEPL, http.MethodGet, "margin/cross/loans", params, nil, &response)
}

// GetSingleBorrowLoanDetail retrieve single borrow loan detail
func (e *Exchange) GetSingleBorrowLoanDetail(ctx context.Context, loanID string) (*CrossMarginLoanResponse, error) {
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	var response *CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetBorrowEPL, http.MethodGet, "margin/cross/loans/"+loanID, nil, nil, &response)
}

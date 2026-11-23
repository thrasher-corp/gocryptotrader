package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// maxFlexibleLoanBorrowCollateralEntries is the most collateral entries one flexible loan borrow accepts
const maxFlexibleLoanBorrowCollateralEntries = 10

var errFlexibleLoanCollateralEntriesExceedMaximum = errors.New("flexible loan collateral entries exceed the maximum")

// GetFlexibleLoanBorrowableCurrencies returns the currencies a flexible loan can borrow
func (e *Exchange) GetFlexibleLoanBorrowableCurrencies(ctx context.Context) ([]FlexibleLoanBorrowableCurrencyResponse, error) {
	var resp []FlexibleLoanBorrowableCurrencyResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanBorrowableCurrenciesEPL, http.MethodGet, "finance/flexible-loan/borrow-currencies", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanCollateralAssets returns the funding account collateral a flexible loan holds; OKX picks the earliest
// order when no order ID is set
func (e *Exchange) GetFlexibleLoanCollateralAssets(ctx context.Context, arg *FlexibleLoanCollateralAssetsRequest) ([]FlexibleLoanCollateralAssetsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	var resp []FlexibleLoanCollateralAssetsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanCollateralAssetsEPL, http.MethodGet, common.EncodeURLValues("finance/flexible-loan/collateral-assets", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanMaximumLoanAmount returns the most a flexible loan can borrow of a currency, optionally with
// supplementary collateral; OKX picks the earliest order when no order ID is set
func (e *Exchange) GetFlexibleLoanMaximumLoanAmount(ctx context.Context, arg *FlexibleLoanMaximumLoanRequest) ([]FlexibleLoanMaximumLoanResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.BorrowCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: borrowCcy", currency.ErrCurrencyCodeEmpty)
	}
	if err := validateFlexibleLoanCurrencyAmounts("supCollateral", arg.SupplementaryCollateral); err != nil {
		return nil, err
	}
	var resp []FlexibleLoanMaximumLoanResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanMaximumLoanAmountEPL, http.MethodPost, "finance/flexible-loan/max-loan", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanMaximumCollateralRedeemAmount returns the most collateral of a currency a flexible loan can release;
// OKX picks the earliest order when no order ID is set
func (e *Exchange) GetFlexibleLoanMaximumCollateralRedeemAmount(ctx context.Context, arg *FlexibleLoanMaximumCollateralRedeemRequest) ([]FlexibleLoanMaximumCollateralRedeemResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", arg.Currency.String())
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	var resp []FlexibleLoanMaximumCollateralRedeemResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanMaximumCollateralRedeemAmountEPL, http.MethodGet, common.EncodeURLValues("finance/flexible-loan/max-collateral-redeem-amount", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// AdjustFlexibleLoanCollateral adds or reduces a flexible loan's collateral; OKX picks the earliest order when no order
// ID is set
func (e *Exchange) AdjustFlexibleLoanCollateral(ctx context.Context, arg *FlexibleLoanAdjustCollateralRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Type == "" {
		return fmt.Errorf("%w: type", errMissingRequiredParameter)
	}
	if arg.CollateralCurrency.IsEmpty() {
		return fmt.Errorf("%w: collateralCcy", currency.ErrCurrencyCodeEmpty)
	}
	if arg.CollateralAmount <= 0 {
		return fmt.Errorf("%w: collateralAmt", limits.ErrAmountBelowMin)
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, adjustFlexibleLoanCollateralEPL, http.MethodPost, "finance/flexible-loan/adjust-collateral", arg, &resp, request.AuthenticatedRequest)
}

// BorrowFlexibleLoan creates a flexible loan, or borrows more on an existing one when an order ID is set
func (e *Exchange) BorrowFlexibleLoan(ctx context.Context, arg *FlexibleLoanBorrowRequest) (*FlexibleLoanBorrowResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.LoanData.Currency.IsEmpty() {
		return nil, fmt.Errorf("%w: loanData", currency.ErrCurrencyCodeEmpty)
	}
	if arg.LoanData.Amount <= 0 {
		return nil, fmt.Errorf("%w: loanData", limits.ErrAmountBelowMin)
	}
	if arg.OrderID == "" && len(arg.CollateralData) == 0 {
		return nil, fmt.Errorf("%w: collateralData when creating a loan", errMissingRequiredParameter)
	}
	if len(arg.CollateralData) > maxFlexibleLoanBorrowCollateralEntries {
		return nil, fmt.Errorf("%w: %d entries exceeds %d", errFlexibleLoanCollateralEntriesExceedMaximum, len(arg.CollateralData), maxFlexibleLoanBorrowCollateralEntries)
	}
	if err := validateFlexibleLoanCurrencyAmounts("collateralData", arg.CollateralData); err != nil {
		return nil, err
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	var resp *FlexibleLoanBorrowResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, borrowFlexibleLoanEPL, http.MethodPost, "finance/flexible-loan/borrow", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RepayFlexibleLoan repays a flexible loan in its borrowed currency; repaying the whole balance closes the loan
func (e *Exchange) RepayFlexibleLoan(ctx context.Context, arg *FlexibleLoanRepayRequest) (*FlexibleLoanRepayResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	var resp *FlexibleLoanRepayResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, repayFlexibleLoanEPL, http.MethodPost, "finance/flexible-loan/repay", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanEModeInfo returns the collateral and borrow currency pairs eligible for flexible loan E-Mode
func (e *Exchange) GetFlexibleLoanEModeInfo(ctx context.Context) ([]FlexibleLoanEModeInfoResponse, error) {
	var resp []FlexibleLoanEModeInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanEModeInfoEPL, http.MethodGet, "finance/flexible-loan/emode-info", nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanInfo returns a flexible loan's borrowing, collateral and risk, every loan when orderID is empty
func (e *Exchange) GetFlexibleLoanInfo(ctx context.Context, orderID string) ([]FlexibleLoanInfoResponse, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	var resp []FlexibleLoanInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanInfoEPL, http.MethodGet, common.EncodeURLValues("finance/flexible-loan/loan-info", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanHistory returns the account's flexible loan actions, every loan's when no order ID is set
func (e *Exchange) GetFlexibleLoanHistory(ctx context.Context, arg *FlexibleLoanHistoryRequest) ([]FlexibleLoanHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	var resp []FlexibleLoanHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/flexible-loan/loan-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFlexibleLoanAccruedInterest returns the flexible loan interest accrued over the past 30 days, every loan's when no
// order ID is set
func (e *Exchange) GetFlexibleLoanAccruedInterest(ctx context.Context, arg *FlexibleLoanAccruedInterestRequest) ([]FlexibleLoanAccruedInterestResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	var resp []FlexibleLoanAccruedInterestResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFlexibleLoanAccruedInterestEPL, http.MethodGet, common.EncodeURLValues("finance/flexible-loan/interest-accrued", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// validateFlexibleLoanCurrencyAmounts checks every entry of a flexible loan currency and amount list names a currency
// and a positive amount
func validateFlexibleLoanCurrencyAmounts(field string, entries []FlexibleLoanCurrencyAmount) error {
	for i := range entries {
		if entries[i].Currency.IsEmpty() {
			return fmt.Errorf("%w: %s[%d]", currency.ErrCurrencyCodeEmpty, field, i)
		}
		if entries[i].Amount <= 0 {
			return fmt.Errorf("%w: %s[%d]", limits.ErrAmountBelowMin, field, i)
		}
	}
	return nil
}

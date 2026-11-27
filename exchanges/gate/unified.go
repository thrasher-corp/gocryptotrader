package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetUnifiedAccount retrieves unified account.
func (e *Exchange) GetUnifiedAccount(ctx context.Context, ccy currency.Code, subAccountUserID string) (*UnifiedUserAccountResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var resp *UnifiedUserAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/accounts", params, nil, &resp)
}

// GetMaximumBorrowableAmountUnifiedAccount query maximum borrowable amount for unified account
func (e *Exchange) GetMaximumBorrowableAmountUnifiedAccount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/borrowable", params, nil, &resp)
}

// GetUnifiedAccountMaximumTransferableAmount query maximum transferable amount for unified account
func (e *Exchange) GetUnifiedAccountMaximumTransferableAmount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferable", params, nil, &resp)
}

// GetMultipleTransferableAmountForUnifiedAccounts batch queries the maximum transferable amount of up to 100 currencies
// in the unified account. A withdrawal changes the transferable amount of every currency.
func (e *Exchange) GetMultipleTransferableAmountForUnifiedAccounts(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmountResponse, error) {
	currencyList, err := currencyListParam(currencies, 100)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp []*CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferables", params, nil, &resp)
}

// GetBatchUnifiedAccountMaximumBorrowableAmount batch queries the maximum borrowable amount of up to 10 currencies in the
// unified account
func (e *Exchange) GetBatchUnifiedAccountMaximumBorrowableAmount(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmountResponse, error) {
	currencyList, err := currencyListParam(currencies, 10)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp []*CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/batch_borrowable", params, nil, &resp)
}

// BorrowOrRepay borrows or repays a currency in the unified account. A borrowed amount must not be below the currency's
// minimum borrowing threshold or exceed the borrowing limits set by the platform and user. Loan interest is deducted
// from the account automatically. Set RepaidAll, without an amount, to repay a loan in full.
func (e *Exchange) BorrowOrRepay(ctx context.Context, arg *BorrowOrRepayRequest) (*BorrowOrRepayResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	switch arg.Type {
	case "":
		return nil, errLoanTypeIsRequired
	case loanTypeBorrow, loanTypeRepay:
	default:
		return nil, fmt.Errorf("%w: %q must be \"borrow\" or \"repay\"", errInvalidLoanType, arg.Type)
	}
	if arg.RepaidAll {
		if arg.Type != loanTypeRepay {
			return nil, errInvalidRepaidAllOperation
		}
		if arg.Amount != 0 {
			return nil, errAmountOverriddenByRepaidAll
		}
	} else if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: borrow or repay amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var resp *BorrowOrRepayResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedBorrowOrRepayEPL, http.MethodPost, "unified/loans", nil, arg, &resp)
}

// GetLoans retrieves the unified account's outstanding loans. Loan type is "platform" or "margin"; both are returned
// when it is empty.
func (e *Exchange) GetLoans(ctx context.Context, arg *UnifiedLoansRequest) ([]*UnifiedLoanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type != "" && arg.Type != "platform" && arg.Type != "margin" {
		return nil, fmt.Errorf("%w: %q must be \"platform\" or \"margin\"", errInvalidLoanType, arg.Type)
	}
	var resp []*UnifiedLoanResponse
	return resp, e.getLoans(ctx, unifiedLoansEPL, "unified/loans", arg, &resp)
}

// GetLoanRecords retrieves the unified account's borrow and repayment records. Record type is "borrow" or "repay"; both
// are returned when it is empty.
func (e *Exchange) GetLoanRecords(ctx context.Context, arg *UnifiedLoansRequest) ([]*UnifiedLoanRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type != "" && arg.Type != loanTypeBorrow && arg.Type != loanTypeRepay {
		return nil, fmt.Errorf("%w: %q must be \"borrow\" or \"repay\"", errInvalidLoanType, arg.Type)
	}
	var resp []*UnifiedLoanRecordResponse
	return resp, e.getLoans(ctx, unifiedLoanRecordsEPL, "unified/loan_records", arg, &resp)
}

// getLoans sends a unified loan or loan record query, which share their filters and paging
func (e *Exchange) getLoans(ctx context.Context, epl request.EndpointLimit, path string, arg *UnifiedLoansRequest, result any) error {
	if arg.Limit > 100 {
		return fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, path, params, nil, result)
}

// GetInterestDeductionRecords retrieves the unified account's interest deduction records
func (e *Exchange) GetInterestDeductionRecords(ctx context.Context, arg *UnifiedInterestRecordsRequest) ([]*InterestDeductionRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	if arg.Type != "" && arg.Type != "platform" && arg.Type != "margin" {
		return nil, fmt.Errorf("%w: %q must be \"platform\" or \"margin\"", errInvalidLoanType, arg.Type)
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	var resp []*InterestDeductionRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedInterestRecordsEPL, http.MethodGet, "unified/interest_records", params, nil, &resp)
}

// SetUnifiedAccountMode sets the unified account mode, optionally switching the mode's settings in the same request.
// Modes are "classic", "multi_currency" (cross-currency margin), "portfolio" and "single_currency".
func (e *Exchange) SetUnifiedAccountMode(ctx context.Context, arg *UnifiedAccountModeRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Mode == "" {
		return errMissingUnifiedAccountMode
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedUpdateUnifiedModeEPL, http.MethodPut, "unified/unified_mode", nil, arg, nil)
}

// GetUnifiedAccountMode query mode of the unified account
func (e *Exchange) GetUnifiedAccountMode(ctx context.Context) (*UnifiedAccountModeResponse, error) {
	var resp *UnifiedAccountModeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetUnifiedModeEPL, http.MethodGet, "unified/unified_mode", nil, nil, &resp)
}

// GetUnifiedAccountEstimatedInterestRate retrieves the estimated hourly interest rate of up to 10 currencies in the
// unified account. Rates change hourly with lending depth, and an unsupported currency's rate is returned as zero.
func (e *Exchange) GetUnifiedAccountEstimatedInterestRate(ctx context.Context, currencies []currency.Code) (map[currency.Code]types.Number, error) {
	currencyList, err := currencyListParam(currencies, 10)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp map[currency.Code]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedEstimateRateEPL, http.MethodGet, "unified/estimate_rate", params, nil, &resp)
}

// GetUnifiedAccountTiered retrieves the unified account's collateral discount tiers
func (e *Exchange) GetUnifiedAccountTiered(ctx context.Context) ([]*UnifiedAccountTieredDetailResponse, error) {
	var resp []*UnifiedAccountTieredDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedCurrencyDiscountTiersEPL, "unified/currency_discount_tiers", &resp)
}

// GetUnifiedAccountTieredLoanMargin retrieves the unified account's borrowing margin tiers
func (e *Exchange) GetUnifiedAccountTieredLoanMargin(ctx context.Context) ([]*UnifiedAccountLoanMarginResponse, error) {
	var resp []*UnifiedAccountLoanMarginResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedLoanMarginTiersEPL, "unified/loan_margin_tiers", &resp)
}

// CalculatePortfolioMargin calculates the portfolio margin requirements of a simulated portfolio of positions and
// orders. Market orders are not supported.
func (e *Exchange) CalculatePortfolioMargin(ctx context.Context, arg *PortfolioMarginCalculatorRequest) (*PortfolioMarginCalculationResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	for _, sb := range arg.SpotBalances {
		if err := common.NilGuard(sb); err != nil {
			return nil, err
		}
		if sb.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if sb.Equity == 0 {
			return nil, errInvalidEquity
		}
	}
	for _, so := range arg.SpotOrders {
		if err := common.NilGuard(so); err != nil {
			return nil, err
		}
		if so.CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if so.OrderPrice <= 0 {
			return nil, fmt.Errorf("%w: order price must be greater than 0", limits.ErrPriceBelowMin)
		}
		if so.Left <= 0 {
			return nil, fmt.Errorf("%w: left, unfilled quantity size must be greater than 0", errInvalidOrderSize)
		}
		if so.Side != order.Buy && so.Side != order.Sell {
			return nil, fmt.Errorf("%w: %q must be buy or sell", order.ErrSideIsInvalid, so.Side)
		}
	}
	for _, fPosition := range arg.FuturesPositions {
		if err := common.NilGuard(fPosition); err != nil {
			return nil, err
		}
		if fPosition.Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if fPosition.Size == 0 {
			return nil, fmt.Errorf("%w: size must not be 0", errInvalidOrderSize)
		}
	}
	for _, fOrder := range arg.FuturesOrders {
		if err := common.NilGuard(fOrder); err != nil {
			return nil, err
		}
		if fOrder.Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if fOrder.Size == 0 || fOrder.Left == 0 {
			return nil, fmt.Errorf("%w: size and left, the unfilled quantity, must not be 0", errInvalidOrderSize)
		}
	}
	for _, op := range arg.OptionsPositions {
		if err := common.NilGuard(op); err != nil {
			return nil, err
		}
		if op.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if op.Size == 0 {
			return nil, fmt.Errorf("%w: size must not be 0", errInvalidOrderSize)
		}
	}
	for _, oo := range arg.OptionsOrders {
		if err := common.NilGuard(oo); err != nil {
			return nil, err
		}
		if oo.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if oo.Size == 0 || oo.Left == 0 {
			return nil, fmt.Errorf("%w: size and left, the unfilled quantity, must not be 0", errInvalidOrderSize)
		}
	}
	var resp *PortfolioMarginCalculationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedPortfolioCalculatorEPL, http.MethodPost, "unified/portfolio_calculator", nil, arg, &resp)
}

// SetAllLoanCurrenciesLeverage sets the leverage of every currency the unified account can borrow and returns the
// currencies that could not be updated. Currencies with outstanding loans keep their leverage, a leverage above a
// currency's limit is capped at that limit, and successful updates are not rolled back when others fail.
func (e *Exchange) SetAllLoanCurrenciesLeverage(ctx context.Context, leverage float64) ([]*LeverageFailedCurrencyResponse, error) {
	if leverage <= 0 {
		return nil, fmt.Errorf("%w: leverage must be greater than 0", errInvalidLeverage)
	}
	var resp []*LeverageFailedCurrencyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedLeverageUserSettingEPL, http.MethodPost, "unified/leverage/user_setting", nil, &UnifiedLeverageRequest{Leverage: types.Number(leverage)}, &resp)
}

// GetMaxMinCurrencyLeverage retrieves the maximum and minimum currency leverage that can be set
// for the unified account.
func (e *Exchange) GetMaxMinCurrencyLeverage(ctx context.Context, ccy currency.Code) (*UnifiedCurrencyLeverageConfigResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *UnifiedCurrencyLeverageConfigResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedLeverageUserCurrencyConfigEPL, http.MethodGet, "unified/leverage/user_currency_config", params, nil, &resp)
}

// GetUserCurrencyLeverage retrieves the leverage multiplier settings for borrowing currencies
// in the unified account. If currency is empty, all currencies are returned.
func (e *Exchange) GetUserCurrencyLeverage(ctx context.Context, ccy currency.Code) ([]*UnifiedLeverageSettingResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*UnifiedLeverageSettingResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetLeverageUserCurrencySettingEPL, http.MethodGet, "unified/leverage/user_currency_setting", params, nil, &resp)
}

// SetLoanCurrencyLeverage sets the loan currency leverage multiplier for the unified account.
func (e *Exchange) SetLoanCurrencyLeverage(ctx context.Context, arg *UnifiedLeverageSettingResponse) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Leverage <= 0 {
		return errInvalidLeverage
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedCreateLeverageUserCurrencySettingEPL, http.MethodPost, "unified/leverage/user_currency_setting", nil, arg, nil)
}

// GetUnifiedLoanCurrencies retrieves the list of loan currencies supported by the unified account.
// Optionally filter by currency. This endpoint does not require authentication.
func (e *Exchange) GetUnifiedLoanCurrencies(ctx context.Context, ccy currency.Code) ([]*UnifiedLoanCurrencyResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*UnifiedLoanCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedCurrenciesEPL, common.EncodeURLValues("unified/currencies", params), &resp)
}

// GetUnifiedHistoricalLendingRates retrieves a currency's hourly lending rates in the unified account, newest first.
// This endpoint does not require authentication.
func (e *Exchange) GetUnifiedHistoricalLendingRates(ctx context.Context, arg *UnifiedHistoricalLendingRatesRequest) (*UnifiedHistoricalLendingRateResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	params := url.Values{}
	params.Set("currency", arg.Currency.String())
	if arg.Tier != "" {
		params.Set("tier", arg.Tier)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *UnifiedHistoricalLendingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedHistoryLoanRateEPL, common.EncodeURLValues("unified/history_loan_rate", params), &resp)
}

// SetUnifiedCollateralCurrency sets collateral currencies for the unified account.
// collateralType: 0 = all currencies, 1 = custom currencies.
// When collateralType is 0, enableList and disableList are invalid.
func (e *Exchange) SetUnifiedCollateralCurrency(ctx context.Context, arg *UnifiedSetCollateralRequest) (*UnifiedSetCollateralResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CollateralType > 1 {
		return nil, fmt.Errorf("%w: %d must be 0 for all currencies or 1 for custom currencies", errInvalidCollateralType, arg.CollateralType)
	}
	var resp *UnifiedSetCollateralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedCollateralCurrenciesEPL, http.MethodPost, "unified/collateral_currencies", nil, arg, &resp)
}

// EstimateQuickRepaymentDetails retrieves the estimated quick repayment details for the unified account.
// Available for unified account cross-currency margin mode and portfolio margin mode.
func (e *Exchange) EstimateQuickRepaymentDetails(ctx context.Context) (*UnifiedQuickEstimateRepaymentResponse, error) {
	var resp *UnifiedQuickEstimateRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedEstimateQuickRepaymentEPL, http.MethodGet, "unified/estimated_quick_repayment", nil, nil, &resp)
}

// QuickRepayment performs a quick repayment for the unified account.
// Available for unified account cross-currency margin mode and portfolio margin mode.
func (e *Exchange) QuickRepayment(ctx context.Context, arg *UnifiedQuickRepaymentRequest) (*UnifiedQuickRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if len(arg.DebtCurrencies) == 0 {
		return nil, fmt.Errorf("%w: debt currencies are required for quick repayment", currency.ErrCurrencyCodesEmpty)
	}
	if len(arg.AvailableCurrencies) == 0 {
		return nil, fmt.Errorf("%w: available currencies are required for quick repayment", currency.ErrCurrencyCodesEmpty)
	}
	if slices.ContainsFunc(arg.DebtCurrencies, currency.Code.IsEmpty) || slices.ContainsFunc(arg.AvailableCurrencies, currency.Code.IsEmpty) {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *UnifiedQuickRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedQuickRepaymentEPL, http.MethodPost, "unified/quick_repayment", nil, arg, &resp)
}

// SetUnifiedDeltaNeutral enables or disables the account Delta-neutral strategy mode and returns the resulting state.
// Enabling it requires VIP level 4 or above and the cross-currency margin mode.
func (e *Exchange) SetUnifiedDeltaNeutral(ctx context.Context, enabled bool) (*DeltaNeutralResponse, error) {
	var resp *DeltaNeutralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedSetDeltaNeutralEPL, http.MethodPost, "unified/delta_neutral", nil, &DeltaNeutralRequest{Enabled: enabled}, &resp)
}

// GetUnifiedDeltaNeutral retrieves whether the account Delta-neutral strategy mode is enabled
func (e *Exchange) GetUnifiedDeltaNeutral(ctx context.Context) (*DeltaNeutralResponse, error) {
	var resp *DeltaNeutralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetDeltaNeutralEPL, http.MethodGet, "unified/delta_neutral", nil, nil, &resp)
}

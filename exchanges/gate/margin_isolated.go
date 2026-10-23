package gate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const frontEndURL = "https://www.gate.com/apiw/v2/"

// Loan operations of the unified and isolated margin borrow or repay endpoints
const (
	loanTypeBorrow = "borrow"
	loanTypeRepay  = "repay"
)

var (
	errInvalidLimit                     = errors.New("invalid limit")
	errInvalidIsolatedMarginLoanType    = errors.New("invalid isolated margin loan type: must be \"borrow\" or \"repay\"")
	errAmountOverriddenByRepaidAll      = errors.New("amount is overridden when repaid_all is true")
	errInvalidRepaidAllOperation        = errors.New("repaid_all is only valid for repay operations")
	errInvalidIsolatedMarginAccountType = errors.New("invalid isolated margin account type")
)

// TransferCollateralToIsolatedMargin transfers collateral from spot account to isolated margin account for a specific currency and pair.
func (e *Exchange) TransferCollateralToIsolatedMargin(ctx context.Context, pair currency.Pair, ccy currency.Code, amount float64) (*TransactionIDResponse, error) {
	return e.TransferCurrency(ctx, &TransferCurrencyRequest{CurrencyPair: pair, Currency: ccy, From: asset.Spot, To: asset.Margin, Amount: types.Number(amount)})
}

// TransferCollateralFromIsolatedMargin transfers collateral from an isolated margin account to spot account for a specific currency and pair.
// NOTE: Collateral can be orphaned when interest deduction has occurred but has not been repaid yet.
func (e *Exchange) TransferCollateralFromIsolatedMargin(ctx context.Context, pair currency.Pair, ccy currency.Code, amount float64) (*TransactionIDResponse, error) {
	return e.TransferCurrency(ctx, &TransferCurrencyRequest{CurrencyPair: pair, Currency: ccy, From: asset.Margin, To: asset.Spot, Amount: types.Number(amount)})
}

// GetIsolatedMarginAccounts retrieves the isolated margin account list, optionally filtered by currency pair.
// GetIsolatedMarginAccountList queries the same accounts through the endpoint that also covers margin ratio accounts.
func (e *Exchange) GetIsolatedMarginAccounts(ctx context.Context, currencyPair currency.Pair) ([]MarginAccountItemResponse, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	var response []MarginAccountItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginIsolatedAccountsEPL, http.MethodGet, "margin/accounts", params, nil, &response)
}

// GetIsolatedMarginAccountBalanceChangeHistory retrieves margin account balance change history
// Only transfers from and to margin account are provided for now. Time range allows 30 days at most
// NOTE: Live testing results that the API does not need currency pair to be set when currency is set, which differs from the documentation.
func (e *Exchange) GetIsolatedMarginAccountBalanceChangeHistory(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, from, to time.Time, page, limit uint64, accountType string) ([]IsolatedMarginAccountBalanceChangeInfoResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
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
	if accountType != "" {
		if accountType != "margin_in" && accountType != "margin_out" {
			return nil, fmt.Errorf("%w: must be \"margin_in\" or \"margin_out\"", errInvalidIsolatedMarginAccountType)
		}
		params.Set("type", accountType)
	}
	var response []IsolatedMarginAccountBalanceChangeInfoResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountBalanceEPL, http.MethodGet, "margin/account_book", params, nil, &response)
}

// GetIsolatedMarginFundingAccountList retrieves funding account list
func (e *Exchange) GetIsolatedMarginFundingAccountList(ctx context.Context, ccy currency.Code) ([]MarginFundingAccountItemResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []MarginFundingAccountItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginFundingAccountListEPL, http.MethodGet, "margin/funding_accounts", params, nil, &response)
}

// UpdateIsolatedMarginUsersAutoRepaymentSetting represents update user's auto repayment setting
func (e *Exchange) UpdateIsolatedMarginUsersAutoRepaymentSetting(ctx context.Context, statusOn bool) (*OnOffStatusResponse, error) {
	status := "on"
	if !statusOn {
		status = "off"
	}
	params := url.Values{}
	params.Set("status", status)
	var response *OnOffStatusResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAutoRepayEPL, http.MethodPost, "margin/auto_repay", params, nil, &response)
}

// GetIsolatedMarginUserAutoRepaymentSetting retrieve user auto repayment setting
func (e *Exchange) GetIsolatedMarginUserAutoRepaymentSetting(ctx context.Context) (*OnOffStatusResponse, error) {
	var response *OnOffStatusResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetAutoRepaySettingsEPL, http.MethodGet, "margin/auto_repay", nil, nil, &response)
}

// GetIsolatedMarginMaxTransferableAmount get the max transferable amount for a specific margin currency.
// If max transferable is required for USDT, a currency pair must be provided. The pair must be used as USDT is the
// quote currency for all isolated margin pairs.
func (e *Exchange) GetIsolatedMarginMaxTransferableAmount(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if !currencyPair.IsEmpty() && (!currencyPair.IsPopulated() || currencyPair.IsInvalid()) {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !currencyPair.IsPopulated() && ccy.Equal(currency.USDT) {
		return nil, fmt.Errorf("%w: required when currency is USDT", currency.ErrCurrencyPairEmpty)
	}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferEPL, http.MethodGet, "margin/transferable", params, nil, &response)
}

// GetIsolatedMarginLendingMarkets retrieves isolated margin lending markets
func (e *Exchange) GetIsolatedMarginLendingMarkets(ctx context.Context) ([]IsolatedMarginLendingMarketResponse, error) {
	var lendingMarkets []IsolatedMarginLendingMarketResponse
	return lendingMarkets, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUniCurrencyPairsMarginEPL, "margin/uni/currency_pairs", &lendingMarkets)
}

// GetIsolatedMarginLendingMarketDetails retrieves isolated margin lending market detail given the currency pair.
func (e *Exchange) GetIsolatedMarginLendingMarketDetails(ctx context.Context, pair currency.Pair) (*IsolatedMarginLendingMarketResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var market *IsolatedMarginLendingMarketResponse
	return market, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUniCurrencyPairDetailMarginEPL, "margin/uni/currency_pairs/"+pair.String(), &market)
}

// GetIsolatedMarginEstimatedInterestRate retrieves estimated interest rate for provided currencies
func (e *Exchange) GetIsolatedMarginEstimatedInterestRate(ctx context.Context, currencies []currency.Code) (map[string]types.Number, error) {
	currencyList, err := currencyListParam(currencies, 10)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var response map[string]types.Number
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniEstimateRateEPL, http.MethodGet, "margin/uni/estimate_rate", params, nil, &response)
}

// IsolatedMarginBorrowOrRepay borrows or repays currency in an isolated margin account. Pass type="borrow" to open a
// loan or type="repay" to close one.
// NOTE: 204 no content returned on success for borrow and repay.
func (e *Exchange) IsolatedMarginBorrowOrRepay(ctx context.Context, arg *IsolatedBorrowRepayRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.CurrencyPair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Type != loanTypeBorrow && arg.Type != loanTypeRepay {
		return errInvalidIsolatedMarginLoanType
	}
	if arg.RepaidAll {
		if arg.Type != loanTypeRepay {
			return errInvalidRepaidAllOperation
		}
		if arg.Amount != 0 {
			return errAmountOverriddenByRepaidAll
		}
	} else if arg.Amount <= 0 {
		return fmt.Errorf("%w: amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateUniLoanEPL, http.MethodPost, "margin/uni/loans", nil, arg, nil)
}

// GetIsolatedMarginLoans retrieves isolated margin loans (borrows) that are open for your account.
func (e *Exchange) GetIsolatedMarginLoans(ctx context.Context, ccy currency.Code, pair currency.Pair, page, limit uint64) ([]IsolatedMarginLoanResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if pair.IsPopulated() {
		params.Set("currency_pair", pair.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		if limit > 100 {
			return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []IsolatedMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniLoansEPL, http.MethodGet, "margin/uni/loans", params, nil, &response)
}

// GetIsolatedMarginLoanRecords retrieves isolated margin loan records. Loan type can be "borrow" or "repay". If not provided, both types will be returned.
func (e *Exchange) GetIsolatedMarginLoanRecords(ctx context.Context, ccy currency.Code, pair currency.Pair, page, limit uint64, loanType string) ([]IsolatedMarginLoanResponse, error) {
	params := url.Values{}
	if loanType != "" {
		if loanType != loanTypeBorrow && loanType != loanTypeRepay {
			return nil, errInvalidIsolatedMarginLoanType
		}
		params.Set("type", loanType)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if pair.IsPopulated() {
		params.Set("currency_pair", pair.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		if limit > 100 {
			return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []IsolatedMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniLoanRecordsEPL, http.MethodGet, "margin/uni/loan_records", params, nil, &response)
}

// GetIsolatedMarginInterestDeductionRecords retrieves interest deduction records for isolated margin loans.
func (e *Exchange) GetIsolatedMarginInterestDeductionRecords(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, page, limit uint64, from, to time.Time) ([]InterestDeductionRecordResponse, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		if limit > 1000 {
			return nil, fmt.Errorf("%w: maximum 1000", errInvalidLimit)
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var response []InterestDeductionRecordResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniInterestRecordsEPL, http.MethodGet, "margin/uni/interest_records", params, nil, &response)
}

// GetIsolatedMarginMaxBorrowableAmount retrieves the max borrowable amount for specific currency
func (e *Exchange) GetIsolatedMarginMaxBorrowableAmount(ctx context.Context, ccy currency.Code, pair currency.Pair) (*MaxBorrowableAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	params.Set("currency", ccy.String())
	var response *MaxBorrowableAmountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniBorrowableEPL, http.MethodGet, "margin/uni/borrowable", params, nil, &response)
}

// GetIsolatedMarginUserLeverageTiers retrieves the user's leverage tiers for isolated margin accounts. See: https://www.gate.com/en/help/trade/margin-trading/42357
func (e *Exchange) GetIsolatedMarginUserLeverageTiers(ctx context.Context, pair currency.Pair) ([]IsolatedMarginLendingTierResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	var response []IsolatedMarginLendingTierResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUserLoanMarginTiersEPL, http.MethodGet, "margin/user/loan_margin_tiers", params, nil, &response)
}

// GetIsolatedMarginMarketLeverageTiers retrieves the market leverage tiers for isolated margin accounts.
func (e *Exchange) GetIsolatedMarginMarketLeverageTiers(ctx context.Context, pair currency.Pair) ([]IsolatedMarginLendingTierResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	var response []IsolatedMarginLendingTierResponse

	path := common.EncodeURLValues("margin/loan_margin_tiers", params)
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, marginMarketLoanMarginTiersEPL, path, &response)
}

// SetUserMarketLeverageMultiplier sets the user's market leverage multiplier for isolated margin accounts.
// Success returns 204 No Content with empty body.
func (e *Exchange) SetUserMarketLeverageMultiplier(ctx context.Context, market currency.Pair, leverage float64) error {
	if market.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if leverage <= 0 {
		return fmt.Errorf("%w, leverage must be greater than 0", errInvalidLeverage)
	}
	payload := &MarginMarketLeverageRequest{CurrencyPair: market, Leverage: types.Number(leverage)}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSetUserMarketLeverageEPL, http.MethodPost, "margin/leverage/user_market_setting", nil, payload, nil)
}

// GetIsolatedMarginAccountList retrieves user's isolated margin account list.
// Supports querying both risk-based and margin-based isolated margin accounts.
// Gate reports outstanding principal in Borrowed, but applies isolated-margin interest deductions to Available
// instead of populating Interest. Available can therefore contain borrowed funds while positive or be negative after
// an interest deduction.
func (e *Exchange) GetIsolatedMarginAccountList(ctx context.Context, currencyPair currency.Pair) ([]MarginAccountItemResponse, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	var response []MarginAccountItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUserAccountListEPL, http.MethodGet, "margin/user/account", params, nil, &response)
}

// GetIsolatedMarginPoolLoans fetches margin loan pool information for all pairs. This endpoint provides actual pool
// availability data (the same data displayed on the Rate & Cap page) which is not exposed through any official API
// endpoint.
// NOTE: This occasionally returns 504 Gateway Timeout errors when the response is too large, so retry logic should be implemented in calling function.
func (e *Exchange) GetIsolatedMarginPoolLoans(ctx context.Context, coin currency.Code, page, limit uint64) (*IsolatedMarginPoolLoanResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("search_coin", coin.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}

	path := common.EncodeURLValues("spot_loan/margin/margin_loan_info", params)

	var resp *IsolatedMarginPoolLoanResponse
	if err := e.sendUnversionedHTTPRequest(ctx, exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, path, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("error code %d: %s", resp.Code, resp.Message)
	}

	return resp, nil
}

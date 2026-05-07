package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	frontEndURL             = "https://www.gate.com/apiw/v2/"
	_spot                   = "spot"
	_margin                 = "margin"
	marginPoolLoanPageLimit = 100
)

var errInvalidLimit = errors.New("invalid limit")

// TransferCollateralToIsolatedMargin transfers collateral from spot account to isolated margin account for a specific currency and pair.
func (e *Exchange) TransferCollateralToIsolatedMargin(ctx context.Context, pair currency.Pair, ccy currency.Code, amount float64) (*TransactionIDResponse, error) {
	return e.TransferCurrency(ctx, &TransferCurrencyParam{CurrencyPair: pair, Currency: ccy, From: _spot, To: _margin, Amount: types.Number(amount)})
}

// TransferCollateralFromIsolatedMargin transfers collateral from an isolated margin account to spot account for a specific currency and pair.
// NOTE: Collateral can be orphaned when interest deduction has occurred but has not been repaid yet.
func (e *Exchange) TransferCollateralFromIsolatedMargin(ctx context.Context, pair currency.Pair, ccy currency.Code, amount float64) (*TransactionIDResponse, error) {
	return e.TransferCurrency(ctx, &TransferCurrencyParam{CurrencyPair: pair, Currency: ccy, From: _margin, To: _spot, Amount: types.Number(amount)})
}

// GetIsolatedMarginAccountBalanceChangeHistory retrieves margin account balance change history
// Only transfers from and to margin account are provided for now. Time range allows 30 days at most
func (e *Exchange) GetIsolatedMarginAccountBalanceChangeHistory(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, from, to time.Time, page, limit uint64) ([]IsolatedMarginAccountBalanceChangeInfo, error) {
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
	var response []IsolatedMarginAccountBalanceChangeInfo
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountBalanceEPL, http.MethodGet, "margin/account_book", params, nil, &response)
}

// GetIsolatedMarginFundingAccountList retrieves funding account list
func (e *Exchange) GetIsolatedMarginFundingAccountList(ctx context.Context, ccy currency.Code) ([]MarginFundingAccountItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []MarginFundingAccountItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginFundingAccountListEPL, http.MethodGet, "margin/funding_accounts", params, nil, &response)
}

// GetIsolatedMarginUserAutoRepaymentSetting retrieve user auto repayment setting
func (e *Exchange) GetIsolatedMarginUserAutoRepaymentSetting(ctx context.Context) (*OnOffStatus, error) {
	var response *OnOffStatus
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetAutoRepaySettingsEPL, http.MethodGet, "margin/auto_repay", nil, nil, &response)
}

// UpdateIsolatedMarginUsersAutoRepaymentSetting represents update user's auto repayment setting
func (e *Exchange) UpdateIsolatedMarginUsersAutoRepaymentSetting(ctx context.Context, statusOn bool) (*OnOffStatus, error) {
	status := "on"
	if !statusOn {
		status = "off"
	}
	params := url.Values{}
	params.Set("status", status)
	var response *OnOffStatus
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAutoRepayEPL, http.MethodPost, "margin/auto_repay", params, nil, &response)
}

// GetIsolatedMarginMaxTransferableAmount get the max transferable amount for a specific margin currency.
func (e *Exchange) GetIsolatedMarginMaxTransferableAmount(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if currencyPair.IsEmpty() && ccy.Equal(currency.USDT) {
		return nil, fmt.Errorf("%w: required when currency is USDT", currency.ErrCurrencyPairEmpty)
	}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferEPL, http.MethodGet, "margin/transferable", params, nil, &response)
}

// GetIsolatedMarginLendingMarkets retrieves isolated margin lending markets
func (e *Exchange) GetIsolatedMarginLendingMarkets(ctx context.Context) ([]IsolatedMarginLendingMarket, error) {
	var lendingMarkets []IsolatedMarginLendingMarket
	return lendingMarkets, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUniCurrencyPairsMarginEPL, "margin/uni/currency_pairs", &lendingMarkets)
}

// GetIsolatedMarginLendingMarketDetails retrieves isolated margin lending market detail given the currency pair.
func (e *Exchange) GetIsolatedMarginLendingMarketDetails(ctx context.Context, pair currency.Pair) (*IsolatedMarginLendingMarket, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var market *IsolatedMarginLendingMarket
	return market, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUniCurrencyPairDetailMarginEPL, "margin/uni/currency_pairs/"+pair.String(), &market)
}

// GetIsolatedMarginEstimatedInterestRate retrieves estimated interest rate for provided currencies
func (e *Exchange) GetIsolatedMarginEstimatedInterestRate(ctx context.Context, currencies []currency.Code) (map[string]types.Number, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodesEmpty
	}
	if len(currencies) > 10 {
		return nil, fmt.Errorf("%w: maximum 10", errTooManyCurrencyCodes)
	}
	var out strings.Builder
	for i, c := range currencies {
		if c.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if i != 0 {
			out.WriteString(",")
		}
		out.WriteString(c.String())
	}
	params := url.Values{}
	params.Set("currencies", out.String())

	var response map[string]types.Number
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniEstimateRateEPL, http.MethodGet, "margin/uni/estimate_rate", params, nil, &response)
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
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []IsolatedMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniLoansEPL, http.MethodGet, "margin/uni/loans", params, nil, &response)
}

// IsolatedMarginBorrowOrRepay borrows or repays currency in an isolated margin account. Pass type="borrow" to open a
// loan or type="repay" to close one.
// NOTE: 204 no content returned on success for borrow and repay.
func (e *Exchange) IsolatedMarginBorrowOrRepay(ctx context.Context, arg *IsolatedBorrowRepayRequest) error {
	if arg == nil {
		return errNilArgument
	}
	if arg.CurrencyPair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Type != "borrow" && arg.Type != "repay" {
		return errors.New("invalid isolated margin loan type: must be \"borrow\" or \"repay\"")
	}
	if arg.Amount <= 0 {
		return fmt.Errorf("%w, amount must be greater than 0", errInvalidAmount)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateUniLoanEPL, http.MethodPost, "margin/uni/loans", nil, arg, nil)
}

// GetIsolatedMarginLoanRecords retrieves isolated margin loan records. Loan type can be "borrow" or "repay". If not provided, both types will be returned.
func (e *Exchange) GetIsolatedMarginLoanRecords(ctx context.Context, loanType string, ccy currency.Code, pair currency.Pair, page, limit uint64) ([]IsolatedMarginLoanResponse, error) {
	params := url.Values{}
	if loanType != "" {
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
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []IsolatedMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniLoanRecordsEPL, http.MethodGet, "margin/uni/loan_records", params, nil, &response)
}

// GetIsolatedMarginInterestDeductionRecords retrieves interest deduction records for isolated margin loans.
func (e *Exchange) GetIsolatedMarginInterestDeductionRecords(ctx context.Context, currencyPair currency.Pair, ccy currency.Code, page, limit uint64, from, to time.Time) ([]LoanInterestDeductionRecord, error) {
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
		if limit > 100 {
			return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var response []LoanInterestDeductionRecord
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniInterestRecordsEPL, http.MethodGet, "margin/uni/interest_records", params, nil, &response)
}

// GetIsolatedMarginMaxBorrowableAmount retrieves the max borrowable amount for specific currency
func (e *Exchange) GetIsolatedMarginMaxBorrowableAmount(ctx context.Context, ccy currency.Code, pair currency.Pair) (*MaxBorrowableAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	params.Set("currency", ccy.String())
	var response *MaxBorrowableAmount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUniBorrowableEPL, http.MethodGet, "margin/uni/borrowable", params, nil, &response)
}

// GetIsolatedMarginUserLeverageTiers retrieves the user's leverage tiers for isolated margin accounts. See: https://www.gate.com/en/help/trade/margin-trading/42357
func (e *Exchange) GetIsolatedMarginUserLeverageTiers(ctx context.Context, pair currency.Pair) ([]IsolatedMarginLendingTier, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	var response []IsolatedMarginLendingTier
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginUserLoanMarginTiersEPL, http.MethodGet, "margin/user/loan_margin_tiers", params, nil, &response)
}

// GetIsolatedMarginMarketLeverageTiers retrieves the market leverage tiers for isolated margin accounts.
func (e *Exchange) GetIsolatedMarginMarketLeverageTiers(ctx context.Context, pair currency.Pair) ([]IsolatedMarginLendingTier, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pair.String())
	var response []IsolatedMarginLendingTier
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginMarketLoanMarginTiersEPL, http.MethodGet, "margin/loan_margin_tiers", params, nil, &response)
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
	payload := struct {
		Leverage float64 `json:"leverage"`
		Pair     string  `json:"currency_pair"`
	}{
		Leverage: leverage,
		Pair:     market.String(),
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSetUserMarketLeverageEPL, http.MethodPost, "margin/leverage/user_market_setting", nil, payload, nil)
}

// GetIsolatedMarginAccountList retrieves user's isolated margin account list.
// Supports querying both risk-based and margin-based isolated margin accounts.
func (e *Exchange) GetIsolatedMarginAccountList(ctx context.Context, currencyPair currency.Pair) ([]MarginAccountItem, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	var response []MarginAccountItem
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
		if limit > marginPoolLoanPageLimit {
			return nil, fmt.Errorf("%w: maximum %d", errInvalidLimit, marginPoolLoanPageLimit)
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
	}

	path := common.EncodeURLValues("spot_loan/margin/margin_loan_info", params)

	var resp *IsolatedMarginPoolLoanResponse
	return resp, e.SendHTTPRequest(ctx, exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, path, &resp)
}

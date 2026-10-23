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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetLendingCurrencyList retrieves the currencies available for lending
func (e *Exchange) GetLendingCurrencyList(ctx context.Context) ([]*LendingCurrencyDetailResponse, error) {
	var resp []*LendingCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnUniCurrenciesEPL, "earn/uni/currencies", &resp)
}

// GetLendingCurrencyDetail retrieves a single lending currency detail
func (e *Exchange) GetLendingCurrencyDetail(ctx context.Context, ccy currency.Code) (*LendingCurrencyDetailResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *LendingCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnUniCurrencyEPL, "earn/uni/currencies/"+ccy.Lower().String(), &resp)
}

// CreateLendingOrRedemption lends to or redeems from the lending pool; arg.Type is "lend" or "redeem"
func (e *Exchange) CreateLendingOrRedemption(ctx context.Context, arg *LendingOrRedemptionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return fmt.Errorf("%w: minimum lending or redemption amount is required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errLoanTypeIsRequired
	}
	// Only lending earns interest, so redemptions carry no minimum rate
	if arg.Type == "lend" && arg.MinHourlyRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniCreateLendsEPL, http.MethodPost, "earn/uni/lends", nil, arg, nil)
}

// GetUserLendingOrderList retrieves the user's lending orders
func (e *Exchange) GetUserLendingOrderList(ctx context.Context, ccy currency.Code, page, limit uint64) ([]*LendOrderDetailResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*LendOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniGetLendsEPL, http.MethodGet, "earn/uni/lends", params, nil, &resp)
}

// AmendUserLendingInformation amends the hourly minimum interest rate of a currency's lending
func (e *Exchange) AmendUserLendingInformation(ctx context.Context, ccy currency.Code, minRate float64) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if minRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniUpdateLendsEPL, http.MethodPatch, "earn/uni/lends", nil, map[string]string{
		"currency": ccy.String(),
		"min_rate": strconv.FormatFloat(minRate, 'f', -1, 64),
	}, nil)
}

// GetLendingTransactionRecords retrieves lending and redemption records
func (e *Exchange) GetLendingTransactionRecords(ctx context.Context, arg *LendingTransactionRecordsRequest) ([]*LendingTransactionRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
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
	var resp []*LendingTransactionRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniLendRecordsEPL, http.MethodGet, "earn/uni/lend_records", params, nil, &resp)
}

// GetUserTotalInterestIncomePerCurrency retrieves the user's total interest income for a currency
func (e *Exchange) GetUserTotalInterestIncomePerCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndInterestIncomeResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyAndInterestIncomeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestsEPL, http.MethodGet, "earn/uni/interests/"+ccy.Lower().String(), nil, nil, &resp)
}

// GetUserDividendRecords retrieves the user's lending interest records
func (e *Exchange) GetUserDividendRecords(ctx context.Context, arg *UserDividendRecordsRequest) ([]*UserDividendRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
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
	var resp []*UserDividendRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestRecordsEPL, http.MethodGet, "earn/uni/interest_records", params, nil, &resp)
}

// GetCurrencyInterestCompoundingStatus retrieves whether a currency's lending interest is paid out or reinvested
func (e *Exchange) GetCurrencyInterestCompoundingStatus(ctx context.Context, ccy currency.Code) (*CurrencyInterestStatusResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInterestStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestStatusEPL, http.MethodGet, "earn/uni/interest_status/"+ccy.Lower().String(), nil, nil, &resp)
}

// GetUniLoanCurrencyAnnualisedTrendChart retrieves a currency's annualised lending rate trend over a range of at most 30 days
func (e *Exchange) GetUniLoanCurrencyAnnualisedTrendChart(ctx context.Context, from, to time.Time, ccy currency.Code) ([]*UniLoanAssetResponse, error) {
	if err := common.StartEndTimeCheck(from, to); err != nil {
		return nil, err
	}
	if to.Sub(from) > 30*24*time.Hour {
		return nil, errUniChartRangeExceeded
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("from", strconv.FormatInt(from.Unix(), 10))
	params.Set("to", strconv.FormatInt(to.Unix(), 10))
	params.Set("asset", ccy.String())
	var resp []*UniLoanAssetResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniChartEPL, http.MethodGet, "earn/uni/chart", params, nil, &resp)
}

// GetCurrencyEstimatedAnnualisedInterestRate retrieves each currency's estimated annualised lending rate
func (e *Exchange) GetCurrencyEstimatedAnnualisedInterestRate(ctx context.Context) ([]*CurrencyEstimatedAnnualInterestRateResponse, error) {
	var resp []*CurrencyEstimatedAnnualInterestRateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniRateEPL, http.MethodGet, "earn/uni/rate", nil, nil, &resp)
}

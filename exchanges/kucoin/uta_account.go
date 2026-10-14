package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetUTAAccountOverview returns the account-level summary of the unified trading account's funds (Get Account
// Overview (UTA))
func (e *Exchange) GetUTAAccountOverview(ctx context.Context) (*UTAAccountOverviewResponse, error) {
	var resp *UTAAccountOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAccountOverviewEPL, http.MethodGet, "/ua/v2/unified/account/overview", nil, &resp)
}

// GetUTAAccountCurrencyAssets returns the unified trading account's assets per currency (Get Account Currency Assets
// (UTA))
func (e *Exchange) GetUTAAccountCurrencyAssets(ctx context.Context) (*UTAAccountCurrencyAssetsResponse, error) {
	var resp *UTAAccountCurrencyAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAccountCurrencyAssetsEPL, http.MethodGet, "/ua/v2/unified/account/balance", nil, &resp)
}

// GetUTAClassicAccountCurrencyAssets returns a classic account type's assets per currency (Get Account Currency Assets
// (Classic))
func (e *Exchange) GetUTAClassicAccountCurrencyAssets(ctx context.Context, req *UTAClassicAccountCurrencyAssetsRequest) (*UTAClassicAccountCurrencyAssetsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	params := url.Values{}
	if req.AccountSubtype != "" {
		params.Set("accountSubtype", req.AccountSubtype)
	}
	params.Set("accountType", req.AccountType)
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	var resp *UTAClassicAccountCurrencyAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAClassicAccountCurrencyAssetsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/balance", params), nil, &resp)
}

// GetUTAAPIKeyInfo returns the details of the API key that signs the request, which may be a master or sub-account key
// (Get Apikey Info)
func (e *Exchange) GetUTAAPIKeyInfo(ctx context.Context) (*UTAAPIKeyInfoResponse, error) {
	var resp *UTAAPIKeyInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAPIKeyInfoEPL, http.MethodGet, "/ua/v2/user/api-key", nil, &resp)
}

// GetUTAAccountMode returns the account's mode and the UIDs of its unified and classic sub-accounts (Get Account Mode)
func (e *Exchange) GetUTAAccountMode(ctx context.Context) (*UTAAccountModeResponse, error) {
	var resp *UTAAccountModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAccountModeEPL, http.MethodGet, "/ua/v2/account/mode", nil, &resp)
}

// SetUTAAccountMode sets the account's mode; accountType is CLASSIC or UNIFIED (Set Account Mode)
func (e *Exchange) SetUTAAccountMode(ctx context.Context, accountType string) error {
	if accountType == "" {
		return errAccountTypeMissing
	}
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, setUTAAccountModeEPL, http.MethodPost, "/ua/v2/account/mode", map[string]any{"accountType": accountType}, &resp)
}

// GetUTALeverage returns the leverage of cross margin currencies or futures symbols (Get Leverage (UTA))
func (e *Exchange) GetUTALeverage(ctx context.Context, req *UTALeverageRequest) ([]UTALeverage, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	params.Set("tradeType", req.TradeType)
	if req.MarginMode != "" {
		params.Set("marginMode", req.MarginMode)
	}
	var resp []UTALeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTALeverageEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/account/leverage", params), nil, &resp)
}

// ModifyUTAFuturesLeverage sets a futures symbol's leverage (Modify Futures Leverage (UTA))
func (e *Exchange) ModifyUTAFuturesLeverage(ctx context.Context, symbol string, leverage float64) (*ModifyUTAFuturesLeverageResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if leverage <= 0 {
		return nil, errInvalidLeverage
	}
	body := make(map[string]any)
	body["symbol"] = symbol
	body["leverage"] = strconv.FormatFloat(leverage, 'f', -1, 64)
	var resp *ModifyUTAFuturesLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyUTAFuturesLeverageEPL, http.MethodPost, "/ua/v2/unified/account/modify-leverage", body, &resp)
}

// ModifyUTACrossMarginLeverage sets the cross margin leverage, of a currency when one is given (Modify Cross Margin
// Leverage (UTA))
func (e *Exchange) ModifyUTACrossMarginLeverage(ctx context.Context, ccy currency.Code, leverage float64) (*ModifyUTACrossMarginLeverageResponse, error) {
	if leverage <= 0 {
		return nil, errInvalidLeverage
	}
	body := map[string]any{"leverage": strconv.FormatFloat(leverage, 'f', -1, 64)}
	if !ccy.IsEmpty() {
		body["currency"] = ccy.Upper().String()
	}
	var resp *ModifyUTACrossMarginLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyUTACrossMarginLeverageEPL, http.MethodPost, "/ua/v2/unified/account/modify-leverage-margin-cross", body, &resp)
}

// GetUTABorrowingRatesAndLimits returns a currency's borrowing rates, borrowing limits and reserved borrowing; it is
// only supported in UTA mode (Get Borrowing Rates and Limits)
func (e *Exchange) GetUTABorrowingRatesAndLimits(ctx context.Context, ccy currency.Code) (*UTABorrowingRatesAndLimitsResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	var resp *UTABorrowingRatesAndLimitsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTABorrowingRatesAndLimitsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/interest-limits", params), nil, &resp)
}

// GetUTAAccountLedger returns a page of an account type's ledger entries (Get Account Ledger)
func (e *Exchange) GetUTAAccountLedger(ctx context.Context, req *UTAAccountLedgerRequest) (*UTAAccountLedgerResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	params := url.Values{}
	params.Set("accountType", req.AccountType)
	if req.BusinessType != "" {
		params.Set("businessType", req.BusinessType)
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.Direction != "" {
		params.Set("direction", req.Direction)
	}
	if err := setTimeRange(params, "startAt", "endAt", req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.LastID != "" {
		params.Set("lastId", req.LastID)
	}
	var resp *UTAAccountLedgerResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAccountLedgerEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/ledger", params), nil, &resp)
}

// GetUTAInterestHistory returns a page of interest accrual records (Get Interest History (UTA))
func (e *Exchange) GetUTAInterestHistory(ctx context.Context, req *UTAInterestHistoryRequest) (*UTAInterestHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	params := url.Values{}
	params.Set("accountType", req.AccountType)
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *UTAInterestHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAInterestHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/interest-history", params), nil, &resp)
}

// GetUTAAPIRateLimit returns the API rate limits of the given UIDs (Get API Rate Limit)
func (e *Exchange) GetUTAAPIRateLimit(ctx context.Context, uids ...string) (*UTAAPIRateLimitResponse, error) {
	if len(uids) == 0 {
		return nil, errUIDRequired
	}
	params := url.Values{}
	params.Set("uids", strings.Join(uids, ","))
	var resp *UTAAPIRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAPIRateLimitEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/rate-limit/query", params), nil, &resp)
}

// GetUTAAllAPIRateLimit returns the API rate limits of the master account and every sub-account (Get All API Rate
// Limit)
func (e *Exchange) GetUTAAllAPIRateLimit(ctx context.Context) (*UTAAPIRateLimitResponse, error) {
	var resp *UTAAPIRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAllAPIRateLimitEPL, http.MethodGet, "/ua/v2/rate-limit/query-all", nil, &resp)
}

// GetUTAAPIRateLimitCap returns the master account's API rate limit quota and its allocation to sub-accounts (Get API
// Rate Limit Cap)
func (e *Exchange) GetUTAAPIRateLimitCap(ctx context.Context) (*UTAAPIRateLimitCapResponse, error) {
	var resp *UTAAPIRateLimitCapResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAPIRateLimitCapEPL, http.MethodGet, "/ua/v2/rate-limit/query-cap", nil, &resp)
}

// GetUTAFeeRate returns the actual fee rates of symbols; tradeType is SPOT, for up to 10 symbols, or FUTURES, for one
// symbol. Sub-accounts have their master account's fee rates (Get Fee Rate)
func (e *Exchange) GetUTAFeeRate(ctx context.Context, tradeType string, symbols []string) (*UTAFeeRateResponse, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	if len(symbols) == 0 {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	params.Set("symbol", strings.Join(symbols, ","))
	var resp *UTAFeeRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAFeeRateEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/user/fee-rate", params), nil, &resp)
}

// SetUTAKCSFeeDeduction enables or disables paying trading fees with KCS (KCS Fee Deduction Management)
func (e *Exchange) SetUTAKCSFeeDeduction(ctx context.Context, enabled bool) (*SetUTAKCSFeeDeductionResponse, error) {
	params := url.Values{}
	params.Set("enabled", strconv.FormatBool(enabled))
	var resp *SetUTAKCSFeeDeductionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, setUTAKCSFeeDeductionEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/fee/kcs-deduct", params), nil, &resp)
}

// GetUTAThirdPartyCustodyAccountCurrencyLimits returns the remaining third-party custody quota of settlement
// currencies. Both filters are optional; a sub-account key's custodian is fixed, so custodian is ignored for it (Get
// Third-Party Custody Account Currency Limits)
func (e *Exchange) GetUTAThirdPartyCustodyAccountCurrencyLimits(ctx context.Context, custodian string, ccy currency.Code) ([]UTAThirdPartyCustodyCurrencyLimit, error) {
	params := url.Values{}
	if custodian != "" {
		params.Set("custodian", custodian)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []UTAThirdPartyCustodyCurrencyLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAThirdPartyCustodyAccountCurrencyLimitsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/oes/custody-quota", params), nil, &resp)
}

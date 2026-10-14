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

// GetAccountSummaryInformation returns the VIP level and the sub-account counts and limits of the account (Get Account
// Summary Info)
func (e *Exchange) GetAccountSummaryInformation(ctx context.Context) (*AccountSummaryInformationResponse, error) {
	var resp *AccountSummaryInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, &resp)
}

// GetAPIKeyInfo returns the details of the API key that signs the request, for master and sub-account keys alike (Get
// Apikey Info)
func (e *Exchange) GetAPIKeyInfo(ctx context.Context) (*APIKeyInfoResponse, error) {
	var resp *APIKeyInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAPIKeyInfoEPL, http.MethodGet, "/v1/user/api-key", nil, &resp)
}

// GetSpotAccountType reports whether the user is a high-frequency spot user, who transfers and queries assets with the
// trade_hf account type, rather than a low-frequency one, who uses trade (Get Account Type - Spot). Only users who
// opened high-frequency trading before 2024 need it
func (e *Exchange) GetSpotAccountType(ctx context.Context) (bool, error) {
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSpotAccountTypeEPL, http.MethodGet, "/v1/hf/accounts/opened", nil, &resp)
}

// GetAllAccounts returns the spot accounts, optionally filtered by currency and by account type, main or trade (Get
// Account List - Spot)
func (e *Exchange) GetAllAccounts(ctx context.Context, ccy currency.Code, accountType string) ([]AccountInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	if accountType != "" {
		params.Set("type", accountType)
	}
	var resp []AccountInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAllAccountsEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts", params), nil, &resp)
}

// GetAccountDetail returns the balance of a single spot account (Get Account Detail - Spot)
func (e *Exchange) GetAccountDetail(ctx context.Context, accountID string) (*AccountDetailResponse, error) {
	if accountID == "" {
		return nil, errAccountIDMissing
	}
	var resp *AccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAccountDetailEPL, http.MethodGet, "/v1/accounts/"+accountID, nil, &resp)
}

// GetCrossMarginAccountsDetail returns the cross margin account valued in quoteCurrency, USDT, KCS or BTC (Get Account
// - Cross Margin). queryType is MARGIN, MARGIN_V2 (being phased out, same meaning) or ALL
func (e *Exchange) GetCrossMarginAccountsDetail(ctx context.Context, quoteCurrency currency.Code, queryType string) (*CrossMarginAccountsDetailResponse, error) {
	params := url.Values{}
	if !quoteCurrency.IsEmpty() {
		params.Set("quoteCurrency", quoteCurrency.Upper().String())
	}
	if queryType != "" {
		params.Set("queryType", queryType)
	}
	var resp *CrossMarginAccountsDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getCrossMarginAccountsDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/accounts", params), nil, &resp)
}

// GetIsolatedMarginAccountDetail returns the isolated margin accounts, every symbol's unless one is given (Get Account
// - Isolated Margin)
func (e *Exchange) GetIsolatedMarginAccountDetail(ctx context.Context, req *IsolatedMarginAccountDetailRequest) (*IsolatedMarginAccountDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.QuoteCurrency.IsEmpty() {
		params.Set("quoteCurrency", req.QuoteCurrency.Upper().String())
	}
	if req.QueryType != "" {
		params.Set("queryType", req.QueryType)
	}
	var resp *IsolatedMarginAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getIsolatedMarginAccountDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/isolated/accounts", params), nil, &resp)
}

// GetFuturesAccountOverview returns the futures account of a settlement currency, XBT unless one is given (Get Account
// - Futures)
func (e *Exchange) GetFuturesAccountOverview(ctx context.Context, ccy currency.Code) (*FuturesAccountOverviewResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp *FuturesAccountOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesAccountOverviewEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview", params), nil, &resp)
}

// GetAccountLedgers returns a page of the ledger entries of every account type, newest first (Get Account Ledgers -
// Spot/Margin). The time window is at most 24 hours and the history one year; given only one bound, KuCoin derives the
// other 24 hours away
func (e *Exchange) GetAccountLedgers(ctx context.Context, req *AccountLedgersRequest) (*AccountLedgersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if len(req.Currencies) != 0 {
		currencies, err := joinLedgerCurrencies(req.Currencies)
		if err != nil {
			return nil, err
		}
		params.Set("currency", currencies)
	}
	if req.Direction != "" {
		params.Set("direction", req.Direction)
	}
	if req.BusinessType != "" {
		params.Set("bizType", req.BusinessType)
	}
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.UnixMilli(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.UnixMilli(), 10))
	}
	if req.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *AccountLedgersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAccountLedgersEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgersHFTrade returns the ledger entries of the high-frequency trading account, newest first, from the
// last seven days (Get Account Ledgers - Trade_hf). Entries before LastID are returned when it is set
func (e *Exchange) GetAccountLedgersHFTrade(ctx context.Context, req *HFAccountLedgersRequest) ([]HFAccountLedger, error) {
	params, err := hfAccountLedgersParams(req)
	if err != nil {
		return nil, err
	}
	var resp []HFAccountLedger
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAccountLedgersHFTradeEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgerHFMargin returns the ledger entries of the high-frequency cross and isolated margin accounts, newest
// first, from the last three days (Get Account Ledgers - Margin_hf). Entries before LastID are returned when it is set
func (e *Exchange) GetAccountLedgerHFMargin(ctx context.Context, req *HFAccountLedgersRequest) ([]HFMarginAccountLedger, error) {
	params, err := hfAccountLedgersParams(req)
	if err != nil {
		return nil, err
	}
	var resp []HFMarginAccountLedger
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAccountLedgerHFMarginEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/account/ledgers", params), nil, &resp)
}

// hfAccountLedgersParams builds the query of the two high-frequency ledger endpoints, which take the same parameters
func hfAccountLedgersParams(req *HFAccountLedgersRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if len(req.Currencies) != 0 {
		currencies, err := joinLedgerCurrencies(req.Currencies)
		if err != nil {
			return nil, err
		}
		params.Set("currency", currencies)
	}
	if req.Direction != "" {
		params.Set("direction", req.Direction)
	}
	if req.BusinessType != "" {
		params.Set("bizType", req.BusinessType)
	}
	if req.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(req.LastID, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.UnixMilli(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.UnixMilli(), 10))
	}
	return params, nil
}

// joinLedgerCurrencies joins currencies into the comma-separated list the ledger endpoints accept
func joinLedgerCurrencies(codes []currency.Code) (string, error) {
	symbols := make([]string, len(codes))
	for i := range codes {
		if codes[i].IsEmpty() {
			return "", currency.ErrCurrencyCodeEmpty
		}
		symbols[i] = codes[i].Upper().String()
	}
	return strings.Join(symbols, ","), nil
}

// GetFuturesAccountLedgers returns a page of the futures ledger entries (Get Account Ledgers - Futures). While
// positions are open the first entry is Pending: the realised PNL of the current 8-hour settlement period. Page with the
// smallest Offset of the previous page; the time window is at most one day
func (e *Exchange) GetFuturesAccountLedgers(ctx context.Context, req *FuturesAccountLedgersRequest) (*FuturesAccountLedgersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Forward != nil {
		params.Set("forward", strconv.FormatBool(*req.Forward))
	}
	if req.MaxCount != 0 {
		params.Set("maxCount", strconv.FormatUint(req.MaxCount, 10))
	}
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.UnixMilli(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.UnixMilli(), 10))
	}
	var resp *FuturesAccountLedgersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesAccountLedgersEPL, http.MethodGet, common.EncodeURLValues("/v1/transaction-history", params), nil, &resp)
}

package binanceus

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetAccount returns the account's information and balances
func (e *Exchange) GetAccount(ctx context.Context) (*AccountResponse, error) {
	return e.getAccount(ctx, false)
}

func (e *Exchange) getAccount(ctx context.Context, useWebsocket bool) (*AccountResponse, error) {
	var resp *AccountResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/account", "account.status", nil, accountInformationRate, &resp)
}

// GetUserAccountStatus returns whether the account is normal or suspended, and why
func (e *Exchange) GetUserAccountStatus(ctx context.Context) (*AccountStatusResponse, error) {
	var resp *AccountStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/accountStatus", nil, defaultRate, &resp)
}

// GetUserAPITradingStatus returns whether the account's API trading is locked, and the order ratios that lock it
func (e *Exchange) GetUserAPITradingStatus(ctx context.Context) (*APITradingStatusResponse, error) {
	var resp *APITradingStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/apiTradingStatus", nil, defaultRate, &resp)
}

// GetAssetDistributionHistory returns the account's asset distributions, such as rebates and airdrops. A nil request
// returns the most recent
func (e *Exchange) GetAssetDistributionHistory(ctx context.Context, req *AssetDistributionHistoryRequest) (*AssetDistributionHistoryResponse, error) {
	params := url.Values{}
	if req != nil {
		if !req.Asset.IsEmpty() {
			params.Set("asset", req.Asset.Upper().String())
		}
		if req.Category != "" {
			params.Set("category", req.Category)
		}
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp *AssetDistributionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/assetDistributionHistory", params, defaultRate, &resp)
}

// GetTradeFee returns the account's maker and taker commission rates of a symbol, or of every symbol for an empty pair
func (e *Exchange) GetTradeFee(ctx context.Context, symbol currency.Pair) ([]TradeFeeResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.formatSymbol(symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []TradeFeeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/query/trading-fee", params, defaultRate, &resp)
}

// GetCommissionRates returns the account's current commission rates for a symbol
func (e *Exchange) GetCommissionRates(ctx context.Context, symbol currency.Pair) (*AccountCommissionResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AccountCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/account/commission", params, accountCommissionRate, &resp)
}

// GetPast30DaysTradeVolume returns the account's trading volume over the past 30 days
func (e *Exchange) GetPast30DaysTradeVolume(ctx context.Context) (*TradeVolumeResponse, error) {
	var resp *TradeVolumeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/query/trading-volume", nil, defaultRate, &resp)
}

// GetMyFilters returns the filters that apply to the account on a symbol, the only source of its MAX_ASSET filters
func (e *Exchange) GetMyFilters(ctx context.Context, symbol currency.Pair) (*MyFiltersResponse, error) {
	return e.getMyFilters(ctx, symbol, false)
}

func (e *Exchange) getMyFilters(ctx context.Context, symbol currency.Pair, useWebsocket bool) (*MyFiltersResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *MyFiltersResponse
	return resp, e.sendSignedRequest(ctx, useWebsocket, http.MethodGet, "/api/v3/myFilters", "myFilters", params, myFiltersRate, &resp)
}

package kraken

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesSubaccounts calls Get subaccounts, returning the master account's subaccounts with their balances
func (e *Exchange) GetFuturesSubaccounts(ctx context.Context) (*FuturesSubaccountsResponse, error) {
	var resp *FuturesSubaccountsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/subaccounts", nil, nil, &resp)
}

// GetFuturesSubaccountTradingStatus calls Check subaccount trading status
func (e *Exchange) GetFuturesSubaccountTradingStatus(ctx context.Context, subaccountUID string) (*FuturesSubaccountTradingStatusResponse, error) {
	if subaccountUID == "" {
		return nil, errFuturesSubaccountUIDEmpty
	}
	var resp *FuturesSubaccountTradingStatusResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/subaccount/"+url.PathEscape(subaccountUID)+"/trading-enabled", nil, nil, &resp)
}

// UpdateFuturesSubaccountTradingStatus calls Update subaccount trading status, enabling or disabling a subaccount's
// trading
func (e *Exchange) UpdateFuturesSubaccountTradingStatus(ctx context.Context, subaccountUID string, tradingEnabled bool) (*FuturesSubaccountTradingStatusResponse, error) {
	if subaccountUID == "" {
		return nil, errFuturesSubaccountUIDEmpty
	}
	params := url.Values{}
	params.Set("tradingEnabled", strconv.FormatBool(tradingEnabled))
	var resp *FuturesSubaccountTradingStatusResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/subaccount/"+url.PathEscape(subaccountUID)+"/trading-enabled", params, nil, &resp)
}

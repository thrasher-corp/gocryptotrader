package coinbase

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetAccount returns an account by its UUID, as ListAccounts lists them
func (e *Exchange) GetAccount(ctx context.Context, accountUUID string) (*Account, error) {
	if accountUUID == "" {
		return nil, errAccountIDRequired
	}
	var resp *accountResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"accounts/"+accountUUID, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Account == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Account, nil
}

// ListAccounts returns a page of the accounts of the portfolio the API key is permissioned for, 49 by default and at
// most 250; while HasNext is set, the next page starts after the page's Cursor
func (e *Exchange) ListAccounts(ctx context.Context, req *ListAccountsRequest) (*AccountsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *AccountsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"accounts", req.params(), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// params returns an account list request's query
func (r *ListAccountsRequest) params() url.Values {
	params := url.Values{}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	if r.RetailPortfolioID != "" {
		params.Set("retail_portfolio_id", r.RetailPortfolioID)
	}
	return params
}

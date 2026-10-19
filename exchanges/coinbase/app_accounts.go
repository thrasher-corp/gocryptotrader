package coinbase

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errAppAccountOrCurrencyRequired = errors.New("account ID or currency required")
	errAppAccountAndCurrencySet     = errors.New("account ID and currency cannot both be set")
)

// GetCurrentUser returns the public profile of the user the credentials act for
func (a AppAPI) GetCurrentUser(ctx context.Context) (*AppUser, error) {
	resp, err := sendAppRequest[*AppUser](ctx, a, http.MethodGet, "user", nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetPersonalDetails returns the user's legal name and KYC verified primary address. Coinbase documents only OAuth2
// scopes for it, so whether a CDP API key may call it is unverified; its documented response has no data envelope
func (a AppAPI) GetPersonalDetails(ctx context.Context) (*AppPersonalDetails, error) {
	var resp *AppPersonalDetails
	if err := a.e.SendAuthenticatedHTTPRequest(ctx, appEPL, http.MethodGet, appPath+"user/personal-details", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ListAccounts returns a page of the accounts the credentials can access: wallets, fiat accounts, vaults and loan
// collateral accounts
func (a AppAPI) ListAccounts(ctx context.Context, req *AppListAccountsRequest) (*AppListResponse[AppAccount], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := req.Page.params()
	if err != nil {
		return nil, err
	}
	if req.IncludeAvailableToSend {
		params.Set("include_available_to_send", "true")
	}
	resp, err := sendAppRequest[[]AppAccount](ctx, a, http.MethodGet, "accounts", params, nil)
	if err != nil {
		return nil, err
	}
	return listResponse(resp), nil
}

// GetAccount returns an account by its ID, or a currency's primary account by the currency
func (a AppAPI) GetAccount(ctx context.Context, req *AppGetAccountRequest) (*AppAccount, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	account := req.AccountID
	switch {
	case account != "" && !req.Currency.IsEmpty():
		return nil, errAppAccountAndCurrencySet
	case !req.Currency.IsEmpty():
		// Coinbase takes a currency code in place of an account ID for the currency's primary account
		account = req.Currency.Upper().String()
	case account == "":
		return nil, errAppAccountOrCurrencyRequired
	}
	params := url.Values{}
	if req.IncludeAvailableToSend {
		params.Set("include_available_to_send", "true")
	}
	resp, err := sendAppRequest[*AppAccount](ctx, a, http.MethodGet, "accounts/"+account, params, nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

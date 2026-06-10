package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// ListSubAccounts retrieves all sub-accounts for the main account.
func (e *Exchange) ListSubAccounts(ctx context.Context, subAccountType int64) ([]*SubAccount, error) {
	params := url.Values{}
	if subAccountType >= 0 {
		params.Set("type", strconv.FormatInt(subAccountType, 10))
	}
	var resp []*SubAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, common.EncodeURLValues("sub_accounts", params), nil, nil, &resp)
}

// CreateSubAccount creates a new sub-account under the main account.
func (e *Exchange) CreateSubAccount(ctx context.Context, arg *CreateSubAccountRequest) (*SubAccount, error) {
	if arg.LoginName == "" {
		return nil, fmt.Errorf("%w: login name is required", errInvalidSubAccount)
	}
	var resp *SubAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, "sub_accounts", nil, arg, &resp)
}

// GetSubAccount retrieves detailed information about a specific sub-account.
func (e *Exchange) GetSubAccount(ctx context.Context, userID uint64) (*SubAccount, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	var resp *SubAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10), nil, nil, &resp)
}

// ListSubAccountAPIKeys retrieves all API key pairs associated with a specific sub-account.
func (e *Exchange) ListSubAccountAPIKeys(ctx context.Context, userID uint64) ([]*SubAccountAPIKey, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	var resp []*SubAccountAPIKey
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys", nil, nil, &resp)
}

// CreateSubAccountAPIKey creates a new API key pair for a specific sub-account.
func (e *Exchange) CreateSubAccountAPIKey(ctx context.Context, userID uint64, arg *SubAccountKeyRequest) (*SubAccountAPIKey, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	var resp *SubAccountAPIKey
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys", nil, arg, &resp)
}

// GetSubAccountAPIKey retrieves a specific API key pair of a sub-account.
func (e *Exchange) GetSubAccountAPIKey(ctx context.Context, userID uint64, key string) (*SubAccountAPIKey, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if key == "" {
		return nil, errMissingAPIKey
	}
	var resp *SubAccountAPIKey
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, nil, &resp)
}

// UpdateSubAccountAPIKey modifies an existing API key pair of a sub-account.
func (e *Exchange) UpdateSubAccountAPIKey(ctx context.Context, userID uint64, key string, arg *SubAccountKeyRequest) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	if key == "" {
		return errMissingAPIKey
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPatch, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, arg, nil)
}

// DeleteSubAccountAPIKey removes an API key pair from a sub-account.
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, userID uint64, key string) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	if key == "" {
		return errMissingAPIKey
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodDelete, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, nil, nil)
}

// LockSubAccount locks a sub-account, preventing further access to it.
func (e *Exchange) LockSubAccount(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/lock", nil, nil, nil)
}

// UnlockSubAccount restores access to a previously locked sub-account.
func (e *Exchange) UnlockSubAccount(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/unlock", nil, nil, nil)
}

// GetSubAccountMode retrieves the unified account mode for all sub-accounts.
// Unified account mode values: classic, multi_currency, portfolio, single_currency.
func (e *Exchange) GetSubAccountMode(ctx context.Context) ([]*SubAccountMode, error) {
	var resp []*SubAccountMode
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "sub_accounts/unified_mode", nil, nil, &resp)
}

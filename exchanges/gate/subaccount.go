package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// ListSubAccounts retrieves the main account's sub-accounts. A subAccountType of 0 lists every type of sub-account and 1
// lists regular sub-accounts only; a negative value omits the filter, which Gate treats as 1
func (e *Exchange) ListSubAccounts(ctx context.Context, subAccountType int64) ([]*SubAccountResponse, error) {
	params := url.Values{}
	if subAccountType >= 0 {
		params.Set("type", strconv.FormatInt(subAccountType, 10))
	}
	var resp []*SubAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountListEPL, http.MethodGet, "sub_accounts", params, nil, &resp)
}

// CreateSubAccount creates a new sub-account under the main account.
func (e *Exchange) CreateSubAccount(ctx context.Context, arg *CreateSubAccountRequest) (*SubAccountResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.LoginName == "" {
		return nil, fmt.Errorf("%w: login name is required", errInvalidSubAccount)
	}
	var resp *SubAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountCreateEPL, http.MethodPost, "sub_accounts", nil, arg, &resp)
}

// GetSubAccount retrieves detailed information about a specific sub-account.
func (e *Exchange) GetSubAccount(ctx context.Context, userID uint64) (*SubAccountResponse, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	var resp *SubAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountGetEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10), nil, nil, &resp)
}

// ListSubAccountAPIKeys retrieves all API key pairs associated with a specific sub-account.
func (e *Exchange) ListSubAccountAPIKeys(ctx context.Context, userID uint64) ([]*SubAccountAPIKeyResponse, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	var resp []*SubAccountAPIKeyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountListKeysEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys", nil, nil, &resp)
}

// CreateSubAccountAPIKey creates a new API key pair for a specific sub-account.
func (e *Exchange) CreateSubAccountAPIKey(ctx context.Context, userID uint64, arg *SubAccountKeyRequest) (*SubAccountAPIKeyResponse, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *SubAccountAPIKeyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountCreateKeyEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys", nil, arg, &resp)
}

// GetSubAccountAPIKey retrieves a specific API key pair of a sub-account.
func (e *Exchange) GetSubAccountAPIKey(ctx context.Context, userID uint64, key string) (*SubAccountAPIKeyResponse, error) {
	if userID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if key == "" {
		return nil, errMissingAPIKey
	}
	var resp *SubAccountAPIKeyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountGetKeyEPL, http.MethodGet, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, nil, &resp)
}

// UpdateSubAccountAPIKey modifies an existing API key pair of a sub-account. Its mode cannot be changed, and Gate
// clears the permissions and IP whitelist when they are omitted
func (e *Exchange) UpdateSubAccountAPIKey(ctx context.Context, userID uint64, key string, arg *SubAccountKeyRequest) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	if key == "" {
		return errMissingAPIKey
	}
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountUpdateKeyEPL, http.MethodPut, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, arg, nil)
}

// DeleteSubAccountAPIKey removes an API key pair from a sub-account.
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, userID uint64, key string) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	if key == "" {
		return errMissingAPIKey
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountDeleteKeyEPL, http.MethodDelete, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/keys/"+key, nil, nil, nil)
}

// LockSubAccount locks a sub-account, preventing further access to it.
func (e *Exchange) LockSubAccount(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountLockEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/lock", nil, nil, nil)
}

// UnlockSubAccount restores access to a previously locked sub-account.
func (e *Exchange) UnlockSubAccount(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountUnlockEPL, http.MethodPost, "sub_accounts/"+strconv.FormatUint(userID, 10)+"/unlock", nil, nil, nil)
}

// GetSubAccountMode retrieves the unified account mode for all sub-accounts.
// Unified account mode values: classic, multi_currency, portfolio, single_currency.
func (e *Exchange) GetSubAccountMode(ctx context.Context) ([]*SubAccountModeResponse, error) {
	var resp []*SubAccountModeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountUnifiedModeEPL, http.MethodGet, "sub_accounts/unified_mode", nil, nil, &resp)
}

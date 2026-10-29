package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Sub-account validation errors
var (
	errSubAccountPasswordRequired = errors.New("sub-account password is required")
	errSubAccountAccessRequired   = errors.New("sub-account access permissions are required")
	errSubAccountUIDRequired      = errors.New("sub-account UID is required")
)

// CreateSubUser creates a sub-account, which logs in with its SubName (Add Sub Account). Access combines Spot, Futures
// and Margin with commas; how many sub-accounts can be created depends on the VIP level
func (e *Exchange) CreateSubUser(ctx context.Context, req *CreateSubUserRequest) (*CreateSubUserResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.Password == "" {
		return nil, errSubAccountPasswordRequired
	}
	if req.Access == "" {
		return nil, errSubAccountAccessRequired
	}
	body := make(map[string]any)
	body["subName"] = req.SubName
	body["password"] = req.Password
	body["access"] = req.Access
	if req.Remarks != "" {
		body["remarks"] = req.Remarks
	}
	if req.Mode != "" {
		body["mode"] = req.Mode
	}
	var resp *CreateSubUserResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, createSubUserEPL, http.MethodPost, "/v2/sub/user/created", body, &resp)
}

// AddSubAccountMarginPermission grants a sub-account margin trading (Add sub-account Margin Permission). The master
// API key needs the Margin permission and margin trading activated. KuCoin returns no data
func (e *Exchange) AddSubAccountMarginPermission(ctx context.Context, uid uint64) error {
	if uid == 0 {
		return errSubAccountUIDRequired
	}
	var resp any
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addSubAccountMarginPermissionEPL, http.MethodPost, "/v3/sub/user/margin/enable", map[string]any{"uid": strconv.FormatUint(uid, 10)}, &resp)
}

// AddSubAccountFuturesPermission grants a sub-account futures trading (Add sub-account Futures Permission). The master
// API key needs the Futures permission and futures trading activated. KuCoin returns no data
func (e *Exchange) AddSubAccountFuturesPermission(ctx context.Context, uid uint64) error {
	if uid == 0 {
		return errSubAccountUIDRequired
	}
	var resp any
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addSubAccountFuturesPermissionEPL, http.MethodPost, "/v3/sub/user/futures/enable", map[string]any{"uid": strconv.FormatUint(uid, 10)}, &resp)
}

// GetAllSubAccountsInfoV2 returns a page of the sub-accounts (Get sub-account List - Summary Info). pageSize is 1 to
// 100, 10 by default
func (e *Exchange) GetAllSubAccountsInfoV2(ctx context.Context, currentPage, pageSize uint64) (*AllSubAccountsInfoV2Response, error) {
	params := url.Values{}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *AllSubAccountsInfoV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAllSubAccountsInfoV2EPL, http.MethodGet, common.EncodeURLValues("/v2/sub/user", params), nil, &resp)
}

// GetSubAccountDetailBalance returns the funding, spot and margin balances of a sub-account (Get sub-account Detail -
// Balance)
func (e *Exchange) GetSubAccountDetailBalance(ctx context.Context, req *SubAccountDetailBalanceRequest) (*SubAccountDetailBalanceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubUserID == "" {
		return nil, errSubUserIDRequired
	}
	params := url.Values{}
	if req.IncludeBaseAmount {
		params.Set("includeBaseAmount", "true")
	}
	if !req.BaseCurrency.IsEmpty() {
		params.Set("baseCurrency", req.BaseCurrency.Upper().String())
	}
	if req.BaseAmount != 0 {
		params.Set("baseAmount", strconv.FormatFloat(req.BaseAmount, 'f', -1, 64))
	}
	var resp *SubAccountDetailBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSubAccountDetailBalanceEPL, http.MethodGet, common.EncodeURLValues("/v1/sub-accounts/"+req.SubUserID, params), nil, &resp)
}

// GetAllSubAccountsBalanceV2 returns a page of the sub-accounts' funding, spot and margin balances (Get sub-account
// List - Spot Balance (V2)). pageSize is 10 to 100, 10 by default
func (e *Exchange) GetAllSubAccountsBalanceV2(ctx context.Context, currentPage, pageSize uint64) (*AllSubAccountsBalanceV2Response, error) {
	params := url.Values{}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *AllSubAccountsBalanceV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAllSubAccountsBalanceV2EPL, http.MethodGet, common.EncodeURLValues("/v2/sub-accounts", params), nil, &resp)
}

// GetAllFuturesSubAccountBalances returns the futures accounts of the master and every sub-account in a settlement
// currency, XBT unless one is given (Get sub-account List - Futures Balance (V2))
func (e *Exchange) GetAllFuturesSubAccountBalances(ctx context.Context, ccy currency.Code) (*AllFuturesSubAccountBalancesResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp *AllFuturesSubAccountBalancesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getAllFuturesSubAccountBalancesEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview-all", params), nil, &resp)
}

// GetSubAccountAPIList returns the API keys of a sub-account, or only apiKey when it is given (Get sub-account API
// List). The request must be signed with a master account key
func (e *Exchange) GetSubAccountAPIList(ctx context.Context, subName, apiKey string) ([]SubAccountAPIKeyResponse, error) {
	if subName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subName", subName)
	if apiKey != "" {
		params.Set("apiKey", apiKey)
	}
	var resp []SubAccountAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSubAccountAPIListEPL, http.MethodGet, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

// CreateSubAccountAPIKey creates an API key for a sub-account (Add sub-account API). An IP whitelist is required when
// the permissions include Transfer
func (e *Exchange) CreateSubAccountAPIKey(ctx context.Context, req *CreateSubAccountAPIKeyRequest) (*CreateSubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.Passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	if req.Remark == "" {
		return nil, errRemarkIsRequired
	}
	body := make(map[string]any)
	body["subName"] = req.SubName
	body["passphrase"] = req.Passphrase
	body["remark"] = req.Remark
	if req.Permission != "" {
		body["permission"] = req.Permission
	}
	if req.IPWhitelist != "" {
		body["ipWhitelist"] = req.IPWhitelist
	}
	if req.Expire != "" {
		body["expire"] = req.Expire
	}
	var resp *CreateSubAccountAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, createSubAccountAPIKeyEPL, http.MethodPost, "/v1/sub/api-key", body, &resp)
}

// ModifySubAccountAPIKey changes the permissions, IP whitelist or expiry of a sub-account API key (Modify sub-account
// API). An IP whitelist is required when the permissions include Transfer
func (e *Exchange) ModifySubAccountAPIKey(ctx context.Context, req *ModifySubAccountAPIKeyRequest) (*ModifySubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if req.Passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	body := make(map[string]any)
	body["subName"] = req.SubName
	body["apiKey"] = req.APIKey
	body["passphrase"] = req.Passphrase
	if req.Permission != "" {
		body["permission"] = req.Permission
	}
	if req.IPWhitelist != "" {
		body["ipWhitelist"] = req.IPWhitelist
	}
	if req.Expire != "" {
		body["expire"] = req.Expire
	}
	var resp *ModifySubAccountAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifySubAccountAPIKeyEPL, http.MethodPost, "/v1/sub/api-key/update", body, &resp)
}

// DeleteSubAccountAPIKey deletes a sub-account API key (Delete sub-account API)
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, req *DeleteSubAccountAPIKeyRequest) (*DeleteSubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if req.Passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	params := url.Values{}
	params.Set("apiKey", req.APIKey)
	params.Set("subName", req.SubName)
	params.Set("passphrase", req.Passphrase)
	var resp *DeleteSubAccountAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, deleteSubAccountAPIKeyEPL, http.MethodDelete, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

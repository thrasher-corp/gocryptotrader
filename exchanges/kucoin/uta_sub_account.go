package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Unified trading account sub-account errors
var (
	errPasswordRequired = errors.New("password is required")
	errAccessRequired   = errors.New("access is required")
	errNoRateLimits     = errors.New("no rate limits provided")
	errRateRequired     = errors.New("rate is required")
)

// GetUTASubAccountCurrencyAssets returns sub-accounts' assets per account and currency, for every sub-account unless a
// UID is given (Get Sub Account Currency Assets)
func (e *Exchange) GetUTASubAccountCurrencyAssets(ctx context.Context, req *UTASubAccountCurrencyAssetsRequest) (*UTASubAccountCurrencyAssetsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.UID != "" {
		params.Set("uid", req.UID)
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(req.LastID, 10))
	}
	var resp *UTASubAccountCurrencyAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTASubAccountCurrencyAssetsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/sub-account/balance", params), nil, &resp)
}

// GetUTASubAccountList returns a page of sub-accounts; zero values use the defaults of page 1 and 10 per page (Get Sub
// Account List)
func (e *Exchange) GetUTASubAccountList(ctx context.Context, currentPage, pageSize uint64) (*UTASubAccountListResponse, error) {
	params := url.Values{}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *UTASubAccountListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTASubAccountListEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/user/sub-account-list", params), nil, &resp)
}

// GetUTASubAccountAPIList returns a sub-account's API keys, or the one apiKey names; it must be signed with the master
// account's key and does not cover ND broker sub-accounts (Get Sub Account API List)
func (e *Exchange) GetUTASubAccountAPIList(ctx context.Context, apiKey, subName string) ([]UTASubAccountAPI, error) {
	if subName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	if apiKey != "" {
		params.Set("apiKey", apiKey)
	}
	params.Set("subName", subName)
	var resp []UTASubAccountAPI
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTASubAccountAPIListEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/user/sub-api-key", params), nil, &resp)
}

// SetUTASubAccountTransferPermission sets whether sub-accounts' API keys can initiate SUB_TO_SUB transfers (Set Sub
// Account Transfer Permission)
func (e *Exchange) SetUTASubAccountTransferPermission(ctx context.Context, subUIDs []string, subToSub bool) ([]UTASubAccountTransferPermission, error) {
	if len(subUIDs) == 0 {
		return nil, errSubUserIDRequired
	}
	body := make(map[string]any)
	body["subUids"] = strings.Join(subUIDs, ",")
	body["subToSub"] = subToSub
	var resp []UTASubAccountTransferPermission
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, setUTASubAccountTransferPermissionEPL, http.MethodPost, "/ua/v2/sub-account/canTransferOut", body, &resp)
}

// AddUTASubAccount creates a sub-account, which logs in with its sub-account name. How many can be created depends on
// the VIP level (Add Sub Account)
func (e *Exchange) AddUTASubAccount(ctx context.Context, req *AddUTASubAccountRequest) (*AddUTASubAccountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, errPasswordRequired
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.Access == "" {
		return nil, errAccessRequired
	}
	body := make(map[string]any)
	body["password"] = req.Password
	body["subName"] = req.SubName
	body["access"] = req.Access
	if req.Remarks != "" {
		body["remarks"] = req.Remarks
	}
	if req.Mode != "" {
		body["mode"] = req.Mode
	}
	var resp *AddUTASubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addUTASubAccountEPL, http.MethodPost, "/ua/v2/user/sub/create-sub-account", body, &resp)
}

// AddUTASubAccountAPI creates an API key for a sub-account; the response holds the key's secret (Add Sub Account API)
func (e *Exchange) AddUTASubAccountAPI(ctx context.Context, req *AddUTASubAccountAPIRequest) (*AddUTASubAccountAPIResponse, error) {
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
	var resp *AddUTASubAccountAPIResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addUTASubAccountAPIEPL, http.MethodPost, "/ua/v2/user/create-sub-api-key", body, &resp)
}

// ModifyUTASubAccountAPI changes a sub-account API key's permissions, IP whitelist or expiry (Modify Sub Account API)
func (e *Exchange) ModifyUTASubAccountAPI(ctx context.Context, req *ModifyUTASubAccountAPIRequest) (*ModifyUTASubAccountAPIResponse, error) {
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
	var resp *ModifyUTASubAccountAPIResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyUTASubAccountAPIEPL, http.MethodPost, "/ua/v2/user/modify-sub-api-key", body, &resp)
}

// SetUTASubAccountsAPIRateLimit sets sub-accounts' API rate limits out of the master account's quota. Each limit is set
// or rejected on its own, so every result's Code must be checked (Set Sub Accounts API Rate Limit)
func (e *Exchange) SetUTASubAccountsAPIRateLimit(ctx context.Context, limits []SetUTASubAccountAPIRateLimitRequest) (*SetUTASubAccountsAPIRateLimitResponse, error) {
	if len(limits) == 0 {
		return nil, errNoRateLimits
	}
	list := make([]map[string]any, len(limits))
	for i := range limits {
		if limits[i].UID == "" {
			return nil, fmt.Errorf("%w: limit %d", errUIDRequired, i)
		}
		if limits[i].Rate == 0 {
			return nil, fmt.Errorf("%w: limit %d", errRateRequired, i)
		}
		list[i] = map[string]any{
			"uid":  limits[i].UID,
			"rate": limits[i].Rate,
		}
	}
	var resp *SetUTASubAccountsAPIRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, setUTASubAccountsAPIRateLimitEPL, http.MethodPost, "/ua/v2/rate-limit/set", map[string]any{"list": list}, &resp)
}

// DeleteUTASubAccountAPI deletes a sub-account API key (Delete Sub Account API)
func (e *Exchange) DeleteUTASubAccountAPI(ctx context.Context, req *DeleteUTASubAccountAPIRequest) (*DeleteUTASubAccountAPIResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if req.SubName == "" {
		return nil, errInvalidSubAccountName
	}
	if req.Passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	params := url.Values{}
	params.Set("apiKey", req.APIKey)
	params.Set("subName", req.SubName)
	params.Set("passphrase", req.Passphrase)
	var resp *DeleteUTASubAccountAPIResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, deleteUTASubAccountAPIEPL, http.MethodDelete, common.EncodeURLValues("/ua/v2/user/sub-api-key", params), nil, &resp)
}

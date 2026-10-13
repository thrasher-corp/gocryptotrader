package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errAgreementCategoryRequired = errors.New("agreement category required")
	errAPIKeyPermissionsRequired = errors.New("API key permissions required")
)

// SignAgreement signs the trading agreement of a commodity contract category with the master account's API key; its
// subaccounts can trade once the master account has signed (Sign Agreement)
func (e *Exchange) SignAgreement(ctx context.Context, req *SignAgreementRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Category == 0 && req.CategoryV2 == 0 {
		return errAgreementCategoryRequired
	}
	body := map[string]any{"agree": true}
	if req.Category != 0 {
		body["category"] = req.Category
	}
	if req.CategoryV2 != 0 {
		body["categoryV2"] = req.CategoryV2
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/agreement", nil, body, nil, signAgreementEPL)
}

// CreateNewSubUserID creates a sub UID with the master account's API key (Create Sub UID)
func (e *Exchange) CreateNewSubUserID(ctx context.Context, req *CreateSubUserRequest) (*CreateSubUserResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Username == "" {
		return nil, errMissingUsername
	}
	if req.MemberType == 0 {
		return nil, errInvalidMemberType
	}
	body := map[string]any{
		"username":   req.Username,
		"memberType": req.MemberType,
	}
	if req.Password != "" {
		body["password"] = req.Password
	}
	if req.Switch {
		body["switch"] = 1
	}
	if req.Note != "" {
		body["note"] = req.Note
	}
	var resp *CreateSubUserResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-sub-member", nil, body, &resp, createNewSubUserIDEPL)
}

// CreateSubUIDAPIKey creates an API key for a sub UID with the master account's API key (Create Sub UID API Key)
func (e *Exchange) CreateSubUIDAPIKey(ctx context.Context, req *CreateSubUIDAPIKeyRequest) (*CreateSubUIDAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubUID == 0 {
		return nil, errMissingUserID
	}
	permissions := req.Permissions.body()
	if len(permissions) == 0 {
		return nil, errAPIKeyPermissionsRequired
	}
	body := map[string]any{
		"subuid":      req.SubUID,
		"readOnly":    boolToInt(req.ReadOnly),
		"permissions": permissions,
	}
	if req.Note != "" {
		body["note"] = req.Note
	}
	if len(req.IPs) != 0 {
		body["ips"] = strings.Join(req.IPs, ",")
	}
	var resp *CreateSubUIDAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-sub-api", nil, body, &resp, createSubUIDAPIKeyEPL)
}

// GetSubUIDList returns up to 1000 sub UIDs of the master account; GetSubUIDListUnlimited pages through more (Get
// Sub UID List (Limited))
func (e *Exchange) GetSubUIDList(ctx context.Context) (*SubUIDListResponse, error) {
	var resp *SubUIDListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-sub-members", nil, nil, &resp, getSubUIDListEPL)
}

// GetSubUIDListUnlimited returns a page of the master account's sub UIDs, up to 100 per page (Get Sub UID List
// (Unlimited))
func (e *Exchange) GetSubUIDListUnlimited(ctx context.Context, pageSize uint64, nextCursor string) (*SubUIDListUnlimitedResponse, error) {
	params := url.Values{}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	if nextCursor != "" {
		params.Set("nextCursor", nextCursor)
	}
	var resp *SubUIDListUnlimitedResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/submembers", params, nil, &resp, getSubUIDListUnlimitedEPL)
}

// GetFundCustodialSubAccounts returns a page of an institutional client's fund custodial subaccounts, up to 100 per
// page (Get Fund Custodial Sub Acct)
func (e *Exchange) GetFundCustodialSubAccounts(ctx context.Context, pageSize uint64, nextCursor string) (*FundCustodialSubAccountsResponse, error) {
	params := url.Values{}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	if nextCursor != "" {
		params.Set("nextCursor", nextCursor)
	}
	var resp *FundCustodialSubAccountsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/escrow_sub_members", params, nil, &resp, getFundCustodialSubAccountsEPL)
}

// FreezeSubUID freezes or unfreezes a sub UID with the master account's API key (Freeze Sub UID)
func (e *Exchange) FreezeSubUID(ctx context.Context, subUID uint64, frozen bool) error {
	if subUID == 0 {
		return errMissingUserID
	}
	body := map[string]any{
		"subuid": subUID,
		"frozen": boolToInt(frozen),
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/frozen-sub-member", nil, body, nil, freezeSubUIDEPL)
}

// GetAPIKeyInformation returns the details of the API key making the request (Get API Key Information)
func (e *Exchange) GetAPIKeyInformation(ctx context.Context) (*APIKeyInformationResponse, error) {
	var resp *APIKeyInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-api", nil, nil, &resp, getAPIKeyInformationEPL)
}

// GetSubAccountAllAPIKeys returns a page of a sub UID's API keys, with the master account's API key (Get Sub Account
// All API Keys)
func (e *Exchange) GetSubAccountAllAPIKeys(ctx context.Context, req *SubAccountAllAPIKeysRequest) (*SubAccountAllAPIKeysResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubMemberID == "" {
		return nil, errMemberIDRequired
	}
	params := url.Values{}
	params.Set("subMemberId", req.SubMemberID)
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *SubAccountAllAPIKeysResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/sub-apikeys", params, nil, &resp, getSubAccountAllAPIKeysEPL)
}

// GetUIDWalletType returns the wallet types of the account, and of up to 200 of its sub UIDs when a master account's
// API key passes their memberIDs; the master account is always listed first (Get UID Wallet Type)
func (e *Exchange) GetUIDWalletType(ctx context.Context, memberIDs ...string) (*UIDWalletTypeResponse, error) {
	params := url.Values{}
	if len(memberIDs) != 0 {
		params.Set("memberIds", strings.Join(memberIDs, ","))
	}
	var resp *UIDWalletTypeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/get-member-type", params, nil, &resp, getUIDWalletTypeEPL)
}

// ModifyMasterAPIKey changes the settings of the master account API key making the request (Modify Master API Key)
func (e *Exchange) ModifyMasterAPIKey(ctx context.Context, req *ModifyMasterAPIKeyRequest) (*ModifyMasterAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if req.ReadOnly != nil {
		body["readOnly"] = boolToInt(*req.ReadOnly)
	}
	if permissions := req.Permissions.body(); len(permissions) != 0 {
		body["permissions"] = permissions
	}
	var resp *ModifyMasterAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-api", nil, body, &resp, modifyMasterAPIKeyEPL)
}

// ModifySubAPIKey changes the settings of a sub account API key: the key making the request, or the key APIKey names
// when a master account's key makes it (Modify Sub API Key)
func (e *Exchange) ModifySubAPIKey(ctx context.Context, req *ModifySubAPIKeyRequest) (*ModifySubAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if req.APIKey != "" {
		body["apikey"] = req.APIKey
	}
	if req.ReadOnly != nil {
		body["readOnly"] = boolToInt(*req.ReadOnly)
	}
	if len(req.IPs) != 0 {
		body["ips"] = strings.Join(req.IPs, ",")
	}
	if permissions := req.Permissions.body(); len(permissions) != 0 {
		body["permissions"] = permissions
	}
	var resp *ModifySubAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-sub-api", nil, body, &resp, modifySubAPIKeyEPL)
}

// DeleteSubUID deletes a sub UID whose assets are worth no more than 0.001 USDT, with the master account's API key
// (Delete Sub UID)
func (e *Exchange) DeleteSubUID(ctx context.Context, subMemberID string) error {
	if subMemberID == "" {
		return errMemberIDRequired
	}
	body := map[string]any{"subMemberId": subMemberID}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/del-submember", nil, body, nil, deleteSubUIDEPL)
}

// DeleteMasterAPIKey deletes the master account API key making the request, which stops working immediately (Delete
// Master API Key)
func (e *Exchange) DeleteMasterAPIKey(ctx context.Context) error {
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-api", nil, nil, nil, deleteMasterAPIKeyEPL)
}

// DeleteSubAccountAPIKey deletes a sub account API key, which stops working immediately: the key making the request
// when apiKey is empty, or apiKey when a master account's key makes it (Delete Sub API Key)
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, apiKey string) error {
	body := map[string]any{}
	if apiKey != "" {
		body["apikey"] = apiKey
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-sub-api", nil, body, nil, deleteSubAccountAPIKeyEPL)
}

// GetFriendReferrals returns a page of the users the account has referred (Get Friend Referrals)
func (e *Exchange) GetFriendReferrals(ctx context.Context, req *FriendReferralsRequest) (*FriendReferralsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FriendReferralsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/invitation/referrals", params, nil, &resp, getFriendReferralsEPL)
}

// GetReferralCode returns the account's active referral codes and their registration links; a subaccount gets its
// master account's codes (Get Referral Code)
func (e *Exchange) GetReferralCode(ctx context.Context) (*ReferralCodeResponse, error) {
	var resp *ReferralCodeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/invitation/code", nil, nil, &resp, defaultEPL)
}

// boolToInt returns the 0 or 1 Bybit takes for a boolean integer parameter
func boolToInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// addPermission adds a permission list to a request's permissions unless it is nil, so that callers can send an
// empty list
func addPermission(permissions map[string]any, name string, list []string) {
	if list != nil {
		permissions[name] = list
	}
}

// body returns the permissions to send
func (p *CreateSubUIDAPIKeyPermissionsRequest) body() map[string]any {
	permissions := map[string]any{}
	addPermission(permissions, "ContractTrade", p.ContractTrade)
	addPermission(permissions, "Spot", p.Spot)
	addPermission(permissions, "Options", p.Options)
	addPermission(permissions, "Wallet", p.Wallet)
	addPermission(permissions, "Exchange", p.Exchange)
	addPermission(permissions, "Earn", p.Earn)
	return permissions
}

// body returns the permissions to send
func (p *ModifyMasterAPIKeyPermissionsRequest) body() map[string]any {
	permissions := map[string]any{}
	addPermission(permissions, "ContractTrade", p.ContractTrade)
	addPermission(permissions, "Spot", p.Spot)
	addPermission(permissions, "Wallet", p.Wallet)
	addPermission(permissions, "Options", p.Options)
	addPermission(permissions, "Exchange", p.Exchange)
	addPermission(permissions, "Earn", p.Earn)
	addPermission(permissions, "FiatP2P", p.FiatP2P)
	addPermission(permissions, "FiatBitPay", p.FiatBitPay)
	addPermission(permissions, "FiatConvertBroker", p.FiatConvertBroker)
	addPermission(permissions, "BitCard", p.BitCard)
	addPermission(permissions, "ByXPost", p.ByXPost)
	addPermission(permissions, "Affiliate", p.Affiliate)
	addPermission(permissions, "Derivatives", p.Derivatives)
	addPermission(permissions, "BlockTrade", p.BlockTrade)
	return permissions
}

// body returns the permissions to send
func (p *ModifySubAPIKeyPermissionsRequest) body() map[string]any {
	permissions := map[string]any{}
	addPermission(permissions, "ContractTrade", p.ContractTrade)
	addPermission(permissions, "Spot", p.Spot)
	addPermission(permissions, "Wallet", p.Wallet)
	addPermission(permissions, "Options", p.Options)
	addPermission(permissions, "Derivatives", p.Derivatives)
	addPermission(permissions, "Exchange", p.Exchange)
	addPermission(permissions, "Earn", p.Earn)
	return permissions
}

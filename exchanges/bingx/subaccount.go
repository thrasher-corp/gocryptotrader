package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSubAccountStringRequired   = errors.New("sub-account username is required")
	errSubAccountUIDRequired      = errors.New("sub-account UID is required")
	errAccountUIDRequired         = errors.New("account UID is required")
	errSubAccountPageRequired     = errors.New("sub-account query page is required")
	errSubAccountPageSizeRequired = errors.New("sub-account query page size is required")
	errAPIKeyRequired             = errors.New("API key is required")
	errAPIKeyNoteRequired         = errors.New("API key note is required")
	errAPIKeyPermissionsRequired  = errors.New("API key permissions are required")
	errDepositNetworkRequired     = errors.New("deposit network is required")
	errSubAccountTypeRequired     = errors.New("sub-account type is required")
	errTransferRemarkRequired     = errors.New("transfer remark is required")
)

// CreateSubAccount creates a sub-account. Only the main account can create one, and the API key needs the sub-account
// management permission
func (e *Exchange) CreateSubAccount(ctx context.Context, req *CreateSubAccountRequest) (*CreateSubAccountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubAccountString == "" {
		return nil, errSubAccountStringRequired
	}
	var resp *CreateSubAccountResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/subAccount/v1/create", req, createSubAccountRate, &resp)
}

// GetAPIKeyPermissions returns the permissions of the API key making the request
func (e *Exchange) GetAPIKeyPermissions(ctx context.Context) (*APIKeyPermissionsResponse, error) {
	var resp *APIKeyPermissionsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/v1/account/apiPermissions", nil, apiKeyPermissionsRate, &resp)
}

// GetAccountUID returns the UID and status of the account making the request
func (e *Exchange) GetAccountUID(ctx context.Context) (*AccountUIDResponse, error) {
	var resp *AccountUIDResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/account/v1/uid", nil, accountUIDRate, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.UID == 0 {
		return nil, fmt.Errorf("%w: account UID", common.ErrNoResponse)
	}
	return resp, nil
}

// GetSubAccountList returns a page of the main account's sub-accounts. Only the main account can list them
func (e *Exchange) GetSubAccountList(ctx context.Context, req *SubAccountListRequest) (*SubAccountListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Page == 0 {
		return nil, errSubAccountPageRequired
	}
	if req.Limit == 0 {
		return nil, errSubAccountPageSizeRequired
	}
	params := url.Values{}
	if req.SubUID != 0 {
		params.Set("subUid", strconv.FormatUint(req.SubUID, 10))
	}
	if req.SubAccountString != "" {
		params.Set("subAccountString", req.SubAccountString)
	}
	if req.IsFeeze != nil {
		params.Set("isFeeze", strconv.FormatBool(*req.IsFeeze))
	}
	params.Set("page", strconv.FormatUint(req.Page, 10))
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	var resp *SubAccountListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/subAccount/v1/list", params, subAccountListRate, &resp)
}

// GetSubAccountAssets calls Query Sub-account Asset Account, returning a sub-account's fund account balance of each
// asset. Only the main account can query it
func (e *Exchange) GetSubAccountAssets(ctx context.Context, subUID uint64) (*AssetBalancesResponse, error) {
	if subUID == 0 {
		return nil, errSubAccountUIDRequired
	}
	params := url.Values{}
	params.Set("subUid", strconv.FormatUint(subUID, 10))
	var resp *AssetBalancesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/subAccount/v1/assets", params, subAccountAssetsRate, &resp)
}

// CreateSubAccountAPIKey creates an API key for a sub-account. Only the main account can create one, and the API key
// needs the sub-account management permission
func (e *Exchange) CreateSubAccountAPIKey(ctx context.Context, req *CreateSubAccountAPIKeyRequest) (*CreateSubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubUID == 0 {
		return nil, errSubAccountUIDRequired
	}
	if req.Note == "" {
		return nil, errAPIKeyNoteRequired
	}
	if len(req.Permissions) == 0 {
		return nil, errAPIKeyPermissionsRequired
	}
	var resp *CreateSubAccountAPIKeyResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/subAccount/v1/apiKey/create", req, createSubAccountAPIKeyRate, &resp)
}

// GetAPIKeyInformation returns the API keys of the main account or one of its sub-accounts, or the key apiKey names
// when it is set
func (e *Exchange) GetAPIKeyInformation(ctx context.Context, uid uint64, apiKey string) (*APIKeyInformationResponse, error) {
	if uid == 0 {
		return nil, errAccountUIDRequired
	}
	params := url.Values{}
	params.Set("uid", strconv.FormatUint(uid, 10))
	if apiKey != "" {
		params.Set("apiKey", apiKey)
	}
	var resp *APIKeyInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/account/v1/apiKey/query", params, apiKeyInformationRate, &resp)
}

// EditSubAccountAPIKey edits the note, permissions and IP whitelist of a sub-account API key. Only the main account can
// edit one, and the API key needs the sub-account management permission
func (e *Exchange) EditSubAccountAPIKey(ctx context.Context, req *EditSubAccountAPIKeyRequest) (*EditSubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubUID == 0 {
		return nil, errSubAccountUIDRequired
	}
	if req.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if req.Note == "" {
		return nil, errAPIKeyNoteRequired
	}
	if len(req.Permissions) == 0 {
		return nil, errAPIKeyPermissionsRequired
	}
	var resp *EditSubAccountAPIKeyResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/subAccount/v1/apiKey/edit", req, editSubAccountAPIKeyRate, &resp)
}

// DeleteSubAccountAPIKey deletes a sub-account API key. Only the main account can delete one, and the API key needs the
// sub-account management permission
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, req *DeleteSubAccountAPIKeyRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.SubUID == 0 {
		return errSubAccountUIDRequired
	}
	if req.APIKey == "" {
		return errAPIKeyRequired
	}
	return e.SendAuthJSONBodyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/subAccount/v1/apiKey/del", req, deleteSubAccountAPIKeyRate, nil)
}

// UpdateSubAccountStatus calls Freeze/Unfreeze Sub-Account; a frozen sub-account cannot log in or trade. Only the main
// account can change it, and the API key needs the sub-account management permission. BingX documents the reply as the
// sub-account and status requested, without an example, so only the outcome is returned
func (e *Exchange) UpdateSubAccountStatus(ctx context.Context, req *UpdateSubAccountStatusRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.SubUID == 0 {
		return errSubAccountUIDRequired
	}
	return e.SendAuthJSONBodyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/subAccount/v1/updateStatus", req, updateSubAccountStatusRate, nil)
}

// AuthoriseSubAccountInternalTransfer sets whether sub-accounts may transfer assets to other accounts through the
// internal transfer endpoints, and returns BingX's result. Only the main account can set it, and the API key needs the
// sub-account management permission
func (e *Exchange) AuthoriseSubAccountInternalTransfer(ctx context.Context, subUIDs []uint64, transferable bool) (bool, error) {
	if len(subUIDs) == 0 {
		return false, errSubAccountUIDRequired
	}
	uids := make([]string, len(subUIDs))
	for i, uid := range subUIDs {
		if uid == 0 {
			return false, errSubAccountUIDRequired
		}
		uids[i] = strconv.FormatUint(uid, 10)
	}
	params := url.Values{}
	params.Set("subUids", strings.Join(uids, ","))
	params.Set("transferable", strconv.FormatBool(transferable))
	var resp bool
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/account/v1/innerTransfer/authorizeSubAccount", params, authoriseSubAccountInternalTransferRate, &resp); err != nil {
		return false, err
	}
	return resp, nil
}

// SubAccountInternalTransfer transfers a coin from the sub-account making the request to another account. Only a
// sub-account can transfer, and its API key needs the sub-account internal transfer permission
func (e *Exchange) SubAccountInternalTransfer(ctx context.Context, req *InternalTransferRequest) (*InternalTransferResponse, error) {
	body, err := fundAccountInternalTransferBody(req)
	if err != nil {
		return nil, err
	}
	var resp *InternalTransferResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/wallets/v1/capital/subAccountInnerTransfer/apply", body, subAccountInternalTransferRate, &resp)
}

// CreateSubAccountDepositAddress returns a sub-account's deposit address of a coin on a network. BingX's Chinese
// documentation titles the endpoint Create Sub-account Deposit Address, its English documentation Query Sub-account
// Deposit Address
func (e *Exchange) CreateSubAccountDepositAddress(ctx context.Context, req *CreateSubAccountDepositAddressRequest) (*CreateSubAccountDepositAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.SubUID == 0 {
		return nil, errSubAccountUIDRequired
	}
	if req.Network == "" {
		return nil, errDepositNetworkRequired
	}
	if req.WalletType == 0 {
		return nil, errWalletTypeRequired
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	params.Set("subUid", strconv.FormatUint(req.SubUID, 10))
	params.Set("network", req.Network)
	params.Set("walletType", strconv.FormatUint(uint64(req.WalletType), 10))
	var resp *CreateSubAccountDepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/deposit/querySubAddress", params, querySubAccountDepositAddressRate, &resp)
}

// GetSubAccountDepositAddress returns a sub-account's deposit address of a coin on each of its networks. Only the main
// account can query them
func (e *Exchange) GetSubAccountDepositAddress(ctx context.Context, req *SubAccountDepositAddressRequest) (*DepositAddressListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.SubUID == 0 {
		return nil, errSubAccountUIDRequired
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	params.Set("subUid", strconv.FormatUint(req.SubUID, 10))
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *DepositAddressListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/subAccount/deposit/address", params, subAccountDepositAddressRate, &resp)
}

// GetSubAccountDepositRecords returns the deposits of a sub-account, or of every sub-account when no UID is set. Only
// the main account can query them
func (e *Exchange) GetSubAccountDepositRecords(ctx context.Context, req *SubAccountDepositRecordsRequest) (*SubAccountDepositRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.SubUID != 0 {
		params.Set("subUid", strconv.FormatUint(req.SubUID, 10))
	}
	if req.Status != nil {
		params.Set("status", strconv.FormatUint(uint64(*req.Status), 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.TxID != "" {
		params.Set("txId", req.TxID)
	}
	var resp *SubAccountDepositRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/deposit/subHisrec", params, subAccountDepositRecordsRate, &resp)
}

// GetSubAccountInternalTransferRecords returns the internal transfers of a coin by the sub-account making the request.
// Only a sub-account can query them
func (e *Exchange) GetSubAccountInternalTransferRecords(ctx context.Context, req *InternalTransferRecordsRequest) (*InternalTransferRecordsResponse, error) {
	params, err := fundAccountInternalTransferRecordsParams(req)
	if err != nil {
		return nil, err
	}
	var resp *InternalTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/subAccount/innerTransfer/records", params, subAccountInternalTransferRecordsRate, &resp)
}

// GetSubMotherAccountTransferHistory returns an account's transfers between the parent account and its sub-accounts.
// Only the parent account can query them
func (e *Exchange) GetSubMotherAccountTransferHistory(ctx context.Context, req *SubMotherAccountTransferHistoryRequest) (*SubMotherAccountTransferHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UID == 0 {
		return nil, errAccountUIDRequired
	}
	params := url.Values{}
	params.Set("uid", strconv.FormatUint(req.UID, 10))
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	if req.TransactionID != "" {
		params.Set("tranId", req.TransactionID)
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.PageID != 0 {
		params.Set("pageId", strconv.FormatUint(req.PageID, 10))
	}
	if req.PagingSize != 0 {
		params.Set("pagingSize", strconv.FormatUint(req.PagingSize, 10))
	}
	var resp *SubMotherAccountTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/account/transfer/v1/subAccount/asset/transferHistory", params, subAccountTransferHistoryRate, &resp)
}

// GetSubMotherAccountTransferableAmount returns the coins that can be transferred from one account to another between
// the parent account and its sub-accounts, with the amount of each. Only the parent account can query them
func (e *Exchange) GetSubMotherAccountTransferableAmount(ctx context.Context, req *SubMotherAccountTransferableAmountRequest) (*SubMotherAccountTransferableAmountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromUID == 0 {
		return nil, fmt.Errorf("%w: fromUid", errAccountUIDRequired)
	}
	if req.FromAccountType == 0 {
		return nil, fmt.Errorf("%w: fromAccountType", errWalletTypeRequired)
	}
	if req.ToUID == 0 {
		return nil, fmt.Errorf("%w: toUid", errAccountUIDRequired)
	}
	if req.ToAccountType == 0 {
		return nil, fmt.Errorf("%w: toAccountType", errWalletTypeRequired)
	}
	var resp *SubMotherAccountTransferableAmountResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/account/transfer/v1/subAccount/transferAsset/supportCoins", req, subAccountTransferableAmountRate, &resp)
}

// SubMotherAccountAssetTransfer transfers an asset between the parent account and a sub-account. Only the parent
// account can transfer, and the API key needs the universal transfer permission
func (e *Exchange) SubMotherAccountAssetTransfer(ctx context.Context, req *SubMotherAccountAssetTransferRequest) (*SubMotherAccountAssetTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AssetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.TransferAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.FromUID == 0 {
		return nil, fmt.Errorf("%w: fromUid", errAccountUIDRequired)
	}
	if req.FromType == 0 {
		return nil, fmt.Errorf("%w: fromType", errSubAccountTypeRequired)
	}
	if req.FromAccountType == 0 {
		return nil, fmt.Errorf("%w: fromAccountType", errWalletTypeRequired)
	}
	if req.ToUID == 0 {
		return nil, fmt.Errorf("%w: toUid", errAccountUIDRequired)
	}
	if req.ToType == 0 {
		return nil, fmt.Errorf("%w: toType", errSubAccountTypeRequired)
	}
	if req.ToAccountType == 0 {
		return nil, fmt.Errorf("%w: toAccountType", errWalletTypeRequired)
	}
	if req.Remark == "" {
		return nil, errTransferRemarkRequired
	}
	body := *req
	body.AssetName = req.AssetName.Upper()
	var nested json.RawMessage
	if err := e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/account/transfer/v1/subAccount/transferAsset", &body, subAccountAssetTransferRate, &nested); err != nil {
		return nil, err
	}
	// BingX's reply nests a second response envelope in its data member
	if err := checkErrorResponse(nested); err != nil {
		return nil, err
	}
	var resp *SubMotherAccountAssetTransferResponse
	if err := decodeResponse(nested, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSubAccountAssetOverview calls Batch Query Sub-Account Asset Overview, returning a page of the USDT value of each of
// the sub-accounts' accounts, or of one sub-account's or account type's when set. Only the main account can query them
func (e *Exchange) GetSubAccountAssetOverview(ctx context.Context, req *SubAccountAssetOverviewRequest) (*SubAccountAssetOverviewResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PageIndex == 0 {
		return nil, errSubAccountPageRequired
	}
	if req.PageSize == 0 {
		return nil, errSubAccountPageSizeRequired
	}
	params := url.Values{}
	if req.SubUID != 0 {
		params.Set("subUid", strconv.FormatUint(req.SubUID, 10))
	}
	if req.AccountType != "" {
		params.Set("accountType", string(req.AccountType))
	}
	params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	var resp *SubAccountAssetOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/subAccount/v1/allAccountBalance", params, subAccountAssetOverviewRate, &resp)
}

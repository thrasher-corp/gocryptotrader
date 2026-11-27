package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSubAccountNameRequired     = errors.New("sub-account name is required")
	errSubAccountTypeRequired     = errors.New("sub-account type is required")
	errAPIKeyLabelRequired        = errors.New("API key label is required")
	errAPIKeyPassphraseRequired   = errors.New("API key passphrase is required")
	errIPAddressesWithUnlink      = errors.New("IP addresses cannot be set while unlinking every IP address")
	errFromSubAccountNameRequired = errors.New("from sub-account name is required")
	errToSubAccountNameRequired   = errors.New("to sub-account name is required")
)

const (
	// maxSubAccountAPIKeyIPAddresses is the most IP addresses OKX links to a sub-account API key
	maxSubAccountAPIKeyIPAddresses = 20
	// maxTransferOutPermissionSubAccounts is the most sub-accounts one SetPermissionOfTransferOut request covers
	maxTransferOutPermissionSubAccounts = 20
)

// GetSubAccountList returns the master account's sub-accounts
func (e *Exchange) GetSubAccountList(ctx context.Context, req *SubAccountListRequest) ([]SubAccountResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Enabled != nil {
		params.Set("enable", strconv.FormatBool(*req.Enabled))
	}
	if req.SubAccountName != "" {
		params.Set("subAcct", req.SubAccountName)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []SubAccountResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, viewSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/subaccount/list", params), nil, &resp, request.AuthenticatedRequest)
}

// CreateSubAccount creates a sub-account; the master account's API key must be linked to IP addresses
func (e *Exchange) CreateSubAccount(ctx context.Context, req *CreateSubAccountRequest) (*CreateSubAccountResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	if req.Type == "" {
		return nil, errSubAccountTypeRequired
	}
	var resp *CreateSubAccountResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createSubAccountEPL, http.MethodPost, "users/subaccount/create-subaccount", req, &resp, request.AuthenticatedRequest)
}

// CreateSubAccountAPIKey creates an API key for a sub-account; the master account's API key must be linked to IP
// addresses
func (e *Exchange) CreateSubAccountAPIKey(ctx context.Context, req *CreateSubAccountAPIKeyRequest) (*CreateSubAccountAPIKeyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	if req.Label == "" {
		return nil, errAPIKeyLabelRequired
	}
	if req.Passphrase == "" {
		return nil, errAPIKeyPassphraseRequired
	}
	if len(req.IPAddresses) > maxSubAccountAPIKeyIPAddresses {
		return nil, fmt.Errorf("%w: at most %d IP addresses", errExceedLimit, maxSubAccountAPIKeyIPAddresses)
	}
	var resp *CreateSubAccountAPIKeyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createSubAccountAPIKeyEPL, http.MethodPost, "users/subaccount/apikey", req, &resp, request.AuthenticatedRequest)
}

// GetSubAccountAPIKeys returns a sub-account's API keys
func (e *Exchange) GetSubAccountAPIKeys(ctx context.Context, req *SubAccountAPIKeysRequest) ([]SubAccountAPIKeyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	params := url.Values{}
	params.Set("subAcct", req.SubAccountName)
	if req.APIKey != "" {
		params.Set("apiKey", req.APIKey)
	}
	var resp []SubAccountAPIKeyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountAPIKeysEPL, http.MethodGet, common.EncodeURLValues("users/subaccount/apikey", params), nil, &resp, request.AuthenticatedRequest)
}

// ResetSubAccountAPIKey resets a sub-account API key's label, permissions or linked IP addresses; the master account's
// API key must be linked to IP addresses
func (e *Exchange) ResetSubAccountAPIKey(ctx context.Context, req *ResetSubAccountAPIKeyRequest) (*ResetSubAccountAPIKeyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	if req.APIKey == "" {
		return nil, errInvalidAPIKey
	}
	if req.UnlinkIPAddresses && len(req.IPAddresses) > 0 {
		return nil, errIPAddressesWithUnlink
	}
	if len(req.IPAddresses) > maxSubAccountAPIKeyIPAddresses {
		return nil, fmt.Errorf("%w: at most %d IP addresses", errExceedLimit, maxSubAccountAPIKeyIPAddresses)
	}
	var resp *ResetSubAccountAPIKeyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, resetSubAccountAPIKeyEPL, http.MethodPost, "users/subaccount/modify-apikey", req, &resp, request.AuthenticatedRequest)
}

// DeleteSubAccountAPIKey deletes a sub-account API key; the master account's API key must be linked to IP addresses
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, req *DeleteSubAccountAPIKeyRequest) (*SubAccountNameResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	if req.APIKey == "" {
		return nil, errInvalidAPIKey
	}
	var resp *SubAccountNameResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, deleteSubAccountAPIKeyEPL, http.MethodPost, "users/subaccount/delete-apikey", req, &resp, request.AuthenticatedRequest)
}

// GetSubAccountTradingBalance returns a sub-account's trading account balance
func (e *Exchange) GetSubAccountTradingBalance(ctx context.Context, subAccountName string) ([]SubAccountTradingBalanceResponse, error) {
	if subAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	params := url.Values{}
	params.Set("subAcct", subAccountName)
	var resp []SubAccountTradingBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountTradingBalanceEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSubAccountFundingBalance returns a sub-account's funding account balances
func (e *Exchange) GetSubAccountFundingBalance(ctx context.Context, req *SubAccountFundingBalanceRequest) ([]FundingBalanceResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	currencies, err := formatFundingCurrencies(req.Currencies, maxFundingCurrencyFilter)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("subAcct", req.SubAccountName)
	if currencies != "" {
		params.Set("ccy", currencies)
	}
	var resp []FundingBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountFundingBalanceEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSubAccountMaximumWithdrawal returns the most each currency can withdraw from a sub-account, for every currency the
// sub-account holds when no currency is given
func (e *Exchange) GetSubAccountMaximumWithdrawal(ctx context.Context, req *SubAccountMaximumWithdrawalRequest) ([]SubAccountMaximumWithdrawalResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.SubAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	currencies, err := formatFundingCurrencies(req.Currencies, maxFundingCurrencyFilter)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("subAcct", req.SubAccountName)
	if currencies != "" {
		params.Set("ccy", currencies)
	}
	var resp []SubAccountMaximumWithdrawalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountMaxWithdrawalEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/max-withdrawal", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSubAccountTransferHistory returns the master account's transfers with its sub-accounts since 28 September 2022
func (e *Exchange) GetSubAccountTransferHistory(ctx context.Context, req *SubAccountTransferHistoryRequest) ([]SubAccountTransferResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.TransferType != "" {
		params.Set("type", req.TransferType)
	}
	if req.SubAccountName != "" {
		params.Set("subAcct", req.SubAccountName)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []SubAccountTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, historyOfSubaccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/bills", params), nil, &resp, request.AuthenticatedRequest)
}

// GetManagedSubAccountTransferHistory returns the transfers of the managed sub-accounts entrusted to a trading team's
// master account
func (e *Exchange) GetManagedSubAccountTransferHistory(ctx context.Context, req *ManagedSubAccountTransferHistoryRequest) ([]ManagedSubAccountTransferResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.TransferType != "" {
		params.Set("type", req.TransferType)
	}
	if req.SubAccountName != "" {
		params.Set("subAcct", req.SubAccountName)
	}
	if req.SubAccountUID != "" {
		params.Set("subUid", req.SubAccountUID)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []ManagedSubAccountTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, managedSubAccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/managed-subaccount-bills", params), nil, &resp, request.AuthenticatedRequest)
}

// TransferBetweenSubAccounts transfers funds between two of the master account's sub-accounts
func (e *Exchange) TransferBetweenSubAccounts(ctx context.Context, req *TransferBetweenSubAccountsRequest) (*TransferIDResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.RemittingAccountType == "" {
		return nil, errRemittingAccountTypeRequired
	}
	if req.BeneficiaryAccountType == "" {
		return nil, errBeneficiaryAccountTypeRequired
	}
	if req.FromSubAccount == "" {
		return nil, errFromSubAccountNameRequired
	}
	if req.ToSubAccount == "" {
		return nil, errToSubAccountNameRequired
	}
	var resp *TransferIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, masterAccountsManageTransfersBetweenSubaccountEPL, http.MethodPost, "asset/subaccount/transfer", req, &resp, request.AuthenticatedRequest)
}

// SetPermissionOfTransferOut sets whether sub-accounts can transfer out; sub-accounts can transfer to the master
// account by default
func (e *Exchange) SetPermissionOfTransferOut(ctx context.Context, req *PermissionOfTransferOutRequest) ([]PermissionOfTransferOutResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if len(req.SubAccountNames) == 0 {
		return nil, errSubAccountNameRequired
	}
	if len(req.SubAccountNames) > maxTransferOutPermissionSubAccounts {
		return nil, fmt.Errorf("%w: at most %d sub-accounts", errExceedLimit, maxTransferOutPermissionSubAccounts)
	}
	if slices.Contains(req.SubAccountNames, "") {
		return nil, errSubAccountNameRequired
	}
	var resp []PermissionOfTransferOutResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setPermissionOfTransferOutEPL, http.MethodPost, "users/subaccount/set-transfer-out", req, &resp, request.AuthenticatedRequest)
}

// GetCustodyTradingSubAccountList returns the sub-accounts a trading team holds in custody
func (e *Exchange) GetCustodyTradingSubAccountList(ctx context.Context, req *CustodyTradingSubAccountListRequest) ([]SubAccountNameResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.SubAccountName != "" {
		params.Set("subAcct", req.SubAccountName)
	}
	var resp []SubAccountNameResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCustodyTradingSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/entrust-subaccount-list", params), nil, &resp, request.AuthenticatedRequest)
}

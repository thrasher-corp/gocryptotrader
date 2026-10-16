package binance

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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSubAccountStringRequired    = errors.New("sub-account string is required")
	errSubAccountIPAddressRequired = errors.New("sub-account API key IP address is required")
	errFromAndToEmailBothSet       = errors.New("from and to email cannot both be set")
)

// CreateVirtualSubAccount calls Create a Virtual Sub-account (For Master Account), which registers a virtual email made
// from subAccountString. The API key needs the trade permission
func (e *Exchange) CreateVirtualSubAccount(ctx context.Context, subAccountString string) (*CreateVirtualSubAccountResponse, error) {
	if subAccountString == "" {
		return nil, errSubAccountStringRequired
	}
	params := url.Values{}
	params.Set("subAccountString", subAccountString)
	var resp *CreateVirtualSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/virtualSubAccount", params, sapiDefaultRate, &resp)
}

// EnableFuturesSubAccount calls Enable Futures for Sub-account (For Master Account)
func (e *Exchange) EnableFuturesSubAccount(ctx context.Context, email string) (*EnableFuturesSubAccountResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *EnableFuturesSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/futures/enable", params, sapiDefaultRate, &resp)
}

// subAccountEmailParams returns parameters holding a required sub-account email
func subAccountEmailParams(email string) (url.Values, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	return params, nil
}

// EnableOptionsForSubAccount calls Enable Options for Sub-account (For Master Account)
func (e *Exchange) EnableOptionsForSubAccount(ctx context.Context, email string) (*EnableOptionsForSubAccountResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *EnableOptionsForSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/eoptions/enable", params, sapiDefaultRate, &resp)
}

// GetV1FuturesPositionRiskSubAccount returns the Futures Position-Risk of Sub-account (For Master Account): its USDⓈ-M
// futures positions
func (e *Exchange) GetV1FuturesPositionRiskSubAccount(ctx context.Context, email string) ([]SubAccountFuturesPositionRisk, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp []SubAccountFuturesPositionRisk
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/futures/positionRisk", params, getFuturesPositionRiskOfSubAccountV1Rate, &resp)
}

// GetV2FuturesPositionRiskSubAccount returns the Futures Position-Risk of Sub-account V2 (For Master Account) of the
// futures type: 1 USDⓈ-M or 2 COIN-M
func (e *Exchange) GetV2FuturesPositionRiskSubAccount(ctx context.Context, email string, futuresType uint64) (*SubAccountFuturesPositionRiskV2Response, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	if futuresType == 0 {
		return nil, errInvalidFuturesType
	}
	params.Set("futuresType", strconv.FormatUint(futuresType, 10))
	var resp *SubAccountFuturesPositionRiskV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/sub-account/futures/positionRisk", params, sapiDefaultRate, &resp)
}

// GetSubAccountStatusOnMarginFutures returns the Sub-account's Status on Margin Or Futures (For Master Account) of email,
// or of every sub-account when email is empty
func (e *Exchange) GetSubAccountStatusOnMarginFutures(ctx context.Context, email string) ([]SubAccountStatus, error) {
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	var resp []SubAccountStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/status", params, getSubAccountStatusOnMarginOrFuturesRate, &resp)
}

// GetSubAccountList returns the Sub-account List (For Master Account)
func (e *Exchange) GetSubAccountList(ctx context.Context, req *SubAccountListRequest) (*SubAccountListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Email != "" {
		params.Set("email", req.Email)
	}
	if req.IsFreeze != nil {
		params.Set("isFreeze", strconv.FormatBool(*req.IsFreeze))
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *SubAccountListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/list", params, sapiDefaultRate, &resp)
}

// GetSubAccountTransactionStatistics returns the Sub-account Transaction Statistics (For Master Account) of a managed
// sub-account, or of the master account when email is empty
func (e *Exchange) GetSubAccountTransactionStatistics(ctx context.Context, email string) (*SubAccountTransactionStatisticsResponse, error) {
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	var resp *SubAccountTransactionStatisticsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/transaction-statistics", params, getSubAccountTransactionStatisticsRate, &resp)
}

// AddIPRestrictionForSubAccountAPIKey calls Add IP Restriction for Sub-Account API key (For Master Account). The API key
// needs the Enable Spot & Margin Trading option
func (e *Exchange) AddIPRestrictionForSubAccountAPIKey(ctx context.Context, req *SubAccountAPIIPRestrictionRequest) (*AddSubAccountAPIIPRestrictionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := subAccountAPIKeyParams(req.Email, req.SubAccountAPIKey)
	if err != nil {
		return nil, err
	}
	if req.RestrictToTrustedIPs {
		params.Set("status", "2")
	} else {
		params.Set("status", "1")
	}
	if len(req.IPAddresses) > 0 {
		params.Set("ipAddress", strings.Join(req.IPAddresses, ","))
	}
	var resp *AddSubAccountAPIIPRestrictionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/sub-account/subAccountApi/ipRestriction", params, addIPRestrictionSubAccountAPIKeyRate, &resp)
}

// subAccountAPIKeyParams returns parameters holding a required sub-account email and API key
func subAccountAPIKeyParams(email, subAccountAPIKey string) (url.Values, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params.Set("subAccountApiKey", subAccountAPIKey)
	return params, nil
}

// DeleteIPListForSubAccountAPIKey calls Delete IP List For a Sub-account API Key (For Master Account). The API key needs
// the Enable Spot & Margin Trading option
func (e *Exchange) DeleteIPListForSubAccountAPIKey(ctx context.Context, email, subAccountAPIKey string, ipAddresses []string) (*SubAccountAPIIPRestrictionResponse, error) {
	params, err := subAccountAPIKeyParams(email, subAccountAPIKey)
	if err != nil {
		return nil, err
	}
	if len(ipAddresses) == 0 {
		return nil, errSubAccountIPAddressRequired
	}
	params.Set("ipAddress", strings.Join(ipAddresses, ","))
	var resp *SubAccountAPIIPRestrictionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/sub-account/subAccountApi/ipRestriction/ipList", params, deleteIPListForSubAccountAPIKeyRate, &resp)
}

// GetIPRestrictionForSubAccountAPIKey returns the IP Restriction for a Sub-account API Key (For Master Account)
func (e *Exchange) GetIPRestrictionForSubAccountAPIKey(ctx context.Context, email, subAccountAPIKey string) (*SubAccountAPIIPRestrictionResponse, error) {
	params, err := subAccountAPIKeyParams(email, subAccountAPIKey)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountAPIIPRestrictionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/subAccountApi/ipRestriction", params, ipRestrictionForSubAccountAPIKeyRate, &resp)
}

// FuturesTransferSubAccount calls Futures Transfer for Sub-account (For Master Account), moving an asset between a
// sub-account's spot and futures accounts. The transfer type is 1 spot to USDⓈ-M, 2 USDⓈ-M to spot, 3 spot to COIN-M or
// 4 COIN-M to spot. The API key needs the Enable Spot & Margin Trading permission
func (e *Exchange) FuturesTransferSubAccount(ctx context.Context, req *SubAccountWalletTransferRequest) (*SubAccountTransferResponse, error) {
	return e.subAccountWalletTransfer(ctx, "/sapi/v1/sub-account/futures/transfer", req)
}

func (e *Exchange) subAccountWalletTransfer(ctx context.Context, path string, req *SubAccountWalletTransferRequest) (*SubAccountTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := subAccountEmailParams(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Type == 0 {
		return nil, errTransferTypeRequired
	}
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("type", strconv.FormatUint(req.Type, 10))
	var resp *SubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, path, params, sapiDefaultRate, &resp)
}

// GetDetailSubAccountFuturesAccount returns the Detail on Sub-account's Futures Account (For Master Account)
func (e *Exchange) GetDetailSubAccountFuturesAccount(ctx context.Context, email string) (*SubAccountFuturesAccountResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountFuturesAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/futures/account", params, getDetailSubAccountFuturesAccountRate, &resp)
}

// GetDetailOnSubAccountsFuturesAccountV2 returns the Detail on Sub-account's Futures Account V2 (For Master Account) of
// the futures type: 1 USDⓈ-M or 2 COIN-M
func (e *Exchange) GetDetailOnSubAccountsFuturesAccountV2(ctx context.Context, email string, futuresType uint64) (*SubAccountFuturesAccountV2Response, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	if futuresType == 0 {
		return nil, errInvalidFuturesType
	}
	params.Set("futuresType", strconv.FormatUint(futuresType, 10))
	var resp *SubAccountFuturesAccountV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/sub-account/futures/account", params, sapiDefaultRate, &resp)
}

// GetDetailOnSubAccountMarginAccount returns the Detail on Sub-account's Margin Account (For Master Account)
func (e *Exchange) GetDetailOnSubAccountMarginAccount(ctx context.Context, email string) (*SubAccountMarginAccountDetailResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountMarginAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/margin/account", params, subAccountMarginAccountDetailRate, &resp)
}

// GetSubAccountDepositAddress returns the Sub-account Deposit Address (For Master Account)
func (e *Exchange) GetSubAccountDepositAddress(ctx context.Context, req *SubAccountDepositAddressRequest) (*SubAccountDepositAddressResponse, error) {
	params, err := subAccountDepositAddressParams(req)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountDepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/subAddress", params, sapiDefaultRate, &resp)
}

// subAccountDepositAddressParams builds the parameters shared by the sub-account and managed sub-account deposit
// address queries
func subAccountDepositAddressParams(req *SubAccountDepositAddressRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := subAccountEmailParams(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params.Set("coin", req.Coin.Upper().String())
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if req.Amount > 0 {
		params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	}
	return params, nil
}

// GetSubAccountDepositHistory returns the Sub-account Deposit History (For Master Account)
func (e *Exchange) GetSubAccountDepositHistory(ctx context.Context, req *SubAccountDepositHistoryRequest) ([]SubAccountDepositRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := subAccountEmailParams(req.Email)
	if err != nil {
		return nil, err
	}
	if req.IncludeSource {
		params.Set("includeSource", "true")
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.Status != nil {
		params.Set("status", strconv.FormatUint(*req.Status, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.TxID != "" {
		params.Set("txId", req.TxID)
	}
	var resp []SubAccountDepositRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/subHisrec", params, sapiDefaultRate, &resp)
}

// GetSummaryOfSubAccountFuturesAccount returns a page of the Summary of Sub-account's Futures Account (For Master
// Account). limit is up to 500
func (e *Exchange) GetSummaryOfSubAccountFuturesAccount(ctx context.Context, page, limit uint64) (*SubAccountFuturesAccountSummaryResponse, error) {
	if page == 0 {
		return nil, errPageNumberRequired
	}
	if limit == 0 {
		return nil, errLimitNumberRequired
	}
	params := url.Values{}
	params.Set("page", strconv.FormatUint(page, 10))
	params.Set("limit", strconv.FormatUint(limit, 10))
	var resp *SubAccountFuturesAccountSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/futures/accountSummary", params, sapiDefaultRate, &resp)
}

// GetSummaryOfSubAccountsFuturesAccountV2 returns the Summary of Sub-account's Futures Account V2 (For Master Account)
// of the futures type: 1 USDⓈ-M or 2 COIN-M. limit is up to 20
func (e *Exchange) GetSummaryOfSubAccountsFuturesAccountV2(ctx context.Context, futuresType, page, limit uint64) (*SubAccountFuturesAccountSummaryV2Response, error) {
	if futuresType == 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("futuresType", strconv.FormatUint(futuresType, 10))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *SubAccountFuturesAccountSummaryV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/sub-account/futures/accountSummary", params, getFuturesSubAccountSummaryV2Rate, &resp)
}

// GetSummaryOfSubAccountMarginAccount returns the Summary of Sub-account's Margin Account (For Master Account)
func (e *Exchange) GetSummaryOfSubAccountMarginAccount(ctx context.Context) (*SubAccountMarginAccountSummaryResponse, error) {
	var resp *SubAccountMarginAccountSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/margin/accountSummary", nil, getSubAccountSummaryOfMarginAccountRate, &resp)
}

// MarginTransferForSubAccount calls Margin Transfer for Sub-account (For Master Account), moving an asset between a
// sub-account's spot and margin accounts. The transfer type is 1 spot to margin or 2 margin to spot. The API key needs
// the Enable Spot & Margin Trading permission
func (e *Exchange) MarginTransferForSubAccount(ctx context.Context, req *SubAccountWalletTransferRequest) (*SubAccountTransferResponse, error) {
	return e.subAccountWalletTransfer(ctx, "/sapi/v1/sub-account/margin/transfer", req)
}

// GetSubAccountAssetsV3 returns the Sub-account Assets (For Master Account)
func (e *Exchange) GetSubAccountAssetsV3(ctx context.Context, email string) (*SubAccountAssetsResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/sub-account/assets", params, getV3SubAccountAssetsRate, &resp)
}

// GetSubAccountAssets returns the Sub-account Assets V4 (For Master Account)
func (e *Exchange) GetSubAccountAssets(ctx context.Context, email string) (*SubAccountAssetsResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v4/sub-account/assets", params, getSubAccountAssetRate, &resp)
}

// GetSubAccountFuturesAssetTransferHistory returns the Sub-account Futures Asset Transfer History (For Master Account)
func (e *Exchange) GetSubAccountFuturesAssetTransferHistory(ctx context.Context, req *SubAccountFuturesAssetTransferHistoryRequest) (*SubAccountFuturesAssetTransferHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := subAccountEmailParams(req.Email)
	if err != nil {
		return nil, err
	}
	if req.FuturesType == 0 {
		return nil, errInvalidFuturesType
	}
	params.Set("futuresType", strconv.FormatUint(req.FuturesType, 10))
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *SubAccountFuturesAssetTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/futures/internalTransfer", params, sapiDefaultRate, &resp)
}

// SubAccountFuturesAssetTransfer calls Sub-account Futures Asset Transfer (For Master Account), moving a futures asset
// between the master account and sub-accounts. A master account can transfer up to 2000 times a minute
func (e *Exchange) SubAccountFuturesAssetTransfer(ctx context.Context, req *SubAccountFuturesAssetTransferRequest) (*SubAccountFuturesAssetTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !common.MatchesEmailPattern(req.FromEmail) {
		return nil, fmt.Errorf("%w: from email", errValidEmailRequired)
	}
	if !common.MatchesEmailPattern(req.ToEmail) {
		return nil, fmt.Errorf("%w: to email", errValidEmailRequired)
	}
	if req.FuturesType == 0 {
		return nil, errInvalidFuturesType
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromEmail", req.FromEmail)
	params.Set("toEmail", req.ToEmail)
	params.Set("futuresType", strconv.FormatUint(req.FuturesType, 10))
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	var resp *SubAccountFuturesAssetTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/futures/internalTransfer", params, subAccountFuturesAssetTransferRate, &resp)
}

// GetSubAccountSpotAssetsSummary returns the Sub-account Spot Assets Summary (For Master Account): the BTC valued
// spot assets of email, or of every sub-account when email is empty. size is up to 20
func (e *Exchange) GetSubAccountSpotAssetsSummary(ctx context.Context, email string, page, size uint64) (*SubAccountSpotAssetsSummaryResponse, error) {
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	var resp *SubAccountSpotAssetsSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/spotSummary", params, sapiDefaultRate, &resp)
}

// GetSubAccountSpotAssetTransferHistory returns the Sub-account Spot Asset Transfer History (For Master Account). The
// master account's transfers out are returned when neither email is set
func (e *Exchange) GetSubAccountSpotAssetTransferHistory(ctx context.Context, req *SubAccountSpotAssetTransferHistoryRequest) ([]SubAccountSpotAssetTransfer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromEmail != "" && req.ToEmail != "" {
		return nil, errFromAndToEmailBothSet
	}
	params := url.Values{}
	if req.FromEmail != "" {
		params.Set("fromEmail", req.FromEmail)
	}
	if req.ToEmail != "" {
		params.Set("toEmail", req.ToEmail)
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []SubAccountSpotAssetTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/sub/transfer/history", params, sapiDefaultRate, &resp)
}

// GetUniversalTransferHistoryForMasterAccount returns the Universal Transfer History (For Master Account). The master
// account's transfers out are returned when neither email is set
func (e *Exchange) GetUniversalTransferHistoryForMasterAccount(ctx context.Context, req *SubAccountUniversalTransferHistoryRequest) (*SubAccountUniversalTransferHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromEmail != "" && req.ToEmail != "" {
		return nil, errFromAndToEmailBothSet
	}
	params := url.Values{}
	if req.FromEmail != "" {
		params.Set("fromEmail", req.FromEmail)
	}
	if req.ToEmail != "" {
		params.Set("toEmail", req.ToEmail)
	}
	if req.ClientTransactionID != "" {
		params.Set("clientTranId", req.ClientTransactionID)
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *SubAccountUniversalTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/universalTransfer", params, sapiDefaultRate, &resp)
}

// UniversalTransferForMasterAccount calls Universal Transfer (For Master Account), moving an asset between the accounts
// of the master account and its sub-accounts. The API key needs the internal transfer option
func (e *Exchange) UniversalTransferForMasterAccount(ctx context.Context, req *SubAccountUniversalTransferRequest) (*SubAccountUniversalTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromAccountType == "" {
		return nil, fmt.Errorf("%w: from account type", errInvalidAccountType)
	}
	if req.ToAccountType == "" {
		return nil, fmt.Errorf("%w: to account type", errInvalidAccountType)
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	if req.FromEmail != "" {
		params.Set("fromEmail", req.FromEmail)
	}
	if req.ToEmail != "" {
		params.Set("toEmail", req.ToEmail)
	}
	params.Set("fromAccountType", req.FromAccountType)
	params.Set("toAccountType", req.ToAccountType)
	if req.ClientTransactionID != "" {
		params.Set("clientTranId", req.ClientTransactionID)
	}
	if !req.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(req.Symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	var resp *SubAccountUniversalTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/universalTransfer", params, universalTransferForMasterAccountRate, &resp)
}

// SubAccountTransferHistoryForSubAccount returns the Sub-account Transfer History (For Sub-account)
func (e *Exchange) SubAccountTransferHistoryForSubAccount(ctx context.Context, req *SubAccountTransferHistoryRequest) ([]SubAccountTransferHistoryRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.Type > 0 {
		params.Set("type", strconv.FormatUint(req.Type, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.ReturnFailHistory {
		params.Set("returnFailHistory", "true")
	}
	var resp []SubAccountTransferHistoryRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/transfer/subUserHistory", params, sapiDefaultRate, &resp)
}

// FromSubAccountTransferToMaster calls Transfer to Master (For Sub-account). The API key needs the Enable Spot & Margin
// Trading permission
func (e *Exchange) FromSubAccountTransferToMaster(ctx context.Context, ccy currency.Code, amount float64) (*SubAccountTransferResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/transfer/subToMaster", params, sapiDefaultRate, &resp)
}

// TransferToSubAccountOfSameMaster calls Transfer to Sub-account of Same Master (For Sub-account). The API key needs the
// Enable Spot & Margin Trading permission
func (e *Exchange) TransferToSubAccountOfSameMaster(ctx context.Context, toEmail string, ccy currency.Code, amount float64) (*SubAccountTransferResponse, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return nil, errValidEmailRequired
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sub-account/transfer/subToSub", params, sapiDefaultRate, &resp)
}

// DepositAssetsIntoTheManagedSubAccount calls Deposit Assets Into The Managed Sub-account (For Investor Master Account).
// The API key needs the Enable Spot & Margin Trading option
func (e *Exchange) DepositAssetsIntoTheManagedSubAccount(ctx context.Context, toEmail string, ccy currency.Code, amount float64) (*ManagedSubAccountTransferResponse, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return nil, errValidEmailRequired
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *ManagedSubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/managed-subaccount/deposit", params, sapiDefaultRate, &resp)
}

// GetManagedSubAccountDepositAddress returns the Managed Sub-account Deposit Address (For Investor Master Account)
func (e *Exchange) GetManagedSubAccountDepositAddress(ctx context.Context, req *SubAccountDepositAddressRequest) (*SubAccountDepositAddressResponse, error) {
	params, err := subAccountDepositAddressParams(req)
	if err != nil {
		return nil, err
	}
	var resp *SubAccountDepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/deposit/address", params, sapiUIDDefaultRate, &resp)
}

// GetManagedSubAccountAssetsDetails returns the Managed Sub-account Asset Details (For Investor Master Account)
func (e *Exchange) GetManagedSubAccountAssetsDetails(ctx context.Context, email string) ([]ManagedSubAccountAsset, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	var resp []ManagedSubAccountAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/asset", params, sapiDefaultRate, &resp)
}

// GetManagedSubAccountFuturesAssetDetails returns the Managed Sub-account Futures Asset Details (For Investor Master
// Account). accountType is USDT_FUTURE, the default, or COIN_FUTURE
func (e *Exchange) GetManagedSubAccountFuturesAssetDetails(ctx context.Context, email, accountType string) (*ManagedSubAccountFuturesAssetDetailsResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *ManagedSubAccountFuturesAssetDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/fetch-future-asset", params, managedSubAccountFuturesAssetDetailRate, &resp)
}

// GetManagedSubAccountList returns the Managed Sub-account List (For Investor). limit is up to 20
func (e *Exchange) GetManagedSubAccountList(ctx context.Context, email string, page, limit uint64) (*ManagedSubAccountListResponse, error) {
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *ManagedSubAccountListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/info", params, getManagedSubAccountListRate, &resp)
}

// GetManagedSubAccountMarginAssetDetails returns the Managed Sub-account Margin Asset Details (For Investor Master
// Account). accountType is MARGIN, the default, or ISOLATED_MARGIN
func (e *Exchange) GetManagedSubAccountMarginAssetDetails(ctx context.Context, email, accountType string) (*ManagedSubAccountMarginAssetDetailsResponse, error) {
	params, err := subAccountEmailParams(email)
	if err != nil {
		return nil, err
	}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *ManagedSubAccountMarginAssetDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/marginAsset", params, sapiDefaultRate, &resp)
}

// GetManagedSubAccountSnapshot returns the Managed Sub-account Snapshot (For Investor Master Account). Only the last
// month is kept, and the last 7 days are returned when no time window is set
func (e *Exchange) GetManagedSubAccountSnapshot(ctx context.Context, email string, req *AccountSnapshotRequest) (*AccountSnapshotResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params, err := accountSnapshotParams(req)
	if err != nil {
		return nil, err
	}
	params.Set("email", email)
	var resp *AccountSnapshotResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/accountSnapshot", params, getManagedSubAccountSnapshotRate, &resp)
}

// GetManagedSubAccountTransferLogForInvestorMasterAccount returns the Managed Sub Account Transfer Log For Investor
// Master Account of a managed sub-account
func (e *Exchange) GetManagedSubAccountTransferLogForInvestorMasterAccount(ctx context.Context, email string, req *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error) {
	return e.managedSubAccountTransferLogForEmail(ctx, "/sapi/v1/managed-subaccount/queryTransLogForInvestor", email, req, sapiDefaultRate)
}

func (e *Exchange) managedSubAccountTransferLogForEmail(ctx context.Context, path, email string, req *ManagedSubAccountTransferLogRequest, f request.EndpointLimit) (*ManagedSubAccountTransferLogResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params, err := managedSubAccountTransferLogParams(req)
	if err != nil {
		return nil, err
	}
	params.Set("email", email)
	var resp *ManagedSubAccountTransferLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, params, f, &resp)
}

// managedSubAccountTransferLogParams builds the parameters shared by the managed sub-account transfer log queries
func managedSubAccountTransferLogParams(req *ManagedSubAccountTransferLogRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page == 0 {
		return nil, errPageNumberRequired
	}
	if req.Limit == 0 {
		return nil, errLimitNumberRequired
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	params.Set("page", strconv.FormatUint(req.Page, 10))
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	if req.Transfers != "" {
		params.Set("transfers", req.Transfers)
	}
	if req.TransferFunctionAccountType != "" {
		params.Set("transferFunctionAccountType", req.TransferFunctionAccountType)
	}
	return params, nil
}

// GetManagedSubAccountTransferLogForTradingTeam returns the Managed Sub Account Transfer Log For Trading Team Master
// Account of a managed sub-account
func (e *Exchange) GetManagedSubAccountTransferLogForTradingTeam(ctx context.Context, email string, req *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error) {
	return e.managedSubAccountTransferLogForEmail(ctx, "/sapi/v1/managed-subaccount/queryTransLogForTradeParent", email, req, managedSubAccountTransferLogRate)
}

// GetManagedSubAccountTransferLog returns the Managed Sub Account Transfer Log (For Trading Team Sub Account)
func (e *Exchange) GetManagedSubAccountTransferLog(ctx context.Context, req *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error) {
	params, err := managedSubAccountTransferLogParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ManagedSubAccountTransferLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/managed-subaccount/query-trans-log", params, managedSubAccountTransferLogRate, &resp)
}

// WithdrawAssetsFromManagedSubAccount calls Withdraw Assets From The Managed Sub-account (For Investor Master Account).
// The API key needs the Enable Spot & Margin Trading permission
func (e *Exchange) WithdrawAssetsFromManagedSubAccount(ctx context.Context, req *ManagedSubAccountWithdrawRequest) (*ManagedSubAccountTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !common.MatchesEmailPattern(req.FromEmail) {
		return nil, errValidEmailRequired
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromEmail", req.FromEmail)
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if !req.TransferDate.IsZero() {
		params.Set("transferDate", strconv.FormatInt(req.TransferDate.UnixMilli(), 10))
	}
	var resp *ManagedSubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/managed-subaccount/withdraw", params, sapiDefaultRate, &resp)
}

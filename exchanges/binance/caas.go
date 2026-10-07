package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// The Crypto-As-A-Service (CAAS) endpoints, formerly Binance Link "Exchange Link", manage a link master account's
// sub-accounts

var (
	errFromOrToIDRequired         = errors.New("fromId or toId is required")
	errSubAccountIDOrSizeRequired = errors.New("sub-account ID or size is required")
)

// ChangeSubAccountAPIPermission calls Change Sub Account API Permission, setting every permission of a sub-account
// API key; margin and futures trading need margin and futures enabled on the sub-account first
func (e *Exchange) ChangeSubAccountAPIPermission(ctx context.Context, arg *BrokerSubAccountAPIPermissionRequest) (*BrokerSubAccountAPIPermissionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerSubAccountAPIKeyParams(arg.SubAccountID, arg.SubAccountAPIKey)
	if err != nil {
		return nil, err
	}
	params.Set("canTrade", strconv.FormatBool(arg.CanTrade))
	params.Set("marginTrade", strconv.FormatBool(arg.MarginTrade))
	params.Set("futuresTrade", strconv.FormatBool(arg.FuturesTrade))
	var resp *BrokerSubAccountAPIPermissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi/permission", params, sapiDefaultRate, &resp)
}

// GetBrokerSubAccounts calls Query Sub Account; size is at most 500, the default
func (e *Exchange) GetBrokerSubAccounts(ctx context.Context, subAccountID string, page, size uint64) ([]BrokerSubAccount, error) {
	params := brokerSubAccountPageParams(subAccountID, page, size)
	var resp []BrokerSubAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccount", params, sapiDefaultRate, &resp)
}

// CreateBrokerSubAccount calls Create a Sub Account, creating a link sub-account with an optional tag of under 32
// characters
func (e *Exchange) CreateBrokerSubAccount(ctx context.Context, tag string) (*CreateBrokerSubAccountResponse, error) {
	params := url.Values{}
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *CreateBrokerSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccount", params, sapiDefaultRate, &resp)
}

// CreateAPIKeyForSubAccount calls Create Api Key for Sub Account; an account may create one key per sub-account a
// second, and a key created with a public key is asymmetric, so the response carries no secret key
func (e *Exchange) CreateAPIKeyForSubAccount(ctx context.Context, arg *BrokerSubAccountAPIKeyRequest) (*BrokerSubAccountAPIKeyResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", arg.SubAccountID)
	if arg.PublicKey != "" {
		params.Set("publicKey", arg.PublicKey)
	}
	params.Set("canTrade", strconv.FormatBool(arg.CanTrade))
	if arg.MarginTrade {
		params.Set("marginTrade", "true")
	}
	if arg.FuturesTrade {
		params.Set("futuresTrade", "true")
	}
	var resp *BrokerSubAccountAPIKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi", params, createAPIKeyForSubAccountRate, &resp)
}

// DeleteSubAccountAPIKey calls Delete Sub Account Api Key; an account may delete one key per sub-account a second
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey string) error {
	params, err := brokerSubAccountAPIKeyParams(subAccountID, subAccountAPIKey)
	if err != nil {
		return err
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/broker/subAccountApi", params, deleteAPIKeyForSubAccountRate, nil)
}

// DeleteIPRestrictionForSubAccountAPIKey calls Delete IP Restriction for Sub Account API Key, removing ipAddress, or
// every IP address when it is empty, from a sub-account API key's trusted IPs
func (e *Exchange) DeleteIPRestrictionForSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey, ipAddress string) (*BrokerSubAccountIPRestrictionResponse, error) {
	params, err := brokerSubAccountAPIKeyParams(subAccountID, subAccountAPIKey)
	if err != nil {
		return nil, err
	}
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *BrokerSubAccountIPRestrictionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/broker/subAccountApi/ipRestriction/ipList", params, sapiDefaultRate, &resp)
}

// EnableFuturesForSubAccount calls Enable Futures for Sub Account; futures cannot be disabled through the API
func (e *Exchange) EnableFuturesForSubAccount(ctx context.Context, subAccountID string) (*BrokerEnableFuturesResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("futures", "true")
	var resp *BrokerEnableFuturesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccount/futures", params, sapiDefaultRate, &resp)
}

// EnableOrDisableBNBBurnForSubAccountMarginInterest calls Enable Or Disable BNB Burn for Sub Account Margin Interest,
// setting whether margin interest is paid in BNB; the sub-account needs margin enabled
func (e *Exchange) EnableOrDisableBNBBurnForSubAccountMarginInterest(ctx context.Context, subAccountID string, interestBNBBurn bool) (*BrokerBNBBurnMarginInterestResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("interestBNBBurn", strconv.FormatBool(interestBNBBurn))
	var resp *BrokerBNBBurnMarginInterestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccount/bnbBurn/marginInterest", params, sapiDefaultRate, &resp)
}

// EnableOrDisableBNBBurnForSubAccountSpotAndMargin calls Enable Or Disable BNB Burn for Sub Account SPOT and MARGIN,
// setting whether spot and margin trading fees are paid in BNB
func (e *Exchange) EnableOrDisableBNBBurnForSubAccountSpotAndMargin(ctx context.Context, subAccountID string, spotBNBBurn bool) (*BrokerBNBBurnSpotResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("spotBNBBurn", strconv.FormatBool(spotBNBBurn))
	var resp *BrokerBNBBurnSpotResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccount/bnbBurn/spot", params, sapiDefaultRate, &resp)
}

// EnableUniversalTransferPermissionForSubAccountAPIKey calls Enable Universal Transfer Permission for Sub Account API
// Key, setting whether the key may call POST /sapi/v1/asset/transfer
func (e *Exchange) EnableUniversalTransferPermissionForSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey string, canUniversalTransfer bool) (*BrokerUniversalTransferPermissionResponse, error) {
	params, err := brokerSubAccountAPIKeyParams(subAccountID, subAccountAPIKey)
	if err != nil {
		return nil, err
	}
	params.Set("canUniversalTransfer", strconv.FormatBool(canUniversalTransfer))
	var resp *BrokerUniversalTransferPermissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi/permission/universalTransfer", params, sapiDefaultRate, &resp)
}

// GetBNBBurnStatusForSubAccount calls Get BNB Burn Status for Sub Account
func (e *Exchange) GetBNBBurnStatusForSubAccount(ctx context.Context, subAccountID string) (*BrokerBNBBurnStatusResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	var resp *BrokerBNBBurnStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccount/bnbBurn/status", params, sapiDefaultRate, &resp)
}

// LinkAccountInformation calls Link Account Information (USER_DATA)
func (e *Exchange) LinkAccountInformation(ctx context.Context) (*LinkAccountInformationResponse, error) {
	var resp *LinkAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/info", nil, sapiDefaultRate, &resp)
}

// UpdateIPRestrictionForSubAccountAPIKey calls Update IP Restriction for Sub-Account API Key (For Master Account)
// (USER_DATA)
func (e *Exchange) UpdateIPRestrictionForSubAccountAPIKey(ctx context.Context, arg *BrokerIPRestrictionUpdateRequest) (*BrokerIPRestrictionUpdateResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerSubAccountAPIKeyParams(arg.SubAccountID, arg.SubAccountAPIKey)
	if err != nil {
		return nil, err
	}
	if arg.Status == 0 {
		return nil, errSubAccountStatusMissing
	}
	params.Set("status", strconv.FormatUint(arg.Status, 10))
	if arg.IPAddress != "" {
		params.Set("ipAddress", arg.IPAddress)
	}
	var resp *BrokerIPRestrictionUpdateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/broker/subAccountApi/ipRestriction", params, updateIPRestrictionForSubAccountAPIKeyRate, &resp)
}

// GetSubAccountDepositHistoryWithBroker calls Get Sub Account Deposit History for a window of at most 7 days; without
// a window the API returns the last 7 days
func (e *Exchange) GetSubAccountDepositHistoryWithBroker(ctx context.Context, arg *BrokerSubAccountDepositHistoryRequest) ([]BrokerSubAccountDeposit, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTimeWindowParams(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	if arg.SubAccountID != "" {
		params.Set("subAccountId", arg.SubAccountID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Offset > 0 {
		params.Set("offset", strconv.FormatUint(arg.Offset, 10))
	}
	var resp []BrokerSubAccountDeposit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccount/depositHist", params, brokerSubAccountDepositHistoryRate, &resp)
}

// GetSubAccountFuturesAssetInfo calls Query Sub Account Futures Asset Info (V3)
func (e *Exchange) GetSubAccountFuturesAssetInfo(ctx context.Context, arg *BrokerFuturesAssetInfoRequest) (*BrokerFuturesAssetInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := brokerSubAccountPageParams(arg.SubAccountID, arg.Page, arg.Size)
	params.Set("futuresType", brokerFuturesType(arg.CoinMargined))
	var resp *BrokerFuturesAssetInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/broker/subAccount/futuresSummary", params, brokerSubAccountFuturesAssetInfoRate, &resp)
}

// GetSubAccountMarginAssetInfo calls Query Sub Account Margin Asset Info; size, at most 20, is required when
// subAccountID is empty
func (e *Exchange) GetSubAccountMarginAssetInfo(ctx context.Context, subAccountID string, page, size uint64) (*BrokerMarginAssetInfoResponse, error) {
	if subAccountID == "" && size == 0 {
		return nil, errSubAccountIDOrSizeRequired
	}
	params := brokerSubAccountPageParams(subAccountID, page, size)
	var resp *BrokerMarginAssetInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccount/marginSummary", params, sapiDefaultRate, &resp)
}

// GetSubAccountSpotAssetInfo calls Query Sub Account Spot Asset Info; size, at most 20, is required when subAccountID
// is empty
func (e *Exchange) GetSubAccountSpotAssetInfo(ctx context.Context, subAccountID string, page, size uint64) (*BrokerSpotAssetInfoResponse, error) {
	if subAccountID == "" && size == 0 {
		return nil, errSubAccountIDOrSizeRequired
	}
	params := brokerSubAccountPageParams(subAccountID, page, size)
	var resp *BrokerSpotAssetInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccount/spotSummary", params, brokerSubAccountSpotAssetInfoRate, &resp)
}

// GetFuturesBrokerSubAccountTransferHistory calls Query Sub Account Transfer History（FUTURES）, covering the last 30
// days
func (e *Exchange) GetFuturesBrokerSubAccountTransferHistory(ctx context.Context, arg *BrokerFuturesTransferHistoryRequest) (*BrokerFuturesTransferHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params, err := brokerTimeWindowParams(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	params.Set("subAccountId", arg.SubAccountID)
	params.Set("futuresType", brokerFuturesType(arg.CoinMargined))
	if arg.ClientTransferID != "" {
		params.Set("clientTranId", arg.ClientTransferID)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *BrokerFuturesTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/transfer/futures", params, sapiDefaultRate, &resp)
}

// SubAccountTransferWithFuturesBroker calls Sub Account Transfer（FUTURES）; an empty FromID or ToID is the master
// account, and the API key needs internal transfers enabled
func (e *Exchange) SubAccountTransferWithFuturesBroker(ctx context.Context, arg *BrokerFuturesTransferRequest) (*BrokerFuturesTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTransferParams(arg.FromID, arg.ToID, arg.Asset, arg.Amount, arg.ClientTransferID)
	if err != nil {
		return nil, err
	}
	params.Set("futuresType", brokerFuturesType(arg.CoinMargined))
	var resp *BrokerFuturesTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/transfer/futures", params, brokerFuturesTransferRate, &resp)
}

// GetSpotBrokerSubAccountTransferHistory calls Query Sub Account Transfer History（SPOT）; FromID or ToID is required
// and the query covers at most 100 days
func (e *Exchange) GetSpotBrokerSubAccountTransferHistory(ctx context.Context, arg *BrokerSpotTransferHistoryRequest) ([]BrokerSpotTransfer, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTransferHistoryParams(arg.FromID, arg.ToID, arg.ClientTransferID, arg.ShowAllStatus, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []BrokerSpotTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/transfer", params, sapiDefaultRate, &resp)
}

// SubAccountTransferWithSpotBroker calls Sub Account Transfer（SPOT）; an empty FromID or ToID is the master account,
// and the API key needs internal transfers enabled
func (e *Exchange) SubAccountTransferWithSpotBroker(ctx context.Context, arg *BrokerSpotTransferRequest) (*BrokerSpotTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTransferParams(arg.FromID, arg.ToID, arg.Asset, arg.Amount, arg.ClientTransferID)
	if err != nil {
		return nil, err
	}
	var resp *BrokerSpotTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/transfer", params, sapiDefaultRate, &resp)
}

// GetUniversalTransferHistoryThroughBroker calls Query Universal Transfer History; FromID or ToID is required and the
// query covers at most 100 days
func (e *Exchange) GetUniversalTransferHistoryThroughBroker(ctx context.Context, arg *BrokerUniversalTransferHistoryRequest) ([]BrokerUniversalTransfer, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTransferHistoryParams(arg.FromID, arg.ToID, arg.ClientTransferID, arg.ShowAllStatus, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []BrokerUniversalTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/universalTransfer", params, sapiDefaultRate, &resp)
}

// UniversalTransferWithBroker calls Universal Transfer between the spot and futures accounts of the master account
// and its sub-accounts; an empty FromID or ToID is the master account, transfers between futures accounts are not
// supported, and the API key needs internal transfers enabled
func (e *Exchange) UniversalTransferWithBroker(ctx context.Context, arg *BrokerUniversalTransferRequest) (*BrokerUniversalTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: fromAccountType is required", errInvalidAccountType)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: toAccountType is required", errInvalidAccountType)
	}
	params, err := brokerTransferParams(arg.FromID, arg.ToID, arg.Asset, arg.Amount, arg.ClientTransferID)
	if err != nil {
		return nil, err
	}
	params.Set("fromAccountType", arg.FromAccountType)
	params.Set("toAccountType", arg.ToAccountType)
	var resp *BrokerUniversalTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/universalTransfer", params, sapiDefaultRate, &resp)
}

// GetSubAccountCoinMarginedFuturesCommissionAdjustment calls Query Sub Account COIN-Ⓜ Futures Commission Adjustment
// for a COIN-M pair such as BTCUSD; a symbol's commission is its fee tier's base commission plus the adjustment
func (e *Exchange) GetSubAccountCoinMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID, pair string) ([]BrokerCoinFuturesCommission, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if pair == "" {
		return nil, fmt.Errorf("%w: pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("pair", pair)
	var resp []BrokerCoinFuturesCommission
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccountApi/commission/coinFutures", params, sapiDefaultRate, &resp)
}

// ChangeSubAccountCoinMarginedFuturesCommissionAdjustment calls Change Sub Account COIN-Ⓜ Futures Commission
// Adjustment; the sub-account needs futures enabled
func (e *Exchange) ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(ctx context.Context, arg *BrokerCoinFuturesCommissionAdjustmentRequest) (*BrokerCoinFuturesCommissionAdjustmentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if arg.Pair == "" {
		return nil, fmt.Errorf("%w: pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("subAccountId", arg.SubAccountID)
	params.Set("pair", arg.Pair)
	params.Set("makerAdjustment", strconv.FormatUint(arg.MakerAdjustment, 10))
	params.Set("takerAdjustment", strconv.FormatUint(arg.TakerAdjustment, 10))
	var resp *BrokerCoinFuturesCommissionAdjustmentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission/coinFutures", params, sapiDefaultRate, &resp)
}

// ChangeSubAccountCommission calls Change Sub Account Commission; the margin commissions default to the spot ones and
// cannot be sent while margin is disabled
func (e *Exchange) ChangeSubAccountCommission(ctx context.Context, arg *BrokerCommissionRequest) (*BrokerCommissionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if arg.MakerCommission <= 0 {
		return nil, fmt.Errorf("%w: makerCommission is required", errCommissionValueRequired)
	}
	if arg.TakerCommission <= 0 {
		return nil, fmt.Errorf("%w: takerCommission is required", errCommissionValueRequired)
	}
	params := url.Values{}
	params.Set("subAccountId", arg.SubAccountID)
	params.Set("makerCommission", strconv.FormatFloat(arg.MakerCommission, 'f', -1, 64))
	params.Set("takerCommission", strconv.FormatFloat(arg.TakerCommission, 'f', -1, 64))
	if arg.MarginMakerCommission > 0 {
		params.Set("marginMakerCommission", strconv.FormatFloat(arg.MarginMakerCommission, 'f', -1, 64))
	}
	if arg.MarginTakerCommission > 0 {
		params.Set("marginTakerCommission", strconv.FormatFloat(arg.MarginTakerCommission, 'f', -1, 64))
	}
	var resp *BrokerCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission", params, brokerSubAccountCommissionRate, &resp)
}

// GetSubAccountUSDTMarginedFuturesCommissionAdjustment calls Query Sub Account USDT-Ⓜ Futures Commission Adjustment;
// a symbol's commission is its fee tier's base commission plus the adjustment
func (e *Exchange) GetSubAccountUSDTMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID string, symbol currency.Pair) ([]BrokerUSDTFuturesCommission, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("symbol", symbolValue)
	var resp []BrokerUSDTFuturesCommission
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/subAccountApi/commission/futures", params, sapiDefaultRate, &resp)
}

// ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment calls Change Sub Account USDT-Ⓜ Futures Commission
// Adjustment; the sub-account needs futures enabled
func (e *Exchange) ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(ctx context.Context, arg *BrokerUSDTFuturesCommissionAdjustmentRequest) (*BrokerUSDTFuturesCommissionAdjustmentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("subAccountId", arg.SubAccountID)
	params.Set("symbol", symbolValue)
	params.Set("makerAdjustment", strconv.FormatUint(arg.MakerAdjustment, 10))
	params.Set("takerAdjustment", strconv.FormatUint(arg.TakerAdjustment, 10))
	var resp *BrokerUSDTFuturesCommissionAdjustmentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission/futures", params, sapiDefaultRate, &resp)
}

// GetSpotBrokerCommissionRebateRecentRecord calls Query Spot Commission Rebate Recent Record for a window of at most
// 7 days; without a window the API returns the last 7 days
func (e *Exchange) GetSpotBrokerCommissionRebateRecentRecord(ctx context.Context, arg *BrokerSpotCommissionRebateRequest) ([]BrokerCommissionRebate, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTimeWindowParams(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	if arg.SubAccountID != "" {
		params.Set("subAccountId", arg.SubAccountID)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	var resp []BrokerCommissionRebate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/rebate/recentRecord", params, sapiDefaultRate, &resp)
}

// GetFuturesBrokerCommissionRebateRecentRecord calls Query Broker Futures Commission Rebate Record
func (e *Exchange) GetFuturesBrokerCommissionRebateRecentRecord(ctx context.Context, arg *BrokerFuturesCommissionRebateRequest) ([]BrokerCommissionRebate, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("futuresType", brokerFuturesType(arg.CoinMargined))
	params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	if arg.FilterResult {
		params.Set("filterResult", "true")
	}
	var resp []BrokerCommissionRebate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/broker/rebate/futures/recentRecord", params, sapiDefaultRate, &resp)
}

// brokerSubAccountAPIKeyParams builds the sub-account and API key parameters the sub-account API key endpoints require
func brokerSubAccountAPIKeyParams(subAccountID, subAccountAPIKey string) (url.Values, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	return params, nil
}

// brokerSubAccountPageParams builds the optional sub-account and paging parameters of the sub-account queries
func brokerSubAccountPageParams(subAccountID string, page, size uint64) url.Values {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	return params
}

// brokerTimeWindowParams builds the optional start and end time parameters, checking the window when both are set
func brokerTimeWindowParams(startTime, endTime time.Time) (url.Values, error) {
	params := url.Values{}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	return params, nil
}

// brokerTransferParams builds the parameters every link sub-account transfer shares
func brokerTransferParams(fromID, toID string, ccy currency.Code, amount float64, clientTransferID string) (url.Values, error) {
	if ccy.IsEmpty() {
		return nil, fmt.Errorf("%w: asset is required", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount is required", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	return params, nil
}

// brokerTransferHistoryParams builds the parameters the spot and universal transfer history queries share
func brokerTransferHistoryParams(fromID, toID, clientTransferID string, showAllStatus bool, startTime, endTime time.Time, page, limit uint64) (url.Values, error) {
	if fromID == "" && toID == "" {
		return nil, errFromOrToIDRequired
	}
	params, err := brokerTimeWindowParams(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	if showAllStatus {
		params.Set("showAllStatus", "true")
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// brokerFuturesType returns the futuresType parameter: 1 for USDT-M and 2 for COIN-M futures
func brokerFuturesType(coinMargined bool) string {
	if coinMargined {
		return "2"
	}
	return "1"
}

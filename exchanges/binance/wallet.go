package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errBNBBurnSettingRequired     = errors.New("a spot or interest BNB burn setting is required")
	errSnapshotTypeRequired       = errors.New("snapshot type is required")
	errHistoricalDataTypeRequired = errors.New("historical data type is required")
)

// GetAccountTradingAPIStatus returns the Account API Trading Status
func (e *Exchange) GetAccountTradingAPIStatus(ctx context.Context) (*AccountAPITradingStatusResponse, error) {
	var resp *AccountAPITradingStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/account/apiTradingStatus", nil, sapiDefaultRate, &resp)
}

// GetUserAccountInfo returns the Account info: the VIP level and which account types are enabled
func (e *Exchange) GetUserAccountInfo(ctx context.Context) (*AccountInfoResponse, error) {
	var resp *AccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/account/info", nil, sapiDefaultRate, &resp)
}

// GetAccountStatus returns the Account Status
func (e *Exchange) GetAccountStatus(ctx context.Context) (*AccountStatusResponse, error) {
	var resp *AccountStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/account/status", nil, sapiDefaultRate, &resp)
}

// GetDailyAccountSnapshot returns the Daily Account Snapshot of the spot, margin or futures account. Only the last month
// is kept, and the last 7 days are returned when no time window is set
func (e *Exchange) GetDailyAccountSnapshot(ctx context.Context, req *AccountSnapshotRequest) (*AccountSnapshotResponse, error) {
	params, err := accountSnapshotParams(req)
	if err != nil {
		return nil, err
	}
	var resp *AccountSnapshotResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/accountSnapshot", params, dailyAccountSnapshotRate, &resp)
}

// accountSnapshotParams builds the parameters shared by the daily account and managed sub-account snapshot queries
func accountSnapshotParams(req *AccountSnapshotRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" {
		return nil, errSnapshotTypeRequired
	}
	params := url.Values{}
	params.Set("type", req.Type)
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

// DisableFastWithdrawalSwitch calls Disable Fast Withdraw Switch, which needs the API key's trade permission
func (e *Exchange) DisableFastWithdrawalSwitch(ctx context.Context) error {
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/account/disableFastWithdrawSwitch", nil, sapiDefaultRate, nil)
}

// EnableFastWithdrawalSwitch calls Enable Fast Withdraw Switch, which needs the API key's trade permission. While it is
// on, withdrawals to Binance accounts are instant off-chain transfers without a fee
func (e *Exchange) EnableFastWithdrawalSwitch(ctx context.Context) error {
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/account/enableFastWithdrawSwitch", nil, sapiDefaultRate, nil)
}

// GetAPIKeyPermission returns the API Key Permission
func (e *Exchange) GetAPIKeyPermission(ctx context.Context) (*APIKeyPermissionResponse, error) {
	var resp *APIKeyPermissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/account/apiRestrictions", nil, sapiDefaultRate, &resp)
}

// GetAssetDetail returns the Asset Detail of every asset, or of ccy when it is set
func (e *Exchange) GetAssetDetail(ctx context.Context, ccy currency.Code) (map[currency.Code]AssetDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	var resp map[currency.Code]AssetDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/assetDetail", params, sapiDefaultRate, &resp)
}

// GetAssetDividendRecord returns the Asset Dividend Record. The time window cannot exceed 180 days
func (e *Exchange) GetAssetDividendRecord(ctx context.Context, req *AssetDividendRecordRequest) (*AssetDividendRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *AssetDividendRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/assetDividend", params, assetDividendRecordRate, &resp)
}

// GetDustLog returns the DustLog: the last 100 dust conversions since 2020-12-01. accountType is SPOT or MARGIN
func (e *Exchange) GetDustLog(ctx context.Context, accountType string, startTime, endTime time.Time) (*DustLogResponse, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	var resp *DustLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/dribblet", params, sapiDefaultRate, &resp)
}

// DustTransfer calls Dust Transfer, converting the dust of assets to BNB. accountType is SPOT, the default, or MARGIN.
// The API key needs the Enable Spot & Margin Trading permission
func (e *Exchange) DustTransfer(ctx context.Context, assets []currency.Code, accountType string) (*DustTransferResponse, error) {
	if len(assets) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	codes := make([]string, len(assets))
	for i := range assets {
		if assets[i].IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		codes[i] = assets[i].Upper().String()
	}
	params := url.Values{}
	params.Set("asset", strings.Join(codes, ","))
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *DustTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/asset/dust", params, dustTransferRate, &resp)
}

// GetFundingAssets returns the Funding Wallet balances of Binance Pay, Binance Card, Binance Gift Card and Stock Token
// assets, or of ccy when it is set
func (e *Exchange) GetFundingAssets(ctx context.Context, ccy currency.Code, needBTCValuation bool) ([]FundingAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []FundingAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/asset/get-funding-asset", params, sapiDefaultRate, &resp)
}

// GetAssetsThatCanBeConvertedIntoBNB returns the Assets That Can Be Converted Into BNB. accountType is SPOT, the
// default, or MARGIN
func (e *Exchange) GetAssetsThatCanBeConvertedIntoBNB(ctx context.Context, accountType string) (*AssetsConvertibleToBNBResponse, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *AssetsConvertibleToBNBResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/asset/dust-btc", params, sapiDefaultRate, &resp)
}

// GetCloudMiningPaymentAndRefundHistory returns the successful Cloud-Mining payments and refunds in a time window
func (e *Exchange) GetCloudMiningPaymentAndRefundHistory(ctx context.Context, req *CloudMiningHistoryRequest) (*CloudMiningHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.TransactionID != 0 {
		params.Set("tranId", strconv.FormatUint(req.TransactionID, 10))
	}
	if req.ClientTransactionID != "" {
		params.Set("clientTranId", req.ClientTransactionID)
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size > 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *CloudMiningHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/ledger-transfer/cloud-mining/queryByPage", params, cloudMiningPaymentAndRefundHistoryRate, &resp)
}

// GetUserDelegationHistory returns the User Delegation History (For Master Account) of a sub-account
func (e *Exchange) GetUserDelegationHistory(ctx context.Context, req *UserDelegationHistoryRequest) (*UserDelegationHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !common.MatchesEmailPattern(req.Email) {
		return nil, errValidEmailRequired
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("email", req.Email)
	params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size > 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *UserDelegationHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/custody/transfer-history", params, getUserDelegationHistoryRate, &resp)
}

// GetUserUniversalTransferHistory returns the User Universal Transfer History of a transfer type. Only the last 6 months
// are kept, and the last 7 days are returned when no time window is set
func (e *Exchange) GetUserUniversalTransferHistory(ctx context.Context, req *UserUniversalTransferHistoryRequest) (*UserUniversalTransferHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" {
		return nil, errTransferTypeRequired
	}
	params := url.Values{}
	params.Set("type", req.Type)
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Current > 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size > 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	if err := e.setIsolatedSymbols(params, req.FromSymbol, req.ToSymbol); err != nil {
		return nil, err
	}
	var resp *UserUniversalTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/transfer", params, sapiDefaultRate, &resp)
}

// UserUniversalTransfer calls User Universal Transfer, moving an asset between two of the user's wallets. The API key
// needs the Permits Universal Transfer option
func (e *Exchange) UserUniversalTransfer(ctx context.Context, req *UserUniversalTransferRequest) (*UserUniversalTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" {
		return nil, errTransferTypeRequired
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("type", req.Type)
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if err := e.setIsolatedSymbols(params, req.FromSymbol, req.ToSymbol); err != nil {
		return nil, err
	}
	var resp *UserUniversalTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/asset/transfer", params, userUniversalTransferRate, &resp)
}

// setIsolatedSymbols adds the isolated margin symbols of a universal transfer to params
func (e *Exchange) setIsolatedSymbols(params url.Values, fromSymbol, toSymbol currency.Pair) error {
	if !fromSymbol.IsEmpty() {
		symbol, err := e.FormatSymbol(fromSymbol, asset.Margin)
		if err != nil {
			return err
		}
		params.Set("fromSymbol", symbol)
	}
	if !toSymbol.IsEmpty() {
		symbol, err := e.FormatSymbol(toSymbol, asset.Margin)
		if err != nil {
			return err
		}
		params.Set("toSymbol", symbol)
	}
	return nil
}

// GetUserWalletBalance returns the User Wallet Balance of each wallet, valued in quoteAsset (BTC when unset), with each
// wallet's asset balances when needBalanceDetail is true
func (e *Exchange) GetUserWalletBalance(ctx context.Context, quoteAsset currency.Code, needBalanceDetail bool) ([]UserWalletBalance, error) {
	params := url.Values{}
	if !quoteAsset.IsEmpty() {
		params.Set("quoteAsset", quoteAsset.Upper().String())
	}
	if needBalanceDetail {
		params.Set("needBalanceDetail", "true")
	}
	var resp []UserWalletBalance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/wallet/balance", params, getUserWalletBalanceRate, &resp)
}

// ToggleBNBBurn calls Toggle BNB Burn On Spot Trade And Margin Interest, setting whether BNB pays spot trading fees,
// margin loan interest or both; a nil setting is left as it is
func (e *Exchange) ToggleBNBBurn(ctx context.Context, spotBNBBurn, interestBNBBurn *bool) (*BNBBurnStatusResponse, error) {
	if spotBNBBurn == nil && interestBNBBurn == nil {
		return nil, errBNBBurnSettingRequired
	}
	params := url.Values{}
	if spotBNBBurn != nil {
		params.Set("spotBNBBurn", strconv.FormatBool(*spotBNBBurn))
	}
	if interestBNBBurn != nil {
		params.Set("interestBNBBurn", strconv.FormatBool(*interestBNBBurn))
	}
	var resp *BNBBurnStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/bnbBurn", params, sapiDefaultRate, &resp)
}

// GetTradeFees returns the Trade Fee of every spot symbol, or of symbol when it is set
func (e *Exchange) GetTradeFees(ctx context.Context, symbol currency.Pair) ([]TradeFee, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []TradeFee
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/tradeFee", params, sapiDefaultRate, &resp)
}

// GetUserAssets returns the User Asset balances: every positive balance, or ccy's when it is set
func (e *Exchange) GetUserAssets(ctx context.Context, ccy currency.Code, needBTCValuation bool) ([]UserAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []UserAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v3/asset/getUserAsset", params, userAssetsRate, &resp)
}

// GetAllCoinsInfo returns All Coins' Information: balances and the deposit and withdrawal networks of each coin
func (e *Exchange) GetAllCoinsInfo(ctx context.Context) ([]CoinInfo, error) {
	var resp []CoinInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/config/getall", nil, allCoinInfoRate, &resp)
}

// GetDepositAddressForCurrency returns the Deposit Address of a coin on network, or on the coin's default network when
// network is empty. amount is required for the LIGHTNING network
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, coin currency.Code, network string, amount float64) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	if network != "" {
		params.Set("network", network)
	}
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/address", params, depositAddressesRate, &resp)
}

// DepositHistory returns the Deposit History. The time window cannot exceed 90 days and defaults to the last 90 days
func (e *Exchange) DepositHistory(ctx context.Context, req *DepositHistoryRequest) ([]DepositRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
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
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.TxID != "" {
		params.Set("txId", req.TxID)
	}
	var resp []DepositRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/hisrec", params, sapiDefaultRate, &resp)
}

// GetDepositAddressListWithNetwork returns the deposit address list of a coin on network, or on the coin's default
// network when network is empty
func (e *Exchange) GetDepositAddressListWithNetwork(ctx context.Context, coin currency.Code, network string) ([]DepositAddressWithNetwork, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	if network != "" {
		params.Set("network", network)
	}
	var resp []DepositAddressWithNetwork
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/address/list", params, getDepositAddressListInNetworkRate, &resp)
}

// GetWithdrawAddressList returns the withdraw address list of the address book
func (e *Exchange) GetWithdrawAddressList(ctx context.Context) ([]WithdrawAddress, error) {
	var resp []WithdrawAddress
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/withdraw/address/list", nil, withdrawAddressListRate, &resp)
}

// OneClickArrivalDepositApply calls One click arrival deposit apply, crediting a deposit made to an expired address
func (e *Exchange) OneClickArrivalDepositApply(ctx context.Context, req *DepositCreditApplyRequest) (*DepositCreditApplyResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.DepositID != 0 {
		params.Set("depositId", strconv.FormatUint(req.DepositID, 10))
	}
	if req.TxID != "" {
		params.Set("txId", req.TxID)
	}
	if req.SubAccountID != "" {
		params.Set("subAccountId", req.SubAccountID)
	}
	if req.SubUserID != 0 {
		params.Set("subUserId", strconv.FormatUint(req.SubUserID, 10))
	}
	var resp *DepositCreditApplyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/capital/deposit/credit-apply", params, sapiDefaultRate, &resp)
}

// WithdrawCrypto calls Withdraw, submitting a withdrawal. Local entities that require the travel rule cannot use it
func (e *Exchange) WithdrawCrypto(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Address == "" {
		return nil, errAddressRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	params.Set("address", req.Address)
	if req.AddressTag != "" {
		params.Set("addressTag", req.AddressTag)
	}
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.TransactionFeeFlag {
		params.Set("transactionFeeFlag", "true")
	}
	if req.Name != "" {
		params.Set("name", req.Name)
	}
	if req.WalletType != nil {
		params.Set("walletType", strconv.FormatUint(*req.WalletType, 10))
	}
	var resp *WithdrawResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/capital/withdraw/apply", params, fundWithdrawalRate, &resp)
}

// WithdrawHistory returns the Withdraw History. The time window cannot exceed 90 days and defaults to the last 90 days,
// or to the last 7 days when a withdraw order ID is set
func (e *Exchange) WithdrawHistory(ctx context.Context, req *WithdrawHistoryRequest) ([]WithdrawalRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	if req.Status != nil {
		params.Set("status", strconv.FormatUint(*req.Status, 10))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if len(req.IDList) > 0 {
		params.Set("idList", strings.Join(req.IDList, ","))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp []WithdrawalRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/withdraw/history", params, withdrawalHistoryRate, &resp)
}

// GetSymbolsDelistScheduleForSpot returns the Spot Delist Schedule
func (e *Exchange) GetSymbolsDelistScheduleForSpot(ctx context.Context) ([]DelistSchedule, error) {
	var resp []DelistSchedule
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/spot/delist-schedule", symbolDelistScheduleForSpotRate, &resp)
}

// GetSystemStatus returns the System Status
func (e *Exchange) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	var resp *SystemStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/sapi/v1/system/status", sapiDefaultRate, &resp)
}

// GetLocalEntitiesDepositHistory returns the Deposit History Travel Rule of local entities that require the travel
// rule. The time window cannot exceed 90 days and defaults to the last 90 days
func (e *Exchange) GetLocalEntitiesDepositHistory(ctx context.Context, req *LocalEntityDepositHistoryRequest) ([]LocalEntityDepositRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(req.TravelRuleRecordIDs) > 0 {
		params.Set("trId", joinUint64s(req.TravelRuleRecordIDs))
	}
	if len(req.TxIDs) > 0 {
		params.Set("txId", strings.Join(req.TxIDs, ","))
	}
	if len(req.TransactionIDs) > 0 {
		params.Set("tranId", joinUint64s(req.TransactionIDs))
	}
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.TravelRuleStatus != nil {
		params.Set("travelRuleStatus", strconv.FormatUint(*req.TravelRuleStatus, 10))
	}
	if req.PendingQuestionnaire {
		params.Set("pendingQuestionnaire", "true")
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []LocalEntityDepositRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/localentity/deposit/history", params, sapiDefaultRate, &resp)
}

// SubmitDepositQuestionnaire calls Submit Deposit Questionnaire for a deposit of a local entity that requires the travel
// rule. Only deposits from unhosted wallets or VASPs not yet onboarded with GTR need one. The questionnaire's answers,
// which differ by local entity, are sent JSON encoded
func (e *Exchange) SubmitDepositQuestionnaire(ctx context.Context, transactionID uint64, questionnaire map[string]any) (*DepositQuestionnaireResponse, error) {
	if transactionID == 0 {
		return nil, errTransactionIDRequired
	}
	if len(questionnaire) == 0 {
		return nil, errQuestionnaireRequired
	}
	questionnaireJSON, err := json.Marshal(questionnaire)
	if err != nil {
		return nil, fmt.Errorf("error encoding deposit questionnaire: %w", err)
	}
	params := url.Values{}
	params.Set("tranId", strconv.FormatUint(transactionID, 10))
	params.Set("questionnaire", string(questionnaireJSON))
	var resp *DepositQuestionnaireResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPut, "/sapi/v1/localentity/deposit/provide-info", params, depositQuestionnaireRate, &resp)
}

// GetOnboardedVASPList returns the VASP list of local entities that require the travel rule
func (e *Exchange) GetOnboardedVASPList(ctx context.Context) ([]VASP, error) {
	var resp []VASP
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/localentity/vasp", nil, sapiDefaultRate, &resp)
}

// WithdrawalHistoryV1 returns the Withdraw History Travel Rule of local entities that require the travel rule. The time
// window cannot exceed 90 days and defaults to the last 90 days
func (e *Exchange) WithdrawalHistoryV1(ctx context.Context, req *LocalEntityWithdrawHistoryRequest) ([]LocalEntityWithdrawalRecord, error) {
	return e.localEntityWithdrawHistory(ctx, "/sapi/v1/localentity/withdraw/history", req)
}

// WithdrawalHistoryV2 returns the Withdraw History V2 of local entities that require the travel rule, which may omit
// withdrawals made through WithdrawCrypto. The time window cannot exceed 90 days and defaults to the last 90 days, or
// to the last 7 days when a withdraw order ID is set
func (e *Exchange) WithdrawalHistoryV2(ctx context.Context, req *LocalEntityWithdrawHistoryRequest) ([]LocalEntityWithdrawalRecord, error) {
	return e.localEntityWithdrawHistory(ctx, "/sapi/v2/localentity/withdraw/history", req)
}

func (e *Exchange) localEntityWithdrawHistory(ctx context.Context, path string, req *LocalEntityWithdrawHistoryRequest) ([]LocalEntityWithdrawalRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(req.TravelRuleRecordIDs) > 0 {
		params.Set("trId", joinUint64s(req.TravelRuleRecordIDs))
	}
	if len(req.TxIDs) > 0 {
		params.Set("txId", strings.Join(req.TxIDs, ","))
	}
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.TravelRuleStatus != nil {
		params.Set("travelRuleStatus", strconv.FormatUint(*req.TravelRuleStatus, 10))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp []LocalEntityWithdrawalRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, params, sapiDefaultRate, &resp)
}

// GetFutureTickLevelOrderbookHistoricalDataDownloadLink returns the download links of a futures symbol's tick-level
// order book data. The endpoint is no longer documented
func (e *Exchange) GetFutureTickLevelOrderbookHistoricalDataDownloadLink(ctx context.Context, req *HistoricalDataDownloadLinkRequest) (*HistoricalDataDownloadLinkResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.DataType == "" {
		return nil, errHistoricalDataTypeRequired
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("dataType", req.DataType)
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp *HistoricalDataDownloadLinkResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/futures/histDataLink", params, sapiDefaultRate, &resp)
}

// joinUint64s formats IDs as the comma separated list Binance takes
func joinUint64s(ids []uint64) string {
	formatted := make([]string, len(ids))
	for i := range ids {
		formatted[i] = strconv.FormatUint(ids[i], 10)
	}
	return strings.Join(formatted, ",")
}

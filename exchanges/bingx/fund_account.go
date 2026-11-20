package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errTransferTypeOrIDRequired = errors.New("transfer type or transaction ID is required")
	errTransferAccountRequired  = errors.New("transfer from and to accounts are required")
	errUserAccountTypeRequired  = errors.New("user account type is required")
	errUserAccountRequired      = errors.New("user account is required")
	errWalletTypeRequired       = errors.New("wallet type is required")
	errCallingCodeRequired      = errors.New("calling code is required for a phone number")
)

// GetSpotAccountAssets calls Query Assets, returning the spot account's balance of each asset
func (e *Exchange) GetSpotAccountAssets(ctx context.Context) (*AssetBalancesResponse, error) {
	var resp *AssetBalancesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/spot/v1/account/balance", nil, spotBalanceRate, &resp)
}

// GetAssetTransferRecords returns the asset transfers of a transfer type, or the transfer with a transaction ID
func (e *Exchange) GetAssetTransferRecords(ctx context.Context, req *AssetTransferRecordsRequest) (*AssetTransferRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" && req.TransactionID == 0 {
		return nil, errTransferTypeOrIDRequired
	}
	params := url.Values{}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	if req.TransactionID != 0 {
		params.Set("tranId", strconv.FormatUint(req.TransactionID, 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Current != 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *AssetTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/api/v3/asset/transfer", params, assetTransferRecordsRate, &resp)
}

// MainAccountInternalTransfer transfers a coin from the main account to another main account or to one of its
// sub-accounts. The API key needs the withdraw permission
func (e *Exchange) MainAccountInternalTransfer(ctx context.Context, req *InternalTransferRequest) (*InternalTransferResponse, error) {
	body, err := fundAccountInternalTransferBody(req)
	if err != nil {
		return nil, err
	}
	var resp *InternalTransferResponse
	return resp, e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/wallets/v1/capital/innerTransfer/apply", body, mainAccountInternalTransferRate, &resp)
}

// fundAccountInternalTransferBody validates an internal transfer and returns the body to send, with its coin upper case
func fundAccountInternalTransferBody(req *InternalTransferRequest) (*InternalTransferRequest, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.UserAccountType == 0 {
		return nil, errUserAccountTypeRequired
	}
	if req.UserAccount == "" {
		return nil, errUserAccountRequired
	}
	if req.UserAccountType == UserAccountTypePhoneNumber && req.CallingCode == "" {
		return nil, errCallingCodeRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.WalletType == 0 {
		return nil, errWalletTypeRequired
	}
	body := *req
	body.Coin = req.Coin.Upper()
	return &body, nil
}

// AssetTransfer calls Asset Transfer New, transferring an asset between two of the account's own accounts. The API key
// needs the universal transfer permission
func (e *Exchange) AssetTransfer(ctx context.Context, req *AssetTransferRequest) (*AssetTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromAccount == "" || req.ToAccount == "" {
		return nil, errTransferAccountRequired
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromAccount", string(req.FromAccount))
	params.Set("toAccount", string(req.ToAccount))
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	var resp *AssetTransferResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/api/asset/v1/transfer", params, assetTransferRate, &resp)
}

// GetTransferableCurrencies returns the coins that can be transferred from one of the account's own accounts to
// another, with the amount of each
func (e *Exchange) GetTransferableCurrencies(ctx context.Context, fromAccount, toAccount TransferAccount) (*TransferableCurrenciesResponse, error) {
	if fromAccount == "" || toAccount == "" {
		return nil, errTransferAccountRequired
	}
	params := url.Values{}
	params.Set("fromAccount", string(fromAccount))
	params.Set("toAccount", string(toAccount))
	var resp *TransferableCurrenciesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/api/asset/v1/transfer/supportCoins", params, transferableCurrenciesRate, &resp)
}

// GetAssetTransferRecordsNew returns the transfers from one of the account's own accounts to another, or the transfer
// with a transaction ID
func (e *Exchange) GetAssetTransferRecordsNew(ctx context.Context, req *AssetTransferRecordsNewRequest) (*AssetTransferRecordsNewResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TransactionID == "" && (req.FromAccount == "" || req.ToAccount == "") {
		return nil, fmt.Errorf("%w without a transaction ID", errTransferAccountRequired)
	}
	params := url.Values{}
	if req.FromAccount != "" {
		params.Set("fromAccount", string(req.FromAccount))
	}
	if req.ToAccount != "" {
		params.Set("toAccount", string(req.ToAccount))
	}
	if req.TransactionID != "" {
		params.Set("tranId", req.TransactionID)
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *AssetTransferRecordsNewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/api/v3/asset/transferRecord", params, assetTransferRecordsNewRate, &resp)
}

// GetFundAccountAssets returns the fund account's balance of each asset, or of ccy when it is set
func (e *Exchange) GetFundAccountAssets(ctx context.Context, ccy currency.Code) (*AssetBalancesResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	var resp *AssetBalancesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/fund/v1/account/balance", params, fundAccountBalanceRate, &resp)
}

// GetMainAccountInternalTransferRecords returns the main account's internal transfers of a coin
func (e *Exchange) GetMainAccountInternalTransferRecords(ctx context.Context, req *InternalTransferRecordsRequest) (*InternalTransferRecordsResponse, error) {
	params, err := fundAccountInternalTransferRecordsParams(req)
	if err != nil {
		return nil, err
	}
	var resp *InternalTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/innerTransfer/records", params, mainAccountInternalTransferRecordsRate, &resp)
}

// fundAccountInternalTransferRecordsParams builds the parameters of the main account and sub-account internal transfer
// record queries
func fundAccountInternalTransferRecordsParams(req *InternalTransferRecordsRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.ID != "" {
		params.Set("id", req.ID)
	}
	if req.TransferClientID != "" {
		params.Set("transferClientId", req.TransferClientID)
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
	return params, nil
}

// GetAssetOverview returns the USDT value of each of the account's accounts, or of the accountType account when it is
// set
func (e *Exchange) GetAssetOverview(ctx context.Context, accountType AccountType) (*AssetOverviewResponse, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", string(accountType))
	}
	var resp *AssetOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/account/v1/allAccountBalance", params, assetOverviewRate, &resp)
}

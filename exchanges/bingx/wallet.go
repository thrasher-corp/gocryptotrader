package bingx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetDepositRecords returns the account's deposits
func (e *Exchange) GetDepositRecords(ctx context.Context, req *DepositRecordsRequest) ([]DepositRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
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
	var resp []DepositRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/api/v3/capital/deposit/hisrec", params, depositRecordsRate, &resp)
}

// GetWithdrawRecords returns the account's withdrawals
func (e *Exchange) GetWithdrawRecords(ctx context.Context, req *WithdrawRecordsRequest) ([]WithdrawRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.ID != "" {
		params.Set("id", req.ID)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	if req.Status != 0 {
		params.Set("status", strconv.FormatUint(uint64(req.Status), 10))
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
	var resp []WithdrawRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/api/v3/capital/withdraw/history", params, withdrawRecordsRate, &resp)
}

// GetCurrencyConfig calls Query currency deposit and withdrawal data, returning the deposit and withdrawal configuration
// of every coin on each of its networks, or of the coin or display name given
func (e *Exchange) GetCurrencyConfig(ctx context.Context, coin currency.Code, displayName string) ([]CurrencyConfigResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	if displayName != "" {
		params.Set("displayName", displayName)
	}
	var resp []CurrencyConfigResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/config/getall", params, currencyConfigRate, &resp)
}

// Withdraw withdraws a coin to an address. The API key needs the withdraw permission
func (e *Exchange) Withdraw(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.WalletType == 0 {
		return nil, errWalletTypeRequired
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	params.Set("address", req.Address)
	if req.AddressTag != "" {
		params.Set("addressTag", req.AddressTag)
	}
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("walletType", strconv.FormatUint(uint64(req.WalletType), 10))
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	if req.VASPEntityID != "" {
		params.Set("vaspEntityId", string(req.VASPEntityID))
	}
	if req.RecipientLastName != "" {
		params.Set("recipientLastName", req.RecipientLastName)
	}
	if req.RecipientFirstName != "" {
		params.Set("recipientFirstName", req.RecipientFirstName)
	}
	if !req.DateOfBirth.IsZero() {
		// A date of birth is a calendar date, so it keeps the location it was given in
		params.Set("dateOfbirth", req.DateOfBirth.Format(time.DateOnly))
	}
	var resp *WithdrawResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/wallets/v1/capital/withdraw/apply", params, withdrawRate, &resp)
}

// GetMainAccountDepositAddress returns the main account's deposit address of a coin on each of its networks
func (e *Exchange) GetMainAccountDepositAddress(ctx context.Context, req *MainAccountDepositAddressRequest) (*DepositAddressListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *DepositAddressListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/deposit/address", params, depositAddressRate, &resp)
}

// GetDepositRiskControlRecords returns the deposits of the account and its sub-accounts that risk control holds
func (e *Exchange) GetDepositRiskControlRecords(ctx context.Context) ([]DepositRiskControlRecordResponse, error) {
	var resp []DepositRiskControlRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wallets/v1/capital/deposit/riskRecords", nil, depositRiskRecordsRate, &resp)
}

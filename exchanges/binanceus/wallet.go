package binanceus

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errAddressRequired        = errors.New("address required")
	errNetworkRequired        = errors.New("network required")
	errPaymentAccountRequired = errors.New("payment account required")
	errPaymentMethodRequired  = errors.New("payment method required")
)

// GetAssetFeesAndWalletStatus returns every crypto asset's balances and, for each of its networks, whether deposits
// and withdrawals are enabled and the withdrawal fee and limits
func (e *Exchange) GetAssetFeesAndWalletStatus(ctx context.Context) ([]CoinInfoResponse, error) {
	var resp []CoinInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/config/getall", nil, defaultRate, &resp)
}

// WithdrawFiat submits a USD withdrawal to a linked payment account
func (e *Exchange) WithdrawFiat(ctx context.Context, req *WithdrawFiatRequest) (*WithdrawFiatResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PaymentMethod == "" {
		return nil, errPaymentMethodRequired
	}
	if req.PaymentAccount == "" {
		return nil, errPaymentAccountRequired
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("paymentMethod", string(req.PaymentMethod))
	params.Set("paymentAccount", req.PaymentAccount)
	if !req.FiatCurrency.IsEmpty() {
		params.Set("fiatCurrency", req.FiatCurrency.Upper().String())
	}
	params.Set("amount", formatFloat(req.Amount))
	var resp *WithdrawFiatResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/fiatpayment/withdraw/apply", params, defaultRate, &resp)
}

// WithdrawCrypto submits a crypto withdrawal
func (e *Exchange) WithdrawCrypto(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.Address == "" {
		return nil, errAddressRequired
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	params.Set("network", req.Network)
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
	}
	params.Set("address", req.Address)
	if req.AddressTag != "" {
		params.Set("addressTag", req.AddressTag)
	}
	params.Set("amount", formatFloat(req.Amount))
	var resp *WithdrawResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/capital/withdraw/apply", params, defaultRate, &resp)
}

// WithdrawalHistory returns the account's crypto withdrawals of a coin, by default those of the last 90 days
func (e *Exchange) WithdrawalHistory(ctx context.Context, req *WithdrawalHistoryRequest) ([]WithdrawalRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
	if req.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", req.WithdrawOrderID)
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
	var resp []WithdrawalRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/withdraw/history", params, defaultRate, &resp)
}

// FiatWithdrawalHistory returns the account's fiat withdrawals, by default those of the last 90 days
func (e *Exchange) FiatWithdrawalHistory(ctx context.Context, req *FiatHistoryRequest) (*FiatHistoryResponse, error) {
	return e.fiatHistory(ctx, "/sapi/v1/fiatpayment/query/withdraw/history", req)
}

// fiatHistory returns the fiat withdrawals or deposits a fiat history path holds that match a request's filters
func (e *Exchange) fiatHistory(ctx context.Context, path string, req *FiatHistoryRequest) (*FiatHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.FiatCurrency.IsEmpty() {
		params.Set("fiatCurrency", req.FiatCurrency.Upper().String())
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.PaymentChannel != "" {
		params.Set("paymentChannel", req.PaymentChannel)
	}
	if req.PaymentMethod != "" {
		params.Set("paymentMethod", string(req.PaymentMethod))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	var resp *FiatHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, params, defaultRate, &resp)
}

// GetDepositAddressForCurrency returns a deposit address of a coin on a network, or on the coin's default network when
// network is empty
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, coin currency.Code, network string) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	if network != "" {
		params.Set("network", network)
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/address", params, defaultRate, &resp)
}

// DepositHistory returns the account's crypto deposits of a coin, by default those of the last 90 days
func (e *Exchange) DepositHistory(ctx context.Context, req *DepositHistoryRequest) ([]DepositRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", req.Coin.Upper().String())
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
	var resp []DepositRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/hisrec", params, defaultRate, &resp)
}

// FiatDepositHistory returns the account's fiat deposits, by default those of the last 90 days
func (e *Exchange) FiatDepositHistory(ctx context.Context, req *FiatHistoryRequest) (*FiatHistoryResponse, error) {
	return e.fiatHistory(ctx, "/sapi/v1/fiatpayment/query/deposit/history", req)
}

// GetSubAccountDepositAddress returns a sub-account's deposit address of a coin on a network, or on the coin's default
// network when no network is set
func (e *Exchange) GetSubAccountDepositAddress(ctx context.Context, req *SubAccountDepositAddressRequest) (*DepositAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Email == "" {
		return nil, errEmailRequired
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("email", req.Email)
	params.Set("coin", req.Coin.Upper().String())
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/sub-account/deposit/address", params, defaultRate, &resp)
}

// GetSubAccountDepositHistory returns a sub-account's crypto deposits
func (e *Exchange) GetSubAccountDepositHistory(ctx context.Context, req *SubAccountDepositHistoryRequest) ([]DepositRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Email == "" {
		return nil, errEmailRequired
	}
	params := url.Values{}
	params.Set("email", req.Email)
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
	var resp []DepositRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/sub-account/deposit/history", params, defaultRate, &resp)
}

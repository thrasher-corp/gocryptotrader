package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Withdrawal request validation errors
var (
	errWithdrawalIDRequired = errors.New("withdrawal ID is required")
	errWithdrawTypeRequired = errors.New("withdraw type is required")
)

// GetWithdrawalQuotas returns a currency's withdrawal quota, optionally on one chain (Get Withdrawal Quotas)
func (e *Exchange) GetWithdrawalQuotas(ctx context.Context, ccy currency.Code, chain string) (*WithdrawalQuotasResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *WithdrawalQuotasResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getWithdrawalQuotasEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals/quotas", params), nil, &resp)
}

// WithdrawV3 withdraws a currency to an address, UID, email or phone number. When favourite withdrawal addresses are
// enforced on the web, a destination that is not one is rejected with code 260325 (Withdraw (V3))
func (e *Exchange) WithdrawV3(ctx context.Context, arg *WithdrawV3Request) (*WithdrawalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.ToAddress == "" {
		return nil, errAddressRequired
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.WithdrawType == "" {
		return nil, errWithdrawTypeRequired
	}
	body := make(map[string]any)
	body["currency"] = arg.Currency.Upper().String()
	body["toAddress"] = arg.ToAddress
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["withdrawType"] = arg.WithdrawType
	if arg.Chain != "" {
		body["chain"] = arg.Chain
	}
	if arg.Memo != "" {
		body["memo"] = arg.Memo
	}
	if arg.IsInner {
		body["isInner"] = true
	}
	if arg.Remark != "" {
		body["remark"] = arg.Remark
	}
	if arg.FeeDeductType != "" {
		body["feeDeductType"] = arg.FeeDeductType
	}
	var resp *WithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, withdrawV3EPL, http.MethodPost, "/v3/withdrawals", body, &resp)
}

// CancelWithdrawal cancels a withdrawal that is still PROCESSING (Cancel Withdrawal)
func (e *Exchange) CancelWithdrawal(ctx context.Context, withdrawalID string) error {
	if withdrawalID == "" {
		return errWithdrawalIDRequired
	}
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodDelete, "/v1/withdrawals/"+withdrawalID, nil, &resp)
}

// GetWithdrawalHistory returns a page of a currency's withdrawals, latest first (Get Withdrawal History)
func (e *Exchange) GetWithdrawalHistory(ctx context.Context, arg *WithdrawalHistoryRequest) (*WithdrawalHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", arg.Currency.Upper().String())
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if err := setTimeRange(params, "startAt", "endAt", arg.StartAt, arg.EndAt); err != nil {
		return nil, err
	}
	if arg.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(arg.CurrentPage, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *WithdrawalHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals", params), nil, &resp)
}

// GetWithdrawalHistoryByID returns a withdrawal (Get Withdrawal History By ID)
func (e *Exchange) GetWithdrawalHistoryByID(ctx context.Context, withdrawalID string) (*WithdrawalHistoryByIDResponse, error) {
	if withdrawalID == "" {
		return nil, errWithdrawalIDRequired
	}
	var resp *WithdrawalHistoryByIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryByIDEPL, http.MethodGet, "/v1/withdrawals/"+withdrawalID, nil, &resp)
}

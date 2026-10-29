package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetUTAWithdrawalQuotas returns a currency's withdrawal availability, quotas, fees and limits on a chain (Get
// Withdrawal Quotas)
func (e *Exchange) GetUTAWithdrawalQuotas(ctx context.Context, req *UTAWithdrawalQuotasRequest) (*UTAWithdrawalQuotasResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.WithdrawType == "" {
		return nil, errWithdrawTypeRequired
	}
	params := url.Values{}
	if req.Chain != "" {
		params.Set("chain", req.Chain)
	}
	params.Set("currency", req.Currency.Upper().String())
	if req.IsInner {
		params.Set("isInner", "true")
	}
	params.Set("withdrawType", req.WithdrawType)
	var resp *UTAWithdrawalQuotasResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAWithdrawalQuotasEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/withdrawals/quotas", params), nil, &resp)
}

// GetUTAWithdrawalHistory returns a page of withdrawal records (Get Withdrawal History)
func (e *Exchange) GetUTAWithdrawalHistory(ctx context.Context, req *UTAAssetHistoryRequest) (*UTAWithdrawalHistoryResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp *UTAWithdrawalHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAWithdrawalHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/asset/withdrawal/history", params), nil, &resp)
}

// UTAWithdraw submits a withdrawal of a currency to an address or, internally, to a UID, email or phone (Withdraw)
func (e *Exchange) UTAWithdraw(ctx context.Context, req *UTAWithdrawRequest) (*UTAWithdrawResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.ToAddress == "" {
		return nil, errAddressRequired
	}
	if req.WithdrawType == "" {
		return nil, errWithdrawTypeRequired
	}
	body := make(map[string]any)
	body["currency"] = req.Currency.Upper().String()
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["toAddress"] = req.ToAddress
	body["withdrawType"] = req.WithdrawType
	if req.Chain != "" {
		body["chain"] = req.Chain
	}
	if req.Memo != "" {
		body["memo"] = req.Memo
	}
	if req.Remark != "" {
		body["remark"] = req.Remark
	}
	if req.IsInner {
		body["isInner"] = true
	}
	if req.FeeDeductType != "" {
		body["feeDeductType"] = req.FeeDeductType
	}
	var resp *UTAWithdrawResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, utaWithdrawEPL, http.MethodPost, "/ua/v2/asset/withdrawal", body, &resp)
}

// CancelUTAWithdrawal cancels a withdrawal while it is still cancellable (Cancel Withdrawal)
func (e *Exchange) CancelUTAWithdrawal(ctx context.Context, withdrawID string) (*CancelUTAWithdrawalResponse, error) {
	if withdrawID == "" {
		return nil, errWithdrawalIDRequired
	}
	var resp *CancelUTAWithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelUTAWithdrawalEPL, http.MethodPost, "/ua/v2/asset/withdraw/cancel", map[string]any{"withdrawId": withdrawID}, &resp)
}

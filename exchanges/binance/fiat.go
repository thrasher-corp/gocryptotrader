package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFiatDepositAndWithdrawalHistory calls Get Fiat Deposit/Withdraw History, where transaction type 0 is deposits
// and 1 withdrawals; without a window the API returns the last 30 days
func (e *Exchange) GetFiatDepositAndWithdrawalHistory(ctx context.Context, arg *FiatHistoryRequest) (*FiatOrdersResponse, error) {
	params, err := fiatHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *FiatOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/fiat/orders", params, fiatDepositWithdrawHistRate, &resp)
}

// GetFiatPaymentHistory calls Get Fiat Payments History, where transaction type 0 is buys and 1 sells; without a
// window the API returns the last 30 days
func (e *Exchange) GetFiatPaymentHistory(ctx context.Context, arg *FiatHistoryRequest) (*FiatPaymentsResponse, error) {
	params, err := fiatHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *FiatPaymentsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/fiat/payments", params, sapiDefaultRate, &resp)
}

// fiatHistoryParams builds the parameters both fiat history queries share
func fiatHistoryParams(arg *FiatHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TransactionType > 1 {
		return nil, fmt.Errorf("%w: %d, want 0 or 1", errInvalidTransactionType, arg.TransactionType)
	}
	if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("transactionType", strconv.FormatUint(arg.TransactionType, 10))
	if !arg.BeginTime.IsZero() {
		params.Set("beginTime", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Rows > 0 {
		params.Set("rows", strconv.FormatUint(arg.Rows, 10))
	}
	return params, nil
}

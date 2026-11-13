package gate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WithdrawCurrency withdraws a currency. No fee is charged when the recipient address also belongs to Gate
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawalRequest) (*WithdrawalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: withdrawal amount must be greater than zero", order.ErrAmountIsInvalid)
	}
	if arg.Currency.IsEmpty() {
		return nil, fmt.Errorf("%w: withdrawal currency must be specified", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Chain == "" {
		return nil, errInvalidCurrencyChain
	}
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawEPL, http.MethodPost, "withdrawals", nil, arg, &resp)
}

// UIDTransfer transfers funds between the spot accounts of two main accounts by UID. Neither party can be a sub-account
func (e *Exchange) UIDTransfer(ctx context.Context, arg *UIDTransferRequest) (*UIDTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.RecipientUserID == 0 {
		return nil, fmt.Errorf("%w: recipient UID", errUserIDRequired)
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *UIDTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletUIDTransferEPL, http.MethodPost, "withdrawals/push", nil, arg, &resp)
}

// CancelWithdrawalWithSpecifiedID requests the cancellation of a withdrawal. The returned record status shows whether
// the cancellation took effect
func (e *Exchange) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	if withdrawalID == "" {
		return nil, errMissingWithdrawalID
	}
	var response *WithdrawalResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletCancelWithdrawEPL, http.MethodDelete, "withdrawals/"+withdrawalID, nil, nil, &response)
}

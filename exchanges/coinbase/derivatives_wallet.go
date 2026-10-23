package coinbase

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Gateway transfer types
const (
	DerivativesTransferTypeUser       = "user"
	DerivativesTransferTypeSubaccount = "subaccount"
)

// Gateway transfer directions
const (
	DerivativesTransferDirectionPayment = "payment"
	DerivativesTransferDirectionIncome  = "income"
)

// Gateway transfer states; a transfer the gateway rejects reports the reason in place of a state
const (
	DerivativesTransferStatePrepared          = "prepared"
	DerivativesTransferStateConfirmed         = "confirmed"
	DerivativesTransferStateCancelled         = "cancelled"
	DerivativesTransferStateWaitingForAdmin   = "waiting_for_admin"
	DerivativesTransferStateInsufficientFunds = "insufficient_funds"
	DerivativesTransferStateWithdrawalLimit   = "withdrawal_limit"
)

const (
	// derivativesMinimumTransferNonceLength and derivativesMaximumTransferNonceLength bound a transfer's idempotency
	// nonce's characters
	derivativesMinimumTransferNonceLength = 8
	derivativesMaximumTransferNonceLength = 128
)

var (
	errDerivativesTransferDestinationRequired = errors.New("gateway transfer destination required")
	errDerivativesInvalidTransferNonce        = errors.New("gateway transfer nonce must be 8 to 128 characters")
)

// SubmitTransferBetweenSubaccounts transfers funds between subaccounts, or a subaccount and the main account; it tops
// up an isolated margin subaccount's collateral, which the gateway otherwise allocates with each order. The
// transfer's state is not proof of completion, which an account summary or the portfolio channel confirms
func (g GlobalDerivativesAPI) SubmitTransferBetweenSubaccounts(ctx context.Context, req *DerivativesSubmitTransferBetweenSubaccountsRequest) (*DerivativesTransferResponse, error) {
	return g.submitTransferBetweenSubaccounts(ctx, false, req)
}

func (g GlobalDerivativesAPI) submitTransferBetweenSubaccounts(ctx context.Context, overWebsocket bool, req *DerivativesSubmitTransferBetweenSubaccountsRequest) (*DerivativesTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ccy, err := derivativesAccountCurrency(req.Currency, derivativesAccountCurrencies)
	if err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if req.Destination == 0 {
		return nil, errDerivativesTransferDestinationRequired
	}
	if err := validateDerivativesTransferNonce(req.Nonce); err != nil {
		return nil, err
	}
	params := &DerivativesSubmitTransferBetweenSubaccountsWireRequest{
		Currency:    ccy,
		Amount:      req.Amount,
		Destination: req.Destination,
		Source:      req.Source,
		Nonce:       req.Nonce,
	}
	var resp *DerivativesTransferResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/submit_transfer_between_subaccounts", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateDerivativesTransferNonce checks a transfer's idempotency nonce is left out or 8 to 128 characters long
func validateDerivativesTransferNonce(nonce string) error {
	if nonce == "" {
		return nil
	}
	if length := utf8.RuneCountInString(nonce); length < derivativesMinimumTransferNonceLength || length > derivativesMaximumTransferNonceLength {
		return fmt.Errorf("%w: %d characters", errDerivativesInvalidTransferNonce, length)
	}
	return nil
}

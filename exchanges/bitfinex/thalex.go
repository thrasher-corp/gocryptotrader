package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errInvalidThalexCurrency  = errors.New("currency must be USE, ETH or BTC for Thalex")
	errTwoFactorTokenRequired = errors.New("two factor token required")
)

// RequestThalexDeposit transfers funds from the account to its Thalex derivatives account, which Bitfinex validates
// with two factor authentication; two deposits every 24 hours are free
func (e *Exchange) RequestThalexDeposit(ctx context.Context, req *ThalexTransferRequest) (*ThalexTransfer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newThalexTransferBody(req)
	if err != nil {
		return nil, err
	}
	var resp *ThalexTransfer
	if err := e.SendAuthenticatedHTTPRequest(ctx, thalexDepositEPL, "auth/w/ext/wallets/deposits/request", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// RequestThalexWithdrawal transfers funds from the account's Thalex derivatives account to the account, which
// Bitfinex validates with two factor authentication
func (e *Exchange) RequestThalexWithdrawal(ctx context.Context, req *ThalexTransferRequest) (*ThalexTransfer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newThalexTransferBody(req)
	if err != nil {
		return nil, err
	}
	var resp *ThalexTransfer
	if err := e.SendAuthenticatedHTTPRequest(ctx, thalexWithdrawalEPL, "auth/w/ext/wallets/withdrawals/request", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetThalexFreeTransferCount returns how many free transfers to and from Thalex the account has left, and when they
// renew
func (e *Exchange) GetThalexFreeTransferCount(ctx context.Context) (*ThalexFreeTransferCountResponse, error) {
	// The docs name this provider Provider, unlike the transfers' provider, which the same service reads
	body := &thalexProviderBody{Provider: thalexProvider}
	var resp *ThalexFreeTransferCountResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, thalexFreeTransferCountEPL, "auth/r/ext/wallets/transfers/free/count", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// newThalexTransferBody returns the body of a transfer to or from Thalex, erroring for a transfer Bitfinex would refuse
func newThalexTransferBody(req *ThalexTransferRequest) (*thalexTransferBody, error) {
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	ccy := req.Currency.Upper().String()
	switch ccy {
	case "USE", "ETH", "BTC":
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidThalexCurrency, ccy)
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if req.TwoFactorToken == nil {
		return nil, errTwoFactorTokenRequired
	}
	if err := req.TwoFactorToken.validate(); err != nil {
		return nil, err
	}
	return &thalexTransferBody{
		Provider:       thalexProvider,
		Amount:         strconv.FormatFloat(req.Amount, 'f', -1, 64),
		Currency:       ccy,
		TwoFactorToken: req.TwoFactorToken,
	}, nil
}

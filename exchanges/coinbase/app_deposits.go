package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errAppPaymentMethodRequired = errors.New("payment method ID required")
	errAppTransferIDRequired    = errors.New("deposit or withdrawal ID required")
)

// DepositFunds deposits funds from a payment method into a fiat account. The deposit completes at once only with
// Commit set: since 11 April 2025 Coinbase leaves it uncommitted otherwise, without a transaction, until CommitDeposit
// commits it
func (a AppAPI) DepositFunds(ctx context.Context, req *AppFiatTransferRequest) (*AppTransfer, error) {
	return a.createFiatTransfer(ctx, "deposits", req)
}

// CommitDeposit completes a deposit DepositFunds left uncommitted
func (a AppAPI) CommitDeposit(ctx context.Context, req *AppFiatTransferIDRequest) (*AppTransfer, error) {
	return a.commitFiatTransfer(ctx, "deposits", req)
}

// ListDeposits returns a page of a fiat account's deposits; a crypto account's deposits are among its transactions
func (a AppAPI) ListDeposits(ctx context.Context, req *AppListFiatTransfersRequest) (*AppListResponse[AppFiatTransfer], error) {
	return a.listFiatTransfers(ctx, "deposits", req)
}

// GetDeposit returns one of a fiat account's deposits
func (a AppAPI) GetDeposit(ctx context.Context, req *AppFiatTransferIDRequest) (*AppFiatTransfer, error) {
	return a.getFiatTransfer(ctx, "deposits", req)
}

// createFiatTransfer starts a deposit or a withdrawal, sending its commit flag whatever its value
func (a AppAPI) createFiatTransfer(ctx context.Context, kind string, req *AppFiatTransferRequest) (*AppTransfer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.PaymentMethodID == "" {
		return nil, errAppPaymentMethodRequired
	}
	body := &appFiatTransferBody{
		Amount:        formatFloat(req.Amount),
		Currency:      req.Currency.Upper().String(),
		PaymentMethod: req.PaymentMethodID,
		Commit:        req.Commit,
	}
	return a.sendTransferRequest(ctx, "accounts/"+req.AccountID+"/"+kind, body)
}

// commitFiatTransfer completes an uncommitted deposit or withdrawal
func (a AppAPI) commitFiatTransfer(ctx context.Context, kind string, req *AppFiatTransferIDRequest) (*AppTransfer, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	return a.sendTransferRequest(ctx, "accounts/"+req.AccountID+"/"+kind+"/"+req.TransferID+"/commit", nil)
}

// listFiatTransfers returns a page of an account's deposits or withdrawals
func (a AppAPI) listFiatTransfers(ctx context.Context, kind string, req *AppListFiatTransfersRequest) (*AppListResponse[AppFiatTransfer], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	params, err := req.Page.params()
	if err != nil {
		return nil, err
	}
	resp, err := sendAppRequest[[]AppFiatTransfer](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/"+kind, params, nil)
	if err != nil {
		return nil, err
	}
	return listResponse(resp), nil
}

// getFiatTransfer returns one of an account's deposits or withdrawals
func (a AppAPI) getFiatTransfer(ctx context.Context, kind string, req *AppFiatTransferIDRequest) (*AppFiatTransfer, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	resp, err := sendAppRequest[*AppFiatTransfer](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/"+kind+"/"+req.TransferID, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// sendTransferRequest posts a deposit or withdrawal request and returns the transfer, which its response holds
// outside the data envelope the Coinbase App API's other responses use
func (a AppAPI) sendTransferRequest(ctx context.Context, path string, body any) (*AppTransfer, error) {
	var resp *appTransferResponse
	if err := a.e.SendAuthenticatedHTTPRequest(ctx, appEPL, http.MethodPost, appPath+path, nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Transfer == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Transfer, nil
}

// validate checks a deposit or withdrawal is named
func (r *AppFiatTransferIDRequest) validate() error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.AccountID == "" {
		return errAccountIDRequired
	}
	if r.TransferID == "" {
		return errAppTransferIDRequired
	}
	return nil
}

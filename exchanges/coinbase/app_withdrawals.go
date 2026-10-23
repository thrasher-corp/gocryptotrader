package coinbase

import "context"

// WithdrawFunds withdraws funds from a fiat account to a payment method. The withdrawal completes at once only with
// Commit set: since 11 April 2025 Coinbase leaves it uncommitted otherwise, without a transaction, until
// CommitWithdrawal commits it
func (a AppAPI) WithdrawFunds(ctx context.Context, req *AppFiatTransferRequest) (*AppTransfer, error) {
	return a.createFiatTransfer(ctx, "withdrawals", req)
}

// CommitWithdrawal completes a withdrawal WithdrawFunds left uncommitted
func (a AppAPI) CommitWithdrawal(ctx context.Context, req *AppFiatTransferIDRequest) (*AppTransfer, error) {
	return a.commitFiatTransfer(ctx, "withdrawals", req)
}

// ListWithdrawals returns a page of a fiat account's withdrawals
func (a AppAPI) ListWithdrawals(ctx context.Context, req *AppListFiatTransfersRequest) (*AppListResponse[AppFiatTransferResponse], error) {
	return a.listFiatTransfers(ctx, "withdrawals", req)
}

// GetWithdrawal returns one of a fiat account's withdrawals
func (a AppAPI) GetWithdrawal(ctx context.Context, req *AppFiatTransferIDRequest) (*AppFiatTransferResponse, error) {
	return a.getFiatTransfer(ctx, "withdrawals", req)
}

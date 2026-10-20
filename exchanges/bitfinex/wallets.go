package bitfinex

import (
	"context"
)

// GetWallets returns the account's exchange, margin and funding wallets
func (e *Exchange) GetWallets(ctx context.Context) ([]WalletResponse, error) {
	var resp []WalletResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, walletsEPL, "auth/r/wallets", nil, &resp)
}

package kraken

import (
	"context"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
)

// CreateSubaccount calls Create Subaccount, returning whether the trading subaccount was created. It needs a master
// account API key
func (e *Exchange) CreateSubaccount(ctx context.Context, username, email string) (bool, error) {
	if username == "" {
		return false, errSubaccountUsernameRequired
	}
	if email == "" {
		return false, errSubaccountEmailRequired
	}
	var created bool
	err := e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CreateSubaccount", nil, map[string]any{"username": username, "email": email}, &created)
	return created, err
}

// AccountTransfer calls Account Transfer, moving funds between the master account and its subaccounts. It needs a
// master account API key
func (e *Exchange) AccountTransfer(ctx context.Context, req *SubaccountTransferRequest) (*SubaccountTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrAmountBelowMin, req.Amount)
	}
	if req.FromAccountID == "" || req.ToAccountID == "" {
		return nil, errSubaccountAccountIDRequired
	}
	body := map[string]any{
		"asset":  req.Asset.Upper().String(),
		"amount": strconv.FormatFloat(req.Amount, 'f', -1, 64),
		"from":   req.FromAccountID,
		"to":     req.ToAccountID,
	}
	if req.AssetClass != "" {
		body["asset_class"] = req.AssetClass
	}
	var resp *SubaccountTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/AccountTransfer", nil, body, &resp)
}

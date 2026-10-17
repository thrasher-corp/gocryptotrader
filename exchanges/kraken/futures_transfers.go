package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// FuturesWalletTransfer calls Initiate wallet transfer, moving funds between two margin wallets with the same
// collateral currency, or between a margin wallet and the cash wallet
func (e *Exchange) FuturesWalletTransfer(ctx context.Context, req *FuturesWalletTransferRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	params, err := futuresTransferParams(req.FromWallet, req.ToWallet, req.Currency, req.Amount)
	if err != nil {
		return err
	}
	return e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/transfer", params, nil, nil)
}

// FuturesSubaccountTransfer calls Initiate sub account transfer, moving funds between the account and a subaccount, or
// between two wallets as FuturesWalletTransfer does
func (e *Exchange) FuturesSubaccountTransfer(ctx context.Context, req *FuturesSubaccountTransferRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.FromUser == "" || req.ToUser == "" {
		return errFuturesUserEmpty
	}
	params, err := futuresTransferParams(req.FromWallet, req.ToWallet, req.Currency, req.Amount)
	if err != nil {
		return err
	}
	params.Set("fromUser", req.FromUser)
	params.Set("toUser", req.ToUser)
	return e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/transfer/subaccount", params, nil, nil)
}

// FuturesWithdrawToSpotWallet calls Initiate withdrawal to Spot wallet, which needs a key with withdrawal access
func (e *Exchange) FuturesWithdrawToSpotWallet(ctx context.Context, req *FuturesWithdrawToSpotWalletRequest) (*FuturesWithdrawToSpotWalletResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrAmountBelowMin, req.Amount)
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Lower().String())
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.SourceWallet != "" {
		params.Set("sourceWallet", req.SourceWallet)
	}
	var resp *FuturesWithdrawToSpotWalletResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/withdrawal", params, nil, &resp)
}

// futuresTransferParams returns the wallet, currency and amount parameters the transfer endpoints share
func futuresTransferParams(fromWallet, toWallet string, ccy currency.Code, amount float64) (url.Values, error) {
	if fromWallet == "" || toWallet == "" {
		return nil, errFuturesWalletEmpty
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrAmountBelowMin, amount)
	}
	params := url.Values{}
	params.Set("fromAccount", fromWallet)
	params.Set("toAccount", toWallet)
	params.Set("unit", ccy.Lower().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return params, nil
}

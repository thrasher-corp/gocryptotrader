package deribit

import (
	"context"
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// wsHandleOrders relays a user.orders notification's orders
func (e *Exchange) wsHandleOrders(ctx context.Context, channel string, respRaw []byte) error {
	// Raw channels send each order as it changes, and the others a list of the orders changed over the interval
	var orders []Order
	if strings.HasSuffix(channel, "."+intervalRaw) {
		orders = make([]Order, 1)
		if err := decodeNotification(respRaw, &orders[0]); err != nil {
			return err
		}
	} else if err := decodeNotification(respRaw, &orders); err != nil {
		return err
	}
	details := make([]order.Detail, 0, len(orders))
	var errs error
	for i := range orders {
		detail, err := e.orderDetail(&orders[i])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		details = append(details, *detail)
	}
	if len(details) > 0 {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, details))
	}
	return errs
}

// wsHandlePortfolio saves a user.portfolio notification's balance as UpdateAccountBalances does, under each asset that
// holds the main account's balances, then relays the summary
func (e *Exchange) wsHandlePortfolio(ctx context.Context, respRaw []byte) error {
	summary := new(AccountSummary)
	if err := decodeNotification(respRaw, summary); err != nil {
		return err
	}
	assets, err := e.mainAccountAssets(ctx)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if _, err := e.Accounts.UpdateBalance(ctx, "", a, summary.Currency, func(b *accounts.Balance) {
			b.Total = summary.Balance
			b.Hold = max(summary.Balance-summary.AvailableWithdrawalFunds, 0)
			b.Free = summary.AvailableWithdrawalFunds
		}); err != nil {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, summary)
}

// mainAccountAssets returns the assets the main account's balances are saved under, or spot when none are saved yet.
// Deribit's balances back every asset, so a copy saved under another asset would be counted again in asset.All totals,
// and the portfolio manager saves the balances of an exchange without segregated asset accounts under spot
func (e *Exchange) mainAccountAssets(ctx context.Context) (asset.Items, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	subAccounts, err := e.Accounts.SubAccounts(creds, asset.All)
	if err != nil && !errors.Is(err, accounts.ErrNoSubAccounts) {
		return nil, err
	}
	var assets asset.Items
	for _, s := range subAccounts {
		if s.ID == "" {
			assets = append(assets, s.AssetType)
		}
	}
	if len(assets) == 0 {
		return asset.Items{asset.Spot}, nil
	}
	return assets, nil
}

// wsHandleUserTrades passes the account's trades to the fills feed
func (e *Exchange) wsHandleUserTrades(_ context.Context, respRaw []byte) error {
	var trades []UserTrade
	if err := decodeNotification(respRaw, &trades); err != nil {
		return err
	}
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	fills := make([]fill.Data, 0, len(trades))
	var errs error
	for i := range trades {
		data, err := e.userTradeFill(&trades[i])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		fills = append(fills, *data)
	}
	return common.AppendError(errs, e.Websocket.Fills.Update(fills...))
}

package deribit

import (
	"context"
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
	var orders []OrderResponse
	if strings.HasSuffix(channel, "."+intervalRaw) {
		orders = make([]OrderResponse, 1)
		if err := decodeNotification(respRaw, &orders[0]); err != nil {
			return err
		}
	} else if err := decodeNotification(respRaw, &orders); err != nil {
		return err
	}
	return e.relayOrders(ctx, orders)
}

// relayOrders relays the orders it can convert as order details, and returns the errors of those it cannot
func (e *Exchange) relayOrders(ctx context.Context, orders []OrderResponse) error {
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

// wsHandleUserChanges relays a user.changes notification's orders as user.orders does, passes its trades to the fills
// feed as user.trades does, then relays the notification for its positions
func (e *Exchange) wsHandleUserChanges(ctx context.Context, respRaw []byte) error {
	changes := new(WebsocketUserChanges)
	if err := decodeNotification(respRaw, changes); err != nil {
		return err
	}
	errs := e.relayOrders(ctx, changes.Orders)
	errs = common.AppendError(errs, e.updateFills(changes.Trades))
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, changes))
}

// wsHandlePortfolio saves a user.portfolio notification's balance as UpdateAccountBalances does, under spot, then
// relays the summary; each balance it saves is recorded, even when publishing the change fails, so an update requested
// before it keeps the notified balance over its reply's
func (e *Exchange) wsHandlePortfolio(ctx context.Context, respRaw []byte) error {
	summary := new(AccountSummary)
	if err := decodeNotification(respRaw, summary); err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.walletMu.Lock()
	before, beforeErr := e.Accounts.GetBalance("", creds, asset.Spot, summary.Currency)
	_, err = e.Accounts.UpdateBalance(ctx, "", asset.Spot, summary.Currency, func(b *accounts.Balance) {
		b.Total = summary.Balance
		b.Hold = max(summary.Balance-summary.AvailableWithdrawalFunds, 0)
		b.Free = summary.AvailableWithdrawalFunds
	})
	// Read back, as a save whose publication failed has still stored the balance, which an update requested before it
	// must keep
	if after, afterErr := e.Accounts.GetBalance("", creds, asset.Spot, summary.Currency); afterErr == nil && (beforeErr != nil || after != before) {
		e.recordWalletNotification(creds, summary.Currency)
	}
	e.walletMu.Unlock()
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, summary)
}

// wsHandleUserTrades passes the account's trades to the fills feed
func (e *Exchange) wsHandleUserTrades(_ context.Context, respRaw []byte) error {
	var trades []UserTradeResponse
	if err := decodeNotification(respRaw, &trades); err != nil {
		return err
	}
	return e.updateFills(trades)
}

// updateFills passes the account's trades it can convert to the fills feed when the feed is enabled, and returns the
// errors of those it cannot
func (e *Exchange) updateFills(trades []UserTradeResponse) error {
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

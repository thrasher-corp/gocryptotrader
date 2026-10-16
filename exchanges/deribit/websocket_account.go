package deribit

import (
	"context"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
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

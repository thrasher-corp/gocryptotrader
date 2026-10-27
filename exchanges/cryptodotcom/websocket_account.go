package cryptodotcom

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// The channels Crypto.com names for GoCryptoTrader's myOrders, myTrades and myAccount channels, and in their pushes
const (
	channelUserOrder   = "user.order"
	channelUserTrade   = "user.trade"
	channelUserBalance = "user.balance"
	// channelUserAdvancedOrder is the name some of Crypto.com's docs give the advanced order channel, whose subscriptions,
	// pushes and channel listings name it user.advance.order; a push naming it so is taken as an advanced order's
	channelUserAdvancedOrder = "user.advanced.order"
)

// userChannel returns the channel Crypto.com names for a user subscription's channel, erring for a channel Crypto.com
// does not serve, and for Crypto.com's names of the order, trade and balance channels, which are subscribed by
// GoCryptoTrader's
func userChannel(s *subscription.Subscription) (string, error) {
	switch s.Channel {
	case subscription.MyOrdersChannel:
		return channelUserOrder, nil
	case subscription.MyTradesChannel:
		return channelUserTrade, nil
	case subscription.MyAccountChannel:
		return channelUserBalance, nil
	case AdvancedOrdersChannel, TransactionsChannel, PositionsChannel, AccountRiskChannel, PositionBalanceChannel, OTCQuoteRequestsChannel, OTCQuotesChannel, OTCDealsChannel:
		return s.Channel, nil
	case channelUserOrder, channelUserTrade, channelUserBalance:
		return "", fmt.Errorf("%w: %q", subscription.ErrUseConstChannelName, s.Channel)
	default:
		return "", fmt.Errorf("%w: %q", subscription.ErrNotSupported, s.Channel)
	}
}

// userChannelByInstrument reports whether a user subscription names a channel for each instrument: an order or trade
// subscription for an asset's pairs does, and every other names one channel for the whole account
func userChannelByInstrument(s *subscription.Subscription, a asset.Item) (bool, error) {
	if _, err := userChannel(s); err != nil {
		return false, err
	}
	switch s.Channel {
	case subscription.MyOrdersChannel, AdvancedOrdersChannel, subscription.MyTradesChannel:
		return a != asset.Empty, nil
	default:
		return false, nil
	}
}

// userInstrumentChannel returns the channel Crypto.com names for a user subscription's channel of a pair of an asset
func (e *Exchange) userInstrumentChannel(s *subscription.Subscription, a asset.Item, p currency.Pair) (string, error) {
	channel, err := userChannel(s)
	if err != nil {
		return "", err
	}
	instrument, err := e.FormatSymbol(p, a)
	if err != nil {
		return "", err
	}
	return channel + "." + instrument, nil
}

// wsHandleUserOrders relays the account's orders, from the order and advanced order channels, as order details:
// subscribing returns the open orders, and later pushes carry orders as they change. An order whose detail cannot be
// told is reported, and the others relayed
func (e *Exchange) wsHandleUserOrders(ctx context.Context, push *websocketChannelPush) error {
	var orders []OrderResponse
	if err := decodePushData(push, &orders); err != nil {
		return err
	}
	details := make([]order.Detail, 0, len(orders))
	var errs error
	for i := range orders {
		d, err := e.userOrderDetail(&orders[i])
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s order %s: %w", push.Channel, orders[i].OrderID, err))
			continue
		}
		details = append(details, *d)
	}
	if len(details) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, details))
}

// userOrderDetail returns an order as the detail of the asset it trades as, which a spot instrument's order names in its
// execution instructions when it trades on margin, and tracks it as a margin order or not for its trades
func (e *Exchange) userOrderDetail(o *OrderResponse) (*order.Detail, error) {
	a := orderAsset(o.InstrumentName, o.ExecutionInstructions)
	e.trackMarginOrder(a, o.OrderID, o.ClientOrderID)
	p, err := pairFromInstrument(a, o.InstrumentName)
	if err != nil {
		return nil, err
	}
	return e.orderDetail(o, a, p)
}

// marginOrdersCapacity is how many orders the margin order cache tracks, by order ID and client order ID, before it
// forgets those least recently used
const marginOrdersCapacity = 10000

// marginOrderKey is an order's key in the margin order cache: its order ID, or its client order ID when client is set
type marginOrderKey struct {
	id     string
	client bool
}

// trackMarginOrder tracks whether an order on a spot instrument trades on margin, for its trades, which name no asset,
// to be filled as the asset it trades as: by its order ID, and by its client order ID as a margin order's until its
// order ID is known, as orders can share a client order ID. Either ID may be empty
func (e *Exchange) trackMarginOrder(a asset.Item, orderID, clientOrderID string) {
	if e.marginOrders == nil || (a != asset.Spot && a != asset.Margin) {
		return
	}
	if orderID != "" {
		e.marginOrders.Add(marginOrderKey{id: orderID}, a == asset.Margin)
	}
	if a == asset.Margin && clientOrderID != "" {
		e.marginOrders.Add(marginOrderKey{id: clientOrderID, client: true}, true)
		return
	}
	e.untrackClientOrderID(clientOrderID)
}

// untrackClientOrderID stops tracking a client order ID as a margin order's, as when Crypto.com did not place the order
func (e *Exchange) untrackClientOrderID(clientOrderID string) {
	if e.marginOrders != nil && clientOrderID != "" {
		e.marginOrders.Remove(marginOrderKey{id: clientOrderID, client: true})
	}
}

// isMarginOrder reports whether an order is tracked as a margin order: by its order ID when that is tracked, else by
// its client order ID
func (e *Exchange) isMarginOrder(orderID, clientOrderID string) bool {
	if e.marginOrders == nil {
		return false
	}
	if orderID != "" {
		if isMargin, ok := e.marginOrders.Get(marginOrderKey{id: orderID}).(bool); ok {
			return isMargin
		}
	}
	return clientOrderID != "" && e.marginOrders.Contains(marginOrderKey{id: clientOrderID, client: true})
}

// wsHandleUserTrades relays the account's trades through the fills feed when it is enabled, else to the data handler as
// Crypto.com reports them. A trade carries no execution instructions, so a trade on a spot instrument is filled as margin
// when its order is tracked as a margin order, from the order channel or from placing it, and otherwise as spot; a
// trade whose fill cannot be told is reported, and the others filled
func (e *Exchange) wsHandleUserTrades(ctx context.Context, push *websocketChannelPush) error {
	var trades []WebsocketUserTrade
	if err := decodePushData(push, &trades); err != nil {
		return err
	}
	if len(trades) == 0 {
		return nil
	}
	if !e.IsFillsFeedEnabled() {
		return e.Websocket.DataHandler.Send(ctx, trades)
	}
	fills := make([]fill.Data, 0, len(trades))
	var errs error
	for i := range trades {
		f, err := e.userTradeFill(&trades[i])
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s trade %s: %w", push.Channel, trades[i].TradeID, err))
			continue
		}
		fills = append(fills, f)
	}
	return common.AppendError(errs, e.Websocket.Fills.Update(fills...))
}

// userTradeFill returns a trade as a fill of the asset its order trades as, timed to the nanosecond where Crypto.com
// sends the time so
func (e *Exchange) userTradeFill(t *WebsocketUserTrade) (fill.Data, error) {
	a := orderAsset(t.InstrumentName, nil)
	if a == asset.Spot && e.isMarginOrder(t.OrderID, t.ClientOrderID) {
		a = asset.Margin
	}
	p, err := pairFromInstrument(a, t.InstrumentName)
	if err != nil {
		return fill.Data{}, err
	}
	side, err := orderSideFromString(t.Side)
	if err != nil {
		return fill.Data{}, err
	}
	at := t.CreateTimeNano.Time()
	if at.IsZero() {
		at = t.CreateTime.Time()
	}
	return fill.Data{
		Timestamp:     at,
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  p,
		Side:          side,
		OrderID:       t.OrderID,
		ClientOrderID: t.ClientOrderID,
		TradeID:       t.TradeID,
		Price:         t.TradedPrice.Float64(),
		Amount:        t.TradedQuantity.Float64(),
	}, nil
}

// wsHandleTransactions relays the account's transactions, trades and every other change to its balances
func (e *Exchange) wsHandleTransactions(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[TransactionResponse](ctx, e, push)
}

// wsHandleUserBalance saves the account's balances to the account store and relays them. Crypto.com keeps one wallet
// for spot, margin and derivatives, which is saved once, under spot, as the portfolio's balance refresh saves it, since
// the account store adds up every asset's balances. Crypto.com's docs leave out whether a push holds every currency,
// so the currencies a push leaves out are kept
func (e *Exchange) wsHandleUserBalance(ctx context.Context, push *websocketChannelPush) error {
	var balances []UserBalanceResponse
	if err := decodePushData(push, &balances); err != nil {
		return err
	}
	if len(balances) == 0 {
		return nil
	}
	subAccts := accounts.SubAccounts{balanceSubAccount(asset.Spot, balances)}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// wsHandlePositions relays the account's open positions
func (e *Exchange) wsHandlePositions(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[PositionResponse](ctx, e, push)
}

// wsHandleAccountRisk relays the account's risk, with the account the push names
func (e *Exchange) wsHandleAccountRisk(ctx context.Context, push *websocketChannelPush) error {
	var risks []WebsocketAccountRisk
	if err := decodePushData(push, &risks); err != nil {
		return err
	}
	if len(risks) == 0 {
		return nil
	}
	for i := range risks {
		risks[i].AccountID = push.AccountID
	}
	return e.Websocket.DataHandler.Send(ctx, risks)
}

// wsHandlePositionBalance relays the account's collateral balances and positions
func (e *Exchange) wsHandlePositionBalance(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[WebsocketPositionBalance](ctx, e, push)
}

// wsHandleOTCQuoteRequests relays the account's OTC quote requests as their statuses change
func (e *Exchange) wsHandleOTCQuoteRequests(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[OTCQuoteRequestResponse](ctx, e, push)
}

// wsHandleOTCQuotes relays the quotes liquidity providers make for the account's OTC quote requests
func (e *Exchange) wsHandleOTCQuotes(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[OTCQuoteResponse](ctx, e, push)
}

// wsHandleOTCDeals relays the account's OTC deals as their statuses change
func (e *Exchange) wsHandleOTCDeals(ctx context.Context, push *websocketChannelPush) error {
	return relayUserData[OTCDealResponse](ctx, e, push)
}

// relayUserData relays the data a user channel's push holds, unless it holds none
func relayUserData[T any](ctx context.Context, e *Exchange, push *websocketChannelPush) error {
	var data []T
	if err := decodePushData(push, &data); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

package bitfinex

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// wsWalletCalcs holds the wallets Bitfinex has been asked to calculate the available balances of, by their calc request
// names, until it sends them
type wsWalletCalcs struct {
	mu      sync.Mutex
	pending map[string]struct{}
}

// request returns the calc requests of the wallets not already asked about, marking them asked
func (w *wsWalletCalcs) request(calcs []string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pending == nil {
		w.pending = make(map[string]struct{})
	}
	var asked []string
	for _, c := range calcs {
		if _, ok := w.pending[c]; !ok {
			w.pending[c] = struct{}{}
			asked = append(asked, c)
		}
	}
	return asked
}

// done forgets wallets' calc requests, as their available balances have arrived or the requests were not sent
func (w *wsWalletCalcs) done(calcs ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, c := range calcs {
		delete(w.pending, c)
	}
}

// reset forgets every calc request, as a new account connection's snapshot asks again
func (w *wsWalletCalcs) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	clear(w.pending)
}

// wsHandleAccountEvent handles an event on the account channel, which an authenticated connection receives its
// account's events on: [0, TYPE, DATA]
func (e *Exchange) wsHandleAccountEvent(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	data := wsFrameData(respRaw)
	if len(data) == 0 || data[0] != '"' {
		return fmt.Errorf("%w: %.256s", errWebsocketMalformedFrame, respRaw)
	}
	eventType, _, _, err := jsonparser.Get(data)
	if err != nil {
		return fmt.Errorf("%w: %w: %.256s", errWebsocketMalformedFrame, err, respRaw)
	}
	switch string(eventType) {
	case "hb":
		return nil
	case "n":
		return e.wsHandleNotification(ctx, conn, respRaw)
	case "os":
		return e.wsHandleOrders(ctx, respRaw)
	case "on", "ou", "oc":
		return e.wsHandleOrder(ctx, string(eventType) == "oc", respRaw)
	case "ps":
		return wsRelayAccountSnapshot[PositionResponse](ctx, e, respRaw)
	case "pn", "pu", "pc":
		return wsRelayAccountUpdate[PositionResponse](ctx, e, respRaw)
	case "te", "tu":
		return e.wsHandleAccountTrade(ctx, string(eventType), respRaw)
	case "fos":
		return wsRelayAccountSnapshot[FundingOfferResponse](ctx, e, respRaw)
	case "fon", "fou", "foc":
		return wsRelayAccountUpdate[FundingOfferResponse](ctx, e, respRaw)
	case "fcs", "fls":
		return wsRelayAccountSnapshot[FundingCreditResponse](ctx, e, respRaw)
	case "fcn", "fcu", "fcc", "fln", "flu", "flc":
		return wsRelayAccountUpdate[FundingCreditResponse](ctx, e, respRaw)
	case "fte", "ftu":
		return wsRelayAccountUpdate[AccountFundingTradeResponse](ctx, e, respRaw)
	case "ws":
		return e.wsHandleWallets(ctx, conn, respRaw)
	case "wu":
		return e.wsHandleWalletUpdate(ctx, conn, respRaw)
	case "bu":
		return wsRelayAccountUpdate[BalanceInfo](ctx, e, respRaw)
	case "miu":
		return e.wsHandleMarginInfo(ctx, respRaw)
	case "fiu":
		return wsRelayAccountUpdate[FundingInfoResponse](ctx, e, respRaw)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

// wsRelayAccountSnapshot relays a snapshot of the account's records, such as its positions
func wsRelayAccountSnapshot[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	var f wsTypedFrame[[]T]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, f.Data)
}

// wsRelayAccountUpdate relays an update of one of the account's records
func wsRelayAccountUpdate[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	var f wsTypedFrame[T]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &f.Data)
}

// wsHandleOrders relays the snapshot of the account's open orders
func (e *Exchange) wsHandleOrders(ctx context.Context, respRaw []byte) error {
	var f wsTypedFrame[[]OrderResponse]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	details := make([]order.Detail, 0, len(f.Data))
	var errs error
	for i := range f.Data {
		d, err := e.orderDetail(&f.Data[i])
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order %d: %w", f.Data[i].ID, err))
			continue
		}
		details = append(details, *d)
	}
	if len(details) > 0 {
		if err := e.Websocket.DataHandler.Send(ctx, details); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// wsHandleOrder relays an order placed, updated, or closed by being cancelled or filled
func (e *Exchange) wsHandleOrder(ctx context.Context, closed bool, respRaw []byte) error {
	var f wsTypedFrame[OrderResponse]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	d, err := e.orderDetail(&f.Data)
	if err != nil {
		return fmt.Errorf("order %d: %w", f.Data.ID, err)
	}
	if closed {
		d.Status = closedOrderStatus(d.Status)
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

// wsHandleAccountTrade relays a trade one of the account's orders made. Its execution update, which follows with the
// fee, is also relayed as a fill, when the fills feed is on
func (e *Exchange) wsHandleAccountTrade(ctx context.Context, eventType string, respRaw []byte) error {
	var f wsTypedFrame[AccountTradeResponse]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, &f.Data); err != nil {
		return err
	}
	if eventType != "tu" || !e.IsFillsFeedEnabled() {
		return nil
	}
	t := &f.Data
	pair, err := pairFromSymbol(t.Symbol)
	if err != nil {
		return err
	}
	side, err := orderSideFromAmount(t.ExecutedAmount)
	if err != nil {
		return err
	}
	tradeID := strconv.FormatUint(t.ID, 10)
	fd := fill.Data{
		ID:           tradeID,
		Timestamp:    t.ExecutedAt.Time(),
		Exchange:     e.Name,
		AssetType:    orderAsset(t.OrderType, pair),
		CurrencyPair: pair,
		Side:         side,
		OrderID:      strconv.FormatUint(t.OrderID, 10),
		TradeID:      tradeID,
		Price:        t.ExecutedPrice,
		Amount:       math.Abs(t.ExecutedAmount),
	}
	if t.ClientOrderID != 0 {
		fd.ClientOrderID = strconv.FormatUint(t.ClientOrderID, 10)
	}
	return e.Websocket.Fills.Update(fd)
}

// wsHandleWallets stores and relays the snapshot of the account's wallets
func (e *Exchange) wsHandleWallets(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var f wsTypedFrame[[]WalletResponse]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	return e.wsStoreWallets(ctx, conn, f.Data)
}

// wsHandleWalletUpdate stores and relays an update of one of the account's wallets
func (e *Exchange) wsHandleWalletUpdate(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var f wsTypedFrame[WalletResponse]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	return e.wsStoreWallets(ctx, conn, []WalletResponse{f.Data})
}

// wsStoreWallets stores the balances of wallets, each under the asset its balance funds, and relays them. Bitfinex
// sends no available balance until it has calculated it, so until then none of the wallet is free, and Bitfinex is
// asked to calculate it
func (e *Exchange) wsStoreWallets(ctx context.Context, conn websocket.Connection, wallets []WalletResponse) error {
	var subAccts accounts.SubAccounts
	var calcs []string
	var errs error
	for i := range wallets {
		w := &wallets[i]
		a, err := walletAsset(w.Type, w.Currency)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		calc := "wallet_" + w.Type + "_" + w.Currency.String()
		if w.AvailableBalance == nil {
			calcs = append(calcs, calc)
		} else {
			e.wsWalletCalcs.done(calc)
		}
		balance, err := e.Accounts.UpdateBalance(ctx, "", a, w.Currency, func(b *accounts.Balance) {
			b.Total = w.Balance
			b.Free = 0
			if w.AvailableBalance != nil {
				b.Free = *w.AvailableBalance
			}
			b.Hold = b.Total - b.Free
		})
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		subAcct := accounts.NewSubAccount(a, "")
		subAcct.Balances.Set(w.Currency, balance)
		subAccts = subAccts.Merge(subAcct)
	}
	if len(subAccts) > 0 {
		if err := e.Websocket.DataHandler.Send(ctx, subAccts); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return common.AppendError(errs, e.wsRequestWalletCalcs(ctx, conn, calcs))
}

// wsRequestWalletCalcs asks Bitfinex to calculate the available balances of wallets it has yet to, once each until it
// sends one
func (e *Exchange) wsRequestWalletCalcs(ctx context.Context, conn websocket.Connection, calcs []string) error {
	calcs = e.wsWalletCalcs.request(calcs)
	var errs error
	for batch := range slices.Chunk(calcs, wsMaxCalcRequests) {
		requests := make([][1]string, len(batch))
		for i := range batch {
			requests[i] = [1]string{batch[i]}
		}
		if err := conn.SendJSONMessage(ctx, request.Unset, &wsInput{Type: wsCalc, Data: requests}); err != nil {
			e.wsWalletCalcs.done(batch...)
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// wsHandleMarginInfo relays the account's margin summary, or its buying power on a trading pair
func (e *Exchange) wsHandleMarginInfo(ctx context.Context, respRaw []byte) error {
	data, _, _, err := jsonparser.Get(respRaw, "[2]", "[0]")
	if err != nil {
		return fmt.Errorf("%w: %w: %.256s", errWebsocketMalformedFrame, err, respRaw)
	}
	switch string(data) {
	case marginInfoKeyBase:
		return wsRelayAccountUpdate[MarginInfoBaseResponse](ctx, e, respRaw)
	case marginInfoKeySymbol:
		return wsRelayAccountUpdate[MarginInfoSymbolResponse](ctx, e, respRaw)
	}
	return fmt.Errorf("%w: margin info %q: %.256s", errWebsocketMalformedFrame, data, respRaw)
}

// wsHandleNotification passes a notification answering an input request to the request, and relays any other, such as
// a price alert or the notification of a request another client sent
func (e *Exchange) wsHandleNotification(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var f wsTypedFrame[Notification[json.RawMessage]]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	if signature := wsNotificationSignature(&f.Data); signature != "" && conn.IncomingWithData(signature, respRaw) {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, &f.Data)
}

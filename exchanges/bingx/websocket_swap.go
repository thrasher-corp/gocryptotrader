package bingx

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// wsHandleSwapData routes a USDT-M perpetual market data frame
func (e *Exchange) wsHandleSwapData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	symbol, stream, data, err := wsMarketFrame(respRaw)
	if err != nil {
		return err
	}
	switch {
	case stream == "trade":
		return e.wsProcessSwapTrades(data)
	case stream == "ticker":
		return e.wsProcessSwapTicker(ctx, data)
	case stream == WebsocketLastPriceChannel:
		return wsRelay[WebsocketSwapLastPrice](ctx, e, data)
	case stream == WebsocketMarkPriceChannel:
		return wsRelay[WebsocketMarkPrice](ctx, e, data)
	case stream == WebsocketBookTickerChannel:
		return wsRelay[WebsocketSwapBookTicker](ctx, e, data)
	case strings.HasPrefix(stream, "kline_"):
		return e.wsProcessSwapKlines(ctx, symbol, strings.TrimPrefix(stream, "kline_"), data)
	case strings.HasPrefix(stream, "depth"):
		return e.wsProcessSwapOrderbook(ctx, conn, symbol+"@"+stream, respRaw, data)
	}
	return fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
}

// wsProcessSwapTrades relays the trades of a USDT-M perpetual trade push, an array, for an enabled pair
func (e *Exchange) wsProcessSwapTrades(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp []WebsocketSwapTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(resp))
	for i := range resp {
		pair, enabled, err := e.wsMatchPair(resp[i].Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}
		trades = append(trades, trade.Data{
			TID:          resp[i].TradeID.String(),
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.USDTMarginedFutures,
			Side:         wsTakerSide(resp[i].IsBuyerMaker.Bool()),
			Price:        resp[i].Price.Float64(),
			Amount:       resp[i].Quantity.Float64(),
			Timestamp:    resp[i].TradeTime.Time(),
		})
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// wsProcessSwapTicker stores a USDT-M perpetual 24 hour ticker of an enabled pair and relays it
func (e *Exchange) wsProcessSwapTicker(ctx context.Context, data []byte) error {
	var resp WebsocketSwapTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	return e.wsProcessTicker(ctx, &ticker.Price{
		Last:                resp.LatestTransactionPrice.Float64(),
		LastSize:            resp.LatestTransactionAmount.Float64(),
		High:                resp.HighPrice.Float64(),
		Low:                 resp.LowPrice.Float64(),
		Bid:                 resp.BestBidPrice.Float64(),
		BidSize:             resp.BestBidQuantity.Float64(),
		Ask:                 resp.BestAskPrice.Float64(),
		AskSize:             resp.BestAskQuantity.Float64(),
		BaseVolume:          resp.Volume.Float64(),
		QuoteVolume:         resp.Turnover.Float64(),
		Open:                resp.FirstPrice.Float64(),
		PercentChange24Hour: resp.PriceChangePercentage.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.USDTMarginedFutures,
		LastUpdated:         resp.EventTime.Time(),
	})
}

// wsProcessSwapKlines relays the candles of a USDT-M perpetual candle push, an array, for an enabled pair. The push
// carries no time of its own, so a candle is partial while it is received before its end
func (e *Exchange) wsProcessSwapKlines(ctx context.Context, symbol, intervalName string, data []byte) error {
	pair, enabled, err := e.wsMatchPair(symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	interval, err := wsParseInterval(intervalName, intervalToString)
	if err != nil {
		return err
	}
	var resp []WebsocketSwapKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp) == 0 {
		return nil
	}
	received := time.Now()
	candles := make([]kline.Candle, len(resp))
	for i := range resp {
		candles[i] = kline.Candle{
			Time:             resp[i].Time.Time(),
			Open:             resp[i].Open.Float64(),
			High:             resp[i].High.Float64(),
			Low:              resp[i].Low.Float64(),
			Close:            resp[i].Close.Float64(),
			Volume:           resp[i].Volume.Float64(),
			ValidationIssues: wsCandleValidation(received, resp[i].Time.Time(), interval, time.UTC),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.USDTMarginedFutures,
		Interval: interval,
		Candles:  candles,
	})
}

// wsProcessSwapOrderbook loads a USDT-M perpetual partial depth push of an enabled pair as its book's snapshot, dated by
// ts, the time the server built the message. The data carries no symbol, so it comes from the dataType
func (e *Exchange) wsProcessSwapOrderbook(ctx context.Context, conn websocket.Connection, dataType string, respRaw, data []byte) error {
	symbol, _, _ := strings.Cut(dataType, "@")
	pair, enabled, err := e.wsMatchPair(symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	var resp WebsocketSwapOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	bids, asks := resp.Bids.Levels(), resp.Asks.Levels()
	wsSortLevels(bids, asks)
	pushed := wsFrameTime(respRaw, "ts")
	return e.wsLoadBook(ctx, conn, dataType, &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.USDTMarginedFutures,
		LastUpdated:       pushed,
		LastPushed:        pushed,
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsHandleSwapAccountData routes a USDT-M perpetual account data frame
func (e *Exchange) wsHandleSwapAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.wsHandlePerpetualAccountData(ctx, conn, respRaw, asset.USDTMarginedFutures)
}

// wsHandlePerpetualAccountData routes a USDT-M or coin-M perpetual account data frame, which BingX pushes every event to
// without a subscription. Order events arrive as ORDER_TRADE_UPDATE, or TRADE_UPDATE as the USDT-M docs example names
// them, with or without an envelope
func (e *Exchange) wsHandlePerpetualAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	event, data, err := wsAccountEvent(respRaw)
	if err != nil {
		return err
	}
	switch event {
	case "ORDER_TRADE_UPDATE", "TRADE_UPDATE":
		if e.wsHoldOrderEvent(conn, data) {
			return nil
		}
		state, _ := e.wsConnection(conn)
		return e.wsProcessPerpetualOrderUpdate(ctx, state, a, data)
	case "ACCOUNT_UPDATE":
		return e.wsProcessPerpetualAccountUpdate(ctx, conn, a, data)
	case "ACCOUNT_CONFIG_UPDATE":
		return wsRelay[WebsocketPerpetualConfigUpdate](ctx, e, data)
	case "listenKeyExpired":
		return e.wsProcessListenKeyExpired(ctx, conn, data)
	}
	return fmt.Errorf("%w %q on the %s account data stream", errWebsocketUnhandledEvent, event, a)
}

// wsProcessPerpetualOrderUpdate relays a perpetual order event as an order detail:
//   - p, q, sp, ap and z are the order's price, quantity, trigger price, average price and executed quantity
//   - the remaining amount is q less z, and ro marks a reduce only order
//   - the event carries no creation time, so the order is dated by its update: the event's T, or E when T is unset
//   - a USDT-M order's cost is ap times z in the quote currency; a coin-M order's cost needs the contract size the event
//     lacks
//
// The event reports a fill's fee alone, while an order detail carries the order's fee, so the fee of a TRADE or
// CALCULATED fill is added to the order's running total on state, the account connection the event arrived on, which
// the detail carries; nil, for a connection no longer tracked, keeps no totals. BingX reports a charged fee as a
// negative amount. The detail is relayed through the exchange's perpetualOrderRelay
func (e *Exchange) wsProcessPerpetualOrderUpdate(ctx context.Context, state *websocketConnection, a asset.Item, data []byte) error {
	var resp WebsocketPerpetualOrderUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	o := &resp.Order
	orderID := o.OrderID.String()
	pair, err := e.wsAccountPair(o.Symbol, a)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	status, err := wsPerpetualOrderStatus(a, o)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	orderType, err := convertOrderType(o.OrderType)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	side, err := convertOrderSide(o.Side)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	updated := o.TradeTime.Time()
	if updated.IsZero() {
		updated = resp.EventTime.Time()
	}
	executed := o.CumulativeFilledQuantity.Decimal()
	d := &order.Detail{
		ReduceOnly:           o.ReduceOnly.Bool(),
		Price:                o.Price.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.TriggerPrice.Float64(),
		AverageExecutedPrice: o.AverageFilled.Float64(),
		ExecutedAmount:       executed.InexactFloat64(),
		RemainingAmount:      max(o.Quantity.Decimal().Sub(executed).InexactFloat64(), 0),
		FeeAsset:             o.FeeAsset,
		Exchange:             e.Name,
		OrderID:              orderID,
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            a,
		Date:                 updated,
		LastUpdated:          updated,
		Pair:                 pair,
	}
	if a == asset.USDTMarginedFutures {
		d.Cost = o.AverageFilled.Decimal().Mul(executed).InexactFloat64()
	}
	if orderTypeTrails(o.OrderType) {
		// Reported as swapOrderDetail reports a trailing order, as the order manager hands the price it stores to a
		// modification without one, which takes a trailing order's price as its trailing distance
		d.Price = 0
	}
	if state != nil {
		fees := &state.fees
		switch o.ExecutionType {
		case ExecutionTypeNew:
			if executed.IsZero() {
				fees.start(orderID)
			}
		case ExecutionTypeTrade, ExecutionTypeCalculated:
			d.Fee = fees.add(orderID, executed, o.Fee.Decimal().Neg()).InexactFloat64()
		}
		if d.IsInactive() {
			fees.forget(orderID)
		}
	}
	return e.wsState.orderRelay.relay(d, executed, func() error { return e.Websocket.DataHandler.Send(ctx, d) })
}

// wsPerpetualOrderStatus converts a perpetual order event's status. Coin-M events are converted as coin-M replies are,
// in either case, and BingX's coin-M example reports Unknown for a fill, so a TRADE or CALCULATED event reporting
// Unknown whose executed quantity equals its positive quantity has filled the order. Any other Unknown stays unknown,
// which the order store keeps the status it holds for, as a part fill taken as partly filled would reopen an order the
// store holds as cancelled when the fill's event follows the cancellation's
func wsPerpetualOrderStatus(a asset.Item, o *WebsocketPerpetualOrder) (order.Status, error) {
	if a != asset.CoinMarginedFutures {
		return convertOrderStatus(o.OrderStatus)
	}
	status, err := convertCoinMarginedOrderStatus(o.OrderStatus)
	if err != nil || status != order.UnknownStatus {
		return status, err
	}
	if (o.ExecutionType == ExecutionTypeTrade || o.ExecutionType == ExecutionTypeCalculated) && o.Quantity > 0 && o.CumulativeFilledQuantity == o.Quantity {
		return order.Filled, nil
	}
	return order.UnknownStatus, nil
}

// wsHoldOrderEvent holds an order event that arrives while its account connection reads its open orders, reporting
// whether it did, and notes the connection holding an event of its order with the exchange's perpetualOrderRelay
func (e *Exchange) wsHoldOrderEvent(conn websocket.Connection, data []byte) bool {
	state, err := e.wsConnection(conn)
	if err != nil {
		return false
	}
	state.heldOrders.Lock()
	defer state.heldOrders.Unlock()
	if !state.holding {
		return false
	}
	state.held = append(state.held, bytes.Clone(data))
	var event WebsocketPerpetualOrderUpdate
	if err := json.Unmarshal(data, &event); err == nil {
		e.wsState.orderRelay.hold(state, event.Order.OrderID.String())
	}
	return true
}

// wsReleaseOrderEvents handles the order events an account connection held while it read its open orders, in the order
// they arrived, adding their fills to the connection's own totals whether or not it is still tracked, and stops holding
// them. An event arriving meanwhile waits, and is handled after them. With lookup set, the events stay held while any
// is a fill whose order's total would start without the fills before the connection came up, and those orders are
// returned to be looked up, each once: settled names the orders the read listed or that were looked up
func (e *Exchange) wsReleaseOrderEvents(ctx context.Context, state *websocketConnection, settled map[string]bool, lookup bool) []perpetualOrderLookup {
	state.heldOrders.Lock()
	defer state.heldOrders.Unlock()
	if lookup {
		if lookups := heldOrdersMissingFees(state, settled); len(lookups) != 0 {
			return lookups
		}
	}
	for _, data := range state.held {
		if err := e.wsProcessPerpetualOrderUpdate(ctx, state, state.account, data); err != nil {
			log.Errorf(log.WebsocketMgr, "%s %s held order event failed: %v", e.Name, state.account, err)
		}
	}
	state.held, state.holding = nil, false
	e.wsState.orderRelay.unhold(state)
	return nil
}

// perpetualOrderLookup is an order an account connection holds events of, to be looked up by its ID and symbol
type perpetualOrderLookup struct {
	orderID, symbol string
}

// perpetualHeldOrder is what an account connection's held events show of an order: its symbol, whether the first of
// them is a NEW event reporting nothing executed, and whether any is a fill
type perpetualHeldOrder struct {
	symbol          string
	started, filled bool
}

// heldOrdersMissingFees returns the orders of a connection's held events whose totals the events would start without
// the fills before the connection came up: an order with a held fill, whose first held event is not a NEW event
// reporting nothing executed, which has no total counting all its fills and which settled does not name. The caller
// holds state.heldOrders
func heldOrdersMissingFees(state *websocketConnection, settled map[string]bool) []perpetualOrderLookup {
	held := make(map[string]*perpetualHeldOrder)
	var orderIDs []string
	for _, data := range state.held {
		var event WebsocketPerpetualOrderUpdate
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		o := &event.Order
		orderID := o.OrderID.String()
		h, ok := held[orderID]
		if !ok {
			h = &perpetualHeldOrder{symbol: o.Symbol, started: o.ExecutionType == ExecutionTypeNew && o.CumulativeFilledQuantity.Decimal().IsZero()}
			held[orderID] = h
			orderIDs = append(orderIDs, orderID)
		}
		if o.ExecutionType == ExecutionTypeTrade || o.ExecutionType == ExecutionTypeCalculated {
			h.filled = true
		}
	}
	var lookups []perpetualOrderLookup
	for _, orderID := range orderIDs {
		if h := held[orderID]; h.filled && !h.started && !settled[orderID] && !state.fees.counted(orderID) {
			lookups = append(lookups, perpetualOrderLookup{orderID: orderID, symbol: h.symbol})
		}
	}
	return lookups
}

// wsLookUpOrderFees looks up orders an account connection holds events of, at once so that one slow lookup leaves the
// others their time, and starts each one's total on the connection from the executed quantity and commission BingX
// reports, open or closed, as the order's held events are handled after and add the fills beyond it. Each order is
// settled whatever the outcome, so it is looked up once
func (e *Exchange) wsLookUpOrderFees(ctx context.Context, state *websocketConnection, lookups []perpetualOrderLookup, settled map[string]bool) {
	var wg sync.WaitGroup
	for _, l := range lookups {
		settled[l.orderID] = true
		wg.Go(func() {
			executed, commission, err := e.perpetualOrderFees(ctx, state.account, l)
			if err != nil {
				log.Errorf(log.WebsocketMgr, "%s %s order %s fee lookup failed: %v", e.Name, state.account, l.orderID, err)
				return
			}
			state.fees.seed(l.orderID, executed.Decimal(), commission.Decimal().Neg())
		})
	}
	wg.Wait()
}

// perpetualOrderFees returns the executed quantity and commission BingX reports for a perpetual order
func (e *Exchange) perpetualOrderFees(ctx context.Context, a asset.Item, l perpetualOrderLookup) (executed, commission types.Number, err error) {
	pair, err := e.wsAccountPair(l.symbol, a)
	if err != nil {
		return 0, 0, err
	}
	id, err := parseOrderID(l.orderID)
	if err != nil {
		return 0, 0, err
	}
	switch a {
	case asset.USDTMarginedFutures:
		o, err := e.GetSwapOrderDetails(ctx, pair, id, "")
		if err != nil {
			return 0, 0, err
		}
		return o.ExecutedQuantity, o.Commission, nil
	case asset.CoinMarginedFutures:
		o, err := e.GetCoinMarginedOrderDetail(ctx, pair, id, "")
		if err != nil {
			return 0, 0, err
		}
		return o.ExecutedQuantity, o.Commission, nil
	default:
		return 0, 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// perpetualOrderRelay relays the order events of the perpetual account connections. An account connection retired
// while it reads its open orders, as a flush or a failed connection attempt can retire one within its session, hands
// its held events over once the read ends, which can be after a connection dialled in its place has relayed newer
// events of the same orders. So while a connection holds events of an order, how far the order's relayed events have
// taken it is kept, and an event behind that is dropped rather than relayed
type perpetualOrderRelay struct {
	mu sync.Mutex
	// holders holds the orders each connection holding its order events holds events of
	holders map[*websocketConnection]map[perpetualOrderKey]struct{}
	// heldBy counts the connections holding events of each order
	heldBy  map[perpetualOrderKey]int
	relayed map[perpetualOrderKey]perpetualOrderProgress
}

// perpetualOrderKey is a perpetual order of a product
type perpetualOrderKey struct {
	asset   asset.Item
	orderID string
}

// perpetualOrderProgress is how far an order's last relayed event took it: the executed quantity, and whether the event
// finished the order, with an inactive status or its whole quantity executed. An order of no quantity, such as one
// closing a position, finishes only with its status
type perpetualOrderProgress struct {
	executed decimal.Decimal
	finished bool
}

// hold notes a connection holding an event of one of its product's orders
func (r *perpetualOrderRelay) hold(state *websocketConnection, orderID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.holders == nil {
		r.holders = make(map[*websocketConnection]map[perpetualOrderKey]struct{})
		r.heldBy = make(map[perpetualOrderKey]int)
	}
	orders, ok := r.holders[state]
	if !ok {
		orders = make(map[perpetualOrderKey]struct{})
		r.holders[state] = orders
	}
	k := perpetualOrderKey{asset: state.account, orderID: orderID}
	if _, ok := orders[k]; !ok {
		orders[k] = struct{}{}
		r.heldBy[k]++
	}
}

// unhold notes a connection no longer holding its order events, forgetting how far each order it held events of went
// unless another connection holds events of the order, and everything once no connection holds any
func (r *perpetualOrderRelay) unhold(state *websocketConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.holders[state] {
		r.heldBy[k]--
		if r.heldBy[k] == 0 {
			delete(r.heldBy, k)
			delete(r.relayed, k)
		}
	}
	delete(r.holders, state)
	if len(r.holders) == 0 {
		r.holders, r.heldBy, r.relayed = nil, nil, nil
	}
}

// relay relays an order event's detail with send. While a connection holds events of the order, an event behind how
// far the order's relayed events have taken it is dropped: one at a lower executed quantity, or one leaving the order
// unfinished at the quantity an event finished it at. Only an event send succeeds with counts as relayed
func (r *perpetualOrderRelay) relay(d *order.Detail, executed decimal.Decimal, send func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := perpetualOrderKey{asset: d.AssetType, orderID: d.OrderID}
	if r.heldBy[k] == 0 {
		return send()
	}
	finished := d.Status.IsInactive() || d.Amount > 0 && d.ExecutedAmount >= d.Amount
	if p, ok := r.relayed[k]; ok && (executed.LessThan(p.executed) || executed.Equal(p.executed) && p.finished && !finished) {
		return nil
	}
	if err := send(); err != nil {
		return err
	}
	if r.relayed == nil {
		r.relayed = make(map[perpetualOrderKey]perpetualOrderProgress)
	}
	r.relayed[k] = perpetualOrderProgress{executed: executed, finished: finished}
	return nil
}

// perpetualOrderFeeTotals keeps the fees of the open perpetual orders an account connection streams. A total starts
// from the order's NEW event, from a REST read of the order begun while the connection was up or by the connection
// itself, from a lookup of an order the connection held events of during its own read, or from the first fill the
// connection streams, in which case the fills before it are left out until such a read
type perpetualOrderFeeTotals struct {
	mu     sync.Mutex
	orders map[string]*perpetualOrderFeeTotal
	// seq numbers the REST reads and the orders' closes. reads holds the number each read in flight began at, and
	// closed the number each order closed at while a read that began before it was in flight, as that read may report
	// the order open
	seq    uint64
	reads  map[uint64]struct{}
	closed map[string]uint64
}

// perpetualOrderFeeTotal is the fee of an order's fills up to its executed quantity. Until a REST read or the order's
// NEW event shows its first fills are counted, the fills its events reported are kept, so a read taken before the
// latest of them can still supply the fills before them
type perpetualOrderFeeTotal struct {
	executed, fee decimal.Decimal
	complete      bool
	fills         []perpetualOrderFill
}

// perpetualOrderFill is the executed quantity an order event reported and the fee of the fill that took it there
type perpetualOrderFill struct {
	executed, fee decimal.Decimal
}

// total returns an order's total, starting it if it has none
func (t *perpetualOrderFeeTotals) total(orderID string) *perpetualOrderFeeTotal {
	total, ok := t.orders[orderID]
	if !ok {
		if t.orders == nil {
			t.orders = make(map[string]*perpetualOrderFeeTotal)
		}
		total = new(perpetualOrderFeeTotal)
		t.orders[orderID] = total
	}
	return total
}

// counted reports whether an order's total counts all its fills so far
func (t *perpetualOrderFeeTotals) counted(orderID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	total, ok := t.orders[orderID]
	return ok && total.complete
}

// start marks an order whose NEW event reports nothing executed as having all its fills counted, unless it has a total
func (t *perpetualOrderFeeTotals) start(orderID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.orders[orderID]; !ok {
		t.total(orderID).complete = true
	}
}

// add counts a fill's fee once, when its event raises the order's executed quantity, which a repeated event does not,
// and returns the order's fee so far
func (t *perpetualOrderFeeTotals) add(orderID string, executed, fee decimal.Decimal) decimal.Decimal {
	t.mu.Lock()
	defer t.mu.Unlock()
	total, ok := t.orders[orderID]
	if !ok {
		if !executed.IsPositive() {
			return decimal.Zero
		}
		total = t.total(orderID)
	}
	if !executed.GreaterThan(total.executed) {
		return total.fee
	}
	total.executed = executed
	total.fee = total.fee.Add(fee)
	if !total.complete {
		total.fills = append(total.fills, perpetualOrderFill{executed: executed, fee: fee})
	}
	return total.fee
}

// seed takes a REST read's executed quantity and fee for an open order as its total, unless its events or another read
// have counted further. A total missing its first fills takes them from the read and keeps the fills its events
// counted beyond it. An order closed since a read in flight began gets no total from a read
func (t *perpetualOrderFeeTotals) seed(orderID string, executed, fee decimal.Decimal) {
	if !executed.IsPositive() {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.closed[orderID]; ok {
		return
	}
	total := t.total(orderID)
	if total.complete {
		if executed.GreaterThanOrEqual(total.executed) {
			total.executed, total.fee = executed, fee
		}
		return
	}
	for _, f := range total.fills {
		if f.executed.GreaterThan(executed) {
			fee = fee.Add(f.fee)
		}
	}
	if executed.GreaterThan(total.executed) {
		total.executed = executed
	}
	total.fee, total.complete, total.fills = fee, true, nil
}

// forget drops the total of an order that is no longer open, noting its close while a read is in flight
func (t *perpetualOrderFeeTotals) forget(orderID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.orders, orderID)
	if len(t.reads) == 0 {
		return
	}
	if t.closed == nil {
		t.closed = make(map[string]uint64)
	}
	t.seq++
	t.closed[orderID] = t.seq
}

// beginRead numbers a REST read whose open orders may start totals
func (t *perpetualOrderFeeTotals) beginRead() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.reads == nil {
		t.reads = make(map[uint64]struct{})
	}
	t.seq++
	t.reads[t.seq] = struct{}{}
	return t.seq
}

// endRead ends a REST read, dropping the closes noted before every read still in flight began
func (t *perpetualOrderFeeTotals) endRead(read uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.reads, read)
	oldest := uint64(math.MaxUint64)
	for r := range t.reads {
		oldest = min(oldest, r)
	}
	for orderID, closed := range t.closed {
		if closed < oldest {
			delete(t.closed, orderID)
		}
	}
}

// perpetualOrderFeeRead is a REST read of perpetual orders, whose open orders start the fee totals of the account
// connections that were streaming the read's account and product when it began, and of the connection whose own read
// it is
type perpetualOrderFeeRead struct {
	totals []*perpetualOrderFeeTotals
	reads  []uint64
}

// perpetualOrderFeeReadOwnerKey is the context key of the account connection whose own read of its open orders the
// context carries
type perpetualOrderFeeReadOwnerKey struct{}

// beginPerpetualOrderFeeRead begins a REST read of asset a's orders for ctx's account, returning ctx carrying that
// account's credentials, so the read requests and the totals it starts are that account's although the exchange's
// default credentials change meanwhile. The connection whose own read ctx carries counts although it is no longer up,
// as it hands the events it held during the read over with its own totals. Without credentials the read starts no
// totals, and ctx is returned unchanged
func (e *Exchange) beginPerpetualOrderFeeRead(ctx context.Context, a asset.Item) (context.Context, *perpetualOrderFeeRead) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return ctx, nil
	}
	states := e.wsAccountConnections(a, creds)
	if owner, ok := ctx.Value(perpetualOrderFeeReadOwnerKey{}).(*websocketConnection); ok && !slices.Contains(states, owner) {
		states = append(states, owner)
	}
	read := new(perpetualOrderFeeRead)
	for _, state := range states {
		read.totals = append(read.totals, &state.fees)
		read.reads = append(read.reads, state.fees.beginRead())
	}
	return accounts.DeployCredentialsToContext(ctx, creds), read
}

// seed starts the fee total of an open order the read reports from its executed quantity and its commission, negated
// as an event's fill fee is, as BingX reports a charged fee as negative
func (r *perpetualOrderFeeRead) seed(d *order.Detail, executed, commission types.Number) {
	if r == nil || !d.IsActive() {
		return
	}
	for _, t := range r.totals {
		t.seed(d.OrderID, executed.Decimal(), commission.Decimal().Neg())
	}
}

// end ends the read
func (r *perpetualOrderFeeRead) end() {
	if r == nil {
		return
	}
	for i, t := range r.totals {
		t.endRead(r.reads[i])
	}
}

// wsProcessPerpetualAccountUpdate relays a perpetual ACCOUNT_UPDATE event's balances and positions. A balance's wb is
// its wallet balance, and nothing in the event gives the margin available or held, so the balances are relayed as the
// event reports them rather than stored, and the asset's balances are refreshed over REST instead, once for a burst of
// events, for the account of the credentials the connection's listen key was created for. A connection no longer
// tracked refreshes nothing, as its session has ended. A position whose symbol cannot be read is left out
func (e *Exchange) wsProcessPerpetualAccountUpdate(ctx context.Context, conn websocket.Connection, a asset.Item, data []byte) error {
	var resp WebsocketPerpetualAccountUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp.Account.Balances) != 0 {
		if state, err := e.wsConnection(conn); err == nil {
			e.refreshBalances(accounts.DeployCredentialsToContext(ctx, state.credentials), a, state.session)
		}
		if err := e.Websocket.DataHandler.Send(ctx, resp.Account.Balances); err != nil {
			return err
		}
	}
	positions := make([]futures.Position, 0, len(resp.Account.Positions))
	for i := range resp.Account.Positions {
		p := &resp.Account.Positions[i]
		pair, err := e.wsAccountPair(p.Symbol, a)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s %s account update: leaving out a position: %v", e.Name, a, err)
			continue
		}
		positions = append(positions, wsPerpetualPosition(e.Name, a, pair, p, resp.EventTime.Time()))
	}
	if len(positions) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

// wsPerpetualPosition builds a position from a perpetual account event's: pa is its signed size, ep its entry price,
// iw its isolated margin, and up and cr its unrealised and realised profit. Hedge mode reports the side, while one-way
// mode, BOTH, signs the size; a zero size closes the position
func wsPerpetualPosition(exchangeName string, a asset.Item, pair currency.Pair, p *WebsocketPerpetualPosition, updated time.Time) futures.Position {
	amount := p.PositionAmount.Decimal()
	position := futures.Position{
		Exchange:       exchangeName,
		Asset:          a,
		Pair:           pair,
		PositionMargin: p.IsolatedPositionMargin.Decimal(),
		RealisedPNL:    p.RealisedPNL.Decimal(),
		UnrealisedPNL:  p.UnrealisedPNL.Decimal(),
		Status:         order.Open,
		OpeningPrice:   p.EntryPrice.Decimal(),
		LatestSize:     amount.Abs(),
		LastUpdated:    updated,
	}
	switch {
	case p.PositionSide == PositionSideLong, p.PositionSide != PositionSideShort && amount.IsPositive():
		position.OpeningDirection = order.Long
	case p.PositionSide == PositionSideShort, amount.IsNegative():
		position.OpeningDirection = order.Short
	}
	position.LatestDirection = position.OpeningDirection
	if amount.IsZero() {
		position.Status = order.Closed
		position.CloseDate = updated
	}
	return position
}

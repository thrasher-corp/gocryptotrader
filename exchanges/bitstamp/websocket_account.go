package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// Order events of the private orders channel
const (
	orderEventCreated      = "order_created"
	orderEventChanged      = "order_changed"
	orderEventDeleted      = "order_deleted"
	orderEventReplaced     = "order_replaced"
	orderEventStopActive   = "stop_active"
	orderEventStopInactive = "stop_inactive"
)

// maxOrderEventRecoveryFailures is the number of failed recoveries in a row after which missed order events are given
// up on, so that a recovery which keeps failing does not hold back every later event
const maxOrderEventRecoveryFailures = 3

// maxKeptOrderEvents is the number of received order events kept for a market and order source while they cannot be
// relayed
const maxKeptOrderEvents = 1000

// keptOrderEventRetryInterval is the least time from the end of one retry of the kept order events to the next. Every
// connection's heartbeat reply retries, and several connections' replies can arrive together, which would use up the
// failed recoveries allowed before giving up within moments; one connection's replies, a heartbeat interval apart, each
// still retry
const keptOrderEventRetryInterval = hbInterval / 2

func (e *Exchange) handleWSSelfTrade(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	selfTrade := &WebsocketSelfTrade{Pair: pair, AssetType: a}
	if err := json.Unmarshal(data, selfTrade); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, selfTrade)
}

// handleWSAccountOrder processes an event for one of the account's orders
// Each event names the event before it, so events missed by a reconnection or a server restart are recovered through
// the REST API before the event is processed. Events are relayed in order, so an event whose recovery fails is kept,
// and relayed once a later recovery reaches it, or before the current event when recovery gives up
func (e *Exchange) handleWSAccountOrder(ctx context.Context, respRaw []byte, event string, data []byte, pair currency.Pair, a asset.Item) error {
	var o OrderEventData
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	e.orderEventHandlingMu.Lock()
	defer e.orderEventHandlingMu.Unlock()
	eventID, eventErr := jsonparser.GetString(respRaw, "event_id")
	preEventID, preEventType, _, preEventErr := jsonparser.Get(respRaw, "pre_event_id")
	if eventErr != nil || eventID == "" || preEventErr != nil {
		return e.processOrderEvent(ctx, event, &o, pair, a)
	}
	orderSource, _ := jsonparser.GetString(respRaw, "order_source")
	key := pair.String() + "|" + a.String() + "|" + orderSource
	cur := &keptOrderEvent{eventID: eventID, event: event, data: o, pair: pair, asset: a, orderSource: orderSource}
	// A null preceding event follows a server restart, which may have dropped any number of events
	return e.handleKeyedOrderEvent(ctx, key, cur, preEventType == jsonparser.Null, string(preEventID))
}

// handleKeyedOrderEvent handles an order event of a market and order source, recovering the events missed before it
func (e *Exchange) handleKeyedOrderEvent(ctx context.Context, key string, cur *keptOrderEvent, restart bool, preEventID string) error {
	e.orderEventMu.Lock()
	checkpoint, ok := e.orderEventCheckpoints[key]
	e.orderEventMu.Unlock()
	var errs error
	switch {
	case ok && checkpoint.eventID == cur.eventID:
		// A repeated event has nothing to recover before it, and is only relayed again if it was refused
		if !checkpoint.pending {
			return nil
		}
	case ok && (checkpoint.pending || restart || preEventID != checkpoint.eventID):
		failures := checkpoint.failures
		checkpoint, errs = e.recoverOrderEvents(ctx, key, cur.pair, cur.asset, cur.orderSource, checkpoint, cur.eventID)
		gaveUp := errors.Is(errs, errOrderEventRecovery)
		if gaveUp {
			// The API refuses a recovery range when the event is not after the checkpoint, so it is dropped rather than
			// relayed over the newer order state
			if errors.Is(errs, errInvalidEventIDRange) {
				e.removeKeptOrderEvent(key, cur.eventID)
				return fmt.Errorf("%w: %s for %s %s", errStaleOrderEvent, cur.eventID, cur.pair, cur.asset)
			}
			// The event is kept until the events missed before it are recovered, unless recovery keeps failing or the
			// API key lacks permission for it, as every later event would be held back too
			checkpoint.failures = failures + 1
			if checkpoint.failures < maxOrderEventRecoveryFailures && !errors.Is(errs, errNoPermission) {
				e.setOrderEventCheckpoint(key, checkpoint)
				e.keepOrderEvent(key, cur)
				return errs
			}
			errs = common.AppendError(errs, fmt.Errorf("%w for %s %s before event %s", errOrderEventsSkipped, cur.pair, cur.asset, cur.eventID))
		}
		// The kept events were received, so those a recovery did not return, or all of them when recovery gives up, are
		// relayed in the order they were kept, before the current one; giving up skips the events never received and
		// any dropped beyond maxKeptOrderEvents
		done, last, err := e.flushKeptOrderEvents(ctx, key, cur.eventID)
		errs = common.AppendError(errs, err)
		if last != nil {
			checkpoint = *last
		}
		if errors.Is(err, errOrderEventRecovery) {
			// The rest stay kept. After giving up, the next recovery to fail gives up at once rather than after three more
			// failures, whereas after a recovery which succeeded the count starts again
			checkpoint.failures = 0
			if gaveUp {
				checkpoint.failures = maxOrderEventRecoveryFailures - 1
			}
			e.setOrderEventCheckpoint(key, checkpoint)
			if !done {
				e.keepOrderEvent(key, cur)
			}
			return errs
		}
		if done {
			// The current event was kept before, and was relayed in its place among the kept events
			e.setOrderEventCheckpoint(key, checkpoint)
			return errs
		}
	}
	err := e.processOrderEvent(ctx, cur.event, &cur.data, cur.pair, cur.asset)
	pending := isOrderEventRetryable(err)
	e.setOrderEventCheckpoint(key, orderEventCheckpoint{eventID: cur.eventID, pending: pending})
	if pending {
		e.keepOrderEvent(key, cur)
	} else {
		e.removeKeptOrderEvent(key, cur.eventID)
	}
	return common.AppendError(errs, err)
}

// recoverOrderEvents processes the account's order events after a checkpoint and before the current event, and returns
// the checkpoint of the last event handled
// The API returns events from the since ID inclusive to the until ID exclusive, at most 1000 at a time. The error wraps
// errOrderEventRecovery when recovery stopped before reaching the current event
func (e *Exchange) recoverOrderEvents(ctx context.Context, key string, pair currency.Pair, a asset.Item, orderSource string, checkpoint orderEventCheckpoint, untilID string) (orderEventCheckpoint, error) {
	req := &OrderEventsRequest{Pair: pair, OrderSource: orderSource, SinceID: checkpoint.eventID, UntilID: untilID}
	processed := map[string]bool{checkpoint.eventID: !checkpoint.pending}
	var errs error
	for {
		events, err := e.GetAccountOrderData(ctx, req)
		if err != nil {
			return checkpoint, common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderEventRecovery, pair, a, err))
		}
		var added bool
		for i := range events {
			if processed[events[i].EventID] {
				continue
			}
			processed[events[i].EventID] = true
			added = true
			err := e.processOrderEvent(ctx, events[i].Event, &events[i].Data, pair, a)
			if isOrderEventRetryable(err) {
				// The refused event and the events fetched after it are kept, so that giving up relays them too
				recovered := make([]keptOrderEvent, 0, len(events)-i)
				for j := i; j < len(events); j++ {
					if j == i || !processed[events[j].EventID] {
						recovered = append(recovered, keptOrderEvent{eventID: events[j].EventID, event: events[j].Event, data: events[j].Data, pair: pair, asset: a, orderSource: orderSource})
					}
				}
				e.keepRecoveredOrderEvents(key, recovered)
				return orderEventCheckpoint{eventID: events[i].EventID, pending: true}, common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderEventRecovery, pair, a, err))
			}
			e.removeKeptOrderEvent(key, events[i].EventID)
			errs = common.AppendError(errs, err)
			checkpoint = orderEventCheckpoint{eventID: events[i].EventID}
		}
		if !added || len(events) < maxResultLimit {
			return checkpoint, errs
		}
		req.SinceID = events[len(events)-1].EventID
	}
}

// flushKeptOrderEvents relays the kept events of a market and order source in event order. It returns whether the event
// currentID was among them, the checkpoint of the last one relayed, and any errors, which wrap errOrderEventRecovery when
// the data handler refused an event, which keeps it and the events after it
func (e *Exchange) flushKeptOrderEvents(ctx context.Context, key, currentID string) (done bool, last *orderEventCheckpoint, errs error) {
	for {
		e.orderEventMu.Lock()
		kept := e.orderEventsKept[key]
		if len(kept) == 0 {
			e.orderEventMu.Unlock()
			return done, last, errs
		}
		k := kept[0]
		e.orderEventMu.Unlock()
		err := e.processOrderEvent(ctx, k.event, &k.data, k.pair, k.asset)
		if isOrderEventRetryable(err) {
			return done, last, common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderEventRecovery, k.pair, k.asset, err))
		}
		errs = common.AppendError(errs, err)
		e.removeKeptOrderEvent(key, k.eventID)
		last = &orderEventCheckpoint{eventID: k.eventID}
		if k.eventID == currentID {
			done = true
		}
	}
}

// keepOrderEvent keeps a live event which has not been relayed, once per event ID, after the events already kept
func (e *Exchange) keepOrderEvent(key string, k *keptOrderEvent) {
	e.orderEventMu.Lock()
	defer e.orderEventMu.Unlock()
	kept := e.orderEventsKept[key]
	if slices.ContainsFunc(kept, func(existing keptOrderEvent) bool { return existing.eventID == k.eventID }) {
		return
	}
	e.setKeptOrderEvents(key, append(slices.Clone(kept), *k))
}

// keepRecoveredOrderEvents keeps recovered events which have not been relayed ahead of the events already kept, as the
// recovery which fetched them ran up to those events
func (e *Exchange) keepRecoveredOrderEvents(key string, recovered []keptOrderEvent) {
	e.orderEventMu.Lock()
	defer e.orderEventMu.Unlock()
	combined := make([]keptOrderEvent, 0, len(recovered)+len(e.orderEventsKept[key]))
	seen := make(map[string]bool, cap(combined))
	for _, events := range [][]keptOrderEvent{recovered, e.orderEventsKept[key]} {
		for i := range events {
			if !seen[events[i].eventID] {
				seen[events[i].eventID] = true
				combined = append(combined, events[i])
			}
		}
	}
	e.setKeptOrderEvents(key, combined)
}

// setKeptOrderEvents stores the kept events of a market and order source, dropping the oldest beyond
// maxKeptOrderEvents, so that a data handler which refuses every event does not make them grow without limit; a later
// recovery still relays a dropped event, but giving up skips it
// It must be called with orderEventMu held
func (e *Exchange) setKeptOrderEvents(key string, kept []keptOrderEvent) {
	if dropped := len(kept) - maxKeptOrderEvents; dropped > 0 {
		log.Warnf(log.WebsocketMgr, "%s dropped %d kept order events for %s, which giving up on recovery will skip", e.Name, dropped, key)
		kept = kept[dropped:]
	}
	if e.orderEventsKept == nil {
		e.orderEventsKept = make(map[string][]keptOrderEvent)
	}
	e.orderEventsKept[key] = kept
}

// removeKeptOrderEvent removes a kept event once it has been relayed
func (e *Exchange) removeKeptOrderEvent(key, eventID string) {
	e.orderEventMu.Lock()
	defer e.orderEventMu.Unlock()
	kept := e.orderEventsKept[key]
	if i := slices.IndexFunc(kept, func(k keptOrderEvent) bool { return k.eventID == eventID }); i >= 0 {
		e.orderEventsKept[key] = slices.Delete(slices.Clone(kept), i, i+1)
	}
}

// retryKeptOrderEvents retries the recovery of each market and order source with kept events, up to its last kept event,
// so that kept events are relayed although no later event arrives. Every connection's heartbeat reply calls it, and it
// retries once keptOrderEventRetryInterval has passed since its last retry. A failed retry counts towards giving up, as
// each recovery holds up its read loop, and the private order events of the others, until it fails
func (e *Exchange) retryKeptOrderEvents(ctx context.Context) error {
	e.orderEventHandlingMu.Lock()
	defer e.orderEventHandlingMu.Unlock()
	if time.Since(e.keptOrderEventsRetried) < keptOrderEventRetryInterval {
		return nil
	}
	defer func() { e.keptOrderEventsRetried = time.Now() }()
	e.orderEventMu.Lock()
	keys := make([]string, 0, len(e.orderEventsKept))
	lasts := make(map[string]keptOrderEvent, len(e.orderEventsKept))
	for key, kept := range e.orderEventsKept {
		if len(kept) > 0 {
			keys = append(keys, key)
			lasts[key] = kept[len(kept)-1]
		}
	}
	e.orderEventMu.Unlock()
	slices.Sort(keys)
	var errs error
	for _, key := range keys {
		last := lasts[key]
		errs = common.AppendError(errs, e.handleKeyedOrderEvent(ctx, key, &last, false, ""))
	}
	return errs
}

// setOrderEventCheckpoint stores the last private order event handled for a market and order source
func (e *Exchange) setOrderEventCheckpoint(key string, checkpoint orderEventCheckpoint) {
	e.orderEventMu.Lock()
	defer e.orderEventMu.Unlock()
	if e.orderEventCheckpoints == nil {
		e.orderEventCheckpoints = make(map[string]orderEventCheckpoint)
	}
	e.orderEventCheckpoints[key] = checkpoint
}

// isOrderEventRetryable returns whether an order event which failed processing is retried by the next recovery
// Only events the data handler refused are retried, since an event without an order ID is recovered the same way
func isOrderEventRetryable(err error) bool {
	return err != nil && !errors.Is(err, errOrderIDMissing)
}

// processOrderEvent relays an order event as order details; a replacement also reports the replaced order as cancelled,
// in the same payload
func (e *Exchange) processOrderEvent(ctx context.Context, event string, o *OrderEventData, pair currency.Pair, a asset.Item) error {
	if o.ID == 0 && o.ClientOrderID == "" {
		return fmt.Errorf("%w: %s %+v", errOrderIDMissing, event, o)
	}

	amount := o.AmountAtCreate.Float64()
	remaining := o.RemainingAmount.Float64()
	// AmountTraded only holds the amount executed by this event, so the total executed amount is derived instead, from
	// the amounts' exact decimals, as a float can lose a digit of an amount with more than 15 significant digits
	executedDecimal := o.AmountAtCreate.Decimal().Sub(o.RemainingAmount.Decimal())
	executed := executedDecimal.InexactFloat64()

	var status order.Status
	switch event {
	case orderEventCreated, orderEventReplaced:
		status = order.New
	case orderEventChanged:
		status = order.Open
		if executed > 0 {
			status = order.PartiallyFilled
			executed = openExecutedAmount(executed, amount, remaining)
		}
	case orderEventDeleted:
		switch {
		case remaining == 0 && amount > 0:
			status = order.Filled
		case executed > 0:
			status = order.PartiallyCancelled
		default:
			status = order.Cancelled
		}
	case orderEventStopActive:
		status = order.Active
	case orderEventStopInactive:
		status = order.Cancelled
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: fmt.Sprintf("%s%s%s %+v", e.Name, websocket.UnhandledMessage, event, o)})
	}

	orderType, tif := orderTypeFromSubtypeID(o.OrderSubtype)
	details := []order.Detail{{
		Price:           o.Price.Float64(),
		Amount:          amount,
		RemainingAmount: remaining,
		ExecutedAmount:  executed,
		Exchange:        e.Name,
		ClientOrderID:   o.ClientOrderID,
		Type:            orderType,
		TimeInForce:     tif,
		Side:            o.Side.Side(),
		Status:          status,
		AssetType:       a,
		Date:            o.Microtimestamp.Time(),
		LastUpdated:     o.Microtimestamp.Time(),
		Pair:            pair,
		ReduceOnly:      o.ReduceOnly,
		TriggerPrice:    triggerPrice(o.StopPrice.Float64(), o.ActivationPrice.Float64()),
	}}
	if o.ID != 0 {
		details[0].OrderID = strconv.FormatUint(o.ID, 10)
	}
	if o.TradeAccountID != 0 {
		details[0].AccountID = strconv.FormatUint(o.TradeAccountID, 10)
	}
	if event == orderEventReplaced && o.OriginalOrderID != 0 {
		// A replacement order has a new ID, so the replaced order is reported as cancelled
		details = append(details, order.Detail{
			Exchange:    e.Name,
			OrderID:     strconv.FormatUint(o.OriginalOrderID, 10),
			Status:      order.Cancelled,
			AssetType:   a,
			Pair:        pair,
			LastUpdated: o.Microtimestamp.Time(),
		})
	}
	if len(details) > 1 {
		// Both orders are relayed in one payload, so a refused replacement is retried without repeating either of them
		if err := e.Websocket.DataHandler.Send(ctx, details); err != nil {
			return err
		}
		// The event holds only the replacement's amounts, so the replaced order's executions are read back too; a read
		// which shows the replaced order no longer open is from after the replacement
		replaced := details[1]
		e.fetchOrderTotals(ctx, &replaced, decimal.Zero)
		return nil
	}
	deleted := details[0]
	if err := e.Websocket.DataHandler.Send(ctx, &details[0]); err != nil {
		return err
	}
	if event == orderEventDeleted && executedDecimal.IsPositive() && deleted.OrderID != "" {
		e.fetchOrderTotals(ctx, &deleted, executedDecimal)
	}
	return nil
}

// fetchOrderTotals relays a deleted or replaced order's detail again with the fee, cost and average price which its
// event does not carry, and the status, executed amount and date the API reports for it, fetched off the read loop
// The order store takes an order's fee from its own fee field, not its trades, and the order manager never polls an
// order which is no longer active, so a filled order would otherwise keep a fee of zero. A read which shows the order
// still open, or less executed than its event, is older than the event and is not relayed
func (e *Exchange) fetchOrderTotals(ctx context.Context, closed *order.Detail, executed decimal.Decimal) {
	e.orderTotalFetches.Go(func() {
		e.orderTotalFetchMu.Lock()
		defer e.orderTotalFetchMu.Unlock()
		fetched, err := e.GetOrderInfo(ctx, closed.OrderID, closed.Pair, closed.AssetType)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s unable to fetch the totals of closed order %s: %v", e.Name, closed.OrderID, err)
			return
		}
		fetchedExecuted, err := gctmath.DecimalFromFloat(fetched.ExecutedAmount)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s unable to fetch the totals of closed order %s: %v", e.Name, closed.OrderID, err)
			return
		}
		switch fetched.Status {
		case order.Filled, order.Cancelled, order.PartiallyCancelled, order.Expired:
		default:
			log.Warnf(log.WebsocketMgr, "%s closed order %s is still %s, so its totals are not relayed", e.Name, closed.OrderID, fetched.Status)
			return
		}
		// The read's executed amount is a float, which can lose a digit of an amount with more than 15 significant digits,
		// so the event's exact amount is compared as a float too
		if fetched.ExecutedAmount < executed.InexactFloat64() {
			log.Warnf(log.WebsocketMgr, "%s closed order %s has executed %s, less than its event's %s, so its totals are not relayed", e.Name, closed.OrderID, fetchedExecuted, executed)
			return
		}
		if fetchedExecuted.IsZero() {
			// A replaced order which executed nothing has no totals to add
			return
		}
		// The event's last update is kept, and its trades left out, as the API's trade times are rounded to seconds
		d := *closed
		if executed.IsZero() {
			// A replaced order is relayed without its own fields, which the replacement event does not carry, so an order
			// the store does not hold yet takes its side from the read. The read's amount is executed plus remaining, and is
			// only taken while some remained, as an order the store holds would lose its amount if none is reported
			d.Side = fetched.Side
			if fetched.RemainingAmount > 0 {
				d.Amount = fetched.Amount
			}
		}
		d.Status = fetched.Status
		d.ExecutedAmount = fetched.ExecutedAmount
		d.Date = fetched.Date
		d.Fee = fetched.Fee
		d.FeeAsset = fetched.FeeAsset
		d.Cost = fetched.Cost
		d.CostAsset = fetched.CostAsset
		d.AverageExecutedPrice = fetched.AverageExecutedPrice
		d.Trades = nil
		if err := e.Websocket.DataHandler.Send(ctx, &d); err != nil {
			log.Errorf(log.WebsocketMgr, "%s unable to relay the totals of closed order %s: %v", e.Name, closed.OrderID, err)
		}
	})
}

// orderTypeFromSubtypeID returns the order type and time in force of a websocket order subtype
func orderTypeFromSubtypeID(subtype uint8) (order.Type, order.TimeInForce) {
	switch subtype {
	case 0:
		return order.Limit, order.GoodTillCancel
	case 1, 2, 7:
		return order.Market, order.UnknownTIF
	case 3:
		return order.Limit, order.GoodTillDay
	case 4:
		return order.Limit, order.ImmediateOrCancel
	case 5:
		return order.Limit, order.PostOnly
	case 6:
		return order.Limit, order.FillOrKill
	case 8:
		return order.Limit, order.GoodTillTime
	case 20:
		return order.StopMarket, order.UnknownTIF
	case 21:
		return order.TakeProfitMarket, order.UnknownTIF
	case 22:
		return order.StopLimit, order.UnknownTIF
	case 23:
		return order.TakeProfit | order.Limit, order.UnknownTIF
	case 24, 25:
		return order.TrailingStop, order.UnknownTIF
	case 26, 27:
		return order.TrailingStopLimit, order.UnknownTIF
	default:
		return order.UnknownType, order.UnknownTIF
	}
}

// handleWSMyTrade relays an execution of one of the account's orders as a fill when the fills feed is enabled
// It is not relayed as an order update, so a stored order's trades come only from reading the order over REST: an
// execution which reaches the order store before its order's events, while a gap recovery holds them back or when it
// beats a placement's reply, would store the order without a date, which no later update restores, and make the
// placement report the order as not placed
func (e *Exchange) handleWSMyTrade(data []byte, pair currency.Pair, a asset.Item) error {
	var t WebsocketMyTrade
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	side, err := order.StringToOrderSide(t.Side)
	if err != nil {
		return err
	}
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	var orderID, tradeID string
	if t.OrderID != 0 {
		orderID = strconv.FormatUint(t.OrderID, 10)
	}
	if t.ID != 0 {
		tradeID = strconv.FormatUint(t.ID, 10)
	}
	return e.Websocket.Fills.Update(fill.Data{
		Timestamp:     t.Microtimestamp.Time(),
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       orderID,
		ClientOrderID: t.ClientOrderID,
		TradeID:       tradeID,
		Price:         t.Price.Float64(),
		Amount:        t.Amount.Float64(),
	})
}

func (e *Exchange) handleWSSettlement(ctx context.Context, event string, data []byte) error {
	settlement := &WebsocketSettlement{Event: event}
	if err := json.Unmarshal(data, settlement); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, settlement)
}

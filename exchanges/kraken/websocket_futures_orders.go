package kraken

import (
	"math"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// futuresOrderRetention is how long the record of a finished futures order, or of an order known only by its fills, is
// kept after the feeds last reported it, for the fills Kraken reports late or repeats
const futuresOrderRetention = 10 * time.Minute

// futuresOrderTracker holds what the futures feeds have reported of each order. The open_orders feed names only the
// order when it leaves the open orders and reports its filled quantity only while it stays open, and the fills feed
// reports each fill on its own, so an order's executed quantity and totals are put together here
type futuresOrderTracker struct {
	mu     sync.Mutex
	orders map[string]*futuresOrderState
	// conn is the connection the feeds last reported on. ordersSnapshotSeen and fillsSnapshotSeen are set once each
	// feed's snapshot arrives on it, and cleared when the feed is unsubscribed. An order the feeds first report on it
	// while both are set, and open_orders never named, is taken as placed after both snapshots, as an order open when
	// they were taken is one open_orders named; its fills then all follow while the fills feed stays subscribed. An order
	// reported before them may have filled unseen, so it never is
	conn                                  websocket.Connection
	ordersSnapshotSeen, fillsSnapshotSeen bool
	// fillsCoverage counts the fills snapshots. Each starts the fills feed's coverage anew, as it can follow fills that
	// went unseen
	fillsCoverage uint64
}

// futuresOrderState is what the futures feeds have reported of an order
type futuresOrderState struct {
	// quantity is the order's quantity and filled its filled quantity, as open_orders last reported them; zero for an
	// order open_orders has not named. Once open_orders stops reporting an order whose quantity can still change, its
	// quantity is zero too, until open_orders names it again
	quantity, filled float64
	// open is set while open_orders reports the order open, so a snapshot leaving it out shows it has left
	open bool
	// inverse is set for an inverse contract, whose quantity is in its quote currency
	inverse bool
	// fillIDs holds the fills counted into the totals, each once, as a fills snapshot repeats recent fills
	fillIDs map[string]struct{}
	// fillQuantity totals the fills counted; value totals their quantity times price, and inverseValue their quantity
	// over price, which is an inverse contract's value in its base currency
	fillQuantity, value, inverseValue float64
	fee                               float64
	feeAsset                          currency.Code
	// mixedFees is set when the fills charged fees in more than one currency, which add up to no total
	mixedFees bool
	// liveOnly is set while the feeds first reported the order on a connection both feeds' snapshots covered, no
	// open_orders snapshot has held it, and every fill counted was reported as it happened on it within the same fills
	// coverage, so no fill of the order came before or went unseen. A fills snapshot can hold only the latest of an
	// order's fills
	liveOnly bool
	// fillsCoverage is the tracker's fills coverage when the feeds first reported the order
	fillsCoverage uint64
	// filledInFull is set once a fill leaves nothing of the order remaining
	filledInFull bool
	// reason is why open_orders removed the order, empty while it has not
	reason string
	// updated is when the feeds last reported the order
	updated time.Time
}

// get returns what the feeds have reported of an order
func (t *futuresOrderTracker) get(orderID string) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok := t.orders[orderID]; ok {
		return *s
	}
	return futuresOrderState{}
}

// snapshot records the open orders an open_orders snapshot holds, and forgets the open orders it leaves out, which
// left the open orders while the feed was not subscribed. An order it holds can have filled before the fills feed's
// snapshot, so its fills are never taken as all reported as they happened
func (t *futuresOrderTracker) snapshot(conn websocket.Connection, orders []WebsocketFuturesOpenOrder) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reportedOn(conn)
	t.ordersSnapshotSeen = true
	listed := make(map[string]bool, len(orders))
	for i := range orders {
		listed[orders[i].OrderID] = true
		t.record(conn, &orders[i]).liveOnly = false
	}
	for id, s := range t.orders {
		if s.open && !listed[id] {
			delete(t.orders, id)
		}
	}
	t.prune()
}

// fillsSnapshot records that a fills snapshot arrived on a connection, which starts the fills feed's coverage anew
func (t *futuresOrderTracker) fillsSnapshot(conn websocket.Connection) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reportedOn(conn)
	t.fillsSnapshotSeen = true
	t.fillsCoverage++
}

// reportedOn records the connection a report came on. A report on another connection than the last is the first of a
// new connection, which no snapshot has covered yet and on which open_orders has reported no order's changes
func (t *futuresOrderTracker) reportedOn(conn websocket.Connection) {
	if t.conn != conn {
		t.conn, t.ordersSnapshotSeen, t.fillsSnapshotSeen = conn, false, false
		t.forgetQuantities()
	}
}

// unsubscribed records that a feed stopped on a connection. What it reports while it is not subscribed goes unseen,
// so it covers the connection again only from its next snapshot
func (t *futuresOrderTracker) unsubscribed(conn websocket.Connection, feed string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != conn {
		return
	}
	switch feed {
	case wsFuturesFeedOpenOrders:
		t.ordersSnapshotSeen = false
		t.forgetQuantities()
	case wsFuturesFeedFills:
		t.fillsSnapshotSeen = false
	}
}

// forgetQuantities forgets the quantity open_orders last reported of each order whose quantity can still change, once
// open_orders no longer reports its changes: an open order, or a stop removed as triggered, as the order the stop places
// takes its ID. A finished order's quantity cannot change, so it is kept for the fills Kraken reports late
func (t *futuresOrderTracker) forgetQuantities() {
	for _, s := range t.orders {
		if s.open || s.reason == "stop_order_triggered" {
			s.quantity = 0
		}
	}
}

// update records an open order an open_orders update reports, returning what the feeds have reported of it
func (t *futuresOrderTracker) update(conn websocket.Connection, o *WebsocketFuturesOpenOrder) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return *t.record(conn, o)
}

// record records an open order's quantity and filled quantity. An order open_orders removed and reports open again, as
// a stop is once it triggers, is open again
func (t *futuresOrderTracker) record(conn websocket.Connection, o *WebsocketFuturesOpenOrder) *futuresOrderState {
	s := t.state(o.OrderID, conn)
	s.quantity, s.filled, s.inverse, s.open, s.reason = o.Quantity, o.Filled, inverseFuturesSymbol(o.Instrument), true, ""
	s.updated = time.Now()
	return s
}

// finish records that open_orders removed an order, and why, returning what the feeds have reported of it. An order
// not yet recorded is recorded too, as its fills can follow its removal
func (t *futuresOrderTracker) finish(conn websocket.Connection, orderID, reason string) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.prune()
	s := t.state(orderID, conn)
	s.open, s.reason, s.updated = false, reason, time.Now()
	return *s
}

// fill counts a fill into its order's totals, once whatever feed reports it again, returning what the feeds have
// reported of the order
func (t *futuresOrderTracker) fill(conn websocket.Connection, f *WebsocketFuturesFill, live bool, feeAsset currency.Code) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.prune()
	s := t.state(f.OrderID, conn)
	s.inverse = inverseFuturesSymbol(f.Instrument)
	if _, counted := s.fillIDs[f.FillID]; !counted {
		s.liveOnly = s.liveOnly && live && t.fillsCovered(conn) && s.fillsCoverage == t.fillsCoverage
		s.fillIDs[f.FillID] = struct{}{}
		switch {
		case s.fillQuantity == 0:
			s.feeAsset = feeAsset
		case !s.feeAsset.Equal(feeAsset):
			s.mixedFees = true
		}
		s.fillQuantity += f.Quantity
		// The product is rounded on its own, so arm64 cannot fuse it into the sum and change the total
		s.value += float64(f.Quantity * f.Price)
		if f.Price > 0 {
			s.inverseValue += f.Quantity / f.Price
		}
		s.fee += f.FeePaid
		s.filledInFull = s.filledInFull || f.RemainingOrderQuantity == 0
	}
	s.updated = time.Now()
	return *s
}

// state returns the record of an order reported on conn, making one when there is none
func (t *futuresOrderTracker) state(orderID string, conn websocket.Connection) *futuresOrderState {
	t.reportedOn(conn)
	if t.orders == nil {
		t.orders = make(map[string]*futuresOrderState)
	}
	s, ok := t.orders[orderID]
	if !ok {
		s = &futuresOrderState{fillIDs: make(map[string]struct{}), liveOnly: t.covered(conn), fillsCoverage: t.fillsCoverage, updated: time.Now()}
		t.orders[orderID] = s
	}
	return s
}

// covered reports whether a report on conn follows both an open_orders and a fills snapshot on it, with neither feed
// unsubscribed since
func (t *futuresOrderTracker) covered(conn websocket.Connection) bool {
	return t.fillsCovered(conn) && t.ordersSnapshotSeen
}

// fillsCovered reports whether a fill on conn follows a fills snapshot on it, with the feed not unsubscribed since
func (t *futuresOrderTracker) fillsCovered(conn websocket.Connection) bool {
	return t.conn == conn && t.fillsSnapshotSeen
}

// prune forgets the orders no longer open that the feeds have not reported for futuresOrderRetention
func (t *futuresOrderTracker) prune() {
	for id, s := range t.orders {
		if !s.open && time.Since(s.updated) > futuresOrderRetention {
			delete(t.orders, id)
		}
	}
}

// executed returns the most of the order known to have been filled: what open_orders last reported filled, what the
// fills counted add up to, and all of it once open_orders removed it as filled or a fill left nothing remaining. Each
// is a part of what filled, so the most of them is too. A fill's remaining quantity is otherwise left out, as it is of
// the quantity when it filled, which an amendment can have changed since
func (s *futuresOrderState) executed() float64 {
	executed := max(s.filled, s.fillQuantity)
	if s.reason == "full_fill" || s.filledInFull {
		executed = max(executed, s.quantity)
	}
	return executed
}

// fillStatus returns the status of an order a fill reports: as open_orders removed it, when it has, with the fills
// counted since, so a cancellation reported before its fills is partly filled once they are counted; filled, once a
// fill leaves nothing of an order whose quantities are known remaining; otherwise none, leaving the status to
// open_orders
func (s *futuresOrderState) fillStatus() order.Status {
	switch {
	case s.reason != "":
		return futuresOrderUpdateStatus(s.reason, true, s.executed())
	case s.filledInFull && (s.quantity > 0 || s.liveOnly):
		return order.Filled
	default:
		return order.UnknownStatus
	}
}

// complete reports whether the fills counted make up the order's executed quantity, so their totals are the order's
func (s *futuresOrderState) complete() bool {
	return s.fillQuantity > 0 && sameQuantity(s.fillQuantity, s.executed())
}

// addTo adds to an order update what the feeds have reported of the order. An order whose quantity open_orders has
// reported, and which the tracker has not forgotten since, gets its quantity and executed quantity; any other, such as a
// market order, which rests in no book, gets them only once its fills, all reported as they happened, have filled it in
// full, as fills alone tell neither an order's quantity nor whether earlier fills were missed. The average price, cost and fee are added once the fills counted
// make up the executed quantity, as partial totals would replace fuller ones the order store holds. An inverse
// contract's cost would be in its base currency, so it is left out
func (s *futuresOrderState) addTo(d *order.Detail) {
	// The order store keeps the fee currency of the update that first stores an order, which can be a fill's
	if !s.mixedFees && !s.feeAsset.IsEmpty() {
		d.FeeAsset = s.feeAsset
	}
	switch {
	case s.quantity > 0:
		d.Amount, d.ExecutedAmount = s.quantity, s.executed()
		d.RemainingAmount = max(d.Amount-d.ExecutedAmount, 0)
	case s.liveOnly && s.filledInFull:
		d.Amount, d.ExecutedAmount, d.RemainingAmount = s.fillQuantity, s.fillQuantity, 0
	default:
		return
	}
	if !s.complete() {
		return
	}
	if s.inverse {
		d.AverageExecutedPrice = s.fillQuantity / s.inverseValue
	} else {
		d.Cost, d.AverageExecutedPrice = s.value, s.value/s.fillQuantity
	}
	if !s.mixedFees {
		d.Fee, d.FeeAsset = s.fee, s.feeAsset
	}
}

// sameQuantity reports whether two quantities agree but for the rounding of adding fills up
func sameQuantity(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*max(math.Abs(a), math.Abs(b))
}

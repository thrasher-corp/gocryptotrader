package kraken

import (
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// futuresOrderRetention is how long the record of a finished futures order, or of an order known only by its fills, is
// kept after the feeds last reported it, for the fills Kraken reports late or repeats
const futuresOrderRetention = 10 * time.Minute

// futuresOrderTracker holds what the futures feeds have reported of each order. The open_orders feed names only the
// order when it leaves the open orders and reports its filled quantity only while it stays open, and the fills feed
// reports each fill on its own, so an order's executed quantity is put together here
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
	// fillIDs holds the fills counted, each once, as a fills snapshot repeats recent fills, and fillTotal adds up their
	// quantities as decimals, so that fills of an order's quantity add up to it: in floating point, 0.1 and 0.7 add up
	// to less than 0.8. fillQuantity is that total as a float
	fillIDs      map[string]struct{}
	fillTotal    decimal.Decimal
	fillQuantity float64
	// liveOnly is set while the feeds first reported the order on a connection both feeds' snapshots covered, no
	// open_orders snapshot has held it while some of it remained, and every fill counted was reported as it happened on
	// it within the same fills coverage, so no fill of the order came before or went unseen. A fills snapshot can hold
	// only the latest of an order's fills
	liveOnly bool
	// fillsCoverage is the tracker's fills coverage when the feeds first reported the order
	fillsCoverage uint64
	// filledInFull is set once a fill leaves nothing of the order remaining, and fillsBeforeLast is what the fills
	// counted before that fill add up to
	filledInFull    bool
	fillsBeforeLast float64
	// reports holds each quantity and filled quantity open_orders has reported of the order together, while a later fill
	// can match it. Its report of a fill carries the quantity in force when the order filled, which the fill's remaining
	// quantity is of
	reports []futuresOrderReport
	// reportedAhead is set once open_orders has reported more of the order filled than the fills counted add up to,
	// which a fill that went unseen explains as well as one the fills feed has yet to report. A later fill clears it
	// once fillsComplete takes the fills counted as complete, as it does once the fill a report was ahead of arrives
	reportedAhead bool
	// reason is why open_orders removed the order, empty while it has not
	reason string
	// updated is when the feeds last reported the order
	updated time.Time
}

// futuresOrderReport is an order's quantity and filled quantity as one open_orders report gave them
type futuresOrderReport struct {
	quantity, filled float64
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
// left the open orders while the feed was not subscribed, keeping one a fill has left nothing of as finished, for its
// later reports. An order it holds can have filled before the fills feed's snapshot, so its fills are no longer taken
// as all reported as they happened, unless they already left nothing of it remaining, as nothing more of it can then
// execute
func (t *futuresOrderTracker) snapshot(conn websocket.Connection, orders []WebsocketFuturesOpenOrder) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reportedOn(conn)
	t.ordersSnapshotSeen = true
	listed := make(map[string]bool, len(orders))
	for i := range orders {
		listed[orders[i].OrderID] = true
		s := t.record(conn, &orders[i])
		s.liveOnly = s.liveOnly && s.filledInFull
		s.forgetReports()
	}
	for id, s := range t.orders {
		switch {
		case !s.open || listed[id]:
		case s.filledInFull:
			s.open = false
		default:
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
// takes its ID, until a fill leaves nothing of the order remaining. A finished order's quantity cannot change, so it is
// kept for the reports Kraken makes late
func (t *futuresOrderTracker) forgetQuantities() {
	for _, s := range t.orders {
		if (s.open || s.reason == "stop_order_triggered") && !s.filledInFull {
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
	s.quantity, s.filled, s.open, s.reason = o.Quantity, o.Filled, true, ""
	if r := (futuresOrderReport{o.Quantity, o.Filled}); !slices.Contains(s.reports, r) {
		s.reports = append(s.reports, r)
		s.forgetReports()
	}
	s.reportedAhead = s.reportedAhead || o.Filled > s.fillQuantity
	s.updated = time.Now()
	return s
}

// finish records that open_orders removed an order, and why, returning what the feeds have reported of it. An order
// not yet recorded is recorded too, as its fills can follow its removal. An edit's rejection names an order it cannot
// find, so an order open_orders has already removed keeps the reason it was removed for
func (t *futuresOrderTracker) finish(conn websocket.Connection, orderID, reason string) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.prune()
	s := t.state(orderID, conn)
	if reason != "order_for_edit_not_found" || s.reason == "" {
		s.reason = reason
	}
	s.open, s.updated = false, time.Now()
	return *s
}

// fill counts a fill towards its order's executed quantity, once whatever feed reports it again, returning what the
// feeds have reported of the order
func (t *futuresOrderTracker) fill(conn websocket.Connection, f *WebsocketFuturesFill, live bool) futuresOrderState {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.prune()
	s := t.state(f.OrderID, conn)
	if _, counted := s.fillIDs[f.FillID]; !counted {
		s.liveOnly = s.liveOnly && live && t.fillsCovered(conn) && s.fillsCoverage == t.fillsCoverage
		s.fillIDs[f.FillID] = struct{}{}
		if f.RemainingOrderQuantity == 0 && !s.filledInFull {
			s.filledInFull, s.fillsBeforeLast = true, s.fillQuantity
		}
		s.fillTotal = s.fillTotal.Add(decimal.MustFromFloat(f.Quantity))
		s.fillQuantity = s.fillTotal.InexactFloat64()
		s.reportedAhead = s.reportedAhead && !s.fillsComplete(f.RemainingOrderQuantity)
		s.forgetReports()
	}
	s.updated = time.Now()
	return *s
}

// fillsComplete reports whether the fills counted, up to one leaving remaining of the order, reach what open_orders
// last reported filled and match its report of that fill: one of the order filled by what they add up to, at a quantity
// of which the fill leaves the rest. An unseen fill is in every filled quantity open_orders reports after it, so fills
// that miss one match no report, unless they add up to an earlier report's filled quantity and an amendment
// open_orders has yet to report makes the fill's remaining quantity agree with that report's quantity
func (s *futuresOrderState) fillsComplete(remaining float64) bool {
	return s.fillQuantity >= s.filled && slices.ContainsFunc(s.reports, func(r futuresOrderReport) bool {
		return decimal.MustFromFloat(r.filled).Equal(s.fillTotal) && decimal.MustFromFloat(r.quantity).Sub(decimal.MustFromFloat(remaining)).Equal(s.fillTotal)
	})
}

// forgetReports forgets the reports no later fill can match: every report once the order's fills are not all taken as
// reported as they happened, as fillsMakeUpQuantity then never takes them, and otherwise each report of what the fills
// counted already add up to, or less
func (s *futuresOrderState) forgetReports() {
	if s.liveOnly {
		s.reports = slices.DeleteFunc(s.reports, func(r futuresOrderReport) bool {
			return !decimal.MustFromFloat(r.filled).GreaterThan(s.fillTotal)
		})
	}
	if !s.liveOnly || len(s.reports) == 0 {
		s.reports = nil
	}
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

// knownFilled reports whether a fill has left nothing of an order whose quantities are known remaining. Later reports
// of the order leave it filled: open_orders can report its earlier changes, or the trigger of the stop that placed it,
// after the fills feed reports the fill, and an edit arriving after it is rejected as the order cannot be found
func (s *futuresOrderState) knownFilled() bool {
	return s.filledInFull && (s.quantity > 0 || s.liveOnly)
}

// reportStatus returns the status of an order open_orders reports for a reason: filled once a fill has left nothing of
// an order whose quantities are known remaining, otherwise the status the reason gives
func (s *futuresOrderState) reportStatus(reason string, isCancel bool) order.Status {
	if s.knownFilled() {
		return order.Filled
	}
	return futuresOrderUpdateStatus(reason, isCancel, s.executed())
}

// fillStatus returns the status of an order a fill reports: filled, once a fill leaves nothing of an order whose
// quantities are known remaining; as open_orders removed it, when it has, with the fills counted since, so a
// cancellation reported before its fills is partly filled once they are counted; otherwise none, leaving the status
// to open_orders
func (s *futuresOrderState) fillStatus() order.Status {
	switch {
	case s.knownFilled():
		return order.Filled
	case s.reason != "":
		return futuresOrderUpdateStatus(s.reason, true, s.executed())
	default:
		return order.UnknownStatus
	}
}

// fillsMakeUpQuantity reports whether an order whose fills the tracker takes as all reported as they happened, and
// which they have filled in full, takes its quantity from them, as open_orders can report an amendment of its quantity
// after the fill that left nothing remaining. The fills feed reports an order's fills in turn, so open_orders reporting
// more of it filled than the fills before that one add up to shows that a fill went unseen, as one of an order placed
// between the two feeds' snapshots can. Reporting more filled than the fills counted at the time can show that too, so
// the reported quantity is taken unless fillsComplete takes a later fill's fills as complete, as when the fills feed
// was only behind, or the fills reach the reported quantity
func (s *futuresOrderState) fillsMakeUpQuantity() bool {
	return s.liveOnly && s.filledInFull && s.fillsBeforeLast >= s.filled && (!s.reportedAhead || s.fillQuantity >= s.quantity)
}

// addTo adds to an order update the order's quantities, as far as the feeds have reported them. An order whose fills
// make up its quantity gets them from its fills. Any other order whose quantity open_orders has reported, and which the
// tracker has not forgotten since, gets its quantity and executed quantity; the rest, such as a market order, which
// rests in no book, get none, as fills alone tell neither an order's quantity nor whether earlier fills were missed.
// The order's average price, cost and fee are left unset, each fill's price and fee staying on its trade: the fills
// counted can add up to all the feeds have reported filled while fills from before the feeds subscribed, which the
// fills snapshot can leave out, are missing, as when a fill is reported before open_orders reports it, and Kraken's
// futures order reads carry no totals that would correct partial ones
func (s *futuresOrderState) addTo(d *order.Detail) {
	switch {
	case s.fillsMakeUpQuantity():
		d.Amount, d.ExecutedAmount, d.RemainingAmount = s.fillQuantity, s.fillQuantity, 0
	case s.quantity > 0:
		d.Amount, d.ExecutedAmount = s.quantity, s.executed()
		d.RemainingAmount = max(d.Amount-d.ExecutedAmount, 0)
	}
}

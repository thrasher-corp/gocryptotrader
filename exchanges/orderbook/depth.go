package orderbook

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Depth defines a linked list of orderbook items
type Depth struct {
	asks
	bids

	// unexported stack of nodes
	stack *stack

	Alert

	mux *dispatch.Mux
	id  uuid.UUID

	options
	m sync.Mutex
}

// NewDepth returns a new depth item
func newDepth(id uuid.UUID) *Depth {
	return &Depth{
		stack: newStack(),
		id:    id,
		mux:   service.Mux,
	}
}

// Publish alerts any subscribed routines using a dispatch mux
func (d *Depth) Publish() {
	err := d.mux.Publish([]uuid.UUID{d.id}, d.Retrieve())
	if err != nil {
		log.Errorf(log.ExchangeSys, "Cannot publish orderbook update to mux %v", err)
	}
}

// GetAskLength returns length of asks
func (d *Depth) GetAskLength() int {
	d.m.Lock()
	defer d.m.Unlock()
	return d.asks.length
}

// GetBidLength returns length of bids
func (d *Depth) GetBidLength() int {
	d.m.Lock()
	defer d.m.Unlock()
	return d.bids.length
}

// Retrieve returns the orderbook base a copy of the underlying linked list
// spread
func (d *Depth) Retrieve() *Base {
	d.m.Lock()
	defer d.m.Unlock()
	return &Base{
		Bids:             d.bids.retrieve(),
		Asks:             d.asks.retrieve(),
		Exchange:         d.exchange,
		Asset:            d.asset,
		Pair:             d.pair,
		LastUpdated:      d.lastUpdated,
		LastUpdateID:     d.lastUpdateID,
		PriceDuplication: d.priceDuplication,
		IsFundingRate:    d.isFundingRate,
		VerifyOrderbook:  d.VerifyOrderbook,
	}
}

// TotalBidAmounts returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidAmounts() (liquidity, value float64) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.bids.amount()
}

// TotalAskAmounts returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAskAmounts() (liquidity, value float64) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.asks.amount()
}

// LoadSnapshot flushes the bids and asks with a snapshot
func (d *Depth) LoadSnapshot(bids, asks []Item) {
	d.m.Lock()
	d.bids.load(bids, d.stack)
	d.asks.load(asks, d.stack)
	d.alert()
	d.m.Unlock()
}

// Flush flushes the bid and ask depths
func (d *Depth) Flush() {
	d.m.Lock()
	d.bids.load(nil, d.stack)
	d.asks.load(nil, d.stack)
	d.alert()
	d.m.Unlock()
}

// UpdateBidAskByPrice updates the bid and ask spread by supplied updates, this
// will trim total length of depth level to a specified supplied number
func (d *Depth) UpdateBidAskByPrice(bidUpdts, askUpdts Items, maxDepth int) {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return
	}
	d.m.Lock()
	tn := getNow()
	if len(bidUpdts) != 0 {
		d.bids.updateInsertByPrice(bidUpdts, d.stack, maxDepth, tn)
	}
	if len(askUpdts) != 0 {
		d.asks.updateInsertByPrice(askUpdts, d.stack, maxDepth, tn)
	}
	d.alert()
	d.m.Unlock()
}

// UpdateBidAskByID amends details by ID
func (d *Depth) UpdateBidAskByID(bidUpdts, askUpdts Items) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.m.Lock()
	defer d.m.Unlock()
	if len(bidUpdts) != 0 {
		err := d.bids.updateByID(bidUpdts)
		if err != nil {
			return err
		}
	}
	if len(askUpdts) != 0 {
		err := d.asks.updateByID(askUpdts)
		if err != nil {
			return err
		}
	}
	d.alert()
	return nil
}

// DeleteBidAskByID deletes a price level by ID
func (d *Depth) DeleteBidAskByID(bidUpdts, askUpdts Items, bypassErr bool) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.m.Lock()
	defer d.m.Unlock()
	if len(bidUpdts) != 0 {
		err := d.bids.deleteByID(bidUpdts, d.stack, bypassErr)
		if err != nil {
			return err
		}
	}
	if len(askUpdts) != 0 {
		err := d.asks.deleteByID(askUpdts, d.stack, bypassErr)
		if err != nil {
			return err
		}
	}
	d.alert()
	return nil
}

// InsertBidAskByID inserts new updates
func (d *Depth) InsertBidAskByID(bidUpdts, askUpdts Items) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.m.Lock()
	defer d.m.Unlock()
	if len(bidUpdts) != 0 {
		err := d.bids.insertUpdates(bidUpdts, d.stack)
		if err != nil {
			return err
		}
	}
	if len(askUpdts) != 0 {
		err := d.asks.insertUpdates(askUpdts, d.stack)
		if err != nil {
			return err
		}
	}
	d.alert()
	return nil
}

// UpdateInsertByID updates or inserts by ID at current price level.
func (d *Depth) UpdateInsertByID(bidUpdts, askUpdts Items) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.m.Lock()
	defer d.m.Unlock()
	if len(bidUpdts) != 0 {
		err := d.bids.updateInsertByID(bidUpdts, d.stack)
		if err != nil {
			return err
		}
	}
	if len(askUpdts) != 0 {
		err := d.asks.updateInsertByID(askUpdts, d.stack)
		if err != nil {
			return err
		}
	}
	d.alert()
	return nil
}

// AssignOptions assigns the initial options for the depth instance
func (d *Depth) AssignOptions(b *Base) {
	d.m.Lock()
	d.options = options{
		exchange:         b.Exchange,
		pair:             b.Pair,
		asset:            b.Asset,
		lastUpdated:      b.LastUpdated,
		lastUpdateID:     b.LastUpdateID,
		priceDuplication: b.PriceDuplication,
		isFundingRate:    b.IsFundingRate,
		VerifyOrderbook:  b.VerifyOrderbook,
		restSnapshot:     b.RestSnapshot,
		idAligned:        b.IDAlignment,
	}
	d.m.Unlock()
}

// SetLastUpdate sets details of last update information
func (d *Depth) SetLastUpdate(lastUpdate time.Time, lastUpdateID int64, updateByREST bool) {
	d.m.Lock()
	d.lastUpdated = lastUpdate
	d.lastUpdateID = lastUpdateID
	d.restSnapshot = updateByREST
	d.m.Unlock()
}

// GetName returns name of exchange
func (d *Depth) GetName() string {
	d.m.Lock()
	defer d.m.Unlock()
	return d.exchange
}

// IsRestSnapshot returns if the depth item was updated via REST
func (d *Depth) IsRestSnapshot() bool {
	d.m.Lock()
	defer d.m.Unlock()
	return d.restSnapshot
}

// LastUpdateID returns the last Update ID
func (d *Depth) LastUpdateID() int64 {
	d.m.Lock()
	defer d.m.Unlock()
	return d.lastUpdateID
}

// IsFundingRate returns if the depth is a funding rate
func (d *Depth) IsFundingRate() bool {
	d.m.Lock()
	defer d.m.Unlock()
	return d.isFundingRate
}

// Alert defines fields required to alert sub-systems of a change of state to
// re-check depth list
type Alert struct {
	// Channel to wait for an alert on.
	forAlert chan struct{}
	// Lets the updater functions know if there are any routines waiting for an
	// alert.
	sema uint32
	// After closing the forAlert channel this will notify when all the routines
	// that have waited, have either checked the orderbook depth or finished.
	wg sync.WaitGroup
	// Segregated lock only for waiting routines, so as this does not interfere
	// with the main depth lock, acts as a rolling gate.
	m sync.Mutex
}

// alert establishes a state change on the orderbook depth.
func (a *Alert) alert() {
	// CompareAndSwap is used to swap from 1 -> 2 so we don't keep actuating
	// the opposing compare and swap in method wait. This function can return
	// freely when an alert operation is in process.
	if !atomic.CompareAndSwapUint32(&a.sema, 1, 2) {
		// Return if no waiting routines or currently alerting.
		return
	}

	go func() {
		// Actuate lock in a different routine, as alerting is a second order
		// priority compared to updating and releasing calling routine.
		a.m.Lock()
		// Closing; alerts many waiting routines.
		close(a.forAlert)
		// Wait for waiting routines to receive alert and return.
		a.wg.Wait()
		atomic.SwapUint32(&a.sema, 0) // Swap back to neutral state.
		a.m.Unlock()
	}()
}

// Wait pauses calling routine until depth change has been established via depth
// method alert. Kick allows for cancellation of waiting or when the caller has
// has been shut down, if this is not needed it can be set to nil. This
// returns a channel so strategies can cleanly wait on a select statement case.
func (a *Alert) Wait(kick <-chan struct{}) <-chan bool {
	reply := make(chan bool)
	a.m.Lock()
	a.wg.Add(1)
	if atomic.CompareAndSwapUint32(&a.sema, 0, 1) {
		a.forAlert = make(chan struct{})
	}
	go a.hold(reply, kick)
	a.m.Unlock()
	return reply
}

// hold waits on either channel in the event that the routine has finished or an
// alert from a depth update has occurred.
func (a *Alert) hold(ch chan<- bool, kick <-chan struct{}) {
	select {
	// In a select statement, if by chance there is no receiver or its late,
	// we can still close and return, limiting dead-lock potential.
	case <-a.forAlert: // Main waiting channel from alert
		select {
		case ch <- false:
		default:
		}
	case <-kick: // This can be nil.
		select {
		case ch <- true:
		default:
		}
	}
	a.wg.Done()
	close(ch)
}

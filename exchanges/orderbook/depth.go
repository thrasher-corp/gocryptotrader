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

	// Change of state to re-check depth list
	wait    chan struct{}
	waiting uint32
	wg      sync.WaitGroup
	wMtx    sync.Mutex
	// -----

	mux *dispatch.Mux
	id  uuid.UUID

	options
	sync.Mutex
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
	d.Lock()
	defer d.Unlock()
	return d.asks.length
}

// GetBidLength returns length of bids
func (d *Depth) GetBidLength() int {
	d.Lock()
	defer d.Unlock()
	return d.bids.length
}

// Retrieve returns the orderbook base a copy of the underlying linked list
// spread
func (d *Depth) Retrieve() *Base {
	d.Lock()
	defer d.Unlock()
	return &Base{
		Bids:          d.bids.retrieve(),
		Asks:          d.asks.retrieve(),
		Exchange:      d.exchange,
		Asset:         d.asset,
		Pair:          d.pair,
		LastUpdated:   d.lastUpdated,
		LastUpdateID:  d.lastUpdateID,
		NotAggregated: d.notAggregated,
		IsFundingRate: d.isFundingRate,
	}
}

// TotalBidAmounts returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidAmounts() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.bids.amount()
}

// TotalAskAmounts returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAskAmounts() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.asks.amount()
}

// LoadSnapshot flushes the bids and asks with a snapshot
func (d *Depth) LoadSnapshot(bids, asks []Item) {
	d.Lock()
	d.bids.load(bids, d.stack)
	d.asks.load(asks, d.stack)
	d.alert()
	d.Unlock()
}

// Flush flushes the bid and ask depths
func (d *Depth) Flush() {
	d.Lock()
	d.flush()
	d.Unlock()
}

// flush will pop entire bid and ask node chain onto stack when invalidated or
// required for full flush when resubscribing
func (d *Depth) flush() {
	d.bids.load(nil, d.stack)
	d.asks.load(nil, d.stack)
}

// UpdateBidAskByPrice updates the bid and ask spread by supplied updates, this
// will trim total length of depth level to a specified supplied number
func (d *Depth) UpdateBidAskByPrice(bidUpdts, askUpdts Items, maxDepth int) {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return
	}
	d.Lock()
	if len(bidUpdts) != 0 {
		d.bids.updateInsertByPrice(bidUpdts, d.stack, maxDepth)
	}
	if len(askUpdts) != 0 {
		d.asks.updateInsertByPrice(askUpdts, d.stack, maxDepth)
	}
	d.alert()
	d.Unlock()
}

// UpdateBidAskByID amends details by ID
func (d *Depth) UpdateBidAskByID(bidUpdts, askUpdts Items) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.Lock()
	defer d.Unlock()
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
	d.Lock()
	defer d.Unlock()
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
	d.Lock()
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
	d.Unlock()
	return nil
}

// UpdateInsertByID updates or inserts by ID at current price level.
func (d *Depth) UpdateInsertByID(bidUpdts, askUpdts Items) error {
	if len(bidUpdts) == 0 && len(askUpdts) == 0 {
		return nil
	}
	d.Lock()
	defer d.Unlock()
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

// POC: alert establishes state change for depth to all waiting routines
func (d *Depth) alert() {
	if !atomic.CompareAndSwapUint32(&d.waiting, 1, 0) {
		// return if no waiting routines
		return
	}
	d.wMtx.Lock()
	go func() {
		close(d.wait)
		d.wg.Wait()
		d.wait = make(chan struct{})
		d.wMtx.Unlock()
	}()
}

// POC: kicker defines a channel that allows a system to kick routine away from
// waiting for a change on the linked list
type kicker chan struct{}

// POC: timeInForce allows a kick
func timeInForce(t time.Duration) kicker {
	ch := make(chan struct{})
	go func(ch chan<- struct{}) {
		time.Sleep(t)
		close(ch)
	}(ch)
	return ch
}

// Wait pauses routine until depth change has been established (POC)
func (d *Depth) Wait(kick <-chan struct{}) bool {
	d.wMtx.Lock()
	if atomic.CompareAndSwapUint32(&d.waiting, 0, 1) {
		d.wait = make(chan struct{})
	}
	d.wg.Add(1)
	defer d.wg.Done()
	d.wMtx.Unlock()
	select {
	case <-d.wait:
		return true
	case <-kick:
		return false
	}
}

// AssignOptions assigns the initial options for the depth instance
func (d *Depth) AssignOptions(b *Base) {
	d.Lock()
	d.options = options{
		exchange:              b.Exchange,
		pair:                  b.Pair,
		asset:                 b.Asset,
		lastUpdated:           b.LastUpdated,
		lastUpdateID:          b.LastUpdateID,
		notAggregated:         b.NotAggregated,
		isFundingRate:         b.IsFundingRate,
		verificationBypass:    b.VerificationBypass,
		hasChecksumValidation: b.HasChecksumValidation,
		restSnapshot:          b.RestSnapshot,
		idAligned:             b.IDAlignment,
	}
	d.Unlock()
}

// SetLastUpdate sets details of last update information
func (d *Depth) SetLastUpdate(lastUpdate time.Time, lastUpdateID int64, updateByREST bool) {
	d.Lock()
	d.lastUpdated = lastUpdate
	d.lastUpdateID = lastUpdateID
	d.restSnapshot = updateByREST
	d.Unlock()
}

// GetName returns name of exchange
func (d *Depth) GetName() string {
	d.Lock()
	defer d.Unlock()
	return d.exchange
}

// IsRestSnapshot returns if the depth item was updated via REST
func (d *Depth) IsRestSnapshot() bool {
	d.Lock()
	defer d.Unlock()
	return d.restSnapshot
}

// LastUpdateID returns the last Update ID
func (d *Depth) LastUpdateID() int64 {
	d.Lock()
	defer d.Unlock()
	return d.lastUpdateID
}

// IsFundingRate returns if the depth is a funding rate
func (d *Depth) IsFundingRate() bool {
	d.Lock()
	defer d.Unlock()
	return d.isFundingRate
}

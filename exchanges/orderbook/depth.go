package orderbook

import (
	"sync"
	"sync/atomic"
	"time"
)

// Depth defines a linked list of orderbook items
type Depth struct {
	asks
	bids

	// unexported stack of nodes
	stack Stack

	// Change of state to re-check depth list
	wait    chan struct{}
	waiting uint32
	wMtx    sync.Mutex
	// -----

	options
	sync.Mutex
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
		Exchange:      d.Exchange,
		Asset:         d.Asset,
		Pair:          d.Pair,
		LastUpdated:   d.LastUpdated,
		LastUpdateID:  d.LastUpdateID,
		NotAggregated: d.NotAggregated,
		IsFundingRate: d.IsFundingRate,
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
func (d *Depth) LoadSnapshot(bids, asks []Item, REST bool) error {
	d.Lock()
	defer d.Unlock()
	d.bids.load(bids, &d.stack)
	d.asks.load(asks, &d.stack)
	d.alert()
	return nil
}

// Flush attempts to flush bid and ask sides
func (d *Depth) Flush() {
	d.Lock()
	d.flush()
	d.Unlock()
}

// Process processes incoming orderbook snapshots
func (d *Depth) Process(bids, asks Items) {
	d.Lock()
	d.bids.load(bids, &d.stack)
	d.asks.load(asks, &d.stack)
	d.alert()
	d.Unlock()
}

// flush will pop entire bid and ask node chain onto stack when invalidated or
// required for full flush when resubscribing
func (d *Depth) flush() {
	d.bids.load(nil, &d.stack)
	d.asks.load(nil, &d.stack)
}

// UpdateBidAskByPrice updates the bid and ask spread by supplied updates
func (d *Depth) UpdateBidAskByPrice(bid, ask Items, maxDepth int) error {
	d.Lock()
	d.bids.updateInsertByPrice(bid, &d.stack, maxDepth)
	d.asks.updateInsertByPrice(ask, &d.stack, maxDepth)
	d.alert()
	d.Unlock()
	return nil
}

// UpdateBidAskByID amends details by ID
func (d *Depth) UpdateBidAskByID(bid, ask Items) error {
	d.Lock()
	defer d.Unlock()
	err := d.bids.updateByID(bid)
	if err != nil {
		return err
	}

	err = d.asks.updateByID(ask)
	if err != nil {
		return err
	}
	d.alert()
	return nil
}

// DeleteBidAskByID deletes a price level by ID
func (d *Depth) DeleteBidAskByID(bid, ask Items, bypassErr bool) error {
	d.Lock()
	defer d.Unlock()

	err := d.bids.deleteByID(bid, &d.stack, bypassErr)
	if err != nil {
		return err
	}

	err = d.asks.deleteByID(ask, &d.stack, bypassErr)
	if err != nil {
		return err
	}

	d.alert()
	return nil
}

// InsertBidAskByID inserts new updates
func (d *Depth) InsertBidAskByID(bid, ask Items) {
	d.Lock()
	d.bids.insertUpdates(bid, &d.stack)
	d.asks.insertUpdates(ask, &d.stack)
	d.alert()
	d.Unlock()
}

// UpdateInsertByID ...
func (d *Depth) UpdateInsertByID(bidUpdates, askUpdates Items) {
	d.Lock()
	d.bids.updateInsertByID(bidUpdates, &d.stack)
	d.asks.updateInsertByID(askUpdates, &d.stack)
	d.alert()
	d.Unlock()
}

// POC: alert establishes state change for depth to all waiting routines
func (d *Depth) alert() {
	if !atomic.CompareAndSwapUint32(&d.waiting, 1, 0) {
		// return if no waiting routines
		return
	}
	go func() {
		d.wMtx.Lock()
		close(d.wait)
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
	if d.wait == nil {
		d.wait = make(chan struct{})
	}
	atomic.StoreUint32(&d.waiting, 1)
	d.wMtx.Unlock()
	select {
	case <-d.wait:
		return true
	case <-kick:
		return false
	}
}

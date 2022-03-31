package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// ErrOrderbookInvalid defines an error for when the orderbook is invalid and
// should not be trusted
var ErrOrderbookInvalid = errors.New("orderbook data integrity compromised")

var errInvalidAction = errors.New("invalid action")

// Depth defines a linked list of orderbook items
type Depth struct {
	asks
	bids

	// unexported stack of nodes
	stack *stack

	alert.Notice

	mux *dispatch.Mux
	_ID uuid.UUID

	options

	// validationError defines current book state and why it was invalidated.
	validationError error

	m sync.Mutex
}

// NewDepth returns a new depth item
func NewDepth(id uuid.UUID) *Depth {
	return &Depth{
		stack: newStack(),
		_ID:   id,
		mux:   service.Mux,
	}
}

// Publish alerts any subscribed routines using a dispatch mux
func (d *Depth) Publish() {
	if err := d.mux.Publish(d, d._ID); err != nil {
		log.Errorf(log.ExchangeSys, "Cannot publish orderbook update to mux %v", err)
	}
}

// GetAskLength returns length of asks
func (d *Depth) GetAskLength() (int, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.asks.length, nil
}

// GetBidLength returns length of bids
func (d *Depth) GetBidLength() (int, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.bids.length, nil
}

// Retrieve returns the orderbook base a copy of the underlying linked list
// spread
func (d *Depth) Retrieve() (*Base, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
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
	}, nil
}

// TotalBidAmounts returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidAmounts() (liquidity, value float64, err error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return 0, 0, d.validationError
	}
	liquidity, value = d.bids.amount()
	return liquidity, value, nil
}

// TotalAskAmounts returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAskAmounts() (liquidity, value float64, err error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return 0, 0, d.validationError
	}
	liquidity, value = d.asks.amount()
	return liquidity, value, nil
}

// LoadSnapshot flushes the bids and asks with a snapshot
func (d *Depth) LoadSnapshot(bids, asks []Item, lastUpdateID int64, lastUpdated time.Time, updateByREST bool) {
	d.m.Lock()
	d.lastUpdateID = lastUpdateID
	d.lastUpdated = lastUpdated
	d.restSnapshot = updateByREST
	d.bids.load(bids, d.stack)
	d.asks.load(asks, d.stack)
	d.validationError = nil
	d.Alert()
	d.m.Unlock()
}

// invalidate flushes all values back to zero so as to not allow strategy
// traversal on compromised data.
func (d *Depth) invalidate(withReason error) error {
	d.lastUpdateID = 0
	d.lastUpdated = time.Time{}
	d.bids.load(nil, d.stack)
	d.asks.load(nil, d.stack)
	d.validationError = fmt.Errorf("%s %s %s %w Reason: [%v]",
		d.exchange,
		d.pair,
		d.asset,
		ErrOrderbookInvalid,
		withReason)
	d.Alert()
	return d.validationError
}

// Invalidate flushes all values back to zero so as to not allow strategy
// traversal on compromised data.
func (d *Depth) Invalidate(withReason error) error {
	d.m.Lock()
	defer d.m.Unlock()
	return d.invalidate(withReason)
}

// IsValid returns if the underlying book is valid.
func (d *Depth) IsValid() bool {
	d.m.Lock()
	valid := d.validationError == nil
	d.m.Unlock()
	return valid
}

// UpdateBidAskByPrice updates the bid and ask spread by supplied updates, this
// will trim total length of depth level to a specified supplied number
func (d *Depth) UpdateBidAskByPrice(update *Update) {
	tn := getNow()
	d.m.Lock()
	if len(update.Bids) != 0 {
		d.bids.updateInsertByPrice(update.Bids, d.stack, update.MaxDepth, tn)
	}
	if len(update.Asks) != 0 {
		d.asks.updateInsertByPrice(update.Asks, d.stack, update.MaxDepth, tn)
	}
	d.updateAndAlert(update)
	d.m.Unlock()
}

// UpdateBidAskByID amends details by ID
func (d *Depth) UpdateBidAskByID(update *Update) error {
	d.m.Lock()
	defer d.m.Unlock()
	if len(update.Bids) != 0 {
		err := d.bids.updateByID(update.Bids)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.asks.updateByID(update.Asks)
		if err != nil {
			return d.invalidate(err)
		}
	}
	d.updateAndAlert(update)
	return nil
}

// DeleteBidAskByID deletes a price level by ID
func (d *Depth) DeleteBidAskByID(update *Update, bypassErr bool) error {
	d.m.Lock()
	defer d.m.Unlock()
	if len(update.Bids) != 0 {
		err := d.bids.deleteByID(update.Bids, d.stack, bypassErr)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.asks.deleteByID(update.Asks, d.stack, bypassErr)
		if err != nil {
			return d.invalidate(err)
		}
	}
	d.updateAndAlert(update)
	return nil
}

// InsertBidAskByID inserts new updates
func (d *Depth) InsertBidAskByID(update *Update) error {
	d.m.Lock()
	defer d.m.Unlock()
	if len(update.Bids) != 0 {
		err := d.bids.insertUpdates(update.Bids, d.stack)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.asks.insertUpdates(update.Asks, d.stack)
		if err != nil {
			return d.invalidate(err)
		}
	}
	d.updateAndAlert(update)
	return nil
}

// UpdateInsertByID updates or inserts by ID at current price level.
func (d *Depth) UpdateInsertByID(update *Update) error {
	d.m.Lock()
	defer d.m.Unlock()
	if len(update.Bids) != 0 {
		err := d.bids.updateInsertByID(update.Bids, d.stack)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.asks.updateInsertByID(update.Asks, d.stack)
		if err != nil {
			return d.invalidate(err)
		}
	}
	d.updateAndAlert(update)
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

// GetName returns name of exchange
func (d *Depth) GetName() string {
	d.m.Lock()
	defer d.m.Unlock()
	return d.exchange
}

// IsRESTSnapshot returns if the depth item was updated via REST
func (d *Depth) IsRESTSnapshot() (bool, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return false, d.validationError
	}
	return d.restSnapshot, nil
}

// LastUpdateID returns the last Update ID
func (d *Depth) LastUpdateID() (int64, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.lastUpdateID, nil
}

// IsFundingRate returns if the depth is a funding rate
func (d *Depth) IsFundingRate() bool {
	d.m.Lock()
	defer d.m.Unlock()
	return d.isFundingRate
}

// updateAndAlert updates the last updated ID and when it was updated to the
// recent update. Then alerts all pending routines.
func (d *Depth) updateAndAlert(update *Update) {
	d.lastUpdateID = update.UpdateID
	d.lastUpdated = update.UpdateTime
	d.Alert()
}

// GetActionFromString matches a string action to an internal action.
func GetActionFromString(s string) (Action, error) {
	switch s {
	case "update":
		return Amend, nil
	case "delete":
		return Delete, nil
	case "insert":
		return Insert, nil
	case "update/insert":
		return UpdateInsert, nil
	}
	return 0, fmt.Errorf("%s %w", s, errInvalidAction)
}

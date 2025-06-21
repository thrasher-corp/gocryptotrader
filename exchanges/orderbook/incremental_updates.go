package orderbook

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ActionType defines the behaviour of an orderbook update
type ActionType uint8

// ActionType constants for use with ProcessUpdate
const (
	UnknownAction ActionType = iota
	InsertAction
	UpdateOrInsertAction
	UpdateAction
	DeleteAction
)

// Public error vars
var (
	ErrDepthNotFound = errors.New("orderbook depth not found")
	ErrEmptyUpdate   = errors.New("update contains no bids or asks")
)

var (
	errInvalidAction          = errors.New("invalid action")
	errUpdateFailed           = errors.New("orderbook update failed")
	errDeleteFailed           = errors.New("orderbook update delete failed")
	errRESTSnapshot           = errors.New("cannot update REST protocol loaded snapshot")
	errChecksumMismatch       = errors.New("checksum mismatch")
	errChecksumGeneratorUnset = errors.New("checksum generator unset")
)

// Update holds changes that are to be applied to a stored orderbook
type Update struct {
	UpdateID   int64
	UpdateTime time.Time
	LastPushed time.Time
	Asset      asset.Item
	Bids       Levels
	Asks       Levels
	Pair       currency.Pair

	// ExpectedChecksum defines the expected value when the books have been verified
	ExpectedChecksum uint32
	// GenerateChecksum is a function that will be called to generate a checksum from the stored orderbook post update
	GenerateChecksum func(snapshot *Book) uint32
	// AllowEmpty, when true, permits loading an empty order book update to set an UpdateID without including actual data
	AllowEmpty bool
	// Action defines the action to be performed on the orderbook e.g. amend, delete, insert, update/insert
	// Orderbook IDs are used to identify the orderbook level to be updated, deleted or inserted
	Action ActionType

	SkipOutOfOrderLastUpdateID bool
}

// ProcessUpdate applies updates to the orderbook depth, on error it will invalidate the orderbook and return the
// error, this is to ensure the orderbook is always in a valid state.
func (d *Depth) ProcessUpdate(u *Update) error {
	if len(u.Bids) == 0 && len(u.Asks) == 0 && !u.AllowEmpty {
		return d.Invalidate(ErrEmptyUpdate)
	}

	// TODO: Enforce LastPushed set to determine server latency

	d.m.Lock()
	defer d.m.Unlock()

	if d.validationError != nil {
		return d.validationError
	}

	// This will process out of order updates but will not error on them.
	// TODO: Error on out of order updates; this is intentionally kept as is from the buffer package.
	// Add update.UpdateTime time check to ensure that the update is newer than the last update,
	// this should screen zero values as well.
	if u.SkipOutOfOrderLastUpdateID && d.lastUpdateID >= u.UpdateID {
		return nil
	}

	if d.options.restSnapshot {
		return d.invalidate(errRESTSnapshot)
	}

	if u.Action != UnknownAction {
		if err := d.update(u); err != nil {
			return d.invalidate(err)
		}
	} else {
		if err := d.updateBidAskByPrice(u); err != nil {
			return d.invalidate(err)
		}
	}

	if !d.validateOrderbook {
		return nil
	}

	if u.ExpectedChecksum != 0 {
		if u.GenerateChecksum == nil {
			return d.invalidate(errChecksumGeneratorUnset)
		}
		if checksum := u.GenerateChecksum(d.snapshot()); checksum != u.ExpectedChecksum {
			return d.invalidate(fmt.Errorf("%s %s %s %w: expected '%d', got '%d'", d.exchange, d.pair, d.asset, errChecksumMismatch, u.ExpectedChecksum, checksum))
		}
	}

	if err := validate(d.snapshot()); err != nil {
		return d.invalidate(err)
	}

	return nil
}

func (d *Depth) snapshot() *Book {
	return &Book{
		Bids:                   d.bidLevels.Levels,
		Asks:                   d.askLevels.Levels,
		Exchange:               d.options.exchange,
		Pair:                   d.pair,
		Asset:                  d.asset,
		IsFundingRate:          d.options.isFundingRate,
		PriceDuplication:       d.options.priceDuplication,
		IDAlignment:            d.options.idAligned,
		ChecksumStringRequired: d.options.checksumStringRequired,
	}
}

// update will receive an action to execute against the orderbook it will then match by IDs instead of
// price to perform the action
func (d *Depth) update(u *Update) error {
	switch u.Action {
	case UpdateAction:
		if err := d.updateBidAskByID(u); err != nil {
			return fmt.Errorf("%w for %q: %w", errUpdateFailed, u.Action, err)
		}
	case DeleteAction:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := d.options.exchange == "Bitfinex" && d.options.isFundingRate // TODO: Confirm this is still correct
		if err := d.delete(u, bypassErr); err != nil {
			return fmt.Errorf("%w for %q: %w", errDeleteFailed, u.Action, err)
		}
	case InsertAction:
		if err := d.insert(u); err != nil {
			return fmt.Errorf("%w for %q: %w", errUpdateFailed, u.Action, err)
		}
	case UpdateOrInsertAction:
		if err := d.updateOrInsert(u); err != nil {
			return fmt.Errorf("%w for %q: %w", errUpdateFailed, u.Action, err)
		}
	default:
		return fmt.Errorf("%w [%s]", errInvalidAction, u.Action)
	}
	return nil
}

// updateBidAskByPrice updates the bid and ask spread and enforces Depth.options.maxDepth
func (d *Depth) updateBidAskByPrice(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, ErrLastUpdatedNotSet)
	}
	d.bidLevels.updateInsertByPrice(update.Bids, d.options.maxDepth)
	d.askLevels.updateInsertByPrice(update.Asks, d.options.maxDepth)
	d.updateAndAlert(update)
	return nil
}

// updateBidAskByID amends details by ID
func (d *Depth) updateBidAskByID(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, ErrLastUpdatedNotSet)
	}
	if err := d.bidLevels.updateByID(update.Bids); err != nil {
		return err
	}
	if err := d.askLevels.updateByID(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// delete deletes a price level by ID
func (d *Depth) delete(update *Update, bypassErr bool) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, ErrLastUpdatedNotSet)
	}
	if err := d.bidLevels.deleteByID(update.Bids, bypassErr); err != nil {
		return err
	}
	if err := d.askLevels.deleteByID(update.Asks, bypassErr); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// insert inserts new updates
func (d *Depth) insert(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, ErrLastUpdatedNotSet)
	}
	if err := d.bidLevels.insertUpdates(update.Bids); err != nil {
		return err
	}
	if err := d.askLevels.insertUpdates(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// updateOrInsert updates or inserts by ID at current price level.
func (d *Depth) updateOrInsert(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, ErrLastUpdatedNotSet)
	}
	if err := d.bidLevels.updateInsertByID(update.Bids); err != nil {
		return err
	}
	if err := d.askLevels.updateInsertByID(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// String returns a string representation of the ActionType
func (a ActionType) String() string {
	switch a {
	case UnknownAction:
		return "Unknown"
	case InsertAction:
		return "Insert"
	case UpdateOrInsertAction:
		return "UpdateOrInsert"
	case UpdateAction:
		return "Update"
	case DeleteAction:
		return "Delete"
	default:
		return fmt.Sprintf("Unknown(%d)", a)
	}
}

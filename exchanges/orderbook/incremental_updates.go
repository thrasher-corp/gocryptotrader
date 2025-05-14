package orderbook

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Action defines a set of differing states required to implement an incoming orderbook update used in conjunction with
// UpdateEntriesByID
type Action uint8

const (
	// Amend applies amount adjustment by ID
	Amend Action = iota + 1
	// Delete removes price level from book by ID
	Delete
	// Insert adds price level to book
	Insert
	// UpdateInsert on conflict applies amount adjustment or appends new amount to book
	UpdateInsert
)

// Public error vars
var (
	ErrDepthNotFound = errors.New("orderbook depth not found")
	ErrEmptyUpdate   = errors.New("update contains no bids or asks")
)

var (
	errInvalidAction          = errors.New("invalid action")
	errAmendFailure           = errors.New("amend update failure")
	errDeleteFailure          = errors.New("delete update failure")
	errInsertFailure          = errors.New("insert update failure")
	errUpdateInsertFailure    = errors.New("update/insert update failure")
	errRESTSnapshot           = errors.New("cannot update REST protocol loaded snapshot")
	errChecksumMismatch       = errors.New("checksum mismatch")
	errChecksumGeneratorUnset = errors.New("checksum generator unset")
)

// Update holds changes that are to be applied to a stored orderbook
type Update struct {
	UpdateID       int64
	UpdateTime     time.Time
	UpdatePushedAt time.Time
	Asset          asset.Item
	Bids           []Tranche
	Asks           []Tranche
	Pair           currency.Pair

	// ExpectedChecksum defines the expected value when the books have been verified
	ExpectedChecksum uint32
	// GenerateChecksum is a function that will be called to generate a checksum from the stored orderbook post update
	GenerateChecksum func(snapshot *Base) uint32
	// AllowEmpty, when true, permits loading an empty order book update to set an UpdateID without including actual data
	AllowEmpty bool
	// Action defines the action to be performed on the orderbook e.g. amend, delete, insert, update/insert
	// Orderbook IDs are used to identify the orderbook level to be updated, deleted or inserted
	Action Action

	SkipOutOfOrderLastUpdateID bool
}

// ProcessUpdate applies updates to the orderbook depth, on error it will invalidate the orderbook and return the
// error, this is to ensure the orderbook is always in a valid state.
func (d *Depth) ProcessUpdate(u *Update) error {
	if len(u.Bids) == 0 && len(u.Asks) == 0 && !u.AllowEmpty {
		return d.Invalidate(ErrEmptyUpdate)
	}

	// TODO: Enforce UpdatePushedAt set to determine server latency

	d.m.Lock()
	defer d.m.Unlock()

	if d.validationError != nil {
		return d.validationError
	}

	// This will process out of order updates but will not error on them.
	// TODO: Error on out of order updates; this is intentionally kept as is from the buffer package.
	if u.SkipOutOfOrderLastUpdateID && d.lastUpdateID >= u.UpdateID {
		return nil
	}

	if d.options.restSnapshot {
		return d.invalidate(errRESTSnapshot)
	}

	if u.Action != 0 {
		if err := d.updateByIDAndAction(u); err != nil {
			return d.invalidate(err)
		}
	} else {
		if err := d.updateBidAskByPrice(u); err != nil {
			return d.invalidate(err)
		}
	}

	if u.ExpectedChecksum != 0 {
		if u.GenerateChecksum == nil {
			return d.invalidate(errChecksumGeneratorUnset)
		}
		if checksum := u.GenerateChecksum(d.snapshot()); checksum != u.ExpectedChecksum {
			return d.invalidate(fmt.Errorf("%s %s %s %w: expected '%d', got '%d'", d.exchange, d.pair, d.asset, errChecksumMismatch, u.ExpectedChecksum, checksum))
		}
	} else if d.verifyOrderbook {
		if err := verify(d.snapshot()); err != nil {
			return d.invalidate(err)
		}
	}

	return nil
}

// // TODO: Confirm no alloc as it shouldn't escape
func (d *Depth) snapshot() *Base {
	return &Base{
		Bids:                   d.bidTranches.Tranches,
		Asks:                   d.askTranches.Tranches,
		Exchange:               d.options.exchange,
		Pair:                   d.pair,
		Asset:                  d.asset,
		IsFundingRate:          d.options.isFundingRate,
		PriceDuplication:       d.options.priceDuplication,
		IDAlignment:            d.options.idAligned,
		ChecksumStringRequired: d.options.checksumStringRequired,
	}
}

// updateByIDAndAction will receive an action to execute against the orderbook it will then match by IDs instead of
// price to perform the action
func (d *Depth) updateByIDAndAction(u *Update) error {
	switch u.Action {
	case Amend:
		if err := d.updateBidAskByID(u); err != nil {
			return fmt.Errorf("%w %w", errAmendFailure, err)
		}
	case Delete:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := d.options.exchange == "Bitfinex" && d.options.isFundingRate // TODO: Confirm this is still correct
		if err := d.deleteBidAskByID(u, bypassErr); err != nil {
			return fmt.Errorf("%w %w", errDeleteFailure, err)
		}
	case Insert:
		if err := d.insertBidAskByID(u); err != nil {
			return fmt.Errorf("%w %w", errInsertFailure, err)
		}
	case UpdateInsert:
		if err := d.updateInsertByID(u); err != nil {
			return fmt.Errorf("%w %w", errUpdateInsertFailure, err)
		}
	default:
		return fmt.Errorf("%w [%d]", errInvalidAction, u.Action)
	}
	return nil
}

// updateBidAskByPrice updates the bid and ask spread by supplied updates, this will trim total length of depth level to
// a specified supplied number
func (d *Depth) updateBidAskByPrice(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, errLastUpdatedNotSet)
	}
	d.bidTranches.updateInsertByPrice(update.Bids, d.options.maxDepth)
	d.askTranches.updateInsertByPrice(update.Asks, d.options.maxDepth)
	d.updateAndAlert(update)
	return nil
}

// updateBidAskByID amends details by ID
func (d *Depth) updateBidAskByID(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, errLastUpdatedNotSet)
	}
	if err := d.bidTranches.updateByID(update.Bids); err != nil {
		return err
	}
	if err := d.askTranches.updateByID(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// deleteBidAskByID deletes a price level by ID
func (d *Depth) deleteBidAskByID(update *Update, bypassErr bool) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, errLastUpdatedNotSet)
	}
	if err := d.bidTranches.deleteByID(update.Bids, bypassErr); err != nil {
		return err
	}
	if err := d.askTranches.deleteByID(update.Asks, bypassErr); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// insertBidAskByID inserts new updates
func (d *Depth) insertBidAskByID(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, errLastUpdatedNotSet)
	}
	if err := d.bidTranches.insertUpdates(update.Bids); err != nil {
		return err
	}
	if err := d.askTranches.insertUpdates(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

// updateInsertByID updates or inserts by ID at current price level.
func (d *Depth) updateInsertByID(update *Update) error {
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w", d.exchange, d.pair, d.asset, errLastUpdatedNotSet)
	}
	if err := d.bidTranches.updateInsertByID(update.Bids); err != nil {
		return err
	}
	if err := d.askTranches.updateInsertByID(update.Asks); err != nil {
		return err
	}
	d.updateAndAlert(update)
	return nil
}

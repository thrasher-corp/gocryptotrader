package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrOrderbookInvalid defines an error for when the orderbook is invalid and
	// should not be trusted
	ErrOrderbookInvalid = errors.New("orderbook data integrity compromised")
	// ErrInvalidAction defines and error when an action is invalid
	ErrInvalidAction = errors.New("invalid action")

	errLastUpdatedNotSet = errors.New("last updated not set")
	errInvalidBookDepth  = errors.New("invalid book depth")
)

// Outbound restricts outbound usage of depth. NOTE: Type assert to
// *orderbook.Depth.
type Outbound interface {
	Retrieve() (*Base, error)
}

// Depth defines a store of orderbook tranches
type Depth struct {
	askTranches
	bidTranches

	alert.Notice

	mux *dispatch.Mux
	_ID uuid.UUID

	options

	// validationError defines current book state and why it was invalidated.
	validationError error

	m sync.RWMutex
}

// NewDepth returns a new orderbook depth
func NewDepth(id uuid.UUID) *Depth {
	return &Depth{_ID: id, mux: service.Mux}
}

// Publish alerts any subscribed routines using a dispatch mux
func (d *Depth) Publish() {
	if err := d.mux.Publish(Outbound(d), d._ID); err != nil {
		log.Errorf(log.ExchangeSys, "Cannot publish orderbook update to mux %v", err)
	}
}

// Retrieve returns the orderbook base a copy of the underlying linked list
// spread
func (d *Depth) Retrieve() (*Base, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return &Base{
		Bids:                   d.bidTranches.retrieve(0),
		Asks:                   d.askTranches.retrieve(0),
		Exchange:               d.exchange,
		Asset:                  d.asset,
		Pair:                   d.pair,
		LastUpdated:            d.lastUpdated,
		UpdatePushedAt:         d.updatePushedAt,
		InsertedAt:             d.insertedAt,
		LastUpdateID:           d.lastUpdateID,
		PriceDuplication:       d.priceDuplication,
		IsFundingRate:          d.isFundingRate,
		VerifyOrderbook:        d.verifyOrderbook,
		MaxDepth:               d.maxDepth,
		ChecksumStringRequired: d.checksumStringRequired,
		RestSnapshot:           d.restSnapshot,
		IDAlignment:            d.idAligned,
	}, nil
}

// LoadSnapshot flushes the bids and asks with a snapshot
func (d *Depth) LoadSnapshot(bids, asks []Tranche, lastUpdateID int64, lastUpdated, updatePushedAt time.Time, updateByREST bool) error {
	d.m.Lock()
	defer d.m.Unlock()
	if lastUpdated.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}
	d.lastUpdateID = lastUpdateID
	d.lastUpdated = lastUpdated
	d.updatePushedAt = updatePushedAt
	d.insertedAt = time.Now()
	d.restSnapshot = updateByREST
	d.bidTranches.load(bids)
	d.askTranches.load(asks)
	d.validationError = nil
	d.Alert()
	return nil
}

// invalidate flushes all values back to zero so as to not allow strategy
// traversal on compromised data. NOTE: This requires locking.
func (d *Depth) invalidate(withReason error) error {
	d.lastUpdateID = 0
	d.lastUpdated = time.Time{}
	d.bidTranches.load(nil)
	d.askTranches.load(nil)
	d.validationError = fmt.Errorf("%s %s %s Reason: [%w]",
		d.exchange,
		d.pair,
		d.asset,
		common.AppendError(ErrOrderbookInvalid, withReason))
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
	d.m.RLock()
	defer d.m.RUnlock()
	return d.validationError == nil
}

// UpdateBidAskByPrice updates the bid and ask spread by supplied updates, this
// will trim total length of depth level to a specified supplied number
func (d *Depth) UpdateBidAskByPrice(update *Update) error {
	d.m.Lock()
	defer d.m.Unlock()
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}
	if len(update.Bids) != 0 {
		d.bidTranches.updateInsertByPrice(update.Bids, d.options.maxDepth)
	}
	if len(update.Asks) != 0 {
		d.askTranches.updateInsertByPrice(update.Asks, d.options.maxDepth)
	}
	d.updateAndAlert(update)
	return nil
}

// UpdateBidAskByID amends details by ID
func (d *Depth) UpdateBidAskByID(update *Update) error {
	d.m.Lock()
	defer d.m.Unlock()

	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}

	if len(update.Bids) != 0 {
		err := d.bidTranches.updateByID(update.Bids)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.askTranches.updateByID(update.Asks)
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
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}
	if len(update.Bids) != 0 {
		err := d.bidTranches.deleteByID(update.Bids, bypassErr)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.askTranches.deleteByID(update.Asks, bypassErr)
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
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}
	if len(update.Bids) != 0 {
		err := d.bidTranches.insertUpdates(update.Bids)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.askTranches.insertUpdates(update.Asks)
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
	if update.UpdateTime.IsZero() {
		return fmt.Errorf("%s %s %s %w",
			d.exchange,
			d.pair,
			d.asset,
			errLastUpdatedNotSet)
	}
	if len(update.Bids) != 0 {
		err := d.bidTranches.updateInsertByID(update.Bids)
		if err != nil {
			return d.invalidate(err)
		}
	}
	if len(update.Asks) != 0 {
		err := d.askTranches.updateInsertByID(update.Asks)
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
		exchange:               b.Exchange,
		pair:                   b.Pair,
		asset:                  b.Asset,
		lastUpdated:            b.LastUpdated,
		lastUpdateID:           b.LastUpdateID,
		priceDuplication:       b.PriceDuplication,
		isFundingRate:          b.IsFundingRate,
		verifyOrderbook:        b.VerifyOrderbook,
		restSnapshot:           b.RestSnapshot,
		idAligned:              b.IDAlignment,
		maxDepth:               b.MaxDepth,
		checksumStringRequired: b.ChecksumStringRequired,
	}
	d.m.Unlock()
}

// GetName returns name of exchange
func (d *Depth) GetName() string {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.exchange
}

// IsRESTSnapshot returns if the depth was updated via REST
func (d *Depth) IsRESTSnapshot() (bool, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return false, d.validationError
	}
	return d.restSnapshot, nil
}

// LastUpdateID returns the last Update ID
func (d *Depth) LastUpdateID() (int64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.lastUpdateID, nil
}

// IsFundingRate returns if the depth is a funding rate
func (d *Depth) IsFundingRate() bool {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.isFundingRate
}

// VerifyOrderbook returns if the verify orderbook option is set
func (d *Depth) VerifyOrderbook() bool {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.verifyOrderbook
}

// GetAskLength returns length of asks
func (d *Depth) GetAskLength() (int, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return len(d.askTranches.Tranches), nil
}

// GetBidLength returns length of bids
func (d *Depth) GetBidLength() (int, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return len(d.bidTranches.Tranches), nil
}

// TotalBidAmounts returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidAmounts() (liquidity, value float64, err error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, 0, d.validationError
	}
	liquidity, value = d.bidTranches.amount()
	return liquidity, value, nil
}

// TotalAskAmounts returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAskAmounts() (liquidity, value float64, err error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, 0, d.validationError
	}
	liquidity, value = d.askTranches.amount()
	return liquidity, value, nil
}

// updateAndAlert updates the last updated ID and when it was updated to the
// recent update. Then alerts all pending routines. NOTE: This requires locking.
func (d *Depth) updateAndAlert(update *Update) {
	d.lastUpdateID = update.UpdateID
	d.lastUpdated = update.UpdateTime
	d.updatePushedAt = update.UpdatePushedAt
	d.insertedAt = time.Now()
	d.Alert()
}

// HitTheBidsByNominalSlippage hits the bids by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByNominalSlippage(maxSlippage, refPrice float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return d.bidTranches.hitBidsByNominalSlippage(maxSlippage, refPrice)
}

// HitTheBidsByNominalSlippageFromMid hits the bids by the required nominal
// slippage percentage, calculated from the mid price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByNominalSlippageFromMid(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.bidTranches.hitBidsByNominalSlippage(maxSlippage, mid)
}

// HitTheBidsByNominalSlippageFromBest hits the bids by the required nominal
// slippage percentage, calculated from the best bid price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByNominalSlippageFromBest(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.bidTranches.hitBidsByNominalSlippage(maxSlippage, head)
}

// LiftTheAsksByNominalSlippage lifts the asks by the required nominal slippage
// percentage, calculated from the reference price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByNominalSlippage(maxSlippage, refPrice float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return d.askTranches.liftAsksByNominalSlippage(maxSlippage, refPrice)
}

// LiftTheAsksByNominalSlippageFromMid lifts the asks by the required nominal
// slippage percentage, calculated from the mid price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByNominalSlippageFromMid(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.askTranches.liftAsksByNominalSlippage(maxSlippage, mid)
}

// LiftTheAsksByNominalSlippageFromBest lifts the asks by the required nominal
// slippage percentage, calculated from the best ask price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByNominalSlippageFromBest(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.askTranches.liftAsksByNominalSlippage(maxSlippage, head)
}

// HitTheBidsByImpactSlippage hits the bids by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByImpactSlippage(maxSlippage, refPrice float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return d.bidTranches.hitBidsByImpactSlippage(maxSlippage, refPrice)
}

// HitTheBidsByImpactSlippageFromMid hits the bids by the required impact
// slippage percentage, calculated from the mid price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByImpactSlippageFromMid(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.bidTranches.hitBidsByImpactSlippage(maxSlippage, mid)
}

// HitTheBidsByImpactSlippageFromBest hits the bids by the required impact
// slippage percentage, calculated from the best bid price and returns orderbook
// movement details for the bid side.
func (d *Depth) HitTheBidsByImpactSlippageFromBest(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.bidTranches.hitBidsByImpactSlippage(maxSlippage, head)
}

// LiftTheAsksByImpactSlippage lifts the asks by the required impact slippage
// percentage, calculated from the reference price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByImpactSlippage(maxSlippage, refPrice float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return d.askTranches.liftAsksByImpactSlippage(maxSlippage, refPrice)
}

// LiftTheAsksByImpactSlippageFromMid lifts the asks by the required impact
// slippage percentage, calculated from the mid price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByImpactSlippageFromMid(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.askTranches.liftAsksByImpactSlippage(maxSlippage, mid)
}

// LiftTheAsksByImpactSlippageFromBest lifts the asks by the required impact
// slippage percentage, calculated from the best ask price and returns orderbook
// movement details for the ask side.
func (d *Depth) LiftTheAsksByImpactSlippageFromBest(maxSlippage float64) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	return d.askTranches.liftAsksByImpactSlippage(maxSlippage, head)
}

// HitTheBids derives full orderbook slippage information from reference price
// using an amount. Purchase refers to how much quote currency is desired else
// the amount would refer to base currency deployed to orderbook bid side.
func (d *Depth) HitTheBids(amount, refPrice float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	if purchase {
		return d.bidTranches.getMovementByQuotation(amount, refPrice, false)
	}
	return d.bidTranches.getMovementByBase(amount, refPrice, false)
}

// HitTheBidsFromMid derives full orderbook slippage information from mid price
// using an amount. Purchase refers to how much quote currency is desired else
// the amount would refer to base currency deployed to orderbook bid side.
func (d *Depth) HitTheBidsFromMid(amount float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	if purchase {
		return d.bidTranches.getMovementByQuotation(amount, mid, false)
	}
	return d.bidTranches.getMovementByBase(amount, mid, false)
}

// HitTheBidsFromBest derives full orderbook slippage information from best bid
// price using an amount. Purchase refers to how much quote currency is desired
// else the amount would refer to base currency deployed to orderbook bid side.
func (d *Depth) HitTheBidsFromBest(amount float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	if purchase {
		return d.bidTranches.getMovementByQuotation(amount, head, false)
	}
	return d.bidTranches.getMovementByBase(amount, head, false)
}

// LiftTheAsks derives full orderbook slippage information from reference price
// using an amount. Purchase refers to how much base currency is desired else
// the amount would refer to quote currency deployed to orderbook ask side.
func (d *Depth) LiftTheAsks(amount, refPrice float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	if purchase {
		return d.askTranches.getMovementByBase(amount, refPrice, true)
	}
	return d.askTranches.getMovementByQuotation(amount, refPrice, true)
}

// LiftTheAsksFromMid derives full orderbook slippage information from mid price
// using an amount. Purchase refers to how much base currency is desired else
// the amount would refer to quote currency deployed to orderbook ask side.
func (d *Depth) LiftTheAsksFromMid(amount float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		return nil, err
	}
	if purchase {
		return d.askTranches.getMovementByBase(amount, mid, true)
	}
	return d.askTranches.getMovementByQuotation(amount, mid, true)
}

// LiftTheAsksFromBest derives full orderbook slippage information from best ask
// price using an amount. Purchase refers to how much base currency is desired
// else the amount would refer to quote currency deployed to orderbook ask side.
func (d *Depth) LiftTheAsksFromBest(amount float64, purchase bool) (*Movement, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	head, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return nil, err
	}
	if purchase {
		return d.askTranches.getMovementByBase(amount, head, true)
	}
	return d.askTranches.getMovementByQuotation(amount, head, true)
}

// GetMidPrice returns the mid price between the ask and bid spread
func (d *Depth) GetMidPrice() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.getMidPriceNoLock()
}

// getMidPriceNoLock is an unprotected helper that gets mid price
func (d *Depth) getMidPriceNoLock() (float64, error) {
	bidHead, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	askHead, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	return (bidHead + askHead) / 2, nil
}

// GetBestBid returns the best bid price
func (d *Depth) GetBestBid() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.bidTranches.getHeadPriceNoLock()
}

// GetBestAsk returns the best ask price
func (d *Depth) GetBestAsk() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	return d.askTranches.getHeadPriceNoLock()
}

// GetSpreadAmount returns the spread as a quotation amount
func (d *Depth) GetSpreadAmount() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	askHead, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	bidHead, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	return askHead - bidHead, nil
}

// GetSpreadPercentage returns the spread as a percentage
func (d *Depth) GetSpreadPercentage() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	askHead, err := d.askTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	bidHead, err := d.bidTranches.getHeadPriceNoLock()
	if err != nil {
		return 0, err
	}
	return (askHead - bidHead) / askHead * 100, nil
}

// GetImbalance returns top orderbook imbalance
func (d *Depth) GetImbalance() (float64, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return 0, d.validationError
	}
	askVolume, err := d.askTranches.getHeadVolumeNoLock()
	if err != nil {
		return 0, err
	}
	bidVolume, err := d.bidTranches.getHeadVolumeNoLock()
	if err != nil {
		return 0, err
	}
	return (bidVolume - askVolume) / (bidVolume + askVolume), nil
}

// GetTranches returns the desired tranche for the required depth count. If
// count is 0, it will return the entire orderbook. Count == 1 will retrieve the
// best bid and ask. If the required count exceeds the orderbook depth, it will
// return the entire orderbook.
func (d *Depth) GetTranches(count int) (ask, bid []Tranche, err error) {
	if count < 0 {
		return nil, nil, errInvalidBookDepth
	}
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return nil, nil, d.validationError
	}
	return d.askTranches.retrieve(count), d.bidTranches.retrieve(count), nil
}

// Pair returns the pair associated with the depth
func (d *Depth) Pair() currency.Pair {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.pair
}

// Asset returns the asset associated with the depth
func (d *Depth) Asset() asset.Item {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.asset
}

// Exchange returns the exchange associated with the depth
func (d *Depth) Exchange() string {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.exchange
}

// Key returns a combined key for the depth
func (d *Depth) Key() key.ExchangePairAsset {
	d.m.RLock()
	defer d.m.RUnlock()
	return key.ExchangePairAsset{Exchange: d.exchange, Base: d.pair.Base.Item, Quote: d.pair.Quote.Item, Asset: d.asset}
}

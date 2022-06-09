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

var (
	// ErrOrderbookInvalid defines an error for when the orderbook is invalid and
	// should not be trusted
	ErrOrderbookInvalid = errors.New("orderbook data integrity compromised")
	// ErrInvalidAction defines and error when an action is invalid
	ErrInvalidAction = errors.New("invalid action")
)

// Outbound restricts outbound usage of depth. NOTE: Type assert to
// *orderbook.Depth or alternatively retrieve orderbook.Unsafe type to access
// underlying linked list.
type Outbound interface {
	Retrieve() (*Base, error)
}

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
	if err := d.mux.Publish(Outbound(d), d._ID); err != nil {
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
// traversal on compromised data. NOTE: This requires locking.
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
// recent update. Then alerts all pending routines. NOTE: This requires locking.
func (d *Depth) updateAndAlert(update *Update) {
	d.lastUpdateID = update.UpdateID
	d.lastUpdated = update.UpdateTime
	d.Alert()
}

// GetBaseFromNominalSlippage returns the base amount when hitting the bids to
// result in a max nominal slippage percentage from a reference price.
// Warning: This is not accurate.
func (d *Depth) GetBaseFromNominalSlippage(maxSlippage float64, refPrice float64) (float64, error) {
	d.m.Lock()
	baseAmt, err := d.bids.getBaseAmountFromNominalSlippage(maxSlippage, refPrice)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetBaseFromNominalSlippageFromMid return the base amount when hitting the
// bids to result in a nominal slippage percentage from the orderbook mid price.
// Warning: this is not accurate.
func (d *Depth) GetBaseFromNominalSlippageFromMid(maxSlippage float64) (float64, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}

	baseAmt, err := d.bids.getBaseAmountFromNominalSlippage(maxSlippage, mid)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetBaseFromNominalSlippageFromBest return the base amount when hitting the
// bids to result in a nominal slippage percentage from the bid best price.
// Warning: this is not accurate.
func (d *Depth) GetBaseFromNominalSlippageFromBest(maxSlippage float64) (float64, error) {
	d.m.Lock()
	head, err := d.bids.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	fmt.Println(head)
	baseAmt, err := d.bids.getBaseAmountFromNominalSlippage(maxSlippage, head)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetQuoteFromNominalSlippage return the quote amount when lifting the asks to
// result in a nominal slippage percentage from the reference price.
// Warning: this is not accurate.
func (d *Depth) GetQuoteFromNominalSlippage(maxSlippage float64, refPrice float64) (float64, error) {
	d.m.Lock()
	quoteAmt, err := d.asks.getQuoteAmountFromNominalSlippage(maxSlippage, refPrice)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetQuoteFromNominalSlippageFromMid return the quote amount when lifting the
// asks to result in a nominal slippage percentage from the orderbook mid price.
// Warning: this is not accurate.
func (d *Depth) GetQuoteFromNominalSlippageFromMid(maxSlippage float64) (float64, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	quoteAmt, err := d.asks.getQuoteAmountFromNominalSlippage(maxSlippage, mid)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetQuoteFromNominalSlippageFromBest return the quote amount when lifting the
// asks to result in a nominal slippage percentage from the best ask price.
// Warning: this is not accurate.
func (d *Depth) GetQuoteFromNominalSlippageFromBest(maxSlippage float64) (float64, error) {
	d.m.Lock()
	head, err := d.asks.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	quoteAmt, err := d.asks.getQuoteAmountFromNominalSlippage(maxSlippage, head)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetBaseFromImpactSlippage return the base amount when hitting the bids to
// result in an impact (how much the book price has shifted) slippage percentage
// from the reference price.
func (d *Depth) GetBaseFromImpactSlippage(maxSlippage float64, refPrice float64) (float64, error) {
	d.m.Lock()
	baseAmt, err := d.bids.getBaseAmountFromImpact(maxSlippage, refPrice)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetBaseFromImpactSlippageFromMid return the base amount when hitting the bids to
// result in an impact (how much the book price has shifted) slippage percentage
// from the orderbook mid price.
func (d *Depth) GetBaseFromImpactSlippageFromMid(maxSlippage float64) (float64, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	baseAmt, err := d.bids.getBaseAmountFromImpact(maxSlippage, mid)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetBaseFromImpactSlippageFromBest return the base amount when hitting the bids to
// result in an impact (how much the book price has shifted) slippage percentage
// from the bid best price.
func (d *Depth) GetBaseFromImpactSlippageFromBest(maxSlippage float64) (float64, error) {
	d.m.Lock()
	head, err := d.bids.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	baseAmt, err := d.bids.getBaseAmountFromImpact(maxSlippage, head)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, true)
	}
	return baseAmt, nil
}

// GetQuoteFromImpactSlippage return the quote amount when lifting the asks to
// result in an impact (how much the book price has shifted) slippage percentage
// from the reference price.
func (d *Depth) GetQuoteFromImpactSlippage(maxSlippage float64, refPrice float64) (float64, error) {
	d.m.Lock()
	quoteAmt, err := d.asks.getQuoteAmountFromImpact(maxSlippage, refPrice)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetQuoteFromImpactSlippageFromMid return the quote amount when lifting the asks to
// result in an impact (how much the book price has shifted) slippage percentage
// from the orderbook mid price.
func (d *Depth) GetQuoteFromImpactSlippageFromMid(maxSlippage float64) (float64, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	quoteAmt, err := d.asks.getQuoteAmountFromImpact(maxSlippage, mid)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetQuoteFromImpactSlippageFromBest return the quote amount when lifting the
// asks to result in an impact (how much the book price has shifted) slippage
// percentage from the ask best price.
func (d *Depth) GetQuoteFromImpactSlippageFromBest(maxSlippage float64) (float64, error) {
	d.m.Lock()
	head, err := d.asks.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return 0, err
	}
	quoteAmt, err := d.asks.getQuoteAmountFromImpact(maxSlippage, head)
	d.m.Unlock()
	if err != nil {
		return 0, d.AddTrimmingToError(err, false)
	}
	return quoteAmt, nil
}

// GetMovementByBase derives your slippage from the reference price to the
// potential deployment average order cost using the quote amount. This hits the
// bids when you are ask/sell side.
func (d *Depth) GetMovementByBase(base float64, refPrice float64) (*Movement, error) {
	d.m.Lock()
	move, err := d.bids.getMovementByBaseAmount(base, refPrice)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, true)
	}
	return move, nil
}

// GetMovementByQuotationFromMid derives your slippage from the mid price
// between top bid ask quotations to the potential deployment average order
// cost using the quote amount. This hits the bids when you are ask/sell side.
func (d *Depth) GetMovementByBaseFromMid(base float64) (*Movement, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return nil, err
	}
	move, err := d.bids.getMovementByBaseAmount(base, mid)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, true)
	}
	return move, nil
}

// GetMovementByQuotationFromBest derives your slippage from the best price ask
// quotation to the potential deployment average order cost using the quote
// amount. This hits the bids when you are ask/sell side.
func (d *Depth) GetMovementByBaseFromBest(base float64) (*Movement, error) {
	d.m.Lock()
	head, err := d.bids.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return nil, err
	}
	move, err := d.bids.getMovementByBaseAmount(base, head)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, true)
	}
	return move, nil
}

// GetMovementByQuote derives your slippage from the reference price to the
// potential deployment average order cost using the quote amount. This lifts
// the offers when you are bid/buy side.
func (d *Depth) GetMovementByQuote(quote float64, refPrice float64) (*Movement, error) {
	d.m.Lock()
	move, err := d.asks.getMovementByQuoteAmount(quote, refPrice)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, false)
	}
	return move, nil
}

// GetMovementByQuoteFromMid derives your slippage from the mid price
// between top bid ask quotations to the potential deployment average order
// cost using the quote amount. This lifts the offers when you are bid/buy side.
func (d *Depth) GetMovementByQuoteFromMid(quote float64) (*Movement, error) {
	d.m.Lock()
	mid, err := d.getMidPriceNoLock()
	if err != nil {
		d.m.Unlock()
		return nil, err
	}
	move, err := d.asks.getMovementByQuoteAmount(quote, mid)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, false)
	}
	return move, nil
}

// GetMovementByQuoteFromBest derives your slippage from the best price ask
// quotation to the potential deployment average order cost using the quote
// amount. This lifts the offers when you are bid/buy side.
func (d *Depth) GetMovementByQuoteFromBest(quote float64) (*Movement, error) {
	d.m.Lock()
	head, err := d.asks.getHeadPrice()
	if err != nil {
		d.m.Unlock()
		return nil, err
	}
	move, err := d.asks.getMovementByQuoteAmount(quote, head)
	d.m.Unlock()
	if err != nil {
		return nil, d.AddTrimmingToError(err, false)
	}
	return move, nil
}

// GetMidPrice returns the mid price between the ask and bid spread
func (d *Depth) GetMidPrice() (float64, error) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.getMidPriceNoLock()
}

// getMidPriceNoLock is an unprotected helper that gets mid price
func (d *Depth) getMidPriceNoLock() (float64, error) {
	bidHead, err := d.bids.getHeadPrice()
	if err != nil {
		return 0, err
	}
	askHead, err := d.asks.getHeadPrice()
	if err != nil {
		return 0, err
	}
	return (bidHead + askHead) / 2, nil
}

// GetBestBid returns the best bid price
func (d *Depth) GetBestBid() (float64, error) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.bids.getHeadPrice()
}

// GetBestAsk returns the best ask price
func (d *Depth) GetBestAsk() (float64, error) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.asks.getHeadPrice()
}

// AddTrimmingToError add liquidity side information to error if there is not
// enough liquidity on the book. WRT websocket books this might mean there is
// only a limited amount of returned data compared to REST.
func (d *Depth) AddTrimmingToError(err error, bid bool) error {
	if err == nil || !errors.Is(err, errNotEnoughLiquidity) {
		fmt.Println("bruh")
		return err
	}
	fmt.Println("bruh2")
	var amounts SideAmounts
	var liquiditySide string

	if bid {
		d.m.Lock()
		amounts = d.bids.getSideAmounts()
		d.m.Unlock()
		liquiditySide = "bid"
	} else {
		d.m.Lock()
		amounts = d.asks.getSideAmounts()
		d.m.Unlock()
		liquiditySide = "ask"
	}

	return fmt.Errorf("%w on current %s depth - tranche levels:%v max value:%v max amount:%v ",
		err,
		liquiditySide,
		amounts.Tranches,
		amounts.QuoteValue,
		amounts.BaseAmount)
}

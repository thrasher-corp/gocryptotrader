package buffer

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const packageError = "websocket orderbook buffer error: %w"

var (
	errExchangeConfigNil            = errors.New("exchange config is nil")
	errBufferConfigNil              = errors.New("buffer config is nil")
	errUnsetDataHandler             = errors.New("datahandler unset")
	errIssueBufferEnabledButNoLimit = errors.New("buffer enabled but no limit set")
	errUpdateIsNil                  = errors.New("update is nil")
	errUpdateNoTargets              = errors.New("update bid/ask targets cannot be nil")
	errDepthNotFound                = errors.New("orderbook depth not found")
	errRESTOverwrite                = errors.New("orderbook has been overwritten by REST protocol")
	errInvalidAction                = errors.New("invalid action")
	errAmendFailure                 = errors.New("orderbook amend update failure")
	errDeleteFailure                = errors.New("orderbook delete update failure")
	errInsertFailure                = errors.New("orderbook insert update failure")
	errUpdateInsertFailure          = errors.New("orderbook update/insert update failure")
	errRESTTimerLapse               = errors.New("rest sync timer lapse with active websocket connection")
	errOrderbookFlushed             = errors.New("orderbook flushed")
)

// Setup sets private variables
func (w *Orderbook) Setup(exchangeConfig *config.Exchange, c *Config, dataHandler chan<- interface{}) error {
	if exchangeConfig == nil { // exchange config fields are checked in stream package
		// prior to calling this, so further checks are not needed.
		return fmt.Errorf(packageError, errExchangeConfigNil)
	}
	if c == nil {
		return fmt.Errorf(packageError, errBufferConfigNil)
	}
	if dataHandler == nil {
		return fmt.Errorf(packageError, errUnsetDataHandler)
	}
	if exchangeConfig.Orderbook.WebsocketBufferEnabled &&
		exchangeConfig.Orderbook.WebsocketBufferLimit < 1 {
		return fmt.Errorf(packageError, errIssueBufferEnabledButNoLimit)
	}

	// NOTE: These variables are set by config.json under "orderbook" for each
	// individual exchange.
	w.bufferEnabled = exchangeConfig.Orderbook.WebsocketBufferEnabled
	w.obBufferLimit = exchangeConfig.Orderbook.WebsocketBufferLimit

	w.sortBuffer = c.SortBuffer
	w.sortBufferByUpdateIDs = c.SortBufferByUpdateIDs
	w.updateEntriesByID = c.UpdateEntriesByID
	w.exchangeName = exchangeConfig.Name
	w.dataHandler = dataHandler
	w.ob = make(map[Key]*orderbookHolder)
	w.verbose = exchangeConfig.Verbose

	// set default publish period if missing
	orderbookPublishPeriod := config.DefaultOrderbookPublishPeriod
	if exchangeConfig.Orderbook.PublishPeriod != nil {
		orderbookPublishPeriod = *exchangeConfig.Orderbook.PublishPeriod
	}
	w.publishPeriod = orderbookPublishPeriod
	w.updateIDProgression = c.UpdateIDProgression
	w.checksum = c.Checksum
	return nil
}

// validate validates update against setup values
func (w *Orderbook) validate(u *orderbook.Update) error {
	if u == nil {
		return fmt.Errorf(packageError, errUpdateIsNil)
	}
	if len(u.Bids) == 0 && len(u.Asks) == 0 {
		return fmt.Errorf(packageError, errUpdateNoTargets)
	}
	return nil
}

// Update updates a stored pointer to an orderbook.Depth struct containing a
// linked list, this switches between the usage of a buffered update
func (w *Orderbook) Update(u *orderbook.Update) error {
	if err := w.validate(u); err != nil {
		return err
	}
	w.mtx.Lock()
	defer w.mtx.Unlock()
	book, ok := w.ob[Key{Base: u.Pair.Base.Item, Quote: u.Pair.Quote.Item, Asset: u.Asset}]
	if !ok {
		return fmt.Errorf("%w for Exchange %s CurrencyPair: %s AssetType: %s",
			errDepthNotFound,
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	// out of order update ID can be skipped
	if w.updateIDProgression && u.UpdateID <= book.updateID {
		if w.verbose {
			log.Warnf(log.WebsocketMgr,
				"Exchange %s CurrencyPair: %s AssetType: %s out of order websocket update received",
				w.exchangeName,
				u.Pair,
				u.Asset)
		}
		return nil
	}

	// Checks for when the rest protocol overwrites a streaming dominated book
	// will stop updating book via incremental updates. This occurs because our
	// sync manager (engine/sync.go) timer has elapsed for streaming. Usually
	// because the book is highly illiquid.
	isREST, err := book.ob.IsRESTSnapshot()
	if err != nil {
		if !errors.Is(err, orderbook.ErrOrderbookInvalid) {
			return err
		}
		// In the event a checksum or processing error invalidates the book, all
		// updates that could be stored in the websocket buffer, skip applying
		// until a new snapshot comes through.
		if w.verbose {
			log.Warnf(log.WebsocketMgr,
				"Exchange %s CurrencyPair: %s AssetType: %s underlying book is invalid, cannot apply update.",
				w.exchangeName,
				u.Pair,
				u.Asset)
		}
		return nil
	}

	if isREST {
		if w.verbose {
			log.Warnf(log.WebsocketMgr,
				"%s for Exchange %s CurrencyPair: %s AssetType: %s consider extending synctimeoutwebsocket",
				errRESTOverwrite,
				w.exchangeName,
				u.Pair,
				u.Asset)
		}
		// Instance of illiquidity, this signal notifies that there is websocket
		// activity. We can invalidate the book and request a new snapshot. All
		// further updates through the websocket should be caught above in the
		// IsRestSnapshot() call.
		return book.ob.Invalidate(errRESTTimerLapse)
	}

	if w.bufferEnabled {
		var processed bool
		processed, err = w.processBufferUpdate(book, u)
		if err != nil {
			return err
		}

		if !processed {
			return nil
		}
	} else {
		err = w.processObUpdate(book, u)
		if err != nil {
			return err
		}
	}

	var ret *orderbook.Base
	if book.ob.VerifyOrderbook {
		// This is used here so as to not retrieve book if verification is off.
		// On every update, this will retrieve and verify orderbook depth.
		ret, err = book.ob.Retrieve()
		if err != nil {
			return err
		}
		err = ret.Verify()
		if err != nil {
			return book.ob.Invalidate(err)
		}
	}

	// Publish all state changes, disregarding verbosity or sync requirements.
	book.ob.Publish()

	if book.ticker != nil {
		select {
		case <-book.ticker.C:
			// Send update to engine websocket manager to update engine
			// sync manager to reset websocket orderbook sync timeout. This will
			// stop the fall over to REST protocol fetching of orderbook data.
		default:
			if !w.verbose {
				// We do not need to send an update to the sync manager within
				// this time window unless verbose is turned on.
				return nil
			}
		}
	}

	// A nil ticker means that a zero publish period has been set and the entire
	// websocket updates will be sent to the engine websocket manager for
	// display purposes. Same as being verbose.
	w.dataHandler <- book.ob
	return nil
}

// processBufferUpdate stores update into buffer, when buffer at capacity as
// defined by w.obBufferLimit it well then sort and apply updates.
func (w *Orderbook) processBufferUpdate(o *orderbookHolder, u *orderbook.Update) (bool, error) {
	*o.buffer = append(*o.buffer, *u)
	if len(*o.buffer) < w.obBufferLimit {
		return false, nil
	}

	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(*o.buffer, func(i, j int) bool {
				return (*o.buffer)[i].UpdateID < (*o.buffer)[j].UpdateID
			})
		} else {
			sort.Slice(*o.buffer, func(i, j int) bool {
				return (*o.buffer)[i].UpdateTime.Before((*o.buffer)[j].UpdateTime)
			})
		}
	}
	for i := range *o.buffer {
		err := w.processObUpdate(o, &(*o.buffer)[i])
		if err != nil {
			return false, err
		}
	}
	// clear buffer of old updates
	*o.buffer = nil
	return true, nil
}

// processObUpdate processes updates either by its corresponding id or by
// price level
func (w *Orderbook) processObUpdate(o *orderbookHolder, u *orderbook.Update) error {
	if w.updateEntriesByID {
		return o.updateByIDAndAction(u)
	}
	o.updateByPrice(u)
	if w.checksum != nil {
		compare, err := o.ob.Retrieve()
		if err != nil {
			return err
		}
		err = w.checksum(compare, u.Checksum)
		if err != nil {
			return o.ob.Invalidate(err)
		}
		o.updateID = u.UpdateID
	}
	return nil
}

// updateByPrice amends amount if match occurs by price, deletes if amount is
// zero or less and inserts if not found.
func (o *orderbookHolder) updateByPrice(updts *orderbook.Update) {
	o.ob.UpdateBidAskByPrice(updts)
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (o *orderbookHolder) updateByIDAndAction(updts *orderbook.Update) error {
	switch updts.Action {
	case orderbook.Amend:
		err := o.ob.UpdateBidAskByID(updts)
		if err != nil {
			return fmt.Errorf("%v %w", errAmendFailure, err)
		}
	case orderbook.Delete:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := o.ob.GetName() == "Bitfinex" && o.ob.IsFundingRate()
		err := o.ob.DeleteBidAskByID(updts, bypassErr)
		if err != nil {
			return fmt.Errorf("%v %w", errDeleteFailure, err)
		}
	case orderbook.Insert:
		err := o.ob.InsertBidAskByID(updts)
		if err != nil {
			return fmt.Errorf("%v %w", errInsertFailure, err)
		}
	case orderbook.UpdateInsert:
		err := o.ob.UpdateInsertByID(updts)
		if err != nil {
			return fmt.Errorf("%v %w", errUpdateInsertFailure, err)
		}
	default:
		return fmt.Errorf("%w [%d]", errInvalidAction, updts.Action)
	}
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (w *Orderbook) LoadSnapshot(book *orderbook.Base) error {
	// Checks if book can deploy to linked list
	err := book.Verify()
	if err != nil {
		return err
	}

	w.mtx.Lock()
	defer w.mtx.Unlock()
	holder, ok := w.ob[Key{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}]
	if !ok {
		// Associate orderbook pointer with local exchange depth map
		var depth *orderbook.Depth
		depth, err = orderbook.DeployDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			return err
		}
		depth.AssignOptions(book)
		buffer := make([]orderbook.Update, w.obBufferLimit)

		var ticker *time.Ticker
		if w.publishPeriod != 0 {
			ticker = time.NewTicker(w.publishPeriod)
		}
		holder = &orderbookHolder{ob: depth, buffer: &buffer, ticker: ticker}
		w.ob[Key{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}] = holder
	}

	holder.updateID = book.LastUpdateID
	holder.ob.LoadSnapshot(book.Bids,
		book.Asks,
		book.LastUpdateID,
		book.LastUpdated,
		false)

	if holder.ob.VerifyOrderbook {
		// This is used here so as to not retrieve book if verification is off.
		// Checks to see if orderbook snapshot that was deployed has not been
		// altered in any way
		book, err = holder.ob.Retrieve()
		if err != nil {
			return err
		}
		err = book.Verify()
		if err != nil {
			return holder.ob.Invalidate(err)
		}
	}

	holder.ob.Publish()
	w.dataHandler <- holder.ob
	return nil
}

// GetOrderbook returns an orderbook copy as orderbook.Base
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	book, ok := w.ob[Key{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", w.exchangeName, p, a, errDepthNotFound)
	}
	return book.ob.Retrieve()
}

// FlushBuffer flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *Orderbook) FlushBuffer() {
	w.mtx.Lock()
	w.ob = make(map[Key]*orderbookHolder)
	w.mtx.Unlock()
}

// FlushOrderbook flushes independent orderbook
func (w *Orderbook) FlushOrderbook(p currency.Pair, a asset.Item) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	book, ok := w.ob[Key{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	if !ok {
		return fmt.Errorf("cannot flush orderbook %s %s %s %w",
			w.exchangeName,
			p,
			a,
			errDepthNotFound)
	}
	// error not needed in this return
	_ = book.ob.Invalidate(errOrderbookFlushed)
	return nil
}

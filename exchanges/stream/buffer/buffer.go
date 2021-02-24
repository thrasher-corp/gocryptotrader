package buffer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const packageError = "websocket orderbook buffer error: %w"

var (
	errUnsetExchangeName            = errors.New("exchange name unset")
	errUnsetDataHandler             = errors.New("datahandler unset")
	errIssueBufferEnabledButNoLimit = errors.New("buffer enabled but no limit set")
	errUpdateIsNil                  = errors.New("update is nil")
	errUpdateNoTargets              = errors.New("update bid/ask targets cannot be nil")
	errDepthNotFound                = errors.New("orderbook depth not found")
)

// Setup sets private variables
func (w *Orderbook) Setup(obBufferLimit int,
	bufferEnabled,
	sortBuffer,
	sortBufferByUpdateIDs,
	updateEntriesByID bool, exchangeName string, dataHandler chan interface{}) error {
	if exchangeName == "" {
		return fmt.Errorf(packageError, errUnsetExchangeName)
	}
	if dataHandler == nil {
		return fmt.Errorf(packageError, errUnsetDataHandler)
	}
	if bufferEnabled && obBufferLimit < 1 {
		return fmt.Errorf(packageError, errIssueBufferEnabledButNoLimit)
	}
	w.obBufferLimit = obBufferLimit
	w.bufferEnabled = bufferEnabled
	w.sortBuffer = sortBuffer
	w.sortBufferByUpdateIDs = sortBufferByUpdateIDs
	w.updateEntriesByID = updateEntriesByID
	w.exchangeName = exchangeName
	w.dataHandler = dataHandler
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
	return nil
}

// validate validates update against setup values
func (w *Orderbook) validate(u *Update) error {
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
func (w *Orderbook) Update(u *Update) error {
	if err := w.validate(u); err != nil {
		return err
	}
	w.m.Lock()
	obLookup, ok := w.ob[u.Pair.Base][u.Pair.Quote][u.Asset]
	if !ok {
		w.m.Unlock()
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	if w.bufferEnabled {
		processed, err := w.processBufferUpdate(obLookup, u)
		w.m.Unlock()
		if err != nil {
			return err
		}

		if !processed {
			return nil
		}
	} else {
		err := w.processObUpdate(obLookup, u)
		w.m.Unlock()
		if err != nil {
			return err
		}
	}

	// Send pointer to orderbook.Depth to datahandler for logging purposes
	select {
	case w.dataHandler <- obLookup.ob:
	default:
		// If no receiver, discard alert as this will slow down future updates
	}
	return nil
}

// processBufferUpdate stores update into buffer, when buffer at capacity as
// defined by w.obBufferLimit it well then sort and apply updates.
func (w *Orderbook) processBufferUpdate(o *orderbookHolder, u *Update) (bool, error) {
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
func (w *Orderbook) processObUpdate(o *orderbookHolder, u *Update) error {
	o.LastUpdateID = u.UpdateID
	if w.updateEntriesByID {
		return o.updateByIDAndAction(u)
	}
	return o.updateByPrice(u)
}

// updateByPrice ammends amount if match occurs by price, deletes if amount is
// zero or less and inserts if not found.
func (o *orderbookHolder) updateByPrice(updts *Update) error {
	return o.ob.UpdateBidAskByPrice(updts.Bids, updts.Asks, updts.MaxDepth)
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (o *orderbookHolder) updateByIDAndAction(updts *Update) error {
	switch updts.Action {
	case Amend:
		return o.ob.UpdateBidAskByID(updts.Bids, updts.Asks)
	case Delete:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := o.ob.Exchange == "Bitfinex" && o.ob.IsFundingRate
		return o.ob.DeleteBidAskByID(updts.Bids, updts.Asks, bypassErr)
	case Insert:
		o.ob.InsertBidAskByID(updts.Bids, updts.Asks)
	case UpdateInsert:
		o.ob.UpdateInsertByID(updts.Bids, updts.Asks)
	default:
		return fmt.Errorf("invalid action [%s]", updts.Action)
	}
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (w *Orderbook) LoadSnapshot(book *orderbook.Base) error {
	// fmt.Printf("BOOK: %+v\n", book)
	w.m.Lock()
	m1, ok := w.ob[book.Pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*orderbookHolder)
		w.ob[book.Pair.Base] = m1
	}
	m2, ok := m1[book.Pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*orderbookHolder)
		m1[book.Pair.Quote] = m2
	}
	holder, ok := m2[book.Asset]
	if !ok {
		// Associate orderbook pointer with local exchange depth map
		depth, err := orderbook.GetDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			w.m.Unlock()
			return err
		}
		// TODO ADD THIS IN!!!
		// m3.ob.LastUpdateID = book.LastUpdateID
		depth.Asset = book.Asset
		depth.Exchange = book.Exchange
		depth.Asset = book.Asset
		depth.HasChecksumValidation = book.HasChecksumValidation
		depth.IsFundingRate = book.IsFundingRate
		depth.LastUpdateID = book.LastUpdateID
		depth.LastUpdated = book.LastUpdated
		depth.NotAggregated = book.NotAggregated
		depth.Pair = book.Pair
		depth.RestSnapshot = book.RestSnapshot
		depth.VerificationBypass = book.VerificationBypass
		buffer := make([]Update, w.obBufferLimit)
		holder = &orderbookHolder{ob: depth, buffer: &buffer}
		m2[book.Asset] = holder
	}

	if book.CanVerify() {
		// Checks if book can deploy to linked list
		err := book.Verify()
		if err != nil {
			return err
		}
	}

	err := holder.ob.LoadSnapshot(book.Bids, book.Asks, false)
	w.m.Unlock()
	if err != nil {
		return err
	}

	if book.CanVerify() {
		// Checks to see if booked that was deployed has not been altered in
		// any way
		err = holder.ob.Retrieve().Verify()
		if err != nil {
			return err
		}
	}

	select {
	case w.dataHandler <- book:
	default:
		// If no receiver, discard alert as this will slow down future updates
	}
	return nil
}

// GetOrderbook returns orderbook stored in current buffer
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	w.m.Lock()
	book, ok := w.ob[p.Base][p.Quote][a]
	w.m.Unlock()
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w",
			w.exchangeName,
			p,
			a,
			errDepthNotFound)
	}
	return book.ob.Retrieve(), nil
}

// empty eager bucket
var empty map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder

// FlushBuffer flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *Orderbook) FlushBuffer() {
	w.m.Lock()
	w.ob = empty
	w.m.Unlock()
	empty = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
}

// FlushOrderbook flushes independent orderbook
func (w *Orderbook) FlushOrderbook(p currency.Pair, a asset.Item) error {
	w.m.Lock()
	book, ok := w.ob[p.Base][p.Quote][a]
	w.m.Unlock()
	if !ok {
		return fmt.Errorf("%s %s %s %w",
			w.exchangeName,
			p,
			a,
			errDepthNotFound)
	}
	book.ob.Flush()
	return nil
}

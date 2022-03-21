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
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
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
	defer w.m.Unlock()
	book, ok := w.ob[u.Pair.Base][u.Pair.Quote][u.Asset]
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
	// because the book is highly illiquid. TODO: Book resubscribe on websocket.
	if book.ob.IsRestSnapshot() {
		if w.verbose {
			log.Warnf(log.WebsocketMgr,
				"%s for Exchange %s CurrencyPair: %s AssetType: %s consider extending synctimeoutwebsocket",
				errRESTOverwrite,
				w.exchangeName,
				u.Pair,
				u.Asset)
		}
		return fmt.Errorf("%w for Exchange %s CurrencyPair: %s AssetType: %s",
			errRESTOverwrite,
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	if w.bufferEnabled {
		processed, err := w.processBufferUpdate(book, u)
		if err != nil {
			return err
		}

		if !processed {
			return nil
		}
	} else {
		err := w.processObUpdate(book, u)
		if err != nil {
			return err
		}
	}

	if book.ob.VerifyOrderbook { // This is used here so as to not retrieve
		// book if verification is off.
		// On every update, this will retrieve and verify orderbook depths
		err := book.ob.Retrieve().Verify()
		if err != nil {
			return err
		}
	}

	// a nil ticker means that a zero publish period has been requested,
	// this means publish now whatever was received with no throttling
	if book.ticker == nil {
		go func() {
			w.dataHandler <- book.ob.Retrieve()
			book.ob.Publish()
		}()

		return nil
	}

	select {
	case <-book.ticker.C:
		// Opted to wait for receiver because we are limiting here and the sync
		// manager requires update
		go func() {
			w.dataHandler <- book.ob.Retrieve()
			book.ob.Publish()
		}()
	default:
		// We do not need to send an update to the sync manager within this time
		// window unless verbose is turned on
		if w.verbose {
			w.dataHandler <- book.ob.Retrieve()
			book.ob.Publish()
		}
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
	if w.updateEntriesByID {
		return o.updateByIDAndAction(u)
	}
	o.updateByPrice(u)
	if w.checksum != nil {
		err := w.checksum(o.ob.Retrieve(), u.Checksum)
		if err != nil {
			return err
		}
		o.updateID = u.UpdateID
	}
	return nil
}

// updateByPrice ammends amount if match occurs by price, deletes if amount is
// zero or less and inserts if not found.
func (o *orderbookHolder) updateByPrice(updts *Update) {
	o.ob.UpdateBidAskByPrice(updts.Bids,
		updts.Asks,
		updts.MaxDepth,
		updts.UpdateID,
		updts.UpdateTime)
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (o *orderbookHolder) updateByIDAndAction(updts *Update) error {
	switch updts.Action {
	case Amend:
		return o.ob.UpdateBidAskByID(updts.Bids,
			updts.Asks,
			updts.UpdateID,
			updts.UpdateTime)
	case Delete:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := o.ob.GetName() == "Bitfinex" && o.ob.IsFundingRate()
		return o.ob.DeleteBidAskByID(updts.Bids,
			updts.Asks,
			bypassErr,
			updts.UpdateID,
			updts.UpdateTime)
	case Insert:
		return o.ob.InsertBidAskByID(updts.Bids,
			updts.Asks,
			updts.UpdateID,
			updts.UpdateTime)
	case UpdateInsert:
		return o.ob.UpdateInsertByID(updts.Bids,
			updts.Asks,
			updts.UpdateID,
			updts.UpdateTime)
	default:
		return fmt.Errorf("invalid action [%s]", updts.Action)
	}
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (w *Orderbook) LoadSnapshot(book *orderbook.Base) error {
	w.m.Lock()
	defer w.m.Unlock()
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
		depth, err := orderbook.DeployDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			return err
		}
		depth.AssignOptions(book)
		buffer := make([]Update, w.obBufferLimit)

		var ticker *time.Ticker
		if w.publishPeriod != 0 {
			ticker = time.NewTicker(w.publishPeriod)
		}
		holder = &orderbookHolder{
			ob:     depth,
			buffer: &buffer,
			ticker: ticker,
		}
		m2[book.Asset] = holder
	}

	holder.updateID = book.LastUpdateID

	// Checks if book can deploy to linked list
	err := book.Verify()
	if err != nil {
		return err
	}

	holder.ob.LoadSnapshot(book.Bids,
		book.Asks,
		book.LastUpdateID,
		book.LastUpdated,
		false,
	)

	if holder.ob.VerifyOrderbook { // This is used here so as to not retrieve
		// book if verification is off.
		// Checks to see if orderbook snapshot that was deployed has not been
		// altered in any way
		err = holder.ob.Retrieve().Verify()
		if err != nil {
			return err
		}
	}

	w.dataHandler <- holder.ob.Retrieve()
	holder.ob.Publish()
	return nil
}

// GetOrderbook returns an orderbook copy as orderbook.Base
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	w.m.Lock()
	defer w.m.Unlock()
	book, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w",
			w.exchangeName,
			p,
			a,
			errDepthNotFound)
	}
	return book.ob.Retrieve(), nil
}

// FlushBuffer flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *Orderbook) FlushBuffer() {
	w.m.Lock()
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
	w.m.Unlock()
}

// FlushOrderbook flushes independent orderbook
func (w *Orderbook) FlushOrderbook(p currency.Pair, a asset.Item) error {
	w.m.Lock()
	defer w.m.Unlock()
	book, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return fmt.Errorf("cannot flush orderbook %s %s %s %w",
			w.exchangeName,
			p,
			a,
			errDepthNotFound)
	}
	book.ob.Flush()
	return nil
}

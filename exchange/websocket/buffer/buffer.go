package buffer

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const packageError = "websocket orderbook buffer error: %w"

var (
	errExchangeConfigNil            = errors.New("exchange config is nil")
	errBufferConfigNil              = errors.New("buffer config is nil")
	errUnsetDataHandler             = errors.New("datahandler unset")
	errIssueBufferEnabledButNoLimit = errors.New("buffer enabled but no limit set")
	errOrderbookFlushed             = errors.New("orderbook flushed")
)

// Setup sets private variables
func (o *Orderbook) Setup(exchangeConfig *config.Exchange, c *Config, dataHandler chan<- any) error {
	if exchangeConfig == nil { // exchange config fields are checked in websocket package prior to calling this, so further checks are not needed
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

	// NOTE: These variables are set by config.json under "orderbook" for each individual exchange
	o.bufferEnabled = exchangeConfig.Orderbook.WebsocketBufferEnabled
	o.obBufferLimit = exchangeConfig.Orderbook.WebsocketBufferLimit

	o.sortBuffer = c.SortBuffer
	o.sortBufferByUpdateIDs = c.SortBufferByUpdateIDs
	o.exchangeName = exchangeConfig.Name
	o.dataHandler = dataHandler
	o.ob = make(map[key.PairAsset]*orderbookHolder)
	o.verbose = exchangeConfig.Verbose
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (o *Orderbook) LoadSnapshot(book *orderbook.Book) error {
	if err := book.Validate(); err != nil {
		return err
	}

	o.m.RLock()
	holder, ok := o.ob[key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}]
	o.m.RUnlock()
	if !ok {
		o.m.Lock()
		// Associate orderbook pointer with local exchange depth map
		depth, err := orderbook.DeployDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			o.m.Unlock()
			return err
		}
		depth.AssignOptions(book)
		holder = &orderbookHolder{ob: depth, buffer: make([]orderbook.Update, 0, o.obBufferLimit)}
		o.ob[key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}] = holder
		o.m.Unlock()
	}

	book.RestSnapshot = false
	if err := holder.ob.LoadSnapshot(book); err != nil {
		return err
	}

	holder.ob.Publish()
	o.dataHandler <- holder.ob
	return nil
}

// Update updates a stored pointer to an orderbook.Depth struct containing bid and ask Tranches, this switches between
// the usage of a buffered update
func (o *Orderbook) Update(u *orderbook.Update) error {
	o.m.RLock()
	holder, ok := o.ob[key.PairAsset{Base: u.Pair.Base.Item, Quote: u.Pair.Quote.Item, Asset: u.Asset}]
	o.m.RUnlock()
	if !ok {
		return fmt.Errorf("%w for Exchange %s CurrencyPair: %s AssetType: %s", orderbook.ErrDepthNotFound, o.exchangeName, u.Pair, u.Asset)
	}

	if o.bufferEnabled {
		if processed, err := o.processBufferUpdate(holder, u); err != nil || !processed {
			return err
		}
	} else {
		if err := holder.ob.ProcessUpdate(u); err != nil {
			return err
		}
	}

	// Publish all state changes, disregarding verbosity or sync requirements.
	holder.ob.Publish()
	o.dataHandler <- holder.ob
	return nil
}

// processBufferUpdate stores update into buffer, when buffer at capacity as
// defined by o.obBufferLimit it well then sort and apply updates.
func (o *Orderbook) processBufferUpdate(holder *orderbookHolder, u *orderbook.Update) (bool, error) {
	holder.buffer = append(holder.buffer, *u)
	if len(holder.buffer) < o.obBufferLimit {
		return false, nil
	}

	if o.sortBuffer {
		// sort by last updated to ensure each update is in order
		if o.sortBufferByUpdateIDs {
			slices.SortFunc(holder.buffer, func(a, b orderbook.Update) int {
				return cmp.Compare(a.UpdateID, b.UpdateID)
			})
		} else {
			slices.SortFunc(holder.buffer, func(a, b orderbook.Update) int {
				return a.UpdateTime.Compare(b.UpdateTime)
			})
		}
	}

	// Always empty the buffer after processing, even if there's an error
	defer func() { holder.buffer = holder.buffer[:0] }()

	for i := range holder.buffer {
		if err := holder.ob.ProcessUpdate(&holder.buffer[i]); err != nil {
			return false, err
		}
	}

	return true, nil
}

// GetOrderbook returns an orderbook copy as orderbook.Book
func (o *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrInvalidAsset
	}
	o.m.RLock()
	holder, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%s %w: %s.%s", o.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return holder.ob.Retrieve()
}

// LastUpdateID returns the last update ID of the orderbook
func (o *Orderbook) LastUpdateID(p currency.Pair, a asset.Item) (int64, error) {
	if p.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return 0, asset.ErrInvalidAsset
	}
	o.m.RLock()
	book, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return 0, fmt.Errorf("%s %w: %s.%s", o.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return book.ob.LastUpdateID()
}

// FlushBuffer flushes individual orderbook buffers while keeping the orderbook lookups intact and ready for new updates
// when a connection is re-established.
func (o *Orderbook) FlushBuffer() {
	o.m.Lock()
	for _, holder := range o.ob {
		holder.buffer = holder.buffer[:0]
	}
	o.m.Unlock()
}

// InvalidateOrderbook invalidates the orderbook so no trading can occur on potential corrupted data
// TODO: Add in reason for invalidation for debugging purposes.
func (o *Orderbook) InvalidateOrderbook(p currency.Pair, a asset.Item) error {
	o.m.RLock()
	holder, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return fmt.Errorf("cannot invalidate orderbook %s %s %s %w", o.exchangeName, p, a, orderbook.ErrDepthNotFound)
	}
	// Invalidate returns a formatted version of the error it's passed
	// In this context we don't need that, since this method only returns an error if it cannot invalidate
	_ = holder.ob.Invalidate(errOrderbookFlushed)
	return nil
}

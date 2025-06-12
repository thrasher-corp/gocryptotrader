package buffer

import (
	"errors"
	"fmt"
	"sort"

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
func (w *Orderbook) Setup(exchangeConfig *config.Exchange, c *Config, dataHandler chan<- any) error {
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
	w.bufferEnabled = exchangeConfig.Orderbook.WebsocketBufferEnabled
	w.obBufferLimit = exchangeConfig.Orderbook.WebsocketBufferLimit

	w.sortBuffer = c.SortBuffer
	w.sortBufferByUpdateIDs = c.SortBufferByUpdateIDs
	w.exchangeName = exchangeConfig.Name
	w.dataHandler = dataHandler
	w.ob = make(map[key.PairAsset]*orderbookHolder)
	w.verbose = exchangeConfig.Verbose
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (w *Orderbook) LoadSnapshot(book *orderbook.Base) error {
	if err := book.Verify(); err != nil {
		return err
	}

	w.m.RLock()
	holder, ok := w.ob[key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}]
	w.m.RUnlock()
	if !ok {
		w.m.Lock()
		// Associate orderbook pointer with local exchange depth map
		depth, err := orderbook.DeployDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			w.m.Unlock()
			return err
		}
		depth.AssignOptions(book)
		holder = &orderbookHolder{ob: depth, buffer: make([]orderbook.Update, 0, w.obBufferLimit)}
		w.ob[key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}] = holder
		w.m.Unlock()
	}

	book.RestSnapshot = false
	if err := holder.ob.LoadSnapshot(book); err != nil {
		return err
	}

	holder.ob.Publish()
	w.dataHandler <- holder.ob
	return nil
}

// Update updates a stored pointer to an orderbook.Depth struct containing bid and ask Tranches, this switches between
// the usage of a buffered update
func (w *Orderbook) Update(u *orderbook.Update) error {
	w.m.RLock()
	holder, ok := w.ob[key.PairAsset{Base: u.Pair.Base.Item, Quote: u.Pair.Quote.Item, Asset: u.Asset}]
	w.m.RUnlock()
	if !ok {
		return fmt.Errorf("%w for Exchange %s CurrencyPair: %s AssetType: %s", orderbook.ErrDepthNotFound, w.exchangeName, u.Pair, u.Asset)
	}

	if w.bufferEnabled {
		if processed, err := w.processBufferUpdate(holder, u); err != nil || !processed {
			return err
		}
	} else {
		if err := holder.ob.ProcessUpdate(u); err != nil {
			return err
		}
	}

	// Publish all state changes, disregarding verbosity or sync requirements.
	holder.ob.Publish()
	w.dataHandler <- holder.ob
	return nil
}

// processBufferUpdate stores update into buffer, when buffer at capacity as
// defined by w.obBufferLimit it well then sort and apply updates.
func (w *Orderbook) processBufferUpdate(holder *orderbookHolder, u *orderbook.Update) (bool, error) {
	holder.buffer = append(holder.buffer, *u)
	if len(holder.buffer) < w.obBufferLimit {
		return false, nil
	}

	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(holder.buffer, func(i, j int) bool {
				return holder.buffer[i].UpdateID < holder.buffer[j].UpdateID
			})
		} else {
			sort.Slice(holder.buffer, func(i, j int) bool {
				return holder.buffer[i].UpdateTime.Before(holder.buffer[j].UpdateTime)
			})
		}
	}

	// clear buffer of old updates on all error pathways as any error will invalidate the orderbook and will require
	// a new snapshot to be loaded
	defer func() { holder.buffer = holder.buffer[:0] }()

	for i := range holder.buffer {
		if err := holder.ob.ProcessUpdate(&holder.buffer[i]); err != nil {
			return false, err
		}
	}

	return true, nil
}

// GetOrderbook returns an orderbook copy as orderbook.Base
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrInvalidAsset
	}
	w.m.RLock()
	holder, ok := w.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	w.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%s %w: %s.%s", w.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return holder.ob.Retrieve()
}

// LastUpdateID returns the last update ID of the orderbook
func (w *Orderbook) LastUpdateID(p currency.Pair, a asset.Item) (int64, error) {
	if p.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return 0, asset.ErrInvalidAsset
	}
	w.m.RLock()
	book, ok := w.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	w.m.RUnlock()
	if !ok {
		return 0, fmt.Errorf("%s %w: %s.%s", w.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return book.ob.LastUpdateID()
}

// FlushBuffer flushes individual orderbook buffers while keeping the orderbook lookups intact and ready for new updates
// when a connection is re-established.
func (w *Orderbook) FlushBuffer() {
	w.m.Lock()
	for _, holder := range w.ob {
		holder.buffer = holder.buffer[:0]
	}
	w.m.Unlock()
}

// InvalidateOrderbook invalidates the orderbook so no trading can occur on potential corrupted data
// TODO: Add in reason for invalidation for debugging purposes.
func (w *Orderbook) InvalidateOrderbook(p currency.Pair, a asset.Item) error {
	w.m.RLock()
	holder, ok := w.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	w.m.RUnlock()
	if !ok {
		return fmt.Errorf("cannot invalidate orderbook %s %s %s %w", w.exchangeName, p, a, orderbook.ErrDepthNotFound)
	}
	// error not needed in this return
	_ = holder.ob.Invalidate(errOrderbookFlushed)
	return nil
}

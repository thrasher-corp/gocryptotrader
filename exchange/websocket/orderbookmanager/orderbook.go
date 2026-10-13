package orderbookmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var errOrderbookInvalidated = errors.New("orderbook invalidated")

// Orderbook defines a local cache of orderbooks for amending, appending
// and deleting changes and updates the main store for a stream.
type Orderbook struct {
	ob           map[key.PairAsset]*orderbook.Depth
	exchangeName string
	dataHandler  *stream.Relay
	verbose      bool
	m            sync.RWMutex
}

// Setup initialises the orderbook manager.
func (o *Orderbook) Setup(exchangeName string, dataHandler *stream.Relay, verbose bool) error {
	if err := common.NilGuard(dataHandler); err != nil {
		return err
	}
	o.exchangeName = exchangeName
	o.dataHandler = dataHandler
	o.ob = make(map[key.PairAsset]*orderbook.Depth)
	o.verbose = verbose
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data from websocket
func (o *Orderbook) LoadSnapshot(ctx context.Context, book *orderbook.Book) error {
	if err := book.Validate(); err != nil {
		return err
	}

	bookKey := key.PairAsset{Base: book.Pair.Base.Item, Quote: book.Pair.Quote.Item, Asset: book.Asset}
	o.m.RLock()
	depth, ok := o.ob[bookKey]
	o.m.RUnlock()
	if !ok {
		o.m.Lock()
		depth, ok = o.ob[bookKey]
		if !ok {
			// Associate orderbook pointer with local exchange depth map
			newDepth, err := orderbook.DeployDepth(book.Exchange, book.Pair, book.Asset)
			if err != nil {
				o.m.Unlock()
				return err
			}
			newDepth.AssignOptions(book)
			depth = newDepth
			o.ob[bookKey] = depth
		}
		o.m.Unlock()
	}

	book.RestSnapshot = false
	if err := depth.LoadSnapshot(book); err != nil {
		return err
	}

	depth.Publish()
	return o.dataHandler.Send(ctx, depth)
}

// Update updates a stored pointer to an orderbook.Depth struct containing bid and ask levels.
func (o *Orderbook) Update(ctx context.Context, u *orderbook.Update) error {
	o.m.RLock()
	depth, ok := o.ob[key.PairAsset{Base: u.Pair.Base.Item, Quote: u.Pair.Quote.Item, Asset: u.Asset}]
	o.m.RUnlock()
	if !ok {
		return fmt.Errorf("%w for Exchange %s CurrencyPair: %s AssetType: %s", orderbook.ErrDepthNotFound, o.exchangeName, u.Pair, u.Asset)
	}
	return o.updateDepth(ctx, depth, u)
}

// updateDepth avoids repeating the map lookup when the caller already has the depth for the update key.
func (o *Orderbook) updateDepth(ctx context.Context, depth *orderbook.Depth, u *orderbook.Update) error {
	if err := depth.ProcessUpdate(u); err != nil {
		return err
	}

	// Publish all state changes, disregarding verbosity or sync requirements.
	depth.Publish()
	return o.dataHandler.Send(ctx, depth)
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
	depth, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%s %w: %s.%s", o.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return depth.Retrieve()
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
	depth, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return 0, fmt.Errorf("%s %w: %s.%s", o.exchangeName, orderbook.ErrDepthNotFound, a, p)
	}
	return depth.LastUpdateID()
}

// InvalidateOrderbook invalidates the orderbook so no trading can occur on potential corrupted data
// TODO: Add in reason for invalidation for debugging purposes.
func (o *Orderbook) InvalidateOrderbook(p currency.Pair, a asset.Item) error {
	o.m.RLock()
	depth, ok := o.ob[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	o.m.RUnlock()
	if !ok {
		return fmt.Errorf("cannot invalidate orderbook %s %s %s %w", o.exchangeName, p, a, orderbook.ErrDepthNotFound)
	}
	// Invalidate returns a formatted version of the error it's passed
	// In this context we don't need that, since this method only returns an error if it cannot invalidate
	_ = depth.Invalidate(errOrderbookInvalidated)
	return nil
}

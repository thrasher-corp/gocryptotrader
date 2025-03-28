package gateio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
)

var (
	errOrderbookSnapshotBehind = errors.New("orderbook snapshot is behind update")
	wsOBUpdateMgr              = &wsOBUpdateManager{m: make(map[key.PairAsset]*updateCache)}
)

type wsOBUpdateManager struct {
	m   map[key.PairAsset]*updateCache
	mtx sync.Mutex
}

type updateCache struct {
	buffer   []updateWithFirstID
	updating bool
	mtx      sync.Mutex
}

type updateWithFirstID struct {
	update  *orderbook.Update
	firstID int64
}

func (m *wsOBUpdateManager) applyUpdate(ctx context.Context, g *Gateio, limit uint64, firstID int64, update *orderbook.Update) error {
	book, err := g.Websocket.Orderbook.GetOrderbook(update.Pair, update.Asset)
	if err != nil && !errors.Is(err, buffer.ErrDepthNotFound) {
		return err
	}

	cache := m.getCache(update.Pair, update.Asset)
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	if cache.updating {
		cache.buffer = append(cache.buffer, updateWithFirstID{update: update, firstID: firstID})
		return nil
	}

	if book == nil || book.LastUpdateID+1 < firstID /*orderbook is behind notifications so refresh*/ {
		cache.updating = true
		go func() {
			// REST Orderbook IDs are behind the ws update feed by about 50-100 changes; inline rudimentary delay to cache
			time.Sleep(time.Second)
			if err := cache.updateOrderbookAndApply(ctx, g, update.Pair, update.Asset, limit); err != nil {
				g.Websocket.DataHandler <- fmt.Errorf("%v %v update and apply orderbook: %w", update.Pair, update.Asset, err)
			}
		}()
		cache.buffer = append(cache.buffer, updateWithFirstID{update: update, firstID: firstID})
		return nil
	}
	return cache.applyUpdate(g, update)
}

func (m *wsOBUpdateManager) getCache(p currency.Pair, a asset.Item) *updateCache {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	cache, ok := m.m[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	if !ok {
		cache = &updateCache{}
		m.m[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
	}
	return cache
}

func (c *updateCache) updateOrderbookAndApply(ctx context.Context, g *Gateio, pair currency.Pair, a asset.Item, limit uint64) error {
	defer c.mtx.Unlock()
	defer func() { c.buffer = nil; /*gc buffer; unused until snapshot required*/ c.updating = false }()

	// TODO: When templates are introduced for all assets define channel key and use g.Websocket.GetSubscription(ChannelKey{&subscription.Subscription{Channel: channelName}})
	// to get the subscription and levels for limit. So that this can scale to config changes. Spot is currently the only
	// asset with templates but it has only one level.
	book, err := g.UpdateOrderbookWithLimit(ctx, pair, a, limit)
	c.mtx.Lock() // lock here to prevent ws handle data interference with REST request above
	if err != nil {
		return err
	}

	if a != asset.Spot {
		err = g.Websocket.Orderbook.LoadSnapshot(book)
		if err != nil {
			return err
		}
	} else {
		// Spot, Margin, and Cross Margin books are the same
		for _, a := range standardMarginAssetTypes {
			if enabled, _ := g.CurrencyPairs.IsPairEnabled(pair, a); !enabled {
				continue
			}
			book.Asset = a
			err = g.Websocket.Orderbook.LoadSnapshot(book)
			if err != nil {
				return err
			}
		}
	}
	return c.applyPending(g, a)
}

func (c *updateCache) applyPending(g *Gateio, a asset.Item) error {
	for _, data := range c.buffer {
		book, err := g.Websocket.Orderbook.GetOrderbook(data.update.Pair, a)
		if err != nil {
			return err
		}
		nextID := book.LastUpdateID + 1
		if data.firstID > nextID {
			return errOrderbookSnapshotBehind
		}

		if data.update.UpdateID < nextID {
			continue // skip updates that are behind the current orderbook
		}
		if err := c.applyUpdate(g, data.update); err != nil {
			return err
		}
	}
	return nil
}

func (c *updateCache) applyUpdate(g *Gateio, update *orderbook.Update) error {
	if update.Asset != asset.Spot {
		return g.Websocket.Orderbook.Update(update)
	}

	for _, a := range standardMarginAssetTypes {
		if enabled, _ := g.CurrencyPairs.IsPairEnabled(update.Pair, a); !enabled {
			continue
		}
		update.Asset = a
		if err := g.Websocket.Orderbook.Update(update); err != nil {
			return err
		}
	}

	return nil
}

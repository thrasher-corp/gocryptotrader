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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	errOrderbookSnapshotOutdated     = errors.New("orderbook snapshot is outdated")
	errOrderbookUpdateMessageDropped = errors.New("orderbook update message dropped between updates check")
	errPendingUpdatesNotApplied      = errors.New("pending updates not applied")
	errWSOrderbookUpdateDeadline     = errors.New("websocket orderbook update deadline exceeded")

	defaultWSOrderbookUpdateDeadline = time.Minute * 2
)

type wsOBUpdateManager struct {
	lookup  map[key.PairAsset]*updateCache
	timeout time.Duration
	mtx     sync.RWMutex
}

type updateCache struct {
	updates  []pendingUpdate
	updating bool
	ch       chan int64
	mtx      sync.Mutex
}

type pendingUpdate struct {
	update        *orderbook.Update
	firstUpdateID int64
}

func newWsOBUpdateManager(timeout time.Duration) *wsOBUpdateManager {
	return &wsOBUpdateManager{lookup: make(map[key.PairAsset]*updateCache), timeout: timeout}
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, g *Exchange, firstUpdateID int64, update *orderbook.Update) error {
	cache := m.LoadCache(update.Pair, update.Asset)
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	if cache.updating {
		cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
		select {
		case cache.ch <- update.UpdateID: // Notify SyncOrderbook of most recent update ID for inspection
		default:
		}
		return nil
	}

	lastUpdateID, err := g.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	if lastUpdateID+1 == firstUpdateID {
		return applyOrderbookUpdate(g, update)
	}

	// Orderbook is behind notifications, therefore Invalidate store
	if err := g.Websocket.Orderbook.InvalidateOrderbook(update.Pair, update.Asset); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	cache.updating = true
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})

	go func() {
		if err := cache.SyncOrderbook(ctx, g, update.Pair, update.Asset, m.timeout); err != nil {
			g.Websocket.DataHandler <- fmt.Errorf("failed to sync orderbook for %v %v: %w", update.Pair, update.Asset, err)
		}
	}()

	return nil
}

// LoadCache loads the cache for the given pair and asset. If the cache does not exist, it creates a new one.
func (m *wsOBUpdateManager) LoadCache(p currency.Pair, a asset.Item) *updateCache {
	m.mtx.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.mtx.RUnlock()
	if !ok {
		m.mtx.Lock()
		cache = &updateCache{ch: make(chan int64)}
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.mtx.Unlock()
	}
	return cache
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (c *updateCache) SyncOrderbook(ctx context.Context, g *Exchange, pair currency.Pair, a asset.Item, deadline time.Duration) error {
	// TODO: When subscription config is added for all assets update limits to use sub.Levels
	var limit uint64
	switch a {
	case asset.Spot:
		sub := g.Websocket.GetSubscription(spotOrderbookUpdateKey)
		if sub == nil {
			c.clearWithLock()
			return fmt.Errorf("no subscription found for %q", spotOrderbookUpdateKey)
		}
		// There is no way to set levels when we subscribe for this specific subscription case.
		// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
		limit = uint64(sub.Interval.Duration().Milliseconds()) //nolint:gosec // No overflow risk
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		limit = futuresOrderbookUpdateLimit
	case asset.DeliveryFutures:
		limit = deliveryFuturesUpdateLimit
	case asset.Options:
		limit = optionOrderbookUpdateLimit
	}

	// prevents rate limiter from blocking across multiple enabled pairs across connections, which consumes resources
	// when appending to the update cache
	ctxWDeadline, cancel := context.WithDeadlineCause(ctx, time.Now().Add(deadline), errWSOrderbookUpdateDeadline)
	defer cancel()

	book, err := g.fetchOrderbook(ctxWDeadline, pair, a, limit)
	if err != nil {
		c.clearWithLock()
		return err
	}

	var updateListLastUpdateID int64
	c.mtx.Lock()
	updateListLastUpdateID = c.updates[len(c.updates)-1].update.UpdateID
	c.mtx.Unlock()

	nextUpdateID := book.LastUpdateID + 1

	if updateListLastUpdateID < nextUpdateID {
	wait:
		for {
			select {
			case <-ctxWDeadline.Done():
				c.clearWithLock()
				return ctxWDeadline.Err()
			case recentPendingUpdateID := <-c.ch:
				if recentPendingUpdateID >= nextUpdateID {
					break wait
				}
			}
		}
	}

	c.mtx.Lock() // lock here to prevent ws handle data interference with REST request above
	defer func() {
		c.clearNoLock()
		c.mtx.Unlock()
	}()

	if a != asset.Spot {
		if err := g.Websocket.Orderbook.LoadSnapshot(book); err != nil {
			return err
		}
	} else {
		// Spot, Margin, and Cross Margin books are all classified as spot
		for i := range standardMarginAssetTypes {
			if enabled, _ := g.IsPairEnabled(pair, standardMarginAssetTypes[i]); !enabled {
				continue
			}
			book.Asset = standardMarginAssetTypes[i]
			if err := g.Websocket.Orderbook.LoadSnapshot(book); err != nil {
				return err
			}
		}
	}
	return c.applyPendingUpdates(g, a)
}

// applyPendingUpdates applies all pending updates to the orderbook
func (c *updateCache) applyPendingUpdates(g *Exchange, a asset.Item) error {
	var updateApplied bool
	for _, data := range c.updates {
		bookLastUpdateID, err := g.Websocket.Orderbook.LastUpdateID(data.update.Pair, a)
		if err != nil {
			return fmt.Errorf("applying pending updates: %w", err)
		}

		lastUpdateID := data.update.UpdateID // u
		nextUpdateID := bookLastUpdateID + 1 // baseId + 1
		if lastUpdateID < nextUpdateID {
			continue
		}

		firstUpdateID := data.firstUpdateID // U
		if nextUpdateID < firstUpdateID {
			return fmt.Errorf("applying pending updates: %w", errOrderbookSnapshotOutdated)
		}

		if updateApplied && firstUpdateID > nextUpdateID {
			return fmt.Errorf("applying pending updates: %w", errOrderbookUpdateMessageDropped)
		}

		if err := applyOrderbookUpdate(g, data.update); err != nil {
			return fmt.Errorf("applying pending updates: %w", err)
		}

		updateApplied = true
	}

	if !updateApplied {
		return fmt.Errorf("applying pending updates: %w", errPendingUpdatesNotApplied)
	}

	return nil
}

func (c *updateCache) clearWithLock() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.clearNoLock()
}

func (c *updateCache) clearNoLock() {
	c.updates = nil
	c.updating = false
}

// applyOrderbookUpdate applies an orderbook update to the orderbook
func applyOrderbookUpdate(g *Exchange, update *orderbook.Update) error {
	if update.Asset != asset.Spot {
		return g.Websocket.Orderbook.Update(update)
	}

	for i := range standardMarginAssetTypes {
		if enabled, _ := g.IsPairEnabled(update.Pair, standardMarginAssetTypes[i]); !enabled {
			continue
		}
		update.Asset = standardMarginAssetTypes[i]
		if err := g.Websocket.Orderbook.Update(update); err != nil {
			return err
		}
	}

	return nil
}

var spotOrderbookUpdateKey = channelKey{&subscription.Subscription{Channel: subscription.OrderbookChannel}}

var _ subscription.MatchableKey = channelKey{}

type channelKey struct {
	*subscription.Subscription
}

func (k channelKey) Match(eachKey subscription.MatchableKey) bool {
	return k.Subscription.Channel == eachKey.GetSubscription().Channel
}

func (k channelKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

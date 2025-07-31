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
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errOrderbookSnapshotOutdated = errors.New("orderbook snapshot is outdated")
	errPendingUpdatesNotApplied  = errors.New("pending updates not applied")
	errWSOrderbookUpdateDeadline = errors.New("websocket orderbook update deadline exceeded")

	defaultWSOrderbookUpdateDeadline  = time.Minute * 2
	defaultWsOrderbookUpdateTimeDelay = time.Second
	spotOrderbookUpdateKey            = subscription.MustChannelKey(subscription.OrderbookChannel)
)

type wsOBUpdateManager struct {
	lookup   map[key.PairAsset]*updateCache
	deadline time.Duration
	delay    time.Duration
	mtx      sync.RWMutex
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

func newWsOBUpdateManager(delay, deadline time.Duration) *wsOBUpdateManager {
	return &wsOBUpdateManager{lookup: make(map[key.PairAsset]*updateCache), deadline: deadline, delay: delay}
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update) error {
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

	lastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	if lastUpdateID+1 == firstUpdateID {
		return applyOrderbookUpdate(e, update)
	}

	// Orderbook is behind notifications, therefore Invalidate store
	if err := e.Websocket.Orderbook.InvalidateOrderbook(update.Pair, update.Asset); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	cache.updating = true
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})

	go func() {
		if err := cache.SyncOrderbook(ctx, e, update.Pair, update.Asset, m.delay, m.deadline); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", e.Name, update.Pair, update.Asset, err)
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
func (c *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, pair currency.Pair, a asset.Item, delay, deadline time.Duration) error {
	limit, err := c.extractOrderbookLimit(e, a)
	if err != nil {
		c.clearWithLock()
		return err
	}

	// Rest requests can be behind websocket updates by a large margin, we need to wait here so as to allow the cache to
	// fill with updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		c.clearWithLock()
		return ctx.Err()
	case <-time.After(delay):
	}

	// prevents rate limiter from blocking across multiple enabled pairs and connections, which consumes resources
	// when appending to the update cache.
	ctxWDeadline, cancel := context.WithDeadline(ctx, time.Now().Add(deadline))
	defer cancel()

	book, err := e.fetchOrderbook(ctxWDeadline, pair, a, limit)
	if err != nil {
		c.clearWithLock()
		return err
	}

	if err := c.waitForUpdate(ctxWDeadline, book.LastUpdateID+1); err != nil {
		c.clearWithLock()
		return err
	}

	c.mtx.Lock() // lock here to prevent ws handle data interference with REST request above
	defer func() {
		c.clearNoLock()
		c.mtx.Unlock()
	}()

	if a != asset.Spot {
		if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
			return err
		}
	} else {
		// Spot, Margin, and Cross Margin books are all classified as spot
		for i := range standardMarginAssetTypes {
			if enabled, _ := e.IsPairEnabled(pair, standardMarginAssetTypes[i]); !enabled {
				continue
			}
			book.Asset = standardMarginAssetTypes[i]
			if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
				return err
			}
		}
	}
	return c.applyPendingUpdates(e)
}

// TODO: When subscription config is added for all assets update limits to use sub.Levels
func (c *updateCache) extractOrderbookLimit(e *Exchange, a asset.Item) (uint64, error) {
	switch a {
	case asset.Spot:
		sub := e.Websocket.GetSubscription(spotOrderbookUpdateKey)
		if sub == nil {
			return 0, fmt.Errorf("%w for %q", subscription.ErrNotFound, spotOrderbookUpdateKey)
		}
		// There is no way to set levels when we subscribe for this specific subscription case.
		// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
		return uint64(sub.Interval.Duration().Milliseconds()), nil //nolint:gosec // No overflow risk
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		return futuresOrderbookUpdateLimit, nil
	case asset.DeliveryFutures:
		return deliveryFuturesUpdateLimit, nil
	case asset.Options:
		return optionOrderbookUpdateLimit, nil
	default:
		return 0, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

// waitForUpdate waits for the next update with the specified ID to be available in the cache that exceeds the next
// update ID, this ensures that an update can be applied to the orderbook. This is needed for illiquid pairs and the
// REST book ID's are out of sync.
func (c *updateCache) waitForUpdate(ctx context.Context, nextUpdateID int64) error {
	var updateListLastUpdateID int64
	c.mtx.Lock()
	updateListLastUpdateID = c.updates[len(c.updates)-1].update.UpdateID
	c.mtx.Unlock()

	if updateListLastUpdateID >= nextUpdateID {
		return nil // No need to wait, the update is already in the cache
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recentPendingUpdateID := <-c.ch:
			if recentPendingUpdateID >= nextUpdateID {
				return nil // Update is now available
			}
		}
	}
}

// applyPendingUpdates applies all pending updates to the orderbook
func (c *updateCache) applyPendingUpdates(e *Exchange) error {
	var updateApplied bool
	for _, data := range c.updates {
		bookLastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(data.update.Pair, data.update.Asset)
		if err != nil {
			return fmt.Errorf("applying pending updates: %w", err)
		}

		nextUpdateID := bookLastUpdateID + 1 // `baseId+1`

		// Dump all notifications which satisfy `u` < `baseId+1`
		if data.update.UpdateID < nextUpdateID {
			continue
		}

		pendingFirstUpdateID := data.firstUpdateID // `U`
		// `baseID+1`` < first notification `U` current base order book falls behind notifications
		if nextUpdateID < pendingFirstUpdateID {
			return fmt.Errorf("applying pending updates: %w", errOrderbookSnapshotOutdated)
		}

		if err := applyOrderbookUpdate(e, data.update); err != nil {
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

	var updateApplied bool
	for i := range standardMarginAssetTypes {
		if enabled, _ := g.IsPairEnabled(update.Pair, standardMarginAssetTypes[i]); !enabled {
			continue
		}
		update.Asset = standardMarginAssetTypes[i]
		if err := g.Websocket.Orderbook.Update(update); err != nil {
			return err
		}
		updateApplied = true
	}

	if updateApplied {
		return nil
	}

	return fmt.Errorf("apply orderbook update: %q %q %w", update.Pair, update.Asset, currency.ErrPairNotEnabled)
}

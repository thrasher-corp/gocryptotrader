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
	errOrderbookSnapshotOutdated      = errors.New("orderbook snapshot is outdated")
	errPendingUpdatesNotApplied       = errors.New("pending updates not applied")
	errInvalidOrderbookUpdateInterval = errors.New("invalid orderbook update interval")

	defaultWSOrderbookUpdateDeadline  = time.Minute * 2
	defaultWsOrderbookUpdateTimeDelay = time.Second * 2
	spotOrderbookUpdateKey            = subscription.MustChannelKey(subscription.OrderbookChannel)
	errUnhandledCacheState            = errors.New("unhandled cache state")
)

type wsOBUpdateManager struct {
	lookup   map[key.PairAsset]*updateCache
	deadline time.Duration
	delay    time.Duration
	m        sync.RWMutex
}

type updateCache struct {
	updates []pendingUpdate
	ch      chan int64
	m       sync.Mutex
	state   cacheState
}

type cacheState uint32

const (
	cacheStateUninitialised cacheState = iota
	cacheStateInitialised
	cacheStateQueuing
	cacheStateSynced
)

type pendingUpdate struct {
	update        *orderbook.Update
	firstUpdateID int64
}

func newWsOBUpdateManager(delay, deadline time.Duration) *wsOBUpdateManager {
	return &wsOBUpdateManager{lookup: make(map[key.PairAsset]*updateCache), deadline: deadline, delay: delay}
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update) error {
	cache, err := m.LoadCache(update.Pair, update.Asset)
	if err != nil {
		return err
	}

	cache.m.Lock()
	defer cache.m.Unlock()
	switch cache.state {
	case cacheStateSynced:
		return m.applyUpdate(ctx, e, cache, firstUpdateID, update)
	case cacheStateInitialised:
		m.initialiseOrderbookCache(ctx, e, firstUpdateID, update, cache)
	case cacheStateQueuing:
		cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
		select {
		case cache.ch <- update.UpdateID: // Notify SyncOrderbook of most recent update ID for inspection
		default:
		}
	default:
		return fmt.Errorf("%w: %d for %v %v", errUnhandledCacheState, cache.state, update.Pair, update.Asset)
	}
	return nil
}

// applyUpdate verifies and applies an orderbook update
// Invalidates the cache on error
// Does not benefit from concurrent lock protection
func (m *wsOBUpdateManager) applyUpdate(ctx context.Context, e *Exchange, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	lastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", e.Name, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, e, firstUpdateID, update, cache)
	}
	if lastUpdateID+1 != firstUpdateID {
		if e.Verbose { // disconnection will pollute logs
			log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: desync detected", e.Name, update.Pair, update.Asset)
		}
		return m.invalidateCache(ctx, e, firstUpdateID, update, cache)
	}
	if err := e.Websocket.Orderbook.Update(update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", e.Name, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, e, firstUpdateID, update, cache)
	}
	return nil
}

// invalidateCache invalidates the existing orderbook, clears the update queue and reinitialises the orderbook cache
// assumes lock already active on cache
func (m *wsOBUpdateManager) invalidateCache(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update, cache *updateCache) error {
	if err := e.Websocket.Orderbook.InvalidateOrderbook(update.Pair, update.Asset); err != nil {
		return err
	}
	m.initialiseOrderbookCache(ctx, e, firstUpdateID, update, cache)
	return nil
}

// initialiseOrderbookCache sets the cache state to queuing, appends the update to the cache and spawns a goroutine
// to fetch and synchronise the orderbook snapshot
// assumes lock already active on cache
func (m *wsOBUpdateManager) initialiseOrderbookCache(ctx context.Context, e *Exchange, firstUpdateID int64, update *orderbook.Update, cache *updateCache) {
	cache.state = cacheStateQueuing
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
	go func() {
		if err := cache.SyncOrderbook(ctx, e, update.Pair, update.Asset, m.delay, m.deadline); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", e.Name, update.Pair, update.Asset, err)
		}
	}()
}

// LoadCache loads the cache for the given pair and asset. If the cache does not exist, it creates a new one.
func (m *wsOBUpdateManager) LoadCache(p currency.Pair, a asset.Item) (*updateCache, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w: %q", asset.ErrInvalidAsset, a)
	}
	m.m.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.m.RUnlock()
	if !ok {
		cache = &updateCache{ch: make(chan int64), state: cacheStateInitialised}
		m.m.Lock()
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.m.Unlock()
	}
	return cache, nil
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (c *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, pair currency.Pair, a asset.Item, delay, deadline time.Duration) error {
	limit, err := e.extractOrderbookLimit(a)
	if err != nil {
		c.clearWithLock()
		return err
	}

	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		c.clearWithLock()
		return ctx.Err()
	case <-time.After(delay):
	}

	// Setting deadline to error out instead of waiting for rate limiter delay which excessively builds a backlog of
	// pending updates.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(deadline))
	defer cancel()

	book, err := e.fetchOrderbook(ctx, pair, a, limit)
	if err != nil {
		c.clearWithLock()
		return err
	}

	if err := c.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
		c.clearWithLock()
		return err
	}

	c.m.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		c.clearNoLock()
		c.m.Unlock()
	}()

	if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
		return err
	}

	return c.applyPendingUpdates(e)
}

// waitForUpdate waits for an update with an ID >= nextUpdateID
func (c *updateCache) waitForUpdate(ctx context.Context, nextUpdateID int64) error {
	c.m.Lock()
	updateListLastUpdateID := c.updates[len(c.updates)-1].update.UpdateID
	c.m.Unlock()
	if updateListLastUpdateID >= nextUpdateID {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recentPendingUpdateID := <-c.ch:
			if recentPendingUpdateID >= nextUpdateID {
				return nil
			}
		}
	}
}

// applyPendingUpdates applies all pending updates to the orderbook
// assumes lock already active on cache
func (c *updateCache) applyPendingUpdates(e *Exchange) error {
	var updated bool
	for _, data := range c.updates {
		bookLastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(data.update.Pair, data.update.Asset)
		if err != nil {
			return err
		}

		nextUpdateID := bookLastUpdateID + 1 // From docs: `baseId+1`

		// From docs: Dump all notifications which satisfy `u` < `baseId+1`
		if data.update.UpdateID < nextUpdateID {
			continue
		}

		pendingFirstUpdateID := data.firstUpdateID // `U`
		// From docs: `baseID+1` < first notification `U` current base order book falls behind notifications
		if nextUpdateID < pendingFirstUpdateID {
			return errOrderbookSnapshotOutdated
		}

		if err := e.Websocket.Orderbook.Update(data.update); err != nil {
			return err
		}

		updated = true
	}

	if !updated {
		return errPendingUpdatesNotApplied
	}
	c.state = cacheStateSynced
	return nil
}

func (c *updateCache) clearWithLock() {
	c.m.Lock()
	defer c.m.Unlock()
	c.clearNoLock()
}

func (c *updateCache) clearNoLock() {
	c.updates = nil
}

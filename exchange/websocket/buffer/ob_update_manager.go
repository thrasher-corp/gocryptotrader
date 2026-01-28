package buffer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// public errors
var (
	ErrOrderbookSnapshotOutdated = errors.New("orderbook snapshot is outdated")
)

// time settings
var (
	DefaultWSOrderbookUpdateTimeDelay = time.Second * 2
	DefaultWSOrderbookUpdateDeadline  = time.Minute * 2
)

var (
	errPendingUpdatesNotApplied = errors.New("pending updates not applied")
	errUnhandledCacheState      = errors.New("unhandled cache state")
)

// UpdateManager manages orderbook updates for websocket connections
// TODO: Directly couple with orderbook struct and optimise locking paths.
type UpdateManager struct {
	lookup             map[key.PairAsset]*updateCache
	deadline           time.Duration
	delay              time.Duration
	fetchOrderbook     func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	checkPendingUpdate func(lastUpdateID int64, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	ob                 *Orderbook
	m                  sync.RWMutex
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

// UpdateParams contains parameters used to create a new UpdateManager
type UpdateParams struct {
	// FetchDelay defines the delay before the REST orderbook is retrieved. In some cases REST requests can be behind
	// websocket updates by a large margin, this allows the cache to fill with updates before we fetch the orderbook so
	// they can be correctly applied.
	FetchDelay time.Duration
	// FetchDeadline defines the maximum time to wait for the REST orderbook to be retrieved. This prevents excessive
	// backlogs of pending updates building up while waiting for rate limiter delays.
	FetchDeadline  time.Duration
	FetchOrderbook func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	// CheckPendingUpdate allows custom logic to determine if a pending update added to cache should be skipped or if an
	// error has occurred.
	CheckPendingUpdate func(lastUpdateID int64, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	BufferInstance     *Orderbook // TODO: Integrate directly with orderbook struct
}

// NewUpdateManager creates a new websocket orderbook update manager
func NewUpdateManager(params *UpdateParams) *UpdateManager {
	if params.FetchDeadline <= 0 {
		panic("fetch deadline must be greater than zero")
	}
	if params.FetchDelay < 0 {
		panic("fetch delay must be greater than or equal to zero")
	}
	if err := common.NilGuard(params.FetchOrderbook, params.CheckPendingUpdate, params.BufferInstance); err != nil {
		panic(err)
	}
	return &UpdateManager{
		lookup:             make(map[key.PairAsset]*updateCache),
		deadline:           params.FetchDeadline,
		delay:              params.FetchDelay,
		fetchOrderbook:     params.FetchOrderbook,
		checkPendingUpdate: params.CheckPendingUpdate,
		ob:                 params.BufferInstance,
	}
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *UpdateManager) ProcessOrderbookUpdate(ctx context.Context, firstUpdateID int64, update *orderbook.Update) error {
	cache, err := m.loadCache(update.Pair, update.Asset)
	if err != nil {
		return err
	}

	cache.m.Lock()
	defer cache.m.Unlock()
	switch cache.state {
	case cacheStateSynced:
		return m.applyUpdate(ctx, cache, firstUpdateID, update)
	case cacheStateInitialised:
		m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache)
	case cacheStateQueuing:
		cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
		select {
		case cache.ch <- update.UpdateID: // Notify syncOrderbook of most recent update ID for inspection
		default:
		}
	default:
		return fmt.Errorf("%w: %d for %v %v", errUnhandledCacheState, cache.state, update.Pair, update.Asset)
	}
	return nil
}

// loadCache loads the cache for the given pair and asset. If the cache does not exist, it creates a new one.
func (m *UpdateManager) loadCache(p currency.Pair, a asset.Item) (*updateCache, error) {
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

// applyUpdate verifies and applies an orderbook update
// Invalidates the cache on error
// Does not benefit from concurrent lock protection
func (m *UpdateManager) applyUpdate(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	lastUpdateID, err := m.ob.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache)
	}
	if isOutOfSequence(lastUpdateID, firstUpdateID) {
		if m.ob.verbose { // disconnection will pollute logs
			log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: desync detected", m.ob.exchangeName, update.Pair, update.Asset)
		}
		return m.invalidateCache(ctx, firstUpdateID, update, cache)
	}
	if err := m.ob.Update(update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache)
	}
	return nil
}

// initialiseOrderbookCache sets the cache state to queuing, appends the update to the cache and spawns a goroutine
// to fetch and synchronise the orderbook snapshot
// assumes lock already active on cache
func (m *UpdateManager) initialiseOrderbookCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache) {
	cache.state = cacheStateQueuing
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
	go func() {
		if err := m.syncOrderbook(ctx, cache, update.Pair, update.Asset); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		}
	}()
}

// invalidateCache invalidates the existing orderbook, clears the update queue and reinitialises the orderbook cache
// assumes lock already active on cache
func (m *UpdateManager) invalidateCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache) error {
	err := m.ob.InvalidateOrderbook(update.Pair, update.Asset)
	m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache)
	return err
}

// syncOrderbook fetches and synchronises an orderbook snapshot so that pending updates can be applied to the orderbook.
func (m *UpdateManager) syncOrderbook(ctx context.Context, cache *updateCache, pair currency.Pair, a asset.Item) error {
	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		cache.clearPreserveStateWithLock()
		return ctx.Err()
	case <-time.After(m.delay):
	}

	// Setting deadline to error out instead of waiting for rate limiter delay which excessively builds a backlog of
	// pending updates.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(m.deadline))
	defer cancel()

	book, err := m.fetchOrderbook(ctx, pair, a)
	if err != nil {
		cache.clearWithLock()
		return err
	}

	if err := cache.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
		cache.clearWithLock()
		return err
	}

	if err := m.ob.LoadSnapshot(book); err != nil {
		cache.clearWithLock()
		return err
	}

	cache.m.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		cache.clearNoLock()
		cache.m.Unlock()
	}()

	if err := m.applyPendingUpdates(cache); err != nil {
		cache.resetStateNoLock()
		return common.AppendError(err, m.ob.InvalidateOrderbook(pair, a))
	}

	return nil
}

// applyPendingUpdates applies all pending updates to the orderbook
// assumes lock already active on cache
func (m *UpdateManager) applyPendingUpdates(cache *updateCache) error {
	var updated bool
	for _, data := range cache.updates {
		bookLastUpdateID, err := m.ob.LastUpdateID(data.update.Pair, data.update.Asset)
		if err != nil {
			return err
		}

		if !updated {
			skip, err := m.checkPendingUpdate(bookLastUpdateID, data.firstUpdateID, data.update)
			if err != nil {
				return err
			}
			if skip {
				continue
			}
		} else if isOutOfSequence(bookLastUpdateID, data.firstUpdateID) {
			return fmt.Errorf("apply pending updates %w: last update ID %d, first update ID %d", ErrOrderbookSnapshotOutdated, bookLastUpdateID, data.firstUpdateID)
		}

		if err := m.ob.Update(data.update); err != nil {
			return err
		}

		updated = true
	}

	if !updated {
		return errPendingUpdatesNotApplied
	}
	cache.state = cacheStateSynced
	return nil
}

// isOutOfSequence checks if the update is out of sequence
func isOutOfSequence(lastUpdateID int64, firstUpdateID int64) bool {
	return lastUpdateID+1 != firstUpdateID
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

func (c *updateCache) clearWithLock() {
	c.m.Lock()
	defer c.m.Unlock()
	c.resetStateNoLock()
	c.clearNoLock()
}

func (c *updateCache) clearPreserveStateWithLock() {
	c.m.Lock()
	defer c.m.Unlock()
	c.clearNoLock()
}

func (c *updateCache) clearNoLock() {
	c.updates = nil
}

func (c *updateCache) resetStateNoLock() {
	c.state = cacheStateInitialised
}

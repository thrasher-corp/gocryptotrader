package orderbookmanager

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
const (
	DefaultWSOrderbookUpdateTimeDelay = time.Second * 2
	DefaultWSOrderbookUpdateDeadline  = time.Minute * 2
)

var (
	errPendingUpdateKeyMismatch = errors.New("pending update key mismatch")
	errPendingUpdatesNotApplied = errors.New("pending updates not applied")
	errUpdatesNotSupplied       = errors.New("updates not supplied")
	errUnhandledCacheState      = errors.New("unhandled cache state")
)

// UpdateManager manages order book REST snapshots to queued websocket order book updates
// TODO: Directly couple with orderbook struct and optimise locking paths.
type UpdateManager struct {
	lookup   map[key.PairAsset]*updateCache
	lookupMu sync.RWMutex

	deadline           time.Duration
	delay              time.Duration
	fetchOrderbook     func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	checkPendingUpdate func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	ob                 *Orderbook
	// loadSnapshotsAtOnce loads a snapshot no queued update follows yet instead of waiting for one
	loadSnapshotsAtOnce bool
}

type updateCache struct {
	updates []pendingUpdate
	ch      chan int64
	m       sync.Mutex
	state   cacheState
	// withdrawnUpdateID is the last update ID of a published book that was invalidated, kept until the book is
	// synchronised again. When snapshots load at once, an update the book already holds can invalidate it and REST
	// snapshots can lag the stream, so a snapshot no queued update reaches past is not loaded when it is older. One a
	// queued update reaches past is, as that update shows the stream is at its level, which is also how a stream whose
	// update IDs restart synchronises again
	withdrawnUpdateID int64
}

type cacheState uint32

const (
	cacheStateUninitialised cacheState = iota
	cacheStateInitialised
	cacheStateQueuing
	cacheStateSynced
	// cacheStateSnapshotLoaded holds a book loaded at once from a snapshot that no queued update followed. A snapshot's
	// update ID need not end an update, so the next update is checked the way queued updates are before the book is in
	// sequence
	cacheStateSnapshotLoaded
)

type pendingUpdate struct {
	update        *orderbook.Update
	firstUpdateID int64
	ctx           common.FrozenContext
}

// UpdateManagerParams contains parameters used to create a new UpdateManager
type UpdateManagerParams struct {
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
	CheckPendingUpdate func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	// Orderbook stores the snapshots and updates managed by UpdateManager.
	Orderbook *Orderbook
	// LoadSnapshotsAtOnce loads a fetched snapshot that no queued update follows yet, and checks the first update to
	// arrive against it, as Binance documents its local order book procedure; otherwise a snapshot waits for a queued
	// update past it before loading, so a book that changes rarely is withheld until its next change. A snapshot older
	// than a book already published is still only loaded once a queued update is past it
	LoadSnapshotsAtOnce bool
}

// NewUpdateManager creates a new websocket orderbook update manager
func NewUpdateManager(params *UpdateManagerParams) *UpdateManager {
	if params.FetchDeadline <= 0 {
		panic("fetch deadline must be greater than zero")
	}
	if params.FetchDelay < 0 {
		panic("fetch delay must be greater than or equal to zero")
	}
	if err := common.NilGuard(params.FetchOrderbook, params.CheckPendingUpdate, params.Orderbook); err != nil {
		panic(err)
	}
	return &UpdateManager{
		lookup:              make(map[key.PairAsset]*updateCache),
		deadline:            params.FetchDeadline,
		delay:               params.FetchDelay,
		fetchOrderbook:      params.FetchOrderbook,
		checkPendingUpdate:  params.CheckPendingUpdate,
		ob:                  params.Orderbook,
		loadSnapshotsAtOnce: params.LoadSnapshotsAtOnce,
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
	case cacheStateSnapshotLoaded:
		return m.applyFirstUpdate(ctx, cache, firstUpdateID, update)
	case cacheStateInitialised:
		m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache)
	case cacheStateQueuing:
		cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID, ctx: common.FreezeContext(ctx)})
		select {
		case cache.ch <- update.UpdateID: // Wake syncOrderbook to recheck the queue
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
	m.lookupMu.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.lookupMu.RUnlock()
	if !ok {
		cache = &updateCache{ch: make(chan int64, 1), state: cacheStateInitialised}
		m.lookupMu.Lock()
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.lookupMu.Unlock()
	}
	return cache, nil
}

// applyUpdate verifies and applies an orderbook update
// Invalidates the cache on error
// assumes lock already active on cache
func (m *UpdateManager) applyUpdate(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	lastUpdateID, err := m.ob.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache, 0)
	}
	if lastUpdateID+1 != firstUpdateID {
		if m.ob.verbose { // disconnection will pollute logs
			log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: desync detected", m.ob.exchangeName, update.Pair, update.Asset)
		}
		return m.invalidateCache(ctx, firstUpdateID, update, cache, lastUpdateID)
	}
	if err := m.ob.Update(ctx, update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache, lastUpdateID)
	}
	return nil
}

// applyFirstUpdate applies the first update after a snapshot no queued update followed, checked as a queued update is:
// an update the snapshot already holds is skipped, and one that does not follow it invalidates the book and
// synchronises it again
// assumes lock already active on cache
func (m *UpdateManager) applyFirstUpdate(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	lastUpdateID, err := m.ob.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache, 0)
	}
	skip, err := m.checkPendingUpdate(lastUpdateID, firstUpdateID, update)
	if err != nil {
		if m.ob.verbose { // disconnection will pollute logs
			log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		}
		return m.invalidateCache(ctx, firstUpdateID, update, cache, lastUpdateID)
	}
	if skip {
		return nil
	}
	if err := m.ob.Update(ctx, update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache, lastUpdateID)
	}
	cache.state = cacheStateSynced
	return nil
}

// initialiseOrderbookCache sets the cache state to queuing, appends the update to the cache and spawns a goroutine
// to fetch and synchronise the orderbook snapshot
// assumes lock already active on cache
func (m *UpdateManager) initialiseOrderbookCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache) {
	cache.state = cacheStateQueuing
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID, ctx: common.FreezeContext(ctx)})
	go func() {
		if err := m.syncOrderbook(ctx, cache, update.Pair, update.Asset); err != nil {
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		}
	}()
}

// invalidateCache invalidates the existing orderbook, keeping the update ID it was published at, clears the update queue
// and reinitialises the orderbook cache. The update ID is the one read before the update that failed, which may have
// invalidated the book and its update ID with it, or zero when no book was published; or the book's own, when the
// update was published and then failed to reach the data handler
// assumes lock already active on cache
func (m *UpdateManager) invalidateCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache, publishedUpdateID int64) error {
	if lastUpdateID, err := m.ob.LastUpdateID(update.Pair, update.Asset); err == nil {
		publishedUpdateID = max(publishedUpdateID, lastUpdateID)
	}
	cache.withdrawnUpdateID = max(cache.withdrawnUpdateID, publishedUpdateID)
	err := m.ob.InvalidateOrderbook(update.Pair, update.Asset)
	m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache)
	return err
}

// syncOrderbook fetches an orderbook snapshot, loads it and applies the pending updates that follow it. Unless snapshots
// load at once, the snapshot waits for a queued update past it before loading
func (m *UpdateManager) syncOrderbook(ctx context.Context, cache *updateCache, pair currency.Pair, a asset.Item) error {
	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		cache.clearPendingUpdatesWithLock()
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

	if m.loadSnapshotsAtOnce {
		if err := m.checkSnapshot(cache, pair, a, book); err != nil {
			return err
		}
	} else if err := cache.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
		cache.clearWithLock()
		return err
	}

	if err := m.ob.LoadSnapshot(ctx, book); err != nil {
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

// applyPendingUpdates applies the pending updates that follow the loaded snapshot. When none does and snapshots load at
// once, the next update to arrive is checked against the snapshot
// assumes lock already active on cache
func (m *UpdateManager) applyPendingUpdates(cache *updateCache) error {
	if len(cache.updates) == 0 {
		return errUpdatesNotSupplied
	}

	firstUpdate := cache.updates[0].update
	if firstUpdate.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if !firstUpdate.Asset.IsValid() {
		return asset.ErrInvalidAsset
	}

	pendingKey := key.PairAsset{Base: firstUpdate.Pair.Base.Item, Quote: firstUpdate.Pair.Quote.Item, Asset: firstUpdate.Asset}
	for _, data := range cache.updates[1:] {
		currentKey := key.PairAsset{Base: data.update.Pair.Base.Item, Quote: data.update.Pair.Quote.Item, Asset: data.update.Asset}
		if currentKey != pendingKey {
			return fmt.Errorf("%w: expected %s.%s, got %s.%s", errPendingUpdateKeyMismatch, pendingKey.Asset, pendingKey.Pair(), currentKey.Asset, currentKey.Pair())
		}
	}

	m.ob.m.RLock()
	depth, ok := m.ob.ob[pendingKey]
	m.ob.m.RUnlock()
	if !ok {
		return fmt.Errorf("%s %w: %s.%s", m.ob.exchangeName, orderbook.ErrDepthNotFound, firstUpdate.Asset, firstUpdate.Pair)
	}

	// The book is published at each update ID it passes through, which a failure keeps for the synchronisation after:
	// an update that fails to apply can invalidate the book, and its update ID with it, while one published before it
	// failed to reach the data handler leaves the book at that update
	var published int64
	withdraw := func(err error) error {
		if lastUpdateID, idErr := depth.LastUpdateID(); idErr == nil {
			published = max(published, lastUpdateID)
		}
		cache.withdrawnUpdateID = max(cache.withdrawnUpdateID, published)
		return err
	}
	var updated bool
	for _, data := range cache.updates {
		bookLastUpdateID, err := depth.LastUpdateID()
		if err != nil {
			return withdraw(err)
		}
		published = bookLastUpdateID

		if !updated {
			skip, err := m.checkPendingUpdate(bookLastUpdateID, data.firstUpdateID, data.update)
			if err != nil {
				return withdraw(err)
			}
			if skip {
				continue
			}
		} else if bookLastUpdateID+1 != data.firstUpdateID {
			return withdraw(fmt.Errorf("apply pending updates %w: last update ID %d, first update ID %d", ErrOrderbookSnapshotOutdated, bookLastUpdateID, data.firstUpdateID))
		}

		if err := m.ob.updateDepth(common.ThawContext(data.ctx), depth, data.update); err != nil {
			return withdraw(err)
		}

		updated = true
	}

	if !updated && !m.loadSnapshotsAtOnce {
		return errPendingUpdatesNotApplied
	}
	cache.state, cache.withdrawnUpdateID = cacheStateSynced, 0
	if !updated {
		cache.state = cacheStateSnapshotLoaded
	}
	return nil
}

// checkSnapshot rejects a snapshot older than the book last published when no queued update reaches past it, and resets
// the cache, in the same lock, for the next update to synchronise again. Nothing then shows the stream is at the
// snapshot's level, so it is not loaded, which would show the book rolled back; a queued update past it is checked as
// it is applied. The book last published is the one the cache withdrew, or one still published, such as a REST
// snapshot or a snapshot loaded before its delivery to the data handler failed
func (m *UpdateManager) checkSnapshot(cache *updateCache, pair currency.Pair, a asset.Item, book *orderbook.Book) error {
	cache.m.Lock()
	defer cache.m.Unlock()
	published := cache.withdrawnUpdateID
	if lastUpdateID, err := m.ob.publishedUpdateID(pair, a); err == nil {
		published = max(published, lastUpdateID)
	}
	if book.LastUpdateID >= published {
		return nil
	}
	for i := range cache.updates {
		if cache.updates[i].update.UpdateID > book.LastUpdateID {
			return nil
		}
	}
	cache.resetStateNoLock()
	cache.clearNoLock()
	return fmt.Errorf("%w: snapshot update ID %d precedes published update ID %d, and no queued update follows it", ErrOrderbookSnapshotOutdated, book.LastUpdateID, published)
}

// waitForUpdate waits for an update with an ID >= nextUpdateID
func (c *updateCache) waitForUpdate(ctx context.Context, nextUpdateID int64) error {
	for {
		c.m.Lock()
		updateListLastUpdateID := c.updates[len(c.updates)-1].update.UpdateID
		c.m.Unlock()
		if updateListLastUpdateID >= nextUpdateID {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ch:
		}
	}
}

func (c *updateCache) clearWithLock() {
	c.m.Lock()
	defer c.m.Unlock()
	c.resetStateNoLock()
	c.clearNoLock()
}

func (c *updateCache) clearPendingUpdatesWithLock() {
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

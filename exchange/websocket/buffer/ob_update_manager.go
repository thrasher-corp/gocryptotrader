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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/metrics"
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
	DefaultWSOrderbookUpdateTimeDelay           = time.Second * 2
	DefaultWSOrderbookUpdateDeadline            = time.Minute * 2
	DefaultWSOrderbookOutdatedSnapshotRetryWait = time.Millisecond * 250
)

var (
	errPendingUpdatesNotApplied = errors.New("pending updates not applied")
	errUnhandledCacheState      = errors.New("unhandled cache state")
)

// UpdateManager manages order book REST snapshots to queued websocket order book updates
// TODO: Directly couple with orderbook struct and optimise locking paths.
type UpdateManager struct {
	lookup   map[key.PairAsset]*updateCache
	lookupMu sync.RWMutex

	deadline           time.Duration
	delay              time.Duration
	retryDelay         time.Duration
	syncLimit          chan struct{}
	fetchOrderbook     func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	checkPendingUpdate func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	checkLiveUpdates   bool
	panicOnDesync      bool
	recordMetrics      bool
	ob                 *Orderbook
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
	// OutdatedSnapshotRetryDelay defines how long to wait before fetching another REST snapshot when the first snapshot
	// is too old for the queued websocket updates.
	OutdatedSnapshotRetryDelay time.Duration
	// SnapshotSyncLimit caps concurrent REST snapshot synchronisation attempts. Zero leaves synchronisation unbounded.
	SnapshotSyncLimit int
	// CheckPendingUpdate allows custom logic to determine if a pending update added to cache should be skipped or if an
	// error has occurred.
	CheckPendingUpdate func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	// CheckLiveUpdates reuses CheckPendingUpdate for already-synchronised books, allowing exchanges with overlapping
	// update ranges to trim stale levels instead of forcing a REST resync.
	CheckLiveUpdates bool
	// PanicOnDesync intentionally crashes on live sequence gaps so short-lived investigations can capture the exact
	// exchange, pair, asset and sequence IDs causing an orderbook resync.
	PanicOnDesync bool
	// RecordMetrics emits expvar counters for orderbook desyncs and resync outcomes.
	RecordMetrics  bool
	BufferInstance *Orderbook // TODO: Integrate directly with orderbook struct
}

// NewUpdateManager creates a new websocket orderbook update manager
func NewUpdateManager(params *UpdateManagerParams) *UpdateManager {
	if params.FetchDeadline <= 0 {
		panic("fetch deadline must be greater than zero")
	}
	if params.FetchDelay < 0 {
		panic("fetch delay must be greater than or equal to zero")
	}
	if params.OutdatedSnapshotRetryDelay < 0 {
		panic("outdated snapshot retry delay must be greater than or equal to zero")
	}
	if params.SnapshotSyncLimit < 0 {
		panic("snapshot sync limit must be greater than or equal to zero")
	}
	if params.OutdatedSnapshotRetryDelay == 0 {
		params.OutdatedSnapshotRetryDelay = DefaultWSOrderbookOutdatedSnapshotRetryWait
	}
	if err := common.NilGuard(params.FetchOrderbook, params.CheckPendingUpdate, params.BufferInstance); err != nil {
		panic(err)
	}
	manager := &UpdateManager{
		lookup:             make(map[key.PairAsset]*updateCache),
		deadline:           params.FetchDeadline,
		delay:              params.FetchDelay,
		retryDelay:         params.OutdatedSnapshotRetryDelay,
		fetchOrderbook:     params.FetchOrderbook,
		checkPendingUpdate: params.CheckPendingUpdate,
		checkLiveUpdates:   params.CheckLiveUpdates,
		panicOnDesync:      params.PanicOnDesync,
		recordMetrics:      params.RecordMetrics,
		ob:                 params.BufferInstance,
	}
	if params.SnapshotSyncLimit > 0 {
		manager.syncLimit = make(chan struct{}, params.SnapshotSyncLimit)
	}
	return manager
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
		m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache, false)
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
	m.lookupMu.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.lookupMu.RUnlock()
	if !ok {
		cache = &updateCache{ch: make(chan int64), state: cacheStateInitialised}
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
		return m.invalidateCache(ctx, firstUpdateID, update, cache, "last_update_id")
	}
	if m.checkLiveUpdates {
		skip, err := m.checkPendingUpdate(lastUpdateID, firstUpdateID, update)
		if err != nil {
			return m.handleDesync(ctx, cache, firstUpdateID, update, lastUpdateID, err)
		}
		if skip {
			return nil
		}
	} else if lastUpdateID+1 != firstUpdateID {
		return m.handleDesync(ctx, cache, firstUpdateID, update, lastUpdateID, nil)
	}
	if err := m.ob.Update(update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache, "update_failed")
	}
	return nil
}

func (m *UpdateManager) handleDesync(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update, lastUpdateID int64, reason error) error {
	metricReason := orderbookDesyncReason(reason)
	if m.recordMetrics {
		metrics.RecordOrderbookDesync(&metrics.OrderbookSyncEvent{
			Exchange:      m.ob.exchangeName,
			Pair:          update.Pair,
			Asset:         update.Asset,
			Channel:       "orderbook",
			Reason:        metricReason,
			LastUpdateID:  lastUpdateID,
			FirstUpdateID: firstUpdateID,
			UpdateID:      update.UpdateID,
		})
	}
	if m.panicOnDesync {
		if reason != nil {
			panic(fmt.Sprintf("%s websocket orderbook manager desync for %v %v: last update ID %d, first update ID %d, update ID %d: %v", m.ob.exchangeName, update.Pair, update.Asset, lastUpdateID, firstUpdateID, update.UpdateID, reason))
		}
		panic(fmt.Sprintf("%s websocket orderbook manager desync for %v %v: last update ID %d, first update ID %d, update ID %d", m.ob.exchangeName, update.Pair, update.Asset, lastUpdateID, firstUpdateID, update.UpdateID))
	}
	if m.ob.verbose { // disconnection will pollute logs
		log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: desync detected", m.ob.exchangeName, update.Pair, update.Asset)
	}
	return m.invalidateCache(ctx, firstUpdateID, update, cache, metricReason)
}

func orderbookDesyncReason(reason error) string {
	if errors.Is(reason, ErrOrderbookSnapshotOutdated) {
		return "snapshot_outdated"
	}
	if reason != nil {
		return "desync"
	}
	return "sequence_gap"
}

// initialiseOrderbookCache sets the cache state to queuing, appends the update to the cache and spawns a goroutine
// to fetch and synchronise the orderbook snapshot
// assumes lock already active on cache
func (m *UpdateManager) initialiseOrderbookCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache, resync bool) {
	cache.state = cacheStateQueuing
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
	go func() {
		if err := m.syncOrderbook(ctx, cache, update.Pair, update.Asset, resync); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		}
	}()
}

// invalidateCache invalidates the existing orderbook, clears the update queue and reinitialises the orderbook cache
// assumes lock already active on cache
func (m *UpdateManager) invalidateCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache, reason string) error {
	err := m.ob.InvalidateOrderbook(update.Pair, update.Asset)
	if m.recordMetrics {
		metrics.RecordOrderbookResync(&metrics.OrderbookSyncEvent{
			Exchange: m.ob.exchangeName,
			Pair:     update.Pair,
			Asset:    update.Asset,
			Channel:  "orderbook",
			Reason:   reason,
			Result:   "started",
		})
	}
	m.initialiseOrderbookCache(ctx, firstUpdateID, update, cache, true)
	return err
}

// syncOrderbook fetches and synchronises an orderbook snapshot so that pending updates can be applied to the orderbook.
func (m *UpdateManager) syncOrderbook(ctx context.Context, cache *updateCache, pair currency.Pair, a asset.Item, resync bool) error {
	var resyncStarted time.Time
	if m.recordMetrics && resync {
		resyncStarted = time.Now()
	}
	if m.syncLimit != nil {
		select {
		case <-ctx.Done():
			cache.clearPendingUpdatesWithLock()
			m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
			return ctx.Err()
		case m.syncLimit <- struct{}{}:
			defer func() { <-m.syncLimit }()
		}
	}

	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		cache.clearPendingUpdatesWithLock()
		m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
		return ctx.Err()
	case <-time.After(m.delay):
	}

	// Setting deadline to error out instead of waiting for rate limiter delay which excessively builds a backlog of
	// pending updates.
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(m.deadline))
	defer cancel()

	var book *orderbook.Book
	for {
		var err error
		book, err = m.fetchOrderbook(ctx, pair, a)
		if err != nil {
			cache.clearWithLock()
			m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
			return err
		}

		if err := cache.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
			cache.clearWithLock()
			m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
			return err
		}

		if err := m.checkSnapshotCanApply(cache, book.LastUpdateID); err != nil {
			if !errors.Is(err, ErrOrderbookSnapshotOutdated) {
				cache.clearWithLock()
				m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
				return err
			}
			select {
			case <-ctx.Done():
				cache.clearWithLock()
				m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
				return ctx.Err()
			case <-time.After(m.retryDelay):
				continue
			}
		}
		break
	}

	if err := m.ob.LoadSnapshot(book); err != nil {
		cache.clearWithLock()
		m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
		return err
	}

	cache.m.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		cache.clearNoLock()
		cache.m.Unlock()
	}()

	if err := m.applyPendingUpdates(cache); err != nil {
		cache.resetStateNoLock()
		m.recordResyncResult(resync, pair, a, "failed", resyncStarted)
		return common.AppendError(err, m.ob.InvalidateOrderbook(pair, a))
	}

	m.recordResyncResult(resync, pair, a, "succeeded", resyncStarted)
	return nil
}

func (m *UpdateManager) recordResyncResult(
	resync bool,
	pair currency.Pair,
	a asset.Item,
	result string,
	started time.Time,
) {
	if !m.recordMetrics || !resync {
		return
	}
	var duration time.Duration
	if !started.IsZero() {
		duration = time.Since(started)
	}
	metrics.RecordOrderbookResync(&metrics.OrderbookSyncEvent{
		Exchange: m.ob.exchangeName,
		Pair:     pair,
		Asset:    a,
		Channel:  "orderbook",
		Result:   result,
		Duration: duration,
	})
}

func (m *UpdateManager) checkSnapshotCanApply(cache *updateCache, lastUpdateID int64) error {
	cache.m.Lock()
	defer cache.m.Unlock()
	for _, data := range cache.updates {
		update := cloneOrderbookUpdate(data.update)
		skip, err := m.checkPendingUpdate(lastUpdateID, data.firstUpdateID, update)
		if err != nil {
			return err
		}
		if !skip {
			return nil
		}
	}
	return errPendingUpdatesNotApplied
}

func cloneOrderbookUpdate(update *orderbook.Update) *orderbook.Update {
	if update == nil {
		return nil
	}
	clone := *update
	clone.Bids = append(orderbook.Levels(nil), update.Bids...)
	clone.Asks = append(orderbook.Levels(nil), update.Asks...)
	return &clone
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
		} else if bookLastUpdateID+1 != data.firstUpdateID {
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

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

	deadline                         time.Duration
	delay                            time.Duration
	retryDelay                       time.Duration
	syncLimit                        chan struct{}
	fetchOrderbook                   func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	checkPendingUpdate               func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error)
	checkLiveUpdates                 bool
	acceptSnapshotWhenUpdatesCovered bool
	initialSnapshotFallbackDelay     time.Duration
	initialSnapshotFallbackRetry     time.Duration
	initialSnapshotFallbackLimit     chan struct{}
	ob                               *Orderbook
}

type updateCache struct {
	updates           []pendingUpdate
	ch                chan int64
	m                 sync.Mutex
	state             cacheState
	fallbackScheduled bool
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
	// AcceptSnapshotWhenUpdatesCovered allows a snapshot to establish the live baseline when its sequence already
	// includes every queued websocket update, without waiting for another update from a quiet market.
	AcceptSnapshotWhenUpdatesCovered bool
	// InitialSnapshotFallbackDelay enables delayed REST bootstrap for subscribed books which have not received their
	// first websocket update. Zero disables fallback bootstrap.
	InitialSnapshotFallbackDelay time.Duration
	// InitialSnapshotFallbackRetryDelay controls how long failed fallback snapshots wait before another attempt.
	InitialSnapshotFallbackRetryDelay time.Duration
	// InitialSnapshotFallbackLimit caps concurrent fallback snapshots separately from normal snapshot synchronisation.
	InitialSnapshotFallbackLimit int
	BufferInstance               *Orderbook // TODO: Integrate directly with orderbook struct
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
	if params.InitialSnapshotFallbackDelay < 0 {
		panic("initial snapshot fallback delay must be greater than or equal to zero")
	}
	if params.InitialSnapshotFallbackRetryDelay < 0 {
		panic("initial snapshot fallback retry delay must be greater than or equal to zero")
	}
	if params.InitialSnapshotFallbackLimit < 0 {
		panic("initial snapshot fallback limit must be greater than or equal to zero")
	}
	if params.InitialSnapshotFallbackDelay > 0 && params.InitialSnapshotFallbackLimit == 0 {
		panic("initial snapshot fallback limit must be greater than zero when fallback is enabled")
	}
	if params.InitialSnapshotFallbackDelay > 0 && params.InitialSnapshotFallbackRetryDelay == 0 {
		params.InitialSnapshotFallbackRetryDelay = time.Second * 30
	}
	if params.OutdatedSnapshotRetryDelay == 0 {
		params.OutdatedSnapshotRetryDelay = DefaultWSOrderbookOutdatedSnapshotRetryWait
	}
	if err := common.NilGuard(params.FetchOrderbook, params.CheckPendingUpdate, params.BufferInstance); err != nil {
		panic(err)
	}
	manager := &UpdateManager{
		lookup:                           make(map[key.PairAsset]*updateCache),
		deadline:                         params.FetchDeadline,
		delay:                            params.FetchDelay,
		retryDelay:                       params.OutdatedSnapshotRetryDelay,
		fetchOrderbook:                   params.FetchOrderbook,
		checkPendingUpdate:               params.CheckPendingUpdate,
		checkLiveUpdates:                 params.CheckLiveUpdates,
		acceptSnapshotWhenUpdatesCovered: params.AcceptSnapshotWhenUpdatesCovered,
		initialSnapshotFallbackDelay:     params.InitialSnapshotFallbackDelay,
		initialSnapshotFallbackRetry:     params.InitialSnapshotFallbackRetryDelay,
		ob:                               params.BufferInstance,
	}
	if params.SnapshotSyncLimit > 0 {
		manager.syncLimit = make(chan struct{}, params.SnapshotSyncLimit)
	}
	if params.InitialSnapshotFallbackLimit > 0 {
		manager.initialSnapshotFallbackLimit = make(chan struct{}, params.InitialSnapshotFallbackLimit)
	}
	return manager
}

// ScheduleInitialSnapshot registers a subscribed orderbook for delayed REST bootstrap if no websocket update arrives.
func (m *UpdateManager) ScheduleInitialSnapshot(ctx context.Context, p currency.Pair, a asset.Item) error {
	if m.initialSnapshotFallbackDelay == 0 {
		return nil
	}
	cache, err := m.loadCache(p, a)
	if err != nil {
		return err
	}

	cache.m.Lock()
	if cache.fallbackScheduled {
		cache.m.Unlock()
		return nil
	}
	cache.fallbackScheduled = true
	cache.m.Unlock()

	go func() {
		defer func() {
			cache.m.Lock()
			cache.fallbackScheduled = false
			cache.m.Unlock()
		}()
		wait := m.initialSnapshotFallbackDelay
		for {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			select {
			case <-ctx.Done():
				return
			case m.initialSnapshotFallbackLimit <- struct{}{}:
			}

			cache.m.Lock()
			switch cache.state {
			case cacheStateSynced:
				cache.m.Unlock()
				<-m.initialSnapshotFallbackLimit
				return
			case cacheStateInitialised:
				cache.state = cacheStateQueuing
				cache.m.Unlock()
			case cacheStateQueuing:
				cache.m.Unlock()
				<-m.initialSnapshotFallbackLimit
				wait = m.initialSnapshotFallbackRetry
				continue
			default:
				cache.m.Unlock()
				<-m.initialSnapshotFallbackLimit
				return
			}

			err := m.syncOrderbook(ctx, cache, p, a)
			<-m.initialSnapshotFallbackLimit
			if err == nil {
				return
			}
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return
			}
			log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: fallback snapshot failed for %v %v: %v", m.ob.exchangeName, p, a, err)
			wait = m.initialSnapshotFallbackRetry
		}
	}()
	return nil
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *UpdateManager) ProcessOrderbookUpdate(ctx context.Context, firstUpdateID int64, update *orderbook.Update) error {
	if update == nil {
		return common.ErrNilPointer
	}
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
	cacheKey := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
	m.lookupMu.RLock()
	cache, ok := m.lookup[cacheKey]
	m.lookupMu.RUnlock()
	if ok {
		return cache, nil
	}

	m.lookupMu.Lock()
	defer m.lookupMu.Unlock()
	if cache, ok = m.lookup[cacheKey]; ok {
		return cache, nil
	}
	cache = &updateCache{ch: make(chan int64), state: cacheStateInitialised}
	m.lookup[cacheKey] = cache
	return cache, nil
}

// applyUpdate verifies and applies an orderbook update
// Invalidates the cache on error
// assumes lock already active on cache
func (m *UpdateManager) applyUpdate(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	lastUpdateID, err := m.ob.LastUpdateID(update.Pair, update.Asset)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache)
	}
	if m.checkLiveUpdates {
		skip, err := m.checkPendingUpdate(lastUpdateID, firstUpdateID, update)
		if err != nil {
			return m.handleDesync(ctx, cache, firstUpdateID, update)
		}
		if skip {
			return nil
		}
	} else if lastUpdateID+1 != firstUpdateID {
		return m.handleDesync(ctx, cache, firstUpdateID, update)
	}
	if err := m.ob.Update(update); err != nil {
		log.Errorf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: %v", m.ob.exchangeName, update.Pair, update.Asset, err)
		return m.invalidateCache(ctx, firstUpdateID, update, cache)
	}
	return nil
}

func (m *UpdateManager) handleDesync(ctx context.Context, cache *updateCache, firstUpdateID int64, update *orderbook.Update) error {
	if m.ob.verbose { // disconnection will pollute logs
		log.Warnf(log.ExchangeSys, "%s websocket orderbook manager: failed to sync orderbook for %v %v: desync detected", m.ob.exchangeName, update.Pair, update.Asset)
	}
	return m.invalidateCache(ctx, firstUpdateID, update, cache)
}

// initialiseOrderbookCache sets the cache state to queuing, appends the update to the cache and spawns a goroutine
// to fetch and synchronise the orderbook snapshot
// assumes lock already active on cache
func (m *UpdateManager) initialiseOrderbookCache(ctx context.Context, firstUpdateID int64, update *orderbook.Update, cache *updateCache) {
	cache.state = cacheStateQueuing
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
	go func() {
		if err := m.syncOrderbook(ctx, cache, update.Pair, update.Asset); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
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
	if m.syncLimit != nil {
		select {
		case <-ctx.Done():
			cache.clearWithLock()
			return ctx.Err()
		case m.syncLimit <- struct{}{}:
			defer func() { <-m.syncLimit }()
		}
	}

	// REST requests can be behind websocket updates by a large margin, so we wait here to allow the cache to fill with
	// updates before we fetch the orderbook snapshot.
	select {
	case <-ctx.Done():
		cache.clearWithLock()
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
			return err
		}
		if book == nil {
			cache.clearWithLock()
			return common.ErrNilPointer
		}
		if !m.acceptSnapshotWhenUpdatesCovered {
			if err := cache.waitForUpdate(ctx, book.LastUpdateID+1); err != nil {
				cache.clearWithLock()
				return err
			}
		}

		if err := m.checkSnapshotCanApply(cache, book.LastUpdateID); err != nil {
			if m.acceptSnapshotWhenUpdatesCovered && errors.Is(err, errPendingUpdatesNotApplied) {
				break
			}
			if !errors.Is(err, ErrOrderbookSnapshotOutdated) {
				cache.clearWithLock()
				return err
			}
			select {
			case <-ctx.Done():
				cache.clearWithLock()
				return ctx.Err()
			case <-time.After(m.retryDelay):
				continue
			}
		}
		break
	}

	err := m.ob.LoadSnapshot(book)
	if err != nil {
		cache.clearWithLock()
		return err
	}

	cache.m.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		cache.clearNoLock()
		cache.m.Unlock()
	}()

	err = m.applyPendingUpdates(cache)
	if err != nil {
		if m.acceptSnapshotWhenUpdatesCovered && errors.Is(err, errPendingUpdatesNotApplied) {
			cache.state = cacheStateSynced
		} else {
			cache.resetStateNoLock()
			return common.AppendError(err, m.ob.InvalidateOrderbook(pair, a))
		}
	}

	return nil
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
	var updateListLastUpdateID int64
	if len(c.updates) > 0 && c.updates[len(c.updates)-1].update != nil {
		updateListLastUpdateID = c.updates[len(c.updates)-1].update.UpdateID
	}
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

func (c *updateCache) clearNoLock() {
	c.updates = nil
}

func (c *updateCache) resetStateNoLock() {
	c.state = cacheStateInitialised
}

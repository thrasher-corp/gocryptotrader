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
	cache, err := m.LoadCache(update.Pair, update.Asset)
	if err != nil {
		return err
	}

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

	lastUpdateID, updateErr := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if updateErr == nil && lastUpdateID+1 == firstUpdateID {
		if updateErr = e.Websocket.Orderbook.Update(update); updateErr == nil {
			return nil
		}
	}

	if updateErr != nil && errors.Is(updateErr, orderbook.ErrDepthNotFound) {
		updateErr = nil // silence error as this is expected initial sync behaviour
	}

	// Orderbook notifications are desynced, therefore invalidate store.
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

	return updateErr
}

// LoadCache loads the cache for the given pair and asset. If the cache does not exist, it creates a new one.
func (m *wsOBUpdateManager) LoadCache(p currency.Pair, a asset.Item) (*updateCache, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w: %q", asset.ErrInvalidAsset, a)
	}
	m.mtx.RLock()
	cache, ok := m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	m.mtx.RUnlock()
	if !ok {
		cache = &updateCache{ch: make(chan int64)}
		m.mtx.Lock()
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.mtx.Unlock()
	}
	return cache, nil
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (c *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, pair currency.Pair, a asset.Item, delay, deadline time.Duration) error {
	limit, err := c.extractOrderbookLimit(e, a)
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

	c.mtx.Lock() // Lock here to prevent ws handle data interference with REST request above.
	defer func() {
		c.clearNoLock()
		c.mtx.Unlock()
	}()

	if err := e.Websocket.Orderbook.LoadSnapshot(book); err != nil {
		return err
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
		// There is no way to set levels when we subscribe for this specific channel
		// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
		lim := uint64(sub.Interval.Duration().Milliseconds()) //nolint:gosec // No overflow risk
		if lim != 20 && lim != 100 {
			return 0, fmt.Errorf("%w: %d. Valid limits are 20 and 100", errInvalidOrderbookUpdateInterval, lim)
		}
		return lim, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return futuresOrderbookUpdateLimit, nil
	case asset.DeliveryFutures:
		return deliveryFuturesUpdateLimit, nil
	case asset.Options:
		return optionOrderbookUpdateLimit, nil
	default:
		return 0, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

// waitForUpdate waits for an update with an ID >= nextUpdateID
func (c *updateCache) waitForUpdate(ctx context.Context, nextUpdateID int64) error {
	c.mtx.Lock()
	updateListLastUpdateID := c.updates[len(c.updates)-1].update.UpdateID
	c.mtx.Unlock()

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
// Does not lock cache
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

package gateio

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const defaultWSSnapshotSyncDelay = 2 * time.Second

var errOrderbookSnapshotOutdated = errors.New("orderbook snapshot is outdated")

type wsOBUpdateManager struct {
	lookup            map[key.PairAsset]*updateCache
	snapshotSyncDelay time.Duration
	mtx               sync.RWMutex
}

type updateCache struct {
	updates  []pendingUpdate
	updating bool
	mtx      sync.Mutex
}

type pendingUpdate struct {
	update        *orderbook.Update
	firstUpdateID int64
}

func newWsOBUpdateManager(snapshotSyncDelay time.Duration) *wsOBUpdateManager {
	return &wsOBUpdateManager{lookup: make(map[key.PairAsset]*updateCache), snapshotSyncDelay: snapshotSyncDelay}
}

// ProcessOrderbookUpdate processes an orderbook update by syncing snapshot, caching updates and applying them
func (m *wsOBUpdateManager) ProcessOrderbookUpdate(ctx context.Context, g *Exchange, firstUpdateID int64, update *orderbook.Update) error {
	cache := m.LoadCache(update.Pair, update.Asset)
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	if cache.updating {
		cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})
		return nil
	}

	lastUpdateID, err := g.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset)
	if err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	if lastUpdateID+1 >= firstUpdateID {
		return applyOrderbookUpdate(g, update)
	}

	// Orderbook is behind notifications, therefore Invalidate store
	if err := g.Websocket.Orderbook.InvalidateOrderbook(update.Pair, update.Asset); err != nil && !errors.Is(err, orderbook.ErrDepthNotFound) {
		return err
	}

	cache.updating = true
	cache.updates = append(cache.updates, pendingUpdate{update: update, firstUpdateID: firstUpdateID})

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(m.snapshotSyncDelay):
			if err := cache.SyncOrderbook(ctx, g, update.Pair, update.Asset); err != nil {
				g.Websocket.DataHandler <- fmt.Errorf("failed to sync orderbook for %v %v: %w", update.Pair, update.Asset, err)
			}
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
		cache = &updateCache{}
		m.lookup[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}] = cache
		m.mtx.Unlock()
	}
	return cache
}

// SyncOrderbook fetches and synchronises an orderbook snapshot to the limit size so that pending updates can be
// applied to the orderbook.
func (c *updateCache) SyncOrderbook(ctx context.Context, e *Exchange, pair currency.Pair, a asset.Item) error {
	wanted := &orderbookSubKey{Subscription: &subscription.Subscription{
		Asset:   a,
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{pair},
	}}
	sub := e.Websocket.GetSubscription(wanted)
	if sub == nil {
		return fmt.Errorf("no subscription found for %q", wanted)
	}
	obParams := make(map[string]string)
	if err := orderbookPayload(sub, a, channelName(sub, a), obParams); err != nil {
		return err
	}
	var levels uint64
	if levelsStr, ok := obParams["depth"]; !ok {
		return fmt.Errorf("error syncing orderbook: %w from sub %q", subscription.ErrInvalidLevel, sub)
	} else { //nolint:revive // using local scope levelsStr variable
		var err error
		if levels, err = strconv.ParseUint(levelsStr, 10, 64); err != nil {
			return err
		}
	}

	book, err := e.UpdateOrderbookWithLimit(ctx, pair, a, levels)

	c.mtx.Lock() // lock here to prevent ws handle data interference with REST request above
	defer func() {
		c.updates = nil
		c.updating = false
		c.mtx.Unlock()
	}()

	if err != nil {
		return err
	}

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
	return c.applyPendingUpdates(e, a)
}

// ApplyPendingUpdates applies all pending updates to the orderbook
func (c *updateCache) applyPendingUpdates(g *Exchange, a asset.Item) error {
	for _, data := range c.updates {
		lastUpdateID, err := g.Websocket.Orderbook.LastUpdateID(data.update.Pair, a)
		if err != nil {
			return err
		}
		nextID := lastUpdateID + 1
		if data.firstUpdateID > nextID {
			return errOrderbookSnapshotOutdated
		}
		if data.update.UpdateID < nextID {
			continue // skip updates that are behind the current orderbook
		}
		if err := applyOrderbookUpdate(g, data.update); err != nil {
			return err
		}
	}
	return nil
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

type orderbookSubKey struct {
	*subscription.Subscription
}

var _ subscription.MatchableKey = orderbookSubKey{}

// Match returns if the sub has the correct Channel, Asset and contains all of the Pairs
func (k orderbookSubKey) Match(eachKey subscription.MatchableKey) bool {
	eachSub := eachKey.GetSubscription()
	switch {
	case eachSub.Channel != k.Channel,
		eachSub.Asset != k.Asset,
		// len(eachSub.Pairs) == 0 && len(s.Pairs) == 0: Okay; continue to next non-pairs check
		len(eachSub.Pairs) == 0 && len(k.Pairs) != 0,
		len(eachSub.Pairs) != 0 && len(k.Pairs) == 0,
		len(k.Pairs) != 0 && eachSub.Pairs.ContainsAll(k.Pairs, true) != nil:
		return false
	}
	return true
}

// String implements Stringer; returns the Channel name, Assets and Pairs
func (k orderbookSubKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised orderbookSubKey"
	}
	return fmt.Sprintf("%s %s %s", s.Channel, s.Asset, s.Pairs)
}

func (k orderbookSubKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

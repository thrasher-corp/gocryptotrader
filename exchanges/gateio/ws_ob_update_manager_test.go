package gateio

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()

	m := newWsOBUpdateManager(time.Millisecond*200, time.Millisecond*200)
	err := m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	cache, err := m.LoadCache(pair, asset.USDTMarginedFutures)
	require.NoError(t, err)
	cache.mtx.Lock()
	assert.Equal(t, cacheStateInitialised, cache.state)
	cache.mtx.Unlock()

	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     e.Name,
		Pair:         pair,
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	})
	require.NoError(t, err)

	// demonstrate an update is queued
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)
	cache.mtx.Lock()
	assert.Equal(t, cacheStateQueuing, cache.state)
	cache.mtx.Unlock()
	// demonstrate an additional update is queued
	var wg1, wg2 sync.WaitGroup
	wg1.Add(1)
	wg2.Go(func() {
		wg1.Done()
		updatedID := <-cache.ch
		assert.Equal(t, int64(1339), updatedID)
	})
	wg1.Wait()
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1339,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	wg2.Wait()

	require.NoError(t, err)
	cache.mtx.Lock()
	assert.Equal(t, cacheStateQueuing, cache.state)
	cache.mtx.Unlock()

	// demonstrate an error state and forces everything to queue
	assert.Eventually(t, func() bool {
		cache.mtx.Lock()
		defer cache.mtx.Unlock()
		return cache.state == cacheStateQueuing
	}, time.Second, time.Millisecond*10, "sync should eventually fail as BABYBABYDOGE is not a supported pair")

	// demonstrate an error state is recovered
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	// demonstrate successful processing
	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     e.Name,
		Pair:         pair,
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	})
	require.NoError(t, err)
	cache.mtx.Lock()
	cache.state = cacheStateSynced
	cache.mtx.Unlock()
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache.mtx.Lock()
	cache.state = 100
	cache.mtx.Unlock()
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1339,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.ErrorIs(t, err, errUnhandledCacheState)
}

func TestLoadCache(t *testing.T) {
	t.Parallel()

	m := newWsOBUpdateManager(0, 0)
	_, err := m.LoadCache(currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = m.LoadCache(currency.NewBTCUSDT(), 1336)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	cache, err := m.LoadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.NotNil(t, cache)
	assert.Len(t, m.lookup, 1)

	// Test cache is reused
	cache2, err := m.LoadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.Equal(t, cache, cache2)
}

func TestSyncOrderbook(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	cache := &updateCache{}
	pair := currency.NewPair(currency.ETH, currency.USDT)
	err := cache.SyncOrderbook(t.Context(), e, pair, asset.Spot, 0, defaultWSOrderbookUpdateDeadline)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	require.NoError(t, err)

	ctxCancel, cancel := context.WithCancel(t.Context())
	cancel()
	err = cache.SyncOrderbook(ctxCancel, e, pair, asset.Spot, 0, defaultWSOrderbookUpdateDeadline)
	require.ErrorIs(t, err, context.Canceled)

	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}}}
	err = cache.SyncOrderbook(t.Context(), e, pair, asset.Spot, 0, 0)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}}}
	err = cache.SyncOrderbook(t.Context(), e, pair, asset.Spot, 0, time.Second)
	require.ErrorContains(t, err, context.DeadlineExceeded.Error())

	err = e.Base.SetPairs([]currency.Pair{pair}, asset.Spot, true)
	require.NoError(t, err)
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: math.MaxInt64}}}
	err = cache.SyncOrderbook(t.Context(), e, pair, asset.Spot, 0, time.Second)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)

	err = e.Base.SetPairs([]currency.Pair{pair}, asset.USDTMarginedFutures, true)
	require.NoError(t, err)
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: math.MaxInt64}}}
	err = cache.SyncOrderbook(t.Context(), e, pair, asset.USDTMarginedFutures, 0, time.Second)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)
}

func TestWaitForUpdate(t *testing.T) {
	t.Parallel()

	cache := &updateCache{
		updates: []pendingUpdate{
			{update: &orderbook.Update{Pair: currency.NewBTCUSD(), Asset: asset.Spot, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()}},
		},
	}

	err := cache.waitForUpdate(t.Context(), 1337)
	require.NoError(t, err)

	ctx, cancel := context.WithDeadline(t.Context(), time.Now())
	defer cancel()
	err = cache.waitForUpdate(ctx, 1338)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	cache.ch = make(chan int64, 1) // Reset channel to avoid deadlock
	var wg sync.WaitGroup
	wg.Go(func() {
		err = cache.waitForUpdate(t.Context(), 1338)
	})
	cache.ch <- 1338
	wg.Wait()
	assert.NoError(t, err)
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	pair := currency.NewPair(currency.LTC, currency.USDT)
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures}}}}
	err := cache.applyPendingUpdates(e)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	dummy := &orderbook.Book{
		Exchange:     e.Name,
		Pair:         pair,
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1335,
	}

	err = e.Websocket.Orderbook.LoadSnapshot(dummy)
	require.NoError(t, err)

	err = cache.applyPendingUpdates(e)
	require.ErrorIs(t, err, errPendingUpdatesNotApplied)

	cache.updates[0].firstUpdateID = 1337
	cache.updates[0].update.UpdateID = 1338
	err = cache.applyPendingUpdates(e)
	require.ErrorIs(t, err, errOrderbookSnapshotOutdated)

	cache.updates[0].firstUpdateID = 1336
	cache.updates[0].update.UpdateID = 1338
	err = cache.applyPendingUpdates(e)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)

	err = e.Websocket.Orderbook.LoadSnapshot(dummy)
	require.NoError(t, err)

	cache.updates[0].update.AllowEmpty = true
	cache.updates[0].update.UpdateTime = time.Now()
	err = cache.applyPendingUpdates(e)
	require.NoError(t, err)

	cache.updates[0].firstUpdateID = 1339
	cache.updates[0].update.UpdateID = 1342
	cache.updates = append(cache.updates, pendingUpdate{
		firstUpdateID: 1344,
		update:        &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: 1345, AllowEmpty: true, UpdateTime: time.Now()},
	})
	err = cache.applyPendingUpdates(e)
	require.ErrorIs(t, err, errOrderbookSnapshotOutdated)
}

func TestClearWithLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}}
	cache.clearWithLock()
	require.Empty(t, cache.updates)
}

func TestClearNoLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}}
	cache.clearNoLock()
	require.Empty(t, cache.updates)
}

func TestHandleSynchronisedState(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	err := testexch.Setup(e)
	require.NoError(t, err, "Setup must not error")
	e.Name = "HandleSyncStateTest"

	m := newWsOBUpdateManager(0, 0)
	cache, err := m.LoadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")

	err = m.handleSynchronisedState(t.Context(), e, cache, 1, &orderbook.Update{
		Pair:  currency.NewBTCUSDT(),
		Asset: asset.USDTMarginedFutures,
	})
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "handleSynchronisedState must error when not initialised")

	snapshot := &orderbook.Book{
		Exchange:     e.Name,
		Pair:         currency.NewBTCUSDT(),
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	}

	err = e.Websocket.Orderbook.LoadSnapshot(snapshot)
	require.NoError(t, err)

	err = m.handleSynchronisedState(t.Context(), e, cache, 1, &orderbook.Update{
		UpdateID: 1338,
		Pair:     currency.NewBTCUSDT(),
		Asset:    asset.USDTMarginedFutures,
	})
	require.NoError(t, err, "handleSynchronisedState must not error when desynced")

	_, err = e.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "LastUpdateID must error after handleInvalidCache is called")

	err = e.Websocket.Orderbook.LoadSnapshot(snapshot)
	require.NoError(t, err)

	err = m.handleSynchronisedState(t.Context(), e, cache, 1337, &orderbook.Update{
		UpdateID: 1339,
		Pair:     currency.NewBTCUSDT(),
		Asset:    asset.USDTMarginedFutures,
	})
	require.NoError(t, err, "handleSynchronisedState must not error when in sync but update failed to apply")

	_, err = e.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "LastUpdateID must error after handleInvalidCache is called")

	err = e.Websocket.Orderbook.LoadSnapshot(snapshot)
	require.NoError(t, err)

	err = m.handleSynchronisedState(t.Context(), e, cache, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
	})
	require.NoError(t, err, "handleSynchronisedState must not error when in sync and update applied")
}

func TestOBManagerProcessOrderbookUpdateHTTPMocked(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	e.Name = "ManagerHTTPMocked"
	err := testexch.MockHTTPInstance(e, "/api/v4/")
	require.NoError(t, err, "MockHTTPInstance must not error")

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.TwentyMilliseconds})
	require.NoError(t, err)

	m := newWsOBUpdateManager(0, defaultWSOrderbookUpdateDeadline)
	err = m.ProcessOrderbookUpdate(t.Context(), e, 27596272446, &orderbook.Update{
		UpdateID:   27596272447,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error")

	time.Sleep(time.Millisecond * 100) // allow some time for rest mock request routine to complete

	err = m.ProcessOrderbookUpdate(t.Context(), e, 27596272448, &orderbook.Update{
		UpdateID:   27596272449,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error on synced orderbook")

	id, err := e.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(27596272449), id, "LastUpdateID must be updated to last update")
}

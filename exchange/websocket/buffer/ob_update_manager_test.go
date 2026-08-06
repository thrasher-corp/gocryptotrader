package buffer

import (
	"context"
	"errors"
	"expvar"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/metrics"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var errTestDesyncReason = errors.New("test desync")

func newTestParams() UpdateManagerParams {
	return UpdateManagerParams{
		FetchDeadline:  time.Second,
		FetchOrderbook: fetchOrderbookNotFoundError,
		CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (bool, error) {
			return false, nil
		},
		BufferInstance: &Orderbook{exchangeName: "TestExchange", ob: make(map[key.PairAsset]*orderbookHolder), dataHandler: stream.NewRelay(1000), verbose: true},
	}
}

func fetchOrderbookMock(_ context.Context, pair currency.Pair, a asset.Item) (*orderbook.Book, error) {
	return &orderbook.Book{
		Exchange:     "TestExchange",
		Pair:         pair,
		Asset:        a,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	}, nil
}

func fetchOrderbookNotFoundError(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
	return nil, orderbook.ErrDepthNotFound
}

func fetchOrderbookFailure(_ context.Context, pair currency.Pair, a asset.Item) (*orderbook.Book, error) {
	return &orderbook.Book{
		Exchange:     "TestExchange",
		Pair:         pair,
		Asset:        a,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdateID: 1336,
	}, nil
}

func TestNewUpdateManager(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { NewUpdateManager(nil) })

	params := &UpdateManagerParams{FetchDelay: -time.Second, FetchDeadline: -time.Second}
	require.Panics(t, func() { NewUpdateManager(params) })
	params.FetchDeadline = time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.FetchDelay = time.Second
	params.OutdatedSnapshotRetryDelay = -time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.OutdatedSnapshotRetryDelay = 0
	params.SnapshotSyncLimit = -1
	require.Panics(t, func() { NewUpdateManager(params) })
	params.SnapshotSyncLimit = 2
	params.InitialSnapshotFallbackDelay = -time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.InitialSnapshotFallbackDelay = time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.InitialSnapshotFallbackLimit = -1
	require.Panics(t, func() { NewUpdateManager(params) })
	params.InitialSnapshotFallbackLimit = 1
	params.InitialSnapshotFallbackRetryDelay = -time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.InitialSnapshotFallbackRetryDelay = 0
	require.Panics(t, func() { NewUpdateManager(params) })

	params.FetchOrderbook = func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) { return nil, nil }
	params.CheckPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	}
	params.AcceptSnapshotWhenUpdatesCovered = true
	params.BufferInstance = &Orderbook{}
	got := NewUpdateManager(params)
	require.NotNil(t, got)
	assert.NotNil(t, got.lookup)
	assert.Equal(t,
		2,
		cap(got.syncLimit),
		"sync limit should match configured snapshot sync limit")
	assert.True(t,
		got.acceptSnapshotWhenUpdatesCovered,
		"covered snapshot acceptance should match configuration")
	assert.Equal(t, time.Second, got.initialSnapshotFallbackDelay)
	assert.Equal(t, 30*time.Second, got.initialSnapshotFallbackRetry)
	assert.Equal(t, 1, cap(got.initialSnapshotFallbackLimit))
}

func TestScheduleInitialSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("fallback_initialises_quiet_book_once", func(t *testing.T) {
		t.Parallel()
		pair := currency.NewBTCUSDT()
		params := newTestParams()
		params.BufferInstance.exchangeName = "TestExchangeFallbackQuiet"
		params.AcceptSnapshotWhenUpdatesCovered = true
		params.InitialSnapshotFallbackDelay = time.Millisecond
		params.InitialSnapshotFallbackRetryDelay = 10 * time.Millisecond
		params.InitialSnapshotFallbackLimit = 1
		var fetches atomic.Int64
		params.FetchOrderbook = func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			fetches.Add(1)
			return &orderbook.Book{
				Exchange:     params.BufferInstance.exchangeName,
				Pair:         p,
				Asset:        a,
				Bids:         orderbook.Levels{{Price: 1, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 2, Amount: 1}},
				LastUpdated:  time.Now(),
				LastPushed:   time.Now(),
				LastUpdateID: 1336,
			}, nil
		}
		manager := NewUpdateManager(&params)

		require.NoError(t, manager.ScheduleInitialSnapshot(t.Context(), pair, asset.Spot))
		require.NoError(t, manager.ScheduleInitialSnapshot(t.Context(), pair, asset.Spot))
		require.Eventually(t, func() bool {
			id, err := params.BufferInstance.LastUpdateID(pair, asset.Spot)
			return err == nil && id == 1336
		}, time.Second, time.Millisecond)
		require.Equal(t, int64(1), fetches.Load(), "duplicate schedules must not duplicate REST snapshots")

		stats := manager.BootstrapStats()[asset.Spot]
		assert.Equal(t, 1, stats.Subscribed)
		assert.Zero(t, stats.FirstUpdateSeen)
		assert.Equal(t, 1, stats.SnapshotStarted)
		assert.Equal(t, 1, stats.SnapshotSucceeded)
		assert.Zero(t, stats.SnapshotFailed)
		assert.Zero(t, stats.WaitingForFirstUpdate)
		assert.Equal(t, 1, stats.FallbackStarted)
	})

	t.Run("first_update_wins", func(t *testing.T) {
		t.Parallel()
		pair := currency.NewPair(currency.LTC, currency.USDT)
		params := newTestParams()
		params.BufferInstance.exchangeName = "TestExchangeFirstUpdateWins"
		params.AcceptSnapshotWhenUpdatesCovered = true
		params.InitialSnapshotFallbackDelay = 30 * time.Millisecond
		params.InitialSnapshotFallbackRetryDelay = 10 * time.Millisecond
		params.InitialSnapshotFallbackLimit = 1
		var fetches atomic.Int64
		params.FetchOrderbook = func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			fetches.Add(1)
			return &orderbook.Book{
				Exchange:     params.BufferInstance.exchangeName,
				Pair:         p,
				Asset:        a,
				Bids:         orderbook.Levels{{Price: 1, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 2, Amount: 1}},
				LastUpdated:  time.Now(),
				LastPushed:   time.Now(),
				LastUpdateID: 1336,
			}, nil
		}
		params.CheckPendingUpdate = func(lastUpdateID, firstUpdateID int64, _ *orderbook.Update) (bool, error) {
			return firstUpdateID <= lastUpdateID, nil
		}
		manager := NewUpdateManager(&params)

		require.NoError(t, manager.ScheduleInitialSnapshot(t.Context(), pair, asset.Spot))
		require.NoError(t, manager.ProcessOrderbookUpdate(t.Context(), 1337, &orderbook.Update{
			Pair:       pair,
			Asset:      asset.Spot,
			AllowEmpty: true,
			UpdateID:   1337,
			UpdateTime: time.Now(),
		}))
		require.Eventually(t, func() bool {
			id, err := params.BufferInstance.LastUpdateID(pair, asset.Spot)
			return err == nil && id == 1337
		}, time.Second, time.Millisecond)
		time.Sleep(50 * time.Millisecond)
		require.Equal(t, int64(1), fetches.Load(), "fallback must not fetch after websocket bootstrap succeeds")

		stats := manager.BootstrapStats()[asset.Spot]
		assert.Equal(t, 1, stats.FirstUpdateSeen)
		assert.Equal(t, 1, stats.SnapshotSucceeded)
		assert.Zero(t, stats.FallbackStarted)
	})

	t.Run("fallback_retries_after_failure", func(t *testing.T) {
		t.Parallel()
		pair := currency.NewPair(currency.XRP, currency.USDT)
		params := newTestParams()
		params.BufferInstance.exchangeName = "TestExchangeFallbackRetry"
		params.AcceptSnapshotWhenUpdatesCovered = true
		params.InitialSnapshotFallbackDelay = time.Millisecond
		params.InitialSnapshotFallbackRetryDelay = time.Millisecond
		params.InitialSnapshotFallbackLimit = 1
		var fetches atomic.Int64
		params.FetchOrderbook = func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			if fetches.Add(1) == 1 {
				return nil, errTestDesyncReason
			}
			return &orderbook.Book{
				Exchange:     params.BufferInstance.exchangeName,
				Pair:         p,
				Asset:        a,
				Bids:         orderbook.Levels{{Price: 1, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 2, Amount: 1}},
				LastUpdated:  time.Now(),
				LastPushed:   time.Now(),
				LastUpdateID: 1336,
			}, nil
		}
		manager := NewUpdateManager(&params)

		require.NoError(t, manager.ScheduleInitialSnapshot(t.Context(), pair, asset.Spot))
		require.Eventually(t, func() bool {
			id, err := params.BufferInstance.LastUpdateID(pair, asset.Spot)
			return err == nil && id == 1336
		}, time.Second, time.Millisecond)
		assert.Equal(t, int64(2), fetches.Load())
		stats := manager.BootstrapStats()[asset.Spot]
		assert.Equal(t, 1, stats.SnapshotFailed)
		assert.Equal(t, 1, stats.SnapshotSucceeded)
	})

	t.Run("cancelled_before_fallback", func(t *testing.T) {
		t.Parallel()
		params := newTestParams()
		params.AcceptSnapshotWhenUpdatesCovered = true
		params.InitialSnapshotFallbackDelay = 10 * time.Millisecond
		params.InitialSnapshotFallbackLimit = 1
		var fetches atomic.Int64
		params.FetchOrderbook = func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
			fetches.Add(1)
			return nil, errTestDesyncReason
		}
		manager := NewUpdateManager(&params)
		ctx, cancel := context.WithCancel(t.Context())
		require.NoError(t, manager.ScheduleInitialSnapshot(ctx, currency.NewBTCUSDT(), asset.Spot))
		cancel()
		time.Sleep(30 * time.Millisecond)
		assert.Zero(t, fetches.Load())
	})
}

func TestBootstrapStats(t *testing.T) {
	t.Parallel()
	manager := NewUpdateManager(func() *UpdateManagerParams {
		params := newTestParams()
		return &params
	}())
	spotCache, err := manager.loadCache(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err)
	spotCache.fallbackScheduled = true
	spotCache.firstUpdateSeen = true
	spotCache.snapshotStarted = true
	spotCache.snapshotSucceeded = true

	futuresCache, err := manager.loadCache(currency.NewPair(currency.ETH, currency.USDT), asset.Futures)
	require.NoError(t, err)
	futuresCache.fallbackScheduled = true
	futuresCache.snapshotStarted = true
	futuresCache.snapshotFailures = 2
	futuresCache.fallbackStarted = true

	stats := manager.BootstrapStats()
	assert.Equal(t, OrderbookBootstrapStats{
		Subscribed:        1,
		FirstUpdateSeen:   1,
		SnapshotStarted:   1,
		SnapshotSucceeded: 1,
	}, stats[asset.Spot])
	assert.Equal(t, OrderbookBootstrapStats{
		Subscribed:      1,
		SnapshotStarted: 1,
		SnapshotFailed:  2,
		FallbackStarted: 1,
	}, stats[asset.Futures])
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	tp := newTestParams()
	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	tp.FetchOrderbook = func(_ context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Book, error) {
		return &orderbook.Book{
			Exchange:     "TestExchange",
			Pair:         pair,
			Asset:        asset.USDTMarginedFutures,
			Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
			Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
			LastUpdated:  time.Now(),
			LastPushed:   time.Now(),
			LastUpdateID: 1336,
		}, nil
	}

	m := NewUpdateManager(&tp)
	err := m.ProcessOrderbookUpdate(t.Context(), 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "should error on loadcache method")

	cache, err := m.loadCache(pair, asset.USDTMarginedFutures)
	require.NoError(t, err)
	cache.m.Lock()
	assert.Equal(t, cacheStateInitialised, cache.state, "state should be initialised after first load")
	cache.m.Unlock()

	err = m.ProcessOrderbookUpdate(t.Context(), 1337, &orderbook.Update{
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error on synced orderbook")

	eventuallyCondition := func() bool {
		id, err := tp.BufferInstance.LastUpdateID(pair, asset.USDTMarginedFutures)
		return err == nil && id == 1338
	}
	require.Eventually(t, eventuallyCondition, time.Second, time.Millisecond*50, "LastUpdateID must return to snapshot and update applied to state after invalidateCache is processed")

	cache.m.Lock()
	cache.state = cacheStateUninitialised
	cache.m.Unlock()
	err = m.ProcessOrderbookUpdate(t.Context(), 1337, &orderbook.Update{
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	})
	require.ErrorIs(t, err, errUnhandledCacheState, "ProcessOrderbookUpdate must error due to unhandled cache state")

	cache.m.Lock()
	cache.state = cacheStateQueuing
	cache.ch = make(chan int64, 1)
	cache.m.Unlock()
	err = m.ProcessOrderbookUpdate(t.Context(), 1337, &orderbook.Update{
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error when queuing update")
	cache.m.Lock()
	assert.Equal(t, 1, len(cache.updates), "should have one queued update")
	assert.NotEmpty(t, cache.ch)
	cache.m.Unlock()
}

func TestLoadCache(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	_, err := m.loadCache(currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = m.loadCache(currency.NewBTCUSDT(), 1336)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	cache, err := m.loadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.NotNil(t, cache)
	assert.Len(t, m.lookup, 1)
	assert.Equal(t, cacheStateInitialised, cache.state, "state should be initialised after first load")

	cache2, err := m.loadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.Equal(t, cache, cache2, "should be the same cache instance")
}

func TestApplyUpdate(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.fetchOrderbook = fetchOrderbookMock
	m.checkPendingUpdate = func(_, firstUpdateID int64, _ *orderbook.Update) (bool, error) {
		return firstUpdateID != 1337, nil
	}
	cache, err := m.loadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "loadCache must not error")

	checkForRoutineRefresh := func() bool {
		id, err := m.ob.LastUpdateID(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
		return err == nil && id == 1338
	}

	goodUpdate := &orderbook.Update{
		UpdateID:   1338,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	cache.m.Lock()
	err = m.applyUpdate(t.Context(), cache, 1337, goodUpdate)
	cache.m.Unlock()
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "applyUpdate must error when not initialised")

	require.Eventually(t, checkForRoutineRefresh, time.Second, time.Millisecond*50, "LastUpdateID must return to snapshot and update applied to state after invalidateCache is processed")

	cache.m.Lock()
	err = m.applyUpdate(t.Context(), cache, 1337, goodUpdate)
	cache.m.Unlock()
	require.NoError(t, err, "applyUpdate must not error when in desync and update stored")

	require.Eventually(t, checkForRoutineRefresh, time.Second, time.Millisecond*50, "LastUpdateID must return to snapshot and update applied to state after invalidateCache is processed")

	m.deadline = time.Second * 5
	badUpdate := *goodUpdate
	badUpdate.UpdateTime = time.Time{}
	badUpdate.UpdateID = 1333
	cache.m.Lock()
	err = m.applyUpdate(t.Context(), cache, 1334, &badUpdate)
	cache.m.Unlock()
	require.NoError(t, err, "applyUpdate must not error when applying update fails, this will be filtered because it will be behind last update ID")

	err = m.ProcessOrderbookUpdate(t.Context(), 1337, goodUpdate)
	require.NoError(t, err, "ProcessOrderbookUpdate must not error when queueing good update")
	require.Eventually(t, checkForRoutineRefresh, time.Second, time.Millisecond*50, "LastUpdateID must return to snapshot state after invalidateCache is processed")

	goodUpdate.UpdateID = 1339
	err = m.ProcessOrderbookUpdate(t.Context(), 1339, goodUpdate)
	require.NoError(t, err, "ProcessOrderbookUpdate must not error when applying good update")
	id, err := m.ob.LastUpdateID(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LastUpdateID must not error after successful update application")
	require.Equal(t, int64(1339), id, "LastUpdateID must match the last applied update ID")

	t.Run("check_live_updates", func(t *testing.T) {
		t.Parallel()
		pair := currency.NewBTCUSDT()
		tp := newTestParams()
		tp.CheckLiveUpdates = true
		tp.CheckPendingUpdate = func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (bool, error) {
			target := lastUpdateID + 1
			if firstUpdateID > target {
				return false, ErrOrderbookSnapshotOutdated
			}
			if update.UpdateID < target {
				return true, nil
			}
			bids := make(orderbook.Levels, 0, len(update.Bids))
			for i := range update.Bids {
				if update.Bids[i].ID >= target {
					bids = append(bids, update.Bids[i])
				}
			}
			update.Bids = bids
			return false, nil
		}
		m := NewUpdateManager(&tp)
		require.NoError(t, m.ob.LoadSnapshot(&orderbook.Book{
			Exchange:     m.ob.exchangeName,
			Pair:         pair,
			Asset:        asset.Spot,
			Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
			Asks:         []orderbook.Level{{Price: 2, Amount: 1}},
			LastUpdated:  time.Now(),
			LastPushed:   time.Now(),
			LastUpdateID: 1337,
		}), "LoadSnapshot must not error")
		cache, err := m.loadCache(pair, asset.Spot)
		require.NoError(t, err, "loadCache must not error")

		updateTime := time.Now()
		cache.m.Lock()
		err = m.applyUpdate(t.Context(), cache, 1336, &orderbook.Update{
			Pair:       pair,
			Asset:      asset.Spot,
			UpdateID:   1339,
			UpdateTime: updateTime,
			LastPushed: updateTime,
			Bids: orderbook.Levels{
				{Price: 0.9, Amount: 1, ID: 1336},
				{Price: 1.1, Amount: 2, ID: 1338},
			},
		})
		cache.m.Unlock()
		require.NoError(t, err, "applyUpdate must accept and trim overlapping live updates")

		book, err := m.ob.GetOrderbook(pair, asset.Spot)
		require.NoError(t, err, "GetOrderbook must not error")
		require.Equal(t, int64(1339), book.LastUpdateID, "LastUpdateID must match the overlapping update end")
		require.Len(t, book.Bids, 2, "bids must include original and non-stale overlapping level")
		assert.Equal(t, 1.1, book.Bids[0].Price, "fresh overlapping level should be applied")

		cache.m.Lock()
		err = m.applyUpdate(t.Context(), cache, 1336, &orderbook.Update{
			Pair:       pair,
			Asset:      asset.Spot,
			UpdateID:   1337,
			UpdateTime: updateTime,
			LastPushed: updateTime,
			Bids:       orderbook.Levels{{Price: 1.2, Amount: 2, ID: 1337}},
		})
		cache.m.Unlock()
		require.NoError(t, err, "applyUpdate must skip live updates that are already behind")
		id, err := m.ob.LastUpdateID(pair, asset.Spot)
		require.NoError(t, err, "LastUpdateID must not error")
		assert.Equal(t, int64(1339), id, "skipped update should not move LastUpdateID")
	})
}

func TestCheckSnapshotCanApply(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSDT()
	tp := newTestParams()
	tp.CheckPendingUpdate = func(lastUpdateID, firstUpdateID int64, _ *orderbook.Update) (bool, error) {
		if firstUpdateID > lastUpdateID+1 {
			return false, ErrOrderbookSnapshotOutdated
		}
		return false, nil
	}
	m := NewUpdateManager(&tp)
	cache, err := m.loadCache(pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")
	cache.updates = []pendingUpdate{{
		firstUpdateID: 1337,
		update: &orderbook.Update{
			Pair:       pair,
			Asset:      asset.Spot,
			UpdateID:   1339,
			UpdateTime: time.Now(),
			Bids:       orderbook.Levels{{Price: 1, Amount: 1, ID: 1337}},
			Asks:       orderbook.Levels{{Price: 2, Amount: 1, ID: 1339}},
		},
	}}

	require.NoError(t, m.checkSnapshotCanApply(cache, 1336), "checkSnapshotCanApply must accept a snapshot bridged by the queued update")
	require.ErrorIs(t, m.checkSnapshotCanApply(cache, 1335), ErrOrderbookSnapshotOutdated, "checkSnapshotCanApply must reject a stale snapshot")
	require.Equal(t, orderbook.Levels{{Price: 1, Amount: 1, ID: 1337}}, cache.updates[0].update.Bids, "queued bids must not be mutated by the snapshot check")
	require.Equal(t, orderbook.Levels{{Price: 2, Amount: 1, ID: 1339}}, cache.updates[0].update.Asks, "queued asks must not be mutated by the snapshot check")
}

func TestCloneOrderbookUpdate(t *testing.T) {
	t.Parallel()

	require.Nil(t, cloneOrderbookUpdate(nil), "cloneOrderbookUpdate must return nil for nil updates")
	update := &orderbook.Update{
		UpdateID:   1337,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		Bids:       orderbook.Levels{{Price: 1, Amount: 2, ID: 3}},
		Asks:       orderbook.Levels{{Price: 4, Amount: 5, ID: 6}},
		UpdateTime: time.Now(),
	}

	clone := cloneOrderbookUpdate(update)
	require.NotSame(t, update, clone, "cloneOrderbookUpdate must return a different update pointer")
	require.Equal(t, update, clone, "cloneOrderbookUpdate must preserve update values")
	clone.Bids[0].Amount = 99
	clone.Asks[0].Amount = 100
	assert.Equal(t, 2.0, update.Bids[0].Amount, "source bids should not change when clone bids mutate")
	assert.Equal(t, 5.0, update.Asks[0].Amount, "source asks should not change when clone asks mutate")
}

func TestApplyUpdatePanicOnDesync(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	tp.PanicOnDesync = true
	m := NewUpdateManager(&tp)
	pair := currency.NewBTCUSDT()
	require.NoError(t, m.ob.LoadSnapshot(&orderbook.Book{
		Exchange:     m.ob.exchangeName,
		Pair:         pair,
		Asset:        asset.Spot,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 2, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1337,
	}), "LoadSnapshot must not error")

	cache, err := m.loadCache(pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")

	cache.m.Lock()
	defer cache.m.Unlock()
	require.PanicsWithValue(t,
		"TestExchange websocket orderbook manager desync for BTCUSDT spot: last update ID 1337, first update ID 1339, update ID 1340",
		func() {
			_ = m.applyUpdate(t.Context(), cache, 1339, &orderbook.Update{
				Pair:       pair,
				Asset:      asset.Spot,
				UpdateID:   1340,
				UpdateTime: time.Now(),
				Bids:       orderbook.Levels{{Price: 1, Amount: 2}},
			})
		},
		"applyUpdate must panic with desync details when diagnostic mode is enabled")
}

func TestHandleDesync(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	tp.PanicOnDesync = true
	m := NewUpdateManager(&tp)
	pair := currency.NewBTCUSDT()
	cache, err := m.loadCache(pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")

	require.PanicsWithValue(t,
		"TestExchange websocket orderbook manager desync for BTCUSDT spot: last update ID 1337, first update ID 1339, update ID 1340: orderbook snapshot is outdated",
		func() {
			_ = m.handleDesync(t.Context(), cache, 1339, &orderbook.Update{
				Pair:     pair,
				Asset:    asset.Spot,
				UpdateID: 1340,
			}, 1337, ErrOrderbookSnapshotOutdated)
		},
		"handleDesync must panic with desync details when diagnostic mode is enabled")
}

func TestOrderbookDesyncReason(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		err    error
		reason string
	}{
		{
			name:   "snapshot_outdated",
			err:    ErrOrderbookSnapshotOutdated,
			reason: "snapshot_outdated",
		},
		{
			name:   "generic_desync",
			err:    errTestDesyncReason,
			reason: "desync",
		},
		{
			name:   "sequence_gap",
			reason: "sequence_gap",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.reason, orderbookDesyncReason(tc.err), "reason should match desync input")
		})
	}
}

func TestRecordResyncResult(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	tp.RecordMetrics = true
	m := NewUpdateManager(&tp)
	pair := currency.NewPair(currency.XRP, currency.USDT)
	timing := orderbookSyncTiming{
		started:         time.Now().Add(-time.Millisecond),
		permitWait:      time.Microsecond,
		fetchDelay:      2 * time.Microsecond,
		snapshotFetch:   3 * time.Microsecond,
		retryWait:       4 * time.Microsecond,
		snapshotLoad:    5 * time.Microsecond,
		cacheLockWait:   6 * time.Microsecond,
		pendingApply:    7 * time.Microsecond,
		fetchAttempts:   2,
		queuedUpdates:   3,
		snapshotID:      100,
		pendingFirstID:  101,
		pendingUpdateID: 103,
	}

	m.recordResyncResult(pair, asset.Spot, "succeeded", &timing)

	got := expvar.Get("gct_websocket_orderbook_resync_total").String()
	assert.Contains(t, got, `"exchange=TestExchange,asset=spot,pair=XRPUSDT,channel=orderbook,result=succeeded": 1`, "resync result metric should be recorded")
	summary := metrics.SnapshotOrderbookSyncSummary()
	for _, item := range summary.Items {
		if item.Exchange != "TestExchange" || item.Pair != pair.String() || item.Asset != asset.Spot.String() {
			continue
		}
		assert.Positive(t, item.ResyncTime, "resync time should be recorded")
		assert.Positive(t, item.ResyncTimed, "resync timed count should be recorded")
		assert.Equal(t, timing.permitWait, item.PermitWait)
		assert.Equal(t, timing.fetchDelay, item.FetchDelay)
		assert.Equal(t, timing.snapshotFetch, item.SnapshotFetch)
		assert.Equal(t, timing.retryWait, item.RetryWait)
		assert.Equal(t, timing.snapshotLoad, item.SnapshotLoad)
		assert.Equal(t, timing.cacheLockWait, item.CacheLockWait)
		assert.Equal(t, timing.pendingApply, item.PendingApply)
		assert.Equal(t, timing.fetchAttempts, item.FetchAttempts)
		assert.Equal(t, timing.queuedUpdates, item.QueuedUpdates)
		assert.Equal(t, timing.queuedUpdates, item.MaxQueuedUpdates)
		assert.Equal(t, timing.snapshotID, item.LastSnapshotUpdateID)
		assert.Equal(t, timing.pendingFirstID, item.LastPendingFirstUpdateID)
		assert.Equal(t, timing.pendingUpdateID, item.LastPendingUpdateID)
		return
	}
	require.Fail(t, "summary must include recorded resync result")
}

func TestSyncOrderbookRecordsResyncPhases(t *testing.T) {
	t.Parallel()

	pair, err := currency.NewPairFromStrings("RESYNCPHASE", "USDT")
	require.NoError(t, err)
	tp := newTestParams()
	tp.BufferInstance.exchangeName = "TestExchangeResyncPhases"
	tp.FetchDelay = time.Millisecond
	tp.FetchOrderbook = fetchOrderbookMock
	tp.RecordMetrics = true
	m := NewUpdateManager(&tp)
	cache := &updateCache{
		updates: []pendingUpdate{{
			firstUpdateID: 1337,
			update: &orderbook.Update{
				Pair:       pair,
				Asset:      asset.Spot,
				UpdateID:   1337,
				AllowEmpty: true,
				UpdateTime: time.Now(),
			},
		}},
		state: cacheStateQueuing,
	}

	require.NoError(t, m.syncOrderbook(t.Context(), cache, pair, asset.Spot, true))

	summary := metrics.SnapshotOrderbookSyncSummary()
	for _, item := range summary.Items {
		if item.Exchange != tp.BufferInstance.exchangeName || item.Pair != pair.String() {
			continue
		}
		assert.Equal(t, int64(1), item.ResyncSucceeded)
		assert.Equal(t, int64(1), item.ResyncTimed)
		assert.Positive(t, item.ResyncTime)
		assert.Positive(t, item.FetchDelay)
		assert.GreaterOrEqual(t, item.SnapshotFetch, time.Duration(0))
		assert.GreaterOrEqual(t, item.SnapshotLoad, time.Duration(0))
		assert.GreaterOrEqual(t, item.CacheLockWait, time.Duration(0))
		assert.GreaterOrEqual(t, item.PendingApply, time.Duration(0))
		assert.Equal(t, int64(1), item.FetchAttempts)
		assert.Equal(t, int64(1), item.QueuedUpdates)
		assert.Equal(t, int64(1), item.MaxQueuedUpdates)
		assert.Equal(t, int64(1336), item.LastSnapshotUpdateID)
		assert.Equal(t, int64(1337), item.LastPendingFirstUpdateID)
		assert.Equal(t, int64(1337), item.LastPendingUpdateID)
		return
	}
	require.Fail(t, "summary must include recorded resync phases")
}

func TestSyncOrderbookRecordsEmptySnapshot(t *testing.T) {
	t.Parallel()

	pair, err := currency.NewPairFromStrings("EMPTYSNAPSHOT", "USDT")
	require.NoError(t, err)
	book := &orderbook.Book{
		Exchange:          "TestExchangeEmptySnapshot",
		Pair:              pair,
		Asset:             asset.Spot,
		LastUpdated:       time.Now(),
		LastUpdateID:      1336,
		ValidateOrderbook: true,
	}
	tp := newTestParams()
	tp.BufferInstance.exchangeName = book.Exchange
	tp.FetchOrderbook = func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return book, nil
	}
	tp.RecordMetrics = true
	tp.AcceptSnapshotWhenUpdatesCovered = true
	m := NewUpdateManager(&tp)
	cache := &updateCache{
		updates: []pendingUpdate{{
			firstUpdateID: 1335,
			update: &orderbook.Update{
				Pair:       pair,
				Asset:      asset.Spot,
				UpdateID:   1336,
				AllowEmpty: true,
				UpdateTime: time.Now(),
			},
		}},
		state: cacheStateQueuing,
	}

	require.NoError(t, m.syncOrderbook(t.Context(), cache, pair, asset.Spot, false))
	assert.True(t, book.SuppressEmptyBookWarning, "empty snapshot warning should be represented by aggregate metrics")
	assert.Positive(t, metrics.SnapshotOrderbookSyncSummary().TotalEmptyOrderbooks)
}

func TestApplyUpdateInvalidateOnUpdateError(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.delay = time.Second
	pair, err := currency.NewPairFromStrings("APPLYERR", "SPOT")
	require.NoError(t, err)
	require.NoError(t, m.ob.LoadSnapshot(&orderbook.Book{
		Exchange:     m.ob.exchangeName,
		Pair:         pair,
		Asset:        asset.Spot,
		LastUpdated:  time.Now(),
		LastUpdateID: 1336,
	}), "LoadSnapshot must not error")

	cache, err := m.loadCache(pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cache.m.Lock()
	err = m.applyUpdate(ctx, cache, 1337, &orderbook.Update{Pair: pair, Asset: asset.Spot})
	cache.m.Unlock()
	require.NoError(t, err, "applyUpdate must not error when invalidating after update failure")

	cache.m.Lock()
	require.Equal(t, cacheStateQueuing, cache.state, "cache must enter queuing state after invalidation")
	require.NotEmpty(t, cache.updates, "cache must queue the update before sync starts")
	cache.m.Unlock()

	cancel()

	require.Eventually(t, func() bool {
		cache.m.Lock()
		defer cache.m.Unlock()
		return cache.state == cacheStateQueuing && len(cache.updates) == 0
	}, time.Second, time.Millisecond*50, "cache must preserve queuing state and clear pending updates after cancellation")
}

func TestInitialiseOrderbookCache(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.delay = time.Second
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	update := &orderbook.Update{
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	}
	cache := &updateCache{}
	m.initialiseOrderbookCache(ctx, 1337, update, cache, false)
	cache.m.Lock()
	require.Equal(t, cacheStateQueuing, cache.state, "state must be queuing")
	require.NotEmpty(t, cache.updates, "updates must have queued update")
	cache.m.Unlock()

	eventuallyCondition := func() bool {
		cache.m.Lock()
		defer cache.m.Unlock()
		return cache.state == cacheStateQueuing && len(cache.updates) == 0
	}
	require.Eventually(t, eventuallyCondition, time.Second, time.Millisecond*50, "state must be queuing and updates cleared after syncOrderbook completes when it fails on context cancellation")
}

func TestInvalidateCache(t *testing.T) {
	t.Parallel()
	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.delay = time.Second
	cache, err := m.loadCache(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "loadCache must not error")

	cache.m.Lock()
	cache.state = cacheStateSynced
	cache.m.Unlock()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cache.m.Lock()
	err = m.invalidateCache(ctx, 1337, &orderbook.Update{
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	}, cache, "test")
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "invalidateCache must error but still trigger syncOrderbook")

	require.Equal(t, cacheStateQueuing, cache.state, "state must be uninitialised after invalidateCache")
	require.NotEmpty(t, cache.updates, "updates must not be empty after invalidateCache")
	cache.m.Unlock()

	cancel()

	eventuallyCondition := func() bool {
		cache.m.Lock()
		defer cache.m.Unlock()
		return cache.state == cacheStateQueuing && len(cache.updates) == 0
	}
	require.Eventually(t, eventuallyCondition, time.Second, time.Millisecond*50, "state must be queuing and updates cleared after syncOrderbook completes when it fails on context cancellation")
}

func TestSyncOrderbook(t *testing.T) {
	t.Parallel()

	cache := &updateCache{}
	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair := currency.NewPair(currency.ETH, currency.USDT)

	ctxCancel, cancel := context.WithCancel(t.Context())
	cancel()
	m.delay = time.Millisecond * 10
	err := m.syncOrderbook(ctxCancel, cache, pair, asset.Spot, false)
	require.ErrorIs(t, err, context.Canceled, "must error due to context cancellation on select case")

	m.fetchOrderbook = fetchOrderbookNotFoundError
	err = m.syncOrderbook(t.Context(), cache, currency.NewBTCUSD(), asset.Spot, false)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "must error due to depth not found when calling fetch orderbook")

	m.deadline = time.Millisecond * 10
	m.fetchOrderbook = fetchOrderbookFailure
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.Spot, false)
	require.ErrorIs(t, err, context.DeadlineExceeded, "must error due to deadline exceeded when waiting for update")

	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1337}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.Spot, false)
	require.ErrorIs(t, err, orderbook.ErrLastUpdatedNotSet, "must error due to orderbook invalid when loading snapshot")

	m.fetchOrderbook = fetchOrderbookMock
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.USDTMarginedFutures, false)
	require.NoError(t, err)
}

func TestSyncOrderbookSnapshotSyncLimit(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSDT()
	tp := newTestParams()
	tp.SnapshotSyncLimit = 1
	m := NewUpdateManager(&tp)
	m.syncLimit <- struct{}{}
	cache := &updateCache{
		updates: []pendingUpdate{{
			update: &orderbook.Update{
				Pair:       pair,
				Asset:      asset.Spot,
				UpdateID:   1337,
				AllowEmpty: true,
				UpdateTime: time.Now(),
			},
		}},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := m.syncOrderbook(ctx, cache, pair, asset.Spot, false)
	require.ErrorIs(t,
		err,
		context.Canceled,
		"syncOrderbook must return cancellation while waiting for a sync slot")
	assert.Empty(t,
		cache.updates,
		"pending updates should be cleared after cancellation while waiting for a sync slot")
}

func TestSyncOrderbookApplyPendingUpdatesFailure(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.fetchOrderbook = fetchOrderbookMock
	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return true, nil
	}
	pair, err := currency.NewPairFromStrings("SYNCFAIL", "BTC")
	require.NoError(t, err)
	cache := &updateCache{
		updates: []pendingUpdate{{
			firstUpdateID: 1337,
			update:        &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()},
		}},
		state: cacheStateQueuing,
	}

	err = m.syncOrderbook(t.Context(), cache, pair, asset.USDTMarginedFutures, false)
	require.ErrorIs(t, err, errPendingUpdatesNotApplied, "syncOrderbook must surface applyPendingUpdates errors")
	assert.Equal(t, cacheStateInitialised, cache.state, "syncOrderbook should reset cache state on pending update failures")
	assert.Empty(t, cache.updates, "syncOrderbook should clear pending updates after failure")
}

func TestSyncOrderbookAcceptSnapshotWhenUpdatesCovered(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		firstUpdateID    int64
		updateID         int64
		expectedUpdateID int64
	}{
		{
			name:             "snapshot_covers_pending_update",
			firstUpdateID:    1335,
			updateID:         1336,
			expectedUpdateID: 1336,
		},
		{
			name:             "newer_pending_update_is_applied",
			firstUpdateID:    1337,
			updateID:         1337,
			expectedUpdateID: 1337,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tp := newTestParams()
			tp.AcceptSnapshotWhenUpdatesCovered = true
			tp.FetchOrderbook = fetchOrderbookMock
			tp.CheckPendingUpdate = func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (bool, error) {
				if firstUpdateID > lastUpdateID+1 {
					return false, ErrOrderbookSnapshotOutdated
				}
				return update.UpdateID <= lastUpdateID, nil
			}
			m := NewUpdateManager(&tp)
			pair, err := currency.NewPairFromStrings(tc.name, "USDT")
			require.NoError(t, err, "currency pair must be created")
			cache := &updateCache{
				updates: []pendingUpdate{{
					firstUpdateID: tc.firstUpdateID,
					update: &orderbook.Update{
						Pair:       pair,
						Asset:      asset.Spot,
						UpdateID:   tc.updateID,
						AllowEmpty: true,
						UpdateTime: time.Now(),
					},
				}},
				state: cacheStateQueuing,
			}

			err = m.syncOrderbook(t.Context(), cache, pair, asset.Spot, false)
			require.NoError(t, err, "syncOrderbook must accept a usable snapshot")
			assert.Equal(t, cacheStateSynced, cache.state, "cache should be synced")
			assert.Empty(t, cache.updates, "pending updates should be cleared")
			lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
			require.NoError(t, err, "LastUpdateID must return the final sequence")
			assert.Equal(t, tc.expectedUpdateID, lastUpdateID, "final sequence should match expected state")
		})
	}
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair := currency.NewPair(currency.LTC, currency.USDT)

	err := m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "must error due to depth not found when calling fetch orderbook")

	err = m.ob.LoadSnapshot(&orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now()})
	require.NoError(t, err)

	expectedErr := errors.New("test error")
	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, expectedErr
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, expectedErr, "must error due to checkPendingUpdate returning an error")

	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return true, nil
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, errPendingUpdatesNotApplied, "must error due to pending updates not applied when skipped")

	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "must error due to orderbook invalid when update application fails")

	err = m.ob.LoadSnapshot(&orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now()})
	require.NoError(t, err)

	cache := &updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot, AllowEmpty: true, UpdateTime: time.Now()}},
	}}
	err = m.applyPendingUpdates(cache)
	require.NoError(t, err, "must not error when update application succeeds")
	assert.Equal(t, cacheStateSynced, cache.state, "state should be synced after successful application of pending updates")

	pair, err = currency.NewPairFromStrings("PENDSEQ", "USDT")
	require.NoError(t, err)
	err = m.ob.LoadSnapshot(&orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 1336})
	require.NoError(t, err)

	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{firstUpdateID: 1337, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()}},
		{firstUpdateID: 1339, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1339, AllowEmpty: true, UpdateTime: time.Now()}},
	}})
	require.ErrorIs(t, err, ErrOrderbookSnapshotOutdated, "must error when a later pending update is out of sequence")
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

func TestClearWithLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}}
	cache.clearWithLock()
	require.Empty(t, cache.updates)
}

func TestClearPendingUpdatesWithLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{
		updates: []pendingUpdate{{update: &orderbook.Update{}}},
		state:   cacheStateQueuing,
	}
	cache.clearPendingUpdatesWithLock()
	require.Empty(t, cache.updates)
	assert.Equal(t, cacheStateQueuing, cache.state)
}

func TestClearNoLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}}
	cache.clearNoLock()
	require.Empty(t, cache.updates)
}

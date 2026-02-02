package buffer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var testParams = UpdateParams{
	FetchDeadline:  time.Second,
	FetchOrderbook: func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) { return nil, nil },
	CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	},
	BufferInstance: &Orderbook{exchangeName: "TestExchange", ob: make(map[key.PairAsset]*orderbookHolder), dataHandler: stream.NewRelay(1000), verbose: true},
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

	params := &UpdateParams{FetchDelay: -time.Second, FetchDeadline: -time.Second}
	require.Panics(t, func() { NewUpdateManager(params) })
	params.FetchDeadline = time.Second
	require.Panics(t, func() { NewUpdateManager(params) })
	params.FetchDelay = time.Second
	require.Panics(t, func() { NewUpdateManager(params) })

	params.FetchOrderbook = func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) { return nil, nil }
	params.CheckPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	}
	params.BufferInstance = &Orderbook{}
	got := NewUpdateManager(params)
	require.NotNil(t, got)
	assert.NotNil(t, got.lookup)
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	tp := testParams
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

	m := NewUpdateManager(&testParams)
	_, err := m.loadCache(currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = m.loadCache(currency.NewBTCUSDT(), 1336)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	cache, err := m.loadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.NotNil(t, cache)
	assert.Len(t, m.lookup, 1)
	cache.m.Lock()
	assert.Equal(t, cacheStateInitialised, cache.state, "state should be initialised after first load")
	cache.m.Unlock()

	cache2, err := m.loadCache(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")
	assert.Equal(t, cache, cache2, "should be the same cache instance")
}

func TestApplyUpdate(t *testing.T) {
	t.Parallel()

	m := NewUpdateManager(&testParams)
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
}

func TestInitialiseOrderbookCache(t *testing.T) {
	t.Parallel()

	m := NewUpdateManager(&testParams)
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
	m.initialiseOrderbookCache(ctx, 1337, update, cache)
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
	m := NewUpdateManager(&testParams)
	m.delay = time.Second
	cache, err := m.loadCache(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "loadCache must not error")

	cache.m.Lock()
	cache.state = cacheStateSynced
	cache.m.Unlock()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = m.invalidateCache(ctx, 1337, &orderbook.Update{
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateID:   1338,
		UpdateTime: time.Now(),
	}, cache)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "invalidateCache must error but still trigger syncOrderbook")

	cache.m.Lock()
	require.Equal(t, cacheStateQueuing, cache.state, "state must be uninitialised after invalidateCache")
	require.NotEmpty(t, cache.updates, "updates must not be empty after invalidateCache")
	cache.m.Unlock()

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
	m := NewUpdateManager(&testParams)
	pair := currency.NewPair(currency.ETH, currency.USDT)

	ctxCancel, cancel := context.WithCancel(t.Context())
	cancel()
	m.delay = time.Millisecond * 10
	err := m.syncOrderbook(ctxCancel, cache, pair, asset.Spot)
	require.ErrorIs(t, err, context.Canceled, "must error due to context cancellation on select case")

	m.fetchOrderbook = fetchOrderbookNotFoundError
	err = m.syncOrderbook(t.Context(), cache, currency.NewBTCUSD(), asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "must error due to depth not found when calling fetch orderbook")

	m.deadline = time.Millisecond * 10
	m.fetchOrderbook = fetchOrderbookFailure
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.Spot)
	require.ErrorIs(t, err, context.DeadlineExceeded, "must error due to deadline exceeded when waiting for update")

	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1337}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrLastUpdatedNotSet, "must error due to orderbook invalid when loading snapshot")

	m.fetchOrderbook = fetchOrderbookMock
	cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: asset.USDTMarginedFutures, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()}}}
	err = m.syncOrderbook(t.Context(), cache, pair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	m := NewUpdateManager(&testParams)
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

func TestClearNoLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}}
	cache.clearNoLock()
	require.Empty(t, cache.updates)
}

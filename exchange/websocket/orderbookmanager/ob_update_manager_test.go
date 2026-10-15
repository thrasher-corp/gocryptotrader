package orderbookmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

func newTestParams() UpdateManagerParams {
	return UpdateManagerParams{
		FetchDeadline:  time.Second,
		FetchOrderbook: fetchOrderbookNotFoundError,
		CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (bool, error) {
			return false, nil
		},
		Orderbook: &Orderbook{exchangeName: "TestExchange", ob: make(map[key.PairAsset]*orderbook.Depth), dataHandler: stream.NewRelay(1000), verbose: true},
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
	require.Panics(t, func() { NewUpdateManager(params) })

	params.FetchOrderbook = func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) { return nil, nil }
	params.CheckPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	}
	params.Orderbook = &Orderbook{}
	got := NewUpdateManager(params)
	require.NotNil(t, got)
	assert.NotNil(t, got.lookup)
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
		id, err := tp.Orderbook.LastUpdateID(pair, asset.USDTMarginedFutures)
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

func TestQueuedUpdatesPreserveCallerContext(t *testing.T) {
	type contextKey struct{}
	contextMarker := &contextKey{}
	common.RegisterContextKey(contextMarker)

	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})
	var fetchOnce sync.Once
	params := newTestParams()
	params.FetchOrderbook = func(ctx context.Context, pair currency.Pair, a asset.Item) (*orderbook.Book, error) {
		fetchOnce.Do(func() { close(fetchStarted) })
		select {
		case <-releaseFetch:
			book, err := fetchOrderbookMock(ctx, pair, a)
			if err != nil {
				return nil, err
			}
			book.LastUpdated = time.Now().Add(-time.Minute)
			return book, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	manager := NewUpdateManager(&params)
	newUpdate := func(id int64) *orderbook.Update {
		return &orderbook.Update{
			UpdateID:   id,
			Pair:       currency.NewBTCUSDT(),
			Asset:      asset.USDTMarginedFutures,
			AllowEmpty: true,
			UpdateTime: time.Now(),
		}
	}
	callerContext := func(value string) context.Context {
		return context.WithValue(t.Context(), contextMarker, value)
	}

	require.NoError(t, manager.ProcessOrderbookUpdate(callerContext("first"), 1337, newUpdate(1337)), "first update must start synchronisation")
	select {
	case <-fetchStarted:
	case <-time.After(time.Second):
		require.FailNow(t, "snapshot fetch must start")
	}
	require.NoError(t, manager.ProcessOrderbookUpdate(callerContext("second"), 1338, newUpdate(1338)), "second update must queue during synchronisation")
	close(releaseFetch)

	for _, want := range []string{"first", "first", "second"} {
		select {
		case payload := <-params.Orderbook.dataHandler.C:
			assert.Equal(t, want, payload.Ctx[contextMarker], "relayed message should retain its originating context")
		case <-time.After(time.Second):
			require.FailNow(t, "snapshot and queued updates must be relayed")
		}
	}

	require.NoError(t, manager.ProcessOrderbookUpdate(callerContext("third"), 1339, newUpdate(1339)), "third update must apply after synchronisation")
	select {
	case payload := <-params.Orderbook.dataHandler.C:
		assert.Equal(t, "third", payload.Ctx[contextMarker], "direct update should retain its originating context")
	case <-time.After(time.Second):
		require.FailNow(t, "direct update must be relayed")
	}
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
}

func TestApplyUpdateInvalidateOnUpdateError(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	m.delay = time.Second
	pair, err := currency.NewPairFromStrings("APPLYERR", "SPOT")
	require.NoError(t, err)
	require.NoError(t, m.ob.LoadSnapshot(t.Context(), &orderbook.Book{
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
	cache.m.Lock()
	m.initialiseOrderbookCache(ctx, 1337, update, cache)
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
	}, cache, 1336)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "invalidateCache must error but still trigger syncOrderbook")
	assert.Equal(t, int64(1336), cache.withdrawnUpdateID, "invalidateCache should keep the update ID the book was published to")

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

	err = m.syncOrderbook(t.Context(), cache, pair, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errPendingUpdatesNotApplied, "syncOrderbook must surface applyPendingUpdates errors")
	assert.Equal(t, cacheStateInitialised, cache.state, "syncOrderbook should reset cache state on pending update failures")
	assert.Empty(t, cache.updates, "syncOrderbook should clear pending updates after failure")
	_, err = m.ob.LastUpdateID(pair, asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "syncOrderbook should invalidate the book after pending update failures")
}

func TestSyncOrderbookWithoutFollowingUpdate(t *testing.T) {
	t.Parallel()

	// A snapshot newer than every queued update is loaded at once rather than withheld until an update follows it
	tp := newTestParams()
	tp.LoadSnapshotsAtOnce = true
	m := NewUpdateManager(&tp)
	m.fetchOrderbook = fetchOrderbookMock
	m.checkPendingUpdate = func(lastUpdateID, _ int64, update *orderbook.Update) (bool, error) {
		return update.UpdateID <= lastUpdateID, nil
	}
	pair, err := currency.NewPairFromStrings("SYNCQUIET", "BTC")
	require.NoError(t, err)
	cache := &updateCache{
		updates: []pendingUpdate{{
			firstUpdateID: 1300,
			update:        &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1300, AllowEmpty: true, UpdateTime: time.Now()},
		}},
		state: cacheStateQueuing,
	}
	require.NoError(t, m.syncOrderbook(t.Context(), cache, pair, asset.Spot), "syncOrderbook must load a snapshot no queued update follows")
	assert.Equal(t, cacheStateSnapshotLoaded, cache.state, "syncOrderbook should await the update following the snapshot")
	assert.Empty(t, cache.updates, "syncOrderbook should clear the queued updates")
	lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error for the loaded snapshot")
	assert.Equal(t, int64(1336), lastUpdateID, "syncOrderbook should load the snapshot")
}

func TestApplyFirstUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		base          string // the order book store is shared by every test, so each case has a pair of its own
		check         func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (bool, error)
		update        orderbook.Update
		state         cacheState
		lastUpdateID  int64
		queuedUpdates int
	}{
		{
			name: "held by the snapshot",
			base: "FIRSTHELD",
			check: func(lastUpdateID, _ int64, update *orderbook.Update) (bool, error) {
				return update.UpdateID <= lastUpdateID, nil
			},
			update:       orderbook.Update{UpdateID: 1336, AllowEmpty: true},
			state:        cacheStateSnapshotLoaded,
			lastUpdateID: 1336,
		},
		{
			name:         "following the snapshot",
			base:         "FIRSTNEXT",
			check:        func(int64, int64, *orderbook.Update) (bool, error) { return false, nil },
			update:       orderbook.Update{UpdateID: 1340, AllowEmpty: true},
			state:        cacheStateSynced,
			lastUpdateID: 1340,
		},
		{
			name:          "after a gap",
			base:          "FIRSTGAP",
			check:         func(int64, int64, *orderbook.Update) (bool, error) { return false, ErrOrderbookSnapshotOutdated },
			update:        orderbook.Update{UpdateID: 1340, AllowEmpty: true},
			state:         cacheStateQueuing,
			queuedUpdates: 1,
		},
		{
			name:          "failing to apply",
			base:          "FIRSTFAIL",
			check:         func(int64, int64, *orderbook.Update) (bool, error) { return false, nil },
			update:        orderbook.Update{UpdateID: 1340},
			state:         cacheStateQueuing,
			queuedUpdates: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tp := newTestParams()
			tp.LoadSnapshotsAtOnce = true
			m := NewUpdateManager(&tp)
			m.delay = time.Hour
			m.checkPendingUpdate = tc.check
			pair, err := currency.NewPairFromStrings(tc.base, "USDT")
			require.NoError(t, err, "NewPairFromStrings must not error")
			require.NoError(t, m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 1336}), "LoadSnapshot must not error")
			cache, err := m.loadCache(pair, asset.Spot)
			require.NoError(t, err, "loadCache must not error")
			cache.m.Lock()
			cache.state = cacheStateSnapshotLoaded
			cache.m.Unlock()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			update := tc.update
			update.Pair, update.Asset, update.UpdateTime = pair, asset.Spot, time.Now()
			require.NoError(t, m.ProcessOrderbookUpdate(ctx, 1337, &update), "ProcessOrderbookUpdate must not error")
			cache.m.Lock()
			assert.Equal(t, tc.state, cache.state, "ProcessOrderbookUpdate should leave the expected state")
			assert.Len(t, cache.updates, tc.queuedUpdates, "ProcessOrderbookUpdate should queue the update for a new snapshot only after invalidating")
			cache.m.Unlock()
			lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
			if tc.lastUpdateID == 0 {
				assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "ProcessOrderbookUpdate should invalidate a book the update does not follow")
				return
			}
			require.NoError(t, err, "LastUpdateID must not error")
			assert.Equal(t, tc.lastUpdateID, lastUpdateID, "ProcessOrderbookUpdate should apply only an update following the snapshot")
		})
	}
}

func TestQuietBookLoads(t *testing.T) {
	t.Parallel()

	// One update, then a snapshot newer than it, then nothing for a while: the book loads from the snapshot, and the next
	// update applies to it
	tp := newTestParams()
	tp.Orderbook.exchangeName = uniqueTestExchange(t)
	tp.FetchOrderbook = func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		book, err := fetchOrderbookMock(ctx, p, a)
		book.Exchange = tp.Orderbook.exchangeName
		return book, err
	}
	tp.CheckPendingUpdate = checkSequencedUpdate
	tp.LoadSnapshotsAtOnce = true
	m := NewUpdateManager(&tp)
	pair, err := currency.NewPairFromStrings("QUIET", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	newUpdate := func(id int64) *orderbook.Update {
		return &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: id, AllowEmpty: true, UpdateTime: time.Now()}
	}
	require.NoError(t, m.ProcessOrderbookUpdate(t.Context(), 1300, newUpdate(1300)), "ProcessOrderbookUpdate must not error")
	cache, err := m.loadCache(pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")
	require.Eventually(t, func() bool {
		cache.m.Lock()
		defer cache.m.Unlock()
		return cache.state == cacheStateSnapshotLoaded
	}, 5*time.Second, 10*time.Millisecond, "the synchronisation must finish without a later update")
	lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(1336), lastUpdateID, "the snapshot should load without a later update")
	require.NoError(t, m.ProcessOrderbookUpdate(t.Context(), 1330, newUpdate(1337)), "ProcessOrderbookUpdate must not error")
	lastUpdateID, err = m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(1337), lastUpdateID, "the update following the snapshot should apply")
	require.NoError(t, m.ProcessOrderbookUpdate(t.Context(), 1338, newUpdate(1338)), "ProcessOrderbookUpdate must not error")
	lastUpdateID, err = m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(1338), lastUpdateID, "later updates should apply in sequence")
}

func TestSyncAfterStaleUpdate(t *testing.T) {
	t.Parallel()

	// An update the book already holds invalidates it, and the REST snapshot fetched to synchronise it again can lag the
	// published book, whose later updates are no longer queued
	for _, tc := range []struct {
		name     string
		base     string // the order book store is shared by every test, so each case has a pair of its own
		stale    int64  // the update ID of the update the book already holds
		snapshot int64
		loaded   int64 // the book's update ID after the synchronisation, zero when it stays invalid
	}{
		{name: "older than the published book", base: "ROLLBACK", stale: 1330, snapshot: 1336},
		{name: "brought level by the queued update", base: "BRIDGED", stale: 1337, snapshot: 1336, loaded: 1337},
		{name: "as new as the published book", base: "CAUGHTUP", stale: 1330, snapshot: 1337, loaded: 1337},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newSyncFixture(t, tc.base)
			f.publish(t)
			f.process(t, 1337, 1337)
			f.process(t, tc.stale, tc.stale)
			f.synchronise(t, tc.snapshot)
			f.requireLastUpdateID(t, tc.loaded)
		})
	}
}

func TestSyncKeepsWithdrawnBook(t *testing.T) {
	t.Parallel()

	// A published book's update ID outlasts synchronisations that fail, so a lagging snapshot cannot roll it back later
	t.Run("stale updates in turn", func(t *testing.T) {
		t.Parallel()
		f := newSyncFixture(t, "STALETWICE")
		f.publish(t)
		f.process(t, 1337, 1337)
		for _, stale := range []int64{1330, 1331} {
			f.process(t, stale, stale)
			published := len(f.m.ob.dataHandler.C)
			f.synchronise(t, 1336)
			f.requireLastUpdateID(t, 0)
			assert.Equal(t, published, len(f.m.ob.dataHandler.C), "a rejected snapshot should never be published")
		}
		// An update that follows its snapshot shows the stream is at that level, which is how a stream whose update IDs
		// restart synchronises again
		f.process(t, 7, 7)
		f.synchronise(t, 6)
		f.requireLastUpdateID(t, 7)
		// Synchronised again, the book is held to its new update ID alone
		f.process(t, 20, 20)
		f.synchronise(t, 21)
		f.requireLastUpdateID(t, 21)
		assert.Equal(t, int32(4), f.fetches.Load(), "each synchronisation should fetch one snapshot")
	})

	t.Run("update failing to apply", func(t *testing.T) {
		t.Parallel()
		// A failed update invalidates the book, and its update ID with it, before the manager invalidates it
		f := newSyncFixture(t, "FAILAPPLY")
		f.publish(t)
		f.process(t, 1337, 1337)
		f.processFailing(t, 1338, 1338)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
		f.process(t, 1330, 1330)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
	})

	t.Run("update ending at the snapshot", func(t *testing.T) {
		t.Parallel()
		// Binance's derivatives streams apply an update whose final update ID is the snapshot's, which does not take
		// the book past the snapshot
		f := newSyncFixture(t, "EQUALUPDATE")
		f.m.checkPendingUpdate = func(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (bool, error) {
			if update.UpdateID < lastUpdateID {
				return true, nil
			}
			if firstUpdateID > lastUpdateID+1 {
				return false, ErrOrderbookSnapshotOutdated
			}
			return false, nil
		}
		f.publish(t)
		f.process(t, 1337, 1337)
		f.process(t, 1336, 1336)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
	})

	t.Run("update published before its delivery failed", func(t *testing.T) {
		t.Parallel()
		// A full data handler fails an update the book has already published
		f := newSyncFixture(t, "RELAYFULL")
		f.useRelay()
		f.publish(t)
		f.drainRelay()
		f.process(t, 1337, 1337)
		f.process(t, 1338, 1338)
		f.drainRelay()
		f.synchronise(t, 1330)
		f.requireLastUpdateID(t, 0)
		f.drainRelay()
		f.process(t, 1335, 1335)
		f.synchronise(t, 1337)
		f.requireLastUpdateID(t, 0)
	})

	t.Run("replayed update published before its delivery failed", func(t *testing.T) {
		t.Parallel()
		f := newSyncFixture(t, "RELAYREPLAY")
		f.useRelay()
		f.process(t, 1337, 1337)
		f.process(t, 1338, 1338)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
		f.drainRelay()
		f.process(t, 1330, 1330)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
	})

	t.Run("snapshot published before its delivery failed", func(t *testing.T) {
		t.Parallel()
		// The book stays published at the snapshot, which a later snapshot must not roll back
		f := newSyncFixture(t, "SNAPFULL")
		f.useRelay()
		f.fillRelay(t)
		f.process(t, 1300, 1300)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 1336)
		f.drainRelay()
		f.process(t, 1301, 1301)
		f.synchronise(t, 1330)
		f.requireLastUpdateID(t, 1336)
	})

	t.Run("book published by REST before the stream synchronised", func(t *testing.T) {
		t.Parallel()
		// A REST request publishes to the exchange's order book store, before this manager has loaded a book of its own
		f := newSyncFixture(t, "RESTBOOK")
		rest := &orderbook.Book{Pair: f.pair, Asset: asset.Spot, Exchange: f.m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 1340}
		require.NoError(t, rest.Process(), "Process must not error")
		f.process(t, 1335, 1335)
		f.synchronise(t, 1338)
		depth, err := orderbook.GetDepth(f.m.ob.exchangeName, f.pair, asset.Spot)
		require.NoError(t, err, "GetDepth must not error")
		lastUpdateID, err := depth.LastUpdateID()
		require.NoError(t, err, "LastUpdateID must not error for the REST book")
		assert.Equal(t, int64(1340), lastUpdateID, "the older snapshot should not replace the REST book")
	})

	t.Run("synchronisation failing after it published updates", func(t *testing.T) {
		t.Parallel()
		f := newSyncFixture(t, "PARTSYNC")
		f.process(t, 1337, 1337)
		f.process(t, 1340, 1340)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
		f.process(t, 1330, 1330)
		f.synchronise(t, 1336)
		f.requireLastUpdateID(t, 0)
		f.process(t, 1341, 1341)
		f.synchronise(t, 1341)
		f.requireLastUpdateID(t, 1341)
	})
}

// syncFixture drives an update manager whose snapshots are the update IDs sent to it, for a pair of its own, as the
// order book store is shared by every test
type syncFixture struct {
	m         *UpdateManager
	pair      currency.Pair
	cache     *updateCache
	snapshots chan int64
	fetches   atomic.Int32
}

func newSyncFixture(t *testing.T, base string) *syncFixture {
	t.Helper()
	f := &syncFixture{snapshots: make(chan int64)}
	tp := newTestParams()
	tp.Orderbook.exchangeName = uniqueTestExchange(t)
	tp.FetchOrderbook = func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		f.fetches.Add(1)
		select {
		case id := <-f.snapshots:
			return &orderbook.Book{Exchange: tp.Orderbook.exchangeName, Pair: p, Asset: a, LastUpdated: time.Now(), LastUpdateID: id}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	tp.CheckPendingUpdate = checkSequencedUpdate
	tp.LoadSnapshotsAtOnce = true
	f.m = NewUpdateManager(&tp)
	var err error
	f.pair, err = currency.NewPairFromStrings(base, "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	f.cache, err = f.m.loadCache(f.pair, asset.Spot)
	require.NoError(t, err, "loadCache must not error")
	return f
}

// publish loads a snapshot at update ID 1336 as the synchronised book a synchronisation leaves
func (f *syncFixture) publish(t *testing.T) {
	t.Helper()
	require.NoError(t, f.m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: f.pair, Asset: asset.Spot, Exchange: f.m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 1336}), "LoadSnapshot must not error")
	f.cache.m.Lock()
	f.cache.state = cacheStateSynced
	f.cache.m.Unlock()
}

func (f *syncFixture) process(t *testing.T, firstUpdateID, updateID int64) {
	t.Helper()
	update := &orderbook.Update{Pair: f.pair, Asset: asset.Spot, UpdateID: updateID, AllowEmpty: true, UpdateTime: time.Now()}
	require.NoError(t, f.m.ProcessOrderbookUpdate(t.Context(), firstUpdateID, update), "ProcessOrderbookUpdate must not error")
}

// useRelay gives the order books a data handler holding one delivery, so a test can fill it; set before any update
func (f *syncFixture) useRelay() {
	f.m.ob.dataHandler = stream.NewRelay(1)
}

// fillRelay fills the data handler, so the next delivery to it fails
func (f *syncFixture) fillRelay(t *testing.T) {
	t.Helper()
	require.NoError(t, f.m.ob.dataHandler.Send(t.Context(), nil), "Send must not error")
}

// drainRelay empties the data handler
func (f *syncFixture) drainRelay() {
	for len(f.m.ob.dataHandler.C) > 0 {
		<-f.m.ob.dataHandler.C
	}
}

// processFailing passes an update the book rejects, which invalidates it
func (f *syncFixture) processFailing(t *testing.T, firstUpdateID, updateID int64) {
	t.Helper()
	update := &orderbook.Update{Pair: f.pair, Asset: asset.Spot, UpdateID: updateID, UpdateTime: time.Now()}
	require.NoError(t, f.m.ProcessOrderbookUpdate(t.Context(), firstUpdateID, update), "ProcessOrderbookUpdate must not error")
}

// synchronise serves the synchronisation in progress its snapshot and waits for it to finish
func (f *syncFixture) synchronise(t *testing.T, snapshot int64) {
	t.Helper()
	select {
	case f.snapshots <- snapshot:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the synchronisation must request a snapshot")
	}
	require.Eventually(t, func() bool {
		f.cache.m.Lock()
		defer f.cache.m.Unlock()
		return f.cache.state != cacheStateQueuing
	}, 5*time.Second, 10*time.Millisecond, "the synchronisation must finish")
}

// requireLastUpdateID checks the book's update ID, or that it is invalid when exp is zero
func (f *syncFixture) requireLastUpdateID(t *testing.T, exp int64) {
	t.Helper()
	lastUpdateID, err := f.m.ob.LastUpdateID(f.pair, asset.Spot)
	if exp == 0 {
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the book must stay invalid")
		return
	}
	require.NoError(t, err, "LastUpdateID must not error for a synchronised book")
	require.Equal(t, exp, lastUpdateID, "the book must be synchronised to the expected update")
}

var testExchangeRuns atomic.Int64

// uniqueTestExchange returns an exchange name no other test run uses: order books persist in a store every test and
// every -count iteration shares, and a run's book is a floor for the next run's snapshots
func uniqueTestExchange(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s%d", t.Name(), testExchangeRuns.Add(1))
}

// checkSequencedUpdate skips an update the book holds and rejects one that does not follow it, as sequenced streams do
func checkSequencedUpdate(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (bool, error) {
	if update.UpdateID <= lastUpdateID {
		return true, nil
	}
	if firstUpdateID > lastUpdateID+1 {
		return false, ErrOrderbookSnapshotOutdated
	}
	return false, nil
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair := currency.NewPair(currency.LTC, currency.USDT)

	err := m.applyPendingUpdates(&updateCache{})
	require.ErrorIs(t, err, errUpdatesNotSupplied, "applyPendingUpdates must error when the pending-update queue is empty")

	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "applyPendingUpdates must error when the currency pair is empty")

	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair}},
	}})
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "applyPendingUpdates must error when the asset is invalid")

	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "applyPendingUpdates must error when the orderbook depth is not found")

	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now()})
	require.NoError(t, err, "LoadSnapshot must not error")

	expectedErr := errors.New("test error")
	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, expectedErr
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, expectedErr, "applyPendingUpdates must return the pending-update check error")

	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return true, nil
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, errPendingUpdatesNotApplied, "applyPendingUpdates must error when every pending update is skipped")
	m.loadSnapshotsAtOnce = true
	skipped := &updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}}
	require.NoError(t, m.applyPendingUpdates(skipped), "applyPendingUpdates must not error when every pending update is skipped and snapshots load at once")
	assert.Equal(t, cacheStateSnapshotLoaded, skipped.state, "applyPendingUpdates should await the update following the snapshot when snapshots load at once")
	m.loadSnapshotsAtOnce = false

	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		return false, nil
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot}},
	}})
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "applyPendingUpdates must error when update application invalidates the orderbook")

	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now()})
	require.NoError(t, err, "LoadSnapshot must not error after orderbook invalidation")

	cache := &updateCache{updates: []pendingUpdate{
		{update: &orderbook.Update{Pair: pair, Asset: asset.Spot, AllowEmpty: true, UpdateTime: time.Now()}},
	}}
	err = m.applyPendingUpdates(cache)
	require.NoError(t, err, "applyPendingUpdates must not error when update application succeeds")
	assert.Equal(t, cacheStateSynced, cache.state, "applyPendingUpdates should set the cache state to synced")

	pair, err = currency.NewPairFromStrings("PENDSEQ", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 1336})
	require.NoError(t, err, "LoadSnapshot must not error for the pending-sequence orderbook")

	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{firstUpdateID: 1337, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1337, AllowEmpty: true, UpdateTime: time.Now()}},
		{firstUpdateID: 1339, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 1339, AllowEmpty: true, UpdateTime: time.Now()}},
	}})
	require.ErrorIs(t, err, ErrOrderbookSnapshotOutdated, "applyPendingUpdates must error when a later pending update is out of sequence")
}

func TestApplyPendingUpdatesKeyMismatch(t *testing.T) {
	t.Parallel()

	pairMismatchKey, err := currency.NewPairFromStrings("PENDKEYPAIR", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error for the pair-mismatch pending-update key")
	assetMismatchKey, err := currency.NewPairFromStrings("PENDKEYASSET", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error for the asset-mismatch pending-update key")
	otherPair, err := currency.NewPairFromStrings("PENDOTHER", "BTC")
	require.NoError(t, err, "NewPairFromStrings must not error for the mismatched pending-update key")

	for _, tc := range []struct {
		name        string
		pendingPair currency.Pair
		updatePair  currency.Pair
		updateAsset asset.Item
		wantContext string
	}{
		{name: "pair", pendingPair: pairMismatchKey, updatePair: otherPair, updateAsset: asset.Spot, wantContext: otherPair.String()},
		{name: "asset", pendingPair: assetMismatchKey, updatePair: assetMismatchKey, updateAsset: asset.Futures, wantContext: asset.Futures.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tp := newTestParams()
			m := NewUpdateManager(&tp)
			err := m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: tc.pendingPair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 10})
			require.NoError(t, err, "LoadSnapshot must not error for the pending-update key")
			<-m.ob.dataHandler.C
			cache := &updateCache{
				state: cacheStateQueuing,
				updates: []pendingUpdate{
					{firstUpdateID: 11, update: &orderbook.Update{Pair: tc.pendingPair, Asset: asset.Spot, UpdateID: 11, AllowEmpty: true, UpdateTime: time.Now()}},
					{firstUpdateID: 12, update: &orderbook.Update{Pair: tc.updatePair, Asset: tc.updateAsset, UpdateID: 12, AllowEmpty: true, UpdateTime: time.Now()}},
				},
			}

			err = m.applyPendingUpdates(cache)
			require.ErrorIs(t, err, errPendingUpdateKeyMismatch, "applyPendingUpdates must error when pending update keys differ")
			assert.Contains(t, err.Error(), tc.wantContext, "applyPendingUpdates error should identify the mismatched pending-update key")
			lastUpdateID, lastUpdateErr := m.ob.LastUpdateID(tc.pendingPair, asset.Spot)
			require.NoError(t, lastUpdateErr, "LastUpdateID must not error after rejecting mismatched pending updates")
			assert.Equal(t, int64(10), lastUpdateID, "applyPendingUpdates should reject a mismatched queue before applying an update")
			assert.Empty(t, m.ob.dataHandler.C, "applyPendingUpdates should reject a mismatched queue before publishing an update")
			assert.Equal(t, cacheStateQueuing, cache.state, "applyPendingUpdates should not sync pending updates with different keys")
		})
	}
}

func TestApplyPendingUpdatesCachedDepthAfterSkip(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair, err := currency.NewPairFromStrings("PENDSKIP", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 10})
	require.NoError(t, err, "LoadSnapshot must not error for the skipped-update orderbook")
	m.checkPendingUpdate = func(_, _ int64, update *orderbook.Update) (bool, error) {
		return update.UpdateID == 10, nil
	}

	cache := &updateCache{updates: []pendingUpdate{
		{firstUpdateID: 10, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 10, AllowEmpty: true, UpdateTime: time.Now()}},
		{firstUpdateID: 11, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 11, AllowEmpty: true, UpdateTime: time.Now()}},
	}}
	err = m.applyPendingUpdates(cache)
	require.NoError(t, err, "applyPendingUpdates must not error after skipping an outdated pending update")
	assert.Equal(t, cacheStateSynced, cache.state, "applyPendingUpdates should sync after skipping an outdated pending update")
	lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error for the skipped-update orderbook")
	assert.Equal(t, int64(11), lastUpdateID, "LastUpdateID should return the applied update ID after a skip")
}

func TestApplyPendingUpdatesCachedDepthAdvances(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair, err := currency.NewPairFromStrings("PENDADVANCE", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 10})
	require.NoError(t, err, "LoadSnapshot must not error for the advancing orderbook")

	cache := &updateCache{updates: []pendingUpdate{
		{firstUpdateID: 11, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 11, AllowEmpty: true, UpdateTime: time.Now()}},
		{firstUpdateID: 12, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 12, AllowEmpty: true, UpdateTime: time.Now()}},
	}}
	err = m.applyPendingUpdates(cache)
	require.NoError(t, err, "applyPendingUpdates must not error for consecutive in-sequence updates")
	assert.Equal(t, cacheStateSynced, cache.state, "applyPendingUpdates should sync consecutive in-sequence updates")
	lastUpdateID, err := m.ob.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error for the advancing orderbook")
	assert.Equal(t, int64(12), lastUpdateID, "LastUpdateID should advance to the final applied update ID")
}

func TestApplyPendingUpdatesCachedDepthInvalidated(t *testing.T) {
	t.Parallel()

	tp := newTestParams()
	m := NewUpdateManager(&tp)
	pair, err := currency.NewPairFromStrings("PENDINVALID", "USDT")
	require.NoError(t, err, "NewPairFromStrings must not error")
	err = m.ob.LoadSnapshot(t.Context(), &orderbook.Book{Pair: pair, Asset: asset.Spot, Exchange: m.ob.exchangeName, LastUpdated: time.Now(), LastUpdateID: 10})
	require.NoError(t, err, "LoadSnapshot must not error for the invalidation orderbook")

	var invalidationErr error
	m.checkPendingUpdate = func(_, _ int64, _ *orderbook.Update) (bool, error) {
		invalidationErr = m.ob.InvalidateOrderbook(pair, asset.Spot)
		return true, nil
	}
	err = m.applyPendingUpdates(&updateCache{updates: []pendingUpdate{
		{firstUpdateID: 10, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 10, AllowEmpty: true, UpdateTime: time.Now()}},
		{firstUpdateID: 11, update: &orderbook.Update{Pair: pair, Asset: asset.Spot, UpdateID: 11, AllowEmpty: true, UpdateTime: time.Now()}},
	}})
	assert.NoError(t, invalidationErr, "InvalidateOrderbook should not error")
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "applyPendingUpdates must return the cached depth invalidation error")
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

	cache.ch = make(chan int64, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		err = cache.waitForUpdate(t.Context(), 1338)
	})
	cache.m.Lock()
	cache.updates = append(cache.updates, pendingUpdate{update: &orderbook.Update{UpdateID: 1338}})
	cache.m.Unlock()
	cache.ch <- 1338
	wg.Wait()
	assert.NoError(t, err)
}

func TestWaitForUpdateRechecksQueue(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		cache := &updateCache{ch: make(chan int64, 1), updates: []pendingUpdate{{update: &orderbook.Update{UpdateID: 1337}}}}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		errCh := make(chan error, 1)
		go func() { errCh <- cache.waitForUpdate(ctx, 1339) }()
		synctest.Wait()

		cache.ch <- 1337 // A stale notification must not end the wait.
		synctest.Wait()
		require.Empty(t, errCh, "waitForUpdate must keep waiting after a stale notification")

		cache.m.Lock()
		cache.updates = append(cache.updates, pendingUpdate{update: &orderbook.Update{UpdateID: 1338}})
		cache.m.Unlock()
		cache.ch <- 1338
		synctest.Wait()
		require.Empty(t, errCh, "waitForUpdate must keep waiting until the queue reaches the update")

		cache.m.Lock()
		cache.updates = append(cache.updates, pendingUpdate{update: &orderbook.Update{UpdateID: 1340}})
		cache.m.Unlock()
		cache.ch <- 1337 // A stale notification must not hide the latest queued update.
		assert.NoError(t, <-errCh, "waitForUpdate should observe the latest queued update despite a stale notification")
	})
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

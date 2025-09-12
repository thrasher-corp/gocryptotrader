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

	m := newWsOBUpdateManager(0, 0)
	err := m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
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

	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	// Test orderbook snapshot is behind update
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1340, &orderbook.Update{
		UpdateID:   1341,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache, err := m.LoadCache(pair, asset.USDTMarginedFutures)
	require.NoError(t, err, "LoadCache must not error")

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 1)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	// Test orderbook snapshot is behind update
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1342, &orderbook.Update{
		UpdateID:   1343,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 2)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	time.Sleep(time.Millisecond * 2) // Allow sync delay to pass

	cache.mtx.Lock()
	assert.Empty(t, cache.updates)
	assert.False(t, cache.updating)
	cache.mtx.Unlock()
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

func TestExtractOrderbookLimit(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	cache := &updateCache{}

	_, err := cache.extractOrderbookLimit(e, 1337)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = cache.extractOrderbookLimit(e, asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	require.NoError(t, err)

	for _, tc := range []struct {
		asset asset.Item
		exp   uint64
	}{
		{asset: asset.Spot, exp: 100},
		{asset: asset.USDTMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.CoinMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.DeliveryFutures, exp: deliveryFuturesUpdateLimit},
		{asset: asset.Options, exp: optionOrderbookUpdateLimit},
	} {
		limit, err := cache.extractOrderbookLimit(e, tc.asset)
		require.NoError(t, err)
		require.Equal(t, tc.exp, limit)
	}
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
	require.NoError(t, err)
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
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}, updating: true}
	cache.clearWithLock()
	require.Empty(t, cache.updates)
	require.False(t, cache.updating)
}

func TestClearWithNoLock(t *testing.T) {
	t.Parallel()
	cache := &updateCache{updates: []pendingUpdate{{update: &orderbook.Update{}}}, updating: true}
	cache.clearNoLock()
	require.Empty(t, cache.updates)
	require.False(t, cache.updating)
}

func TestApplyOrderbookUpdate(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)

	update := &orderbook.Update{
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	err := applyOrderbookUpdate(e, update)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	update.Asset = asset.Spot
	err = applyOrderbookUpdate(e, update)
	require.ErrorIs(t, err, currency.ErrPairNotEnabled)

	pair = currency.NewBTCUSD()
	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     e.Name,
		Pair:         pair,
		Asset:        asset.Spot,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	})
	require.NoError(t, err)

	err = e.Base.SetPairs([]currency.Pair{pair}, asset.Spot, true)
	require.NoError(t, err)

	update.Pair = pair
	err = applyOrderbookUpdate(e, update)
	require.NoError(t, err)

	update.AllowEmpty = false
	err = applyOrderbookUpdate(e, update)
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)
}

package gateio

import (
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

	m := newWsOBUpdateManager(0)
	err := m.ProcessOrderbookUpdate(t.Context(), g, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     g.Name,
		Pair:         pair,
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1336,
	})
	require.NoError(t, err)

	err = m.ProcessOrderbookUpdate(t.Context(), g, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	// Test orderbook snapshot is behind update
	err = m.ProcessOrderbookUpdate(t.Context(), g, 1340, &orderbook.Update{
		UpdateID:   1341,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache := m.LoadCache(pair, asset.USDTMarginedFutures)

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 1)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	// Test orderbook snapshot is behind update
	err = m.ProcessOrderbookUpdate(t.Context(), g, 1342, &orderbook.Update{
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

	m := newWsOBUpdateManager(0)
	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	cache := m.LoadCache(pair, asset.USDTMarginedFutures)
	assert.NotNil(t, cache)
	assert.Len(t, m.lookup, 1)

	// Test cache is reused
	cache2 := m.LoadCache(pair, asset.USDTMarginedFutures)
	assert.Equal(t, cache, cache2)
}

func TestSyncOrderbook(t *testing.T) {
	t.Parallel()

	g := new(Gateio)
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	require.NoError(t, g.UpdateTradablePairs(t.Context(), false))

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err := g.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	require.NoError(t, err)

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)

	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		pair := currency.NewPair(currency.ETH, currency.USDT)
		err := g.CurrencyPairs.EnablePair(a, pair)
		require.NoError(t, err)
		cache := m.LoadCache(pair, a)

		cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: a}}}
		cache.updating = true
		err = cache.SyncOrderbook(t.Context(), g, pair, a)
		require.NoError(t, err)
		require.False(t, cache.updating)
		require.Empty(t, cache.updates)

		expectedLimit := 20
		if a == asset.Spot {
			expectedLimit = 100
		}

		b, err := g.Websocket.Orderbook.GetOrderbook(pair, a)
		require.NoError(t, err)
		require.Len(t, b.Bids, expectedLimit)
		require.Len(t, b.Asks, expectedLimit)
	}
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	g := new(Gateio)
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	require.NoError(t, g.UpdateTradablePairs(t.Context(), false))

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)
	pair := currency.NewPair(currency.LTC, currency.USDT)
	err := g.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     g.Name,
		Pair:         pair,
		Asset:        asset.USDTMarginedFutures,
		Bids:         []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:         []orderbook.Level{{Price: 1, Amount: 1}},
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1335,
	})
	require.NoError(t, err)

	cache := m.LoadCache(pair, asset.USDTMarginedFutures)

	update := &orderbook.Update{
		UpdateID:   1339,
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	cache.updates = []pendingUpdate{{update: update, firstUpdateID: 1337}}
	err = cache.applyPendingUpdates(g, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errOrderbookSnapshotOutdated)

	cache.updates[0].firstUpdateID = 1336
	err = cache.applyPendingUpdates(g, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestApplyOrderbookUpdate(t *testing.T) {
	t.Parallel()

	g := new(Gateio)
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	require.NoError(t, g.UpdateTradablePairs(t.Context(), false))

	pair := currency.NewBTCUSDT()

	update := &orderbook.Update{
		Pair:       pair,
		Asset:      asset.USDTMarginedFutures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	err := applyOrderbookUpdate(g, update)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	update.Asset = asset.Spot
	err = applyOrderbookUpdate(g, update)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	update.Pair = currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = applyOrderbookUpdate(g, update)
	require.NoError(t, err)
}

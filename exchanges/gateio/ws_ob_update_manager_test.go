package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestProcessUpdate(t *testing.T) {
	t.Parallel()

	m := newWsOBUpdateManager(0)
	err := m.ProcessUpdate(t.Context(), g, 20, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Exchange:       g.Name,
		Pair:           pair,
		Asset:          asset.Futures,
		Bids:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		Asks:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		LastUpdated:    time.Now(),
		UpdatePushedAt: time.Now(),
		LastUpdateID:   1336,
	})
	require.NoError(t, err)

	err = m.ProcessUpdate(t.Context(), g, 20, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	// Test orderbook snapshot is behind update
	err = m.ProcessUpdate(t.Context(), g, 20, 1340, &orderbook.Update{
		UpdateID:   1341,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache := m.LoadCache(pair, asset.Futures)

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 1)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	// Test orderbook snapshot is behind update
	err = m.ProcessUpdate(t.Context(), g, 20, 1342, &orderbook.Update{
		UpdateID:   1343,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 2)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	time.Sleep(time.Millisecond) // Allow sync delay to pass

	cache.mtx.Lock()
	assert.Empty(t, cache.updates)
	assert.False(t, cache.updating)
	cache.mtx.Unlock()
}

func TestLoadCache(t *testing.T) {
	t.Parallel()

	m := newWsOBUpdateManager(0)
	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	cache := m.LoadCache(pair, asset.Futures)
	assert.NotNil(t, cache)
	assert.Len(t, m.m, 1)

	// Test cache is reused
	cache2 := m.LoadCache(pair, asset.Futures)
	assert.Equal(t, cache, cache2)
}

func TestSyncOrderbook(t *testing.T) {
	t.Parallel()

	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	require.NoError(t, g.UpdateTradablePairs(t.Context(), false))

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)

	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		pair := currency.NewPair(currency.ETH, currency.USDT)
		err := g.CurrencyPairs.EnablePair(a, pair)
		require.NoError(t, err)
		cache := m.LoadCache(pair, a)

		cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: a}}}
		cache.updating = true
		err = cache.SyncOrderbook(t.Context(), g, pair, a, 100)
		require.NoError(t, err)
		require.False(t, cache.updating)
		require.Empty(t, cache.updates)

		b, err := g.Websocket.Orderbook.GetOrderbook(pair, a)
		require.NoError(t, err)
		require.Len(t, b.Bids, 100)
		require.Len(t, b.Asks, 100)
	}
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	require.NoError(t, g.UpdateTradablePairs(t.Context(), false))

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)
	pair := currency.NewPair(currency.LTC, currency.USDT)
	err := g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Exchange:       g.Name,
		Pair:           pair,
		Asset:          asset.Futures,
		Bids:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		Asks:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		LastUpdated:    time.Now(),
		UpdatePushedAt: time.Now(),
		LastUpdateID:   1335,
	})
	require.NoError(t, err)

	cache := m.LoadCache(pair, asset.Futures)

	update := &orderbook.Update{
		UpdateID:   1339,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	cache.updates = []pendingUpdate{{update: update, firstUpdateID: 1337}}
	err = cache.ApplyPendingUpdates(g, asset.Futures)
	require.ErrorIs(t, err, errOrderbookSnapshotOutdated)

	cache.updates[0].firstUpdateID = 1336
	err = cache.ApplyPendingUpdates(g, asset.Futures)
	require.NoError(t, err)
}

func TestApplyOrderbookUpdate(t *testing.T) {
	t.Parallel()

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)
	pair := currency.NewPair(currency.BTC, currency.USDT)
	cache := m.LoadCache(pair, asset.Futures)

	update := &orderbook.Update{
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}

	err := cache.ApplyOrderbookUpdate(g, update)
	require.ErrorIs(t, err, buffer.ErrDepthNotFound)

	update.Asset = asset.Spot
	err = cache.ApplyOrderbookUpdate(g, update)
	require.ErrorIs(t, err, buffer.ErrDepthNotFound)

	update.Pair = currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = cache.ApplyOrderbookUpdate(g, update)
	require.NoError(t, err)
}

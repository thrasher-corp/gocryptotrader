package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	pair := currency.NewPair(currency.DOGE, currency.BABYDOGE)

	err := e.Websocket.AddSubscriptions(nil, &subscription.Subscription{
		Channel:  subscription.OrderbookChannel,
		Interval: kline.TwentyMilliseconds,
		Levels:   20,
		Asset:    asset.USDTMarginedFutures,
		Pairs:    currency.Pairs{pair},
	})
	require.NoError(t, err, "AddSubscriptions must not error")

	m := newWsOBUpdateManager(0)
	err = m.ProcessOrderbookUpdate(t.Context(), e, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

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

	cache := m.LoadCache(pair, asset.USDTMarginedFutures)

	cache.mtx.Lock()
	assert.Len(t, cache.updates, 1)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	for done := false; !done; {
		select {
		case v := <-e.Websocket.DataHandler:
			if e, ok := v.(error); ok {
				err = common.AppendError(err, e)
			}
		default:
			done = true
		}
	}
	require.NoError(t, err, "ProcessOrderbookUpdate must not have sent errors to Datahandler")

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

	time.Sleep(time.Millisecond * 4) // Allow sync delay to pass

	for done := false; !done; {
		select {
		case v := <-e.Websocket.DataHandler:
			if e, ok := v.(error); ok {
				err = common.AppendError(err, e)
			}
		default:
			done = true
		}
	}
	require.ErrorIs(t, err, errInvalidSettlementQuote, "ProcessOrderbookUpdate must not send errors to DataHandler")

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

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	require.NoError(t, e.UpdateTradablePairs(t.Context()))

	pair := currency.NewPair(currency.ETH, currency.USDT)

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)

	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		s := &subscription.Subscription{
			Channel:  subscription.OrderbookChannel,
			Interval: kline.TwentyMilliseconds,
			Levels:   20,
			Asset:    a,
			Pairs:    currency.Pairs{pair},
		}
		if a == asset.Spot {
			s.Levels = 100
			s.Interval = kline.HundredMilliseconds
		}
		err := e.Websocket.AddSubscriptions(nil, s)
		require.NoError(t, err, "AddSubscriptions must not error")

		require.NoError(t, e.CurrencyPairs.EnablePair(a, pair), "EnablePair must not error")
		cache := m.LoadCache(pair, a)

		cache.updates = []pendingUpdate{{update: &orderbook.Update{Pair: pair, Asset: a}}}
		cache.updating = true
		err = cache.SyncOrderbook(t.Context(), e, pair, a)
		require.NoError(t, err)
		require.False(t, cache.updating)
		require.Empty(t, cache.updates)

		b, err := e.Websocket.Orderbook.GetOrderbook(pair, a)
		require.NoError(t, err)
		require.Len(t, b.Bids, s.Levels)
		require.Len(t, b.Asks, s.Levels)
	}
}

func TestApplyPendingUpdates(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	require.NoError(t, e.UpdateTradablePairs(t.Context()))

	m := newWsOBUpdateManager(defaultWSSnapshotSyncDelay)
	pair := currency.NewPair(currency.LTC, currency.USDT)
	err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     e.Name,
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
	err = cache.applyPendingUpdates(e, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errOrderbookSnapshotOutdated)

	cache.updates[0].firstUpdateID = 1336
	err = cache.applyPendingUpdates(e, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestApplyOrderbookUpdate(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	require.NoError(t, e.UpdateTradablePairs(t.Context()))

	pair := currency.NewBTCUSDT()

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
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)

	update.Pair = currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = applyOrderbookUpdate(e, update)
	require.NoError(t, err)
}

// TestOrderbookSubKeyMatch exercises the orderbookSubKey MatchableKey interface implementation
// Ensures A.Pairs must be a subset of B.Pairs
func TestOrderbookSubKeyMatch(t *testing.T) {
	t.Parallel()

	btcusdtPair := currency.NewBTCUSDT()
	ethusdcPair := currency.NewPair(currency.ETH, currency.USDC)
	ltcusdcPair := currency.NewPair(currency.LTC, currency.USDC)

	key := &orderbookSubKey{&subscription.Subscription{Channel: subscription.OrderbookChannel}}
	try := &orderbookSubKey{&subscription.Subscription{Channel: subscription.TickerChannel}}

	assert.NotNil(t, key.EnsureKeyed(), "EnsureKeyed should work")

	require.False(t, key.Match(try), "Gate 1: Match must reject a bad Channel")
	try.Channel = subscription.OrderbookChannel
	require.True(t, key.Match(try), "Gate 1: Match must accept a good Channel")
	key.Asset = asset.Spot
	require.False(t, key.Match(try), "Gate 2: Match must reject a bad Asset")
	try.Asset = asset.Spot
	require.True(t, key.Match(try), "Gate 2: Match must accept a good Asset")

	key.Pairs = currency.Pairs{btcusdtPair}
	require.False(t, key.Match(try), "Gate 3: Match must reject B empty Pairs when key has Pairs")
	try.Pairs = currency.Pairs{btcusdtPair}
	key.Pairs = nil
	require.False(t, key.Match(try), "Gate 4: Match must reject B has Pairs when key has empty Pairs")
	key.Pairs = currency.Pairs{btcusdtPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept matching pairs")
	key.Pairs = currency.Pairs{ethusdcPair}
	require.False(t, key.Match(try), "Gate 5: Match must reject when key.Pairs not in try.Pairs")
	try.Pairs = currency.Pairs{btcusdtPair, ethusdcPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept one of the key.Pairs in try.Pairs")
	key.Pairs = currency.Pairs{btcusdtPair, ethusdcPair}
	try.Pairs = currency.Pairs{btcusdtPair, ltcusdcPair}
	require.False(t, key.Match(try), "Gate 5: Match must reject when key.Pairs not in try.Pairs")
	try.Pairs = currency.Pairs{btcusdtPair, ethusdcPair, ltcusdcPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept when all key.Pairs are subset of try.Pairs")
}

// TestOrderbookSubKeyString exercises orderbookSubKey.String
func TestOrderbookSubKeyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Uninitialised orderbookSubKey", orderbookSubKey{}.String())
	key := &orderbookSubKey{&subscription.Subscription{
		Asset:   asset.Spot,
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{currency.NewPair(currency.ETH, currency.USDC), currency.NewBTCUSDT()},
	}}
	assert.Equal(t, "orderbook spot [ETHUSDC BTCUSDT]", key.String())
}

// TestGetSubscription exercises GetSubscription
func TestOrderbookSubKeyGetSubscription(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{Asset: asset.Spot}
	assert.Same(t, s, orderbookSubKey{s}.GetSubscription(), "orderbookSub.GetSubscription should return a pointer to the subscription")
}

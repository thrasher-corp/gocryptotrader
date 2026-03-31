package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestStartWebsocketSyncWorkers(t *testing.T) {
	t.Parallel()

	m := &SyncManager{
		config:   config.SyncManagerConfig{NumWorkers: 0},
		shutdown: make(chan bool),
	}
	m.startWebsocketSyncWorkers()
	require.NotNil(t, m.wsUpdateQueue)
	assert.Equal(t, websocketSyncQueueSize, cap(m.wsUpdateQueue))

	close(m.shutdown)
	m.wsSyncWG.Wait()
}

func TestEnqueueWebsocketUpdate(t *testing.T) {
	t.Parallel()

	var nilSyncer *SyncManager
	err := nilSyncer.EnqueueWebsocketUpdate("test", &ticker.Price{})
	require.ErrorIs(t, err, ErrNilSubsystem)

	m := &SyncManager{}
	err = m.EnqueueWebsocketUpdate("test", &ticker.Price{})
	require.NoError(t, err)

	m.wsUpdateQueue = make(chan websocketSyncUpdate, 4)
	err = m.EnqueueWebsocketUpdate("test", struct{}{})
	require.ErrorIs(t, err, errUnsupportedWebsocketSyncData)

	err = m.EnqueueWebsocketUpdate("test", (*ticker.Price)(nil))
	require.NoError(t, err)
	err = m.EnqueueWebsocketUpdate("test", (*orderbook.Depth)(nil))
	require.NoError(t, err)

	price := ticker.Price{
		ExchangeName: "test",
		Pair:         currency.NewBTCUSD(),
		AssetType:    asset.Spot,
	}
	originalPair := price.Pair
	err = m.EnqueueWebsocketUpdate("test", &price)
	require.NoError(t, err)
	price.Pair = currency.NewPair(currency.ETH, currency.USD)

	update := <-m.wsUpdateQueue
	require.NotNil(t, update.ticker)
	assert.True(t, update.ticker.Pair.Equal(originalPair))

	batch := []ticker.Price{{
		ExchangeName: "test",
		Pair:         currency.NewBTCUSD(),
		AssetType:    asset.Spot,
	}}
	originalBatchPair := batch[0].Pair
	err = m.EnqueueWebsocketUpdate("test", batch)
	require.NoError(t, err)
	batch[0].Pair = currency.NewPair(currency.ETH, currency.USD)

	update = <-m.wsUpdateQueue
	require.Len(t, update.tickerBatch, 1)
	assert.True(t, update.tickerBatch[0].Pair.Equal(originalBatchPair))

	m.wsUpdateQueue = make(chan websocketSyncUpdate, 1)
	m.wsUpdateQueue <- websocketSyncUpdate{}
	err = m.EnqueueWebsocketUpdate("test", &ticker.Price{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(m.wsUpdateQueue))
}

func TestWebsocketSyncWorker(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSD()
	m := newWebsocketSyncTestManager()
	m.config.NumWorkers = 1
	m.shutdown = make(chan bool)
	m.startWebsocketSyncWorkers()
	m.add(key.NewExchangeAssetPair("test", asset.Spot, pair), syncBase{})

	err := m.EnqueueWebsocketUpdate("test", &ticker.Price{
		ExchangeName: "test",
		Pair:         pair,
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		c := m.get(key.NewExchangeAssetPair("test", asset.Spot, pair))
		return c != nil && c.trackers[SyncItemTicker].HaveData
	}, time.Second, 10*time.Millisecond)

	close(m.shutdown)
	m.wsSyncWG.Wait()
}

func TestHandleWebsocketSyncUpdate(t *testing.T) {
	t.Parallel()

	m := newWebsocketSyncTestManager()
	err := m.handleWebsocketSyncUpdate(websocketSyncUpdate{})
	require.NoError(t, err)

	pair := currency.NewBTCUSD()
	m.add(key.NewExchangeAssetPair("test", asset.Spot, pair), syncBase{})
	err = m.handleWebsocketSyncUpdate(websocketSyncUpdate{
		exchangeName: "test",
		ticker: &ticker.Price{
			ExchangeName: "test",
			Pair:         pair,
			AssetType:    asset.Spot,
		},
	})
	require.NoError(t, err)

	depth := newDepthForPair(t, "test", pair, asset.Spot)
	err = m.handleWebsocketSyncUpdate(websocketSyncUpdate{
		exchangeName: "test",
		orderbook:    depth,
	})
	require.NoError(t, err)
}

func TestHandleWebsocketTicker(t *testing.T) {
	t.Parallel()

	m := newWebsocketSyncTestManager()
	untrackedPair := currency.NewBTCUSD()
	err := m.handleWebsocketTicker("test", &ticker.Price{
		ExchangeName: "test",
		Pair:         untrackedPair,
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)

	trackedPair := currency.NewPair(currency.ETH, currency.USD)
	m.add(key.NewExchangeAssetPair("test", asset.Spot, trackedPair), syncBase{})
	err = m.handleWebsocketTicker("test", &ticker.Price{
		ExchangeName: "test",
		Pair:         trackedPair,
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)
	c := m.get(key.NewExchangeAssetPair("test", asset.Spot, trackedPair))
	require.NotNil(t, c)
	assert.True(t, c.trackers[SyncItemTicker].HaveData)
}

func TestHandleWebsocketTickerBatch(t *testing.T) {
	t.Parallel()

	m := newWebsocketSyncTestManager()
	trackedPair := currency.NewBTCUSD()
	untrackedPair := currency.NewPair(currency.ETH, currency.USD)
	m.add(key.NewExchangeAssetPair("test", asset.Spot, trackedPair), syncBase{})

	err := m.handleWebsocketTickerBatch("test", []ticker.Price{
		{ExchangeName: "test", Pair: untrackedPair, AssetType: asset.Spot},
		{ExchangeName: "test", Pair: trackedPair, AssetType: asset.Spot},
	})
	require.NoError(t, err)

	c := m.get(key.NewExchangeAssetPair("test", asset.Spot, trackedPair))
	require.NotNil(t, c)
	assert.True(t, c.trackers[SyncItemTicker].HaveData)
	assert.Nil(t, m.get(key.NewExchangeAssetPair("test", asset.Spot, untrackedPair)))
}

func TestHandleWebsocketOrderbook(t *testing.T) {
	t.Parallel()

	m := newWebsocketSyncTestManager()
	trackedPair := currency.NewBTCUSD()
	m.add(key.NewExchangeAssetPair("test", asset.Spot, trackedPair), syncBase{})

	depth := newDepthForPair(t, "test", trackedPair, asset.Spot)
	err := m.handleWebsocketOrderbook("test", depth)
	require.NoError(t, err)
	c := m.get(key.NewExchangeAssetPair("test", asset.Spot, trackedPair))
	require.NotNil(t, c)
	assert.True(t, c.trackers[SyncItemOrderbook].HaveData)

	untrackedDepth := newDepthForPair(t, "test", currency.NewPair(currency.ETH, currency.USD), asset.Spot)
	err = m.handleWebsocketOrderbook("test", untrackedDepth)
	require.NoError(t, err)

	invalidDepth := orderbook.NewDepth(uuid.Must(uuid.NewV4()))
	_ = invalidDepth.Invalidate(errors.New("bad depth"))
	err = m.handleWebsocketOrderbook("test", invalidDepth)
	require.Error(t, err)
}

func TestIsTrackedPair(t *testing.T) {
	t.Parallel()

	m := newWebsocketSyncTestManager()
	pair := currency.NewBTCUSD()
	assert.False(t, m.isTrackedPair("test", pair, asset.Spot))

	m.add(key.NewExchangeAssetPair("test", asset.Spot, pair), syncBase{})
	assert.True(t, m.isTrackedPair("test", pair, asset.Spot))
}

func newWebsocketSyncTestManager() *SyncManager {
	return &SyncManager{
		started:         1,
		initSyncStarted: 1,
		config: config.SyncManagerConfig{
			SynchronizeTicker:    true,
			SynchronizeOrderbook: true,
		},
		currencyPairs: make(map[key.ExchangeAssetPair]*currencyPairSyncAgent),
	}
}

func newDepthForPair(t *testing.T, exchangeName string, pair currency.Pair, a asset.Item) *orderbook.Depth {
	t.Helper()

	depth := orderbook.NewDepth(uuid.Must(uuid.NewV4()))
	book := &orderbook.Book{
		Exchange:    exchangeName,
		Pair:        pair,
		Asset:       a,
		LastUpdated: time.Now(),
		Bids:        []orderbook.Level{{Price: 1, Amount: 1}},
		Asks:        []orderbook.Level{{Price: 2, Amount: 1}},
	}
	depth.AssignOptions(book)
	require.NoError(t, depth.LoadSnapshot(book))
	return depth
}

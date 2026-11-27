package engine

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type syncTickerContextExchange struct {
	exchange.IBotExchange
	updateTickerFn func(context.Context, currency.Pair, asset.Item) (*ticker.Price, error)
}

func (s syncTickerContextExchange) SupportsRESTTickerBatchUpdates() bool {
	return false
}

func (s syncTickerContextExchange) UpdateTicker(ctx context.Context, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
	if s.updateTickerFn != nil {
		return s.updateTickerFn(ctx, pair, item)
	}
	return nil, nil
}

func TestSetupSyncManager(t *testing.T) {
	t.Parallel()
	_, err := SetupSyncManager(nil, nil, nil, false)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = SetupSyncManager(&config.SyncManagerConfig{}, nil, nil, false)
	assert.ErrorIs(t, err, errNoSyncItemsEnabled)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, nil, nil, false)
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, nil, false)
	assert.ErrorIs(t, err, errNilConfig)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.BTC}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, currency.ErrFiatDisplayCurrencyIsNotFiat)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	m, err := SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.NoError(t, err)

	if m == nil {
		t.Error("expected manager")
	}
}

func TestSyncManagerStart(t *testing.T) {
	t.Parallel()
	m, err := SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.NoError(t, err)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m.exchangeManager = em
	m.config.SynchronizeContinuously = true
	err = m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	m = nil
	err = m.Start(t.Context())
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestSyncManagerStop(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

type restTickerSyncExchange struct {
	exchange.IBotExchange
	batches, fail                             bool
	slot                                      *atomic.Int64
	requests, requestsHoldingSlot, cacheReads atomic.Int32
}

func (e *restTickerSyncExchange) GetName() string                      { return "Bitstamp" }
func (e *restTickerSyncExchange) SupportsRESTTickerBatchUpdates() bool { return e.batches }

func (e *restTickerSyncExchange) request() error {
	e.requests.Add(1)
	if e.slot.Load() == 1 {
		e.requestsHoldingSlot.Add(1)
	}
	if e.fail {
		return errors.New("request refused")
	}
	return nil
}

func (e *restTickerSyncExchange) UpdateTickers(context.Context, asset.Item) error {
	return e.request()
}

func (e *restTickerSyncExchange) UpdateTicker(_ context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.request(); err != nil {
		return nil, err
	}
	return &ticker.Price{Pair: p, AssetType: a}, nil
}

func (e *restTickerSyncExchange) GetCachedTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
	e.cacheReads.Add(1)
	return &ticker.Price{Pair: p, AssetType: a}, nil
}

func TestSyncManagerSyncTicker(t *testing.T) {
	t.Parallel()

	t.Run("returns once the runtime context is cancelled", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		exch, err := em.NewExchangeByName("Bitstamp")
		require.NoError(t, err)
		exch.SetDefaults()

		started := make(chan context.Context, 1)
		wrapped := syncTickerContextExchange{
			IBotExchange: exch,
			updateTickerFn: func(ctx context.Context, _ currency.Pair, _ asset.Item) (*ticker.Price, error) {
				started <- ctx
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}

		m, err := SetupSyncManager(&config.SyncManagerConfig{
			SynchronizeTicker:   true,
			FiatDisplayCurrency: currency.USD,
			PairFormatDisplay:   &currency.EMPTYFORMAT,
			TimeoutREST:         time.Millisecond,
			TimeoutWebsocket:    time.Second,
		}, em, &config.RemoteControlConfig{}, false)
		require.NoError(t, err)
		m.started.Store(true)
		m.initSyncCompleted.Store(true)

		runtimeCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		pair := currency.NewBTCUSD()
		agent := &currencyPairSyncAgent{
			Key:      key.NewExchangeAssetPair(wrapped.GetName(), asset.Spot, pair),
			Pair:     pair,
			Created:  time.Now().Add(-time.Minute),
			trackers: make([]*syncBase, 3),
			locks:    make([]sync.Mutex, 3),
		}
		agent.trackers[SyncItemTicker] = &syncBase{IsUsingREST: true}

		done := make(chan struct{})
		go func() {
			m.syncTicker(runtimeCtx, agent, wrapped)
			close(done)
		}()

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected update ticker call to start")
		}

		cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("expected syncTicker to return after runtime context cancellation")
		}
	})

	for _, tc := range []struct {
		name              string
		batches           bool
		freshBatch        bool
		expiredBatch      bool
		fail              bool
		updatedWhileTaken bool
		requests          int32
		cacheReads        int32
	}{
		{name: "a fresh batch is read from the cache while the REST calls are taken", batches: true, freshBatch: true, updatedWhileTaken: true, cacheReads: 1},
		{name: "a due batch waits for a free REST call", batches: true, requests: 1, cacheReads: 1},
		{name: "an expired batch waits for a free REST call", batches: true, expiredBatch: true, requests: 1, cacheReads: 1},
		{name: "a failed batch releases its REST call", batches: true, fail: true, requests: 1},
		{name: "a single ticker waits for a free REST call", requests: 1},
		{name: "a failed single ticker releases its REST call", fail: true, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := SetupSyncManager(&config.SyncManagerConfig{
				SynchronizeTicker:   true,
				NumWorkers:          3,
				TimeoutREST:         time.Minute,
				TimeoutWebsocket:    time.Minute,
				FiatDisplayCurrency: currency.USD,
				PairFormatDisplay:   &currency.EMPTYFORMAT,
			}, NewExchangeManager(), &config.RemoteControlConfig{}, false)
			require.NoError(t, err, "SetupSyncManager must not error")
			m.started.Store(true)
			m.initSyncCompleted.Store(true)
			m.exchangeCount.Store(2)
			held, ok := m.tryAcquireExchangeREST("Bitstamp")
			require.True(t, ok, "an order book sync must take the exchange's only REST call")
			if tc.freshBatch {
				m.tickerBatchLastRequested[key.ExchangeAsset{Exchange: "Bitstamp", Asset: asset.Spot}] = time.Now()
			}
			if tc.expiredBatch {
				m.tickerBatchLastRequested[key.ExchangeAsset{Exchange: "Bitstamp", Asset: asset.Spot}] = time.Now().Add(-2 * time.Minute)
			}
			c := m.add(key.NewExchangeAssetPair("Bitstamp", asset.Spot, currency.NewBTCUSD()), syncBase{IsUsingREST: true})
			e := &restTickerSyncExchange{batches: tc.batches, fail: tc.fail, slot: held}

			m.syncTicker(t.Context(), c, e)
			require.Zero(t, e.requests.Load(), "no REST request must be made while the exchange's REST calls are taken")
			require.Equal(t, tc.updatedWhileTaken, c.trackers[SyncItemTicker].HaveData, "the ticker must only be updated from a fresh batch while the REST calls are taken")

			held.Add(-1)
			m.syncTicker(t.Context(), c, e)
			assert.Equal(t, tc.requests, e.requests.Load(), "REST requests should be made once a REST call is free")
			assert.Equal(t, tc.requests, e.requestsHoldingSlot.Load(), "each REST request should hold the exchange's REST call")
			assert.Equal(t, tc.cacheReads, e.cacheReads.Load(), "the batch cache should be read as expected")
			assert.True(t, c.trackers[SyncItemTicker].HaveData, "the ticker should be updated")
			assert.Zero(t, held.Load(), "every REST call should be released")
		})
	}
}

func TestPrintCurrencyFormat(t *testing.T) {
	t.Parallel()
	c := printCurrencyFormat(1337, currency.BTC)
	if c == "" {
		t.Error("expected formatted currency")
	}
}

func TestPrintConvertCurrencyFormat(t *testing.T) {
	t.Parallel()
	c := printConvertCurrencyFormat(1337, currency.BTC, currency.USD)
	if c == "" {
		t.Error("expected formatted currency")
	}
}

func TestPrintTickerSummary(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	m.PrintTickerSummary(&ticker.Price{}, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	m.started.Store(true)
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewBTCUSDT(),
	}, "REST", nil)
	m.fiatDisplayCurrency = currency.USD
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.JPY
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", errors.New("test"))

	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", common.ErrNotYetImplemented)
}

func TestPrintOrderbookSummary(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	m.PrintOrderbookSummary(nil, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	m.started.Store(true)
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.USD
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.JPY
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", common.ErrNotYetImplemented)

	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", errors.New("test"))

	m.PrintOrderbookSummary(nil, "REST", errors.New("test"))
}

func TestWaitForInitialSync(t *testing.T) {
	var m *SyncManager
	err := m.WaitForInitialSync()
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = &SyncManager{}
	err = m.WaitForInitialSync()
	require.ErrorIs(t, err, ErrSubSystemNotStarted)

	m.started.Store(true)
	err = m.WaitForInitialSync()
	require.NoError(t, err)
}

func TestSyncManagerWebsocketUpdate(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	err := m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = &SyncManager{}
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.ErrorIs(t, err, ErrSubSystemNotStarted)

	m.started.Store(true)
	// not started initial sync
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.NoError(t, err)

	m.initSyncStarted.Store(true)
	// orderbook not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	require.NoError(t, err)

	m.config.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, nil)
	require.NoError(t, err)

	m.config.SynchronizeTicker = true
	// trades not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, nil)
	require.NoError(t, err)

	m.config.SynchronizeTrades = true
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, 1336, nil)
	require.ErrorIs(t, err, errUnknownSyncItem)

	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	require.ErrorIs(t, err, errCouldNotSyncNewData)

	m.add(key.NewExchangeAssetPair("", asset.Spot, currency.EMPTYPAIR), syncBase{})
	m.initSyncWG.Add(3)
	// orderbook match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, errors.New("test"))
	require.NoError(t, err)

	// ticker match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, errors.New("test"))
	require.NoError(t, err)

	// trades match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, errors.New("test"))
	require.NoError(t, err)
}

type isolatedRESTSyncExchange struct {
	exchange.IBotExchange
	name              string
	pairs             currency.Pairs
	updateTickerFn    func(context.Context, currency.Pair, asset.Item) (*ticker.Price, error)
	updateOrderbookFn func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error)
}

func (e isolatedRESTSyncExchange) GetName() string                           { return e.name }
func (e isolatedRESTSyncExchange) SupportsWebsocket() bool                   { return false }
func (e isolatedRESTSyncExchange) SupportsREST() bool                        { return true }
func (e isolatedRESTSyncExchange) SupportsRESTTickerBatchUpdates() bool      { return false }
func (e isolatedRESTSyncExchange) IsAssetWebsocketSupported(asset.Item) bool { return false }

func (e isolatedRESTSyncExchange) GetAssetTypes(bool) asset.Items {
	return asset.Items{asset.Spot}
}

func (e isolatedRESTSyncExchange) GetEnabledPairs(asset.Item) (currency.Pairs, error) {
	return e.pairs, nil
}

func (e isolatedRESTSyncExchange) UpdateTicker(ctx context.Context, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
	return e.updateTickerFn(ctx, pair, item)
}

func (e isolatedRESTSyncExchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, item asset.Item) (*orderbook.Book, error) {
	return e.updateOrderbookFn(ctx, pair, item)
}

func TestSyncManagerSlowExchangeDoesNotStarveOtherExchanges(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()

	baseSlow, err := em.NewExchangeByName("Bitstamp")
	require.NoError(t, err)
	baseSlow.SetDefaults()

	baseFast, err := em.NewExchangeByName("Bitstamp")
	require.NoError(t, err)
	baseFast.SetDefaults()

	slowPairs := currency.Pairs{
		currency.NewBTCUSD(),
		currency.NewPair(currency.ETH, currency.USD),
		currency.NewPair(currency.LTC, currency.USD),
		currency.NewPair(currency.XRP, currency.USD),
	}
	fastPairs := currency.Pairs{currency.NewBTCUSD()}

	var slowInFlight, slowMaxInFlight atomic.Int32
	var fastTickers, fastOrderbooks atomic.Int32
	blockCh := make(chan struct{})
	defer close(blockCh)

	recordSlow := func(ctx context.Context) error {
		cur := slowInFlight.Add(1)
		defer slowInFlight.Add(-1)
		for {
			prev := slowMaxInFlight.Load()
			if cur <= prev || slowMaxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		select {
		case <-blockCh:
		case <-ctx.Done():
		}
		return ctx.Err()
	}

	require.NoError(t, em.Add(isolatedRESTSyncExchange{
		IBotExchange: baseSlow,
		name:         "SlowExchange",
		pairs:        slowPairs,
		updateTickerFn: func(ctx context.Context, _ currency.Pair, _ asset.Item) (*ticker.Price, error) {
			return nil, recordSlow(ctx)
		},
		updateOrderbookFn: func(ctx context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Book, error) {
			return nil, recordSlow(ctx)
		},
	}))
	require.NoError(t, em.Add(isolatedRESTSyncExchange{
		IBotExchange: baseFast,
		name:         "FastExchange",
		pairs:        fastPairs,
		updateTickerFn: func(_ context.Context, p currency.Pair, _ asset.Item) (*ticker.Price, error) {
			fastTickers.Add(1)
			return &ticker.Price{Pair: p}, nil
		},
		updateOrderbookFn: func(_ context.Context, p currency.Pair, _ asset.Item) (*orderbook.Book, error) {
			fastOrderbooks.Add(1)
			return &orderbook.Book{Pair: p}, nil
		},
	}))

	m, err := SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeContinuously: true,
		SynchronizeTicker:       true,
		SynchronizeOrderbook:    true,
		NumWorkers:              3,
		TimeoutREST:             40 * time.Millisecond,
		TimeoutWebsocket:        40 * time.Millisecond,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &currency.EMPTYFORMAT,
	}, em, &config.RemoteControlConfig{}, false)
	require.NoError(t, err)
	m.initSyncCompleted.Store(true)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, m.Start(ctx))
	defer func() {
		cancel()
		_ = m.Stop()
	}()

	require.Eventually(t, func() bool {
		return slowMaxInFlight.Load() >= 1 && fastTickers.Load() >= 3 && fastOrderbooks.Load() >= 3
	}, 2*time.Second, 20*time.Millisecond, "slow exchange must hold a REST worker while the fast exchange ticker and orderbook keep refreshing")

	assert.Equal(t, int32(1), slowMaxInFlight.Load(), "slow exchange should hold exactly its one REST worker")
}

func TestSyncManagerWorker(t *testing.T) {
	t.Parallel()

	t.Run("an exchange loaded at run time joins the share", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		require.NoError(t, em.Add(isolatedRESTSyncExchange{name: "FirstExchange"}), "Add must not error")
		m, err := SetupSyncManager(&config.SyncManagerConfig{
			SynchronizeTicker:   true,
			NumWorkers:          3,
			TimeoutREST:         10 * time.Millisecond,
			TimeoutWebsocket:    10 * time.Millisecond,
			FiatDisplayCurrency: currency.USD,
			PairFormatDisplay:   &currency.EMPTYFORMAT,
		}, em, &config.RemoteControlConfig{}, false)
		require.NoError(t, err, "SetupSyncManager must not error")
		m.started.Store(true)
		m.shutdown = make(chan bool)
		stopped := make(chan struct{})
		go func() {
			m.worker(t.Context())
			close(stopped)
		}()
		t.Cleanup(func() {
			close(m.shutdown)
			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				t.Error("worker should stop once shut down")
			}
		})

		require.Eventually(t, func() bool { return m.exchangeCount.Load() == 1 }, 5*time.Second, 10*time.Millisecond, "the worker must count the loaded exchange")
		require.NoError(t, em.Add(isolatedRESTSyncExchange{name: "SecondExchange"}), "Add must not error")
		assert.Eventually(t, func() bool { return m.exchangeCount.Load() == 2 }, 5*time.Second, 10*time.Millisecond, "the worker should count an exchange loaded at run time")
	})
}

func TestSyncManagerTryAcquireExchangeREST(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		workers   int
		exchanges int64
		admitted  int
	}{
		{name: "exchange count not yet stored admits every worker", workers: 15, exchanges: 0, admitted: 15},
		{name: "one exchange may use every worker", workers: 15, exchanges: 1, admitted: 15},
		{name: "one worker is always admitted", workers: 1, exchanges: 2, admitted: 1},
		{name: "workers are shared between exchanges", workers: 15, exchanges: 2, admitted: 7},
		{name: "a sixth of the workers when many exchanges are loaded", workers: 15, exchanges: 23, admitted: 2},
		{name: "share rounds down", workers: 3, exchanges: 2, admitted: 1},
		{name: "more exchanges than workers still admits one", workers: 3, exchanges: 5, admitted: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &SyncManager{config: config.SyncManagerConfig{NumWorkers: tc.workers}}
			m.exchangeCount.Store(tc.exchanges)
			var admitted int
			for range tc.workers {
				if _, ok := m.tryAcquireExchangeREST("Bitstamp"); ok {
					admitted++
				}
			}
			assert.Equal(t, tc.admitted, admitted, "tryAcquireExchangeREST should admit the exchange's share of workers")
		})
	}

	t.Run("a call admitted with one exchange loaded still counts once a second loads", func(t *testing.T) {
		t.Parallel()
		m := &SyncManager{config: config.SyncManagerConfig{NumWorkers: 2}}
		m.exchangeCount.Store(1)
		_, ok := m.tryAcquireExchangeREST("Bitstamp")
		require.True(t, ok, "first call must be admitted")
		m.exchangeCount.Store(2)
		_, ok = m.tryAcquireExchangeREST("Bitstamp")
		assert.False(t, ok, "the first call should already fill a share of one")
	})

	t.Run("a released slot is admitted again and exchanges are capped apart", func(t *testing.T) {
		t.Parallel()
		m := &SyncManager{config: config.SyncManagerConfig{NumWorkers: 3}}
		m.exchangeCount.Store(2)
		counter, ok := m.tryAcquireExchangeREST("Bitstamp")
		require.True(t, ok, "first slot must be admitted")
		require.NotNil(t, counter, "a capped slot must return its counter")
		_, ok = m.tryAcquireExchangeREST("Bitstamp")
		assert.False(t, ok, "a full exchange should refuse another slot")
		_, ok = m.tryAcquireExchangeREST("Kraken")
		assert.True(t, ok, "another exchange should keep its own slot")
		counter.Add(-1)
		_, ok = m.tryAcquireExchangeREST("Bitstamp")
		assert.True(t, ok, "a released slot should be admitted again")
	})
}

func TestSyncManagerGetExchangeRESTCounter(t *testing.T) {
	t.Parallel()
	m := &SyncManager{}
	counter := m.getExchangeRESTCounter("Bitstamp")
	require.NotNil(t, counter, "a new exchange must get a counter")
	assert.Same(t, counter, m.getExchangeRESTCounter("Bitstamp"), "the same exchange should get the same counter")
	assert.NotSame(t, counter, m.getExchangeRESTCounter("Kraken"), "another exchange should get its own counter")

	for _, tc := range []struct {
		name     string
		existing bool
	}{
		{name: "a new counter does not wait for the lock a ticker batch holds"},
		{name: "an existing counter does not wait for the lock a ticker batch holds", existing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &SyncManager{}
			if tc.existing {
				m.getExchangeRESTCounter("Bitstamp")
			}
			m.mux.Lock()
			got := make(chan *atomic.Int64, 1)
			go func() { got <- m.getExchangeRESTCounter("Bitstamp") }()
			select {
			case gotCounter := <-got:
				m.mux.Unlock()
				assert.NotNil(t, gotCounter, "a counter should be returned while m.mux is held")
			case <-time.After(5 * time.Second):
				m.mux.Unlock()
				require.Fail(t, "getExchangeRESTCounter must not wait for m.mux")
			}
		})
	}
}

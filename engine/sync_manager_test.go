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

func TestSyncManagerSyncTickerUsesRuntimeContextCancellation(t *testing.T) {
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

type blockingBatchExchange struct {
	exchange.IBotExchange
	started     chan struct{}
	release     chan struct{}
	startOnce   sync.Once
	updateCount atomic.Int32
}

func (b *blockingBatchExchange) SupportsRESTTickerBatchUpdates() bool { return true }

func (b *blockingBatchExchange) UpdateTickers(ctx context.Context, _ asset.Item) error {
	b.updateCount.Add(1)
	b.startOnce.Do(func() { close(b.started) })
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *blockingBatchExchange) GetCachedTicker(currency.Pair, asset.Item) (*ticker.Price, error) {
	return &ticker.Price{}, nil
}

func TestSyncTickerBatchDoesNotBlockOtherExchanges(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Binance")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	batching := &blockingBatchExchange{
		IBotExchange: exch,
		started:      make(chan struct{}),
		release:      make(chan struct{}),
	}

	m, err := SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeTicker:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   &currency.EMPTYFORMAT,
		TimeoutREST:         time.Millisecond,
		TimeoutWebsocket:    time.Second,
	}, em, &config.RemoteControlConfig{}, false)
	require.NoError(t, err, "SetupSyncManager must not error")
	m.started.Store(true)
	m.initSyncCompleted.Store(true)

	pair := currency.NewBTCUSDT()
	agent := newCurrencyPairSyncAgent(key.NewExchangeAssetPair(batching.GetName(), asset.Spot, pair))
	agent.trackers[SyncItemTicker] = &syncBase{IsUsingREST: true}

	done := make(chan struct{})
	go func() {
		m.syncTicker(t.Context(), agent, batching)
		close(done)
	}()
	<-batching.started

	looked := make(chan struct{})
	go func() {
		m.get(key.NewExchangeAssetPair("Kraken", asset.Spot, pair))
		close(looked)
	}()
	select {
	case <-looked:
	case <-time.After(time.Second):
		t.Error("another exchange's sync lookup should not wait for this exchange's batch request")
	}
	close(batching.release)
	<-looked
	<-done
}

func TestSyncTickerBatchReChecksAfterLock(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Binance")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	batching := &blockingBatchExchange{
		IBotExchange: exch,
		started:      make(chan struct{}),
		release:      make(chan struct{}),
	}

	m, err := SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeTicker:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   &currency.EMPTYFORMAT,
		TimeoutREST:         time.Minute,
		TimeoutWebsocket:    time.Second,
	}, em, &config.RemoteControlConfig{}, false)
	require.NoError(t, err, "SetupSyncManager must not error")
	m.started.Store(true)
	m.initSyncCompleted.Store(true)

	pair1 := currency.NewBTCUSDT()
	agent1 := newCurrencyPairSyncAgent(key.NewExchangeAssetPair(batching.GetName(), asset.Spot, pair1))
	agent1.trackers[SyncItemTicker] = &syncBase{IsUsingREST: true}

	pair2 := currency.NewPair(currency.ETH, currency.USDT)
	agent2 := newCurrencyPairSyncAgent(key.NewExchangeAssetPair(batching.GetName(), asset.Spot, pair2))
	agent2.trackers[SyncItemTicker] = &syncBase{IsUsingREST: true}

	done1 := make(chan struct{})
	go func() {
		m.syncTicker(t.Context(), agent1, batching)
		close(done1)
	}()
	<-batching.started

	done2 := make(chan struct{})
	go func() {
		m.syncTicker(t.Context(), agent2, batching)
		close(done2)
	}()

	close(batching.release)
	<-done1
	<-done2

	require.Equal(t, int32(1), batching.updateCount.Load(), "concurrent workers for the same exchange and asset must only send one batch request")
}

func TestSyncManagerWebsocketUpdateConcurrentAdd(t *testing.T) {
	t.Parallel()
	m, err := SetupSyncManager(&config.SyncManagerConfig{
		SynchronizeTicker:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   &currency.EMPTYFORMAT,
		TimeoutREST:         time.Second,
		TimeoutWebsocket:    time.Second,
	}, &ExchangeManager{}, &config.RemoteControlConfig{}, false)
	require.NoError(t, err, "SetupSyncManager must not error")
	m.started.Store(true)
	m.initSyncStarted.Store(true)
	m.initSyncCompleted.Store(true)

	pair := currency.NewBTCUSDT()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 200 {
			p := currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USD"+string(rune('A'+i%26))))
			m.add(key.NewExchangeAssetPair("Binance", asset.Spot, p), syncBase{IsUsingWebsocket: true})
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			_ = m.WebsocketUpdate("Binance", pair, asset.Spot, SyncItemTicker, nil)
		}
	}()
	wg.Wait()
}

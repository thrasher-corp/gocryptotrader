package engine

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestWebsocketRoutineManagerSetup(t *testing.T) {
	_, err := setupWebsocketRoutineManager(nil, nil, nil, nil, false)
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), (*OrderManager)(nil), nil, nil, false)
	assert.ErrorIs(t, err, errNilCurrencyPairSyncer)

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, nil, false)
	assert.ErrorIs(t, err, errNilCurrencyConfig)

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, &currency.Config{}, true)
	assert.ErrorIs(t, err, errNilCurrencyPairFormat)

	m, err := setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
	assert.NoError(t, err)

	if m == nil {
		t.Error("expecting manager")
	}
}

func TestWebsocketRoutineManagerStart(t *testing.T) {
	var m *WebsocketRoutineManager
	err := m.Start(t.Context())
	assert.ErrorIs(t, err, ErrNilSubsystem)

	cfg := &currency.Config{CurrencyPairFormat: &currency.PairFormat{
		Uppercase: false,
		Delimiter: "-",
	}}
	m, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, cfg, true)
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestWebsocketRoutineManagerIsRunning(t *testing.T) {
	var m *WebsocketRoutineManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
	assert.NoError(t, err)

	if m.IsRunning() {
		t.Error("expected false")
	}

	err = m.Start(t.Context())
	assert.NoError(t, err)

	for m.state.Load() == startingState {
		<-time.After(time.Second / 100)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestWebsocketRoutineManagerStop(t *testing.T) {
	var m *WebsocketRoutineManager
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	m, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestWebsocketRoutineManagerConcurrentStartStop(t *testing.T) {
	cfg := &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}
	for range 128 {
		m, err := setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &SyncManager{}, cfg, false)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = m.Start(t.Context())
		}()
		go func() {
			defer wg.Done()
			_ = m.Stop()
		}()
		wg.Wait()

		if m.state.Load() != stoppedState {
			require.NoError(t, m.Stop())
		}
		assert.Nil(t, m.connectionCancel)
	}
}

func TestWebsocketRoutineManagerHandleData(t *testing.T) {
	exchName := "Bitstamp"
	var wg sync.WaitGroup
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	require.NoError(t, err)

	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	err = om.Start(t.Context())
	assert.NoError(t, err)

	cfg := &currency.Config{CurrencyPairFormat: &currency.PairFormat{
		Uppercase: false,
		Delimiter: "-",
	}}
	m, err := setupWebsocketRoutineManager(em, om, &SyncManager{}, cfg, true)
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.NoError(t, err)

	orderID := "1337"
	err = m.websocketDataHandler(exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = m.websocketDataHandler(exchName, websocket.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = m.websocketDataHandler(exchName, &ticker.Price{
		ExchangeName: exchName,
		Pair:         currency.NewPair(currency.BTC, currency.USDC),
		AssetType:    asset.Spot,
	})
	assert.NoError(t, err)

	err = m.websocketDataHandler(exchName, kline.Item{})
	require.NoError(t, err)
	origOrder := &order.Detail{
		Exchange: exchName,
		OrderID:  orderID,
		Amount:   1337,
		Price:    1337,
	}
	err = m.websocketDataHandler(exchName, origOrder)
	if err != nil {
		t.Error(err)
	}
	// Send it again since it exists now
	err = m.websocketDataHandler(exchName, &order.Detail{
		Exchange: exchName,
		OrderID:  orderID,
		Amount:   1338,
	})
	if err != nil {
		t.Error(err)
	}
	updated, err := m.orderManager.GetByExchangeAndID(origOrder.Exchange, origOrder.OrderID)
	if err != nil {
		t.Error(err)
	}
	if updated.Amount != 1338 {
		t.Error("Bad pipeline")
	}

	err = m.websocketDataHandler(exchName, &order.Detail{
		Exchange: "Bitstamp",
		OrderID:  orderID,
		Status:   order.Active,
	})
	if err != nil {
		t.Error(err)
	}
	updated, err = m.orderManager.GetByExchangeAndID(origOrder.Exchange, origOrder.OrderID)
	if err != nil {
		t.Error(err)
	}
	if updated.Status != order.Active {
		t.Error("Expected order to be modified to Active")
	}

	// Send some gibberish
	err = m.websocketDataHandler(exchName, order.Stop)
	if err != nil {
		t.Error(err)
	}

	err = m.websocketDataHandler(exchName, websocket.UnhandledMessageWarning{
		Message: "there's an issue here's a tissue",
	})
	if err != nil {
		t.Error(err)
	}

	err = m.websocketDataHandler(exchName, &orderbook.Book{
		Exchange: "Bitstamp",
		Pair:     currency.NewBTCUSD(),
	})
	if err != nil {
		t.Error(err)
	}
	err = m.websocketDataHandler(exchName, "this is a test string")
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterWebsocketDataHandlerWithFunctionality(t *testing.T) {
	t.Parallel()
	var m *WebsocketRoutineManager
	err := m.registerWebsocketDataHandler(nil, false)
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = new(WebsocketRoutineManager)
	m.shutdown = make(chan struct{})

	err = m.registerWebsocketDataHandler(nil, false)
	require.ErrorIs(t, err, errNilWebsocketDataHandlerFunction)

	// externally defined capture device
	dataChan := make(chan any)
	fn := func(_ string, data any) error {
		switch data.(type) {
		case string:
			dataChan <- data
		default:
		}
		return nil
	}

	err = m.registerWebsocketDataHandler(fn, true)
	require.NoError(t, err)

	if len(m.dataHandlers) != 1 {
		t.Fatal("unexpected data handlers registered")
	}

	mock := websocket.NewManager()
	m.state.Store(readyState)
	err = m.websocketDataReceiver(mock)
	if err != nil {
		t.Fatal(err)
	}

	err = mock.DataHandler.Send(t.Context(), nil)
	require.NoError(t, err)
	err = mock.DataHandler.Send(t.Context(), 1336)
	require.NoError(t, err)
	err = mock.DataHandler.Send(t.Context(), "intercepted")
	require.NoError(t, err)

	if r := <-dataChan; r != "intercepted" {
		t.Fatal("unexpected value received")
	}

	close(m.shutdown)
	m.wg.Wait()
}

func TestSetWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	var m *WebsocketRoutineManager
	err := m.setWebsocketDataHandler(nil)
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = new(WebsocketRoutineManager)
	m.shutdown = make(chan struct{})

	err = m.setWebsocketDataHandler(nil)
	require.ErrorIs(t, err, errNilWebsocketDataHandlerFunction)

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	require.NoError(t, err)

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	require.NoError(t, err)

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	require.NoError(t, err)

	if len(m.dataHandlers) != 3 {
		t.Fatal("unexpected data handler count")
	}

	err = m.setWebsocketDataHandler(m.websocketDataHandler)
	require.NoError(t, err)

	if len(m.dataHandlers) != 1 {
		t.Fatal("unexpected data handler count")
	}
}

func TestWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSD()
	untracked := currency.NewPair(currency.ETH, currency.USD)
	for _, tc := range []struct {
		name    string
		data    any
		tracked bool
	}{
		{name: "untracked single", data: &ticker.Price{Pair: untracked, AssetType: asset.Spot}},
		{name: "tracked single", data: &ticker.Price{Pair: pair, AssetType: asset.Spot}, tracked: true},
		{name: "untracked batch", data: []ticker.Price{{Pair: untracked, AssetType: asset.Spot}}},
		{name: "untracked before tracked", data: []ticker.Price{{Pair: untracked, AssetType: asset.Spot}, {Pair: pair, AssetType: asset.Spot}}, tracked: true},
		{name: "tracked before untracked", data: []ticker.Price{{Pair: pair, AssetType: asset.Spot}, {Pair: untracked, AssetType: asset.Spot}}, tracked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			syncer := &SyncManager{}
			syncer.config.SynchronizeTicker = true
			syncer.started.Store(true)
			syncer.initSyncStarted.Store(true)
			syncer.initSyncCompleted.Store(true)
			tracked := syncer.add(key.NewExchangeAssetPair(t.Name(), asset.Spot, pair), syncBase{})
			manager := &WebsocketRoutineManager{syncer: syncer}
			require.NoError(t, manager.websocketDataHandler(t.Name(), tc.data), "untracked tickers must be tolerated")
			assert.Equal(t, tc.tracked, tracked.trackers[SyncItemTicker].HaveData, "tracked ticker should be synchronised when present")
			assert.Equal(t, !tc.tracked, tracked.trackers[SyncItemTicker].LastUpdated.IsZero(), "tracked sync timestamp should advance when present")
			_, err := ticker.GetTicker(t.Name(), pair, asset.Spot)
			assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "routine manager should not cache tickers")
		})
	}
}

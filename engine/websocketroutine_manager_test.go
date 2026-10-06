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
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type tickerSyncRecorder struct {
	*SyncManager
	synced    []key.ExchangeAssetPair
	syncErrs  []error
	summaries []tickerSummary
}

type tickerSummary struct {
	price    ticker.Price
	protocol string
	err      error
}

func (r *tickerSyncRecorder) WebsocketUpdate(exchangeName string, p currency.Pair, a asset.Item, syncType syncItemType, err error) error {
	r.synced = append(r.synced, key.NewExchangeAssetPair(exchangeName, a, p))
	err = r.SyncManager.WebsocketUpdate(exchangeName, p, a, syncType, err)
	if err != nil {
		r.syncErrs = append(r.syncErrs, err)
	}
	return err
}

func (r *tickerSyncRecorder) PrintTickerSummary(p *ticker.Price, protocol string, err error) {
	r.summaries = append(r.summaries, tickerSummary{price: *p, protocol: protocol, err: err})
}

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
	testPair := currency.NewPair(currency.NewCode("AAA"), currency.NewCode("BBB"))
	err = m.websocketDataHandler(exchName, &ticker.Price{
		ExchangeName: exchName,
		Pair:         testPair,
		AssetType:    asset.Spot,
	})
	assert.NoError(t, err)
	_, err = ticker.GetTicker(exchName, testPair, asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound)
	err = m.websocketDataHandler(exchName, []ticker.Price{{
		ExchangeName: exchName,
		Pair:         testPair,
		AssetType:    asset.Spot,
	}})
	assert.NoError(t, err, "websocketDataHandler should accept a ticker batch")
	_, err = ticker.GetTicker(exchName, testPair, asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "websocketDataHandler should not store a ticker batch")

	err = m.websocketDataHandler(exchName, kline.Item{})
	require.NoError(t, err)
	err = m.websocketDataHandler(exchName, []futures.Position{{
		Exchange: exchName,
		Asset:    asset.USDTMarginedFutures,
		Pair:     currency.NewBTCUSDT(),
	}})
	require.NoError(t, err)
	err = m.websocketDataHandler(exchName, accounts.SubAccounts{{
		AssetType: asset.USDTMarginedFutures,
		Balances: accounts.CurrencyBalances{
			currency.USDT: {Total: 42},
		},
	}})
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

	classificationError := order.ClassificationError{
		Exchange: "test",
		OrderID:  "one",
		Err:      errors.New("lol"),
	}
	err = m.websocketDataHandler(exchName, classificationError)
	if err == nil {
		t.Error("Expected error")
	}
	assert.ErrorIs(t, err, classificationError.Err)

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

func TestWebsocketDataHandlerTickerBatchSyncsPastUntrackedEntries(t *testing.T) {
	t.Parallel()
	sm := &SyncManager{
		currencyPairs: make(map[key.ExchangeAssetPair]*currencyPairSyncAgent),
		config:        config.SyncManagerConfig{SynchronizeTicker: true},
	}
	sm.started.Store(true)
	sm.initSyncStarted.Store(true)
	sm.initSyncCompleted.Store(true)
	btc, eth := currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)
	tracked := make([]*currencyPairSyncAgent, 0, 2)
	for _, p := range []currency.Pair{btc, eth} {
		c := newCurrencyPairSyncAgent(key.NewExchangeAssetPair(t.Name(), asset.Margin, p))
		c.trackers[SyncItemTicker] = &syncBase{IsUsingREST: true}
		sm.currencyPairs[c.Key] = c
		tracked = append(tracked, c)
	}

	updated := time.UnixMilli(1790000000000)
	batch := []ticker.Price{
		{ExchangeName: t.Name(), Pair: eth, AssetType: asset.Margin},
		{ExchangeName: t.Name(), Pair: btc, AssetType: asset.Spot, Last: 1},
		{
			ExchangeName: t.Name(), Pair: btc, AssetType: asset.Margin, LastUpdated: updated,
			Last: 2, LastSize: 0.2, High: 2.2, Low: 1.8, Bid: 1.9, BidSize: 1.1, Ask: 2.1, AskSize: 1.2,
			BaseVolume: 20, QuoteVolume: 40, Open: 1.7, Close: 1.95, OpenInterest: 50,
			MarkPrice: 2.05, IndexPrice: 2.02, FlashReturnRate: 0.01, BidPeriod: 4, AskPeriod: 30, FlashReturnRateAmount: 100,
		},
		{ExchangeName: t.Name(), Pair: eth, AssetType: asset.Spot, Last: 5},
		{ExchangeName: t.Name(), Pair: btc, AssetType: asset.Futures, Last: 3},
		{ExchangeName: t.Name(), Pair: eth, AssetType: asset.Margin, Bid: 3.9, Ask: 4.1, LastUpdated: updated},
		{ExchangeName: t.Name(), Pair: btc, AssetType: asset.Margin, Last: 6, Bid: 5.9, Ask: 6.1, BaseVolume: 60, LastUpdated: updated.Add(time.Second)},
		{ExchangeName: t.Name(), Pair: btc, AssetType: asset.Margin, Last: 7, Bid: 6.9, Ask: 7.1, BaseVolume: 70, LastUpdated: updated.Add(2 * time.Second)},
		{ExchangeName: t.Name(), Pair: btc, AssetType: asset.Margin, Last: 7, Bid: 6.9, Ask: 7.1, BaseVolume: 70, LastUpdated: updated.Add(2 * time.Second)},
	}
	expSynced := make([]key.ExchangeAssetPair, len(batch))
	for i := range batch {
		expSynced[i] = key.NewExchangeAssetPair(t.Name(), batch[i].AssetType, batch[i].Pair)
	}
	accepted := []int{0, 2, 5, 6, 7, 8}
	expSummaries := make([]tickerSummary, len(accepted))
	for i, x := range accepted {
		expSummaries[i] = tickerSummary{price: batch[x], protocol: "websocket"}
	}
	r := &tickerSyncRecorder{SyncManager: sm}
	m := &WebsocketRoutineManager{syncer: r}
	err := m.websocketDataHandler(t.Name(), batch)
	assert.Equal(t, expSynced, r.synced, "websocketDataHandler should sync every entry in order")
	assert.ErrorIs(t, err, errCouldNotSyncNewData, "websocketDataHandler should report the untracked entries")
	var joined interface{ Unwrap() []error }
	require.ErrorAs(t, err, &joined, "websocketDataHandler must join the sync errors")
	require.Len(t, r.syncErrs, 3, "the syncer must reject the three untracked entries")
	children := joined.Unwrap()
	require.Len(t, children, len(r.syncErrs), "websocketDataHandler must report every rejected entry")
	for i := range children {
		assert.Same(t, r.syncErrs[i], children[i], "websocketDataHandler should report each sync error as returned")
	}
	for _, c := range tracked {
		assert.True(t, c.trackers[SyncItemTicker].IsUsingWebsocket, "websocketDataHandler should sync each tracked entry")
		assert.Zero(t, c.trackers[SyncItemTicker].NumErrors, "websocketDataHandler should sync each tracked entry without an error")
	}
	assert.Equal(t, expSummaries, r.summaries, "websocketDataHandler should summarise only the synced entries, unchanged")
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

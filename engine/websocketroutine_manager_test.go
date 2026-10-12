package engine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/okx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type bookSyncRecorder struct {
	*SyncManager
	err       error
	updates   int
	summaries []asset.Item
	failed    []error
}

func (r *bookSyncRecorder) WebsocketUpdate(name string, pair currency.Pair, a asset.Item, kind syncItemType, err error) error {
	r.updates++
	if r.err != nil {
		return r.err
	}
	err = r.SyncManager.WebsocketUpdate(name, pair, a, kind, err)
	if err != nil {
		r.failed = append(r.failed, err)
	}
	return err
}

func (r *bookSyncRecorder) PrintOrderbookSummary(book *orderbook.Book, _ string, _ error) {
	r.summaries = append(r.summaries, book.Asset)
}

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
			Last: 2, LastSize: 0.2, VolumeWeightedAveragePrice: 1.98, High: 2.2, Low: 1.8, Bid: 1.9, BidSize: 1.1, Ask: 2.1, AskSize: 1.2,
			BaseVolume: 20, QuoteVolume: 40, Open: 1.7, Open24Hour: 1.6, PercentChange24Hour: 25, Close: 1.95,
			OpenInterest: 50, OpenInterestValue: 100, MarkPrice: 2.05, IndexPrice: 2.02,
			FlashReturnRate: 0.01, BidPeriod: 4, AskPeriod: 30, FlashReturnRateAmount: 100,
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
	require.NoError(t, err, "available but untracked tickers must be tolerated")
	require.Len(t, r.syncErrs, 3, "the syncer must reject the three untracked entries")
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

func TestWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		asset   asset.Item
		tracked bool
		syncErr error
		invalid bool
		wantErr error
	}{
		{name: "tracked orderbook", asset: asset.Spot, tracked: true},
		{name: "untracked margin orderbook", asset: asset.Margin},
		{name: "untracked futures orderbook", asset: asset.Futures},
		{name: "wrapped untracked error", asset: asset.Spot, syncErr: fmt.Errorf("wrapped: %w", errCouldNotSyncNewData)},
		{name: "other sync error", asset: asset.Spot, syncErr: ErrSubSystemNotStarted, wantErr: ErrSubSystemNotStarted},
		{name: "wrapped other sync error", asset: asset.Spot, syncErr: fmt.Errorf("wrapped: %w", ErrNilSubsystem), wantErr: ErrNilSubsystem},
		{name: "invalid orderbook", asset: asset.Spot, invalid: true, wantErr: orderbook.ErrOrderbookInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sm := &SyncManager{config: config.SyncManagerConfig{SynchronizeOrderbook: true}}
			sm.started.Store(true)
			sm.initSyncStarted.Store(true)
			sm.initSyncCompleted.Store(true)
			pair := currency.NewBTCUSDT()
			if tc.tracked {
				sm.add(key.NewExchangeAssetPair(t.Name(), tc.asset, pair), syncBase{})
			}
			r := &bookSyncRecorder{SyncManager: sm, err: tc.syncErr}
			m := &WebsocketRoutineManager{syncer: r}
			depth, err := orderbook.DeployDepth(t.Name(), pair, tc.asset)
			require.NoError(t, err)
			book := &orderbook.Book{Exchange: t.Name(), Pair: pair, Asset: tc.asset, LastUpdated: time.Now()}
			depth.AssignOptions(book)
			require.NoError(t, depth.LoadSnapshot(book))
			if tc.invalid {
				require.ErrorIs(t, depth.Invalidate(ErrNilSubsystem), orderbook.ErrOrderbookInvalid)
			}
			require.ErrorIs(t, m.websocketDataHandler(t.Name(), depth), tc.wantErr)
			if tc.invalid {
				assert.Zero(t, r.updates)
			} else {
				assert.Equal(t, 1, r.updates)
			}
			if tc.tracked {
				assert.Equal(t, []asset.Item{tc.asset}, r.summaries)
				assert.True(t, sm.currencyPairs[key.NewExchangeAssetPair(t.Name(), tc.asset, pair)].trackers[SyncItemOrderbook].HaveData)
			} else {
				assert.Empty(t, r.summaries, "untracked and failed books should not produce successful summaries")
			}
		})
	}
	t.Run("OKX books5 available margin fan-out", func(t *testing.T) {
		t.Parallel()
		ex := new(okx.Exchange)
		require.NoError(t, testexch.Setup(ex))
		ex.Name = t.Name()
		pair := currency.NewBTCUSDT()
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			store := &currency.PairStore{Available: currency.Pairs{pair}, AssetEnabled: true, ConfigFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true}, RequestFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true}}
			if a == asset.Spot {
				store.Enabled = currency.Pairs{pair}
			}
			require.NoError(t, ex.CurrencyPairs.Store(a, store))
		}
		require.NoError(t, ex.Websocket.Orderbook.Setup(ex.Name, ex.Websocket.DataHandler, false))
		var sent atomic.Bool
		upgrader := gws.Upgrader{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			conn, err := upgrader.Upgrade(w, request, nil)
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()
			if sent.CompareAndSwap(false, true) {
				assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"arg":{"channel":"books5","instId":"BTC-USDT"},"data":[{"asks":[["101","3","0","1"]],"bids":[["100","2","0","1"]],"ts":"1700000000000"}]}`)))
			}
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}))
		t.Cleanup(server.Close)
		t.Cleanup(func() {
			require.NoError(t, ex.Websocket.Disable())
			if err := ex.Websocket.Shutdown(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected)
			}
		})
		ex.Features.Subscriptions = subscription.List{}
		ex.Websocket.SetSubscriptionsNotRequired()
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+server.URL[len("http"):]))
		require.NoError(t, ex.Websocket.Connect(t.Context()))
		sm := &SyncManager{config: config.SyncManagerConfig{SynchronizeOrderbook: true}}
		sm.started.Store(true)
		sm.initSyncStarted.Store(true)
		sm.initSyncCompleted.Store(true)
		tracked := sm.add(key.NewExchangeAssetPair(ex.Name, asset.Spot, pair), syncBase{})
		r := &bookSyncRecorder{SyncManager: sm}
		m := &WebsocketRoutineManager{syncer: r}
		var received []asset.Item
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		for len(received) < 2 {
			select {
			case payload := <-ex.Websocket.DataHandler.C:
				depth, ok := payload.Data.(*orderbook.Depth)
				if !ok {
					continue
				}
				book, err := depth.Retrieve()
				require.NoError(t, err)
				received = append(received, book.Asset)
				require.NoError(t, m.websocketDataHandler(ex.Name, depth), "real OKX available-only margin pushes must not be logged as sync failures")
			case <-ctx.Done():
				require.FailNow(t, "OKX must fan out the raw snapshot to spot and available-only margin")
			}
		}
		assert.ElementsMatch(t, []asset.Item{asset.Spot, asset.Margin}, received)
		assert.Equal(t, []asset.Item{asset.Spot}, r.summaries)
		require.Len(t, r.failed, 1)
		assert.ErrorIs(t, r.failed[0], errCouldNotSyncNewData)
		assert.True(t, tracked.trackers[SyncItemOrderbook].HaveData)
		for _, a := range received {
			cached, err := orderbook.Get(ex.Name, pair, a)
			require.NoError(t, err)
			assert.Equal(t, 100.0, cached.Bids[0].Price)
		}
	})
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

package engine

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestSetupSyncManager(t *testing.T) {
	t.Parallel()
	_, err := setupSyncManager(nil, nil, nil, false)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("error '%v', expected '%v'", err, common.ErrNilPointer)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{}, nil, nil, false)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, nil, nil, false)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, nil, false)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("error '%v', expected '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.BTC}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, currency.ErrFiatDisplayCurrencyIsNotFiat) {
		t.Errorf("error '%v', expected '%v'", err, currency.ErrFiatDisplayCurrencyIsNotFiat)
	}

	_, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("error '%v', expected '%v'", err, common.ErrNilPointer)
	}

	m, err := setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestSyncManagerStart(t *testing.T) {
	t.Parallel()
	m, err := setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	m.exchangeManager = em
	m.config.SynchronizeContinuously = true
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}

	m = nil
	err = m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestSyncManagerStop(t *testing.T) {
	t.Parallel()
	var m *syncManager
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	m, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
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
	var m *syncManager
	m.PrintTickerSummary(&ticker.Price{}, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	m, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.BTC, currency.USDT),
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
	var m *syncManager
	m.PrintOrderbookSummary(nil, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	m, err = setupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	m.PrintOrderbookSummary(&orderbook.Base{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.USD
	m.PrintOrderbookSummary(&orderbook.Base{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.JPY
	m.PrintOrderbookSummary(&orderbook.Base{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.PrintOrderbookSummary(&orderbook.Base{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", common.ErrNotYetImplemented)

	m.PrintOrderbookSummary(&orderbook.Base{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", errors.New("test"))

	m.PrintOrderbookSummary(nil, "REST", errors.New("test"))
}

func TestRelayWebsocketEvent(t *testing.T) {
	t.Parallel()

	relayWebsocketEvent(nil, "", "", "")
}

func TestWaitForInitialSync(t *testing.T) {
	var m *syncManager
	err := m.WaitForInitialSync()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received %v, but expected: %v", err, ErrNilSubsystem)
	}

	m = &syncManager{}
	err = m.WaitForInitialSync()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received %v, but expected: %v", err, ErrSubSystemNotStarted)
	}

	m.started = 1
	err = m.WaitForInitialSync()
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

func TestSyncManagerWebsocketUpdate(t *testing.T) {
	t.Parallel()
	var m *syncManager
	err := m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received %v, but expected: %v", err, ErrNilSubsystem)
	}

	m = &syncManager{}
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received %v, but expected: %v", err, ErrSubSystemNotStarted)
	}

	m.started = 1
	// not started initial sync
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.initSyncStarted = 1
	// orderbook not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeTicker = true
	// trades not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeTrades = true
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, 1336, nil)
	if !errors.Is(err, errUnknownSyncItem) {
		t.Fatalf("received %v, but expected: %v", err, errUnknownSyncItem)
	}

	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	if !errors.Is(err, errCouldNotSyncNewData) {
		t.Fatalf("received %v, but expected: %v", err, errCouldNotSyncNewData)
	}

	m.add(currencyPairKey{
		AssetType: asset.Spot,
		Pair:      currency.EMPTYPAIR.Format(currency.PairFormat{Uppercase: true}),
	}, syncBase{})
	m.initSyncWG.Add(3)
	// orderbook match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// trades match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

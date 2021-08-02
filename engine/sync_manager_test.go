package engine

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestSetupSyncManager(t *testing.T) {
	t.Parallel()
	_, err := setupSyncManager(&Config{}, nil, nil, nil)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	_, err = setupSyncManager(&Config{SyncTrades: true}, nil, nil, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = setupSyncManager(&Config{SyncTrades: true}, &ExchangeManager{}, nil, nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := setupSyncManager(&Config{SyncTrades: true}, &ExchangeManager{}, nil, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestSyncManagerStart(t *testing.T) {
	t.Parallel()
	m, err := setupSyncManager(&Config{SyncTrades: true}, &ExchangeManager{}, nil, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m.exchangeManager = em
	m.config.SyncContinuously = true
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

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err = setupSyncManager(&Config{SyncTrades: true, SyncContinuously: true}, em, nil, &config.RemoteControlConfig{})
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
	c := printConvertCurrencyFormat(currency.BTC, 1337, currency.USD)
	if c == "" {
		t.Error("expected formatted currency")
	}
}

func TestPrintTickerSummary(t *testing.T) {
	t.Parallel()
	var m *syncManager
	m.PrintTickerSummary(&ticker.Price{}, "REST", nil)

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err = setupSyncManager(&Config{SyncTrades: true, SyncContinuously: true}, em, nil, &config.RemoteControlConfig{})
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

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err = setupSyncManager(&Config{SyncTrades: true, SyncContinuously: true}, em, nil, &config.RemoteControlConfig{})
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

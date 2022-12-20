package synchronize

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
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

	_, err = setupSyncManager(&SyncManagerConfig{}, nil, nil, false)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	_, err = setupSyncManager(&SyncManagerConfig{SynchronizeTrades: true},
		nil, nil, false)
	if !errors.Is(err, subsystem.ErrNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNilExchangeManager)
	}

	_, err = setupSyncManager(&SyncManagerConfig{SynchronizeTrades: true},
		&ExchangeManager{}, nil, false)
	if !errors.Is(err, subsystem.ErrNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNilConfig)
	}

	_, err = setupSyncManager(&SyncManagerConfig{SynchronizeTrades: true},
		&ExchangeManager{},
		&config.RemoteControlConfig{}, true)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("error '%v', expected '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	_, err = setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.BTC,
	}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, currency.ErrFiatDisplayCurrencyIsNotFiat) {
		t.Errorf("error '%v', expected '%v'", err, currency.ErrFiatDisplayCurrencyIsNotFiat)
	}

	_, err = setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
	}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("error '%v', expected '%v'", err, common.ErrNilPointer)
	}

	m, err := setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   &currency.EMPTYFORMAT,
	}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestSyncManagerStart(t *testing.T) {
	t.Parallel()
	m, err := setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   &currency.EMPTYFORMAT,
	}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
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
	m.config.SynchronizeContinuously = true
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, subsystem.ErrAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrAlreadyStarted)
	}

	m = nil
	err = m.Start()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}
}

func TestSyncManagerStop(t *testing.T) {
	t.Parallel()
	var m *syncManager
	err := m.Stop()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err = setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:       true,
		SynchronizeContinuously: true,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &currency.EMPTYFORMAT,
	}, em, &config.RemoteControlConfig{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNotStarted)
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

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err = setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:       true,
		SynchronizeContinuously: true,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &currency.EMPTYFORMAT,
	}, em, &config.RemoteControlConfig{}, false)
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
	m, err = setupSyncManager(&SyncManagerConfig{
		SynchronizeTrades:       true,
		SynchronizeContinuously: true,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay:       &currency.EMPTYFORMAT,
	}, em, &config.RemoteControlConfig{}, false)
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
	if !errors.Is(err, subsystem.ErrNil) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNil)
	}

	m = &syncManager{}
	err = m.WaitForInitialSync()
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNotStarted)
	}

	m.started = 1
	err = m.WaitForInitialSync()
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

func TestSyncManagerUpdate(t *testing.T) {
	t.Parallel()
	var m *syncManager
	err := m.Update("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, subsystem.ErrNil) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNil)
	}

	m = &syncManager{}
	err = m.Update("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNotStarted)
	}

	m.started = 1
	// not started initial sync
	err = m.Update("", currency.EMPTYPAIR, 1, 47, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.initSyncStarted = 1
	// orderbook not enabled
	err = m.Update("", currency.EMPTYPAIR, 1, 1, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.Update("", currency.EMPTYPAIR, 1, 0, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeTicker = true
	// trades not enabled
	err = m.Update("", currency.EMPTYPAIR, 1, 2, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.config.SynchronizeTrades = true
	err = m.Update("", currency.EMPTYPAIR, 1, 1336, nil)
	if !errors.Is(err, errUnknownSyncItem) {
		t.Fatalf("received %v, but expected: %v", err, errUnknownSyncItem)
	}

	err = m.Update("", currency.EMPTYPAIR, 1, 1, nil)
	if !errors.Is(err, errSyncerNotFound) {
		t.Fatalf("received %v, but expected: %v", err, errSyncerNotFound)
	}

	m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
	m.currencyPairs[""] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
	m.currencyPairs[""][nil] = make(map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
	m.currencyPairs[""][nil][nil] = make(map[asset.Item]*currencyPairSyncAgent)
	m.currencyPairs[""][nil][nil][1] = &currencyPairSyncAgent{AssetType: 1}

	m.initSyncWG.Add(3)
	// orderbook match
	err = m.Update("", currency.EMPTYPAIR, 1, 1, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.Update("", currency.EMPTYPAIR, 1, 0, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// trades match
	err = m.Update("", currency.EMPTYPAIR, 1, 2, errors.New("test"))
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

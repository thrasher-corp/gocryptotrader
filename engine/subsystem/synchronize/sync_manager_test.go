package synchronize

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type TestExchangeManager struct {
	hold  []exchange.IBotExchange
	Error bool
}

func (em *TestExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	if em.Error && len(em.hold) == 0 {
		return nil, errExpectedTestError
	}
	return em.hold, nil
}

func (em *TestExchangeManager) GetExchangeByName(string) (exchange.IBotExchange, error) {
	return nil, nil
}

type TestExchange struct {
	exchange.IBotExchange
}

func (e *TestExchange) GetName() string            { return "TEST" }
func (e *TestExchange) SupportsWebsocket() bool    { return true }
func (e *TestExchange) SupportsREST() bool         { return true }
func (e *TestExchange) IsWebsocketEnabled() bool   { return true }
func (e *TestExchange) GetAssetTypes() asset.Items { return asset.Items{asset.Spot} }

type NoProtocolSupported TestExchange

func (e *NoProtocolSupported) SupportsWebsocket() bool { return false }
func (e *NoProtocolSupported) SupportsREST() bool      { return false }

var testName = "test"
var testPair = currency.NewPair(currency.BTC, currency.USD)
var errExpectedTestError = errors.New("expected test error")

func TestNewManager(t *testing.T) {
	t.Parallel()
	_, err := NewManager(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("error '%v', expected '%v'", err, common.ErrNilPointer)
	}

	cfg := &ManagerConfig{}
	_, err = NewManager(cfg)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Fatalf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	cfg.SynchronizeOrderbook = true
	_, err = NewManager(cfg)
	if !errors.Is(err, subsystem.ErrNilExchangeManager) {
		t.Fatalf("error '%v', expected '%v'", err, subsystem.ErrNilExchangeManager)
	}

	cfg.ExchangeManager = &TestExchangeManager{}
	_, err = NewManager(cfg)
	if !errors.Is(err, subsystem.ErrNilConfig) {
		t.Fatalf("error '%v', expected '%v'", err, subsystem.ErrNilConfig)
	}

	cfg.RemoteConfig = &config.RemoteControlConfig{}
	_, err = NewManager(cfg)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("error '%v', expected '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	cfg.FiatDisplayCurrency = currency.BTC
	_, err = NewManager(cfg)
	if !errors.Is(err, currency.ErrFiatDisplayCurrencyIsNotFiat) {
		t.Fatalf("error '%v', expected '%v'", err, currency.ErrFiatDisplayCurrencyIsNotFiat)
	}

	cfg.FiatDisplayCurrency = currency.USD
	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	var m *Manager
	if m.IsRunning() {
		t.Fatalf("received: '%v', but expected '%v'", m.IsRunning(), false)
	}

	m = &Manager{}
	if m.IsRunning() {
		t.Fatalf("received: '%v', but expected '%v'", m.IsRunning(), false)
	}

	m.started = 1
	if !m.IsRunning() {
		t.Fatalf("received: '%v', but expected '%v'", m.IsRunning(), true)
	}
}

func TestManagerStart(t *testing.T) {
	t.Parallel()
	cfg := &ManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   currency.EMPTYFORMAT,
		ExchangeManager:     &TestExchangeManager{},
		RemoteConfig:        &config.RemoteControlConfig{},
	}

	var m *Manager
	err := m.Start()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}

	m, err = NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, subsystem.ErrAlreadyStarted) {
		t.Fatalf("error '%v', expected '%v'", err, subsystem.ErrAlreadyStarted)
	}
}

func TestSyncManagerStop(t *testing.T) {
	t.Parallel()
	var m *Manager
	err := m.Stop()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}

	cfg := &ManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   currency.EMPTYFORMAT,
		ExchangeManager:     &TestExchangeManager{},
		RemoteConfig:        &config.RemoteControlConfig{},
	}

	m, err = NewManager(cfg)
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

func TestSyncManagerUpdate(t *testing.T) {
	t.Parallel()
	var m *Manager
	err := m.Update("", "", currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, subsystem.ErrNil) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNil)
	}

	m = &Manager{}
	err = m.Update("", "", currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNotStarted)
	}

	m.started = 1
	err = m.Update("", "", currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received %v, but expected: %v", err, errExchangeNameUnset)
	}

	err = m.Update(testName, "", currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, errProtocolUnset) {
		t.Fatalf("received %v, but expected: %v", err, errProtocolUnset)
	}

	err = m.Update(testName, WebsocketUpdate, currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received %v, but expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	err = m.Update(testName, WebsocketUpdate, testPair, 0, 47, nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received %v, but expected: %v", err, asset.ErrNotSupported)
	}

	// not started initial sync
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, 47, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.initSyncStarted = 1
	// orderbook not enabled
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, 1, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, 0, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.SynchronizeTicker = true
	// trades not enabled
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, 2, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.SynchronizeTrades = true
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, 1336, nil)
	if !errors.Is(err, errUnknownSyncType) {
		t.Fatalf("received %v, but expected: %v", err, errUnknownSyncType)
	}

	err = m.Update("bruh?", "bruh?", currency.NewPair(currency.NOO, currency.BRAIN), asset.Spot, 1, nil)
	if !errors.Is(err, errAgentNotFound) {
		t.Fatalf("received %v, but expected: %v", err, errAgentNotFound)
	}

	m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Agent)
	m.currencyPairs[testName] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Agent)
	m.currencyPairs[testName][currency.BTC.Item] = make(map[*currency.Item]map[asset.Item]*Agent)
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item] = make(map[asset.Item]*Agent)
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item][asset.Spot] = &Agent{AssetType: 1}

	m.initSyncWG.Add(3)
	// orderbook match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Orderbook), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Ticker), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// trades match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Trade), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// Should not call done

	// orderbook match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Orderbook), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Ticker), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// trades match
	err = m.Update(testName, WebsocketUpdate, testPair, asset.Spot, int(Trade), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

func TestCheckAllExchangeAssets(t *testing.T) {
	t.Parallel()

	cfg := &ManagerConfig{
		SynchronizeTrades:   true,
		FiatDisplayCurrency: currency.USD,
		PairFormatDisplay:   currency.EMPTYFORMAT,
		ExchangeManager:     &TestExchangeManager{Error: true},
		RemoteConfig:        &config.RemoteControlConfig{},
	}

	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.checkAllExchangeAssets()
	if !errors.Is(err, errExpectedTestError) {
		t.Fatalf("received %v, but expected: %v", err, errExpectedTestError)
	}

}

// func TestPrintCurrencyFormat(t *testing.T) {
// 	t.Parallel()
// 	c := printCurrencyFormat(1337, currency.BTC)
// 	if c == "" {
// 		t.Error("expected formatted currency")
// 	}
// }

// func TestPrintConvertCurrencyFormat(t *testing.T) {
// 	t.Parallel()
// 	c := printConvertCurrencyFormat(1337, currency.BTC, currency.USD)
// 	if c == "" {
// 		t.Error("expected formatted currency")
// 	}
// }

// func TestPrintTickerSummary(t *testing.T) {
// 	t.Parallel()
// 	var m *syncManager
// 	m.PrintTickerSummary(&ticker.Price{}, "REST", nil)

// 	em := SetupExchangeManager()
// 	exch, err := em.NewExchangeByName("Bitstamp")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	exch.SetDefaults()
// 	em.Add(exch)
// 	m, err = setupSyncManager(&SyncManagerConfig{
// 		SynchronizeTrades:       true,
// 		SynchronizeContinuously: true,
// 		FiatDisplayCurrency:     currency.USD,
// 		PairFormatDisplay:       &currency.EMPTYFORMAT,
// 	}, em, &config.RemoteControlConfig{}, false)
// 	if !errors.Is(err, nil) {
// 		t.Errorf("error '%v', expected '%v'", err, nil)
// 	}
// 	atomic.StoreInt32(&m.started, 1)
// 	m.PrintTickerSummary(&ticker.Price{
// 		Pair: currency.NewPair(currency.BTC, currency.USDT),
// 	}, "REST", nil)
// 	m.fiatDisplayCurrency = currency.USD
// 	m.PrintTickerSummary(&ticker.Price{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", nil)

// 	m.fiatDisplayCurrency = currency.JPY
// 	m.PrintTickerSummary(&ticker.Price{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", nil)

// 	m.PrintTickerSummary(&ticker.Price{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", errors.New("test"))

// 	m.PrintTickerSummary(&ticker.Price{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", common.ErrNotYetImplemented)
// }

// func TestPrintOrderbookSummary(t *testing.T) {
// 	t.Parallel()
// 	var m *syncManager
// 	m.PrintOrderbookSummary(nil, "REST", nil)

// 	em := SetupExchangeManager()
// 	exch, err := em.NewExchangeByName("Bitstamp")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	exch.SetDefaults()
// 	em.Add(exch)
// 	m, err = setupSyncManager(&SyncManagerConfig{
// 		SynchronizeTrades:       true,
// 		SynchronizeContinuously: true,
// 		FiatDisplayCurrency:     currency.USD,
// 		PairFormatDisplay:       &currency.EMPTYFORMAT,
// 	}, em, &config.RemoteControlConfig{}, false)
// 	if !errors.Is(err, nil) {
// 		t.Errorf("error '%v', expected '%v'", err, nil)
// 	}
// 	atomic.StoreInt32(&m.started, 1)
// 	m.PrintOrderbookSummary(&orderbook.Base{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", nil)

// 	m.fiatDisplayCurrency = currency.USD
// 	m.PrintOrderbookSummary(&orderbook.Base{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", nil)

// 	m.fiatDisplayCurrency = currency.JPY
// 	m.PrintOrderbookSummary(&orderbook.Base{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", nil)

// 	m.PrintOrderbookSummary(&orderbook.Base{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", common.ErrNotYetImplemented)

// 	m.PrintOrderbookSummary(&orderbook.Base{
// 		Pair: currency.NewPair(currency.AUD, currency.USD),
// 	}, "REST", errors.New("test"))

// 	m.PrintOrderbookSummary(nil, "REST", errors.New("test"))
// }

// func TestRelayWebsocketEvent(t *testing.T) {
// 	t.Parallel()

// 	relayWebsocketEvent(nil, "", "", "")
// }

// func TestWaitForInitialSync(t *testing.T) {
// 	var m *syncManager
// 	err := m.WaitForInitialSync()
// 	if !errors.Is(err, subsystem.ErrNil) {
// 		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNil)
// 	}

// 	m = &syncManager{}
// 	err = m.WaitForInitialSync()
// 	if !errors.Is(err, subsystem.ErrNotStarted) {
// 		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNotStarted)
// 	}

// 	m.started = 1
// 	err = m.WaitForInitialSync()
// 	if !errors.Is(err, nil) {
// 		t.Fatalf("received %v, but expected: %v", err, nil)
// 	}
// }

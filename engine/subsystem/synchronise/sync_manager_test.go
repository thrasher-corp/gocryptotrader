package synchronise

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var (
	errGetExchanges    = errors.New("get exchanges error")
	errGetWebsocket    = errors.New("get websocket error")
	errGetEnabledPairs = errors.New("get enabled pairs error")
	testName           = "test"
	testPair           = currency.NewPair(currency.BTC, currency.USD)
)

type TestExchangeManager struct {
	hold  []exchange.IBotExchange
	Error bool
}

func (em *TestExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	if !em.Error {
		return em.hold, nil
	}
	return nil, errGetExchanges
}
func (em *TestExchangeManager) GetExchangeByName(string) (exchange.IBotExchange, error) {
	return nil, nil
}

type testStreamer stream.Websocket

func (e *testStreamer) IsConnected() bool { return true }

type TestExchange struct{ exchange.IBotExchange }

func (e *TestExchange) GetName() string                           { return "test" }
func (e *TestExchange) SupportsWebsocket() bool                   { return true }
func (e *TestExchange) SupportsREST() bool                        { return true }
func (e *TestExchange) IsWebsocketEnabled() bool                  { return true }
func (e *TestExchange) IsAssetWebsocketSupported(asset.Item) bool { return true }
func (e *TestExchange) GetAssetTypes(bool) asset.Items            { return asset.Items{asset.Spot} }
func (e *TestExchange) GetWebsocket() (*stream.Websocket, error) {
	return (*stream.Websocket)(&testStreamer{}), nil
}
func (e *TestExchange) GetEnabledPairs(asset.Item) (currency.Pairs, error) {
	return currency.Pairs{testPair}, nil
}

type ProblemWithGettingWebsocketP struct {
	TestExchange
}

func (e *ProblemWithGettingWebsocketP) GetWebsocket() (*stream.Websocket, error) {
	return nil, errGetWebsocket
}

type ProblemWithGettingEnabledPairs struct {
	TestExchange
}

func (e *ProblemWithGettingEnabledPairs) GetEnabledPairs(asset.Item) (currency.Pairs, error) {
	return nil, errGetEnabledPairs
}

func TestNewManager(t *testing.T) {
	t.Parallel()
	_, err := NewManager(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("error '%v', expected '%v'", err, common.ErrNilPointer)
	}

	cfg := &ManagerConfig{}
	_, err = NewManager(cfg)
	if !errors.Is(err, ErrNoItemsEnabled) {
		t.Fatalf("error '%v', expected '%v'", err, ErrNoItemsEnabled)
	}

	cfg.SynchronizeOrderbook = true
	_, err = NewManager(cfg)
	if !errors.Is(err, subsystem.ErrNilExchangeManager) {
		t.Fatalf("error '%v', expected '%v'", err, subsystem.ErrNilExchangeManager)
	}

	cfg.ExchangeManager = &TestExchangeManager{}
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
		SynchronizeOrderbook: true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{},
		WebsocketRPCEnabled:  true,
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
		SynchronizeOrderbook: true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{},
		WebsocketRPCEnabled:  true,
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

	err = m.Update(testName, subsystem.Websocket, currency.EMPTYPAIR, 0, 47, nil)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received %v, but expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	err = m.Update(testName, subsystem.Websocket, testPair, 0, 47, nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received %v, but expected: %v", err, asset.ErrNotSupported)
	}

	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, 47, nil)
	if !errors.Is(err, errUnknownSyncType) {
		t.Fatalf("received %v, but expected: %v", err, errUnknownSyncType)
	}

	// not started initial sync
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Orderbook, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// orderbook not enabled
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Orderbook, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Ticker, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	m.SynchronizeTicker = true
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, 1336, nil)
	if !errors.Is(err, errUnknownSyncType) {
		t.Fatalf("received %v, but expected: %v", err, errUnknownSyncType)
	}

	m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]map[subsystem.SynchronizationType]*Agent)
	m.currencyPairs[testName] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]map[subsystem.SynchronizationType]*Agent)
	m.currencyPairs[testName][currency.BTC.Item] = make(map[*currency.Item]map[asset.Item]map[subsystem.SynchronizationType]*Agent)
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item] = make(map[asset.Item]map[subsystem.SynchronizationType]*Agent)
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item][asset.Spot] = make(map[subsystem.SynchronizationType]*Agent)
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item][asset.Spot][subsystem.Orderbook] = &Agent{Asset: 1}
	m.currencyPairs[testName][currency.BTC.Item][currency.USD.Item][asset.Spot][subsystem.Ticker] = &Agent{Asset: 1}

	m.initSyncWG.Add(3)
	// orderbook match
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Orderbook, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Ticker, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// Should not call done

	// orderbook match
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Orderbook, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	// ticker match
	err = m.Update(testName, subsystem.Websocket, testPair, asset.Spot, subsystem.Ticker, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}
}

func TestCheckAllExchangeAssets(t *testing.T) {
	t.Parallel()

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.checkAllExchangeAssets()
	if !errors.Is(err, errGetExchanges) {
		t.Fatalf("received %v, but expected: %v", err, errGetExchanges)
	}

	m.ExchangeManager = &TestExchangeManager{hold: []exchange.IBotExchange{&ProblemWithGettingWebsocketP{}}}
	_, err = m.checkAllExchangeAssets()
	if !errors.Is(err, errGetWebsocket) {
		t.Fatalf("received %v, but expected: %v", err, errGetWebsocket)
	}

	m.ExchangeManager = &TestExchangeManager{hold: []exchange.IBotExchange{&ProblemWithGettingEnabledPairs{}}}
	_, err = m.checkAllExchangeAssets()
	if !errors.Is(err, errGetEnabledPairs) {
		t.Fatalf("received %v, but expected: %v", err, errGetEnabledPairs)
	}

	// No sync agents enabled should just return the lowest protocol time.
	m.ExchangeManager = &TestExchangeManager{hold: []exchange.IBotExchange{&TestExchange{}}}
	wait, err := m.checkAllExchangeAssets()
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, but expected: %v", err, nil)
	}

	if wait != time.Second {
		t.Fatalf("received %v, but expected: %v", wait, time.Second)
	}
}

func TestGetSmallestTimeout(t *testing.T) {
	t.Parallel()

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	if to := m.getSmallestTimeout(); to != time.Second {
		t.Fatalf("received %v, but expected: %v", to, time.Second)
	}

	cfg.TimeoutREST = time.Second * 3
	m, err = NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	if to := m.getSmallestTimeout(); to != time.Second*2 {
		t.Fatalf("received %v, but expected: %v", to, time.Second*2)
	}
}

func TestCheckSyncItems(t *testing.T) {
	t.Parallel()

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		SynchronizeTicker:    true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	wait := m.getSmallestTimeout()
	wait = m.checkSyncItems(&TestExchange{}, testPair, asset.Spot, true, wait)
	if wait != time.Second {
		t.Fatalf("received %v, but expected: %v", wait, time.Second)
	}

	wait = m.checkSyncItems(&TestExchange{}, testPair, asset.Spot, true, wait)
	if wait < time.Second {
		t.Fatalf("received %v, but expected less than: %v", wait, time.Second)
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
	var m *Manager
	m.PrintTickerSummary(&ticker.Price{}, subsystem.Rest, nil)

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		SynchronizeTicker:    true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	var err error
	m, err = NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)
	m.PrintTickerSummary(&ticker.Price{Pair: pair}, subsystem.Rest, nil)
	m.FiatDisplayCurrency = currency.USD
	pair = currency.NewPair(currency.AUD, currency.USD)
	m.PrintTickerSummary(&ticker.Price{Pair: pair}, subsystem.Rest, nil)
	m.FiatDisplayCurrency = currency.JPY
	m.PrintTickerSummary(&ticker.Price{Pair: pair}, subsystem.Rest, nil)
	m.PrintTickerSummary(&ticker.Price{Pair: pair}, subsystem.Rest, errors.New("test"))
	m.PrintTickerSummary(&ticker.Price{Pair: pair}, subsystem.Rest, common.ErrNotYetImplemented)
}

func TestPrintOrderbookSummary(t *testing.T) {
	t.Parallel()
	var m *Manager
	m.PrintOrderbookSummary(nil, subsystem.Rest, nil)

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		SynchronizeTicker:    true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	var err error
	m, err = NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	pair := currency.NewPair(currency.AUD, currency.USD)
	m.PrintOrderbookSummary(&orderbook.Base{Pair: pair}, subsystem.Rest, nil)

	m.FiatDisplayCurrency = currency.USD
	m.PrintOrderbookSummary(&orderbook.Base{Pair: pair}, subsystem.Rest, nil)
	m.FiatDisplayCurrency = currency.JPY
	m.PrintOrderbookSummary(&orderbook.Base{Pair: pair}, subsystem.Rest, nil)
	m.PrintOrderbookSummary(&orderbook.Base{Pair: pair}, subsystem.Rest, common.ErrNotYetImplemented)
	m.PrintOrderbookSummary(&orderbook.Base{Pair: pair}, subsystem.Rest, errors.New("test"))
	m.PrintOrderbookSummary(nil, subsystem.Rest, errors.New("test"))
}

func TestRelayWebsocketEvent(t *testing.T) {
	t.Parallel()

	cfg := &ManagerConfig{
		SynchronizeOrderbook: true,
		SynchronizeTicker:    true,
		FiatDisplayCurrency:  currency.USD,
		PairFormatDisplay:    currency.EMPTYFORMAT,
		ExchangeManager:      &TestExchangeManager{Error: true},
		WebsocketRPCEnabled:  true,
		TimeoutREST:          time.Second,
		TimeoutWebsocket:     time.Second * 2,
	}

	m, err := NewManager(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.relayWebsocketEvent(nil, "", "", "")
}

func TestWaitForInitialSync(t *testing.T) {
	var m *Manager
	err := m.WaitForInitialSync()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Fatalf("received %v, but expected: %v", err, subsystem.ErrNil)
	}

	m = &Manager{}
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

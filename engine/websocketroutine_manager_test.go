package engine

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestWebsocketRoutineManagerSetup(t *testing.T) {
	_, err := setupWebsocketRoutineManager(nil, nil, nil, nil, false)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), nil, nil, nil, false)
	if !errors.Is(err, errNilOrderManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilOrderManager)
	}

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, nil, nil, false)
	if !errors.Is(err, errNilCurrencyPairSyncer) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyPairSyncer)
	}
	_, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, nil, false)
	if !errors.Is(err, errNilCurrencyConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyConfig)
	}

	_, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, &currency.Config{}, true)
	if !errors.Is(err, errNilCurrencyPairFormat) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyPairFormat)
	}

	m, err := setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expecting manager")
	}
}

func TestWebsocketRoutineManagerStart(t *testing.T) {
	var m *WebsocketRoutineManager
	err := m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
	cfg := &currency.Config{CurrencyPairFormat: &currency.PairFormat{
		Uppercase: false,
		Delimiter: "-",
	}}
	m, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}
}

func TestWebsocketRoutineManagerIsRunning(t *testing.T) {
	var m *WebsocketRoutineManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m.IsRunning() {
		t.Error("expected false")
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	for atomic.LoadInt32(&m.state) == startingState {
		<-time.After(time.Second / 100)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestWebsocketRoutineManagerStop(t *testing.T) {
	var m *WebsocketRoutineManager
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = setupWebsocketRoutineManager(NewExchangeManager(), &OrderManager{}, &syncManager{}, &currency.Config{CurrencyPairFormat: &currency.PairFormat{}}, false)
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

func TestWebsocketRoutineManagerHandleData(t *testing.T) {
	var exchName = "Bitstamp"
	var wg sync.WaitGroup
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, 0)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = om.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	cfg := &currency.Config{CurrencyPairFormat: &currency.PairFormat{
		Uppercase: false,
		Delimiter: "-",
	}}
	m, err := setupWebsocketRoutineManager(em, om, &syncManager{}, cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var orderID = "1337"
	err = m.websocketDataHandler(exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = m.websocketDataHandler(exchName, stream.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = m.websocketDataHandler(exchName, &ticker.Price{
		ExchangeName: exchName,
		Pair:         currency.NewPair(currency.BTC, currency.USDC),
		AssetType:    asset.Spot,
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.websocketDataHandler(exchName, stream.KlineData{})
	if err != nil {
		t.Error(err)
	}
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

	err = m.websocketDataHandler(exchName, stream.UnhandledMessageWarning{
		Message: "there's an issue here's a tissue"},
	)
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
	if !errors.Is(err, classificationError.Err) {
		t.Errorf("error '%v', expected '%v'", err, classificationError.Err)
	}

	err = m.websocketDataHandler(exchName, &orderbook.Base{
		Exchange: "Bitstamp",
		Pair:     currency.NewPair(currency.BTC, currency.USD),
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
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = new(WebsocketRoutineManager)
	m.shutdown = make(chan struct{})

	err = m.registerWebsocketDataHandler(nil, false)
	if !errors.Is(err, errNilWebsocketDataHandlerFunction) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilWebsocketDataHandlerFunction)
	}

	// externally defined capture device
	dataChan := make(chan interface{})
	fn := func(_ string, data interface{}) error {
		switch data.(type) {
		case string:
			dataChan <- data
		default:
		}
		return nil
	}

	err = m.registerWebsocketDataHandler(fn, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(m.dataHandlers) != 1 {
		t.Fatal("unexpected data handlers registered")
	}

	mock := stream.New()
	mock.ToRoutine = make(chan interface{})
	m.state = readyState
	err = m.websocketDataReceiver(mock)
	if err != nil {
		t.Fatal(err)
	}

	mock.ToRoutine <- nil
	mock.ToRoutine <- 1336
	mock.ToRoutine <- "intercepted"

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
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = new(WebsocketRoutineManager)
	m.shutdown = make(chan struct{})

	err = m.setWebsocketDataHandler(nil)
	if !errors.Is(err, errNilWebsocketDataHandlerFunction) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilWebsocketDataHandlerFunction)
	}

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = m.registerWebsocketDataHandler(m.websocketDataHandler, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(m.dataHandlers) != 3 {
		t.Fatal("unexpected data handler count")
	}

	err = m.setWebsocketDataHandler(m.websocketDataHandler)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(m.dataHandlers) != 1 {
		t.Fatal("unexpected data handler count")
	}
}

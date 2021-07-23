package engine

import (
	"errors"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
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

	_, err = setupWebsocketRoutineManager(SetupExchangeManager(), nil, nil, nil, false)
	if !errors.Is(err, errNilOrderManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilOrderManager)
	}

	_, err = setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, nil, nil, false)
	if !errors.Is(err, errNilCurrencyPairSyncer) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyPairSyncer)
	}
	_, err = setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, nil, false)
	if !errors.Is(err, errNilCurrencyConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyConfig)
	}

	_, err = setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, &config.CurrencyConfig{}, true)
	if !errors.Is(err, errNilCurrencyPairFormat) {
		t.Errorf("error '%v', expected '%v'", err, errNilCurrencyPairFormat)
	}

	m, err := setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, &config.CurrencyConfig{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expecting manager")
	}
}

func TestWebsocketRoutineManagerStart(t *testing.T) {
	var m *websocketRoutineManager
	err := m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
	cfg := &config.CurrencyConfig{CurrencyPairFormat: &config.CurrencyPairFormatConfig{
		Uppercase: false,
		Delimiter: "-",
	}}
	m, err = setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, cfg, true)
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
	var m *websocketRoutineManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, &config.CurrencyConfig{}, false)
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
	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestWebsocketRoutineManagerStop(t *testing.T) {
	var m *websocketRoutineManager
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = setupWebsocketRoutineManager(SetupExchangeManager(), &OrderManager{}, &syncManager{}, &config.CurrencyConfig{}, false)
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
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	em.Add(exch)

	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = om.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	cfg := &config.CurrencyConfig{CurrencyPairFormat: &config.CurrencyPairFormatConfig{
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
	err = m.WebsocketDataHandler(exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = m.WebsocketDataHandler(exchName, nil)
	if err == nil {
		t.Error("Expected nil data error")
	}
	err = m.WebsocketDataHandler(exchName, stream.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = m.WebsocketDataHandler(exchName, &ticker.Price{
		ExchangeName: exchName,
		Pair:         currency.NewPair(currency.BTC, currency.USDC),
		AssetType:    asset.Spot,
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.WebsocketDataHandler(exchName, stream.KlineData{})
	if err != nil {
		t.Error(err)
	}
	origOrder := &order.Detail{
		Exchange: exchName,
		ID:       orderID,
		Amount:   1337,
		Price:    1337,
	}
	err = m.WebsocketDataHandler(exchName, origOrder)
	if err != nil {
		t.Error(err)
	}
	// Send it again since it exists now
	err = m.WebsocketDataHandler(exchName, &order.Detail{
		Exchange: exchName,
		ID:       orderID,
		Amount:   1338,
	})
	if err != nil {
		t.Error(err)
	}
	updated, err := m.orderManager.GetByExchangeAndID(origOrder.Exchange, origOrder.ID)
	if err != nil {
		t.Error(err)
	}
	if updated.Amount != 1338 {
		t.Error("Bad pipeline")
	}

	err = m.WebsocketDataHandler(exchName, &order.Modify{
		Exchange: "Bitstamp",
		ID:       orderID,
		Status:   order.Active,
	})
	if err != nil {
		t.Error(err)
	}
	updated, err = m.orderManager.GetByExchangeAndID(origOrder.Exchange, origOrder.ID)
	if err != nil {
		t.Error(err)
	}
	if updated.Status != order.Active {
		t.Error("Expected order to be modified to Active")
	}

	// Send some gibberish
	err = m.WebsocketDataHandler(exchName, order.Stop)
	if err != nil {
		t.Error(err)
	}

	err = m.WebsocketDataHandler(exchName, stream.UnhandledMessageWarning{
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
	err = m.WebsocketDataHandler(exchName, classificationError)
	if err == nil {
		t.Error("Expected error")
	}
	if !errors.Is(err, classificationError.Err) {
		t.Errorf("error '%v', expected '%v'", err, classificationError.Err)
	}

	err = m.WebsocketDataHandler(exchName, &orderbook.Base{
		Exchange: "Bitstamp",
		Pair:     currency.NewPair(currency.BTC, currency.USD),
	})
	if err != nil {
		t.Error(err)
	}
	err = m.WebsocketDataHandler(exchName, "this is a test string")
	if err != nil {
		t.Error(err)
	}
}

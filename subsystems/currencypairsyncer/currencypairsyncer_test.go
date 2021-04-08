package currencypairsyncer

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

func TestNewCurrencyPairSyncer(t *testing.T) {
	_, err := Setup(Config{}, nil, nil, nil)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	_, err = Setup(Config{SyncTrades: true}, nil, nil, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = Setup(Config{SyncTrades: true}, &exchangemanager.Manager{}, nil, nil)
	if !errors.Is(err, errNilWebsocketDataReceiver) {
		t.Errorf("error '%v', expected '%v'", err, errNilWebsocketDataReceiver)
	}

	_, err = Setup(Config{SyncTrades: true}, &exchangemanager.Manager{}, &fakeBot{}, nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := Setup(Config{SyncTrades: true}, &exchangemanager.Manager{}, &fakeBot{}, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

// fakeBot is a basic implementation of the iBot interface used for testing
type fakeBot struct{}

// SetupExchanges is a basic implementation of the iBot interface used for testing
func (f *fakeBot) WebsocketDataReceiver(ws *stream.Websocket) {}

func TestStart(t *testing.T) {
	m, err := Setup(Config{SyncTrades: true}, &exchangemanager.Manager{}, &fakeBot{}, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	em := exchangemanager.Setup()
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
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}

	m = nil
	err = m.Start()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

/*
func TestWebsocketDataHandlerProcess(t *testing.T) {
	ws := sharedtestvalues.NewTestWebsocket()
	b := ordermanager.OrdersSetup(t)
	go b.WebsocketDataReceiver(ws)
	ws.DataHandler <- "string"
	time.Sleep(time.Second)
	close(shutdowner)
}

func TestHandleData(t *testing.T) {
	b := ordermanager.OrdersSetup(t)
	var exchName = "exch"
	var orderID = "testOrder.Detail"
	err := b.WebsocketDataHandler(exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = b.WebsocketDataHandler(exchName, nil)
	if err == nil {
		t.Error("Expected nil data error")
	}
	err = b.WebsocketDataHandler(exchName, stream.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = b.WebsocketDataHandler(exchName, &ticker.Price{})
	if err != nil {
		t.Error(err)
	}
	err = b.WebsocketDataHandler(exchName, stream.KlineData{})
	if err != nil {
		t.Error(err)
	}
	origOrder := &order.Detail{
		Exchange: exchangemanager.fakePassExchange,
		ID:       orderID,
		Amount:   1337,
		Price:    1337,
	}
	err = b.WebsocketDataHandler(exchName, origOrder)
	if err != nil {
		t.Error(err)
	}
	// Send it again since it exists now
	err = b.WebsocketDataHandler(exchName, &order.Detail{
		Exchange: exchangemanager.fakePassExchange,
		ID:       orderID,
		Amount:   1338,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Amount != 1338 {
		t.Error("Bad pipeline")
	}

	err = b.WebsocketDataHandler(exchName, &order.Modify{
		Exchange: exchangemanager.fakePassExchange,
		ID:       orderID,
		Status:   order.Active,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Status != order.Active {
		t.Error("Expected order to be modified to Active")
	}

	err = b.WebsocketDataHandler(exchName, &order.Cancel{
		Exchange: exchangemanager.fakePassExchange,
		ID:       orderID,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Status != order.Cancelled {
		t.Error("Expected order status to be cancelled")
	}
	// Send some gibberish
	err = b.WebsocketDataHandler(exchName, order.Stop)
	if err != nil {
		t.Error(err)
	}

	err = b.WebsocketDataHandler(exchName, stream.UnhandledMessageWarning{
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
	err = b.WebsocketDataHandler(exchName, classificationError)
	if err == nil {
		t.Error("Expected error")
	}
	if err != nil && err.Error() != classificationError.Error() {
		t.Errorf("Problem formatting error. Expected %v Received %v", classificationError.Error(), err.Error())
	}

	err = b.WebsocketDataHandler(exchName, &orderbook.Base{
		ExchangeName: exchangemanager.fakePassExchange,
		Pair:         currency.NewPair(currency.BTC, currency.USD),
	})
	if err != nil {
		t.Error(err)
	}
	err = b.WebsocketDataHandler(exchName, "this is a test string")
	if err != nil {
		t.Error(err)
	}
}
*/

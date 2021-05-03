package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestWebsocketDataHandlerProcess(t *testing.T) {
	ws := sharedtestvalues.NewTestWebsocket()
	b := OrdersSetup(t)
	go b.WebsocketDataReceiver(ws)
	ws.DataHandler <- "string"
	time.Sleep(time.Second)
	close(shutdowner)
}

func TestHandleData(t *testing.T) {
	b := OrdersSetup(t)
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
		Exchange: fakePassExchange,
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
		Exchange: fakePassExchange,
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
		Exchange: fakePassExchange,
		ID:       orderID,
		Status:   order.Active,
	})
	if err != nil {
		t.Error(err)
	}
	if origOrder.Status != order.Active {
		t.Error("Expected order to be modified to Active")
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
		Exchange: fakePassExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USD),
	})
	if err != nil {
		t.Error(err)
	}
	err = b.WebsocketDataHandler(exchName, "this is a test string")
	if err != nil {
		t.Error(err)
	}
}

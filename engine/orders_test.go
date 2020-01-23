package engine

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var oManager orderManager
var setupRan bool

func OrdersSetup(t *testing.T) {
	if !setupRan {
		SetupTest(t)
		err := oManager.Start()
		if err != nil {
			t.Fatal(err)
		}
		setupRan = true
	}
}

func TestOrdersGet(t *testing.T) {
	OrdersSetup(t)
	if oManager.orderStore.get() == nil {
		t.Error("orderStore not established")
	}
}

func TestOrdersAdd(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange: "Bitstamp",
		ID:       "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = oManager.orderStore.Add(&order.Detail{
		Exchange: "testTest",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error from non existent exchange")
	}

	err = oManager.orderStore.Add(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	err = oManager.orderStore.Add(&order.Detail{
		Exchange: "Bitstamp",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange:        "Bitstamp",
		ID:              "TestGetByInternalOrderID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := oManager.orderStore.GetByInternalOrderID("internalTest")
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" {
		t.Error("Expected to retrieve order")
	}

	_, err = oManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange:        "Bitstamp",
		ID:              "TestGetByExchangeAndID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := oManager.orderStore.GetByExchangeAndID("Bitstamp", "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.ID != "TestGetByExchangeAndID" {
		t.Error("Expected to retrieve order")
	}

	o, err = oManager.orderStore.GetByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}

	o, err = oManager.orderStore.GetByExchangeAndID("Bitstamp", "")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

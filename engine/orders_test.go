package engine

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var oManager orderManager
var setupRan bool

func OrdersSetup(t *testing.T) {
	if !setupRan {
		SetupTest(t)
		if oManager.Started() {
			t.Fatal("Order manager already started")
		}
		err := oManager.Start()
		if !oManager.Started() {
			t.Fatal("Order manager not started")
		}
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
		Exchange: testExchange,
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
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
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
		Exchange: testExchange,
		ID:       "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := oManager.orderStore.GetByExchangeAndID(testExchange, "TestGetByExchangeAndID")
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

	o, err = oManager.orderStore.GetByExchangeAndID(testExchange, "")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestExistsWithLock(t *testing.T) {
	OrdersSetup(t)
	oManager.orderStore.exists(nil)
	oManager.orderStore.existsWithLock(nil)
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock",
	}
	err := oManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	b := oManager.orderStore.existsWithLock(o)
	if !b {
		t.Error("Expected true")
	}
	o2 := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock2",
	}
	go oManager.orderStore.existsWithLock(o)
	go oManager.orderStore.Add(o2)
	go oManager.orderStore.existsWithLock(o)
}

func TestCancelOrder(t *testing.T) {
	OrdersSetup(t)
	err := oManager.Cancel(nil)
	if err == nil {
		t.Error("Expected error due to nil cancel")
	}

	err = oManager.Cancel(&order.Cancel{})
	if err == nil {
		t.Error("Expected error due to nil cancel")
	}

	err = oManager.Cancel(&order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = oManager.Cancel(&order.Cancel{
		ID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = oManager.Cancel(&order.Cancel{
		ID:        "ID",
		Exchange:  testExchange,
		AssetType: asset.Binary,
	})
	if err == nil {
		t.Error("Expected error due to bad asset type")
	}

	o := &order.Detail{
		Exchange: fakePassExchange,
		ID:       "TestCancelOrder",
		Status:   order.New,
	}
	err = oManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	err = oManager.Cancel(&order.Cancel{
		ID:        "Unknown",
		Exchange:  fakePassExchange,
		AssetType: asset.Spot,
	})
	if err == nil {
		t.Error("Expected error due to no order found")
	}

	cancel := &order.Cancel{
		Exchange:  fakePassExchange,
		ID:        "TestCancelOrder",
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Date:      time.Now(),
		Pair:      currency.NewPairFromString("BTCUSD"),
	}
	err = oManager.Cancel(cancel)
	if err != nil {
		t.Error(err)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestCancelAllOrders(t *testing.T) {
	OrdersSetup(t)
	o := &order.Detail{
		Exchange: fakePassExchange,
		ID:       "TestCancelAllOrders",
		Status:   order.New,
	}
	err := oManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	oManager.CancelAllOrders([]string{"NotFound"})
	if o.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	oManager.CancelAllOrders([]string{fakePassExchange})
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}

	o.Status = order.New

	oManager.CancelAllOrders(nil)
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}
}

func TestSubmit(t *testing.T) {
	OrdersSetup(t)
	_, err := oManager.Submit(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	o := &order.Submit{
		Exchange: "",
		ID:       "FakePassingExchangeOrder",
		Status:   order.New,
		Type:     order.Market,
	}
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = fakePassExchange
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected error from validation")
	}

	oManager.cfg.EnforceLimitConfig = true
	oManager.cfg.AllowMarketOrders = false
	o.Pair = currency.NewPairFromString("BTCUSD")
	o.AssetType = asset.Spot
	o.Side = order.Buy
	o.Amount = 1
	o.Price = 1
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	oManager.cfg.AllowMarketOrders = true
	oManager.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	oManager.cfg.LimitAmount = 0
	oManager.cfg.AllowedExchanges = []string{"fake"}
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	oManager.cfg.AllowedExchanges = nil
	oManager.cfg.AllowedPairs = currency.Pairs{currency.NewPairFromString("BTCAUD")}
	_, err = oManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	oManager.cfg.AllowedPairs = nil
	_, err = oManager.Submit(o)
	if err != nil {
		t.Error(err)
	}

	o2, err := oManager.orderStore.GetByExchangeAndID(fakePassExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID == "" {
		t.Error("Failed to assign internal order id")
	}
}

func TestProcessOrders(t *testing.T) {
	OrdersSetup(t)
	oManager.processOrders()
}

func TestShutdown(t *testing.T) {
	OrdersSetup(t)
	oManager.cfg.CancelOrdersOnShutdown = true
	err := oManager.Stop()
	if err != nil {
		t.Error(err)
	}
	if oManager.started == 1 {
		t.Error("Has not stopped")
	}
}

package engine

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var ordersSetupRan bool

func OrdersSetup(t *testing.T) {
	SetupTest(t)
	if !ordersSetupRan {
		err := Bot.OrderManager.Start()
		if err != nil {
			t.Fatal(err)
		}
		if !Bot.OrderManager.Started() {
			t.Fatal("Order manager not started")
		}
		ordersSetupRan = true
	}
}

func TestOrdersGet(t *testing.T) {
	OrdersSetup(t)
	if Bot.OrderManager.orderStore.get() == nil {
		t.Error("orderStore not established")
	}
}

func TestOrdersAdd(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: "testTest",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error from non existent exchange")
	}

	err = Bot.OrderManager.orderStore.Add(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	err = Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByInternalOrderID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := Bot.OrderManager.orderStore.GetByInternalOrderID("internalTest")
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" {
		t.Error("Expected to retrieve order")
	}

	_, err = Bot.OrderManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := Bot.OrderManager.orderStore.GetByExchangeAndID(testExchange, "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.ID != "TestGetByExchangeAndID" {
		t.Error("Expected to retrieve order")
	}

	o, err = Bot.OrderManager.orderStore.GetByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}

	o, err = Bot.OrderManager.orderStore.GetByExchangeAndID(testExchange, "")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestExistsWithLock(t *testing.T) {
	OrdersSetup(t)
	Bot.OrderManager.orderStore.exists(nil)
	Bot.OrderManager.orderStore.existsWithLock(nil)
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock",
	}
	err := Bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	b := Bot.OrderManager.orderStore.existsWithLock(o)
	if !b {
		t.Error("Expected true")
	}
	o2 := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock2",
	}
	go Bot.OrderManager.orderStore.existsWithLock(o)
	go Bot.OrderManager.orderStore.Add(o2)
	go Bot.OrderManager.orderStore.existsWithLock(o)
}

func TestCancelOrder(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.Cancel(nil)
	if err == nil {
		t.Error("Expected error due to nil cancel")
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{})
	if err == nil {
		t.Error("Expected error due to nil cancel")
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{
		ID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{
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
	err = Bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{
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
	err = Bot.OrderManager.Cancel(cancel)
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
	err := Bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	Bot.OrderManager.CancelAllOrders([]string{"NotFound"})
	if o.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	Bot.OrderManager.CancelAllOrders([]string{fakePassExchange})
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}

	o.Status = order.New

	Bot.OrderManager.CancelAllOrders(nil)
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}
}

func TestSubmit(t *testing.T) {
	OrdersSetup(t)
	_, err := Bot.OrderManager.Submit(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	o := &order.Submit{
		Exchange: "",
		ID:       "FakePassingExchangeOrder",
		Status:   order.New,
		Type:     order.Market,
	}
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = fakePassExchange
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected error from validation")
	}

	Bot.OrderManager.cfg.EnforceLimitConfig = true
	Bot.OrderManager.cfg.AllowMarketOrders = false
	o.Pair = currency.NewPairFromString("BTCUSD")
	o.AssetType = asset.Spot
	o.Side = order.Buy
	o.Amount = 1
	o.Price = 1
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	Bot.OrderManager.cfg.AllowMarketOrders = true
	Bot.OrderManager.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	Bot.OrderManager.cfg.LimitAmount = 0
	Bot.OrderManager.cfg.AllowedExchanges = []string{"fake"}
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	Bot.OrderManager.cfg.AllowedExchanges = nil
	Bot.OrderManager.cfg.AllowedPairs = currency.Pairs{currency.NewPairFromString("BTCAUD")}
	_, err = Bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	Bot.OrderManager.cfg.AllowedPairs = nil
	_, err = Bot.OrderManager.Submit(o)
	if err != nil {
		t.Error(err)
	}

	o2, err := Bot.OrderManager.orderStore.GetByExchangeAndID(fakePassExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID == "" {
		t.Error("Failed to assign internal order id")
	}
}

func TestProcessOrders(t *testing.T) {
	OrdersSetup(t)
	Bot.OrderManager.processOrders()
}

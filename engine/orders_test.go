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
	SetupTestHelpers(t)
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
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestGetByExchange(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange",
		InternalOrderID: "internalTestGetByExchange",
	})
	if err != nil {
		t.Error(err)
	}

	err = Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange2",
		InternalOrderID: "internalTestGetByExchange2",
	})
	if err != nil {
		t.Error(err)
	}

	err = Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        fakePassExchange,
		ID:              "TestGetByExchange3",
		InternalOrderID: "internalTest3",
	})
	if err != nil {
		t.Error(err)
	}
	var o []*order.Detail
	o, err = Bot.OrderManager.orderStore.GetByExchange(testExchange)
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Error("Expected non nil response")
	}
	var o1Found, o2Found bool
	for i := range o {
		if o[i].ID == "TestGetByExchange" && o[i].Exchange == testExchange {
			o1Found = true
		}
		if o[i].ID == "TestGetByExchange2" && o[i].Exchange == testExchange {
			o2Found = true
		}
	}
	if !o1Found || !o2Found {
		t.Error("Expected orders 'TestGetByExchange' and 'TestGetByExchange2' to be returned")
	}

	_, err = Bot.OrderManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
	err = Bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: "thisWillFail",
	})
	if err == nil {
		t.Error("Expected exchange not found error")
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

	_, err = Bot.OrderManager.orderStore.GetByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrExchangeNotFound {
		t.Error(err)
	}

	_, err = Bot.OrderManager.orderStore.GetByExchangeAndID(testExchange, "")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestExists(t *testing.T) {
	OrdersSetup(t)
	if Bot.OrderManager.orderStore.exists(nil) {
		t.Error("Expected false")
	}
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExists",
	}
	err := Bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	b := Bot.OrderManager.orderStore.exists(o)
	if !b {
		t.Error("Expected true")
	}
}

func TestCancelOrder(t *testing.T) {
	OrdersSetup(t)
	err := Bot.OrderManager.Cancel(nil)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = Bot.OrderManager.Cancel(&order.Cancel{})
	if err == nil {
		t.Error("Expected error due to empty order")
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

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	cancel := &order.Cancel{
		Exchange:  fakePassExchange,
		ID:        "TestCancelOrder",
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Date:      time.Now(),
		Pair:      pair,
	}
	err = Bot.OrderManager.Cancel(cancel)
	if err != nil {
		t.Error(err)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestGetOrderInfo(t *testing.T) {
	OrdersSetup(t)
	_, err := Bot.OrderManager.GetOrderInfo("", "", currency.Pair{}, "")
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	var result order.Detail
	result, err = Bot.OrderManager.GetOrderInfo(fakePassExchange, "1234", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "fakeOrder" {
		t.Error("unexpected order returned")
	}

	result, err = Bot.OrderManager.GetOrderInfo(fakePassExchange, "1234", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "fakeOrder" {
		t.Error("unexpected order returned")
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
	if o.Status != order.New {
		t.Error("Order should not be cancelled")
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

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	Bot.OrderManager.cfg.EnforceLimitConfig = true
	Bot.OrderManager.cfg.AllowMarketOrders = false
	o.Pair = pair
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

	failPair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		t.Fatal(err)
	}

	Bot.OrderManager.cfg.AllowedExchanges = nil
	Bot.OrderManager.cfg.AllowedPairs = currency.Pairs{failPair}
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

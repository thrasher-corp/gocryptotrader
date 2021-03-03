package engine

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func OrdersSetup(t *testing.T) *Engine {
	bot := CreateTestBot(t)
	err := bot.OrderManager.Start(bot)
	if err != nil {
		t.Fatal(err)
	}
	bot.ServicesWG.Wait()
	if !bot.OrderManager.Started() {
		t.Fatal("Order manager not started")
	}
	return bot
}

func TestOrdersGet(t *testing.T) {
	bot := OrdersSetup(t)
	if bot.OrderManager.orderStore.get() == nil {
		t.Error("orderStore not established")
	}
}

func TestOrdersAdd(t *testing.T) {
	bot := OrdersSetup(t)
	err := bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: "testTest",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error from non existent exchange")
	}

	err = bot.OrderManager.orderStore.Add(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	err = bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	bot := OrdersSetup(t)
	err := bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByInternalOrderID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := bot.OrderManager.orderStore.GetByInternalOrderID("internalTest")
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" {
		t.Error("Expected to retrieve order")
	}

	_, err = bot.OrderManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestGetByExchange(t *testing.T) {
	bot := OrdersSetup(t)
	err := bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange",
		InternalOrderID: "internalTestGetByExchange",
	})
	if err != nil {
		t.Error(err)
	}

	err = bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange2",
		InternalOrderID: "internalTestGetByExchange2",
	})
	if err != nil {
		t.Error(err)
	}

	err = bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange:        fakePassExchange,
		ID:              "TestGetByExchange3",
		InternalOrderID: "internalTest3",
	})
	if err != nil {
		t.Error(err)
	}
	var o []*order.Detail
	o, err = bot.OrderManager.orderStore.GetByExchange(testExchange)
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

	_, err = bot.OrderManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
	err = bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: "thisWillFail",
	})
	if err == nil {
		t.Error("Expected exchange not found error")
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	bot := OrdersSetup(t)
	err := bot.OrderManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := bot.OrderManager.orderStore.GetByExchangeAndID(testExchange, "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.ID != "TestGetByExchangeAndID" {
		t.Error("Expected to retrieve order")
	}

	_, err = bot.OrderManager.orderStore.GetByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrExchangeNotFound {
		t.Error(err)
	}

	_, err = bot.OrderManager.orderStore.GetByExchangeAndID(testExchange, "")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestExists(t *testing.T) {
	bot := OrdersSetup(t)
	if bot.OrderManager.orderStore.exists(nil) {
		t.Error("Expected false")
	}
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExists",
	}
	err := bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	b := bot.OrderManager.orderStore.exists(o)
	if !b {
		t.Error("Expected true")
	}
}

func TestCancelOrder(t *testing.T) {
	bot := OrdersSetup(t)
	err := bot.OrderManager.Cancel(nil)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = bot.OrderManager.Cancel(&order.Cancel{})
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = bot.OrderManager.Cancel(&order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = bot.OrderManager.Cancel(&order.Cancel{
		ID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = bot.OrderManager.Cancel(&order.Cancel{
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
	err = bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	err = bot.OrderManager.Cancel(&order.Cancel{
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
	err = bot.OrderManager.Cancel(cancel)
	if err != nil {
		t.Error(err)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestGetOrderInfo(t *testing.T) {
	bot := OrdersSetup(t)
	_, err := bot.OrderManager.GetOrderInfo("", "", currency.Pair{}, "")
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	var result order.Detail
	result, err = bot.OrderManager.GetOrderInfo(fakePassExchange, "1234", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "fakeOrder" {
		t.Error("unexpected order returned")
	}

	result, err = bot.OrderManager.GetOrderInfo(fakePassExchange, "1234", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "fakeOrder" {
		t.Error("unexpected order returned")
	}
}

func TestCancelAllOrders(t *testing.T) {
	bot := OrdersSetup(t)
	o := &order.Detail{
		Exchange: fakePassExchange,
		ID:       "TestCancelAllOrders",
		Status:   order.New,
	}
	err := bot.OrderManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}

	bot.OrderManager.CancelAllOrders([]string{"NotFound"})
	if o.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	bot.OrderManager.CancelAllOrders([]string{fakePassExchange})
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}

	o.Status = order.New
	bot.OrderManager.CancelAllOrders(nil)
	if o.Status != order.New {
		t.Error("Order should not be cancelled")
	}
}

func TestSubmit(t *testing.T) {
	bot := OrdersSetup(t)
	_, err := bot.OrderManager.Submit(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	o := &order.Submit{
		Exchange: "",
		ID:       "FakePassingExchangeOrder",
		Status:   order.New,
		Type:     order.Market,
	}
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = fakePassExchange
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected error from validation")
	}

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	bot.OrderManager.cfg.EnforceLimitConfig = true
	bot.OrderManager.cfg.AllowMarketOrders = false
	o.Pair = pair
	o.AssetType = asset.Spot
	o.Side = order.Buy
	o.Amount = 1
	o.Price = 1
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	bot.OrderManager.cfg.AllowMarketOrders = true
	bot.OrderManager.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	bot.OrderManager.cfg.LimitAmount = 0
	bot.OrderManager.cfg.AllowedExchanges = []string{"fake"}
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	failPair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		t.Fatal(err)
	}

	bot.OrderManager.cfg.AllowedExchanges = nil
	bot.OrderManager.cfg.AllowedPairs = currency.Pairs{failPair}
	_, err = bot.OrderManager.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	bot.OrderManager.cfg.AllowedPairs = nil
	_, err = bot.OrderManager.Submit(o)
	if err != nil {
		t.Error(err)
	}

	o2, err := bot.OrderManager.orderStore.GetByExchangeAndID(fakePassExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID == "" {
		t.Error("Failed to assign internal order id")
	}
}

func TestProcessOrders(t *testing.T) {
	bot := OrdersSetup(t)
	bot.OrderManager.processOrders()
}

package engine

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// omfExchange aka ordermanager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type omfExchange struct {
	exchange.IBotExchange
}

// CancelOrder overrides testExchange's cancel order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) CancelOrder(o *order.Cancel) error {
	o.Status = order.Cancelled
	return nil
}

// GetOrderInfo overrides testExchange's get order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	if orderID == "" {
		return order.Detail{}, errors.New("")
	}

	return order.Detail{
		Exchange:  testExchange,
		ID:        orderID,
		Pair:      pair,
		AssetType: assetType,
	}, nil
}

func (f omfExchange) ModifyOrder(action *order.Modify) (order.Modify, error) {
	ans := *action
	ans.ID = "modified_order_id"
	return ans, nil
}

func TestSetupOrderManager(t *testing.T) {
	_, err := SetupOrderManager(nil, nil, nil, false)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = SetupOrderManager(SetupExchangeManager(), nil, nil, false)
	if !errors.Is(err, errNilCommunicationsManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilCommunicationsManager)
	}
	_, err = SetupOrderManager(SetupExchangeManager(), &CommunicationManager{}, nil, false)
	if !errors.Is(err, errNilWaitGroup) {
		t.Errorf("error '%v', expected '%v'", err, errNilWaitGroup)
	}
	var wg sync.WaitGroup
	_, err = SetupOrderManager(SetupExchangeManager(), &CommunicationManager{}, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestOrderManagerStart(t *testing.T) {
	var m *OrderManager
	err := m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
	var wg sync.WaitGroup
	m, err = SetupOrderManager(SetupExchangeManager(), &CommunicationManager{}, &wg, false)
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

func TestOrderManagerIsRunning(t *testing.T) {
	var m *OrderManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	var wg sync.WaitGroup
	m, err := SetupOrderManager(SetupExchangeManager(), &CommunicationManager{}, &wg, false)
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

func TestOrderManagerStop(t *testing.T) {
	var m *OrderManager
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	var wg sync.WaitGroup
	m, err = SetupOrderManager(SetupExchangeManager(), &CommunicationManager{}, &wg, false)
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

func OrdersSetup(t *testing.T) *OrderManager {
	var wg sync.WaitGroup
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()

	fakeExchange := omfExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	return m
}

func TestOrdersGet(t *testing.T) {
	m := OrdersSetup(t)
	if m.orderStore.get() == nil {
		t.Error("orderStore not established")
	}
}

func TestOrdersAdd(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = m.orderStore.add(&order.Detail{
		Exchange: "testTest",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error from non existent exchange")
	}

	err = m.orderStore.add(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	err = m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByInternalOrderID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := m.orderStore.getByInternalOrderID("internalTest")
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" {
		t.Error("Expected to retrieve order")
	}

	_, err = m.orderStore.getByInternalOrderID("NoOrder")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestGetByExchange(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange",
		InternalOrderID: "internalTestGetByExchange",
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange2",
		InternalOrderID: "internalTestGetByExchange2",
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchange3",
		InternalOrderID: "internalTest3",
	})
	if err != nil {
		t.Error(err)
	}
	var o []*order.Detail
	o, err = m.orderStore.getByExchange(testExchange)
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

	_, err = m.orderStore.getByInternalOrderID("NoOrder")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
	err = m.orderStore.add(&order.Detail{
		Exchange: "thisWillFail",
	})
	if err == nil {
		t.Error("Expected exchange not found error")
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := m.orderStore.getByExchangeAndID(testExchange, "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.ID != "TestGetByExchangeAndID" {
		t.Error("Expected to retrieve order")
	}

	_, err = m.orderStore.getByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrExchangeNotFound {
		t.Error(err)
	}

	_, err = m.orderStore.getByExchangeAndID(testExchange, "")
	if err != ErrOrderNotFound {
		t.Error(err)
	}
}

func TestExists(t *testing.T) {
	m := OrdersSetup(t)
	if m.orderStore.exists(nil) {
		t.Error("Expected false")
	}
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExists",
	}
	err := m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}
	b := m.orderStore.exists(o)
	if !b {
		t.Error("Expected true")
	}
}

func TestStore_modifyOrder(t *testing.T) {
	m := OrdersSetup(t)
	pair := currency.Pair{
		Base:  currency.NewCode("XXXXX"),
		Quote: currency.NewCode("YYYYY"),
	}
	err := m.orderStore.add(&order.Detail{
		Exchange:  testExchange,
		AssetType: asset.Spot,
		Pair:      pair,
		ID:        "fake_order_id",

		Price:  8,
		Amount: 128,
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.modifyExisting("fake_order_id", &order.Modify{
		Exchange: testExchange,

		ID:     "another_fake_order_id",
		Price:  16,
		Amount: 256,
	})
	if err != nil {
		t.Error(err)
	}

	_, err = m.orderStore.getByExchangeAndID(testExchange, "fake_order_id")
	if err == nil {
		// Expected error, such an order should not exist anymore in the store.
		t.Error("Expected error")
	}

	det, err := m.orderStore.getByExchangeAndID(testExchange, "another_fake_order_id")
	if det == nil || err != nil {
		t.Error("Failed to fetch order details")
	}
	if det.ID != "another_fake_order_id" || det.Price != 16 || det.Amount != 256 {
		t.Errorf(
			"have (%s,%f,%f), want (%s,%f,%f)",
			det.ID, det.Price, det.Amount,
			"another_fake_order_id", 16., 256.,
		)
	}
}

func TestCancelOrder(t *testing.T) {
	m := OrdersSetup(t)

	err := m.Cancel(nil)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(&order.Cancel{})
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(&order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = m.Cancel(&order.Cancel{
		ID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = m.Cancel(&order.Cancel{
		ID:        "ID",
		Exchange:  testExchange,
		AssetType: asset.Binary,
	})
	if err == nil {
		t.Error("Expected error due to bad asset type")
	}

	o := &order.Detail{
		Exchange: testExchange,
		ID:       "1337",
		Status:   order.New,
	}
	err = m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}

	err = m.Cancel(&order.Cancel{
		ID:        "Unknown",
		Exchange:  testExchange,
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
		Exchange:  testExchange,
		ID:        "1337",
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Date:      time.Now(),
		Pair:      pair,
	}
	err = m.Cancel(cancel)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestGetOrderInfo(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.GetOrderInfo("", "", currency.Pair{}, "")
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	var result order.Detail
	result, err = m.GetOrderInfo(testExchange, "1337", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "1337" {
		t.Error("unexpected order returned")
	}

	result, err = m.GetOrderInfo(testExchange, "1337", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "1337" {
		t.Error("unexpected order returned")
	}
}

func TestCancelAllOrders(t *testing.T) {
	m := OrdersSetup(t)
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestCancelAllOrders",
		Status:   order.New,
	}
	err := m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}
	exch := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	m.CancelAllOrders([]exchange.IBotExchange{})
	if o.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	m.CancelAllOrders([]exchange.IBotExchange{exch})
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}

	o.Status = order.New
	m.CancelAllOrders(nil)
	if o.Status != order.New {
		t.Error("Order should not be cancelled")
	}
}

func TestSubmit(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.Submit(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	o := &order.Submit{
		Exchange: "",
		ID:       "FakePassingExchangeOrder",
		Status:   order.New,
		Type:     order.Market,
	}
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = testExchange
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected error from validation")
	}

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	m.cfg.EnforceLimitConfig = true
	m.cfg.AllowMarketOrders = false
	o.Pair = pair
	o.AssetType = asset.Spot
	o.Side = order.Buy
	o.Amount = 1
	o.Price = 1
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	m.cfg.AllowMarketOrders = true
	m.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	m.cfg.LimitAmount = 0
	m.cfg.AllowedExchanges = []string{"fake"}
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	failPair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		t.Fatal(err)
	}

	m.cfg.AllowedExchanges = nil
	m.cfg.AllowedPairs = currency.Pairs{failPair}
	_, err = m.Submit(o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	m.cfg.AllowedPairs = nil
	_, err = m.Submit(o)
	if !errors.Is(err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet) {
		t.Errorf("error '%v', expected '%v'", err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		ID:       "FakePassingExchangeOrder",
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	o2, err := m.orderStore.getByExchangeAndID(testExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID == "" {
		t.Error("Failed to assign internal order id")
	}
}

func TestOrderManager_Modify(t *testing.T) {
	pair := currency.Pair{
		Base:  currency.NewCode("XXXXX"),
		Quote: currency.NewCode("YYYYY"),
	}
	f := func(mod order.Modify, expectError bool, price, amount float64) {
		t.Helper()

		m := OrdersSetup(t)
		err := m.orderStore.add(&order.Detail{
			Exchange:  testExchange,
			AssetType: asset.Spot,
			Pair:      pair,
			ID:        "fake_order_id",
			//
			Price:  8,
			Amount: 128,
		})
		if err != nil {
			t.Error(err)
		}

		resp, err := m.Modify(&mod)
		if expectError {
			if err == nil {
				t.Fatal("Expected error")
			}
			return
		} else if err != nil {
			t.Fatal(err)
		}

		if resp.OrderID != "modified_order_id" {
			t.Errorf("have \"%s\", want \"modified_order_id\"", resp.OrderID)
		}

		det, err := m.orderStore.getByExchangeAndID(testExchange, resp.OrderID)
		if err != nil {
			t.Fatal(err)
		}
		if det.ID != resp.OrderID || det.Price != price || det.Amount != amount {
			t.Errorf(
				"have (%s,%f,%f), want (%s,%f,%f)",
				det.ID, det.Price, det.Amount,
				resp.OrderID, price, amount,
			)
		}
	}

	model := order.Modify{
		// These fields identify the order.
		Exchange:  testExchange,
		AssetType: asset.Spot,
		Pair:      pair,
		ID:        "fake_order_id",
		// These fields modify the order.
		Price:  0,
		Amount: 0,
	}

	// [1] Test if nonexistent order returns an error.
	one := model
	one.ID = "nonexistent_order_id"
	f(one, true, 0, 0)

	// [2] Test if price of 0 is ignored.
	two := model
	two.Price = 0
	two.Amount = 256
	f(two, false, 8, 256)

	// [3] Test if amount of 0 is ignored.
	three := model
	three.Price = 16
	three.Amount = 0
	f(three, false, 16, 128)

	// [4] Test if both fields provided set both.
	four := model
	four.Price = 16
	four.Amount = 256
	f(four, false, 16, 256)

	// [5] Test if both fields missing modifies anything but the ID.
	five := model
	five.Price = 0
	five.Amount = 0
	f(five, false, 8, 128)
}

func TestProcessOrders(t *testing.T) {
	m := OrdersSetup(t)
	m.processOrders()
}

func TestGetOrdersFiltered(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.GetOrdersFiltered(nil)
	if err == nil {
		t.Error("Expected error from nil filter")
	}
	orders := []order.Detail{
		{
			Exchange: testExchange,
			ID:       "Test1",
		},
		{
			Exchange: testExchange,
			ID:       "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res, err := m.GetOrdersFiltered(&order.Filter{ID: "Test2"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
}

func Test_getFilteredOrders(t *testing.T) {
	m := OrdersSetup(t)

	_, err := m.orderStore.getFilteredOrders(nil)
	if err == nil {
		t.Error("Error expected when Filter is nil")
	}

	orders := []order.Detail{
		{
			Exchange: testExchange,
			ID:       "Test1",
		},
		{
			Exchange: testExchange,
			ID:       "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res, err := m.orderStore.getFilteredOrders(&order.Filter{ID: "Test1"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
}

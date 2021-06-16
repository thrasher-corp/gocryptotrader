package engine

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var ftxTestExchange = "ftx"

// omfExchange aka ordermanager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type omfExchange struct {
	exchange.IBotExchange
}

// CancelOrder overrides ftxTestExchange's cancel order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) CancelOrder(o *order.Cancel) error {
	o.Status = order.Cancelled
	return nil
}

// GetOrderInfo overrides ftxTestExchange's get order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	if orderID == "" {
		return order.Detail{}, errors.New("")
	}

	return order.Detail{
		Exchange:  ftxTestExchange,
		ID:        orderID,
		Pair:      pair,
		AssetType: assetType,
	}, nil
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

var orderManager *OrderManager
var m sync.Mutex

func OrdersSetup(t *testing.T) *OrderManager {
	m.Lock()
	defer m.Unlock()
	if orderManager == nil {
		var wg sync.WaitGroup
		em := SetupExchangeManager()
		exch, err := em.NewExchangeByName("ftx")
		if err != nil {
			t.Error(err)
		}
		exch.SetDefaults()
		conf, err := exch.GetDefaultConfig()
		if err != nil {
			t.Error(err)
		}

		err = exch.Setup(conf)
		if err != nil {
			t.Error(err)
		}

		b := exch.GetBase()
		err = b.Holdings.LoadHoldings(string(account.Main),
			true,
			asset.Spot,
			account.HoldingsSnapshot{
				currency.BTC: account.Balance{Total: 10},
			})
		if err != nil {
			t.Error(err)
		}

		fakeExchange := omfExchange{
			IBotExchange: exch,
		}
		em.Add(&fakeExchange)
		orderManager, err = SetupOrderManager(em, &CommunicationManager{}, &wg, false)
		if !errors.Is(err, nil) {
			t.Errorf("error '%v', expected '%v'", err, nil)
		}
		err = orderManager.Start()
		if !errors.Is(err, nil) {
			t.Errorf("error '%v', expected '%v'", err, nil)
		}
	}
	return orderManager
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
		Exchange: ftxTestExchange,
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
		Exchange: ftxTestExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange:        ftxTestExchange,
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
		Exchange:        ftxTestExchange,
		ID:              "TestGetByExchange",
		InternalOrderID: "internalTestGetByExchange",
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange:        ftxTestExchange,
		ID:              "TestGetByExchange2",
		InternalOrderID: "internalTestGetByExchange2",
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange:        ftxTestExchange,
		ID:              "TestGetByExchange3",
		InternalOrderID: "internalTest3",
	})
	if err != nil {
		t.Error(err)
	}
	var o []*order.Detail
	o, err = m.orderStore.getByExchange(ftxTestExchange)
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Error("Expected non nil response")
	}
	var o1Found, o2Found bool
	for i := range o {
		if o[i].ID == "TestGetByExchange" && o[i].Exchange == ftxTestExchange {
			o1Found = true
		}
		if o[i].ID == "TestGetByExchange2" && o[i].Exchange == ftxTestExchange {
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
		Exchange: ftxTestExchange,
		ID:       "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := m.orderStore.getByExchangeAndID(ftxTestExchange, "TestGetByExchangeAndID")
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

	_, err = m.orderStore.getByExchangeAndID(ftxTestExchange, "")
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
		Exchange: ftxTestExchange,
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
		Exchange: ftxTestExchange,
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
		Exchange:  ftxTestExchange,
		AssetType: asset.Binary,
	})
	if err == nil {
		t.Error("Expected error due to bad asset type")
	}

	o := &order.Detail{
		Exchange: ftxTestExchange,
		ID:       "1337",
		Status:   order.New,
	}
	err = m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}

	err = m.Cancel(&order.Cancel{
		ID:        "Unknown",
		Exchange:  ftxTestExchange,
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
		Exchange:  ftxTestExchange,
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
	result, err = m.GetOrderInfo(ftxTestExchange, "1337", currency.Pair{}, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "1337" {
		t.Error("unexpected order returned")
	}

	result, err = m.GetOrderInfo(ftxTestExchange, "1337", currency.Pair{}, "")
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
		Exchange: ftxTestExchange,
		ID:       "TestCancelAllOrders",
		Status:   order.New,
	}
	err := m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}
	exch := m.orderStore.exchangeManager.GetExchangeByName(ftxTestExchange)
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

	o.Exchange = ftxTestExchange
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
	if !errors.Is(err, account.ErrAccountNameUnset) {
		t.Errorf("error '%v', expected '%v'", err, account.ErrAccountNameUnset)
	}

	o.Account = string(account.Main)
	_, err = m.Submit(o)
	if !errors.Is(err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet) {
		t.Errorf("error '%v', expected '%v'", err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}

	err = m.orderStore.add(&order.Detail{
		Exchange: ftxTestExchange,
		ID:       "FakePassingExchangeOrder",
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	o2, err := m.orderStore.getByExchangeAndID(ftxTestExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID == "" {
		t.Error("Failed to assign internal order id")
	}
}

func TestProcessOrders(t *testing.T) {
	m := OrdersSetup(t)
	m.processOrders()
}

package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// omfExchange aka ordermanager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type omfExchange struct {
	exchange.IBotExchange
}

// CancelOrder overrides testExchange's cancel order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	o.Status = order.Cancelled
	return nil
}

// GetOrderInfo overrides testExchange's get order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	switch orderID {
	case "":
		return order.Detail{}, errors.New("")
	case "Order1-unknown-to-active":
		return order.Detail{
			Exchange:    testExchange,
			Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Buy,
			Status:      order.Active,
			LastUpdated: time.Now().Add(-time.Hour),
			ID:          "Order1-unknown-to-active",
		}, nil
	case "Order2-active-to-inactive":
		return order.Detail{
			Exchange:    testExchange,
			Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Sell,
			Status:      order.Cancelled,
			LastUpdated: time.Now().Add(-time.Hour),
			ID:          "Order2-active-to-inactive",
		}, nil
	}

	return order.Detail{
		Exchange:  testExchange,
		ID:        orderID,
		Pair:      pair,
		AssetType: assetType,
		Status:    order.Cancelled,
	}, nil
}

// GetActiveOrders overrides the function used by processOrders to return 1 active order
func (f omfExchange) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	return []order.Detail{{
		Exchange:    testExchange,
		Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
		AssetType:   asset.Spot,
		Amount:      2.0,
		Side:        order.Sell,
		Status:      order.Active,
		LastUpdated: time.Now().Add(-time.Hour),
		ID:          "Order3-unknown-to-active",
	}}, nil
}

func (f omfExchange) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
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
	t.Helper()
	var wg sync.WaitGroup
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()

	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}

	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}

	fakeExchange := omfExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.started = 1
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
		t.Fatal(err)
	}
	if o == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
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
	if err := m.orderStore.add(o); err != nil {
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
		t.Fatal("Expected error")
	}

	det, err := m.orderStore.getByExchangeAndID(testExchange, "another_fake_order_id")
	if det == nil || err != nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("Failed to fetch order details")
	}
	if det.ID != "another_fake_order_id" || det.Price != 16 || det.Amount != 256 { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Errorf(
			"have (%s,%f,%f), want (%s,%f,%f)",
			det.ID, det.Price, det.Amount,
			"another_fake_order_id", 16., 256.,
		)
	}
}

func TestCancelOrder(t *testing.T) {
	m := OrdersSetup(t)

	err := m.Cancel(context.Background(), nil)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(context.Background(), &order.Cancel{})
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(context.Background(), &order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = m.Cancel(context.Background(), &order.Cancel{
		ID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = m.Cancel(context.Background(),
		&order.Cancel{
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

	err = m.Cancel(context.Background(),
		&order.Cancel{
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
	err = m.Cancel(context.Background(), cancel)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestGetOrderInfo(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.GetOrderInfo(context.Background(), "", "", currency.EMPTYPAIR, "")
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	var result order.Detail
	result, err = m.GetOrderInfo(context.Background(),
		testExchange, "1337", currency.EMPTYPAIR, "")
	if err != nil {
		t.Error(err)
	}
	if result.ID != "1337" {
		t.Error("unexpected order returned")
	}

	result, err = m.GetOrderInfo(context.Background(),
		testExchange, "1337", currency.EMPTYPAIR, "")
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
	if err := m.orderStore.add(o); err != nil {
		t.Error(err)
	}

	m.CancelAllOrders(context.Background(), []exchange.IBotExchange{})
	if o.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	exch, err := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	m.CancelAllOrders(context.Background(), []exchange.IBotExchange{exch})
	if o.Status != order.Cancelled {
		t.Error("Order should be cancelled")
	}

	o.Status = order.New
	m.CancelAllOrders(context.Background(), nil)
	if o.Status != order.New {
		t.Error("Order should not be cancelled")
	}
}

func TestSubmit(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.Submit(context.Background(), nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	o := &order.Submit{
		Exchange: "",
		ID:       "FakePassingExchangeOrder",
		Status:   order.New,
		Type:     order.Market,
	}
	_, err = m.Submit(context.Background(), o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = testExchange
	_, err = m.Submit(context.Background(), o)
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
	_, err = m.Submit(context.Background(), o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	m.cfg.AllowMarketOrders = true
	m.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = m.Submit(context.Background(), o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	m.cfg.LimitAmount = 0
	m.cfg.AllowedExchanges = []string{"fake"}
	_, err = m.Submit(context.Background(), o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	failPair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		t.Fatal(err)
	}

	m.cfg.AllowedExchanges = nil
	m.cfg.AllowedPairs = currency.Pairs{failPair}
	_, err = m.Submit(context.Background(), o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	m.cfg.AllowedPairs = nil
	_, err = m.Submit(context.Background(), o)
	if !errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrAuthenticationSupportNotEnabled)
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
			Price:     8,
			Amount:    128,
		})
		if err != nil {
			t.Error(err)
		}

		resp, err := m.Modify(context.Background(), &mod)
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

	// [4] Test if both fields work together.
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
	m.started = 1
	pairs := currency.Pairs{
		currency.Pair{Base: currency.BTC, Quote: currency.USD},
	}
	// Ensure processOrders() can run the REST calls to GetActiveOrders
	// and to GetOrders
	exch.GetBase().API = exchange.API{
		AuthenticatedSupport:          true,
		AuthenticatedWebsocketSupport: false,
	}
	exch.GetBase().Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
			RESTCapabilities: protocol.Features{
				GetOrder: true,
			},
		},
	}
	exch.GetBase().CurrencyPairs = currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
				Enabled:      pairs,
				Available:    pairs,
			},
		},
	}
	exch.GetBase().Config = &config.Exchange{
		CurrencyPairs: &currency.PairsManager{
			UseGlobalFormat: true,
			RequestFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
			ConfigFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					AssetEnabled: convert.BoolPtr(true),
					Enabled:      pairs,
					Available:    pairs,
				},
			},
		},
	}

	orders := []order.Detail{
		{
			Exchange:    testExchange,
			Pair:        pairs[0],
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Buy,
			Status:      order.UnknownStatus,
			LastUpdated: time.Now().Add(-time.Hour),
			ID:          "Order1-unknown-to-active",
		},
		{
			Exchange:    testExchange,
			Pair:        pairs[0],
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Sell,
			Status:      order.Active,
			LastUpdated: time.Now().Add(-time.Hour),
			ID:          "Order2-active-to-inactive",
		},
		{
			Exchange:    testExchange,
			Pair:        pairs[0],
			AssetType:   asset.Spot,
			Amount:      2.0,
			Side:        order.Sell,
			Status:      order.UnknownStatus,
			LastUpdated: time.Now().Add(-time.Hour),
			ID:          "Order3-unknown-to-active",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}

	m.processOrders()

	// Order1 is not returned by exch.GetActiveOrders()
	// It will be fetched by exch.GetOrderInfo(), which will say it is active
	res, err := m.GetOrdersFiltered(&order.Filter{ID: "Order1-unknown-to-active"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 3 result, got: %d", len(res))
	}
	if res[0].Status != order.Active {
		t.Errorf("Order 1 should be active, but status is %s", string(res[0].Status))
	}

	// Order2 is not returned by exch.GetActiveOrders()
	// It will be fetched by exch.GetOrderInfo(), which will say it is cancelled
	res, err = m.GetOrdersFiltered(&order.Filter{ID: "Order2-active-to-inactive"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].Status != order.Cancelled {
		t.Errorf("Order 2 should be cancelled, but status is %s", string(res[0].Status))
	}

	// Order3 is returned by exch.GetActiveOrders(), which will say it is active
	res, err = m.GetOrdersFiltered(&order.Filter{ID: "Order3-unknown-to-active"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].Status != order.Active {
		t.Errorf("Order 3 should be active, but status is %s", string(res[0].Status))
	}
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

func TestGetOrdersActive(t *testing.T) {
	m := OrdersSetup(t)
	var err error
	orders := []order.Detail{
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Buy,
			Status:   order.Cancelled,
			ID:       "Test1",
		},
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Sell,
			Status:   order.Active,
			ID:       "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res, err := m.GetOrdersActive(nil)
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("TestGetOrdersActive - Expected 1 result, got: %d", len(res))
	}
	res, err = m.GetOrdersActive(&order.Filter{Side: order.Sell})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("TestGetOrdersActive - Expected 1 result, got: %d", len(res))
	}
	res, err = m.GetOrdersActive(&order.Filter{Side: order.Buy})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 0 {
		t.Errorf("TestGetOrdersActive - Expected 0 results, got: %d", len(res))
	}
}

func Test_processMatchingOrders(t *testing.T) {
	m := OrdersSetup(t)
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	orders := []order.Detail{
		{
			Exchange:    testExchange,
			ID:          "Test1",
			LastUpdated: time.Now(),
		},
		{
			Exchange:    testExchange,
			ID:          "Test2",
			LastUpdated: time.Now(),
		},
		{
			Exchange:    testExchange,
			ID:          "Test3",
			LastUpdated: time.Now().Add(-time.Hour),
		},
		{
			Exchange:    testExchange,
			ID:          "Test4",
			LastUpdated: time.Now().Add(-time.Hour),
		},
	}
	requiresProcessing := make(map[string]bool, len(orders))
	for i := range orders {
		orders[i].GenerateInternalOrderID()
		if i%2 == 0 {
			requiresProcessing[orders[i].InternalOrderID] = false
		} else {
			requiresProcessing[orders[i].InternalOrderID] = true
		}
	}
	var wg sync.WaitGroup
	wg.Add(1)
	m.processMatchingOrders(exch, orders, requiresProcessing, &wg)
	wg.Wait()
	res, err := m.GetOrdersFiltered(&order.Filter{Exchange: testExchange})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].ID != "Test4" {
		t.Error("Order Test4 should have been fetched and updated")
	}
}

func TestFetchAndUpdateExchangeOrder(t *testing.T) {
	m := OrdersSetup(t)
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	err = m.FetchAndUpdateExchangeOrder(exch, nil, asset.Spot)
	if err == nil {
		t.Error("Error expected when order is nil")
	}
	o := &order.Detail{
		Exchange: testExchange,
		Amount:   1.0,
		Side:     order.Sell,
		Status:   order.Active,
		ID:       "Test",
	}
	err = m.FetchAndUpdateExchangeOrder(exch, o, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if o.Status != order.Active {
		t.Error("Order should be active")
	}
	res, err := m.GetOrdersFiltered(&order.Filter{Exchange: testExchange})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}

	o.Status = order.PartiallyCancelled
	err = m.FetchAndUpdateExchangeOrder(exch, o, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	res, err = m.GetOrdersFiltered(&order.Filter{Exchange: testExchange})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
}

func Test_getActiveOrders(t *testing.T) {
	m := OrdersSetup(t)
	var err error
	orders := []order.Detail{
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Buy,
			Status:   order.Cancelled,
			ID:       "Test1",
		},
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Sell,
			Status:   order.Active,
			ID:       "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res := m.orderStore.getActiveOrders(nil)
	if len(res) != 1 {
		t.Errorf("Test_getActiveOrders - Expected 1 result, got: %d", len(res))
	}
	res = m.orderStore.getActiveOrders(&order.Filter{Side: order.Sell})
	if len(res) != 1 {
		t.Errorf("Test_getActiveOrders - Expected 1 result, got: %d", len(res))
	}
	res = m.orderStore.getActiveOrders(&order.Filter{Side: order.Buy})
	if len(res) != 0 {
		t.Errorf("Test_getActiveOrders - Expected 0 results, got: %d", len(res))
	}
}

func TestGetFuturesPositionsForExchange(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := o.GetFuturesPositionsForExchange("test", asset.Spot, cp)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	o.started = 1
	_, err = o.GetFuturesPositionsForExchange("test", asset.Spot, cp)
	if !errors.Is(err, errFuturesTrackerNotSetup) {
		t.Errorf("received '%v', expected '%v'", err, errFuturesTrackerNotSetup)
	}
	o.orderStore.futuresPositionController = order.SetupPositionController()
	_, err = o.GetFuturesPositionsForExchange("test", asset.Spot, cp)
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}

	_, err = o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	if !errors.Is(err, order.ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrPositionsNotLoadedForExchange)
	}

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		ID:        "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	resp, err := o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	if len(resp) != 1 {
		t.Error("expected 1 position")
	}

	o = nil
	_, err = o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestClearFuturesPositionsForExchange(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	err := o.ClearFuturesTracking("test", asset.Spot, cp)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	o.started = 1
	err = o.ClearFuturesTracking("test", asset.Spot, cp)
	if !errors.Is(err, errFuturesTrackerNotSetup) {
		t.Errorf("received '%v', expected '%v'", err, errFuturesTrackerNotSetup)
	}
	o.orderStore.futuresPositionController = order.SetupPositionController()
	err = o.ClearFuturesTracking("test", asset.Spot, cp)
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}

	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	if !errors.Is(err, order.ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrPositionsNotLoadedForExchange)
	}

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		ID:        "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	resp, err := o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	if len(resp) != 0 {
		t.Errorf("expected no position, received '%v'", len(resp))
	}

	o = nil
	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	o.started = 1
	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	if !errors.Is(err, errFuturesTrackerNotSetup) {
		t.Errorf("received '%v', expected '%v'", err, errFuturesTrackerNotSetup)
	}
	o.orderStore.futuresPositionController = order.SetupPositionController()
	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}

	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Futures, cp, 1, time.Now())
	if !errors.Is(err, order.ErrPositionsNotLoadedForExchange) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrPositionsNotLoadedForExchange)
	}

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		ID:        "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	unrealised, err := o.UpdateOpenPositionUnrealisedPNL("test", asset.Futures, cp, 2, time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	if !unrealised.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v', expected '%v'", unrealised, 1)
	}

	o = nil
	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestSubmitFakeOrder(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	resp := order.SubmitResponse{}
	_, err := o.SubmitFakeOrder(nil, resp, false)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	o.started = 1
	_, err = o.SubmitFakeOrder(nil, resp, false)
	if !errors.Is(err, errNilOrder) {
		t.Errorf("received '%v', expected '%v'", err, errNilOrder)
	}
	ord := &order.Submit{}
	_, err = o.SubmitFakeOrder(ord, resp, false)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNameIsEmpty)
	}
	ord.Exchange = testExchange
	ord.AssetType = asset.Spot
	ord.Pair = currency.NewPair(currency.BTC, currency.DOGE)
	ord.Side = order.Buy
	ord.Type = order.Market
	ord.Amount = 1337
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	o.orderStore.exchangeManager = em

	_, err = o.SubmitFakeOrder(ord, resp, false)
	if !errors.Is(err, errUnableToPlaceOrder) {
		t.Errorf("received '%v', expected '%v'", err, errUnableToPlaceOrder)
	}

	resp.IsOrderPlaced = true
	resp.FullyMatched = true
	o.orderStore.commsManager = &CommunicationManager{}
	o.orderStore.Orders = make(map[string][]*order.Detail)
	_, err = o.SubmitFakeOrder(ord, resp, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetOrdersSnapshot(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	o.GetOrdersSnapshot(order.AnyStatus)
	o.started = 1
	o.orderStore.Orders = make(map[string][]*order.Detail)
	o.orderStore.Orders[testExchange] = []*order.Detail{
		{
			Status: order.Open,
		},
	}
	snap := o.GetOrdersSnapshot(order.Open)
	if len(snap) != 1 {
		t.Error("expected 1")
	}
	snap = o.GetOrdersSnapshot(order.Closed)
	if len(snap) != 0 {
		t.Error("expected 0")
	}
}

func TestUpdateExisting(t *testing.T) {
	t.Parallel()
	s := &store{}
	s.Orders = make(map[string][]*order.Detail)
	err := s.updateExisting(nil)
	if !errors.Is(err, errNilOrder) {
		t.Errorf("received '%v', expected '%v'", err, errNilOrder)
	}
	od := &order.Detail{Exchange: testExchange}
	err = s.updateExisting(od)
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNotFound)
	}
	s.Orders[strings.ToLower(testExchange)] = nil
	err = s.updateExisting(od)
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrOrderNotFound)
	}
	od.AssetType = asset.Futures
	od.ID = "123"
	od.Pair = currency.NewPair(currency.BTC, currency.USDT)
	od.Side = order.Buy
	od.Type = order.Market
	od.Date = time.Now()
	od.Amount = 1337
	s.Orders[strings.ToLower(testExchange)] = []*order.Detail{
		od,
	}
	s.futuresPositionController = order.SetupPositionController()
	err = s.updateExisting(od)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	pos, err := s.futuresPositionController.GetPositionsForExchange(testExchange, asset.Futures, od.Pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	if len(pos) != 1 {
		t.Error("expected 1")
	}
}

func TestOrderManagerExists(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	if o.Exists(nil) {
		t.Error("expected false")
	}
	o.started = 1
	if o.Exists(nil) {
		t.Error("expected false")
	}

	o = nil
	if o.Exists(nil) {
		t.Error("expected false")
	}
}

func TestOrderManagerAdd(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	err := o.Add(nil)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	o.started = 1
	err = o.Add(nil)
	if !errors.Is(err, errNilOrder) {
		t.Errorf("received '%v', expected '%v'", err, errNilOrder)
	}

	o = nil
	err = o.Add(nil)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// omfExchange aka order manager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type omfExchange struct {
	exchange.IBotExchange
}

var btcusdPair = currency.NewBTCUSD()

// CancelOrder overrides testExchange's cancel order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) CancelOrder(_ context.Context, _ *order.Cancel) error {
	return nil
}

func (f omfExchange) GetCachedTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return &ticker.Price{
		Last:                  1337,
		High:                  1337,
		Low:                   1337,
		Bid:                   1337,
		Ask:                   1337,
		Volume:                1337,
		QuoteVolume:           1337,
		PriceATH:              1337,
		Open:                  1337,
		Close:                 1337,
		Pair:                  p,
		ExchangeName:          f.GetName(),
		AssetType:             a,
		LastUpdated:           time.Now(),
		FlashReturnRate:       1337,
		BidPeriod:             1337,
		BidSize:               1337,
		AskPeriod:             1337,
		AskSize:               1337,
		FlashReturnRateAmount: 1337,
	}, nil
}

// GetOrderInfo overrides testExchange's get order function
// to do the bare minimum required with no API calls or credentials required
func (f omfExchange) GetOrderInfo(_ context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	switch orderID {
	case "":
		return nil, errors.New("")
	case "Order1-unknown-to-active":
		return &order.Detail{
			Exchange:    testExchange,
			Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Buy,
			Status:      order.Active,
			LastUpdated: time.Now().Add(-time.Hour),
			OrderID:     "Order1-unknown-to-active",
		}, nil
	case "Order2-active-to-inactive":
		return &order.Detail{
			Exchange:    testExchange,
			Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Sell,
			Status:      order.Cancelled,
			LastUpdated: time.Now().Add(-time.Hour),
			OrderID:     "Order2-active-to-inactive",
		}, nil
	}

	return &order.Detail{
		Exchange:  testExchange,
		OrderID:   orderID,
		Pair:      pair,
		AssetType: assetType,
		Status:    order.Cancelled,
	}, nil
}

// GetActiveOrders overrides the function used by processOrders to return 1 active order
func (f omfExchange) GetActiveOrders(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return []order.Detail{{
		Exchange:    testExchange,
		Pair:        currency.Pair{Base: currency.BTC, Quote: currency.USD},
		AssetType:   asset.Spot,
		Amount:      2.0,
		Side:        order.Sell,
		Status:      order.Active,
		LastUpdated: time.Now().Add(-time.Hour),
		OrderID:     "Order3-unknown-to-active",
	}}, nil
}

func (f omfExchange) ModifyOrder(_ context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	modResp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	modResp.OrderID = "modified_order_id"
	return modResp, nil
}

func (f omfExchange) GetFuturesPositions(_ context.Context, req *futures.PositionsRequest) ([]futures.PositionDetails, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionDetails, len(req.Pairs))
	tt := time.Now()
	for i := range req.Pairs {
		resp[i] = futures.PositionDetails{
			Exchange: f.GetName(),
			Asset:    req.Asset,
			Pair:     req.Pairs[i],
			Orders: []order.Detail{
				{
					Exchange:        f.GetName(),
					Price:           1337,
					Amount:          1337,
					InternalOrderID: id,
					OrderID:         "1337",
					ClientOrderID:   "1337",
					Type:            order.Market,
					Side:            order.Short,
					Status:          order.Open,
					AssetType:       req.Asset,
					Date:            tt,
					CloseTime:       tt,
					LastUpdated:     tt,
					Pair:            req.Pairs[i],
				},
			},
		}
	}
	return resp, nil
}

func TestSetupOrderManager(t *testing.T) {
	_, err := SetupOrderManager(nil, nil, nil, nil)
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = SetupOrderManager(NewExchangeManager(), nil, nil, nil)
	assert.ErrorIs(t, err, errNilCommunicationsManager)

	_, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, nil, &config.OrderManager{})
	assert.ErrorIs(t, err, errNilWaitGroup)

	var wg sync.WaitGroup
	_, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{})
	require.NoError(t, err)

	_, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{ActivelyTrackFuturesPositions: true})
	require.ErrorIs(t, err, errInvalidFuturesTrackingSeekDuration)

	_, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{ActivelyTrackFuturesPositions: true, FuturesTrackingSeekDuration: time.Hour})
	require.NoError(t, err)
}

func TestOrderManagerStart(t *testing.T) {
	var m *OrderManager
	err := m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	var wg sync.WaitGroup
	m, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)
}

func TestOrderManagerIsRunning(t *testing.T) {
	var m *OrderManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	var wg sync.WaitGroup
	m, err := SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	if m.IsRunning() {
		t.Error("expected false")
	}

	err = m.Start()
	assert.NoError(t, err)

	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestOrderManagerStop(t *testing.T) {
	var m *OrderManager
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	var wg sync.WaitGroup
	m, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func OrdersSetup(t *testing.T) *OrderManager {
	t.Helper()
	var wg sync.WaitGroup
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := exchange.GetDefaultConfig(t.Context(), exch)
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
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

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
		OrderID:  "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = m.orderStore.add(&order.Detail{
		Exchange: "testTest",
		OrderID:  "TestOrdersAdd",
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
		OrderID:  "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	m := OrdersSetup(t)
	err := m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		OrderID:  "TestGetByExchangeAndID",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := m.orderStore.getByExchangeAndID(testExchange, "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.OrderID != "TestGetByExchangeAndID" {
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
		OrderID:  "TestExists",
	}
	if err := m.orderStore.add(o); err != nil {
		t.Error(err)
	}
	if b := m.orderStore.exists(o); !b {
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
		OrderID:   "fake_order_id",

		Price:  8,
		Amount: 128,
	})
	if err != nil {
		t.Error(err)
	}

	err = m.orderStore.modifyExisting("fake_order_id", &order.ModifyResponse{
		Exchange: testExchange,
		OrderID:  "another_fake_order_id",
		Price:    16,
		Amount:   256,
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
	if det.OrderID != "another_fake_order_id" || det.Price != 16 || det.Amount != 256 { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Errorf(
			"have (%s,%f,%f), want (%s,%f,%f)",
			det.OrderID, det.Price, det.Amount,
			"another_fake_order_id", 16., 256.,
		)
	}
}

func TestCancelOrder(t *testing.T) {
	m := OrdersSetup(t)

	err := m.Cancel(t.Context(), nil)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(t.Context(), &order.Cancel{})
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	err = m.Cancel(t.Context(), &order.Cancel{
		Exchange: testExchange,
	})
	if err == nil {
		t.Error("Expected error due to no order ID")
	}

	err = m.Cancel(t.Context(), &order.Cancel{
		OrderID: "ID",
	})
	if err == nil {
		t.Error("Expected error due to no Exchange")
	}

	err = m.Cancel(t.Context(), &order.Cancel{
		OrderID:   "ID",
		Exchange:  testExchange,
		AssetType: asset.Binary,
	})
	if err == nil {
		t.Error("Expected error due to bad asset type")
	}

	o := &order.Detail{
		Exchange: testExchange,
		OrderID:  "1337",
		Status:   order.New,
	}
	err = m.orderStore.add(o)
	if err != nil {
		t.Error(err)
	}

	err = m.Cancel(t.Context(), &order.Cancel{
		OrderID:   "Unknown",
		Exchange:  testExchange,
		AssetType: asset.Spot,
	})
	if err == nil {
		t.Error("Expected error due to no order found")
	}

	cancel := &order.Cancel{
		Exchange:  testExchange,
		OrderID:   "1337",
		Side:      order.Sell,
		AssetType: asset.Spot,
		Pair:      btcusdPair,
	}
	err = m.Cancel(t.Context(), cancel)
	assert.NoError(t, err)

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

func TestGetOrderInfo(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.GetOrderInfo(t.Context(), "", "", currency.EMPTYPAIR, asset.Empty)
	if err == nil {
		t.Error("Expected error due to empty order")
	}

	var result order.Detail
	result, err = m.GetOrderInfo(t.Context(),
		testExchange, "1337", currency.EMPTYPAIR, asset.Empty)
	if err != nil {
		t.Error(err)
	}
	if result.OrderID != "1337" {
		t.Error("unexpected order returned")
	}

	result, err = m.GetOrderInfo(t.Context(),
		testExchange, "1337", currency.EMPTYPAIR, asset.Empty)
	if err != nil {
		t.Error(err)
	}
	if result.OrderID != "1337" {
		t.Error("unexpected order returned")
	}
}

func TestCancelAllOrders(t *testing.T) {
	m := OrdersSetup(t)
	o := &order.Detail{
		Exchange: testExchange,
		OrderID:  "TestCancelAllOrders",
		Status:   order.New,
	}
	if err := m.orderStore.add(o); err != nil {
		t.Error(err)
	}

	m.CancelAllOrders(t.Context(), []exchange.IBotExchange{})
	checkDeets, err := m.orderStore.getByExchangeAndID(testExchange, "TestCancelAllOrders")
	if err != nil {
		t.Fatal(err)
	}
	if checkDeets.Status == order.Cancelled {
		t.Error("Order should not be cancelled")
	}

	exch, err := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	m.CancelAllOrders(t.Context(), []exchange.IBotExchange{exch})
	checkDeets, err = m.orderStore.getByExchangeAndID(testExchange, "TestCancelAllOrders")
	if err != nil {
		t.Fatal(err)
	}

	if checkDeets.Status != order.Cancelled {
		t.Error("Order should be cancelled", checkDeets.Status)
	}
}

func TestSubmit(t *testing.T) {
	m := OrdersSetup(t)
	_, err := m.Submit(t.Context(), nil)
	require.ErrorIs(t, err, errNilOrder)

	o := &order.Submit{Type: order.Market}
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected error from empty exchange")
	}

	o.Exchange = testExchange
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected error from validation")
	}

	m.cfg.EnforceLimitConfig = true
	m.cfg.AllowMarketOrders = false
	o.Pair = btcusdPair
	o.AssetType = asset.Spot
	o.Side = order.Buy
	o.Amount = 1
	o.Price = 1
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected fail due to order market type is not allowed")
	}
	m.cfg.AllowMarketOrders = true
	m.cfg.LimitAmount = 1
	o.Amount = 2
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected fail due to order limit exceeds allowed limit")
	}
	m.cfg.LimitAmount = 0
	m.cfg.AllowedExchanges = []string{"fake"}
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected fail due to order exchange not found in allowed list")
	}

	failPair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		t.Fatal(err)
	}

	m.cfg.AllowedExchanges = nil
	m.cfg.AllowedPairs = currency.Pairs{failPair}
	_, err = m.Submit(t.Context(), o)
	if err == nil {
		t.Error("Expected fail due to order pair not found in allowed list")
	}

	m.cfg.AllowedPairs = nil
	_, err = m.Submit(t.Context(), o)
	assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	err = m.orderStore.add(&order.Detail{
		Exchange: testExchange,
		OrderID:  "FakePassingExchangeOrder",
	})
	assert.NoError(t, err)

	o2, err := m.orderStore.getByExchangeAndID(testExchange, "FakePassingExchangeOrder")
	if err != nil {
		t.Error(err)
	}
	if o2.InternalOrderID.IsNil() {
		t.Error("Failed to assign internal order id")
	}
}

// TestSubmitOrderAlreadyInStore ensures that if an order is submitted, but the WS sees the conf before processSubmittedOrder
// then we don't error that it was there already
func TestSubmitOrderAlreadyInStore(t *testing.T) {
	m := OrdersSetup(t)
	submitReq := &order.Submit{
		Type:      order.Market,
		Pair:      btcusdPair,
		AssetType: asset.Spot,
		Side:      order.Buy,
		Amount:    1,
		Price:     1,
		Exchange:  testExchange,
	}
	submitResp, err := submitReq.DeriveSubmitResponse("batman.obvs")
	assert.NoError(t, err, "Deriving a SubmitResp should not error")

	id, err := uuid.NewV4()
	assert.NoError(t, err, "uuid should not error")
	d, err := submitResp.DeriveDetail(id)
	assert.NoError(t, err, "Derive Detail should not error")

	d.ClientOrderID = "SecretSquirrelSauce"
	err = m.orderStore.add(d)
	assert.NoError(t, err, "Adding an order should not error")

	resp, err := m.SubmitFakeOrder(submitReq, submitResp, false)

	if assert.NoError(t, err, "SumbitFakeOrder should not error that the order is already in the store") {
		assert.Equal(t, d.ClientOrderID, resp.ClientOrderID, "resp should contain the ClientOrderID from the store")
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
			OrderID:   "fake_order_id",
			Price:     8,
			Amount:    128,
		})
		if err != nil {
			t.Error(err)
		}

		resp, err := m.Modify(t.Context(), &mod)
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
		if det.OrderID != resp.OrderID || det.Price != price || det.Amount != amount {
			t.Errorf(
				"have (%s,%f,%f), want (%s,%f,%f)",
				det.OrderID, det.Price, det.Amount,
				resp.OrderID, price, amount,
			)
		}
	}

	model := order.Modify{
		// These fields identify the order.
		Exchange:  testExchange,
		AssetType: asset.Spot,
		Pair:      pair,
		OrderID:   "fake_order_id",
		// These fields modify the order.
		Price:  0,
		Amount: 0,
	}

	// [1] Test if nonexistent order returns an error.
	one := model
	one.OrderID = "nonexistent_order_id"
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
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	fakeExchange := omfExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

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
				AssetEnabled: true,
				Enabled:      pairs,
				Available:    pairs,
			},
			asset.Futures: {
				AssetEnabled: true,
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
					AssetEnabled: true,
					Enabled:      pairs,
					Available:    pairs,
				},
				asset.Futures: {
					AssetEnabled: true,
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
			OrderID:     "Order1-unknown-to-active",
		},
		{
			Exchange:    testExchange,
			Pair:        pairs[0],
			AssetType:   asset.Spot,
			Amount:      1.0,
			Side:        order.Sell,
			Status:      order.Active,
			LastUpdated: time.Now().Add(-time.Hour),
			OrderID:     "Order2-active-to-inactive",
		},
		{
			Exchange:    testExchange,
			Pair:        pairs[0],
			AssetType:   asset.Spot,
			Amount:      2.0,
			Side:        order.Sell,
			Status:      order.UnknownStatus,
			LastUpdated: time.Now().Add(-time.Hour),
			OrderID:     "Order3-unknown-to-active",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}

	m.orderStore.futuresPositionController = futures.SetupPositionController()
	if err = m.orderStore.add(&order.Detail{
		Exchange:    testExchange,
		Pair:        pairs[0],
		AssetType:   asset.Futures,
		Amount:      2.0,
		Side:        order.Short,
		Status:      order.Open,
		LastUpdated: time.Now().Add(-time.Hour),
		OrderID:     "4",
		Date:        time.Now(),
	}); err != nil {
		t.Error(err)
	}

	m.processOrders()

	// Order1 is not returned by exch.GetActiveOrders()
	// It will be fetched by exch.GetOrderInfo(), which will say it is active
	res, err := m.GetOrdersFiltered(&order.Filter{OrderID: "Order1-unknown-to-active"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].Status != order.Active {
		t.Errorf("Order 1 should be active, but status is %s", res[0].Status)
	}

	// Order2 is not returned by exch.GetActiveOrders()
	// It will be fetched by exch.GetOrderInfo(), which will say it is cancelled
	res, err = m.GetOrdersFiltered(&order.Filter{OrderID: "Order2-active-to-inactive"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].Status != order.Cancelled {
		t.Errorf("Order 2 should be cancelled, but status is %s", res[0].Status)
	}

	// Order3 is returned by exch.GetActiveOrders(), which will say it is active
	res, err = m.GetOrdersFiltered(&order.Filter{OrderID: "Order3-unknown-to-active"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].Status != order.Active {
		t.Errorf("Order 3 should be active, but status is %s", res[0].Status)
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
			OrderID:  "Test1",
		},
		{
			Exchange: testExchange,
			OrderID:  "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res, err := m.GetOrdersFiltered(&order.Filter{OrderID: "Test2"})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
}

func TestGetFilteredOrders(t *testing.T) {
	m := OrdersSetup(t)

	_, err := m.orderStore.getFilteredOrders(nil)
	if err == nil {
		t.Error("Error expected when Filter is nil")
	}

	orders := []order.Detail{
		{
			Exchange: testExchange,
			OrderID:  "Test1",
		},
		{
			Exchange: testExchange,
			OrderID:  "Test2",
		},
	}
	for i := range orders {
		if err = m.orderStore.add(&orders[i]); err != nil {
			t.Error(err)
		}
	}
	res, err := m.orderStore.getFilteredOrders(&order.Filter{OrderID: "Test1"})
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
			OrderID:  "Test1",
		},
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Sell,
			Status:   order.Active,
			OrderID:  "Test2",
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

func TestProcessMatchingOrders(t *testing.T) {
	m := OrdersSetup(t)
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	orders := []order.Detail{
		{
			Exchange:    testExchange,
			OrderID:     "Test2",
			LastUpdated: time.Now(),
		},
		{
			Exchange:    testExchange,
			OrderID:     "Test4",
			LastUpdated: time.Now().Add(-time.Hour),
		},
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go m.processMatchingOrders(exch, orders, &wg)
	wg.Wait()
	res, err := m.GetOrdersFiltered(&order.Filter{Exchange: testExchange})
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got: %d", len(res))
	}
	if res[0].OrderID != "Test4" {
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
		OrderID:  "Test",
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

func TestGetActiveOrders(t *testing.T) {
	m := OrdersSetup(t)
	var err error
	orders := []order.Detail{
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Buy,
			Status:   order.Cancelled,
			OrderID:  "Test1",
		},
		{
			Exchange: testExchange,
			Amount:   1.0,
			Side:     order.Sell,
			Status:   order.Active,
			OrderID:  "Test2",
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
	cp := currency.NewBTCUSDT()
	_, err := o.GetFuturesPositionsForExchange("test", asset.Spot, cp)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	o.orderStore.futuresPositionController = futures.SetupPositionController()
	_, err = o.GetFuturesPositionsForExchange("test", asset.Spot, cp)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	assert.ErrorIs(t, err, futures.ErrPositionNotFound)

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		OrderID:   "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1,
	})
	assert.NoError(t, err)

	resp, err := o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	assert.NoError(t, err)

	if len(resp) != 1 {
		t.Error("expected 1 position")
	}

	o = nil
	_, err = o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestClearFuturesPositionsForExchange(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	cp := currency.NewBTCUSDT()
	err := o.ClearFuturesTracking("test", asset.Spot, cp)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	o.orderStore.futuresPositionController = futures.SetupPositionController()
	err = o.ClearFuturesTracking("test", asset.Spot, cp)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	assert.ErrorIs(t, err, futures.ErrPositionNotFound)

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		OrderID:   "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1,
	})
	assert.NoError(t, err)

	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	assert.NoError(t, err)

	resp, err := o.GetFuturesPositionsForExchange("test", asset.Futures, cp)
	assert.NoError(t, err)

	if len(resp) != 0 {
		t.Errorf("expected no position, received '%v'", len(resp))
	}

	o = nil
	err = o.ClearFuturesTracking("test", asset.Futures, cp)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestUpdateOpenPositionUnrealisedPNL(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	cp := currency.NewBTCUSDT()
	_, err := o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	o.orderStore.futuresPositionController = futures.SetupPositionController()
	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Futures, cp, 1, time.Now())
	assert.ErrorIs(t, err, futures.ErrPositionNotFound)

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		OrderID:   "test",
		Date:      time.Now(),
		Exchange:  "test",
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      order.Buy,
		Amount:    1,
		Price:     1,
	})
	assert.NoError(t, err)

	unrealised, err := o.UpdateOpenPositionUnrealisedPNL("test", asset.Futures, cp, 2, time.Now())
	assert.NoError(t, err)

	if !unrealised.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v', expected '%v'", unrealised, 1)
	}

	o = nil
	_, err = o.UpdateOpenPositionUnrealisedPNL("test", asset.Spot, cp, 1, time.Now())
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestSubmitFakeOrder(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	resp := &order.SubmitResponse{}
	_, err := o.SubmitFakeOrder(nil, resp, false)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	_, err = o.SubmitFakeOrder(nil, resp, false)
	assert.ErrorIs(t, err, errNilOrder)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	o.orderStore.exchangeManager = em

	ord := &order.Submit{}
	_, err = o.SubmitFakeOrder(ord, resp, false)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	ord.Exchange = testExchange
	ord.AssetType = asset.Spot
	ord.Pair = currency.NewPair(currency.BTC, currency.DOGE)
	ord.Side = order.Buy
	ord.Type = order.Market
	ord.Amount = 1337

	resp, err = ord.DeriveSubmitResponse("1234")
	if err != nil {
		t.Fatal(err)
	}

	resp.Status = order.Filled
	o.orderStore.commsManager = &CommunicationManager{}
	o.orderStore.Orders = make(map[string][]*order.Detail)
	_, err = o.SubmitFakeOrder(ord, resp, false)
	assert.NoError(t, err)
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
	assert.ErrorIs(t, err, errNilOrder)

	od := &order.Detail{Exchange: testExchange}
	err = s.updateExisting(od)
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	s.Orders[strings.ToLower(testExchange)] = nil
	err = s.updateExisting(od)
	assert.ErrorIs(t, err, ErrOrderNotFound)

	od.Exchange = testExchange
	od.AssetType = asset.Futures
	od.OrderID = "123"
	od.Pair = currency.NewBTCUSDT()
	od.Side = order.Buy
	od.Type = order.Market
	od.Date = time.Now()
	od.Amount = 1337
	s.Orders[strings.ToLower(testExchange)] = []*order.Detail{
		od,
	}
	s.futuresPositionController = futures.SetupPositionController()
	err = s.futuresPositionController.TrackNewOrder(od)
	assert.NoError(t, err)

	err = s.updateExisting(od)
	assert.NoError(t, err)

	pos, err := s.futuresPositionController.GetPositionsForExchange(testExchange, asset.Futures, od.Pair)
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	err = o.Add(nil)
	assert.ErrorIs(t, err, errNilOrder)

	o = nil
	err = o.Add(nil)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestGetAllOpenFuturesPositions(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	o, err := SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

	o.started = 0
	_, err = o.GetAllOpenFuturesPositions()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	o.activelyTrackFuturesPositions = true
	o.orderStore.futuresPositionController = futures.SetupPositionController()
	_, err = o.GetAllOpenFuturesPositions()
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound)

	o = nil
	_, err = o.GetAllOpenFuturesPositions()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestGetOpenFuturesPosition(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	o, err := SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

	o.started = 0
	cp := currency.NewPair(currency.BTC, currency.PERP)
	_, err = o.GetOpenFuturesPosition(testExchange, asset.Spot, cp)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started = 1
	_, err = o.GetOpenFuturesPosition(testExchange, asset.Spot, cp)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	o, err = SetupOrderManager(em, &CommunicationManager{}, wg, &config.OrderManager{
		Enabled:                       true,
		FuturesTrackingSeekDuration:   time.Hour,
		Verbose:                       true,
		ActivelyTrackFuturesPositions: true,
	})
	assert.NoError(t, err)

	o.started = 1

	_, err = o.GetOpenFuturesPosition(testExchange, asset.Spot, cp)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = o.GetOpenFuturesPosition(testExchange, asset.Futures, cp)
	assert.ErrorIs(t, err, futures.ErrPositionNotFound)

	err = o.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		AssetType: asset.Futures,
		OrderID:   "123",
		Pair:      cp,
		Side:      order.Buy,
		Type:      order.Market,
		Date:      time.Now(),
		Amount:    1337,
		Exchange:  testExchange,
	})
	assert.NoError(t, err)

	_, err = o.GetOpenFuturesPosition(testExchange, asset.Futures, cp)
	assert.NoError(t, err)

	o = nil
	_, err = o.GetOpenFuturesPosition(testExchange, asset.Spot, cp)
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestProcessFuturesPositions(t *testing.T) {
	t.Parallel()
	o := &OrderManager{}
	err := o.processFuturesPositions(nil, nil)
	assert.ErrorIs(t, err, errFuturesTrackingDisabled)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-perp")
	if err != nil {
		t.Fatal(err)
	}
	cp2, err := currency.NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	o, err = SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{ActivelyTrackFuturesPositions: true, FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

	o.started = 1

	err = o.processFuturesPositions(fakeExchange, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	position := &futures.PositionResponse{
		Asset:  asset.Spot,
		Pair:   cp,
		Orders: nil,
	}
	err = o.processFuturesPositions(fakeExchange, position)
	assert.ErrorIs(t, err, errNilOrder)

	od := &order.Detail{
		AssetType: asset.Spot,
		OrderID:   "123",
		Pair:      cp,
		Side:      order.Buy,
		Type:      order.Market,
		Date:      time.Now().Add(-time.Hour),
		Amount:    1337,
		Exchange:  b.Name,
	}
	position.Orders = []order.Detail{
		*od,
	}
	err = o.processFuturesPositions(fakeExchange, position)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	position.Orders[0].AssetType = asset.Futures
	position.Asset = asset.Futures
	err = o.processFuturesPositions(fakeExchange, position)
	assert.NoError(t, err)

	b.Features.Supports.FuturesCapabilities.FundingRates = true
	err = o.processFuturesPositions(fakeExchange, position)
	assert.NoError(t, err)
}

// TestGetByDetail tests orderstore.getByDetail
func TestGetByDetail(t *testing.T) {
	t.Parallel()
	m := OrdersSetup(t)
	assert.Nil(t, m.orderStore.getByDetail(nil), "Fetching a nil order should return nil")
	od := &order.Detail{
		Exchange:      testExchange,
		AssetType:     asset.Spot,
		OrderID:       "AdmiralHarold",
		ClientOrderID: "DuskyLeafMonkey",
	}
	id := &order.Detail{
		Exchange: od.Exchange,
		OrderID:  od.OrderID,
	}

	assert.Nil(t, m.orderStore.getByDetail(od), "Fetching a non-stored order should return nil")
	assert.NoError(t, m.orderStore.add(od), "Adding the details should not error")

	byOrig := m.orderStore.getByDetail(od)
	byID := m.orderStore.getByDetail(id)

	if assert.NotNil(t, byOrig, "Retrieve by orig pointer should find a record") {
		assert.NotSame(t, byOrig, od, "Retrieve by orig pointer should return a new pointer")
		assert.Equal(t, od.ClientOrderID, byOrig.ClientOrderID, "Retrieve by orig pointer should contain the correct ClientOrderID")
	}

	if assert.NotNil(t, byID, "Retrieve by new pointer should find a record") {
		assert.NotSame(t, byID, id, "Retrieve by new pointer should return a different new pointer than we passed in")
		assert.NotSame(t, byID, od, "Retrieve by new pointer should return a different new pointer than the original object")
		assert.Equal(t, od.ClientOrderID, byID.ClientOrderID, "Retrieve by id pointer should contain the correct ClientOrderID")
	}
}

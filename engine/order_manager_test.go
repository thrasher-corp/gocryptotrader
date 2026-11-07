package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// omfExchange aka order manager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type omfExchange struct {
	exchange.IBotExchange
}

type gracefulShutdownCtxObservation struct {
	err         error
	hasDeadline bool
}

type gracefulShutdownCtxExchange struct {
	omfExchange
	seenCtx chan gracefulShutdownCtxObservation
}

func (f gracefulShutdownCtxExchange) CancelOrder(ctx context.Context, _ *order.Cancel) error {
	_, hasDeadline := ctx.Deadline()
	f.seenCtx <- gracefulShutdownCtxObservation{err: ctx.Err(), hasDeadline: hasDeadline}
	return ctx.Err()
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
		BaseVolume:            1337,
		QuoteVolume:           1337,
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
	id := uuid.NewV4()
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
	err := m.Start(t.Context())
	assert.ErrorIs(t, err, ErrNilSubsystem)

	var wg sync.WaitGroup
	m, err = SetupOrderManager(NewExchangeManager(), &CommunicationManager{}, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	err = m.Start(t.Context())
	assert.NoError(t, err)

	err = m.Start(t.Context())
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

	err = m.Start(t.Context())
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

	err = m.Start(t.Context())
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

	m.started.Store(true)
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

// cancelRecordingExchange records each cancellation it receives and returns err for it
type cancelRecordingExchange struct {
	exchange.IBotExchange
	err       error
	cancelled []order.Cancel
}

func (c *cancelRecordingExchange) CancelOrder(_ context.Context, o *order.Cancel) error {
	c.cancelled = append(c.cancelled, *o)
	return c.err
}

// cancelRecordingSetup returns a started order manager for a cancelRecordingExchange, set up without API calls
func cancelRecordingSetup(t *testing.T) (*OrderManager, *cancelRecordingExchange) {
	t.Helper()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	fake := &cancelRecordingExchange{IBotExchange: exch}
	require.NoError(t, em.Add(fake), "Add must not error")
	m, err := SetupOrderManager(em, &CommunicationManager{}, &sync.WaitGroup{}, &config.OrderManager{})
	require.NoError(t, err, "SetupOrderManager must not error")
	m.started.Store(true)
	return m, fake
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	var nilManager *OrderManager
	assert.ErrorIs(t, nilManager.Cancel(t.Context(), nil), ErrNilSubsystem, "Cancel should error for a nil order manager")
	assert.ErrorIs(t, new(OrderManager).Cancel(t.Context(), nil), ErrSubSystemNotStarted, "Cancel should error before the order manager starts")

	m, exch := cancelRecordingSetup(t)
	assert.ErrorContains(t, m.Cancel(t.Context(), nil), "order cancel param is nil", "Cancel should error for a nil cancellation")
	assert.ErrorContains(t, m.Cancel(t.Context(), &order.Cancel{}), "order exchange name is empty", "Cancel should error for an empty cancellation")
	assert.ErrorContains(t, m.Cancel(t.Context(), &order.Cancel{Exchange: testExchange}), "order id is empty", "Cancel should error without an order ID")
	assert.ErrorContains(t, m.Cancel(t.Context(), &order.Cancel{OrderID: "ID"}), "order exchange name is empty", "Cancel should error without an exchange")
	assert.ErrorIs(t, m.Cancel(t.Context(), &order.Cancel{OrderID: "ID", Exchange: "unloaded"}), ErrExchangeNotFound, "Cancel should error for an exchange that is not loaded")
	assert.ErrorIs(t, m.Cancel(t.Context(), &order.Cancel{OrderID: "ID", Exchange: testExchange, AssetType: asset.Binary}), asset.ErrNotSupported, "Cancel should error for an asset the exchange does not support")
	assert.Empty(t, exch.cancelled, "Cancel should not send an invalid cancellation to the exchange")

	stop := order.Detail{Exchange: testExchange, OrderID: "stop", Pair: btcusdPair, AssetType: asset.Spot, Type: order.StopLimit, Side: order.Sell, Status: order.New}
	stored := []order.Detail{
		{Exchange: testExchange, OrderID: "1337", Status: order.New},
		stop,
		{Exchange: testExchange, OrderID: "futures without a pair", AssetType: asset.Futures, Type: order.Limit, Side: order.Buy, Status: order.New},
	}
	for _, tc := range []struct {
		name         string
		cancel       order.Cancel
		exchangeErr  error
		sent         order.Cancel
		err          error
		storedStatus map[string]order.Status
	}{
		{
			name:         "a stored order",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "1337", Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			sent:         order.Cancel{Exchange: testExchange, OrderID: "1337", Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			storedStatus: map[string]order.Status{"1337": order.Cancelled, "stop": order.New, "futures without a pair": order.New},
		},
		{
			name:         "a stored order without a type in the cancellation",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "stop", Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			sent:         order.Cancel{Exchange: testExchange, OrderID: "stop", Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			storedStatus: map[string]order.Status{"1337": order.New, "stop": order.Cancelled, "futures without a pair": order.New},
		},
		{
			name:         "a stored order with a type in the cancellation",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "stop", Type: order.Limit, Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			sent:         order.Cancel{Exchange: testExchange, OrderID: "stop", Type: order.Limit, Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			storedStatus: map[string]order.Status{"1337": order.New, "stop": order.Cancelled, "futures without a pair": order.New},
		},
		{
			name:         "an order the store does not hold",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "Unknown", AssetType: asset.Spot},
			sent:         order.Cancel{Exchange: testExchange, OrderID: "Unknown", AssetType: asset.Spot},
			err:          ErrOrderNotFound,
			storedStatus: map[string]order.Status{"1337": order.New, "stop": order.New, "futures without a pair": order.New},
		},
		{
			name:         "a cancellation the exchange rejects",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "stop", Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			exchangeErr:  errExpectedTestError,
			sent:         order.Cancel{Exchange: testExchange, OrderID: "stop", Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Pair: btcusdPair},
			err:          errExpectedTestError,
			storedStatus: map[string]order.Status{"1337": order.New, "stop": order.New, "futures without a pair": order.New},
		},
		{
			name:         "an order the store cannot track",
			cancel:       order.Cancel{Exchange: testExchange, OrderID: "futures without a pair"},
			sent:         order.Cancel{Exchange: testExchange, OrderID: "futures without a pair", Type: order.Limit},
			err:          order.ErrPairIsEmpty,
			storedStatus: map[string]order.Status{"1337": order.New, "stop": order.New, "futures without a pair": order.Cancelled},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, exch := cancelRecordingSetup(t)
			exch.err = tc.exchangeErr
			for i := range stored {
				m.orderStore.Orders[strings.ToLower(testExchange)] = append(m.orderStore.Orders[strings.ToLower(testExchange)], stored[i].CopyToPointer())
			}
			cancel := tc.cancel
			assert.ErrorIs(t, m.Cancel(t.Context(), &cancel), tc.err, "Cancel should return the expected error")
			assert.Equal(t, []order.Cancel{tc.sent}, exch.cancelled, "Cancel should send the exchange the expected cancellation")
			assert.Equal(t, tc.cancel, cancel, "Cancel should not change the caller's cancellation")
			statuses := make(map[string]order.Status)
			for _, d := range m.orderStore.get()[strings.ToLower(testExchange)] {
				statuses[d.OrderID] = d.Status
			}
			assert.Equal(t, tc.storedStatus, statuses, "Cancel should leave the stored orders with the expected statuses")
		})
	}

	t.Run("a stopped order manager when running is not required", func(t *testing.T) {
		t.Parallel()
		m, exch := cancelRecordingSetup(t)
		m.started.Store(false)
		m.orderStore.Orders[strings.ToLower(testExchange)] = []*order.Detail{stop.CopyToPointer()}
		require.NoError(t, m.cancel(t.Context(), &order.Cancel{Exchange: testExchange, OrderID: "stop"}, false), "cancel must not error when running is not required")
		assert.Equal(t, []order.Cancel{{Exchange: testExchange, OrderID: "stop", Type: order.StopLimit}}, exch.cancelled, "cancel should send the exchange the stored order's type")
		got, err := m.orderStore.getByExchangeAndID(testExchange, "stop")
		require.NoError(t, err, "getByExchangeAndID must not error")
		assert.Equal(t, order.Cancelled, got.Status, "cancel should mark the stored order cancelled")
	})
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

func TestOrderManagerGracefulShutdownUsesBoundedContextWhenRuntimeCancelled(t *testing.T) {
	var wg sync.WaitGroup
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)

	cfg, err := exchange.GetDefaultConfig(t.Context(), exch)
	require.NoError(t, err)
	require.NoError(t, exch.Setup(cfg))

	seenCtx := make(chan gracefulShutdownCtxObservation, 1)
	fakeExchange := gracefulShutdownCtxExchange{
		IBotExchange: exch,
		seenCtx:      seenCtx,
	}
	require.NoError(t, em.Add(fakeExchange))

	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{CancelOrdersOnShutdown: true})
	require.NoError(t, err)
	m.started.Store(true)

	require.NoError(t, m.orderStore.add(&order.Detail{
		Exchange:  testExchange,
		OrderID:   "shutdown-context-cancel",
		Status:    order.New,
		Pair:      btcusdPair,
		AssetType: asset.Spot,
	}))

	runtimeCtx, cancel := context.WithCancel(t.Context())
	cancel()

	m.gracefulShutdown(runtimeCtx)

	select {
	case obs := <-seenCtx:
		require.NoError(t, obs.err)
		assert.True(t, obs.hasDeadline)
	default:
		t.Fatal("expected cancel order to be called with shutdown context")
	}
}

func TestOrderManagerStopCancelsOrdersOnShutdown(t *testing.T) {
	var wg sync.WaitGroup
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)

	cfg, err := exchange.GetDefaultConfig(t.Context(), exch)
	require.NoError(t, err)
	require.NoError(t, exch.Setup(cfg))

	seenCtx := make(chan gracefulShutdownCtxObservation, 1)
	fakeExchange := gracefulShutdownCtxExchange{
		IBotExchange: exch,
		seenCtx:      seenCtx,
	}
	require.NoError(t, em.Add(fakeExchange))

	m, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{CancelOrdersOnShutdown: true})
	require.NoError(t, err)
	require.NoError(t, m.Start(t.Context()))

	const orderID = "shutdown-stop-cancel"
	require.NoError(t, m.orderStore.add(&order.Detail{
		Exchange:  testExchange,
		OrderID:   orderID,
		Status:    order.New,
		Pair:      btcusdPair,
		AssetType: asset.Spot,
	}))

	require.NoError(t, m.Stop())

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for order manager shutdown")
	}

	select {
	case obs := <-seenCtx:
		require.NoError(t, obs.err)
		assert.True(t, obs.hasDeadline)
	default:
		t.Fatal("expected cancel order to be called during Stop")
	}

	checkDeets, err := m.orderStore.getByExchangeAndID(testExchange, orderID)
	require.NoError(t, err)
	assert.Equal(t, order.Cancelled, checkDeets.Status)
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
	if o2.InternalOrderID == uuid.Nil() {
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

	id := uuid.NewV4()
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

	m.started.Store(true)
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

	m.processOrders(t.Context())

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
	go m.processMatchingOrders(t.Context(), exch, orders, &wg)
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
	err = m.FetchAndUpdateExchangeOrder(t.Context(), exch, nil, asset.Spot)
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
	err = m.FetchAndUpdateExchangeOrder(t.Context(), exch, o, asset.Spot)
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
	err = m.FetchAndUpdateExchangeOrder(t.Context(), exch, o, asset.Spot)
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

	o.started.Store(true)
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

	o.started.Store(true)
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

	o.started.Store(true)
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

	o.started.Store(true)
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
	var nilManager *OrderManager
	assert.Nil(t, nilManager.GetOrdersSnapshot(order.AnyStatus), "GetOrdersSnapshot should return no orders for a nil order manager")
	o := &OrderManager{}
	assert.Nil(t, o.GetOrdersSnapshot(order.AnyStatus), "GetOrdersSnapshot should return no orders before the order manager starts")
	o.started.Store(true)
	o.orderStore.Orders = make(map[string][]*order.Detail)
	o.orderStore.Orders[testExchange] = []*order.Detail{
		{
			Status: order.Open,
		},
		{
			OrderID: "filled",
			Status:  order.Filled,
		},
	}
	all := []order.Detail{{Status: order.Open}, {OrderID: "filled", Status: order.Filled}}
	for _, tc := range []struct {
		status order.Status
		exp    []order.Detail
	}{
		{status: order.Open, exp: []order.Detail{{Status: order.Open}}},
		{status: order.Closed},
		{status: order.AnyStatus, exp: all},
		{status: order.UnknownStatus, exp: all},
	} {
		assert.Equalf(t, tc.exp, o.GetOrdersSnapshot(tc.status), "GetOrdersSnapshot should return the orders for status %s", tc.status)
	}

	t.Run("waits for the order store's lock", func(t *testing.T) {
		t.Parallel()
		m := &OrderManager{}
		m.started.Store(true)
		stored := &order.Detail{OrderID: "1", Status: order.Open}
		m.orderStore.Orders = map[string][]*order.Detail{testExchange: {stored}}
		m.orderStore.m.Lock()
		snapshot := make(chan []order.Detail, 1)
		go func() { snapshot <- m.GetOrdersSnapshot(order.AnyStatus) }()
		// Gives a snapshot that ignores the lock time to read the order before it changes
		time.Sleep(20 * time.Millisecond)
		stored.Status = order.Filled
		m.orderStore.m.Unlock()
		select {
		case got := <-snapshot:
			assert.Equal(t, []order.Detail{{OrderID: "1", Status: order.Filled}}, got, "GetOrdersSnapshot should wait for the order store's lock")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "GetOrdersSnapshot must return once the order store's lock is released")
		}
	})

	t.Run("reads alongside another reader", func(t *testing.T) {
		t.Parallel()
		m := &OrderManager{}
		m.started.Store(true)
		m.orderStore.Orders = map[string][]*order.Detail{testExchange: {{OrderID: "1", Status: order.Open}}}
		m.orderStore.m.RLock()
		defer m.orderStore.m.RUnlock()
		snapshot := make(chan []order.Detail, 1)
		go func() { snapshot <- m.GetOrdersSnapshot(order.AnyStatus) }()
		select {
		case got := <-snapshot:
			assert.Equal(t, []order.Detail{{OrderID: "1", Status: order.Open}}, got, "GetOrdersSnapshot should return the stored order")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "GetOrdersSnapshot must not wait for another reader of the order store")
		}
	})

	t.Run("concurrent store updates", func(t *testing.T) {
		t.Parallel()
		m := &OrderManager{}
		m.started.Store(true)
		m.orderStore.Orders = map[string][]*order.Detail{strings.ToLower(testExchange): {{Exchange: testExchange, OrderID: "1", Status: order.Open}}}
		// The workers report to the test, so one finishing after a failed wait cannot fail a test that has ended
		updateErrs, snapshotLens := make(chan error, 100), make(chan int, 100)
		var wg sync.WaitGroup
		for i := range 100 {
			wg.Go(func() {
				updateErrs <- m.orderStore.updateExisting(&order.Detail{Exchange: testExchange, OrderID: "1", Price: float64(i + 1)})
			})
			wg.Go(func() {
				snapshotLens <- len(m.GetOrdersSnapshot(order.AnyStatus))
			})
		}
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "GetOrdersSnapshot and the order store's updates must not deadlock")
		}
		close(updateErrs)
		close(snapshotLens)
		for err := range updateErrs {
			assert.NoError(t, err, "updateExisting should not error")
		}
		for n := range snapshotLens {
			assert.Equal(t, 1, n, "GetOrdersSnapshot should return the stored order")
		}
	})
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
	o.started.Store(true)
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

	o.started.Store(true)
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

	o.started.Store(false)
	_, err = o.GetAllOpenFuturesPositions()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started.Store(true)
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

	o.started.Store(false)
	cp := currency.NewPair(currency.BTC, currency.PERP)
	_, err = o.GetOpenFuturesPosition(testExchange, asset.Spot, cp)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	o.started.Store(true)
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

	o.started.Store(true)

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
	err := o.processFuturesPositions(t.Context(), nil, nil)
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

	o.started.Store(true)

	err = o.processFuturesPositions(t.Context(), fakeExchange, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	position := &futures.PositionResponse{
		Asset:  asset.Spot,
		Pair:   cp,
		Orders: nil,
	}
	err = o.processFuturesPositions(t.Context(), fakeExchange, position)
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
	err = o.processFuturesPositions(t.Context(), fakeExchange, position)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	position.Orders[0].AssetType = asset.Futures
	position.Asset = asset.Futures
	err = o.processFuturesPositions(t.Context(), fakeExchange, position)
	assert.NoError(t, err)

	b.Features.Supports.FuturesCapabilities.FundingRates = true
	err = o.processFuturesPositions(t.Context(), fakeExchange, position)
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

// feesExchange serves programmed order lookups, active orders, submissions and position orders, and records the last
// submission it receives, so tests can follow fees and orders through the order manager and RPC server without API calls
type feesExchange struct {
	exchange.IBotExchange
	orderInfo      *order.Detail
	activeOrders   []order.Detail
	submitted      *order.SubmitResponse
	submission     *order.Submit
	positionOrders []order.Detail
}

func (f *feesExchange) GetOrderInfo(context.Context, string, currency.Pair, asset.Item) (*order.Detail, error) {
	return f.orderInfo.CopyToPointer(), nil
}

func (f *feesExchange) GetActiveOrders(context.Context, *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return f.activeOrders, nil
}

func (f *feesExchange) SubmitOrder(_ context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	f.submission = s
	return f.submitted, nil
}

func (f *feesExchange) GetFuturesPositionOrders(_ context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	return []futures.PositionResponse{{Asset: req.Asset, Pair: req.Pairs[0], Orders: f.positionOrders}}, nil
}

// setupFeesExchange returns an RPC server whose order manager tracks orders from a feesExchange trading BTCUSDT spot
// and USDT margined futures, set up without API calls
func setupFeesExchange(t *testing.T) (*RPCServer, *feesExchange) {
	t.Helper()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Binance")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	b := exch.GetBase()
	b.States = currencystate.NewCurrencyStates()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		b.CurrencyPairs.Pairs[a] = &currency.PairStore{
			Available:     currency.Pairs{currency.NewBTCUSDT()},
			Enabled:       currency.Pairs{currency.NewBTCUSDT()},
			AssetEnabled:  true,
			ConfigFormat:  &currency.PairFormat{Uppercase: true},
			RequestFormat: &currency.PairFormat{Uppercase: true},
		}
	}
	fake := &feesExchange{IBotExchange: exch}
	require.NoError(t, em.Add(fake), "Add must not error")
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
	require.NoError(t, err, "SetupOrderManager must not error")
	om.started.Store(true)
	return &RPCServer{Engine: &Engine{Config: &config.Config{}, ExchangeManager: em, OrderManager: om}}, fake
}

func newTestFees(t *testing.T, amounts ...order.FeeAmount) order.Fees {
	t.Helper()
	f, err := order.NewFees(amounts...)
	require.NoError(t, err, "NewFees must not error")
	return f
}

// TestOrderManagerFees follows #2456's lookups through the order manager's REST paths, GetOrderInfo and
// FetchAndUpdateExchangeOrder: a second currency must replace the stored single fee, a lookup without fees keeps the
// stored ones even when it carries a single fee, and an empty snapshot clears them
func TestOrderManagerFees(t *testing.T) {
	t.Parallel()
	s, exch := setupFeesExchange(t)
	btc := newTestFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.001})
	btcUSDT := newTestFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.001}, order.FeeAmount{Currency: currency.USDT, Amount: 0.2})
	none := newTestFees(t)
	for _, step := range []struct {
		name     string
		fees     order.Fees
		fee      float64
		expFees  order.Fees
		expFee   float64
		expAsset currency.Code
	}{
		{name: "one currency", fees: btc, expFees: btc, expFee: 0.001, expAsset: currency.BTC},
		{name: "two currencies", fees: btcUSDT, expFees: btcUSDT},
		{name: "a single fee without fees", fee: 0.3, expFees: btcUSDT},
		{name: "no fees", fees: none, expFees: none},
	} {
		exch.orderInfo = &order.Detail{Exchange: exch.GetName(), OrderID: "1", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Status: order.PartiallyFilled, Fee: step.fee}
		exch.orderInfo.SetFees(step.fees)
		got, err := s.OrderManager.GetOrderInfo(t.Context(), exch.GetName(), "1", currency.NewBTCUSDT(), asset.Spot)
		require.NoErrorf(t, err, "GetOrderInfo must not error for %s", step.name)
		stored, err := s.OrderManager.GetByExchangeAndID(exch.GetName(), "1")
		require.NoErrorf(t, err, "GetByExchangeAndID must not error for %s", step.name)
		for _, d := range []*order.Detail{&got, stored} {
			assert.Equalf(t, step.expFees, d.Fees, "the order should have the expected fees for %s", step.name)
			assert.Equalf(t, step.expFee, d.Fee, "the order should have the expected single fee for %s", step.name)
			assert.Equalf(t, step.expAsset, d.FeeAsset, "the order should have the expected single fee asset for %s", step.name)
		}
	}

	exch.orderInfo.SetFees(btcUSDT)
	err := s.OrderManager.FetchAndUpdateExchangeOrder(t.Context(), exch, &order.Detail{OrderID: "1", Pair: currency.NewBTCUSDT()}, asset.Spot)
	require.NoError(t, err, "FetchAndUpdateExchangeOrder must not error")
	stored, err := s.OrderManager.GetByExchangeAndID(exch.GetName(), "1")
	require.NoError(t, err, "GetByExchangeAndID must not error")
	assert.Equal(t, btcUSDT, stored.Fees, "FetchAndUpdateExchangeOrder should replace the stored fees")
	assert.Zero(t, stored.Fee, "FetchAndUpdateExchangeOrder should not leave a single fee for fees in two currencies")
}

// TestOrderStoreFeesIsolation checks that an order's fees in the order store cannot be changed from outside it: not
// through the update that was merged into it, nor through any order the store returns
func TestOrderStoreFeesIsolation(t *testing.T) {
	t.Parallel()
	s, exch := setupFeesExchange(t)
	m := s.OrderManager
	exp := []order.FeeAmount{{Currency: currency.BTC, Amount: 0.001}, {Currency: currency.USDT, Amount: 0.2}}
	fees := newTestFees(t, exp...)
	other := newTestFees(t, order.FeeAmount{Currency: currency.ETH, Amount: 1})

	_, err := m.UpsertOrder(&order.Detail{Exchange: exch.GetName(), OrderID: "1", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Status: order.Active, Amount: 1})
	require.NoError(t, err, "UpsertOrder must not error adding the order")
	update := &order.Detail{Exchange: exch.GetName(), OrderID: "1", Trades: []order.TradeHistory{{TID: "1"}}}
	update.SetFees(fees)
	update.Trades[0].SetFees(fees)
	upserted, err := m.UpsertOrder(update)
	require.NoError(t, err, "UpsertOrder must not error updating the order")
	byID, err := m.GetByExchangeAndID(exch.GetName(), "1")
	require.NoError(t, err, "GetByExchangeAndID must not error")
	filtered, err := m.GetOrdersFiltered(&order.Filter{Exchange: exch.GetName()})
	require.NoError(t, err, "GetOrdersFiltered must not error")
	require.Len(t, filtered, 1, "GetOrdersFiltered must return the order")
	active, err := m.GetOrdersActive(&order.Filter{Exchange: exch.GetName()})
	require.NoError(t, err, "GetOrdersActive must not error")
	require.Len(t, active, 1, "GetOrdersActive must return the order")
	snapshot := m.GetOrdersSnapshot(order.AnyStatus)
	require.Len(t, snapshot, 1, "GetOrdersSnapshot must return the order")
	all := m.orderStore.get()[strings.ToLower(exch.GetName())]
	require.Len(t, all, 1, "get must return the order")

	for _, d := range []*order.Detail{update, &upserted.OrderDetails, byID, &filtered[0], &active[0], &snapshot[0], all[0]} {
		require.Len(t, d.Trades, 1, "every copy must hold the fill")
		d.SetFees(other)
		d.Trades[0].SetFees(other)
	}
	stored, err := m.GetByExchangeAndID(exch.GetName(), "1")
	require.NoError(t, err, "GetByExchangeAndID must not error")
	assert.Equal(t, exp, stored.Fees.Amounts(), "the stored order's fees should not change from outside the store")
	require.Len(t, stored.Trades, 1, "the stored order must hold the fill")
	assert.Equal(t, exp, stored.Trades[0].Fees.Amounts(), "the stored fill's fees should not change from outside the store")
}

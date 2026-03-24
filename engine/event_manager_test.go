package engine

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type testCommsManager struct {
	events []base.Event
}

func (t *testCommsManager) PushEvent(evt base.Event) {
	t.events = append(t.events, evt)
}

type testExchangeManager struct {
	validExchange string
}

func (t *testExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	return nil, nil
}

func (t *testExchangeManager) GetExchangeByName(exchangeName string) (exchange.IBotExchange, error) {
	if strings.EqualFold(exchangeName, t.validExchange) {
		return nil, nil
	}
	return nil, errors.New("exchange should not be found")
}

func setupTestEventManager(t *testing.T, exchangeManager iExchangeManager) *eventManager {
	t.Helper()
	m, err := setupEventManager(&CommunicationManager{}, exchangeManager, 0, false)
	require.NoError(t, err, "setupEventManager must not error")
	return m
}

func startTestEventManager(t *testing.T, m *eventManager) {
	t.Helper()
	require.NoError(t, m.Start(), "Start must not error")
}

func seedTicker(t *testing.T, exchangeName string, pair currency.Pair, a asset.Item, last, bid, ask float64) {
	t.Helper()
	err := ticker.ProcessTicker(&ticker.Price{
		ExchangeName: exchangeName,
		Pair:         pair,
		AssetType:    a,
		Last:         last,
		Bid:          bid,
		Ask:          ask,
	})
	require.NoError(t, err, "ProcessTicker must not error")
}

func seedOrderbook(t *testing.T, exchangeName string, pair currency.Pair, a asset.Item, bids, asks orderbook.Levels) {
	t.Helper()
	err := (&orderbook.Book{
		Exchange: exchangeName,
		Pair:     pair,
		Asset:    a,
		Bids:     bids,
		Asks:     asks,
	}).Process()
	require.NoError(t, err, "Orderbook Process must not error")
}

func newPriceEvent(exchangeName string, threshold float64) Event {
	return Event{
		Exchange: exchangeName,
		Item:     ItemPrice,
		Pair:     currency.NewBTCUSD(),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition:       ConditionGreaterThan,
			Price:           threshold,
			OrderbookAmount: 1337,
		},
	}
}

func TestSetupEventManager(t *testing.T) {
	t.Parallel()
	_, err := setupEventManager(nil, nil, 0, false)
	assert.ErrorIs(t, err, errNilComManager, "setupEventManager should return nil communication manager error")

	_, err = setupEventManager(&CommunicationManager{}, nil, 0, false)
	assert.ErrorIs(t, err, errNilExchangeManager, "setupEventManager should return nil exchange manager error")

	m, err := setupEventManager(&CommunicationManager{}, &ExchangeManager{}, 0, false)
	require.NoError(t, err, "setupEventManager must not error")

	require.NotNil(t, m, "event manager must not be nil")
	assert.NotZero(t, m.sleepDelay, "sleep delay should be set to default")
}

func TestEventManagerStart(t *testing.T) {
	m := setupTestEventManager(t, &ExchangeManager{})

	err := m.Start()
	assert.NoError(t, err, "Start should not error")

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted, "Start should return already started error")

	m = nil
	err = m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem, "Start should return nil subsystem error")
}

func TestEventManagerIsRunning(t *testing.T) {
	t.Parallel()
	m := setupTestEventManager(t, &ExchangeManager{})
	startTestEventManager(t, m)

	assert.True(t, m.IsRunning(), "IsRunning should return true when started")
	atomic.StoreInt32(&m.started, 0)
	assert.False(t, m.IsRunning(), "IsRunning should return false when stopped")
	m = nil
	assert.False(t, m.IsRunning(), "IsRunning should return false for nil manager")
}

func TestEventManagerStop(t *testing.T) {
	t.Parallel()
	m := setupTestEventManager(t, &ExchangeManager{})
	startTestEventManager(t, m)

	err := m.Stop()
	assert.NoError(t, err, "Stop should not error when started")

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted, "Stop should return not started error")

	m = nil
	err = m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem, "Stop should return nil subsystem error")
}

func TestEventManagerAdd(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	m := setupTestEventManager(t, em)
	pair := currency.NewPair(currency.BTC, currency.USDC)

	_, err := m.Add("", "", EventConditionParams{}, pair, asset.Spot, "")
	assert.ErrorIs(t, err, ErrSubSystemNotStarted, "Add should return not started error when manager is stopped")

	startTestEventManager(t, m)

	_, err = m.Add("", "", EventConditionParams{}, pair, asset.Spot, "")
	assert.ErrorIs(t, err, errExchangeDisabled, "Add should return exchange disabled error when exchange is missing")

	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err, "Add must not error when loading exchange")

	_, err = m.Add(testExchange, "", EventConditionParams{}, pair, asset.Spot, "")
	assert.ErrorIs(t, err, errInvalidItem, "Add should return invalid item error")

	cond := EventConditionParams{
		Condition:       ConditionGreaterThan,
		Price:           1337,
		OrderbookAmount: 1337,
	}
	_, err = m.Add(testExchange, ItemPrice, cond, pair, asset.Spot, "")
	assert.ErrorIs(t, err, errInvalidAction, "Add should return invalid action error")

	_, err = m.Add(testExchange, ItemPrice, cond, pair, asset.Spot, ActionTest)
	assert.NoError(t, err, "Add should not error for valid action")

	action := ActionSMSNotify + "," + ActionTest
	_, err = m.Add(testExchange, ItemPrice, cond, pair, asset.Spot, action)
	assert.NoError(t, err, "Add should not error for valid action list")
}

func TestEventManagerRemove(t *testing.T) {
	t.Parallel()
	m := setupTestEventManager(t, &ExchangeManager{})

	assert.False(t, m.Remove(0), "Remove should return false when manager is stopped")
	startTestEventManager(t, m)

	assert.False(t, m.Remove(0), "Remove should return false when event is not found")

	m.events = []Event{{ID: 42}}
	assert.True(t, m.Remove(42), "Remove should return true when event exists")
}

func TestGetEventCounter(t *testing.T) {
	t.Parallel()
	m := setupTestEventManager(t, &ExchangeManager{})

	total, executed := m.getEventCounter()
	assert.Zero(t, total, "getEventCounter should return zero total when manager is stopped")
	assert.Zero(t, executed, "getEventCounter should return zero executed when manager is stopped")
	startTestEventManager(t, m)

	total, executed = m.getEventCounter()
	assert.Zero(t, total, "getEventCounter should return zero total with no events")
	assert.Zero(t, executed, "getEventCounter should return zero executed with no events")

	m.events = []Event{
		{ID: 1, Executed: true},
		{ID: 2, Executed: false},
	}

	total, _ = m.getEventCounter()
	assert.Equal(t, 2, total, "getEventCounter should return the correct total count")

	_, executed = m.getEventCounter()
	assert.Equal(t, 1, executed, "getEventCounter should return the correct executed count")
}

func TestCheckEventCondition(t *testing.T) {
	m := setupTestEventManager(t, &ExchangeManager{})

	err := m.checkEventCondition(nil)
	assert.ErrorIs(t, err, ErrSubSystemNotStarted, "checkEventCondition should return not started error")

	startTestEventManager(t, m)

	err = m.checkEventCondition(nil)
	assert.ErrorIs(t, err, errNilEvent, "checkEventCondition should return nil event error")

	exchangeName := newUniqueFakeExchangeName()

	event := newPriceEvent(exchangeName, 1337)

	err = m.checkEventCondition(&event)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "checkEventCondition should return ticker not found error")

	seedTicker(t, exchangeName, currency.NewBTCUSD(), asset.Spot, 1500, 1499, 1501)

	err = m.checkEventCondition(&event)
	require.NoError(t, err, "checkEventCondition must not error")

	event.Item = ItemOrderbook
	event.Executed = false
	event.Condition.CheckAsks = true
	event.Condition.CheckBids = true

	err = m.checkEventCondition(&event)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookNotFound, "checkEventCondition should return orderbook not found error")

	seedOrderbook(t, exchangeName, currency.NewBTCUSD(), asset.Spot,
		orderbook.Levels{
			{Amount: 1, Price: 2000},
		},
		orderbook.Levels{
			{Amount: 1, Price: 2100},
		},
	)

	err = m.checkEventCondition(&event)
	assert.NoError(t, err, "checkEventCondition should not error")
}

func TestEventManagerAddNilManager(t *testing.T) {
	var m *eventManager
	_, err := m.Add("", ItemPrice, EventConditionParams{}, currency.NewBTCUSD(), asset.Spot, ActionTest)
	assert.ErrorIs(t, err, ErrNilSubsystem, "Add should return nil subsystem error")
}

func TestCheckEventConditionNilManager(t *testing.T) {
	var m *eventManager
	err := m.checkEventCondition(&Event{})
	assert.ErrorIs(t, err, ErrNilSubsystem, "checkEventCondition should return nil subsystem error")
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()
	assert.True(t, isValidCondition(ConditionGreaterThan), "isValidCondition should support greater than")
	assert.True(t, isValidCondition(ConditionGreaterThanOrEqual), "isValidCondition should support greater than or equal")
	assert.True(t, isValidCondition(ConditionLessThan), "isValidCondition should support less than")
	assert.True(t, isValidCondition(ConditionLessThanOrEqual), "isValidCondition should support less than or equal")
	assert.True(t, isValidCondition(ConditionIsEqual), "isValidCondition should support equality")
	assert.False(t, isValidCondition("!="), "isValidCondition should reject unsupported conditions")
}

func TestShouldProcessEvent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		condition string
		actual    float64
		threshold float64
		wantErr   bool
	}{
		{
			name:      "greater than true",
			condition: ConditionGreaterThan,
			actual:    2,
			threshold: 1,
			wantErr:   false,
		},
		{
			name:      "greater than or equal true",
			condition: ConditionGreaterThanOrEqual,
			actual:    2,
			threshold: 2,
			wantErr:   false,
		},
		{
			name:      "less than true",
			condition: ConditionLessThan,
			actual:    1,
			threshold: 2,
			wantErr:   false,
		},
		{
			name:      "less than or equal true",
			condition: ConditionLessThanOrEqual,
			actual:    2,
			threshold: 2,
			wantErr:   false,
		},
		{
			name:      "is equal true",
			condition: ConditionIsEqual,
			actual:    2,
			threshold: 2,
			wantErr:   false,
		},
		{
			name:      "unsupported condition",
			condition: "invalid",
			actual:    2,
			threshold: 2,
			wantErr:   true,
		},
		{
			name:      "condition not met",
			condition: ConditionGreaterThan,
			actual:    1,
			threshold: 2,
			wantErr:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := Event{
				Condition: EventConditionParams{Condition: tc.condition},
			}
			err := e.shouldProcessEvent(tc.actual, tc.threshold)
			if tc.wantErr {
				assert.Error(t, err, "shouldProcessEvent should return an error when conditions are not met")
			} else {
				assert.NoError(t, err, "shouldProcessEvent should not error when conditions are met")
			}
		})
	}
}

func TestIsValidEvent(t *testing.T) {
	t.Parallel()
	exchangeName := newUniqueFakeExchangeName()
	m := &eventManager{
		exchangeManager: &testExchangeManager{validExchange: exchangeName},
	}

	err := m.isValidEvent(exchangeName, ItemPrice, EventConditionParams{
		Condition: ConditionGreaterThan,
		Price:     100,
	}, ActionTest)
	assert.NoError(t, err, "isValidEvent should not error for valid price event")

	err = m.isValidEvent(exchangeName, ItemPrice, EventConditionParams{
		Condition: "bad",
		Price:     100,
	}, ActionTest)
	assert.ErrorIs(t, err, errInvalidCondition, "isValidEvent should return invalid condition error")

	err = m.isValidEvent(exchangeName, ItemPrice, EventConditionParams{
		Condition: ConditionGreaterThan,
		Price:     0,
	}, ActionTest)
	assert.ErrorIs(t, err, errInvalidCondition, "isValidEvent should return invalid condition error for zero price")

	err = m.isValidEvent(exchangeName, ItemOrderbook, EventConditionParams{
		Condition:       ConditionGreaterThan,
		OrderbookAmount: 0,
	}, ActionTest)
	assert.ErrorIs(t, err, errInvalidCondition, "isValidEvent should return invalid condition error for zero orderbook amount")

	err = m.isValidEvent(exchangeName, ItemPrice, EventConditionParams{
		Condition: ConditionGreaterThan,
		Price:     100,
	}, ActionConsolePrint+","+ActionTest)
	assert.ErrorIs(t, err, errInvalidAction, "isValidEvent should return invalid action error for invalid action list prefix")
}

func TestProcessTicker(t *testing.T) {
	t.Parallel()
	exchangeName := newUniqueFakeExchangeName()
	e := newPriceEvent(exchangeName, 10)

	seedTicker(t, exchangeName, currency.NewBTCUSD(), asset.Spot, 0, 0, 0)

	err := e.processTicker()
	assert.ErrorIs(t, err, errTickerLastPriceZero, "processTicker should return error when last price is zero")
}

func TestProcessOrderbookNoChecks(t *testing.T) {
	t.Parallel()
	exchangeName := newUniqueFakeExchangeName()
	e := Event{
		Exchange: exchangeName,
		Pair:     currency.NewBTCUSD(),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition:       ConditionGreaterThan,
			OrderbookAmount: 1,
		},
	}

	seedOrderbook(t, exchangeName, currency.NewBTCUSD(), asset.Spot,
		orderbook.Levels{
			{Amount: 1, Price: 2000},
		},
		orderbook.Levels{
			{Amount: 1, Price: 2100},
		},
	)

	err := e.processOrderbook()
	assert.NoError(t, err, "processOrderbook should not error when neither bids nor asks are checked")
}

func TestProcessOrderbookTruncation(t *testing.T) {
	t.Parallel()
	exchangeName := newUniqueFakeExchangeName()
	const manyLevels = 11

	bids := make(orderbook.Levels, 0, manyLevels)
	asks := make(orderbook.Levels, 0, manyLevels)
	for i := range manyLevels {
		bids = append(bids, orderbook.Level{Amount: 1, Price: float64(2000 - i)})
		asks = append(asks, orderbook.Level{Amount: 1, Price: float64(2100 + i)})
	}

	seedOrderbook(t, exchangeName, currency.NewBTCUSD(), asset.Spot, bids, asks)

	e := Event{
		Exchange: exchangeName,
		Pair:     currency.NewBTCUSD(),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition:       ConditionGreaterThan,
			OrderbookAmount: 1,
			CheckBids:       true,
			CheckAsks:       true,
		},
	}

	err := e.processOrderbook()
	assert.NoError(t, err, "processOrderbook should not error when matching levels exceed debug threshold")
}

func TestExecuteEventVerbose(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name            string
		threshold       float64
		wantExecuted    bool
		wantCommsEvents int
	}{
		{
			name:            "trigger",
			threshold:       1000,
			wantExecuted:    true,
			wantCommsEvents: 1,
		},
		{
			name:            "no trigger",
			threshold:       2000,
			wantExecuted:    false,
			wantCommsEvents: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exchangeName := newUniqueFakeExchangeName()
			seedTicker(t, exchangeName, currency.NewBTCUSD(), asset.Spot, 1500, 1499, 1501)

			comms := &testCommsManager{}
			m := &eventManager{
				verbose: true,
				comms:   comms,
			}
			atomic.StoreInt32(&m.started, 1)
			event := newPriceEvent(exchangeName, tc.threshold)
			event.ID = 1
			m.events = []Event{event}

			m.executeEvent(0)
			assert.Equal(t, tc.wantExecuted, m.events[0].Executed, "executeEvent executed state should match expected outcome")
			assert.Len(t, comms.events, tc.wantCommsEvents, "executeEvent communication event count should match expected outcome")
		})
	}
}

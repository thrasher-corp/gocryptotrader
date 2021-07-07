package engine

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestSetupEventManager(t *testing.T) {
	t.Parallel()
	_, err := setupEventManager(nil, nil, 0, false)
	if !errors.Is(err, errNilComManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilComManager)
	}

	_, err = setupEventManager(&CommunicationManager{}, nil, 0, false)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	m, err := setupEventManager(&CommunicationManager{}, &ExchangeManager{}, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
	if m.sleepDelay == 0 {
		t.Error("expected default set")
	}
}

func TestEventManagerStart(t *testing.T) {
	m, err := setupEventManager(&CommunicationManager{}, &ExchangeManager{}, 0, false)
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

	m = nil
	err = m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestEventManagerIsRunning(t *testing.T) {
	t.Parallel()
	m, err := setupEventManager(&CommunicationManager{}, &ExchangeManager{}, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
	atomic.StoreInt32(&m.started, 0)
	if m.IsRunning() {
		t.Error("expected false")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestEventManagerStop(t *testing.T) {
	t.Parallel()
	m, err := setupEventManager(&CommunicationManager{}, &ExchangeManager{}, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestEventManagerAdd(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	m, err := setupEventManager(&CommunicationManager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.Add("", "", EventConditionParams{}, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, "")
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.Add("", "", EventConditionParams{}, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, "")
	if !errors.Is(err, errExchangeDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errExchangeDisabled)
	}
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	_, err = m.Add(testExchange, "", EventConditionParams{}, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, "")
	if !errors.Is(err, errInvalidItem) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidItem)
	}

	cond := EventConditionParams{
		Condition:       ConditionGreaterThan,
		Price:           1337,
		OrderbookAmount: 1337,
	}
	_, err = m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, "")
	if !errors.Is(err, errInvalidAction) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidAction)
	}

	_, err = m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, ActionTest)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	action := ActionSMSNotify + "," + ActionTest
	_, err = m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, action)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestEventManagerRemove(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	m, err := setupEventManager(&CommunicationManager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m.Remove(0) {
		t.Error("expected false")
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m.Remove(0) {
		t.Error("expected false")
	}
	action := ActionSMSNotify + "," + ActionTest
	cond := EventConditionParams{
		Condition:       ConditionGreaterThan,
		Price:           1337,
		OrderbookAmount: 1337,
	}
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	id, err := m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, action)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	if !m.Remove(id) {
		t.Error("expected true")
	}
}

func TestGetEventCounter(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	m, err := setupEventManager(&CommunicationManager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	total, executed := m.getEventCounter()
	if total != 0 && executed != 0 {
		t.Error("expected 0")
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	total, executed = m.getEventCounter()
	if total != 0 && executed != 0 {
		t.Error("expected 0")
	}
	action := ActionSMSNotify + "," + ActionTest
	cond := EventConditionParams{
		Condition:       ConditionGreaterThan,
		Price:           1337,
		OrderbookAmount: 1337,
	}
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	_, err = m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, action)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	total, _ = m.getEventCounter()
	if total == 0 {
		t.Error("expected 1")
	}
}

func TestCheckEventCondition(t *testing.T) {
	em := SetupExchangeManager()
	m, err := setupEventManager(&CommunicationManager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(nil)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m.m.Unlock()
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(nil)
	if !errors.Is(err, errNilEvent) {
		t.Errorf("error '%v', expected '%v'", err, errNilEvent)
	}
	m.m.Unlock()

	action := ActionSMSNotify + "," + ActionTest
	cond := EventConditionParams{
		Condition:       ConditionGreaterThan,
		Price:           1337,
		OrderbookAmount: 1337,
	}
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	_, err = m.Add(testExchange, ItemPrice, cond, currency.NewPair(currency.BTC, currency.USD), asset.Spot, action)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(&m.events[0])
	if err != nil && !strings.Contains(err.Error(), "no tickers for") {
		t.Error(err)
	} else if err == nil {
		t.Error("expected error")
	}
	m.m.Unlock()
	_, err = exch.FetchTicker(currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(&m.events[0])
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Unlock()

	m.events[0].Item = ItemOrderbook
	m.events[0].Executed = false
	m.events[0].Condition.CheckAsks = true
	m.events[0].Condition.CheckBids = true
	m.m.Lock()
	err = m.checkEventCondition(&m.events[0])
	if err != nil && !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Error(err)
	} else if err == nil {
		t.Error("expected error")
	}
	m.m.Unlock()

	_, err = exch.FetchOrderbook(currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(&m.events[0])
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Unlock()
}

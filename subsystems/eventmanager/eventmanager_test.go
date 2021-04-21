package eventmanager

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/communicationmanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

const testExchange = "Bitstamp"

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, 0, false)
	if !errors.Is(err, errNilComManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilComManager)
	}

	_, err = Setup(&communicationmanager.Manager{}, nil, 0, false)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	m, err := Setup(&communicationmanager.Manager{}, &exchangemanager.Manager{}, 0, false)
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

func TestStart(t *testing.T) {
	m, err := Setup(&communicationmanager.Manager{}, &exchangemanager.Manager{}, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}

	m = nil
	err = m.Start()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	m, err := Setup(&communicationmanager.Manager{}, &exchangemanager.Manager{}, 0, false)
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

func TestStop(t *testing.T) {
	t.Parallel()
	m, err := Setup(&communicationmanager.Manager{}, &exchangemanager.Manager{}, 0, false)
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
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	em := exchangemanager.Setup()
	m, err := Setup(&communicationmanager.Manager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.Add("", "", EventConditionParams{}, currency.NewPair(currency.BTC, currency.USDC), asset.Spot, "")
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
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
		t.Error(err)
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

func TestRemove(t *testing.T) {
	t.Parallel()
	em := exchangemanager.Setup()
	m, err := Setup(&communicationmanager.Manager{}, em, 0, false)
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
		t.Error(err)
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
	em := exchangemanager.Setup()
	m, err := Setup(&communicationmanager.Manager{}, em, 0, false)
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
		t.Error(err)
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
	t.Parallel()
	em := exchangemanager.Setup()
	m, err := Setup(&communicationmanager.Manager{}, em, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.m.Lock()
	err = m.checkEventCondition(nil)
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
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
		t.Error(err)
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
	m.events[0].Condition.CheckBidsAndAsks = true
	m.m.Lock()
	err = m.checkEventCondition(&m.events[0])
	if err != nil && !strings.Contains(err.Error(), "no orderbooks for") {
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

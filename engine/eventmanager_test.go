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

/*
const (
	testExchange = "Bitstamp"
)

func addValidEvent() (int64, error) {
	return Add(testExchange,
		ItemPrice,
		EventConditionParams{Condition: ConditionGreaterThan, Price: 1},
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"SMS,test")
}

func TestAdd(t *testing.T) {
	bot := CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	_, err := Add("", "", EventConditionParams{}, currency.Pair{}, "", "")
	if err == nil {
		t.Error("should err on invalid params")
	}

	_, err = addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	_, err = addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	if len(Events) != 2 {
		t.Error("2 events should be stored")
	}
}

func TestRemove(t *testing.T) {
	bot := CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	id, err := addValidEvent()
	if err != nil {
		t.Error("unexpected result", err)
	}

	if s := Remove(id); !s {
		t.Error("unexpected result")
	}

	if s := Remove(id); s {
		t.Error("unexpected result")
	}
}


func TestExecuteAction(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	if Bot == nil {
		Bot = bot
	}
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}

	var e Event
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}

	e.Action = "SMS,test"
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}

	e.Action = "SMS,ALL"
	if r := e.ExecuteAction(); !r {
		t.Error("unexpected result")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	e := Event{
		Exchange: testExchange,
		Item:     ItemPrice,
		Condition: EventConditionParams{
			Condition: ConditionGreaterThan,
			Price:     1,
		},
		Pair:   currency.NewPair(currency.BTC, currency.USD),
		Asset:  asset.Spot,
		Action: "SMS,ALL",
	}

	if r := e.String(); r != "If the BTCUSD [SPOT] PRICE on Bitstamp meets the following {> 1 false false 0} then SMS,ALL." {
		t.Error("unexpected result")
	}
}

func TestProcessTicker(t *testing.T) {
	e := Event{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition: ConditionGreaterThan,
			Price:     1,
		},
	}

	// now populate it with a 0 entry
	tick := ticker.Price{
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		ExchangeName: e.Exchange,
		AssetType:    e.Asset,
	}
	if err := ticker.ProcessTicker(&tick); err != nil {
		t.Fatal("unexpected result:", err)
	}
	if r := e.processTicker(false); r {
		t.Error("unexpected result")
	}

	// now populate it with a number > 0
	tick.Last = 1337
	if err := ticker.ProcessTicker(&tick); err != nil {
		t.Fatal("unexpected result:", err)
	}
	if r := e.processTicker(false); !r {
		t.Error("unexpected result")
	}
}

func TestProcessCondition(t *testing.T) {
	t.Parallel()
	var e Event
	tester := []struct {
		Condition      string
		Actual         float64
		Threshold      float64
		ExpectedResult bool
	}{
		{ConditionGreaterThan, 1, 2, false},
		{ConditionGreaterThan, 2, 1, true},
		{ConditionGreaterThanOrEqual, 1, 2, false},
		{ConditionGreaterThanOrEqual, 2, 1, true},
		{ConditionIsEqual, 1, 1, true},
		{ConditionIsEqual, 1, 2, false},
		{ConditionLessThan, 1, 2, true},
		{ConditionLessThan, 2, 1, false},
		{ConditionLessThanOrEqual, 1, 2, true},
		{ConditionLessThanOrEqual, 2, 1, false},
	}
	for x := range tester {
		e.Condition.Condition = tester[x].Condition
		if r := e.processCondition(tester[x].Actual, tester[x].Threshold); r != tester[x].ExpectedResult {
			t.Error("unexpected result")
		}
	}
}

func TestProcessOrderbook(t *testing.T) {
	e := Event{
		Exchange: testExchange,
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asset:    asset.Spot,
		Condition: EventConditionParams{
			Condition:        ConditionGreaterThan,
			CheckBidsAndAsks: true,
			OrderbookAmount:  100,
		},
	}

	// now populate it with a 0 entry
	o := orderbook.Base{
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Bids:     []orderbook.Item{{Amount: 24, Price: 23}},
		Asks:     []orderbook.Item{{Amount: 24, Price: 23}},
		Exchange: e.Exchange,
		Asset:    e.Asset,
	}
	if err := o.Process(); err != nil {
		t.Fatal("unexpected result:", err)
	}

	if r := e.processOrderbook(false); !r {
		t.Error("unexpected result")
	}
}


func TestIsValidEvent(t *testing.T) {
	bot := CreateTestBot(t)
	if config.Cfg.Name == "" && bot != nil {
		config.Cfg = *bot.Config
	}
	// invalid exchange name
	if err := IsValidEvent("meow", "", EventConditionParams{}, ""); err != errExchangeDisabled {
		t.Error("unexpected result:", err)
	}

	// invalid item
	if err := IsValidEvent(testExchange, "", EventConditionParams{}, ""); err != errInvalidItem {
		t.Error("unexpected result:", err)
	}

	// invalid condition
	if err := IsValidEvent(testExchange, ItemPrice, EventConditionParams{}, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty price which will still throw an errInvalidCondition
	c := EventConditionParams{
		Condition: ConditionGreaterThan,
	}
	if err := IsValidEvent(testExchange, ItemPrice, c, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// valid condition but empty orderbook amount will still still throw an errInvalidCondition
	if err := IsValidEvent(testExchange, ItemOrderbook, c, ""); err != errInvalidCondition {
		t.Error("unexpected result:", err)
	}

	// test action splitting, but invalid
	c.OrderbookAmount = 1337
	if err := IsValidEvent(testExchange, ItemOrderbook, c, "a,meow"); err != errInvalidAction {
		t.Error("unexpected result:", err)
	}

	// check for invalid action without splitting
	if err := IsValidEvent(testExchange, ItemOrderbook, c, "hi"); err != errInvalidAction {
		t.Error("unexpected result:", err)
	}

	// valid event
	if err := IsValidEvent(testExchange, ItemOrderbook, c, "SMS,test"); err != nil {
		t.Error("unexpected result:", err)
	}
}

func TestIsValidExchange(t *testing.T) {
	t.Parallel()
	if s := IsValidExchange("invalidexchangerino"); s {
		t.Error("unexpected result")
	}
	CreateTestBot(t)
	if s := IsValidExchange(testExchange); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()
	if s := IsValidCondition("invalidconditionerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidCondition(ConditionGreaterThan); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidAction(t *testing.T) {
	t.Parallel()
	if s := IsValidAction("invalidactionerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidAction(ActionSMSNotify); !s {
		t.Error("unexpected result")
	}
}

func TestIsValidItem(t *testing.T) {
	t.Parallel()
	if s := IsValidItem("invaliditemerino"); s {
		t.Error("unexpected result")
	}
	if s := IsValidItem(ItemPrice); !s {
		t.Error("unexpected result")
	}
}


*/

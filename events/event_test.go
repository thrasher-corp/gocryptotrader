package events

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

var (
	loaded = false
)

func testSetup(t *testing.T) {
	if !loaded {
		cfg := config.GetConfig()
		err := cfg.LoadConfig("")
		if err != nil {
			t.Fatalf("Test failed. Failed to load config %s", err)
		}
		smsglobal.New(cfg.SMS.Username, cfg.SMS.Password, cfg.Name, cfg.SMS.Contacts)
		loaded = true
	}
}

func TestAddEvent(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	eventID, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. AddEvent: Error, %s", err)
	}
	eventID, err = AddEvent("ANXX", "price", ">,==", pair, "SPOT", actionTest)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Exchange")
	}
	eventID, err = AddEvent("ANX", "prices", ">,==", pair, "SPOT", actionTest)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Item")
	}
	eventID, err = AddEvent("ANX", "price", "3===D", pair, "SPOT", actionTest)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Condition")
	}
	eventID, err = AddEvent("ANX", "price", ">,==", pair, "SPOT", "console_prints")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Action")
	}

	if !RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestRemoveEvent(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	eventID, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. RemoveEvent: Error, %s", err)
	}
	if !RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
	if RemoveEvent(1234) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestGetEventCounter(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	one, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	two, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	three, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}

	Events[three-1].Executed = true

	total, _ := GetEventCounter()
	if total <= 0 {
		t.Errorf("Test Failed. GetEventCounter: Total = %d", total)
	}
	if !RemoveEvent(one) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}
	if !RemoveEvent(two) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}
	if !RemoveEvent(three) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}

	total2, _ := GetEventCounter()
	if total2 != 0 {
		t.Errorf("Test Failed. GetEventCounter: Total = %d", total2)
	}
}

func TestExecuteAction(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	one, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil {
		t.Fatalf("Test Failed. ExecuteAction: Error, %s", err)
	}
	isExecuted := Events[one].ExecuteAction()
	if !isExecuted {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
	if !RemoveEvent(one) {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}

	action := actionSMSNotify + "," + "ALL"
	one, err = AddEvent("ANX", "price", ">,==", pair, "SPOT", action)
	if err != nil {
		t.Fatalf("Test Failed. ExecuteAction: Error, %s", err)
	}

	isExecuted = Events[one].ExecuteAction()
	if !isExecuted {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
	if !RemoveEvent(one) {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}

	action = actionSMSNotify + "," + "StyleGherkin"
	one, err = AddEvent("ANX", "price", ">,==", pair, "SPOT", action)
	if err != nil {
		t.Fatalf("Test Failed. ExecuteAction: Error, %s", err)
	}

	isExecuted = Events[one].ExecuteAction()
	if !isExecuted {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
	if !RemoveEvent(one) {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
	// More tests when ExecuteAction is expanded
}

func TestEventToString(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	one, err := AddEvent("ANX", "price", ">,==", pair, "SPOT", actionTest)
	if err != nil {
		t.Errorf("Test Failed. EventToString: Error, %s", err)
	}

	eventString := Events[one].String()
	if eventString != "If the BTCUSD [SPOT] price on ANX is > == then ACTION_TEST." {
		t.Error("Test Failed. EventToString: Error, incorrect return string")
	}

	if !RemoveEvent(one) {
		t.Error("Test Failed. EventToString: Error, error removing event")
	}
}

func TestCheckCondition(t *testing.T) {
	testSetup(t)

	// Test invalid currency pair
	newPair := pair.NewCurrencyPair("A", "B")
	one, err := AddEvent("ANX", "price", ">=,10", newPair, "SPOT", actionTest)
	if err != nil {
		t.Errorf("Test Failed. CheckCondition: Error, %s", err)
	}
	conditionBool := Events[one].CheckCondition()
	if conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	// Test last price == 0
	var tickerNew ticker.Price
	tickerNew.Last = 0
	newPair = pair.NewCurrencyPair("BTC", "USD")
	ticker.ProcessTicker("ANX", newPair, tickerNew, ticker.Spot)
	Events[one].Pair = newPair
	conditionBool = Events[one].CheckCondition()
	if conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	// Test last pricce > 0 and conditional logic
	tickerNew.Last = 11
	ticker.ProcessTicker("ANX", newPair, tickerNew, ticker.Spot)
	Events[one].Condition = ">,10"
	conditionBool = Events[one].CheckCondition()
	if !conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	// Test last price >= 10
	Events[one].Condition = ">=,10"
	conditionBool = Events[one].CheckCondition()
	if !conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	// Test last price <= 10
	Events[one].Condition = "<,100"
	conditionBool = Events[one].CheckCondition()
	if !conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	// Test last price <= 10
	Events[one].Condition = "<=,100"
	conditionBool = Events[one].CheckCondition()
	if !conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	Events[one].Condition = "==,11"
	conditionBool = Events[one].CheckCondition()
	if !conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	Events[one].Condition = "^,11"
	conditionBool = Events[one].CheckCondition()
	if conditionBool {
		t.Error("Test Failed. CheckCondition: Error, wrong conditional.")
	}

	if !RemoveEvent(one) {
		t.Error("Test Failed. CheckCondition: Error, error removing event")
	}
}

func TestIsValidEvent(t *testing.T) {
	testSetup(t)

	err := IsValidEvent("ANX", "price", ">,==", actionTest)
	if err != nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}
	err = IsValidEvent("ANX", "price", ">,", actionTest)
	if err == nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}
	err = IsValidEvent("ANX", "Testy", ">,==", actionTest)
	if err == nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}
	err = IsValidEvent("Testys", "price", ">,==", actionTest)
	if err == nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}

	action := "blah,blah"
	err = IsValidEvent("ANX", "price", ">=,10", action)
	if err == nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}

	action = "SMS,blah"
	err = IsValidEvent("ANX", "price", ">=,10", action)
	if err == nil {
		t.Errorf("Test Failed. IsValidEvent: %s", err)
	}

	//Function tests need to appended to this function when more actions are
	//implemented
}

func TestCheckEvents(t *testing.T) {
	testSetup(t)

	pair := pair.NewCurrencyPair("BTC", "USD")
	_, err := AddEvent("ANX", "price", ">=,10", pair, "SPOT", actionTest)
	if err != nil {
		t.Fatal("Test failed. TestChcheckEvents add event")
	}

	go CheckEvents()
}

func TestIsValidExchange(t *testing.T) {
	testSetup(t)

	boolean := IsValidExchange("ANX")
	if !boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect Exchange")
	}
	boolean = IsValidExchange("OBTUSE")
	if boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect return")
	}
}

func TestIsValidCondition(t *testing.T) {
	testSetup(t)

	boolean := IsValidCondition(">")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = IsValidCondition(">=")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = IsValidCondition("<")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = IsValidCondition("<=")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = IsValidCondition("==")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = IsValidCondition("**********")
	if boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect return")
	}
}

func TestIsValidAction(t *testing.T) {
	testSetup(t)

	boolean := IsValidAction("sms")
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = IsValidAction(actionTest)
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = IsValidAction("randomstring")
	if boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect return")
	}
}

func TestIsValidItem(t *testing.T) {
	testSetup(t)

	boolean := IsValidItem("price")
	if !boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect Item")
	}
	boolean = IsValidItem("obtuse")
	if boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect return")
	}
}

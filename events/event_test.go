package events

import (
	"testing"
)

func TestAddEvent(t *testing.T) {
	eventID, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. AddEvent: Error, %s", err)
	}
	eventID, err = AddEvent("ANXX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Exchange")
	}
	eventID, err = AddEvent("ANX", "prices", ">,==", "BTC", "LTC", ACTION_TEST)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Item")
	}
	eventID, err = AddEvent("ANX", "price", "3===D", "BTC", "LTC", ACTION_TEST)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Condition")
	}
	eventID, err = AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_prints")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Action")
	}
	eventID, err = AddEvent("ANX", "price", ">,==", "BATMAN", "ROBIN", ACTION_TEST)
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Action")
	}
	if !RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestRemoveEvent(t *testing.T) {
	eventID, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. RemoveEvent: Error, %s", err)
	}
	if !RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestGetEventCounter(t *testing.T) {
	one, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	two, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	three, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}

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
	t.Parallel()

	one, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. ExecuteAction: Error, %s", err)
	}
	isExecuted := Events[one].ExecuteAction()
	if !isExecuted {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}

	if !RemoveEvent(one) {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
}

func TestEventToString(t *testing.T) {
	t.Parallel()

	one, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. EventToString: Error, %s", err)
	}

	eventString := Events[one].EventToString()
	if eventString != "If the BTCLTC price on ANX is > == then ACTION_TEST." {
		t.Error("Test Failed. EventToString: Error, incorrect return string")
	}

	if !RemoveEvent(one) {
		t.Error("Test Failed. EventToString: Error, error removing event")
	}

}

func TestCheckCondition(t *testing.T) { //error handling needs to be implemented
	t.Parallel()

	one, err := AddEvent("ANX", "price", ">,==", "BTC", "LTC", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. EventToString: Error, %s", err)
	}

	conditionBool := Events[one].CheckCondition()
	if conditionBool { //check once error handling is implemented
		t.Error("Test Failed. EventToString: Error, wrong conditional.")
	}

	if !RemoveEvent(one) {
		t.Error("Test Failed. EventToString: Error, error removing event")
	}

}

func TestIsValidEvent(t *testing.T) {
	err := IsValidEvent("ANX", "price", ">,==", ACTION_TEST)
	if err != nil {
		t.Errorf("Test Failed. IsValidExchange: Error %s", err)
	}
}

func TestCheckEvents(t *testing.T) { //Add error handling
	//CheckEvents() //check once error handling is implemented
}

func TestIsValidExchange(t *testing.T) {
	boolean := IsValidExchange("ANX", CONFIG_PATH_TEST)
	if !boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect Exchange")
	}
	boolean = IsValidExchange("OBTUSE", CONFIG_PATH_TEST)
	if boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect return")
	}
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	boolean := IsValidAction("sms")
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = IsValidAction(ACTION_TEST)
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = IsValidAction("randomstring")
	if boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect return")
	}
}

func TestIsValidItem(t *testing.T) {
	t.Parallel()

	boolean := IsValidItem("price")
	if !boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect Item")
	}
	boolean = IsValidItem("obtuse")
	if boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect return")
	}
}

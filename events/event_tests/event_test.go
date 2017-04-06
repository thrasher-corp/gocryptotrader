package tests

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/events"
)

func TestAddEvent(t *testing.T) {
	eventID, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. AddEvent: Error, %s", err)
	}
	eventID, err = events.AddEvent("ANXX", "price", ">,==", "BTC", "LTC", "console_print")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Exchange")
	}
	eventID, err = events.AddEvent("ANX", "prices", ">,==", "BTC", "LTC", "console_print")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Item")
	}
	eventID, err = events.AddEvent("ANX", "price", "3===D", "BTC", "LTC", "console_print")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Condition")
	}
	eventID, err = events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_prints")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Action")
	}
	eventID, err = events.AddEvent("ANX", "price", ">,==", "BATMAN", "ROBIN", "console_print")
	if err == nil && eventID == 0 {
		t.Error("Test Failed. AddEvent: Error, error not captured in Action")
	}
	if !events.RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestRemoveEvent(t *testing.T) {
	eventID, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil && eventID != 0 {
		t.Errorf("Test Failed. RemoveEvent: Error, %s", err)
	}
	if !events.RemoveEvent(eventID) {
		t.Error("Test Failed. RemoveEvent: Error, error removing event")
	}
}

func TestGetEventCounter(t *testing.T) {
	one, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	two, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}
	three, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. GetEventCounter: Error, %s", err)
	}

	total, _ := events.GetEventCounter()
	if total <= 0 {
		t.Errorf("Test Failed. GetEventCounter: Total = %d", total)
	}

	if !events.RemoveEvent(one) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}
	if !events.RemoveEvent(two) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}
	if !events.RemoveEvent(three) {
		t.Error("Test Failed. GetEventCounter: Error, error removing event")
	}

	total2, _ := events.GetEventCounter()
	t.Log(total2)
	if total2 != 0 {
		t.Errorf("Test Failed. GetEventCounter: Total = %d", total2)
	}
}

func TestExecuteAction(t *testing.T) {
	t.Parallel()

	one, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. ExecuteAction: Error, %s", err)
	}
	isExecuted := events.Events[one].ExecuteAction()
	if !isExecuted {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}

	if !events.RemoveEvent(one) {
		t.Error("Test Failed. ExecuteAction: Error, error removing event")
	}
}

func TestEventToString(t *testing.T) {
	t.Parallel()

	one, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. EventToString: Error, %s", err)
	}

	eventString := events.Events[one].EventToString()
	if eventString != "If the BTCLTC price on ANX is > == then console_print." {
		t.Error("Test Failed. EventToString: Error, incorrect return string")
	}

	if !events.RemoveEvent(one) {
		t.Error("Test Failed. EventToString: Error, error removing event")
	}

}

func TestCheckCondition(t *testing.T) { //error handling needs to be implemented
	t.Parallel()

	one, err := events.AddEvent("ANX", "price", ">,==", "BTC", "LTC", "console_print")
	if err != nil {
		t.Errorf("Test Failed. EventToString: Error, %s", err)
	}

	conditionBool := events.Events[one].CheckCondition()
	if conditionBool { //check once error handling is implemented
		t.Error("Test Failed. EventToString: Error, wrong conditional.")
	}

	if !events.RemoveEvent(one) {
		t.Error("Test Failed. EventToString: Error, error removing event")
	}

}

func TestIsValidEvent(t *testing.T) {
	err := events.IsValidEvent("ANX", "price", ">,==", "console_print")
	if err != nil {
		t.Errorf("Test Failed. IsValidExchange: Error %s", err)
	}
}

func TestCheckEvents(t *testing.T) { //Add error handling
	//events.CheckEvents() //check once error handling is implemented
}

func TestIsValidExchange(t *testing.T) {
	boolean := events.IsValidExchange("ANX")
	if !boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect Exchange")
	}
	boolean = events.IsValidExchange("OBTUSE")
	if boolean {
		t.Error("Test Failed. IsValidExchange: Error, incorrect return")
	}
}

func TestIsValidCondition(t *testing.T) {
	t.Parallel()

	boolean := events.IsValidCondition(">")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = events.IsValidCondition(">=")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = events.IsValidCondition("<")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = events.IsValidCondition("<=")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = events.IsValidCondition("==")
	if !boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect Condition")
	}
	boolean = events.IsValidCondition("**********")
	if boolean {
		t.Error("Test Failed. IsValidCondition: Error, incorrect return")
	}
}

func TestIsValidAction(t *testing.T) {
	t.Parallel()

	boolean := events.IsValidAction("sms")
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = events.IsValidAction("console_print")
	if !boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect Action")
	}
	boolean = events.IsValidAction("randomstring")
	if boolean {
		t.Error("Test Failed. IsValidAction: Error, incorrect return")
	}
}

func TestIsValidItem(t *testing.T) {
	t.Parallel()

	boolean := events.IsValidItem("price")
	if !boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect Item")
	}
	boolean = events.IsValidItem("obtuse")
	if boolean {
		t.Error("Test Failed. IsValidItem: Error, incorrect return")
	}
}

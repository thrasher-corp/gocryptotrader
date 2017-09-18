package events

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

const (
	itemPrice          = "PRICE"
	greaterThan        = ">"
	greaterThanOrEqual = ">="
	lessThan           = "<"
	lessThanOrEqual    = "<="
	isEqual            = "=="
	actionSMSNotify    = "SMS"
	actionConsolePrint = "CONSOLE_PRINT"
	actionTest         = "ACTION_TEST"
)

var (
	errInvalidItem      = errors.New("invalid item")
	errInvalidCondition = errors.New("invalid conditional option")
	errInvalidAction    = errors.New("invalid action")
	errExchangeDisabled = errors.New("desired exchange is disabled")
)

// Event struct holds the event variables
type Event struct {
	ID        int
	Exchange  string
	Item      string
	Condition string
	Pair      pair.CurrencyPair
	Asset     string
	Action    string
	Executed  bool
}

// Events variable is a pointer array to the event structures that will be
// appended
var Events []*Event

// AddEvent adds an event to the Events chain and returns an index/eventID
// and an error
func AddEvent(Exchange, Item, Condition string, CurrencyPair pair.CurrencyPair, Asset, Action string) (int, error) {
	err := IsValidEvent(Exchange, Item, Condition, Action)
	if err != nil {
		return 0, err
	}

	Event := &Event{}

	if len(Events) == 0 {
		Event.ID = 0
	} else {
		Event.ID = len(Events) + 1
	}

	Event.Exchange = Exchange
	Event.Item = Item
	Event.Condition = Condition
	Event.Pair = CurrencyPair
	Event.Asset = Asset
	Event.Action = Action
	Event.Executed = false
	Events = append(Events, Event)
	return Event.ID, nil
}

// RemoveEvent deletes and event by its ID
func RemoveEvent(EventID int) bool {
	for i, x := range Events {
		if x.ID == EventID {
			Events = append(Events[:i], Events[i+1:]...)
			return true
		}
	}
	return false
}

// GetEventCounter displays the emount of total events on the chain and the
// events that have been executed.
func GetEventCounter() (int, int) {
	total := len(Events)
	executed := 0

	for _, x := range Events {
		if x.Executed {
			executed++
		}
	}
	return total, executed
}

// ExecuteAction will execute the action pending on the chain
func (e *Event) ExecuteAction() bool {
	if common.StringContains(e.Action, ",") {
		action := common.SplitStrings(e.Action, ",")
		if action[0] == actionSMSNotify {
			message := fmt.Sprintf("Event triggered: %s", e.String())
			s := smsglobal.SMSGlobal
			if action[1] == "ALL" {
				s.SendMessageToAll(message)
			} else {
				contact, _ := s.GetContactByName(action[1])
				s.SendMessage(contact.Number, message)
			}
		}
	} else {
		log.Printf("Event triggered: %s", e.String())
	}
	return true
}

// EventToString turns the structure event into a string
func (e *Event) String() string {
	condition := common.SplitStrings(e.Condition, ",")
	return fmt.Sprintf(
		"If the %s%s [%s] %s on %s is %s then %s.", e.Pair.FirstCurrency.String(),
		e.Pair.SecondCurrency.String(), e.Asset, e.Item, e.Exchange, condition[0]+" "+condition[1], e.Action,
	)
}

// CheckCondition will check the event structure to see if there is a condition
// met
func (e *Event) CheckCondition() bool {
	condition := common.SplitStrings(e.Condition, ",")
	targetPrice, _ := strconv.ParseFloat(condition[1], 64)

	t, err := ticker.GetTicker(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		return false
	}

	lastPrice := t.Last

	if lastPrice == 0 {
		return false
	}

	switch condition[0] {
	case greaterThan:
		{
			if lastPrice > targetPrice {
				return e.ExecuteAction()
			}
		}
	case greaterThanOrEqual:
		{
			if lastPrice >= targetPrice {
				return e.ExecuteAction()
			}
		}
	case lessThan:
		{
			if lastPrice < targetPrice {
				return e.ExecuteAction()
			}
		}
	case lessThanOrEqual:
		{
			if lastPrice <= targetPrice {
				return e.ExecuteAction()
			}
		}
	case isEqual:
		{
			if lastPrice == targetPrice {
				return e.ExecuteAction()
			}
		}
	}
	return false
}

// IsValidEvent checks the actions to be taken and returns an error if incorrect
func IsValidEvent(Exchange, Item, Condition, Action string) error {
	Exchange = common.StringToUpper(Exchange)
	Item = common.StringToUpper(Item)
	Action = common.StringToUpper(Action)

	if !IsValidExchange(Exchange) {
		return errExchangeDisabled
	}

	if !IsValidItem(Item) {
		return errInvalidItem
	}

	if !common.StringContains(Condition, ",") {
		return errInvalidCondition
	}

	condition := common.SplitStrings(Condition, ",")

	if !IsValidCondition(condition[0]) || len(condition[1]) == 0 {
		return errInvalidCondition
	}

	if common.StringContains(Action, ",") {
		action := common.SplitStrings(Action, ",")

		if action[0] != actionSMSNotify {
			return errInvalidAction
		}

		if action[1] != "ALL" {
			s := smsglobal.SMSGlobal
			_, err := s.GetContactByName(action[1])
			if err != nil {
				return errInvalidAction
			}
		}
	} else {
		if Action != actionConsolePrint && Action != actionTest {
			return errInvalidAction
		}
	}
	return nil
}

// CheckEvents is the overarching routine that will iterate through the Events
// chain
func CheckEvents() {
	for {
		total, executed := GetEventCounter()
		if total > 0 && executed != total {
			for _, event := range Events {
				if !event.Executed {
					success := event.CheckCondition()
					if success {
						log.Printf(
							"Event %d triggered on %s successfully.\n", event.ID,
							event.Exchange,
						)
						event.Executed = true
					}
				}
			}
		}
	}
}

// IsValidExchange validates the exchange
func IsValidExchange(Exchange string) bool {
	Exchange = common.StringToUpper(Exchange)
	cfg := config.GetConfig()
	for _, x := range cfg.Exchanges {
		if x.Name == Exchange && x.Enabled {
			return true
		}
	}
	return false
}

// IsValidCondition validates passed in condition
func IsValidCondition(Condition string) bool {
	switch Condition {
	case greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual, isEqual:
		return true
	}
	return false
}

// IsValidAction validates passed in action
func IsValidAction(Action string) bool {
	Action = common.StringToUpper(Action)
	switch Action {
	case actionSMSNotify, actionConsolePrint, actionTest:
		return true
	}
	return false
}

// IsValidItem validates passed in Item
func IsValidItem(Item string) bool {
	Item = common.StringToUpper(Item)
	switch Item {
	case itemPrice:
		return true
	}
	return false
}

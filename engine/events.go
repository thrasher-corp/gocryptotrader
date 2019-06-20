package engine

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/asset"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// TO-DO MAKE THIS A SERVICE SUBSYSTEM

// Event const vars
const (
	ItemPrice     = "PRICE"
	ItemOrderbook = "ORDERBOOK"

	ConditionGreaterThan        = ">"
	ConditionGreaterThanOrEqual = ">="
	ConditionLessThan           = "<"
	ConditionLessThanOrEqual    = "<="
	ConditionIsEqual            = "=="

	ActionSMSNotify    = "SMS"
	ActionConsolePrint = "CONSOLE_PRINT"
	ActionTest         = "ACTION_TEST"

	defaultSleepDelay = time.Millisecond * 500
)

// vars related to events package
var (
	errInvalidItem      = errors.New("invalid item")
	errInvalidCondition = errors.New("invalid conditional option")
	errInvalidAction    = errors.New("invalid action")
	errExchangeDisabled = errors.New("desired exchange is disabled")
	EventSleepDelay     = defaultSleepDelay
)

// EventConditionParams holds the event condition variables
type EventConditionParams struct {
	Condition string
	Price     float64

	CheckBids        bool
	CheckBidsAndAsks bool
	OrderbookAmount  float64
}

// Event struct holds the event variables
type Event struct {
	ID        int64
	Exchange  string
	Item      string
	Condition EventConditionParams
	Pair      currency.Pair
	Asset     asset.Item
	Action    string
	Executed  bool
}

// Events variable is a pointer array to the event structures that will be
// appended
var Events []*Event

// Add adds an event to the Events chain and returns an index/eventID
// and an error
func Add(exchange, item string, condition EventConditionParams, currencyPair currency.Pair, asset asset.Item, action string) (int64, error) {
	err := IsValidEvent(exchange, item, condition, action)
	if err != nil {
		return 0, err
	}

	evt := &Event{}

	if len(Events) == 0 {
		evt.ID = 0
	} else {
		evt.ID = int64(len(Events) + 1)
	}

	evt.Exchange = exchange
	evt.Item = item
	evt.Condition = condition
	evt.Pair = currencyPair
	evt.Asset = asset
	evt.Action = action
	evt.Executed = false
	Events = append(Events, evt)
	return evt.ID, nil
}

// Remove deletes and event by its ID
func Remove(eventID int64) bool {
	for i, x := range Events {
		if x.ID == eventID {
			Events = append(Events[:i], Events[i+1:]...)
			return true
		}
	}
	return false
}

// GetEventCounter displays the emount of total events on the chain and the
// events that have been executed.
func GetEventCounter() (total, executed int) {
	total = len(Events)

	for _, x := range Events {
		if x.Executed {
			executed++
		}
	}
	return total, executed
}

// ExecuteAction will execute the action pending on the chain
func (e *Event) ExecuteAction() bool {
	if strings.Contains(e.Action, ",") {
		action := strings.Split(e.Action, ",")
		if action[0] == ActionSMSNotify {
			message := fmt.Sprintf("Event triggered: %s", e.String())
			if action[1] == "ALL" {
				Bot.CommsManager.PushEvent(base.Event{
					Type:    "event",
					Message: message,
				})
			}
		}
	} else {
		log.Debugf("Event triggered: %s", e.String())
	}
	return true
}

// String turns the structure event into a string
func (e *Event) String() string {
	return fmt.Sprintf(
		"If the %s [%s] %s on %s meets the following %v then %s.", e.Pair.String(),
		strings.ToUpper(e.Asset.String()), e.Item, e.Exchange, e.Condition, e.Action,
	)
}

func (e *Event) processTicker() bool {
	t, err := ticker.GetTicker(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		if Bot.Settings.Verbose {
			log.Debugf("Events: failed to get ticker. Err: %s", err)
		}
		return false
	}

	if t.Last == 0 {
		if Bot.Settings.Verbose {
			log.Debugln("Events: ticker last price is 0")
		}
		return false
	}
	return e.processCondition(t.Last, e.Condition.Price)
}

func (e *Event) processCondition(actual, threshold float64) bool {
	switch e.Condition.Condition {
	case ConditionGreaterThan:
		if actual > threshold {
			return e.ExecuteAction()
		}
	case ConditionGreaterThanOrEqual:
		if actual >= threshold {
			return e.ExecuteAction()
		}
	case ConditionLessThan:
		if actual < threshold {
			return e.ExecuteAction()
		}
	case ConditionLessThanOrEqual:
		if actual <= threshold {
			return e.ExecuteAction()
		}
	case ConditionIsEqual:
		if actual == threshold {
			return e.ExecuteAction()
		}
	}
	return false
}

func (e *Event) processOrderbook() bool {
	ob, err := orderbook.Get(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		if Bot.Settings.Verbose {
			log.Debugf("Events: Failed to get orderbook. Err: %s", err)
		}
		return false
	}

	success := false
	if e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Bids {
			subtotal := ob.Bids[x].Amount * ob.Bids[x].Price
			result := e.processCondition(subtotal, e.Condition.OrderbookAmount)
			if result {
				success = true
				log.Debugf("Events: Bid Amount: %f Price: %v Subtotal: %v", ob.Bids[x].Amount, ob.Bids[x].Price, subtotal)
			}
		}
	}

	if !e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Asks {
			subtotal := ob.Asks[x].Amount * ob.Asks[x].Price
			result := e.processCondition(subtotal, e.Condition.OrderbookAmount)
			if result {
				success = true
				log.Debugf("Events: Ask Amount: %f Price: %v Subtotal: %v", ob.Asks[x].Amount, ob.Asks[x].Price, subtotal)
			}
		}
	}
	return success
}

// CheckEventCondition will check the event structure to see if there is a condition
// met
func (e *Event) CheckEventCondition() bool {
	if e.Item == ItemPrice {
		return e.processTicker()
	}
	return e.processOrderbook()
}

// IsValidEvent checks the actions to be taken and returns an error if incorrect
func IsValidEvent(exchange, item string, condition EventConditionParams, action string) error {
	exchange = strings.ToUpper(exchange)
	item = strings.ToUpper(item)
	action = strings.ToUpper(action)

	if !IsValidExchange(exchange) {
		return errExchangeDisabled
	}

	if !IsValidItem(item) {
		return errInvalidItem
	}

	if !IsValidCondition(condition.Condition) {
		return errInvalidCondition
	}

	if item == ItemPrice {
		if condition.Price <= 0 {
			return errInvalidCondition
		}
	}

	if item == ItemOrderbook {
		if condition.OrderbookAmount <= 0 {
			return errInvalidCondition
		}
	}

	if strings.Contains(action, ",") {
		a := strings.Split(action, ",")

		if a[0] != ActionSMSNotify {
			return errInvalidAction
		}
	} else if action != ActionConsolePrint && action != ActionTest {
		return errInvalidAction
	}

	return nil
}

// EventManger is the overarching routine that will iterate through the Events
// chain
func EventManger() {
	log.Debugf("EventManager started. SleepDelay: %v", EventSleepDelay.String())

	for {
		total, executed := GetEventCounter()
		if total > 0 && executed != total {
			for _, event := range Events {
				if !event.Executed {
					if Bot.Settings.Verbose {
						log.Debugf("Events: Processing event %s.", event.String())
					}
					success := event.CheckEventCondition()
					if success {
						msg := fmt.Sprintf(
							"Events: ID: %d triggered on %s successfully [%v]\n", event.ID,
							event.Exchange, event.String(),
						)
						log.Info(msg)
						Bot.CommsManager.PushEvent(base.Event{Type: "event", Message: msg})
						event.Executed = true
					}
				}
			}
		}
		time.Sleep(EventSleepDelay)
	}
}

// IsValidExchange validates the exchange
func IsValidExchange(exchangeName string) bool {
	exchangeName = strings.ToLower(exchangeName)
	cfg := config.GetConfig()
	for x := range cfg.Exchanges {
		if strings.EqualFold(cfg.Exchanges[x].Name, exchangeName) && cfg.Exchanges[x].Enabled {
			return true
		}
	}
	return false
}

// IsValidCondition validates passed in condition
func IsValidCondition(condition string) bool {
	switch condition {
	case ConditionGreaterThan, ConditionGreaterThanOrEqual, ConditionLessThan, ConditionLessThanOrEqual, ConditionIsEqual:
		return true
	}
	return false
}

// IsValidAction validates passed in action
func IsValidAction(action string) bool {
	action = strings.ToUpper(action)
	switch action {
	case ActionSMSNotify, ActionConsolePrint, ActionTest:
		return true
	}
	return false
}

// IsValidItem validates passed in Item
func IsValidItem(item string) bool {
	item = strings.ToUpper(item)
	switch item {
	case ItemPrice, ItemOrderbook:
		return true
	}
	return false
}

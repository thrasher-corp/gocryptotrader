package eventmanager

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/subsystems"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup loads and validates the communications manager config
func Setup(comManager iCommsManager, verbose bool) *Manager {
	if comManager == nil {
		return nil
	}
	return &Manager{
		comms:   comManager,
		verbose: verbose,
	}
}

// Start is the overarching routine that will iterate through the Events
// chain
func (m *Manager) Start() error {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("event manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.EventMgr, "Event Manager started. SleepDelay: %v\n", EventSleepDelay.String())
	for {
		total, executed := m.GetEventCounter()
		if total > 0 && executed != total {
			for i := range m.events {
				if !m.events[i].Executed {
					if m.verbose {
						log.Debugf(log.EventMgr, "Events: Processing event %s.\n", m.events[i].String())
					}
					success := m.CheckEventCondition(&m.events[i])
					if success {
						msg := fmt.Sprintf(
							"Events: ID: %d triggered on %s successfully [%v]\n", m.events[i].ID,
							m.events[i].Exchange, m.events[i].String(),
						)
						log.Infoln(log.EventMgr, msg)
						m.comms.PushEvent(base.Event{Type: "event", Message: msg})
						m.events[i].Executed = true
					}
				}
			}
		}
		time.Sleep(EventSleepDelay)
	}
}

// Add adds an event to the Events chain and returns an index/eventID
// and an error
func (m *Manager) Add(exchange, item string, condition EventConditionParams, p currency.Pair, a asset.Item, action string) (int64, error) {
	err := isValidEvent(exchange, item, condition, action)
	if err != nil {
		return 0, err
	}
	evt := Event{
		Exchange:  exchange,
		Item:      item,
		Condition: condition,
		Pair:      p,
		Asset:     a,
		Action:    action,
		Executed:  false,
	}
	if len(m.events) > 0 {
		evt.ID = int64(len(m.events) + 1)
	}
	m.events = append(m.events, evt)
	return evt.ID, nil
}

// Remove deletes and event by its ID
func (m *Manager) Remove(eventID int64) bool {
	for i := range m.events {
		if m.events[i].ID == eventID {
			m.events = append(m.events[:i], m.events[i+1:]...)
			return true
		}
	}
	return false
}

// GetEventCounter displays the emount of total events on the chain and the
// events that have been executed.
func (m *Manager) GetEventCounter() (total, executed int) {
	total = len(m.events)
	for i := range m.events {
		if m.events[i].Executed {
			executed++
		}
	}
	return total, executed
}

// ExecuteAction will execute the action pending on the chain
func (m *Manager) ExecuteAction(e *Event) bool {
	if strings.Contains(e.Action, ",") {
		action := strings.Split(e.Action, ",")
		if action[0] == ActionSMSNotify {
			if action[1] == "ALL" {
				m.comms.PushEvent(base.Event{
					Type:    "event",
					Message: "Event triggered: " + e.String(),
				})
			}
		}
	} else {
		log.Debugf(log.EventMgr, "Event triggered: %s\n", e.String())
	}
	return true
}

// CheckEventCondition will check the event structure to see if there is a condition
// met
func (m *Manager) CheckEventCondition(e *Event) bool {
	if e.Item == ItemPrice {
		return e.processTicker(m.verbose)
	}
	return e.processOrderbook(m.verbose)
}

// String turns the structure event into a string
func (e *Event) String() string {
	return fmt.Sprintf(
		"If the %s [%s] %s on %s meets the following %v then %s.", e.Pair.String(),
		strings.ToUpper(e.Asset.String()), e.Item, e.Exchange, e.Condition, e.Action,
	)
}

func (e *Event) processTicker(verbose bool) bool {
	t, err := ticker.GetTicker(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		if verbose {
			log.Debugf(log.EventMgr, "Events: failed to get ticker. Err: %s\n", err)
		}
		return false
	}

	if t.Last == 0 {
		if verbose {
			log.Debugln(log.EventMgr, "Events: ticker last price is 0")
		}
		return false
	}
	return e.shouldProcessEvent(t.Last, e.Condition.Price)
}

func (e *Event) shouldProcessEvent(actual, threshold float64) bool {
	switch e.Condition.Condition {
	case ConditionGreaterThan:
		if actual > threshold {
			return true
		}
	case ConditionGreaterThanOrEqual:
		if actual >= threshold {
			return true
		}
	case ConditionLessThan:
		if actual < threshold {
			return true
		}
	case ConditionLessThanOrEqual:
		if actual <= threshold {
			return true
		}
	case ConditionIsEqual:
		if actual == threshold {
			return true
		}
	}
	return false
}

func (e *Event) processOrderbook(verbose bool) bool {
	ob, err := orderbook.Get(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		if verbose {
			log.Debugf(log.EventMgr, "Events: Failed to get orderbook. Err: %s\n", err)
		}
		return false
	}

	success := false
	if e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Bids {
			subtotal := ob.Bids[x].Amount * ob.Bids[x].Price
			result := e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if result {
				success = true
				log.Debugf(log.EventMgr, "Events: Bid Amount: %f Price: %v Subtotal: %v\n", ob.Bids[x].Amount, ob.Bids[x].Price, subtotal)
			}
		}
	}

	if !e.Condition.CheckBids || e.Condition.CheckBidsAndAsks {
		for x := range ob.Asks {
			subtotal := ob.Asks[x].Amount * ob.Asks[x].Price
			result := e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if result {
				success = true
				log.Debugf(log.EventMgr, "Events: Ask Amount: %f Price: %v Subtotal: %v\n", ob.Asks[x].Amount, ob.Asks[x].Price, subtotal)
			}
		}
	}
	return success
}

// isValidEvent checks the actions to be taken and returns an error if incorrect
func isValidEvent(exchange, item string, condition EventConditionParams, action string) error {
	exchange = strings.ToUpper(exchange)
	item = strings.ToUpper(item)
	action = strings.ToUpper(action)

	if !isValidExchange(exchange) {
		return errExchangeDisabled
	}

	if !isValidItem(item) {
		return errInvalidItem
	}

	if !isValidCondition(condition.Condition) {
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

// isValidExchange validates the exchange
func isValidExchange(exchangeName string) bool {
	cfg := config.GetConfig()
	for x := range cfg.Exchanges {
		if strings.EqualFold(cfg.Exchanges[x].Name, exchangeName) && cfg.Exchanges[x].Enabled {
			return true
		}
	}
	return false
}

// isValidCondition validates passed in condition
func isValidCondition(condition string) bool {
	switch condition {
	case ConditionGreaterThan, ConditionGreaterThanOrEqual, ConditionLessThan, ConditionLessThanOrEqual, ConditionIsEqual:
		return true
	}
	return false
}

// isValidAction validates passed in action
func isValidAction(action string) bool {
	action = strings.ToUpper(action)
	switch action {
	case ActionSMSNotify, ActionConsolePrint, ActionTest:
		return true
	}
	return false
}

// isValidItem validates passed in Item
func isValidItem(item string) bool {
	item = strings.ToUpper(item)
	switch item {
	case ItemPrice, ItemOrderbook:
		return true
	}
	return false
}

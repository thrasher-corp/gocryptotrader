package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupEventManager loads and validates the communications manager config
func setupEventManager(comManager iCommsManager, exchangeManager iExchangeManager, sleepDelay time.Duration, verbose bool) (*eventManager, error) {
	if comManager == nil {
		return nil, errNilComManager
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if sleepDelay <= 0 {
		sleepDelay = EventSleepDelay
	}
	return &eventManager{
		comms:           comManager,
		exchangeManager: exchangeManager,
		verbose:         verbose,
		sleepDelay:      sleepDelay,
		shutdown:        make(chan struct{}),
	}, nil
}

// Start runs the subsystem
func (m *eventManager) Start() error {
	if m == nil {
		return fmt.Errorf("event manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("event manager %w", ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.EventMgr, "Event Manager started. SleepDelay: %v\n", m.sleepDelay.String())
	m.shutdown = make(chan struct{})
	go m.run()
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (m *eventManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Stop attempts to shutdown the subsystem
func (m *eventManager) Stop() error {
	if m == nil {
		return fmt.Errorf("event manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("event manager %w", ErrSubSystemNotStarted)
	}
	close(m.shutdown)
	return nil
}

func (m *eventManager) run() {
	t := time.NewTicker(m.sleepDelay)
	select {
	case <-m.shutdown:
		return
	case <-t.C:
		total, executed := m.getEventCounter()
		if total > 0 && executed != total {
			m.m.Lock()
			for i := range m.events {
				m.executeEvent(i)
			}
			m.m.Unlock()
		}
	}
}

func (m *eventManager) executeEvent(i int) {
	if !m.events[i].Executed {
		if m.verbose {
			log.Debugf(log.EventMgr, "Events: Processing event %s.\n", m.events[i].String())
		}
		err := m.checkEventCondition(&m.events[i])
		if err != nil {
			msg := fmt.Sprintf(
				"Events: ID: %d triggered on %s successfully [%v]\n", m.events[i].ID,
				m.events[i].Exchange, m.events[i].String(),
			)
			log.Infoln(log.EventMgr, msg)
			m.comms.PushEvent(base.Event{Type: "event", Message: msg})
			m.events[i].Executed = true
		} else if m.verbose {
			log.Debugf(log.EventMgr, "%v", err)
		}
	}
}

// Add adds an event to the Events chain and returns an index/eventID
// and an error
func (m *eventManager) Add(exchange, item string, condition EventConditionParams, p currency.Pair, a asset.Item, action string) (int64, error) {
	if m == nil {
		return 0, fmt.Errorf("event manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return 0, fmt.Errorf("event manager %w", ErrSubSystemNotStarted)
	}
	err := m.isValidEvent(exchange, item, condition, action)
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
	m.m.Lock()
	if len(m.events) > 0 {
		evt.ID = int64(len(m.events) + 1)
	}
	m.events = append(m.events, evt)
	m.m.Unlock()

	return evt.ID, nil
}

// Remove deletes an event by its ID
func (m *eventManager) Remove(eventID int64) bool {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return false
	}
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.events {
		if m.events[i].ID == eventID {
			m.events = slices.Delete(m.events, i, i+1)
			return true
		}
	}
	return false
}

// getEventCounter displays the amount of total events on the chain and the
// events that have been executed.
func (m *eventManager) getEventCounter() (total, executed int) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return 0, 0
	}
	m.m.Lock()
	defer m.m.Unlock()
	total = len(m.events)
	for i := range m.events {
		if m.events[i].Executed {
			executed++
		}
	}
	return total, executed
}

// checkEventCondition will check the event structure to see if there is a condition
// met
func (m *eventManager) checkEventCondition(e *Event) error {
	if m == nil {
		return fmt.Errorf("event manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("event manager %w", ErrSubSystemNotStarted)
	}
	if e == nil {
		return errNilEvent
	}
	if e.Item == ItemPrice {
		return e.processTicker()
	}
	return e.processOrderbook()
}

// isValidEvent checks the actions to be taken and returns an error if incorrect
func (m *eventManager) isValidEvent(exchange, item string, condition EventConditionParams, action string) error {
	exchange = strings.ToUpper(exchange)
	item = strings.ToUpper(item)
	action = strings.ToUpper(action)

	if !m.isValidExchange(exchange) {
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
func (m *eventManager) isValidExchange(exchangeName string) bool {
	_, err := m.exchangeManager.GetExchangeByName(exchangeName)
	return err == nil
}

// isValidCondition validates passed in condition
func isValidCondition(condition string) bool {
	switch condition {
	case ConditionGreaterThan, ConditionGreaterThanOrEqual, ConditionLessThan, ConditionLessThanOrEqual, ConditionIsEqual:
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

// String turns the structure event into a string
func (e *Event) String() string {
	return fmt.Sprintf(
		"If the %s [%s] %s on %s meets the following %v then %s.", e.Pair.String(),
		strings.ToUpper(e.Asset.String()), e.Item, e.Exchange, e.Condition, e.Action,
	)
}

func (e *Event) processTicker() error {
	t, err := ticker.GetTicker(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		return fmt.Errorf("failed to get ticker. Err: %w", err)
	}

	if t.Last == 0 {
		return errTickerLastPriceZero
	}
	return e.shouldProcessEvent(t.Last, e.Condition.Price)
}

func (e *Event) shouldProcessEvent(actual, threshold float64) error {
	switch e.Condition.Condition {
	case ConditionGreaterThan:
		if actual > threshold {
			return nil
		}
	case ConditionGreaterThanOrEqual:
		if actual >= threshold {
			return nil
		}
	case ConditionLessThan:
		if actual < threshold {
			return nil
		}
	case ConditionLessThanOrEqual:
		if actual <= threshold {
			return nil
		}
	case ConditionIsEqual:
		if actual == threshold {
			return nil
		}
	}
	return errors.New("does not meet conditions")
}

func (e *Event) processOrderbook() error {
	ob, err := orderbook.Get(e.Exchange, e.Pair, e.Asset)
	if err != nil {
		return fmt.Errorf("events: Failed to get orderbook. Err: %w", err)
	}
	if !e.Condition.CheckBids && !e.Condition.CheckAsks {
		return nil
	}

	if e.Condition.CheckBids {
		for x := range ob.Bids {
			subtotal := ob.Bids[x].Amount * ob.Bids[x].Price
			err = e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if err == nil {
				log.Debugf(log.EventMgr, "Events: Bid Amount: %f Price: %v Subtotal: %v\n", ob.Bids[x].Amount, ob.Bids[x].Price, subtotal)
			}
		}
	}

	if e.Condition.CheckAsks {
		for x := range ob.Asks {
			subtotal := ob.Asks[x].Amount * ob.Asks[x].Price
			err = e.shouldProcessEvent(subtotal, e.Condition.OrderbookAmount)
			if err == nil {
				log.Debugf(log.EventMgr, "Events: Ask Amount: %f Price: %v Subtotal: %v\n", ob.Asks[x].Amount, ob.Asks[x].Price, subtotal)
			}
		}
	}
	return err
}

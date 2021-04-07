package eventmanager

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications/base"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

// iCommsManager limits exposure of accessible functions to communication manager
type iCommsManager interface {
	PushEvent(evt base.Event)
}

// Manager holds communication manager data
type Manager struct {
	started int32
	comms   iCommsManager
	events  []Event
	verbose bool
}

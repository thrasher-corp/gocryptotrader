package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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
	EventSleepDelay        = defaultSleepDelay
	errInvalidItem         = errors.New("invalid item")
	errInvalidCondition    = errors.New("invalid conditional option")
	errInvalidAction       = errors.New("invalid action")
	errExchangeDisabled    = errors.New("desired exchange is disabled")
	errNilEvent            = errors.New("nil event received")
	errNilComManager       = errors.New("nil communications manager received")
	errTickerLastPriceZero = errors.New("ticker last price is 0")
)

// EventConditionParams holds the event condition variables
type EventConditionParams struct {
	Condition string
	Price     float64

	CheckBids       bool
	CheckAsks       bool
	OrderbookAmount float64
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

// eventManager holds communication manager data
type eventManager struct {
	started         int32
	comms           iCommsManager
	events          []Event
	verbose         bool
	sleepDelay      time.Duration
	exchangeManager iExchangeManager
	shutdown        chan struct{}
	m               sync.Mutex
}

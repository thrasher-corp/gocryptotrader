package eventmanager

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"

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
	ErrInvalidItem         = errors.New("invalid item")
	ErrInvalidCondition    = errors.New("invalid conditional option")
	ErrInvalidAction       = errors.New("invalid action")
	ErrExchangeDisabled    = errors.New("desired exchange is disabled")
	errNilEvent            = errors.New("nil event received")
	EventSleepDelay        = defaultSleepDelay
	errNilComManager       = errors.New("nil communications manager received")
	errNilExchangeManager  = errors.New("nil exchange manager received")
	errTickerLastPriceZero = errors.New("ticker last price is 0")
	errTickerFail          = errors.New("failed to get ticker")
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

// iExchangeManager limits exposure of accessible functions to order manager
type iExchangeManager interface {
	GetExchangeByName(string) exchange.IBotExchange
}

// Manager holds communication manager data
type Manager struct {
	started         int32
	comms           iCommsManager
	events          []Event
	verbose         bool
	sleepDelay      time.Duration
	exchangeManager iExchangeManager
}

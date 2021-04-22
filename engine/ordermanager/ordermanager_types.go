package ordermanager

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Name is an exported subsystem name
const Name = "orders"

// vars for the fund manager package
var (
	orderManagerDelay = time.Second * 10
	// ErrOrdersAlreadyExists occurs when the order already exists in the manager
	ErrOrdersAlreadyExists = errors.New("order already exists")
	// ErrOrderNotFound occurs when an order is not found in the orderstore
	ErrOrderNotFound            = errors.New("order does not exist")
	errNilExchangeManager       = errors.New("cannot start with nil exchange manager")
	errNilCommunicationsManager = errors.New("cannot start with nil communications manager")
	errNilWaitGroup             = errors.New("cannot start with nil waitgroup")
	// ErrOrderIDCannotBeEmpty occurs when an order does not have an ID
	ErrOrderIDCannotBeEmpty = errors.New("orderID cannot be empty")
)

// iExchangeManager limits exposure of accessible functions to order manager
type iExchangeManager interface {
	GetExchanges() []exchange.IBotExchange
	GetExchangeByName(string) exchange.IBotExchange
}

// iCommsManager limits exposure of accessible functions to communication manager
type iCommsManager interface {
	PushEvent(evt base.Event)
}

type orderManagerConfig struct {
	EnforceLimitConfig     bool
	AllowMarketOrders      bool
	CancelOrdersOnShutdown bool
	LimitAmount            float64
	AllowedPairs           currency.Pairs
	AllowedExchanges       []string
	OrderSubmissionRetries int64
}

// store holds all orders by exchange
type store struct {
	m               sync.RWMutex
	Orders          map[string][]*order.Detail
	commsManager    iCommsManager
	exchangeManager iExchangeManager
	wg              *sync.WaitGroup
}

// Manager processes and stores orders across enabled exchanges
type Manager struct {
	started    int32
	shutdown   chan struct{}
	orderStore store
	cfg        orderManagerConfig
	verbose    bool
}

// OrderSubmitResponse contains the order response along with an internal order ID
type OrderSubmitResponse struct {
	order.SubmitResponse
	InternalOrderID string
}

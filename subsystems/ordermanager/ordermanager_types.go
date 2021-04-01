package ordermanager

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/communications/base"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errOrderIDCannotBeEmpty = errors.New("orderID cannot be empty")
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

type store struct {
	m               sync.RWMutex
	Orders          map[string][]*order.Detail
	commsManager    iCommsManager
	exchangeManager iExchangeManager
	wg              *sync.WaitGroup
}

type Manager struct {
	started    int32
	shutdown   chan struct{}
	orderStore store
	cfg        orderManagerConfig
	verbose    bool
}

type orderSubmitResponse struct {
	order.SubmitResponse
	InternalOrderID string
}

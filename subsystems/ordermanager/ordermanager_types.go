package ordermanager

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errOrderCannotBeEmpty = errors.New("order cannot be empty")
)

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
	m      sync.RWMutex
	Orders map[string][]*order.Detail
	bot    *engine.Engine
}

type Manager struct {
	started    int32
	shutdown   chan struct{}
	orderStore store
	cfg        orderManagerConfig
}

type orderSubmitResponse struct {
	order.SubmitResponse
	InternalOrderID string
}

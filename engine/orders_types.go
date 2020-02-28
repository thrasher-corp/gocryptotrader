package engine

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

type orderStore struct {
	m      sync.RWMutex
	Orders map[string][]*order.Detail
}

type orderManager struct {
	started    int32
	stopped    int32
	shutdown   chan struct{}
	orderStore orderStore
	cfg        orderManagerConfig
}

type orderSubmitResponse struct {
	order.SubmitResponse
	InternalOrderID string
}

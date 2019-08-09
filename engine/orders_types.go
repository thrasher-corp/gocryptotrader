package engine

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	m      sync.Mutex
	Orders map[string][]exchange.OrderDetail
}

type orderManager struct {
	started    int32
	stopped    int32
	shutdown   chan struct{}
	orderStore orderStore
	cfg        orderManagerConfig
}

type orderSubmitResponse struct {
	exchange.SubmitOrderResponse
	OurOrderID string
}

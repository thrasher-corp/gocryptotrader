package ordermanager

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/subsystems/communicationmanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errOrderIDCannotBeEmpty = errors.New("orderID cannot be empty")
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
	m               sync.RWMutex
	Orders          map[string][]*order.Detail
	commsManager    *communicationmanager.Manager
	exchangeManager *exchangemanager.Manager
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

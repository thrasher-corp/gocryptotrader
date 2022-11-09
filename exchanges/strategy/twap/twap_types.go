package twap

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	*Config
	strategy.Requirement
	*strategy.Scheduler
	Selling    *account.ProtectedBalance
	allocation *Allocation
	orderbook  *orderbook.Depth
	wg         sync.WaitGroup
	shutdown   chan struct{}
	running    bool
	mtx        sync.Mutex
}

// Allocation defines the full allocation of funds and information of strategy
// break down.
type Allocation struct {
	// Total defines the entire pool of funds to be deployed
	Total float64
	// Deployment defines the 'at-signal' funds to be deployed
	Deployment float64
	// Window defines estimated window of strategy operation
	Window time.Duration
	// Interval defines the spacing between deployment
	Interval kline.Interval
	// Deployments define the estimated orders to complete strategy
	Deployments int64
	// Start defines the time at which the strategy is scheduled to commence
	Start time.Time
	// End defines the time at which the strategy is estimated to be completed
	End time.Time
	// Deployed defines what has been deployed to the exchange
	Deployed float64
	// Runs define how many executions have occured
	Runs int64
}

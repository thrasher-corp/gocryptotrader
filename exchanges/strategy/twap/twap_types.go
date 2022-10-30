package twap

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/strategy"
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	strategy.Base
	*Config
	Buying           *account.ProtectedBalance
	Selling          *account.ProtectedBalance
	Reporter         chan Report
	TradeInformation []OrderExecutionInformation

	FullDeployment   float64
	DeploymentAmount float64
	AmountDeployed   float64

	orderbook *orderbook.Depth
	wg        sync.WaitGroup
	// shutdown  chan struct{}
	// pause     chan struct{}
	// running   bool
	// paused    bool
	// finished  bool
	// mtx       sync.Mutex
}

type Holding struct {
	Currency currency.Code
	Amount   *account.ProtectedBalance
}

type Holdings struct {
	Current map[currency.Code]*account.ProtectedBalance
}

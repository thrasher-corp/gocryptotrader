package twap

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	*Config
	Buying  *account.ProtectedBalance
	Selling *account.ProtectedBalance

	Reporter strategy.Reporter

	FullDeployment   float64
	DeploymentAmount float64
	AmountDeployed   float64

	orderbook *orderbook.Depth

	wg       sync.WaitGroup
	shutdown chan struct{}
	running  bool
	mtx      sync.Mutex
}

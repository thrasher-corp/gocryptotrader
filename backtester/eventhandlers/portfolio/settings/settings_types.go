package settings

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

type Settings struct {
	InitialFunds      float64
	Fee               float64
	BuySideSizing     config.MinMax
	SellSideSizing    config.MinMax
	Leverage          config.Leverage
	HoldingsSnapshots holdings.Snapshots
	ComplianceManager compliance.Manager
}

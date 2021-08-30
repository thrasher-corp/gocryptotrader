package settings

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/holdings"
)

// Settings holds all important information for the portfolio manager
// to assess purchasing decisions
type Settings struct {
	Fee               decimal.Decimal
	BuySideSizing     config.MinMax
	SellSideSizing    config.MinMax
	Leverage          config.Leverage
	HoldingsSnapshots []holdings.Holding
	ComplianceManager compliance.Manager
}

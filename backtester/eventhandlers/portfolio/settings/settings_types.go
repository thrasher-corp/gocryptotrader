package settings

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

// Settings holds all important information for the portfolio manager
// to assess purchasing decisions
type Settings struct {
	Fee               decimal.Decimal
	BuySideSizing     exchange.MinMax
	SellSideSizing    exchange.MinMax
	Leverage          exchange.Leverage
	HoldingsSnapshots []holdings.Holding
	ComplianceManager compliance.Manager
}

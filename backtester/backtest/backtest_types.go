package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

type BackTest struct {
	shutdown   chan struct{}
	Data       interfaces.DataHandler
	Strategy   strategies.StrategyHandler
	Portfolio  portfolio.PortfolioHandler
	Exchange   exchange.ExecutionHandler
	Statistic  statistics.StatisticHandler
	EventQueue []interfaces.EventHandler
	Bot        *engine.Engine
	Compliance compliance.Manager
}

package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

// BackTest is the main hodler of all backtesting
type BackTest struct {
	shutdown   chan struct{}
	Data       interfaces.DataHandler
	Strategy   strategies.Handler
	Portfolio  portfolio.Handler
	Exchange   exchange.ExecutionHandler
	Statistic  statistics.Handler
	EventQueue []interfaces.EventHandler
	Bot        *engine.Engine
}

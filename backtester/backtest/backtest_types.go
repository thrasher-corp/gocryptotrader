package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	datahandler2 "github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
)

type BackTest struct {
	Data       interfaces.DataHandler
	Strategy   strategies.StrategyHandler
	Portfolio  portfolio.PortfolioHandler
	Orders     datahandler2.Orders
	Exchange   exchange.ExecutionHandler
	Statistic  statistics.StatisticHandler
	EventQueue []interfaces.EventHandler
}

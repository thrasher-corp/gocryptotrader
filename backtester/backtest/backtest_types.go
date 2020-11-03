package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	datahandler2 "github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
)

type BackTest struct {
	Data       interfaces.DataHandler
	Strategy   strategies.StrategyHandler
	Portfolio  portfolio.PortfolioHandler
	Orders     datahandler2.Orders
	Exchange   datahandler2.ExecutionHandler
	Statistic  statistics.StatisticHandler
	EventQueue []interfaces.EventHandler
}

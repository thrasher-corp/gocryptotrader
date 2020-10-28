package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/execution"
	datahandler2 "github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/strategy"
)

type BackTest struct {
	Data       datahandler.DataHandler
	Strategy   strategy.StrategyHandler
	Portfolio  portfolio.PortfolioHandler
	Orderbook  datahandler2.OrderBook
	Exchange   execution.ExecutionHandler
	Statistic  statistics.StatisticHandler
	EventQueue []datahandler.EventHandler
}

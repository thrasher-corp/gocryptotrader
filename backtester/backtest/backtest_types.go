package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	datahandler2 "github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/strategies"
)

type BackTest struct {
	Data       datahandler.DataHandler
	Strategy   strategies.StrategyHandler
	Portfolio  portfolio.PortfolioHandler
	Orderbook  datahandler2.OrderBook
	Exchange   datahandler2.ExecutionHandler
	Statistic  statistics.StatisticHandler
	EventQueue []datahandler.EventHandler
}

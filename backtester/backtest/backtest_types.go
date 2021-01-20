package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

// BackTest is the main holder of all backtesting functionality
type BackTest struct {
	Bot             *engine.Engine
	hasHandledEvent bool
	shutdown        chan struct{}
	Datas           data.Holder
	Strategy        strategies.Handler
	Portfolio       portfolio.Handler
	Exchange        exchange.ExecutionHandler
	Statistic       statistics.Handler
	EventQueue      eventholder.EventHolder
	Reports         report.Handler
}

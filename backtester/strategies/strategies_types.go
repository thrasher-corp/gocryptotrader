package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
)

type StrategyHandler interface {
	Name() string
	OnSignal(datahandler.DataHandler, portfolio.PortfolioHandler) (signal.SignalEvent, error)
}

const errNotFound = "strategy %v not found"

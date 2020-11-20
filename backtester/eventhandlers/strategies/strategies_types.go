package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type StrategyHandler interface {
	Name() string
	OnSignal(interfaces.DataHandler, portfolio.PortfolioHandler) (signal.SignalEvent, error)
	SetCustomSettings(map[string]interface{}) error
	SetDefaults()
}

const errNotFound = "strategy %v not found"

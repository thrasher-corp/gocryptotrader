package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type Handler interface {
	Name() string
	OnSignal(interfaces.DataHandler, portfolio.Handler) (signal.SignalEvent, error)
	SetCustomSettings(map[string]interface{}) error
	SetDefaults()
}

const errNotFound = "strategy %v not found"

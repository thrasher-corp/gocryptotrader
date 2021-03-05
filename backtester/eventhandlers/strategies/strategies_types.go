package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

// Handler defines all functions required to run strategies against data events
type Handler interface {
	Name() string
	Description() string
	OnSignal(data.Handler, portfolio.Handler) (signal.Event, error)
	OnSimultaneousSignals([]data.Handler, portfolio.Handler) ([]signal.Event, error)
	UseSimultaneousProcessing() bool
	SupportsSimultaneousProcessing() bool
	SetSimultaneousProcessing(bool)
	SetCustomSettings(map[string]interface{}) error
	SetDefaults()
}

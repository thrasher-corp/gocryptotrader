package strategies

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

// Handler defines all functions required to run strategies against data events
type Handler interface {
	Name() string
	OnSignal(data.Handler, portfolio.Handler) (signal.Event, error)
	OnSignals([]data.Handler, portfolio.Handler) ([]signal.Event, error)
	IsMultiCurrency() bool
	SupportsMultiCurrency() bool
	SetMultiCurrency(bool)
	SetCustomSettings(map[string]interface{}) error
	SetDefaults()
}

const errNotFound = "strategy '%v' not found. Please ensure the strategy-settings field 'name' is spelled properly in your .strat config"

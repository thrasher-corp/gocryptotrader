package strategies

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

// ErrStrategyAlreadyExists returned when a strategy matches the same name
var ErrStrategyAlreadyExists = errors.New("strategy already exists")

// StrategyHolder holds strategies
type StrategyHolder []Handler

// Handler defines all functions required to run strategies against data events
type Handler interface {
	Name() string
	Description() string
	OnSignal(data.Handler, funding.IFundingTransferer, portfolio.Handler) (signal.Event, error)
	OnSimultaneousSignals([]data.Handler, funding.IFundingTransferer, portfolio.Handler) ([]signal.Event, error)
	UsingSimultaneousProcessing() bool
	SupportsSimultaneousProcessing() bool
	SetSimultaneousProcessing(bool)
	SetCustomSettings(map[string]any) error
	SetDefaults()
	CloseAllPositions([]holdings.Holding, []data.Event) ([]signal.Event, error)
}

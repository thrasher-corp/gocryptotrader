package main

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func main() {
	// required for plugin system
}

// CustomStrategy the type used to define custom strategy functions
type CustomStrategy struct {
	base.Strategy
}

// GetStrategies is required to load the strategy or strategies into the GoCryptoTrader Backtester
func GetStrategies() []strategies.Handler {
	return []strategies.Handler{&CustomStrategy{}}
}

// Name returns the name of the strategy
func (s *CustomStrategy) Name() string {
	return "custom-strategy"
}

// Description describes the strategy
func (s *CustomStrategy) Description() string {
	return "this is a demonstration of loading strategies via custom plugins"
}

// SupportsSimultaneousProcessing this strategy only supports simultaneous signal processing
func (s *CustomStrategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSignal handles a data event and returns what action the strategy believes should occur
func (s *CustomStrategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return s.createSignal(d)
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *CustomStrategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	response := make([]signal.Event, len(d))
	for i := range d {
		sig, err := s.createSignal(d[i])
		if err != nil {
			return nil, err
		}
		response[i] = sig
	}
	return response, nil
}

func (s *CustomStrategy) createSignal(d data.Handler) (*signal.Signal, error) {
	es, err := s.GetBaseData(d)
	if err != nil {
		return nil, err
	}

	sig := &signal.Signal{
		Base:       es.Base,
		OpenPrice:  es.OpenPrice,
		HighPrice:  es.HighPrice,
		LowPrice:   es.LowPrice,
		ClosePrice: es.ClosePrice,
		Volume:     es.Volume,
		BuyLimit:   es.BuyLimit,
		SellLimit:  es.SellLimit,
		Amount:     es.Amount,
		Direction:  gctorder.Buy,
	}
	sig.AppendReasonf("Signalling purchase of %s", es.Base.Pair())
	return sig, nil
}

// SetCustomSettings can override default settings
func (s *CustomStrategy) SetCustomSettings(map[string]any) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults sets default values for overridable custom settings
func (s *CustomStrategy) SetDefaults() {}

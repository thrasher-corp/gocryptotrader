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

}

type ImAStrategy struct {
	base.Strategy
}

func GetStrategy() strategies.Handler {
	return &ImAStrategy{}
}

// Name returns the name of the strategy
func (s *ImAStrategy) Name() string {
	return "hello"
}

// Description describes the strategy
func (s *ImAStrategy) Description() string {
	return "moto"
}

// SupportsSimultaneousProcessing this strategy only supports simultaneous signal processing
func (s *ImAStrategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For rsi, this means returning a buy signal when rsi is at or below a certain level, and a
// sell signal when it is at or above a certain level
func (s *ImAStrategy) OnSignal(data.Handler, funding.IFundingTransferer, portfolio.Handler) (signal.Event, error) {
	return nil, base.ErrSimultaneousProcessingOnly
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *ImAStrategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, p portfolio.Handler) ([]signal.Event, error) {
	var response []signal.Event
	for i := range d {
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}

		response = append(response, &signal.Signal{
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
		})
	}
	return response, nil
}

// SetCustomSettings can override default settings
func (s *ImAStrategy) SetCustomSettings(map[string]interface{}) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults sets default values for overridable custom settings
func (s *ImAStrategy) SetDefaults() {
}

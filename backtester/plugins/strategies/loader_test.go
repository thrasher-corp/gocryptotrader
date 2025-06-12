package strategies

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

func TestAddStrategies(t *testing.T) {
	t.Parallel()
	err := addStrategies(nil)
	assert.ErrorIs(t, err, errNoStrategies)

	err = addStrategies([]strategies.Handler{&dollarcostaverage.Strategy{}})
	assert.ErrorIs(t, err, strategies.ErrStrategyAlreadyExists)

	err = addStrategies([]strategies.Handler{&CustomStrategy{}})
	assert.NoError(t, err)
}

type CustomStrategy struct {
	base.Strategy
}

func (s *CustomStrategy) Name() string {
	return "custom-strategy"
}

func (s *CustomStrategy) Description() string {
	return "this is a demonstration of loading strategies via custom plugins"
}

func (s *CustomStrategy) SupportsSimultaneousProcessing() bool {
	return true
}

func (s *CustomStrategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return s.createSignal(d)
}

func (s *CustomStrategy) OnSimultaneousSignals(_ []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	return nil, nil
}

func (s *CustomStrategy) createSignal(_ data.Handler) (*signal.Signal, error) {
	return nil, nil
}

func (s *CustomStrategy) SetCustomSettings(map[string]any) error {
	return nil
}

func (s *CustomStrategy) SetDefaults() {}

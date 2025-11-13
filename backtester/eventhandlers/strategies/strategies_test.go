package strategies

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestGetStrategies(t *testing.T) {
	t.Parallel()
	if resp := GetSupportedStrategies(); len(resp) < 2 {
		t.Error("expected at least 2 strategies to be loaded")
	}
}

func TestLoadStrategyByName(t *testing.T) {
	t.Parallel()
	var resp Handler
	_, err := LoadStrategyByName("test", false)
	assert.ErrorIs(t, err, base.ErrStrategyNotFound)

	_, err = LoadStrategyByName("test", true)
	assert.ErrorIs(t, err, base.ErrStrategyNotFound)

	resp, err = LoadStrategyByName(dollarcostaverage.Name, false)
	assert.NoError(t, err)

	if resp.Name() != dollarcostaverage.Name {
		t.Error("expected dca")
	}
	resp, err = LoadStrategyByName(dollarcostaverage.Name, true)
	assert.NoError(t, err)

	if !resp.UsingSimultaneousProcessing() {
		t.Error("expected true")
	}

	resp, err = LoadStrategyByName(rsi.Name, false)
	assert.NoError(t, err)

	if resp.Name() != rsi.Name {
		t.Error("expected rsi")
	}
	_, err = LoadStrategyByName(rsi.Name, true)
	assert.NoError(t, err)
}

func TestAddStrategy(t *testing.T) {
	t.Parallel()
	err := AddStrategy(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	err = AddStrategy(new(dollarcostaverage.Strategy))
	assert.ErrorIs(t, err, ErrStrategyAlreadyExists)

	err = AddStrategy(new(customStrategy))
	assert.NoError(t, err)
}

func TestCreateNewStrategy(t *testing.T) {
	t.Parallel()

	// invalid Handler
	_, err := createNewStrategy(dollarcostaverage.Name, false, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	// mismatched name
	resp, err := createNewStrategy(dollarcostaverage.Name, false, &customStrategy{})
	assert.NoError(t, err, "createNewStrategy should not error")
	assert.Nil(t, resp)

	// nil Handler
	var h Handler = (*customStrategy)(nil)
	_, err = createNewStrategy("custom-strategy", false, h)
	assert.ErrorContains(t, err, "be a non-nil pointer")

	// valid
	h = new(dollarcostaverage.Strategy)
	resp, err = createNewStrategy(dollarcostaverage.Name, true, h)
	assert.NoError(t, err, "createNewStrategy should not error")
	assert.NotSame(t, h, resp, "createNewStrategy should return a new pointer")

	// simultaneous processing desired but not supported
	h = &customStrategy{allowSimultaneousProcessing: false}
	_, err = createNewStrategy("custom-strategy", true, h)
	assert.ErrorIs(t, err, base.ErrSimultaneousProcessingNotSupported)
}

type customStrategy struct {
	allowSimultaneousProcessing bool
	base.Strategy
}

func (s *customStrategy) Name() string {
	return "custom-strategy"
}

func (s *customStrategy) Description() string {
	return "this is a demonstration of loading strategies via custom plugins"
}

func (s *customStrategy) SupportsSimultaneousProcessing() bool {
	return s.allowSimultaneousProcessing
}

func (s *customStrategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return s.createSignal(d)
}

func (s *customStrategy) OnSimultaneousSignals(_ []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	return nil, nil
}

func (s *customStrategy) createSignal(_ data.Handler) (*signal.Signal, error) {
	return nil, nil
}

func (s *customStrategy) SetCustomSettings(map[string]any) error {
	return nil
}

// SetDefaults sets default values for overridable custom settings
func (s *customStrategy) SetDefaults() {}

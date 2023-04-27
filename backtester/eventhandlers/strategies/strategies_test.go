package strategies

import (
	"errors"
	"testing"

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
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}
	_, err = LoadStrategyByName("test", true)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	resp, err = LoadStrategyByName(dollarcostaverage.Name, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.Name() != dollarcostaverage.Name {
		t.Error("expected dca")
	}
	resp, err = LoadStrategyByName(dollarcostaverage.Name, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !resp.UsingSimultaneousProcessing() {
		t.Error("expected true")
	}

	resp, err = LoadStrategyByName(rsi.Name, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.Name() != rsi.Name {
		t.Error("expected rsi")
	}
	_, err = LoadStrategyByName(rsi.Name, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddStrategy(t *testing.T) {
	t.Parallel()
	err := AddStrategy(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilPointer)
	}
	err = AddStrategy(new(dollarcostaverage.Strategy))
	if !errors.Is(err, ErrStrategyAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrStrategyAlreadyExists)
	}

	err = AddStrategy(new(customStrategy))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestCreateNewStrategy(t *testing.T) {
	t.Parallel()

	// invalid Handler
	resp, err := createNewStrategy(dollarcostaverage.Name, false, nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilPointer)
	}
	if resp != nil {
		t.Errorf("received '%v' expected '%v'", resp, nil)
	}

	// mismatched name
	resp, err = createNewStrategy(dollarcostaverage.Name, false, &customStrategy{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp != nil {
		t.Errorf("received '%v' expected '%v'", resp, nil)
	}

	// valid
	h := new(dollarcostaverage.Strategy)
	resp, err = createNewStrategy(dollarcostaverage.Name, false, h)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp == nil {
		t.Errorf("received '%v' expected '%v'", resp, h)
	}
}

type customStrategy struct {
	base.Strategy
}

func (s *customStrategy) Name() string {
	return "custom-strategy"
}
func (s *customStrategy) Description() string {
	return "this is a demonstration of loading strategies via custom plugins"
}
func (s *customStrategy) SupportsSimultaneousProcessing() bool {
	return true
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
func (s *customStrategy) SetCustomSettings(map[string]interface{}) error {
	return nil
}

// SetDefaults sets default values for overridable custom settings
func (s *customStrategy) SetDefaults() {}

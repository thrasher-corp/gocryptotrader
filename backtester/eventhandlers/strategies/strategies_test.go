package strategies

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
)

func TestGetStrategies(t *testing.T) {
	resp := GetStrategies()
	if len(resp) < 2 {
		t.Error("expected at least 2 strategies to be loaded")
	}
}

func TestLoadStrategyByName(t *testing.T) {
	var resp Handler
	_, err := LoadStrategyByName("test", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("expected: %v, received %v", base.ErrStrategyNotFound, err)
	}
	_, err = LoadStrategyByName("test", true)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("expected: %v, received %v", base.ErrStrategyNotFound, err)
	}

	resp, err = LoadStrategyByName(dollarcostaverage.Name, false)
	if err != nil {
		t.Error(err)
	}
	if resp.Name() != dollarcostaverage.Name {
		t.Error("expected dca")
	}
	resp, err = LoadStrategyByName(dollarcostaverage.Name, true)
	if err != nil {
		t.Error(err)
	}
	if !resp.UseSimultaneousProcessing() {
		t.Error("expected true")
	}

	resp, err = LoadStrategyByName(rsi.Name, false)
	if err != nil {
		t.Error(err)
	}
	if resp.Name() != rsi.Name {
		t.Error("expected rsi")
	}
	_, err = LoadStrategyByName(rsi.Name, true)
	if !errors.Is(err, base.ErrSimultaneousProcessingNotSupported) {
		t.Errorf("expected: %v, received %v", base.ErrSimultaneousProcessingNotSupported, err)
	}
}

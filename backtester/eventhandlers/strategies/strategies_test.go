package strategies

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
)

func TestGetStrategies(t *testing.T) {
	resp := getStrategies()
	if len(resp) < 2 {
		t.Error("expected at least 2 strategies to be loaded")
	}
}

func TestLoadStrategyByName(t *testing.T) {
	var resp Handler
	_, err := LoadStrategyByName("test", false)
	if err != nil && err.Error() != "strategy 'test' not found" {
		t.Error(err)
	}
	_, err = LoadStrategyByName("test", true)
	if err != nil && err.Error() != "strategy 'test' not found" {
		t.Error(err)
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
	if !resp.IsMultiCurrency() {
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
	if err != nil && err.Error() != "strategy 'rsi' does not support multi-currency assessment and could not be loaded" {
		t.Error(err)
	}
}

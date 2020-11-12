package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/RSI420BlazeIt"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string) (StrategyHandler, error) {
	strats := getStrategies()
	for i := range strats {
		if !strings.EqualFold(name, strats[i].Name()) {
			continue
		}
		return strats[i], nil
	}
	return nil, fmt.Errorf(errNotFound, name)
}

func getStrategies() []StrategyHandler {
	var strats []StrategyHandler
	strats = append(strats, new(dollarcostaverage.Strategy))
	strats = append(strats, new(RSI420BlazeIt.Strategy))

	return strats
}

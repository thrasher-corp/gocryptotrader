package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/strategies/RSI420BlazeIt"
	"github.com/thrasher-corp/gocryptotrader/backtester/strategies/buyandhold"
)

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
	strats = append(strats, new(buyandhold.Strategy))
	strats = append(strats, new(RSI420BlazeIt.Strategy))

	return strats
}

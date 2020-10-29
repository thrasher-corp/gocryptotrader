package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/strategies/RSI420BlazeIt"
	"github.com/thrasher-corp/gocryptotrader/backtester/strategies/smacross"
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
	var smaCross StrategyHandler
	var rsi420BlazeIt StrategyHandler
	smaCross = new(smacross.Strategy)
	rsi420BlazeIt = new(RSI420BlazeIt.Strategy)

	strats = append(strats, smaCross)
	strats = append(strats, rsi420BlazeIt)

	return strats
}

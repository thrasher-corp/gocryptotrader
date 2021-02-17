package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, useSimultaneousProcessing bool) (Handler, error) {
	strats := getStrategies()
	for i := range strats {
		if !strings.EqualFold(name, strats[i].Name()) {
			continue
		}
		if useSimultaneousProcessing {
			if !strats[i].SupportsSimultaneousProcessing() {
				return nil, fmt.Errorf(
					"strategy '%v' does not support simultaneous processing and could not be loaded",
					name)
			}
			strats[i].SetSimultaneousProcessing(useSimultaneousProcessing)
		}
		return strats[i], nil
	}
	return nil, fmt.Errorf(errNotFound, name)
}

func getStrategies() []Handler {
	var strats []Handler
	strats = append(strats,
		new(dollarcostaverage.Strategy),
		new(rsi.Strategy),
	)

	return strats
}

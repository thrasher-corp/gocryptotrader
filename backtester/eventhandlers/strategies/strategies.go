package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, isMultiCurrency bool) (Handler, error) {
	strats := getStrategies()
	for i := range strats {
		if !strings.EqualFold(name, strats[i].Name()) {
			continue
		}
		if isMultiCurrency {
			if !strats[i].SupportsMultiCurrency() {
				return nil, fmt.Errorf(
					"strategy '%v' does not support multi-currency assessment and could not be loaded",
					name)
			}
			strats[i].SetMultiCurrency(isMultiCurrency)
		}
		return strats[i], nil
	}
	return nil, fmt.Errorf(errNotFound, name)
}

func getStrategies() []Handler {
	var strats []Handler
	strats = append(strats, new(dollarcostaverage.Strategy))
	strats = append(strats, new(rsi.Strategy))

	return strats
}

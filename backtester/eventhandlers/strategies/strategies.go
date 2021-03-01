package strategies

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, useSimultaneousProcessing bool) (Handler, error) {
	strats := GetStrategies()
	for i := range strats {
		if !strings.EqualFold(name, strats[i].Name()) {
			continue
		}
		if useSimultaneousProcessing {
			if !strats[i].SupportsSimultaneousProcessing() {
				return nil, fmt.Errorf(
					"strategy '%v' %w",
					name,
					base.ErrSimultaneousProcessingNotSupported)
			}
			strats[i].SetSimultaneousProcessing(useSimultaneousProcessing)
		}
		return strats[i], nil
	}
	return nil, fmt.Errorf("strategy '%v' %w", name, base.ErrStrategyNotFound)
}

// GetStrategies returns a static list of set strategies
// they must be set in here for the backtester to recognise them
func GetStrategies() []Handler {
	var strats []Handler
	strats = append(strats,
		new(dollarcostaverage.Strategy),
		new(rsi.Strategy),
	)

	return strats
}

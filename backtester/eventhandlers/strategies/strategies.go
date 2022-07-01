package strategies

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/ftxcashandcarry"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, useSimultaneousProcessing bool) (Handler, error) {
	strategies := GetStrategies()
	for i := range strategies {
		if !strings.EqualFold(name, strategies[i].Name()) {
			continue
		}
		if useSimultaneousProcessing {
			if !strategies[i].SupportsSimultaneousProcessing() {
				return nil, fmt.Errorf(
					"strategy '%v' %w",
					name,
					base.ErrSimultaneousProcessingNotSupported)
			}
			strategies[i].SetSimultaneousProcessing(useSimultaneousProcessing)
		}
		return strategies[i], nil
	}
	return nil, fmt.Errorf("strategy '%v' %w", name, base.ErrStrategyNotFound)
}

// GetStrategies returns a static list of set strategies
// they must be set in here for the backtester to recognise them
func GetStrategies() StrategyHolder {
	m.Lock()
	defer m.Unlock()
	return strategyHolder
}

// AddStrategy will add a strategy to the list of strategies
func AddStrategy(strategy Handler) error {
	if strategy == nil {
		return fmt.Errorf("%w strategy handler", common.ErrNilPointer)
	}
	m.Lock()
	defer m.Unlock()
	for i := range strategyHolder {
		if strings.EqualFold(strategyHolder[i].Name(), strategy.Name()) {
			return fmt.Errorf("'%v' %w", strategy.Name(), errStrategyAlreadyExists)
		}
	}
	strategyHolder = append(strategyHolder, strategy)
	return nil
}

var (
	m sync.Mutex

	strategyHolder = StrategyHolder{
		new(dollarcostaverage.Strategy),
		new(rsi.Strategy),
		new(top2bottom2.Strategy),
		new(ftxcashandcarry.Strategy),
	}
)

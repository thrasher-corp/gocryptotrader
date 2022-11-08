package strategies

import (
	"fmt"
	"reflect"
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
	strategies := GetSupportedStrategies()
	for i := range strategies {
		if !strings.EqualFold(name, strategies[i].Name()) {
			continue
		}
		// create new instance so strategy is not shared across all tasks
		s, ok := reflect.New(reflect.ValueOf(strategies[i]).Elem().Type()).Interface().(Handler)
		if !ok {
			return nil, fmt.Errorf("'%v' %w when creating new strategy", name, base.ErrStrategyNotFound)
		}
		if useSimultaneousProcessing && !s.SupportsSimultaneousProcessing() {
			return nil, base.ErrSimultaneousProcessingNotSupported
		}
		s.SetSimultaneousProcessing(useSimultaneousProcessing)
		return s, nil
	}
	return nil, fmt.Errorf("strategy '%v' %w", name, base.ErrStrategyNotFound)
}

// GetSupportedStrategies returns a static list of set strategies
// they must be set in here for the backtester to recognise them
func GetSupportedStrategies() StrategyHolder {
	m.Lock()
	defer m.Unlock()
	return supportedStrategies
}

// AddStrategy will add a strategy to the list of strategies
func AddStrategy(strategy Handler) error {
	if strategy == nil {
		return fmt.Errorf("%w strategy handler", common.ErrNilPointer)
	}
	m.Lock()
	defer m.Unlock()
	for i := range supportedStrategies {
		if strings.EqualFold(supportedStrategies[i].Name(), strategy.Name()) {
			return fmt.Errorf("'%v' %w", strategy.Name(), ErrStrategyAlreadyExists)
		}
	}
	supportedStrategies = append(supportedStrategies, strategy)
	return nil
}

var (
	m                   sync.Mutex
	supportedStrategies = StrategyHolder{
		new(dollarcostaverage.Strategy),
		new(rsi.Strategy),
		new(top2bottom2.Strategy),
		new(ftxcashandcarry.Strategy),
	}
)

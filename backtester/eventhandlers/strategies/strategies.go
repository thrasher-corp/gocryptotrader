package strategies

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/rsi"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// LoadStrategyByName returns the strategy by its name
func LoadStrategyByName(name string, useSimultaneousProcessing bool) (Handler, error) {
	strategies := GetSupportedStrategies()
	for i := range strategies {
		strategy, err := createNewStrategy(name, useSimultaneousProcessing, strategies[i])
		if err != nil {
			return nil, err
		}
		if strategy != nil {
			return strategy, err
		}
	}
	return nil, fmt.Errorf("strategy '%v' %w", name, base.ErrStrategyNotFound)
}

func createNewStrategy(name string, useSimultaneousProcessing bool, h Handler) (Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("cannot load strategy %q: %w", name, common.ErrNilPointer)
	}

	if !strings.EqualFold(name, h.Name()) {
		return nil, nil
	}

	strategyValue := reflect.ValueOf(h)
	if strategyValue.Kind() != reflect.Ptr || strategyValue.IsNil() {
		return nil, fmt.Errorf("cannot load strategy %q: handler must be a non-nil pointer, got %T", name, h)
	}

	// create new instance so strategy is not shared across all tasks
	strategy, ok := reflect.New(strategyValue.Elem().Type()).Interface().(Handler)
	if !ok {
		return nil, fmt.Errorf("cannot load strategy %q: type %T doesn't implement Handler interface: %w",
			name, strategy, common.ErrTypeAssertFailure)
	}

	if useSimultaneousProcessing && !strategy.SupportsSimultaneousProcessing() {
		return nil, base.ErrSimultaneousProcessingNotSupported
	}

	strategy.SetSimultaneousProcessing(useSimultaneousProcessing)
	return strategy, nil
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
	m sync.Mutex

	supportedStrategies = StrategyHolder{
		new(dollarcostaverage.Strategy),
		new(rsi.Strategy),
		new(top2bottom2.Strategy),
		new(binancecashandcarry.Strategy),
	}
)

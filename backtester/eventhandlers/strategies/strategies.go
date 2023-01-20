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
		return nil, fmt.Errorf("cannot load %v supported strategies contains %w", name, common.ErrNilPointer)
	}
	if !strings.EqualFold(name, h.Name()) {
		return nil, nil
	}
	// create new instance so strategy is not shared across all tasks
	strategyValue := reflect.ValueOf(h)
	if strategyValue.IsNil() {
		return nil, fmt.Errorf("cannot load %v supported strategies element is a %w", name, common.ErrNilPointer)
	}
	strategyElement := strategyValue.Elem()
	if !strategyElement.IsValid() {
		return nil, fmt.Errorf("cannot load %v strategy element is invalid %w", name, common.ErrTypeAssertFailure)
	}
	strategyType := strategyElement.Type()
	if strategyType == nil {
		return nil, fmt.Errorf("cannot load %v strategy type is a %w", name, common.ErrNilPointer)
	}
	newStrategy := reflect.New(strategyType)
	if newStrategy.IsNil() {
		return nil, fmt.Errorf("cannot load %v new instance of strategy is a %w", name, common.ErrNilPointer)
	}
	strategyInterface := newStrategy.Interface()
	if strategyInterface == nil {
		return nil, fmt.Errorf("cannot load %v new instance of strategy is not an interface. %w", name, common.ErrTypeAssertFailure)
	}
	strategy, ok := strategyInterface.(Handler)
	if !ok {
		return nil, fmt.Errorf("cannot load %v new instance of strategy is not a Handler interface. %w", name, common.ErrTypeAssertFailure)
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

package strategies

import (
	"errors"
	"fmt"
	"plugin"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

var errNoStrategies = errors.New("no strategies contained in plugin. please refer to docs")

// LoadCustomStrategies utilises Go's plugin system to load
// custom strategies into the backtester.
func LoadCustomStrategies(strategyPluginPath string) error {
	p, err := plugin.Open(strategyPluginPath)
	if err != nil {
		return fmt.Errorf("could not open plugin: %w", err)
	}
	v, err := p.Lookup("GetStrategies")
	if err != nil {
		return fmt.Errorf("could not lookup plugin. Plugin must have function `GetStrategy`. Error: %w", err)
	}
	customStrategies, ok := v.(func() []strategies.Handler)
	if !ok {
		return gctcommon.GetTypeAssertError("[]strategies.Handler", customStrategies)
	}
	return addStrategies(customStrategies())
}

func addStrategies(s []strategies.Handler) error {
	if len(s) == 0 {
		return errNoStrategies
	}
	var err error
	for i := range s {
		err = strategies.AddStrategy(s[i])
		if err != nil {
			return err
		}
	}
	return nil
}

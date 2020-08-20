package backtester

import "github.com/d5/tengo/v2"

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{}

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range Modules {
		names = append(names, name)
	}
	return names
}

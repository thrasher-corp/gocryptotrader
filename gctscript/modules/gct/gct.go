package gct

import (
	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/stdlib"
)

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range GCTModules {
		names = append(names, name)
	}
	return names
}

// GetModuleMap returns the module map that includes all modules
// for the given module names.
func GetModuleMap(names ...string) *objects.ModuleMap {
	modules := objects.NewModuleMap()

	for _, name := range names {
		if mod := GCTModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}
	}

	stdLib := stdlib.AllModuleNames()
	for _, name := range stdLib {
		if mod := stdlib.BuiltinModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}
	}

	return modules
}

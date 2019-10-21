package loader

import (
	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/stdlib"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
)

// GetModuleMap returns the module map that includes all modules
// for the given module names.
func GetModuleMap() *objects.ModuleMap {
	modules := objects.NewModuleMap()

	gctModuleList := gct.AllModuleNames()

	for _, name := range gctModuleList {
		if mod := gct.GCTModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}

	}

	stdLib := stdlib.AllModuleNames()
	for _, name := range stdLib {
		if mod := stdlib.BuiltinModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}
		if mod := stdlib.SourceModules[name]; mod != "" {
			modules.AddSourceModule(name, []byte(mod))
		}
	}

	return modules
}

package ta

import (
	"maps"
	"slices"
)

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	return slices.AppendSeq(make([]string, 0, len(Modules)), maps.Keys(Modules))
}

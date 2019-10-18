package gct

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range GCTModules {
		names = append(names, name)
	}
	return names
}

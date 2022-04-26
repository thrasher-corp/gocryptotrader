package gct

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	names := make([]string, 0, len(Modules))
	for name := range Modules {
		names = append(names, name)
	}
	return names
}

package ta

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range Modules {
		names = append(names, name)
	}
	return names
}

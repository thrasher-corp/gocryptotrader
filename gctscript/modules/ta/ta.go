package ta

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	names := make([]string, 0, len(Modules))
	for x := range Modules {
		names = append(names, x)
	}
	return names
}

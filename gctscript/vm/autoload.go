package vm

// RemoveAutoload remove entry from autoload slice
func RemoveAutoload(name string) {
	for x := range GCTScriptConfig.AutoLoad {
		if GCTScriptConfig.AutoLoad[x] != name {
			continue
		}
	}
}

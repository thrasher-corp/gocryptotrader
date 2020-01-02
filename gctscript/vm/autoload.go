package vm

func RemoveAutoload(name string) {
	for x := range GCTScriptConfig.AutoLoad {
		if GCTScriptConfig.AutoLoad[x] != name {
			continue
		}
		//GCTScriptConfig.AutoLoad =
	}
}

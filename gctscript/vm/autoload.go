package vm

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Autoload remove entry from autoload slice
func Autoload(name string, remove bool) error {
	if filepath.Ext(name) != ".gct" {
		name += ".gct"
	}
	if remove {
		for x := range GCTScriptConfig.AutoLoad {
			if GCTScriptConfig.AutoLoad[x] != name {
				continue
			}
			GCTScriptConfig.AutoLoad = append(GCTScriptConfig.AutoLoad[:x], GCTScriptConfig.AutoLoad[x+1:]...)
			if GCTScriptConfig.Verbose {
				log.Debugf(log.GCTScriptMgr, "Removing script: %s from autoload", name)
			}
			return nil
		}
		return fmt.Errorf("%v - not found", name)
	}

	script := filepath.Join(ScriptPath, name)
	_, err := os.Stat(script)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%v - not found", script)
		}
		return err
	}
	GCTScriptConfig.AutoLoad = append(GCTScriptConfig.AutoLoad, name)
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "Adding script: %s to autoload", name)
	}
	return nil
}

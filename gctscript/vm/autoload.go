package vm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Autoload remove entry from autoload slice
func Autoload(name string, remove bool) error {
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
		return errors.New("script not found")
	}

	var scriptNameWithExtension string
	if name[0:4] != ".gct" {
		scriptNameWithExtension = name + ".gct"
	}
	script := filepath.Join(ScriptPath, scriptNameWithExtension)
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

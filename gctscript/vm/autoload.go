package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Autoload remove entry from autoload slice
func (g *GctScriptManager) Autoload(name string, remove bool) error {
	if filepath.Ext(name) != common.GctExt {
		name += common.GctExt
	}
	if remove {
		for x := range g.config.AutoLoad {
			if g.config.AutoLoad[x] != name {
				continue
			}
			g.config.AutoLoad = append(g.config.AutoLoad[:x], g.config.AutoLoad[x+1:]...)
			if g.config.Verbose {
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
	g.config.AutoLoad = append(g.config.AutoLoad, name)
	if g.config.Verbose {
		log.Debugf(log.GCTScriptMgr, "Adding script: %s to autoload", name)
	}
	return nil
}

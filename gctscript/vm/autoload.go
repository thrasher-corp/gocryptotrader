package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

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
			g.config.AutoLoad = slices.Delete(g.config.AutoLoad, x, x+1)
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

func (g *GctScriptManager) autoLoad() {
	for x := range g.config.AutoLoad {
		temp := g.New()
		if temp == nil {
			log.Errorf(log.GCTScriptMgr, "Unable to create Virtual Machine, autoload failed for: %v",
				g.config.AutoLoad[x])
			continue
		}
		name := g.config.AutoLoad[x]
		if filepath.Ext(name) != common.GctExt {
			name += common.GctExt
		}
		scriptPath := filepath.Join(ScriptPath, name)
		err := temp.Load(scriptPath)
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "%v failed to load: %v", filepath.Base(scriptPath), err)
			err = temp.unregister()
			if err != nil {
				log.Errorf(log.GCTScriptMgr, "%v failed to unregister: %v", filepath.Base(scriptPath), err)
			}
			continue
		}
		go temp.CompileAndRun()
	}
}

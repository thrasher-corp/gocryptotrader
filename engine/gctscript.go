package engine

import (
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const gctscriptManagerName = "gctscript"

type gctScriptManager struct {
	running  atomic.Value
	shutdown chan struct{}
}

// Started returns if gctscript manager subsystem is started
func (g *gctScriptManager) Started() bool {
	return g.running.Load() == true
}

// Start starts gctscript subsystem and creates shutdown channel
func (g *gctScriptManager) Start() error {
	if g.Started() {
		return fmt.Errorf("%s %s", gctscriptManagerName, ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.Global, "%s %s", gctscriptManagerName, MsgSubSystemStarting)
	g.shutdown = make(chan struct{})
	go g.run()
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *gctScriptManager) Stop() error {
	if !g.Started() {
		return fmt.Errorf("%s %s", gctscriptManagerName, ErrSubSystemAlreadyStopped)
	}

	log.Debugf(log.Global, "%s %s", gctscriptManagerName, MsgSubSystemShuttingDown)
	close(g.shutdown)
	err := vm.ShutdownAll()
	if err != nil {
		return err
	}
	return nil
}

func (g *gctScriptManager) run() {
	log.Debugf(log.GCTScriptMgr, "%s %s", gctscriptManagerName, MsgSubSystemStarted)

	Bot.ServicesWG.Add(1)
	g.running.Store(true)
	g.autoLoad()

	defer func() {
		g.running.Store(false)
		Bot.ServicesWG.Done()
		log.Debugf(log.GCTScriptMgr, "%s %s", gctscriptManagerName, MsgSubSystemShutdown)
	}()

	<-g.shutdown
}

func (g *gctScriptManager) autoLoad() {
	for x := range Bot.Config.GCTScript.AutoLoad {
		temp := vm.New()
		if temp == nil {
			log.Errorf(log.GCTScriptMgr, "Unable to create Virtual Machine autoload failed for: %v",
				Bot.Config.GCTScript.AutoLoad[x])
			continue
		}
		scriptPath := filepath.Join(vm.ScriptPath, Bot.Config.GCTScript.AutoLoad[x]+".gct")
		err := temp.Load(scriptPath)
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "%v failed to load: %v", filepath.Base(scriptPath), err)
			continue
		}
		go temp.CompileAndRun()
	}
}

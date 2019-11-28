package engine

import (
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const name = "gctscript"

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
		return fmt.Errorf("%s %s", name, ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.Global, "%s %s", name, MsgSubSystemStarting)
	g.shutdown = make(chan struct{})
	go g.run()
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *gctScriptManager) Stop() error {
	if !g.Started() {
		return fmt.Errorf("%s %s", name, ErrSubSystemAlreadyStarted)
	}

	log.Debugf(log.Global, "%s %s", name, MsgSubSystemShuttingDown)
	close(g.shutdown)
	err := vm.ShutdownAll()
	if err != nil {
		return err
	}
	return nil
}

func (g *gctScriptManager) run() {
	log.Debugf(log.GCTScriptMgr, "%s %s", name, MsgSubSystemStarted)

	Bot.ServicesWG.Add(1)
	g.running.Store(true)
	g.autoLoad()

	defer func() {
		g.running.Store(false)
		Bot.ServicesWG.Done()
		log.Debugf(log.GCTScriptMgr, "%s %s", name, MsgSubSystemShutdown)
	}()

	<-g.shutdown
}

func (g *gctScriptManager) autoLoad() {
	for x := range Bot.Config.GCTScript.AutoLoad {
		temp := vm.New()
		scriptPath := filepath.Join(vm.ScriptPath, Bot.Config.GCTScript.AutoLoad[x]+".gct")
		err := temp.Load(scriptPath)
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "%v failed to load: %v", filepath.Base(scriptPath), err)
			continue
		}
		go temp.CompileAndRun()
	}
}

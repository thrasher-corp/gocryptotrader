package engine

import (
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const gctscriptManagerName = "GCTScript"

type gctScriptManager struct {
	started  int32
	stopped  int32
	shutdown chan struct{}
}

// Started returns if gctscript manager subsystem is started
func (g *gctScriptManager) Started() bool {
	return atomic.LoadInt32(&g.started) == 1
}

// Start starts gctscript subsystem and creates shutdown channel
func (g *gctScriptManager) Start() (err error) {
	if atomic.AddInt32(&g.started, 1) != 1 {
		return fmt.Errorf("%s %s", gctscriptManagerName, ErrSubSystemAlreadyStarted)
	}

	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&g.started, 1, 0)
		}
	}()

	log.Debugln(log.Global, gctscriptManagerName, MsgSubSystemStarting)

	g.shutdown = make(chan struct{})
	go g.run()
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *gctScriptManager) Stop() error {
	if atomic.LoadInt32(&g.started) == 0 {
		return fmt.Errorf("%s %s", gctscriptManagerName, ErrSubSystemNotStarted)
	}

	if atomic.AddInt32(&g.stopped, 1) != 1 {
		return fmt.Errorf("%s %s", gctscriptManagerName, ErrSubSystemAlreadyStopped)
	}

	log.Debugln(log.GCTScriptMgr, gctscriptManagerName, MsgSubSystemShuttingDown)
	close(g.shutdown)
	err := vm.ShutdownAll()
	if err != nil {
		return err
	}
	return nil
}

func (g *gctScriptManager) run() {
	log.Debugln(log.Global, gctscriptManagerName, MsgSubSystemStarted)

	Bot.ServicesWG.Add(1)
	vm.SetDefaultScriptOutput()
	g.autoLoad()
	defer func() {
		atomic.CompareAndSwapInt32(&g.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&g.started, 1, 0)
		Bot.ServicesWG.Done()
		log.Debugln(log.GCTScriptMgr, gctscriptManagerName, MsgSubSystemShutdown)
	}()

	<-g.shutdown
}

func (g *gctScriptManager) autoLoad() {
	for x := range Bot.Config.GCTScript.AutoLoad {
		temp := vm.New()
		if temp == nil {
			log.Errorf(log.GCTScriptMgr, "Unable to create Virtual Machine, autoload failed for: %v",
				Bot.Config.GCTScript.AutoLoad[x])
			continue
		}
		var name = Bot.Config.GCTScript.AutoLoad[x]
		if filepath.Ext(name) != ".gct" {
			name += ".gct"
		}
		scriptPath := filepath.Join(vm.ScriptPath, name)
		err := temp.Load(scriptPath)
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "%v failed to load: %v", filepath.Base(scriptPath), err)
			continue
		}
		go temp.CompileAndRun()
	}
}

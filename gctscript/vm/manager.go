package vm

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const gctscriptManagerName = "GCTScript"

// GctScriptManager loads and runs GCT Tengo scripts
type GctScriptManager struct {
	config   *Config
	started  int32
	stopped  int32
	shutdown chan struct{}
	// Optional values to override stored config ('nil' if not overridden)
	MaxVirtualMachines *uint8
}

// NewManager creates a new instance of script manager
func NewManager(config *Config) (*GctScriptManager, error) {
	if config == nil {
		return nil, errors.New("config must be provided for script manager")
	}
	return &GctScriptManager{
		config: config,
	}, nil
}

// Started returns if gctscript manager subsystem is started
func (g *GctScriptManager) Started() bool {
	return atomic.LoadInt32(&g.started) == 1
}

// Start starts gctscript subsystem and creates shutdown channel
func (g *GctScriptManager) Start(wg *sync.WaitGroup) (err error) {
	if atomic.AddInt32(&g.started, 1) != 1 {
		return fmt.Errorf("%s %s", gctscriptManagerName, subsystem.ErrSubSystemAlreadyStarted)
	}

	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&g.started, 1, 0)
		}
	}()
	log.Debugln(log.Global, gctscriptManagerName, subsystem.MsgSubSystemStarting)

	g.shutdown = make(chan struct{})
	go g.run(wg)
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *GctScriptManager) Stop() error {
	if atomic.LoadInt32(&g.started) == 0 {
		return fmt.Errorf("%s %s", gctscriptManagerName, subsystem.ErrSubSystemNotStarted)
	}

	if atomic.AddInt32(&g.stopped, 1) != 1 {
		return fmt.Errorf("%s %s", gctscriptManagerName, subsystem.ErrSubSystemAlreadyStopped)
	}

	log.Debugln(log.GCTScriptMgr, gctscriptManagerName, subsystem.MsgSubSystemShuttingDown)
	close(g.shutdown)
	err := g.ShutdownAll()
	if err != nil {
		return err
	}
	return nil
}

func (g *GctScriptManager) run(wg *sync.WaitGroup) {
	log.Debugln(log.Global, gctscriptManagerName, subsystem.MsgSubSystemStarted)

	wg.Add(1)
	SetDefaultScriptOutput()
	g.autoLoad()
	defer func() {
		atomic.CompareAndSwapInt32(&g.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&g.started, 1, 0)
		wg.Done()
		log.Debugln(log.GCTScriptMgr, gctscriptManagerName, subsystem.MsgSubSystemShutdown)
	}()

	<-g.shutdown
}

func (g *GctScriptManager) autoLoad() {
	for x := range g.config.AutoLoad {
		temp := g.New()
		if temp == nil {
			log.Errorf(log.GCTScriptMgr, "Unable to create Virtual Machine, autoload failed for: %v",
				g.config.AutoLoad[x])
			continue
		}
		var name = g.config.AutoLoad[x]
		if filepath.Ext(name) != common.GctExt {
			name += common.GctExt
		}
		scriptPath := filepath.Join(ScriptPath, name)
		err := temp.Load(scriptPath)
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "%v failed to load: %v", filepath.Base(scriptPath), err)
			continue
		}
		go temp.CompileAndRun()
	}
}

// GetMaxVirtualMachines returns the max number of VMs to create
func (g *GctScriptManager) GetMaxVirtualMachines() uint8 {
	if g.MaxVirtualMachines != nil {
		return *g.MaxVirtualMachines
	}
	return g.config.MaxVirtualMachines
}

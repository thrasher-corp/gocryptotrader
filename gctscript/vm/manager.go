package vm

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	caseName = "GCTScript"
	// Name is an exported subsystem name
	Name = "gctscript"
)

// GctScriptManager loads and runs GCT Tengo scripts
type GctScriptManager struct {
	config   *Config
	started  int32
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

// IsRunning returns if gctscript manager subsystem is started
func (g *GctScriptManager) IsRunning() bool {
	if g == nil {
		return false
	}
	return atomic.LoadInt32(&g.started) == 1
}

// Start starts gctscript subsystem and creates shutdown channel
func (g *GctScriptManager) Start(wg *sync.WaitGroup) (err error) {
	if !atomic.CompareAndSwapInt32(&g.started, 0, 1) {
		return fmt.Errorf("%s %s", caseName, ErrScriptFailedValidation)
	}
	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&g.started, 1, 0)
		}
	}()

	g.shutdown = make(chan struct{})
	wg.Add(1)
	go g.run(wg)
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *GctScriptManager) Stop() error {
	if atomic.LoadInt32(&g.started) == 0 {
		return fmt.Errorf("%s not running", caseName)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&g.started, 1, 0)
	}()

	err := g.ShutdownAll()
	if err != nil {
		return err
	}
	close(g.shutdown)
	return nil
}

func (g *GctScriptManager) run(wg *sync.WaitGroup) {
	log.Debugf(log.Global, "%s starting", caseName)

	SetDefaultScriptOutput()
	g.autoLoad()
	defer func() {
		wg.Done()
	}()

	<-g.shutdown
}

// GetMaxVirtualMachines returns the max number of VMs to create
func (g *GctScriptManager) GetMaxVirtualMachines() uint8 {
	if g.MaxVirtualMachines != nil {
		return *g.MaxVirtualMachines
	}
	return g.config.MaxVirtualMachines
}

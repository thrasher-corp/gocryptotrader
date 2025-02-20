package vm

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	caseName = "GCTScript"
	// Name is an exported subsystem name
	Name = "gctscript"
)

// ErrNilSubsystem returned when script manager has not been set up
var ErrNilSubsystem = errors.New("gct script has not been set up")

// GctScriptManager loads and runs GCT Tengo scripts
type GctScriptManager struct {
	config   *Config
	started  int32
	shutdown chan struct{}
	// Optional values to override stored config ('nil' if not overridden)
	MaxVirtualMachines *uint64
}

// NewManager creates a new instance of script manager
func NewManager(config *Config) (*GctScriptManager, error) {
	if config == nil {
		return nil, errors.New("config must be provided for script manager")
	}
	return &GctScriptManager{config: config}, nil
}

// IsRunning returns if gctscript manager subsystem is started
func (g *GctScriptManager) IsRunning() bool {
	return g != nil && atomic.LoadInt32(&g.started) == 1
}

// Start starts gctscript subsystem and creates shutdown channel
func (g *GctScriptManager) Start(wg *sync.WaitGroup) (err error) {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	if !atomic.CompareAndSwapInt32(&g.started, 0, 1) {
		return fmt.Errorf("%s %s", caseName, ErrScriptFailedValidation)
	}
	g.shutdown = make(chan struct{})
	wg.Add(1)
	go g.run(wg)
	return nil
}

// Stop stops gctscript subsystem along with all running Virtual Machines
func (g *GctScriptManager) Stop() error {
	if g == nil {
		return fmt.Errorf("%s %w", caseName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&g.started) == 0 {
		return fmt.Errorf("%s not running", caseName)
	}
	defer atomic.CompareAndSwapInt32(&g.started, 1, 0)

	if err := g.ShutdownAll(); err != nil {
		return err
	}
	close(g.shutdown)
	return nil
}

func (g *GctScriptManager) run(wg *sync.WaitGroup) {
	log.Debugf(log.Global, "%s starting", caseName)

	SetDefaultScriptOutput()
	g.autoLoad()
	defer wg.Done()

	<-g.shutdown
}

// GetMaxVirtualMachines returns the max number of VMs to create
func (g *GctScriptManager) GetMaxVirtualMachines() uint64 {
	if g.MaxVirtualMachines != nil {
		return *g.MaxVirtualMachines
	}
	return g.config.MaxVirtualMachines
}

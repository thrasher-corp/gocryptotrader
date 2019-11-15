package vm

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/d5/tengo/script"
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/loader"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

func newVM() *VM {
	newUUID, err := uuid.NewV4()
	if err != nil {
		log.Error(log.GCTScriptMgr, Error{
			Action: "New -> UUID",
			Cause:  err,
		})
		return nil
	}

	if GCTScriptConfig.DebugMode {
		log.Debugln(log.GCTScriptMgr, "New GCTScript VM created")
	}

	return &VM{
		ID:     newUUID,
		Script: pool.Get().(*script.Script),
	}
}

// Load parses and creates a new instance of tengo script vm
func (vm *VM) Load(file string) error {
	if !GCTScriptConfig.Enabled {
		return &Error{
			Cause: ErrScriptingDisabled,
		}
	}

	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "Loading script: %v", file)
	}

	f, err := os.Open(file)
	if err != nil {
		return &Error{
			Action: "Load -> Open",
			Script: file,
			Cause:  err,
		}
	}

	code, err := ioutil.ReadAll(f)
	if err != nil {
		return &Error{
			Action: "Load -> Read",
			Script: file,
			Cause:  err,
		}
	}

	vm.file = f.Name()
	vm.Name = vm.shortName(file)
	vm.Script = script.New(code)
	vm.Script.SetImports(loader.GetModuleMap())

	if GCTScriptConfig.AllowImports {
		if GCTScriptConfig.DebugMode {
			log.Debugf(log.GCTScriptMgr, "file imports enabled for vm: %v", vm.ID)
		}
		vm.Script.EnableFileImport(true)
	}

	return nil
}

// Compile compiles to byte code loaded copy of vm script
func (vm *VM) Compile() (err error) {
	vm.Compiled = new(script.Compiled)
	vm.Compiled, err = vm.Script.Compile()
	return
}

// Run runs byte code
func (vm *VM) Run() (err error) {
	return vm.Compiled.Run()
}

// RunCtx runs compiled byte code with context.Context support.
func (vm *VM) RunCtx() (err error) {
	if vm.ctx == nil {
		vm.ctx = context.Background()
	}

	ct, cancel := context.WithTimeout(vm.ctx, GCTScriptConfig.ScriptTimeout)
	defer cancel()

	return vm.Compiled.RunContext(ct)
}

// CompileAndRun Compile and Run script
func (vm *VM) CompileAndRun() (err error) {
	err = vm.Compile()
	if err != nil {
		return
	}

	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "Running script: %v", vm.ID)
	}

	err = vm.RunCtx()
	if err != nil {
		return err
	}

	if vm.Compiled.Get("timer").String() != "" {
		vm.T, err = time.ParseDuration(vm.Compiled.Get("timer").String())
		if err != nil {
			return err
		}
		if vm.T < time.Nanosecond {
			return errors.New("repeat timer cannot be under 1 nano second")
		}
		vm.runner()
	} else {
		return vm.Shutdown()
	}

	return err
}

// Shutdown shuts down current VM
func (vm *VM) Shutdown() error {
	if vm.S != nil {
		vm.S <- struct{}{}
		close(vm.S)
	}
	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "Shutting script: %v", vm.ID)
	}
	return RemoveVM(vm.ID)
}

func (vm *VM) Read() ([]byte, error) {
	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "Read script: %v", vm.ID)
	}
	return ioutil.ReadFile(vm.file)
}

func (vm *VM) shortName(file string) string {
	if file[0] == '.' {
		file = file[2:]
	}

	return filepath.Base(file)
}

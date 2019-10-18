package vm

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/d5/tengo/script"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/loader"
)

func newVM() *VM {
	return &VM{
		Script: VMPool.Get().(*script.Script),
	}
}

func TemrinateAllVM() error {
	return nil
}

// Load parses and creates a new instance of tengo script vm
func (vm *VM) Load(file string) error {
	if !GCTScriptConfig.Enabled {
		return &Error{
			Cause: ErrScriptingDisabled,
		}
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

	vm.Script = script.New(code)
	vm.Script.SetImports(loader.GetModuleMap())

	if GCTScriptConfig.AllowImports {

		vm.Script.EnableFileImport(true)
	}

	return nil
}

// Compile compiles to byte code loaded copy of vm script
func (vm *VM) Compile() (err error) {
	if vm == nil {
		return &Error{
			Action: "Compile",
			Cause:  ErrNoVMLoaded,
		}
	}

	vm.Compiled = new(script.Compiled)
	vm.Compiled, err = vm.Script.Compile()

	return
}

// Run runs byte code
func (vm *VM) Run() (err error) {
	if vm == nil {
		return &Error{
			Action: "Run",
			Cause:  ErrNoVMLoaded,
		}
	}
	return vm.Compiled.Run()
}

func (vm *VM) RunCtx() (err error) {
	if vm == nil {
		return &Error{
			Action: "RunCtx",
			Cause:  ErrNoVMLoaded,
		}
	}

	if vm.ctx == nil {
		vm.ctx = context.Background()
	}

	err = vm.Compiled.RunContext(vm.ctx)
	return
}

// CompileAndRun Compile and Run script
func (vm *VM) CompileAndRun() (err error) {
	err = vm.Compile()
	if err != nil {
		return
	}

	err = vm.Run()
	if err != nil {
		return err
	}

	if vm.Compiled.Get("name") != nil {
		vm.name = vm.Compiled.Get("name").String()
	}

	if vm.Compiled.Get("timer").String() != "" {
		vm.t, err = time.ParseDuration(vm.Compiled.Get("timer").String())
		if err != nil {
			return err
		}
		if vm.t < time.Nanosecond {
			return errors.New("repeat timer cannot be under 1 nano second")
		}
		vm.addTask()
	}

	return err
}

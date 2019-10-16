package gctscript

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/d5/tengo/stdlib"

	"github.com/d5/tengo/script"
)

// New returns a new instance of VM
func New() *VM {
	return &VM{
		Script: new(script.Script),
	}
}

// Load parses and creates a new instance of tengo script vm
func (vm *VM) Load(file string) error {
	if !GCTScriptConfig.Enabled {
		return &VMError{
			Cause: ErrScriptingDisabled,
		}
	}
	f, err := os.Open(file)
	if err != nil {
		return &VMError{
			Action: "Load -> Open",
			Script: file,
			Cause:  err,
		}
	}

	code, err := ioutil.ReadAll(f)
	if err != nil {
		return &VMError{
			Action: "Load -> Read",
			Script: file,
			Cause:  err,
		}
	}

	vm.Script = script.New(code)

	if GCTScriptConfig.AllowImports {
		vm.Script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
		vm.Script.EnableFileImport(true)
	}

	return nil
}

// Compile compiles to byte code loaded copy of vm script
func (vm *VM) Compile() (err error) {
	if vm == nil {
		return errors.New("vm: no Virtual machine loaded")
	}
	vm.Compiled = new(script.Compiled)

	vm.Compiled, err = vm.Script.Compile()

	return
}

// Run runs byte code
func (vm *VM) Run() (err error) {
	err = vm.Compiled.Run()
	return
}

// CompileAndRun Compile and Run script
func (vm *VM) CompileAndRun() (err error) {
	err = vm.Compile()
	if err != nil {
		return
	}
	return vm.Run()
}

func (e VMError) Error() string {
	var scriptName, action string
	if e.Script != "" {
		scriptName = fmt.Sprintf("(SCRIPT) %s ", filepath.Base(e.Script))
	}

	if e.Action != "" {
		action = fmt.Sprintf("(ACTION) %s ", e.Action)
	}

	return fmt.Sprintf("%s: %s%s%s", gctScript, action, scriptName, e.Cause)
}

func (e VMError) Unwrap() error {
	return e.Cause
}

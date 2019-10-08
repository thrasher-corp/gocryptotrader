package core

import (
	"errors"
	"io/ioutil"
	"os"

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
	f, err := os.Open(file)
	if err != nil {

		return err
	}
	code, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	vm.Script = script.New(code)
	vm.Script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	vm.Script.EnableFileImport(true)

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

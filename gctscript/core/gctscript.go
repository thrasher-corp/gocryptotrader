package core

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/d5/tengo/script"
)

func New() *VM {
	return &VM{
		Script: new(script.Script),
	}
}

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
	return nil
}

func (vm *VM) Compile() (err error) {
	if vm == nil {
		return errors.New("vm: no Virtual machine loaded")
	}
	vm.Compiled = new(script.Compiled)

	vm.Compiled, err = vm.Script.Compile()

	return
}

func (vm *VM) Run() (err error) {
	err = vm.Compiled.Run()
	return
}

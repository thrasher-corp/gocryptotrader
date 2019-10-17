package vm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/d5/tengo/script"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
)

// New returns a new instance of VM
func New() *VM {
	vm := newVM()
	VMList = append(VMList, vm)

	return vm
}

func newVM() *VM {
	return &VM{
		Script: VMPool.Get().(*script.Script),
	}
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
	vm.Script.SetImports(gct.GetModuleMap(gct.AllModuleNames()...))

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

	if vm.Compiled.Get("timer") != nil {
		v := vmtask{
			name:    vm.name,
			nextRun: time.Now(),
		}
		addTask(v)
		vm.timer = time.Unix(vm.Compiled.Get("timer").Int64(), 0)
	}

	return
}

func TemrinateAllVM() error {
	for x := range VMList {
		VMList[x].ctx.Done()
	}
	return nil
}

func RunVMTasks() error {
	for x := range scheduledItem {
		fmt.Println(scheduledItem[x].name)
		scheduledItem = scheduledItem[:x+copy(scheduledItem[x:], scheduledItem[x+1:])]

	}
	return nil
}

func runTask() {

}

func addTask(v vmtask) {
	scheduledItem = append(scheduledItem, v)
}

func (e Error) Error() string {
	var scriptName, action string
	if e.Script != "" {
		scriptName = fmt.Sprintf("(SCRIPT) %s ", filepath.Base(e.Script))
	}

	if e.Action != "" {
		action = fmt.Sprintf("(ACTION) %s ", e.Action)
	}

	return fmt.Sprintf("%s: %s%s%s", gctScript, action, scriptName, e.Cause)
}

func (e Error) Unwrap() error {
	return e.Cause
}

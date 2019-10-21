package vm

import (
	"time"
)

func (vm *VM) addTask() {
	time.Sleep(vm.t)
	close(vm.c)
}

func (vm *VM) runner() {
	vm.c = make(chan struct{})

	go func() {
		for {
			select {
			case <-vm.c:
				err := vm.CompileAndRun()
				if err != nil {
					return
				}
				return
			}
		}
	}()
}

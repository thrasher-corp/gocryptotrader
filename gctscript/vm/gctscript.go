package vm

import "fmt"

// New returns a new instance of VM
func New() *VM {
	vm := newVM()
	VMList = append(VMList, *vm)

	return vm
}

func Running() {
	for x := range VMList {
		fmt.Println(VMList[x].name)
	}
}

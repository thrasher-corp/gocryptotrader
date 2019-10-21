package vm

// New returns a new instance of VM
func New() *VM {
	vm := newVM()
	VMList = append(VMList, *vm)

	return vm
}

func ShutdownAll() error {
	for x := range VMList {
		_ = VMList[x].Shutdown()
	}
	return nil
}

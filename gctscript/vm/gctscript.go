package vm

import (
	"github.com/gofrs/uuid"
)

// New returns a new instance of VM
func New() *VM {
	vm := newVM()
	if AllVMs == nil {
		AllVMs = make(map[uuid.UUID]*VM)
	}
	AllVMs[vm.ID] = vm
	return vm
}

// ShutdownAll shutdown all
func ShutdownAll() error {
	for x := range AllVMs {
		_ = AllVMs[x].Shutdown()
	}
	return nil
}

// RemoveVM remove VM from list
func RemoveVM(id uuid.UUID) error {
	if _, f := AllVMs[id]; !f {
		return ErrNoVMFound
	}
	delete(AllVMs, id)
	return nil
}

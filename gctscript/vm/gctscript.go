package vm

import (
	"errors"

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
		AllVMs[x].S <- struct{}{}
	}
	return nil
}

// RemoveVM remove VM from list
func RemoveVM(id uuid.UUID) error {
	if _, f := AllVMs[id]; !f {
		return errors.New("no VM found")
	}
	delete(AllVMs, id)
	return nil
}

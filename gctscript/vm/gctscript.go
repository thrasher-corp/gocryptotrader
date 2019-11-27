package vm

import (
	"fmt"

	"github.com/gofrs/uuid"

	log "github.com/thrasher-corp/gocryptotrader/logger"
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
func ShutdownAll() (err error) {
	if GCTScriptConfig.DebugMode {
		log.Debugln(log.GCTScriptMgr, "Shutting down all Virtual Machines")
	}

	var errors []error
	for x := range AllVMs {
		err = AllVMs[x].Shutdown()
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		err = fmt.Errorf("failed to shutdown the follow Virtual Machines: %v", errors)
	}

	return
}

// RemoveVM remove VM from list
func RemoveVM(id uuid.UUID) error {
	if _, f := AllVMs[id]; !f {
		return ErrNoVMFound
	}

	delete(AllVMs, id)
	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

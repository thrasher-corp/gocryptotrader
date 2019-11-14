package vm

import (
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
func ShutdownAll() error {
	if GCTScriptConfig.DebugMode {
		log.Debugln(log.GCTScriptMgr, "Shutting down all Virtual Machines")
	}
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
	if GCTScriptConfig.DebugMode {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

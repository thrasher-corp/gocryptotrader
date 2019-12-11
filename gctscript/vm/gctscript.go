package vm

import (
	"fmt"

	"github.com/gofrs/uuid"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// New returns a new instance of VM
func New() *VM {
	if AllVMs == nil {
		AllVMs = make(map[uuid.UUID]*VM)
	}

	if len(AllVMs) >= int(GCTScriptConfig.MaxVirtualMachines) {
		log.Warnln(log.GCTScriptMgr, "GCTScript MaxVirtualMachines hit unable to start further instances")
		return nil
	}

	vm := newVM()
	AllVMs[vm.ID] = vm
	return vm
}

func Validate(file string) (err error) {
	tempVM := newVM()
	err = tempVM.Load(file)
	if err != nil {
		return
	}
	return tempVM.Compile()
}

// ShutdownAll shutdown all
func ShutdownAll() (err error) {
	if GCTScriptConfig.Verbose {
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

	return err
}

// RemoveVM remove VM from list
func RemoveVM(id uuid.UUID) error {
	if _, f := AllVMs[id]; !f {
		return ErrNoVMFound
	}

	delete(AllVMs, id)
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

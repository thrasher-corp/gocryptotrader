package vm

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// New returns a new instance of VM
func New() *VM {
	if AllVMs == nil {
		AllVMs = make(map[uuid.UUID]*VM)
	}

	if len(AllVMs) >= int(GCTScriptConfig.MaxVirtualMachines) {
		log.Warnf(log.GCTScriptMgr, "GCTScript MaxVirtualMachines (%v) hit, unable to start further instances",
			GCTScriptConfig.MaxVirtualMachines)
		return nil
	}

	vm := NewVM()
	AllVMs[vm.ID] = vm
	return vm
}

// Validate will attempt to execute a script in a test/non-live environment
// to confirm it passes requirements for execution
func Validate(file string) (err error) {
	defer func() {
		validator.IsTestExecution = false
		validator.RWValidatorLock.Unlock()
	}()
	validator.RWValidatorLock.Lock()
	validator.IsTestExecution = true
	tempVM := NewVM()
	err = tempVM.Load(file)
	if err != nil {
		return
	}
	err = tempVM.Compile()
	if err != nil {
		return
	}
	return tempVM.Run()
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
		err = fmt.Errorf("failed to shutdown the following Virtual Machines: %v", errors)
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

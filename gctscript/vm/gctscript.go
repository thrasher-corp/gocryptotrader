package vm

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// New returns a new instance of VM
func New() *VM {
	if VMSCount.Len() >= int32(GCTScriptConfig.MaxVirtualMachines) {
		if GCTScriptConfig.Verbose {
			log.Warnf(log.GCTScriptMgr, "GCTScript MaxVirtualMachines (%v) hit, unable to start further instances",
				GCTScriptConfig.MaxVirtualMachines)
		}
		return nil
	}
	VMSCount.add()
	vm := NewVM()
	if vm == nil {
		VMSCount.remove()
	} else {
		AllVMSync.Store(vm.ID, vm)
	}
	return vm
}

// Validate will attempt to execute a script in a test/non-live environment
// to confirm it passes requirements for execution
func Validate(file string) (err error) {
	validator.IsTestExecution.Store(true)
	defer validator.IsTestExecution.Store(false)
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
	AllVMSync.Range(func(k, v interface{}) bool {
		errShutdown := v.(*VM).Shutdown()
		if err != nil {
			errors = append(errors, errShutdown)
		}
		return true
	})

	if len(errors) > 0 {
		err = fmt.Errorf("failed to shutdown the following Virtual Machines: %v", errors)
	}

	return err
}

// RemoveVM remove VM from list
func RemoveVM(id uuid.UUID) error {
	if _, f := AllVMSync.Load(id); !f {
		return fmt.Errorf(ErrNoVMFound, id.String())
	}

	AllVMSync.Delete(id)
	VMSCount.remove()
	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

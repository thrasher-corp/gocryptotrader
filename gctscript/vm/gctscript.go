package vm

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// New returns a new instance of VM
func New() *VM {
	if len(AllVMs) >= int(GCTScriptConfig.MaxVirtualMachines) {
		if GCTScriptConfig.Verbose {
			log.Warnf(log.GCTScriptMgr, "GCTScript MaxVirtualMachines (%v) hit, unable to start further instances",
				GCTScriptConfig.MaxVirtualMachines)
		}
		return nil
	}

	vm := NewVM()
	storeVM(vm.ID, vm)
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
	_, err := loadVM(id);
	if err != nil{
		return fmt.Errorf(ErrNoVMFound, id.String())
	}
	deleteVM(id)

	if GCTScriptConfig.Verbose {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

func loadVM(id uuid.UUID) (*VM, error) {
	rmw.RLock()
	defer rmw.RUnlock()
	if _, f := AllVMs[id]; !f {
		return nil, fmt.Errorf(ErrNoVMFound, id.String())
	}
	return AllVMs[id], nil
}

func storeVM(k uuid.UUID,v *VM) {
	rmw.Lock()
	defer rmw.Unlock()
	AllVMs[k] = v
}

func deleteVM(id uuid.UUID) {
	rmw.Lock()
	defer rmw.Unlock()
	delete(AllVMs, id)
}
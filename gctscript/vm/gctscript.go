package vm

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new instance of VM
func (g *GctScriptManager) New() *VM {
	if VMSCount.Len() >= g.GetMaxVirtualMachines() {
		if g.config.Verbose {
			log.Warnf(log.GCTScriptMgr, "GCTScript MaxVirtualMachines (%v) hit, unable to start further instances",
				g.GetMaxVirtualMachines())
		}
		return nil
	}
	VMSCount.add()
	vm := g.NewVM()
	if vm == nil {
		VMSCount.remove()
	} else {
		AllVMSync.Store(vm.ID, vm)
	}
	return vm
}

// Validate will attempt to execute a script in a test/non-live environment
// to confirm it passes requirements for execution
func (g *GctScriptManager) Validate(file string) (err error) {
	validator.IsTestExecution.Store(true)
	defer validator.IsTestExecution.Store(false)
	tempVM := g.NewVM()
	err = tempVM.Load(file)
	if err != nil {
		return
	}
	err = tempVM.Compile()
	if err != nil {
		return
	}
	return tempVM.RunCtx()
}

// ShutdownAll shutdown all
func (g *GctScriptManager) ShutdownAll() (err error) {
	if g.config.Verbose {
		log.Debugln(log.GCTScriptMgr, "Shutting down all Virtual Machines")
	}

	var shutdownErrors []error
	AllVMSync.Range(func(_, v any) bool {
		vm, ok := v.(*VM)
		if !ok {
			shutdownErrors = append(shutdownErrors, common.GetTypeAssertError("*VM", v))
			return true
		}
		errShutdown := vm.Shutdown()
		if err != nil {
			shutdownErrors = append(shutdownErrors, errShutdown)
		}
		return true
	})

	if len(shutdownErrors) > 0 {
		err = fmt.Errorf("failed to shutdown the following Virtual Machines: %v", shutdownErrors)
	}

	return err
}

// RemoveVM remove VM from list
func (g *GctScriptManager) RemoveVM(id uuid.UUID) error {
	if _, ok := AllVMSync.Load(id); !ok {
		return fmt.Errorf(ErrNoVMFound, id.String())
	}

	AllVMSync.Delete(id)
	VMSCount.remove()
	if g.config.Verbose {
		log.Debugf(log.GCTScriptMgr, "VM %v removed from AllVMs", id)
	}
	return nil
}

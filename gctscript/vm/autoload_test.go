package vm

import (
	"testing"
)

func TestGctScriptManagerAutoLoadNonExisting(t *testing.T) {
	var vms uint64 = 1
	g := &GctScriptManager{
		config: &Config{
			AutoLoad: []string{"non-existing"},
		},
		MaxVirtualMachines: &vms,
	}
	g.started.Store(true)
	g.autoLoad()
	if VMSCount.Len() != 0 {
		t.Errorf("Expected no VMs, got %v", VMSCount.Len())
	}
}

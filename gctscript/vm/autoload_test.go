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
		started:            1,
		MaxVirtualMachines: &vms,
	}
	g.autoLoad()
	if VMSCount != 0 {
		t.Errorf("Expected no VMs, got %v", VMSCount)
	}
}

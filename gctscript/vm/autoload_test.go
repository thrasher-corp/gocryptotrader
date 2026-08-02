package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Zero(t, VMSCount.Len(), "autoLoad should not register virtual machines for missing scripts")
}

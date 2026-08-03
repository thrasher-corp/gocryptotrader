package vm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxTestVirtualMachines    uint64 = 30
	testVirtualMachineTimeout        = time.Minute
	scriptName                       = "1D01TH0RS3.gct"
)

var (
	testScript               = filepath.Join("..", "..", "testdata", "gctscript", "once.gct")
	testInvalidScript        = filepath.Join("..", "..", "testdata", "gctscript", "invalid.gct")
	testBrokenScript         = filepath.Join("..", "..", "testdata", "gctscript", "broken.gct")
	testScriptRunner         = filepath.Join("..", "..", "testdata", "gctscript", "timer.gct")
	testScriptRunner1s       = filepath.Join("..", "..", "testdata", "gctscript", "1s_timer.gct")
	testScriptRunnerNegative = filepath.Join("..", "..", "testdata", "gctscript", "negative_timer.gct")
	testScriptRunnerInvalid  = filepath.Join("..", "..", "testdata", "gctscript", "invalid_timer.gct")
)

func TestNewVM(t *testing.T) {
	manager := GctScriptManager{
		config: configHelper(true, true, maxTestVirtualMachines),
	}
	assert.Nil(t, manager.New(), "New should not create a VM when manager not started")
	manager.started = 1
	assert.NotNil(t, manager.New(), "New should create a VM when manager is started")
}

func TestVMLoad(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	assert.NoError(t, testVM.Load(testScript), "Load should accept the valid script")

	testScript = testScript[0 : len(testScript)-4]
	testVM = manager.New()
	assert.NoError(t, testVM.Load(testScript), "Load should accept the script without an extension")

	manager.config = configHelper(false, false, maxTestVirtualMachines)
	assert.NoError(t, testVM.Load(testScript), "Load should accept the extensionless script on repeat")
}

func TestVMLoad1s(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunner1s), "Load must accept the one-second timer script")

	testVM.CompileAndRun()
	assert.NoError(t, testVM.Shutdown(), "Shutdown should stop the one-second timer script")
}

func TestVMLoadNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunnerNegative), "Load must accept the negative-timer script")

	testVM.CompileAndRun()
	assert.Error(t, testVM.Shutdown(), "Shutdown should report the negative timer")
}

func TestVMLoadNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	assert.NoError(t, testVM.Load(testScript), "Load should accept the valid script")

	testVM = nil
	assert.ErrorIs(t, testVM.Load(testScript), ErrNoVMLoaded, "Load should reject a nil virtual machine")
}

func TestCompileAndRunNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmcount := VMSCount.Len()
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript), "Load must accept the valid script")

	assert.NoError(t, testVM.Load(testScript), "Load should remain idempotent for the valid script")

	testVM = nil
	testVM.CompileAndRun()
	assert.ErrorIs(t, testVM.Shutdown(), ErrNoVMLoaded, "Shutdown should reject a nil virtual machine")
	assert.Equal(t, vmcount+1, VMSCount.Len(), "CompileAndRun should preserve the registered virtual-machine count after nil reassignment")
}

func TestVMLoadNoFile(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	assert.ErrorIs(t, testVM.Load("missing file"), os.ErrNotExist, "Load should return the missing-file error")
}

func TestVMCompile(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScript)
	require.NoError(t, err, "Load must accept the valid script")

	err = testVM.Compile()
	assert.NoError(t, err, "Compile should compile the valid script")
}

func TestVMRun(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.NewVM()
	err := testVM.Load(testScript)
	require.NoError(t, err, "Load must accept the valid script")

	err = testVM.Compile()
	require.NoError(t, err, "Compile must compile the valid script")

	err = testVM.RunCtx()
	assert.NoError(t, err, "RunCtx should run the compiled script")
}

func TestVMRunTX(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.NewVM()
	err := testVM.Load(testScript)
	require.NoError(t, err, "Load must accept the valid script")

	err = testVM.Compile()
	require.NoError(t, err, "Compile must compile the valid script")

	err = testVM.RunCtx()
	assert.NoError(t, err, "RunCtx should run the compiled transaction script")
}

func TestVMWithRunner(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	require.NotNil(t, VM, "New must allocate a virtual machine")
	err := VM.Load(testScriptRunner)
	require.NoError(t, err, "Load must accept the timer script")
	assert.Equal(t, vmCount+1, VMSCount.Len(), "New should increment the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	assert.NoError(t, err, "Shutdown should stop the timer script")
	assert.Equal(t, vmCount, VMSCount.Len(), "Shutdown should restore the virtual-machine count")
}

func TestVMWithRunnerOnce(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	require.NotNil(t, VM, "New must allocate a virtual machine")
	err := VM.Load(testScript)
	require.NoError(t, err, "Load must accept the run-once script")
	assert.Equal(t, vmCount+1, VMSCount.Len(), "New should increment the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	assert.Error(t, err, "Shutdown should report the completed run-once script")
	assert.Equal(t, vmCount, VMSCount.Len(), "CompileAndRun should restore the virtual-machine count after a run-once script")
}

func TestVMWithRunnerNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	require.NotNil(t, VM, "New must allocate a virtual machine")
	err := VM.Load(testScriptRunnerNegative)
	require.NoError(t, err, "Load must accept the negative-timer script")
	assert.Equal(t, vmCount+1, VMSCount.Len(), "New should increment the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	assert.Error(t, err, "Shutdown should report the negative timer")
	assert.Equal(t, vmCount, VMSCount.Len(), "Shutdown should restore the virtual-machine count")
}

func TestVMWithRunnerInvalidTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	require.NotNil(t, VM, "New must allocate a virtual machine")
	err := VM.Load(testScriptRunnerInvalid)
	require.NoError(t, err, "Load must accept the invalid-timer script")
	assert.Equal(t, vmCount+1, VMSCount.Len(), "New should increment the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	assert.Error(t, err, "Shutdown should report the invalid timer")
	assert.Equal(t, vmCount, VMSCount.Len(), "Shutdown should restore the virtual-machine count")
}

func TestShutdownAll(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	err := VM.Load(testScriptRunner)
	require.NoError(t, err, "Load must accept the timer script")

	VM.CompileAndRun()

	assert.Equal(t, vmCount+1, VMSCount.Len(), "New should increment the virtual-machine count")
	err = manager.ShutdownAll()
	assert.NoError(t, err, "ShutdownAll should stop every virtual machine")

	assert.Zero(t, VMSCount.Len(), "ShutdownAll should clear the virtual-machine count")
}

func TestRead(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	VM := manager.NewVM()
	err := VM.Load(testScriptRunner)
	require.NoError(t, err, "Load must accept the timer script")

	ScriptPath = filepath.Join("..", "..", "testdata", "gctscript")
	data, err := VM.Read()
	require.NoError(t, err, "Read must read the loaded script")
	assert.NotEmpty(t, data, "Read should return script data")
	_ = VM.Shutdown()
}

func TestRemoveVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	id, _ := uuid.FromString("6f20c907-64a0-48f2-848a-7837dee61672")
	err := manager.RemoveVM(id)
	assert.EqualError(t, err, "VM 6f20c907-64a0-48f2-848a-7837dee61672 not found", "RemoveVM should reject an unknown virtual machine")
}

func TestError_Error(t *testing.T) {
	x := Error{
		Script: "noscript.gct",
		Action: "test",
		Cause:  errors.New("HELLO ERROR"),
	}

	assert.Equal(t, "GCT Script: (ACTION) test (SCRIPT) noscript.gct HELLO ERROR", x.Error(), "Error should format the script failure")
}

func TestVM_CompileInvalid(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testInvalidScript)
	require.NoError(t, err, "Load must accept the invalid-runtime script")

	err = testVM.Compile()
	require.NoError(t, err, "Compile must compile the invalid-runtime script")
	err = testVM.RunCtx()
	assert.Error(t, err, "RunCtx should reject the invalid-runtime script")

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	require.NoError(t, err, "Load must accept the invalid-runtime script on repeat")

	err = testVM.Compile()
	require.NoError(t, err, "Compile must compile the invalid-runtime script on repeat")

	err = testVM.RunCtx()
	assert.Error(t, err, "RunCtx should reject the invalid-runtime script on repeat")

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	require.NoError(t, err, "Load must accept the invalid-runtime script for asynchronous execution")

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	assert.Error(t, err, "Shutdown should report the invalid-runtime script")
}

func TestVM_CompileBroken(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testBrokenScript)
	require.NoError(t, err, "Load must accept the syntactically broken script")

	err = testVM.Compile()
	assert.Error(t, err, "Compile should reject the syntactically broken script")
}

func TestVM_CompileAndRunBroken(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testBrokenScript)
	require.NoError(t, err, "Load must accept the syntactically broken script")

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	assert.Error(t, err, "Shutdown should report the broken virtual machine")
}

func TestValidate(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	err := manager.Validate(testBrokenScript)
	assert.Error(t, err, "Validate should reject the broken script")
	err = manager.Validate(testScript)
	assert.NoError(t, err, "Validate should accept the valid script")
}

func TestVMLimit(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, false, 0),
		started: 1,
	}
	assert.Nil(t, manager.New(), "New should enforce the virtual-machine limit")
}

func TestAutoload(t *testing.T) {
	manager := GctScriptManager{
		config: &Config{
			Enabled: true,
			AutoLoad: []string{
				scriptName,
			},
			Verbose: true,
		},
	}

	ScriptPath = filepath.Join("..", "..", "testdata", "gctscript")
	err := manager.Autoload(scriptName, true)
	require.NoError(t, err, "Autoload must remove the configured script")
	err = manager.Autoload(scriptName, true)
	assert.Error(t, err, "Autoload should reject removing an unlisted script")
	err = manager.Autoload("once", false)
	assert.NoError(t, err, "Autoload should add the extensionless script")
	err = manager.Autoload(scriptName, false)
	assert.Error(t, err, "Autoload should reject adding a missing script")
}

func TestVMCount(t *testing.T) {
	var c vmscount
	c.add()
	assert.Equal(t, uint64(1), c.Len(), "add should increment the virtual-machine count")
	c.remove()
	assert.Zero(t, c.Len(), "remove should decrement the virtual-machine count")
}

func configHelper(enabled, imports bool, maxVMs uint64) *Config {
	return &Config{
		Enabled:            enabled,
		AllowImports:       imports,
		ScriptTimeout:      testVirtualMachineTimeout,
		MaxVirtualMachines: maxVMs,
		Verbose:            true,
	}
}

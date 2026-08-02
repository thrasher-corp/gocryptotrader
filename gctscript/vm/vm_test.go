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
	require.Nil(t, manager.New(), "New must not create a VM when manager not started")
	manager.started = 1
	require.NotNil(t, manager.New(), "New must create a VM when manager is started")
}

func TestVMLoad(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript), "Load must accept the valid script")

	testScript = testScript[0 : len(testScript)-4]
	testVM = manager.New()
	require.NoError(t, testVM.Load(testScript), "Load must accept the script without an extension")

	manager.config = configHelper(false, false, maxTestVirtualMachines)
	require.NoError(t, testVM.Load(testScript), "Load must accept the script when imports are disabled")
}

func TestVMLoad1s(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunner1s), "Load must accept the one-second timer script")

	testVM.CompileAndRun()
	require.NoError(t, testVM.Shutdown(), "Shutdown must stop the one-second timer script")
}

func TestVMLoadNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunnerNegative), "Load must accept the negative-timer script")

	testVM.CompileAndRun()
	require.Error(t, testVM.Shutdown(), "Shutdown must report the negative timer")
}

func TestVMLoadNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript), "Load must accept the valid script")

	testVM = nil
	require.ErrorIs(t, testVM.Load(testScript), ErrNoVMLoaded, "Load must reject a nil virtual machine")
}

func TestCompileAndRunNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmcount := VMSCount.Len()
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript), "Load must accept the valid script")

	require.NoError(t, testVM.Load(testScript), "Load must remain idempotent for the valid script")

	testVM = nil
	testVM.CompileAndRun()
	require.ErrorIs(t, testVM.Shutdown(), ErrNoVMLoaded, "Shutdown must reject a nil virtual machine")
	assert.NotEqual(t, vmcount-1, VMSCount.Len(), "CompileAndRun should not decrement the virtual-machine count after nil reassignment")
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
	require.NoError(t, err, "Compile must compile the valid script")
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
	require.NoError(t, err, "RunCtx must run the compiled script")
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
	require.NoError(t, err, "RunCtx must run the compiled transaction script")
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
	require.NotEqual(t, vmCount, VMSCount.Len(), "New must increase the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	require.NoError(t, err, "Shutdown must stop the timer script")
	require.NotEqual(t, vmCount-1, VMSCount.Len(), "Shutdown must update the virtual-machine count")
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
	require.NotEqual(t, vmCount, VMSCount.Len(), "New must increase the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	require.Error(t, err, "Shutdown must report the completed run-once script")
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
	require.NotEqual(t, vmCount, VMSCount.Len(), "New must increase the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	require.Error(t, err, "Shutdown must report the negative timer")
	require.NotEqual(t, vmCount-1, VMSCount.Len(), "Shutdown must update the virtual-machine count")
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
	require.NotEqual(t, vmCount, VMSCount.Len(), "New must increase the virtual-machine count")
	VM.CompileAndRun()
	err = VM.Shutdown()
	require.Error(t, err, "Shutdown must report the invalid timer")
	require.NotEqual(t, vmCount-1, VMSCount.Len(), "Shutdown must update the virtual-machine count")
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

	require.NotEqual(t, vmCount, VMSCount.Len(), "New must increase the virtual-machine count")
	err = manager.ShutdownAll()
	require.NoError(t, err, "ShutdownAll must stop every virtual machine")

	require.NotEqual(t, vmCount-1, VMSCount.Len(), "ShutdownAll must update the virtual-machine count")
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
	require.NotEmpty(t, data, "Read must return script data")
	_ = VM.Shutdown()
}

func TestRemoveVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	id, _ := uuid.FromString("6f20c907-64a0-48f2-848a-7837dee61672")
	err := manager.RemoveVM(id)
	require.EqualError(t, err, "VM 6f20c907-64a0-48f2-848a-7837dee61672 not found", "RemoveVM must reject an unknown virtual machine")
}

func TestError_Error(t *testing.T) {
	x := Error{
		Script: "noscript.gct",
		Action: "test",
		Cause:  errors.New("HELLO ERROR"),
	}

	require.Equal(t, "GCT Script: (ACTION) test (SCRIPT) noscript.gct HELLO ERROR", x.Error(), "Error must format the script failure")
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
	require.Error(t, err, "RunCtx must reject the invalid-runtime script")

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	require.NoError(t, err, "Load must accept the invalid-runtime script on repeat")

	err = testVM.Compile()
	require.NoError(t, err, "Compile must compile the invalid-runtime script on repeat")

	err = testVM.RunCtx()
	require.Error(t, err, "RunCtx must reject the invalid-runtime script on repeat")

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	require.NoError(t, err, "Load must accept the invalid-runtime script for asynchronous execution")

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	require.Error(t, err, "Shutdown must report the invalid-runtime script")
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
	require.Error(t, err, "Compile must reject the syntactically broken script")
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
	require.Error(t, err, "Shutdown must report the broken virtual machine")
}

func TestValidate(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	err := manager.Validate(testBrokenScript)
	require.Error(t, err, "Validate must reject the broken script")
	err = manager.Validate(testScript)
	require.NoError(t, err, "Validate must accept the valid script")
}

func TestVMLimit(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, false, 0),
		started: 1,
	}
	require.Nil(t, manager.New(), "New must enforce the virtual-machine limit")
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
	require.NoError(t, err, "Autoload must load the named script")
	err = manager.Autoload(scriptName, true)
	require.Error(t, err, "Autoload must reject a duplicate named script")
	err = manager.Autoload("once", false)
	require.NoError(t, err, "Autoload must load the extensionless script")
	err = manager.Autoload(scriptName, false)
	require.Error(t, err, "Autoload must reject the missing extensionless script")
}

func TestVMCount(t *testing.T) {
	var c vmscount
	c.add()
	require.Equal(t, uint64(1), c.Len(), "add must increment the virtual-machine count")
	c.remove()
	require.Zero(t, c.Len(), "remove must decrement the virtual-machine count")
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

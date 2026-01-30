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
	require.NoError(t, testVM.Load(testScript))

	testScript = testScript[0 : len(testScript)-4]
	testVM = manager.New()
	require.NoError(t, testVM.Load(testScript))

	manager.config = configHelper(false, false, maxTestVirtualMachines)
	require.NoError(t, testVM.Load(testScript))
}

func TestVMLoad1s(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunner1s))

	testVM.CompileAndRun()
	require.NoError(t, testVM.Shutdown())
}

func TestVMLoadNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScriptRunnerNegative))

	testVM.CompileAndRun()
	require.Error(t, testVM.Shutdown())
}

func TestVMLoadNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript))

	testVM = nil
	require.ErrorIs(t, testVM.Load(testScript), ErrNoVMLoaded)
}

func TestCompileAndRunNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmcount := VMSCount.Len()
	testVM := manager.New()
	require.NoError(t, testVM.Load(testScript))

	require.NoError(t, testVM.Load(testScript))

	testVM = nil
	testVM.CompileAndRun()
	require.ErrorIs(t, testVM.Shutdown(), ErrNoVMLoaded)
	assert.NotEqual(t, vmcount-1, VMSCount.Len(), "Expected vmcount to decrease")
}

func TestVMLoadNoFile(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	assert.ErrorIs(t, testVM.Load("missing file"), os.ErrNotExist)
}

func TestVMCompile(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMRun(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.NewVM()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.RunCtx()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMRunTX(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.NewVM()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.RunCtx()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMWithRunner(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}
	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
	if VMSCount.Len() == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestVMWithRunnerOnce(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}
	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}
}

func TestVMWithRunnerNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScriptRunnerNegative)
	if err != nil {
		t.Fatal(err)
	}
	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}
	if VMSCount.Len() == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestVMWithRunnerInvalidTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScriptRunnerInvalid)
	if err != nil {
		t.Fatal(err)
	}
	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}

	if VMSCount.Len() == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestShutdownAll(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmCount := VMSCount.Len()
	VM := manager.New()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	err = manager.ShutdownAll()
	if err != nil {
		t.Fatal(err)
	}

	if VMSCount.Len() == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestRead(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	VM := manager.NewVM()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	ScriptPath = filepath.Join("..", "..", "testdata", "gctscript")
	data, err := VM.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 1 {
		t.Fatal("expected data to be returned")
	}
	_ = VM.Shutdown()
}

func TestRemoveVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	id, _ := uuid.FromString("6f20c907-64a0-48f2-848a-7837dee61672")
	err := manager.RemoveVM(id)
	if err != nil {
		if err.Error() != "VM 6f20c907-64a0-48f2-848a-7837dee61672 not found" {
			t.Fatal(err)
		}
	}
}

func TestError_Error(t *testing.T) {
	x := Error{
		Script: "noscript.gct",
		Action: "test",
		Cause:  errors.New("HELLO ERROR"),
	}

	if x.Error() != "GCT Script: (ACTION) test (SCRIPT) noscript.gct HELLO ERROR" {
		t.Fatal(x.Error())
	}
}

func TestVM_CompileInvalid(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testInvalidScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}
	err = testVM.RunCtx()
	if err == nil {
		t.Fatal("unexpected result broken script compiled successfully ")
	}

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.RunCtx()
	if err == nil {
		t.Fatal("unexpected result broken script compiled successfully ")
	}

	testVM = manager.New()
	err = testVM.Load(testInvalidScript)
	if err != nil {
		t.Fatal(err)
	}

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	if err == nil {
		t.Fatal("Shutdown() passed successfully but expected to fail with invalid script")
	}
}

func TestVM_CompileBroken(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testBrokenScript)
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err == nil {
		t.Fatal("unexpected result broken script compiled successfully ")
	}
}

func TestVM_CompileAndRunBroken(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testBrokenScript)
	if err != nil {
		t.Fatal(err)
	}

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	if err == nil {
		t.Fatal("expect error on shutdown due to invalid VM")
	}
}

func TestValidate(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	err := manager.Validate(testBrokenScript)
	if err == nil {
		t.Fatal(err)
	}
	err = manager.Validate(testScript)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMLimit(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, false, 0),
		started: 1,
	}
	if testVM := manager.New(); testVM != nil {
		t.Fatal("expected nil but received pointer to VM")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	err = manager.Autoload(scriptName, true)
	if err == nil {
		t.Fatal("expected err to be script not found received nil")
	}
	err = manager.Autoload("once", false)
	if err != nil {
		t.Fatal(err)
	}
	err = manager.Autoload(scriptName, false)
	if err == nil {
		t.Fatal("expected err to be script not found received nil")
	}
}

func TestVMCount(t *testing.T) {
	var c vmscount
	c.add()
	if c.Len() != 1 {
		t.Fatalf("expect c len to be 1 instead received %v", c.Len())
	}
	c.remove()
	if c.Len() != 0 {
		t.Fatalf("expect c len to be 0 instead received %v", c.Len())
	}
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

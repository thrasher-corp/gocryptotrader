package vm

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	maxTestVirtualMachines    uint8 = 30
	testVirtualMachineTimeout       = time.Minute
	scriptName                      = "1D01TH0RS3.gct"
)

var (
	testScript               = filepath.Join("..", "..", "testdata", "gctscript", "once.gct")
	testInvalidScript        = filepath.Join("..", "..", "testdata", "gctscript", "invalid.gct")
	testBrokenScript         = filepath.Join("..", "..", "testdata", "gctscript", "broken.gct")
	testScriptRunner         = filepath.Join("..", "..", "testdata", "gctscript", "timer.gct")
	testScriptRunner1s       = filepath.Join("..", "..", "testdata", "gctscript", "1s_timer.gct")
	testScriptRunnerInvalid  = filepath.Join("..", "..", "testdata", "gctscript", "invalid_timer.gct")
	testScriptRunnerNegative = filepath.Join("..", "..", "testdata", "gctscript", "negative_timer.gct")
)

func TestMain(m *testing.M) {
	c := log.GenDefaultSettings()
	c.Enabled = convert.BoolPtr(false)
	log.RWM.Lock()
	log.GlobalLogConfig = &c
	log.RWM.Unlock()
	os.Exit(m.Run())
}

func TestNewVM(t *testing.T) {
	manager := GctScriptManager{
		config: configHelper(true, true, maxTestVirtualMachines),
	}
	x := manager.New()
	if x != nil {
		t.Error("Should not create a VM when manager not started")
	}
	manager.started = 1
	x = manager.New()
	xType := reflect.TypeOf(x).String()
	if xType != "*vm.VM" {
		t.Fatalf("vm.New should return pointer to VM instead received: %v", x)
	}
}

func TestVMLoad(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	testScript = testScript[0 : len(testScript)-4]
	testVM = manager.New()
	err = testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	manager.config = configHelper(false, false, maxTestVirtualMachines)
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrScriptingDisabled) {
			t.Fatal(err)
		}
	}
}

func TestVMLoad1s(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScriptRunner1s)
	if err != nil {
		t.Fatal(err)
	}

	testVM.CompileAndRun()
	err = testVM.Shutdown()
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
}

func TestVMLoadNegativeTimer(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScriptRunnerNegative)
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
	testVM.CompileAndRun()
	err = testVM.Shutdown()
	if err == nil {
		t.Fatal("expect error on shutdown due to invalid VM")
	}
}

func TestVMLoadNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
	testVM = nil
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
}

func TestCompileAndRunNilVM(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	vmcount := VMSCount.Len()
	testVM := manager.New()
	err := testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}

	testVM = nil
	testVM.CompileAndRun()
	err = testVM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}
	if VMSCount.Len() == vmcount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestVMLoadNoFile(t *testing.T) {
	manager := GctScriptManager{
		config:  configHelper(true, true, maxTestVirtualMachines),
		started: 1,
	}
	testVM := manager.New()
	err := testVM.Load("missing file")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
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

	err = testVM.Run()
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
	err = testVM.Run()
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
	testVM := manager.New()
	if testVM != nil {
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

func configHelper(enabled, imports bool, max uint8) *Config {
	return &Config{
		Enabled:            enabled,
		AllowImports:       imports,
		ScriptTimeout:      testVirtualMachineTimeout,
		MaxVirtualMachines: max,
		Verbose:            true,
	}
}

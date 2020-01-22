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
	"github.com/thrasher-corp/gocryptotrader/logger"
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
	c := logger.GenDefaultSettings()
	c.Enabled = convert.BoolPtr(false)
	logger.GlobalLogConfig = &c
	GCTScriptConfig = configHelper(true, true, maxTestVirtualMachines)
	os.Exit(m.Run())
}

func TestNewVM(t *testing.T) {
	x := New()
	xType := reflect.TypeOf(x).String()
	if xType != "*vm.VM" {
		t.Fatalf("vm.New should return pointer to VM instead received: %v", x)
	}
}

func TestVMLoad(t *testing.T) {
	GCTScriptConfig = configHelper(true, true, maxTestVirtualMachines)
	testVM := New()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	testScript = testScript[0 : len(testScript)-4]
	testVM = New()
	err = testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	GCTScriptConfig = configHelper(false, false, maxTestVirtualMachines)
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrScriptingDisabled) {
			t.Fatal(err)
		}
	}
	GCTScriptConfig = configHelper(true, true, maxTestVirtualMachines)
}

func TestVMLoad1s(t *testing.T) {
	testVM := New()
	err := testVM.Load(testScriptRunner1s)
	if err != nil {
		t.Fatal(err)
	}

	testVM.CompileAndRun()
	time.Sleep(5000)
	err = testVM.Shutdown()
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
	}
}

func TestVMLoadNegativeTimer(t *testing.T) {
	testVM := New()
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
	testVM := New()
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
	vmcount := VMSCount.Len()
	testVM := New()
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
	testVM := New()
	err := testVM.Load("missing file")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
}

func TestVMCompile(t *testing.T) {
	testVM := New()
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
	testVM := NewVM()
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
	testVM := NewVM()
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
	vmCount := VMSCount.Len()
	VM := New()
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
	vmCount := VMSCount.Len()
	VM := New()
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
	vmCount := VMSCount.Len()
	VM := New()
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
	vmCount := VMSCount.Len()
	VM := New()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if VMSCount.Len() == vmCount {
		t.Fatal("expected VM count to increase")
	}
	err = ShutdownAll()
	if err != nil {
		t.Fatal(err)
	}

	if VMSCount.Len() == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestRead(t *testing.T) {
	VM := NewVM()
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
	id, _ := uuid.FromString("6f20c907-64a0-48f2-848a-7837dee61672")
	err := RemoveVM(id)

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
	testVM := New()
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

	testVM = New()
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

	testVM = New()
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
	testVM := New()
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
	testVM := New()
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
	err := Validate(testBrokenScript)
	if err == nil {
		t.Fatal(err)
	}
	err = Validate(testScript)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMLimit(t *testing.T) {
	GCTScriptConfig = configHelper(true, false, 0)
	testVM := New()
	if testVM != nil {
		t.Fatal("expected nil but received pointer to VM")
	}
	GCTScriptConfig = configHelper(true, true, maxTestVirtualMachines)
}

func TestAutoload(t *testing.T) {
	GCTScriptConfig = &Config{
		Enabled: true,
		AutoLoad: []string{
			scriptName,
		},
		Verbose: true,
	}

	ScriptPath = filepath.Join("..", "..", "testdata", "gctscript")
	err := Autoload(scriptName, true)
	if err != nil {
		t.Fatal(err)
	}
	err = Autoload(scriptName, true)
	if err == nil {
		t.Fatal("expected err to be script not found received nil")
	}
	err = Autoload("once", false)
	if err != nil {
		t.Fatal(err)
	}
	err = Autoload(scriptName, false)
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

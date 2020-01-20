package vm

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	maxTestVirtualMachines     uint8         = 30
	testVirtualMachineTimeout  time.Duration = 6000000
	testVirtualMachineTimeout0 time.Duration = 0
)

var (
	testScript               = filepath.Join("..", "..", "testdata", "gctscript", "once.gct")
	testInvalidScript        = filepath.Join("..", "..", "testdata", "gctscript", "broken.gct")
	testScriptRunner         = filepath.Join("..", "..", "testdata", "gctscript", "timer.gct")
	testScriptRunner1s         = filepath.Join("..", "..", "testdata", "gctscript", "1s_timer.gct")
	testScriptRunnerNegative = filepath.Join("..", "..", "testdata", "gctscript", "negative_timer.gct")
)

func TestMain(m *testing.M) {
	c := logger.GenDefaultSettings()
	//c.Enabled = convert.BoolPtr(false)
	logger.GlobalLogConfig = &c
	GCTScriptConfig = configHelper(true, true, testVirtualMachineTimeout, maxTestVirtualMachines)
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
	GCTScriptConfig = configHelper(true, true, testVirtualMachineTimeout, maxTestVirtualMachines)
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

	GCTScriptConfig = configHelper(false, false, testVirtualMachineTimeout0, maxTestVirtualMachines)
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrScriptingDisabled) {
			t.Fatal(err)
		}
	}
	GCTScriptConfig = configHelper(true, true, testVirtualMachineTimeout, maxTestVirtualMachines)
}

func TestVMLoad1s(t *testing.T) {
	testVM := New()
	err := testVM.Load(testScriptRunner1s)
	if err != nil {
		t.Fatal(err)
	}

	testVM.CompileAndRun()
	time.Sleep(2)
	err = testVM.Shutdown()
	if err != nil {
		if !errors.Is(err, ErrNoVMLoaded) {
			t.Fatal(err)
		}
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
	vmCount := len(AllVMs)
	VM := New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}
	if len(AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
	if len(AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestVMWithRunnerOnce(t *testing.T) {
	vmCount := len(AllVMs)
	VM := New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}
	if len(AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}
	if len(AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestVMWithRunnerNegativeTimer(t *testing.T) {
	vmCount := len(AllVMs)
	VM := New()
	if VM == nil {
		t.Fatal("Failed to allocate new VM exiting")
	}
	err := VM.Load(testScriptRunnerNegative)
	if err != nil {
		t.Fatal(err)
	}
	if len(AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}
	VM.CompileAndRun()
	err = VM.Shutdown()
	if err == nil {
		t.Fatal("VM should not be running with invalid timer")
	}
	if len(AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestShutdownAll(t *testing.T) {
	AllVMs = make(map[uuid.UUID]*VM)
	vmCount := len(AllVMs)
	VM := New()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if len(AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}
	err = ShutdownAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(AllVMs) == vmCount-1 {
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
	if err == nil {
		t.Fatal("unexpected result broken script compiled successfully ")
	}
}

func TestValidate(t *testing.T) {
	err := Validate(testInvalidScript)
	if err == nil {
		t.Fatal(err)
	}
	err = Validate(testScript)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMLimit(t *testing.T) {
	GCTScriptConfig = configHelper(true, false, testVirtualMachineTimeout0, 0)
	testVM := New()
	if testVM != nil {
		t.Fatal("expected nil but received pointer to VM")
	}
	GCTScriptConfig = configHelper(true, true, testVirtualMachineTimeout, maxTestVirtualMachines)
}

func configHelper(enabled, imports bool, timeout time.Duration, max uint8) *Config {
	return &Config{
		Enabled:            enabled,
		AllowImports:       imports,
		ScriptTimeout:      timeout,
		MaxVirtualMachines: max,
		Verbose:            true,
	}
}

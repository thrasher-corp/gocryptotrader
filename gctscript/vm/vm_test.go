package vm

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

var (
	testScript       = filepath.Join("..","..","testdata","gctscript","once.gct")
	testInvalidScript       = filepath.Join("..","..","testdata","gctscript","broken.gct")
	testScriptRunner = filepath.Join("..","..","testdata","gctscript","timer.gct")
)

func TestNewVM(t *testing.T) {
	x := New()
	xType := reflect.TypeOf(x).String()
	if xType != "*vm.VM" {
		t.Fatalf("vm.New should return pointer to VM instead received: %v", x)
	}
}

func TestVMLoad(t *testing.T) {
	GCTScriptConfig = configHelper(true, true, 0, 6)
	testVM := New()
	err := testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	testScript = testScript[0:len(testScript)-4]
	GCTScriptConfig = configHelper(true, true, 0, 6)
	testVM = New()
	err = testVM.Load(testScript)
	if err != nil {
		t.Fatal(err)
	}

	GCTScriptConfig = configHelper(false, false, 0, 6)
	err = testVM.Load(testScript)
	if err != nil {
		if !errors.Is(err, ErrScriptingDisabled) {
			t.Fatal(err)
		}
	}
}

func TestVMLoadNoFile(t *testing.T) {
	GCTScriptConfig = configHelper(true, false, 0, 6)
	testVM := New()
	err := testVM.Load("missing file")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
}

func TestVMCompile(t *testing.T) {
	GCTScriptConfig = configHelper(true, true, 6000000, 6)
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
	GCTScriptConfig = configHelper(true, true, 10000, 6)
	testVM := New()
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
	GCTScriptConfig = configHelper(true, true, 6000000, 6)
	testVM := New()
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
	GCTScriptConfig = configHelper(true, true, 6000000, 6)
	VM := New()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if len(AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}

	err = VM.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
	if len(AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestShutdownAll(t *testing.T) {
	vmCount := len(AllVMs)
	GCTScriptConfig = configHelper(true, true, 6000000, 6)
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
	GCTScriptConfig = configHelper(true, true, 6000000, 1)
	VM := New()
	err := VM.Load(testScriptRunner)
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	ScriptPath = "../../testdata/gctscript/"
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

	if !errors.Is(err, ErrNoVMFound) {
		t.Fatal(err)
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
	GCTScriptConfig = configHelper(true, true, 6000000, 6)
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

func configHelper(enabled, imports bool, timeout time.Duration, max uint8) *Config {
	return &Config{
		Enabled:            enabled,
		AllowImports:       imports,
		ScriptTimeout:      timeout,
		MaxVirtualMachines: max,
	}
}

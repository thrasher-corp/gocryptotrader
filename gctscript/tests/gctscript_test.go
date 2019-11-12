package tests

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
)

var ()

func TestMain(m *testing.M) {
	t := m.Run()

	os.Exit(t)
}

func TestNewVM(t *testing.T) {
	x := vm.New()
	xType := reflect.TypeOf(x).String()
	if xType != "*vm.VM" {
		t.Fatalf("vm.New should return pointer to VM instead received: %v", x)
	}
}

func TestVMLoad(t *testing.T) {
	testVM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 0)
	err := testVM.Load("../../testdata/gctscript/test.gctgo")
	if err != nil {
		t.Fatal(err)
	}

	vm.GCTScriptConfig = configHelper(false, false, 0)
	err = testVM.Load("../../testdata/gctscript/test.gctgo")
	if err != nil {
		if !errors.Is(err, vm.ErrScriptingDisabled) {
			t.Fatal(err)
		}
	}
}

func TestVMCompile(t *testing.T) {
	testVM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 0)
	err := testVM.Load("../../testdata/gctscript/test.gctgo")
	if err != nil {
		t.Fatal(err)
	}

	err = testVM.Compile()
	if err != nil {
		t.Fatal(err)
	}
}

func TestVMRun(t *testing.T) {
	testVM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 10000)
	err := testVM.Load("../../testdata/gctscript/test.gctgo")
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
	testVM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 600000)
	err := testVM.Load("../../testdata/gctscript/test.gctgo")
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
	vmCount := len(vm.AllVMs)

	VM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 6000000)
	err := VM.Load("../../testdata/gctscript/runner.gctgo")
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if len(vm.AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}

	err = VM.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
	if len(vm.AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestShutdownAll(t *testing.T) {
	vmCount := len(vm.AllVMs)

	VM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 6000000)
	err := VM.Load("../../testdata/gctscript/runner.gctgo")
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	if len(vm.AllVMs) == vmCount {
		t.Fatal("expected VM count to increase")
	}

	err = vm.ShutdownAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(vm.AllVMs) == vmCount-1 {
		t.Fatal("expected VM count to decrease")
	}
}

func TestRead(t *testing.T) {
	VM := vm.New()
	vm.GCTScriptConfig = configHelper(true, true, 6000000)
	err := VM.Load("../../testdata/gctscript/runner.gctgo")
	if err != nil {
		t.Fatal(err)
	}

	VM.CompileAndRun()

	vm.ScriptPath = "../../testdata/gctscript/"
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
	err := vm.RemoveVM(id)

	if !errors.Is(err, vm.ErrNoVMFound) {
		t.Fatal(err)
	}
}

func configHelper(enabled, imports bool, timeout time.Duration) *vm.Config {
	return &vm.Config{
		Enabled:       enabled,
		AllowImports:  imports,
		ScriptTimeout: timeout,
	}
}

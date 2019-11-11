package tests

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"
)

var (
	testVM = vm.New()
)

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

func TestVMRunTX(t *testing.T) {
	vm.GCTScriptConfig = configHelper(true, true, 10000)
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

func configHelper(enabled, imports bool, timeout time.Duration) *vm.Config {
	return &vm.Config{
		Enabled:       enabled,
		AllowImports:  imports,
		ScriptTimeout: timeout,
	}
}

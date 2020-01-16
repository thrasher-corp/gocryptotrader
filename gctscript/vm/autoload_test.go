package vm

import (
	"path/filepath"
	"testing"
)

const scriptName string = "1D01TH0RS3.gct"

var (
	GCTConfig = &Config{
		Enabled: true,
		AutoLoad: []string{
			scriptName,
		},
	}
)

func TestAutoload(t *testing.T) {
	GCTScriptConfig = GCTConfig
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

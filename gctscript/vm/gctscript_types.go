package vm

import (
	"errors"
)

const gctScript = "GCT Script"

// Config user configurable options for gctscript
type Config struct {
	Enabled      bool     `json:"enabled"`
	AllowImports bool     `json:"allow_imports"`
	AutoStart    []string `json:"auto_start"`
}

// Error error interface for VM
type Error struct {
	Script string
	Action string
	Cause  error
}

var (
	// GCTScriptConfig initialised global copy of Config{}
	GCTScriptConfig = &Config{}
	// ScriptPath path to load/save scripts
	ScriptPath string
)

var (
	// ErrScriptingDisabled error message displayed when gctscript is disabled
	ErrScriptingDisabled = errors.New("scripting is disabled")
	// ErrNoVMLoaded error message displayed if a virtual machine has not been initialised
	ErrNoVMLoaded = errors.New("no virtual machine loaded")
)

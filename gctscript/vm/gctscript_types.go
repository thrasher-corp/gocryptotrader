package vm

import (
	"errors"
	"time"
)

const (
	gctScript = "GCT Script"
	// ErrScriptFailedValidation message to display when a script fails its validation
	ErrScriptFailedValidation = "validation failed"
)

// Config user configurable options for gctscript
type Config struct {
	Enabled            bool          `json:"enabled"`
	ScriptTimeout      time.Duration `json:"timeout"`
	MaxVirtualMachines uint8         `json:"max_virtual_machines"`
	AllowImports       bool          `json:"allow_imports"`
	AutoLoad           []string      `json:"auto_load"`
	Verbose            bool          `json:"verbose"`
}

// Error interface to meet error requirements
type Error struct {
	Script string
	Action string
	Cause  error
}

var (
	// ScriptPath path to load/save scripts
	ScriptPath string
)

var (
	// ErrScriptingDisabled error message displayed when gctscript is disabled
	ErrScriptingDisabled = errors.New("scripting is disabled")
	// ErrNoVMLoaded error message displayed if a virtual machine has not been initialised
	ErrNoVMLoaded = errors.New("no virtual machine loaded")
)

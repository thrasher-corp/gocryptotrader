package vm

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/d5/tengo/script"
)

const gctScript = "GCT Script"

// VM pointer to "script" (precompiled source) and "compiled" (compiled byte code) instances
type VM struct {
	name  string
	timer time.Time

	Script   *script.Script
	Compiled *script.Compiled
	ctx      context.Context
}

// Config user configurable options for gctscript
type Config struct {
	Enabled      bool `json:"enabled"`
	AllowImports bool `json:"allow_imports"`
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

var (
	// VMPool stuff
	VMPool = &sync.Pool{
		New: func() interface{} {
			return new(script.Script)
		},
	}
)

var VMList []*VM

var scheduledItem []vmtask

type vmtask struct {
	name    string
	nextRun time.Time
}

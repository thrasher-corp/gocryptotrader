package gctscript

import (
	"errors"

	"github.com/d5/tengo/script"
)

const gctScript = "GCT Script"

type VM struct {
	Script   *script.Script
	Compiled *script.Compiled
}

type Config struct {
	Enabled      bool `json:"enabled"`
	AllowImports bool `json:"allow_imports"`
}

type VMError struct {
	Script string
	Action string
	Cause  error
}

var (
	GCTScriptConfig = &Config{}
	ScriptPath      string
)

var (
	ErrScriptingDisabled = errors.New("scripting is disabled")
)

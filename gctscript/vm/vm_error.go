package vm

import (
	"errors"
	"fmt"
	"path/filepath"
)

func (e Error) Error() string {
	var scriptName, action string
	if e.Script != "" {
		scriptName = fmt.Sprintf("(SCRIPT) %s ", filepath.Base(e.Script))
	}

	if e.Action != "" {
		action = fmt.Sprintf("(ACTION) %s ", e.Action)
	}

	return fmt.Sprintf("%s: %s%s%s", gctScript, action, scriptName, e.Cause)
}

// Unwrap returns e.Cause meeting errors interface requirements.
func (e Error) Unwrap() error {
	return e.Cause
}

var (
	ErrNoVMFound = errors.New("no VM found")
)

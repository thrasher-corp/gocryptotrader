package vm

import (
	"context"
	"sync"
	"time"

	"github.com/d5/tengo/script"
	"github.com/gofrs/uuid"
)

const (
	AuditEventID = "gctscript"
	// DefaultTimeoutValue default timeout value for virtual machines
	DefaultTimeoutValue = 1 * time.Minute
	// DefaultMaxVirtualMachines max number of virtual machines that can be loaded at one time
	DefaultMaxVirtualMachines = 10

	// TypeLoad text to display in script_event table when a VM is loaded
	TypeLoad = "load"
	// TypeCreate text to display in script_event table when a VM is created
	TypeCreate = "create"
	// TypeExecute text to display in script_event table when a script is executed
	TypeExecute = "execute"
	// TypeStop text to display in script_event table when a running script is stopped
	TypeStop = "stop"
	// TypeRead text to display in script_event table when a script contents is read
	TypeRead = "read"

	// StatusSuccess text to display in script_event table on successful execution
	StatusSuccess = "success"
	// StatusFailure text to display in script_event table when script execution fails
	StatusFailure = "failure"
	// StatusError text to display in script_event table when there was an error in execution
	StatusError = "error"
)

var (
	pool = &sync.Pool{
		New: func() interface{} {
			return new(script.Script)
		},
	}
	// AllVMs stores all current Virtual Machine instances
	AllVMs map[uuid.UUID]*VM
)

// VM contains a pointer to "script" (precompiled source) and "compiled" (compiled byte code) instances
type VM struct {
	ID       uuid.UUID
	File     string
	Path     string
	Script   *script.Script
	Compiled *script.Compiled
	ctx      context.Context
	T        time.Duration
	NextRun  time.Time
	S        chan struct{}
}

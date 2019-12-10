package vm

import (
	"context"
	"sync"
	"time"

	"github.com/d5/tengo/script"
	"github.com/gofrs/uuid"
)

const (
	// AuditEventName name to use for audit event logging
	AuditEventName = "gctscript"
	// DefaultTimeoutValue default timeout value for virtual machines
	DefaultTimeoutValue = 1 * time.Minute
	// DefaultMaxVirtualMachines max number of virtual machines that can be loaded at one time
	DefaultMaxVirtualMachines = 10

	TypeExecute = "execute"
	TypeUpload = "upload"
	TypeStop = "stop"
	TypeStatus = "status"

	StatusSuccess = "success"
	StatusFailure = "failure"
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
	Name     string
	File     string
	Script   *script.Script
	Compiled *script.Compiled

	ctx context.Context
	T   time.Duration

	NextRun time.Time

	S chan struct{}
}

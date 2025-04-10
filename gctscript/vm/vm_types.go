package vm

import (
	"sync"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/gofrs/uuid"
)

const (
	// DefaultTimeoutValue default timeout value for virtual machines
	DefaultTimeoutValue = 30 * time.Second
	// DefaultMaxVirtualMachines max number of virtual machines that can be loaded at one time
	DefaultMaxVirtualMachines uint64 = 10

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
)

type vmscount uint64

var (
	pool = &sync.Pool{New: func() any { return new(tengo.Script) }}
	// AllVMSync stores all current Virtual Machine instances
	AllVMSync = &sync.Map{}
	// VMSCount running total count of Virtual Machines
	VMSCount vmscount
)

// VM contains a pointer to "script" (precompiled source) and "compiled" (compiled byte code) instances
type VM struct {
	ID         uuid.UUID
	Hash       string
	File       string
	Path       string
	Script     *tengo.Script
	Compiled   *tengo.Compiled
	T          time.Duration
	NextRun    time.Time
	S          chan struct{}
	config     *Config
	unregister func() error
}

package vm

import (
	"context"
	"sync"
	"time"

	"github.com/d5/tengo/script"
	"github.com/gofrs/uuid"
)

// VM contains a pointer to "script" (precompiled source) and "compiled" (compiled byte code) instances
type VM struct {
	ID       uuid.UUID
	Name     string
	file     string
	Script   *script.Script
	Compiled *script.Compiled

	ctx context.Context
	T   time.Duration

	NextRun time.Time

	S chan struct{}
}

// AllVMs stores all current Virtual Machine instances
var AllVMs map[uuid.UUID]*VM

var (
	// pool stuff
	pool = &sync.Pool{
		New: func() interface{} {
			return new(script.Script)
		},
	}
)

const GCTScriptAuditEvent = "gctscript"
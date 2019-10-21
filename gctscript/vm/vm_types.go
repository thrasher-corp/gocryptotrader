package vm

import (
	"context"
	"sync"
	"time"

	"github.com/d5/tengo/script"
)

// VM pointer to "script" (precompiled source) and "compiled" (compiled byte code) instances
type VM struct {
	name string

	Script   *script.Script
	Compiled *script.Compiled

	ctx context.Context

	t time.Duration

	c chan struct{}
}

// VMList stores all current Virtual Machine instances
var VMList []VM

var (
	// VMPool stuff
	VMPool = &sync.Pool{
		New: func() interface{} {
			return new(script.Script)
		},
	}
)

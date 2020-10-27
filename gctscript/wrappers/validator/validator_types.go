package validator

import (
	"errors"
	"sync/atomic"

	objects "github.com/d5/tengo/v2"
)

var (
	// IsTestExecution if test is executed under test conditions
	IsTestExecution atomic.Value

	exchError = &objects.String{
		Value: "",
	}
	errTestFailed = errors.New("test failed")
)

// Wrapper for validator interface
type Wrapper struct{}

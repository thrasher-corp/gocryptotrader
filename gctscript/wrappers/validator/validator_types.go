package validator

import (
	"errors"
	"sync"

	objects "github.com/d5/tengo/v2"
)

var (
	// RWValidatorLock mutex lock
	RWValidatorLock = &sync.RWMutex{}
	// IsTestExecution if test is executed under test conditions
	IsTestExecution bool

	exchError = &objects.String{
		Value: "error",
	}
	errTestFailed = errors.New("test failed")
)

type Wrapper struct{}

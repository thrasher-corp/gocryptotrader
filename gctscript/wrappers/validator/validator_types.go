package validator

import (
	"errors"
	"sync"

	"github.com/d5/tengo/objects"
)

var (
	// RWValidatorLock mutex lock
	RWValidatorLock sync.RWMutex
	// IsTestExecution if test is executed under test conditions
	IsTestExecution bool

	exch = &objects.String{
		Value: "BTC Markets",
	}
	exchError = &objects.String{
		Value: "error",
	}
	currencyPair = &objects.String{
		Value: "BTC-AUD",
	}
	delimiter = &objects.String{
		Value: "-",
	}
	assetType = &objects.String{
		Value: "SPOT",
	}
	orderID = &objects.String{
		Value: "1235",
	}

	tv            = objects.TrueValue
	fv            = objects.FalseValue
	errTestFailed = errors.New("test failed")
)

type Wrapper struct{}

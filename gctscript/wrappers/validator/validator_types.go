package validator

import (
	"errors"

	"github.com/d5/tengo/objects"
)

var (
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

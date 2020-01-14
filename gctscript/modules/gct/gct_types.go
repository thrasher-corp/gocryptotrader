package gct

import (
	"github.com/d5/tengo/v2"
)

const (
	ErrParameterConvertFailed string = "%v failed conversion"
)

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{
	"exchange": exchangeModule,
}

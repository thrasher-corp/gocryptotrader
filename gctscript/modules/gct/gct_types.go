package gct

import "github.com/d5/tengo/objects"

const (
	ErrParameterConvertFailed string = "%v failed conversion"
)

// Modules map of all loadable modules
var Modules = map[string]map[string]objects.Object{
	"exchange": exchangeModule,
}

package gct

import "github.com/d5/tengo/objects"

// Modules map of all loadable modules
var Modules = map[string]map[string]objects.Object{
	"exchange": exchangeModule,
}

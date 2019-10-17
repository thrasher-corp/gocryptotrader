package gct

import "github.com/d5/tengo/objects"

var GCTModules = map[string]map[string]objects.Object{
	"exchange": exchangeModule,
}

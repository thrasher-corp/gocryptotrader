package gct

import (
	objects "github.com/d5/tengo/v2"
)

const (
	// ErrParameterConvertFailed error to return when type conversion fails
	ErrParameterConvertFailed = "%v failed conversion"
	// ErrEmptyParameter error to return when empty parameter is received
	ErrEmptyParameter = "received empty parameter for %v"
)

var supportedDurations = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "24h", "1d", "3d", "1w", "1M"}

// Modules map of all loadable modules
var Modules = map[string]map[string]objects.Object{
	"exchange": exchangeModule,
	"common":   commonModule,
	"global":   globalModules,
}

// Context defines a juncture for script context to go context awareness
type Context struct {
	objects.Map
}

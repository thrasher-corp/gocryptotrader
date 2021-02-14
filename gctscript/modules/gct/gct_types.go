package gct

import (
	"errors"

	"github.com/d5/tengo/v2"
)

const (
	// ErrParameterConvertFailed error to return when type conversion fails
	ErrParameterConvertFailed = "%v failed conversion"
	// ErrEmptyParameter error to return when empty parameter is received
	ErrEmptyParameter = "received empty parameter for %v"
)

var errInvalidInterval = errors.New("invalid interval")
var supportedDurations = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "24h", "1d", "3d", "1w"}

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{
	"exchange": exchangeModule,
	"common":   commonModule,
}

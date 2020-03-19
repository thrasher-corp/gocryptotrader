package indicators

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var supportedDurations = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "24h", "1d", "3d", "1w"}

func toFloat64(data interface{}) (float64, error) {
	switch d := data.(type) {
	case float64:
		return d, nil
	case int:
		return float64(d), nil
	case int32:
		return float64(d), nil
	case int64:
		return float64(d), nil
	default:
		return 0, fmt.Errorf(modules.ErrParameterConvertFailed, d)
	}
}

func ParseInterval(in string) (time.Duration, error) {
	if !common.StringDataContainsInsensitive(supportedDurations, in) {
		return time.Nanosecond, errors.New("invalid interval")
	}
	switch in {
	case "1d":
		in = "24h"
	case "3d":
		in = "72h"
	}
	return time.ParseDuration(in)
}

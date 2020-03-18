package indicators

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

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

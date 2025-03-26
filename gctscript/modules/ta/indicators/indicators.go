package indicators

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// OHLCV locale string for OHLCV data conversion failure
const OHLCV = "OHLCV data"

var errInvalidSelector = errors.New("invalid selector")

func toFloat64(data any) (float64, error) {
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

// ParseIndicatorSelector returns indicator number from string for slice selection
func ParseIndicatorSelector(in string) (int, error) {
	switch in {
	case "open":
		return 1, nil
	case "high":
		return 2, nil
	case "low":
		return 3, nil
	case "close":
		return 4, nil
	case "vol":
		return 5, nil
	default:
		return 0, errInvalidSelector
	}
}

// ParseMAType returns moving average from string
func ParseMAType(in string) (indicators.MaType, error) {
	in = strings.ToLower(in)
	switch in {
	case "sma":
		return indicators.Sma, nil
	case "ema":
		return indicators.Ema, nil
	default:
		return 0, errInvalidSelector
	}
}

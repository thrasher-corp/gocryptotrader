package indicators

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// OHLCV locale string for OHLCV data conversion failure
const OHLCV = "OHLCV data"

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

// ParseIndicatorSelector returns indidcator number from string for slice selection
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
		return 0, errors.New("invalid selector")
	}
}

// ParseMAType returns moving average from sring
func ParseMAType(in string) (indicators.MaType, error) {
	in = strings.ToLower(in)
	switch in {
	case "sma":
		return indicators.SMA, nil
	case "ema":
		return indicators.EMA, nil
	case "wma":
		return indicators.WMA, nil
	case "dema":
		return indicators.DEMA, nil
	case "tema":
		return indicators.TEMA, nil
	case "trima":
		return indicators.TRIMA, nil
	case "kama":
		return indicators.KAMA, nil
	case "mama":
		return indicators.MAMA, nil
	case "t3ma":
		return indicators.T3MA, nil
	default:
		return 0, errors.New("invalid selector")
	}
}

package indicators

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gct-ta/indicators"
)

// OHLCV locale string for OHLCV data conversion failure
const OHLCV = "OHLCV data"

var errInvalidSelector = errors.New("invalid selector")

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

// ParseMAType returns moving average from sring
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

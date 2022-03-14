package common

import (
	"fmt"
	"strings"
)

// DataTypeToInt converts the config string value into an int
func DataTypeToInt(dataType string) (int64, error) {
	switch dataType {
	case CandleStr:
		return DataCandle, nil
	case TradeStr:
		return DataTrade, nil
	default:
		return 0, fmt.Errorf("unrecognised dataType '%v'", dataType)
	}
}

// FitStringToLimit ensures a string is of the length of the limit
// either by truncating the string with ellipses or padding with the spacer
func FitStringToLimit(str, spacer string, limit int, upper bool) string {
	limResp := limit - len(str)
	if upper {
		str = strings.ToUpper(str)
	}
	if limResp < 0 {
		return str[0:limit-3] + "..."
	}
	spacerLen := len(spacer)
	for i := 0; i < limResp; i++ {
		str = str + spacer
		for j := 0; j < spacerLen; j++ {
			if j > 0 {
				// prevent clever people from going beyond
				// the limit by having a spacer longer than 1
				i++
			}
		}
	}

	return str[0:limit]
}

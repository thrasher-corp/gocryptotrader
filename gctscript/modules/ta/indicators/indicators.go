package indicators

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

func appendData(data []interface{}) ([]float64, error) {
	var appendTo []float64

	for x := range data {
		switch data[x].(type) {
		case float64:
			appendTo = append(appendTo, data[x].(float64))
		case int64:
			appendTo = append(appendTo, float64(data[x].(int64)))
		case int:
			appendTo = append(appendTo, float64(data[x].(int)))
		case int32:
			appendTo = append(appendTo, float64(data[x].(int32)))
		default:
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, data[x], x)
		}
	}
	return appendTo, nil
}
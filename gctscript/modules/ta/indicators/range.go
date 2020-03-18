package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// RangeModule range indicator commands
var RangeModule = map[string]objects.Object{
	"atr": &objects.UserFunction{Name: "atr", Value: atr},
}

func atr(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData := ohlcvInput.([]interface{})
	ohclvData := make([][]float64, 6)

	for x := range ohlcvInputData {
		switch t := ohlcvInputData[x].(type) {
		case []interface{}:
			value, err := toFloat64(t[2])
			if err != nil {
				return nil, err
			}
			ohclvData[2] = append(ohclvData[2], value)

			value, err = toFloat64(t[3])
			if err != nil {
				return nil, err
			}
			ohclvData[3] = append(ohclvData[3], value)

			value, err = toFloat64(t[4])
			if err != nil {
				return nil, err
			}
			ohclvData[4] = append(ohclvData[4], value)

			value, err = toFloat64(t[5])
			if err != nil {
				return nil, err
			}
			ohclvData[5] = append(ohclvData[5], value)
		default:
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed, "OHLCV")
		}
	}

	inTimePeroid, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Atr(ohclvData[2], ohclvData[5], ohclvData[4], inTimePeroid)

	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

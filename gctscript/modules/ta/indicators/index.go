package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// MfiModule index indicator commands
var MfiModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: mfi},
}

func mfi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}
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
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
		}
	}

	inTimePeriod, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeriod)
	}

	ret := indicators.Mfi(ohclvData[2], ohclvData[3], ohclvData[4], ohclvData[5], inTimePeriod)
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

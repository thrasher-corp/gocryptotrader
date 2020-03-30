package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// ObvModule volume indicator commands
var ObvModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: obv},
}

func obv(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcIndicatorType, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, ohlcIndicatorType)
	}

	ohlcvInput := objects.ToInterface(args[1])
	ohlcvInputData, valid := ohlcvInput.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}
	ohclvData := make([][]float64, len(ohlcvInputData))

	for x := range ohlcvInputData {
		ohclvData[x] = make([]float64, 6)
		switch t := ohlcvInputData[x].(type) {
		case []interface{}:
			ohclvData[x][0] = 0

			value, err := toFloat64(t[1])
			if err != nil {
				return nil, err
			}
			ohclvData[x][1] = value

			value, err = toFloat64(t[2])
			if err != nil {
				return nil, err
			}
			ohclvData[x][2] = value

			value, err = toFloat64(t[3])
			if err != nil {
				return nil, err
			}
			ohclvData[x][3] = value

			value, err = toFloat64(t[4])
			if err != nil {
				return nil, err
			}
			ohclvData[x][4] = value

			value, err = toFloat64(t[5])
			if err != nil {
				return nil, err
			}
			ohclvData[x][5] = value
		default:
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
		}
	}

	var ret []float64
	if validator.IsTestExecution.Load() != true {
		ret = indicators.OBV(ohclvData, false)
	}
	r := &objects.Array{}
	for x := range ret {
		temp := &objects.Float{Value: ret[x]}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

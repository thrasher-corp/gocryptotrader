package indicators

import (
	"errors"
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// VolumeModule volume indicator commands
var VolumeModule = map[string]objects.Object{
	"obv": &objects.UserFunction{Name: "obv", Value: obv},
}

func obv(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcIndicatorType, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, ohlcIndicatorType)
	}

	var selector int
	switch ohlcIndicatorType {
	case "open":
		selector = 1
	case "high":
		selector = 2
	case "low":
		selector = 3
	case "close":
		selector = 4
	case "vol":
		selector = 5
	default:
		return nil, errors.New("invalid selection")
	}

	ohlcvInput := objects.ToInterface(args[1])
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

	ret := indicators.Obv(ohclvData[selector], ohclvData[5])
	r := &objects.Array{}
	for x := range ret {
		temp := &objects.Float{Value: ret[x]}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

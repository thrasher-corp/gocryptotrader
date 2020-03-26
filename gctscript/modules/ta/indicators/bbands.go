package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var BBandsModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: bbands},
}

func bbands(args ...objects.Object) (objects.Object, error) {
	if len(args) != 6 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcIndicatorType, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, ohlcIndicatorType)
	}

	selector, errIndSelector := ParseIndicatorSelector(ohlcIndicatorType)
	if errIndSelector != nil {
		return nil, errIndSelector
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
	inTimePeriod, ok := objects.ToInt(args[2])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeriod)
	}

	inNbDevUp, ok := objects.ToFloat64(args[3])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inNbDevUp)
	}

	inNbDevDn, ok := objects.ToFloat64(args[4])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inNbDevDn)
	}

	inMAType, ok := objects.ToString(args[5])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inMAType)
	}

	MAType, err := ParseMAType(inMAType)
	if err != nil {
		return nil, err
	}
	retUpper, retMiddle, retLower := indicators.BBands(ohclvData[selector], inTimePeriod, inNbDevDn, inNbDevDn, MAType)

	var ret objects.Array
	for x := range retUpper {
		temp := &objects.Array{}
		temp.Value = append(temp.Value, &objects.Float{Value: retUpper[x]})
		if retMiddle != nil {
			temp.Value = append(temp.Value, &objects.Float{Value: retMiddle[x]})
		}
		if retLower != nil {
			temp.Value = append(temp.Value, &objects.Float{Value: retLower[x]})
		}
		ret.Value = append(ret.Value, temp)
	}

	return &ret, nil
}

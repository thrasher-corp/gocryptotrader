package indicators

import (
	"errors"
	"fmt"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// BBandsModule bollinger bands indicator commands
var BBandsModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: bbands},
}

func bbands(args ...objects.Object) (objects.Object, error) {
	if len(args) != 6 {
		return nil, objects.ErrWrongNumArguments
	}

	var ret objects.Array
	if validator.IsTestExecution.Load() == true {
		return &ret, nil
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
	var allErrors []string
	for x := range ohlcvInputData {
		t := ohlcvInputData[x].([]interface{})
		value, err := toFloat64(t[2])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[2] = append(ohclvData[2], value)

		value, err = toFloat64(t[3])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[3] = append(ohclvData[3], value)

		value, err = toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[4] = append(ohclvData[4], value)

		value, err = toFloat64(t[5])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[5] = append(ohclvData[5], value)
	}

	inTimePeriod, ok := objects.ToInt(args[2])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inTimePeriod))
	}

	inNbDevUp, ok := objects.ToFloat64(args[3])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inNbDevUp))
	}

	inNbDevDn, ok := objects.ToFloat64(args[4])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inNbDevDn))
	}

	inMAType, ok := objects.ToString(args[5])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inMAType))
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	MAType, err := ParseMAType(inMAType)
	if err != nil {
		return nil, err
	}

	retUpper, retMiddle, retLower := indicators.BBands(ohclvData[selector], inTimePeriod, inNbDevDn, inNbDevDn, MAType)
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

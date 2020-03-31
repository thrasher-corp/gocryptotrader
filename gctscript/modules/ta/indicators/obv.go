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

// ObvModule volume indicator commands
var ObvModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: obv},
}

func obv(args ...objects.Object) (objects.Object, error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	r := &objects.Array{}
	if validator.IsTestExecution.Load() == true {
		return r, nil
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}

	ohclvData := make([][]float64, len(ohlcvInputData))
	var allErrors []string
	for x := range ohlcvInputData {
		ohclvData[x] = make([]float64, 6)
		t := ohlcvInputData[x].([]interface{})
		ohclvData[x][0] = 0

		value, err := toFloat64(t[1])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[x][1] = value

		value, err = toFloat64(t[2])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[x][2] = value

		value, err = toFloat64(t[3])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[x][3] = value

		value, err = toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[x][4] = value

		value, err = toFloat64(t[5])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohclvData[x][5] = value
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}
	ret := indicators.OBV(ohclvData, false)
	for x := range ret {
		temp := &objects.Float{Value: ret[x]}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

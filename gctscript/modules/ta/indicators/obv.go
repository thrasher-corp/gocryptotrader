package indicators

import (
	"errors"
	"fmt"
	"math"
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

	ohlcvData := make([][]float64, len(ohlcvInputData))
	var allErrors []string
	for x := range ohlcvInputData {
		ohlcvData[x] = make([]float64, 6)
		t := ohlcvInputData[x].([]interface{})
		ohlcvData[x][0] = 0
		ohlcvData[x][1] = 0
		ohlcvData[x][2] = 0
		ohlcvData[x][3] = 0

		value, err := toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[x][4] = value

		value, err = toFloat64(t[5])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[x][5] = value
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	ret := indicators.OBV(ohlcvData, true)
	for x := range ret {
		temp := &objects.Float{Value: math.Round(ret[x]*100) / 100}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

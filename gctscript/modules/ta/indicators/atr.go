package indicators

import (
	"errors"
	"fmt"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// AtrModule range indicator commands
var AtrModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: atr},
}

// AverageTrueRange is the string constant
const AverageTrueRange = "Average True Range"

// ATR defines a custom Average True Range indicator tengo object
type ATR struct {
	objects.Array
	Period int
}

// TypeName returns the name of the custom type.
func (o *ATR) TypeName() string {
	return AverageTrueRange
}

func atr(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}
	r := new(ATR)
	if validator.IsTestExecution.Load() == true {
		return r, nil
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]any)
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}

	ohlcvData := make([][]float64, 6)
	var allErrors []string
	for x := range ohlcvInputData {
		t, ok := ohlcvInputData[x].([]any)
		if !ok {
			return nil, errors.New("ohlcvInputData type assert failed")
		}
		if len(t) < 6 {
			return nil, errors.New("ohlcvInputData invalid data length")
		}
		value, err := toFloat64(t[2])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[2] = append(ohlcvData[2], value)

		value, err = toFloat64(t[3])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[3] = append(ohlcvData[3], value)

		value, err = toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[4] = append(ohlcvData[4], value)

		value, err = toFloat64(t[5])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvData[5] = append(ohlcvData[5], value)
	}

	inTimePeriod, ok := objects.ToInt(args[1])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inTimePeriod))
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	r.Period = inTimePeriod
	ret := indicators.ATR(ohlcvData[2], ohlcvData[3], ohlcvData[4], inTimePeriod)
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

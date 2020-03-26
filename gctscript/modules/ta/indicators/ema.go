package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// EMAModule EMA indicator commands
var EMAModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: ema},
}

func ema(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}

	var ohlcvClose []float64
	for x := range ohlcvInputData {
		switch t := ohlcvInputData[x].(type) {
		case []interface{}:
			value, err := toFloat64(t[4])
			if err != nil {
				return nil, err
			}
			ohlcvClose = append(ohlcvClose, value)
		default:
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
		}
	}

	inTimePeriod, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeriod)
	}

	var ret []float64
	if validator.IsTestExecution.Load() != true {
		ret = indicators.Ma(ohlcvClose, inTimePeriod, indicators.EMA)
	}
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

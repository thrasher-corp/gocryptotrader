package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

var MACDModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: macd},
}

func macd(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
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

	inFastPeriod, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inFastPeriod)
	}
	inSlowPeriod, ok := objects.ToInt(args[2])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inSlowPeriod)
	}
	inTimePeriod, ok := objects.ToInt(args[3])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeriod)
	}

	var macd, macdSignal, macdHist []float64
	if validator.IsTestExecution.Load() != true {
		macd, macdSignal, macdHist = indicators.Macd(ohlcvClose, inFastPeriod, inSlowPeriod, inTimePeriod)
	}

	var MACDRet objects.Array
	for x := range macd {
		tempMACD := &objects.Array{}
		tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macd[x]})
		if macdSignal != nil {
			tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macdSignal[x]})
		}
		if macdHist != nil {
			tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macdHist[x]})
		}
		MACDRet.Value = append(MACDRet.Value, tempMACD)
	}

	return &MACDRet, nil
}

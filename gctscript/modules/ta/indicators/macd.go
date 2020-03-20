package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
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

	macd, macdSignal, macdHist := indicators.Macd(ohlcvClose, inFastPeriod, inSlowPeriod, inTimePeriod)

	retMACD := &objects.Array{}
	for x := range macd {
		retMACD.Value = append(retMACD.Value, &objects.Float{Value: macd[x]})
	}

	retMACDSignal := &objects.Array{}
	for x := range macdSignal {
		retMACDSignal.Value = append(retMACDSignal.Value, &objects.Float{Value: macdSignal[x]})
	}

	retMACDHist := &objects.Array{}
	for x := range macdHist {
		retMACDHist.Value = append(retMACDHist.Value, &objects.Float{Value: macdHist[x]})
	}

	ret := &objects.Array{}
	ret.Value = append(ret.Value, retMACD, retMACDSignal, retMACDSignal)
	return ret, nil
}
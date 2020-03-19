package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// RsiModule relative strength index indicator commands
var RsiModule = map[string]objects.Object{
	"rsi": &objects.UserFunction{Name: "rsi", Value: rsi},
}

func rsi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvData := ohlcvInput.([]interface{})

	var ohlcvClose []float64
	for x := range ohlcvData {
		switch t := ohlcvData[x].(type) {
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

	inTimePeroid, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Rsi(ohlcvClose, inTimePeroid)
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

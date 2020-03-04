package indicators

import (
	"fmt"
	"reflect"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var RsiModule = map[string]objects.Object{
	"rsi": &objects.UserFunction{Name: "rsi", Value: rsi},
}

func rsi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	var ohlcCloseData []float64
	val := ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
				ohlcCloseData = append(ohlcCloseData, val[x].(float64))
		} else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	inTimePeroid, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Rsi(ohlcCloseData, inTimePeroid)
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

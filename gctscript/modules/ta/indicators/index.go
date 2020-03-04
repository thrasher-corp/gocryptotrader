package indicators

import (
	"fmt"
	"reflect"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var IndexModule = map[string]objects.Object{
	"mfi": &objects.UserFunction{Name: "mfi", Value: mfi},
}

func mfi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	var ohlcHighData []float64
	val := ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
			ohlcHighData = append(ohlcHighData, val[x].(float64))
		} else if reflect.TypeOf(val[x]).Kind() == reflect.Int64 {
			ohlcHighData = append(ohlcHighData, float64(val[x].(int64)))
		} else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	ohlcData = objects.ToInterface(args[1])
	var ohlcLowData []float64
	val = ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
			ohlcLowData = append(ohlcLowData, val[x].(float64))
		} else if reflect.TypeOf(val[x]).Kind() == reflect.Int64 {
			ohlcLowData = append(ohlcLowData, float64(val[x].(int64)))
		} else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	ohlcData = objects.ToInterface(args[2])
	var ohlcCloseData []float64
	val = ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
			ohlcCloseData = append(ohlcCloseData, val[x].(float64))
		} else if reflect.TypeOf(val[x]).Kind() == reflect.Int64 {
			ohlcCloseData = append(ohlcCloseData, float64(val[x].(int64)))
		}else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	ohlcData = objects.ToInterface(args[3])
	var ohlcVolData []float64
	val = ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
			ohlcVolData = append(ohlcVolData, val[x].(float64))
		} else if reflect.TypeOf(val[x]).Kind() == reflect.Int64 {
			ohlcVolData = append(ohlcVolData, float64(val[x].(int64)))
		} else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	inTimePeroid, ok := objects.ToInt(args[4])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Mfi(ohlcHighData, ohlcLowData, ohlcCloseData, ohlcVolData, inTimePeroid)
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}
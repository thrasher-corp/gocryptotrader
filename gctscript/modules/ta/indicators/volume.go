package indicators

import (
	"fmt"
	"reflect"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var VolumeModule = map[string]objects.Object{
	"obv": &objects.UserFunction{Name: "obv", Value: obv},
}

func obv(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	var ohlcInData []float64
	val := ohlcData.([]interface{})
	for x := range val {
		if reflect.TypeOf(val[x]).Kind() == reflect.Float64 {
			ohlcInData = append(ohlcInData, val[x].(float64))
		} else if reflect.TypeOf(val[x]).Kind() == reflect.Int64 {
			ohlcInData = append(ohlcInData, float64(val[x].(int64)))
		}else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	ohlcData = objects.ToInterface(args[1])
	var ohlcVolData []float64
	volVal := ohlcData.([]interface{})
	for x := range volVal {
		if reflect.TypeOf(volVal[x]).Kind() == reflect.Float64 {
			ohlcVolData = append(ohlcVolData, volVal[x].(float64))
		} else if reflect.TypeOf(volVal[x]).Kind() == reflect.Int64 {
			ohlcVolData = append(ohlcVolData, float64(volVal[x].(int64)))
		}else {
			return nil, fmt.Errorf(modules.ErrParameterWithPositionConvertFailed, val[x], x)
		}
	}

	ret := indicators.Obv(ohlcInData, ohlcVolData)
	r := &objects.Array{}
	for x := range ret {
		temp := &objects.Float{Value: ret[x]}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

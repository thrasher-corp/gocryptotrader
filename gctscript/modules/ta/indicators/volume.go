package indicators

import (
	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
)

var VolumeModule = map[string]objects.Object{
	"obv": &objects.UserFunction{Name: "obv", Value: obv},
}

func obv(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	ohlcInData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[0])
	ohlcVolData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ret := indicators.Obv(ohlcInData, ohlcVolData)
	r := &objects.Array{}
	for x := range ret {
		temp := &objects.Float{Value: ret[x]}
		r.Value = append(r.Value, temp)
	}
	return r, nil
}

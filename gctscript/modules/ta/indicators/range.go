package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// RangeModule range indicator commands
var RangeModule = map[string]objects.Object{
	"atr": &objects.UserFunction{Name: "atr", Value: atr},
}

func atr(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	tempOHLCSlice, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[1])
	tempOHLCVolSlice, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[2])
	tempOHLCCloseSlice, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	inTimePeroid, ok := objects.ToInt(args[3])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Atr(tempOHLCSlice, tempOHLCVolSlice, tempOHLCCloseSlice, inTimePeroid)

	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return nil, nil
}

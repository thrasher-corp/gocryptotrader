package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// IndexModule index indicator commands
var IndexModule = map[string]objects.Object{
	"mfi": &objects.UserFunction{Name: "mfi", Value: mfi},
}

func mfi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	ohlcHighData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[1])
	ohlcLowData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[2])
	ohlcCloseData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
	}

	ohlcData = objects.ToInterface(args[3])
	ohlcVolData, err := appendData(ohlcData.([]interface{}))
	if err != nil {
		return nil, err
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

package indicators

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

// RsiModule relative strength index indicator commands
var RsiModule = map[string]objects.Object{
	"rsi": &objects.UserFunction{Name: "rsi", Value: rsi},
	"rsi_exchange": &objects.UserFunction{Name: "rsi_exchange", Value: rsi_exchange},
}

func rsi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcvInput := objects.ToInterface(args[0])
	fmt.Println(ohlcvInput)
	// ohlcvData := ohlcvInput.([]interface{})
	//
	// var ohlcvClose []float64
	// for x := range ohlcvData {
	// 	switch t := ohlcvData[x].(type) {
	// 	case []interface{}:
	// 		value, err := toFloat64(t[4])
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		ohlcvClose = append(ohlcvClose, value)
	// 	}
	// }
	// inTimePeroid, ok := objects.ToInt(args[1])
	// if !ok {
	// 	return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	// }
	//
	// ret := indicators.Rsi(ohlcvClose, inTimePeroid)
	r := &objects.Array{}
	// for x := range ret {
	// 	r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	// }

	return r, nil
}

func rsi_exchange(args ...objects.Object) (objects.Object,error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName,ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, exchangeName)
	}
	inTimePeroid, ok := objects.ToInt(args[4])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	return nil, nil
}
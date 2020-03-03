package indicators

import (
	"fmt"
	"strconv"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var IndexModule = map[string]objects.Object{
	"mfi": &objects.UserFunction{Name: "mfi", Value: mfi},
}

func mfi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcHigh := objects.ToInterface(args[0])
	strNoWhiteSpace := convert.StripSpaceBuilder(ohlcHigh.(string))
	str := strings.Split(strNoWhiteSpace, ",")
	var ohlcHighSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, err
		}
		ohlcHighSlice[x] = v
	}

	ohlcLow := objects.ToInterface(args[1])
	strNoWhiteSpace = convert.StripSpaceBuilder(ohlcLow.(string))
	str = strings.Split(strNoWhiteSpace, ",")
	var ohlcLowSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, err
		}
		ohlcLowSlice[x] = v
	}

	ohlcClose := objects.ToInterface(args[2])
	strNoWhiteSpace = convert.StripSpaceBuilder(ohlcClose.(string))
	str = strings.Split(strNoWhiteSpace, ",")
	var ohlcCloseSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, err
		}
		ohlcCloseSlice[x] = v
	}

	ohlcVol := objects.ToInterface(args[3])
	strNoWhiteSpace = convert.StripSpaceBuilder(ohlcVol.(string))
	str = strings.Split(strNoWhiteSpace, ",")
	var ohlcVolSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, err
		}
		ohlcVolSlice[x] = v
	}

	inTimePeroid, ok := objects.ToInt(args[4])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	ret := indicators.Mfi(ohlcHighSlice, ohlcLowSlice, ohlcCloseSlice, ohlcVolSlice, inTimePeroid)
	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}
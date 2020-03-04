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

var RangeModule = map[string]objects.Object{
	"atr": &objects.UserFunction{Name: "atr", Value: atr},
}

func atr(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	strNoWhiteSpace := convert.StripSpaceBuilder(ohlcData.(string))
	str := strings.Split(strNoWhiteSpace, ",")
	var tempOHLCSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed,  v)
		}
		tempOHLCSlice[x] = v
	}

	ohlcData = objects.ToInterface(args[1])
	strNoWhiteSpace = convert.StripSpaceBuilder(ohlcData.(string))
	str = strings.Split(strNoWhiteSpace, ",")
	var tempOHLCVolSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed,  v)
		}
		tempOHLCVolSlice[x] = v
	}

	ohlcData = objects.ToInterface(args[2])
	strNoWhiteSpace = convert.StripSpaceBuilder(ohlcData.(string))
	str = strings.Split(strNoWhiteSpace, ",")
	var tempOHLCCloseSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed,  v)
		}
		tempOHLCCloseSlice[x] = v
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
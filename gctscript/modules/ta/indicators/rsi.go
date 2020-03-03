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

var RsiModule = map[string]objects.Object{
	"rsi": &objects.UserFunction{Name: "rsi", Value: Rsi},
}

func Rsi(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ohlcData := objects.ToInterface(args[0])
	inTimePeroid, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	strNoWhiteSpace := convert.StripSpaceBuilder(ohlcData.(string))
	str := strings.Split(strNoWhiteSpace, ",")
	var tempOHLCSlice = make([]float64, len(str))
	for x := range str {
		v, err := strconv.ParseFloat(str[x], 64)
		if err != nil {
			return nil, err
		}
		tempOHLCSlice[x] = v
	}
	ret := indicators.Rsi(tempOHLCSlice, inTimePeroid)

	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

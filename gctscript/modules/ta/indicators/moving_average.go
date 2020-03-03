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

var MovingAverageModule = map[string]objects.Object{
	"macd": &objects.UserFunction{Name: "macd", Value: macd},
	"ema":  &objects.UserFunction{Name: "ema", Value: ema},
	"sma":  &objects.UserFunction{Name: "sma", Value: sma},
}

func macd(args ...objects.Object) (objects.Object, error) {
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
			return nil, err
		}
		tempOHLCSlice[x] = v
	}

	inFastPeriod, ok := objects.ToInt(args[1])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inFastPeriod)
	}
	inSlowPeriod, ok := objects.ToInt(args[2])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inSlowPeriod)
	}
	inTimePeroid, ok := objects.ToInt(args[3])
	if !ok {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, inTimePeroid)
	}

	macd, macdSignal, macdHist := indicators.Macd(tempOHLCSlice, inFastPeriod, inSlowPeriod, inTimePeroid)

	retMACD := &objects.Array{}
	for x := range macd {
		retMACD.Value = append(retMACD.Value, &objects.Float{Value: macd[x]})
	}

	retMACDSignal := &objects.Array{}
	for x := range macdSignal {
		retMACDSignal.Value = append(retMACDSignal.Value, &objects.Float{Value: macdSignal[x]})
	}

	retMACDHist := &objects.Array{}
	for x := range macdHist {
		retMACDHist.Value = append(retMACDHist.Value, &objects.Float{Value: macdHist[x]})
	}

	ret := &objects.Array{}
	ret.Value = append(ret.Value, retMACD, retMACDSignal, retMACDSignal)
	return ret, nil
}

func ema(args ...objects.Object) (objects.Object, error) {
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
	ret := indicators.Ma(tempOHLCSlice, inTimePeroid, indicators.EMA)

	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

func sma(args ...objects.Object) (objects.Object, error) {
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
	ret := indicators.Ma(tempOHLCSlice, inTimePeroid, indicators.SMA)

	r := &objects.Array{}
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}
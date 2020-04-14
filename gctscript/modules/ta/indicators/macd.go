package indicators

import (
	"errors"
	"fmt"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// MACDModule MACD indicator commands
var MACDModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: macd},
}

func macd(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
	}

	r := &objects.Array{}
	if validator.IsTestExecution.Load() == true {
		return r, nil
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}

	var ohlcvClose []float64
	var allErrors []string
	for x := range ohlcvInputData {
		t := ohlcvInputData[x].([]interface{})
		value, err := toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvClose = append(ohlcvClose, value)
	}

	inFastPeriod, ok := objects.ToInt(args[1])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inFastPeriod))
	}

	inSlowPeriod, ok := objects.ToInt(args[2])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inSlowPeriod))
	}

	inTimePeriod, ok := objects.ToInt(args[3])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inTimePeriod))
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	macd, macdSignal, macdHist := indicators.Macd(ohlcvClose, inFastPeriod, inSlowPeriod, inTimePeriod)
	for x := range macdHist {
		tempMACD := &objects.Array{}
		tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macdHist[x]})
		if macd != nil {
			tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macd[x]})
		}
		if macdSignal != nil {
			tempMACD.Value = append(tempMACD.Value, &objects.Float{Value: macdSignal[x]})
		}
		r.Value = append(r.Value, tempMACD)
	}

	return r, nil
}

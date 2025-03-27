package indicators

import (
	"errors"
	"fmt"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// SMAModule simple moving average indicator commands
var SMAModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: sma},
}

// SimpleMovingAverage is the string constant
const SimpleMovingAverage = "Simple Moving Average"

// SMA defines a custom Simple Moving Average indicator tengo object type
type SMA struct {
	objects.Array
	Period int
}

// TypeName returns the name of the custom type.
func (o *SMA) TypeName() string {
	return SimpleMovingAverage
}

func sma(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	r := new(SMA)
	if validator.IsTestExecution.Load() == true {
		return r, nil
	}

	ohlcvInput := objects.ToInterface(args[0])
	ohlcvInputData, valid := ohlcvInput.([]any)
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
	}

	ohlcvClose := make([]float64, len(ohlcvInputData))
	var allErrors []string
	for x := range ohlcvInputData {
		t, ok := ohlcvInputData[x].([]any)
		if !ok {
			return nil, common.GetTypeAssertError("[]any", ohlcvInputData[x])
		}
		if len(t) < 5 {
			return nil, errors.New("ohlcvInputData invalid data length")
		}
		value, err := toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		ohlcvClose[x] = value
	}

	inTimePeriod, ok := objects.ToInt(args[1])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inTimePeriod))
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}
	r.Period = inTimePeriod
	ret := indicators.SMA(ohlcvClose, inTimePeriod)
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

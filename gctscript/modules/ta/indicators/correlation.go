package indicators

import (
	"errors"
	"fmt"
	"strings"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

// CorrelationCoefficientModule indicator commands
var CorrelationCoefficientModule = map[string]objects.Object{
	"calculate": &objects.UserFunction{Name: "calculate", Value: correlationCoefficient},
}

// CorrelationCoefficient is the string constant
const CorrelationCoefficient = "Correlation Coefficient"

// Correlation defines a custom correlation coefficient indicator tengo object
type Correlation struct {
	objects.Array
	Period int
}

// TypeName returns the name of the custom type.
func (c *Correlation) TypeName() string {
	return CorrelationCoefficient
}

func correlationCoefficient(args ...objects.Object) (objects.Object, error) {
	if len(args) != 3 {
		return nil, objects.ErrWrongNumArguments
	}

	r := new(Correlation)
	if validator.IsTestExecution.Load() == true {
		return r, nil
	}

	var allErrors []string
	ohlcvProcessor := func(args []objects.Object, idx int) ([]float64, error) {
		ohlcvInput := objects.ToInterface(args[idx])
		ohlcvInputData, valid := ohlcvInput.([]any)
		if !valid {
			return nil, fmt.Errorf(modules.ErrParameterConvertFailed, OHLCV)
		}

		var ohlcvClose []float64
		for x := range ohlcvInputData {
			t, ok := ohlcvInputData[x].([]any)
			if !ok {
				return nil, errors.New("ohlcvInputData type assert failed")
			}
			if len(t) < 5 {
				return nil, errors.New("ohlcvInputData invalid data length")
			}
			value, err := toFloat64(t[4])
			if err != nil {
				allErrors = append(allErrors, err.Error())
			}
			ohlcvClose = append(ohlcvClose, value)
		}
		return ohlcvClose, nil
	}

	closures1, err := ohlcvProcessor(args, 0)
	if err != nil {
		return nil, err
	}

	closures2, err := ohlcvProcessor(args, 1)
	if err != nil {
		return nil, err
	}

	inTimePeriod, ok := objects.ToInt(args[2])
	if !ok {
		allErrors = append(allErrors, fmt.Sprintf(modules.ErrParameterConvertFailed, inTimePeriod))
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	r.Period = inTimePeriod

	ret := indicators.CorrelationCoefficient(closures1, closures2, inTimePeriod)
	for x := range ret {
		r.Value = append(r.Value, &objects.Float{Value: ret[x]})
	}

	return r, nil
}

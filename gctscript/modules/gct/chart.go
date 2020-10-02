package gct

import (
	"errors"
	"fmt"
	"strings"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/charts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var chartsModule = map[string]objects.Object{
	"gen": &objects.UserFunction{Name: "gen", Value: GenerateChart},
}

func GenerateChart(args ...objects.Object) (objects.Object, error) {
	if len(args) == 0 {
		return nil, objects.ErrWrongNumArguments
	}

	chartName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, chartName)
	}

	writeFile, ok := objects.ToBool(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, writeFile)
	}

	input := objects.ToInterface(args[2])
	inputData, valid := input.([]interface{})
	if !valid {
		return nil, fmt.Errorf(modules.ErrParameterConvertFailed, "OHLCV")
	}

	ohlcvData := kline.Item{}
	var allErrors []string
	for x := range inputData {
		var tempCandleData = kline.Candle{}
		t := inputData[x].([]interface{})

		tz, ok := t[0].(int64)
		if !ok {
			allErrors = append(allErrors, "failed to convert time")
		}
		tempCandleData.Time = time.Unix(tz, 0)

		value, err := toFloat64(t[1])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}

		tempCandleData.Open = value

		value, err = toFloat64(t[2])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		tempCandleData.High = value

		value, err = toFloat64(t[3])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		tempCandleData.Low = value

		value, err = toFloat64(t[4])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		tempCandleData.Close = value

		value, err = toFloat64(t[5])
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
		tempCandleData.Volume = value
		ohlcvData.Candles = append(ohlcvData.Candles, tempCandleData)
	}

	if len(allErrors) > 0 {
		return nil, errors.New(strings.Join(allErrors, ", "))
	}

	chart := charts.New(chartName, "timeseries", OutputDir)
	var err error
	chart.WriteFile = writeFile
	chart.Data.Data, err = charts.KlineItemToSeriesData(ohlcvData)
	if err != nil {
		return nil, err
	}
	f, err := chart.Generate()
	if err != nil {
		return nil, err
	}
	if f != nil {
		err = f.Close()
		if err != nil {
			log.Error(log.GCTScriptMgr, err)
		}
	}

	return nil, nil
}

func toFloat64(data interface{}) (float64, error) {
	switch d := data.(type) {
	case float64:
		return d, nil
	case int:
		return float64(d), nil
	case int32:
		return float64(d), nil
	case int64:
		return float64(d), nil
	default:
		return 0, fmt.Errorf(modules.ErrParameterConvertFailed, d)
	}
}

package gct

import (
	"errors"
	"fmt"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
)

var commonModule = map[string]objects.Object{
	"writeascsv": &objects.UserFunction{Name: "writeascsv", Value: WriteAsCSV},
}

// WriteAsCSV takes in a slice matrix to save to file
func WriteAsCSV(args ...objects.Object) (objects.Object, error) {
	if len(args) == 0 {
		return nil, errors.New("cannot write to file, no data present")
	}

	var bucket [][]string
	var err error
	var target string
	for i := range args {
		if args[i] == nil {
			return nil, errors.New("data is nil")
		}
		var front bool
		var temp [][]string
		switch args[i].TypeName() {
		case indicators.AverageTrueRange:
			temp, err = convertATR(args[i])
		case indicators.BollingerBands:
			temp, err = convertBollingerBands(args[i])
		case indicators.ExponentialMovingAverage:
			temp, err = convertEMA(args[i])
		case indicators.MovingAverageConvergenceDivergence:
			temp, err = convertMACD(args[i])
		case indicators.MoneyFlowIndex:
			temp, err = convertMFI(args[i])
		case indicators.OnBalanceVolume:
			temp, err = convertOBV(args[i])
		case indicators.RelativeStrengthIndex:
			temp, err = convertRSI(args[i])
		case indicators.SimpleMovingAverage:
			temp, err = convertSMA(args[i])
		case indicators.OHLCV:
			temp, err = convertOHLCV(args[i])
			front = true
		case "string":
			var ok bool
			target, ok = objects.ToString(args[i])
			if !ok {
				return nil, errors.New("failed to convert incoming output to string")
			}
		default:
			err = fmt.Errorf("%s type is not handled", args[i].TypeName())
		}
		if err != nil {
			return nil, err
		}

		if front {
			var newBucket [][]string
			newBucket = append(newBucket, temp...)
			for x := range bucket {
				newBucket[x] = append(newBucket[x], bucket[x]...)
			}

			bucket = newBucket
			front = false
			continue
		}

		if len(bucket) == 0 {
			bucket = temp
		} else {
			for i := range temp {
				bucket[i] = append(bucket[i], temp[i]...)
			}
		}
	}
	return nil, file.WriteAsCSV(target+".csv", bucket)
}

func convertATR(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.AverageTrueRange,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.ATR)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}

		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertBollingerBands(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.BollingerBands, "", "",
		},
		{
			"Upper_Band", "Middle_Band", "Lower_band",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.BBands)
	if !ok {
		return nil, errors.New("casting failure")
	}

	for x := range obj.Value {
		element := obj.Value[x].Iterate()
		var upper, middle, lower string
		for i := 0; element.Next(); i++ {
			switch i {
			case 0:
				upper, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			case 1:
				middle, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			case 2:
				lower, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			}
		}
		bucket = append(bucket, []string{upper, middle, lower})
	}
	return bucket, nil
}

func convertEMA(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.ExponentialMovingAverage,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.EMA)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}
		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertMACD(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.MovingAverageConvergenceDivergence, "", "",
		},
		{
			"MACD", "Signal", "Histogram",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.MACD)
	if !ok {
		return nil, errors.New("casting failure")
	}

	for x := range obj.Value {
		element := obj.Value[x].Iterate()
		var macd, signal, hist string
		for i := 0; element.Next(); i++ {
			switch i {
			case 0:
				macd, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			case 1:
				signal, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			case 2:
				hist, ok = objects.ToString(element.Value())
				if !ok {
					return nil, errors.New("cannot convert object to string")
				}
			}
		}
		bucket = append(bucket, []string{macd, signal, hist})
	}
	return bucket, nil
}

func convertMFI(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.MoneyFlowIndex,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.MFI)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}
		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertOBV(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.OnBalanceVolume,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.OBV)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}
		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertRSI(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.RelativeStrengthIndex,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.RSI)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}
		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertSMA(a objects.Object) ([][]string, error) {
	var bucket = [][]string{
		{
			indicators.SimpleMovingAverage,
		},
		{
			"",
		},
	}

	obj, ok := objects.ToInterface(a).(*indicators.SMA)
	if !ok {
		return nil, errors.New("casting failure")
	}

	var val string
	for i := range obj.Value {
		val, ok = objects.ToString(obj.Value[i])
		if !ok {
			return nil, errors.New("cannot convert object to string")
		}
		bucket = append(bucket, []string{val})
	}
	return bucket, nil
}

func convertOHLCV(a objects.Object) ([][]string, error) {
	obj, ok := objects.ToInterface(a).(*OHLCV)
	if !ok {
		return nil, errors.New("casting failure")
	}

	exchange, ok := objects.ToString(obj.Value["exchange"])
	if !ok {
		return nil, errors.New("cannot convert object to string")
	}

	pair, ok := objects.ToString(obj.Value["pair"])
	if !ok {
		return nil, errors.New("cannot convert object to string")
	}

	asset, ok := objects.ToString(obj.Value["asset"])
	if !ok {
		return nil, errors.New("cannot convert object to string")
	}

	interval, ok := objects.ToString(obj.Value["intervals"])
	if !ok {
		return nil, errors.New("cannot convert object to string")
	}

	var bucket = [][]string{
		{
			indicators.OHLCV, "Exchange:" + exchange, pair, asset, interval, "",
		},
		{
			"Date", "Open", "High", "Low", "Close", "Volume",
		},
	}

	candles, ok := obj.Value["candles"]
	if !ok {
		return nil, errors.New("candles not found in object map")
	}

	data := candles.Iterate()

	for data.Next() {
		var date, open, high, low, closed, volume string
		candle := data.Value().Iterate()
		for i := 0; candle.Next(); i++ {
			switch i {
			case 0:
				date, ok = objects.ToString(candle.Value())
			case 1:
				open, ok = objects.ToString(candle.Value())
			case 2:
				high, ok = objects.ToString(candle.Value())
			case 3:
				low, ok = objects.ToString(candle.Value())
			case 4:
				closed, ok = objects.ToString(candle.Value())
			case 5:
				volume, ok = objects.ToString(candle.Value())
			}
			if !ok {
				return nil, errors.New("failed to convert")
			}
		}
		bucket = append(bucket, []string{date, volume, open, high, low, closed})
	}
	return bucket, nil
}

package gct

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var commonModule = map[string]objects.Object{
	"writeascsv": &objects.UserFunction{Name: "writeascsv", Value: WriteAsCSV},
}

// OutputDir is the default script output directory
var OutputDir string

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
		case indicators.CorrelationCoefficient:
			temp, err = convertCorrelationCoefficient(args[i])
		case indicators.OHLCV:
			temp, err = convertOHLCV(args[i])
			front = true
		case "scriptContext":
			if target != "" {
				return nil, fmt.Errorf("filename already set, extra string %v cannot be processed", args[i])
			}
			scriptCtx, ok := objects.ToInterface(args[i]).(*Context)
			if !ok {
				return nil, common.GetTypeAssertError("*gct.Context", args[i])
			}

			scriptDetails, ok := scriptCtx.Value["script"]
			if !ok {
				return nil, errors.New("no script details")
			}

			target, ok = objects.ToString(scriptDetails)
			if !ok {
				return nil, errors.New("failed to convert incoming output to string")
			}

			target = processTarget(target)
		case "string":
			if target != "" {
				return nil, fmt.Errorf("filename already set, extra string %v cannot be processed", args[i])
			}
			var ok bool
			target, ok = objects.ToString(args[i])
			if !ok {
				return nil, errors.New("failed to convert incoming output to string")
			}

			if target == "" {
				return nil, errors.New("script context details not specified")
			}

			target = processTarget(target)
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

	if target == "" {
		return nil, errors.New("filename unset please set in writeascsv as ctx or client defined filename")
	}

	err = file.WriteAsCSV(target, bucket)
	if err != nil {
		return nil, err
	}

	log.Debugf(log.GCTScriptMgr,
		"CSV file successfully saved to: %s",
		target)
	return nil, nil
}

func convertATR(a objects.Object) ([][]string, error) {
	obj, ok := objects.ToInterface(a).(*indicators.ATR)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.AverageTrueRange,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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
	obj, ok := objects.ToInterface(a).(*indicators.BBands)
	if !ok {
		return nil, errors.New("casting failure")
	}

	upperS := fmt.Sprintf("Upper_Band (NBDevUp:%f)", obj.STDDevUp)
	lowerS := fmt.Sprintf("Lower_band(NBDevDown:%f)", obj.STDDevDown)
	middleS := fmt.Sprintf("Middle_Band (Period:%d)", obj.Period)
	MAType := "MA_TYPE:SMA"
	if obj.MAType != 0 {
		MAType = "MA_TYPE:EMA"
	}

	bucket := [][]string{
		{
			indicators.BollingerBands, "", MAType,
		},
		{
			upperS, middleS, lowerS,
		},
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
	obj, ok := objects.ToInterface(a).(*indicators.EMA)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.ExponentialMovingAverage,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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
	obj, ok := objects.ToInterface(a).(*indicators.MACD)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.MovingAverageConvergenceDivergence,
			fmt.Sprintf("Period:%d Fast:%d Slow:%d",
				obj.Period,
				obj.PeriodFast,
				obj.PeriodSlow),
			"",
		},
		{
			"MACD", "Signal", "Histogram",
		},
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
	obj, ok := objects.ToInterface(a).(*indicators.MFI)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.MoneyFlowIndex,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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
	bucket := [][]string{
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
	obj, ok := objects.ToInterface(a).(*indicators.RSI)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.RelativeStrengthIndex,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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
	obj, ok := objects.ToInterface(a).(*indicators.SMA)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.SimpleMovingAverage,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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

func convertCorrelationCoefficient(a objects.Object) ([][]string, error) {
	obj, ok := objects.ToInterface(a).(*indicators.Correlation)
	if !ok {
		return nil, errors.New("casting failure")
	}

	bucket := [][]string{
		{
			indicators.CorrelationCoefficient,
		},
		{
			fmt.Sprintf("Period:%d", obj.Period),
		},
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

	bucket := [][]string{
		{
			indicators.OHLCV, "Exchange:" + exchange, pair, asset, interval, "",
		},
		{
			"Date", "Volume", "Open", "High", "Low", "Close",
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

func processTarget(target string) string {
	// Removes file transversal
	target = filepath.Base(target)

	// checks to see if file is context defined, if not it will allow
	// a client defined filename and append a date, forces the use of
	// .csv file extension
	switch {
	case filepath.Ext(target) != ".csv" && strings.Contains(target, common.GctExt):
		target += ".csv"
	case filepath.Ext(target) == ".csv":
		s := strings.Split(target, ".")
		if len(s) == 2 {
			target = s[0] + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".csv"
		}
	default:
		target += "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".csv"
	}
	return filepath.Join(OutputDir, target)
}

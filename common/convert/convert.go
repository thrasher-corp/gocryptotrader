package convert

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FloatFromString format
func FloatFromString(raw interface{}) (float64, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	flt, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("could not convert value: %s Error: %s", str, err)
	}
	return flt, nil
}

// IntFromString format
func IntFromString(raw interface{}) (int, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	n, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("unable to parse as int: %T", raw)
	}
	return n, nil
}

// Int64FromString format
func Int64FromString(raw interface{}) (int64, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse as int64: %T", raw)
	}
	return n, nil
}

// TimeFromUnixTimestampFloat format
func TimeFromUnixTimestampFloat(raw interface{}) (time.Time, error) {
	ts, ok := raw.(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("unable to parse, value not float64: %T", raw)
	}
	return time.Unix(0, int64(ts)*int64(time.Millisecond)), nil
}

// UnixTimestampToTime returns time.time
func UnixTimestampToTime(timeint64 int64) time.Time {
	return time.Unix(timeint64, 0)
}

// UnixTimestampStrToTime returns a time.time and an error
func UnixTimestampStrToTime(timeStr string) (time.Time, error) {
	i, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(i, 0), nil
}

// UnixMillis converts a UnixNano timestamp to milliseconds
func UnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// RecvWindow converts a supplied time.Duration to milliseconds
func RecvWindow(d time.Duration) int64 {
	return int64(d) / int64(time.Millisecond)
}

// SplitFloatDecimals takes in a float64 and splits
// the decimals into their own integers
// Warning. Passing in numbers with many decimals
// can lead to a loss of accuracy
func SplitFloatDecimals(input float64) (baseNum, decimalNum int64, err error) {
	if input > float64(math.MaxInt64) {
		return 0, 0, errors.New("number too large to split into integers")
	}
	if input == float64(int64(input)) {
		return int64(input), 0, nil
	}
	decStr := strconv.FormatFloat(input, 'f', -1, 64)
	splitNum := strings.Split(decStr, ".")
	baseNum, err = strconv.ParseInt(splitNum[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	decimalNum, err = strconv.ParseInt(splitNum[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	if baseNum < 0 {
		decimalNum *= -1
	}
	return baseNum, decimalNum, nil
}

// BoolPtr takes in boolen condition and returns pointer version of it
func BoolPtr(condition bool) *bool {
	b := condition
	return &b
}

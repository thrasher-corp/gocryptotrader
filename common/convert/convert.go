package convert

import (
	"fmt"
	"math"
	"strconv"
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

// TimeFromUnixTimestampDecimal converts a unix timestamp in decimal form to
// a time.Time
func TimeFromUnixTimestampDecimal(input float64) time.Time {
	i, f := math.Modf(input)
	return time.Unix(int64(i), int64(f*(1e9)))
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

// BoolPtr takes in boolen condition and returns pointer version of it
func BoolPtr(condition bool) *bool {
	b := condition
	return &b
}

// UnixMillisToNano converts Unix milli time to UnixNano
func UnixMillisToNano(milli int64) int64 {
	return milli * int64(time.Millisecond)
}

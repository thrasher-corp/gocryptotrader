package convert

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
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
	return time.UnixMilli(int64(ts)), nil
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

// BoolPtr takes in boolen condition and returns pointer version of it
func BoolPtr(condition bool) *bool {
	b := condition
	return &b
}

// IntToCommaSeparatedString converts an int to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func IntToCommaSeparatedString(number int64, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	str := fmt.Sprintf("%v", number)
	return numberToCommaSeparatedString(str, 0, "", thousandsSep, neg)
}

// FloatToCommaSeparatedString converts a float to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func FloatToCommaSeparatedString(number float64, decimals uint, decPoint, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	dec := int(decimals)
	str := fmt.Sprintf("%."+strconv.Itoa(dec)+"F", number)
	return numberToCommaSeparatedString(str, dec, decPoint, thousandsSep, neg)
}

// DecimalToCommaSeparatedString converts a decimal number to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func DecimalToCommaSeparatedString(number decimal.Decimal, rounding int32, decPoint, thousandsSep string) string {
	neg := false
	if number.LessThan(decimal.Zero) {
		number = number.Abs()
		neg = true
	}
	str := fmt.Sprintf("%v", number.Round(rounding))
	return numberToCommaSeparatedString(str, int(rounding), decPoint, thousandsSep, neg)
}

func numberToCommaSeparatedString(str string, dec int, decPoint, thousandsSep string, neg bool) string {
	prefix, suffix := "", ""
	if len(str)-(dec+1) < 0 {
		dec = 0
	}
	if dec > 0 {
		prefix = str[:len(str)-(dec+1)]
		suffix = str[len(str)-dec:]
	} else {
		prefix = str
	}
	sep := []byte(thousandsSep)
	n, l1, l2 := 0, len(prefix), len(sep)
	// thousands sep num
	c := (l1 - 1) / 3
	tmp := make([]byte, l2*c+l1)
	pos := len(tmp) - 1
	for i := l1 - 1; i >= 0; i, n, pos = i-1, n+1, pos-1 {
		if l2 > 0 && n > 0 && n%3 == 0 {
			for j := range sep {
				tmp[pos] = sep[l2-j-1]
				pos--
			}
		}
		tmp[pos] = prefix[i]
	}
	s := string(tmp)
	if dec > 0 {
		s += decPoint + suffix
	}
	if neg {
		s = "-" + s
	}

	return s
}

package convert

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const jsonStringIdent = `"` // immutable byte sequence

var errUnhandledType = errors.New("unhandled type")

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

// BoolPtr takes in boolean condition and returns pointer version of it
func BoolPtr(condition bool) *bool {
	b := condition
	return &b
}

// IntToHumanFriendlyString converts an int to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func IntToHumanFriendlyString(number int64, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	str := fmt.Sprintf("%v", number)
	return numberToHumanFriendlyString(str, 0, "", thousandsSep, neg)
}

// FloatToHumanFriendlyString converts a float to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func FloatToHumanFriendlyString(number float64, decimals uint, decPoint, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	dec := int(decimals)
	str := fmt.Sprintf("%."+strconv.Itoa(dec)+"F", number)
	return numberToHumanFriendlyString(str, dec, decPoint, thousandsSep, neg)
}

// DecimalToHumanFriendlyString converts a decimal number to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func DecimalToHumanFriendlyString(number decimal.Decimal, rounding int, decPoint, thousandsSep string) string {
	neg := false
	if number.LessThan(decimal.Zero) {
		number = number.Abs()
		neg = true
	}
	str := number.String()
	if rnd := strings.Split(str, "."); len(rnd) == 1 {
		rounding = 0
	} else if len(rnd[1]) < rounding {
		rounding = len(rnd[1])
	}
	return numberToHumanFriendlyString(number.StringFixed(int32(rounding)), rounding, decPoint, thousandsSep, neg)
}

func numberToHumanFriendlyString(str string, dec int, decPoint, thousandsSep string, neg bool) string {
	var prefix, suffix string
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

// InterfaceToFloat64OrZeroValue returns the type assertion value or variable zero value
func InterfaceToFloat64OrZeroValue(r interface{}) float64 {
	if v, ok := r.(float64); ok {
		return v
	}
	return 0
}

// InterfaceToIntOrZeroValue returns the type assertion value or variable zero value
func InterfaceToIntOrZeroValue(r interface{}) int {
	if v, ok := r.(int); ok {
		return v
	}
	return 0
}

// InterfaceToStringOrZeroValue returns the type assertion value or variable zero value
func InterfaceToStringOrZeroValue(r interface{}) string {
	if v, ok := r.(string); ok {
		return v
	}
	return ""
}

// StringToFloat64 is a float64 that unmarshals from a string. This is useful
// for APIs that return numbers as strings and return an empty string instead of
// 0.
type StringToFloat64 float64

// UnmarshalJSON implements the json.Unmarshaler interface.
// This implementation is slightly more performant than calling json.Unmarshal
// again.
func (f *StringToFloat64) UnmarshalJSON(data []byte) error {
	if !bytes.HasPrefix(data, []byte(jsonStringIdent)) {
		return fmt.Errorf("%w: %s", errUnhandledType, string(data))
	}

	data = data[1 : len(data)-1] // Remove quotes
	if len(data) == 0 {
		*f = StringToFloat64(0)
		return nil
	}

	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}

	*f = StringToFloat64(val)
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (f StringToFloat64) MarshalJSON() ([]byte, error) {
	if f == 0 {
		return []byte(jsonStringIdent + jsonStringIdent), nil
	}
	val := strconv.FormatFloat(float64(f), 'f', -1, 64)
	return []byte(jsonStringIdent + val + jsonStringIdent), nil
}

// Float64 returns the float64 value of the FloatString.
func (f StringToFloat64) Float64() float64 {
	return float64(f)
}

// Decimal returns the decimal value of the FloatString
// Warning: this does not handle big numbers as the underlying
// is still a float
func (f StringToFloat64) Decimal() decimal.Decimal {
	return decimal.NewFromFloat(float64(f))
}

// ExchangeTime provides timestamp to time conversion method.
type ExchangeTime time.Time

// UnmarshalJSON is custom type json unmarshaller for ExchangeTime
func (k *ExchangeTime) UnmarshalJSON(data []byte) error {
	var timestamp interface{}
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	var standard int64
	switch value := timestamp.(type) {
	case string:
		if value == "" {
			// Setting the time to zero value because some timestamp fields could return an empty string while there is no error
			// So, in such cases, Time returns zero timestamp.
			break
		}
		standard, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
	case int64:
		standard = value
	case float64:
		// Warning: converting float64 to int64 instance may create loss of precision in the timestamp information.
		// be aware or consider customizing this section if found necessary.
		standard = int64(value)
	case nil:
		// for some exchange timestamp fields, if the timestamp information is not specified,
		// the data is 'nil' instead of zero value string or integer value.
	default:
		return fmt.Errorf("unsupported timestamp type %T", timestamp)
	}

	switch {
	case standard == 0:
		*k = ExchangeTime(time.Time{})
	case standard >= 1e13:
		*k = ExchangeTime(time.Unix(standard/1e9, standard%1e9))
	case standard > 9999999999:
		*k = ExchangeTime(time.UnixMilli(standard))
	default:
		*k = ExchangeTime(time.Unix(standard, 0))
	}
	return nil
}

// Time returns a time.Time instance from ExchangeTime instance object.
func (k ExchangeTime) Time() time.Time {
	return time.Time(k)
}

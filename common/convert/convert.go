package convert

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// FloatFromString format
func FloatFromString(raw any) (float64, error) {
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
func IntFromString(raw any) (int, error) {
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
func Int64FromString(raw any) (int64, error) {
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
	return numberToHumanFriendlyString(strconv.FormatInt(number, 10), 0, "", thousandsSep, neg)
}

// FloatToHumanFriendlyString converts a float to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func FloatToHumanFriendlyString(number float64, decimals uint, decPoint, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	str := fmt.Sprintf("%."+strconv.FormatUint(uint64(decimals), 10)+"F", number)
	return numberToHumanFriendlyString(str, decimals, decPoint, thousandsSep, neg)
}

// DecimalToHumanFriendlyString converts a decimal number to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func DecimalToHumanFriendlyString(number decimal.Decimal, rounding uint, decPoint, thousandsSep string) string {
	neg := false
	if number.LessThan(decimal.Zero) {
		number = number.Abs()
		neg = true
	}
	str := number.String()
	if rnd := strings.Split(str, "."); len(rnd) == 1 {
		rounding = 0
	} else if uint(len(rnd[1])) < rounding {
		rounding = uint(len(rnd[1]))
	}

	if rounding > math.MaxInt32 {
		rounding = math.MaxInt32 // Not feasible to test due to the size of the number
	}

	return numberToHumanFriendlyString(number.StringFixed(int32(rounding)), rounding, decPoint, thousandsSep, neg) //nolint:gosec // Checked above
}

func numberToHumanFriendlyString(str string, dec uint, decPoint, thousandsSep string, neg bool) string {
	var prefix, suffix string
	if dec > 0 && (dec)+1 > uint(len(str)) {
		dec = 0
	}
	if dec > 0 {
		prefix = str[:len(str)-(int(dec)+1)]
		suffix = str[len(str)-int(dec):]
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
func InterfaceToFloat64OrZeroValue(r any) float64 {
	if v, ok := r.(float64); ok {
		return v
	}
	return 0
}

// InterfaceToIntOrZeroValue returns the type assertion value or variable zero value
func InterfaceToIntOrZeroValue(r any) int {
	if v, ok := r.(int); ok {
		return v
	}
	return 0
}

// InterfaceToStringOrZeroValue returns the type assertion value or variable zero value
func InterfaceToStringOrZeroValue(r any) string {
	if v, ok := r.(string); ok {
		return v
	}
	return ""
}

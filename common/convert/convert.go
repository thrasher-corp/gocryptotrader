// Package convert formats numeric types into human friendly strings
package convert

import (
	"math"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// IntToHumanFriendlyString converts an int to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func IntToHumanFriendlyString(number int64, thousandsSep string) string {
	// The sign is split off the formatted text rather than the input because negating math.MinInt64 overflows
	magnitude, neg := strings.CutPrefix(strconv.FormatInt(number, 10), "-")
	return numberToHumanFriendlyString(magnitude, 0, "", thousandsSep, neg)
}

// FloatToHumanFriendlyString converts a float to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func FloatToHumanFriendlyString(number float64, decimals uint, decPoint, thousandsSep string) string {
	neg := number < 0
	number = math.Abs(number)
	decimals = min(decimals, math.MaxInt32) // strconv.FormatFloat takes an int precision
	return numberToHumanFriendlyString(strconv.FormatFloat(number, 'f', int(decimals), 64), decimals, decPoint, thousandsSep, neg)
}

// DecimalToHumanFriendlyString converts a decimal number to a comma separated string at the thousand point
// eg 1000 becomes 1,000
func DecimalToHumanFriendlyString(number decimal.Decimal, rounding uint, decPoint, thousandsSep string) string {
	neg := number.IsNegative()
	if neg {
		number = number.Abs()
	}
	if _, frac, ok := strings.Cut(number.String(), "."); ok {
		rounding = min(rounding, uint(len(frac)))
	} else {
		rounding = 0
	}
	rounding = min(rounding, math.MaxInt32) // decimal.StringFixed takes an int32 place count
	return numberToHumanFriendlyString(number.StringFixed(int32(rounding)), rounding, decPoint, thousandsSep, neg)
}

// numberToHumanFriendlyString groups the integer component of an already formatted number with thousandsSep,
// where dec is the number of decimal places str carries
func numberToHumanFriendlyString(str string, dec uint, decPoint, thousandsSep string, neg bool) string {
	if dec+1 > uint(len(str)) {
		dec = 0
	}
	integer, fraction := str, ""
	if dec > 0 {
		integer, fraction = str[:len(str)-int(dec)-1], str[len(str)-int(dec):]
	}

	groups := max(len(integer)-1, 0) / 3
	lead := len(integer) - groups*3 // The leading group carries the remainder so every following group is exactly three digits

	var b strings.Builder
	b.Grow(len(str) + groups*len(thousandsSep) + len(decPoint) + 1)
	if neg {
		b.WriteByte('-')
	}
	b.WriteString(integer[:lead])
	for i := lead; i < len(integer); i += 3 {
		b.WriteString(thousandsSep)
		b.WriteString(integer[i : i+3])
	}
	if dec > 0 {
		b.WriteString(decPoint)
		b.WriteString(fraction)
	}
	return b.String()
}

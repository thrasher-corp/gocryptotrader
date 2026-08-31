//go:build udecimal_on

package decimal

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "quagmt/udecimal"

const maxPrecision = 19

// mustParseUdecimal reconstructs values beyond udecimal's 200-character parser
// limit because arithmetic can produce larger values that must still round-trip.
func mustParseUdecimal(value string) udecimal.Decimal {
	if len(value) <= maxStringDigits {
		return udecimal.MustParse(value)
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	decimalPoint := strings.IndexByte(value, '.')
	fractionalDigits := 0
	if decimalPoint >= 0 {
		fractionalDigits = len(value) - decimalPoint - 1
		value = value[:decimalPoint] + value[decimalPoint+1:]
	}
	result := udecimal.Zero
	for value != "" {
		chunkLength := min(len(value), maxStringDigits)
		result = result.Mul(powerOfTenUdecimal(chunkLength)).Add(udecimal.MustParse(value[:chunkLength]))
		value = value[chunkLength:]
	}
	if fractionalDigits > 0 {
		result = result.MustDiv(powerOfTenUdecimal(fractionalDigits))
	}
	if negative {
		return result.Neg()
	}
	return result
}

// newFromFloatUdecimal accepts every finite float64 because udecimal's native
// constructor uses fixed notation and rejects large finite values over its
// parser length limit.
func newFromFloatUdecimal(value float64) udecimal.Decimal {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		panic(fmt.Errorf("%w: non-finite float %v", ErrInvalidDecimal, value))
	}
	formatted := strconv.FormatFloat(value, 'g', -1, 64)
	exponentIndex := strings.IndexAny(formatted, "eE")
	if exponentIndex < 0 {
		normalised, err := normalise(formatted, true)
		if err != nil {
			panic(err)
		}
		return mustParseUdecimal(normalised)
	}
	exponent, _ := strconv.Atoi(formatted[exponentIndex+1:])
	if exponent < 0 {
		normalised, err := normalise(formatted, true)
		if err != nil {
			panic(err)
		}
		return mustParseUdecimal(normalised)
	}
	mantissa, err := normalise(formatted[:exponentIndex], true)
	if err != nil {
		panic(err)
	}
	return mustParseUdecimal(mantissa).Mul(powerOfTenUdecimal(exponent))
}

// powerOfTenUdecimal builds large powers in parser-sized chunks because
// udecimal.MustParse rejects strings longer than 200 characters.
func powerOfTenUdecimal(exponent int) udecimal.Decimal {
	result := udecimal.One
	for exponent > 0 {
		chunkLength := min(exponent, maxStringDigits-1)
		result = result.Mul(udecimal.MustParse("1" + strings.Repeat("0", chunkLength)))
		exponent -= chunkLength
	}
	return result
}

// normaliseUdecimalPrecision applies udecimal's native 19-place precision limit after
// the package has expanded scientific notation unsupported by udecimal.Parse.
func normaliseUdecimalPrecision(digits string, scale int64, truncate bool, original string) (normalised string, normalisedScale int64, err error) {
	if digits == "" {
		return "0", 0, nil
	}
	if scale <= maxPrecision {
		return digits, scale, nil
	}
	if !truncate {
		return "", 0, fmt.Errorf("%w: %q requires %d fractional digits", ErrPrecisionOutOfRange, original, scale)
	}
	excess := int(scale - maxPrecision)
	if excess >= len(digits) {
		return "0", 0, nil
	}
	digits = digits[:len(digits)-excess]
	scale = maxPrecision
	for scale > 0 && strings.HasSuffix(digits, "0") {
		digits = strings.TrimSuffix(digits, "0")
		scale--
	}
	return digits, scale, nil
}

// powUdecimal rejects fractional exponents because udecimal.PowToIntPart
// silently truncates them, which would make the existing Pow API misleading.
func powUdecimal(value, exponent udecimal.Decimal) (udecimal.Decimal, error) {
	if !exponent.Trunc(0).Equal(exponent) {
		return udecimal.Decimal{}, ErrInvalidDecimal
	}
	if value.IsZero() {
		return udecimal.Decimal{}, nil
	}
	return value.PowToIntPart(exponent)
}

// roundUdecimal handles negative places because udecimal.RoundHAZ accepts only
// an unsigned fractional precision while the existing Round API accepts int32.
func roundUdecimal(value udecimal.Decimal, places int32) udecimal.Decimal {
	if places >= 0 {
		if places >= maxPrecision {
			return value
		}
		return value.RoundHAZ(uint8(places))
	}

	zeroes := -int64(places)
	if zeroes >= maxStringDigits {
		panic(fmt.Errorf("%w: rounding precision %d", ErrPrecisionOutOfRange, places))
	}
	scale := udecimal.MustParse("1" + strings.Repeat("0", int(zeroes)))
	return value.MustDiv(scale).RoundHAZ(0).Mul(scale)
}

//go:build udecimal_on

package decimal

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "quagmt/udecimal"

const (
	limitedPrecision = true
	maxPrecision     = 19
	// divisionPrecision matches shopspring.DivisionPrecision's default.
	divisionPrecision = 16
)

type backendDecimal = udecimal.Decimal

func mustParseBackend(value string) backendDecimal {
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
		result = result.Mul(powerOfTenBackend(chunkLength)).Add(udecimal.MustParse(value[:chunkLength]))
		value = value[chunkLength:]
	}
	if fractionalDigits > 0 {
		result = result.MustDiv(powerOfTenBackend(fractionalDigits))
	}
	if negative {
		return result.Neg()
	}
	return result
}

func newFromFloatBackend(value float64) backendDecimal {
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
		return mustParseBackend(normalised)
	}
	exponent, _ := strconv.Atoi(formatted[exponentIndex+1:])
	if exponent < 0 {
		normalised, err := normalise(formatted, true)
		if err != nil {
			panic(err)
		}
		return mustParseBackend(normalised)
	}
	mantissa, err := normalise(formatted[:exponentIndex], true)
	if err != nil {
		panic(err)
	}
	return mustParseBackend(mantissa).Mul(powerOfTenBackend(exponent))
}

func powerOfTenBackend(exponent int) backendDecimal {
	result := udecimal.One
	for exponent > 0 {
		chunkLength := min(exponent, maxStringDigits-1)
		result = result.Mul(udecimal.MustParse("1" + strings.Repeat("0", chunkLength)))
		exponent -= chunkLength
	}
	return result
}

func normalisePrecision(digits string, scale int64, truncate bool, original string) (normalised string, normalisedScale int64, err error) {
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

func mulBackend(left, right backendDecimal) backendDecimal {
	return left.Mul(right)
}

func divBackend(left, right backendDecimal) (backendDecimal, error) {
	if right.IsZero() {
		return backendDecimal{}, udecimal.ErrDivideByZero
	}
	leftRat, _ := new(big.Rat).SetString(left.String())
	rightRat, _ := new(big.Rat).SetString(right.String())
	return mustParseBackend(new(big.Rat).Quo(leftRat, rightRat).FloatString(divisionPrecision)), nil
}

func modBackend(left, right backendDecimal) (backendDecimal, error) {
	return left.Mod(right)
}

func powBackend(value, exponent backendDecimal) (backendDecimal, error) {
	if !exponent.Trunc(0).Equal(exponent) {
		return backendDecimal{}, ErrInvalidDecimal
	}
	if value.IsZero() {
		return backendDecimal{}, nil
	}
	return value.PowToIntPart(exponent)
}

func isPositiveBackend(value backendDecimal) bool {
	return value.IsPos()
}

func isNegativeBackend(value backendDecimal) bool {
	return value.IsNeg()
}

func roundBackend(value backendDecimal, places int32) backendDecimal {
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

func truncateBackend(value backendDecimal, precision int32) backendDecimal {
	if precision < 0 {
		return value
	}
	if precision >= maxPrecision {
		return value
	}
	return value.Trunc(uint8(precision))
}

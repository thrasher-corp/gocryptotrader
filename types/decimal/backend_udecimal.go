//go:build udecimal_on

package decimal

import (
	"fmt"
	"strings"

	udecimal "github.com/quagmt/udecimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "quagmt/udecimal"

const (
	limitedPrecision = true
	maxPrecision     = 19
)

type backendDecimal = udecimal.Decimal

func mustParseBackend(value string) backendDecimal {
	return udecimal.MustParse(value)
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
	result, err := left.Div(right)
	if err != nil {
		return backendDecimal{}, err
	}
	return result.RoundHAZ(divisionPrecision), nil
}

func modBackend(left, right backendDecimal) (backendDecimal, error) {
	return left.Mod(right)
}

func powBackend(value, exponent backendDecimal) (backendDecimal, error) {
	if value.IsZero() {
		return backendDecimal{}, nil
	}
	if !exponent.Trunc(0).Equal(exponent) {
		return backendDecimal{}, ErrInvalidDecimal
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
	if zeroes >= maxStringLength {
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

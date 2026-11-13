//go:build udecimal_on

package decimal

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "quagmt/udecimal"

const (
	maxPrecision             = 19
	maxStringDigits          = 200
	maxEncodedBinaryBytes    = 255
	u128MaxDecimalDigits     = 39
	udecimalDivisionOverflow = "runtime error: integer overflow"
)

var (
	errInvalidDecimal      = errors.New("invalid decimal")
	errPrecisionOutOfRange = errors.New("decimal precision out of range")
)

// udecimalBigIntDivisionFactor exceeds u128 while retaining zero precision.
// Multiplying both operands by it preserves their quotient and precision but
// makes udecimal select its native big.Int division path.
var udecimalBigIntDivisionFactor = udecimal.MustParse("1" + strings.Repeat("0", u128MaxDecimalDigits))

// parseUdecimal reconstructs values beyond udecimal's 200-character parser
// limit because internal arithmetic can produce larger values.
func parseUdecimal(value string) (udecimal.Decimal, error) {
	if len(value) <= maxStringDigits {
		return udecimal.Parse(value)
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
		chunk, err := udecimal.Parse(value[:chunkLength])
		if err != nil {
			return udecimal.Decimal{}, err
		}
		result = result.Mul(powerOfTenUdecimal(chunkLength)).Add(chunk)
		value = value[chunkLength:]
	}
	if fractionalDigits > 0 {
		result = result.MustDiv(powerOfTenUdecimal(fractionalDigits))
	}
	if negative {
		return result.Neg(), nil
	}
	return result, nil
}

// divideUdecimal preserves udecimal's native zero-allocation division path
// while repairing a specific valid-operand overflow in its u128 implementation.
func divideUdecimal(left, right udecimal.Decimal) (result udecimal.Decimal) {
	// udecimal v1.10.1's u128 fast division can violate bits.Div64's divisor
	// precondition for a narrow operand range. Recover only that known runtime
	// failure so divide-by-zero and any unrelated backend panic remain visible.
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		runtimeError, ok := recovered.(runtime.Error)
		if !ok || runtimeError.Error() != udecimalDivisionOverflow {
			panic(recovered)
		}

		// Multiplying both operands by the same precision-zero factor leaves the
		// quotient unchanged. Because the factor exceeds u128, both products use
		// udecimal's big.Int representation and MustDiv bypasses the faulty fast
		// path. Its native fallback retains sign handling and 19-place truncation.
		scaledLeft := left.Mul(udecimalBigIntDivisionFactor)
		scaledRight := right.Mul(udecimalBigIntDivisionFactor)
		result = scaledLeft.MustDiv(scaledRight)
	}()
	return left.MustDiv(right)
}

// moduloUdecimal preserves udecimal's native remainder path while repairing
// the same valid-operand overflow as divideUdecimal's u128 implementation.
func moduloUdecimal(left, right udecimal.Decimal) (result udecimal.Decimal) {
	// MustMod obtains the quotient through udecimal's u128 fast division, so it
	// can reach the same bits.Div64 precondition failure as MustDiv. Recover only
	// that known runtime error; zero divisors and unrelated panics must propagate.
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		runtimeError, ok := recovered.(runtime.Error)
		if !ok || runtimeError.Error() != udecimalDivisionOverflow {
			panic(recovered)
		}

		// Scaling both operands by the same power preserves the remainder after
		// scaling it back down. The factor exceeds u128, forcing MustMod onto the
		// backend's big.Int path without reimplementing sign or precision rules.
		scaledLeft := left.Mul(udecimalBigIntDivisionFactor)
		scaledRight := right.Mul(udecimalBigIntDivisionFactor)
		result = scaledLeft.MustMod(scaledRight).MustDiv(udecimalBigIntDivisionFactor)
	}()
	return left.MustMod(right)
}

// newUdecimalFromFloat accepts every finite float64 because udecimal's native
// constructor uses fixed notation and rejects large finite values over its
// parser length limit.
func newUdecimalFromFloat(value float64) (udecimal.Decimal, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return udecimal.Decimal{}, fmt.Errorf("%w: non-finite float %v", errInvalidDecimal, value)
	}
	formatted := strconv.FormatFloat(value, 'g', -1, 64)
	exponentIndex := strings.IndexAny(formatted, "eE")
	if exponentIndex < 0 {
		normalised, err := normalise(formatted, true)
		if err != nil {
			return udecimal.Decimal{}, err
		}
		return parseUdecimal(normalised)
	}
	// FormatFloat guarantees a valid base-10 exponent after e or E.
	exponent, _ := strconv.Atoi(formatted[exponentIndex+1:])
	if exponent < 0 {
		normalised, err := normalise(formatted, true)
		if err != nil {
			return udecimal.Decimal{}, err
		}
		return parseUdecimal(normalised)
	}
	mantissa, err := normalise(formatted[:exponentIndex], true)
	if err != nil {
		return udecimal.Decimal{}, err
	}
	result, err := parseUdecimal(mantissa)
	if err != nil {
		return udecimal.Decimal{}, err
	}
	return result.Mul(powerOfTenUdecimal(exponent)), nil
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

// applyUdecimalPrecision applies udecimal's native 19-place precision limit
// after scientific notation has been converted into digits and a scale.
func applyUdecimalPrecision(digits string, scale int64, truncate bool, original string) (normalised string, normalisedScale int64, err error) {
	if digits == "" {
		return "0", 0, nil
	}
	if scale <= maxPrecision {
		return digits, scale, nil
	}
	if !truncate {
		return "", 0, fmt.Errorf("%w: %q requires %d fractional digits", errPrecisionOutOfRange, original, scale)
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
		return udecimal.Decimal{}, fmt.Errorf("%w: fractional exponent %s", errInvalidDecimal, exponent)
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
	integerText := strings.TrimPrefix(value.String(), "-")
	if decimalPoint := strings.IndexByte(integerText, '.'); decimalPoint >= 0 {
		integerText = integerText[:decimalPoint]
	}
	integerDigits := len(strings.TrimLeft(integerText, "0"))
	if zeroes > int64(integerDigits) {
		// The rounding position is left of every significant digit, so the result
		// is zero. Besides avoiding unnecessary work, this makes extreme int32
		// places safe without attempting to construct an enormous power of ten.
		return udecimal.Zero
	}
	scale := powerOfTenUdecimal(int(zeroes))
	return value.MustDiv(scale).RoundHAZ(0).Mul(scale)
}

// parseDecimalParts validates a decimal string and converts its mantissa and
// exponent into unsigned significant digits, a fractional scale and a sign.
func parseDecimalParts(value string) (digits string, scale int64, negative bool, err error) {
	if value == "" {
		return "", 0, false, fmt.Errorf("%w: empty input", errInvalidDecimal)
	}

	mantissa := value
	var exponent int64
	if exponentIndex := strings.IndexAny(value, "eE"); exponentIndex >= 0 {
		var parseErr error
		exponent, parseErr = strconv.ParseInt(value[exponentIndex+1:], 10, 32)
		if parseErr != nil {
			return "", 0, false, fmt.Errorf("%w: invalid exponent in %q", errInvalidDecimal, value)
		}
		mantissa = value[:exponentIndex]
	}

	if mantissa != "" && (mantissa[0] == '-' || mantissa[0] == '+') {
		negative = mantissa[0] == '-'
		mantissa = mantissa[1:]
	}
	if mantissa == "" {
		return "", 0, false, fmt.Errorf("%w: invalid input %q", errInvalidDecimal, value)
	}

	decimalPoint := strings.IndexByte(mantissa, '.')
	if decimalPoint != strings.LastIndexByte(mantissa, '.') {
		return "", 0, false, fmt.Errorf("%w: invalid input %q", errInvalidDecimal, value)
	}
	integerPart, fractionalPart := mantissa, ""
	if decimalPoint >= 0 {
		integerPart = mantissa[:decimalPoint]
		fractionalPart = mantissa[decimalPoint+1:]
	}
	if integerPart == "" && fractionalPart == "" {
		return "", 0, false, fmt.Errorf("%w: invalid input %q", errInvalidDecimal, value)
	}
	digits = integerPart + fractionalPart
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return "", 0, false, fmt.Errorf("%w: invalid input %q", errInvalidDecimal, value)
		}
	}

	digits = strings.TrimLeft(digits, "0")
	scale = int64(len(fractionalPart)) - exponent
	for scale > 0 && strings.HasSuffix(digits, "0") {
		digits = strings.TrimSuffix(digits, "0")
		scale--
	}
	if digits == "" {
		return "0", 0, negative, nil
	}
	return digits, scale, negative, nil
}

// formatDecimalParts rebuilds canonical plain-decimal text after parsing and
// precision enforcement, rejecting exponent expansion beyond the parser cap.
func formatDecimalParts(digits string, scale int64, negative bool) (string, error) {
	if digits == "0" {
		return digits, nil
	}
	if scale < 0 {
		if int64(len(digits))-scale > maxStringDigits {
			return "", fmt.Errorf("%w: input exceeds %d digits", errInvalidDecimal, maxStringDigits)
		}
		digits += strings.Repeat("0", int(-scale))
		scale = 0
	}
	if int64(len(digits)) <= scale {
		digits = strings.Repeat("0", int(scale)-len(digits)+1) + digits
	}
	if scale > 0 {
		decimalIndex := len(digits) - int(scale)
		digits = digits[:decimalIndex] + "." + digits[decimalIndex:]
	}
	if negative {
		digits = "-" + digits
	}
	return digits, nil
}

// normalise expands scientific notation because udecimal.Parse does not
// support it, then enforces udecimal's precision and parser limits.
func normalise(value string, truncate bool) (string, error) {
	digits, scale, negative, err := parseDecimalParts(value)
	if err != nil {
		return "", err
	}
	digits, scale, err = applyUdecimalPrecision(digits, scale, truncate, value)
	if err != nil {
		return "", err
	}
	return formatDecimalParts(digits, scale, negative)
}

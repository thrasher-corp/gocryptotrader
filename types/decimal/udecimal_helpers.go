//go:build udecimal_on

package decimal

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "quagmt/udecimal"

const (
	maxPrecision          = 19
	maxStringDigits       = 200
	maxEncodedBinaryBytes = 255

	// shopspringMaxDecodeExponent is shopspring's default MaxDecodeExponent.
	// udecimal's precision and expansion bounds already reject large exponents
	// except on zero and trailing-zero input; this limit rejects those too, so
	// the udecimal build accepts no exponent that the default build rejects.
	shopspringMaxDecodeExponent = 10000
)

var (
	errInvalidDecimal      = errors.New("invalid decimal")
	errPrecisionOutOfRange = errors.New("decimal precision out of range")
	errExponentOutOfRange  = errors.New("decimal exponent out of range")
	errUnsupportedScanType = errors.New("unsupported decimal scan type")
)

// useNativeParser reports whether text may go to udecimal's own parser. Beyond
// its u128 fast path, that parser removes a leading '-' and lets big.Int read
// the rest, which accepts a second sign, so text starting "-+" or "--" goes to
// NewFromString instead.
func useNativeParser[T string | []byte](text T) bool {
	return len(text) <= maxStringDigits && (len(text) < 2 || text[0] != '-' || (text[1] != '+' && text[1] != '-'))
}

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

// newUdecimalFromFloat accepts every finite float because udecimal's native
// constructor uses fixed notation and rejects large finite values over its
// parser length limit. It converts the shortest decimal form that round-trips
// to a float of bitSize bits.
func newUdecimalFromFloat(value float64, bitSize int) (udecimal.Decimal, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return udecimal.Decimal{}, fmt.Errorf("%w: non-finite float %v", errInvalidDecimal, value)
	}
	formatted := strconv.FormatFloat(value, 'g', -1, bitSize)
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
		// The default build returns zero for every power of zero, including 0^0
		// and negative powers, where PowToIntPart returns one or an error.
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
	if zeroes > integerDigitCount(value) {
		// The rounding position is left of every significant digit, so the result
		// is zero. Besides avoiding unnecessary work, this makes extreme int32
		// places safe without attempting to construct a ginormous power of ten.
		return udecimal.Zero
	}
	scale := powerOfTenUdecimal(int(zeroes))
	return value.MustDiv(scale).RoundHAZ(0).Mul(scale)
}

// truncateUdecimal handles negative precision because udecimal.Trunc accepts
// only an unsigned fractional precision.
func truncateUdecimal(value udecimal.Decimal, precision int32) udecimal.Decimal {
	if precision >= 0 {
		if precision >= maxPrecision {
			return value
		}
		return value.Trunc(uint8(precision))
	}

	zeroes := -int64(precision)
	if zeroes > integerDigitCount(value) {
		// Truncating left of every significant digit leaves zero. As in
		// roundUdecimal, this also keeps extreme int32 precision from
		// constructing a ginormous power of ten.
		return udecimal.Zero
	}
	scale := powerOfTenUdecimal(int(zeroes))
	return value.MustDiv(scale).Trunc(0).Mul(scale)
}

// integerDigitCount returns the number of significant digits before the
// decimal point.
func integerDigitCount(value udecimal.Decimal) int64 {
	integerText := strings.TrimPrefix(value.String(), "-")
	if decimalPoint := strings.IndexByte(integerText, '.'); decimalPoint >= 0 {
		integerText = integerText[:decimalPoint]
	}
	return int64(len(strings.TrimLeft(integerText, "0")))
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
	if scale > shopspringMaxDecodeExponent || scale < -shopspringMaxDecodeExponent {
		// Like the default build, reject these even for zero and trailing-zero
		// input that udecimal could represent.
		return "", 0, false, fmt.Errorf("%w: %q", errExponentOutOfRange, value)
	}
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

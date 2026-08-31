//go:build udecimal_on

// Package decimal provides fixed-point decimal arithmetic with a backend
// selected at build time. The shopspring backend is the default; build with
// the udecimal_on tag to use github.com/quagmt/udecimal.
package decimal

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Selected implementation for the udecimal_on build.
)

const maxStringDigits = 200

var (
	// ErrInvalidDecimal is returned when input cannot be represented as a decimal.
	ErrInvalidDecimal = errors.New("invalid decimal")
	// ErrPrecisionOutOfRange is returned when input exceeds the selected backend's precision.
	ErrPrecisionOutOfRange = errors.New("decimal precision out of range")
	// ErrDivideByZero exposes udecimal's native division error in this build.
	ErrDivideByZero = udecimal.ErrDivideByZero

	// Zero is the zero-value Decimal.
	Zero Decimal
)

// Decimal is an immutable fixed-point decimal value.
type Decimal struct {
	value udecimal.Decimal
}

// NewFromInt returns a Decimal equal to value.
func NewFromInt(value int64) Decimal {
	return RequireFromString(strconv.FormatInt(value, 10))
}

// NewFromInt32 returns a Decimal equal to value.
func NewFromInt32(value int32) Decimal {
	return NewFromInt(int64(value))
}

// NewFromFloat returns the shortest decimal representation that round-trips
// to value. The adapter preserves support for every finite float64 because
// udecimal's native constructor rejects large finite values over its parser
// length limit.
func NewFromFloat(value float64) Decimal {
	return Decimal{value: newFromFloatUdecimal(value)}
}

// NewFromString parses value. Scientific notation is normalised before using
// udecimal because its native parser does not support exponent notation.
func NewFromString(value string) (Decimal, error) {
	normalised, err := normalise(value, false)
	if err != nil {
		return Decimal{}, err
	}
	return Decimal{value: mustParseUdecimal(normalised)}, nil
}

// RequireFromString parses value and panics if it is invalid.
func RequireFromString(value string) Decimal {
	result, err := NewFromString(value)
	if err != nil {
		panic(err)
	}
	return result
}

// Add returns d+other.
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{value: d.value.Add(other.value)}
}

// Sub returns d-other.
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{value: d.value.Sub(other.value)}
}

// Mul returns d*other, truncating beyond the selected backend's precision.
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{value: d.value.Mul(other.value)}
}

// Div returns d/other using udecimal's native division, which truncates results
// beyond its configured precision. MustDiv adapts udecimal's error-returning
// API to the shopspring-shaped method signature exposed by this build tag.
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{value: d.value.MustDiv(other.value)}
}

// Mod returns the remainder of d/other. MustMod adapts udecimal's
// error-returning API to the shopspring-shaped method signature exposed by
// this build tag.
func (d Decimal) Mod(other Decimal) Decimal {
	return Decimal{value: d.value.MustMod(other.value)}
}

// Pow returns d raised to other. The common contract requires an integer exponent.
func (d Decimal) Pow(other Decimal) Decimal {
	result, err := powUdecimal(d.value, other.value)
	if err != nil {
		panic(fmt.Errorf("%w: %w", ErrInvalidDecimal, err))
	}
	return Decimal{value: result}
}

// Abs returns the absolute value of d.
func (d Decimal) Abs() Decimal {
	return Decimal{value: d.value.Abs()}
}

// Neg returns d with its sign inverted.
func (d Decimal) Neg() Decimal {
	return Decimal{value: d.value.Neg()}
}

// Cmp compares d and other and returns -1, 0 or 1.
func (d Decimal) Cmp(other Decimal) int {
	return d.value.Cmp(other.value)
}

// Compare compares d and other and returns -1, 0 or 1.
func (d Decimal) Compare(other Decimal) int {
	return d.Cmp(other)
}

// Equal reports whether d and other have the same numeric value.
func (d Decimal) Equal(other Decimal) bool {
	return d.value.Equal(other.value)
}

// GreaterThan reports whether d is greater than other.
func (d Decimal) GreaterThan(other Decimal) bool {
	return d.value.GreaterThan(other.value)
}

// GreaterThanOrEqual reports whether d is greater than or equal to other.
func (d Decimal) GreaterThanOrEqual(other Decimal) bool {
	return d.value.GreaterThanOrEqual(other.value)
}

// LessThan reports whether d is less than other.
func (d Decimal) LessThan(other Decimal) bool {
	return d.value.LessThan(other.value)
}

// LessThanOrEqual reports whether d is less than or equal to other.
func (d Decimal) LessThanOrEqual(other Decimal) bool {
	return d.value.LessThanOrEqual(other.value)
}

// IsZero reports whether d is zero.
func (d Decimal) IsZero() bool {
	return d.value.IsZero()
}

// IsPositive reports whether d is greater than zero.
func (d Decimal) IsPositive() bool {
	return d.value.IsPos()
}

// IsNegative reports whether d is less than zero.
func (d Decimal) IsNegative() bool {
	return d.value.IsNeg()
}

// Round rounds d half away from zero to places fractional digits. Negative
// places round digits in the integer component.
func (d Decimal) Round(places int32) Decimal {
	return Decimal{value: roundUdecimal(d.value, places)}
}

// Truncate removes fractional digits beyond precision without rounding.
func (d Decimal) Truncate(precision int32) Decimal {
	// udecimal.Trunc accepts only an unsigned fractional precision, while this
	// shopspring-shaped API permits negative and out-of-range values as no-ops.
	if precision < 0 || precision >= maxPrecision {
		return d
	}
	return Decimal{value: d.value.Trunc(uint8(precision))}
}

// Floor returns the greatest integer less than or equal to d.
func (d Decimal) Floor() Decimal {
	return Decimal{value: d.value.Floor()}
}

// Ceil returns the least integer greater than or equal to d.
func (d Decimal) Ceil() Decimal {
	return Decimal{value: d.value.Ceil()}
}

// String returns the canonical decimal representation of d.
func (d Decimal) String() string {
	return d.value.String()
}

// StringFixed returns d rounded to places and padded with trailing zeroes.
func (d Decimal) StringFixed(places int32) string {
	// udecimal.StringFixed neither accepts negative places nor rounds when the
	// requested precision is smaller, so adapt it to this method's public API.
	rounded := d.Round(places).String()
	if places <= 0 {
		return rounded
	}
	decimalPoint := strings.IndexByte(rounded, '.')
	if decimalPoint < 0 {
		return rounded + "." + strings.Repeat("0", int(places))
	}
	fractionalDigits := len(rounded) - decimalPoint - 1
	if fractionalDigits >= int(places) {
		return rounded
	}
	return rounded + strings.Repeat("0", int(places)-fractionalDigits)
}

// Float64 returns the nearest float64 and whether the conversion was exact.
func (d Decimal) Float64() (float64, bool) {
	rational, _ := new(big.Rat).SetString(d.String())
	return rational.Float64()
}

// InexactFloat64 returns the nearest float64 representation of d.
func (d Decimal) InexactFloat64() float64 {
	return d.value.InexactFloat64()
}

// IntPart returns the integer component of d, truncated toward zero.
func (d Decimal) IntPart() int64 {
	// udecimal.Int64 returns an error for large values, while this established
	// method cannot return one and follows big.Int's wrapping conversion.
	value := d.String()
	if decimalPoint := strings.IndexByte(value, '.'); decimalPoint >= 0 {
		value = value[:decimalPoint]
	}
	integer, _ := new(big.Int).SetString(value, 10)
	return integer.Int64()
}

// MarshalJSON implements json.Marshaler using a quoted decimal string.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return d.value.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler for quoted or bare decimals.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	return d.value.UnmarshalJSON(data)
}

// MarshalText implements encoding.TextMarshaler.
func (d Decimal) MarshalText() ([]byte, error) {
	return d.value.MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Decimal) UnmarshalText(data []byte) error {
	return d.value.UnmarshalText(data)
}

// MarshalBinary implements encoding.BinaryMarshaler using udecimal's native format.
func (d Decimal) MarshalBinary() ([]byte, error) {
	return d.value.MarshalBinary()
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler using udecimal's native format.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	return d.value.UnmarshalBinary(data)
}

// Scan implements sql.Scanner using udecimal's native supported input types.
func (d *Decimal) Scan(value any) error {
	return d.value.Scan(value)
}

// Value implements driver.Valuer using the canonical decimal string.
func (d Decimal) Value() (driver.Value, error) {
	return d.value.Value()
}

func normalise(value string, truncate bool) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%w: empty input", ErrInvalidDecimal)
	}

	mantissa := value
	var exponent int64
	if exponentIndex := strings.IndexAny(value, "eE"); exponentIndex >= 0 {
		var err error
		exponent, err = strconv.ParseInt(value[exponentIndex+1:], 10, 32)
		if err != nil {
			return "", fmt.Errorf("%w: invalid exponent in %q", ErrInvalidDecimal, value)
		}
		mantissa = value[:exponentIndex]
	}

	negative := false
	if mantissa != "" && (mantissa[0] == '-' || mantissa[0] == '+') {
		negative = mantissa[0] == '-'
		mantissa = mantissa[1:]
	}
	if mantissa == "" {
		return "", fmt.Errorf("%w: invalid input %q", ErrInvalidDecimal, value)
	}

	decimalPoint := strings.IndexByte(mantissa, '.')
	if decimalPoint != strings.LastIndexByte(mantissa, '.') {
		return "", fmt.Errorf("%w: invalid input %q", ErrInvalidDecimal, value)
	}
	integerPart, fractionalPart := mantissa, ""
	if decimalPoint >= 0 {
		integerPart = mantissa[:decimalPoint]
		fractionalPart = mantissa[decimalPoint+1:]
	}
	if integerPart == "" && fractionalPart == "" {
		return "", fmt.Errorf("%w: invalid input %q", ErrInvalidDecimal, value)
	}
	digits := integerPart + fractionalPart
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return "", fmt.Errorf("%w: invalid input %q", ErrInvalidDecimal, value)
		}
	}

	digits = strings.TrimLeft(digits, "0")
	scale := int64(len(fractionalPart)) - exponent
	for scale > 0 && strings.HasSuffix(digits, "0") {
		digits = strings.TrimSuffix(digits, "0")
		scale--
	}
	digits, scale, err := normaliseUdecimalPrecision(digits, scale, truncate, value)
	if err != nil {
		return "", err
	}
	if digits == "0" {
		return digits, nil
	}
	if scale < 0 {
		if int64(len(digits))-scale > maxStringDigits {
			return "", fmt.Errorf("%w: input exceeds %d digits", ErrInvalidDecimal, maxStringDigits)
		}
		digits += strings.Repeat("0", int(-scale))
		scale = 0
	}
	if int64(len(digits)) <= scale {
		digits = strings.Repeat("0", int(scale)-len(digits)+1) + digits
	}
	if digitCount := len(digits); digitCount > maxStringDigits {
		return "", fmt.Errorf("%w: input exceeds %d digits", ErrInvalidDecimal, maxStringDigits)
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

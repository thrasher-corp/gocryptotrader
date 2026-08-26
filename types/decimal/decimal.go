// Package decimal provides fixed-point decimal arithmetic with a backend
// selected at build time. The shopspring backend is the default; build with
// the udecimal_on tag to use github.com/quagmt/udecimal.
package decimal

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

const (
	maxPrecision      = 19
	divisionPrecision = 16
	maxStringLength   = 200
)

var (
	// ErrInvalidDecimal is returned when input cannot be represented as a decimal.
	ErrInvalidDecimal = errors.New("invalid decimal")
	// ErrPrecisionOutOfRange is returned when input has more than 19 fractional digits.
	ErrPrecisionOutOfRange = errors.New("decimal precision out of range")
	// ErrDivideByZero is returned when a division operation has a zero divisor.
	ErrDivideByZero = errors.New("decimal division by zero")

	// Zero is the zero-value Decimal.
	Zero Decimal
)

// Decimal is an immutable fixed-point decimal value.
type Decimal struct {
	value backendDecimal
}

// NewFromInt returns a Decimal equal to value.
func NewFromInt(value int64) Decimal {
	result, err := NewFromString(strconv.FormatInt(value, 10))
	if err != nil {
		panic(err)
	}
	return result
}

// NewFromInt32 returns a Decimal equal to value.
func NewFromInt32(value int32) Decimal {
	return NewFromInt(int64(value))
}

// NewFromFloat returns the shortest decimal representation that round-trips
// to value, truncating binary-float noise beyond 19 fractional digits. It
// panics for non-finite values, matching the established constructor's must
// semantics.
func NewFromFloat(value float64) Decimal {
	normalised, err := normalise(strconv.FormatFloat(value, 'f', -1, 64), true)
	if err != nil {
		panic(err)
	}
	parsed, err := parseBackend(normalised)
	if err != nil {
		panic(fmt.Errorf("%w: %w", ErrInvalidDecimal, err))
	}
	return Decimal{value: parsed}
}

// NewFromString parses value using the common backend-independent decimal
// syntax and precision contract.
func NewFromString(value string) (Decimal, error) {
	normalised, err := normalise(value, false)
	if err != nil {
		return Decimal{}, err
	}
	result, err := parseBackend(normalised)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: %w", ErrInvalidDecimal, err)
	}
	return Decimal{value: result}, nil
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

// Mul returns d*other, truncating fractional precision beyond 19 places.
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{value: mulBackend(d.value, other.value)}
}

// Div returns d/other rounded half away from zero to 16 fractional places.
// It panics when other is zero, matching the established shopspring contract.
func (d Decimal) Div(other Decimal) Decimal {
	result, err := divBackend(d.value, other.value)
	if err != nil {
		panic(fmt.Errorf("%w: %w", ErrDivideByZero, err))
	}
	return Decimal{value: result}
}

// Mod returns the remainder of d/other. It panics when other is zero.
func (d Decimal) Mod(other Decimal) Decimal {
	result, err := modBackend(d.value, other.value)
	if err != nil {
		panic(fmt.Errorf("%w: %w", ErrDivideByZero, err))
	}
	return Decimal{value: result}
}

// Pow returns d raised to other. The common contract requires an integer exponent.
func (d Decimal) Pow(other Decimal) Decimal {
	result, err := powBackend(d.value, other.value)
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
	return isPositiveBackend(d.value)
}

// IsNegative reports whether d is less than zero.
func (d Decimal) IsNegative() bool {
	return isNegativeBackend(d.value)
}

// Round rounds d half away from zero to places fractional digits. Negative
// places round digits in the integer component.
func (d Decimal) Round(places int32) Decimal {
	return Decimal{value: roundBackend(d.value, places)}
}

// Truncate removes fractional digits beyond precision without rounding.
func (d Decimal) Truncate(precision int32) Decimal {
	return Decimal{value: truncateBackend(d.value, precision)}
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
	rational, okay := new(big.Rat).SetString(d.String())
	if !okay {
		return math.NaN(), false
	}
	return rational.Float64()
}

// InexactFloat64 returns the nearest float64 representation of d.
func (d Decimal) InexactFloat64() float64 {
	value, _ := d.Float64()
	return value
}

// IntPart returns the integer component of d, truncated toward zero.
func (d Decimal) IntPart() int64 {
	value := d.String()
	if decimalPoint := strings.IndexByte(value, '.'); decimalPoint >= 0 {
		value = value[:decimalPoint]
	}
	integer, okay := new(big.Int).SetString(value, 10)
	if !okay {
		return 0
	}
	return integer.Int64()
}

// MarshalJSON implements json.Marshaler using a quoted decimal string.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(d.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler for quoted or bare decimals.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	value := string(data)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidDecimal, err)
		}
		value = unquoted
	}
	result, err := NewFromString(value)
	if err != nil {
		return err
	}
	*d = result
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Decimal) UnmarshalText(data []byte) error {
	result, err := NewFromString(string(data))
	if err != nil {
		return err
	}
	*d = result
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler using the canonical text form.
func (d Decimal) MarshalBinary() ([]byte, error) {
	return d.MarshalText()
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	return d.UnmarshalText(data)
}

// Scan implements sql.Scanner.
func (d *Decimal) Scan(value any) error {
	var input string
	switch converted := value.(type) {
	case nil:
		*d = Zero
		return nil
	case string:
		input = converted
	case []byte:
		input = string(converted)
	case int64:
		input = strconv.FormatInt(converted, 10)
	case float64:
		input = strconv.FormatFloat(converted, 'f', -1, 64)
	default:
		return fmt.Errorf("%w: unsupported scan type %T", ErrInvalidDecimal, value)
	}
	return d.UnmarshalText([]byte(input))
}

// Value implements driver.Valuer using the canonical decimal string.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
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
	if digits == "" {
		return "0", nil
	}
	scale := int64(len(fractionalPart)) - exponent
	for scale > 0 && strings.HasSuffix(digits, "0") {
		digits = strings.TrimSuffix(digits, "0")
		scale--
	}
	if scale > maxPrecision {
		if !truncate {
			return "", fmt.Errorf("%w: %q requires %d fractional digits", ErrPrecisionOutOfRange, value, scale)
		}
		excess := int(scale - maxPrecision)
		if excess >= len(digits) {
			return "0", nil
		}
		digits = digits[:len(digits)-excess]
		scale = maxPrecision
		for scale > 0 && strings.HasSuffix(digits, "0") {
			digits = strings.TrimSuffix(digits, "0")
			scale--
		}
	}
	if scale < 0 {
		if int64(len(digits))-scale > maxStringLength {
			return "", fmt.Errorf("%w: input exceeds %d characters", ErrInvalidDecimal, maxStringLength)
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
	if len(digits) > maxStringLength {
		return "", fmt.Errorf("%w: input exceeds %d characters", ErrInvalidDecimal, maxStringLength)
	}
	return digits, nil
}

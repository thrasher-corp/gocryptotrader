//go:build udecimal_on

package decimal

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"strings"

	"github.com/quagmt/udecimal" //nolint:depguard // Selected implementation for the udecimal_on build.
)

// Zero is the zero-value Decimal.
var Zero Decimal

// Decimal is an immutable fixed-point decimal value.
type Decimal struct {
	value udecimal.Decimal
}

// NewFromInt returns a Decimal equal to value. Udecimal's Must constructor is
// safe here because zero precision is valid for every int64.
func NewFromInt(value int64) Decimal {
	return Decimal{value: udecimal.MustFromInt64(value, 0)}
}

// NewFromInt32 returns a Decimal equal to value.
func NewFromInt32(value int32) Decimal {
	return NewFromInt(int64(value))
}

// MustFromFloat returns the shortest decimal representation that round-trips
// to value. It panics for non-finite values.
func MustFromFloat(value float64) Decimal {
	result, err := newUdecimalFromFloat(value)
	if err != nil {
		panic(err)
	}
	return Decimal{value: result}
}

// NewFromString parses value. Scientific notation is normalised before using
// udecimal because its native parser does not support exponent notation.
func NewFromString(value string) (Decimal, error) {
	normalised, err := normalise(value, false)
	if err != nil {
		return Decimal{}, err
	}
	result, err := parseUdecimal(normalised)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: %w", errInvalidDecimal, err)
	}
	return Decimal{value: result}, nil
}

// MustFromString parses value and panics if it is invalid.
func MustFromString(value string) Decimal {
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

// Mul returns d*other, truncating beyond udecimal's 19-place precision.
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{value: d.value.Mul(other.value)}
}

// Div returns d/other, truncating beyond udecimal's 19-place precision. It
// panics for a zero divisor because the Decimal API has no error return.
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{value: divideUdecimal(d.value, other.value)}
}

// Mod returns the remainder of d/other. It panics for a zero divisor because
// the Decimal API has no error return.
func (d Decimal) Mod(other Decimal) Decimal {
	return Decimal{value: d.value.MustMod(other.value)}
}

// Pow returns d raised to other and panics if other has a fractional component.
func (d Decimal) Pow(other Decimal) Decimal {
	result, err := powUdecimal(d.value, other.value)
	if err != nil {
		panic(err)
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
	// udecimal.Trunc accepts only unsigned precision, while Decimal.Truncate
	// treats negative and out-of-range precision as no-ops.
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
	// udecimal.StringFixed does not accept negative places or round to smaller
	// precision, so use the Decimal rounding contract before padding.
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
	if len(data) <= maxStringDigits {
		return d.value.UnmarshalJSON(data)
	}
	if data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	result, err := NewFromString(string(data))
	if err != nil {
		return fmt.Errorf("error unmarshalling decimal JSON: %w", err)
	}
	*d = result
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Decimal) MarshalText() ([]byte, error) {
	return d.value.MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Decimal) UnmarshalText(data []byte) error {
	if len(data) <= maxStringDigits {
		return d.value.UnmarshalText(data)
	}
	result, err := NewFromString(string(data))
	if err != nil {
		return fmt.Errorf("error unmarshalling decimal text: %w", err)
	}
	*d = result
	return nil
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
	var text string
	switch value := value.(type) {
	case string:
		if len(value) <= maxStringDigits {
			return d.value.Scan(value)
		}
		text = value
	case []byte:
		if len(value) <= maxStringDigits {
			return d.value.Scan(value)
		}
		text = string(value)
	default:
		return d.value.Scan(value)
	}
	result, err := NewFromString(text)
	if err != nil {
		return fmt.Errorf("error scanning decimal: %w", err)
	}
	*d = result
	return nil
}

// Value implements driver.Valuer using the canonical decimal string.
func (d Decimal) Value() (driver.Value, error) {
	return d.value.Value()
}

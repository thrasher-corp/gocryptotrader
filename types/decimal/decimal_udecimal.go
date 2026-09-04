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

// MustFromFloat returns value truncated to udecimal's 19 fractional digits.
// Finite magnitudes below 1e-19 become zero. It panics for non-finite values.
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
	return Decimal{value: moduloUdecimal(d.value, other.value)}
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

// IsInteger reports whether d has no fractional component.
func (d Decimal) IsInteger() bool {
	return d.Equal(d.Truncate(0))
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
	// String always returns canonical plain decimal text, so SetString cannot fail.
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
	// The integer component of canonical decimal text is always valid base 10.
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
		if err := d.value.UnmarshalJSON(data); err == nil {
			return nil
		}
	}
	if len(data) > 1 && data[0] == '"' && data[len(data)-1] == '"' {
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
		if err := d.value.UnmarshalText(data); err == nil {
			return nil
		}
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
	data, err := d.value.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// The native format stores its total length in one byte. Refuse values that
	// would otherwise produce an encoding its own decoder cannot read.
	if len(data) > maxEncodedBinaryBytes {
		return nil, fmt.Errorf("%w: encoded decimal exceeds %d bytes", udecimal.ErrInvalidBinaryData, maxEncodedBinaryBytes)
	}
	return data, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler using udecimal's native format.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	// udecimal trusts the precision byte and can construct a value which later
	// panics. Decimal's supported precision is fixed, so reject it at the codec
	// boundary before allowing the backend to mutate the receiver.
	if len(data) > 1 && data[1] > maxPrecision {
		return fmt.Errorf("%w: precision %d exceeds %d", udecimal.ErrInvalidBinaryData, data[1], maxPrecision)
	}
	var value udecimal.Decimal
	if err := value.UnmarshalBinary(data); err != nil {
		return err
	}
	if value.IsZero() {
		// The native decoder assigns sign and precision directly, bypassing the
		// constructor which canonicalises zero for consistent comparisons.
		value = udecimal.Zero
	}
	d.value = value
	return nil
}

// Scan implements sql.Scanner while preserving the default backend's accepted
// input types and the Decimal façade's parsing and float-conversion semantics.
func (d *Decimal) Scan(value any) error {
	if value32, ok := value.(float32); ok {
		value = float64(value32)
	}

	var text string
	switch value := value.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	case float64:
		result, err := newUdecimalFromFloat(value)
		if err != nil {
			return fmt.Errorf("error scanning decimal: %w", err)
		}
		d.value = result
		return nil
	case int, int32:
		return fmt.Errorf("can't scan %T to Decimal: %T is not supported", value, value)
	default:
		return d.value.Scan(value)
	}
	// shopspring accepts SQL drivers which return quoted numeric text. Preserve
	// that established behaviour before trying either udecimal parser.
	if len(text) > 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	if len(text) <= maxStringDigits {
		if err := d.value.Scan(text); err == nil {
			return nil
		}
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

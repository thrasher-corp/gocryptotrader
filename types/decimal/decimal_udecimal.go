//go:build udecimal_on

package decimal

import (
	"database/sql/driver"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/quagmt/udecimal"               //nolint:depguard // Selected implementation for the udecimal_on build.
	shopspring "github.com/shopspring/decimal" //nolint:depguard // Scan takes the default build's own type, as the default build does.
)

// MaxFractionalDigits is the fractional precision retained by udecimal arithmetic.
const MaxFractionalDigits = maxPrecision

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
	result, err := newUdecimalFromFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return Decimal{value: result}
}

// NewFromString parses value. Scientific notation is normalised before using
// udecimal because its native parser does not support exponent notation. As in
// the default build, the Unicode minus sign (U+2212) is accepted for '-'.
func NewFromString(value string) (Decimal, error) {
	// Matches the default build, where shopspring v1.6.0 reads the Unicode minus sign as '-'.
	if strings.IndexByte(value, "\u2212"[0]) >= 0 {
		value = strings.ReplaceAll(value, "\u2212", "-")
	}
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
	return Decimal{value: d.value.MustDiv(other.value)}
}

// Mod returns the remainder of d/other. It panics for a zero divisor because
// the Decimal API has no error return.
func (d Decimal) Mod(other Decimal) Decimal {
	return Decimal{value: d.value.MustMod(other.value)}
}

// Pow returns d raised to other. It panics if other has a fractional component
// or, for a non-zero d, a magnitude beyond math.MaxInt32.
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
	// Zero precision guarantees an integer and avoids truncation; nonzero
	// precision may still represent one when the fractional digits are zero.
	return d.value.Prec() == 0 || d.Equal(d.Truncate(0))
}

// Round rounds d half away from zero to places fractional digits. Negative
// places round digits in the integer component.
func (d Decimal) Round(places int32) Decimal {
	return Decimal{value: roundUdecimal(d.value, places)}
}

// Truncate removes digits beyond precision without rounding. Negative
// precision truncates integer digits toward zero.
func (d Decimal) Truncate(precision int32) Decimal {
	return Decimal{value: truncateUdecimal(d.value, precision)}
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

// UnmarshalJSON implements json.Unmarshaler for quoted or bare decimals. As in
// the default build, a bare null leaves d unchanged, the quoted string "null"
// is invalid, and d is zero after any other decoding failure.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	text := data
	if len(text) > 1 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	// udecimal's UnmarshalJSON would read the quoted string "null" as null too,
	// so parse the unquoted text with its text decoder instead. The parser limit
	// counts the JSON as received, quotes included.
	if len(data) <= maxStringDigits && useNativeParser(text) {
		if err := d.value.UnmarshalText(text); err == nil {
			return nil
		}
	}
	result, err := NewFromString(string(text))
	*d = result
	if err != nil {
		return fmt.Errorf("error unmarshalling decimal JSON: %w", err)
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Decimal) MarshalText() ([]byte, error) {
	return d.value.MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler. As in the default build, d
// is zero after a decoding failure.
func (d *Decimal) UnmarshalText(data []byte) error {
	if useNativeParser(data) {
		if err := d.value.UnmarshalText(data); err == nil {
			return nil
		}
	}
	result, err := NewFromString(string(data))
	*d = result
	if err != nil {
		return fmt.Errorf("error unmarshalling decimal text: %w", err)
	}
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
	// Decoding into a local value leaves d untouched, and a nil d unused, on error.
	var value udecimal.Decimal
	if err := value.UnmarshalBinary(data); err != nil {
		return err
	}
	d.value = value
	return nil
}

// Scan implements sql.Scanner. It accepts the default backend's input types and
// applies the Decimal façade's parsing and float-conversion semantics. As in
// the default build, d is zero after text that fails to decode.
func (d *Decimal) Scan(value any) error {
	var text string
	switch value := value.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	case float32:
		// The shortest form that round-trips to float32 keeps 54.33 as 54.33
		// rather than its float64 expansion 54.33000183105469.
		return d.scanFloat(float64(value), 32)
	case float64:
		return d.scanFloat(value, 64)
	case int, int8, int16, int32, int64:
		*d = NewFromInt(reflect.ValueOf(value).Int())
		return nil
	case uint, uint8, uint16, uint32, uint64:
		d.value = udecimal.MustFromUint64(reflect.ValueOf(value).Uint(), 0)
		return nil
	case big.Int:
		return d.scanBigInt(&value)
	case *big.Int:
		if value == nil {
			return fmt.Errorf("error scanning decimal: %w: nil *big.Int", errInvalidDecimal)
		}
		return d.scanBigInt(value)
	case Decimal:
		*d = value
		return nil
	case shopspring.Decimal:
		// String would expand the exponent before NewFromString bounds it, and
		// its form follows shopspring's display settings.
		result, err := NewFromString(value.Coefficient().String() + "e" + strconv.FormatInt(int64(value.Exponent()), 10))
		if err != nil {
			return fmt.Errorf("error scanning decimal: %w", err)
		}
		*d = result
		return nil
	default:
		return fmt.Errorf("error scanning decimal: %w: %T", errUnsupportedScanType, value)
	}
	// shopspring accepts SQL drivers which return quoted numeric text. Preserve
	// that established behaviour before trying either udecimal parser.
	if len(text) > 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	if useNativeParser(text) {
		if err := d.value.Scan(text); err == nil {
			return nil
		}
	}
	result, err := NewFromString(text)
	*d = result
	if err != nil {
		return fmt.Errorf("error scanning decimal: %w", err)
	}
	return nil
}

// scanFloat stores value converted through the shortest decimal form that
// round-trips to a float of bitSize bits.
func (d *Decimal) scanFloat(value float64, bitSize int) error {
	result, err := newUdecimalFromFloat(value, bitSize)
	if err != nil {
		return fmt.Errorf("error scanning decimal: %w", err)
	}
	d.value = result
	return nil
}

// scanBigInt stores value through NewFromString, whose extended parser accepts
// integers beyond udecimal's native 200-character limit.
func (d *Decimal) scanBigInt(value *big.Int) error {
	result, err := NewFromString(value.String())
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

// Format implements fmt.Formatter for the verbs the default backend supports.
// %v, %s, %q, %x and %X format String(). %f and %F print fixed-point notation,
// rounded like StringFixed when a precision is given, with fmt's width and sign
// flags. %#v prints the MustFromString call that recreates d. Other verbs print
// fmt's usual error, such as %!d(decimal.Decimal=3.14).
func (d Decimal) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q', 'x', 'X':
		text := d.String()
		if verb == 'v' && state.Flag('#') {
			verb, text = 's', "decimal.MustFromString("+strconv.Quote(text)+")"
		}
		_, hasWidth := state.Width()
		_, hasPrecision := state.Precision()
		if (verb == 'v' || verb == 's') && !hasWidth && !hasPrecision {
			// Println and plain %v are the common case; skip rebuilding the directive.
			_, _ = io.WriteString(state, text)
			return
		}
		_, _ = fmt.Fprintf(state, fmt.FormatString(state, verb), text)
	case 'f', 'F':
		_, _ = io.WriteString(state, d.formatFixed(state))
	default:
		_, _ = fmt.Fprintf(state, "%%!%c(decimal.Decimal=%s)", verb, d.String())
	}
}

// formatFixed returns d in fixed-point notation with fmt's float sign, padding
// and alignment flags applied.
func (d Decimal) formatFixed(state fmt.State) string {
	var text string
	if precision, ok := state.Precision(); ok {
		text = d.StringFixed(int32(precision)) //nolint:gosec // fmt caps precision far below math.MaxInt32
	} else {
		text = d.String()
	}
	var sign string
	switch {
	case text[0] == '-':
		sign, text = "-", text[1:]
	case state.Flag('+'):
		sign = "+"
	case state.Flag(' '):
		sign = " "
	}
	width, _ := state.Width()
	padding := width - len(sign) - len(text)
	switch {
	case padding <= 0:
		return sign + text
	case state.Flag('-'):
		return sign + text + strings.Repeat(" ", padding)
	case state.Flag('0'):
		return sign + strings.Repeat("0", padding) + text
	default:
		return strings.Repeat(" ", padding) + sign + text
	}
}

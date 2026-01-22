package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

var errInvalidNumberValue = errors.New("invalid value for Number type")

// Number represents a floating point number, and implements json.Unmarshaller and json.Marshaller
type Number float64

// UnmarshalJSON implements json.Unmarshaler
func (f *Number) UnmarshalJSON(data []byte) error {
	switch c := data[0]; c { // From json.decode literalInterface
	case 'n': // null
		*f = Number(0)
		return nil
	case 't', 'f': // true, false
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
	case '"': // string
		if len(data) < 2 || data[len(data)-1] != '"' {
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
		data = data[1 : len(data)-1] // Naive Unquote
	default: // Should be a number
		if c != '-' && (c < '0' || c > '9') { // Invalid json syntax
			return fmt.Errorf("%w: %s", errInvalidNumberValue, data)
		}
	}

	if len(data) == 0 {
		*f = Number(0)
		return nil
	}

	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidNumberValue, data) // We don't use err; We know it's not valid and errInvalidNumberValue is clearer
	}

	*f = Number(val)
	return nil
}

// MarshalJSON implements json.Marshaler by formatting to a json string
// 1337.37 will marshal to "1337.37"
// 0 will marshal to an empty string: ""
func (f Number) MarshalJSON() ([]byte, error) {
	if f == 0 {
		return []byte(`""`), nil
	}
	val := strconv.FormatFloat(float64(f), 'f', -1, 64)
	return []byte(`"` + val + `"`), nil
}

// Float64 returns the underlying float64
func (f Number) Float64() float64 {
	return float64(f)
}

// Int64 returns the truncated integer component of the number
func (f Number) Int64() int64 {
	// It's likely this is sufficient, since Numbers probably have not had floating point math performed on them
	// However if issues arise then we can switch to math.Round
	return int64(f)
}

// Decimal returns a decimal.Decimal
func (f Number) Decimal() decimal.Decimal {
	return decimal.NewFromFloat(float64(f))
}

// String returns a string representation of the number
func (f Number) String() string {
	return strconv.FormatFloat(float64(f), 'f', -1, 64)
}

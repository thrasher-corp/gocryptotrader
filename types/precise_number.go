package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

var errInvalidPreciseNumberValue = errors.New("invalid value for PreciseNumber type")

// PreciseNumber is a numeric type for JSON values that need lossless decimal
// representation alongside a fast floating-point form.
//
// It stores the original numeric string as received from the wire and parses
// a float64 cache for cheap comparisons and sorting (e.g. order book level
// alignment). Decimal() and Int64() then return the value parsed from the
// preserved string rather than from the float, avoiding the precision loss
// that occurs in the existing [Number] type when [Number.Decimal] converts
// from float64.
//
// PreciseNumber is the right choice for fields that flow into accounting,
// reconciliation, settlement, or anything where the exact decimal value
// from the exchange must be reproduced. For market-data fields where the
// value is only used for sorting and rough computation, [Number] remains
// cheaper.
//
// Zero value behaves as the number 0.
type PreciseNumber struct {
	raw string  // original numeric string from the source; "" means zero
	val float64 // parsed cache for fast Float64()
}

// NewPreciseNumberFromString constructs a PreciseNumber from a decimal string
// without any precision loss. The string must be a valid base-10 number.
func NewPreciseNumberFromString(s string) (PreciseNumber, error) {
	if s == "" {
		return PreciseNumber{}, nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return PreciseNumber{}, fmt.Errorf("%w: %s", errInvalidPreciseNumberValue, s)
	}
	return PreciseNumber{raw: s, val: val}, nil
}

// UnmarshalJSON implements json.Unmarshaler. It accepts both JSON strings
// ("1337.37") and JSON numbers (1337.37) and preserves the original text
// verbatim for downstream Decimal()/Int64() calls.
func (p *PreciseNumber) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*p = PreciseNumber{}
		return nil
	}

	switch c := data[0]; c {
	case 'n': // null
		*p = PreciseNumber{}
		return nil
	case 't', 'f':
		return fmt.Errorf("%w: %s", errInvalidPreciseNumberValue, data)
	case '"':
		if len(data) < 2 || data[len(data)-1] != '"' {
			return fmt.Errorf("%w: %s", errInvalidPreciseNumberValue, data)
		}
		data = data[1 : len(data)-1]
	default:
		if c != '-' && (c < '0' || c > '9') {
			return fmt.Errorf("%w: %s", errInvalidPreciseNumberValue, data)
		}
	}

	if len(data) == 0 {
		*p = PreciseNumber{}
		return nil
	}

	s := string(data)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidPreciseNumberValue, data)
	}

	p.raw = s
	p.val = val
	return nil
}

// MarshalJSON implements json.Marshaler by emitting the original numeric
// string as a JSON string. The zero value marshals to `""` to match the
// existing [Number.MarshalJSON] behaviour.
func (p PreciseNumber) MarshalJSON() ([]byte, error) {
	if p.raw == "" {
		return []byte(`""`), nil
	}
	out := make([]byte, 0, len(p.raw)+2)
	out = append(out, '"')
	out = append(out, p.raw...)
	out = append(out, '"')
	return out, nil
}

// Float64 returns the cached float64 form. Suitable for sorting, comparisons,
// and approximate computation. For exact arithmetic use Decimal.
func (p PreciseNumber) Float64() float64 {
	return p.val
}

// Int64 returns the integer value parsed from the original string. If the
// original value contains a fractional part the fractional part is truncated
// toward zero (matching [Number.Int64]). For values that originated as
// integers (trade IDs, integer timestamps) this returns the exact value with
// no float round-trip.
func (p PreciseNumber) Int64() int64 {
	if p.raw == "" {
		return 0
	}
	if i, err := strconv.ParseInt(p.raw, 10, 64); err == nil {
		return i
	}
	// Fractional value; fall back to truncating via Decimal to avoid the
	// float-rounding behaviour exhibited by `int64(float64(p.val))` near
	// representation boundaries.
	if d, err := decimal.NewFromString(p.raw); err == nil {
		return d.Truncate(0).IntPart()
	}
	return int64(p.val)
}

// Decimal returns the exact decimal value parsed from the original string,
// avoiding the float64 round-trip that loses precision in [Number.Decimal].
func (p PreciseNumber) Decimal() decimal.Decimal {
	if p.raw == "" {
		return decimal.Zero
	}
	if d, err := decimal.NewFromString(p.raw); err == nil {
		return d
	}
	return decimal.NewFromFloat(p.val)
}

// String returns the original numeric string, or "0" when the value is the
// zero value.
func (p PreciseNumber) String() string {
	if p.raw == "" {
		return "0"
	}
	return p.raw
}

// IsZero reports whether the value is the uninitialized zero value.
//
// Note: this returns true only for the uninitialized struct, not for an
// explicit "0" parsed from the wire. The distinction matters when
// json.Marshal honours the IsZero() method for `omitempty` (Go 1.24+):
// an explicit "0" must round-trip through JSON for accounting fields,
// whereas an unset field is correctly omitted.
func (p PreciseNumber) IsZero() bool {
	return p.raw == ""
}

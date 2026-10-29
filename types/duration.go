package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var errInvalidDurationValue = errors.New("invalid value for duration type")

// DurationSeconds is a time.Duration that JSON carries as a number of seconds, quoted or bare, such as 30 or "1.5".
// It marshals as a bare number of seconds
type DurationSeconds time.Duration

// UnmarshalJSON implements json.Unmarshaler
func (d *DurationSeconds) UnmarshalJSON(data []byte) error {
	v, err := unmarshalDuration(data, time.Second)
	if err != nil {
		return err
	}
	*d = DurationSeconds(v)
	return nil
}

// MarshalJSON implements json.Marshaler
func (d DurationSeconds) MarshalJSON() ([]byte, error) {
	return formatDuration(time.Duration(d), time.Second), nil
}

// Duration returns the underlying time.Duration
func (d DurationSeconds) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the duration in time.Duration's format, such as 1m30s
func (d DurationSeconds) String() string {
	return time.Duration(d).String()
}

// DurationMilliseconds is a time.Duration that JSON carries as a number of milliseconds, quoted or bare, such as
// 500 or "1500". It marshals as a bare number of milliseconds
type DurationMilliseconds time.Duration

// UnmarshalJSON implements json.Unmarshaler
func (d *DurationMilliseconds) UnmarshalJSON(data []byte) error {
	v, err := unmarshalDuration(data, time.Millisecond)
	if err != nil {
		return err
	}
	*d = DurationMilliseconds(v)
	return nil
}

// MarshalJSON implements json.Marshaler
func (d DurationMilliseconds) MarshalJSON() ([]byte, error) {
	return formatDuration(time.Duration(d), time.Millisecond), nil
}

// Duration returns the underlying time.Duration
func (d DurationMilliseconds) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the duration in time.Duration's format, such as 1.5s
func (d DurationMilliseconds) String() string {
	return time.Duration(d).String()
}

// unmarshalDuration reads a JSON number of units, quoted or bare, as a time.Duration, exactly and dropping any fraction
// of a nanosecond as time.ParseDuration does. null and an empty string read as zero, as they do for Number
func unmarshalDuration(data []byte, unit time.Duration) (time.Duration, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("%w: empty input", errInvalidDurationValue)
	}
	var s string
	switch data[0] {
	case 'n':
		if string(data) != "null" {
			return 0, fmt.Errorf("%w: %s", errInvalidDurationValue, data)
		}
		return 0, nil
	case '"':
		if err := json.Unmarshal(data, &s); err != nil {
			return 0, fmt.Errorf("%w: %s", errInvalidDurationValue, data)
		}
	default:
		s = string(data)
	}
	if s == "" {
		return 0, nil
	}
	return parseDuration(s, unit)
}

// parseDuration reads s, a decimal number with an optional sign, decimal point and exponent, as a number of units.
// It shifts the digits rather than computing a power of ten, so the result is exact and an exponent of any size is
// cheap. The unit must be a power of ten nanoseconds
func parseDuration(s string, unit time.Duration) (time.Duration, error) {
	rest := s
	negative := rest[0] == '-'
	if negative || rest[0] == '+' {
		rest = rest[1:]
	}
	var digits []byte // Significant digits, without leading zeros
	exponent := int64(len(strconv.FormatInt(int64(unit), 10)) - 1)
	seenDigit, seenPoint := false, false
	i := 0
	for ; i < len(rest); i++ {
		switch c := rest[i]; {
		case c >= '0' && c <= '9':
			seenDigit = true
			if seenPoint {
				exponent--
			}
			if c != '0' || len(digits) != 0 {
				digits = append(digits, c)
			}
			continue
		case c == '.' && !seenPoint:
			seenPoint = true
			continue
		}
		break
	}
	if !seenDigit {
		return 0, fmt.Errorf("%w: %q", errInvalidDurationValue, s)
	}
	if i < len(rest) {
		if rest[i] != 'e' && rest[i] != 'E' {
			return 0, fmt.Errorf("%w: %q", errInvalidDurationValue, s)
		}
		i++
		negativeExponent := i < len(rest) && rest[i] == '-'
		if i < len(rest) && (rest[i] == '-' || rest[i] == '+') {
			i++
		}
		start := i
		var e int64
		for ; i < len(rest) && rest[i] >= '0' && rest[i] <= '9'; i++ {
			// Past this size the number is zero or out of range whatever its digits, so the exponent stops growing
			if e < 1<<32 {
				e = e*10 + int64(rest[i]-'0')
			}
		}
		if i == start || i != len(rest) {
			return 0, fmt.Errorf("%w: %q", errInvalidDurationValue, s)
		}
		if negativeExponent {
			e = -e
		}
		exponent += e
	}
	// The digits that end up before the decimal point once in nanoseconds; the rest are the fraction dropped
	whole := int64(len(digits)) + exponent
	if len(digits) == 0 || whole <= 0 {
		return 0, nil
	}
	if whole > 19 {
		return 0, fmt.Errorf("%w: %q out of range", errInvalidDurationValue, s)
	}
	b := make([]byte, 0, 20)
	if negative {
		b = append(b, '-')
	}
	b = append(b, digits[:min(whole, int64(len(digits)))]...)
	for range whole - int64(len(digits)) {
		b = append(b, '0')
	}
	ns, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q out of range", errInvalidDurationValue, s)
	}
	return time.Duration(ns), nil
}

// formatDuration writes d as a bare JSON number of units, exactly to the nanosecond
func formatDuration(d, unit time.Duration) []byte {
	whole, fraction := d/unit, d%unit
	if fraction == 0 {
		return strconv.AppendInt(nil, int64(whole), 10)
	}
	var b []byte
	if d < 0 {
		// Both parts fit once negated, as neither can be math.MinInt64 when unit exceeds one nanosecond
		b = append(b, '-')
		whole, fraction = -whole, -fraction
	}
	b = strconv.AppendInt(b, int64(whole), 10)
	digits := len(strconv.FormatInt(int64(unit), 10)) - 1
	padded := strconv.FormatInt(int64(fraction), 10)
	padded = strings.Repeat("0", digits-len(padded)) + padded
	return append(append(b, '.'), strings.TrimRight(padded, "0")...)
}

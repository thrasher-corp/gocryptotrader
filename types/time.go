package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	onePadding = "0"
	twoPadding = "00"
)

// Time represents a time.Time object that can be unmarshalled from a float64 or string.
// MarshalJSON serializes the time to JSON using RFC 3339 format.
// Note: Not all exchanges may support RFC 3339 for outbound requests, so ensure compatibility with each exchange's time
// format requirements.
type Time time.Time

// UnmarshalJSON deserializes json, and timestamp information.
func (t *Time) UnmarshalJSON(data []byte) error {
	s := string(data)

	switch s {
	case "null", "0", `""`, `"0"`:
		*t = Time(time.Time{})
		return nil
	}

	if s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	badSyntax := false
	target := strings.IndexFunc(s, func(r rune) bool {
		if r == '.' {
			return true
		}
		// types.Time may only parse numbers. The below check ensures an error is thrown. time.Time should be used to
		// parse RFC3339 strings instead.
		badSyntax = r < '0' || r > '9'
		return badSyntax
	})

	if target != -1 {
		if badSyntax {
			return fmt.Errorf("%w for `%v`", strconv.ErrSyntax, string(data))
		}
		s = s[:target] + s[target+1:]
	}

	// The length of the string must be 13, 16, or 19.
	switch len(s) {
	case 12, 15, 18:
		s += onePadding
	case 11, 14, 17:
		s += twoPadding
	}

	standard, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	switch len(s) {
	case 10:
		*t = Time(time.Unix(standard, 0))
	case 13:
		*t = Time(time.UnixMilli(standard))
	case 16:
		*t = Time(time.UnixMicro(standard))
	case 19:
		*t = Time(time.Unix(0, standard))
	default:
		return fmt.Errorf("cannot unmarshal %s into Time", string(data))
	}
	return nil
}

// Time represents a time instance.
func (t Time) Time() time.Time { return time.Time(t) }

// String returns a string representation of the time.
func (t Time) String() string {
	return t.Time().String()
}

// MarshalJSON serializes the time to json.
func (t Time) MarshalJSON() ([]byte, error) {
	return t.Time().MarshalJSON()
}

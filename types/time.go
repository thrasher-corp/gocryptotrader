package types

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
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

	standard, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	switch len(s) {
	case 10:
		// Seconds
		*t = Time(time.Unix(standard, 0))
	case 11, 12:
		// Milliseconds: 1726104395.5 && 1726104395.56
		*t = Time(time.UnixMilli(standard * int64(math.Pow10(13-len(s)))))
	case 13:
		// Milliseconds
		*t = Time(time.UnixMilli(standard))
	case 14:
		// MicroSeconds: 1726106210903.0
		*t = Time(time.UnixMicro(standard * 100))
	case 16:
		// MicroSeconds
		*t = Time(time.UnixMicro(standard))
	case 17:
		// NanoSeconds: 1606292218213.4578
		*t = Time(time.Unix(0, standard*100))
	case 19:
		// NanoSeconds
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

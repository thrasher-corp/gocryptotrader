package types

import (
	"fmt"
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

	if s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	if strings.Trim(s, "0.") == "" || s[0] == 'n' {
		return nil
	}

	if target := strings.Index(s, "."); target != -1 {
		s = s[:target] + s[target+1:]
	}

	// Expects a string of length 10 (seconds), 13 (milliseconds), 16 (microseconds), or 19 (nanoseconds) representing a Unix timestamp
	switch len(s) {
	case 12, 15, 18:
		s += "0"
	case 11, 14, 17:
		s += "00"
	}

	unixTS, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing unix timestamp: %w", err)
	}

	switch len(s) {
	case 10:
		*t = Time(time.Unix(unixTS, 0))
	case 13:
		*t = Time(time.UnixMilli(unixTS))
	case 16:
		*t = Time(time.UnixMicro(unixTS))
	case 19:
		*t = Time(time.Unix(0, unixTS))
	default:
		return fmt.Errorf("cannot unmarshal %s into Time", data)
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

package types

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Time represents a time.Time object that can be unmarshalled from a float64 or string.
// MarshalJSON serialises the time to JSON using RFC 3339 format.
// Note: Not all exchanges may support RFC 3339 for outbound requests, so ensure compatibility with each exchange's time
// format requirements.
type Time time.Time

// ErrInvalidTimestampFormat indicates that a timestamp cannot be parsed into a supported format.
var ErrInvalidTimestampFormat = errors.New("invalid timestamp format")

// UnmarshalJSON deserialises json and timestamp information.
func (t *Time) UnmarshalJSON(data []byte) error {
	timestamp := data

	if timestamp[0] == '"' {
		timestamp = timestamp[1 : len(timestamp)-1]
	}

	if len(timestamp) == 0 || timestamp[0] == 'n' || (len(timestamp) == 1 && timestamp[0] == '0') {
		return nil
	}

	target := bytes.IndexByte(timestamp, '.')
	length := len(timestamp)
	if target != -1 {
		length--
	}
	padding := 0
	switch length {
	case 12, 15, 18: // Expects a string of length 10 (seconds), 13 (milliseconds), 16 (microseconds), or 19 (nanoseconds) representing a Unix timestamp
		padding = 1
	case 11, 14, 17:
		padding = 2
	}
	if target != -1 || padding != 0 {
		var normalised [19]byte // A nanosecond Unix timestamp is the largest supported representation.
		length += padding
		var destination []byte
		if length <= len(normalised) {
			destination = normalised[:length]
		} else {
			destination = make([]byte, length)
		}
		if target == -1 {
			copy(destination, timestamp)
		} else {
			copy(destination, timestamp[:target])
			copy(destination[target:], timestamp[target+1:])
		}
		for x := length - padding; x < length; x++ {
			destination[x] = '0'
		}
		timestamp = destination
		if target != -1 && len(bytes.Trim(timestamp, "0")) == 0 {
			return nil
		}
	}

	if len(timestamp) == 8 {
		value := string(timestamp)
		parsed, err := time.Parse("20060102", value)
		if err != nil {
			return fmt.Errorf("%w error parsing %q into date: %w", ErrInvalidTimestampFormat, value, err)
		}
		*t = Time(parsed)
		return nil
	}

	unixTS, err := strconv.ParseInt(string(timestamp), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing unix timestamp: %w", err)
	}

	switch len(timestamp) {
	case 10:
		*t = Time(time.Unix(unixTS, 0))
	case 13:
		*t = Time(time.UnixMilli(unixTS))
	case 16:
		*t = Time(time.UnixMicro(unixTS))
	case 19:
		*t = Time(time.Unix(0, unixTS))
	default:
		return fmt.Errorf("%w: %q", ErrInvalidTimestampFormat, data)
	}
	return nil
}

// Time represents a time instance.
func (t Time) Time() time.Time { return time.Time(t) }

// String returns a string representation of the time.
func (t Time) String() string {
	return t.Time().String()
}

// MarshalJSON serialises the time to json using RFC 3339 format.
func (t Time) MarshalJSON() ([]byte, error) {
	return t.Time().MarshalJSON()
}

// DateTime represents a time.Time object that can be unmarshalled from a string in the format "2006-01-02 15:04:05".
type DateTime time.Time

// UnmarshalJSON unmarshals json data into a DateTime type.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	var ts string
	if err := json.Unmarshal(data, &ts); err != nil {
		return fmt.Errorf("error unmarshalling %q into string: %w", data, err)
	}

	tm, err := time.Parse(time.DateTime, ts)
	if err != nil {
		return fmt.Errorf("error parsing %q into DateTime: %w", ts, err)
	}

	*d = DateTime(tm)
	return nil
}

// Time converts DateTime to time.Time
func (d DateTime) Time() time.Time {
	return time.Time(d)
}

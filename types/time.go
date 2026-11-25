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

	if len(timestamp) == 0 {
		return nil
	}
	if timestamp[0] == '"' {
		timestamp = timestamp[1 : len(timestamp)-1]
	}

	if len(timestamp) == 0 || timestamp[0] == 'n' || (len(timestamp) == 1 && timestamp[0] == '0') {
		return nil
	}
	if timestamp[0] == '-' {
		return t.unmarshalNegative(data, timestamp)
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

// unmarshalNegative decodes a timestamp before the Unix epoch. The digits before any point pick the unit,
// three widths a unit: 8-10 seconds, 11-13 milliseconds, 14-16 microseconds and 17-19 nanoseconds. For a whole
// part without leading zeros, at least 10,000,000 and under 10,000,000,000 seconds before the epoch, these widths
// name the unit, and a fraction carries up to 9, 6 or 3 digits of a second, millisecond or microsecond;
// nanoseconds fit int64 only back to 1677-09-21. Outside that window another unit can have the same width, as it
// can for positive timestamps.
func (t *Time) unmarshalNegative(data, timestamp []byte) error {
	signedWhole, fraction, point := bytes.Cut(timestamp, []byte{'.'})
	// Keep the receiver for -0 and all-zero values with a point, as the positive zero forms do.
	if (point || len(signedWhole) == 2) && len(bytes.TrimLeft(signedWhole[1:], "0")) == 0 && len(bytes.TrimLeft(fraction, "0")) == 0 {
		return nil
	}
	var unitsPerSecond int64
	var fractionDigits int
	switch len(signedWhole) - 1 {
	case 8, 9, 10:
		unitsPerSecond, fractionDigits = 1, 9
	case 11, 12, 13:
		unitsPerSecond, fractionDigits = 1e3, 6
	case 14, 15, 16:
		unitsPerSecond, fractionDigits = 1e6, 3
	case 17, 18, 19:
		unitsPerSecond, fractionDigits = 1e9, 0
	default:
		return fmt.Errorf("%w: %q", ErrInvalidTimestampFormat, data)
	}
	if len(fraction) > fractionDigits || len(fraction) != 0 && (fraction[0] < '0' || fraction[0] > '9') {
		return fmt.Errorf("%w: %q", ErrInvalidTimestampFormat, data)
	}
	units, err := strconv.ParseInt(string(signedWhole), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing unix timestamp: %w", err)
	}
	nanoseconds := units % unitsPerSecond * (int64(time.Second) / unitsPerSecond)
	if len(fraction) != 0 {
		fractionValue, err := strconv.ParseInt(string(fraction), 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing unix timestamp: %w", err)
		}
		for range fractionDigits - len(fraction) {
			fractionValue *= 10
		}
		nanoseconds -= fractionValue
	}
	*t = Time(time.Unix(units/unitsPerSecond, nanoseconds))
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

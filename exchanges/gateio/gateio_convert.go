package gateio

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	zero     = []byte(`0`)
	emptyStr = []byte(`""`)
	zeroStr  = []byte(`"0"`)
)

// Time represents a time.Time object that can be unmarshalled from a float64 or string.
type Time time.Time

// UnmarshalJSON deserializes json, and timestamp information.
func (a *Time) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, zero) || bytes.Equal(data, emptyStr) || bytes.Equal(data, zeroStr) {
		*a = Time(time.Time{})
		return nil
	}

	s := string(data)
	if s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	target := strings.Index(s, ".")
	if target != -1 {
		s = s[:target] + s[target+1:]
	}

	standard, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	switch len(s) {
	case 10:
		// Seconds
		*a = Time(time.Unix(standard, 0))
	case 11, 12:
		// Milliseconds: 1726104395.5 && 1726104395.56
		*a = Time(time.UnixMilli(standard * int64(math.Pow10(13-len(s)))))
	case 13:
		// Milliseconds
		*a = Time(time.UnixMilli(standard))
	case 14:
		// MicroSeconds: 1726106210903.0
		*a = Time(time.UnixMicro(standard * 100))
	case 16:
		// MicroSeconds
		*a = Time(time.UnixMicro(standard))
	case 17:
		// NanoSeconds: 1606292218213.4578
		*a = Time(time.Unix(0, standard*100))
	case 19:
		// NanoSeconds
		*a = Time(time.Unix(0, standard))
	default:
		return fmt.Errorf("cannot unmarshal %s into Time", string(data))
	}
	return nil
}

// Time represents a time instance.
func (a Time) Time() time.Time { return time.Time(a) }

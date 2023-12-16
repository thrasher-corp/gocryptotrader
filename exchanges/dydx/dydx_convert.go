package dydx

import (
	"strings"
	"time"
)

// dYdXTimeUTC represents a time.Time instance having its own String and Time methods.
type dYdXTimeUTC time.Time

// Time returns a time.Time instance from dydxTimeUTC.
func (a *dYdXTimeUTC) Time() time.Time {
	return time.Time(*a)
}

// String returns a string representation of dydxTimeUTC.
func (a *dYdXTimeUTC) String() string {
	timeString := a.Time().UTC().Format(timeFormat)
	splittedStr := strings.Split(timeString, ".")
	if len(splittedStr) != 2 {
		return "0001-01-01T00:00:00Z"
	}
	end := strings.TrimRight(splittedStr[1], "Z")
	switch len(end) {
	case 1:
		end += "00Z"
	case 2:
		end += "0Z"
	case 3:
		end += "Z"
	default:
		end += "000Z"
	}
	data := []byte{'"'}
	data = append(data, []byte(splittedStr[0]+"."+end)...)
	data = append(data, '"')
	return string(data)
}

// MarshalJSON returns a []byte representation of timestamp
func (a *dYdXTimeUTC) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *dYdXTimeUTC) timeString() string {
	d := []byte(a.String())
	d = d[1 : len(d)-1]
	return string(d)
}

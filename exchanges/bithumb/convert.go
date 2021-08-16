package bithumb

import (
	"encoding/json"
	"strconv"
	"time"
)

// bithumbMSTime provides an internal conversion helper for microsecond parsing
type bithumbTime time.Time

// UnmarshalJSON implements the unmarshal interface
func (t *bithumbTime) UnmarshalJSON(data []byte) error {
	var timestamp string
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}

	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}

	*t = bithumbTime(time.Unix(0, i*int64(time.Microsecond)))
	return nil
}

// Time returns a time.Time object
func (t bithumbTime) Time() time.Time {
	return time.Time(t)
}

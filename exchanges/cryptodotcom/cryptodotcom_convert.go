package cryptodotcom

import (
	"encoding/json"
	"time"
)

type cryptoDotComMilliSec int64

// UnmarshalJSON decerializes timestamp information into a cryptoDotComMilliSec instance.
func (d *cryptoDotComMilliSec) UnmarshalJSON(data []byte) error {
	var value int64
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	*d = cryptoDotComMilliSec(value)
	return nil
}

// Time returns a time.Time instance from the timestamp information.
func (d *cryptoDotComMilliSec) Time() time.Time {
	return time.UnixMilli(int64(*d))
}

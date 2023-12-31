package okcoin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type okcoinTime time.Time

// UnmarshalJSON deserializes timestamp information to time.Time
func (o *okcoinTime) UnmarshalJSON(data []byte) error {
	var timeMilliSecond interface{}
	err := json.Unmarshal(data, &timeMilliSecond)
	if err != nil {
		return err
	}
	var timestamp int64
	switch value := timeMilliSecond.(type) {
	case string:
		if value == "" {
			*o = okcoinTime(time.Time{}) // in case timestamp information is empty string("") reset okcoinTime to zero.
			return nil
		}
		timestamp, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
	case int64:
		timestamp = value
	case float64:
		timestamp = int64(value)
	case float32:
		timestamp = int64(value)
	default:
		return fmt.Errorf("cannot unmarshal %T into okcoinTime", value)
	}
	if timestamp > 9999999999 {
		*o = okcoinTime(time.UnixMilli(timestamp))
	} else {
		*o = okcoinTime(time.Unix(timestamp, 0))
	}
	return nil
}

// Time returns a time.Time instance from okcoinMilliSec instance
func (o *okcoinTime) Time() time.Time {
	return time.Time(*o)
}

package okcoin

import (
	"encoding/json"
	"strconv"
	"time"
)

type okcoinMilliSec int64

// UnmarshalJSON deserializes timestamp information to time.Time
func (o *okcoinMilliSec) UnmarshalJSON(data []byte) error {
	var timeMilliSecond interface{}
	err := json.Unmarshal(data, &timeMilliSecond)
	if err != nil {
		return err
	}
	switch value := timeMilliSecond.(type) {
	case string:
		timeInteger, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		*o = okcoinMilliSec(timeInteger)
	case int64:
		*o = okcoinMilliSec(value)
	}
	return nil
}

// Time returns a time.Time instance from okcoinMilliSec instance
func (o *okcoinMilliSec) Time() time.Time {
	return time.UnixMilli(int64(*o))
}

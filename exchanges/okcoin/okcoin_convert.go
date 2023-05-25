package okcoin

import (
	"encoding/json"
	"strconv"
	"time"
)

type okcoinMilliSec time.Time

// UnmarshalJSON deserializes timestamp information to time.Time
func (o *okcoinMilliSec) UnmarshalJSON(data []byte) error {
	var timeMilliSecond interface{}
	err := json.Unmarshal(data, &timeMilliSecond)
	if err != nil {
		return err
	}
	switch value := timeMilliSecond.(type) {
	case string:
		if value == "" {
			*o = okcoinMilliSec(time.Time{})
			return nil
		}
		timeInteger, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		*o = okcoinMilliSec(time.UnixMilli(timeInteger))
	case int64:
		*o = okcoinMilliSec(time.UnixMilli(value))
	case float64:
		*o = okcoinMilliSec(time.UnixMilli(int64(value)))
	case float32:
		*o = okcoinMilliSec(time.UnixMilli(int64(value)))
	}
	return nil
}

// Time returns a time.Time instance from okcoinMilliSec instance
func (o *okcoinMilliSec) Time() time.Time {
	return time.Time(*o)
}

type okcoinNumber float64

// UnmarshalJSON a custom JSON deserialization function for numeric values to okcoinNumber instance.
func (a *okcoinNumber) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	switch val := value.(type) {
	case string:
		if val == "" {
			*a = okcoinNumber(0)
			return nil
		}
		floatValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*a = okcoinNumber(floatValue)
	case float64:
		*a = okcoinNumber(val)
	case int64:
		*a = okcoinNumber(val)
	case int32:
		*a = okcoinNumber(int64(val))
	}
	return nil
}

// Float64 returns a float64 value from okcoinNumber instance.
func (a *okcoinNumber) Float64() float64 { return float64(*a) }

// String returns string wrapped float64 value from okcoinNumber instance.
func (a *okcoinNumber) String() string { return strconv.FormatFloat(float64(*a), 'f', -1, 64) }

package cryptodotcom

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type cryptoDotComTime time.Time

// UnmarshalJSON converts string embedded unix nano-second timestamp information into cryptoDotComTime instance.
func (a *cryptoDotComTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	var timestamp int64
	switch val := value.(type) {
	case int64:
		timestamp = val
	case int32:
		timestamp = int64(val)
	case float64:
		timestamp = int64(val)
	case string:
		if val == "" {
			*a = cryptoDotComTime(time.Time{})
			return nil
		}
		timestamp, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
	case nil:
		// for some cases when cryptoCom sends a nil value as a zero value timestamp information.
	default:
		return fmt.Errorf("timestamp information of type %T is not supported", value)
	}
	switch {
	case timestamp == 0:
		*a = cryptoDotComTime(time.Time{})
	case timestamp >= 1e13:
		*a = cryptoDotComTime(time.Unix((timestamp / 1e9), timestamp%1e9))
	case timestamp >= 1e10:
		*a = cryptoDotComTime(time.UnixMilli(timestamp))
	default:
		*a = cryptoDotComTime(time.Unix(timestamp, 0))
	}
	return nil
}

// Time returns a time.Time instance from unix nano second timestamp information
func (a *cryptoDotComTime) Time() time.Time {
	return time.Time(*a)
}

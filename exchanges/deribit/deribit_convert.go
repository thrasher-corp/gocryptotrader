package deribit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type deribitMilliSecTime time.Time

// UnmarshalJSON deserializes a byte data into timestamp information
func (a *deribitMilliSecTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	var millisecTimestamp int64
	switch val := value.(type) {
	case int64:
		millisecTimestamp = val
	case int32:
		millisecTimestamp = int64(val)
	case float64:
		millisecTimestamp = int64(val)
	case string:
		if val == "" {
			*a = deribitMilliSecTime(time.Time{}) // reset previous timestamp information if exist
			return nil
		}
		millisecTimestamp, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
	case nil:
		// for some cases when deribit send a nil value as a zero value timestamp information.
	default:
		return fmt.Errorf("unsupported timestamp type %T", val)
	}
	if millisecTimestamp > 0 {
		*a = deribitMilliSecTime(time.UnixMilli(millisecTimestamp))
	} else {
		*a = deribitMilliSecTime(time.Time{})
	}
	return nil
}

// Time returns a time.Time instance information from deribitMilliSecTime timestamp.
func (a *deribitMilliSecTime) Time() time.Time {
	return time.Time(*a)
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *MarkPriceHistory) UnmarshalJSON(data []byte) error {
	var resp [2]float64
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	a.Timestamp = deribitMilliSecTime(time.UnixMilli(int64(resp[0])))
	a.MarkPriceValue = resp[1]
	return nil
}

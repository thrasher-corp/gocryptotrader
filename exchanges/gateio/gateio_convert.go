package gateio

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

type gateioTime time.Time

// UnmarshalJSON deserializes json, and timestamp information.
func (a *gateioTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	var standard int64
	switch val := value.(type) {
	case float64:
		if math.Trunc(val) != val {
			standard = int64(val * 1e3) // Account for 1684981731.098
		} else {
			standard = int64(val)
		}
	case string:
		if val == "" {
			return nil
		}
		parsedValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		if math.Trunc(parsedValue) != parsedValue {
			*a = gateioTime(time.UnixMicro(int64(parsedValue * 1e3))) // Account for "1691122380942.173000" microseconds
			return nil
		}
		standard = int64(parsedValue)
	default:
		return fmt.Errorf("cannot unmarshal %T into gateioTime", val)
	}
	if standard > 9999999999 {
		*a = gateioTime(time.UnixMilli(standard))
	} else {
		*a = gateioTime(time.Unix(standard, 0))
	}
	return nil
}

// Time represents a time instance.
func (a gateioTime) Time() time.Time { return time.Time(a) }

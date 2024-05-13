package gateio

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// GateioTime represents a time.Time object that can be unmarshalled from a float64 or string.
type GateioTime time.Time

// UnmarshalJSON deserializes json, and timestamp information.
func (a *GateioTime) UnmarshalJSON(data []byte) error {
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
			*a = GateioTime(time.UnixMicro(int64(parsedValue * 1e3))) // Account for "1691122380942.173000" microseconds
			return nil
		}
		standard = int64(parsedValue)
	default:
		return fmt.Errorf("cannot unmarshal %T into GateioTime", val)
	}
	if standard > 9999999999 {
		*a = GateioTime(time.UnixMilli(standard))
	} else {
		*a = GateioTime(time.Unix(standard, 0))
	}
	return nil
}

// Time represents a time instance.
func (a GateioTime) Time() time.Time { return time.Time(a) }

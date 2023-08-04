package gateio

import (
	"encoding/json"
	"fmt"
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
		standard = int64(val)
	case int64:
		standard = val
	case int32:
		standard = int64(val)
	case string:
		if val == "" {
			return nil
		}
		parsedValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
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

type gateioNumericalValue float64

// UnmarshalJSON is custom type json unmarshaller for gateioNumericalValue
func (a *gateioNumericalValue) UnmarshalJSON(data []byte) error {
	var num interface{}
	err := json.Unmarshal(data, &num)
	if err != nil {
		return err
	}

	switch d := num.(type) {
	case float64:
		*a = gateioNumericalValue(d)
	case string:
		if d == "" {
			*a = gateioNumericalValue(0)
			return nil
		}
		convNum, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return err
		}
		*a = gateioNumericalValue(convNum)
	}
	return nil
}

// Float64 returns float64 value from gateioNumericalValue instance.
func (a gateioNumericalValue) Float64() float64 { return float64(a) }

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsUserPersonalTrade) UnmarshalJSON(data []byte) error {
	type Alias WsUserPersonalTrade
	chil := &struct {
		*Alias
		CreateTimeMicroS float64 `json:"create_time_ms,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTimeMicroS = time.UnixMicro(int64(chil.CreateTimeMicroS * 1000))
	return nil
}

package bybit

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type bybitNumber float64

// Float64 returns an float64 value from kucoinNumeric instance
func (a *bybitNumber) Float64() float64 {
	return float64(*a)
}

// UnmarshalJSON deserializes float and string data having an float value to float64
func (a *bybitNumber) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	switch val := value.(type) {
	case float64:
		*a = bybitNumber(val)
	case float32:
		*a = bybitNumber(val)
	case string:
		if val == "" {
			*a = bybitNumber(0) // setting empty string value to zero to reset previous value if exist.
			return nil
		}
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*a = bybitNumber(value)
	case int64:
		*a = bybitNumber(val)
	case int32:
		*a = bybitNumber(val)
	default:
		return fmt.Errorf("unsupported input numeric type %T", value)
	}
	return nil
}

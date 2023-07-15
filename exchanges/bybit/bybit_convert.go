package bybit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type bybitNumber float64

// Float64 returns an float64 value from kucoinNumeric instance
func (a *bybitNumber) Float64() float64 {
	return float64(*a)
}

// Int64 returns an int64 value from kucoinNumeric instance
func (a *bybitNumber) Int64() int64 {
	return int64(*a)
}

// UnmarshalJSON decerializes integer and string data having an integer value to int64
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

// UnmarshalJSON deserializes []byte data into []WsFuturesOrderbookData instance.
func (o *wsUSDTOBData) UnmarshalJSON(data []byte) error {
	var resp interface{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	switch resp.(type) {
	case []interface{}:
		var list []WsFuturesOrderbookData
		err := json.Unmarshal(data, &list)
		if err != nil {
			return err
		}
		*o = list
	case map[string]interface{}:
		list := struct {
			OBData []WsFuturesOrderbookData `json:"order_book"`
		}{}
		err := json.Unmarshal(data, &list)
		if err != nil {
			return err
		}
		*o = list.OBData
	default:
		return errors.New("invalid JSON data")
	}
	return nil
}

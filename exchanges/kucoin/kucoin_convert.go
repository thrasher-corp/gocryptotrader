package kucoin

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// UnmarshalJSON valid data to SubAccountsResponse of return nil if the data is empty list.
// this is added to handle the empty list returned when there are no accounts.
func (a *SubAccountsResponse) UnmarshalJSON(data []byte) error {
	var result interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	var ok bool
	if a, ok = result.(*SubAccountsResponse); ok {
		if a == nil {
			return errNoValidResponseFromServer
		}
		return nil
	} else if _, ok := result.([]interface{}); ok {
		return nil
	}
	return fmt.Errorf("%w can not unmarshal to SubAccountsResponse", errMalformedData)
}

// kucoinNumber unmarshals and extract numeric value from a byte slice.
type kucoinNumber float64

// Float64 returns an float64 value from kucoinNumeric instance
func (a *kucoinNumber) Float64() float64 {
	return float64(*a)
}

// UnmarshalJSON decerializes integer and string data having an integer value to int64
func (a *kucoinNumber) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	switch val := value.(type) {
	case float64:
		*a = kucoinNumber(val)
	case float32:
		*a = kucoinNumber(val)
	case string:
		if val == "" {
			*a = kucoinNumber(0) // setting empty string value to zero to reset previous value if exist.
			return nil
		}
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*a = kucoinNumber(value)
	case int64:
		*a = kucoinNumber(val)
	case int32:
		*a = kucoinNumber(val)
	default:
		return fmt.Errorf("unsupported input numeric type %T", value)
	}
	return nil
}

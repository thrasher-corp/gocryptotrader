package kucoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// kucoinTimeSec provides an internal conversion helper
type kucoinTimeSec int64

// Time returns a time.Time object
func (k kucoinTimeSec) Time() time.Time {
	if k < 0 {
		return time.Time{}
	}
	return time.Unix(int64(k), 0)
}

// kucoinTimeNanoSec provides an internal conversion helper
type kucoinTimeNanoSec int64

// Time returns a time.Time object
func (k *kucoinTimeNanoSec) Time() time.Time {
	if *k < 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(*k))
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeSec
func (k *kucoinTimeSec) UnmarshalJSON(data []byte) error {
	var timestamp interface{}
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	switch value := timestamp.(type) {
	case int64:
		*k = kucoinTimeSec(value)
	case int:
		*k = kucoinTimeSec(int64(value))
	case float64:
		*k = kucoinTimeSec(int64(value))
	case string:
		if value == "" {
			// Setting the time to zero because some timestamp fields could return an empty string while there is no error
			// So, in such cases, kucoinTimeSec returns 0 timestamp.
			*k = kucoinTimeSec(-1)
			return nil
		}
		tmsp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		*k = kucoinTimeSec(tmsp)
	default:
		*k = kucoinTimeSec(0)
	}
	return nil
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeMilliSec
func (k *kucoinTimeMilliSec) UnmarshalJSON(data []byte) error {
	var timestamp interface{}
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	switch value := timestamp.(type) {
	case int64:
		*k = kucoinTimeMilliSec(value)
	case int:
		*k = kucoinTimeMilliSec(int64(value))
	case float64:
		*k = kucoinTimeMilliSec(int64(value))
	case string:
		if value == "" {
			// Setting the time to zero because some timestamp fields could return an empty string while there is no error
			// So, in such cases, kucoinTimeMilliSec returns 0 timestamp.
			*k = kucoinTimeMilliSec(-1)
			return nil
		}
		tmsp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		*k = kucoinTimeMilliSec(tmsp)
	default:
		*k = kucoinTimeMilliSec(0)
	}
	return nil
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeNanoSec
func (k *kucoinTimeNanoSec) UnmarshalJSON(data []byte) error {
	var timestamp interface{}
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	switch val := timestamp.(type) {
	case int64:
		*k = kucoinTimeNanoSec(val)
	case string:
		if val == "" {
			// Setting the time to zero because some timestamp fields could return an empty string while there is no error
			// So, in such cases, kucoinTimeNanoSec returns 0 timestamp.
			*k = kucoinTimeNanoSec(-1)
			return nil
		}
		tmsp, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		*k = kucoinTimeNanoSec(tmsp)
	case int:
		*k = kucoinTimeNanoSec(int64(val))
	case float64:
		*k = kucoinTimeNanoSec(int64(val))
	default:
		*k = kucoinTimeNanoSec(0)
	}
	return nil
}

type kucoinAmbiguousFloat float64

// UnmarshalJSON is custom type json unmarshaller for kucoinUmbiguousFloat
func (k *kucoinAmbiguousFloat) UnmarshalJSON(data []byte) error {
	var newVal interface{}
	err := json.Unmarshal(data, &newVal)
	if err != nil {
		return err
	}
	switch payload := newVal.(type) {
	case float64:
		*k = kucoinAmbiguousFloat(payload)
	case string:
		value, err := strconv.ParseFloat(payload, 64)
		if err != nil {
			return err
		}
		*k = kucoinAmbiguousFloat(value)
	default:
		return fmt.Errorf("unhandled type %T", newVal)
	}
	return nil
}

// Float64 returns floating values from kucoinUmbiguousFloat.
func (k *kucoinAmbiguousFloat) Float64() float64 {
	return float64(*k)
}

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

// kucoinInteger created to convert into int64 from string or int and hold the data
type kucoinInteger int64

// Value returns an int64 value from kucoinInteger instance
func (a *kucoinInteger) Value() int64 {
	return int64(*a)
}

// UnmarshalJSON decerializes integer and string data having an integer value to int64
func (a *kucoinInteger) UnmarshalJSON(data []byte) error {
	var integer interface{}
	err := json.Unmarshal(data, &integer)
	if err != nil {
		return err
	}
	switch val := integer.(type) {
	case int64:
		*a = kucoinInteger(val)
	case int:
		*a = kucoinInteger(int64(val))
	case string:
		if val == "" {
			return errors.New("empty string as integer")
		}
		value, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		*a = kucoinInteger(value)
	case float64:
		*a = kucoinInteger(int64(val))
	case float32:
		*a = kucoinInteger(int64(val))
	default:
		return errors.New("unsupported integer value")
	}
	return nil
}

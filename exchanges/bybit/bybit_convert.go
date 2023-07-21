package bybit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type bybitTime time.Time

// UnmarshalJSON deserializes timestamp information to time.Time
func (o *bybitTime) UnmarshalJSON(data []byte) error {
	var timeMilliSecond interface{}
	err := json.Unmarshal(data, &timeMilliSecond)
	if err != nil {
		return err
	}
	var timestamp int64
	switch value := timeMilliSecond.(type) {
	case string:
		if value == "" {
			*o = bybitTime(time.Time{}) // in case timestamp information is empty string("") reset bybitTime to zero.
			return nil
		}
		timestamp, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
	case int64:
		timestamp = value
	case float64:
		timestamp = int64(value)
	case float32:
		timestamp = int64(value)
	default:
		return fmt.Errorf("cannot unmarshal %T into bybitTime", value)
	}

	switch {
	case timestamp == 0:
		*o = bybitTime(time.Time{})
	case timestamp >= 1e18: // Nanoseconds
		*o = bybitTime(time.Unix(timestamp/1e9, timestamp%1e9))
	case timestamp >= 1e10: // Milliseconds
		*o = bybitTime(time.Unix(timestamp/1e3, 0))
	default: // Seconds
		*o = bybitTime(time.Unix(timestamp, 0))
	}
	return nil
}

// Time returns a time.Time instance from bybitMilliSec instance
func (o *bybitTime) Time() time.Time {
	return time.Time(*o)
}

type bybitNumber float64

// Float64 returns an float64 value from kucoinNumeric instance
func (a *bybitNumber) Float64() float64 {
	return float64(*a)
}

// Int64 returns an int64 value from kucoinNumeric instance
func (a *bybitNumber) Int64() int64 {
	return int64(*a)
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

// UnmarshalJSON deserializes []byte data into []WsFuturesOrderbookData instance.
func (o *wsFuturesOBData) UnmarshalJSON(data []byte) error {
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

func (o *WsOrderData) GetTime(a asset.Item) time.Time {
	switch a {
	case asset.USDTMarginedFutures:
		return o.CreateTime
	default:
		return o.Time
	}
}

func (o *WsStopOrderData) GetTime(a asset.Item) time.Time {
	switch a {
	case asset.USDTMarginedFutures:
		return o.CreateTime
	default:
		return o.Time
	}
}

func (t *WsFuturesTickerData) GetVolume24h() float64 {
	if t.Volume24h.Float64() != 0 {
		return t.Volume24h.Float64()
	}
	return t.Volume24hE8.Float64() / 100000000.0
}

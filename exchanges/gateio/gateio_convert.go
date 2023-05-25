package gateio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type gateioMilliSecTime time.Time

func (a *gateioMilliSecTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	switch val := value.(type) {
	case float64:
		*a = gateioMilliSecTime(time.UnixMilli(int64(val)))
	case int64:
		*a = gateioMilliSecTime(time.UnixMilli(val))
	case int32:
		*a = gateioMilliSecTime(time.UnixMilli(int64(val)))
	case string:
		if val == "" {
			return nil
		}
		parsedValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*a = gateioMilliSecTime(time.UnixMilli(int64(parsedValue)))
	default:
		return fmt.Errorf("cannot unmarshal %T into gateioMilliSecTime", val)
	}
	return nil
}

// Time represents a time instance.
func (a *gateioMilliSecTime) Time() time.Time { return time.Time(*a) }

type gateioTime time.Time

// UnmarshalJSON deserializes json, and timestamp information.
func (a *gateioTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	switch val := value.(type) {
	case int64:
		*a = gateioTime(time.Unix(val, 0))
	case int32:
		*a = gateioTime(time.Unix(int64(val), 0))
	case float64:
		*a = gateioTime(time.Unix(int64(val), 0))
	case string:
		if val == "" {
			return nil
		}
		parsedValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		*a = gateioTime(time.Unix(int64(parsedValue), 0))
	default:
		return fmt.Errorf("cannot unmarshal %T into gateioTime", val)
	}
	return nil
}

// Time represents a time instance.
func (a *gateioTime) Time() time.Time { return time.Time(*a) }

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
func (a *gateioNumericalValue) Float64() float64 { return float64(*a) }

// UnmarshalJSON to deserialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *Orderbook) UnmarshalJSON(data []byte) error {
	type Alias Orderbook
	type askorbid struct {
		Price gateioNumericalValue `json:"p"`
		Size  float64              `json:"s"`
	}
	chil := &struct {
		*Alias
		Current float64    `json:"current"`
		Update  float64    `json:"update"`
		Asks    []askorbid `json:"asks"`
		Bids    []askorbid `json:"bids"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, &chil)
	if err != nil {
		return err
	}
	a.Current = time.UnixMilli(int64(chil.Current * 1000))
	a.Update = time.UnixMilli(int64(chil.Update * 1000))
	a.Asks = make([]OrderbookItem, len(chil.Asks))
	a.Bids = make([]OrderbookItem, len(chil.Bids))
	for x := range chil.Asks {
		a.Asks[x] = OrderbookItem{
			Amount: chil.Asks[x].Size,
			Price:  chil.Asks[x].Price.Float64(),
		}
	}
	for x := range chil.Bids {
		a.Bids[x] = OrderbookItem{
			Amount: chil.Bids[x].Size,
			Price:  chil.Bids[x].Price.Float64(),
		}
	}
	return nil
}

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

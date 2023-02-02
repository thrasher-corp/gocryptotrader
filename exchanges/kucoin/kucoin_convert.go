package kucoin

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsOrderbookLevel5) UnmarshalJSON(data []byte) error {
	type Alias WsOrderbookLevel5
	chil := &struct {
		*Alias
		Asks      [][2]float64 `json:"asks"`
		Bids      [][2]float64 `json:"bids"`
		Timestamp int64        `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Asks = make([]orderbook.Item, len(chil.Asks))
	for x := range chil.Asks {
		a.Asks[x] = orderbook.Item{
			Price:  chil.Asks[x][0],
			Amount: chil.Asks[x][1],
		}
	}
	a.Bids = make([]orderbook.Item, len(chil.Bids))
	for x := range chil.Bids {
		a.Bids[x] = orderbook.Item{
			Price:  chil.Bids[x][0],
			Amount: chil.Bids[x][1],
		}
	}
	a.Timestamp = time.Unix(0, chil.Timestamp)
	return nil
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeMilliSec
func (k *kucoinTimeMilliSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*k = kucoinTimeMilliSec(timestamp)
	return nil
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeMilliSecStr
func (k *kucoinTimeMilliSecStr) UnmarshalJSON(data []byte) error {
	var timestamp string
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}

	t, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return err
	}
	*k = kucoinTimeMilliSecStr(time.UnixMilli(t))
	return nil
}

// UnmarshalJSON is custom type json unmarshaller for kucoinTimeNanoSec
func (k *kucoinTimeNanoSec) UnmarshalJSON(data []byte) error {
	var timestamp int64
	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}
	*k = kucoinTimeNanoSec(time.Unix(0, timestamp))
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
	val := reflect.ValueOf(newVal)
	if val.Kind() == reflect.Float64 {
		*k = kucoinAmbiguousFloat(val.Float())
	} else if val.Kind() == reflect.String {
		value, err := strconv.ParseFloat(newVal.(string), 64)
		if err != nil {
			return err
		}
		*k = kucoinAmbiguousFloat(value)
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
	var result []interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		err = json.Unmarshal(data, a)
		if err != nil {
			return err
		}
	}
	return nil
}

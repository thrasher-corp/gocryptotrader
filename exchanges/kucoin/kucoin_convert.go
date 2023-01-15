package kucoin

import (
	"encoding/json"
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

package deribit

import (
	"encoding/json"
	"strconv"
	"time"
)

type deribitMilliSecTime int64

// UnmarshalJSON deserializes a byte data into timestamp information
func (a *deribitMilliSecTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	var millisecTimestamp int64
	switch val := value.(type) {
	case int64:
		millisecTimestamp = val
	case int:
		millisecTimestamp = int64(val)
	case float64:
		millisecTimestamp = int64(val)
	case string:
		millisecTimestamp, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
	default:
		*a = deribitMilliSecTime(-1)
	}
	*a = deribitMilliSecTime(millisecTimestamp)
	return nil
}

// Time returns a time.Time instance information from deribitMilliSecTime timestamp.
func (a *deribitMilliSecTime) Time() time.Time {
	if val := int64(*a); val >= 0 {
		return time.UnixMilli(int64(*a))
	}
	return time.Time{}
}

// UnmarshalJSON deserializes a JSON object to an orderbook struct.
func (a *Orderbook) UnmarshalJSON(data []byte) error {
	type Alias Orderbook
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON deserializes timestamp information to RFQ instance.
func (a *RequestForQuote) UnmarshalJSON(data []byte) error {
	type Alias RequestForQuote
	chil := &struct {
		*Alias
		LastRfqTimestamp int64 `json:"last_rfq_tstamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.LastRFQTimestamp = time.UnixMilli(chil.LastRfqTimestamp)
	return nil
}

// UnmarshalJSON deserializes json data, including timestamp information
func (a *ComboDetail) UnmarshalJSON(data []byte) error {
	type Alias ComboDetail
	chil := &struct {
		*Alias
		CreationTimestamp int64 `json:"creation_timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreationTimestamp = time.UnixMilli(chil.CreationTimestamp)
	return nil
}

// UnmarshalJSON deserializes json data, including timestamp information
func (a *Announcement) UnmarshalJSON(data []byte) error {
	type Alias Announcement
	chil := &struct {
		*Alias
		PublicationTimestamp int64 `json:"publication_timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	a.PublicationTimestamp = time.UnixMilli(chil.PublicationTimestamp)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *AccessLogDetail) UnmarshalJSON(data []byte) error {
	type Alias AccessLogDetail
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *BlockTradeResponse) UnmarshalJSON(data []byte) error {
	type Alias BlockTradeResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMilli(chil.Timestamp)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp.
func (a *MarkPriceHistory) UnmarshalJSON(data []byte) error {
	var resp [2]float64
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	a.Timestamp = deribitMilliSecTime(int64(resp[0]))
	a.MarkPriceValue = resp[1]
	return nil
}

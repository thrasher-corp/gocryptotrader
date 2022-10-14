package deribit

import (
	"encoding/json"
	"time"
)

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
func (a *RFQ) UnmarshalJSON(data []byte) error {
	type Alias RFQ
	chil := &struct {
		*Alias
		LastRfqTimestamp int64 `json:"last_rfq_tstamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.LastRfqTimestamp = time.UnixMilli(chil.LastRfqTimestamp)
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

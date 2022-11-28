package kucoin

import (
	"encoding/json"
	"time"
)

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsPositionStatus) UnmarshalJSON(data []byte) error {
	type Alias WsPositionStatus
	chil := &struct {
		*Alias
		TimestampMS int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.TimestampMS = time.UnixMilli(chil.TimestampMS)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsDebtRatioChange) UnmarshalJSON(data []byte) error {
	type Alias WsDebtRatioChange
	chil := &struct {
		*Alias
		TimestampMS int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMilli(chil.TimestampMS)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsMarginTradeOrderEntersEvent) UnmarshalJSON(data []byte) error {
	type Alias WsMarginTradeOrderEntersEvent
	chil := &struct {
		*Alias
		TimestampNS int64 `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMicro(int64(chil.TimestampNS / 1000))
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsMarginTradeOrderDoneEvent) UnmarshalJSON(data []byte) error {
	type Alias WsMarginTradeOrderDoneEvent
	chil := &struct {
		*Alias
		TimestampNS int64 `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMicro(int64(chil.TimestampNS / 1000))
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsStopOrder) UnmarshalJSON(data []byte) error {
	type Alias WsStopOrder
	chil := &struct {
		*Alias
		CreatedAt   int64 `json:"createdAt"`
		TimestampNS int64 `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &chil); err != nil {
		return err
	}
	a.Timestamp = time.UnixMicro(int64(chil.TimestampNS / 1000))
	a.CreatedAt = time.UnixMicro(int64(chil.CreatedAt / 1000))
	return nil
}

package okx

import (
	"encoding/json"
	"time"
)

func (a *OrderBookResponse) UnmarshalJSON(data []byte) error {
	type Alias OrderBookResponse
	chil := &struct {
		*Alias
		GenerationTimeStamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.GenerationTimeStamp > 0 {
		a.GenerationTimeStamp = time.UnixMilli(chil.GenerationTimeStamp)
	}
	return nil
}

func (a *TradeResponse) UnmarshalJSON(data []byte) error {
	type Alias TradeResponse
	chil := &struct {
		*Alias
		TimeStamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.TimeStamp > 0 {
		a.TimeStamp = time.UnixMilli(chil.TimeStamp)
	}
	return nil
}

// UnmarshalJSON
func (a *TradingVolumdIn24HR) UnmarshalJSON(data []byte) error {
	type Alias TradingVolumdIn24HR
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

func (a *OracleSmartContractResponse) UnmarshalJSON(data []byte) error {
	type Alias OracleSmartContractResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"timestamp,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

func (a *IndexComponent) UnmarshalJSON(data []byte) error {
	type Alias IndexComponent
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	er := json.Unmarshal(data, chil)
	if er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON
func (a *Instrument) UnmarshalJSON(data []byte) error {
	type Alias Instrument
	chil := &struct {
		*Alias
		ListTime int64 `json:"listTime"`
		ExpTime  int64 `json:"expTime"`
	}{Alias: (*Alias)(a)}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.ListTime > 0 {
		a.ListTime = time.UnixMilli(chil.ListTime)
	}
	if chil.ExpTime > 0 {
		a.ExpTime = time.UnixMilli(chil.ExpTime)
	}
	return nil
}

// UnmarshalJSON unmarshals the json obeject to the DeliveryHistoryResponse
func (a *DeliveryHistoryResponse) UnmarshalJSON(data []byte) error {
	type Alias DeliveryHistoryResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

// UnmarshalJSON decoder for OpenInterestResponse instance.
func (a *OpenInterestResponse) UnmarshalJSON(data []byte) error {
	type Alias OpenInterestResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts"`
	}{Alias: (*Alias)(a)}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

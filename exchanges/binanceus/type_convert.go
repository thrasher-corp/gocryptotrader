package binanceus

import (
	"encoding/json"
	"strconv"

	// "strconv"
	"time"
)

// binanceTime provides an internal conversion helper
type binanceTime time.Time

func (t *binanceTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	*t = binanceTime(time.UnixMilli(timestamp))
	return nil
}

// Time returns a time.Time object
func (t binanceTime) Time() time.Time {
	return time.Time(t)
}

// timeString gets the time as Binance timestamp
func timeString(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func (a *RecentTrade) UnmarshalJSON(data []byte) error {
	type Alias RecentTrade
	chil := &struct {
		Time int64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Time > 0 {
		a.Time = time.UnixMilli(chil.Time)
	}
	return nil
}

func (a *HistoricalTrade) UnmarshalJSON(data []byte) error {
	type Alias HistoricalTrade
	chil := &struct {
		Time int64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Time > 0 {
		a.Time = time.UnixMilli(chil.Time)
	}
	return nil
}

func (a *AggregatedTrade) UnmarshalJSON(data []byte) error {
	type Alias AggregatedTrade
	chil := &struct {
		TimeStamp int64 `json:"T"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.TimeStamp > 0 {
		a.TimeStamp = time.UnixMilli(chil.TimeStamp)
	}
	return nil
}

func (a *PriceChangeStats) UnmarshalJSON(data []byte) error {
	type Alias PriceChangeStats
	chil := &struct {
		OpenTime  int64 `json:"openTime"`
		CloseTime int64 `json:"closeTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.OpenTime > 0 {
		a.OpenTime = time.UnixMilli(chil.OpenTime)
	}
	if chil.CloseTime > 0 {
		a.CloseTime = time.UnixMilli(chil.CloseTime)
	}
	return nil
}

func (a *TradeStatus) UnmarshalJSON(data []byte) error {
	type Alias TradeStatus
	chil := &struct {
		UpdateTime int64 `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.UpdateTime > 0 {
		a.UpdateTime = time.UnixMilli(int64(chil.UpdateTime))
	}
	return nil
}

func (a *SubAccount) UnmarshalJSON(data []byte) error {
	type Alias SubAccount
	chil := &struct {
		CreateTime int64 `json:"createTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.CreateTime > 0 {
		a.CreateTime = time.UnixMilli(chil.CreateTime)
	}
	return nil
}
func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	chil := &struct {
		UpdateTime int64 `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.UpdateTime > 0 {
		a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	}
	return nil
}

// UnmarshalJSON  implementing the Unmarshaler interface

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *NewOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias NewOrderResponse
	aux := &struct {
		TransactionTime binanceTime `json:"transactTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux != nil {
		a.TransactionTime = aux.TransactionTime.Time()
	}
	return nil
}

// UnmarshalJSON .. the Struct Transffer history implements the Unmarshaler interface
func (a *TransferHistory) UnmarshalJSON(data []byte) error {
	type Alias TransferHistory
	aux := &struct {
		TimeStamp uint64 `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.TimeStamp == 0 {
		a.TimeStamp = time.UnixMilli(int64(aux.TimeStamp))
	}
	return nil
}

func (a *ExchangeInfo) UnmarshalJSON(data []byte) error {
	type Alias ExchangeInfo
	chil := &struct {
		Servertime uint64 `json:"serverTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Servertime > 0 {
		a.Servertime = time.UnixMilli(int64(chil.Servertime))
	}
	return nil
}

// UnmarshalJSON  for a struct Order
func (a *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	chil := &struct {
		Time       int64 `json:"time"`
		UpdateTime int64 `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Time > 0 {
		a.Time = time.UnixMilli(chil.Time)
	}
	if chil.UpdateTime > 0 {
		a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	}
	return nil
}

func (a *Trade) UnmarshalJSON(data []byte) error {
	type Alie Trade
	chil := &struct {
		Time int64 `json:"time"`
		*Alie
	}{
		Alie: (*Alie)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Time > 0 {
		a.Time = time.UnixMilli(chil.Time)
	}
	return nil
}

func (a *OCOOrderReportItem) UnmarshalJSON(data []byte) error {
	type Alias OCOOrderReportItem
	chil := &struct {
		*Alias
		TransactionTime int64 `json:"transactionTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.TransactionTime > 0 {
		a.TransactionTime = time.UnixMilli(chil.TransactionTime)
	}
	return nil
}

//
func (a *OCOOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias OCOOrderResponse
	chil := &struct {
		*Alias
		TransactionTime int64 `json:"transactionTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.TransactionTime > 0 {
		a.TransactionTime = time.UnixMilli(chil.TransactionTime)
	}
	return nil
}

// UnmarshalJSON
func (a *OTCTradeOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias OTCTradeOrderResponse
	chil := &struct {
		CreateTime int64 `json:"createTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.CreateTime > 0 {
		a.CreateTime = time.UnixMilli(chil.CreateTime)
	}
	return nil
}

func (a *OTCTradeOrder) UnmarshalJSON(data []byte) error {
	type Alias OTCTradeOrder
	chil := &struct {
		CreateTime int64 `json:"createTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.CreateTime > 0 {
		a.CreateTime = time.UnixMilli(chil.CreateTime)
	}
	return nil
}

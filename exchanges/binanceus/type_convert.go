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

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsListStatus) UnmarshalJSON(data []byte) error {
	type Alias wsListStatus
	aux := &struct {
		Data struct {
			EventTime       binanceTime `json:"E"`
			TransactionTime binanceTime `json:"T"`
			*WsListStatusData
		} `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Data = *aux.Data.WsListStatusData
	a.Data.EventTime = aux.Data.EventTime.Time()
	a.Data.TransactionTime = aux.Data.TransactionTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *TickerStream) UnmarshalJSON(data []byte) error {
	type Alias TickerStream
	aux := &struct {
		EventTime binanceTime `json:"E"`
		OpenTime  binanceTime `json:"O"`
		CloseTime binanceTime `json:"C"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.EventTime = aux.EventTime.Time()
	a.OpenTime = aux.OpenTime.Time()
	a.CloseTime = aux.CloseTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *KlineStream) UnmarshalJSON(data []byte) error {
	type Alias KlineStream
	aux := &struct {
		EventTime binanceTime `json:"E"`
		Kline     struct {
			StartTime binanceTime `json:"t"`
			CloseTime binanceTime `json:"T"`
			*KlineStreamData
		} `json:"k"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Kline = *aux.Kline.KlineStreamData
	a.EventTime = aux.EventTime.Time()
	a.Kline.StartTime = aux.Kline.StartTime.Time()
	a.Kline.CloseTime = aux.Kline.CloseTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *TradeStream) UnmarshalJSON(data []byte) error {
	type Alias TradeStream
	aux := &struct {
		TimeStamp binanceTime `json:"T"`
		EventTime binanceTime `json:"E"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.TimeStamp = aux.TimeStamp.Time()
	a.EventTime = aux.EventTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsOrderUpdate) UnmarshalJSON(data []byte) error {
	type Alias wsOrderUpdate
	aux := &struct {
		Data struct {
			EventTime         binanceTime `json:"E"`
			OrderCreationTime binanceTime `json:"O"`
			TransactionTime   binanceTime `json:"T"`
			*WsOrderUpdateData
		} `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Data = *aux.Data.WsOrderUpdateData
	a.Data.EventTime = aux.Data.EventTime.Time()
	a.Data.OrderCreationTime = aux.Data.OrderCreationTime.Time()
	a.Data.TransactionTime = aux.Data.TransactionTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsBalanceUpdate) UnmarshalJSON(data []byte) error {
	type Alias wsBalanceUpdate
	aux := &struct {
		Data struct {
			EventTime binanceTime `json:"E"`
			ClearTime binanceTime `json:"T"`
			*WsBalanceUpdateData
		} `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Data = *aux.Data.WsBalanceUpdateData
	a.Data.EventTime = aux.Data.EventTime.Time()
	a.Data.ClearTime = aux.Data.ClearTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsAccountPosition) UnmarshalJSON(data []byte) error {
	type Alias wsAccountPosition
	aux := &struct {
		Data struct {
			EventTime   binanceTime `json:"E"`
			LastUpdated binanceTime `json:"u"`
			*WsAccountPositionData
		} `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Data = *aux.Data.WsAccountPositionData
	a.Data.EventTime = aux.Data.EventTime.Time()
	a.Data.LastUpdated = aux.Data.LastUpdated.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsAccountInfo) UnmarshalJSON(data []byte) error {
	type Alias wsAccountInfo
	aux := &struct {
		Data struct {
			EventTime   binanceTime `json:"E"`
			LastUpdated binanceTime `json:"u"`
			*WsAccountInfoData
		} `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Data = *aux.Data.WsAccountInfoData
	a.Data.EventTime = aux.Data.EventTime.Time()
	a.Data.LastUpdated = aux.Data.LastUpdated.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WebsocketDepthStream) UnmarshalJSON(data []byte) error {
	type Alias WebsocketDepthStream
	aux := &struct {
		Timestamp binanceTime `json:"E"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Timestamp = aux.Timestamp.Time()
	return nil
}

// UnmarshalJSON .. .
func (a *WebsocketAggregateTradeStream) UnmarshalJSON(data []byte) error {
	type Alias WebsocketAggregateTradeStream
	chil := &struct {
		*Alias
		TradeTime int64 `json:"T"`
		EventTime int64 `json:"E"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	if chil.TradeTime > 0 {
		a.TradeTime = time.UnixMilli(chil.TradeTime)
	}
	if chil.EventTime > 0 {
		a.EventTime = time.UnixMilli(chil.EventTime)
	}
	return nil
}

// func (a *OrderBookTickerStream) UnmarshalJSON(data []byte) error {
// 	type Alias OrderBookTickerStream
// 	child := &struct {
// 		*Alias
// 		S string `json:"s"`
// 	}{
// 		Alias: (*Alias)(a),
// 	}
// 	err := json.Unmarshal(data, child)
// 	if err != nil {
// 		return err
// 	}
// 	if child.S != "" {
// 		a.Symbol, err = currency.NewPairFromString(child.S)
// 		return nil
// 	}
// 	return nil
// }

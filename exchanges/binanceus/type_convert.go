package binanceus

import (
	"encoding/json"
	"time"
)

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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
		a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *NewOrderResponse) UnmarshalJSON(data []byte) error {
	type Alias NewOrderResponse
	aux := &struct {
		TransactionTime binanceusTime `json:"transactTime"`
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the server Time timestamp
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
		a.ServerTime = time.UnixMilli(int64(chil.Servertime))
	}
	return nil
}

// UnmarshalJSON deserialises the JSON infos, including the order time and update time timestamps
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

// UnmarshalJSON deserialises the JSON info, including the timestamp
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

// UnmarshalJSON deserialises the JSON info, including the ( TransactionTime )timestamp
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

// UnmarshalJSON deserialises the JSON info, including the (TransactioTime) timestamp
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

// UnmarshalJSON deserialises the JSON info, including the (Create Time) timestamp
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

// UnmarshalJSON deserialises the JSON info, including the (Create Time) timestamp
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

// UnmarshalJSON deserialises the JSON info, including the (EventTime , and TransactionTime) timestamp
func (a *WsListStatus) UnmarshalJSON(data []byte) error {
	type Alias WsListStatus
	aux := &struct {
		Data struct {
			EventTime       binanceusTime `json:"E"`
			TransactionTime binanceusTime `json:"T"`
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

// UnmarshalJSON deserialises the JSON info, including (EventTime , OpenTime, and TransactionTime) timestamp
func (a *TickerStream) UnmarshalJSON(data []byte) error {
	type Alias TickerStream
	aux := &struct {
		EventTime binanceusTime `json:"E"`
		OpenTime  binanceusTime `json:"O"`
		CloseTime binanceusTime `json:"C"`
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
		EventTime binanceusTime `json:"E"`
		Kline     struct {
			StartTime binanceusTime `json:"t"`
			CloseTime binanceusTime `json:"T"`
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

// UnmarshalJSON deserialises the JSON info, including the (Timestamp and EventTime) timestamp
func (a *TradeStream) UnmarshalJSON(data []byte) error {
	type Alias TradeStream
	aux := &struct {
		TimeStamp binanceusTime `json:"T"`
		EventTime binanceusTime `json:"E"`
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

// UnmarshalJSON deserialises the JSON info, including the (EventTime, OrderCreationTime, and TransactionTime)timestamp
func (a *wsOrderUpdate) UnmarshalJSON(data []byte) error {
	type Alias wsOrderUpdate
	aux := &struct {
		Data struct {
			EventTime         binanceusTime `json:"E"`
			OrderCreationTime binanceusTime `json:"O"`
			TransactionTime   binanceusTime `json:"T"`
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

// UnmarshalJSON deserialises the JSON info, including the (EventTime and ClearTime) timestamp
func (a *wsBalanceUpdate) UnmarshalJSON(data []byte) error {
	type Alias wsBalanceUpdate
	aux := &struct {
		Data struct {
			EventTime binanceusTime `json:"E"`
			ClearTime binanceusTime `json:"T"`
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

// UnmarshalJSON deserialises the JSON info, including the (EventTime and LastUpdated) timestamp
func (a *wsAccountPosition) UnmarshalJSON(data []byte) error {
	type Alias wsAccountPosition
	aux := &struct {
		Data struct {
			EventTime   binanceusTime `json:"E"`
			LastUpdated binanceusTime `json:"u"`
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

// UnmarshalJSON deserialises the JSON info, including the (Timestamp)timestamp
func (a *WebsocketDepthStream) UnmarshalJSON(data []byte) error {
	type Alias WebsocketDepthStream
	aux := &struct {
		Timestamp binanceusTime `json:"E"`
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

// binanceTime provides an internal conversion helper
type binanceusTime time.Time

func (t *binanceusTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	*t = binanceusTime(time.UnixMilli(timestamp))
	return nil
}

// Time returns a time.Time object
func (t binanceusTime) Time() time.Time {
	return time.Time(t)
}

// UnmarshalJSON deserialises createTime timestamp to built in time.
func (a *OCBSOrder) UnmarshalJSON(data []byte) error {
	type Alias OCBSOrder
	chil := &struct {
		*Alias
		CreateTime int64 `json:"createTime"`
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

// UnmarshalJSON deserialises createTime timestamp to built in time.
func (a *ServerTime) UnmarshalJSON(data []byte) error {
	type Alias ServerTime
	chil := &struct {
		*Alias
		Timestamp int64 `json:"serverTime"`
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

// UnmarshalJSON deserialises createTime timestamp to built in time.
func (a *SubAccountStatus) UnmarshalJSON(data []byte) error {
	type Alias SubAccountStatus
	chil := &struct {
		*Alias
		InsertTime int64 `json:"insertTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.InsertTime > 0 {
		a.InsertTime = time.UnixMilli(chil.InsertTime)
	}
	return nil
}

// UnmarshalJSON deserialises ValidTimestamp timestamp to built in time.Time instance.
func (a *Quote) UnmarshalJSON(data []byte) error {
	type Alias Quote
	chil := &struct {
		*Alias
		ValidTimestamp int64 `json:"validTimestamp"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.ValidTimestamp > 0 {
		a.ValidTimestamp = time.UnixMilli(chil.ValidTimestamp)
	}
	return nil
}

// UnmarshalJSON deserialises createTime timestamp to built in time.
func (a *SubAccountDepositItem) UnmarshalJSON(data []byte) error {
	type Alias SubAccountDepositItem
	chil := &struct {
		*Alias
		InsertTime int64 `json:"insertTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.InsertTime > 0 {
		a.InsertTime = time.UnixMilli(chil.InsertTime)
	}
	return nil
}

// UnmarshalJSON deserialises createTime timestamp to built in time.
func (a *ReferralWithdrawalItem) UnmarshalJSON(data []byte) error {
	type Alias ReferralWithdrawalItem
	chil := &struct {
		*Alias
		ReceiveDateTime int64 `json:"receiveDateTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.ReceiveDateTime > 0 {
		a.ReceiveDateTime = time.UnixMilli(chil.ReceiveDateTime)
	}
	return nil
}

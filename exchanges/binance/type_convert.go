package binance

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

// binanceTime provides an internal conversion helper
type binanceTime time.Time

func (t *binanceTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	*t = binanceTime(time.Unix(0, timestamp*int64(time.Millisecond)))
	return nil
}

// Time returns a time.Time object
func (t binanceTime) Time() time.Time {
	return time.Time(t)
}

// timeString gets the time as Binance timestamp
func timeString(t time.Time) string {
	return strconv.FormatInt(convert.UnixMillis(t), 10)
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *ExchangeInfo) UnmarshalJSON(data []byte) error {
	type Alias ExchangeInfo
	aux := &struct {
		Servertime binanceTime `json:"serverTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Servertime = aux.Servertime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *AggregatedTrade) UnmarshalJSON(data []byte) error {
	type Alias AggregatedTrade
	aux := &struct {
		TimeStamp binanceTime `json:"T"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.TimeStamp = aux.TimeStamp.Time()
	return nil
}

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
	// there can be an empty response, then `a` is set to nil
	if aux != nil {
		a.TransactionTime = aux.TransactionTime.Time()
	} else {
		a = nil
	}
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
func (a *PriceChangeStats) UnmarshalJSON(data []byte) error {
	type Alias PriceChangeStats
	aux := &struct {
		OpenTime  binanceTime `json:"openTime"`
		CloseTime binanceTime `json:"closeTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.OpenTime = aux.OpenTime.Time()
	a.CloseTime = aux.CloseTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *RecentTrade) UnmarshalJSON(data []byte) error {
	type Alias RecentTrade
	aux := &struct {
		Time binanceTime `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *HistoricalTrade) UnmarshalJSON(data []byte) error {
	type Alias HistoricalTrade
	aux := &struct {
		Time binanceTime `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *QueryOrderData) UnmarshalJSON(data []byte) error {
	type Alias QueryOrderData
	aux := &struct {
		Time       binanceTime `json:"time"`
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	a.UpdateTime = aux.UpdateTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesOrderData) UnmarshalJSON(data []byte) error {
	type Alias FuturesOrderData
	aux := &struct {
		Time       binanceTime `json:"time"`
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	a.UpdateTime = aux.UpdateTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *UFuturesOrderData) UnmarshalJSON(data []byte) error {
	type Alias UFuturesOrderData
	aux := &struct {
		Time       binanceTime `json:"time"`
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	a.UpdateTime = aux.UpdateTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesOrderGetData) UnmarshalJSON(data []byte) error {
	type Alias FuturesOrderGetData
	aux := &struct {
		Time       binanceTime `json:"time"`
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	a.UpdateTime = aux.UpdateTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *UOrderData) UnmarshalJSON(data []byte) error {
	type Alias UOrderData
	aux := &struct {
		Time       binanceTime `json:"time"`
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.Time = aux.Time.Time()
	a.UpdateTime = aux.UpdateTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		UpdateTime binanceTime `json:"updateTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.UpdateTime = aux.UpdateTime.Time()
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

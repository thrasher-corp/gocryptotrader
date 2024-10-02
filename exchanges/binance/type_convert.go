package binance

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// timeString gets the time as Binance timestamp
func timeString(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *ExchangeInfo) UnmarshalJSON(data []byte) error {
	type Alias ExchangeInfo
	aux := &struct {
		Servertime types.Time `json:"serverTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.ServerTime = aux.Servertime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *AggregatedTrade) UnmarshalJSON(data []byte) error {
	type Alias AggregatedTrade
	aux := &struct {
		TimeStamp types.Time `json:"T"`
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
		TransactionTime types.Time `json:"transactTime"`
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
func (a *TradeStream) UnmarshalJSON(data []byte) error {
	type Alias TradeStream
	aux := &struct {
		TimeStamp types.Time `json:"T"`
		EventTime types.Time `json:"E"`
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
		EventTime types.Time `json:"E"`
		Kline     struct {
			StartTime types.Time `json:"t"`
			CloseTime types.Time `json:"T"`
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
		EventTime types.Time `json:"E"`
		OpenTime  types.Time `json:"O"`
		CloseTime types.Time `json:"C"`
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
		OpenTime  types.Time `json:"openTime"`
		CloseTime types.Time `json:"closeTime"`
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
		Time types.Time `json:"time"`
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
		Time types.Time `json:"time"`
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
		Time       types.Time `json:"time"`
		UpdateTime types.Time `json:"updateTime"`
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
		Time       types.Time `json:"time"`
		UpdateTime types.Time `json:"updateTime"`
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
		Time       types.Time `json:"time"`
		UpdateTime types.Time `json:"updateTime"`
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
		Time       types.Time `json:"time"`
		UpdateTime types.Time `json:"updateTime"`
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
		Time       types.Time `json:"time"`
		UpdateTime types.Time `json:"updateTime"`
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
		UpdateTime types.Time `json:"updateTime"`
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
		Timestamp types.Time `json:"E"`
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
func (a *wsAccountPosition) UnmarshalJSON(data []byte) error {
	type Alias wsAccountPosition
	aux := &struct {
		Data struct {
			EventTime   types.Time `json:"E"`
			LastUpdated types.Time `json:"u"`
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
			EventTime types.Time `json:"E"`
			ClearTime types.Time `json:"T"`
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
			EventTime         types.Time `json:"E"`
			OrderCreationTime types.Time `json:"O"`
			TransactionTime   types.Time `json:"T"`
			WorkingTime       types.Time `json:"W"`
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
	a.Data.WorkingTime = aux.Data.WorkingTime.Time()
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *wsListStatus) UnmarshalJSON(data []byte) error {
	type Alias wsListStatus
	aux := &struct {
		Data struct {
			EventTime       types.Time `json:"E"`
			TransactionTime types.Time `json:"T"`
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
func (a *FuturesAccountInformationPosition) UnmarshalJSON(data []byte) error {
	type Alias FuturesAccountInformationPosition

	aux := &struct {
		UpdateTime types.Time `json:"updateTime"`
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
func (a *FuturesAccountInformation) UnmarshalJSON(data []byte) error {
	type Alias FuturesAccountInformation

	aux := &struct {
		UpdateTime types.Time `json:"updateTime"`
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

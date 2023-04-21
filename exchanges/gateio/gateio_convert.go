package gateio

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"
)

type gateioMilliSecTime int64

// UnmarshalJSON deserializes json, and timestamp information.
func (a *gateioMilliSecTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Int64, reflect.Int32:
		valueInteger, ok := value.(int64)
		if !ok {
			valueInteger = 0
		}
		*a = gateioMilliSecTime(valueInteger)
	case reflect.Float64, reflect.Float32:
		parsedValue, ok := value.(float64)
		if !ok {
			parsedValue = 0
		}
		*a = gateioMilliSecTime(int64(parsedValue))
	default:
		stringValue, ok := value.(string)
		if !ok {
			stringValue = "0"
		}
		parsedValue, err := strconv.ParseFloat(stringValue, 64)
		if err != nil {
			return err
		}
		*a = gateioMilliSecTime(int64(parsedValue))
	}
	return nil
}

// Time represents a time instance.
func (a *gateioMilliSecTime) Time() time.Time {
	return time.UnixMilli(int64(*a))
}

type gateioTime int64

// UnmarshalJSON deserializes json, and timestamp information.
func (a *gateioTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Int64, reflect.Int32:
		valueInteger, ok := value.(int64)
		if !ok {
			valueInteger = 0
		}
		*a = gateioTime(valueInteger)
	case reflect.Float64, reflect.Float32:
		parsedValue, ok := value.(float64)
		if !ok {
			parsedValue = 0
		}
		*a = gateioTime(int64(parsedValue))
	default:
		stringValue, ok := value.(string)
		if !ok {
			stringValue = "0"
		}
		parsedValue, err := strconv.ParseFloat(stringValue, 64)
		if err != nil {
			return err
		}
		*a = gateioTime(int64(parsedValue))
	}
	return nil
}

// Time represents a time instance.
func (a *gateioTime) Time() time.Time {
	return time.Unix(int64(*a), 0)
}

// UnmarshalJSON deserializes json data into CurrencyPairDetail
func (a *CurrencyPairDetail) UnmarshalJSON(data []byte) error {
	type Alias CurrencyPairDetail
	chil := struct {
		*Alias
		Fee            string `json:"fee"`
		MinBaseAmount  string `json:"min_base_amount"`
		MinQuoteAmount string `json:"min_quote_amount"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, &chil)
	if err != nil {
		return err
	}
	if chil.Fee != "" {
		a.TradingFee, err = strconv.ParseFloat(chil.Fee, 64)
		if err != nil {
			return err
		}
	}
	if chil.MinBaseAmount != "" {
		a.MinBaseAmount, err = strconv.ParseFloat(chil.MinBaseAmount, 64)
		if err != nil {
			return err
		}
	}
	if chil.MinQuoteAmount != "" {
		a.MinQuoteAmount, err = strconv.ParseFloat(chil.MinQuoteAmount, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON deserializes json, and timestamp information
func (a *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	child := &struct {
		*Alias
		BaseVolume  string `json:"base_volume"`
		QuoteVolume string `json:"quote_volume"`
		High24H     string `json:"high_24h"`
		Low24H      string `json:"low_24h"`
		Last        string `json:"last"`

		LowestAsk       string `json:"lowest_ask"`
		HighestBid      string `json:"highest_bid"`
		EtfLeverage     string `json:"etf_leverage"`
		EtfPreTimestamp int64  `json:"etf_pre_timestamp"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, child)
	if err != nil {
		return err
	}
	if child.BaseVolume != "" {
		if a.BaseVolume, err = strconv.ParseFloat(child.BaseVolume, 64); err != nil {
			return err
		}
	}
	if child.QuoteVolume != "" {
		if a.QuoteVolume, err = strconv.ParseFloat(child.QuoteVolume, 64); err != nil {
			return err
		}
	}
	if child.High24H != "" {
		if a.High24H, err = strconv.ParseFloat(child.High24H, 64); err != nil {
			return err
		}
	}
	if child.Low24H != "" {
		if a.Low24H, err = strconv.ParseFloat(child.Low24H, 64); err != nil {
			return err
		}
	}
	if child.LowestAsk != "" {
		if a.LowestAsk, err = strconv.ParseFloat(child.LowestAsk, 64); err != nil {
			return err
		}
	}
	if child.HighestBid != "" {
		if a.HighestBid, err = strconv.ParseFloat(child.HighestBid, 64); err != nil {
			return err
		}
	}
	if child.EtfLeverage != "" {
		if a.EtfLeverage, err = strconv.ParseFloat(child.EtfLeverage, 64); err != nil {
			return err
		}
	}
	if child.Last != "" {
		if a.Last, err = strconv.ParseFloat(child.Last, 64); err != nil {
			return err
		}
	}
	a.EtfPreTimestamp = time.Unix(child.EtfPreTimestamp, 0)
	return nil
}

// UnmarshalJSON to deserialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *OptionsTicker) UnmarshalJSON(data []byte) error {
	type Alias OptionsTicker
	chil := &struct {
		*Alias
		LastPrice             string `json:"last_price"`
		MarkPrice             string `json:"mark_price"`
		IndexPrice            string `json:"index_price"`
		MarkImpliedVolatility string `json:"mark_iv"`
		BidImpliedVolatility  string `json:"bid_iv"`
		AskImpliedVolatility  string `json:"ask_iv"`
		Leverage              string `json:"leverage"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.LastPrice != "" {
		a.LastPrice, err = strconv.ParseFloat(chil.LastPrice, 64)
		if err != nil {
			return err
		}
	}
	if chil.MarkPrice != "" {
		a.MarkPrice, err = strconv.ParseFloat(chil.MarkPrice, 64)
		if err != nil {
			return err
		}
	}
	if chil.IndexPrice != "" {
		a.IndexPrice, err = strconv.ParseFloat(chil.IndexPrice, 64)
		if err != nil {
			return err
		}
	}
	if chil.MarkImpliedVolatility != "" {
		a.MarkImpliedVolatility, err = strconv.ParseFloat(chil.MarkImpliedVolatility, 64)
		if err != nil {
			return err
		}
	}
	if chil.BidImpliedVolatility != "" {
		a.BidImpliedVolatility, err = strconv.ParseFloat(chil.BidImpliedVolatility, 64)
		if err != nil {
			return err
		}
	}
	if chil.AskImpliedVolatility != "" {
		a.AskImpliedVolatility, err = strconv.ParseFloat(chil.AskImpliedVolatility, 64)
		if err != nil {
			return err
		}
	}
	if chil.Leverage != "" {
		a.Leverage, err = strconv.ParseFloat(chil.Leverage, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON to deserialize timestamp information and create OrderbookItem instance from the list of asks and bids data.
func (a *Orderbook) UnmarshalJSON(data []byte) error {
	type Alias Orderbook
	type askorbid struct {
		Price string  `json:"p"`
		Size  float64 `json:"s"`
	}
	chil := &struct {
		*Alias
		Current float64    `json:"current"`
		Update  float64    `json:"update"`
		Asks    []askorbid `json:"asks"`
		Bids    []askorbid `json:"bids"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, &chil)
	if err != nil {
		return err
	}
	a.Current = time.UnixMilli(int64(chil.Current * 1000))
	a.Update = time.UnixMilli(int64(chil.Update * 1000))
	a.Asks = make([]OrderbookItem, len(chil.Asks))
	a.Bids = make([]OrderbookItem, len(chil.Bids))
	for x := range chil.Asks {
		a.Asks[x] = OrderbookItem{
			Amount: chil.Asks[x].Size,
		}
		a.Asks[x].Price, err = strconv.ParseFloat(chil.Asks[x].Price, 64)
		if err != nil {
			return err
		}
	}
	for x := range chil.Bids {
		a.Bids[x] = OrderbookItem{
			Amount: chil.Bids[x].Size,
		}
		a.Bids[x].Price, err = strconv.ParseFloat(chil.Bids[x].Price, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *FuturesFundingRate) UnmarshalJSON(data []byte) error {
	type Alias FuturesFundingRate
	chil := &struct {
		*Alias
		Rate string `json:"r"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Rate != "" {
		if a.Rate, err = strconv.ParseFloat(chil.Rate, 64); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *OptionSettlement) UnmarshalJSON(data []byte) error {
	type Alias OptionSettlement
	chil := &struct {
		*Alias
		Time   int64  `json:"time"`
		Profit string `json:"profit"`
		Fee    string `json:"fee"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Profit != "" {
		if a.Profit, err = strconv.ParseFloat(chil.Profit, 64); err != nil {
			return err
		}
	}
	if chil.Fee != "" {
		if a.Fee, err = strconv.ParseFloat(chil.Fee, 64); err != nil {
			return err
		}
	}
	a.Time = time.Unix(chil.Time, 0)
	return nil
}

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *WsUserPersonalTrade) UnmarshalJSON(data []byte) error {
	type Alias WsUserPersonalTrade
	chil := &struct {
		*Alias
		CreateTimeMicroS float64 `json:"create_time_ms,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTimeMicroS = time.UnixMicro(int64(chil.CreateTimeMicroS * 1000))
	return nil
}

// UnmarshalJSON deserializes json, and timestamp information.
func (a *SpotOrder) UnmarshalJSON(data []byte) error {
	type Alias SpotOrder
	chil := &struct {
		*Alias
		Left string `json:"left"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Left != "" {
		if a.Left, err = strconv.ParseFloat(chil.Left, 64); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON deserializes json, and timestamp information.
func (a *WsSpotOrder) UnmarshalJSON(data []byte) error {
	type Alias WsSpotOrder
	chil := &struct {
		*Alias
		Left string `json:"left"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	if chil.Left != "" {
		if a.Left, err = strconv.ParseFloat(chil.Left, 64); err != nil {
			return err
		}
	}
	return nil
}

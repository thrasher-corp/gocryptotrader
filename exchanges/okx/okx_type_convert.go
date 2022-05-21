package okx

import (
	"encoding/json"
	"regexp"
	"strconv"
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

// NumbersOnlyRegexp for checking the value is numberics only
var NumbersOnlyRegexp = regexp.MustCompile("^[0-9]*$")

// UnmarshalJSON
func (a *Instrument) UnmarshalJSON(data []byte) error {
	type Alias Instrument
	chil := &struct {
		*Alias
		ListTime string `json:"listTime"`
		ExpTime  string `json:"expTime"`
	}{Alias: (*Alias)(a)}

	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if NumbersOnlyRegexp.MatchString(chil.ListTime) {
		if val, er := strconv.Atoi(chil.ListTime); er == nil {
			a.ListTime = time.UnixMilli(int64(val))
		}
	}
	if NumbersOnlyRegexp.MatchString(chil.ExpTime) {
		if val, er := strconv.Atoi(chil.ExpTime); er == nil {
			a.ExpTime = time.UnixMilli(int64(val))
		}
	}
	return nil
}

// UnmarshalJSON unmarshals the json obeject to the DeliveryHistoryResponse
func (a *DeliveryHistory) UnmarshalJSON(data []byte) error {
	type Alias DeliveryHistory
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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
		Timestamp int64 `json:"ts,string"`
	}{Alias: (*Alias)(a)}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

func (a *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type Alias FundingRateResponse
	chil := &struct {
		*Alias
		FundingTime     int64  `json:"fundingTime,string"`
		NextFundingTime string `json:"nextFundingTime"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return er
	}
	if chil.FundingTime > 0 {
		a.FundingTime = time.UnixMilli(chil.FundingTime)
	}
	if NumbersOnlyRegexp.MatchString(chil.NextFundingTime) {
		if val, er := strconv.Atoi(chil.NextFundingTime); er == nil {
			a.NextFundingTime = time.UnixMilli(int64(val))
		}
	}
	return nil
}

func (a *LimitPriceResponse) UnmarshalJSON(data []byte) error {
	type Alias LimitPriceResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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

func (a *OptionMarketDataResponse) UnmarshalJSON(data []byte) error {
	type Alias OptionMarketDataResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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

func (a *DeliveryEstimatedPrice) UnmarshalJSON(data []byte) error {
	type Alias DeliveryEstimatedPrice
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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

// UnmarshalJSON
func (a *ServerTime) UnmarshalJSON(data []byte) error {
	type Alias ServerTime
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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

func (a *LiquidationOrderDetailItem) UnmarshalJSON(data []byte) error {
	type Alias LiquidationOrderDetailItem
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
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

// UnmarshalJSON unmarshals the timestamp for mark price data
func (a *MarkPrice) UnmarshalJSON(data []byte) error {
	type Alias MarkPrice
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	if er := json.Unmarshal(data, chil); er != nil {
		return nil
	}
	if chil.Timestamp > 0 {
		a.Timestamp = time.UnixMilli(chil.Timestamp)
	}
	return nil
}

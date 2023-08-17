package okx

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type okxNumericalValue float64

// UnmarshalJSON is custom type json unmarshaller for okxNumericalValue
func (a *okxNumericalValue) UnmarshalJSON(data []byte) error {
	var num interface{}
	err := json.Unmarshal(data, &num)
	if err != nil {
		return err
	}

	switch d := num.(type) {
	case float64:
		*a = okxNumericalValue(d)
	case string:
		if d == "" {
			return nil
		}
		convNum, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return err
		}
		*a = okxNumericalValue(convNum)
	}
	return nil
}

// Float64 returns a float64 value for okxNumericalValue
func (a *okxNumericalValue) Float64() float64 { return float64(*a) }

type okxUnixMilliTime int64

type okxAssetType struct {
	asset.Item
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *okxAssetType) UnmarshalJSON(data []byte) error {
	var t string
	err := json.Unmarshal(data, &t)
	if err != nil {
		return err
	}

	a.Item = GetAssetTypeFromInstrumentType(strings.ToUpper(t))
	return nil
}

// UnmarshalJSON deserializes byte data to okxunixMilliTime instance.
func (a *okxUnixMilliTime) UnmarshalJSON(data []byte) error {
	var num string
	err := json.Unmarshal(data, &num)
	if err != nil {
		return err
	}
	if num == "" {
		return nil
	}
	value, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return err
	}
	*a = okxUnixMilliTime(value)
	return nil
}

// Time returns the time instance from unix value of integer.
func (a *okxUnixMilliTime) Time() time.Time {
	return time.UnixMilli(int64(*a))
}

type okxTime struct {
	time.Time
}

// UnmarshalJSON deserializes byte data to okxTime instance.
func (t *okxTime) UnmarshalJSON(data []byte) error {
	var num string
	err := json.Unmarshal(data, &num)
	if err != nil {
		return err
	}
	if num == "" {
		return nil
	}
	value, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(value)
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *Instrument) UnmarshalJSON(data []byte) error {
	type Alias Instrument
	chil := &struct {
		*Alias
		ListTime                        okxTime           `json:"listTime"`
		ExpTime                         okxTime           `json:"expTime"`
		InstrumentType                  okxAssetType      `json:"instType"`
		MaxLeverage                     okxNumericalValue `json:"lever"`
		TickSize                        okxNumericalValue `json:"tickSz"`
		LotSize                         okxNumericalValue `json:"lotSz"`
		MinimumOrderSize                okxNumericalValue `json:"minSz"`
		MaxQuantityOfSpotLimitOrder     okxNumericalValue `json:"maxLmtSz"`
		MaxQuantityOfMarketLimitOrder   okxNumericalValue `json:"maxMktSz"`
		MaxQuantityOfSpotTwapLimitOrder okxNumericalValue `json:"maxTwapSz"`
		MaxSpotIcebergSize              okxNumericalValue `json:"maxIcebergSz"`
		MaxTriggerSize                  okxNumericalValue `json:"maxTriggerSz"`
		MaxStopSize                     okxNumericalValue `json:"maxStopSz"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}

	a.ListTime = chil.ListTime.Time
	a.ExpTime = chil.ExpTime.Time
	a.InstrumentType = chil.InstrumentType.Item
	a.MaxLeverage = chil.MaxLeverage.Float64()
	a.TickSize = chil.TickSize.Float64()
	a.LotSize = chil.LotSize.Float64()
	a.MinimumOrderSize = chil.MinimumOrderSize.Float64()
	a.MaxQuantityOfSpotLimitOrder = chil.MaxQuantityOfSpotLimitOrder.Float64()
	a.MaxQuantityOfMarketLimitOrder = chil.MaxQuantityOfMarketLimitOrder.Float64()
	a.MaxQuantityOfSpotTwapLimitOrder = chil.MaxQuantityOfSpotTwapLimitOrder.Float64()
	a.MaxSpotIcebergSize = chil.MaxSpotIcebergSize.Float64()
	a.MaxTriggerSize = chil.MaxTriggerSize.Float64()
	a.MaxStopSize = chil.MaxStopSize.Float64()

	return nil
}

// UnmarshalJSON decoder for OpenInterestResponse instance.
func (a *OpenInterest) UnmarshalJSON(data []byte) error {
	type Alias OpenInterest
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{Alias: (*Alias)(a)}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	a.InstrumentType = GetAssetTypeFromInstrumentType(chil.InstrumentType)
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *LimitPriceResponse) UnmarshalJSON(data []byte) error {
	type Alias LimitPriceResponse
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *OrderDetail) UnmarshalJSON(data []byte) error {
	type Alias OrderDetail
	chil := &struct {
		*Alias
		Side         string `json:"side"`
		UpdateTime   int64  `json:"uTime,string"`
		CreationTime int64  `json:"cTime,string"`
		FillTime     string `json:"fillTime"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	var err error
	a.UpdateTime = time.UnixMilli(chil.UpdateTime)
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	a.Side, err = order.StringToOrderSide(chil.Side)
	if chil.FillTime == "" {
		a.FillTime = time.Time{}
	} else {
		var value int64
		value, err = strconv.ParseInt(chil.FillTime, 10, 64)
		if err != nil {
			return err
		}
		a.FillTime = time.UnixMilli(value)
	}
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *PendingOrderItem) UnmarshalJSON(data []byte) error {
	type Alias PendingOrderItem
	chil := &struct {
		*Alias
		Side         string `json:"side"`
		UpdateTime   string `json:"uTime"`
		CreationTime string `json:"cTime"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	uTime, err := strconv.ParseInt(chil.UpdateTime, 10, 64)
	if err != nil {
		return err
	}
	cTime, err := strconv.ParseInt(chil.CreationTime, 10, 64)
	if err != nil {
		return err
	}
	a.Side, err = order.StringToOrderSide(chil.Side)
	if err != nil {
		return err
	}
	a.CreationTime = time.UnixMilli(cTime)
	a.UpdateTime = time.UnixMilli(uTime)
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *RfqTradeResponse) UnmarshalJSON(data []byte) error {
	type Alias RfqTradeResponse
	chil := &struct {
		*Alias
		CreationTime int64 `json:"cTime,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreationTime = time.UnixMilli(chil.CreationTime)
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *BlockTicker) UnmarshalJSON(data []byte) error {
	type Alias BlockTicker
	chil := &struct {
		*Alias
		Timestamp int64 `json:"ts,string"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *BlockTrade) UnmarshalJSON(data []byte) error {
	type Alias BlockTrade
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	switch {
	case strings.EqualFold(chil.Side, "buy"):
		a.Side = order.Buy
	case strings.EqualFold(chil.Side, "sell"):
		a.Side = order.Sell
	default:
		a.Side = order.UnknownSide
	}
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *UnitConvertResponse) UnmarshalJSON(data []byte) error {
	type Alias UnitConvertResponse
	chil := &struct {
		*Alias
		ConvertType int `json:"type,string"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	switch chil.ConvertType {
	case 1:
		a.ConvertType = 1
	case 2:
		a.ConvertType = 2
	}
	return nil
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *QuoteLeg) UnmarshalJSON(data []byte) error {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	chil.Side = strings.ToLower(chil.Side)
	if chil.Side == "buy" {
		a.Side = order.Buy
	} else {
		a.Side = order.Sell
	}
	return nil
}

// MarshalJSON serialized QuoteLeg instance into bytes
func (a *QuoteLeg) MarshalJSON() ([]byte, error) {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if a.Side == order.Buy {
		chil.Side = "buy"
	} else {
		chil.Side = "sell"
	}
	return json.Marshal(chil)
}

// MarshalJSON serialized CreateQuoteParams instance into bytes
func (a *CreateQuoteParams) MarshalJSON() ([]byte, error) {
	type Alias CreateQuoteParams
	chil := &struct {
		*Alias
		QuoteSide string `json:"quoteSide"`
	}{
		Alias: (*Alias)(a),
	}
	if a.QuoteSide == order.Buy {
		chil.QuoteSide = "buy"
	} else {
		chil.QuoteSide = "sell"
	}
	return json.Marshal(chil)
}

// MarshalJSON serializes the WebsocketLoginData object
func (a *WebsocketLoginData) MarshalJSON() ([]byte, error) {
	type Alias WebsocketLoginData
	return json.Marshal(struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: a.Timestamp.UTC().Unix(),
		Alias:     (*Alias)(a),
	})
}

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *WebsocketLoginData) UnmarshalJSON(data []byte) error {
	type Alias WebsocketLoginData
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

// UnmarshalJSON deserializes JSON, and timestamp information.
func (a *CurrencyOneClickRepay) UnmarshalJSON(data []byte) error {
	type Alias CurrencyOneClickRepay
	chil := &struct {
		*Alias
		UpdateTime   int64  `json:"uTime,string"`
		FillToSize   string `json:"fillToSz"`
		FillFromSize string `json:"fillFromSz"`
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	a.UpdateTime = time.Unix(chil.UpdateTime, 0)
	return nil
}

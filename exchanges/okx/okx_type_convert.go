package okx

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type okxUnixMilliTime int64

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

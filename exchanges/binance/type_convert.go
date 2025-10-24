package binance

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// timeString gets the time as Binance timestamp
func timeString(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

// UnmarshalJSON deserializes the data to unmarshal into WsTickerPriceChange or []WsTickerPriceChange
func (a *PriceChanges) UnmarshalJSON(data []byte) error {
	var resp []PriceChangeStats
	err := json.Unmarshal(data, &resp)
	if err != nil {
		var singleResp PriceChangeStats
		err := json.Unmarshal(data, &singleResp)
		if err != nil {
			return err
		}
		*a = []PriceChangeStats{singleResp}
	} else {
		*a = resp
	}
	return nil
}

// UnmarshalJSON deserializes the data to unmarshal into SymbolTickerItem or []SymbolTickerItem
func (a *SymbolTickers) UnmarshalJSON(data []byte) error {
	var resp []SymbolTickerItem
	err := json.Unmarshal(data, &resp)
	if err != nil {
		var singleResp SymbolTickerItem
		err := json.Unmarshal(data, &singleResp)
		if err != nil {
			return err
		}
		*a = []SymbolTickerItem{singleResp}
	} else {
		*a = resp
	}
	return nil
}

// UnmarshalJSON deserializes the data to unmarshal into WsOrderbookTicker or []WsOrderbookTicker
func (a *WsOrderbookTickers) UnmarshalJSON(data []byte) error {
	var resp []WsOrderbookTicker
	err := json.Unmarshal(data, &resp)
	if err != nil {
		var singleResp WsOrderbookTicker
		err := json.Unmarshal(data, &singleResp)
		if err != nil {
			return err
		}
		*a = []WsOrderbookTicker{singleResp}
	} else {
		*a = resp
	}
	return nil
}

// UnmarshalJSON decerializes byte data into PriceChanceWrapper instance.
func (a *PriceChangesWrapper) UnmarshalJSON(data []byte) error {
	var singlePriceChange *PriceChangeStats
	err := json.Unmarshal(data, &singlePriceChange)
	if err != nil {
		var resp []PriceChangeStats
		err = json.Unmarshal(data, a)
		if err != nil {
			return err
		}
		*a = resp
		return nil
	}
	*a = []PriceChangeStats{*singlePriceChange}
	return nil
}

// UnmarshalJSON deserializes incoming object or slice into WsOptionIncomingResps([]WsOptionIncomingResp) instance.
func (a *WsOptionIncomingResps) UnmarshalJSON(data []byte) error {
	var resp []WsOptionIncomingResp
	isSlice := true
	err := json.Unmarshal(data, &resp)
	if err != nil {
		isSlice = false
		var newResp WsOptionIncomingResp
		err = json.Unmarshal(data, &newResp)
		if err != nil {
			return err
		}
		resp = append(resp, newResp)
	}
	a.Instances = resp
	a.IsSlice = isSlice
	return nil
}

// UnmarshalJSON unmarshals a []byte data in an object or array form to AssetIndexResponse([]AssetIndex) instance.
func (a *AssetIndexResponse) UnmarshalJSON(data []byte) error {
	var resp []AssetIndex
	err := json.Unmarshal(data, &resp)
	if err != nil {
		resp = make([]AssetIndex, 1)
		err := json.Unmarshal(data, &resp[0])
		if err != nil {
			return err
		}
	}
	*a = resp
	return nil
}

// UnmarshalJSON unmarshals a []byte data in an object or array form to AccountBalanceResponse([]AccountBalance) instance.
func (a *AccountBalanceResponse) UnmarshalJSON(data []byte) error {
	var resp []AccountBalance
	err := json.Unmarshal(data, &resp)
	if err != nil {
		resp = make([]AccountBalance, 1)
		err := json.Unmarshal(data, &resp[0])
		if err != nil {
			return err
		}
	}
	*a = resp
	return nil
}

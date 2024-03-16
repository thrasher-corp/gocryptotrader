package binance

import (
	"encoding/json"
	"strconv"
	"time"
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

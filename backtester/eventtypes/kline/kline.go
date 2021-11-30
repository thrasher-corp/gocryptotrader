package kline

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// GetClosePrice returns the closing price of a kline
func (k *Kline) GetClosePrice() decimal.Decimal {
	return k.Close
}

// GetHighPrice returns the high price of a kline
func (k *Kline) GetHighPrice() decimal.Decimal {
	return k.High
}

// GetLowPrice returns the low price of a kline
func (k *Kline) GetLowPrice() decimal.Decimal {
	return k.Low
}

// GetOpenPrice returns the open price of a kline
func (k *Kline) GetOpenPrice() decimal.Decimal {
	return k.Open
}

func (k *Kline) GetFuturesDataEventHandler() (common.FuturesDataEventHandler, error) {
	if !k.AssetType.IsFutures() {
		return nil, fmt.Errorf("not futures")
	}
	return k.FuturesData, nil
}

func (f *FuturesData) GetMarkPrice() decimal.Decimal {
	return f.MarkPrice
}

func (f *FuturesData) GetPreviousMarkPrice() decimal.Decimal {
	return f.PrevMarkPrice
}

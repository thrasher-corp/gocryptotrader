package kline

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

// GetUnderlyingPair returns the open price of a kline
func (k *Kline) GetUnderlyingPair() (currency.Pair, error) {
	return k.UnderlyingPair, nil
}

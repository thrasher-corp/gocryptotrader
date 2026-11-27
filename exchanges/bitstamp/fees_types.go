package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TradingFeeResponse holds the trading fees for a market
type TradingFeeResponse struct {
	Market string         `json:"market"`
	Fees   MakerTakerFees `json:"fees"`
}

// MakerTakerFees holds maker and taker fee percentages
type MakerTakerFees struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// WithdrawalFeeResponse holds the withdrawal fee for a currency network
type WithdrawalFeeResponse struct {
	Currency currency.Code `json:"currency"`
	Fee      types.Number  `json:"fee"`
	Network  string        `json:"network"`
}

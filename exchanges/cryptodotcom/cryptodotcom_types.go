package cryptodotcom

import (
	"errors"
	"time"
)

var (
	errSymbolIsRequired = errors.New("symbol is required")
)

// MarketSymbol represents a market symbol detail.
type MarketSymbol struct {
	Symbol          string `json:"symbol"`           // Transaction pairs
	CountCoin       string `json:"count_coin"`       // Money of Account
	AmountPrecision int    `json:"amount_precision"` // Quantitative precision digits (0 is a single digit)
	BaseCurrency    string `json:"base_coin"`        // Base currency
	PricePrecision  int    `json:"price_precision"`  // Price Precision Number (0 is a single digit)
}

// TickerDetail represents a ticker detail
type TickerDetail struct {
	Date   cryptoDotComMilliSec `json:"date"`
	Ticker []MarketTickerItem   `json:"ticker"`
}

// MarketTickerItem represents a market ticker item.
type MarketTickerItem struct {
	Symbol string               `json:"symbol"`
	High   string               `json:"high"`
	Vol    string               `json:"vol"`
	Last   float64              `json:"last"`
	Low    string               `json:"low"`
	Buy    float64              `json:"buy,omitempty"`
	Sell   float64              `json:"sell,omitempty"`
	Change string               `json:"change,omitempty"`
	Rose   string               `json:"rose"`
	Time   cryptoDotComMilliSec `json:"time"`
}

// KlineItem represents a kline data detail.
type KlineItem struct {
	Timestamp    time.Time
	OpeningPrice float64
	HighestPrice float64
	MinimumPrice float64
	ClosingPrice float64
	Volume       float64
}

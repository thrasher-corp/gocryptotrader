package bybit

import "time"

var (
	validFuturesIntervals = []string{
		"1m", "3m", "5m", "15m", "30m",
		"1h", "2h", "4h", "6h", "8h",
		"12h", "1d", "3d", "1w", "1M",
	}
)

// OrderbookData stores ob data for cmargined futures
type OrderbookData struct {
	Symbol int64  `json:"symbol"`
	Price  string `json:"price"`
	Size   int64  `json:"size"`
	Side   string `json:"side"`
}

// SymbolPriceTicker stores ticker price stats
type SymbolPriceTicker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
	Time   int64   `json:"time"`
}

// FuturesCandleStick holds kline data
type FuturesCandleStick struct {
	OpenTime                time.Time
	Open                    float64
	High                    float64
	Low                     float64
	Close                   float64
	Volume                  float64
	CloseTime               time.Time
	BaseAssetVolume         float64
	NumberOfTrades          int64
	TakerBuyVolume          float64
	TakerBuyBaseAssetVolume float64
}

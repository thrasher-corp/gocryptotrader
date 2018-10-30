package exchange

import (
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

// WebsocketResponse defines generalised data from the websocket connection
type WebsocketResponse struct {
	Type int
	Raw  []byte
}

// WebsocketOrderbookUpdate defines a websocket event in which the orderbook
// has been updated in the orderbook package
type WebsocketOrderbookUpdate struct {
	Pair     pair.CurrencyPair
	Asset    string
	Exchange string
}

// TradeData defines trade data
type TradeData struct {
	Timestamp    time.Time
	CurrencyPair pair.CurrencyPair
	AssetType    string
	Exchange     string
	EventType    string
	EventTime    int64
	Price        float64
	Amount       float64
	Side         string
}

// TickerData defines ticker feed
type TickerData struct {
	Timestamp  time.Time
	Pair       pair.CurrencyPair
	AssetType  string
	Exchange   string
	ClosePrice float64
	Quantity   float64
	OpenPrice  float64
	HighPrice  float64
	LowPrice   float64
}

// KlineData defines kline feed
type KlineData struct {
	Timestamp  time.Time
	Pair       pair.CurrencyPair
	AssetType  string
	Exchange   string
	StartTime  time.Time
	CloseTime  time.Time
	Interval   string
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

// WebsocketPositionUpdated reflects a change in orders/contracts on an exchange
type WebsocketPositionUpdated struct {
	Timestamp time.Time
	Pair      pair.CurrencyPair
	AssetType string
	Exchange  string
}

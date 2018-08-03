package gateio

import "time"

// SpotNewOrderRequestParamsType order type (buy or sell)
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestParamsTypeBuy buy order
	SpotNewOrderRequestParamsTypeBuy = SpotNewOrderRequestParamsType("buy")

	// SpotNewOrderRequestParamsTypeSell sell order
	SpotNewOrderRequestParamsTypeSell = SpotNewOrderRequestParamsType("sell")
)

// TimeInterval Interval represents interval enum.
type TimeInterval int

// TimeInterval vars
var (
	TimeIntervalMinute         = TimeInterval(60)
	TimeIntervalThreeMinutes   = TimeInterval(60 * 3)
	TimeIntervalFiveMinutes    = TimeInterval(60 * 5)
	TimeIntervalFifteenMinutes = TimeInterval(60 * 15)
	TimeIntervalThirtyMinutes  = TimeInterval(60 * 30)
	TimeIntervalHour           = TimeInterval(60 * 60)
	TimeIntervalTwoHours       = TimeInterval(2 * 60 * 60)
	TimeIntervalFourHours      = TimeInterval(4 * 60 * 60)
	TimeIntervalSixHours       = TimeInterval(6 * 60 * 60)
	TimeIntervalDay            = TimeInterval(60 * 60 * 24)
)

// MarketInfoResponse holds the market info data
type MarketInfoResponse struct {
	Result string                    `json:"result"`
	Pairs  []MarketInfoPairsResponse `json:"pairs"`
}

// MarketInfoPairsResponse holds the market info response data
type MarketInfoPairsResponse struct {
	Symbol string
	// DecimalPlaces symbol price accuracy
	DecimalPlaces float64
	// MinAmount minimum order amount
	MinAmount float64
	// Fee transaction fee
	Fee float64
}

// BalancesResponse holds the user balances
type BalancesResponse struct {
	Result    string            `json:"result"`
	Available map[string]string `json:"available"`
	Locked    map[string]string `json:"locked"`
}

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol   string // Required field; example LTCBTC,BTCUSDT
	HourSize int    // How many hours of data
	GroupSec TimeInterval
}

// KLineResponse holds the kline response data
type KLineResponse struct {
	ID        float64
	KlineTime time.Time
	Open      float64
	Time      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Amount    float64 `db:"amount"`
}

// TickerResponse  holds the ticker response data
type TickerResponse struct {
	Result        string  `json:"result"`
	Volume        float64 `json:"baseVolume,string"`    // Trading volume
	High          float64 `json:"high24hr,string"`      // 24 hour high price
	Open          float64 `json:"highestBid,string"`    // Openening price
	Last          float64 `json:"last,string"`          // Last price
	Low           float64 `json:"low24hr,string"`       // 24 hour low price
	Close         float64 `json:"lowestAsk,string"`     // Closing price
	PercentChange float64 `json:"percentChange,string"` // Percentage change
	QuoteVolume   float64 `json:"quoteVolume,string"`   // Quote currency volume
}

// OrderbookResponse stores the orderbook data
type OrderbookResponse struct {
	Result  string `json:"result"`
	Elapsed string `json:"elapsed"`
	Asks    [][]string
	Bids    [][]string
}

// OrderbookItem stores an orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Result  string
	Elapsed string
	Bids    []OrderbookItem
	Asks    []OrderbookItem
}

// SpotNewOrderRequestParams Order params
type SpotNewOrderRequestParams struct {
	Amount float64                       `json:"amount"` // Order quantity
	Price  float64                       `json:"price"`  // Order price
	Symbol string                        `json:"symbol"` // Trading pair; btc_usdt, eth_btc......
	Type   SpotNewOrderRequestParamsType `json:"type"`   // Order type (buy or sell),
}

// SpotNewOrderResponse Order response
type SpotNewOrderResponse struct {
	OrderNumber  int64   `json:"orderNumber"`         // OrderID number
	Price        float64 `json:"rate,string"`         // Order price
	LeftAmount   float64 `json:"leftAmount,string"`   // The remaining amount to fill
	FilledAmount float64 `json:"filledAmount,string"` // The filled amount
	Filledrate   float64 `json:"filledRate,string"`   // FilledPrice
}

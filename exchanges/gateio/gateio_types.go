package gateio

import (
	"time"

	"github.com/thrasher-/gocryptotrader/decimal"
)

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
	DecimalPlaces decimal.Decimal
	// MinAmount minimum order amount
	MinAmount decimal.Decimal
	// Fee transaction fee
	Fee decimal.Decimal
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
	ID        decimal.Decimal
	KlineTime time.Time
	Open      float64
	Time      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	Amount    decimal.Decimal `db:"amount"`
}

// TickerResponse  holds the ticker response data
type TickerResponse struct {
	Result        string          `json:"result"`
	Volume        decimal.Decimal `json:"baseVolume,string"`    // Trading volume
	High          decimal.Decimal `json:"high24hr,string"`      // 24 hour high price
	Open          decimal.Decimal `json:"highestBid,string"`    // Openening price
	Last          decimal.Decimal `json:"last,string"`          // Last price
	Low           decimal.Decimal `json:"low24hr,string"`       // 24 hour low price
	Close         decimal.Decimal `json:"lowestAsk,string"`     // Closing price
	PercentChange decimal.Decimal `json:"percentChange,string"` // Percentage change
	QuoteVolume   decimal.Decimal `json:"quoteVolume,string"`   // Quote currency volume
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
	Price  decimal.Decimal
	Amount decimal.Decimal
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
	Amount decimal.Decimal               `json:"amount"` // Order quantity
	Price  decimal.Decimal               `json:"price"`  // Order price
	Symbol string                        `json:"symbol"` // Trading pair; btc_usdt, eth_btc......
	Type   SpotNewOrderRequestParamsType `json:"type"`   // Order type (buy or sell),
}

// SpotNewOrderResponse Order response
type SpotNewOrderResponse struct {
	OrderNumber  int64           `json:"orderNumber"`         // OrderID number
	Price        decimal.Decimal `json:"rate,string"`         // Order price
	LeftAmount   decimal.Decimal `json:"leftAmount,string"`   // The remaining amount to fill
	FilledAmount decimal.Decimal `json:"filledAmount,string"` // The filled amount
	Filledrate   decimal.Decimal `json:"filledRate,string"`   // FilledPrice
}

package bybit

import (
	"errors"
	"time"
)

var (
	errTypeAssert = errors.New("type assertion failed")
	errStrParsing = errors.New("parsing string failed")
)

// PairData stores pair data
type PairData struct {
	Name              string  `json:"name"`
	Alias             string  `json:"alias"`
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	BasePrecision     float64 `json:"basePrecision,string"`
	QuotePrecision    float64 `json:"quotePrecision,string"`
	MinTradeQuantity  float64 `json:"minTradeQuantity,string"`
	MinTradeAmount    float64 `json:"minTradeAmount,string"`
	MinPricePrecision float64 `json:"minPricePrecision,string"`
	MaxTradeQuantity  float64 `json:"maxTradeQuantity,string"`
	MaxTradeAmount    float64 `json:"maxTradeAmount,string"`
	Category          int64   `json:"category"`
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Bids   []OrderbookItem
	Asks   []OrderbookItem
	Symbol string
	Time   time.Time
}

// TradeItem stores a single trade
type TradeItem struct {
	CurrencyPair string
	Price        float64
	Side         string
	Volume       float64
	TradeTime    time.Time
}

// KlineItem stores an individual kline data item
type KlineItem struct {
	StartTime        time.Time
	EndTime          time.Time
	Open             float64
	Close            float64
	High             float64
	Low              float64
	Volume           float64
	QuoteAssetVolume float64
	TakerBaseVolume  float64
	TakerQuoteVolume float64
	TradesCount      int64
}

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Time         time.Time
	Symbol       string
	BestBidPrice float64
	BestAskPrice float64
	LastPrice    float64
	OpenPrice    float64
	HighPrice    float64
	LowPrice     float64
	Volume       float64
	QuoteVolume  float64
}

// LastTradePrice contains price for last trade
type LastTradePrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

// TickerData stores ticker data
type TickerData struct {
	Symbol      string
	BidPrice    float64
	BidQuantity float64
	AskPrice    float64
	AskQuantity float64
	Time        time.Time
}

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

var (
	// BybitRequestParamsOrderLimit Limit order
	BybitRequestParamsOrderLimit = RequestParamsOrderType("LIMIT")

	// BybitRequestParamsOrderMarket Market order
	BybitRequestParamsOrderMarket = RequestParamsOrderType("MARKET")

	// BybitRequestParamsOrderLimitMaker LIMIT_MAKER
	BybitRequestParamsOrderLimitMaker = RequestParamsOrderType("LIMIT_MAKER")
)

// RequestParamsTimeForceType Time in force
type RequestParamsTimeForceType string

var (
	// BybitRequestParamsTimeGTC GTC
	BybitRequestParamsTimeGTC = RequestParamsTimeForceType("GTC")

	// BybitRequestParamsTimeFOK FOK
	BybitRequestParamsTimeFOK = RequestParamsTimeForceType("FOK")

	// BybitRequestParamsTimeIOC IOC
	BybitRequestParamsTimeIOC = RequestParamsTimeForceType("IOC")
)

// PlaceOrderRequest request type
type PlaceOrderRequest struct {
	Symbol      string
	Quantity    float64
	Side        string
	TradeType   RequestParamsOrderType
	TimeInForce RequestParamsTimeForceType
	Price       float64
	OrderLinkID string
}

type PlaceOrderResponse struct {
	OrderID     int64                      `json:"orderId"`
	OrderLinkID string                     `json:"orderLinkId"`
	Symbol      string                     `json:"symbol"`
	Time        int64                      `json:"transactTime"`
	Price       float64                    `json:"price,string"`
	Quantity    float64                    `json:"origQty,string"`
	TradeType   RequestParamsOrderType     `json:"type"`
	Side        string                     `json:"side"`
	Status      string                     `json:"status"`
	TimeInForce RequestParamsTimeForceType `json:timeInForce`
	AccountID   int64                      `json:accountId`
	SymbolName  string                     `json:symbolName`
	ExecutedQty string                     `json:executedQty`
}

// QueryOrderResponse holds query order data
type QueryOrderResponse struct {
	AccountID           int64                      `json:accountId`
	ExchangeID          int64                      `json:exchangeId`
	Symbol              string                     `json:"symbol"`
	SymbolName          string                     `json:symbolName`
	OrderLinkID         string                     `json:"orderLinkId"`
	OrderID             int64                      `json:"orderId"`
	Price               float64                    `json:"price,string"`
	Quantity            float64                    `json:"origQty,string"`
	ExecutedQty         string                     `json:executedQty,string`
	CummulativeQuoteQty string                     `json:cummulativeQuoteQty,string`
	AveragePrice        float64                    `json:"avgPrice,string"`
	Status              string                     `json:"status"`
	TimeInForce         RequestParamsTimeForceType `json:timeInForce`
	TradeType           RequestParamsOrderType     `json:"type"`
	Side                string                     `json:"side"`
	StopPrice           float64                    `json:"stopPrice,string"`
	IcebergQty          float64                    `json:"icebergQty,string"`
	Time                int64                      `json:"time"`
	UpdateTime          int64                      `json:"updateTime"`
	isWorking           bool                       `json:"isWorking"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	OrderID     int64                      `json:"orderId"`
	OrderLinkID string                     `json:"orderLinkId"`
	Symbol      string                     `json:"symbol"`
	Status      string                     `json:"status"`
	AccountID   int64                      `json:accountId`
	Time        int64                      `json:"transactTime"`
	Price       float64                    `json:"price,string"`
	Quantity    float64                    `json:"origQty,string"`
	ExecutedQty string                     `json:executedQty,string`
	TimeInForce RequestParamsTimeForceType `json:timeInForce`
	TradeType   RequestParamsOrderType     `json:"type"`
	Side        string                     `json:"side"`
}

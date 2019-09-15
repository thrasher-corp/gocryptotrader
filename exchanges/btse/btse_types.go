package btse

import "time"

// OverviewData stores market overview data
type OverviewData struct {
	High24Hr         string `json:"high24hr"`
	HighestBid       string `json:"highestbid"`
	Last             string `json:"last"`
	Low24Hr          string `json:"low24hr"`
	LowestAsk        string `json:"lowest_ask"`
	PercentageChange string `json:"percentage_change"`
	Volume           string `json:"volume"`
}

// HighLevelMarketData stores market overview data
type HighLevelMarketData map[string]OverviewData

// Market stores market data
type Market struct {
	ID                  string  `json:"id"`
	BaseCurrency        string  `json:"base_currency"`
	QuoteCurrency       string  `json:"quote_currency"`
	BaseMinSize         float64 `json:"base_min_size"`
	BaseMaxSize         float64 `json:"base_max_size"`
	BaseIncremementSize float64 `json:"base_increment_size"`
	QuoteMinPrice       float64 `json:"quote_min_price"`
	QuoteIncrement      float64 `json:"quote_increment"`
	Status              string  `json:"status"`
}

// Markets stores an array of market data
type Markets []Market

// Trade stores trade data
type Trade struct {
	SerialID string  `json:"serial_id"`
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Time     string  `json:"time"`
	Type     string  `json:"type"`
}

// QuoteData stores quote data
type QuoteData struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	BuyQuote  []QuoteData `json:"buyQuote"`
	SellQuote []QuoteData `json:"sellQuote"`
	Symbol    string      `json:"symbol"`
	Timestamp int64       `json:"timestamp"`
}

// Trades stores an array of trade data
type Trades []Trade

// Ticker stores the ticker data
type Ticker struct {
	Price  float64 `json:"price,string"`
	Size   float64 `json:"size,string"`
	Bid    float64 `json:"bid,string"`
	Ask    float64 `json:"ask,string"`
	Volume float64 `json:"volume,string"`
	Time   string  `json:"time"`
}

// MarketStatistics stores market statistics for a particular product
type MarketStatistics struct {
	Open   float64 `json:"open,string"`
	Low    float64 `json:"low,string"`
	High   float64 `json:"high,string"`
	Close  float64 `json:"close,string"`
	Volume float64 `json:"volume,string"`
	Time   string  `json:"time"`
}

// ServerTime stores the server time data
type ServerTime struct {
	ISO   time.Time `json:"iso"`
	Epoch string    `json:"epoch"`
}

// CurrencyBalance stores the account info data
type CurrencyBalance struct {
	Currency  string  `json:"currency"`
	Total     float64 `json:"total,string"`
	Available float64 `json:"available,string"`
}

// AccountBalance stores an array of currency balances
type AccountBalance []CurrencyBalance

// Order stores the order info
type Order struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Tag       string  `json:"tag"`
	Symbol    string  `json:"symbol"`
	CreatedAt string  `json:"created_at"`
}

// OpenOrder stores an open order info
type OpenOrder struct {
	Order
	Status string `json:"status"`
}

// OpenOrders stores an array of orders
type OpenOrders []OpenOrder

// CancelOrder stores the cancel order response data
type CancelOrder struct {
	Code int   `json:"code"`
	Time int64 `json:"time"`
}

// FilledOrder stores filled order data
type FilledOrder struct {
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Fee       float64 `json:"fee"`
	Side      string  `json:"side"`
	Tag       string  `json:"tag"`
	ID        int64   `json:"id"`
	TradeID   string  `json:"trade_id"`
	Symbol    string  `json:"symbol"`
	OrderID   string  `json:"order_id"`
	CreatedAt string  `json:"created_at"`
}

// FilledOrders stores an array of filled orders
type FilledOrders []FilledOrder

type wsSub struct {
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

type wsQuoteData struct {
	Total string `json:"cumulativeTotal"`
	Price string `json:"price"`
	Size  string `json:"size"`
}

type wsOBData struct {
	Currency  string        `json:"currency"`
	BuyQuote  []wsQuoteData `json:"buyQuote"`
	SellQuote []wsQuoteData `json:"sellQuote"`
}

type wsOrderBook struct {
	Topic string   `json:"topic"`
	Data  wsOBData `json:"data"`
}

type wsTradeData struct {
	Amount           float64 `json:"amount"`
	Gain             int64   `json:"gain"`
	Newest           int64   `json:"newest"`
	Price            float64 `json:"price"`
	ID               int64   `json:"serialId"`
	TrasnsactionTime int64   `json:"transactionUnixTime"`
}

type wsTradeHistory struct {
	Topic string        `json:"topic"`
	Data  []wsTradeData `json:"data"`
}

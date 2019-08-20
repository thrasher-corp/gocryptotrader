package btse

import "time"

// Market stores market data
type Market struct {
	Symbol              string  `json:"symbol"`
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
	ProductID string  `json:"product_id"`
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
	ProductID string  `json:"product_id"`
	OrderID   string  `json:"order_id"`
	CreatedAt string  `json:"created_at"`
}

// FilledOrders stores an array of filled orders
type FilledOrders []FilledOrder

type websocketSubscribe struct {
	Type     string             `json:"type"`
	Channels []websocketChannel `json:"channels"`
}

type websocketChannel struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids"`
}

type wsTicker struct {
	BestAsk   float64     `json:"best_ask,string"`
	BestBids  float64     `json:"best_bid,string"`
	LastSize  float64     `json:"last_size,string"`
	Price     interface{} `json:"price"`
	ProductID string      `json:"product_id"`
}

type websocketOrderbookSnapshot struct {
	ProductID string          `json:"product_id"`
	Type      string          `json:"type"`
	Bids      [][]interface{} `json:"bids"`
	Asks      [][]interface{} `json:"asks"`
}

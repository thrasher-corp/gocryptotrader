package huobi

// Ticker holds ticker information
type Ticker struct {
	High float64
	Low  float64
	Last float64
	Vol  float64
	Buy  float64
	Sell float64
}

// TickerResponse holds the initial response type
type TickerResponse struct {
	Time   string
	Ticker Ticker
}

// Orderbook holds the order book information
type Orderbook struct {
	ID     float64
	TS     float64
	Bids   [][]float64 `json:"bids"`
	Asks   [][]float64 `json:"asks"`
	Symbol string      `json:"string"`
}

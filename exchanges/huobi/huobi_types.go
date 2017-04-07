package huobi

type HuobiTicker struct {
	High float64
	Low  float64
	Last float64
	Vol  float64
	Buy  float64
	Sell float64
}

type HuobiTickerResponse struct {
	Time   string
	Ticker HuobiTicker
}

type HuobiOrderbook struct {
	ID     float64
	TS     float64
	Bids   [][]float64 `json:"bids"`
	Asks   [][]float64 `json:"asks"`
	Symbol string      `json:"string"`
}

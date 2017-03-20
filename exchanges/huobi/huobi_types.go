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

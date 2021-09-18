package bybit

var (
	validFuturesIntervals = []string{
		"1M", "3M", "5M", "15M", "30M", "60M", "120M", "240M", "360M", "720M",
		"1H", "3H", "5H", "15H", "30H", "60H", "120H", "240H", "360H", "720H",
		"1D", "3D", "5D", "15D", "30D", "60D", "120D", "240D", "360D", "720D",
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
	Symbol                 string `json:"symbol"`
	BidPrice               string `json:"bid_price"`
	AskPrice               string `json:"ask_price"`
	LastPrice              string `json:"last_price"`
	LastTickDirection      string `json:"last_tick_direction"`
	Price24hAgo            string `json:"prev_price_24h"`
	PricePcntChange24h     string `json:"price_24h_pcnt"`
	HighPrice24h           string `json:"high_price_24h"`
	LowPrice24h            string `json:"low_price_24h"`
	Price1hAgo             string `json:"prev_price_1h"`
	PricePcntChange1h      string `json:"price_1h_pcnt"`
	MarkPrice              string `json:"mark_price"`
	IndexPrice             string `json:"index_price"`
	OpenInterest           int64  `json:"open_interest"`
	OpenValue              string `json:"open_value"`
	TotalTurnover          string `json:"total_turnover"`
	Turnover24h            string `json:"turnover_24h"`
	TotalVolume            int64  `json:"total_volume"`
	Volume24h              int64  `json:"volume_24h"`
	FundingRate            string `json:"funding_rate"`
	PredictedFundingRate   string `json:"predicted_funding_rate"`
	NextFundingTime        string `json:"next_funding_time"`
	CountdownHour          int64  `json:"countdown_hour"`
	DeliveryFeeRate        string `json:"delivery_fee_rate"`
	PredictedDeliveryPrice string `json:"predicted_delivery_price"`
	DeliveryTime           string `json:"delivery_time"`
}

// FuturesCandleStick holds kline data
type FuturesCandleStick struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	OpenTime int64   `json:"open_time"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
	TurnOver float64 `json:"turnover"`
}

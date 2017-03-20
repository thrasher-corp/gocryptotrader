package itbit

type ItBitTicker struct {
	Pair          string
	Bid           float64 `json:",string"`
	BidAmt        float64 `json:",string"`
	Ask           float64 `json:",string"`
	AskAmt        float64 `json:",string"`
	LastPrice     float64 `json:",string"`
	LastAmt       float64 `json:",string"`
	Volume24h     float64 `json:",string"`
	VolumeToday   float64 `json:",string"`
	High24h       float64 `json:",string"`
	Low24h        float64 `json:",string"`
	HighToday     float64 `json:",string"`
	LowToday      float64 `json:",string"`
	OpenToday     float64 `json:",string"`
	VwapToday     float64 `json:",string"`
	Vwap24h       float64 `json:",string"`
	ServertimeUTC string
}

type ItbitOrderbookEntry struct {
	Quantitiy float64 `json:"quantity,string"`
	Price     float64 `json:"price,string"`
}

type ItBitOrderbookResponse struct {
	ServerTimeUTC      string                `json:"serverTimeUTC"`
	LastUpdatedTimeUTC string                `json:"lastUpdatedTimeUTC"`
	Ticker             string                `json:"ticker"`
	Bids               []ItbitOrderbookEntry `json:"bids"`
	Asks               []ItbitOrderbookEntry `json:"asks"`
}

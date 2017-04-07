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

type ItBitOrderbookResponse struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

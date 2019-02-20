package exchangerates

// Rates holds the latest forex rates info
type Rates struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// HistoricalRates stores the historical rate info
type HistoricalRates Rates

// TimeSeriesRates stores time series rate info
type TimeSeriesRates struct {
	Base    string                 `json:"base"`
	StartAt string                 `json:"start_at"`
	EndAt   string                 `json:"end_at"`
	Rates   map[string]interface{} `json:"rates"`
}

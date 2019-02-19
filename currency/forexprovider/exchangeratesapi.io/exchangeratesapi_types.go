package exchangerates

// Latest holds the latest forex rates info
type Latest struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

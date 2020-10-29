package trade

import "time"

// Data defines trade data in its simplest
// db friendly form
type Data struct {
	ID             string
	TID            string
	Exchange       string
	ExchangeNameID string
	Base           string
	Quote          string
	AssetType      string
	Price          float64
	Amount         float64
	Side           string
	Timestamp      time.Time
}

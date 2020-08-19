package trade

// Data defines trade data in its simplest
// db friendly form
type Data struct {
	ID string
	Timestamp    int64
	Exchange     string
	CurrencyPair string
	AssetType    string
	Price        float64
	Amount       float64
	Side        string
}

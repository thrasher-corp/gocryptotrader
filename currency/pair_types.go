package currency

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter"`
	Base      Code   `json:"base"`
	Quote     Code   `json:"quote"`
}

// Pairs defines a list of pairs
type Pairs []Pair

package currency

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter,omitempty"`
	Base      Code   `json:"base,omitempty"`
	Quote     Code   `json:"quote,omitempty"`
	key       PairKey
}

// Pairs defines a list of pairs
type Pairs []Pair

// PairKey is an identifier for use in maps
type PairKey string

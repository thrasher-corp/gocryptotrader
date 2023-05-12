package currency

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter,omitempty"`
	Base      Code   `json:"base,omitempty"`
	Quote     Code   `json:"quote,omitempty"`
}

// Pairs defines a list of pairs
type Pairs []Pair

// PairDifference defines the difference between a set of pairs including a
// change in format.
type PairDifference struct {
	New              Pairs
	Remove           Pairs
	FormatDifference bool
}

// Common pairs so we don't need to define them all the time
var (
	BTCUSDT = Pair{Base: BTC, Quote: USDT}
)

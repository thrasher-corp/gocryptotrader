package currency

// Pair holds currency pair information
// NOTE: UnmarshalJSON allows string conversion to Pair type but only if there
// is a delimiter present in the string, otherwise it will return an error.
type Pair struct {
	Delimiter string `json:"delimiter,omitempty"`
	Base      Code   `json:"base"`
	Quote     Code   `json:"quote"`
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

// OrderParameters is used to determine the order side, liquidity side and the
// selling & purchasing currency derived from the currency pair.
type OrderParameters struct {
	// SellingCurrency is the currency that will be sold first
	SellingCurrency Code
	// Purchasing is the currency that will be purchased last
	PurchasingCurrency Code
	// IsBuySide is the side of the order that will be placed true for buy/long,
	// false for sell/short.
	IsBuySide bool
	// IsAskLiquidity is the side of the orderbook that will be used, false for
	// bid liquidity.
	IsAskLiquidity bool
	// Pair is the currency pair that the order parameters are derived from.
	Pair Pair
}

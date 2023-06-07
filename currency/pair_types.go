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

// OrderDecisionDetails defines the information that describes an order
// implementation to the actual liquidity. This is used to determine the order
// side, the liquidity side, the currency pair and the selling and purchasing
// currency.
type OrderDecisionDetails struct {
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
	// Pair is the currency pair that will be used
	Pair Pair
}

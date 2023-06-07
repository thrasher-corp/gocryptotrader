package order

// SubmissionConfig defines the order submission configuration for an exchange.
// This allows for a more generic approach to submitting orders.
type SubmissionConfig struct {
	// OrderBaseAmountsRequired refers to an exchange that uses the base currency
	// only as the amount for an order.
	OrderBaseAmountsRequired bool
	// OrderSellingAmountsRequired refers to an exchange that changes the the
	// amount identity when selling or buying. E.g. Bybit BTC-USD when
	// bidding/buying BTC the amount is the selling currency USDT, but when
	// asking/selling, the amount is BTC.
	OrderSellingAmountsRequired bool
	// FeeAppliedToSellingCurrency refers to an exchange that applies the fee to
	// the selling currency. E.g. when buying 1 BTC with 100 USDT, the fee is
	// applied to the 100 USDT. You will receive the entire amount of BTC.
	FeeAppliedToSellingCurrency bool
	// FeeAppliedToPurchasedCurrency refers to an exchange that applies the fee
	// to the purchased currency. E.g. when buying 1 BTC with 100 USDT, the fee
	// is applied to the 1 BTC. You will receive the entire amount of BTC minus
	// the fee.
	FeeAppliedToPurchasedCurrency bool
	// RequiresParameterLimits refers to an exchange that requires
	// exchange parameters to be confined to precision and amount guidelines for
	// order submission.
	RequiresParameterLimits bool
}

// func NewOrderBuilder() *Builder {
// 	return &Builder{}
// }

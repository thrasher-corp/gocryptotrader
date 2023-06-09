package order

// SubmissionConfig defines the order submission configuration for an exchange.
// This struct allows for a more generic approach to submitting orders.
type SubmissionConfig struct {
	// OrderBaseAmountsRequired indicates whether the exchange requires the base
	// currencyonly as the amount for an order.
	OrderBaseAmountsRequired bool
	// OrderSellingAmountsRequired indicates whether the exchange changes the
	// amount identity when selling or buying. For example, some exchanges use a
	// different currency for the amount when bidding/buying versus
	// asking/selling.
	OrderSellingAmountsRequired bool
	// FeeAppliedToSellingCurrency indicates whether the exchange applies the
	// fee to the selling currency. For example, when buying 1 BTC with 100 USDT,
	// the fee is applied to the 100 USDT, and you receive  the entire amount of
	// BTC.
	FeeAppliedToSellingCurrency bool
	// FeeAppliedToPurchasedCurrency indicates whether the exchange applies the
	// fee to the purchased currency. For example, when buying 1 BTC with 100
	// USDT, the fee is applied to the 1 BTC, and you receive the entire amount
	// of BTC minus the fee.
	FeeAppliedToPurchasedCurrency bool
	// FeePostOrderRequiresPrecisionOnAmount indicates whether the exchange
	// performs its fee calculation after converting the amount to the precision.
	// If true, the exchange converts the amount to precision first and then
	// calculates the fee. If false, the exchange calculates the fee first and
	// then converts the amount to precision. The precise behavior depends on
	// the exchange's implementation.
	FeePostOrderRequiresPrecisionOnAmount bool
	// RequiresParameterLimits indicates whether the exchange requires exchange
	// parameters to be confinedto precision and amount guidelines for order
	// submission.
	RequiresParameterLimits bool
}

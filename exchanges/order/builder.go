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
	// FeePostOrderRequiresPrecisionOnAmount refers to when an exchange performs
	// its fee calculation after they convert the amount to the precision. Thus
	// putting is in their favour. e.g.
	// BTC-USDT = $26359.28
	// TradingFee - 0.1%
	// Selling 5 USDT at that rate would give you ~0.00018968651647541208 BTC
	// If true the exchange would convert the amount to precision first 0.000189
	// then calculate the fee 0.000189 - 0.1% = 0.000188811
	// Which would give you the balance of 0.000188811 BTC or 0.000188 BTC (front end precision).
	// If false the exchange would calculate the fee first 0.00018968651647541208 - 0.1% =
	// 0.00018949782782306367
	// Which would give you the balance of 0.00018949782782306367 BTC or 0.000189 BTC (front end precision).
	FeePostOrderRequiresPrecisionOnAmount bool
	// RequiresParameterLimits refers to an exchange that requires
	// exchange parameters to be confined to precision and amount guidelines for
	// order submission.
	RequiresParameterLimits bool
}

// func NewOrderBuilder() *Builder {
// 	return &Builder{}
// }

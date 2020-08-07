package backtest

type ExchangeFeeHandler interface {
	Calculate(amount, price float64) (float64, error)
}

type ExchangeFee struct {
	Fee float64
}

type FixedExchangeFee struct {
	ExchangeFee
}

type PercentageFee struct {
	ExchangeFee
}
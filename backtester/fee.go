package backtest

type ExchangeFeeHandler interface {
	Fee() (float64, error)
}

type FixedExchangeFee struct {
	ExchangeFee float64
}

func (e *FixedExchangeFee) Fee() (float64, error) {
	return e.ExchangeFee, nil
}

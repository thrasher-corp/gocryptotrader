package backtest

func (e *FixedExchangeFee) Calculate(_,_ float64) (float64, error) {
	return e.ExchangeFee.Fee, nil
}

func (c *PercentageFee) Calculate(amount, price float64) (float64, error) {
	if amount == 0 || price == 0 {
		return 0, nil
	}
	return amount * price * c.Fee, nil
}
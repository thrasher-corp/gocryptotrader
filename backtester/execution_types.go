package backtest

type ExecutionHandler interface {
	OnData(DataEvent, *Backtest) (OrderEvent, error)
}

type Exchange struct {
	ExchangeFee ExchangeFeeHandler
}

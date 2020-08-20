package backtest

type ExecutionHandler interface {
	OnData(DataEvent, *Backtest) (OrderEvent, error)
}

type Execution struct {
	ExchangeFee ExchangeFeeHandler
}

package backtest

type Portfolio struct {
	InitialFunds float64
	Funds        float64
	holdings     Position
	orderBook    []OrderEvent
}

type PortfolioHandler interface {
	OnSignal(SignalEvent, DataHandler) (*Order, error)
	OnFill(*Order, DataHandler) (*Order, error)
	IsInvested() (Position, bool)
	IsLong() (Position, bool)
	IsShort() (Position, bool)
	Update(DataEvent)

	InitialCash() float64
	SetInitialCash(float64)
	Cash() float64
	SetCash(float64)

	Value() float64

	Reset() error

	Order(price float64, num int64)

	Position() Position
}
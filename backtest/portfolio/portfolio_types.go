package portfolio

type Handler interface{
	Funds
}

type Funds interface{
	Initial() float64
	SetInitial(float64)
	Funds() float64
	SetFunds(float64)
}

type Portfolio struct {
	InitialFunds float64
	FundsOnHand float64
}

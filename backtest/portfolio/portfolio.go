package portfolio

func New(funds float64) *Portfolio {
	return &Portfolio{
		InitialFunds: funds,
	}
}
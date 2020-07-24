package fee

type Handler interface{
	Calculate(amount, price float64) float64
}
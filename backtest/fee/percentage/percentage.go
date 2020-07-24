package percentage

type Fee struct {
	Fee float64
}

func (f *Fee) Calculate(amount, price float64) float64 {
	return amount * price * f.Fee
}
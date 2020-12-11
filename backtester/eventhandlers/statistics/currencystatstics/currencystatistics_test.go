package currencystatstics

import "testing"

func TestSortinoRatio(t *testing.T) {
	rfr := 0.07
	figures := []float64{0.10, 0.04, 0.15, -0.05, 0.20, -0.02, 0.08, -0.06, 0.13, 0.23}
	negativeOnlyFigures := []float64{-0.05, -0.02, -0.06}
	r := calculateSortinoRatio(figures, negativeOnlyFigures, rfr)
	if r != 0.3922322702763678 {
		t.Errorf("received %v instead", r)
	}
}

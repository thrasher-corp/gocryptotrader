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

func TestInformationRatio(t *testing.T) {
	figures := []float64{0.0665, 0.0283, 0.0911, 0.0008, -0.0203, -0.0978, 0.0164, -0.0537, 0.078, 0.0032, 0.0249, 0}
	comparisonFigures := []float64{0.0216, 0.0048, 0.036, 0.0303, 0.0043, -0.0694, 0.0179, -0.0918, 0.0787, 0.0297, 0.003, 0}
	avg := calculateTheAverage(figures)
	if avg != 0.01145 {
		t.Error(avg)
	}
	avgComparison := calculateTheAverage(comparisonFigures)
	if avgComparison != 0.005425 {
		t.Error(avgComparison)
	}

	var eachDiff []float64
	for i := range figures {
		eachDiff = append(eachDiff, figures[i]-comparisonFigures[i])
	}
	stdDev := calculateStandardDeviation(eachDiff)
	if stdDev != 0.028992588851865803 {
		t.Error(stdDev)
	}
	informationRatio := (avg - avgComparison) / stdDev
	if informationRatio != 0.20781172839666107 {
		t.Error(informationRatio)
	}

	information2 := calculateInformationRatio(figures, comparisonFigures)
	if informationRatio != information2 {
		t.Error(information2)
	}
}

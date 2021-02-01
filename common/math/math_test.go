package math

import (
	"math"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestCalculateFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(0.01)
	actualResult := CalculateFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateAmountWithFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(1.01)
	actualResult := CalculateAmountWithFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageGainOrLoss(t *testing.T) {
	t.Parallel()
	originalInput := float64(9300)
	secondInput := float64(9000)
	expectedOutput := 3.3333333333333335
	actualResult := CalculatePercentageGainOrLoss(originalInput, secondInput)
	if expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageDifference(t *testing.T) {
	t.Parallel()
	originalInput := float64(10)
	secondAmount := float64(5)
	expectedOutput := 66.66666666666666
	actualResult := CalculatePercentageDifference(originalInput, secondAmount)
	if expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateNetProfit(t *testing.T) {
	t.Parallel()
	amount := float64(5)
	priceThen := float64(1)
	priceNow := float64(10)
	costs := float64(1)
	expectedOutput := float64(44)
	actualResult := CalculateNetProfit(amount, priceThen, priceNow, costs)
	if expectedOutput != actualResult {
		t.Errorf(
			"Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestRoundFloat(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result : map[precision]map[testedValue]expectedOutput
	testTableValues := map[int]map[float64]float64{
		0: {
			2.23456789:  2,
			-2.23456789: -2,
		},
		1: {
			2.23456789:  2.2,
			-2.23456789: -2.2,
		},
		2: {
			2.23456789:  2.23,
			-2.23456789: -2.23,
		},
		4: {
			2.23456789:  2.2346,
			-2.23456789: -2.2346,
		},
		8: {
			2.23456781:  2.23456781,
			-2.23456781: -2.23456781,
		},
	}

	for precision, values := range testTableValues {
		for testInput, expectedOutput := range values {
			actualOutput := RoundFloat(testInput, precision)
			if actualOutput != expectedOutput {
				t.Errorf("RoundFloat Expected '%v'. Actual '%v' on precission %d",
					expectedOutput, actualOutput, precision)
			}
		}
	}
}

func TestSortinoRatio(t *testing.T) {
	rfr := 0.07
	figures := []float64{0.10, 0.04, 0.15, -0.05, 0.20, -0.02, 0.08, -0.06, 0.13, 0.23}
	negativeOnlyFigures := []float64{-0.05, -0.02, -0.06}
	r := CalculateSortinoRatio(figures, negativeOnlyFigures, rfr)
	if r != 0.3922322702763678 {
		t.Errorf("received %v instead", r)
	}
}

func TestInformationRatio(t *testing.T) {
	figures := []float64{0.0665, 0.0283, 0.0911, 0.0008, -0.0203, -0.0978, 0.0164, -0.0537, 0.078, 0.0032, 0.0249, 0}
	comparisonFigures := []float64{0.0216, 0.0048, 0.036, 0.0303, 0.0043, -0.0694, 0.0179, -0.0918, 0.0787, 0.0297, 0.003, 0}
	avg := CalculateTheAverage(figures)
	if avg != 0.01145 {
		t.Error(avg)
	}
	avgComparison := CalculateTheAverage(comparisonFigures)
	if avgComparison != 0.005425 {
		t.Error(avgComparison)
	}

	var eachDiff []float64
	for i := range figures {
		eachDiff = append(eachDiff, figures[i]-comparisonFigures[i])
	}
	stdDev := CalculateStandardDeviation(eachDiff)
	if stdDev != 0.028992588851865803 {
		t.Error(stdDev)
	}
	informationRatio := (avg - avgComparison) / stdDev
	if informationRatio != 0.20781172839666107 {
		t.Error(informationRatio)
	}

	information2 := CalculateInformationRatio(figures, comparisonFigures)
	if informationRatio != information2 {
		t.Error(information2)
	}
}

func TestCalmarRatio(t *testing.T) {
	avg := []float64{0.2}
	ratio := CalculateCalmarRatio(avg, 50000, 15000)
	if ratio != 0.28571428571428575 {
		t.Error(ratio)
	}
}

func TestCAGR(t *testing.T) {
	cagr := CalculateCompoundAnnualGrowthRate(
		100,
		147,
		time.Date(2015, 1, 1, 0, 0, 0, 0, time.Local).Round(kline.OneYear.Duration()),
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local).Round(kline.OneYear.Duration()),
		kline.OneYear)
	if cagr != 8.009875865888949 {
		t.Log(cagr)
		t.Error(cagr)
	}
}

func TestCalculateSharpeRatio(t *testing.T) {
	result := CalculateSharpeRatio(nil, 0)
	if result != 0 {
		t.Error("expected 0")
	}

	result = CalculateSharpeRatio([]float64{0.026}, 0.017)
	if result != 0 {
		t.Error("expected 0")
	}

	returns := []float64{
		0.08,
		0.1,
		0.09,
		0.06,
		0.07,
		0.11,
		0.08,
		0.1,
		0.02,
		0.09,
	}
	result = CalculateSharpeRatio(returns, 0.04)
	if result != 1.5491933384829664 {
		t.Error("expected 1.55~")
	}
}

func TestStandardDeviation2(t *testing.T) {
	r := []float64{9, 2, 5, 4, 12, 7}
	mean := CalculateTheAverage(r)
	superMean := []float64{}
	for i := range r {
		result := math.Pow(r[i]-mean, 2)
		superMean = append(superMean, result)
	}
	superMeany := (superMean[0] + superMean[1] + superMean[2] + superMean[3] + superMean[4] + superMean[5]) / 5
	manualCalculation := math.Sqrt(superMeany)
	codeCalcu := CalculateSampleStandardDeviation(r)
	if manualCalculation != codeCalcu && codeCalcu != 3.619 {
		t.Error("expected 3.619")
	}
}

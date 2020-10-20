package math

import "testing"

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

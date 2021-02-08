package math

import (
	"math"
)

// CalculateAmountWithFee returns a calculated fee included amount on fee
func CalculateAmountWithFee(amount, fee float64) float64 {
	return amount + CalculateFee(amount, fee)
}

// CalculateFee returns a simple fee on amount
func CalculateFee(amount, fee float64) float64 {
	return amount * (fee / 100)
}

// CalculatePercentageGainOrLoss returns the percentage rise over a certain
// period
func CalculatePercentageGainOrLoss(priceNow, priceThen float64) float64 {
	return (priceNow - priceThen) / priceThen * 100
}

// CalculatePercentageDifference returns the percentage of difference between
// multiple time periods
func CalculatePercentageDifference(amount, secondAmount float64) float64 {
	return (amount - secondAmount) / ((amount + secondAmount) / 2) * 100
}

// CalculateNetProfit returns net profit
func CalculateNetProfit(amount, priceThen, priceNow, costs float64) float64 {
	return (priceNow * amount) - (priceThen * amount) - costs
}

// RoundFloat rounds your floating point number to the desired decimal place
func RoundFloat(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	return math.Round(x*pow) / pow
}

// CalculateCompoundAnnualGrowthRate Calculates CAGR.
// Using years, intervals per year would be 1 and number of intervals would be the number of years
// Using days, intervals per year would be 365 and number of intervals would be the number of days
// et cetera et cetera
func CalculateCompoundAnnualGrowthRate(openValue, closeValue, intervalsPerYear, numberOfIntervals float64) float64 {
	k := math.Pow(closeValue/openValue, intervalsPerYear/numberOfIntervals) - 1
	return k * 100
}

func CalculateCalmarRatio(values []float64, highestPrice, lowestPrice float64) float64 {
	if highestPrice == 0 {
		return 0
	}
	avg := CalculateTheAverage(values)
	drawdownDiff := (highestPrice - lowestPrice) / highestPrice
	if drawdownDiff == 0 {
		return 0
	}
	return avg / drawdownDiff
}

// CalculateInformationRatio calculates the information ratio
func CalculateInformationRatio(values, riskFreeRates []float64) float64 {
	if len(riskFreeRates) == 1 {
		for i := range values {
			if i == 0 {
				continue
			}
			riskFreeRates = append(riskFreeRates, riskFreeRates[0])
		}
	}
	avgValue := CalculateTheAverage(values)
	avgComparison := CalculateTheAverage(riskFreeRates)
	var diffs []float64
	for i := range values {
		diffs = append(diffs, values[i]-riskFreeRates[i])
	}
	stdDev := CalculatePopulationStandardDeviation(diffs)
	if stdDev == 0 {
		return 0
	}
	return (avgValue - avgComparison) / stdDev
}

// CalculatePopulationStandardDeviation calculates standard deviation using population based calculation
func CalculatePopulationStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := CalculateTheAverage(values)

	diffs := make([]float64, len(values))
	for x := range values {
		diffs[x] = math.Pow(values[x]-avg, 2)
	}
	return math.Sqrt(CalculateTheAverage(diffs))
}

// CalculateSampleStandardDeviation calculates standard deviation using sample based calculation
func CalculateSampleStandardDeviation(vals []float64) float64 {
	if len(vals) <= 1 {
		return 0
	}
	mean := CalculateTheAverage(vals)
	var superMean []float64
	var combined float64
	for i := range vals {
		result := math.Pow(vals[i]-mean, 2)
		superMean = append(superMean, result)
		combined += result
	}

	avg := combined / (float64(len(superMean)) - 1)
	return math.Sqrt(avg)
}

// CalculateTheAverage returns the average value in a slice of floats
func CalculateTheAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sumOfValues float64
	for x := range values {
		sumOfValues += values[x]
	}
	return sumOfValues / float64(len(values))
}

// CalculateSortinoRatio returns sortino ratio of backtest compared to risk-free
func CalculateSortinoRatio(movementPerCandle, excessMovement []float64, riskFreeRate float64) float64 {
	mean := CalculateTheAverage(movementPerCandle)
	if mean == 0 {
		return 0
	}
	if len(excessMovement) == 0 {
		return 0
	}
	totalNegativeResultsSquared := 0.0
	for x := range excessMovement {
		totalNegativeResultsSquared += math.Pow(excessMovement[x], 2)
	}

	averageDownsideDeviation := math.Sqrt(totalNegativeResultsSquared / float64(len(movementPerCandle)))

	return (mean - riskFreeRate) / averageDownsideDeviation
}

// CalculateSharpeRatio returns sharpe ratio of backtest compared to risk-free
func CalculateSharpeRatio(movementPerCandle []float64, riskFreeRate float64) float64 {
	if len(movementPerCandle) <= 1 {
		return 0
	}
	mean := CalculateTheAverage(movementPerCandle)
	var excessReturns []float64
	for i := range movementPerCandle {
		excessReturns = append(excessReturns, movementPerCandle[i]-riskFreeRate)
	}
	standardDeviation := CalculateSampleStandardDeviation(excessReturns)

	if standardDeviation == 0 {
		return 0
	}
	return (mean - riskFreeRate) / standardDeviation
}

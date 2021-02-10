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

// CalculateCalmarRatio is a function of the average compounded annual rate of return versus its maximum drawdown.
// The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months
func CalculateCalmarRatio(values []float64, highestPrice, lowestPrice float64, isGeometric bool) float64 {
	if highestPrice == 0 {
		return 0
	}
	avg := CalculateTheAverage(values, isGeometric)
	drawdownDiff := (highestPrice - lowestPrice) / highestPrice
	if drawdownDiff == 0 {
		return 0
	}
	return avg / drawdownDiff
}

// CalculateInformationRatio The information ratio (IR) is a measurement of portfolio returns beyond the returns of a benchmark,
// usually an index, compared to the volatility of those returns.
// The benchmark used is typically an index that represents the market or a particular sector or industry.
func CalculateInformationRatio(values, riskFreeRates []float64, isGeometric bool) float64 {
	if len(riskFreeRates) == 1 {
		for i := range values {
			if i == 0 {
				continue
			}
			riskFreeRates = append(riskFreeRates, riskFreeRates[0])
		}
	}
	avgValue := CalculateTheAverage(values, isGeometric)
	avgComparison := CalculateTheAverage(riskFreeRates, isGeometric)
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
	avg := CalculateTheAverage(values, false)

	diffs := make([]float64, len(values))
	for x := range values {
		diffs[x] = math.Pow(values[x]-avg, 2)
	}
	return math.Sqrt(CalculateTheAverage(diffs, false))
}

// CalculateSampleStandardDeviation calculates standard deviation using sample based calculation
func CalculateSampleStandardDeviation(vals []float64) float64 {
	if len(vals) <= 1 {
		return 0
	}
	mean := CalculateTheAverage(vals, false)
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
func CalculateTheAverage(values []float64, isGeometric bool) float64 {
	if isGeometric {
		product := 1.0
		for i := range values {
			if values[i] >= 0 {
				// as we cannot have negative or zero value geometric numbers
				// adding a 1 to the percentage movements allows for differentiation between
				// negative numbers (eg -0.1 translates to 0.1) and positive numbers (eg 0.1 becomes 1.1)
				values[i] += 1
			}
			if values[i] < 0 {
				values[i] *= -1
			}
			product *= values[i]
		}

		// we minus 1 because we manipulated the values to be non-zero/negative
		return math.Pow(product, 1/float64(len(values)))
	}

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
func CalculateSortinoRatio(movementPerCandle []float64, riskFreeRate float64, isGeometric bool) float64 {
	mean := CalculateTheAverage(movementPerCandle, isGeometric)
	if mean == 0 {
		return 0
	}
	totalNegativeResultsSquared := 0.0
	for x := range movementPerCandle {
		if movementPerCandle[x]-riskFreeRate < 0 {
			totalNegativeResultsSquared += math.Pow(movementPerCandle[x]-riskFreeRate, 2)

		}
	}

	averageDownsideDeviation := math.Sqrt(totalNegativeResultsSquared / float64(len(movementPerCandle)))

	return (mean - riskFreeRate) / averageDownsideDeviation
}

// CalculateSharpeRatio returns sharpe ratio of backtest compared to risk-free
func CalculateSharpeRatio(movementPerCandle []float64, riskFreeRate float64, isGeometric bool) float64 {
	if len(movementPerCandle) <= 1 {
		return 0
	}
	mean := CalculateTheAverage(movementPerCandle, isGeometric)
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

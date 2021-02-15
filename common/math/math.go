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
func CalculateCompoundAnnualGrowthRate(openValue, closeValue, intervalsPerYear, numberOfIntervals float64) float64 {
	k := math.Pow(closeValue/openValue, intervalsPerYear/numberOfIntervals) - 1
	return k * 100
}

// CalculateCalmarRatio is a function of the average compounded annual rate of return versus its maximum drawdown.
// The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months
func CalculateCalmarRatio(highestPrice, lowestPrice, average float64) float64 {
	if highestPrice == 0 {
		return 0
	}
	drawdownDiff := (highestPrice - lowestPrice) / highestPrice
	if drawdownDiff == 0 {
		return 0
	}
	return average / drawdownDiff
}

// CalculateInformationRatio The information ratio (IR) is a measurement of portfolio returns beyond the returns of a benchmark,
// usually an index, compared to the volatility of those returns.
// The benchmark used is typically an index that represents the market or a particular sector or industry.
func CalculateInformationRatio(values, benchmarkRates []float64, averageValues, averageComparison float64) float64 {
	if len(benchmarkRates) == 1 {
		for i := range values {
			if i == 0 {
				continue
			}
			benchmarkRates = append(benchmarkRates, benchmarkRates[0])
		}
	}
	var diffs []float64
	for i := range values {
		diffs = append(diffs, values[i]-benchmarkRates[i])
	}
	stdDev := PopulationStandardDeviation(diffs)
	if stdDev == 0 {
		return 0
	}
	return (averageValues - averageComparison) / stdDev
}

// PopulationStandardDeviation calculates standard deviation using population based calculation
func PopulationStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := ArithmeticAverage(values)
	diffs := make([]float64, len(values))
	for x := range values {
		diffs[x] = math.Pow(values[x]-avg, 2)
	}
	return math.Sqrt(ArithmeticAverage(diffs))
}

// SampleStandardDeviation standard deviation is a statistic that
// measures the dispersion of a dataset relative to its mean and
// is calculated as the square root of the variance
func SampleStandardDeviation(vals []float64) float64 {
	if len(vals) <= 1 {
		return 0
	}
	mean := ArithmeticAverage(vals)
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

// GeometricAverage is an average which indicates the central tendency or
// typical value of a set of numbers by using the product of their values
// The geometric average can only process positive numbers
func GeometricAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	product := 1.0
	for i := range values {
		if values[i] <= 0 {
			// cannot use negative or zero values in geometric calculation
			return 0
		}
		product *= values[i]
	}
	geometricPower := math.Pow(product, 1/float64(len(values)))
	return geometricPower
}

// FinancialGeometricAverage is a modified geometric average to assess
// the negative returns of investments. It accepts It adds +1 to each
// This does impact the final figures as it is modifying values
// It is still ultimately calculating a geometric average
// which should only be compared to other financial geometric averages
func FinancialGeometricAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	product := 1.0
	for i := range values {
		if values[i] <= -1 {
			// cannot lose more than 100%, figures are incorrect
			return 0
		}
		// as we cannot have negative or zero value geometric numbers
		// adding a 1 to the percentage movements allows for differentiation between
		// negative numbers (eg -0.1 translates to 0.9) and positive numbers (eg 0.1 becomes 1.1)
		modVal := values[i] + 1
		product *= modVal
	}
	geometricPower := math.Pow(product, 1/float64(len(values)))
	// we minus 1 because we manipulated the values to be non-zero/negative
	return geometricPower - 1
}

// ArithmeticAverage is the basic form of calculating an average.
// Divide the sum of all values by the length of values
func ArithmeticAverage(values []float64) float64 {
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
func CalculateSortinoRatio(movementPerCandle []float64, riskFreeRate, average float64) float64 {
	totalNegativeResultsSquared := 0.0
	for x := range movementPerCandle {
		if movementPerCandle[x]-riskFreeRate < 0 {
			totalNegativeResultsSquared += math.Pow(movementPerCandle[x]-riskFreeRate, 2)
		}
	}
	averageDownsideDeviation := math.Sqrt(totalNegativeResultsSquared / float64(len(movementPerCandle)))
	return (average - riskFreeRate) / averageDownsideDeviation
}

// CalculateSharpeRatio returns sharpe ratio of backtest compared to risk-free
func CalculateSharpeRatio(movementPerCandle []float64, riskFreeRate, average float64) float64 {
	if len(movementPerCandle) <= 1 {
		return 0
	}
	var excessReturns []float64
	for i := range movementPerCandle {
		excessReturns = append(excessReturns, movementPerCandle[i]-riskFreeRate)
	}
	standardDeviation := SampleStandardDeviation(excessReturns)
	if standardDeviation == 0 {
		return 0
	}
	return (average - riskFreeRate) / standardDeviation
}

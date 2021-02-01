package math

import (
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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

func CalculateCompoundAnnualGrowthRate(openValue, closeValue float64, start, end time.Time, interval kline.Interval) float64 {
	p := kline.TotalCandlesPerInterval(start, end, interval)

	k := math.Pow(closeValue/openValue, 1/float64(p)) - 1
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
	stdDev := CalculateStandardDeviation(diffs)
	if stdDev == 0 {
		return 0
	}
	return (avgValue - avgComparison) / stdDev
}

func CalculateStandardDeviation(values []float64) float64 {
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

// CalculateSampleStandardDeviation is used in sharpe ratio calculations
// calculates the sample rate standard deviation
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
	standardDeviation := CalculateSampleStandardDeviation(movementPerCandle)

	if standardDeviation == 0 {
		return 0
	}
	return (mean - riskFreeRate) / standardDeviation
}

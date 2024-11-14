package math

import (
	"errors"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

var (
	// ErrNoNegativeResults is returned when no negative results are allowed
	ErrNoNegativeResults = errors.New("cannot calculate with no negative values")
	// ErrInexactConversion is returned when a decimal does not convert to float exactly
	ErrInexactConversion = errors.New("inexact conversion from decimal to float detected")
	// ErrPowerDifferenceTooSmall when values are too close when calculating the exponent value,
	// it returns zero
	ErrPowerDifferenceTooSmall = errors.New("calculated power is too small to use")

	errZeroValue               = errors.New("cannot calculate average of no values")
	errNegativeValueOutOfRange = errors.New("received negative number less than -1")
	errGeometricNegative       = errors.New("cannot calculate a geometric mean with negative values")
	errCalmarHighest           = errors.New("cannot calculate calmar ratio with highest price of 0")
	errCAGRNoIntervals         = errors.New("cannot calculate CAGR with no intervals")
	errCAGRZeroOpenValue       = errors.New("cannot calculate CAGR with an open value of 0")
	errInformationBadLength    = errors.New("benchmark rates length does not match returns rates")

	one        = decimal.NewFromInt(1)
	two        = decimal.NewFromInt(2)
	oneHundred = decimal.NewFromInt(100)
)

// CalculateAmountWithFee returns a calculated fee included amount on fee
func CalculateAmountWithFee(amount, fee float64) float64 {
	return amount + CalculateFee(amount, fee)
}

// CalculateFee returns a simple fee on amount
func CalculateFee(amount, fee float64) float64 {
	return amount * (fee / 100)
}

// PercentageChange returns the percentage change between two numbers, x is reference value.
func PercentageChange(x, y float64) float64 {
	return (y - x) / x * 100
}

// PercentageDifference returns difference between two numbers as a percentage of their average
func PercentageDifference(x, y float64) float64 {
	return math.Abs(x-y) / ((x + y) / 2) * 100
}

// PercentageDifferenceDecimal returns the difference between two decimal values as a percentage of their average
func PercentageDifferenceDecimal(x, y decimal.Decimal) decimal.Decimal {
	if x.IsZero() && y.IsZero() {
		return decimal.Zero
	}
	return x.Sub(y).Abs().Div(x.Add(y).Div(two)).Mul(oneHundred)
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

// CompoundAnnualGrowthRate Calculates CAGR.
// Using years, intervals per year would be 1 and number of intervals would be the number of years
// Using days, intervals per year would be 365 and number of intervals would be the number of days
func CompoundAnnualGrowthRate(openValue, closeValue, intervalsPerYear, numberOfIntervals float64) (float64, error) {
	if numberOfIntervals == 0 {
		return 0, errCAGRNoIntervals
	}
	if openValue == 0 {
		return 0, errCAGRZeroOpenValue
	}
	k := math.Pow(closeValue/openValue, intervalsPerYear/numberOfIntervals) - 1
	return k * 100, nil
}

// CalmarRatio is a function of the average compounded annual rate of return versus its maximum drawdown.
// The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months
func CalmarRatio(highestPrice, lowestPrice, average, riskFreeRateForPeriod float64) (float64, error) {
	if highestPrice == 0 {
		return 0, errCalmarHighest
	}
	drawdownDiff := (highestPrice - lowestPrice) / highestPrice
	if drawdownDiff == 0 {
		return 0, nil
	}
	return (average - riskFreeRateForPeriod) / drawdownDiff, nil
}

// InformationRatio The information ratio (IR) is a measurement of portfolio returns beyond the returns of a benchmark,
// usually an index, compared to the volatility of those returns.
// The benchmark used is typically an index that represents the market or a particular sector or industry.
func InformationRatio(returnsRates, benchmarkRates []float64, averageValues, averageComparison float64) (float64, error) {
	if len(benchmarkRates) != len(returnsRates) {
		return 0, errInformationBadLength
	}
	diffs := make([]float64, len(returnsRates))
	for i := range returnsRates {
		diffs[i] = returnsRates[i] - benchmarkRates[i]
	}
	stdDev, err := PopulationStandardDeviation(diffs)
	if err != nil {
		return 0, err
	}
	if stdDev == 0 {
		return 0, nil
	}
	return (averageValues - averageComparison) / stdDev, nil
}

// PopulationStandardDeviation calculates standard deviation using population based calculation
func PopulationStandardDeviation(values []float64) (float64, error) {
	if len(values) < 2 {
		return 0, nil
	}
	valAvg, err := ArithmeticMean(values)
	if err != nil {
		return 0, err
	}
	diffs := make([]float64, len(values))
	for x := range values {
		diffs[x] = math.Pow(values[x]-valAvg, 2)
	}
	var diffAvg float64
	diffAvg, err = ArithmeticMean(diffs)
	if err != nil {
		return 0, err
	}
	return math.Sqrt(diffAvg), nil
}

// SampleStandardDeviation standard deviation is a statistic that
// measures the dispersion of a dataset relative to its mean and
// is calculated as the square root of the variance
func SampleStandardDeviation(values []float64) (float64, error) {
	if len(values) < 2 {
		return 0, nil
	}
	mean, err := ArithmeticMean(values)
	if err != nil {
		return 0, err
	}
	superMean := make([]float64, len(values))
	var combined float64
	for i := range values {
		result := math.Pow(values[i]-mean, 2)
		superMean[i] = result
		combined += result
	}
	avg := combined / (float64(len(superMean)) - 1)
	return math.Sqrt(avg), nil
}

// GeometricMean is an average which indicates the central tendency or
// typical value of a set of numbers by using the product of their values
// The geometric average can only process positive numbers
func GeometricMean(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, errZeroValue
	}
	product := 1.0
	for i := range values {
		if values[i] <= 0 {
			// cannot use negative or zero values in geometric calculation
			return 0, errGeometricNegative
		}
		product *= values[i]
	}
	geometricPower := math.Pow(product, 1/float64(len(values)))
	return geometricPower, nil
}

// FinancialGeometricMean is a modified geometric average to assess
// the negative returns of investments. It accepts It adds +1 to each
// This does impact the final figures as it is modifying values
// It is still ultimately calculating a geometric average
// which should only be compared to other financial geometric averages
func FinancialGeometricMean(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, errZeroValue
	}
	product := 1.0
	for i := range values {
		if values[i] < -1 {
			// cannot lose more than 100%, figures are incorrect
			// losing exactly 100% will return a 0 value, but is not an error
			return 0, errNegativeValueOutOfRange
		}
		// as we cannot have negative or zero value geometric numbers
		// adding a 1 to the percentage movements allows for differentiation between
		// negative numbers (eg -0.1 translates to 0.9) and positive numbers (eg 0.1 becomes 1.1)
		modVal := values[i] + 1
		product *= modVal
	}
	prod := 1 / float64(len(values))
	geometricPower := math.Pow(product, prod)
	if geometricPower > 0 {
		// we minus 1 because we manipulated the values to be non-zero/negative
		geometricPower--
	}
	return geometricPower, nil
}

// ArithmeticMean is the basic form of calculating an average.
// Divide the sum of all values by the length of values
func ArithmeticMean(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, errZeroValue
	}
	var sumOfValues float64
	for x := range values {
		sumOfValues += values[x]
	}
	return sumOfValues / float64(len(values)), nil
}

// SortinoRatio returns sortino ratio of backtest compared to risk-free
func SortinoRatio(movementPerCandle []float64, riskFreeRatePerInterval, average float64) (float64, error) {
	totalIntervals := float64(len(movementPerCandle))
	if totalIntervals == 0 {
		return 0, errZeroValue
	}
	totalNegativeResultsSquared := 0.0
	for x := range movementPerCandle {
		if movementPerCandle[x]-riskFreeRatePerInterval < 0 {
			totalNegativeResultsSquared += math.Pow(movementPerCandle[x]-riskFreeRatePerInterval, 2)
		}
	}
	averageDownsideDeviation := math.Sqrt(totalNegativeResultsSquared / float64(len(movementPerCandle)))

	return (average - riskFreeRatePerInterval) / averageDownsideDeviation, nil
}

// SharpeRatio returns sharpe ratio of backtest compared to risk-free
func SharpeRatio(movementPerCandle []float64, riskFreeRatePerInterval, average float64) (float64, error) {
	totalIntervals := float64(len(movementPerCandle))
	if totalIntervals == 0 {
		return 0, errZeroValue
	}
	excessReturns := make([]float64, len(movementPerCandle))
	for i := range movementPerCandle {
		excessReturns[i] = movementPerCandle[i] - riskFreeRatePerInterval
	}
	standardDeviation, err := PopulationStandardDeviation(excessReturns)
	if err != nil {
		return 0, err
	}
	if standardDeviation == 0 {
		return 0, nil
	}

	return (average - riskFreeRatePerInterval) / standardDeviation, nil
}

// DecimalCompoundAnnualGrowthRate Calculates CAGR.
// Using years, intervals per year would be 1 and number of intervals would be the number of years
// Using days, intervals per year would be 365 and number of intervals would be the number of days
func DecimalCompoundAnnualGrowthRate(openValue, closeValue, intervalsPerYear, numberOfIntervals decimal.Decimal) (decimal.Decimal, error) {
	if numberOfIntervals.IsZero() {
		return decimal.Zero, errCAGRNoIntervals
	}
	if openValue.IsZero() {
		return decimal.Zero, errCAGRZeroOpenValue
	}
	closeOverOpen := closeValue.Div(openValue)
	exp := intervalsPerYear.Div(numberOfIntervals)
	pow := DecimalPow(closeOverOpen, exp)
	if pow.IsZero() {
		return decimal.Zero, ErrPowerDifferenceTooSmall
	}
	k := pow.Sub(one).Mul(oneHundred)
	return k, nil
}

// DecimalCalmarRatio is a function of the average compounded annual rate of return versus its maximum drawdown.
// The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months
func DecimalCalmarRatio(highestPrice, lowestPrice, average, riskFreeRateForPeriod decimal.Decimal) (decimal.Decimal, error) {
	if highestPrice.IsZero() {
		return decimal.Zero, errCalmarHighest
	}
	drawdownDiff := highestPrice.Sub(lowestPrice).Div(highestPrice)
	if drawdownDiff.IsZero() {
		return decimal.Zero, nil
	}
	return average.Sub(riskFreeRateForPeriod).Div(drawdownDiff), nil
}

// DecimalInformationRatio The information ratio (IR) is a measurement of portfolio returns beyond the returns of a benchmark,
// usually an index, compared to the volatility of those returns.
// The benchmark used is typically an index that represents the market or a particular sector or industry.
func DecimalInformationRatio(returnsRates, benchmarkRates []decimal.Decimal, averageValues, averageComparison decimal.Decimal) (decimal.Decimal, error) {
	if len(benchmarkRates) != len(returnsRates) {
		return decimal.Zero, errInformationBadLength
	}
	diffs := make([]decimal.Decimal, len(returnsRates))
	for i := range returnsRates {
		diffs[i] = returnsRates[i].Sub(benchmarkRates[i])
	}
	stdDev, err := DecimalPopulationStandardDeviation(diffs)
	if err != nil && !errors.Is(err, ErrInexactConversion) {
		return decimal.Zero, err
	}
	if stdDev.IsZero() {
		return decimal.Zero, nil
	}
	return averageValues.Sub(averageComparison).Div(stdDev), nil
}

// DecimalPopulationStandardDeviation calculates standard deviation using population based calculation
func DecimalPopulationStandardDeviation(values []decimal.Decimal) (decimal.Decimal, error) {
	if len(values) < 2 {
		return decimal.Zero, nil
	}
	valAvg, err := DecimalArithmeticMean(values)
	if err != nil {
		return decimal.Zero, err
	}
	diffs := make([]decimal.Decimal, len(values))
	for x := range values {
		val := values[x].Sub(valAvg)
		exp := two
		pow := DecimalPow(val, exp)
		diffs[x] = pow
	}
	var diffAvg decimal.Decimal
	diffAvg, err = DecimalArithmeticMean(diffs)
	if err != nil {
		return decimal.Zero, err
	}
	f, exact := diffAvg.Float64()
	err = nil
	if !exact {
		err = fmt.Errorf("%w from %v to %v", ErrInexactConversion, diffAvg, f)
	}
	resp := decimal.NewFromFloat(math.Sqrt(f))
	return resp, err
}

// DecimalSampleStandardDeviation standard deviation is a statistic that
// measures the dispersion of a dataset relative to its mean and
// is calculated as the square root of the variance
func DecimalSampleStandardDeviation(values []decimal.Decimal) (decimal.Decimal, error) {
	if len(values) < 2 {
		return decimal.Zero, nil
	}
	mean, err := DecimalArithmeticMean(values)
	if err != nil {
		return decimal.Zero, err
	}
	superMean := make([]decimal.Decimal, len(values))
	var combined decimal.Decimal
	for i := range values {
		pow := values[i].Sub(mean).Pow(two)
		superMean[i] = pow
		combined.Add(pow)
	}
	avg := combined.Div(decimal.NewFromInt(int64(len(superMean))).Sub(one))
	f, exact := avg.Float64()
	err = nil
	if !exact {
		err = fmt.Errorf("%w from %v to %v", ErrInexactConversion, avg, f)
	}
	sqrt := math.Sqrt(f)
	return decimal.NewFromFloat(sqrt), err
}

// DecimalGeometricMean is an average which indicates the central tendency or
// typical value of a set of numbers by using the product of their values
// The geometric average can only process positive numbers
func DecimalGeometricMean(values []decimal.Decimal) (decimal.Decimal, error) {
	if len(values) == 0 {
		return decimal.Zero, errZeroValue
	}
	product := one
	for i := range values {
		if values[i].LessThanOrEqual(decimal.Zero) {
			// cannot use negative or zero values in geometric calculation
			return decimal.Zero, errGeometricNegative
		}
		product = product.Mul(values[i])
	}
	exp := one.Div(decimal.NewFromInt(int64(len(values))))
	pow := DecimalPow(product, exp)
	geometricPower := pow
	return geometricPower, nil
}

// DecimalPow is lovely because shopspring decimal cannot
// handle ^0.x and instead returns 1
func DecimalPow(x, y decimal.Decimal) decimal.Decimal {
	pow := math.Pow(x.InexactFloat64(), y.InexactFloat64())
	if math.IsNaN(pow) || math.IsInf(pow, 0) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(pow)
}

// DecimalFinancialGeometricMean is a modified geometric average to assess
// the negative returns of investments. It accepts It adds +1 to each
// This does impact the final figures as it is modifying values
// It is still ultimately calculating a geometric average
// which should only be compared to other financial geometric averages
func DecimalFinancialGeometricMean(values []decimal.Decimal) (decimal.Decimal, error) {
	if len(values) == 0 {
		return decimal.Zero, errZeroValue
	}
	product := 1.0
	for i := range values {
		if values[i].LessThan(decimal.NewFromInt(-1)) {
			// cannot lose more than 100%, figures are incorrect
			// losing exactly 100% will return a 0 value, but is not an error
			return decimal.Zero, errNegativeValueOutOfRange
		}
		// as we cannot have negative or zero value geometric numbers
		// adding a 1 to the percentage movements allows for differentiation between
		// negative numbers (eg -0.1 translates to 0.9) and positive numbers (eg 0.1 becomes 1.1)
		modVal := values[i].Add(one).InexactFloat64()
		product *= modVal
	}
	prod := 1 / float64(len(values))
	geometricPower := math.Pow(product, prod)
	if geometricPower > 0 {
		// we minus 1 because we manipulated the values to be non-zero/negative
		geometricPower--
	}
	return decimal.NewFromFloat(geometricPower), nil
}

// DecimalArithmeticMean is the basic form of calculating an average.
// Divide the sum of all values by the length of values
func DecimalArithmeticMean(values []decimal.Decimal) (decimal.Decimal, error) {
	if len(values) == 0 {
		return decimal.Zero, errZeroValue
	}
	var sumOfValues decimal.Decimal
	for x := range values {
		sumOfValues = sumOfValues.Add(values[x])
	}
	return sumOfValues.Div(decimal.NewFromInt(int64(len(values)))), nil
}

// DecimalSortinoRatio returns sortino ratio of backtest compared to risk-free
func DecimalSortinoRatio(movementPerCandle []decimal.Decimal, riskFreeRatePerInterval, average decimal.Decimal) (decimal.Decimal, error) {
	if len(movementPerCandle) == 0 {
		return decimal.Zero, errZeroValue
	}
	totalNegativeResultsSquared := decimal.Zero
	for x := range movementPerCandle {
		if movementPerCandle[x].Sub(riskFreeRatePerInterval).LessThan(decimal.Zero) {
			totalNegativeResultsSquared = totalNegativeResultsSquared.Add(movementPerCandle[x].Sub(riskFreeRatePerInterval).Pow(two))
		}
	}
	if totalNegativeResultsSquared.IsZero() {
		return decimal.Zero, ErrNoNegativeResults
	}
	f, exact := totalNegativeResultsSquared.Float64()
	var err error
	if !exact {
		err = fmt.Errorf("%w from %v to %v", ErrInexactConversion, totalNegativeResultsSquared, f)
	}
	fAverageDownsideDeviation := math.Sqrt(f / float64(len(movementPerCandle)))
	averageDownsideDeviation := decimal.NewFromFloat(fAverageDownsideDeviation)

	return average.Sub(riskFreeRatePerInterval).Div(averageDownsideDeviation), err
}

// DecimalSharpeRatio returns sharpe ratio of backtest compared to risk-free
func DecimalSharpeRatio(movementPerCandle []decimal.Decimal, riskFreeRatePerInterval, average decimal.Decimal) (decimal.Decimal, error) {
	totalIntervals := decimal.NewFromInt(int64(len(movementPerCandle)))
	if totalIntervals.IsZero() {
		return decimal.Zero, errZeroValue
	}
	excessReturns := make([]decimal.Decimal, len(movementPerCandle))
	for i := range movementPerCandle {
		excessReturns[i] = movementPerCandle[i].Sub(riskFreeRatePerInterval)
	}
	standardDeviation, err := DecimalPopulationStandardDeviation(excessReturns)
	if err != nil && !errors.Is(err, ErrInexactConversion) {
		return decimal.Zero, err
	}
	if standardDeviation.IsZero() {
		return decimal.Zero, nil
	}

	return average.Sub(riskFreeRatePerInterval).Div(standardDeviation), nil
}

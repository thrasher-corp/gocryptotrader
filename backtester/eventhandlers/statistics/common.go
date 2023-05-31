package statistics

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// fSIL shorthand wrapper for FitStringToLimit
func fSIL(str string, limit int) string {
	spacer := " "
	return common.FitStringToLimit(str, spacer, limit, true)
}

// CalculateBiggestEventDrawdown calculates the biggest drawdown using a slice of DataEvents
func CalculateBiggestEventDrawdown(closePrices []data.Event) (Swing, error) {
	if len(closePrices) == 0 {
		return Swing{}, fmt.Errorf("%w to calculate drawdowns", errReceivedNoData)
	}
	var swings []Swing
	lowestPrice := closePrices[0].GetLowPrice()
	highestPrice := closePrices[0].GetHighPrice()
	lowestTime := closePrices[0].GetTime()
	highestTime := closePrices[0].GetTime()
	interval := closePrices[0].GetInterval()

	for i := range closePrices {
		currHigh := closePrices[i].GetHighPrice()
		currLow := closePrices[i].GetLowPrice()
		currTime := closePrices[i].GetTime()
		if lowestPrice.GreaterThan(currLow) && !currLow.IsZero() {
			lowestPrice = currLow
			lowestTime = currTime
		}
		if highestPrice.LessThan(currHigh) {
			if lowestTime.Equal(highestTime) {
				// create distinction if the greatest drawdown occurs within the same candle
				lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
			}
			intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[i].GetInterval(), 0)
			if err != nil {
				return Swing{}, fmt.Errorf("cannot calculate max drawdown, date range error: %w", err)
			}
			if highestPrice.IsPositive() && lowestPrice.IsPositive() {
				swings = append(swings, Swing{
					Highest: ValueAtTime{
						Time:  highestTime,
						Value: highestPrice,
					},
					Lowest: ValueAtTime{
						Time:  lowestTime,
						Value: lowestPrice,
					},
					DrawdownPercent:  lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100)),
					IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
				})
			}
			// reset the drawdown
			highestPrice = currHigh
			highestTime = currTime
			lowestPrice = currLow
			lowestTime = currTime
		}
	}
	if (len(swings) > 0 && swings[len(swings)-1].Lowest.Value != closePrices[len(closePrices)-1].GetLowPrice()) || swings == nil {
		// need to close out the final drawdown
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
		}
		intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[0].GetInterval(), 0)
		if err != nil {
			return Swing{}, fmt.Errorf("cannot close out max drawdown calculation: %w", err)
		}
		drawdownPercent := decimal.Zero
		if highestPrice.GreaterThan(decimal.Zero) {
			drawdownPercent = lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100))
		}
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
		}
		swings = append(swings, Swing{
			Highest: ValueAtTime{
				Time:  highestTime,
				Value: highestPrice,
			},
			Lowest: ValueAtTime{
				Time:  lowestTime,
				Value: lowestPrice,
			},
			DrawdownPercent:  drawdownPercent,
			IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
		})
	}

	var maxDrawdown Swing
	if len(swings) > 0 {
		maxDrawdown = swings[0]
	}
	for i := range swings {
		if swings[i].DrawdownPercent.LessThan(maxDrawdown.DrawdownPercent) {
			maxDrawdown = swings[i]
		}
	}

	return maxDrawdown, nil
}

// CalculateBiggestValueAtTimeDrawdown calculates the biggest drawdown using a slice of ValueAtTimes
func CalculateBiggestValueAtTimeDrawdown(closePrices []ValueAtTime, interval gctkline.Interval) (Swing, error) {
	if len(closePrices) == 0 {
		return Swing{}, fmt.Errorf("%w to calculate drawdowns", errReceivedNoData)
	}
	var swings []Swing
	lowestPrice := closePrices[0].Value
	highestPrice := closePrices[0].Value
	lowestTime := closePrices[0].Time
	highestTime := closePrices[0].Time

	for i := range closePrices {
		currHigh := closePrices[i].Value
		currLow := closePrices[i].Value
		currTime := closePrices[i].Time
		if lowestPrice.GreaterThan(currLow) && !currLow.IsZero() {
			lowestPrice = currLow
			lowestTime = currTime
		}
		if highestPrice.LessThan(currHigh) && highestPrice.IsPositive() {
			if lowestTime.Equal(highestTime) {
				// create distinction if the greatest drawdown occurs within the same candle
				lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
			}
			intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, interval, 0)
			if err != nil {
				return Swing{}, err
			}
			swings = append(swings, Swing{
				Highest: ValueAtTime{
					Time:  highestTime,
					Value: highestPrice,
				},
				Lowest: ValueAtTime{
					Time:  lowestTime,
					Value: lowestPrice,
				},
				DrawdownPercent:  lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100)),
				IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
			})
			// reset the drawdown
			highestPrice = currHigh
			highestTime = currTime
			lowestPrice = currLow
			lowestTime = currTime
		}
	}
	if (len(swings) > 0 && !swings[len(swings)-1].Lowest.Value.Equal(closePrices[len(closePrices)-1].Value)) || swings == nil {
		// need to close out the final drawdown
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
		}
		intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, interval, 0)
		if err != nil {
			log.Errorln(common.CurrencyStatistics, err)
		}
		drawdownPercent := decimal.Zero
		if highestPrice.GreaterThan(decimal.Zero) {
			drawdownPercent = lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100))
		}
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add(interval.Duration() - time.Nanosecond)
		}
		swings = append(swings, Swing{
			Highest: ValueAtTime{
				Time:  highestTime,
				Value: highestPrice,
			},
			Lowest: ValueAtTime{
				Time:  lowestTime,
				Value: lowestPrice,
			},
			DrawdownPercent:  drawdownPercent,
			IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
		})
	}

	var maxDrawdown Swing
	if len(swings) > 0 {
		maxDrawdown = swings[0]
	}
	for i := range swings {
		if swings[i].DrawdownPercent.LessThan(maxDrawdown.DrawdownPercent) {
			maxDrawdown = swings[i]
		}
	}

	return maxDrawdown, nil
}

// CalculateRatios creates arithmetic and geometric ratios from funding or currency pair data
func CalculateRatios(benchmarkRates, returnsPerCandle []decimal.Decimal, riskFreeRatePerCandle decimal.Decimal, maxDrawdown *Swing, logMessage string) (arithmeticStats, geometricStats *Ratios, err error) {
	var arithmeticBenchmarkAverage, geometricBenchmarkAverage decimal.Decimal
	arithmeticBenchmarkAverage, err = gctmath.DecimalArithmeticMean(benchmarkRates)
	if err != nil {
		return nil, nil, err
	}
	geometricBenchmarkAverage, err = gctmath.DecimalFinancialGeometricMean(benchmarkRates)
	if err != nil {
		return nil, nil, err
	}

	riskFreeRateForPeriod := riskFreeRatePerCandle.Mul(decimal.NewFromInt(int64(len(benchmarkRates))))

	var arithmeticReturnsPerCandle, geometricReturnsPerCandle, arithmeticSharpe, arithmeticSortino,
		arithmeticInformation, arithmeticCalmar, geomSharpe, geomSortino, geomInformation, geomCalmar decimal.Decimal

	arithmeticReturnsPerCandle, err = gctmath.DecimalArithmeticMean(returnsPerCandle)
	if err != nil {
		return nil, nil, err
	}
	geometricReturnsPerCandle, err = gctmath.DecimalFinancialGeometricMean(returnsPerCandle)
	if err != nil {
		return nil, nil, err
	}

	arithmeticSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil {
		return nil, nil, err
	}
	arithmeticSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(common.Statistics, "%s funding arithmetic sortino ratio %v", logMessage, err)
		} else {
			return nil, nil, err
		}
	}
	arithmeticInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, arithmeticReturnsPerCandle, arithmeticBenchmarkAverage)
	if err != nil {
		return nil, nil, err
	}
	arithmeticCalmar, err = gctmath.DecimalCalmarRatio(maxDrawdown.Highest.Value, maxDrawdown.Lowest.Value, arithmeticReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		log.Warnf(common.Statistics, "%s funding arithmetic calmar ratio %v", logMessage, err)
	}

	arithmeticStats = &Ratios{}
	if !arithmeticSharpe.IsZero() {
		arithmeticStats.SharpeRatio = arithmeticSharpe
	}
	if !arithmeticSortino.IsZero() {
		arithmeticStats.SortinoRatio = arithmeticSortino
	}
	if !arithmeticInformation.IsZero() {
		arithmeticStats.InformationRatio = arithmeticInformation
	}
	if !arithmeticCalmar.IsZero() {
		arithmeticStats.CalmarRatio = arithmeticCalmar
	}

	geomSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil {
		return nil, nil, err
	}
	geomSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(common.Statistics, "%s geometric sortino ratio %v", logMessage, err)
		} else {
			return nil, nil, err
		}
	}
	geomInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, geometricReturnsPerCandle, geometricBenchmarkAverage)
	if err != nil {
		return nil, nil, err
	}
	geomCalmar, err = gctmath.DecimalCalmarRatio(maxDrawdown.Highest.Value, maxDrawdown.Lowest.Value, geometricReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		log.Warnf(common.Statistics, "%s funding geometric calmar ratio %v", logMessage, err)
	}
	geometricStats = &Ratios{}
	if !arithmeticSharpe.IsZero() {
		geometricStats.SharpeRatio = geomSharpe
	}
	if !arithmeticSortino.IsZero() {
		geometricStats.SortinoRatio = geomSortino
	}
	if !arithmeticInformation.IsZero() {
		geometricStats.InformationRatio = geomInformation
	}
	if !arithmeticCalmar.IsZero() {
		geometricStats.CalmarRatio = geomCalmar
	}

	return arithmeticStats, geometricStats, nil
}

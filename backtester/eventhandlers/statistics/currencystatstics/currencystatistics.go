package currencystatstics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// CalculateResults calculates all statistics for the exchange, asset, currency pair
func (c *CurrencyStatistic) CalculateResults() {
	first := c.Events[0]
	firstPrice := first.DataEvent.ClosePrice()
	last := c.Events[len(c.Events)-1]
	lastPrice := last.DataEvent.ClosePrice()
	for i := range last.Transactions.Orders {
		if last.Transactions.Orders[i].Side == gctorder.Buy {
			c.BuyOrders++
		} else if last.Transactions.Orders[i].Side == gctorder.Sell {
			c.SellOrders++
		}
	}
	for i := range c.Events {
		price := c.Events[i].DataEvent.ClosePrice()
		if c.LowestClosePrice == 0 || price < c.LowestClosePrice {
			c.LowestClosePrice = price
		}
		if c.HighestClosePrice == 0 || price > c.HighestClosePrice {
			c.HighestClosePrice = price
		}
	}
	c.MarketMovement = ((lastPrice - firstPrice) / firstPrice) * 100
	c.StrategyMovement = ((last.Holdings.TotalValue - last.Holdings.InitialFunds) / last.Holdings.InitialFunds) * 100
	c.RiskFreeRate = last.Holdings.RiskFreeRate * 100
	returnPerCandle := make([]float64, len(c.Events))

	var negativeReturns []float64
	for i := range c.Events {
		returnPerCandle[i] = c.Events[i].Holdings.ChangeInTotalValuePercent
		if c.Events[i].Holdings.ChangeInTotalValuePercent < 0 {
			negativeReturns = append(negativeReturns, c.Events[i].Holdings.ChangeInTotalValuePercent)
		}
	}
	var allDataEvents []common.DataEventHandler
	for i := range c.Events {
		allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
	}
	c.MaxDrawdown = calculateMaxDrawdown(allDataEvents)
	c.SharpeRatio = calculateSharpeRatio(returnPerCandle, last.Holdings.RiskFreeRate)
	c.SortinoRatio = calculateSortinoRatio(returnPerCandle, negativeReturns, last.Holdings.RiskFreeRate)
	c.InformationRatio = calculateInformationRatio(returnPerCandle, []float64{last.Holdings.RiskFreeRate})
	c.CalmarRatio = calculateCalmarRatio(returnPerCandle, &c.MaxDrawdown)
	c.CompoundAnnualGrowthRate = calculateCompoundAnnualGrowthRate(
		last.Holdings.InitialFunds,
		last.Holdings.TotalValue,
		first.DataEvent.GetTime(),
		last.DataEvent.GetTime(),
		first.DataEvent.GetInterval())
}

// PrintResults outputs all calculated statistics to the command line
func (c *CurrencyStatistic) PrintResults(e string, a asset.Item, p currency.Pair) {
	var errs gctcommon.Errors
	sort.Slice(c.Events, func(i, j int) bool {
		return c.Events[i].DataEvent.GetTime().Before(c.Events[j].DataEvent.GetTime())
	})
	last := c.Events[len(c.Events)-1]
	first := c.Events[0]
	c.StartingClosePrice = first.DataEvent.ClosePrice()
	c.EndingClosePrice = last.DataEvent.ClosePrice()
	c.TotalOrders = c.BuyOrders + c.SellOrders
	last.Holdings.TotalValueLost = last.Holdings.TotalValueLostToSlippage + last.Holdings.TotalValueLostToVolumeSizing
	currStr := fmt.Sprintf("------------------Stats for %v %v %v-------------------------", e, a, p)
	log.Infof(log.BackTester, currStr[:61])
	log.Infof(log.BackTester, "Initial funds: $%v\n\n", last.Holdings.InitialFunds)

	log.Infof(log.BackTester, "Buy orders: %v", c.BuyOrders)
	log.Infof(log.BackTester, "Buy value: %v", last.Holdings.BoughtValue)
	log.Infof(log.BackTester, "Buy amount: %v", last.Holdings.BoughtAmount)
	log.Infof(log.BackTester, "Sell orders: %v", c.SellOrders)
	log.Infof(log.BackTester, "Sell value: %v", last.Holdings.SoldValue)
	log.Infof(log.BackTester, "Sell amount: %v", last.Holdings.SoldAmount)
	log.Infof(log.BackTester, "Total orders: %v\n\n", c.TotalOrders)

	log.Info(log.BackTester, "------------------Max Drawdown-------------------------------")
	log.Infof(log.BackTester, "Highest Price: $%.2f", c.MaxDrawdown.Highest.Price)
	log.Infof(log.BackTester, "Highest Price Time: %v", c.MaxDrawdown.Highest.Time)
	log.Infof(log.BackTester, "Lowest Price: $%v", c.MaxDrawdown.Lowest.Price)
	log.Infof(log.BackTester, "Lowest Price Time: %v", c.MaxDrawdown.Lowest.Time)
	log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", c.MaxDrawdown.DrawdownPercent)
	log.Infof(log.BackTester, "Difference: $%.2f", c.MaxDrawdown.Highest.Price-c.MaxDrawdown.Lowest.Price)
	log.Infof(log.BackTester, "Drawdown length: %v", c.MaxDrawdown.IntervalDuration)

	log.Info(log.BackTester, "------------------Ratios-------------------------------------")
	log.Infof(log.BackTester, "Risk free rate: %.3f", c.RiskFreeRate)
	log.Infof(log.BackTester, "Sharpe ratio: %.8f", c.SharpeRatio)
	log.Infof(log.BackTester, "Sortino ratio: %.3f", c.SortinoRatio)
	log.Infof(log.BackTester, "Information ratio: %.3f", c.InformationRatio)
	log.Infof(log.BackTester, "Calmar ratio: %.3f", c.CalmarRatio)
	log.Infof(log.BackTester, "Compound Annual Growth Rate: %.2f\n\n", c.CompoundAnnualGrowthRate)

	log.Info(log.BackTester, "------------------Results------------------------------------")
	log.Infof(log.BackTester, "Starting Close Price: $%v", c.StartingClosePrice)
	log.Infof(log.BackTester, "Finishing Close Price: $%v", c.EndingClosePrice)
	log.Infof(log.BackTester, "Lowest Close Price: $%v", c.LowestClosePrice)
	log.Infof(log.BackTester, "Highest Close Price: $%v", c.HighestClosePrice)

	log.Infof(log.BackTester, "Market movement: %v%%", c.MarketMovement)
	log.Infof(log.BackTester, "Strategy movement: %v%%", c.StrategyMovement)
	log.Infof(log.BackTester, "Did it beat the market: %v", c.StrategyMovement > c.MarketMovement)

	log.Infof(log.BackTester, "Value lost to volume sizing: $%v", last.Holdings.TotalValueLostToVolumeSizing)
	log.Infof(log.BackTester, "Value lost to slippage: $%v", last.Holdings.TotalValueLostToSlippage)
	log.Infof(log.BackTester, "Total Value lost: $%v", last.Holdings.TotalValueLost)
	log.Infof(log.BackTester, "Total Fees: $%v\n\n", last.Holdings.TotalFees)

	log.Infof(log.BackTester, "Final funds: $%v", last.Holdings.RemainingFunds)
	log.Infof(log.BackTester, "Final holdings: %v", last.Holdings.PositionsSize)
	log.Infof(log.BackTester, "Final holdings value: $%v", last.Holdings.PositionsValue)
	log.Infof(log.BackTester, "Final total value: $%v\n\n", last.Holdings.TotalValue)

	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}
}

func calculateMaxDrawdown(closePrices []common.DataEventHandler) Swing {
	var lowestPrice, highestPrice float64
	var lowestTime, highestTime time.Time
	var swings []Swing
	if len(closePrices) > 0 {
		lowestPrice = closePrices[0].LowPrice()
		highestPrice = closePrices[0].HighPrice()
		lowestTime = closePrices[0].GetTime()
		highestTime = closePrices[0].GetTime()
	}
	for i := range closePrices {
		currHigh := closePrices[i].HighPrice()
		currLow := closePrices[i].LowPrice()
		currTime := closePrices[i].GetTime()
		if lowestPrice > currLow {
			lowestPrice = currLow
			lowestTime = currTime
		}
		if highestPrice < currHigh {
			intervals := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[i].GetInterval(), 0)
			swings = append(swings, Swing{
				Highest: Iteration{
					Time:  highestTime,
					Price: highestPrice,
				},
				Lowest: Iteration{
					Time:  lowestTime,
					Price: lowestPrice,
				},
				DrawdownPercent:  ((lowestPrice - highestPrice) / highestPrice) * 100,
				IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
			})
			// reset the drawdown
			highestPrice = currHigh
			highestTime = currTime
			lowestPrice = currLow
			lowestTime = currTime
		}
	}
	if (len(swings) > 0 && swings[len(swings)-1].Lowest.Price != closePrices[len(closePrices)-1].LowPrice()) || swings == nil {
		// need to close out the final drawdown
		intervals := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[0].GetInterval(), 0)
		swings = append(swings, Swing{
			Highest: Iteration{
				Time:  highestTime,
				Price: highestPrice,
			},
			Lowest: Iteration{
				Time:  lowestTime,
				Price: lowestPrice,
			},
			DrawdownPercent:  ((lowestPrice - highestPrice) / highestPrice) * 100,
			IntervalDuration: int64(len(intervals.Ranges[0].Intervals)),
		})
	}

	var maxDrawdown Swing
	if len(swings) > 0 {
		maxDrawdown = swings[0]
	}
	for i := range swings {
		if swings[i].DrawdownPercent < maxDrawdown.DrawdownPercent {
			// drawdowns are negative
			maxDrawdown = swings[i]
		}
	}

	return maxDrawdown
}

func calculateCompoundAnnualGrowthRate(openValue, closeValue float64, start, end time.Time, interval gctkline.Interval) float64 {
	p := gctkline.TotalCandlesPerInterval(start, end, interval)

	k := math.Pow(closeValue/openValue, 1/float64(p)) - 1
	return k * 100
}

func calculateCalmarRatio(values []float64, maxDrawdown *Swing) float64 {
	if maxDrawdown == nil {
		return 0
	}
	avg := calculateTheAverage(values)
	drawdownDiff := (maxDrawdown.Highest.Price - maxDrawdown.Lowest.Price) / maxDrawdown.Highest.Price
	if drawdownDiff == 0 {
		return 0
	}
	return avg / drawdownDiff
}

func calculateInformationRatio(values, riskFreeRates []float64) float64 {
	if len(riskFreeRates) == 1 {
		for i := range values {
			if i == 0 {
				continue
			}
			riskFreeRates = append(riskFreeRates, riskFreeRates[0])
		}
	}
	avgValue := calculateTheAverage(values)
	avgComparison := calculateTheAverage(riskFreeRates)
	var diffs []float64
	for i := range values {
		diffs = append(diffs, values[i]-riskFreeRates[i])
	}
	stdDev := calculateStandardDeviation(diffs)
	if stdDev == 0 {
		return 0
	}
	return (avgValue - avgComparison) / stdDev
}

func calculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := calculateTheAverage(values)

	diffs := make([]float64, len(values))
	for x := range values {
		diffs[x] = math.Pow(values[x]-avg, 2)
	}
	return math.Sqrt(calculateTheAverage(diffs))
}

// calculateSampleStandardDeviation is used in sharpe ratio calculations
// calculates the sample rate standard deviation
func calculateSampleStandardDeviation(vals []float64) float64 {
	if len(vals) <= 1 {
		return 0
	}
	mean := calculateTheAverage(vals)
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

func calculateTheAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sumOfValues float64
	for x := range values {
		sumOfValues += values[x]
	}
	return sumOfValues / float64(len(values))
}

// calculateSortinoRatio returns sortino ratio of backtest compared to risk-free
func calculateSortinoRatio(movementPerCandle, excessMovement []float64, riskFreeRate float64) float64 {
	mean := calculateTheAverage(movementPerCandle)
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

// calculateSharpeRatio returns sharpe ratio of backtest compared to risk-free
func calculateSharpeRatio(movementPerCandle []float64, riskFreeRate float64) float64 {
	if len(movementPerCandle) <= 1 {
		return 0
	}
	mean := calculateTheAverage(movementPerCandle)
	standardDeviation := calculateSampleStandardDeviation(movementPerCandle)

	if standardDeviation == 0 {
		return 0
	}
	return (mean - riskFreeRate) / standardDeviation
}

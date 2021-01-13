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

func calculateCompoundAnnualGrowthRate(openValue, closeValue float64, start, end time.Time, interval gctkline.Interval) float64 {
	p := gctkline.TotalCandlesPerInterval(start, end, interval)
	return math.Pow(closeValue/openValue, 1/float64(p)) - 1
}

func calculateCalmarRatio(values []float64, maxDrawdown Swing) float64 {
	avg := calculateTheAverage(values)
	drawdownDiff := (maxDrawdown.Highest.Price - maxDrawdown.Lowest.Price) / maxDrawdown.Highest.Price
	if drawdownDiff == 0 {
		return 0
	}
	ratio := avg / drawdownDiff
	return ratio
}

func calculateInformationRatio(values []float64, riskFreeRates []float64) float64 {
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
	ratio := (avgValue - avgComparison) / stdDev
	return ratio
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
	for i := range vals {
		result := math.Pow(vals[i]-mean, 2)
		superMean = append(superMean, result)
	}

	var combined float64
	for i := range superMean {
		combined += superMean[i]
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
	avg := sumOfValues / float64(len(values))
	return avg
}

// calculateSortinoRatio returns sortino ratio of backtest compared to risk-free
func calculateSortinoRatio(movementPerCandle []float64, excessMovement []float64, riskFreeRate float64) float64 {
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

func (c *CurrencyStatistic) CalculateResults() {
	first := c.Events[0]
	var firstPrice float64
	firstPrice = first.SignalEvent.GetPrice()
	last := c.Events[len(c.Events)-1]
	var lastPrice float64
	lastPrice = last.SignalEvent.GetPrice()
	for i := range last.Transactions.Orders {
		if last.Transactions.Orders[i].Side == gctorder.Buy {
			c.BuyOrders++
		} else if last.Transactions.Orders[i].Side == gctorder.Sell {
			c.SellOrders++
		}
	}
	for i := range c.Events {
		price := c.Events[i].DataEvent.Price()
		if c.LowestClosePrice == 0 || price < c.LowestClosePrice {
			c.LowestClosePrice = price
		}
		if c.HighestClosePrice == 0 || price > c.HighestClosePrice {
			c.HighestClosePrice = price
		}
	}
	c.MarketMovement = ((lastPrice - firstPrice) / firstPrice) * 100
	c.StrategyMovement = ((last.Holdings.TotalValue - last.Holdings.InitialFunds) / last.Holdings.InitialFunds) * 100
	c.RiskFreeRate = last.Holdings.RiskFreeRate
	var returnPerCandle = make([]float64, len(c.Events))

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
	c.DrawDowns = calculateAllDrawDowns(allDataEvents)
	c.SharpeRatio = calculateSharpeRatio(returnPerCandle, c.RiskFreeRate)
	c.SortinoRatio = calculateSortinoRatio(returnPerCandle, negativeReturns, c.RiskFreeRate)
	c.InformationRatio = calculateInformationRatio(returnPerCandle, []float64{c.RiskFreeRate})
	c.CalamariRatio = calculateCalmarRatio(returnPerCandle, c.DrawDowns.MaxDrawDown)
	c.CompoundAnnualGrowthRate = calculateCompoundAnnualGrowthRate(
		last.Holdings.InitialFunds,
		last.Holdings.TotalValue,
		first.DataEvent.GetTime(),
		last.DataEvent.GetTime(),
		first.DataEvent.GetInterval())
}

func (c *CurrencyStatistic) PrintResults(e string, a asset.Item, p currency.Pair) {
	var errs gctcommon.Errors
	sort.Slice(c.Events, func(i, j int) bool {
		return c.Events[i].DataEvent.GetTime().Before(c.Events[j].DataEvent.GetTime())
	})
	last := c.Events[len(c.Events)-1]
	first := c.Events[0]
	c.StartingClosePrice = first.DataEvent.Price()
	c.EndingClosePrice = last.DataEvent.Price()
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
	log.Infof(log.BackTester, "Highest Price: $%.2f", c.DrawDowns.MaxDrawDown.Highest.Price)
	log.Infof(log.BackTester, "Highest Price Time: %v", c.DrawDowns.MaxDrawDown.Highest.Time)
	log.Infof(log.BackTester, "Lowest Price: $%v", c.DrawDowns.MaxDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Lowest Price Time: %v", c.DrawDowns.MaxDrawDown.Lowest.Time)
	log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", c.DrawDowns.MaxDrawDown.CalculatedDrawDown)
	log.Infof(log.BackTester, "Difference: $%.2f", c.DrawDowns.MaxDrawDown.Highest.Price-c.DrawDowns.MaxDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Drawdown length: %v", len(c.DrawDowns.MaxDrawDown.Iterations))

	log.Info(log.BackTester, "------------------Longest Drawdown---------------------------")
	log.Infof(log.BackTester, "Highest Price: $%.2f", c.DrawDowns.LongestDrawDown.Highest.Price)
	log.Infof(log.BackTester, "Highest Price Time: %v", c.DrawDowns.LongestDrawDown.Highest.Time)
	log.Infof(log.BackTester, "Lowest Price: $%.2f", c.DrawDowns.LongestDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Lowest Price Time: %v", c.DrawDowns.LongestDrawDown.Lowest.Time)
	log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", c.DrawDowns.LongestDrawDown.CalculatedDrawDown)
	log.Infof(log.BackTester, "Difference: $%.2f", c.DrawDowns.LongestDrawDown.Highest.Price-c.DrawDowns.LongestDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Drawdown length: %v\n\n", len(c.DrawDowns.LongestDrawDown.Iterations))

	log.Info(log.BackTester, "------------------Ratios-------------------------------------")
	log.Infof(log.BackTester, "Risk free rate: %.3f", c.RiskFreeRate)
	log.Infof(log.BackTester, "Sharpe ratio: %.8f", c.SharpeRatio)
	log.Infof(log.BackTester, "Sortino ratio: %.3f", c.SortinoRatio)
	log.Infof(log.BackTester, "Information ratio: %.3f", c.InformationRatio)
	log.Infof(log.BackTester, "Calmar ratio: %.3f", c.CalamariRatio)
	log.Infof(log.BackTester, "Compound Annual Growth Rate: %.2f\n\n", c.CalamariRatio)

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

func (c *CurrencyStatistic) MaxDrawdown() Swing {
	if len(c.DrawDowns.MaxDrawDown.Iterations) == 0 {
		var allDataEvents []common.DataEventHandler
		for i := range c.Events {
			allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
		}
		c.DrawDowns = calculateAllDrawDowns(allDataEvents)
	}
	return c.DrawDowns.MaxDrawDown
}

func (c *CurrencyStatistic) LongestDrawdown() Swing {
	if len(c.DrawDowns.LongestDrawDown.Iterations) == 0 {
		var allDataEvents []common.DataEventHandler
		for i := range c.Events {
			allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
		}
		c.DrawDowns = calculateAllDrawDowns(allDataEvents)
	}
	return c.DrawDowns.LongestDrawDown
}

func calculateAllDrawDowns(closePrices []common.DataEventHandler) SwingHolder {
	isDrawingDown := false

	var response SwingHolder
	var activeDraw Swing
	for i := range closePrices {
		p := closePrices[i].Price()
		t := closePrices[i].GetTime()
		if i == 0 {
			activeDraw.Highest = Iteration{
				Price: p,
				Time:  t,
			}
			activeDraw.Lowest = Iteration{
				Price: p,
				Time:  t,
			}
			continue
		}

		// create
		if !isDrawingDown && activeDraw.Highest.Price > p {
			isDrawingDown = true
			activeDraw = Swing{
				Highest: Iteration{
					Price: p,
					Time:  t,
				},
				Lowest: Iteration{
					Price: p,
					Time:  t,
				},
			}
		}

		// close
		if isDrawingDown && activeDraw.Lowest.Price < p {
			activeDraw.Lowest = Iteration{
				Price: activeDraw.Iterations[len(activeDraw.Iterations)-1].Price,
				Time:  activeDraw.Iterations[len(activeDraw.Iterations)-1].Time,
			}
			isDrawingDown = false
			response.DrawDowns = append(response.DrawDowns, activeDraw)
			// reset
			activeDraw = Swing{
				Highest: Iteration{
					Price: p,
					Time:  t,
				},
				Lowest: Iteration{
					Price: p,
					Time:  t,
				},
			}
		}

		// append
		if isDrawingDown {
			if p < activeDraw.Lowest.Price {
				activeDraw.Lowest.Price = p
				activeDraw.Lowest.Time = t
			}
			activeDraw.Iterations = append(activeDraw.Iterations, Iteration{
				Time:  t,
				Price: p,
			})
		}
	}

	response.calculateMaxAndLongestDrawDowns()
	response.MaxDrawDown.CalculatedDrawDown = ((response.MaxDrawDown.Lowest.Price - response.MaxDrawDown.Highest.Price) / response.MaxDrawDown.Highest.Price) * 100
	response.LongestDrawDown.CalculatedDrawDown = ((response.LongestDrawDown.Lowest.Price - response.LongestDrawDown.Highest.Price) / response.LongestDrawDown.Highest.Price) * 100

	return response
}

func (s *SwingHolder) calculateMaxAndLongestDrawDowns() {
	for i := range s.DrawDowns {
		if s.DrawDowns[i].Highest.Price-s.DrawDowns[i].Lowest.Price > s.MaxDrawDown.Highest.Price-s.MaxDrawDown.Lowest.Price {
			s.MaxDrawDown = s.DrawDowns[i]
		}
		if len(s.DrawDowns[i].Iterations) > len(s.LongestDrawDown.Iterations) {
			s.LongestDrawDown = s.DrawDowns[i]
		}
	}
}

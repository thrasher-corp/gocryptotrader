package statistics

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// CalculateResults calculates all statistics for the exchange, asset, currency pair
func (c *CurrencyPairStatistic) CalculateResults(riskFreeRate decimal.Decimal) error {
	var errs gctcommon.Errors
	var err error
	first := c.Events[0]
	sep := fmt.Sprintf("%v %v %v |\t", first.DataEvent.GetExchange(), first.DataEvent.GetAssetType(), first.DataEvent.Pair())

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
		if c.LowestClosePrice.IsZero() || price.LessThan(c.LowestClosePrice) {
			c.LowestClosePrice = price
		}
		if price.GreaterThan(c.HighestClosePrice) {
			c.HighestClosePrice = price
		}
	}

	oneHundred := decimal.NewFromInt(100)
	c.MarketMovement = lastPrice.Sub(firstPrice).Div(firstPrice).Mul(oneHundred)
	if first.Holdings.TotalValue.GreaterThan(decimal.Zero) {
		c.StrategyMovement = last.Holdings.TotalValue.Sub(first.Holdings.TotalValue).Div(first.Holdings.TotalValue).Mul(oneHundred)
	}
	c.calculateHighestCommittedFunds()
	returnsPerCandle := make([]decimal.Decimal, len(c.Events))
	benchmarkRates := make([]decimal.Decimal, len(c.Events))

	var allDataEvents []common.DataEventHandler
	for i := range c.Events {
		returnsPerCandle[i] = c.Events[i].Holdings.ChangeInTotalValuePercent
		allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
		if i == 0 {
			continue
		}
		if c.Events[i].SignalEvent != nil && c.Events[i].SignalEvent.GetDirection() == common.MissingData {
			c.ShowMissingDataWarning = true
		}
		benchmarkRates[i] = c.Events[i].DataEvent.ClosePrice().Sub(
			c.Events[i-1].DataEvent.ClosePrice()).Div(
			c.Events[i-1].DataEvent.ClosePrice())
	}

	// remove the first entry as its zero and impacts
	// ratio calculations as no movement has been made
	benchmarkRates = benchmarkRates[1:]
	returnsPerCandle = returnsPerCandle[1:]
	c.MaxDrawdown, err = CalculateBiggestEventDrawdown(allDataEvents)
	if err != nil {
		errs = append(errs, err)
	}

	interval := first.DataEvent.GetInterval()
	intervalsPerYear := interval.IntervalsPerYear()
	riskFreeRatePerCandle := riskFreeRate.Div(decimal.NewFromFloat(intervalsPerYear))
	c.ArithmeticRatios, c.GeometricRatios, err = CalculateRatios(benchmarkRates, returnsPerCandle, riskFreeRatePerCandle, &c.MaxDrawdown, sep)
	if err != nil {
		return err
	}

	if last.Holdings.QuoteInitialFunds.GreaterThan(decimal.Zero) {
		cagr, err := gctmath.DecimalCompoundAnnualGrowthRate(
			last.Holdings.QuoteInitialFunds,
			last.Holdings.TotalValue,
			decimal.NewFromFloat(intervalsPerYear),
			decimal.NewFromInt(int64(len(c.Events))),
		)
		if err != nil {
			errs = append(errs, err)
		}
		if !cagr.IsZero() {
			c.CompoundAnnualGrowthRate = cagr
		}
	}
	c.IsStrategyProfitable = last.Holdings.TotalValue.GreaterThan(first.Holdings.TotalValue)
	c.DoesPerformanceBeatTheMarket = c.StrategyMovement.GreaterThan(c.MarketMovement)

	c.TotalFees = last.Holdings.TotalFees.Round(8)
	c.TotalValueLostToVolumeSizing = last.Holdings.TotalValueLostToVolumeSizing.Round(2)
	c.TotalValueLost = last.Holdings.TotalValueLost.Round(2)
	c.TotalValueLostToSlippage = last.Holdings.TotalValueLostToSlippage.Round(2)
	c.TotalAssetValue = last.Holdings.BaseValue.Round(8)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// PrintResults outputs all calculated statistics to the command line
func (c *CurrencyPairStatistic) PrintResults(e string, a asset.Item, p currency.Pair, usingExchangeLevelFunding bool) {
	var errs gctcommon.Errors
	sort.Slice(c.Events, func(i, j int) bool {
		return c.Events[i].DataEvent.GetTime().Before(c.Events[j].DataEvent.GetTime())
	})
	last := c.Events[len(c.Events)-1]
	first := c.Events[0]
	c.StartingClosePrice = first.DataEvent.ClosePrice()
	c.EndingClosePrice = last.DataEvent.ClosePrice()
	c.TotalOrders = c.BuyOrders + c.SellOrders
	last.Holdings.TotalValueLost = last.Holdings.TotalValueLostToSlippage.Add(last.Holdings.TotalValueLostToVolumeSizing)
	sep := fmt.Sprintf("%v %v %v |\t", e, a, p)
	currStr := fmt.Sprintf("------------------Stats for %v %v %v------------------------------------------", e, a, p)
	log.Infof(log.BackTester, currStr[:61])
	log.Infof(log.BackTester, "%s Highest committed funds: %v at %v\n\n", sep, convert.DecimalToCommaSeparatedString(c.HighestCommittedFunds.Value, 8, ".", ","), c.HighestCommittedFunds.Time)
	log.Infof(log.BackTester, "%s Buy orders: %d", sep, convert.IntToCommaSeparatedString(c.BuyOrders, ","))
	log.Infof(log.BackTester, "%s Buy value: %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.BoughtValue, 8, ".", ","))
	log.Infof(log.BackTester, "%s Buy amount: %v %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.BoughtAmount, 8, ".", ","), last.Holdings.Pair.Base)
	log.Infof(log.BackTester, "%s Sell orders: %d", sep, convert.IntToCommaSeparatedString(c.SellOrders, ","))
	log.Infof(log.BackTester, "%s Sell value: %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.SoldValue, 8, ".", ","))
	log.Infof(log.BackTester, "%s Sell amount: %v %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.SoldAmount, 8, ".", ","), last.Holdings.SoldAmount.Round(8))
	log.Infof(log.BackTester, "%s Total orders: %d\n\n", sep, convert.IntToCommaSeparatedString(c.TotalOrders, ","))

	log.Info(log.BackTester, "------------------Max Drawdown-------------------------------")
	log.Infof(log.BackTester, "%s Highest Price of drawdown: %v", sep, convert.DecimalToCommaSeparatedString(c.MaxDrawdown.Highest.Value, 8, ".", ","))
	log.Infof(log.BackTester, "%s Time of highest price of drawdown: %v", sep, c.MaxDrawdown.Highest.Time)
	log.Infof(log.BackTester, "%s Lowest Price of drawdown: %v", sep, convert.DecimalToCommaSeparatedString(c.MaxDrawdown.Lowest.Value, 8, ".", ","))
	log.Infof(log.BackTester, "%s Time of lowest price of drawdown: %v", sep, c.MaxDrawdown.Lowest.Time)
	log.Infof(log.BackTester, "%s Calculated Drawdown: %v%%", sep, convert.DecimalToCommaSeparatedString(c.MaxDrawdown.DrawdownPercent, 8, ".", ","))
	log.Infof(log.BackTester, "%s Difference: %v", sep, convert.DecimalToCommaSeparatedString(c.MaxDrawdown.Highest.Value.Sub(c.MaxDrawdown.Lowest.Value), 2, ".", ","))
	log.Infof(log.BackTester, "%s Drawdown length: %d\n\n", sep, c.MaxDrawdown.IntervalDuration)
	if !usingExchangeLevelFunding {
		log.Info(log.BackTester, "------------------Ratios------------------------------------------------")
		log.Infof(log.BackTester, "%s Compound Annual Growth Rate: %v\n\n", sep, c.CompoundAnnualGrowthRate.Round(2))
		log.Info(log.BackTester, "------------------Arithmetic--------------------------------------------")
		if c.ShowMissingDataWarning {
			log.Infoln(log.BackTester, "Missing data was detected during this backtesting run")
			log.Infoln(log.BackTester, "Ratio calculations will be skewed")
		}
		log.Infof(log.BackTester, "%s Sharpe ratio: %v", sep, c.ArithmeticRatios.SharpeRatio.Round(4))
		log.Infof(log.BackTester, "%s Sortino ratio: %v", sep, c.ArithmeticRatios.SortinoRatio.Round(4))
		log.Infof(log.BackTester, "%s Information ratio: %v", sep, c.ArithmeticRatios.InformationRatio.Round(4))
		log.Infof(log.BackTester, "%s Calmar ratio: %v\n\n", sep, c.ArithmeticRatios.CalmarRatio.Round(4))

		log.Info(log.BackTester, "------------------Geometric--------------------------------------------")
		if c.ShowMissingDataWarning {
			log.Infoln(log.BackTester, "Missing data was detected during this backtesting run")
			log.Infoln(log.BackTester, "Ratio calculations will be skewed")
		}
		log.Infof(log.BackTester, "%s Sharpe ratio: %v", sep, c.GeometricRatios.SharpeRatio.Round(4))
		log.Infof(log.BackTester, "%s Sortino ratio: %v", sep, c.GeometricRatios.SortinoRatio.Round(4))
		log.Infof(log.BackTester, "%s Information ratio: %v", sep, c.GeometricRatios.InformationRatio.Round(4))
		log.Infof(log.BackTester, "%s Calmar ratio: %v\n\n", sep, c.GeometricRatios.CalmarRatio.Round(4))
	}

	log.Info(log.BackTester, "------------------Results------------------------------------")
	log.Infof(log.BackTester, "%s Starting Close Price: %s", sep, convert.DecimalToCommaSeparatedString(c.StartingClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Finishing Close Price: %s", sep, convert.DecimalToCommaSeparatedString(c.EndingClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Lowest Close Price: %s", sep, convert.DecimalToCommaSeparatedString(c.LowestClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Highest Close Price: %s", sep, convert.DecimalToCommaSeparatedString(c.HighestClosePrice, 8, ".", ","))

	log.Infof(log.BackTester, "%s Market movement: %s%%", sep, convert.DecimalToCommaSeparatedString(c.MarketMovement, 2, ".", ","))
	if !usingExchangeLevelFunding {
		log.Infof(log.BackTester, "%s Strategy movement: %s%%", sep, convert.DecimalToCommaSeparatedString(c.StrategyMovement, 2, ".", ","))
		log.Infof(log.BackTester, "%s Did it beat the market: %v", sep, c.StrategyMovement.GreaterThan(c.MarketMovement))
	}

	log.Infof(log.BackTester, "%s Value lost to volume sizing: %v", sep, c.TotalValueLostToVolumeSizing.Round(2))
	log.Infof(log.BackTester, "%s Value lost to slippage: %v", sep, c.TotalValueLostToSlippage.Round(2))
	log.Infof(log.BackTester, "%s Total Value lost: %v", sep, c.TotalValueLost.Round(2))
	log.Infof(log.BackTester, "%s Total Fees: %v\n\n", sep, c.TotalFees.Round(8))

	log.Infof(log.BackTester, "%s Final holdings value: %v", sep, c.TotalAssetValue.Round(8))
	if !usingExchangeLevelFunding {
		// the following have no direct translation to individual exchange level funds as they
		// combine base and quote values
		log.Infof(log.BackTester, "%s Final funds: %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.QuoteSize, 8, ".", ","))
		log.Infof(log.BackTester, "%s Final holdings: %v", sep, convert.DecimalToCommaSeparatedString(last.Holdings.BaseSize, 8, ".", ","))
		log.Infof(log.BackTester, "%s Final total value: %v\n\n", sep, convert.DecimalToCommaSeparatedString(last.Holdings.TotalValue, 8, ".", ","))
	}
	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}
}

// CalculateBiggestEventDrawdown calculates the biggest drawdown using a slice of DataEvents
func CalculateBiggestEventDrawdown(closePrices []common.DataEventHandler) (Swing, error) {
	var lowestPrice, highestPrice decimal.Decimal
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
		if lowestPrice.GreaterThan(currLow) && !currLow.IsZero() {
			lowestPrice = currLow
			lowestTime = currTime
		}
		if highestPrice.LessThan(currHigh) && highestPrice.IsPositive() {
			if lowestTime.Equal(highestTime) {
				// create distinction if the greatest drawdown occurs within the same candle
				lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
			}
			intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[i].GetInterval(), 0)
			if err != nil {
				log.Error(log.BackTester, err)
				continue
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
	if (len(swings) > 0 && swings[len(swings)-1].Lowest.Value != closePrices[len(closePrices)-1].LowPrice()) || swings == nil {
		// need to close out the final drawdown
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
		}
		intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, closePrices[0].GetInterval(), 0)
		if err != nil {
			return Swing{}, err
		}
		drawdownPercent := decimal.Zero
		if highestPrice.GreaterThan(decimal.Zero) {
			drawdownPercent = lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100))
		}
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
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

func (c *CurrencyPairStatistic) calculateHighestCommittedFunds() {
	for i := range c.Events {
		if c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.ClosePrice()).GreaterThan(c.HighestCommittedFunds.Value) {
			c.HighestCommittedFunds.Value = c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.ClosePrice())
			c.HighestCommittedFunds.Time = c.Events[i].Holdings.Timestamp
		}
	}
}

// CalculateBiggestValueAtTimeDrawdown calculates the biggest drawdown using a slice of ValueAtTimes
func CalculateBiggestValueAtTimeDrawdown(closePrices []ValueAtTime, interval gctkline.Interval) Swing {
	var lowestPrice, highestPrice decimal.Decimal
	var lowestTime, highestTime time.Time
	var swings []Swing
	if len(closePrices) > 0 {
		lowestPrice = closePrices[0].Value
		highestPrice = closePrices[0].Value
		lowestTime = closePrices[0].Time
		highestTime = closePrices[0].Time
	}
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
				lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
			}
			intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, interval, 0)
			if err != nil {
				log.Error(log.BackTester, err)
				return Swing{}
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
			lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
		}
		intervals, err := gctkline.CalculateCandleDateRanges(highestTime, lowestTime, interval, 0)
		if err != nil {
			log.Error(log.BackTester, err)
		}
		drawdownPercent := decimal.Zero
		if highestPrice.GreaterThan(decimal.Zero) {
			drawdownPercent = lowestPrice.Sub(highestPrice).Div(highestPrice).Mul(decimal.NewFromInt(100))
		}
		if lowestTime.Equal(highestTime) {
			// create distinction if the greatest drawdown occurs within the same candle
			lowestTime = lowestTime.Add((time.Hour * 23) + (time.Minute * 59) + (time.Second * 59))
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

	return maxDrawdown
}

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

	firstPrice := first.DataEvent.GetClosePrice()
	last := c.Events[len(c.Events)-1]
	lastPrice := last.DataEvent.GetClosePrice()
	for i := range last.Transactions.Orders {
		if last.Transactions.Orders[i].Order.Side == gctorder.Buy {
			c.BuyOrders++
		} else if last.Transactions.Orders[i].Order.Side == gctorder.Sell {
			c.SellOrders++
		}
	}
	for i := range c.Events {
		price := c.Events[i].DataEvent.GetClosePrice()
		if c.LowestClosePrice.IsZero() || price.LessThan(c.LowestClosePrice) {
			c.LowestClosePrice = price
		}
		if price.GreaterThan(c.HighestClosePrice) {
			c.HighestClosePrice = price
		}
	}

	oneHundred := decimal.NewFromInt(100)
	if !firstPrice.IsZero() {
		c.MarketMovement = lastPrice.Sub(firstPrice).Div(firstPrice).Mul(oneHundred)
	}
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
		if c.Events[i].DataEvent.GetClosePrice().IsZero() || c.Events[i-1].DataEvent.GetClosePrice().IsZero() {
			// closing price for the current candle or previous candle is zero, use the previous
			// benchmark rate to allow some consistency
			c.ShowMissingDataWarning = true
			benchmarkRates[i] = benchmarkRates[i-1]
			continue
		}
		benchmarkRates[i] = c.Events[i].DataEvent.GetClosePrice().Sub(
			c.Events[i-1].DataEvent.GetClosePrice()).Div(
			c.Events[i-1].DataEvent.GetClosePrice())
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
	if last.PNL != nil {
		c.UnrealisedPNL = last.PNL.Result.UnrealisedPNL
		// ????
		c.RealisedPNL = last.PNL.Result.RealisedPNLBeforeFees
	}
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
	c.StartingClosePrice = first.DataEvent.GetClosePrice()
	c.EndingClosePrice = last.DataEvent.GetClosePrice()
	c.TotalOrders = c.BuyOrders + c.SellOrders
	last.Holdings.TotalValueLost = last.Holdings.TotalValueLostToSlippage.Add(last.Holdings.TotalValueLostToVolumeSizing)
	sep := fmt.Sprintf("%v %v %v |\t", e, a, p)
	currStr := fmt.Sprintf("------------------Stats for %v %v %v------------------------------------------", e, a, p)
	log.Infof(log.BackTester, currStr[:61])
	log.Infof(log.BackTester, "%s Highest committed funds: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestCommittedFunds.Value, 8, ".", ","), c.HighestCommittedFunds.Time)
	log.Infof(log.BackTester, "%s Buy orders: %s", sep, convert.IntToHumanFriendlyString(c.BuyOrders, ","))
	log.Infof(log.BackTester, "%s Buy value: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.BoughtValue, 8, ".", ","))
	log.Infof(log.BackTester, "%s Buy amount: %s %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.BoughtAmount, 8, ".", ","), last.Holdings.Pair.Base)
	log.Infof(log.BackTester, "%s Sell orders: %s", sep, convert.IntToHumanFriendlyString(c.SellOrders, ","))
	log.Infof(log.BackTester, "%s Sell value: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.SoldValue, 8, ".", ","))
	log.Infof(log.BackTester, "%s Sell amount: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.SoldAmount, 8, ".", ","))
	log.Infof(log.BackTester, "%s Total orders: %s\n\n", sep, convert.IntToHumanFriendlyString(c.TotalOrders, ","))

	log.Info(log.BackTester, "------------------Max Drawdown-------------------------------")
	log.Infof(log.BackTester, "%s Highest Price of drawdown: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Highest.Value, 8, ".", ","), c.MaxDrawdown.Highest.Time)
	log.Infof(log.BackTester, "%s Lowest Price of drawdown: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Lowest.Value, 8, ".", ","), c.MaxDrawdown.Lowest.Time)
	log.Infof(log.BackTester, "%s Calculated Drawdown: %s%%", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.DrawdownPercent, 8, ".", ","))
	log.Infof(log.BackTester, "%s Difference: %s", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Highest.Value.Sub(c.MaxDrawdown.Lowest.Value), 2, ".", ","))
	log.Infof(log.BackTester, "%s Drawdown length: %s\n\n", sep, convert.IntToHumanFriendlyString(c.MaxDrawdown.IntervalDuration, ","))
	if !usingExchangeLevelFunding {
		log.Info(log.BackTester, "------------------Ratios------------------------------------------------")
		log.Info(log.BackTester, "------------------Rates-------------------------------------------------")
		log.Infof(log.BackTester, "%s Compound Annual Growth Rate: %s", sep, convert.DecimalToHumanFriendlyString(c.CompoundAnnualGrowthRate, 2, ".", ","))
		log.Info(log.BackTester, "------------------Arithmetic--------------------------------------------")
		if c.ShowMissingDataWarning {
			log.Infoln(log.BackTester, "Missing data was detected during this backtesting run")
			log.Infoln(log.BackTester, "Ratio calculations will be skewed")
		}
		log.Infof(log.BackTester, "%s Sharpe ratio: %v", sep, c.ArithmeticRatios.SharpeRatio.Round(4))
		log.Infof(log.BackTester, "%s Sortino ratio: %v", sep, c.ArithmeticRatios.SortinoRatio.Round(4))
		log.Infof(log.BackTester, "%s Information ratio: %v", sep, c.ArithmeticRatios.InformationRatio.Round(4))
		log.Infof(log.BackTester, "%s Calmar ratio: %v", sep, c.ArithmeticRatios.CalmarRatio.Round(4))

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
	log.Infof(log.BackTester, "%s Starting Close Price: %s", sep, convert.DecimalToHumanFriendlyString(c.StartingClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Finishing Close Price: %s", sep, convert.DecimalToHumanFriendlyString(c.EndingClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Lowest Close Price: %s", sep, convert.DecimalToHumanFriendlyString(c.LowestClosePrice, 8, ".", ","))
	log.Infof(log.BackTester, "%s Highest Close Price: %s", sep, convert.DecimalToHumanFriendlyString(c.HighestClosePrice, 8, ".", ","))

	log.Infof(log.BackTester, "%s Market movement: %s%%", sep, convert.DecimalToHumanFriendlyString(c.MarketMovement, 2, ".", ","))
	if !usingExchangeLevelFunding {
		log.Infof(log.BackTester, "%s Strategy movement: %s%%", sep, convert.DecimalToHumanFriendlyString(c.StrategyMovement, 2, ".", ","))
		log.Infof(log.BackTester, "%s Did it beat the market: %v", sep, c.StrategyMovement.GreaterThan(c.MarketMovement))
	}

	log.Infof(log.BackTester, "%s Value lost to volume sizing: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLostToVolumeSizing, 2, ".", ","))
	log.Infof(log.BackTester, "%s Value lost to slippage: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLostToSlippage, 2, ".", ","))
	log.Infof(log.BackTester, "%s Total Value lost: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLost, 2, ".", ","))
	log.Infof(log.BackTester, "%s Total Fees: %s\n\n", sep, convert.DecimalToHumanFriendlyString(c.TotalFees, 8, ".", ","))

	log.Infof(log.BackTester, "%s Final holdings value: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalAssetValue, 8, ".", ","))
	if !usingExchangeLevelFunding {
		// the following have no direct translation to individual exchange level funds as they
		// combine base and quote values
		log.Infof(log.BackTester, "%s Final funds: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.QuoteSize, 8, ".", ","))
		log.Infof(log.BackTester, "%s Final holdings: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.BaseSize, 8, ".", ","))
		log.Infof(log.BackTester, "%s Final total value: %s\n\n", sep, convert.DecimalToHumanFriendlyString(last.Holdings.TotalValue, 8, ".", ","))
	}

	if last.PNL != nil {
		log.Infof(log.BackTester, "%s Final unPNL: %s\n\n", sep, convert.DecimalToHumanFriendlyString(last.PNL.Result.UnrealisedPNL, 8, ".", ","))
		log.Infof(log.BackTester, "%s Final PNL: %s\n\n", sep, convert.DecimalToHumanFriendlyString(last.PNL.Result.RealisedPNLBeforeFees, 8, ".", ","))
	}
	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Error(log.BackTester, errs[i].Error())
		}
	}
}

// CalculateBiggestEventDrawdown calculates the biggest drawdown using a slice of DataEvents
func CalculateBiggestEventDrawdown(closePrices []common.DataEventHandler) (Swing, error) {
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
				log.Error(log.BackTester, err)
				continue
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
			return Swing{}, err
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

func (c *CurrencyPairStatistic) calculateHighestCommittedFunds() {
	for i := range c.Events {
		if c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.GetClosePrice()).GreaterThan(c.HighestCommittedFunds.Value) {
			c.HighestCommittedFunds.Value = c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.GetClosePrice())
			c.HighestCommittedFunds.Time = c.Events[i].Holdings.Timestamp
		}
	}
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
			log.Error(log.BackTester, err)
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

package statistics

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
		switch last.Transactions.Orders[i].Order.Side {
		case gctorder.Buy:
			c.BuyOrders++
		case gctorder.Sell:
			c.SellOrders++
		case gctorder.Long:
			c.LongOrders++
		case gctorder.Short:
			c.ShortOrders++
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
		c.RealisedPNL = last.PNL.Result.RealisedPNL
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (c *CurrencyPairStatistic) calculateHighestCommittedFunds() {
	for i := range c.Events {
		if c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.GetClosePrice()).GreaterThan(c.HighestCommittedFunds.Value) {
			c.HighestCommittedFunds.Value = c.Events[i].Holdings.BaseSize.Mul(c.Events[i].DataEvent.GetClosePrice())
			c.HighestCommittedFunds.Time = c.Events[i].Holdings.Timestamp
		}
	}
}

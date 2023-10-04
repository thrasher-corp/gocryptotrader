package statistics

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// CalculateResults calculates all statistics for the exchange, asset, currency pair
func (c *CurrencyPairStatistic) CalculateResults(riskFreeRate decimal.Decimal) error {
	first := c.Events[0]
	if first.DataEvent == nil {
		// you can call stop while a backtester run is running
		// if the first data event isn't present, then it hasn't been properly run
		return errNoDataAtOffset
	}
	sep := fmt.Sprintf("%v %v %v |\t", first.DataEvent.GetExchange(), first.DataEvent.GetAssetType(), first.DataEvent.Pair())

	firstPrice := first.ClosePrice
	last := c.Events[len(c.Events)-1]
	if last.ComplianceSnapshot == nil {
		return errMissingSnapshots
	}
	lastPrice := last.ClosePrice
	for i := range last.ComplianceSnapshot.Orders {
		if last.ComplianceSnapshot.Orders[i].Order.Side.IsLong() {
			c.BuyOrders++
		} else {
			c.SellOrders++
		}
	}
	for i := range c.Events {
		price := c.Events[i].ClosePrice
		if (price.LessThan(c.LowestClosePrice.Value) || !c.LowestClosePrice.Set) && !price.IsZero() {
			c.LowestClosePrice.Value = price
			c.LowestClosePrice.Time = c.Events[i].Time
			c.LowestClosePrice.Set = true
		}
		if price.GreaterThan(c.HighestClosePrice.Value) {
			c.HighestClosePrice.Value = price
			c.HighestClosePrice.Time = c.Events[i].Time
			c.HighestClosePrice.Set = true
		}
	}

	oneHundred := decimal.NewFromInt(100)
	if !firstPrice.IsZero() {
		c.MarketMovement = lastPrice.Sub(firstPrice).Div(firstPrice).Mul(oneHundred)
	}
	if !first.Holdings.TotalValue.IsZero() {
		c.StrategyMovement = last.Holdings.TotalValue.Sub(first.Holdings.TotalValue).Div(first.Holdings.TotalValue).Mul(oneHundred)
	}
	c.analysePNLGrowth()
	err := c.calculateHighestCommittedFunds()
	if err != nil {
		return err
	}
	returnsPerCandle := make([]decimal.Decimal, len(c.Events))
	benchmarkRates := make([]decimal.Decimal, len(c.Events))

	allDataEvents := make([]data.Event, len(c.Events))
	for i := range c.Events {
		returnsPerCandle[i] = c.Events[i].Holdings.ChangeInTotalValuePercent
		allDataEvents[i] = c.Events[i].DataEvent
		if i == 0 {
			continue
		}
		if c.Events[i].SignalEvent != nil && c.Events[i].SignalEvent.GetDirection() == gctorder.MissingData {
			c.ShowMissingDataWarning = true
		}
		if c.Events[i].ClosePrice.IsZero() || c.Events[i-1].ClosePrice.IsZero() {
			// closing price for the current candle or previous candle is zero, use the previous
			// benchmark rate to allow some consistency
			c.ShowMissingDataWarning = true
			benchmarkRates[i] = benchmarkRates[i-1]
			continue
		}
		benchmarkRates[i] = c.Events[i].ClosePrice.Sub(
			c.Events[i-1].ClosePrice).Div(
			c.Events[i-1].ClosePrice)
	}

	// remove the first entry as its zero and impacts
	// ratio calculations as no movement has been made
	benchmarkRates = benchmarkRates[1:]
	returnsPerCandle = returnsPerCandle[1:]
	var errs error
	c.MaxDrawdown, err = CalculateBiggestEventDrawdown(allDataEvents)
	if err != nil {
		errs = gctcommon.AppendError(errs, err)
	}

	interval := first.DataEvent.GetInterval()
	intervalsPerYear := interval.IntervalsPerYear()
	riskFreeRatePerCandle := riskFreeRate.Div(decimal.NewFromFloat(intervalsPerYear))
	c.ArithmeticRatios, c.GeometricRatios, err = CalculateRatios(benchmarkRates, returnsPerCandle, riskFreeRatePerCandle, &c.MaxDrawdown, sep)
	if err != nil {
		return err
	}

	if !last.Holdings.QuoteInitialFunds.IsZero() {
		var cagr decimal.Decimal
		cagr, err = gctmath.DecimalCompoundAnnualGrowthRate(
			last.Holdings.QuoteInitialFunds,
			last.Holdings.TotalValue,
			decimal.NewFromFloat(intervalsPerYear),
			decimal.NewFromInt(int64(len(c.Events))),
		)
		if err != nil && !errors.Is(err, gctmath.ErrPowerDifferenceTooSmall) {
			errs = gctcommon.AppendError(errs, err)
		}
		c.CompoundAnnualGrowthRate = cagr
	}
	c.IsStrategyProfitable = last.Holdings.TotalValue.GreaterThan(first.Holdings.TotalValue)
	c.DoesPerformanceBeatTheMarket = c.StrategyMovement.GreaterThan(c.MarketMovement)
	c.TotalFees = last.Holdings.TotalFees.Round(8)
	c.TotalValueLostToVolumeSizing = last.Holdings.TotalValueLostToVolumeSizing.Round(2)
	c.TotalValueLost = last.Holdings.TotalValueLost.Round(2)
	c.TotalValueLostToSlippage = last.Holdings.TotalValueLostToSlippage.Round(2)
	c.TotalAssetValue = last.Holdings.BaseValue.Round(8)
	if last.PNL != nil {
		c.UnrealisedPNL = last.PNL.GetUnrealisedPNL().PNL
		c.RealisedPNL = last.PNL.GetRealisedPNL().PNL
	}
	return errs
}

func (c *CurrencyPairStatistic) calculateHighestCommittedFunds() error {
	switch {
	case c.Asset == asset.Spot:
		for i := range c.Events {
			if c.Events[i].Holdings.CommittedFunds.GreaterThan(c.HighestCommittedFunds.Value) || !c.HighestCommittedFunds.Set {
				c.HighestCommittedFunds.Value = c.Events[i].Holdings.CommittedFunds
				c.HighestCommittedFunds.Time = c.Events[i].Time
				c.HighestCommittedFunds.Set = true
			}
		}
	case c.Asset.IsFutures():
		for i := range c.Events {
			valueAtTime := c.Events[i].Holdings.BaseSize.Mul(c.Events[i].ClosePrice)
			if valueAtTime.GreaterThan(c.HighestCommittedFunds.Value) || !c.HighestCommittedFunds.Set {
				c.HighestCommittedFunds.Value = valueAtTime
				c.HighestCommittedFunds.Time = c.Events[i].Time
				c.HighestCommittedFunds.Set = true
			}
		}
	default:
		return fmt.Errorf("%v %w", c.Asset, asset.ErrNotSupported)
	}
	return nil
}

func (c *CurrencyPairStatistic) analysePNLGrowth() {
	if !c.Asset.IsFutures() {
		return
	}
	var lowestUnrealised, highestUnrealised, lowestRealised, highestRealised ValueAtTime
	for i := range c.Events {
		if c.Events[i].PNL == nil {
			continue
		}
		unrealised := c.Events[i].PNL.GetUnrealisedPNL()
		realised := c.Events[i].PNL.GetRealisedPNL()
		if unrealised.PNL.LessThan(lowestUnrealised.Value) ||
			(!lowestUnrealised.Set && !unrealised.PNL.IsZero()) {
			lowestUnrealised.Value = unrealised.PNL
			lowestUnrealised.Time = unrealised.Time
			lowestUnrealised.Set = true
		}
		if unrealised.PNL.GreaterThan(highestUnrealised.Value) ||
			(!highestUnrealised.Set && !unrealised.PNL.IsZero()) {
			highestUnrealised.Value = unrealised.PNL
			highestUnrealised.Time = unrealised.Time
			highestUnrealised.Set = true
		}

		if realised.PNL.LessThan(lowestRealised.Value) ||
			(!lowestRealised.Set && !realised.PNL.IsZero()) {
			lowestRealised.Value = realised.PNL
			lowestRealised.Time = realised.Time
			lowestRealised.Set = true
		}
		if realised.PNL.GreaterThan(highestRealised.Value) ||
			(!highestRealised.Set && !realised.PNL.IsZero()) {
			highestRealised.Value = realised.PNL
			highestRealised.Time = realised.Time
			highestRealised.Set = true
		}
	}
	c.LowestRealisedPNL = lowestRealised
	c.LowestUnrealisedPNL = lowestUnrealised
	c.HighestUnrealisedPNL = highestUnrealised
	c.HighestRealisedPNL = highestRealised
}
